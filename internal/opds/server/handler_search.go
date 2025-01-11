package server

import (
	"fmt"
	"net/http"

	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) searchMangaHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			log.Error.Println("searchMangaHandler:", resultErr)
			http.Error(w, resultErr.Error(), http.StatusBadRequest)
		}
	}()

	params, err := s.parseRequestParams(r)
	if err != nil {
		resultErr = fmt.Errorf("parsing request params: %w", resultErr)
		return
	}

	providerFeed := opds.Feed{
		ID:          r.RequestURI,
		Title:       params.Provider,
		LastUpdated: opds.TimeNow(),
		Links: []opds.Link{
			linkStart(),
			linkSearch(params.Provider),
			{
				Rel:  opds.RelSelf,
				Type: opds.FeedTypeNavigation,
				Href: r.RequestURI,
			},
		},
	}

	if err := writeXML(w, providerFeed); err != nil {
		resultErr = fmt.Errorf("writing response: %w", err)
	}
}
