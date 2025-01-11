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
