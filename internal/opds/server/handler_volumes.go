package server

import (
	"fmt"
	"net/http"

	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) mangaVolumesHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			log.Error.Println("mangaVolumesHandler:", resultErr)
			http.Error(w, resultErr.Error(), http.StatusBadRequest)
		}
	}()

	params, err := s.parseRequestParams(r)
	if err != nil {
		resultErr = fmt.Errorf("parsing request params: %w", resultErr)
		return
	}

	// TODO:
	volumes := []string{
		"Volume1",
	}

	volumeEntries := make([]opds.Entry, 0, len(volumes))
	for _, volume := range volumes {
		volumeEntries = append(volumeEntries, opds.Entry{
			Title:       volume,
			LastUpdated: opds.TimeNow(),
			Link: []opds.Link{
				{
					Rel:  opds.RelSubsection,
					Type: opds.FeedTypeNavigation,
					Href: fmt.Sprintf("/opds/%s/%s/%s", params.Provider, params.MangaEncoded, volume),
				},
			},
		})
	}

	mangaFeed := opds.Feed{
		ID:          r.RequestURI,
		Title:       fmt.Sprintf("%s volumes", params.Manga),
		LastUpdated: opds.TimeNow(),
		Links: []opds.Link{
			linkStart(),
			{
				Rel:  opds.RelSelf,
				Type: opds.FeedTypeNavigation,
				Href: r.RequestURI,
			},
		},
		Entries: volumeEntries,
	}

	if err := writeXML(w, mangaFeed); err != nil {
		resultErr = fmt.Errorf("writing response: %w", err)
	}
}
