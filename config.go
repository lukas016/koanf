package koanf

import (
	"fmt"
	"os"
	"strings"

	koanfYaml "github.com/knadh/koanf/parsers/yaml"
	koanfFile "github.com/knadh/koanf/providers/file"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
)

var (
	argDumpDefaultConfig bool
	argDumpLoadedConfig  bool
	argValidateConfig    bool
)

func init() {
	pflag.BoolVar(&argDumpLoadedConfig, "config-dump-loaded", false, "it will print loaded configuration from all sources and end program")
	pflag.BoolVar(&argDumpDefaultConfig, "config-dump-default", false, "it will print default configuration and end program")
	pflag.BoolVar(&argValidateConfig, "config-validate", false, "load and validate configuration")
}

func Load(paths []string, envPrefix string, remapKey map[string]string, cfg any) error {
	if argDumpDefaultConfig {
		err := printConfig(cfg)
		if err != nil {
			return err
		}
		os.Exit(0)
	}

	err := loadSources(paths, envPrefix, remapKey, cfg)
	if err != nil {
		return err
	}

	if argDumpLoadedConfig {
		err := printConfig(cfg)
		if err != nil {
			return err
		}
		os.Exit(0)
	}

	validate := validator.New()
	err = validate.Struct(cfg)
	if argValidateConfig {
		if err == nil {
			os.Exit(0)
		}

		fmt.Println(err)
		os.Exit(1)
	}

	return err
}

func loadSources(paths []string, envPrefix string, remapKey map[string]string, cfg any) error {
	configLoader := koanf.New(".")

	for _, filePath := range paths {
		err := configLoader.Load(koanfFile.Provider(filePath), koanfYaml.Parser())
		if err != nil {
			return err
		}
	}

	configLoader.Load(env.Provider(envPrefix, ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, envPrefix)), "_", ".", -1)
	}), nil)

	configLoader.Load(posflag.ProviderWithFlag(pflag.CommandLine, ".", configLoader, func(fl *pflag.Flag) (string, interface{}) {
		newKey, ok := remapKey[fl.Name]
		if !ok {
			newKey = strings.ReplaceAll(fl.Name, "-", ".")
		}
		if fl.Changed {
			return newKey, posflag.FlagVal(pflag.CommandLine, fl)
		}

		// Discard default values fro key
		return "", nil
	}), nil)

	return configLoader.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{Tag: "yaml"})
}

func printConfig(cfg any) error {
	return yaml.NewEncoder(os.Stdout).Encode(cfg)
}
