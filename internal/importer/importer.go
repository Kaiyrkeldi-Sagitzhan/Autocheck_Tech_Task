// Package importer orchestrates the import pipeline: file loading, parsing,
// normalization, deduplication, and upserting into the repository.
package importer

// Importer coordinates a single import run.
type Importer struct {
	// repository and parser dependencies are injected in a later phase.
}

// New creates an Importer.
func New() *Importer {
	return &Importer{}
}
