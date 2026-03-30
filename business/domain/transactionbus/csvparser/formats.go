package csvparser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Row is a parsed CSV row ready for transaction creation.
type Row struct {
	Source      string
	Date        time.Time
	Description string
	Amount      int // cents, negative = debit
}

// Format defines how to parse a specific bank's CSV export.
type Format struct {
	Name         string
	HeaderMatch  []string // headers that uniquely identify this format
	DateCol      string
	DescCol      string
	AmountCol    string
	DateLayout   string
	AmountNegate bool // true if positive amounts mean charges (Amex)
}

var formats = []Format{
	{
		Name:        "chase_checking",
		HeaderMatch: []string{"Details", "Posting Date", "Description", "Amount", "Type", "Balance"},
		DateCol:     "Posting Date",
		DescCol:     "Description",
		AmountCol:   "Amount",
		DateLayout:  "01/02/2006",
	},
	{
		Name:        "chase_credit",
		HeaderMatch: []string{"Transaction Date", "Post Date", "Description", "Category", "Type", "Amount"},
		DateCol:     "Transaction Date",
		DescCol:     "Description",
		AmountCol:   "Amount",
		DateLayout:  "01/02/2006",
	},
	{
		Name:        "amex",
		HeaderMatch: []string{"Date", "Description", "Amount"},
		DateCol:     "Date",
		DescCol:     "Description",
		AmountCol:   "Amount",
		DateLayout:  "01/02/2006",
		AmountNegate: true,
	},
}

func detect(headers []string) (Format, error) {
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[strings.TrimSpace(h)] = true
	}

	for _, f := range formats {
		match := true
		for _, required := range f.HeaderMatch {
			if !headerSet[required] {
				match = false
				break
			}
		}
		if match {
			return f, nil
		}
	}

	return Format{}, fmt.Errorf("unrecognized CSV format, headers: %v", headers)
}

func lookup(name string) (Format, error) {
	for _, f := range formats {
		if f.Name == name {
			return f, nil
		}
	}
	return Format{}, fmt.Errorf("unknown format %q", name)
}

func parseRow(f Format, headerIndex map[string]int, record []string) (Row, error) {
	dateStr := strings.TrimSpace(record[headerIndex[f.DateCol]])
	date, err := time.Parse(f.DateLayout, dateStr)
	if err != nil {
		return Row{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}

	desc := strings.TrimSpace(record[headerIndex[f.DescCol]])
	desc = strings.Trim(desc, `"`)

	amountStr := strings.TrimSpace(record[headerIndex[f.AmountCol]])
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return Row{}, fmt.Errorf("parse amount %q: %w", amountStr, err)
	}

	if f.AmountNegate {
		amountFloat = -amountFloat
	}

	amountCents := int(math.Round(amountFloat * 100))

	return Row{
		Source:      f.Name,
		Date:        date,
		Description: desc,
		Amount:      amountCents,
	}, nil
}
