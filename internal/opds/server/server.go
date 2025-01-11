package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/abbit/m4k/internal/mangal/client"
	"github.com/luevano/libmangal"
)

type Server struct {
	providers        []string
	providerToClient map[string]*libmangal.Client

	handler http.Handler
}

func New(
	ctx context.Context,
	providers []string,
) *Server {
	s := &Server{
		providers:        providers,
		providerToClient: make(map[string]*libmangal.Client),
	}
	for _, provider := range providers {
		client, err := client.NewClientByID(ctx, provider)
		if err != nil {
			err = fmt.Errorf("while creating client: %w", err)
			log.Fatalln("Error:", err) // TODO: do not panic
		}
		s.providerToClient[provider] = client
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /opds", s.providersHandler)
	mux.HandleFunc("GET /opds/{provider}", s.searchMangaHandler)
	mux.HandleFunc("GET /opds/{provider}/search", s.searchResultsHandler)
	mux.HandleFunc("GET /opds/{provider}/{manga}", s.chaptersRangeHandler)
	mux.HandleFunc("GET /opds/{provider}/{manga}/{chapters_range}", s.mangaChaptersHandler)
	mux.HandleFunc("GET /opds/{provider}/{manga}/{chapters_range}/download", s.downloadHandler)

	s.handler = mux
	s.handler = logRequestMiddleware(s.handler)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) getProviderClient(provider string) (*libmangal.Client, error) {
	if client, ok := s.providerToClient[provider]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("provider %q not found", provider)
}
