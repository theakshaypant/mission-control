package sources

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/theakshaypant/mission-control/internal/config"
	"github.com/theakshaypant/mission-control/internal/core"
)

// Factory creates a Source from a user-defined name and raw config map.
// Each source package registers a Factory via Register in its init() function.
type Factory func(name string, raw map[string]any) (core.Source, error)

var registry = map[string]Factory{}

// Register associates a source kind (e.g. "github") with a Factory.
// Intended to be called from source package init() functions.
func Register(kind string, f Factory) {
	registry[kind] = f
}

// LoadAll instantiates all sources defined in cfg.
// Source packages must be imported for their init() to fire and register
// their factory. Typically done in main via blank imports.
func LoadAll(cfg *config.AppConfig) ([]core.Source, error) {
	sources := make([]core.Source, 0, len(cfg.Sources))
	for _, raw := range cfg.Sources {
		factory, ok := registry[raw.Type]
		if !ok {
			return nil, fmt.Errorf("unknown source type %q (name: %s)", raw.Type, raw.Name)
		}
		src, err := factory(raw.Name, raw.Extra)
		if err != nil {
			return nil, fmt.Errorf("loading source %q: %w", raw.Name, err)
		}
		sources = append(sources, src)
	}
	return sources, nil
}

// UnmarshalRaw is a helper for factory functions: it re-marshals the raw
// map to YAML and unmarshals it into target (a pointer to a typed config struct).
func UnmarshalRaw(raw map[string]any, target any) error {
	data, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("re-marshaling raw config: %w", err)
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}
	return nil
}
