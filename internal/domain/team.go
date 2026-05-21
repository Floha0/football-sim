// Package domain contains the core types of the football simulation league.
package domain

// Power influences match outcome probability (1-100, higher is stronger).
type Team struct {
	ID    int
	Name  string
	Power int
}
