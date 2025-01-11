package loader

import (
	"fmt"
	"time"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
	"github.com/luevano/mangoprovider/apis"
	"github.com/luevano/mangoprovider/scrapers"
)

func MangoLoaders() ([]libmangal.ProviderLoader, error) {
	o := mango.DefaultOptions()
	o.HTTPClient.Timeout = time.Minute

	var loaders []libmangal.ProviderLoader
	loaders = append(loaders, apis.Loaders(o)...)
	loaders = append(loaders, scrapers.Loaders(o)...)

	for _, loader := range loaders {
		if loader == nil {
			// TODO: need to provide more info
			return nil, fmt.Errorf("failed while loading providers")
		}
	}

	return loaders, nil
}
