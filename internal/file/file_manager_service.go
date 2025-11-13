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
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
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

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileServiceOperatorInterface

const (
	maxAttempts = 5
	dirPerm     = 0o755
	filePerm    = 0o600
	executePerm = 0o111
)

type DownloadHeader struct {
	ETag         string
	LastModified string
}

type (
	fileOperator interface {
		Write(ctx context.Context, fileContent []byte, fileName, filePermissions string) error
		CreateFileDirectories(ctx context.Context, fileName string) error
		WriteChunkedFile(
			ctx context.Context,
			fileName, filePermissions string,
			header *mpi.FileDataChunkHeader,
			stream grpc.ServerStreamingClient[mpi.FileDataChunk],
		) error
		ReadChunk(
			ctx context.Context,
			chunkSize uint32,
			reader *bufio.Reader,
			chunkID uint32,
		) (mpi.FileDataChunk_Content, error)
		WriteManifestFile(ctx context.Context, updatedFiles map[string]*model.ManifestFile,
			manifestDir, manifestPath string) (writeError error)
		MoveFile(ctx context.Context, sourcePath, destPath string) error
	}

	fileServiceOperatorInterface interface {
		File(ctx context.Context, file *mpi.File, tempFilePath, expectedHash string) error
		UpdateOverview(ctx context.Context, instanceID string, filesToUpdate []*mpi.File, configPath string,
			iteration int) error
		ChunkedFile(ctx context.Context, file *mpi.File, tempFilePath, expectedHash string) error
		IsConnected() bool
		UpdateFile(
			ctx context.Context,
			instanceID string,
			fileToUpdate *mpi.File,
		) error
		SetIsConnected(isConnected bool)
		RenameFile(ctx context.Context, fileName, tempDir string) error
		ValidateFileHash(ctx context.Context, fileName, expectedHash string) error
		UpdateClient(ctx context.Context, fileServiceClient mpi.FileServiceClient)
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
		) (map[string]*model.FileCache, error)
		IsConnected() bool
		SetIsConnected(isConnected bool)
		ResetClient(ctx context.Context, fileServiceClient mpi.FileServiceClient)
	}
)

type FileManagerService struct {
	manifestLock        *sync.RWMutex
	agentConfig         *config.Config
	fileOperator        fileOperator
	fileServiceOperator fileServiceOperatorInterface
	// map of files and the actions performed on them during config apply
	fileActions map[string]*model.FileCache // key is file path
	// map of the files currently on disk, used to determine the file action during config apply
	currentFilesOnDisk    map[string]*mpi.File // key is file path
	previousManifestFiles map[string]*model.ManifestFile
	externalFileHeaders   map[string]DownloadHeader
	manifestFilePath      string
	rollbackManifest      bool
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
		currentFilesOnDisk:    make(map[string]*mpi.File),
		previousManifestFiles: make(map[string]*model.ManifestFile),
		externalFileHeaders:   make(map[string]DownloadHeader),
		rollbackManifest:      true,
		manifestFilePath:      agentConfig.LibDir + "/manifest.json",
		manifestLock:          manifestLock,
	}
}

func (fms *FileManagerService) ResetClient(ctx context.Context, fileServiceClient mpi.FileServiceClient) {
	fms.fileServiceOperator.UpdateClient(ctx, fileServiceClient)
	slog.DebugContext(ctx, "File manager service reset client successfully")
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
	fms.rollbackManifest = true
	fileOverview := configApplyRequest.GetOverview()

	if fileOverview == nil {
		return model.Error, errors.New("fileOverview is nil")
	}

	// check if any file in request is outside the allowed directories
	allowedErr := fms.checkAllowedDirectory(fileOverview.GetFiles())
	if allowedErr != nil {
		return model.Error, allowedErr
	}

	permissionErr := fms.validateAndUpdateFilePermissions(ctx, fileOverview.GetFiles())
	if permissionErr != nil {
		return model.RollbackRequired, permissionErr
	}

	diffFiles, compareErr := fms.DetermineFileActions(
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

	fms.fileActions = diffFiles

	slog.DebugContext(ctx, "Executing config apply file actions", "actions", diffFiles)

	rollbackTempFilesErr := fms.backupFiles(ctx)
	if rollbackTempFilesErr != nil {
		return model.Error, rollbackTempFilesErr
	}

	fileErr := fms.executeFileActions(ctx)
	if fileErr != nil {
		fms.rollbackManifest = false
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
	slog.Debug("Clearing cache and backup files")

	for _, fileAction := range fms.fileActions {
		if fileAction.Action == model.Update || fileAction.Action == model.Delete {
			tempFilePath := tempBackupFilePath(fileAction.File.GetFileMeta().GetName())
			if err := os.Remove(tempFilePath); err != nil && !os.IsNotExist(err) {
				slog.Warn("Unable to delete backup file",
					"file", fileAction.File.GetFileMeta().GetName(),
					"error", err,
				)
			}
		}
	}

	clear(fms.fileActions)
	clear(fms.previousManifestFiles)
}

//nolint:revive,cyclop // cognitive-complexity of 13 max is 12, loop is needed cant be broken up
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
		case model.Delete, model.Update, model.ExternalFile:
			content, err := fms.restoreFiles(fileAction)
			if err != nil {
				return err
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			fileAction.File.FileMeta.Hash = files.GenerateHash(content)
			fms.currentFilesOnDisk[fileAction.File.GetFileMeta().GetName()] = fileAction.File
		case model.Unchanged:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented")
		}
	}

	if fms.rollbackManifest {
		slog.DebugContext(ctx, "Rolling back manifest file", "manifest_previous", fms.previousManifestFiles)
		manifestFileErr := fms.fileOperator.WriteManifestFile(
			ctx, fms.previousManifestFiles, fms.agentConfig.LibDir, fms.manifestFilePath,
		)
		if manifestFileErr != nil {
			return manifestFileErr
		}
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
	err := fms.fileServiceOperator.UpdateOverview(ctx, nginxConfigContext.InstanceID,
		nginxConfigContext.Files, nginxConfigContext.ConfigPath, 0)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to update file overview",
			"instance_id", nginxConfigContext.InstanceID,
			"error", err,
		)
	}
	slog.InfoContext(ctx, "Finished updating file overview")
}

func (fms *FileManagerService) ConfigUpload(ctx context.Context, configUploadRequest *mpi.ConfigUploadRequest) error {
	uploadFiles := configUploadRequest.GetOverview().GetFiles()
	if len(uploadFiles) == 0 {
		return nil
	}

	errGroup, errGroupCtx := errgroup.WithContext(ctx)
	errGroup.SetLimit(fms.agentConfig.Client.Grpc.MaxParallelFileOperations)

	for _, file := range uploadFiles {
		errGroup.Go(func() error {
			err := fms.fileServiceOperator.UpdateFile(
				errGroupCtx,
				configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
				file,
			)
			if err != nil {
				slog.ErrorContext(
					errGroupCtx,
					"Failed to update file",
					"instance_id", configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
					"file_name", file.GetFileMeta().GetName(),
					"error", err,
				)
			}

			return err
		})
	}

	return errGroup.Wait()
}

// DetermineFileActions compares two sets of files to determine the file action for each file. Returns a map of files
// that have changed and a map of the contents for each updated and deleted file. Key to both maps is file path
//
//nolint:gocognit,revive,cyclop // cognitive complexity is 23
func (fms *FileManagerService) DetermineFileActions(
	ctx context.Context,
	currentFiles map[string]*mpi.File,
	modifiedFiles map[string]*model.FileCache,
) (
	map[string]*model.FileCache,
	error,
) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	fileDiff := make(map[string]*model.FileCache) // Files that have changed, key is file name

	_, filesMap, manifestFileErr := fms.manifestFile()

	if manifestFileErr != nil {
		if !errors.Is(manifestFileErr, os.ErrNotExist) {
			return nil, manifestFileErr
		}
		filesMap = currentFiles
	}

	// if file is in manifestFiles but not in modified files, file has been deleted
	// copy contents, set file action
	for fileName, manifestFile := range filesMap {
		_, existsInReq := modifiedFiles[fileName]

		// allowed directories may have been updated since manifest file was written
		// if file is outside allowed directories skip deletion and return error
		if !fms.agentConfig.IsDirectoryAllowed(fileName) {
			return nil, fmt.Errorf("error deleting file %s: file not in allowed directories", fileName)
		}

		// if file is unmanaged skip deletion
		if manifestFile.GetUnmanaged() {
			slog.DebugContext(ctx, "Skipping unmanaged file deletion", "file_name", fileName)
			continue
		}

		// if file doesn't exist on disk skip deletion
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			slog.DebugContext(ctx, "File already deleted, skipping", "file", fileName)
			continue
		}

		// go ahead and delete the file
		if !existsInReq {
			fileDiff[fileName] = &model.FileCache{
				File:   manifestFile,
				Action: model.Delete,
			}
		}
	}

	for _, modifiedFile := range modifiedFiles {
		fileName := modifiedFile.File.GetFileMeta().GetName()
		currentFile, ok := filesMap[fileName]
		modifiedFile.Action = model.Unchanged

		// If file is unmanaged, action is set to unchanged so file is skipped when performing actions.
		if modifiedFile.File.GetUnmanaged() {
			slog.DebugContext(ctx, "Skipping unmanaged file updates", "file_name", fileName)
			continue
		}

		// If it's external, we DON'T care about disk state or hashes here.
		// We tag it as ExternalFile and let the downloader handle the rest.
		if modifiedFile.File.GetExternalDataSource() != nil || (ok && currentFile.GetExternalDataSource() != nil) {
			slog.DebugContext(ctx, "External file detected - flagging for fetch", "file_name", fileName)
			modifiedFile.Action = model.ExternalFile
			fileDiff[fileName] = modifiedFile
			continue
		}

		// If file currently exists on disk, is being tracked in manifest and file hash is different.
		// Treat it as a file update.
		if ok && modifiedFile.File.GetFileMeta().GetHash() != currentFile.GetFileMeta().GetHash() {
			slog.DebugContext(ctx, "Tracked file requires updating", "file_name", fileName)
			modifiedFile.Action = model.Update
			fileDiff[fileName] = modifiedFile

			continue
		}

		fileStats, statErr := os.Stat(fileName)

		// If file doesn't exist on disk.
		// Treat it as adding a new file.
		if errors.Is(statErr, os.ErrNotExist) {
			slog.DebugContext(ctx, "New untracked file needs to be created", "file_name", fileName)
			modifiedFile.Action = model.Add
			fileDiff[fileName] = modifiedFile

			continue
		}

		// If there is an error other than not existing, return that error.
		if statErr != nil {
			return nil, fmt.Errorf("unable to stat file %s: %w", fileName, statErr)
		}

		// If there is a directory with the same name, return an error.
		if fileStats.IsDir() {
			return nil, fmt.Errorf(
				"unable to create file %s since a directory with the same name already exists",
				fileName,
			)
		}

		// If file already exists on disk but is not being tracked in manifest and the file hash is different.
		// Treat it as a file update.
		metadataOfFileOnDisk, err := files.FileMeta(fileName)
		if err != nil {
			return nil, fmt.Errorf("unable to get file metadata for %s: %w", fileName, err)
		}

		if metadataOfFileOnDisk.GetHash() != modifiedFile.File.GetFileMeta().GetHash() {
			slog.DebugContext(ctx, "Untracked file requires updating", "file_name", fileName)
			modifiedFile.Action = model.Update
			fileDiff[fileName] = modifiedFile
		}
	}

	return fileDiff, nil
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

	err := fms.UpdateManifestFile(ctx, currentFiles, referenced)
	if err != nil {
		return fmt.Errorf("failed to update manifest file: %w", err)
	}

	return nil
}

// seems to be a control flag, avoid control coupling
//
//nolint:revive // referenced is a required flag
func (fms *FileManagerService) UpdateManifestFile(ctx context.Context,
	currentFiles map[string]*mpi.File, referenced bool,
) (err error) {
	slog.DebugContext(ctx, "Updating manifest file", "current_files", currentFiles, "referenced", referenced)
	currentManifestFiles, _, readError := fms.manifestFile()

	// When agent is first started the manifest is updated when an NGINX instance is found, but the manifest file
	// will be empty leading to previousManifestFiles being empty. This was causing issues if the first config
	// apply failed leading to the manifest file being rolled back to an empty file.
	// If the currentManifestFiles is empty then we can assume the Agent has just started and this is the first
	// write of the Manifest file, so set previousManifestFiles to be the currentFiles.
	if len(currentManifestFiles) == 0 {
		currentManifestFiles = fms.convertToManifestFileMap(currentFiles, referenced)
	}

	fms.previousManifestFiles = currentManifestFiles
	if readError != nil && !errors.Is(readError, os.ErrNotExist) {
		slog.DebugContext(ctx, "Error reading manifest file", "current_manifest_files",
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

	return fms.fileOperator.WriteManifestFile(ctx, updatedFiles, fms.agentConfig.LibDir, fms.manifestFilePath)
}

func (fms *FileManagerService) backupFiles(ctx context.Context) error {
	for _, file := range fms.fileActions {
		if file.Action == model.Add || file.Action == model.Unchanged {
			continue
		}

		filePath := file.File.GetFileMeta().GetName()

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			slog.DebugContext(ctx, "Unable to backup file content since file does not exist",
				"file", filePath)

			continue
		}

		tempFilePath := tempBackupFilePath(filePath)
		slog.DebugContext(ctx, "Attempting to backup file content since file exists", "temp_path", tempFilePath)

		moveErr := fms.fileOperator.MoveFile(ctx, filePath, tempFilePath)

		if moveErr != nil {
			return moveErr
		}
	}

	return nil
}

func (fms *FileManagerService) restoreFiles(fileAction *model.FileCache) ([]byte, error) {
	fileMeta := fileAction.File.GetFileMeta()
	fileName := fileMeta.GetName()

	tempFilePath := tempBackupFilePath(fileName)

	// Create parent directories for the target file if they don't exist
	if err := os.MkdirAll(filepath.Dir(fileName), dirPerm); err != nil {
		return nil, fmt.Errorf("failed to create directories for %s: %w", fileName, err)
	}

	moveErr := os.Rename(tempFilePath, fileName)
	if moveErr != nil {
		return nil, fmt.Errorf("failed to rename file, %s to %s: %w", tempFilePath, fileName, moveErr)
	}

	content, readErr := os.ReadFile(fileMeta.GetName())
	if readErr != nil {
		return nil, fmt.Errorf("error reading file, unable to generate hash: %s error: %w",
			fileMeta.GetName(), readErr)
	}

	return content, nil
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

func (fms *FileManagerService) executeFileActions(ctx context.Context) (actionError error) {
	// Download files to temporary location
	downloadError := fms.downloadUpdatedFilesToTempLocation(ctx)
	if downloadError != nil {
		return downloadError
	}

	// Remove temp files if there is a failure moving or deleting files
	actionError = fms.moveOrDeleteFiles(ctx, actionError)
	if actionError != nil {
		fms.deleteTempFiles(ctx)
	}

	return actionError
}

func (fms *FileManagerService) downloadUpdatedFilesToTempLocation(ctx context.Context) (updateError error) {
	var downloadFiles []*model.FileCache
	for _, fileAction := range fms.fileActions {
		if fileAction.Action == model.Add || fileAction.Action == model.Update || fileAction.Action == model.ExternalFile {
			downloadFiles = append(downloadFiles, fileAction)
		}
	}

	if len(downloadFiles) == 0 {
		slog.DebugContext(ctx, "No updated files to download")
		return nil
	}

	errGroup, errGroupCtx := errgroup.WithContext(ctx)
	errGroup.SetLimit(fms.agentConfig.Client.Grpc.MaxParallelFileOperations)

	for _, fileAction := range downloadFiles {
		errGroup.Go(func() error {
			tempFilePath := tempFilePath(fileAction.File.GetFileMeta().GetName())

			switch fileAction.Action {
			case model.ExternalFile:
				return fms.downloadExternalFile(errGroupCtx, fileAction, tempFilePath)
			case model.Add, model.Update:
				slog.DebugContext(
					errGroupCtx,
					"Downloading file to temp location",
					"file", tempFilePath,
				)

				return fms.fileUpdate(errGroupCtx, fileAction.File, tempFilePath)
			case model.Delete, model.Unchanged: // had to add for linter
				return nil
			default:
				return nil
			}
		})
	}

	return errGroup.Wait()
}

//nolint:revive // cognitive-complexity of 14 max is 12, loop is needed cant be broken up
func (fms *FileManagerService) moveOrDeleteFiles(ctx context.Context, actionError error) error {
actionsLoop:
	for _, fileAction := range fms.fileActions {
		var err error
		fileMeta := fileAction.File.GetFileMeta()
		tempFilePath := tempFilePath(fileMeta.GetName())
		switch fileAction.Action {
		case model.Delete:
			slog.DebugContext(ctx, "Deleting file", "file", fileMeta.GetName())
			if err = os.Remove(fileMeta.GetName()); err != nil && !os.IsNotExist(err) {
				actionError = fmt.Errorf("error deleting file: %s error: %w",
					fileMeta.GetName(), err)

				break actionsLoop
			}

			continue
		case model.Add, model.Update:
			err = fms.fileServiceOperator.RenameFile(ctx, tempFilePath, fileMeta.GetName())
			if err != nil {
				actionError = err
				break actionsLoop
			}
			err = fms.fileServiceOperator.ValidateFileHash(ctx, fileMeta.GetName(), fileMeta.GetHash())
		case model.ExternalFile:
			err = fms.fileServiceOperator.RenameFile(ctx, tempFilePath, fileMeta.GetName())
		case model.Unchanged:
			slog.DebugContext(ctx, "File unchanged")
		}
		if err != nil {
			actionError = err
			break actionsLoop
		}
	}

	return actionError
}

func (fms *FileManagerService) deleteTempFiles(ctx context.Context) {
	for _, fileAction := range fms.fileActions {
		if fileAction.Action == model.Add || fileAction.Action == model.Update {
			tempFilePath := tempFilePath(fileAction.File.GetFileMeta().GetName())
			if err := os.Remove(tempFilePath); err != nil && !os.IsNotExist(err) {
				slog.ErrorContext(
					ctx, "Error deleting temp file",
					"file", fileAction.File.GetFileMeta().GetName(),
					"error", err,
				)
			}
		}
	}
}

func (fms *FileManagerService) fileUpdate(ctx context.Context, file *mpi.File, tempFilePath string) error {
	expectedHash := fms.fileActions[file.GetFileMeta().GetName()].File.GetFileMeta().GetHash()

	if file.GetFileMeta().GetSize() <= int64(fms.agentConfig.Client.Grpc.MaxFileSize) {
		return fms.fileServiceOperator.File(ctx, file, tempFilePath, expectedHash)
	}

	return fms.fileServiceOperator.ChunkedFile(ctx, file, tempFilePath, expectedHash)
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

func (fms *FileManagerService) validateAndUpdateFilePermissions(ctx context.Context, fileList []*mpi.File) error {
	for _, file := range fileList {
		if fms.areExecuteFilePermissionsSet(file) {
			resetErr := fms.removeExecuteFilePermissions(ctx, file)
			if resetErr != nil {
				return fmt.Errorf("failed to reset permissions for %s: %w", file.GetFileMeta().GetName(), resetErr)
			}
		}
	}

	return nil
}

func (fms *FileManagerService) areExecuteFilePermissionsSet(file *mpi.File) bool {
	filePermission := file.GetFileMeta().GetPermissions()

	permissionOctal, err := strconv.ParseUint(filePermission, 8, 32)
	if err != nil || len(filePermission) != 4 {
		return false
	}

	return permissionOctal&executePerm > 0
}

func (fms *FileManagerService) removeExecuteFilePermissions(ctx context.Context, file *mpi.File) error {
	filePermission := file.GetFileMeta().GetPermissions()

	permissionOctal, err := strconv.ParseUint(filePermission, 8, 32)
	if err != nil {
		return fmt.Errorf("falied to parse file permissions: %w", err)
	}

	permissionOctal &^= executePerm

	newPermission := "0" + strconv.FormatUint(permissionOctal, 8)
	file.FileMeta.Permissions = newPermission

	slog.DebugContext(ctx, "Permissions have been changed", "file", file.GetFileMeta().GetName(),
		"old_permissions", filePermission, "new_permissions", newPermission)

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
			Unmanaged:  file.GetUnmanaged(),
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
		Unmanaged: manifestFile.ManifestFileMeta.Unmanaged,
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

func tempFilePath(fileName string) string {
	tempFileName := "." + filepath.Base(fileName) + ".agent.tmp"
	return filepath.Join(filepath.Dir(fileName), tempFileName)
}

func tempBackupFilePath(fileName string) string {
	tempFileName := "." + filepath.Base(fileName) + ".agent.backup"
	return filepath.Join(filepath.Dir(fileName), tempFileName)
}

func (fms *FileManagerService) downloadExternalFile(ctx context.Context, fileAction *model.FileCache,
	filePath string,
) error {
	location := fileAction.File.GetExternalDataSource().GetLocation()
	permission := fileAction.File.GetFileMeta().GetPermissions()

	slog.InfoContext(ctx, "Downloading external file from", "location", location)

	var contentToWrite []byte
	var downloadErr, updateError error
	var headers DownloadHeader

	contentToWrite, headers, downloadErr = fms.downloadFileContent(ctx, fileAction.File)

	if downloadErr != nil {
		updateError = fmt.Errorf("failed to download file %s from %s: %w",
			fileAction.File.GetFileMeta().GetName(), location, downloadErr)

		return updateError
	}

	if contentToWrite == nil {
		slog.DebugContext(ctx, "External file unchanged (304), skipping disk write.",
			"file", fileAction.File.GetFileMeta().GetName())

		fileAction.Action = model.Unchanged

		return nil
	}

	fileName := fileAction.File.GetFileMeta().GetName()
	fms.externalFileHeaders[fileName] = headers

	writeErr := fms.fileOperator.Write(
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

// downloadFileContent performs an HTTP GET request to the given URL and returns the file content as a byte slice.
func (fms *FileManagerService) downloadFileContent(
	ctx context.Context,
	file *mpi.File,
) (content []byte, headers DownloadHeader, err error) {
	fileName := file.GetFileMeta().GetName()
	downloadURL := file.GetExternalDataSource().GetLocation()
	externalConfig := fms.agentConfig.ExternalDataSource

	if !isDomainAllowed(downloadURL, externalConfig.AllowedDomains) {
		return nil, DownloadHeader{}, fmt.Errorf("download URL %s is not in the allowed domains list", downloadURL)
	}

	httpClient, err := fms.setupHTTPClient(ctx, externalConfig.ProxyURL.URL)
	if err != nil {
		return nil, DownloadHeader{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, DownloadHeader{}, fmt.Errorf("failed to create request for %s: %w", downloadURL, err)
	}

	if externalConfig.ProxyURL.URL != "" {
		fms.addConditionalHeaders(ctx, req, fileName)
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
		return nil, DownloadHeader{}, nil
	default:
		return nil, DownloadHeader{}, fmt.Errorf("download failed with status code %d", resp.StatusCode)
	}

	reader := io.Reader(resp.Body)
	if fms.agentConfig.ExternalDataSource.MaxBytes > 0 {
		reader = io.LimitReader(resp.Body, fms.agentConfig.ExternalDataSource.MaxBytes)
	}

	content, err = io.ReadAll(reader)
	if err != nil {
		return nil, DownloadHeader{}, fmt.Errorf("failed to read content from response body: %w", err)
	}

	slog.InfoContext(ctx, "Successfully downloaded file content", "file_name", fileName, "size", len(content))

	return content, headers, nil
}

func isDomainAllowed(downloadURL string, allowedDomains []string) bool {
	u, err := url.Parse(downloadURL)
	if err != nil {
		slog.Debug("Failed to parse download URL for domain check", "url", downloadURL, "error", err)
		return false
	}

	hostname := u.Hostname()
	if hostname == "" {
		return false
	}

	for _, domain := range allowedDomains {
		if domain == "" {
			continue
		}

		if domain == hostname || isMatchesWildcardDomain(hostname, domain) {
			return true
		}
	}

	return false
}

func (fms *FileManagerService) setupHTTPClient(ctx context.Context, proxyURLString string) (*http.Client, error) {
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
		Timeout:   fms.agentConfig.Client.FileDownloadTimeout,
	}

	return httpClient, nil
}

func (fms *FileManagerService) addConditionalHeaders(ctx context.Context, req *http.Request, fileName string) {
	slog.DebugContext(ctx, "Proxy configured; adding headers to GET request.")

	manifestFiles, _, manifestFileErr := fms.manifestFile()

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

func isMatchesWildcardDomain(hostname, pattern string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}

	baseDomain := pattern[2:]
	if strings.HasSuffix(hostname, baseDomain) {
		// Check to ensure it's a true subdomain match (e.g., must have a '.'
		// before baseDomain unless it IS the baseDomain)
		// This handles cases like preventing 'foo.com' matching '*.oo.com'
		if hostname == baseDomain || hostname[len(hostname)-len(baseDomain)-1] == '.' {
			return true
		}
	}

	return false
}
