package run

import (
	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/core"
	"github.com/5000K/5000blogs/modules"
	"github.com/5000K/5000blogs/server"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/view"
)

type Runtime struct {
	Renderer         view.Renderer
	OGImageGenerator service.OGImageGenerator
	PostIndexer      service.PostIndexer
	Favicon          []byte
}

func Run(ctx modules.RuntimeContext) error {
	baseConf := ctx.Loader.BaseConfig()

	source, err := constructSource(ctx)

	if err != nil {
		return err
	}

	converter, err := constructConverter(ctx)

	if err != nil {
		return err
	}

	renderer, err := constructRenderer(ctx)

	if err != nil {
		return err
	}

	generator, err := constructImageGenerator(ctx)

	if err != nil {
		return err
	}

	indexer, err := constructPostIndexer(ctx, source, converter)

	if err != nil {
		return err
	}

	err = indexer.Start()
	defer indexer.Stop()

	if err != nil {
		return err
	}

	favicon, err := ctx.Assets.Get(baseConf.Paths.Icon)

	if err != nil {
		return err
	}

	modules, err := getModules(ctx.Loader, indexer, renderer, generator, favicon)

	if err != nil {
		return err
	}

	return server.Listen(ctx.Loader, modules)
}

func getModules(loader *config.ConfigLoader, indexer service.PostIndexer, renderer view.Renderer, generator service.OGImageGenerator, favicon []byte) ([]server.ServerModule, error) {
	var modules []server.ServerModule

	// TODO: make this dynamic based on config
	modules = append(modules, server.NewHealthModule())
	modules = append(modules, server.NewApiModule(indexer))
	modules = append(modules, server.NewHomeModule(indexer, generator, renderer))
	modules = append(modules, server.NewXmlFeedModule(indexer))
	modules = append(modules, server.NewIconModule(favicon))
	modules = append(modules, server.NewPlainModule(indexer))
	modules = append(modules, server.NewPostFeedModule(indexer, renderer))
	modules = append(modules, server.NewDynamicModule(indexer, generator, renderer))

	return modules, nil
}

func constructPostIndexer(ctx modules.RuntimeContext, source service.PostSource, converter service.Converter) (service.PostIndexer, error) {
	var conf = config.TypeConfig{
		Type: "bleve",
	}

	err := ctx.Loader.Load("repository", &conf)

	if err != nil {
		return nil, err
	}

	if !ctx.PostRepositories.Has(conf.Type) {
		ctx.Log.Warn("couldn't find post repository", "type", conf.Type)
		return nil, core.ErrModuleNotFound
	}

	repo, err := ctx.PostRepositories.Get(conf.Type)

	if err != nil {
		ctx.Log.Warn("error getting post repository", "type", conf.Type, "error", err)
		return nil, err
	}

	err = repo.Initialize(source, converter)

	if err != nil {
		ctx.Log.Warn("error initializing post repository", "type", conf.Type, "error", err)
		return nil, err
	}

	ctx.Log.Debug("loaded and initialized post repository", "type", conf.Type)

	return repo, nil
}

func constructImageGenerator(ctx modules.RuntimeContext) (service.OGImageGenerator, error) {
	var conf = config.TypeConfig{
		Type: "default",
	}

	err := ctx.Loader.Load("og_image", &conf)

	if err != nil {
		return nil, err
	}

	if !ctx.OGImageGenerators.Has(conf.Type) {
		ctx.Log.Warn("couldn't find image generator", "type", conf.Type)
		return nil, core.ErrModuleNotFound
	}

	generator, err := ctx.OGImageGenerators.Get(conf.Type)

	if err != nil {
		ctx.Log.Warn("error getting image generator", "type", conf.Type, "error", err)
		return nil, err
	}

	err = generator.Initialize()

	if err != nil {
		ctx.Log.Warn("error initializing image generator", "type", conf.Type, "error", err)
		return nil, err
	}

	ctx.Log.Debug("loaded and initialized image generator", "type", conf.Type)

	return generator, nil
}

func constructRenderer(ctx modules.RuntimeContext) (view.Renderer, error) {
	var conf = config.TypeConfig{
		Type: "default",
	}

	err := ctx.Loader.Load("renderer", &conf)

	if err != nil {
		return nil, err
	}

	if !ctx.Renderers.Has(conf.Type) {
		ctx.Log.Warn("couldn't find renderer", "type", conf.Type)
		return nil, core.ErrModuleNotFound
	}

	renderer, err := ctx.Renderers.Get(conf.Type)

	if err != nil {
		ctx.Log.Warn("error getting renderer", "type", conf.Type, "error", err)
		return nil, err
	}

	err = renderer.Initialize()

	if err != nil {
		ctx.Log.Warn("error initializing renderer", "type", conf.Type, "error", err)
		return nil, err
	}

	ctx.Log.Debug("loaded and initialized renderer", "type", conf.Type)

	return renderer, nil
}

func constructConverter(ctx modules.RuntimeContext) (service.Converter, error) {
	var conf = config.TypeConfig{
		Type: "goldmark",
	}

	err := ctx.Loader.Load("converter", &conf)

	if err != nil {
		return nil, err
	}

	if !ctx.Converters.Has(conf.Type) {
		ctx.Log.Warn("couldn't find converter", "type", conf.Type)
		return nil, core.ErrModuleNotFound
	}

	converter, err := ctx.Converters.Get(conf.Type)

	if err != nil {
		ctx.Log.Warn("error getting converter", "type", conf.Type, "error", err)
		return nil, err
	}

	err = converter.Initialize()

	if err != nil {
		ctx.Log.Warn("error initializing converter", "type", conf.Type, "error", err)
		return nil, err
	}

	ctx.Log.Debug("loaded and initialized converter", "type", conf.Type)

	return converter, nil
}

func constructSource(ctx modules.RuntimeContext) (service.PostSource, error) {
	var sourceConfigs []config.SourceConfig

	err := ctx.Loader.LoadSlice("sources", &sourceConfigs)

	if err != nil {
		return nil, err
	}

	sourceModules := make([]service.PostSource, 0, len(sourceConfigs)+1)

	for _, conf := range sourceConfigs {
		if !ctx.PostSources.Has(conf.Type) {
			ctx.Log.Warn("couldn't find post source", "type", conf.Type)
			continue
		}

		src, err := ctx.PostSources.Get(conf.Type)

		if err != nil {
			ctx.Log.Warn("error loading post source", "type", conf.Type, "error", err)
			continue
		}

		err = src.Initialize(conf)

		if err != nil {
			ctx.Log.Warn("error initializing post source", "type", conf.Type, "error", err)
			continue
		}

		sourceModules = append(sourceModules, src)

		ctx.Log.Debug("loaded and initialized post source", "type", conf.Type)
	}

	sourceModules = append(sourceModules, service.NewBuiltinSource())

	ctx.Log.Debug("loaded and initialized layered post source", "sources_count", len(sourceModules))

	return service.NewLayeredSource(sourceModules...), nil
}
