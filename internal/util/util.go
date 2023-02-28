package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// returns file name without extension
func PathStem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// returns slice of paths to cbz files in given directory
func FindFilesWithExt(dirpath, ext string) ([]string, error) {
	dirEntries, err := os.ReadDir(dirpath)
	if err != nil {
		return nil, fmt.Errorf("when tried to get files from directory: %v", err)
	}

	var files []string
	for _, f := range dirEntries {
		if filepath.Ext(f.Name()) == ext {
			files = append(files, filepath.Join(dirpath, f.Name()))
		}
	}

	return files, nil
}

func RemoveFiles(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
