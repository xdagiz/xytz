package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "charm.land/log/v2"
	"gopkg.in/yaml.v3"
)

func ParseConfig(location Location) (ResolvedConfig, error) {
	overridePath := resolveOverrideConfigPath(location)

	globalReadPath, globalWritePath, globalErr := discoverGlobalConfigPaths()

	globalMap := map[string]any{}
	if globalErr != nil {
		log.Warn("could not discover global config, using defaults", "err", globalErr)
		defaultsData, marshalErr := yaml.Marshal(GetDefault())
		if marshalErr == nil {
			if unmarshalErr := yaml.Unmarshal(defaultsData, &globalMap); unmarshalErr != nil {
				globalMap = map[string]any{}
			}
		}
	} else {
		loadedMap, err := loadConfigMap(globalReadPath)
		if err != nil {
			log.Warn("could not parse global config file, using defaults", "path", globalReadPath, "err", err)
			defaultsData, marshalErr := yaml.Marshal(GetDefault())
			if marshalErr == nil {
				if unmarshalErr := yaml.Unmarshal(defaultsData, &globalMap); unmarshalErr != nil {
					globalMap = map[string]any{}
				}
			}
		} else {
			globalMap = loadedMap
		}
	}

	mergedMap := cloneMap(globalMap)
	if overridePath != "" {
		overrideMap, err := loadConfigMap(overridePath)
		if err != nil {
			return ResolvedConfig{}, fmt.Errorf("failed parsing explicit config %s: %w", overridePath, err)
		}

		mergedMap = mergeMaps(mergedMap, overrideMap)
	}

	mergedData, err := yaml.Marshal(mergedMap)
	if err != nil {
		return ResolvedConfig{}, fmt.Errorf("failed marshalling merged config: %w", err)
	}

	cfg, err := decodeStrictConfig(mergedData)
	if err != nil {
		if overridePath != "" {
			return ResolvedConfig{}, fmt.Errorf("failed validating merged config with explicit override %s: %w", overridePath, err)
		}

		log.Warn("could not validate merged global config, using defaults", "err", err)
		cfg = GetDefault()
	}

	effectivePath := globalWritePath
	if overridePath != "" {
		effectivePath = overridePath
	}

	return ResolvedConfig{
		Config:        cfg,
		GlobalPath:    globalWritePath,
		OverridePath:  overridePath,
		EffectivePath: effectivePath,
	}, nil
}

func resolveOverrideConfigPath(location Location) string {
	if location.ConfigFlag != "" {
		return location.ConfigFlag
	}

	if path := os.Getenv(ConfigEnvVar); path != "" {
		return path
	}

	return ""
}

func discoverGlobalConfigPaths() (readPath string, writePath string, err error) {
	configDir := GetConfigDir()
	if configDir == "" {
		return "", "", errors.New("empty config directory")
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", "", err
	}

	yamlPath := filepath.Join(configDir, ConfigFileName)
	ymlPath := filepath.Join(configDir, ConfigAltFileName)

	switch {
	case fileExists(yamlPath):
		return yamlPath, yamlPath, nil
	case fileExists(ymlPath):
		return ymlPath, yamlPath, nil
	default:
		defaultCfg := GetDefault()
		if err := defaultCfg.SaveToPath(yamlPath); err != nil {
			return "", "", err
		}

		return yamlPath, yamlPath, nil
	}
}

func loadConfigMap(configPath string) (map[string]any, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var mapData map[string]any
	if err := yaml.Unmarshal(data, &mapData); err != nil {
		return nil, err
	}

	if mapData == nil {
		mapData = map[string]any{}
	}

	return mapData, nil
}

func decodeStrictConfig(data []byte) (*Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	applyOmittedBooleanDefaults(&cfg, data)
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func mergeMaps(base map[string]any, override map[string]any) map[string]any {
	result := cloneMap(base)
	for key, overrideValue := range override {
		if baseValue, ok := result[key]; ok {
			baseMap, baseIsMap := asStringAnyMap(baseValue)
			overrideMap, overrideIsMap := asStringAnyMap(overrideValue)
			if baseIsMap && overrideIsMap {
				result[key] = mergeMaps(baseMap, overrideMap)
				continue
			}
		}
		result[key] = overrideValue
	}

	return result
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := asStringAnyMap(value); ok {
			out[key] = cloneMap(nested)
			continue
		}
		out[key] = value
	}

	return out
}

func asStringAnyMap(value any) (map[string]any, bool) {
	casted, ok := value.(map[string]any)
	if ok {
		return casted, true
	}

	return nil, false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
