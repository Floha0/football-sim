package domain

// Prediction is the estimated championship probability for one team,
// computed by Monte Carlo simulation at a specific week of the season.
// Chance is a probability in the range [0.0, 1.0].
type Prediction struct {
	TeamID int
	Chance float64
	Week   int
}
