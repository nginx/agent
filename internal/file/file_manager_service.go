// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/internal/model"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/pkg/files"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileManagerServiceInterface

const (
	maxAttempts = 5
	dirPerm     = 0o755
	filePerm    = 0o600
)

type (
	fileOperator interface {
		Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error
		CreateFileDirectories(ctx context.Context, fileMeta *mpi.FileMeta, filePermission os.FileMode) error
		WriteChunkedFile(
			ctx context.Context,
			file *mpi.File,
			header *mpi.FileDataChunkHeader,
			stream grpc.ServerStreamingClient[mpi.FileDataChunk],
		) error
		ReadChunk(
			ctx context.Context,
			chunkSize uint32,
			reader *bufio.Reader,
			chunkID uint32,
		) (mpi.FileDataChunk_Content, error)
		WriteManifestFile(updatedFiles map[string]*model.ManifestFile,
			manifestDir, manifestPath string) (writeError error)
	}

	fileServiceOperatorInterface interface {
		File(ctx context.Context, file *mpi.File, fileActions map[string]*model.FileCache) error
		UpdateOverview(ctx context.Context, instanceID string, filesToUpdate []*mpi.File, iteration int) error
		ChunkedFile(ctx context.Context, file *mpi.File) error
		IsConnected() bool
		UpdateFile(
			ctx context.Context,
			instanceID string,
			fileToUpdate *mpi.File,
		) error
		SetIsConnected(isConnected bool)
	}

	fileManagerServiceInterface interface {
		ConfigApply(ctx context.Context, configApplyRequest *mpi.ConfigApplyRequest) (writeStatus model.WriteStatus,
			err error)
		Rollback(ctx context.Context, instanceID string) error
		ClearCache()
		ConfigUpload(ctx context.Context, configUploadRequest *mpi.ConfigUploadRequest) error
		ConfigUpdate(ctx context.Context, nginxConfigContext *model.NginxConfigContext)
		UpdateCurrentFilesOnDisk(ctx context.Context, updateFiles map[string]*mpi.File, referenced bool) error
		DetermineFileActions(
			ctx context.Context,
			currentFiles map[string]*mpi.File,
			modifiedFiles map[string]*model.FileCache,
		) (map[string]*model.FileCache, map[string][]byte, error)
		IsConnected() bool
		SetIsConnected(isConnected bool)
	}
)

type FileManagerService struct {
	manifestLock        *sync.RWMutex
	agentConfig         *config.Config
	fileOperator        fileOperator
	fileServiceOperator fileServiceOperatorInterface
	// map of files and the actions performed on them during config apply
	fileActions map[string]*model.FileCache // key is file path
	// map of the contents of files which have been updated or deleted during config apply, used during rollback
	rollbackFileContents map[string][]byte // key is file path
	// map of the files currently on disk, used to determine the file action during config apply
	currentFilesOnDisk    map[string]*mpi.File // key is file path
	previousManifestFiles map[string]*model.ManifestFile
	manifestFilePath      string
	filesMutex            sync.RWMutex
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config,
	manifestLock *sync.RWMutex,
) *FileManagerService {
	return &FileManagerService{
		agentConfig:           agentConfig,
		fileOperator:          NewFileOperator(manifestLock),
		fileServiceOperator:   NewFileServiceOperator(agentConfig, fileServiceClient, manifestLock),
		fileActions:           make(map[string]*model.FileCache),
		rollbackFileContents:  make(map[string][]byte),
		currentFilesOnDisk:    make(map[string]*mpi.File),
		previousManifestFiles: make(map[string]*model.ManifestFile),
		manifestFilePath:      agentConfig.ManifestDir + "/manifest.json",
		manifestLock:          manifestLock,
	}
}

func (fms *FileManagerService) IsConnected() bool {
	return fms.fileServiceOperator.IsConnected()
}

func (fms *FileManagerService) SetIsConnected(isConnected bool) {
	fms.fileServiceOperator.SetIsConnected(isConnected)
}

func (fms *FileManagerService) ConfigApply(ctx context.Context,
	configApplyRequest *mpi.ConfigApplyRequest,
) (status model.WriteStatus, err error) {
	fileOverview := configApplyRequest.GetOverview()

	if fileOverview == nil {
		return model.Error, errors.New("fileOverview is nil")
	}

	allowedErr := fms.checkAllowedDirectory(fileOverview.GetFiles())
	if allowedErr != nil {
		return model.Error, allowedErr
	}

	diffFiles, fileContent, compareErr := fms.DetermineFileActions(
		ctx,
		fms.currentFilesOnDisk,
		ConvertToMapOfFileCache(fileOverview.GetFiles()),
	)

	if compareErr != nil {
		return model.Error, compareErr
	}

	if len(diffFiles) == 0 {
		return model.NoChange, nil
	}

	fms.rollbackFileContents = fileContent
	fms.fileActions = diffFiles

	fileErr := fms.executeFileActions(ctx)
	if fileErr != nil {
		return model.RollbackRequired, fileErr
	}
	fileOverviewFiles := files.ConvertToMapOfFiles(fileOverview.GetFiles())
	// Update map of current files on disk
	manifestFileErr := fms.UpdateCurrentFilesOnDisk(ctx, fileOverviewFiles, false)
	if manifestFileErr != nil {
		return model.RollbackRequired, manifestFileErr
	}

	return model.OK, nil
}

func (fms *FileManagerService) ClearCache() {
	clear(fms.rollbackFileContents)
	clear(fms.fileActions)
	clear(fms.previousManifestFiles)
}

// nolint:revive,cyclop
func (fms *FileManagerService) Rollback(ctx context.Context, instanceID string) error {
	slog.InfoContext(ctx, "Rolling back config for instance", "instance_id", instanceID)

	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case model.Add:
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", fileAction.File.GetFileMeta().GetName(), err)
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			delete(fms.currentFilesOnDisk, fileAction.File.GetFileMeta().GetName())

			continue
		case model.Delete, model.Update:
			content := fms.rollbackFileContents[fileAction.File.GetFileMeta().GetName()]
			err := fms.fileOperator.Write(ctx, content, fileAction.File.GetFileMeta())
			if err != nil {
				return err
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			fileAction.File.GetFileMeta().Hash = files.GenerateHash(content)
			fms.currentFilesOnDisk[fileAction.File.GetFileMeta().GetName()] = fileAction.File
		case model.Unchanged:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented")
		}
	}

	manifestFileErr := fms.fileOperator.WriteManifestFile(fms.previousManifestFiles,
		fms.agentConfig.ManifestDir, fms.manifestFilePath)
	if manifestFileErr != nil {
		return manifestFileErr
	}

	return nil
}

func (fms *FileManagerService) ConfigUpdate(ctx context.Context,
	nginxConfigContext *model.NginxConfigContext,
) {
	updateError := fms.UpdateCurrentFilesOnDisk(
		ctx,
		files.ConvertToMapOfFiles(nginxConfigContext.Files),
		true,
	)
	if updateError != nil {
		slog.ErrorContext(ctx, "Unable to update current files on disk", "error", updateError)
	}

	slog.InfoContext(ctx, "Updating overview after nginx config update")
	err := fms.fileServiceOperator.UpdateOverview(ctx, nginxConfigContext.InstanceID, nginxConfigContext.Files, 0)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to update file overview",
			"instance_id", nginxConfigContext.InstanceID,
			"error", err,
		)
	}
}

func (fms *FileManagerService) ConfigUpload(ctx context.Context, configUploadRequest *mpi.ConfigUploadRequest) error {
	var updatingFilesError error

	for _, file := range configUploadRequest.GetOverview().GetFiles() {
		err := fms.fileServiceOperator.UpdateFile(
			ctx,
			configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			file,
		)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"Failed to update file",
				"instance_id", configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
				"file_name", file.GetFileMeta().GetName(),
				"error", err,
			)

			updatingFilesError = errors.Join(updatingFilesError, err)
		}
	}

	return updatingFilesError
}

// DetermineFileActions compares two sets of files to determine the file action for each file. Returns a map of files
// that have changed and a map of the contents for each updated and deleted file. Key to both maps is file path
// nolint: revive,cyclop,gocognit
func (fms *FileManagerService) DetermineFileActions(
	ctx context.Context,
	currentFiles map[string]*mpi.File,
	modifiedFiles map[string]*model.FileCache,
) (
	map[string]*model.FileCache,
	map[string][]byte,
	error,
) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	fileDiff := make(map[string]*model.FileCache) // Files that have changed, key is file name
	fileContents := make(map[string][]byte)       // contents of the file, key is file name

	_, filesMap, manifestFileErr := fms.manifestFile()

	if manifestFileErr != nil {
		if errors.Is(manifestFileErr, os.ErrNotExist) {
			filesMap = currentFiles
		} else {
			return nil, nil, manifestFileErr
		}
	}

	// if file is in manifestFiles but not in modified files, file has been deleted
	// copy contents, set file action
	for fileName, manifestFile := range filesMap {
		_, exists := modifiedFiles[fileName]

		if !exists {
			// Read file contents before marking it deleted
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				if errors.Is(readErr, os.ErrNotExist) {
					slog.DebugContext(ctx, "Unable to backup file contents since file does not exist", "file", fileName)
					continue
				} else {
					return nil, nil, fmt.Errorf("error reading file %s: %w", fileName, readErr)
				}
			}
			fileContents[fileName] = fileContent

			fileDiff[fileName] = &model.FileCache{
				File:   manifestFile,
				Action: model.Delete,
			}
		}
	}

	for _, modifiedFile := range modifiedFiles {
		fileName := modifiedFile.File.GetFileMeta().GetName()
		currentFile, ok := filesMap[modifiedFile.File.GetFileMeta().GetName()]
		// default to unchanged action
		modifiedFile.Action = model.Unchanged

		// if file is unmanaged, action is set to unchanged so file is skipped when performing actions
		if modifiedFile.File.GetUnmanaged() {
			continue
		}
		// if file doesn't exist in the current files, file has been added
		// set file action
		if _, statErr := os.Stat(modifiedFile.File.GetFileMeta().GetName()); errors.Is(statErr, os.ErrNotExist) {
			modifiedFile.Action = model.Add
			fileDiff[modifiedFile.File.GetFileMeta().GetName()] = modifiedFile

			continue
			// if file currently exists and file hash is different, file has been updated
			// copy contents, set file action
		} else if ok && modifiedFile.File.GetFileMeta().GetHash() != currentFile.GetFileMeta().GetHash() {
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			modifiedFile.Action = model.Update
			fileContents[fileName] = fileContent
			fileDiff[modifiedFile.File.GetFileMeta().GetName()] = modifiedFile
		}
	}

	return fileDiff, fileContents, nil
}

// UpdateCurrentFilesOnDisk updates the FileManagerService currentFilesOnDisk slice which contains the files
// currently on disk
func (fms *FileManagerService) UpdateCurrentFilesOnDisk(
	ctx context.Context,
	currentFiles map[string]*mpi.File,
	referenced bool,
) error {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	clear(fms.currentFilesOnDisk)

	for _, currentFile := range currentFiles {
		fms.currentFilesOnDisk[currentFile.GetFileMeta().GetName()] = currentFile
	}

	err := fms.UpdateManifestFile(currentFiles, referenced)
	if err != nil {
		return fmt.Errorf("failed to update manifest file: %w", err)
	}

	return nil
}

// seems to be a control flag, avoid control coupling
// nolint: revive
func (fms *FileManagerService) UpdateManifestFile(currentFiles map[string]*mpi.File, referenced bool) (err error) {
	slog.Debug("Updating manifest file", "current_files", currentFiles, "referenced", referenced)
	currentManifestFiles, _, readError := fms.manifestFile()
	fms.previousManifestFiles = currentManifestFiles
	if readError != nil && !errors.Is(readError, os.ErrNotExist) {
		slog.Debug("Error reading manifest file", "current_manifest_files",
			currentManifestFiles, "updated_files", currentFiles, "referenced", referenced)

		return fmt.Errorf("unable to read manifest file: %w", readError)
	}

	updatedFiles := make(map[string]*model.ManifestFile)

	manifestFiles := fms.convertToManifestFileMap(currentFiles, referenced)
	// During a config apply every file is set to unreferenced
	// When a new NGINX config context is detected
	// we update the files in the manifest by setting the referenced bool to true
	if currentManifestFiles != nil && referenced {
		for _, currentManifestFile := range currentManifestFiles {
			// if file from manifest file is unreferenced add it to updatedFiles map
			if !currentManifestFile.ManifestFileMeta.Referenced {
				updatedFiles[currentManifestFile.ManifestFileMeta.Name] = currentManifestFile
			}
		}
		for manifestFileName, manifestFile := range manifestFiles {
			updatedFiles[manifestFileName] = manifestFile
		}
	} else {
		updatedFiles = manifestFiles
	}

	return fms.fileOperator.WriteManifestFile(updatedFiles, fms.agentConfig.ManifestDir, fms.manifestFilePath)
}

func (fms *FileManagerService) manifestFile() (map[string]*model.ManifestFile, map[string]*mpi.File, error) {
	if _, err := os.Stat(fms.manifestFilePath); err != nil {
		return nil, nil, err
	}

	fms.manifestLock.Lock()
	defer fms.manifestLock.Unlock()
	file, err := os.ReadFile(fms.manifestFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifestFiles map[string]*model.ManifestFile

	err = json.Unmarshal(file, &manifestFiles)
	if err != nil {
		if len(file) == 0 {
			return nil, nil, fmt.Errorf("manifest file is empty: %w", err)
		}

		return nil, nil, fmt.Errorf("failed to parse manifest file: %w", err)
	}

	fileMap := fms.convertToFileMap(manifestFiles)

	return manifestFiles, fileMap, nil
}

func (fms *FileManagerService) executeFileActions(ctx context.Context) error {
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case model.Delete:
			slog.DebugContext(ctx, "File action, deleting file", "file", fileAction.File.GetFileMeta().GetName())
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w",
					fileAction.File.GetFileMeta().GetName(), err)
			}

			continue
		case model.Add, model.Update:
			slog.DebugContext(ctx, "File action, add or update file", "file", fileAction.File.GetFileMeta().GetName())
			updateErr := fms.fileUpdate(ctx, fileAction.File)
			if updateErr != nil {
				return updateErr
			}
		case model.Unchanged:
			slog.DebugContext(ctx, "File unchanged")
		}
	}

	return nil
}

func (fms *FileManagerService) fileUpdate(ctx context.Context, file *mpi.File) error {
	if file.GetFileMeta().GetSize() <= int64(fms.agentConfig.Client.Grpc.MaxFileSize) {
		return fms.fileServiceOperator.File(ctx, file, fms.fileActions)
	}

	return fms.fileServiceOperator.ChunkedFile(ctx, file)
}

func (fms *FileManagerService) checkAllowedDirectory(checkFiles []*mpi.File) error {
	for _, file := range checkFiles {
		allowed := fms.agentConfig.IsDirectoryAllowed(file.GetFileMeta().GetName())
		if !allowed {
			return fmt.Errorf("file not in allowed directories %s", file.GetFileMeta().GetName())
		}
	}

	return nil
}

func (fms *FileManagerService) convertToManifestFileMap(
	currentFiles map[string]*mpi.File,
	referenced bool,
) map[string]*model.ManifestFile {
	manifestFileMap := make(map[string]*model.ManifestFile)

	for name, currentFile := range currentFiles {
		if currentFile == nil || currentFile.GetFileMeta() == nil {
			continue
		}
		manifestFile := fms.convertToManifestFile(currentFile, referenced)
		manifestFileMap[name] = manifestFile
	}

	return manifestFileMap
}

func (fms *FileManagerService) convertToManifestFile(file *mpi.File, referenced bool) *model.ManifestFile {
	return &model.ManifestFile{
		ManifestFileMeta: &model.ManifestFileMeta{
			Name:       file.GetFileMeta().GetName(),
			Size:       file.GetFileMeta().GetSize(),
			Hash:       file.GetFileMeta().GetHash(),
			Referenced: referenced,
		},
	}
}

func (fms *FileManagerService) convertToFileMap(manifestFiles map[string]*model.ManifestFile) map[string]*mpi.File {
	currentFileMap := make(map[string]*mpi.File)
	for name, manifestFile := range manifestFiles {
		currentFile := fms.convertToFile(manifestFile)
		currentFileMap[name] = currentFile
	}

	return currentFileMap
}

func (fms *FileManagerService) convertToFile(manifestFile *model.ManifestFile) *mpi.File {
	return &mpi.File{
		FileMeta: &mpi.FileMeta{
			Name: manifestFile.ManifestFileMeta.Name,
			Hash: manifestFile.ManifestFileMeta.Hash,
			Size: manifestFile.ManifestFileMeta.Size,
		},
	}
}

// ConvertToMapOfFiles converts a list of files to a map of file caches (file and action) with the file name as the key
func ConvertToMapOfFileCache(convertFiles []*mpi.File) map[string]*model.FileCache {
	filesMap := make(map[string]*model.FileCache)
	for _, convertFile := range convertFiles {
		filesMap[convertFile.GetFileMeta().GetName()] = &model.FileCache{
			File: convertFile,
		}
	}

	return filesMap
}
