package billing

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
	RecordID  string            `json:"record_id"`
	AccountID string            `json:"account_id"`
	Metric    string            `json:"metric"`
	Quantity  float64           `json:"quantity"`
	Period    string            `json:"period"`
	RunID     string            `json:"run_id,omitempty"`
	Unit      string            `json:"unit,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
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
