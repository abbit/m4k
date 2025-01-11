package manager

import (
	"github.com/abbit/m4k/internal/mangal/provider/loader"
	"github.com/luevano/libmangal"
)

func Loaders() ([]libmangal.ProviderLoader, error) {
	var loaders []libmangal.ProviderLoader

	mangoLoaders, err := loader.MangoLoaders()
	if err != nil {
		return nil, err
	}
	loaders = append(loaders, mangoLoaders...)

	return loaders, nil
}
