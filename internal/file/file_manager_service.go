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
	"path"
	"path/filepath"
	"strconv"
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
	executePerm = 0o111
)

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
		RenameFile(ctx context.Context, hash, fileName, tempDir string) error
		UpdateClient(ctx context.Context, fileServiceClient mpi.FileServiceClient)
	}

	fileManagerServiceInterface interface {
		ConfigApply(ctx context.Context, configApplyRequest *mpi.ConfigApplyRequest) (writeStatus model.WriteStatus,
			err error)
		Rollback(ctx context.Context, instanceID string) error
		ClearCache()
		SetConfigPath(configPath string)
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
	manifestFilePath      string
	tempConfigDir         string
	tempRollbackDir       string
	configPath            string
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
		rollbackManifest:      true,
		manifestFilePath:      agentConfig.LibDir + "/manifest.json",
		configPath:            "/etc/nginx/",
		manifestLock:          manifestLock,
	}
}

func (fms *FileManagerService) SetConfigPath(configPath string) {
	fms.configPath = filepath.Dir(configPath)
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
	var configTempErr error
	var rollbackTempErr error

	fms.rollbackManifest = true
	fileOverview := configApplyRequest.GetOverview()

	if fileOverview == nil {
		return model.Error, errors.New("fileOverview is nil")
	}

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

	fms.tempConfigDir, configTempErr = fms.createTempConfigDirectory("config")
	if configTempErr != nil {
		return model.Error, configTempErr
	}

	fms.tempRollbackDir, rollbackTempErr = fms.createTempConfigDirectory("rollback")
	if rollbackTempErr != nil {
		return model.Error, rollbackTempErr
	}

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
	slog.Debug("Clearing cache and temp files after config apply")
	clear(fms.fileActions)
	clear(fms.previousManifestFiles)

	configErr := os.RemoveAll(fms.tempConfigDir)
	if configErr != nil {
		slog.Error("Error removing temp config directory", "path", fms.tempConfigDir, "err", configErr)
	}

	rollbackErr := os.RemoveAll(fms.tempRollbackDir)
	if rollbackErr != nil {
		slog.Error("Error removing temp rollback directory", "path", fms.tempRollbackDir, "err", rollbackErr)
	}
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
		case model.Delete, model.Update:
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
		_, exists := modifiedFiles[fileName]

		if !fms.agentConfig.IsDirectoryAllowed(fileName) {
			return nil, fmt.Errorf("error deleting file %s: file not in allowed directories", fileName)
		}

		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			slog.DebugContext(ctx, "File already deleted, skipping", "file", fileName)
			continue
		}

		if !exists {
			fileDiff[fileName] = &model.FileCache{
				File:   manifestFile,
				Action: model.Delete,
			}
		}
	}

	for _, modifiedFile := range modifiedFiles {
		fileName := modifiedFile.File.GetFileMeta().GetName()
		currentFile, ok := filesMap[fileName]
		// default to unchanged action
		modifiedFile.Action = model.Unchanged

		// if file is unmanaged, action is set to unchanged so file is skipped when performing actions
		if modifiedFile.File.GetUnmanaged() {
			continue
		}
		// if file doesn't exist in the current files, file has been added
		// set file action
		if _, statErr := os.Stat(fileName); errors.Is(statErr, os.ErrNotExist) {
			modifiedFile.Action = model.Add
			fileDiff[fileName] = modifiedFile

			continue
			// if file currently exists and file hash is different, file has been updated
			// copy contents, set file action
		} else if ok && modifiedFile.File.GetFileMeta().GetHash() != currentFile.GetFileMeta().GetHash() {
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

		tempFilePath := filepath.Join(fms.tempRollbackDir, filePath)
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

	tempFilePath := filepath.Join(fms.tempRollbackDir, fileName)

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
	downloadError := fms.downloadUpdatedFilesToTempLocation(ctx, fms.tempConfigDir)
	if downloadError != nil {
		return downloadError
	}

	// Remove temp files if there is a failure moving or deleting files
	actionError = fms.moveOrDeleteFiles(ctx, fms.tempConfigDir, actionError)
	if actionError != nil {
		fms.deleteTempFiles(ctx, fms.tempConfigDir)
	}

	return actionError
}

func (fms *FileManagerService) downloadUpdatedFilesToTempLocation(
	ctx context.Context, tempDir string,
) (updateError error) {
	for _, fileAction := range fms.fileActions {
		if fileAction.Action == model.Add || fileAction.Action == model.Update {
			tempFilePath := filepath.Join(tempDir, fileAction.File.GetFileMeta().GetName())

			slog.DebugContext(
				ctx,
				"Downloading file to temp location",
				"file", tempFilePath,
			)

			updateErr := fms.fileUpdate(ctx, fileAction.File, tempFilePath)
			if updateErr != nil {
				updateError = updateErr
				break
			}
		}
	}

	return updateError
}

func (fms *FileManagerService) moveOrDeleteFiles(ctx context.Context, tempDir string, actionError error) error {
actionsLoop:
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case model.Delete:
			slog.DebugContext(ctx, "Deleting file", "file", fileAction.File.GetFileMeta().GetName())
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				actionError = fmt.Errorf("error deleting file: %s error: %w",
					fileAction.File.GetFileMeta().GetName(), err)

				break actionsLoop
			}

			continue
		case model.Add, model.Update:
			fileMeta := fileAction.File.GetFileMeta()
			err := fms.fileServiceOperator.RenameFile(ctx, fileMeta.GetHash(), fileMeta.GetName(), tempDir)
			if err != nil {
				actionError = err

				break actionsLoop
			}
		case model.Unchanged:
			slog.DebugContext(ctx, "File unchanged")
		}
	}

	return actionError
}

func (fms *FileManagerService) deleteTempFiles(ctx context.Context, tempDir string) {
	for _, fileAction := range fms.fileActions {
		if fileAction.Action == model.Add || fileAction.Action == model.Update {
			tempFile := path.Join(tempDir, fileAction.File.GetFileMeta().GetName())
			if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
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

func (fms *FileManagerService) createTempConfigDirectory(pattern string) (string, error) {
	tempDir, tempDirError := os.MkdirTemp(fms.configPath, pattern)
	if tempDirError != nil {
		return "", fmt.Errorf("failed creating temp config directory: %w", tempDirError)
	}

	return tempDir, nil
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
