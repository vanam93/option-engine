package groww

import (
	"context"
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	symbolregistry "github.com/vanam-gangireddy/option-engine/internal/market/registry"
)

// HistoricalService wraps Groww historical backtesting APIs.
type HistoricalService struct {
	client *Client
	auth   *Authenticator
	cfg    Config
}

func newHistoricalService(cfg Config, client *Client, auth *Authenticator) *HistoricalService {
	return &HistoricalService{client: client, auth: auth, cfg: cfg}
}

// FetchExpiries returns derivative expiry dates.
func (h *HistoricalService) FetchExpiries(ctx context.Context, underlying string, year, month int) ([]string, error) {
	token, err := h.auth.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"exchange":           h.cfg.Exchange,
		"underlying_symbol":  underlying,
	}
	if year > 0 {
		params["year"] = fmt.Sprintf("%d", year)
	}
	if month > 0 {
		params["month"] = fmt.Sprintf("%d", month)
	}
	var payload ExpiriesPayload
	if err := h.client.GetJSON(ctx, "/v1/historical/expiries", params, &payload, token); err != nil {
		return nil, err
	}
	return payload.Expiries, nil
}

// FetchOptionChain returns contract symbols for an expiry.
func (h *HistoricalService) FetchOptionChain(ctx context.Context, underlying, expiryDate string) ([]string, error) {
	return h.FetchContracts(ctx, underlying, expiryDate)
}

// FetchContracts returns derivative contracts for an expiry.
func (h *HistoricalService) FetchContracts(ctx context.Context, underlying, expiryDate string) ([]string, error) {
	token, err := h.auth.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"exchange":           h.cfg.Exchange,
		"underlying_symbol":  underlying,
		"expiry_date":        expiryDate,
	}
	var payload ContractsPayload
	if err := h.client.GetJSON(ctx, "/v1/historical/contracts", params, &payload, token); err != nil {
		return nil, err
	}
	return payload.Contracts, nil
}

// FetchInstrument resolves a groww symbol via the contracts API when possible.
func (h *HistoricalService) FetchInstrument(ctx context.Context, growwSymbol string) (string, error) {
	if growwSymbol == "" {
		return "", fmt.Errorf("groww symbol required")
	}
	return growwSymbol, nil
}

// FetchHistoricalCandles downloads candles for a single request window.
func (h *HistoricalService) FetchHistoricalCandles(ctx context.Context, req CandleRequest, domainSymbol string) ([]market.Candle, error) {
	token, err := h.auth.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"exchange":        req.Exchange,
		"segment":         req.Segment,
		"groww_symbol":    req.GrowwSymbol,
		"start_time":      req.StartTime.Format("2006-01-02 15:04:05"),
		"end_time":        req.EndTime.Format("2006-01-02 15:04:05"),
		"candle_interval": req.CandleInterval,
	}
	var payload CandlesPayload
	if err := h.client.GetJSON(ctx, "/v1/historical/candles", params, &payload, token); err != nil {
		return nil, err
	}
	return mapGrowwCandles(payload.Candles, domainSymbol, h.cfg.Timeframe)
}

// CandleIterator streams candles without loading the full range into memory.
type CandleIterator struct {
	hist     *HistoricalService
	req      CandleRequest
	inst     symbolregistry.Instrument
	symbol   string
	loc      *time.Location
	chunks   []timeWindow
	chunkPos int
	buffer   []market.Candle
	bufPos   int
	done     bool
}

type timeWindow struct {
	start time.Time
	end   time.Time
}

// NewCandleIterator builds a chunked iterator for a symbol range.
func NewCandleIterator(hist *HistoricalService, symbol string, inst symbolregistry.Instrument, start, end time.Time, interval string, tf market.Timeframe) *CandleIterator {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	if loc == nil {
		loc = time.UTC
	}
	if inst.Exchange == "" {
		inst.Exchange = "NSE"
	}
	growwSymbol := GrowwSymbol(inst)
	segment := ResolveSegment(inst)
	if start.IsZero() {
		start = time.Date(end.Year(), end.Month(), end.Day(), 9, 15, 0, 0, loc).AddDate(0, 0, -30)
	}
	if end.IsZero() {
		end = time.Date(start.Year(), start.Month(), start.Day(), 15, 30, 0, 0, loc)
	}
	inst.Symbol = symbol
	return &CandleIterator{
		hist: hist,
		req: CandleRequest{
			Exchange:       inst.Exchange,
			Segment:        segment,
			GrowwSymbol:    growwSymbol,
			CandleInterval: interval,
		},
		inst:   inst,
		symbol: symbol,
		loc:    loc,
		chunks: chunkRange(start.In(loc), end.In(loc), interval),
	}
}

func chunkRange(start, end time.Time, interval string) []timeWindow {
	maxDur := intervalMaxDuration[interval]
	if maxDur == 0 {
		maxDur = 30 * 24 * time.Hour
	}
	var chunks []timeWindow
	cursor := start
	for !cursor.After(end) {
		chunkEnd := cursor.Add(maxDur - time.Second)
		if chunkEnd.After(end) {
			chunkEnd = end
		}
		chunks = append(chunks, timeWindow{start: cursor, end: chunkEnd})
		cursor = chunkEnd.Add(time.Second)
	}
	return chunks
}

// Next returns the next candle or false when exhausted.
func (it *CandleIterator) Next(ctx context.Context) (market.Candle, bool, error) {
	for {
		if it.bufPos < len(it.buffer) {
			c := it.buffer[it.bufPos]
			it.bufPos++
			return c, true, nil
		}
		if it.done {
			return market.Candle{}, false, nil
		}
		if it.chunkPos >= len(it.chunks) {
			it.done = true
			return market.Candle{}, false, nil
		}
		window := it.chunks[it.chunkPos]
		it.chunkPos++
		req := it.req
		req.StartTime = window.start
		req.EndTime = window.end
		candles, err := it.hist.FetchHistoricalCandles(ctx, req, it.symbol)
		if err != nil {
			return market.Candle{}, false, err
		}
		it.buffer = candles
		it.bufPos = 0
		if len(it.buffer) == 0 && it.chunkPos >= len(it.chunks) {
			it.done = true
			return market.Candle{}, false, nil
		}
	}
}
