package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) mangaChaptersHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			log.Error.Println("mangaChaptersHandler:", resultErr)
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
	if _, err = getChapters(ctx, params.Client, params.Manga, params.ChaptersRange); err != nil {
		resultErr = fmt.Errorf("getting chapters: %w", err)
		return
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
						Href:  fmt.Sprintf("/opds/%s/%s/%s/download?format=cbz&for=kindle", params.Provider, params.MangaEncoded, params.ChaptersRangeStr),
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
