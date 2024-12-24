package comicbook

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/abbit/m4k/internal/util"
)

type Page struct {
	Data        []byte
	Number      uint64
	Extension   string
	ChapterInfo ChapterInfo
}

func PageFromFile(zfile *zip.File, chapterInfo ChapterInfo) (*Page, error) {
	file, err := zfile.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	number, err := strconv.ParseUint(util.PathStem(zfile.Name), 10, 64)
	if err != nil {
		return nil, err
	}

	return &Page{
		Data:        buf.Bytes(),
		Number:      number,
		Extension:   filepath.Ext(zfile.Name),
		ChapterInfo: chapterInfo,
	}, nil
}

func (p *Page) Filepath() string {
	// TODO: use config to determine how to format file path
	volumeDirname := fmt.Sprintf("Volume %d", p.ChapterInfo.Volume)
	chapterDirname := p.ChapterInfo.String()
	pageFilename := fmt.Sprintf("%06d%s", p.Number, p.Extension)

	return filepath.Join(
		volumeDirname,
		chapterDirname,
		pageFilename,
	)
}
