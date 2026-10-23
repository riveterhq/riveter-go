package riveter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func datasetFixtureJSON(overrides map[string]any) []byte {
	fixture := map[string]any{
		"id":                 "ds_1",
		"name":               "Wineries",
		"input_columns":      []string{"Winery URL"},
		"index_column":       "Winery URL",
		"enrichment_id":      "enr_1",
		"output_columns":     []string{"Owner"},
		"row_count":          2,
		"rows_expired":       false,
		"import_in_progress": false,
		"runs":               []any{},
		"created_at":         "2026-01-15T12:00:00Z",
		"updated_at":         "2026-01-15T12:00:00Z",
	}
	for key, value := range overrides {
		fixture[key] = value
	}
	payload, err := json.Marshal(fixture)
	if err != nil {
		panic(err)
	}
	return payload
}

func datasetRowFixture(id string, position int) map[string]any {
	return map[string]any{
		"id":       id,
		"position": position,
		"input":    map[string]any{"Winery URL": "ridgewine.com"},
		"cells": map[string]any{
			"Owner": map[string]any{"value": "Otsuka", "state": "ok", "sources": []string{"https://ridgewine.com/about"}},
		},
		"credits":          1.5,
		"not_found_reason": nil,
		"created_at":       "2026-01-15T12:00:00Z",
		"updated_at":       "2026-01-15T12:00:00Z",
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	w.WriteHeader(status)
	w.Write(payload)
}

func decodeJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestDatasetsList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/datasets" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"datasets":[{"id":"ds_1","name":"Wineries","input_column_count":1,"output_column_count":1,"row_count":2,"rows_expired":false,"newest_run":{"id":"run_1","type":"enrichment","status":"success"}}]}`)
	})

	datasets, err := client.Datasets.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(datasets) != 1 || datasets[0].ID != "ds_1" || datasets[0].NewestRun == nil || datasets[0].NewestRun.Status != "success" {
		t.Fatalf("datasets = %+v", datasets)
	}
}

func TestDatasetsCreateWithRowsAndRun(t *testing.T) {
	var capturedBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/datasets" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		capturedBody = decodeJSONBody(t, r)
		var withRun map[string]any
		json.Unmarshal(datasetFixtureJSON(nil), &withRun)
		var run map[string]any
		json.Unmarshal(runFixtureJSON(map[string]any{"type": "dataset_build"}), &run)
		withRun["run"] = run
		writeJSON(t, w, http.StatusCreated, withRun)
	})

	dataset, err := client.Datasets.Create(context.Background(), CreateDatasetParams{
		Name:         "Wineries",
		InputColumns: []string{"Winery URL"},
		Rows:         []map[string]string{{"Winery URL": "ridgewine.com"}},
		EnrichmentID: "enr_1",
		AutoFill:     true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if capturedBody["name"] != "Wineries" || capturedBody["auto_fill"] != true {
		t.Fatalf("body = %v", capturedBody)
	}
	if _, hasCount := capturedBody["count"]; hasCount {
		t.Fatalf("zero Count must be omitted, body = %v", capturedBody)
	}
	if dataset.ID != "ds_1" || dataset.Run == nil || dataset.Run.Type != "dataset_build" {
		t.Fatalf("dataset = %+v", dataset)
	}
}

func TestDatasetsCreateEstimate(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeJSONBody(t, r)
		if body["estimate_cost"] != true || body["count"] != float64(50) {
			t.Fatalf("body = %v", body)
		}
		fmt.Fprint(w, `{"estimate_cost":true,"credit_detail":{"estimate":{"minimum":10,"maximum":75,"expected":40},"credits_remaining":500}}`)
	})

	estimate, err := client.Datasets.Create(context.Background(), CreateDatasetParams{
		Prompt:       "Wineries in Napa",
		Count:        50,
		EstimateCost: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !estimate.EstimateCost || estimate.CreditDetail == nil || estimate.CreditDetail.Estimate.Maximum != 75 {
		t.Fatalf("estimate = %+v", estimate)
	}
}

func TestDatasetsGetUpdatePromote(t *testing.T) {
	var requests []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPatch {
			body := decodeJSONBody(t, r)
			output := body["output"].(map[string]any)
			owner := output["Owner"].(map[string]any)
			if owner["prompt"] != "The winery's owner" {
				t.Fatalf("output op = %v", owner)
			}
			if _, hasTool := owner["tool"]; hasTool {
				t.Fatalf("an empty Tool must be omitted, op = %v", owner)
			}
			if body["budget_locked"] != true {
				t.Fatalf("body = %v", body)
			}
		}
		if r.URL.Path == "/v1/datasets/ds_1/columns/promote" {
			body := decodeJSONBody(t, r)
			if body["column"] != "Owner" {
				t.Fatalf("body = %v", body)
			}
		}
		w.Write(datasetFixtureJSON(nil))
	})

	ctx := context.Background()
	if _, err := client.Datasets.Get(ctx, "ds_1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	locked := true
	if _, err := client.Datasets.Update(ctx, "ds_1", UpdateDatasetParams{
		Output:       map[string]OutputColumnOperation{"Owner": {Prompt: "The winery's owner"}},
		BudgetLocked: &locked,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, err := client.Datasets.PromoteColumn(ctx, "ds_1", "Owner"); err != nil {
		t.Fatalf("PromoteColumn: %v", err)
	}
	want := []string{"GET /v1/datasets/ds_1", "PATCH /v1/datasets/ds_1", "POST /v1/datasets/ds_1/columns/promote"}
	if fmt.Sprint(requests) != fmt.Sprint(want) {
		t.Fatalf("requests = %v", requests)
	}
}

func TestOutputColumnOperationToolPin(t *testing.T) {
	pinned, err := json.Marshal(OutputColumnOperation{Tool: "basic_web_scrape", Prompt: "Scrape the homepage"})
	if err != nil {
		t.Fatalf("marshal pinned: %v", err)
	}
	if string(pinned) != `{"prompt":"Scrape the homepage","tool":"basic_web_scrape"}` {
		t.Fatalf("pinned = %s", pinned)
	}

	unpinned, err := json.Marshal(OutputColumnOperation{UnpinTool: true})
	if err != nil {
		t.Fatalf("marshal unpinned: %v", err)
	}
	if string(unpinned) != `{"tool":null}` {
		t.Fatalf("unpinned = %s", unpinned)
	}

	renamed, err := json.Marshal(OutputColumnOperation{Rename: "Proprietor"})
	if err != nil {
		t.Fatalf("marshal renamed: %v", err)
	}
	if string(renamed) != `{"rename":"Proprietor"}` {
		t.Fatalf("renamed = %s", renamed)
	}
}

func TestDatasetsDelete204(t *testing.T) {
	var requests []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	if err := client.Datasets.Delete(ctx, "ds_1"); err != nil {
		t.Fatalf("Datasets.Delete: %v", err)
	}
	if err := client.Enrichments.Delete(ctx, "enr_1"); err != nil {
		t.Fatalf("Enrichments.Delete: %v", err)
	}
	want := []string{"DELETE /v1/datasets/ds_1", "DELETE /v1/enrichments/enr_1"}
	if fmt.Sprint(requests) != fmt.Sprint(want) {
		t.Fatalf("requests = %v", requests)
	}
}

func TestDatasetsListRowsPages(t *testing.T) {
	var cursors []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/datasets/ds_1/rows" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		if r.URL.Query().Get("limit") != "1" {
			t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("cursor") == "" {
			writeJSON(t, w, http.StatusOK, map[string]any{"rows": []any{datasetRowFixture("row_1", 1)}, "next_cursor": 1, "row_count": 2})
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"rows": []any{datasetRowFixture("row_2", 2)}, "next_cursor": nil, "row_count": 2})
	})

	ctx := context.Background()
	first, err := client.Datasets.ListRows(ctx, "ds_1", &ListDatasetRowsParams{Limit: 1})
	if err != nil {
		t.Fatalf("ListRows: %v", err)
	}
	if !first.HasNextPage() || first.Rows[0].Cells["Owner"].Value != "Otsuka" || first.Rows[0].Credits != 1.5 {
		t.Fatalf("first = %+v", first)
	}
	second, err := first.NextPage(ctx)
	if err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if second.HasNextPage() || second.Rows[0].ID != "row_2" {
		t.Fatalf("second = %+v", second)
	}
	if _, err := second.NextPage(ctx); !errors.Is(err, ErrNoNextPage) {
		t.Fatalf("expected ErrNoNextPage, got %v", err)
	}
	if fmt.Sprint(cursors) != fmt.Sprint([]string{"", "1"}) {
		t.Fatalf("cursors = %v", cursors)
	}
}

func TestDatasetsRowWrites(t *testing.T) {
	var requests []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodPost:
			body := decodeJSONBody(t, r)
			rows := body["rows"].([]any)
			if len(rows) != 1 {
				t.Fatalf("body = %v", body)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{"rows": []any{datasetRowFixture("row_3", 3)}, "skipped_duplicates": 0, "row_count": 3})
		case r.Method == http.MethodPatch:
			body := decodeJSONBody(t, r)
			cells := body["cells"].(map[string]any)
			if cells["Owner"] != "Someone" {
				t.Fatalf("body = %v", body)
			}
			if _, hasInput := body["input"]; hasInput {
				t.Fatalf("nil Input must be omitted, body = %v", body)
			}
			writeJSON(t, w, http.StatusOK, datasetRowFixture("row_3", 3))
		case r.Method == http.MethodDelete:
			writeJSON(t, w, http.StatusOK, map[string]any{"deleted": 2, "row_count": 1})
		}
	})

	ctx := context.Background()
	added, err := client.Datasets.AddRows(ctx, "ds_1", []map[string]string{{"Winery URL": "ridgewine.com"}})
	if err != nil {
		t.Fatalf("AddRows: %v", err)
	}
	if added.RowCount != 3 || added.Rows[0].ID != "row_3" {
		t.Fatalf("added = %+v", added)
	}
	if _, err := client.Datasets.UpdateRow(ctx, "ds_1", "3", UpdateDatasetRowParams{Cells: map[string]any{"Owner": "Someone"}}); err != nil {
		t.Fatalf("UpdateRow: %v", err)
	}
	deleted, err := client.Datasets.DeleteRows(ctx, "ds_1", DeleteDatasetRowsParams{IDs: []string{"row_1"}, Positions: []int{2}})
	if err != nil {
		t.Fatalf("DeleteRows: %v", err)
	}
	if deleted.Deleted != 2 {
		t.Fatalf("deleted = %+v", deleted)
	}
	want := []string{
		"POST /v1/datasets/ds_1/rows",
		"PATCH /v1/datasets/ds_1/rows/3",
		"DELETE /v1/datasets/ds_1/rows?ids=row_1&positions=2",
	}
	if fmt.Sprint(requests) != fmt.Sprint(want) {
		t.Fatalf("requests = %v", requests)
	}
}

func TestDatasetsImportRowsSendsCSV(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/datasets/ds_1/rows/import" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "text/csv" {
			t.Fatalf("Content-Type = %q", got)
		}
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != "Winery URL\nridgewine.com\n" {
			t.Fatalf("body = %q", raw)
		}
		writeJSON(t, w, http.StatusAccepted, map[string]any{"import": map[string]any{"status": "importing", "rows": 1}, "row_count": 2})
	})

	result, err := client.Datasets.ImportRows(context.Background(), "ds_1", "Winery URL\nridgewine.com\n")
	if err != nil {
		t.Fatalf("ImportRows: %v", err)
	}
	if result.Import.Status != "importing" || result.Import.Rows != 1 || result.RowCount != 2 {
		t.Fatalf("result = %+v", result)
	}
}

func TestDatasetsBuildRows(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/datasets/ds_1/rows/build" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeJSONBody(t, r)
		if body["count"] != float64(20) || body["auto_fill"] != true {
			t.Fatalf("body = %v", body)
		}
		var withRun map[string]any
		json.Unmarshal(datasetFixtureJSON(nil), &withRun)
		var run map[string]any
		json.Unmarshal(runFixtureJSON(map[string]any{"type": "dataset_build"}), &run)
		withRun["run"] = run
		writeJSON(t, w, http.StatusCreated, withRun)
	})

	dataset, err := client.Datasets.BuildRows(context.Background(), "ds_1", BuildDatasetRowsParams{Count: 20, AutoFill: true})
	if err != nil {
		t.Fatalf("BuildRows: %v", err)
	}
	if dataset.Run == nil || dataset.Run.ID != "run_abc" {
		t.Fatalf("dataset = %+v", dataset)
	}
}

func TestDatasetsExportFormats(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/datasets/ds_1/export" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("output_format") {
		case "":
			w.Header().Set("Content-Type", "text/csv")
			fmt.Fprint(w, "Winery URL,Owner\nridgewine.com,Otsuka\n")
		case "xlsx":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Write([]byte{0x50, 0x4b, 0x03, 0x04})
		default:
			t.Fatalf("output_format = %q", r.URL.Query().Get("output_format"))
		}
	})

	ctx := context.Background()
	csv, err := client.Datasets.Export(ctx, "ds_1", "")
	if err != nil {
		t.Fatalf("Export csv: %v", err)
	}
	if string(csv) != "Winery URL,Owner\nridgewine.com,Otsuka\n" {
		t.Fatalf("csv = %q", csv)
	}
	xlsx, err := client.Datasets.Export(ctx, "ds_1", "xlsx")
	if err != nil {
		t.Fatalf("Export xlsx: %v", err)
	}
	if !bytes.Equal(xlsx, []byte{0x50, 0x4b, 0x03, 0x04}) {
		t.Fatalf("xlsx = %v", xlsx)
	}
}

func TestEnrichmentsCreateRecipeAndToolsList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/enrichments":
			body := decodeJSONBody(t, r)
			output := body["output"].([]any)
			if output[0] != "Owner" || output[1].(map[string]any)["tool"] != "basic_web_scrape" {
				t.Fatalf("body = %v", body)
			}
			fmt.Fprint(w, `{"id":"enr_1","name":"Wineries","prompt":"Research the winery","input_columns":["Winery URL"],"output":[{"name":"Owner"},{"name":"Homepage text","tool":"basic_web_scrape"}],"tools":["web_search","basic_web_scrape"],"budget_credits_per_row":3,"budget_locked":false,"version":1,"dataset_id":"ds_1","app_url":"https://app.riveterhq.com/v2/datasets/ds_1"}`)
		case "/v1/tools":
			fmt.Fprint(w, `{"tools":[{"name":"web_search","credits_per_call":0.1,"default":true,"requires_opt_in":false,"enabled":true,"requires_api_key":false,"api_key_linked":null,"companions":[],"inputs":[{"name":"query","type":"string","required":true}],"input_groups":null,"returns":"Search results"}],"budget":{"default_credits_per_output_column":1.5,"min_credits_per_row":0.5,"max_credits_per_row":50}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	ctx := context.Background()
	enrichment, err := client.Enrichments.Create(ctx, CreateEnrichmentParams{
		Name:         "Wineries",
		Prompt:       "Research the winery",
		InputColumns: []string{"Winery URL"},
		Output:       []any{"Owner", OutputColumnSpec{Name: "Homepage text", Tool: "basic_web_scrape"}},
	})
	if err != nil {
		t.Fatalf("Enrichments.Create: %v", err)
	}
	columns, err := enrichment.OutputColumns()
	if err != nil {
		t.Fatalf("OutputColumns: %v", err)
	}
	if enrichment.DatasetID != "ds_1" || len(columns) != 2 || columns[1].Tool != "basic_web_scrape" || enrichment.BudgetCreditsPerRow != 3 {
		t.Fatalf("enrichment = %+v, columns = %+v", enrichment, columns)
	}
	legacy, err := enrichment.LegacyOutput()
	if err != nil || legacy != nil {
		t.Fatalf("LegacyOutput on a recipe = %v, %v", legacy, err)
	}

	catalog, err := client.Tools.List(ctx)
	if err != nil {
		t.Fatalf("Tools.List: %v", err)
	}
	if len(catalog.Tools) != 1 || catalog.Tools[0].Name != "web_search" || catalog.Budget.DefaultCreditsPerOutputColumn != 1.5 {
		t.Fatalf("catalog = %+v", catalog)
	}
	if catalog.Tools[0].Inputs[0].Type == nil || *catalog.Tools[0].Inputs[0].Type != "string" {
		t.Fatalf("inputs = %+v", catalog.Tools[0].Inputs)
	}
}

func TestLegacyEnrichmentOutput(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"enr_legacy","name":"Old","status":"ready","app_url":"https://app.riveterhq.com/enrichments/x","input":{"Company":{}},"output":{"CEO":{"prompt":"The CEO","format":"text"}}}`)
	})

	enrichment, err := client.Enrichments.Get(context.Background(), "enr_legacy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	legacy, err := enrichment.LegacyOutput()
	if err != nil {
		t.Fatalf("LegacyOutput: %v", err)
	}
	if legacy["CEO"].Prompt != "The CEO" || enrichment.Status != "ready" {
		t.Fatalf("enrichment = %+v, legacy = %+v", enrichment, legacy)
	}
	columns, err := enrichment.OutputColumns()
	if err != nil || columns != nil {
		t.Fatalf("OutputColumns on a legacy enrichment = %v, %v", columns, err)
	}
}

func TestUpdateEnrichmentWritesOneOutputShape(t *testing.T) {
	var bodies []map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, decodeJSONBody(t, r))
		fmt.Fprint(w, `{"id":"enr_1","name":"Wineries"}`)
	})

	ctx := context.Background()
	if _, err := client.Enrichments.Update(ctx, "enr_1", UpdateEnrichmentParams{
		Output: map[string]OutputColumnOperation{"Owner": {Rename: "Proprietor"}},
	}); err != nil {
		t.Fatalf("Update (recipe): %v", err)
	}
	if _, err := client.Enrichments.Update(ctx, "enr_1", UpdateEnrichmentParams{
		LegacyOutput: map[string]OutputColumnConfig{"CEO": {Prompt: "The CEO"}},
		ColumnOrder:  []string{"CEO"},
	}); err != nil {
		t.Fatalf("Update (legacy): %v", err)
	}
	if _, err := client.Enrichments.Update(ctx, "enr_1", UpdateEnrichmentParams{Name: "Renamed"}); err != nil {
		t.Fatalf("Update (name): %v", err)
	}

	recipeOutput := bodies[0]["output"].(map[string]any)["Owner"].(map[string]any)
	if recipeOutput["rename"] != "Proprietor" {
		t.Fatalf("recipe body = %v", bodies[0])
	}
	legacyOutput := bodies[1]["output"].(map[string]any)["CEO"].(map[string]any)
	if legacyOutput["prompt"] != "The CEO" || bodies[1]["column_order"] == nil {
		t.Fatalf("legacy body = %v", bodies[1])
	}
	if _, hasOutput := bodies[2]["output"]; hasOutput || bodies[2]["name"] != "Renamed" {
		t.Fatalf("name-only body must omit output, body = %v", bodies[2])
	}
}

func TestFeatureNotEnabledError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":{"type":"feature_not_enabled","message":"Your account has not migrated to Datasets yet."}}`)
	})

	_, err := client.Datasets.List(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 || apiErr.Type != ErrorTypeFeatureNotEnabled {
		t.Fatalf("apiErr = %+v", apiErr)
	}
}
