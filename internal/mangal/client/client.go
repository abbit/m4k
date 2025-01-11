package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abbit/m4k/internal/mangal/provider/manager"
	"github.com/abbit/m4k/internal/util"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/mangadata"
)

var (
	clients   []*libmangal.Client
	clientsMu sync.Mutex
)

func Get(loader libmangal.ProviderLoader) *libmangal.Client {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	var client *libmangal.Client
	for _, c := range clients {
		if c.Info().ID == loader.Info().ID {
			client = c
		}
	}
	return client
}

func Exists(loader libmangal.ProviderLoader) bool {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	exists := false
	for _, c := range clients {
		if c.Info().ID == loader.Info().ID {
			exists = true
		}
	}
	return exists
}

func CloseAll() error {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for _, client := range clients {
		if err := client.Close(); err != nil {
			return err
		}
	}

	return nil
}

func NewClient(ctx context.Context, loader libmangal.ProviderLoader) (*libmangal.Client, error) {
	if Exists(loader) {
		return nil, fmt.Errorf("client for loader %q already exists", loader)
	}
	clientsMu.Lock()
	defer clientsMu.Unlock()

	options := libmangal.DefaultClientOptions()
	options.HTTPClient.Timeout = time.Minute
	options.ChapterName = chapterName

	client, err := libmangal.NewClient(ctx, loader, options)
	if err != nil {
		return nil, err
	}
	// guaranteed to exist
	// anilist.Anilist().SetLogger(client.Logger())
	// _ = client.SetMetadataProvider(anilist.Anilist())

	clients = append(clients, client)
	return client, nil
}

func NewClientByID(ctx context.Context, provider string) (*libmangal.Client, error) {
	loaders, err := manager.Loaders()
	if err != nil {
		return nil, err
	}

	loader, ok := findLoader(loaders, func(loader libmangal.ProviderLoader) bool {
		return loader.Info().ID == provider
	})

	if !ok {
		return nil, fmt.Errorf("provider with ID %q not found", provider)
	}

	client, err := NewClient(ctx, loader)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func findLoader(loaders []libmangal.ProviderLoader, f func(libmangal.ProviderLoader) bool) (libmangal.ProviderLoader, bool) {
	for _, loader := range loaders {
		if f(loader) {
			return loader, true
		}
	}
	return nil, false
}

func chapterName(provider libmangal.ProviderInfo, chapter mangadata.Chapter) string {
	res := fmt.Sprintf("[%06.1f] %s", chapter.Info().Number, chapter.Info().Title)

	return util.SanitizePath(res)
}
