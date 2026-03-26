package modules

import (
	"log/slog"

	"github.com/5000K/5000blogs/config"
)

type RuntimeContext struct {
	Converters        ModuleRegistry[ConverterModule]
	PostRepositories  ModuleRegistry[PostIndexerModule]
	PostSources       ModuleRegistry[PostSourceModule]
	OGImageGenerators ModuleRegistry[OGImageGeneratorModule]
	Renderers         ModuleRegistry[RendererModule]
	Assets            ModuleRegistry[[]byte]
	Loader            *config.ConfigLoader
	Log               *slog.Logger
}
