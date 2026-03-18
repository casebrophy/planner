package page

import (
	"fmt"
	"strconv"
)

type Page struct {
	number      int
	rowsPerPage int
}

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

func MustParse(pageStr, rowsStr string) Page {
	p, err := Parse(pageStr, rowsStr)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Page) Number() int      { return p.number }
func (p Page) RowsPerPage() int { return p.rowsPerPage }
func (p Page) Offset() int      { return (p.number - 1) * p.rowsPerPage }
