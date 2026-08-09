package cross

// Above reports whether a crossed above b on the current bar.
func Above(prevA, prevB, currA, currB float64, warmed bool) bool {
	if !warmed {
		return false
	}
	return prevA <= prevB && currA > currB
}

// Below reports whether a crossed below b on the current bar.
func Below(prevA, prevB, currA, currB float64, warmed bool) bool {
	if !warmed {
		return false
	}
	return prevA >= prevB && currA < currB
}
