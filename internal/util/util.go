package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	invalidPathChars = `/:`
)

// returns file name without extension
func PathStem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// FilterDirFilePaths returns list of file paths in directory that pass the filter.
// Paths sorted by filename.
func FilterDirFilePaths(dirpath string, filter func(path string) bool) ([]string, error) {
	dirEntries, err := os.ReadDir(dirpath)
	if err != nil {
		return nil, fmt.Errorf("when tried to get files from directory: %v", err)
	}

	var paths []string
	for _, f := range dirEntries {
		p := filepath.Join(dirpath, f.Name())
		if filter(p) {
			paths = append(paths, p)
		}
	}

	return paths, nil
}

func RemoveFiles(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// returns chapter name without padded index
func WithoutPaddedIndex(name string) string {
	before, after, found := strings.Cut(name, "_")
	if found {
		return after
	}
	return before
}

// checks if file is actual manga page or metadata file
func IsImage(path string) bool {
	ext := filepath.Ext(path)
	if ext == ".jpg" ||
		ext == ".jpeg" ||
		ext == ".png" {
		return true
	}

	return false
}

func SanitizePath(path string) string {
	var (
		sanitized strings.Builder
		prev      rune
	)

	const underscore = '_'

	for _, r := range path {
		var toWrite rune
		if strings.ContainsRune(invalidPathChars, r) {
			toWrite = underscore
		} else {
			toWrite = r
		}

		// replace two or more consecutive underscores with one underscore
		if (toWrite == underscore && prev != underscore) || toWrite != underscore {
			sanitized.WriteRune(toWrite)
		}

		prev = toWrite
	}

	return strings.TrimFunc(sanitized.String(), func(r rune) bool {
		return r == underscore || unicode.IsSpace(r)
	})
}

func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return false, fmt.Errorf("path is a directory: %s", path)
		}
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
