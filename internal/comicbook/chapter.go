package comicbook

import (
	"fmt"
	"strconv"
	"strings"
)

type ChapterInfo struct {
	// Name of chapter
	// Taken from cbz file name
	Name string
	// Number of chapter
	// Stored as float64 to allow for chapter numbers like 1.5, 2.1, etc
	Number float64
	// Volume number
	Volume int
}

func ChapterInfoFromName(name string) *ChapterInfo {
	info := &ChapterInfo{
		Name:   name,
		Volume: 1,
	}

	numberStr, name, ok := strings.Cut(name, " ")
	if !ok {
		return info
	}

	// NOTE: chapter number parsing from path for now is closely tied to how `mangal` saves them

	numberStr = strings.TrimFunc(numberStr, func(r rune) bool {
		return r == '[' || r == ']'
	})
	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return info
	}

	name = strings.ReplaceAll(name, "_", " ")

	info.Number = number
	info.Name = name

	return info
}

func (ci *ChapterInfo) String() string {
	var name string
	if strings.HasPrefix(ci.Name, "Chapter") {
		// if chapter name starts with "Chapter",
		// it probably already already contains chapter number
		// so just use name as is
		name = ci.Name
	} else {
		// format chapter name as "Chapter <number> - <name>"
		name = fmt.Sprintf("Chapter %.1f - %s", ci.Number, ci.Name)
	}
	return name
}
