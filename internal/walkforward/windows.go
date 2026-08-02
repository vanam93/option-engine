package walkforward

import (
	"time"

	"github.com/google/uuid"
)

// Period is an inclusive-start, exclusive-end time range.
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Window describes a single walk-forward train/validation split.
type Window struct {
	Index            int
	WalkForwardID    string
	TrainStart       time.Time
	TrainEnd         time.Time
	ValidationStart  time.Time
	ValidationEnd    time.Time
}

// TrainPeriod returns the training period for this window.
func (w Window) TrainPeriod() Period {
	return Period{Start: w.TrainStart, End: w.TrainEnd}
}

// ValidationPeriod returns the validation period for this window.
func (w Window) ValidationPeriod() Period {
	return Period{Start: w.ValidationStart, End: w.ValidationEnd}
}

// GenerateWalkForwardID creates a new walk-forward batch identifier.
func GenerateWalkForwardID() string {
	return uuid.NewString()
}

// GenerateWindows builds rolling train/validation windows over a data range.
//
// Each window advances by stepDays. When stepDays < trainWindowDays the training
// periods overlap. Validation immediately follows training with no gap.
func GenerateWindows(walkForwardID string, dataStart, dataEnd time.Time, trainDays, validationDays, stepDays int) []Window {
	if walkForwardID == "" {
		walkForwardID = GenerateWalkForwardID()
	}
	if trainDays < 1 || validationDays < 1 || stepDays < 1 {
		return nil
	}
	if dataEnd.Before(dataStart) {
		return nil
	}

	var windows []Window
	index := 0
	for {
		trainStart := dataStart.AddDate(0, 0, index*stepDays)
		trainEnd := trainStart.AddDate(0, 0, trainDays)
		validationStart := trainEnd
		validationEnd := validationStart.AddDate(0, 0, validationDays)

		if validationEnd.After(dataEnd) {
			break
		}

		windows = append(windows, Window{
			Index:           index,
			WalkForwardID:   walkForwardID,
			TrainStart:      trainStart,
			TrainEnd:        trainEnd,
			ValidationStart: validationStart,
			ValidationEnd:   validationEnd,
		})
		index++
	}
	return windows
}
