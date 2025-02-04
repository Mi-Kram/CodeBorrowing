package utils

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func CreateDirectory(path string) (alreadyExisted bool, error error) {
	alreadyExisted = false

	stat, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	} else if stat.IsDir() {
		return true, nil
	}

	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return false, err
	}

	return false, nil
}

func ClearDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func GetDirectorySize(path string) (uint64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return uint64(size), err
}
