package modules

import (
	"log/slog"

	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"
)

func Converters(loader *config.ConfigLoader) ModuleMapRegistry[ConverterModule] {
	return NewModuleMapRegistry(map[string]ModuleCreator[ConverterModule]{
		"goldmark": func() (ConverterModule, error) {
			// todo: better configuration, deprecate base config.
			return service.NewGoldmarkConverter("", loader.BaseConfig().Features), nil
		},
	})
}

func PostRepositories(loader *config.ConfigLoader, log *slog.Logger) ModuleMapRegistry[PostIndexerModule] {
	return NewModuleMapRegistry(map[string]ModuleCreator[PostIndexerModule]{
		"inmemory": func() (PostIndexerModule, error) {
			return service.NewMemoryPostIndexer(loader.BaseConfig(), log), nil
		},
		"bleve": func() (PostIndexerModule, error) {
			return service.NewBlevePostIndexer(loader.BaseConfig(), log)
		},
	})
}

func PostSources(loader *config.ConfigLoader, log *slog.Logger) ModuleMapRegistry[PostSourceModule] {
	return NewModuleMapRegistry(map[string]ModuleCreator[PostSourceModule]{
		"filesystem": func() (PostSourceModule, error) {
			return service.NewFileSystemSource(log), nil
		},
		"git": func() (PostSourceModule, error) {
			return service.NewGitSource(log)
		},
	})
}

func OGImageGenerators(loader *config.ConfigLoader, log *slog.Logger) ModuleMapRegistry[OGImageGeneratorModule] {
	return NewModuleMapRegistry(map[string]ModuleCreator[OGImageGeneratorModule]{
		"default": func() (OGImageGeneratorModule, error) {
			icon, err := config.FetchResource(loader.BaseConfig().Paths.Icon)

			if err != nil {
				log.Warn("failed to load OG image icon, OG images will not include an icon", "error", err)
				return nil, err
			}

			return service.NewOGImageGenerator(loader.BaseConfig().OGImage, loader.BaseConfig().BlogName, icon)
		},
	})
}

func Renderers(loader *config.ConfigLoader, log *slog.Logger) ModuleMapRegistry[RendererModule] {
	return NewModuleMapRegistry(map[string]ModuleCreator[RendererModule]{
		"default": func() (RendererModule, error) {

			tmplData, err := config.FetchResource(loader.BaseConfig().Paths.Template)
			if err != nil {
				log.Error("failed to load template", "error", err)
				return nil, err
			}

			return view.NewRenderer(loader.BaseConfig(), tmplData, log)
		},
	})
}

type AssetFetchRegistry struct {
	cache map[string][]byte
	log   *slog.Logger
}

func NewAssetFetchRegistry(log *slog.Logger) *AssetFetchRegistry {
	return &AssetFetchRegistry{
		cache: make(map[string][]byte),
		log:   log.With("component", "AssetFetchRegistry"),
	}
}

func (r *AssetFetchRegistry) Identity() string {
	return "asset_fetch_registry"
}

func (r *AssetFetchRegistry) Get(name string) ([]byte, error) {
	if data, ok := r.cache[name]; ok {
		return data, nil
	}
	data, err := config.FetchResource(name)
	if err != nil {
		r.log.Warn("failed to fetch asset", "name", name, "error", err)
		return nil, err
	}
	r.cache[name] = data
	return data, nil
}

func (r *AssetFetchRegistry) Has(name string) bool {
	_, err := r.Get(name)
	return err == nil
}
