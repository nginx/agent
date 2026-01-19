// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
)

type ExternalFileOperator struct {
	fileManagerService *FileManagerService
}

const (
	sniffMaxBytes = 500
	mimeSplitN    = 2
	httpScheme    = "http"
	httpsScheme   = "https"
)

func NewExternalFileOperator(fms *FileManagerService) *ExternalFileOperator {
	return &ExternalFileOperator{fileManagerService: fms}
}

func (efo *ExternalFileOperator) DownloadExternalFile(ctx context.Context, fileAction *model.FileCache,
	filePath string,
) error {
	location := fileAction.File.GetExternalDataSource().GetLocation()
	permission := fileAction.File.GetFileMeta().GetPermissions()
	fileName := fileAction.File.GetFileMeta().GetName()

	slog.InfoContext(ctx, "Downloading external file from", "location", location)

	var contentToWrite []byte
	var downloadErr, updateError error
	var headers DownloadHeader

	contentToWrite, headers, downloadErr = efo.downloadFileContent(ctx, fileAction.File)

	if downloadErr != nil {
		updateError = fmt.Errorf("failed to download file %s from %s: %w",
			fileName, location, downloadErr)

		return updateError
	}

	if contentToWrite == nil {
		slog.DebugContext(ctx, "External file unchanged (304), skipping disk write.",
			"file", fileName)

		// preserve previous behavior: mark as unchanged so later rename is skipped
		fileAction.Action = model.Unchanged

		// persist headers if present
		if headers.ETag != "" || headers.LastModified != "" {
			efo.fileManagerService.externalFileHeaders[fileName] = headers
		}

		return nil
	}

	// Validate downloaded file type before writing to temp location.
	if err := efo.validateDownloadedFile(contentToWrite, fileName); err != nil {
		return fmt.Errorf("downloaded file validation failed for %s: %w", fileName, err)
	}

	efo.fileManagerService.externalFileHeaders[fileName] = headers

	writeErr := efo.fileManagerService.fileOperator.Write(
		ctx,
		contentToWrite,
		filePath,
		permission,
	)

	if writeErr != nil {
		return fmt.Errorf("failed to write downloaded content to temp file %s: %w", filePath, writeErr)
	}

	return nil
}

//nolint:revive, cyclop // Can not break this function further without harming readability
func (efo *ExternalFileOperator) downloadFileContent(ctx context.Context, file *mpi.File) (content []byte,
	headers DownloadHeader, err error,
) {
	fileName := file.GetFileMeta().GetName()
	downloadURL := file.GetExternalDataSource().GetLocation()
	externalConfig := efo.fileManagerService.agentConfig.ExternalDataSource

	if !efo.isDomainAllowed(downloadURL, externalConfig.AllowedDomains) {
		return nil, DownloadHeader{}, fmt.Errorf("download URL %s is not in the allowed domains list", downloadURL)
	}

	pasrsedURl, err := url.Parse(downloadURL)
	if err != nil {
		return nil, DownloadHeader{}, fmt.Errorf("failed to parse URL: %w", err)
	}

	originalScheme := pasrsedURl.Scheme
	if originalScheme != httpScheme && originalScheme != httpsScheme {
		pasrsedURl.Scheme = httpsScheme
	}
	networkURL := pasrsedURl.String()

	httpClient, err := efo.setupHTTPClient(ctx, externalConfig.ProxyURL.URL)
	if err != nil {
		return nil, DownloadHeader{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, networkURL, nil)
	if err != nil {
		return nil, DownloadHeader{}, fmt.Errorf("failed to create request for %s: %w", downloadURL, err)
	}

	if originalScheme != httpScheme && originalScheme != httpsScheme {
		req.Header.Set("X-Original-Scheme", originalScheme)
	}

	if externalConfig.ProxyURL.URL != "" {
		efo.addConditionalHeaders(ctx, req, fileName)
	} else {
		slog.DebugContext(ctx, "No proxy configured; sending plain HTTP request without caching headers.")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, DownloadHeader{}, fmt.Errorf("failed to execute download request for %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		headers.ETag = resp.Header.Get("ETag")
		headers.LastModified = resp.Header.Get("Last-Modified")
	case http.StatusNotModified:
		slog.DebugContext(ctx, "File content unchanged (304 Not Modified)", "file_name", fileName)
		// return empty content but preserve headers if present
		header := DownloadHeader{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
		}

		return nil, header, nil
	default:
		const maxErrBody = 4096
		var bodyMsg string

		limited := io.LimitReader(resp.Body, maxErrBody)
		body, readErr := io.ReadAll(limited)
		if readErr != nil {
			slog.DebugContext(ctx, "Failed to read error response body", "error", readErr, "status", resp.StatusCode)
		} else {
			bodyMsg = strings.TrimSpace(string(body))
		}

		if bodyMsg != "" {
			return nil, DownloadHeader{}, fmt.Errorf("download failed with status code %d: %s",
				resp.StatusCode, bodyMsg)
		}

		return nil, DownloadHeader{}, fmt.Errorf("download failed with status code %d", resp.StatusCode)
	}

	reader := io.Reader(resp.Body)
	if efo.fileManagerService.agentConfig.ExternalDataSource.MaxBytes > 0 {
		reader = io.LimitReader(resp.Body, efo.fileManagerService.agentConfig.ExternalDataSource.MaxBytes)
	}

	content, err = io.ReadAll(reader)
	if err != nil {
		return nil, DownloadHeader{}, fmt.Errorf("failed to read content from response body: %w", err)
	}

	slog.InfoContext(ctx, "Successfully downloaded file content", "file_name", fileName, "size", len(content))

	return content, headers, nil
}

func (efo *ExternalFileOperator) isDomainAllowed(downloadURL string, allowedDomains []string) bool {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		slog.Debug("Failed to parse download URL for domain check", "url", downloadURL, "error", err)
		return false
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return false
	}

	for _, domain := range allowedDomains {
		if domain == "" {
			continue
		}

		if domain == hostname || efo.isMatchesWildcardDomain(hostname, domain) {
			return true
		}
	}

	return false
}

func (efo *ExternalFileOperator) setupHTTPClient(ctx context.Context, proxyURLString string) (*http.Client, error) {
	var transport *http.Transport

	if proxyURLString != "" {
		proxyURL, err := url.Parse(proxyURLString)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL configured: %w", err)
		}
		slog.DebugContext(ctx, "Configuring HTTP client to use proxy", "proxy_url", proxyURLString)
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	} else {
		slog.DebugContext(ctx, "Configuring HTTP client for direct connection (no proxy)")
		transport = &http.Transport{
			Proxy: nil,
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   efo.fileManagerService.agentConfig.Client.FileDownloadTimeout,
	}

	return httpClient, nil
}

func (efo *ExternalFileOperator) addConditionalHeaders(ctx context.Context, req *http.Request, fileName string) {
	slog.DebugContext(ctx, "Proxy configured; adding headers to GET request.")

	manifestFiles, _, manifestFileErr := efo.fileManagerService.manifestFile()

	if manifestFileErr != nil && !errors.Is(manifestFileErr, os.ErrNotExist) {
		slog.WarnContext(ctx, "Error reading manifest file for headers", "error", manifestFileErr)
	}

	manifestFile, ok := manifestFiles[fileName]

	if ok && manifestFile != nil && manifestFile.ManifestFileMeta != nil {
		fileMeta := manifestFile.ManifestFileMeta

		if fileMeta.ETag != "" {
			req.Header.Set("If-None-Match", fileMeta.ETag)
		}
		if fileMeta.LastModified != "" {
			req.Header.Set("If-Modified-Since", fileMeta.LastModified)
		}
	} else {
		slog.DebugContext(ctx, "File not found in manifest or missing metadata; skipping conditional headers.",
			"file", fileName)
	}
}

func (efo *ExternalFileOperator) isMatchesWildcardDomain(hostname, pattern string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}

	baseDomain := pattern[2:]
	if strings.HasSuffix(hostname, baseDomain) {
		// Ensure it's a true subdomain match
		if hostname == baseDomain || hostname[len(hostname)-len(baseDomain)-1] == '.' {
			return true
		}
	}

	return false
}

// validate external file content and extension compatibility and enforces allowed file extension types.
func (efo *ExternalFileOperator) validateDownloadedFile(content []byte, fileName string) error {
	sniff := content
	if len(sniff) > sniffMaxBytes {
		sniff = sniff[:sniffMaxBytes]
	}

	detected := mimetype.Detect(sniff).String()
	ext := strings.ToLower(filepath.Ext(fileName)) // includes leading dot or empty

	if err := efo.checkExtCompatibility(ext, detected); err != nil {
		return err
	}

	allowed := efo.fileManagerService.agentConfig.ExternalDataSource.AllowedFileTypes
	if len(allowed) == 0 {
		return nil
	}

	if efo.allowedByConfig(detected, ext) {
		return nil
	}

	return fmt.Errorf("downloaded file type %s (ext %s) is not allowed", detected, ext)
}

// checkExtCompatibility returns an error when the file extension maps to a known MIME
// that is incompatible with the detected MIME.
func (efo *ExternalFileOperator) checkExtCompatibility(ext, detected string) error {
	if ext == "" {
		return nil
	}

	mimeByExt := mime.TypeByExtension(ext) // may include parameters like "; charset=utf-8"
	if mimeByExt == "" {
		return nil
	}

	mimeByExt = strings.SplitN(mimeByExt, ";", mimeSplitN)[0]
	if mimeByExt != detected && !strings.HasPrefix(detected, mimeByExt) {
		return fmt.Errorf("file extension %s implies MIME %s but detected %s", ext, mimeByExt, detected)
	}

	return nil
}

// checks the agent configuration allowed file extension list match it against the detected MIME of the external file.
//
//nolint:revive // simple logic
func (efo *ExternalFileOperator) allowedByConfig(detected, ext string) bool {
	for _, t := range efo.fileManagerService.agentConfig.ExternalDataSource.AllowedFileTypes {
		tt := strings.ToLower(strings.TrimSpace(t))
		if tt == "" {
			continue
		}
		if strings.Contains(tt, "/") {
			if strings.HasPrefix(detected, tt) {
				return true
			}
		} else {
			if !strings.HasPrefix(tt, ".") {
				tt = "." + tt
			}
			if ext == tt {
				return true
			}
		}
	}

	return false
}
