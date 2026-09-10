// Package parser converts raw 1C export file bytes into domain CarRecords.
// Pure functions only: no I/O, no database access.
package parser

import "awesomeProject5/internal/domain"

// Parse decodes raw file bytes (handling encoding/delimiter quirks) and
// returns parsed records plus per-row errors. Not implemented yet.
func Parse(data []byte) ([]domain.CarRecord, []domain.RowErrorInfo, error) {
	// TODO: implement in phase 2 (encoding detection, delimiter sniffing,
	// header mapping, VIN validation).
	return nil, nil, nil
}
