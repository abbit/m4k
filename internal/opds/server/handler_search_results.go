package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/abbit/m4k/internal/log"
	"github.com/abbit/m4k/internal/opds"
)

func (s *Server) searchResultsHandler(w http.ResponseWriter, r *http.Request) {
	var resultErr error
	defer func() {
		if resultErr != nil {
			log.Error.Println("searchResultsHandler:", resultErr)
			http.Error(w, resultErr.Error(), http.StatusBadRequest)
		}
	}()

	params, err := s.parseRequestParams(r)
	if err != nil {
		resultErr = fmt.Errorf("parsing request params: %w", resultErr)
		return
	}
	query := r.URL.Query().Get("q")

	ctx := context.Background()

	log.Info.Printf("searching with %s for %q\n", params.Provider, query)

	searchResults, err := params.Client.SearchMangas(ctx, query)
	if err != nil {
		resultErr = fmt.Errorf("searching mangas: %w", resultErr)
		return
	}

	if len(searchResults) == 0 {
		resultErr = fmt.Errorf("no mangas found with provider ID %q and query %q", params.Provider, query)
		return
	}

	searchResultEntries := make([]opds.Entry, 0, len(searchResults))
	for _, manga := range searchResults {
		mangaEncoded, err := encodeManga(manga)
		if err != nil {
			resultErr = fmt.Errorf("encoding manga: %w", err)
			return
		}

		searchResultEntries = append(searchResultEntries, opds.Entry{
			Title:       manga.Info().Title,
			LastUpdated: opds.TimeNow(),
			Link: []opds.Link{
				{
					Rel:  opds.RelSubsection,
					Type: opds.FeedTypeNavigation,
					Href: fmt.Sprintf("/opds/%s/%s", params.Provider, mangaEncoded),
				},
			},
		})
	}

	searchResultFeed := opds.Feed{
		ID:          r.RequestURI,
		Title:       fmt.Sprintf("Search results for \"%s\"", query),
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
		Entries: searchResultEntries,
	}

	if err := writeXML(w, searchResultFeed); err != nil {
		resultErr = fmt.Errorf("writing response: %w", err)
	}
}
