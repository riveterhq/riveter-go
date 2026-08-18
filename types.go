package riveter

import (
	"encoding/json"
	"time"
)

// RunProgress is the completion estimate on a run.
type RunProgress struct {
	PercentComplete           *float64 `json:"percent_complete"`
	EstimatedSecondsRemaining *float64 `json:"estimated_seconds_remaining"`
	ElapsedSeconds            *float64 `json:"elapsed_seconds"`
	CompletedCells            int      `json:"completed_cells"`
	TotalCellsExpected        int      `json:"total_cells_expected"`
	NotFoundCells             int      `json:"not_found_cells"`
}

// RunError is the error object on a failed run.
type RunError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Run is the uniform envelope returned by kickoffs and every /runs endpoint.
// Endpoint-specific kickoff extras (DatasetID, CreditsCharged, ...) are only
// set where the endpoint returns them; Output is nil until the run finishes.
type Run struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	Progress       *RunProgress    `json:"progress"`
	CreditsUsed    float64         `json:"credits_used"`
	AppURL         string          `json:"app_url"`
	ResultURL      string          `json:"result_url"`
	StartedAt      *time.Time      `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at"`
	Error          *RunError       `json:"error"`
	EnrichmentID   string          `json:"enrichment_id"`
	EnrichmentName string          `json:"enrichment_name"`
	DatasetID      string          `json:"dataset_id"`
	ExtractionID   string          `json:"extraction_id"`
	MonitorID      string          `json:"monitor_id"`
	WebhookURL     string          `json:"webhook_url"`
	Output         json.RawMessage `json:"output"`

	MaxItems            int            `json:"max_items"`
	EnrichmentRunID     string         `json:"enrichment_run_id"`
	SourceDatasetID     string         `json:"source_dataset_id"`
	ConfiguredDatasetID string         `json:"configured_dataset_id"`
	CreditsCharged      float64        `json:"credits_charged"`
	Variables           map[string]any `json:"variables"`
	ValidationWarning   string         `json:"validation_warning"`
	Tier                string         `json:"tier"`
}

// Finished reports whether the run reached a terminal state.
func (r *Run) Finished() bool {
	return r.Status == "success" || r.Status == "stopped"
}

// RunListItem is the slim run shape used by list endpoints.
type RunListItem struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	EnrichmentID   string     `json:"enrichment_id"`
	EnrichmentName string     `json:"enrichment_name"`
	RowCount       *int       `json:"row_count"`
	CreditsUsed    float64    `json:"credits_used"`
	Error          *RunError  `json:"error"`
	CreatedAt      *time.Time `json:"created_at"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	AppURL         string     `json:"app_url"`
	ResultURL      string     `json:"result_url"`
}

// Pagination describes a page of list results.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

// RunsSummary is the all-time run counts by status.
type RunsSummary struct {
	Counts struct {
		Pending    int `json:"pending"`
		Enqueued   int `json:"enqueued"`
		Processing int `json:"processing"`
		Success    int `json:"success"`
		Stopped    int `json:"stopped"`
	} `json:"counts"`
}

// EnrichParams is the request body for Enrich. Provide rows via Input or
// DatasetID, and column config via EnrichmentID, Output, or Prompt+Attributes.
type EnrichParams struct {
	EnrichmentID        string                        `json:"enrichment_id,omitempty"`
	Output              map[string]OutputColumnConfig `json:"output,omitempty"`
	Prompt              string                        `json:"prompt,omitempty"`
	Attributes          []string                      `json:"attributes,omitempty"`
	Input               map[string][]string           `json:"input,omitempty"`
	DatasetID           string                        `json:"dataset_id,omitempty"`
	RunKey              string                        `json:"run_key,omitempty"`
	WebhookURL          string                        `json:"webhook_url,omitempty"`
	AllowDuplicateInput bool                          `json:"allow_duplicate_input,omitempty"`
}

// FormatDetails holds format-specific column options.
type FormatDetails struct {
	Options       []string          `json:"options,omitempty"`
	AllowMultiple bool              `json:"allow_multiple,omitempty"`
	Descriptions  map[string]string `json:"descriptions,omitempty"`
	DecimalPlaces *int              `json:"decimal_places,omitempty"`
}

// RunWhenRule is one rule of a dynamic run condition.
type RunWhenRule struct {
	Column    string `json:"column"`
	Condition string `json:"condition"`
	Value     string `json:"value,omitempty"`
}

// RunWhenConfig is the rule-based run condition for run_when "dynamic".
type RunWhenConfig struct {
	MatchMode string        `json:"match_mode,omitempty"`
	Rules     []RunWhenRule `json:"rules,omitempty"`
}

// OutputColumnConfig is the per-column enrichment config (agent mode or
// tool-only mode).
type OutputColumnConfig struct {
	Prompt           string            `json:"prompt,omitempty"`
	Contexts         []string          `json:"contexts,omitempty"`
	Tools            []string          `json:"tools,omitempty"`
	Format           string            `json:"format,omitempty"`
	FormatDetails    *FormatDetails    `json:"format_details,omitempty"`
	RunWhen          string            `json:"run_when,omitempty"`
	RunWhenConfig    *RunWhenConfig    `json:"run_when_config,omitempty"`
	Tool             string            `json:"tool,omitempty"`
	URL              string            `json:"url,omitempty"`
	Query            string            `json:"query,omitempty"`
	DateStart        string            `json:"date_start,omitempty"`
	DateEnd          string            `json:"date_end,omitempty"`
	Code             string            `json:"code,omitempty"`
	Args             map[string]string `json:"args,omitempty"`
	ProxyCountryCode string            `json:"proxy_country_code,omitempty"`
	WaitLonger       bool              `json:"wait_longer,omitempty"`
	SkipCache        bool              `json:"skip_cache,omitempty"`
	Delete           bool              `json:"delete,omitempty"`
}

// ListRunsParams filters GET /runs.
type ListRunsParams struct {
	// Comma-separated run types: enrichment, dataset_build, extraction, scrape, quick_search.
	Type          string
	Status        string
	EnrichmentID  string
	MonitorID     string
	CreatedAfter  string
	CreatedBefore string
	Page          int
	PerPage       int
}

// EnrichmentSummary is the list-endpoint shape of an enrichment.
type EnrichmentSummary struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	AppURL  string         `json:"app_url"`
	Columns map[string]any `json:"columns"`
}

// Enrichment is an enrichment's structure: input columns and output config.
type Enrichment struct {
	ID        string                        `json:"id"`
	Name      string                        `json:"name"`
	Status    string                        `json:"status"`
	AppURL    string                        `json:"app_url"`
	DatasetID string                        `json:"dataset_id"`
	Input     map[string]any                `json:"input"`
	Output    map[string]OutputColumnConfig `json:"output"`
}

// UpdateEnrichmentParams is the request body for Enrichments.Update.
type UpdateEnrichmentParams struct {
	Output      map[string]OutputColumnConfig `json:"output,omitempty"`
	ColumnOrder []string                      `json:"column_order,omitempty"`
}

// BuildDatasetForEnrichmentParams is the body for Enrichments.BuildDataset.
type BuildDatasetForEnrichmentParams struct {
	Prompt                      string   `json:"prompt"`
	Qualifiers                  []string `json:"qualifiers,omitempty"`
	MaxItems                    int      `json:"max_items,omitempty"`
	DatasetWebhookURL           string   `json:"dataset_webhook_url,omitempty"`
	AutoRunEnrichment           bool     `json:"auto_run_enrichment,omitempty"`
	AutoRunEnrichmentWebhookURL string   `json:"auto_run_enrichment_webhook_url,omitempty"`
}

// BuildDatasetParams is the request body for Datasets.Build.
type BuildDatasetParams struct {
	Prompt                      string   `json:"prompt,omitempty"`
	Identifiers                 []string `json:"identifiers,omitempty"`
	Qualifiers                  []string `json:"qualifiers,omitempty"`
	Attributes                  []string `json:"attributes,omitempty"`
	MaxItems                    int      `json:"max_items,omitempty"`
	DatasetWebhookURL           string   `json:"dataset_webhook_url,omitempty"`
	AutoRunEnrichment           bool     `json:"auto_run_enrichment,omitempty"`
	AutoRunEnrichmentWebhookURL string   `json:"auto_run_enrichment_webhook_url,omitempty"`
}

// ExtendDatasetParams is the request body for Datasets.Extend.
type ExtendDatasetParams struct {
	Prompt                      string   `json:"prompt,omitempty"`
	Qualifiers                  []string `json:"qualifiers,omitempty"`
	MaxItems                    int      `json:"max_items,omitempty"`
	DatasetWebhookURL           string   `json:"dataset_webhook_url,omitempty"`
	AutoRunEnrichment           bool     `json:"auto_run_enrichment,omitempty"`
	AutoRunEnrichmentWebhookURL string   `json:"auto_run_enrichment_webhook_url,omitempty"`
}

// BuildConfiguredDatasetParams is the body for ConfiguredDatasets.Build.
type BuildConfiguredDatasetParams struct {
	Parameters        map[string]string `json:"parameters,omitempty"`
	Tier              string            `json:"tier,omitempty"`
	DatasetWebhookURL string            `json:"dataset_webhook_url,omitempty"`
}

// CreateExtractionParams is the request body for Extractions.Create.
type CreateExtractionParams struct {
	StartingURL            string         `json:"starting_url"`
	GoalDescription        string         `json:"goal_description"`
	OutputRecordJSONSchema map[string]any `json:"output_record_json_schema"`
	Name                   string         `json:"name,omitempty"`
	RequiredKeys           []string       `json:"required_keys,omitempty"`
}

// Extraction is a reusable site scrape/extract recipe.
type Extraction struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	Status                 string     `json:"status"`
	AppURL                 string     `json:"app_url"`
	StartingURL            string     `json:"starting_url"`
	GoalDescription        string     `json:"goal_description"`
	OutputRecordJSONSchema any        `json:"output_record_json_schema"`
	RequiredKeys           []string   `json:"required_keys"`
	Locked                 bool       `json:"locked"`
	ValidationPassing      *bool      `json:"validation_passing"`
	DiscoveredAt           *time.Time `json:"discovered_at"`
	RunCreditsRequired     float64    `json:"run_credits_required"`
	CreditsCharged         float64    `json:"credits_charged"`
}

// RunExtractionParams is the request body for Extractions.Run.
type RunExtractionParams struct {
	Variables  map[string]string `json:"variables,omitempty"`
	WebhookURL string            `json:"webhook_url,omitempty"`
}

// CreateMonitorParams is the request body for Monitors.Create. Minute and
// Hour are always sent (0 is valid for both).
type CreateMonitorParams struct {
	EnrichmentID   string              `json:"enrichment_id"`
	Cadence        string              `json:"cadence"`
	Minute         int                 `json:"minute"`
	Hour           int                 `json:"hour"`
	Timezone       string              `json:"timezone"`
	DayOfWeek      *int                `json:"day_of_week,omitempty"`
	DayOfMonth     *int                `json:"day_of_month,omitempty"`
	WebhookURL     string              `json:"webhook_url,omitempty"`
	AlertRule      string              `json:"alert_rule,omitempty"`
	OutputFormat   string              `json:"output_format,omitempty"`
	RunImmediately bool                `json:"run_immediately,omitempty"`
	Input          map[string][]string `json:"input,omitempty"`
}

// UpdateMonitorParams is the request body for Monitors.Update. Enabled is a
// pointer so `false` (pause) still serializes.
type UpdateMonitorParams struct {
	Enabled    *bool  `json:"enabled,omitempty"`
	WebhookURL string `json:"webhook_url,omitempty"`
}

// Monitor is a scheduled enrichment run.
type Monitor struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Enabled         bool       `json:"enabled"`
	Cadence         string     `json:"cadence"`
	Minute          int        `json:"minute"`
	Hour            int        `json:"hour"`
	DayOfWeek       *int       `json:"day_of_week"`
	DayOfMonth      *int       `json:"day_of_month"`
	Timezone        string     `json:"timezone"`
	WebhookURL      string     `json:"webhook_url"`
	AlertRule       string     `json:"alert_rule"`
	OutputFormat    string     `json:"output_format"`
	NextRunAt       *time.Time `json:"next_run_at"`
	ScheduleSummary string     `json:"schedule_summary"`
	EnrichmentID    string     `json:"enrichment_id"`
	EnrichmentName  string     `json:"enrichment_name"`
	CreatedAt       *time.Time `json:"created_at"`
	HasInput        bool       `json:"has_input"`
}

// ListMonitorRunsParams filters GET /monitors/{id}/runs.
type ListMonitorRunsParams struct {
	Status  string
	Page    int
	PerPage int
}

// QuickSearchParams is the request body for QuickSearch.
type QuickSearchParams struct {
	Query     string `json:"query"`
	DateStart string `json:"date_start,omitempty"`
	DateEnd   string `json:"date_end,omitempty"`
	RunKey    string `json:"run_key,omitempty"`
}

// ScrapeParams is the request body for Scrape.
type ScrapeParams struct {
	URL              string `json:"url"`
	ProxyCountryCode string `json:"proxy_country_code,omitempty"`
	SkipCache        bool   `json:"skip_cache,omitempty"`
}

// ScrapeResult is the synchronous scrape result (legacy request_status
// format, per the spec).
type ScrapeResult struct {
	RequestStatus   string  `json:"request_status"`
	Text            string  `json:"text"`
	URL             string  `json:"url"`
	BaseURLForLinks string  `json:"base_url_for_links"`
	StatusCode      int     `json:"status_code"`
	PossiblyBlocked bool    `json:"possibly_blocked"`
	CreditUsed      float64 `json:"credit_used"`
	RiveterAppLink  string  `json:"riveter_app_link"`
}

// Credit is the account's credit state.
type Credit struct {
	Count   int `json:"count"`
	Max     int `json:"max"`
	Balance int `json:"balance"`
}

// Account is the account associated with the API key.
type Account struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Plan   string `json:"plan"`
	Credit Credit `json:"credit"`
}

// AccountUser is the user who created the API key.
type AccountUser struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// APIKeyInfo is the API key's metadata.
type APIKeyInfo struct {
	Name       string      `json:"name"`
	LastUsedAt *time.Time  `json:"last_used_at"`
	CreatedBy  AccountUser `json:"created_by"`
}

// AccountInfo is the GET /account response (legacy request_status format,
// per the spec).
type AccountInfo struct {
	RequestStatus string     `json:"request_status"`
	Message       string     `json:"message"`
	Account       Account    `json:"account"`
	APIKeyInfo    APIKeyInfo `json:"api_key_info"`
}
