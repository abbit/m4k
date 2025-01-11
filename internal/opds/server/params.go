package server

import (
	"fmt"
	"net/http"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
)

type params struct {
	Provider         string
	Client           *libmangal.Client
	MangaEncoded     string
	Manga            mangadata.Manga
	Volume           string
	ChaptersRangeStr string
	ChaptersRange    []int
}

func (s *Server) parseRequestParams(r *http.Request) (*params, error) {
	p := &params{}

	provider := r.PathValue("provider")
	if len(provider) == 0 {
		return p, nil
	}
	client, err := s.getProviderClient(provider)
	if err != nil {
		return nil, fmt.Errorf("getting provider client: %w", err)
	}
	p.Provider = provider
	p.Client = client

	mangaEncoded := r.PathValue("manga")
	if len(mangaEncoded) == 0 {
		return p, nil
	}
	manga, err := decodeManga(mangaEncoded)
	if err != nil {
		return nil, fmt.Errorf("decoding manga: %w", err)
	}
	p.MangaEncoded = mangaEncoded
	p.Manga = manga

	// TODO: handle Volume string with space
	volume := r.PathValue("volume")
	if len(volume) == 0 {
		return p, nil
	}
	p.Volume = volume

	chaptersRangeStr := r.PathValue("chapters_range")
	if len(chaptersRangeStr) == 0 {
		return p, nil
	}
	chaptersRange, err := parseChaptersRange(chaptersRangeStr)
	if err != nil {
		return nil, fmt.Errorf("parsing chapters range: %w", err)
	}
	p.ChaptersRangeStr = chaptersRangeStr
	p.ChaptersRange = chaptersRange

	return p, nil
}
