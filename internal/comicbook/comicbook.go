package comicbook

import (
	"archive/zip"
	"bytes"
	"io"
	"sort"

	"github.com/abbit/m4k/internal/util"
)

type ComicBook struct {
	Pages   []*Page
	Name    string
	cbzData []byte
}

func ReadComicBook(path string) (*ComicBook, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	name := util.WithoutPaddedIndex(util.PathStem(path))
	chapterInfo := ChapterInfoFromName(name)

	var pages []*Page
	for _, f := range r.File {
		if util.IsImage(f.Name) {
			page, err := PageFromFile(f, chapterInfo)
			if err != nil {
				return nil, err
			}
			pages = append(pages, page)
		}
	}
	// sort pages by page number
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })

	return &ComicBook{Pages: pages, Name: name}, nil
}

func (cb *ComicBook) FileName() string {
	return util.SanitizePath(cb.Name) + ".cbz"
}

func (cb *ComicBook) WriteTo(wr io.Writer) (n int64, err error) {
	w := zip.NewWriter(wr)
	defer w.Close()

	// write pages to zip archive
	for _, page := range cb.Pages {
		file, err := w.Create(page.Filepath())
		if err != nil {
			return n, err
		}

		nfile, err := file.Write(page.Data)
		n += int64(nfile)
		if err != nil {
			return n, err
		}
	}

	return
}

func (cb *ComicBook) fillCbzData() error {
	var buf bytes.Buffer
	if _, err := cb.WriteTo(&buf); err != nil {
		return err
	}
	cb.cbzData = buf.Bytes()
	return nil
}

func (cb *ComicBook) Reader() (*bytes.Reader, error) {
	if len(cb.cbzData) == 0 {
		if err := cb.fillCbzData(); err != nil {
			return nil, err
		}
	}

	return bytes.NewReader(cb.cbzData), nil
}

func MergeComicBooks(comicbooks []*ComicBook, name string) *ComicBook {
	var pages []*Page
	pageNumber := uint64(0)
	for _, comicbook := range comicbooks {
		for _, p := range comicbook.Pages {
			pageNumber++
			pages = append(pages, &Page{
				Data:        p.Data,
				Extension:   p.Extension,
				Number:      pageNumber,
				ChapterInfo: p.ChapterInfo,
			})
		}
	}

	return &ComicBook{Name: name, Pages: pages}
}
