package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadRaw reads the on-disk override tree; a missing file yields an empty tree, never an error.
func LoadRaw(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]interface{}{}, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if raw == nil {
		raw = map[string]interface{}{}
	}
	return raw, nil
}

// SaveRaw pretty-prints the override tree to path, creating parent directories as needed.
func SaveRaw(path string, raw map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Load reads the override tree at path and merges it onto Defaults.
func Load(path string) (Config, error) {
	raw, err := LoadRaw(path)
	if err != nil {
		return Config{}, err
	}
	return Merge(raw), nil
}

// Merge overlays raw onto the defaults and returns the effective Config.
func Merge(raw map[string]interface{}) Config {
	base := toMap(Defaults())
	merged := deepMerge(base, raw)
	var cfg Config
	data, _ := json.Marshal(merged)
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func toMap(v interface{}) map[string]interface{} {
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

func deepMerge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		if bv, ok := result[k]; ok {
			if bm, ok1 := bv.(map[string]interface{}); ok1 {
				if ov, ok2 := v.(map[string]interface{}); ok2 {
					result[k] = deepMerge(bm, ov)
					continue
				}
			}
		}
		result[k] = v
	}
	return result
}
