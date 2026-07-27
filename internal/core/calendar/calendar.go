package calendar

import (
	"fmt"
	"time"

	"github.com/option-engine/option-engine/internal/core/clock"
)

// SessionType classifies a trading session.
type SessionType string

const (
	SessionRegular  SessionType = "regular"
	SessionMuhurat  SessionType = "muhurat"
	SessionEarlyClose SessionType = "early_close"
	SessionHoliday  SessionType = "holiday"
)

// Config holds NSE calendar configuration. All values are config-driven.
type Config struct {
	Timezone      string            `mapstructure:"timezone"`
	RegularOpen   string            `mapstructure:"regular_open"`   // HH:MM
	RegularClose  string            `mapstructure:"regular_close"`  // HH:MM
	MuhuratOpen   string            `mapstructure:"muhurat_open"`
	MuhuratClose  string            `mapstructure:"muhurat_close"`
	EarlyCloseAt  string            `mapstructure:"early_close_at"`
	Holidays      []string          `mapstructure:"holidays"`       // YYYY-MM-DD
	MuhuratDays   []string          `mapstructure:"muhurat_days"`   // YYYY-MM-DD
	EarlyCloseDays []string         `mapstructure:"early_close_days"`
	ExpiryWeekday time.Weekday      `mapstructure:"expiry_weekday"` // Thursday = 4
}

// Calendar knows NSE trading sessions, holidays, and expiry rules.
type Calendar struct {
	cfg      Config
	loc      *time.Location
	holidays map[string]struct{}
	muhurat  map[string]struct{}
	early    map[string]struct{}
	clk      clock.Clock
}

// New creates a Calendar from configuration.
func New(cfg Config, clk clock.Clock) (*Calendar, error) {
	tz := cfg.Timezone
	if tz == "" {
		tz = "Asia/Kolkata"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	if cfg.ExpiryWeekday == 0 {
		cfg.ExpiryWeekday = time.Thursday
	}

	c := &Calendar{
		cfg:      cfg,
		loc:      loc,
		holidays: toSet(cfg.Holidays),
		muhurat:  toSet(cfg.MuhuratDays),
		early:    toSet(cfg.EarlyCloseDays),
		clk:      clk,
	}
	return c, nil
}

func toSet(dates []string) map[string]struct{} {
	m := make(map[string]struct{}, len(dates))
	for _, d := range dates {
		m[d] = struct{}{}
	}
	return m
}

func (c *Calendar) dateKey(t time.Time) string {
	return t.In(c.loc).Format("2006-01-02")
}

// IsWeekend returns true for Saturday and Sunday in the exchange timezone.
func (c *Calendar) IsWeekend(t time.Time) bool {
	wd := t.In(c.loc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// IsHoliday returns true for configured NSE holidays.
func (c *Calendar) IsHoliday(t time.Time) bool {
	_, ok := c.holidays[c.dateKey(t)]
	return ok
}

// IsTradingDay returns true when the exchange is open for the given date.
func (c *Calendar) IsTradingDay(t time.Time) bool {
	if c.IsWeekend(t) || c.IsHoliday(t) {
		return false
	}
	return true
}

// SessionTypeFor returns the session classification for a date.
func (c *Calendar) SessionTypeFor(t time.Time) SessionType {
	key := c.dateKey(t)
	if _, ok := c.holidays[key]; ok {
		return SessionHoliday
	}
	if c.IsWeekend(t) {
		return SessionHoliday
	}
	if _, ok := c.muhurat[key]; ok {
		return SessionMuhurat
	}
	if _, ok := c.early[key]; ok {
		return SessionEarlyClose
	}
	return SessionRegular
}

// MarketOpen returns the session open time for the given date.
func (c *Calendar) MarketOpen(t time.Time) (time.Time, error) {
	switch c.SessionTypeFor(t) {
	case SessionHoliday:
		return time.Time{}, fmt.Errorf("market closed: holiday or weekend")
	case SessionMuhurat:
		return c.parseTime(t, c.cfg.MuhuratOpen)
	case SessionEarlyClose, SessionRegular:
		return c.parseTime(t, c.cfg.RegularOpen)
	default:
		return c.parseTime(t, c.cfg.RegularOpen)
	}
}

// MarketClose returns the session close time for the given date.
func (c *Calendar) MarketClose(t time.Time) (time.Time, error) {
	switch c.SessionTypeFor(t) {
	case SessionHoliday:
		return time.Time{}, fmt.Errorf("market closed: holiday or weekend")
	case SessionMuhurat:
		return c.parseTime(t, c.cfg.MuhuratClose)
	case SessionEarlyClose:
		if c.cfg.EarlyCloseAt != "" {
			return c.parseTime(t, c.cfg.EarlyCloseAt)
		}
		return c.parseTime(t, c.cfg.RegularClose)
	default:
		return c.parseTime(t, c.cfg.RegularClose)
	}
}

// IsMarketOpen returns true if the given instant falls within trading hours.
func (c *Calendar) IsMarketOpen(at time.Time) bool {
	if !c.IsTradingDay(at) {
		return false
	}
	open, err := c.MarketOpen(at)
	if err != nil {
		return false
	}
	close, err := c.MarketClose(at)
	if err != nil {
		return false
	}
	local := at.In(c.loc)
	return !local.Before(open) && local.Before(close)
}

// IsExpiryDay returns true if t falls on the configured weekly expiry weekday.
func (c *Calendar) IsExpiryDay(t time.Time) bool {
	return t.In(c.loc).Weekday() == c.cfg.ExpiryWeekday && c.IsTradingDay(t)
}

// NextTradingDay returns the next open session after t.
func (c *Calendar) NextTradingDay(t time.Time) time.Time {
	d := t.In(c.loc)
	for i := 0; i < 366; i++ {
		d = d.AddDate(0, 0, 1)
		if c.IsTradingDay(d) {
			return d
		}
	}
	return d
}

// Location returns the exchange timezone.
func (c *Calendar) Location() *time.Location {
	return c.loc
}

func (c *Calendar) parseTime(day time.Time, hhmm string) (time.Time, error) {
	if hhmm == "" {
		return time.Time{}, fmt.Errorf("time not configured")
	}
	local := day.In(c.loc)
	parsed, err := time.ParseInLocation("2006-01-02 15:04", local.Format("2006-01-02")+" "+hhmm, c.loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", hhmm, err)
	}
	return parsed, nil
}
