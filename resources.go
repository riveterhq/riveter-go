package riveter

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// EnrichmentsService manages saved enrichment configurations.
type EnrichmentsService struct {
	client *Client
}

// List lists enrichments with their output column configuration. (operationId: listEnrichments)
func (s *EnrichmentsService) List(ctx context.Context) ([]EnrichmentSummary, error) {
	var response struct {
		Enrichments []EnrichmentSummary `json:"enrichments"`
	}
	if err := s.client.request(ctx, http.MethodGet, "/enrichments", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Enrichments, nil
}

// Create creates an enrichment (no run) from a completed dataset build. (operationId: createEnrichment)
func (s *EnrichmentsService) Create(ctx context.Context, datasetID string) (*Enrichment, error) {
	body := map[string]string{"dataset_id": datasetID}
	enrichment := &Enrichment{}
	if err := s.client.request(ctx, http.MethodPost, "/enrichments", nil, body, enrichment); err != nil {
		return nil, err
	}
	return enrichment, nil
}

// Get returns the enrichment's input columns and output column configuration. (operationId: getEnrichment)
func (s *EnrichmentsService) Get(ctx context.Context, id string) (*Enrichment, error) {
	enrichment := &Enrichment{}
	if err := s.client.request(ctx, http.MethodGet, "/enrichments/"+url.PathEscape(id), nil, nil, enrichment); err != nil {
		return nil, err
	}
	return enrichment, nil
}

// Update adds, updates, renames, deletes, or reorders output columns. It
// returns the raw response, which lists the applied changes. (operationId: updateEnrichment)
func (s *EnrichmentsService) Update(ctx context.Context, id string, params UpdateEnrichmentParams) (map[string]any, error) {
	response := map[string]any{}
	if err := s.client.request(ctx, http.MethodPatch, "/enrichments/"+url.PathEscape(id), nil, params, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// BuildDataset builds a dataset shaped for this enrichment. (operationId: buildDatasetForEnrichment)
func (s *EnrichmentsService) BuildDataset(ctx context.Context, id string, params BuildDatasetForEnrichmentParams) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodPost, "/enrichments/"+url.PathEscape(id)+"/datasets", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// DatasetsService builds datasets — generate rows from prompts or structured
// specs.
type DatasetsService struct {
	client *Client
}

// Build builds a dataset from a prompt, a structured spec, or both. (operationId: buildDataset)
func (s *DatasetsService) Build(ctx context.Context, params BuildDatasetParams) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// Extend generates new deduplicated rows for a completed dataset build. (operationId: extendDataset)
func (s *DatasetsService) Extend(ctx context.Context, id string, params ExtendDatasetParams) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodPost, "/datasets/"+url.PathEscape(id)+"/extend", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
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

// MonitorsService manages scheduled enrichment runs with webhooks.
type MonitorsService struct {
	client *Client
}

// Create creates a monitor that re-runs an enrichment on a schedule. (operationId: createMonitor)
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

// Update pauses, resumes, or repoints a monitor. (operationId: updateMonitor)
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
