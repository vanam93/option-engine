package csv

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// Reader opens a CSV file for incremental candle iteration.
type Reader struct {
	path string
}

// NewReader creates a reader for the given CSV file path.
func NewReader(path string) *Reader {
	return &Reader{path: path}
}

// OpenIterator opens the file and returns an iterator positioned after the header.
func (r *Reader) OpenIterator(batchSize int) (*Iterator, error) {
	file, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, r.path)
		}
		return nil, err
	}

	if batchSize <= 0 {
		batchSize = 1000
	}
	buf := bufio.NewReaderSize(file, batchSize*128)
	csvReader := csv.NewReader(buf)
	csvReader.ReuseRecord = true
	csvReader.FieldsPerRecord = -1

	header, err := csvReader.Read()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	if err := ParseHeader(header); err != nil {
		_ = file.Close()
		return nil, err
	}

	return &Iterator{
		path:      r.path,
		file:      file,
		csvReader: csvReader,
		line:      1,
	}, nil
}

// Iterator reads one candle row at a time without loading the full file.
type Iterator struct {
	path        string
	file        *os.File
	csvReader   *csv.Reader
	line        int64
	closed      bool
	parseErrors int64
}

// Path returns the CSV file path being iterated.
func (it *Iterator) Path() string { return it.path }

// Offset returns the current line number in the file.
func (it *Iterator) Offset() int64 { return it.line }

// Close releases the underlying file handle.
func (it *Iterator) Close() error {
	if it.closed {
		return nil
	}
	it.closed = true
	if it.file == nil {
		return nil
	}
	return it.file.Close()
}

// ParseErrors returns the number of malformed rows skipped while iterating.
func (it *Iterator) ParseErrors() int64 { return it.parseErrors }

// Next reads the next valid CSV row. Malformed rows are skipped.
func (it *Iterator) Next() (ParsedRow, bool, error) {
	if it.closed {
		return ParsedRow{}, false, nil
	}
	for {
		record, err := it.csvReader.Read()
		if err == io.EOF {
			return ParsedRow{}, false, nil
		}
		if err != nil {
			return ParsedRow{}, false, err
		}
		it.line++
		if len(record) == 0 || isBlankRecord(record) {
			continue
		}
		row, err := ParseRow(it.line, record)
		if err != nil {
			it.parseErrors++
			continue
		}
		return row, true, nil
	}
}

func isBlankRecord(record []string) bool {
	for _, field := range record {
		if len(field) > 0 {
			return false
		}
	}
	return true
}
