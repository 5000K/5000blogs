package modules

import (
	"github.com/5000K/5000blogs/config"
	"github.com/5000K/5000blogs/core"
	"github.com/5000K/5000blogs/service"
	"github.com/5000K/5000blogs/util"
	"github.com/5000K/5000blogs/view"
)

type ConverterModule interface {
	service.Converter

	Initialize() error
}

type PostIndexerModule interface {
	service.PostIndexer

	Initialize(source service.PostSource, converter service.Converter) error
}

type PostSourceModule interface {
	service.PostSource

	Initialize(conf config.SourceConfig) error
}

type OGImageGeneratorModule interface {
	service.OGImageGenerator

	Initialize() error
}

type RendererModule interface {
	view.Renderer

	Initialize() error
}

type Module interface{}
type ModuleConfig interface{}

type ModuleRegistry[T Module] interface {
	Identity() string
	Get(name string) (T, error)
	Has(name string) bool
}

type ModuleCreator[T Module] func() (T, error)

type ModuleMapRegistry[T Module] map[string]ModuleCreator[T]

func (r ModuleMapRegistry[T]) Identity() string {
	typename := util.TypeNameOf[T]()
	return "map_registry" + "->" + typename
}

func (r ModuleMapRegistry[T]) Get(name string) (T, error) {
	creator, ok := r[name]
	if !ok {
		var def T
		return def, core.ErrModuleNotFound
	}
	return creator()
}

func (r ModuleMapRegistry[T]) Has(name string) bool {
	_, ok := r[name]
	return ok
}

func NewModuleMapRegistry[T Module](creators map[string]ModuleCreator[T]) ModuleMapRegistry[T] {
	return ModuleMapRegistry[T](creators)
}

// a collection of repositories, itself implementing the repository
type RegistryCollection[T Module] struct {
	repositories []ModuleRegistry[T]
}

func NewRegistryCollection[T Module]() *RegistryCollection[T] {
	return &RegistryCollection[T]{
		repositories: []ModuleRegistry[T]{},
	}
}

func (r *RegistryCollection[T]) Add(repository ModuleRegistry[T]) error {
	for _, repo := range r.repositories {
		if repo.Identity() == repository.Identity() {
			return core.ErrRepositoryWithSameIdentity
		}
	}

	r.repositories = append(r.repositories, repository)
	return nil
}

func (r *RegistryCollection[T]) Get(name string) (T, error) {
	for _, repo := range r.repositories {
		if repo.Has(name) {
			return repo.Get(name)
		}
	}

	var def T

	return def, core.ErrModuleNotFound
}

func (r *RegistryCollection[T]) Has(name string) bool {
	for _, repo := range r.repositories {
		if repo.Has(name) {
			return true
		}
	}

	return false
}
