package riveter

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// EnrichmentsService manages saved enrichment configurations.
type EnrichmentsService struct {
	client *Client
}

// List returns a paginated compact list of enrichments: names, status, column
// names and counts. (operationId: listEnrichments)
func (s *EnrichmentsService) List(ctx context.Context, params *ListEnrichmentsParams) (*EnrichmentsPage, error) {
	query := url.Values{}
	page := 1
	if params != nil {
		if params.Status != "" {
			query.Set("status", params.Status)
		}
		if params.Name != "" {
			query.Set("name", params.Name)
		}
		if params.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(params.PerPage))
		}
		if params.Page > 0 {
			page = params.Page
		}
	}
	return s.client.fetchEnrichmentsPage(ctx, query, page)
}

// EnrichmentsPage is one page of enrichments plus the pagination cursor.
type EnrichmentsPage struct {
	Enrichments []EnrichmentSummary `json:"enrichments"`
	Pagination  Pagination          `json:"pagination"`

	fetchPage func(ctx context.Context, page int) (*EnrichmentsPage, error)
}

// HasNextPage reports whether another page follows this one.
func (p *EnrichmentsPage) HasNextPage() bool {
	return p.Pagination.Page < p.Pagination.TotalPages
}

// NextPage fetches the following page, or ErrNoNextPage on the last one.
func (p *EnrichmentsPage) NextPage(ctx context.Context) (*EnrichmentsPage, error) {
	if !p.HasNextPage() {
		return nil, ErrNoNextPage
	}
	return p.fetchPage(ctx, p.Pagination.Page+1)
}

func (c *Client) fetchEnrichmentsPage(ctx context.Context, baseQuery url.Values, page int) (*EnrichmentsPage, error) {
	query := url.Values{}
	for key, values := range baseQuery {
		query[key] = values
	}
	query.Set("page", strconv.Itoa(page))

	enrichmentsPage := &EnrichmentsPage{}
	if err := c.request(ctx, http.MethodGet, "/enrichments", query, nil, enrichmentsPage); err != nil {
		return nil, err
	}
	enrichmentsPage.fetchPage = func(ctx context.Context, nextPage int) (*EnrichmentsPage, error) {
		return c.fetchEnrichmentsPage(ctx, baseQuery, nextPage)
	}
	return enrichmentsPage, nil
}

// Create saves an enrichment (no run). A recipe body — Prompt + Output — saves
// the column configuration and makes an empty Dataset bound to it (DatasetID /
// AppURL on the response); DatasetID alone takes a Dataset's own enrichment. (operationId: createEnrichment)
func (s *EnrichmentsService) Create(ctx context.Context, params CreateEnrichmentParams) (*Enrichment, error) {
	enrichment := &Enrichment{}
	if err := s.client.request(ctx, http.MethodPost, "/enrichments", nil, params, enrichment); err != nil {
		return nil, err
	}
	return enrichment, nil
}

// Delete deletes a saved enrichment. Refused with 409 enrichment_in_use while
// a fill runs on it. (operationId: deleteEnrichment)
func (s *EnrichmentsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, http.MethodDelete, "/enrichments/"+url.PathEscape(id), nil, nil, nil)
}

// Get returns the enrichment's input columns and output column configuration. (operationId: getEnrichment)
func (s *EnrichmentsService) Get(ctx context.Context, id string) (*Enrichment, error) {
	enrichment := &Enrichment{}
	if err := s.client.request(ctx, http.MethodGet, "/enrichments/"+url.PathEscape(id), nil, nil, enrichment); err != nil {
		return nil, err
	}
	return enrichment, nil
}

// Update edits the enrichment: name, prompt, input columns, budget, tools, and
// Output column operations (add / rename / delete by name). Refused with 409
// enrichment_in_use while a fill runs on it. (operationId: updateEnrichment)
func (s *EnrichmentsService) Update(ctx context.Context, id string, params UpdateEnrichmentParams) (*Enrichment, error) {
	enrichment := &Enrichment{}
	if err := s.client.request(ctx, http.MethodPatch, "/enrichments/"+url.PathEscape(id), nil, params, enrichment); err != nil {
		return nil, err
	}
	return enrichment, nil
}

// DatasetsService manages datasets: kept tables of input rows and the output
// columns an enrichment fills (ds_...).
type DatasetsService struct {
	client *Client
}

// List returns the account's datasets, newest first (200 at most). (operationId: listDatasets)
func (s *DatasetsService) List(ctx context.Context) ([]DatasetListItem, error) {
	var response struct {
		Datasets []DatasetListItem `json:"datasets"`
	}
	if err := s.client.request(ctx, http.MethodGet, "/datasets", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Datasets, nil
}

// Create creates a dataset from inline Rows, from Prompt + Count (a paid list
// build; set EstimateCost to price it — Dataset.CreditDetail carries the
// estimate and nothing is created), or empty. Dataset.Run is set when a build
// or fill started. (operationId: buildDataset)
func (s *DatasetsService) Create(ctx context.Context, params CreateDatasetParams) (*Dataset, error) {
	dataset := &Dataset{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets", nil, params, dataset); err != nil {
		return nil, err
	}
	return dataset, nil
}

// Get returns the dataset's columns, row count, and recent runs. (operationId: getDataset)
func (s *DatasetsService) Get(ctx context.Context, id string) (*Dataset, error) {
	dataset := &Dataset{}
	if err := s.client.request(ctx, http.MethodGet, "/datasets/"+url.PathEscape(id), nil, nil, dataset); err != nil {
		return nil, err
	}
	return dataset, nil
}

// Update edits the dataset's name, index column, input and output columns, and
// the bound enrichment's budget in one transaction. Refused with 409
// dataset_in_use while a fill runs. (operationId: updateDataset)
func (s *DatasetsService) Update(ctx context.Context, id string, params UpdateDatasetParams) (*Dataset, error) {
	dataset := &Dataset{}
	if err := s.client.request(ctx, http.MethodPatch, "/datasets/"+url.PathEscape(id), nil, params, dataset); err != nil {
		return nil, err
	}
	return dataset, nil
}

// Delete deletes the dataset and its rows. Refused with 409 dataset_in_use
// while a fill runs. (operationId: deleteDataset)
func (s *DatasetsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, http.MethodDelete, "/datasets/"+url.PathEscape(id), nil, nil, nil)
}

// ListRows returns a page of rows in position order. Columns keeps only the
// named columns. (operationId: listDatasetRows)
func (s *DatasetsService) ListRows(ctx context.Context, id string, params *ListDatasetRowsParams) (*DatasetRowsPage, error) {
	query, cursor := datasetRowsQuery(params)
	return s.client.fetchDatasetRowsPage(ctx, id, query, cursor)
}

// ListRowsCompact returns a page of rows in the values-only shape
// (CompactDatasetRow: Position, Input, Cells as bare values), so a 1,000-row
// page of one column (params.Columns) is one small call. (operationId: listDatasetRows)
func (s *DatasetsService) ListRowsCompact(ctx context.Context, id string, params *ListDatasetRowsParams) (*CompactDatasetRowsPage, error) {
	query, cursor := datasetRowsQuery(params)
	query.Set("compact", "true")
	return s.client.fetchCompactDatasetRowsPage(ctx, id, query, cursor)
}

func datasetRowsQuery(params *ListDatasetRowsParams) (url.Values, int) {
	query := url.Values{}
	if params == nil {
		return query, 0
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.UpdatedSince != "" {
		query.Set("updated_since", params.UpdatedSince)
	}
	if len(params.Columns) > 0 {
		query.Set("columns", strings.Join(params.Columns, ","))
	}
	return query, params.Cursor
}

// Summary returns per-column counts over every row, no rows read: an output
// column's cells by state and its most common values; an input column's filled
// rows and values. (operationId: getDatasetSummary)
func (s *DatasetsService) Summary(ctx context.Context, id string, params *GetDatasetSummaryParams) (*DatasetSummary, error) {
	query := url.Values{}
	if params != nil {
		if len(params.Columns) > 0 {
			query.Set("columns", strings.Join(params.Columns, ","))
		}
		if params.TopValues > 0 {
			query.Set("top_values", strconv.Itoa(params.TopValues))
		}
	}
	summary := &DatasetSummary{}
	if err := s.client.request(ctx, http.MethodGet, "/datasets/"+url.PathEscape(id)+"/summary", query, nil, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

// AddRows appends up to 100 rows keyed by input column name. Rows whose index
// column value a held row already carries are skipped. (operationId: addDatasetRows)
func (s *DatasetsService) AddRows(ctx context.Context, id string, rows []map[string]string) (*DatasetRowsAdded, error) {
	body := map[string]any{"rows": rows}
	added := &DatasetRowsAdded{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets/"+url.PathEscape(id)+"/rows", nil, body, added); err != nil {
		return nil, err
	}
	return added, nil
}

// DeleteRows deletes up to 100 rows by IDs and / or Positions. The other rows
// keep their positions. (operationId: deleteDatasetRows)
func (s *DatasetsService) DeleteRows(ctx context.Context, id string, params DeleteDatasetRowsParams) (*DatasetRowsDeleted, error) {
	query := url.Values{}
	if len(params.IDs) > 0 {
		query.Set("ids", strings.Join(params.IDs, ","))
	}
	if len(params.Positions) > 0 {
		query.Set("positions", joinInts(params.Positions))
	}
	deleted := &DatasetRowsDeleted{}
	if err := s.client.request(ctx, http.MethodDelete, "/datasets/"+url.PathEscape(id)+"/rows", query, nil, deleted); err != nil {
		return nil, err
	}
	return deleted, nil
}

// UpdateRow sets one row's input values and / or output cells by hand. rowID is
// a row_... id or a 1-based position. (operationId: updateDatasetRow)
func (s *DatasetsService) UpdateRow(ctx context.Context, id string, rowID string, params UpdateDatasetRowParams) (*DatasetRow, error) {
	row := &DatasetRow{}
	path := "/datasets/" + url.PathEscape(id) + "/rows/" + url.PathEscape(rowID)
	if err := s.client.request(ctx, http.MethodPatch, path, nil, params, row); err != nil {
		return nil, err
	}
	return row, nil
}

// ImportRows appends the rows of a CSV whose headers are input and / or output
// column names (output values land as manual cells). Big files are appended by
// a job: Import.Status is "importing" and Datasets.Get reports
// ImportInProgress until they land. (operationId: importDatasetRows)
func (s *DatasetsService) ImportRows(ctx context.Context, id string, csv string) (*DatasetImportResult, error) {
	body := &requestBody{content: []byte(csv), contentType: "text/csv"}
	result := &DatasetImportResult{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets/"+url.PathEscape(id)+"/rows/import", nil, body, result); err != nil {
		return nil, err
	}
	return result, nil
}

// BuildRows generates Count more rows for the dataset (a paid list build; set
// EstimateCost to price it). Dataset.Run is the build run. (operationId: buildDatasetRows)
func (s *DatasetsService) BuildRows(ctx context.Context, id string, params BuildDatasetRowsParams) (*Dataset, error) {
	dataset := &Dataset{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets/"+url.PathEscape(id)+"/rows/build", nil, params, dataset); err != nil {
		return nil, err
	}
	return dataset, nil
}

// PromoteColumn turns an output column into an input column, every row's value
// moving with it. (operationId: promoteDatasetColumn)
func (s *DatasetsService) PromoteColumn(ctx context.Context, id string, column string) (*Dataset, error) {
	body := map[string]string{"column": column}
	dataset := &Dataset{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets/"+url.PathEscape(id)+"/columns/promote", nil, body, dataset); err != nil {
		return nil, err
	}
	return dataset, nil
}

// Export returns the grid as a file: outputFormat "csv" (default when empty),
// "xlsx" (an Excel workbook), or "jsonl" (one JSON object per row). (operationId: exportDataset)
func (s *DatasetsService) Export(ctx context.Context, id string, outputFormat string) ([]byte, error) {
	query := url.Values{}
	if outputFormat != "" {
		query.Set("output_format", outputFormat)
	}
	var content []byte
	if err := s.client.request(ctx, http.MethodGet, "/datasets/"+url.PathEscape(id)+"/export", query, nil, &content); err != nil {
		return nil, err
	}
	return content, nil
}

// DatasetRowsPage is one page of dataset rows plus the cursor of the next.
type DatasetRowsPage struct {
	Rows []DatasetRow `json:"rows"`
	// NextCursor is nil on the last page.
	NextCursor *int `json:"next_cursor"`
	RowCount   int  `json:"row_count"`

	fetchPage func(ctx context.Context, cursor int) (*DatasetRowsPage, error)
}

// HasNextPage reports whether another page follows this one.
func (p *DatasetRowsPage) HasNextPage() bool {
	return p.NextCursor != nil
}

// NextPage fetches the following page, or ErrNoNextPage on the last one.
func (p *DatasetRowsPage) NextPage(ctx context.Context) (*DatasetRowsPage, error) {
	if !p.HasNextPage() {
		return nil, ErrNoNextPage
	}
	return p.fetchPage(ctx, *p.NextCursor)
}

func (c *Client) fetchDatasetRowsPage(ctx context.Context, datasetID string, baseQuery url.Values, cursor int) (*DatasetRowsPage, error) {
	page := &DatasetRowsPage{}
	if err := c.requestDatasetRows(ctx, datasetID, baseQuery, cursor, page); err != nil {
		return nil, err
	}
	page.fetchPage = func(ctx context.Context, nextCursor int) (*DatasetRowsPage, error) {
		return c.fetchDatasetRowsPage(ctx, datasetID, baseQuery, nextCursor)
	}
	return page, nil
}

// CompactDatasetRowsPage is one page of values-only rows (ListRowsCompact) plus
// the cursor of the next.
type CompactDatasetRowsPage struct {
	Rows []CompactDatasetRow `json:"rows"`
	// NextCursor is nil on the last page.
	NextCursor *int `json:"next_cursor"`
	RowCount   int  `json:"row_count"`

	fetchPage func(ctx context.Context, cursor int) (*CompactDatasetRowsPage, error)
}

// HasNextPage reports whether another page follows this one.
func (p *CompactDatasetRowsPage) HasNextPage() bool {
	return p.NextCursor != nil
}

// NextPage fetches the following page, or ErrNoNextPage on the last one.
func (p *CompactDatasetRowsPage) NextPage(ctx context.Context) (*CompactDatasetRowsPage, error) {
	if !p.HasNextPage() {
		return nil, ErrNoNextPage
	}
	return p.fetchPage(ctx, *p.NextCursor)
}

func (c *Client) fetchCompactDatasetRowsPage(ctx context.Context, datasetID string, baseQuery url.Values, cursor int) (*CompactDatasetRowsPage, error) {
	page := &CompactDatasetRowsPage{}
	if err := c.requestDatasetRows(ctx, datasetID, baseQuery, cursor, page); err != nil {
		return nil, err
	}
	page.fetchPage = func(ctx context.Context, nextCursor int) (*CompactDatasetRowsPage, error) {
		return c.fetchCompactDatasetRowsPage(ctx, datasetID, baseQuery, nextCursor)
	}
	return page, nil
}

func (c *Client) requestDatasetRows(ctx context.Context, datasetID string, baseQuery url.Values, cursor int, page any) error {
	query := url.Values{}
	for key, values := range baseQuery {
		query[key] = values
	}
	if cursor > 0 {
		query.Set("cursor", strconv.Itoa(cursor))
	}
	return c.request(ctx, http.MethodGet, "/datasets/"+url.PathEscape(datasetID)+"/rows", query, nil, page)
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

// ToolsService reads the tool catalog.
type ToolsService struct {
	client *Client
}

// List returns every tool the row agent can use with its price, whether this
// account may use it, its inputs, and the budget defaults. (operationId: listTools)
func (s *ToolsService) List(ctx context.Context) (*ToolCatalog, error) {
	catalog := &ToolCatalog{}
	if err := s.client.request(ctx, http.MethodGet, "/tools", nil, nil, catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}

// ConfiguredDatasetsService runs reusable, pre-configured dataset templates
// (cds_...).
type ConfiguredDatasetsService struct {
	client *Client
}

// Build runs a configured dataset template. (operationId: buildConfiguredDataset)
func (s *ConfiguredDatasetsService) Build(ctx context.Context, id string, params BuildConfiguredDatasetParams) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodPost, "/configured_datasets/"+url.PathEscape(id)+"/build", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// ExtractionsService manages reusable site scrape/extract recipes and their
// runs.
type ExtractionsService struct {
	client *Client
}

// Create creates an extraction and starts its agent discovery. (operationId: createExtraction)
func (s *ExtractionsService) Create(ctx context.Context, params CreateExtractionParams) (*Extraction, error) {
	extraction := &Extraction{}
	if err := s.client.request(ctx, http.MethodPost, "/extractions", nil, params, extraction); err != nil {
		return nil, err
	}
	return extraction, nil
}

// Get returns the extraction's status and definition. (operationId: getExtraction)
func (s *ExtractionsService) Get(ctx context.Context, id string) (*Extraction, error) {
	extraction := &Extraction{}
	if err := s.client.request(ctx, http.MethodGet, "/extractions/"+url.PathEscape(id), nil, nil, extraction); err != nil {
		return nil, err
	}
	return extraction, nil
}

// Run executes a ready extraction. (operationId: runExtraction)
func (s *ExtractionsService) Run(ctx context.Context, id string, params RunExtractionParams) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodPost, "/extractions/"+url.PathEscape(id)+"/runs", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// MonitorsService manages monitors: a dataset's output columns filled again on
// a schedule, with diffs and webhooks.
type MonitorsService struct {
	client *Client
}

// Create creates a monitor that fills a dataset again on a schedule (DatasetID,
// optionally Columns). (operationId: createMonitor)
func (s *MonitorsService) Create(ctx context.Context, params CreateMonitorParams) (*Monitor, error) {
	monitor := &Monitor{}
	if err := s.client.request(ctx, http.MethodPost, "/monitors", nil, params, monitor); err != nil {
		return nil, err
	}
	return monitor, nil
}

// List lists the account's monitors, newest first. (operationId: listMonitors)
func (s *MonitorsService) List(ctx context.Context) ([]Monitor, error) {
	var response struct {
		Monitors []Monitor `json:"monitors"`
	}
	if err := s.client.request(ctx, http.MethodGet, "/monitors", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Monitors, nil
}

// Get returns the monitor's schedule, webhook, and next run time. (operationId: getMonitor)
func (s *MonitorsService) Get(ctx context.Context, id string) (*Monitor, error) {
	monitor := &Monitor{}
	if err := s.client.request(ctx, http.MethodGet, "/monitors/"+url.PathEscape(id), nil, nil, monitor); err != nil {
		return nil, err
	}
	return monitor, nil
}

// Update pauses, resumes, repoints the webhook, or changes the Columns a tick
// fills. (operationId: updateMonitor)
func (s *MonitorsService) Update(ctx context.Context, id string, params UpdateMonitorParams) (*Monitor, error) {
	monitor := &Monitor{}
	if err := s.client.request(ctx, http.MethodPatch, "/monitors/"+url.PathEscape(id), nil, params, monitor); err != nil {
		return nil, err
	}
	return monitor, nil
}

// Runs returns the monitor's run history, newest first. (operationId: listMonitorRuns)
func (s *MonitorsService) Runs(ctx context.Context, id string, params *ListMonitorRunsParams) (*RunsPage, error) {
	query := url.Values{}
	page := 1
	if params != nil {
		if params.Status != "" {
			query.Set("status", params.Status)
		}
		if params.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(params.PerPage))
		}
		if params.Page > 0 {
			page = params.Page
		}
	}
	return s.client.fetchRunsPage(ctx, "/monitors/"+url.PathEscape(id)+"/runs", query, page)
}
