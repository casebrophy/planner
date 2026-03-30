package csvparser_test

import (
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/transactionbus/csvparser"
)

func TestParse_ChaseChecking(t *testing.T) {
	csv := "Details,Posting Date,Description,Amount,Type,Balance,Check or Slip #\nDEBIT,01/15/2025,STARBUCKS STORE 12345,-4.50,ACH_DEBIT,1234.56,\nCREDIT,01/16/2025,PAYROLL DEPOSIT,3200.00,ACH_CREDIT,4434.56,"

	txns, err := csvparser.Parse(csv, "chase_checking")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	if txns[0].Description != "STARBUCKS STORE 12345" {
		t.Errorf("expected description 'STARBUCKS STORE 12345', got %q", txns[0].Description)
	}
	if txns[0].Amount != -450 {
		t.Errorf("expected amount -450, got %d", txns[0].Amount)
	}
	expected := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !txns[0].Date.Equal(expected) {
		t.Errorf("expected date %v, got %v", expected, txns[0].Date)
	}

	if txns[1].Amount != 320000 {
		t.Errorf("expected amount 320000, got %d", txns[1].Amount)
	}
}

func TestParse_ChaseCredit(t *testing.T) {
	csv := "Transaction Date,Post Date,Description,Category,Type,Amount,Memo\n01/10/2025,01/12/2025,AMAZON.COM,Shopping,Sale,-29.99,\n01/11/2025,01/13/2025,PAYMENT THANK YOU,Payment,Payment,500.00,"

	txns, err := csvparser.Parse(csv, "chase_credit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	if txns[0].Description != "AMAZON.COM" {
		t.Errorf("expected description 'AMAZON.COM', got %q", txns[0].Description)
	}
	if txns[0].Amount != -2999 {
		t.Errorf("expected amount -2999, got %d", txns[0].Amount)
	}
}

func TestParse_Amex(t *testing.T) {
	csv := "Date,Description,Amount\n01/20/2025,WHOLE FOODS MARKET,85.43\n01/21/2025,UBER TRIP,22.50"

	txns, err := csvparser.Parse(csv, "amex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	if txns[0].Amount != -8543 {
		t.Errorf("expected amount -8543, got %d", txns[0].Amount)
	}
}

func TestParse_AutoDetect(t *testing.T) {
	csv := "Details,Posting Date,Description,Amount,Type,Balance,Check or Slip #\nDEBIT,02/01/2025,TEST,-10.00,ACH_DEBIT,100.00,"

	txns, err := csvparser.Parse(csv, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}

	if txns[0].Source != "chase_checking" {
		t.Errorf("expected auto-detected source 'chase_checking', got %q", txns[0].Source)
	}
}

func TestParse_EmptyCSV(t *testing.T) {
	_, err := csvparser.Parse("", "chase_checking")
	if err == nil {
		t.Fatal("expected error for empty CSV")
	}
}
