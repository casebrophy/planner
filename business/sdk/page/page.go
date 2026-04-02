package page

import (
	"fmt"
	"strconv"
)

type Page struct {
	number      int
	rowsPerPage int
}

// Parse creates a Page from string inputs with API-level validation.
// Use this for HTTP handler input where rows must be between 1 and 100.
func Parse(pageStr, rowsStr string) (Page, error) {
	number := 1
	if pageStr != "" {
		var err error
		number, err = strconv.Atoi(pageStr)
		if err != nil {
			return Page{}, fmt.Errorf("page must be an integer: %w", err)
		}
		if number < 1 {
			return Page{}, fmt.Errorf("page must be >= 1, got %d", number)
		}
	}

	rowsPerPage := 10
	if rowsStr != "" {
		var err error
		rowsPerPage, err = strconv.Atoi(rowsStr)
		if err != nil {
			return Page{}, fmt.Errorf("rows must be an integer: %w", err)
		}
		if rowsPerPage < 1 || rowsPerPage > 100 {
			return Page{}, fmt.Errorf("rows must be between 1 and 100, got %d", rowsPerPage)
		}
	}

	return Page{
		number:      number,
		rowsPerPage: rowsPerPage,
	}, nil
}

// New creates a Page directly from integer values. Use this for internal
// callers where the values are known-good and not subject to API limits.
func New(number, rowsPerPage int) Page {
	return Page{
		number:      number,
		rowsPerPage: rowsPerPage,
	}
}

func (p Page) Number() int      { return p.number }
func (p Page) RowsPerPage() int { return p.rowsPerPage }
func (p Page) Offset() int      { return (p.number - 1) * p.rowsPerPage }
