package csv

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// ParsedRow holds a validated OHLCV row from a CSV file.
type ParsedRow struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

var timestampLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

// ParseHeader validates the CSV header row.
func ParseHeader(fields []string) error {
	if len(fields) < len(csvColumns) {
		return fmt.Errorf("expected at least %d columns, got %d", len(csvColumns), len(fields))
	}
	for i, expected := range csvColumns {
		if !strings.EqualFold(strings.TrimSpace(fields[i]), expected) {
			return fmt.Errorf("column %d: expected %q, got %q", i+1, expected, fields[i])
		}
	}
	return nil
}

// ParseRow converts a CSV record into a parsed OHLCV row.
func ParseRow(line int64, fields []string) (ParsedRow, error) {
	if len(fields) < len(csvColumns) {
		return ParsedRow{}, &ParseError{
			Line:    line,
			Message: fmt.Sprintf("expected %d fields, got %d", len(csvColumns), len(fields)),
		}
	}

	ts, err := parseTimestamp(strings.TrimSpace(fields[0]))
	if err != nil {
		return ParsedRow{}, &ParseError{Line: line, Field: "date", Message: err.Error()}
	}

	open, err := parseFloat(fields[1], "open")
	if err != nil {
		return ParsedRow{}, &ParseError{Line: line, Field: "open", Message: err.Error()}
	}
	high, err := parseFloat(fields[2], "high")
	if err != nil {
		return ParsedRow{}, &ParseError{Line: line, Field: "high", Message: err.Error()}
	}
	low, err := parseFloat(fields[3], "low")
	if err != nil {
		return ParsedRow{}, &ParseError{Line: line, Field: "low", Message: err.Error()}
	}
	closePx, err := parseFloat(fields[4], "close")
	if err != nil {
		return ParsedRow{}, &ParseError{Line: line, Field: "close", Message: err.Error()}
	}
	volume, err := parseFloat(fields[5], "volume")
	if err != nil {
		return ParsedRow{}, &ParseError{Line: line, Field: "volume", Message: err.Error()}
	}

	return ParsedRow{
		Timestamp: ts,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closePx,
		Volume:    volume,
	}, nil
}

// RowToCandle maps a parsed row into a domain candle.
func RowToCandle(row ParsedRow, symbol string, tf market.Timeframe) market.Candle {
	return market.Candle{
		Symbol:    symbol,
		Timeframe: tf,
		Open:      row.Open,
		High:      row.High,
		Low:       row.Low,
		Close:     row.Close,
		Volume:    int64(row.Volume),
		OpenTime:  row.Timestamp,
		CloseTime: row.Timestamp,
	}
}

func parseTimestamp(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.UTC
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format: %s", raw)
}

func parseFloat(raw, field string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is empty", field)
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", field, err)
	}
	return v, nil
}
