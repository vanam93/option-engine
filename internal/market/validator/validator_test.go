package validator

import (
	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"testing"
	"time"
)

func TestRejectsDuplicateAndOrder(t *testing.T) {
	now := time.Now()
	v := New(Config{MaxAge: time.Minute}, nil)
	tick := market.Tick{ID: uuid.New(), Symbol: "NIFTY", LTP: 1, ProviderTS: now}
	if err := v.Validate(tick, now); err != nil {
		t.Fatal(err)
	}
	if err := v.Validate(tick, now); err == nil {
		t.Fatal("expected duplicate")
	}
	tick.ProviderTS = now.Add(-time.Second)
	if err := v.Validate(tick, now); err == nil {
		t.Fatal("expected order error")
	}
}

func TestReplayModeAllowsHistoricalTicks(t *testing.T) {
	now := time.Now()
	v := New(Config{MaxAge: time.Second, ReplayMode: true}, nil)
	tick := market.Tick{ID: uuid.New(), Symbol: "NIFTY", LTP: 1, ProviderTS: now.Add(-24 * time.Hour)}
	if err := v.Validate(tick, now); err != nil {
		t.Fatalf("replay mode should skip staleness check: %v", err)
	}
}
