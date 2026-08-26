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
	Targets     map[string]Target `yaml:"targets"`
	Sources     Sources           `yaml:"sources"`
	Attribution []Attribution     `yaml:"attribution"`
}

// Attribution is a notice that must travel with generated code derived from a licensed
// source, and the paths it applies to.
//
// Declared here rather than in the data files for two reasons. A data file has no free
// top-level key — in the named-collection form every top-level key is a collection name, so a
// reserved one would be a wart. And an obligation belongs with the project's authoritative
// statement of its outputs, beside the targets and the source globs, where it is reviewed
// rather than scattered.
//
// The point of emitting it into the header rather than keeping a NOTICE file is that the
// obligation then travels with the artefact automatically. A generated file that leaves the
// repo carries its own attribution, and nobody has to remember.
type Attribution struct {
	// Match is a source path pattern, in the same two shapes the source globs use: a plain
	// glob ("data/cldr/*.yaml") or a recursive one ("data/cldr/**").
	Match string `yaml:"match"`

	// Notice is the text to reproduce, verbatim, in the header of anything generated from a
	// matching source.
	Notice string `yaml:"notice"`
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
