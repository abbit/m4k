package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/abbit/m4k/internal/comicbook"
	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/transform"
	"github.com/abbit/m4k/internal/util"
	"github.com/luevano/libmangal"
)

const (
	baseDirName = "m4k-opds-server"
)

const (
	KindlePW5Width  = 1236 // px
	KindlePW5Height = 1648 // px
)

const (
	maxRetries = 5
)

func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			log.Error.Println("downloadHandler:", resultErr)
			http.Error(w, resultErr.Error(), http.StatusInternalServerError)
		}
	}()

	params, err := s.parseRequestParams(r)
	if err != nil {
		resultErr = fmt.Errorf("parsing request params: %w", err)
		return
	}

	format := r.URL.Query().Get("format")
	forDevice := r.URL.Query().Get("for")

	log.Info.Println(params)
	log.Info.Println(format, forDevice)

	ctx := context.Background()

	chapters, err := getChapters(ctx, params.Client, params.Manga, params.ChaptersRange)
	if err != nil {
		resultErr = fmt.Errorf("getting chapters: %w", err)
		return
	}

	baseDirPath := path.Join(os.TempDir(), baseDirName)

	// check if the directory exists, create it if not
	if _, err := os.Stat(baseDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(baseDirPath, os.ModePerm); err != nil {
			resultErr = fmt.Errorf("creating base dir: %w", err)
			return
		}
	}

	log.Info.Println("Saving to:", baseDirPath)

	downloadOptions := libmangal.DownloadOptions{
		Format:            libmangal.FormatCBZ,
		Directory:         baseDirPath,
		CreateProviderDir: true,
		CreateMangaDir:    true,
		SkipIfExists:      true,
		ImageTransformer:  func(data []byte) ([]byte, error) { return data, nil },
	}

	var resultsDir string

	// FIX: rework this mess
	// TODO: make configurable
	retryCount := 0
	for _, chapter := range chapters {
		retry := true
		for retry {
			retry = false
			res, err := params.Client.DownloadChapter(ctx, chapter, downloadOptions)
			if err != nil {
				errMsg := err.Error()
				// TODO: handle other responses here too if possible
				if strings.Contains(errMsg, "429") && strings.Contains(errMsg, "Retry-After") {
					retry = true
					retryCount++
					if retryCount > maxRetries {
						resultErr = fmt.Errorf("exceeded max retries (%d) while downloading chapters", maxRetries)
						return

					}

					raTemp := strings.Split(errMsg, ":")
					raParsed, err := strconv.Atoi(strings.TrimSpace(raTemp[len(raTemp)-1]))
					if err != nil {
						resultErr = fmt.Errorf("parsing Retry-Count from error message: %w", err)
						return
					}

					retryAfter := time.Duration(min(10, raParsed)) * time.Second
					log.Info.Printf("429 Too Many Requests (retry #%d). Retrying in %s\n", retryCount, retryAfter)
					time.Sleep(retryAfter)
					continue
				}

				// In case that the error is not due to 429 code, return the error
				resultErr = fmt.Errorf("downloading chapter: %w", err)
				return
			}

			resultsDir = res.Directory
		}
	}

	filename := formatMangaTitle(params)
	cbzfile, err := transformCBZ(resultsDir, filename)
	if err != nil {
		resultErr = fmt.Errorf("reading cbz file: %w", err)
		return
	}

	cbzReader, err := cbzfile.Reader()
	if err != nil {
		resultErr = fmt.Errorf("creating cbz reader: %w", err)
		return
	}

	// TODO: save result on disk?

	http.ServeContent(w, r, cbzfile.FileName(), time.Time{}, cbzReader)
}

func transformCBZ(srcdir, mergedFileName string) (*comicbook.ComicBook, error) {
	log.Info.Println("Searching cbz files...")
	cbzFiles, err := util.FindFilesWithExt(srcdir, ".cbz")
	if err != nil {
		return nil, fmt.Errorf("searching cbz files: %w", err)
	}

	log.Info.Println("Reading cbz files...")
	var comicbooks []*comicbook.ComicBook
	for _, path := range cbzFiles {
		cb, err := comicbook.ReadComicBook(path)
		if err != nil {
			return nil, fmt.Errorf("reading comicbook from path %s: %w", path, err)
		}
		comicbooks = append(comicbooks, cb)
	}

	log.Info.Println("Merging cbz files...")
	combined := comicbook.MergeComicBooks(comicbooks, mergedFileName)

	log.Info.Println("Transforming combined file for Kindle...")
	transformOpts := &transform.Options{
		Width:    KindlePW5Width,
		Height:   KindlePW5Height,
		Encoding: "jpg",
	}
	if err := transform.TransformComicBook(combined, transformOpts); err != nil {
		return nil, fmt.Errorf("transforming pages: %w", err)
	}

	log.Info.Println("Done transforming CBZ")

	return combined, nil
}
