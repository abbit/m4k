package server

import (
	"context"

	"github.com/abbit/m4k/internal/log"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
)

func getChapters(ctx context.Context, client *libmangal.Client, manga mangadata.Manga, chaptersRange []int) ([]mangadata.Chapter, error) {
	volumes, err := client.MangaVolumes(ctx, manga)
	if err != nil {
		return nil, err
	}

	chapters, err := getAllVolumeChapters(ctx, client, volumes)
	if err != nil {
		return nil, err
	}

	var selectedChapters []mangadata.Chapter

	var fromChapter, toChapter float32
	fromChapter = float32(chaptersRange[0])
	if len(chaptersRange) == 1 {
		toChapter = fromChapter
	} else {
		toChapter = float32(chaptersRange[1])
	}

	for _, chapter := range chapters {
		chapterNum := chapter.Info().Number
		if chapterNum >= fromChapter && chapterNum <= toChapter {
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
			log.Info.Printf("no chapters found for volume %.1f", volume.Info().Number)
		}
	}
	return chapters, nil
}
