package registry

import (
	"os"

	"gopkg.in/yaml.v3"
)

// FileLoader loads instruments from a YAML file.
type FileLoader struct {
	Path string
}

// Load reads instruments from the configured file path.
func (l FileLoader) Load() ([]Instrument, error) {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return nil, err
	}
	var instruments []Instrument
	if err := yaml.Unmarshal(data, &instruments); err != nil {
		return nil, err
	}
	return instruments, nil
}

// LoadFromFile is a convenience for one-off loading.
func LoadFromFile(path string) ([]Instrument, error) {
	return FileLoader{Path: path}.Load()
}
