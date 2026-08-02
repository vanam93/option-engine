package console

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/delivery"
)

// Renderer writes formatted recommendation blocks to an output stream.
type Renderer struct {
	out       io.Writer
	formatter *Formatter
	overwrite bool

	mu         sync.Mutex
	blockLines map[string]int
}

// NewRenderer creates a console renderer.
func NewRenderer(out io.Writer, overwrite bool) *Renderer {
	if out == nil {
		out = io.Discard
	}
	return &Renderer{
		out:        out,
		formatter:  NewFormatter(),
		overwrite:  overwrite,
		blockLines: make(map[string]int),
	}
}

// Render writes or overwrites a recommendation block.
func (r *Renderer) Render(doc delivery.DeliveryDocument, at time.Time, isUpdate bool) error {
	lines := r.formatter.FormatBlock(doc, at)
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.overwrite && isUpdate {
		if prevLines, ok := r.blockLines[doc.RecommendationID]; ok && prevLines > 0 {
			if _, err := fmt.Fprintf(r.out, "\x1b[%dA", prevLines); err != nil {
				return err
			}
		} else {
			isUpdate = false
		}
	}

	for _, line := range lines {
		if r.overwrite && isUpdate {
			if _, err := fmt.Fprintf(r.out, "\x1b[2K%s\n", line); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintln(r.out, line); err != nil {
			return err
		}
	}

	r.blockLines[doc.RecommendationID] = len(lines)
	return nil
}

// Reset clears tracked block positions.
func (r *Renderer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blockLines = make(map[string]int)
}

// LastOutput returns the rendered block for tests without ANSI control codes.
func (r *Renderer) LastOutput(doc delivery.DeliveryDocument, at time.Time) string {
	lines := r.formatter.FormatBlock(doc, at)
	return strings.Join(lines, "\n")
}
