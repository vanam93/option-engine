package indicators

// Result is the output of an incremental indicator update.
type Result struct {
	Value    float64
	WarmedUp bool
	Samples  int
}
