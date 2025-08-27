package configresolver

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// --- Mock Fetcher ---
type mockFetcher struct {
	data map[string]map[string]interface{}
	errs map[string]error
}

func (m *mockFetcher) Fetch(branch, projectPath string) (map[string]interface{}, error) {
	key := branch + ":" + projectPath
	if m.errs != nil {
		if err, ok := m.errs[key]; ok {
			return nil, err
		}
	}
	if cfg, ok := m.data[key]; ok {
		b, _ := json.Marshal(cfg)
		var out map[string]interface{}
		_ = json.Unmarshal(b, &out)
		return out, nil
	}
	return nil, ErrUnknownReference
}

// --- Tests ---
func TestMergeMaps(t *testing.T) {
	a := map[string]interface{}{
		"rules": map[string]interface{}{
			"maxLen": 80,
			"indent": 2,
		},
		"root": true,
	}
	b := map[string]interface{}{
		"rules": map[string]interface{}{
			"indent": 4,
			"quotes": "double",
		},
	}

	expected := map[string]interface{}{
		"rules": map[string]interface{}{
			"maxLen": 80,
			"indent": 4,
			"quotes": "double",
		},
		"root": true,
	}

	got := mergeMaps(a, b)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("mergeMaps failed.\nExpected: %+v\nGot: %+v", expected, got)
	}
}

func TestToStringSlice(t *testing.T) {
	cases := []struct {
		in       interface{}
		expected []string
		ok       bool
	}{
		{"gitlab>foo/bar", []string{"gitlab>foo/bar"}, true},
		{[]interface{}{"a", "b"}, []string{"a", "b"}, true},
		{123, nil, false},
		{[]interface{}{"a", 123}, nil, false},
	}

	for _, c := range cases {
		got, ok := toStringSlice(c.in)
		if ok != c.ok || !reflect.DeepEqual(c.expected, got) {
			t.Errorf("toStringSlice(%v) = (%v,%v), expected (%v,%v)", c.in, got, ok, c.expected, c.ok)
		}
	}
}

func TestResolveConfigStringWithFetcher_Basic(t *testing.T) {
	mock := &mockFetcher{
		data: map[string]map[string]interface{}{
			"main:html-validate/renovate-config": {
				"rules": map[string]interface{}{"indent": 2},
			},
			"dev:html-validate/base": {
				"rules": map[string]interface{}{"quotes": "single"},
			},
		},
	}

	cfg := map[string]interface{}{
		"extends": []interface{}{"gitlab>html-validate/renovate-config", "gitlab@dev>html-validate/base"},
		"rules":   map[string]interface{}{"maxLen": 100.0}, // use float64
	}

	got, err := resolveWithFetcher(cfg, map[string]bool{}, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]interface{}{
		"rules": map[string]interface{}{
			"indent": 2.0,   // float64
			"quotes": "single",
			"maxLen": 100.0, // float64
		},
	}

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("resolved config mismatch.\nExpected: %+v\nGot: %+v", expected, got)
	}
}

func TestResolveConfigStringWithFetcher_Circular(t *testing.T) {
	mock := &mockFetcher{
		data: map[string]map[string]interface{}{
			"main:foo/bar": {
				"extends": []interface{}{"gitlab>foo/bar"},
			},
		},
	}

	cfg := map[string]interface{}{
		"extends": []interface{}{"gitlab>foo/bar"},
	}

	_, err := resolveWithFetcher(cfg, map[string]bool{}, mock)
	if err == nil || !errors.Is(err, ErrCircularReference) {
		t.Errorf("expected ErrCircularReference, got %v", err)
	}
}

func TestResolveConfigStringWithFetcher_UnknownReference(t *testing.T) {
	mock := &mockFetcher{}
	cfg := map[string]interface{}{
		"extends": []interface{}{"gitlab>not/exist"},
	}
	_, err := resolveWithFetcher(cfg, map[string]bool{}, mock)
	if err == nil || !errors.Is(err, ErrUnknownReference) {
		t.Errorf("expected ErrUnknownReference, got %v", err)
	}
}

func TestResolveConfigStringWithFetcher_InvalidJSON(t *testing.T) {
	_, err := ResolveConfigStringWithFetcher("{ invalid json", &mockFetcher{})
	if err == nil {
		t.Errorf("expected JSON error, got nil")
	}
}

func TestResolveConfigStringWithFetcher_UnsupportedExtends(t *testing.T) {
	mock := &mockFetcher{}
	cfg := map[string]interface{}{
		"extends": []interface{}{"npm>package"},
	}
	_, err := resolveWithFetcher(cfg, map[string]bool{}, mock)
	if err == nil {
		t.Errorf("expected unsupported extend error, got nil")
	}
}
