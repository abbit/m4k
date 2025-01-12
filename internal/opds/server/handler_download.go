package server

import (
	"context"
	"fmt"
	"io"
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
	baseDirName         = "m4k-opds-server"
	transformResultsDir = "transformed"
)

const (
	KindlePW5Width  = 1236 // px
	KindlePW5Height = 1648 // px
)

const (
	maxRetries = 5
)

var (
	baseDirPath               = path.Join(os.TempDir(), baseDirName)
	transformedResultsDirPath = path.Join(baseDirPath, transformResultsDir)
)

func init() {
	// creating transformed results dir implies creating the base dir too
	if err := os.MkdirAll(transformedResultsDirPath, os.ModePerm); err != nil {
		log.Error.Fatalln("creating transformed results dir:", err)
	}
}

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

	forDevice := r.URL.Query().Get("for")

	log.Info.Println("params:", params)
	log.Info.Println("forDevice:", forDevice)

	var deviceWidth, deviceHeight int
	switch forDevice {
	case "kindle-pw5":
		deviceWidth, deviceHeight = KindlePW5Width, KindlePW5Height
	default:
		resultErr = fmt.Errorf("unsupported device: %s", forDevice)
		return
	}

	ctx := context.Background()

	chapters, err := getChapters(ctx, params.Client, params.Manga, params.ChaptersRange)
	if err != nil {
		resultErr = fmt.Errorf("getting chapters: %w", err)
		return
	}

	downloadOptions := libmangal.DownloadOptions{
		Format:            libmangal.FormatCBZ,
		Directory:         baseDirPath,
		CreateProviderDir: true,
		CreateMangaDir:    true,
		SkipIfExists:      true,
		ImageTransformer:  func(data []byte) ([]byte, error) { return data, nil },
	}

	// FIX: rework this mess
	// TODO: make configurable
	var downloadedMangaDir string
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

			downloadedMangaDir = res.Directory
		}
	}

	mangaChaptersTitle := formatMangaChaptersTitle(params)
	transformedFileName := util.SanitizePath(mangaChaptersTitle) + ".cbz"
	transformedFilePath := path.Join(transformedResultsDirPath, transformedFileName)

	exists, err := util.FileExists(transformedFilePath)
	if err != nil {
		resultErr = fmt.Errorf("checking if transformed cbz file exists: %w", err)
		return
	}

	var cbzReader io.ReadSeeker
	if exists {
		// file exists already, serve it
		cbzReader, err = os.Open(transformedFilePath)
		if err != nil {
			resultErr = fmt.Errorf("opening transformed cbz file: %w", err)
			return
		}
	} else {
		// file does not exist, transform it

		// TODO: transform only selected chapters
		transformOpts := &transform.Options{
			Width:  deviceWidth,
			Height: deviceHeight,
			// TODO: make configurable
			Encoding: "jpg",
		}
		cb, err := transformCBZ(downloadedMangaDir, mangaChaptersTitle, transformOpts)
		if err != nil {
			resultErr = fmt.Errorf("transforming cbz file: %w", err)
			return
		}

		cbzReader, err = cb.Reader()
		if err != nil {
			resultErr = fmt.Errorf("creating cbz reader: %w", err)
			return
		}

		// write transformed cbz file to disk
		cbzfile, err := os.Create(transformedFilePath)
		if err != nil {
			resultErr = fmt.Errorf("creating transformed cbz file: %w", err)
			return
		}

		if _, err := io.Copy(cbzfile, cbzReader); err != nil {
			resultErr = fmt.Errorf("writing transformed cbz file: %w", err)
			return
		}

		// reset reader to start of file
		cbzReader.Seek(0, io.SeekStart)
	}

	http.ServeContent(w, r, transformedFileName, time.Time{}, cbzReader)
}

func transformCBZ(srcdir, mergedFileName string, transformOpts *transform.Options) (*comicbook.ComicBook, error) {
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
	if err := transform.TransformComicBook(combined, transformOpts); err != nil {
		return nil, fmt.Errorf("transforming pages: %w", err)
	}

	log.Info.Println("Done transforming CBZ")

	return combined, nil
}
