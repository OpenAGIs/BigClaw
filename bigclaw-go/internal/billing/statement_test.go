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
				Metadata:  map[string]string{"source": "workflow-engine"},
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
	if decoded.Usage[0].Metadata["source"] != "workflow-engine" {
		t.Fatalf("unexpected summary after roundtrip: %#v", decoded)
	}
}
