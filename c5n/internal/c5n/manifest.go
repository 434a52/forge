package c5n

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest is c5n.yaml: which targets to emit, and where the sources live. It is the
// single authoritative statement of the build's outputs (what the drift-guard checks).
type Manifest struct {
	Targets map[string]Target `yaml:"targets"`
	Sources Sources           `yaml:"sources"`
}

// Target is one output language: where generated files go, plus per-language options.
// Namespace is C#-only; the TS target ignores it.
type Target struct {
	Out       string `yaml:"out"`
	Namespace string `yaml:"namespace"`
}

// Sources are the glob patterns for the schema and data files (relative to the manifest).
type Sources struct {
	Schema string `yaml:"schema"`
	Data   string `yaml:"data"`
}

// LoadManifest reads and parses <root>/c5n.yaml.
func LoadManifest(root string) (*Manifest, error) {
	path := filepath.Join(root, "c5n.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, nil
}
