package walkforward

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/experiments"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

func TestMetadataExtractionFromParameters(t *testing.T) {
	params := experiments.ParameterSet{
		"walkforward_id": "wf-test",
		"window_index":   0,
		"phase":          "training",
		"run_id":         "run-1",
	}
	serialized := experiments.SerializeParameters(params)
	require.NotEmpty(t, serialized)
	require.Equal(t, "wf-test", metadataString(serialized, "walkforward_id"))
	require.Equal(t, 0, metadataInt(serialized, "window_index"))
	require.Equal(t, "run-1", experiments.RunIDFromParameters(serialized))

	update := optimization.OptimizationUpdated{Parameters: serialized, Score: 0.9}
	session := &windowSession{
		walkForwardID: "wf-test",
		windowIndex:   0,
		phase:         "training",
		training:      newTrainingCollector([]experiments.ExperimentRun{{RunID: "run-1"}}),
	}
	session.handle(update)
	require.Len(t, session.training.snapshotResults(), 1)
}

func TestTrainingCollectorAcceptsRunnerOutput(t *testing.T) {
	window := Window{
		Index:           0,
		WalkForwardID:   "wf-1",
		TrainStart:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		TrainEnd:        time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		ValidationStart: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		ValidationEnd:   time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC),
	}
	run := ExperimentRunFixture(window)
	params := experiments.SerializeParameters(run.Parameters)
	update := optimization.OptimizationUpdated{
		Parameters: params,
		Score:      0.5,
	}
	collector := newTrainingCollector([]experiments.ExperimentRun{run})
	session := &windowSession{
		walkForwardID: "wf-1",
		windowIndex:   0,
		phase:         "training",
		training:      collector,
	}
	session.handle(update)
	require.Len(t, collector.snapshotResults(), 1)
}

func ExperimentRunFixture(window Window) experiments.ExperimentRun {
	runID := experiments.GenerateRunID()
	run := experiments.ExperimentRun{
		ExperimentID: experiments.GenerateExperimentID(),
		RunID:        runID,
		Strategy:     "trend_following",
		Symbol:       "NIFTY",
		Timeframe:    "5m",
		Parameters: experiments.ParameterSet{
			"run_id": runID,
			"ema_fast": 5,
		},
	}
	return enrichRun(window, run, "training")
}

func TestBusDeliversOptimizationToSubscriber(t *testing.T) {
	bus := eventbus.New()
	sub := bus.Subscribe(16, func(evt events.Event) bool {
		return evt.Type == events.OptimizationUpdated
	})
	defer sub.Close()

	params := experiments.SerializeParameters(experiments.ParameterSet{
		"phase":  "training",
		"run_id": "run-1",
	})
	payload, err := json.Marshal(optimization.OptimizationUpdated{Parameters: params, Score: 0.5})
	require.NoError(t, err)
	bus.Publish(events.Event{Type: events.OptimizationUpdated, Payload: payload})

	select {
	case evt := <-sub.C:
		update, ok := parseOptimizationUpdate(evt.Payload)
		require.True(t, ok)
		require.Equal(t, "training", metadataString(update.Parameters, "phase"))
	case <-time.After(time.Second):
		t.Fatal("expected event on subscription channel")
	}
}

func TestOptimizationUpdatedRoundTrip(t *testing.T) {
	params := experiments.SerializeParameters(experiments.ParameterSet{
		"walkforward_id": "wf-1",
		"run_id":         "r1",
	})
	payload, err := json.Marshal(optimization.OptimizationUpdated{Parameters: params, Score: 1})
	require.NoError(t, err)
	var update optimization.OptimizationUpdated
	require.NoError(t, json.Unmarshal(payload, &update))
	require.Equal(t, "wf-1", metadataString(update.Parameters, "walkforward_id"))
}
