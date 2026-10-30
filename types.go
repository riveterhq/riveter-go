package riveter

import (
	"encoding/json"
	"fmt"
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

// CreditEstimate is the kickoff-time estimate. Maximum is the contract — a
// run never charges more than it.
type CreditEstimate struct {
	Minimum        float64 `json:"minimum"`
	Maximum        float64 `json:"maximum"`
	ChargedUpfront bool    `json:"charged_upfront"`
}

// CreditDetail is the estimate next to what has been charged. CreditsRemaining
// and SufficientCredits are set on estimates and credit errors only.
type CreditDetail struct {
	Estimate          *CreditEstimate `json:"estimate"`
	CreditsUsed       float64         `json:"credits_used"`
	CreditsRemaining  *float64        `json:"credits_remaining,omitempty"`
	SufficientCredits *bool           `json:"sufficient_credits,omitempty"`
}

// LegacyNotice is set on a response when the request took a legacy mode (a
// legacy enrichment's id, a monitor on one, legacy-pipeline column fields):
// what to send instead. The response also carries the Deprecation: true header.
type LegacyNotice struct {
	Notice string `json:"notice"`
	// Successor is "METHOD /v1/path" of the successor.
	Successor string `json:"successor"`
	Docs      string `json:"docs"`
}

// Run is the uniform envelope returned by kickoffs and every /runs endpoint.
// Endpoint-specific kickoff extras (DatasetID, CreditsCharged, ...) are only
// set where the endpoint returns them; Output is nil until the run finishes.
// An EstimateCost kickoff returns this shape with EstimateCost set, no ID, and the priced
// CreditDetail. Legacy is set when the kickoff took a legacy mode.
type Run struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	Progress       *RunProgress    `json:"progress"`
	CreditsUsed    float64         `json:"credits_used"`
	CreditDetail   *CreditDetail   `json:"credit_detail"`
	EstimateCost   bool            `json:"estimate_cost"`
	AppURL         string          `json:"app_url"`
	ResultURL      string          `json:"result_url"`
	StartedAt      *time.Time      `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at"`
	Error          *RunError       `json:"error"`
	StoppedReason  string          `json:"stopped_reason"`
	EnrichmentID   string          `json:"enrichment_id"`
	EnrichmentName string          `json:"enrichment_name"`
	DatasetID      string          `json:"dataset_id"`
	ExtractionID   string          `json:"extraction_id"`
	MonitorID      string          `json:"monitor_id"`
	WebhookURL     string          `json:"webhook_url"`
	Output         json.RawMessage `json:"output"`
	Legacy         *LegacyNotice   `json:"legacy"`

	MaxItems            int            `json:"max_items"`
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
	StoppedReason  string     `json:"stopped_reason"`
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

// DatasetFillTarget is one box of cells to fill on a dataset: the named
// columns × the named rows (all when omitted). Positions are 1-based.
type DatasetFillTarget struct {
	Columns   []string `json:"columns,omitempty"`
	Rows      []string `json:"rows,omitempty"`
	Positions []int    `json:"positions,omitempty"`
}

// EnrichParams is the request body for Enrich. Provide rows via Input or
// DatasetID (a Dataset), and column config via EnrichmentID, Output, or Prompt+Attributes.
// On a Dataset, Columns / Rows / Positions (or Targets) scope the fill and
// States names the cell states to re-run (pending, stale, and failed by
// default; Force runs every cell).
type EnrichParams struct {
	EnrichmentID        string                        `json:"enrichment_id,omitempty"`
	Output              map[string]OutputColumnConfig `json:"output,omitempty"`
	Prompt              string                        `json:"prompt,omitempty"`
	Attributes          []string                      `json:"attributes,omitempty"`
	Input               map[string][]string           `json:"input,omitempty"`
	DatasetID           string                        `json:"dataset_id,omitempty"`
	Columns             []string                      `json:"columns,omitempty"`
	Rows                []string                      `json:"rows,omitempty"`
	Positions           []int                         `json:"positions,omitempty"`
	Targets             []DatasetFillTarget           `json:"targets,omitempty"`
	States              []string                      `json:"states,omitempty"`
	Force               bool                          `json:"force,omitempty"`
	RunKey              string                        `json:"run_key,omitempty"`
	WebhookURL          string                        `json:"webhook_url,omitempty"`
	AllowDuplicateInput bool                          `json:"allow_duplicate_input,omitempty"`
	// EstimateCost prices the run without starting it; MaxCredits refuses a run
	// whose estimated maximum exceeds it (422 credit_cap_exceeded).
	EstimateCost bool     `json:"estimate_cost,omitempty"`
	MaxCredits   *float64 `json:"max_credits,omitempty"`
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

// OutputColumnConfig is one output column of an inline Enrich: Prompt, an
// optional OutputSchema, an optional pinned Tool (Contexts is accepted and
// ignored). Every other field is the legacy pipeline's: a column that sets one
// keeps the run on the legacy pipeline and the response carries Legacy.
type OutputColumnConfig struct {
	Prompt string `json:"prompt,omitempty"`
	// OutputSchema is a JSON Schema for the cell value. Omitted, a plain-value type is inferred.
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	// Tool is a name from Tools.List: one call of it per row is the cell. With the
	// tool parameters below it is the legacy tool-only column.
	Tool     string   `json:"tool,omitempty"`
	Contexts []string `json:"contexts,omitempty"`
	// Legacy pipeline fields.
	Tools            []string          `json:"tools,omitempty"`
	MaxToolCalls     int               `json:"max_tool_calls,omitempty"`
	Format           string            `json:"format,omitempty"`
	FormatDetails    *FormatDetails    `json:"format_details,omitempty"`
	RunWhen          string            `json:"run_when,omitempty"`
	RunWhenConfig    *RunWhenConfig    `json:"run_when_config,omitempty"`
	URL              string            `json:"url,omitempty"`
	Query            string            `json:"query,omitempty"`
	DateStart        string            `json:"date_start,omitempty"`
	DateEnd          string            `json:"date_end,omitempty"`
	Code             string            `json:"code,omitempty"`
	Args             map[string]string `json:"args,omitempty"`
	ProxyCountryCode string            `json:"proxy_country_code,omitempty"`
	WaitLonger       bool              `json:"wait_longer,omitempty"`
	SkipCache        bool              `json:"skip_cache,omitempty"`
}

// ListRunsParams filters GET /runs.
type ListRunsParams struct {
	// Comma-separated run types: enrichment, dataset_build, extraction, scrape, quick_search.
	Type          string
	Status        string
	EnrichmentID  string
	DatasetID     string
	MonitorID     string
	CreatedAfter  string
	CreatedBefore string
	Page          int
	PerPage       int
}

// ListEnrichmentsParams filters EnrichmentsService.List. Zero values are omitted.
type ListEnrichmentsParams struct {
	// Comma-separated: pending, enqueued, processing, success, stopped.
	Status string
	// Case-insensitive substring match on the enrichment name.
	Name    string
	Page    int
	PerPage int // max 50
}

// EnrichmentSummary is the compact list-endpoint shape of an enrichment.
// Column configuration is on EnrichmentsService.Get. Status, AppURL, and
// DatasetID (the dataset copy to fill instead) are set on legacy enrichments only.
type EnrichmentSummary struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Status               string     `json:"status"`
	AppURL               string     `json:"app_url"`
	DatasetID            string     `json:"dataset_id"`
	ColumnCount          int        `json:"column_count"`
	InputColumns         []string   `json:"input_columns"`
	OutputColumns        []string   `json:"output_columns"`
	ColumnNamesTruncated bool       `json:"column_names_truncated"`
	CreatedAt            *time.Time `json:"created_at"`
	UpdatedAt            *time.Time `json:"updated_at"`
}

// OutputColumnSpec is one output column of a dataset-engine enrichment. Tool
// pins the column to one tool (a name from Tools.List): the row agent makes one
// call of it per row and the call's result is the cell.
type OutputColumnSpec struct {
	Name string `json:"name"`
	// Prompt is what to find for this column, on top of the enrichment's task prompt.
	Prompt string `json:"prompt,omitempty"`
	// OutputSchema is a JSON Schema for the cell value. Omitted, a plain-value
	// type is inferred; {"type": "string"} keeps free text.
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	Tool         string         `json:"tool,omitempty"`
}

// Enrichment is an enrichment: Prompt, InputColumns, Output as a list of
// OutputColumnSpec, Tools, the budget fields, and Version. DatasetID / AppURL
// are set on Enrichments.Create. A legacy enrichment's id answers its older
// shape (the legacy API docs) with Legacy set; Output is a json.RawMessage so
// that shape still decodes, and OutputColumns reads the current one.
type Enrichment struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Prompt       string          `json:"prompt"`
	InputColumns []string        `json:"input_columns"`
	Output       json.RawMessage `json:"output"`
	Tools        []string        `json:"tools"`
	// BudgetCreditsPerRow is the most the row agent may spend on one row. It moves
	// by 1.5 per output column added or removed unless BudgetLocked.
	BudgetCreditsPerRow float64 `json:"budget_credits_per_row"`
	BudgetLocked        bool    `json:"budget_locked"`
	// Version goes up by one on every column change; a cell filled under an
	// older version reads stale.
	Version   int           `json:"version"`
	CreatedAt *time.Time    `json:"created_at"`
	UpdatedAt *time.Time    `json:"updated_at"`
	AppURL    string        `json:"app_url"`
	DatasetID string        `json:"dataset_id"`
	Legacy    *LegacyNotice `json:"legacy"`
}

// OutputColumns decodes Output as the recipe's column list. It returns nil, nil
// when Output is empty or not a list (a legacy enrichment's shape).
func (e *Enrichment) OutputColumns() ([]OutputColumnSpec, error) {
	if len(e.Output) == 0 || e.Output[0] != '[' {
		return nil, nil
	}
	var columns []OutputColumnSpec
	if err := json.Unmarshal(e.Output, &columns); err != nil {
		return nil, fmt.Errorf("riveter: decode enrichment output: %w", err)
	}
	return columns, nil
}

// CreateEnrichmentParams is the request body for Enrichments.Create. Prompt +
// Output save a recipe (an empty Dataset bound to it comes back as DatasetID on
// the response); DatasetID alone takes a Dataset's own enrichment. Output
// entries are OutputColumnSpec values or plain column-name strings.
type CreateEnrichmentParams struct {
	DatasetID       string   `json:"dataset_id,omitempty"`
	Name            string   `json:"name,omitempty"`
	Prompt          string   `json:"prompt,omitempty"`
	InputColumns    []string `json:"input_columns,omitempty"`
	Output          []any    `json:"output,omitempty"`
	Budget          *float64 `json:"budget,omitempty"`
	BudgetLocked    *bool    `json:"budget_locked,omitempty"`
	OnlyTools       []string `json:"only_tools,omitempty"`
	AdditionalTools []string `json:"additional_tools,omitempty"`
}

// OutputColumnOperation is one output column operation of UpdateEnrichmentParams
// / UpdateDatasetParams, keyed by column name: add or edit (Prompt /
// OutputSchema / Tool), Rename, or Delete. UnpinTool sends tool: null, which
// unpins the column's tool.
type OutputColumnOperation struct {
	Prompt       string         `json:"prompt,omitempty"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	Tool         string         `json:"tool,omitempty"`
	UnpinTool    bool           `json:"-"`
	Rename       string         `json:"rename,omitempty"`
	Delete       bool           `json:"delete,omitempty"`
}

// MarshalJSON writes "tool": null when UnpinTool is set.
func (op OutputColumnOperation) MarshalJSON() ([]byte, error) {
	type plain OutputColumnOperation
	if !op.UnpinTool {
		return json.Marshal(plain(op))
	}
	// The outer Tool shadows the embedded one, so a nil pointer writes null.
	wire := struct {
		plain
		Tool *string `json:"tool"`
	}{plain: plain(op)}
	return json.Marshal(wire)
}

// UpdateEnrichmentParams is the request body for Enrichments.Update: Name,
// Prompt, InputColumns, Budget, BudgetLocked, the tool lists, and Output
// operations keyed by column name.
type UpdateEnrichmentParams struct {
	Name            string                           `json:"name,omitempty"`
	Prompt          string                           `json:"prompt,omitempty"`
	InputColumns    []string                         `json:"input_columns,omitempty"`
	Budget          *float64                         `json:"budget,omitempty"`
	BudgetLocked    *bool                            `json:"budget_locked,omitempty"`
	OnlyTools       []string                         `json:"only_tools,omitempty"`
	AdditionalTools []string                         `json:"additional_tools,omitempty"`
	Output          map[string]OutputColumnOperation `json:"output,omitempty"`
}

// DatasetCell is one cell. State is pending / running / ok / not_found /
// failed / manual / stale; Reason and Error are set on failed cells only.
type DatasetCell struct {
	Value   any      `json:"value"`
	State   string   `json:"state"`
	Reason  string   `json:"reason,omitempty"`
	Error   string   `json:"error,omitempty"`
	Sources []string `json:"sources"`
}

// DatasetRow is a dataset row: Input by input column name, Cells by output
// column name. Position is 1-based; Credits is the row's charge across its runs.
type DatasetRow struct {
	ID             string                 `json:"id"`
	Position       int                    `json:"position"`
	Input          map[string]*string     `json:"input"`
	Cells          map[string]DatasetCell `json:"cells"`
	Credits        float64                `json:"credits"`
	NotFoundReason string                 `json:"not_found_reason"`
	CreatedAt      *time.Time             `json:"created_at"`
	UpdatedAt      *time.Time             `json:"updated_at"`
}

// DatasetRunSummary is a run on a dataset as the dataset lists it. Type is
// dataset_build (list generation) or enrichment (a fill).
type DatasetRunSummary struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	CreatedAt  *time.Time `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// Dataset is a dataset on the dataset engine (ds_...). Run is set on
// Datasets.Create / Datasets.BuildRows when a build or fill started; the column
// configuration is on Enrichments.Get(EnrichmentID). EstimateCost and
// CreditDetail are set on an EstimateCost call.
type Dataset struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	InputColumns     []string            `json:"input_columns"`
	IndexColumn      string              `json:"index_column"`
	EnrichmentID     string              `json:"enrichment_id"`
	OutputColumns    []string            `json:"output_columns"`
	RowCount         int                 `json:"row_count"`
	RowsExpired      bool                `json:"rows_expired"`
	ImportInProgress bool                `json:"import_in_progress"`
	Runs             []DatasetRunSummary `json:"runs"`
	Run              *Run                `json:"run"`
	EstimateCost     bool                `json:"estimate_cost"`
	CreditDetail     *CreditDetail       `json:"credit_detail"`
	CreatedAt        *time.Time          `json:"created_at"`
	UpdatedAt        *time.Time          `json:"updated_at"`
}

// DatasetListItem is the compact list-endpoint shape of a dataset.
type DatasetListItem struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	InputColumnCount  int                `json:"input_column_count"`
	OutputColumnCount int                `json:"output_column_count"`
	RowCount          int                `json:"row_count"`
	RowsExpired       bool               `json:"rows_expired"`
	NewestRun         *DatasetRunSummary `json:"newest_run"`
	CreatedAt         *time.Time         `json:"created_at"`
	UpdatedAt         *time.Time         `json:"updated_at"`
}

// CreateDatasetParams is the request body for Datasets.Create: rows sent inline
// (Rows, ≤ 100 per call; more through AddRows / ImportRows), generated from
// Prompt + Count (a paid list build on a credit Budget, default 50 + 1.5 per row;
// EstimateCost prices it), or empty (Name /
// InputColumns alone). EnrichmentID binds a saved recipe; AutoFill starts the
// fill once the rows land.
type CreateDatasetParams struct {
	Name           string              `json:"name,omitempty"`
	InputColumns   []string            `json:"input_columns,omitempty"`
	Rows           []map[string]string `json:"rows,omitempty"`
	Prompt         string              `json:"prompt,omitempty"`
	Count          int                 `json:"count,omitempty"`
	Budget         *float64            `json:"budget,omitempty"`
	EnrichmentID   string              `json:"enrichment_id,omitempty"`
	Attributes     []string            `json:"attributes,omitempty"`
	AutoFill       bool                `json:"auto_fill,omitempty"`
	WebhookURL     string              `json:"webhook_url,omitempty"`
	FillWebhookURL string              `json:"fill_webhook_url,omitempty"`
	EstimateCost   bool                `json:"estimate_cost,omitempty"`
	MaxCredits     *float64            `json:"max_credits,omitempty"`
}

// InputColumnOperation is one input column operation of UpdateDatasetParams,
// keyed by column name: add (the zero value), Rename, or Delete.
type InputColumnOperation struct {
	Rename string `json:"rename,omitempty"`
	Delete bool   `json:"delete,omitempty"`
}

// UpdateDatasetParams is the request body for Datasets.Update: Name,
// IndexColumn, InputColumns and Output column operations by name, and the
// bound enrichment's Budget / BudgetLocked, in one transaction.
type UpdateDatasetParams struct {
	Name         string                           `json:"name,omitempty"`
	IndexColumn  string                           `json:"index_column,omitempty"`
	InputColumns map[string]InputColumnOperation  `json:"input_columns,omitempty"`
	Output       map[string]OutputColumnOperation `json:"output,omitempty"`
	Budget       *float64                         `json:"budget,omitempty"`
	BudgetLocked *bool                            `json:"budget_locked,omitempty"`
}

// ListDatasetRowsParams filters Datasets.ListRows. Zero values are omitted.
type ListDatasetRowsParams struct {
	// Limit is 1–1000 (default 100).
	Limit int
	// Cursor is the previous page's NextCursor.
	Cursor int
	// UpdatedSince (ISO 8601) returns only rows changed since then.
	UpdatedSince string
	// Columns are the column names (input or output) to keep; empty keeps every column.
	Columns []string
}

// CompactDatasetRow is a row as Datasets.ListRowsCompact answers it: the values
// alone. Cells holds each output column's value (nil when the cell has none).
// Address the row by Position elsewhere.
type CompactDatasetRow struct {
	Position int                `json:"position"`
	Input    map[string]*string `json:"input"`
	Cells    map[string]any     `json:"cells"`
}

// GetDatasetSummaryParams narrows Datasets.Summary.
type GetDatasetSummaryParams struct {
	// Columns are the column names to summarize; empty summarizes every column.
	Columns []string
	// TopValues is how many of a column's most common values to list (1–100, default 20).
	TopValues int
}

// DatasetColumnValueCount is one of a column's most common values (a string is
// cut at 200 characters) and its row count.
type DatasetColumnValueCount struct {
	Value any `json:"value"`
	Count int `json:"count"`
}

// DatasetColumnSummary is one column's counts over every row. States is set for
// output columns only: the column's cells by state (pending, stale, failed,
// not_found, ok, manual).
type DatasetColumnSummary struct {
	Name           string                    `json:"name"`
	Kind           string                    `json:"kind"`
	States         map[string]int            `json:"states"`
	Filled         int                       `json:"filled"`
	DistinctValues int                       `json:"distinct_values"`
	Values         []DatasetColumnValueCount `json:"values"`
}

// DatasetSummary is the Datasets.Summary response.
type DatasetSummary struct {
	RowCount int                    `json:"row_count"`
	Columns  []DatasetColumnSummary `json:"columns"`
}

// DatasetRowsAdded is the Datasets.AddRows response.
type DatasetRowsAdded struct {
	Rows              []DatasetRow `json:"rows"`
	SkippedDuplicates int          `json:"skipped_duplicates"`
	RowCount          int          `json:"row_count"`
}

// DeleteDatasetRowsParams names up to 100 rows by IDs (row_...) and / or
// 1-based Positions.
type DeleteDatasetRowsParams struct {
	IDs       []string
	Positions []int
}

// DatasetRowsDeleted is the Datasets.DeleteRows response.
type DatasetRowsDeleted struct {
	Deleted  int `json:"deleted"`
	RowCount int `json:"row_count"`
}

// UpdateDatasetRowParams is the request body for Datasets.UpdateRow. Input sets
// input values by input column name (a nil pointer removes one; refused with 409
// row_in_use while the row runs). Cells sets output cells by output column name
// as manual cells (nil clears one; skipped by a fill unless Force).
type UpdateDatasetRowParams struct {
	Input map[string]*string `json:"input,omitempty"`
	Cells map[string]any     `json:"cells,omitempty"`
}

// DatasetImportResult is the Datasets.ImportRows response. Import.Status is
// "done" when the rows landed in the call, "importing" when a job is appending
// them (Datasets.Get reports ImportInProgress until it does).
type DatasetImportResult struct {
	Import struct {
		Status            string `json:"status"`
		Rows              int    `json:"rows"`
		RowsAdded         int    `json:"rows_added"`
		SkippedDuplicates int    `json:"skipped_duplicates"`
	} `json:"import"`
	RowCount int `json:"row_count"`
}

// BuildDatasetRowsParams is the request body for Datasets.BuildRows: generate
// Count more rows (a paid list build on a credit Budget, default 50 + 1.5 per row);
// AutoFill fills them once they land.
type BuildDatasetRowsParams struct {
	Prompt         string   `json:"prompt,omitempty"`
	Count          int      `json:"count"`
	Budget         *float64 `json:"budget,omitempty"`
	AutoFill       bool     `json:"auto_fill,omitempty"`
	WebhookURL     string   `json:"webhook_url,omitempty"`
	FillWebhookURL string   `json:"fill_webhook_url,omitempty"`
	EstimateCost   bool     `json:"estimate_cost,omitempty"`
	MaxCredits     *float64 `json:"max_credits,omitempty"`
}

// ToolInput is one input of a tool in the catalog. Type is nil when the input
// takes more than one shape.
type ToolInput struct {
	Name        string  `json:"name"`
	Type        *string `json:"type"`
	Required    bool    `json:"required"`
	Description *string `json:"description"`
}

// ToolCatalogEntry is one tool of Tools.List.
type ToolCatalogEntry struct {
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	CreditsPerCall float64     `json:"credits_per_call"`
	PriceNote      string      `json:"price_note"`
	Default        bool        `json:"default"`
	RequiresOptIn  bool        `json:"requires_opt_in"`
	Enabled        bool        `json:"enabled"`
	RequiresAPIKey bool        `json:"requires_api_key"`
	APIKeyLinked   *bool       `json:"api_key_linked"`
	Companions     []string    `json:"companions"`
	Inputs         []ToolInput `json:"inputs"`
	InputGroups    [][]string  `json:"input_groups"`
	Returns        string      `json:"returns"`
}

// ToolCatalog is the Tools.List response: every tool with its price, whether
// this account may use it, and the budget defaults.
type ToolCatalog struct {
	Tools  []ToolCatalogEntry `json:"tools"`
	Budget struct {
		// DefaultCreditsPerOutputColumn times the output columns is a new enrichment's budget per row.
		DefaultCreditsPerOutputColumn float64 `json:"default_credits_per_output_column"`
		MinCreditsPerRow              float64 `json:"min_credits_per_row"`
		MaxCreditsPerRow              float64 `json:"max_credits_per_row"`
	} `json:"budget"`
}

// BuildConfiguredDatasetParams is the body for ConfiguredDatasets.Build.
type BuildConfiguredDatasetParams struct {
	Parameters        map[string]string `json:"parameters,omitempty"`
	Tier              string            `json:"tier,omitempty"`
	DatasetWebhookURL string            `json:"dataset_webhook_url,omitempty"`
	EstimateCost      bool              `json:"estimate_cost,omitempty"`
	MaxCredits        *float64          `json:"max_credits,omitempty"`
}

// CreateExtractionParams is the request body for Extractions.Create.
type CreateExtractionParams struct {
	StartingURL            string         `json:"starting_url"`
	GoalDescription        string         `json:"goal_description"`
	OutputRecordJSONSchema map[string]any `json:"output_record_json_schema"`
	Name                   string         `json:"name,omitempty"`
	RequiredKeys           []string       `json:"required_keys,omitempty"`
	EstimateCost           bool           `json:"estimate_cost,omitempty"`
	MaxCredits             *float64       `json:"max_credits,omitempty"`
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
	// Set on the Extractions.Create response (and on an EstimateCost call of it).
	CreditDetail *CreditDetail `json:"credit_detail"`
	EstimateCost bool          `json:"estimate_cost"`
}

// RunExtractionParams is the request body for Extractions.Run.
type RunExtractionParams struct {
	Variables    map[string]string `json:"variables,omitempty"`
	WebhookURL   string            `json:"webhook_url,omitempty"`
	EstimateCost bool              `json:"estimate_cost,omitempty"`
	MaxCredits   *float64          `json:"max_credits,omitempty"`
}

// CreateMonitorParams is the request body for Monitors.Create: DatasetID (or
// EnrichmentID naming the dataset's enrichment), the cadence, and optionally
// Columns, the output column names each tick fills (every one when omitted).
// AlertRule is each_run / on_diff / disabled; OutputFormat is current_only /
// include_diffs. Minute and Hour are always sent (0 is valid for both).
// Creation is free; EstimateCost prices ONE tick and MaxCredits refuses the
// monitor when that estimate is higher. Input belongs to a monitor on a legacy
// enrichment only.
type CreateMonitorParams struct {
	DatasetID      string              `json:"dataset_id,omitempty"`
	EnrichmentID   string              `json:"enrichment_id,omitempty"`
	Columns        []string            `json:"columns,omitempty"`
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
	EstimateCost   bool                `json:"estimate_cost,omitempty"`
	MaxCredits     *float64            `json:"max_credits,omitempty"`
}

// UpdateMonitorParams is the request body for Monitors.Update. Enabled is a
// pointer so `false` (pause) still serializes. Columns replaces the output
// columns a tick fills; EveryColumn sends columns: null (every output column again).
type UpdateMonitorParams struct {
	Enabled     *bool    `json:"enabled,omitempty"`
	WebhookURL  string   `json:"webhook_url,omitempty"`
	Columns     []string `json:"-"`
	EveryColumn bool     `json:"-"`
}

// MarshalJSON writes Columns, or "columns": null when EveryColumn is set.
func (p UpdateMonitorParams) MarshalJSON() ([]byte, error) {
	type plain UpdateMonitorParams
	if p.EveryColumn {
		return json.Marshal(struct {
			plain
			Columns *[]string `json:"columns"`
		}{plain: plain(p)})
	}
	return json.Marshal(struct {
		plain
		Columns []string `json:"columns,omitempty"`
	}{plain: plain(p), Columns: p.Columns})
}

// Monitor fills DatasetID's Columns (every output column when empty) again on a
// schedule. EnrichmentName and HasInput belong to a monitor on a legacy
// enrichment; Legacy is set on Get / Update of one (list items carry the JSON
// boolean true, which decodes as nil here: read LegacyListItem).
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
	DatasetID       string     `json:"dataset_id"`
	DatasetName     string     `json:"dataset_name"`
	Columns         []string   `json:"columns"`
	EnrichmentID    string     `json:"enrichment_id"`
	EnrichmentName  string     `json:"enrichment_name"`
	CreatedAt       *time.Time `json:"created_at"`
	HasInput        bool       `json:"has_input"`
	// Legacy is the notice on Get / Update of a monitor on a legacy enrichment;
	// LegacyListItem is true on such an item of List.
	Legacy         *LegacyNotice `json:"-"`
	LegacyListItem bool          `json:"-"`
	// Set on an EstimateCost call of Monitors.Create: the estimate for one tick.
	CreditDetail *CreditDetail `json:"credit_detail"`
	EstimateCost bool          `json:"estimate_cost"`
}

// UnmarshalJSON reads "legacy" as either the boolean of a list item or the notice object.
func (m *Monitor) UnmarshalJSON(data []byte) error {
	type plain Monitor
	wire := struct {
		*plain
		Legacy json.RawMessage `json:"legacy"`
	}{plain: (*plain)(m)}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if len(wire.Legacy) == 0 || string(wire.Legacy) == "null" {
		return nil
	}
	if wire.Legacy[0] == '{' {
		m.Legacy = &LegacyNotice{}
		return json.Unmarshal(wire.Legacy, m.Legacy)
	}
	return json.Unmarshal(wire.Legacy, &m.LegacyListItem)
}

// ListMonitorRunsParams filters GET /monitors/{id}/runs.
type ListMonitorRunsParams struct {
	Status  string
	Page    int
	PerPage int
}

// QuickSearchParams is the request body for QuickSearch.
type QuickSearchParams struct {
	Query        string   `json:"query"`
	DateStart    string   `json:"date_start,omitempty"`
	DateEnd      string   `json:"date_end,omitempty"`
	RunKey       string   `json:"run_key,omitempty"`
	EstimateCost bool     `json:"estimate_cost,omitempty"`
	MaxCredits   *float64 `json:"max_credits,omitempty"`
}

// SearchAgentParams is the request body for SearchAgent. Wait is the
// long-poll budget in seconds (0–50; nil means the server default of 50).
type SearchAgentParams struct {
	Prompt       string         `json:"prompt"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	Wait         *int           `json:"wait,omitempty"`
	RunKey       string         `json:"run_key,omitempty"`
	EstimateCost bool           `json:"estimate_cost,omitempty"`
	MaxCredits   *float64       `json:"max_credits,omitempty"`
}

// ScrapeParams is the request body for Scrape.
type ScrapeParams struct {
	URL              string   `json:"url"`
	ProxyCountryCode string   `json:"proxy_country_code,omitempty"`
	SkipCache        bool     `json:"skip_cache,omitempty"`
	EstimateCost     bool     `json:"estimate_cost,omitempty"`
	MaxCredits       *float64 `json:"max_credits,omitempty"`
}

// ScrapeResult is the synchronous scrape result (legacy request_status
// format, per the spec).
type ScrapeResult struct {
	RequestStatus   string        `json:"request_status"`
	Text            string        `json:"text"`
	URL             string        `json:"url"`
	BaseURLForLinks string        `json:"base_url_for_links"`
	StatusCode      int           `json:"status_code"`
	PossiblyBlocked bool          `json:"possibly_blocked"`
	CreditUsed      float64       `json:"credit_used"`
	CreditDetail    *CreditDetail `json:"credit_detail"`
	EstimateCost    bool          `json:"estimate_cost"`
	RiveterAppLink  string        `json:"riveter_app_link"`
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
