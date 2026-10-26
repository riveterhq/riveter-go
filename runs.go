package riveter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// RunsService is the uniform lifecycle for every async operation — status,
// results, stop.
type RunsService struct {
	client *Client
}

// Get returns the status and progress of a run. (operationId: getRun)
func (s *RunsService) Get(ctx context.Context, id string) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodGet, "/runs/"+url.PathEscape(id), nil, nil, run); err != nil {
		return nil, err
	}
	return run, nil
}

// ResultOptions configures RunsService.Result.
type ResultOptions struct {
	// Wait long-polls server-side for up to 50 seconds.
	Wait int
}

// Result returns the run plus its Output (nil until finished). (operationId: getRunResult)
func (s *RunsService) Result(ctx context.Context, id string, opts *ResultOptions) (*Run, error) {
	query := url.Values{}
	if opts != nil && opts.Wait > 0 {
		query.Set("wait", strconv.Itoa(opts.Wait))
	}
	run := &Run{}
	if err := s.client.request(ctx, http.MethodGet, "/runs/"+url.PathEscape(id)+"/result", query, nil, run); err != nil {
		return nil, err
	}
	return run, nil
}

// Stop stops a run early. (operationId: stopRun)
func (s *RunsService) Stop(ctx context.Context, id string) (*Run, error) {
	run := &Run{}
	if err := s.client.request(ctx, http.MethodPost, "/runs/"+url.PathEscape(id)+"/stop", nil, nil, run); err != nil {
		return nil, err
	}
	return run, nil
}

// List lists the account's runs, newest first. (operationId: listRuns)
func (s *RunsService) List(ctx context.Context, params *ListRunsParams) (*RunsPage, error) {
	query := url.Values{}
	page := 1
	if params != nil {
		if params.Type != "" {
			query.Set("type", params.Type)
		}
		if params.Status != "" {
			query.Set("status", params.Status)
		}
		if params.EnrichmentID != "" {
			query.Set("enrichment_id", params.EnrichmentID)
		}
		if params.DatasetID != "" {
			query.Set("dataset_id", params.DatasetID)
		}
		if params.MonitorID != "" {
			query.Set("monitor_id", params.MonitorID)
		}
		if params.CreatedAfter != "" {
			query.Set("created_after", params.CreatedAfter)
		}
		if params.CreatedBefore != "" {
			query.Set("created_before", params.CreatedBefore)
		}
		if params.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(params.PerPage))
		}
		if params.Page > 0 {
			page = params.Page
		}
	}
	return s.client.fetchRunsPage(ctx, "/runs", query, page)
}

// Summary returns all-time run counts by status. (operationId: runsSummary)
func (s *RunsService) Summary(ctx context.Context) (*RunsSummary, error) {
	summary := &RunsSummary{}
	if err := s.client.request(ctx, http.MethodGet, "/runs/summary", nil, nil, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

// WaitOptions configures RunsService.WaitForResult.
type WaitOptions struct {
	// Timeout is the overall budget (default 10 minutes).
	Timeout time.Duration
	// PollWaitSeconds is the server-side long-poll per request (default 50).
	PollWaitSeconds int
}

// WaitForResult polls Result with 50s long-polls until the run reaches
// success or stopped; returns a *TimeoutError past the budget.
func (s *RunsService) WaitForResult(ctx context.Context, id string, opts *WaitOptions) (*Run, error) {
	timeout := 10 * time.Minute
	pollWait := 50
	if opts != nil {
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
		if opts.PollWaitSeconds > 0 {
			pollWait = opts.PollWaitSeconds
		}
	}
	deadline := time.Now().Add(timeout)
	for {
		run, err := s.Result(ctx, id, &ResultOptions{Wait: pollWait})
		if err != nil {
			return nil, err
		}
		if run.Finished() {
			return run, nil
		}
		if time.Now().After(deadline) {
			return nil, &TimeoutError{Message: fmt.Sprintf("run %s did not finish within %s", id, timeout)}
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
}

// ErrNoNextPage is returned by RunsPage.NextPage on the last page.
var ErrNoNextPage = errors.New("riveter: no next page")

// RunsPage is one page of runs plus the pagination cursor.
type RunsPage struct {
	MonitorID  string        `json:"monitor_id"`
	Runs       []RunListItem `json:"runs"`
	Pagination Pagination    `json:"pagination"`

	fetchPage func(ctx context.Context, page int) (*RunsPage, error)
}

// HasNextPage reports whether another page follows this one.
func (p *RunsPage) HasNextPage() bool {
	return p.Pagination.Page < p.Pagination.TotalPages
}

// NextPage fetches the following page, or ErrNoNextPage on the last one.
func (p *RunsPage) NextPage(ctx context.Context) (*RunsPage, error) {
	if !p.HasNextPage() {
		return nil, ErrNoNextPage
	}
	return p.fetchPage(ctx, p.Pagination.Page+1)
}

func (c *Client) fetchRunsPage(ctx context.Context, path string, baseQuery url.Values, page int) (*RunsPage, error) {
	query := url.Values{}
	for key, values := range baseQuery {
		query[key] = values
	}
	query.Set("page", strconv.Itoa(page))

	runsPage := &RunsPage{}
	if err := c.request(ctx, http.MethodGet, path, query, nil, runsPage); err != nil {
		return nil, err
	}
	runsPage.fetchPage = func(ctx context.Context, nextPage int) (*RunsPage, error) {
		return c.fetchRunsPage(ctx, path, baseQuery, nextPage)
	}
	return runsPage, nil
}
