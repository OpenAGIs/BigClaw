package billing

import (
	"encoding/json"
	"testing"
)

func TestBillingStatementRoundTrip(t *testing.T) {
	summary := BillingSummaryModel{
		StatementID:   "bill-1",
		AccountID:     "acct-1",
		BillingPeriod: "2026-03",
		Currency:      "USD",
		Rates: []BillingRate{
			{
				Metric:          "orchestration-run",
				Interval:        BillingMonthly,
				IncludedUnits:   100,
				UnitPriceUSD:    0,
				OveragePriceUSD: 1.5,
			},
		},
		Usage: []BillingUsageRecord{
			{
				RecordID:  "usage-1",
				AccountID: "acct-1",
				Metric:    "orchestration-run",
				Quantity:  124,
				Period:    "2026-03",
				RunID:     "flow-run-1",
				Unit:      "run",
				Metadata:  map[string]any{"source": "workflow-engine", "attempt": float64(2), "billable": true},
			},
		},
		SubtotalUSD: 0,
		OverageUSD:  36,
		TotalUSD:    36,
	}
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded BillingSummaryModel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.TotalUSD != summary.TotalUSD || decoded.Rates[0].Interval != BillingMonthly {
		t.Fatalf("unexpected summary after roundtrip: %#v", decoded)
	}
	if decoded.Usage[0].Metadata["source"] != "workflow-engine" || decoded.Usage[0].Metadata["attempt"] != float64(2) || decoded.Usage[0].Metadata["billable"] != true {
		t.Fatalf("unexpected summary after roundtrip: %#v", decoded)
	}
}

func TestBillingStatementJSONEmitsPythonContractDefaults(t *testing.T) {
	summary := BillingSummaryModel{
		StatementID:   "bill-2",
		AccountID:     "acct-2",
		BillingPeriod: "2026-04",
		Currency:      "USD",
	}
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"rates", "usage", "subtotal_usd", "overage_usd", "total_usd"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected key %q in billing JSON, got %#v", key, decoded)
		}
	}

	recordData, err := json.Marshal(BillingUsageRecord{
		RecordID:  "usage-2",
		AccountID: "acct-2",
		Metric:    "orchestration-run",
		Quantity:  1,
		Period:    "2026-04",
	})
	if err != nil {
		t.Fatalf("marshal usage record: %v", err)
	}
	var decodedRecord map[string]any
	if err := json.Unmarshal(recordData, &decodedRecord); err != nil {
		t.Fatalf("decode usage record: %v", err)
	}
	for _, key := range []string{"run_id", "unit", "metadata"} {
		if _, ok := decodedRecord[key]; !ok {
			t.Fatalf("expected key %q in usage JSON, got %#v", key, decodedRecord)
		}
	}
}
