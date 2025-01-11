package server

import (
	"fmt"
	"net/http"

	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) providersHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			log.Error.Println("providersHandler:", resultErr)
			http.Error(w, resultErr.Error(), http.StatusInternalServerError)
		}
	}()

	providerEntries := make([]opds.Entry, 0, len(s.providers))
	for _, provider := range s.providers {
		providerEntries = append(providerEntries, opds.Entry{
			Title:       provider,
			LastUpdated: opds.TimeNow(),
			Link: []opds.Link{
				{
					Rel:  opds.RelSubsection,
					Type: opds.FeedTypeNavigation,
					Href: fmt.Sprintf("/opds/%s", provider),
				},
			},
		})
	}

	providersFeed := opds.Feed{
		ID:          r.RequestURI,
		Title:       "Mangal OPDS Server",
		LastUpdated: opds.TimeNow(),
		Author: &opds.Author{
			Name: "Mangal OPDS Server",
			URI:  r.Host,
		},
		Links: []opds.Link{
			linkStart(),
			{
				Rel:  opds.RelSelf,
				Type: opds.FeedTypeNavigation,
				Href: r.RequestURI,
			},
		},
		Entries: providerEntries,
	}

	if err := writeXML(w, providersFeed); err != nil {
		resultErr = fmt.Errorf("writing response: %w", err)
	}
}
