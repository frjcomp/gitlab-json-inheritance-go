package configresolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ResolveConfigStringWithFetcher resolves a JSON config string using a Fetcher.
func ResolveConfigStringWithFetcher(configJSON string, fetcher Fetcher) (map[string]interface{}, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	visited := map[string]bool{}
	return resolveWithFetcher(cfg, visited, fetcher)
}

func resolveWithFetcher(cfg map[string]interface{}, visited map[string]bool, fetcher Fetcher) (map[string]interface{}, error) {
	if fetcher == nil {
		return nil, errors.New("fetcher must not be nil")
	}

	extendsRaw, ok := cfg["extends"]
	if !ok {
		return cfg, nil
	}

	extendsList, ok := toStringSlice(extendsRaw)
	if !ok {
		return nil, errors.New("extends must be a string or array of strings")
	}

	merged := map[string]interface{}{}

	for _, ext := range extendsList {
		if visited[ext] {
			return nil, fmt.Errorf("%w: %s", ErrCircularReference, ext)
		}
		visited[ext] = true

		var extCfg map[string]interface{}
		if strings.HasPrefix(ext, "gitlab") {
			branch := "main"
			project := ""

			// Supported syntaxes:
			// - gitlab>namespace/project
			// - gitlab@branch>namespace/project
			if strings.HasPrefix(ext, "gitlab@") {
				rest := strings.TrimPrefix(ext, "gitlab@")
				parts := strings.SplitN(rest, ">", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid extends syntax: %s", ext)
				}
				branch = parts[0]
				project = parts[1]
			} else {
				project = strings.TrimPrefix(ext, "gitlab>")
			}

			var err error
			extCfg, err = fetcher.Fetch(branch, project)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupported extend: %s", ext)
		}

		// Recursively resolve nested extends on the fetched config
		if subExtRaw, ok := extCfg["extends"]; ok {
			subExtCfg := map[string]interface{}{"extends": subExtRaw}
			resolvedSub, err := resolveWithFetcher(subExtCfg, visited, fetcher)
			if err != nil {
				return nil, err
			}
			extCfg = mergeMaps(resolvedSub, extCfg)
		}

		merged = mergeMaps(merged, extCfg)
	}

	// Finally, overlay the base config
	delete(cfg, "extends")
	merged = mergeMaps(merged, cfg)
	return merged, nil
}

// mergeMaps recursively merges two maps. Values from b override values from a.
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, vb := range b {
		if va, ok := out[k]; ok {
			mapA, okA := va.(map[string]interface{})
			mapB, okB := vb.(map[string]interface{})
			if okA && okB {
				out[k] = mergeMaps(mapA, mapB)
				continue
			}
		}
		out[k] = vb
	}
	return out
}

// toStringSlice converts a string or []interface{} (strings) into []string.
func toStringSlice(raw interface{}) ([]string, bool) {
	switch v := raw.(type) {
	case string:
		return []string{v}, true
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}
