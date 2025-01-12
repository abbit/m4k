package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) chaptersRangeHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			slog.Error("chaptersRangeHandler", slog.Any("error", resultErr))
			http.Error(w, resultErr.Error(), http.StatusBadRequest)
		}
	}()

	params, err := s.parseRequestParams(r)
	if err != nil {
		resultErr = fmt.Errorf("while parsing request params: %w", err)
		return
	}

	chaptersRangeFeed := opds.Feed{
		ID:          r.RequestURI,
		Title:       fmt.Sprintf("Enter %s chapters range", params.Manga),
		LastUpdated: opds.TimeNow(),
		Links: []opds.Link{
			linkStart(),
			// pseudo-search for rendering chapters range input field
			{
				Rel:  opds.RelSearch,
				Type: opds.FeedTypeAcquisition,
				Href: fmt.Sprintf("/opds/%s/%s/{searchTerms}", params.Provider, params.MangaEncoded),
			},
		},
	}

	if err := writeXML(w, chaptersRangeFeed); err != nil {
		resultErr = fmt.Errorf("writing response: %w", err)
	}
}
