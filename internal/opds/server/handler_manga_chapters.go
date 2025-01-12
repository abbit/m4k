package server

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"

	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) mangaChaptersHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			slog.Error("mangaChaptersHandler", slog.Any("error", resultErr))
			http.Error(w, resultErr.Error(), http.StatusInternalServerError)
		}
	}()

	params, err := s.parseRequestParams(r)
	if err != nil {
		resultErr = fmt.Errorf("parsing request params: %w", err)
		return
	}

	ctx := context.Background()

	// Try to get chapters to check if they exist now
	chapters, err := getChapters(ctx, params.Client, params.Manga, params.ChaptersRange)
	if err != nil {
		resultErr = fmt.Errorf("getting chapters: %w", err)
		return
	}
	if len(chapters) == 0 {
		resultErr = nil
		errMsg := fmt.Sprintf("got 0 chapters for manga %s", params.Manga.Info().Title)
		http.Error(w, errMsg, http.StatusNotFound)
		return
	}

	// set max chapter to the last chapter, if bigger chapter number is used
	var maxChapter float32
	for _, chapter := range chapters {
		maxChapter = max(maxChapter, chapter.Info().Number)
	}
	maxChapterInt := int(math.Ceil(float64(maxChapter)))

	if params.ChaptersRange[0] > maxChapterInt {
		resultErr = nil
		http.Error(w, fmt.Sprintf("'from' chapter number %d is bigger than last chapter number %d", params.ChaptersRange[0], maxChapterInt), http.StatusBadRequest)
		return
	}

	if len(params.ChaptersRange) == 2 {
		if params.ChaptersRange[1] > maxChapterInt {
			params.ChaptersRange[1] = maxChapterInt
		}
	}

	title := formatMangaChaptersTitle(params)
	mangaChaptersFeed := opds.Feed{
		ID:          r.RequestURI,
		Title:       title,
		LastUpdated: opds.TimeNow(),
		Links: []opds.Link{
			linkStart(),
			{
				Rel:  opds.RelSelf,
				Type: opds.FeedTypeAcquisition,
				Href: r.RequestURI,
			},
		},
		Entries: []opds.Entry{
			{
				Title:       title,
				LastUpdated: opds.TimeNow(),
				Link: []opds.Link{
					{
						Rel:   opds.RelAcquisition,
						Type:  opds.FileTypeCBZ,
						Href:  fmt.Sprintf("/opds/%s/%s/%s/download?for=kindle-pw5", params.Provider, params.MangaEncoded, encodeChaptersRange(params.ChaptersRange)),
						Title: "cbz",
					},
				},
			},
		},
	}

	if err := writeXML(w, mangaChaptersFeed); err != nil {
		resultErr = fmt.Errorf("writing response: %w", err)
	}
}
