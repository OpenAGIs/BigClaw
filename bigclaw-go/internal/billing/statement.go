package billing

import "encoding/json"

type Interval string

const (
	IntervalMonthly Interval = "monthly"
	IntervalAnnual  Interval = "annual"
	IntervalUsage   Interval = "usage"
)

type Rate struct {
	Metric          string   `json:"metric"`
	Interval        Interval `json:"interval"`
	IncludedUnits   int      `json:"included_units"`
	UnitPriceUSD    float64  `json:"unit_price_usd"`
	OveragePriceUSD float64  `json:"overage_price_usd"`
}

type UsageRecord struct {
	RecordID  string         `json:"record_id"`
	AccountID string         `json:"account_id"`
	Metric    string         `json:"metric"`
	Quantity  float64        `json:"quantity"`
	Period    string         `json:"period"`
	RunID     string         `json:"run_id,omitempty"`
	Unit      string         `json:"unit,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func (record UsageRecord) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"record_id":  record.RecordID,
		"account_id": record.AccountID,
		"metric":     record.Metric,
		"quantity":   record.Quantity,
		"period":     record.Period,
		"run_id":     record.RunID,
		"unit":       record.Unit,
		"metadata":   metadataOrEmpty(record.Metadata),
	}
	return json.Marshal(payload)
}

type BillingSummary struct {
	StatementID   string        `json:"statement_id"`
	AccountID     string        `json:"account_id"`
	BillingPeriod string        `json:"billing_period"`
	Currency      string        `json:"currency"`
	Rates         []Rate        `json:"rates,omitempty"`
	Usage         []UsageRecord `json:"usage,omitempty"`
	SubtotalUSD   float64       `json:"subtotal_usd"`
	OverageUSD    float64       `json:"overage_usd"`
	TotalUSD      float64       `json:"total_usd"`
}

func (summary BillingSummary) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"statement_id":   summary.StatementID,
		"account_id":     summary.AccountID,
		"billing_period": summary.BillingPeriod,
		"currency":       summary.Currency,
		"rates":          ratesOrEmpty(summary.Rates),
		"usage":          usageOrEmpty(summary.Usage),
		"subtotal_usd":   summary.SubtotalUSD,
		"overage_usd":    summary.OverageUSD,
		"total_usd":      summary.TotalUSD,
	}
	return json.Marshal(payload)
}

func ratesOrEmpty(values []Rate) []Rate {
	if values == nil {
		return []Rate{}
	}
	return values
}

func usageOrEmpty(values []UsageRecord) []UsageRecord {
	if values == nil {
		return []UsageRecord{}
	}
	return values
}

func metadataOrEmpty(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}

// These aliases expose the Python-side billing contract names while reusing the
// existing billing contract structures so there is no duplication.
type BillingInterval = Interval
type BillingRate = Rate
type BillingUsageRecord = UsageRecord
type BillingSummaryModel = BillingSummary

const (
	BillingMonthly = IntervalMonthly
	BillingAnnual  = IntervalAnnual
	BillingUsage   = IntervalUsage
)
