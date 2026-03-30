package csvparser

import (
	"encoding/csv"
	"fmt"
	"strings"
)

// Parse reads a CSV string and returns parsed rows.
// If formatName is empty, auto-detects the format from headers.
func Parse(csvData string, formatName string) ([]Row, error) {
	csvData = strings.TrimSpace(csvData)
	if csvData == "" {
		return nil, fmt.Errorf("empty CSV data")
	}

	reader := csv.NewReader(strings.NewReader(csvData))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	headers := records[0]

	var f Format
	if formatName == "" {
		f, err = detect(headers)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = lookup(formatName)
		if err != nil {
			return nil, err
		}
	}

	headerIndex := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIndex[strings.TrimSpace(h)] = i
	}

	for _, col := range []string{f.DateCol, f.DescCol, f.AmountCol} {
		if _, ok := headerIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column %q", col)
		}
	}

	var rows []Row
	for i, record := range records[1:] {
		row, err := parseRow(f, headerIndex, record)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+2, err)
		}
		rows = append(rows, row)
	}

	return rows, nil
}
