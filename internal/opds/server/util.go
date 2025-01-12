package server

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/abbit/m4k/internal/opds"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"

	mango "github.com/luevano/mangoprovider"
)

func writeXML(w http.ResponseWriter, v any) error {
	w.Header().Add("Content-Type", "application/xml; charset=utf-8")

	if _, err := fmt.Fprint(w, xml.Header); err != nil {
		return fmt.Errorf("writing xml header: %w", err)
	}

	if err := xml.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encoding xml: %w", err)
	}

	return nil
}

func encodeManga(manga mangadata.Manga) (string, error) {
	// all used providers should be from mangoprovider
	// get concrete type here to be able to reconstruct it in `decode` later
	mangaStruct, ok := manga.(*mango.Manga)
	if !ok {
		return "", fmt.Errorf("manga is not `mango.Manga` type")
	}

	mangaBytes, err := json.Marshal(mangaStruct)
	if err != nil {
		return "", fmt.Errorf("marshaling manga: %w", err)
	}

	return base64.URLEncoding.EncodeToString(mangaBytes), nil
}

func decodeManga(encoded string) (mangadata.Manga, error) {
	mangaBytes, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding json bytes from url: %w", err)
	}

	// fix for nil pointer dereference in `Client.DownloadChapter`
	// TODO: do a proper fix
	var meta metadata.Metadata
	manga := mango.Manga{
		Metadata_: &meta,
	}
	if err := json.Unmarshal(mangaBytes, &manga); err != nil {
		return nil, fmt.Errorf("unmarshaling manga: %w", err)
	}

	return &manga, nil
}

func linkStart() opds.Link {
	return opds.Link{
		Rel:  opds.RelStart,
		Type: opds.FeedTypeNavigation,
		Href: "/opds",
	}
}

func linkSearch(provider string) opds.Link {
	return opds.Link{
		Rel:  opds.RelSearch,
		Type: opds.FeedTypeSearchTemplate,
		Href: fmt.Sprintf("/opds/%s/search?q={searchTerms}", provider),
	}
}

func parseChaptersRange(rangeStr string) ([]int, error) {
	splittedRange := strings.Split(strings.TrimSpace(rangeStr), "-")
	if len(splittedRange) == 0 {
		return nil, fmt.Errorf("empty chapters range")
	}
	if len(splittedRange) > 2 {
		return nil, fmt.Errorf("invalid chapters range: %s", rangeStr)
	}

	var chaptersRange []int
	for _, numStr := range splittedRange {
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("parsing chapter number '%s': %w", numStr, err)
		}

		chaptersRange = append(chaptersRange, num)
	}

	// TODO: can there be negative chapter numbers?
	if chaptersRange[0] < 0 {
		return nil, fmt.Errorf("'from' chapter is less than 0: %s", rangeStr)
	}

	if len(chaptersRange) == 2 && chaptersRange[0] > chaptersRange[1] {
		return nil, fmt.Errorf("'from' chapter is greater than 'to' chapter: %s", rangeStr)
	}

	return chaptersRange, nil
}

func encodeChaptersRange(chaptersRange []int) string {
	if len(chaptersRange) == 1 {
		return fmt.Sprintf("%d", chaptersRange[0])
	}
	return fmt.Sprintf("%d-%d", chaptersRange[0], chaptersRange[1])
}

func formatChaptersRange(chaptersRange []int) string {
	if len(chaptersRange) == 1 {
		return fmt.Sprintf("Chapter %d", chaptersRange[0])
	}

	return fmt.Sprintf("Chapters %d-%d", chaptersRange[0], chaptersRange[1])
}

func formatMangaChaptersTitle(params *params) string {
	return fmt.Sprintf("%s %s", params.Manga, formatChaptersRange(params.ChaptersRange))
}
