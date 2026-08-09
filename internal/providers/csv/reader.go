package csv

import (
	"fmt"
	"os"
)

// OpenDataFile opens an iterator for the configured CSV data file.
func OpenDataFile(cfg Config) (*Iterator, error) {
	path := cfg.DataFilePath()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("csv data path is a directory: %s", path)
	}
	reader := NewReader(path)
	return reader.OpenIterator(cfg.BatchSize)
}
