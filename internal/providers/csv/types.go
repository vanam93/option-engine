package csv

import "github.com/vanam-gangireddy/option-engine/internal/domain/market"

const providerName = "csv"

// csvColumn names expected in historical CSV files.
var csvColumns = []string{"date", "open", "high", "low", "close", "volume"}

// timeframeToFilename maps internal timeframes to on-disk CSV filenames.
var timeframeToFilename = map[market.Timeframe]string{
	market.TF3m:  "3min.csv",
	market.TF5m:  "5min.csv",
	market.TF15m: "15min.csv",
	market.TF1h:  "1h.csv",
	market.TF1d:  "1d.csv",
}
