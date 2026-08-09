package strategylib

// Decision is the output of a research strategy evaluation.
type Decision string

const (
	DecisionBuy    Decision = "BUY"
	DecisionSell   Decision = "SELL"
	DecisionExit   Decision = "EXIT"
	DecisionIgnore Decision = "IGNORE"
)

// Position describes the open position state passed into strategy evaluation.
type Position string

const (
	PositionFlat  Position = "FLAT"
	PositionLong  Position = "LONG"
	PositionShort Position = "SHORT"
)
