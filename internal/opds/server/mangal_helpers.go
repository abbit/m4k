package server

import (
	"context"
	"fmt"
	"log"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
)

func getChapters(ctx context.Context, client *libmangal.Client, manga mangadata.Manga, chaptersRange []int) ([]mangadata.Chapter, error) {
	volumes, err := client.MangaVolumes(ctx, manga)
	if err != nil {
		return nil, err
	}
	if len(volumes) == 0 {
		// TODO: use query instead of title?
		return nil, fmt.Errorf("no manga volumes found with provider %q title %q", client.Info().Name, manga.Info().Title)
	}

	chapters, err := getAllVolumeChapters(ctx, client, volumes)
	if err != nil {
		return nil, err
	}

	var selectedChapters []mangadata.Chapter
	var fromChapter, toChapter int
	// TODO: if len = 0; if len > 2
	// TODO: optimize
	fromChapter = chaptersRange[0]
	if len(chaptersRange) == 1 {
		toChapter = fromChapter
	} else {
		toChapter = chaptersRange[1]
	}
	for _, chapter := range chapters {
		chapterNum := chapter.Info().Number
		if chapterNum >= float32(fromChapter) && chapterNum <= float32(toChapter) {
			selectedChapters = append(selectedChapters, chapter)
		}
	}

	return selectedChapters, nil
}

func getAllVolumeChapters(ctx context.Context, client *libmangal.Client, volumes []mangadata.Volume) ([]mangadata.Chapter, error) {
	var chapters []mangadata.Chapter
	for _, volume := range volumes {
		volumeChapters, err := client.VolumeChapters(ctx, volume)
		if err != nil {
			return nil, err
		}

		if len(volumeChapters) != 0 {
			chapters = append(chapters, volumeChapters...)
		} else {
			log.Printf("no chapters found for volume %.1f", volume.Info().Number)
		}
	}
	return chapters, nil
}
