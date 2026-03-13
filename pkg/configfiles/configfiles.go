package configfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DesiredFile represents a config file on disk that should map to a desired entity in the system.
type DesiredFile[T any] struct {
	Entity T
	Name   string
	Path   string
	Hash   string
}

// ReadJSONFile reads and unmarshals a JSON config file.
func ReadJSONFile[T any](filePath string) (T, error) {
	var input T

	data, err := os.ReadFile(filePath)
	if err != nil {
		return input, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return input, fmt.Errorf("failed to parse config file: %w", err)
	}

	return input, nil
}

// LoadDesired reads JSON config files from dir and returns desired entities keyed by name.
func LoadDesired[T any](dir string, nameFn func(T) string) (map[string]DesiredFile[T], map[string]bool, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, []error{fmt.Errorf("failed to read directory %s: %w", dir, err)}
	}

	result := map[string]DesiredFile[T]{}
	blocked := map[string]bool{}
	var errs []error

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(full)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read config file %s: %w", full, err))
			blocked[full] = true
			continue
		}

		var conf T
		if err := json.Unmarshal(raw, &conf); err != nil {
			errs = append(errs, fmt.Errorf("invalid JSON in %s: %w", full, err))
			blocked[full] = true
			continue
		}

		name := strings.TrimSpace(nameFn(conf))
		if name == "" {
			errs = append(errs, fmt.Errorf("config file %s does not define a valid name", full))
			blocked[full] = true
			continue
		}

		if existing, ok := result[name]; ok {
			errs = append(errs, fmt.Errorf("conflict: duplicate %s defined by %s and %s", name, existing.Path, full))
			blocked[full] = true
			blocked[existing.Path] = true
			continue
		}

		h := sha256.Sum256(raw)
		result[name] = DesiredFile[T]{
			Entity: conf,
			Name:   name,
			Path:   full,
			Hash:   hex.EncodeToString(h[:]),
		}
	}

	return result, blocked, errs
}
