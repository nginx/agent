/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nginx/agent/sdk/v2/zip"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
	log "github.com/sirupsen/logrus"
)

// ConfigApply facilitates synchronizing the current config against incoming config_apply request. By keeping track
// of the current files, mark them off as they are getting applied, and delete any leftovers that's not in the incoming
// apply payload.
type ConfigApply struct {
	writer       *zip.Writer
	existing     map[string]struct{}
	notExists    map[string]struct{} // set of files that exists in the config provided payload, but not on disk
	notExistDirs map[string]struct{} // set of directories that exists in the config provided payload, but not on disk
}

func NewConfigApplyWithIgnoreDirectives(
	confFile string,
	allowedDirectories map[string]struct{},
	ignoreDirectives []string,
) (*ConfigApply, error) {
	w, err := zip.NewWriter("/")
	if err != nil {
		return nil, err
	}
	b := &ConfigApply{
		writer:       w,
		existing:     make(map[string]struct{}),
		notExists:    make(map[string]struct{}),
		notExistDirs: make(map[string]struct{}),
	}
	if confFile != "" {
		return b, b.mapCurrentFiles(confFile, allowedDirectories, ignoreDirectives)
	}
	return b, nil
}

// to ignore directives use NewConfigApplyWithIgnoreDirectives()
func NewConfigApply(
	confFile string,
	allowedDirectories map[string]struct{},
) (*ConfigApply, error) {
	return NewConfigApplyWithIgnoreDirectives(confFile, allowedDirectories, []string{})
}

// Rollback dumps the saved file content, and delete the notExists file. Best effort, will log error and continue
// if file operation failed during rollback.
func (b *ConfigApply) Rollback(cause error) error {
	log.Warnf("config_apply: rollback from cause: %s", cause)

	filesProto, err := b.writer.Proto()
	if err != nil {
		return fmt.Errorf("unrecoverable error during rollback (proto): %s", err)
	}

	r, err := zip.NewReader(filesProto)
	if err != nil {
		return fmt.Errorf("unrecoverable error during rollback (reader): %s", err)
	}

	for fullPath := range b.notExists {
		err = os.Remove(fullPath)
		if err != nil {
			log.Warnf("error during rollback (remove) for %s: %s", fullPath, err)
		}
	}

	for fullPath := range b.notExistDirs {
		err = os.RemoveAll(fullPath)
		if err != nil {
			log.Warnf("error during rollback (remove dir) for %s: %s", fullPath, err)
		}
	}

	r.RangeFileReaders(func(innerErr error, path string, mode os.FileMode, r io.Reader) bool {
		if innerErr != nil {
			log.Warnf("error during rollback for %s: %s", path, innerErr)
			return true
		}
		var f *os.File
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			log.Warnf("error during rollback (open) for %s: %s", path, err)
			return true
		}
		defer f.Close()
		_, err = io.Copy(f, r)
		if err != nil {
			log.Warnf("error during rollback (copy) for %s: %s", path, err)
			return true
		}
		log.Tracef("config_apply: rollback wrote to %s", path)
		return true
	})

	log.Info("config_apply: rollback complete")

	return nil
}

// Complete deletes any leftover files in the existing list, return error if failed to do so
func (b *ConfigApply) Complete() error {
	log.Debugf("config_apply: complete, removing %d leftover files", len(b.existing))
	for file := range b.existing {
		log.Infof("config_apply: deleting %s", file)
		if err := os.Remove(file); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
	}
	return nil
}

// MarkAndSave marks the provided fullPath, and save the content of the file in the provided fullPath
func (b *ConfigApply) MarkAndSave(fullPath string) error {
	// delete from existing list, so we don't delete them during Complete
	delete(b.existing, fullPath)

	p, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			b.notExists[fullPath] = struct{}{}
			log.Debugf("backup: %s does not exist", fullPath)

			for dir := range b.notExistDirs {
				if strings.HasPrefix(fullPath, dir) {
					return nil
				}
			}

			paths := strings.Split(fullPath, "/")
			for i := 2; i < len(paths); i++ {
				dirPath := strings.Join(paths[0:i], "/")

				_, err := os.Stat(dirPath)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						b.notExistDirs[dirPath] = struct{}{}
						log.Debugf("backup: dir %s does not exist", dirPath)
						return nil
					}

					log.Warnf("backup: dir %s error: %s", dirPath, err)
					return err
				}
			}

			return nil
		}

		log.Warnf("backup: %s error: %s", fullPath, err)
		return err
	}

	r, err := os.Open(fullPath)
	if err != nil {
		log.Warnf("backup: %s open error: %s", fullPath, err)
		return err
	}
	defer r.Close()
	log.Tracef("backup: %s mode=%s bytes=%d", fullPath, p.Mode(), p.Size())
	return b.writer.Add(fullPath, p.Mode(), r)
}

func (b *ConfigApply) RemoveFromNotExists(fullPath string) {
	delete(b.notExists, fullPath)
}

// mapCurrentFiles parse the provided file via cross-plane, generate a list of files, which should be identical to the
// DirectoryMap, will mark off the files as the config is being applied, any leftovers after complete should be deleted.
func (b *ConfigApply) mapCurrentFiles(confFile string, allowedDirectories map[string]struct{}, ignoreDirectives []string) error {
	log.Debugf("parsing %s", confFile)
	payload, err := crossplane.Parse(confFile,
		&crossplane.ParseOptions{
			IgnoreDirectives:   ignoreDirectives,
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		log.Debugf("failed to parse %s: %s", confFile, err)
		return err
	}
	seen := make(map[string]struct{})
	for _, xpc := range payload.Config {
		if !allowedPath(xpc.File, allowedDirectories) {
			continue
		}
		log.Debugf("config_apply: marking file (%s): %s", confFile, xpc.File)
		_, err = os.Stat(xpc.File)
		if err != nil {
			return fmt.Errorf("config_apply: %s read error %s", xpc.File, err)
		}
		b.existing[xpc.File] = struct{}{}
		err = CrossplaneConfigTraverse(&xpc,
			func(parent *crossplane.Directive, directive *crossplane.Directive) (bool, error) {
				switch directive.Directive {
				case "root":
					if err = b.walkRoot(directive.Args[0], seen, allowedDirectories); err != nil {
						log.Warnf("config_apply: walk root error %s: %s", directive.Args[0], err)
						return false, err
					}
				}
				return true, nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *ConfigApply) walkRoot(dir string, seen, allowedDirectories map[string]struct{}) error {
	if _, ok := seen[dir]; ok {
		return nil
	}
	seen[dir] = struct{}{}
	if !allowedPath(dir, allowedDirectories) {
		return nil
	}
	return filepath.WalkDir(dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			// the Info call here is, so we are as close as possible to the config code
			_, err = d.Info()
			if err != nil {
				return err
			}
			b.existing[path] = struct{}{}
			return nil
		},
	)
}

func (b *ConfigApply) GetExisting() map[string]struct{} {
	return b.existing
}

func (b *ConfigApply) GetNotExists() map[string]struct{} {
	return b.notExists
}

func (b *ConfigApply) GetNotExistDirs() map[string]struct{} {
	return b.notExistDirs
}
