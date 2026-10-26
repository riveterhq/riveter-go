# riveter-go

Official Go SDK for the [Riveter API](https://docs.riveterhq.com) — enrich data,
build datasets, scrape pages, and run web searches.

Requires Go 1.22+. Zero dependencies (stdlib `net/http`).

## Install

```bash
go get github.com/riveterhq/riveter-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	riveter "github.com/riveterhq/riveter-go"
)

func main() {
	// Reads RIVETER_API_KEY from the environment when WithAPIKey is not used.
	// Get a key at https://app.riveterhq.com/settings/api
	client, err := riveter.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	run, err := client.Enrich(ctx, riveter.EnrichParams{
		Prompt:     "Research each company",
		Attributes: []string{"CEO", "Employee Count"},
		Input:      map[string][]string{"Company": {"Apple", "Google"}},
	})
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Runs.WaitForResult(ctx, run.ID, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(result.Output))
}
```

## The run lifecycle

Every async kickoff (`Enrich`, `Datasets.Build`, `Extractions.Run`, ...) returns a run.

```go
client.Runs.Get(ctx, run.ID)                                        // status + progress
client.Runs.Result(ctx, run.ID, &riveter.ResultOptions{Wait: 50})   // output (long-polls up to 50s)
client.Runs.WaitForResult(ctx, run.ID, nil)                         // poll until finished
client.Runs.Stop(ctx, run.ID)                                       // stop early
```

List runs with pagination:

```go
page, err := client.Runs.List(ctx, &riveter.ListRunsParams{Status: "success"})
for {
	if err != nil {
		log.Fatal(err)
	}
	for _, run := range page.Runs {
		fmt.Println(run.ID)
	}
	if !page.HasNextPage() {
		break
	}
	page, err = page.NextPage(ctx)
}
```

## Datasets

A dataset is a kept table: input rows you add, output columns an enrichment fills.

```go
dataset, err := client.Datasets.Create(ctx, riveter.CreateDatasetParams{
	Name:         "Wineries",
	InputColumns: []string{"Winery URL"},
	Rows:         []map[string]string{{"Winery URL": "ridgewine.com"}, {"Winery URL": "bonnydoonvineyard.com"}},
})
_, err = client.Datasets.Update(ctx, dataset.ID, riveter.UpdateDatasetParams{
	Output: map[string]riveter.OutputColumnOperation{"Owner": {Prompt: "The winery's current owner"}},
})
fill, err := client.Enrich(ctx, riveter.EnrichParams{DatasetID: dataset.ID})
_, err = client.Runs.WaitForResult(ctx, fill.ID, nil)
page, err := client.Datasets.ListRows(ctx, dataset.ID, nil)
for _, row := range page.Rows {
	fmt.Println(*row.Input["Winery URL"], row.Cells["Owner"].Value)
}
csv, err := client.Datasets.Export(ctx, dataset.ID, "csv")
```

`Datasets.Create` with `Prompt` + `Count` generates the rows instead (paid; set
`EstimateCost: true` first and read `CreditDetail`). `Datasets.ImportRows` appends a CSV.

## Surface

- `client.Enrich`, `client.QuickSearch`, `client.SearchAgent`, `client.Scrape`, `client.Account`
- `client.Runs` — `Get`, `Result`, `Stop`, `List`, `Summary`, `WaitForResult`
- `client.Enrichments` — `List`, `Create`, `Get`, `Update`, `Delete`
- `client.Datasets` — `List`, `Create`, `Get`, `Update`, `Delete`, `ListRows`, `AddRows`, `DeleteRows`, `UpdateRow`, `ImportRows`, `BuildRows`, `PromoteColumn`, `Export`
- `client.Tools` — `List`
- `client.ConfiguredDatasets` — `Build`
- `client.Extractions` — `Create`, `Get`, `Run`
- `client.Monitors` — `Create`, `List`, `Get`, `Update`, `Runs`

## Errors and retries

API failures return `*riveter.APIError` with `StatusCode`, `Type` (`not_found`,
`insufficient_credits`, `feature_not_enabled`, ...; the known values are the
`riveter.ErrorType*` constants), `Message`, and optional `Details`; use `errors.As` to
inspect. Network failures return `*riveter.ConnectionError`. 429s are retried
automatically using the `X-RateLimit-Reset` header; 5xx and network failures are retried
for GETs. Configure with `WithMaxRetries` and `WithHTTPClient` (keep its `Timeout` above
50s for `wait` long-polls).

## Options

```go
riveter.NewClient(
	riveter.WithAPIKey("..."),                              // default: env RIVETER_API_KEY
	riveter.WithBaseURL("https://api.riveterhq.com/v1"),    // default: env RIVETER_BASE_URL, then this
	riveter.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	riveter.WithMaxRetries(2),
)
```

## About this repository

This repository is a read-only publishing mirror of `sdk/go` in the main Riveter
monorepo, so that `go get` can resolve the module. Issues and changes happen in the
main repo.
