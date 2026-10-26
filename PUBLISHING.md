# Publishing riveter-go (Go module)

Go modules resolve from the repository path, and this monorepo is private, so the
module lives at `github.com/riveterhq/riveter-go` — a public mirror repo that contains
only this directory. There is no registry upload; "publishing" is pushing the mirror
and tagging.

`.github/workflows/publish-clients.yml` (`go-mirror` job) does both on every push to
`main` that touches this directory: it runs `gofmt` / `go vet` / `go test`, splits the
directory with `git subtree split`, force-pushes the split over the mirror's `main`
with the `MIRROR_PUSH_TOKEN` secret, and, when the mirror has no `v<Version>` tag for
the `Version` const in `riveter.go`, tags the split commit and pushes the tag (that
tag is the publish), then asks `proxy.golang.org` for it so the proxy caches it.
Never push from an agent (see the repo git rules); the workflow and the user do.

## Release checklist

1. Bump the `Version` const in `riveter.go` (sent in the User-Agent header). The
   four SDKs share one version number.
2. Run the checks: `cd sdk/go && gofmt -l . && go vet ./... && go build ./...` (no unit tests).
3. Commit the changes to the monorepo as usual and merge to `main`. The workflow
   pushes the mirror and tags `vX.Y.Z`; `published-versions.yml` fails the push
   when the source changed without a bump.
4. Verify (also warms the module proxy if the workflow's warm-up did not):

   ```bash
   GOPROXY=proxy.golang.org go list -m github.com/riveterhq/riveter-go@vX.Y.Z
   ```

## By hand

The same steps, from the monorepo root on `main`, when the workflow cannot run:

```bash
git subtree split --prefix=sdk/go -b riveter-go-sync
git push git@github.com:riveterhq/riveter-go.git riveter-go-sync:main
git tag vX.Y.Z riveter-go-sync
git push git@github.com:riveterhq/riveter-go.git vX.Y.Z
git tag -d vX.Y.Z && git branch -D riveter-go-sync
```

## One-time setup (done)

The public GitHub repo `riveterhq/riveter-go` exists (empty at creation — no README,
no license; this directory provides both). `v0.1.0` was tagged by hand on
2026-08-18; later versions come from the workflow.
