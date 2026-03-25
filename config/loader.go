package config

import (
	"bytes"
	"os"

	"github.com/5000K/5000blogs/core"
	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

type ConfigLoader struct {
	base map[string]interface{}
	core *Config
}

// for any generic configs that rely on a type being set (e.g. converter)
// plugins should then parse their respective config on their own however they want to handle it.
type TypeConfig struct {
	Type string `yaml:"type"`
}

func readEnvOrDefault(variable string, def string) string {
	str := os.Getenv(variable)

	if len(str) == 0 {
		return def
	}

	return str
}

func NewConfigLoader() (*ConfigLoader, error) {
	path := readEnvOrDefault("CONFIG_PATH", "config.yml")
	raw, err := FetchResource(path)

	if err != nil {
		return nil, err
	}

	var base map[string]interface{}
	yaml.Unmarshal(raw, &base)

	core, err := Get()

	if err != nil {
		return nil, err
	}

	err = core.Validate()

	if err != nil {
		return nil, err
	}

	return &ConfigLoader{
		base: base,
		core: core,
	}, nil
}

// BaseConfig reads the core 5000blogs config. Use for compatibility, not for main source of truth.
func (loader *ConfigLoader) BaseConfig() Config {
	return *loader.core
}

func (loader *ConfigLoader) Load(prefix string, target any) error {
	if err := cleanenv.ReadEnv(target); err != nil {
		return err
	}

	sub, err := loader.getRawSub(prefix)

	if err != nil {
		return nil // rely on values set by ReadEnv
	}

	return cleanenv.ParseYAML(bytes.NewReader(sub), target)
}

func (loader *ConfigLoader) LoadSlice(prefix string, target any) error {
	sub, err := loader.getRawSub(prefix)

	if err != nil {
		return nil // rely on values set by ReadEnv
	}

	return cleanenv.ParseYAML(bytes.NewReader(sub), target)
}

func (loader *ConfigLoader) getRawSub(prefix string) ([]byte, error) {

	if prefix == "" {
		return yaml.Marshal(loader.base)
	}

	if v, ok := loader.base[prefix]; ok {
		return yaml.Marshal(v)
	} else {
		return nil, core.ErrSubConfigNotFound
	}
}
