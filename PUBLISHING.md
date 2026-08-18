# Publishing riveter-go (Go module)

User-run only — never push from an agent (see the repo git rules).

Go modules resolve from the repository path, and this monorepo is private, so the
module lives at `github.com/riveterhq/riveter-go` — a public mirror repo that contains
only this directory. There is no registry upload; "publishing" is pushing the mirror
and tagging.

## One-time setup

1. Create the public GitHub repo `riveterhq/riveter-go` (empty — no README, no
   license; this directory provides both).

## Release checklist

1. Bump the `Version` const in `riveter.go` (sent in the User-Agent header).
2. Run the tests: `cd sdk/go && gofmt -l . && go vet ./... && go test ./...`.
3. Commit the changes to the monorepo as usual (the mirror is derived from the
   monorepo's `main`).
4. From the monorepo root, split and push the mirror:

   ```bash
   git subtree split --prefix=sdk/go -b riveter-go-sync
   git push git@github.com:riveterhq/riveter-go.git riveter-go-sync:main
   ```

5. Tag the release on the mirror (tag the split commit, then clean up):

   ```bash
   git tag vX.Y.Z riveter-go-sync
   git push git@github.com:riveterhq/riveter-go.git vX.Y.Z
   git tag -d vX.Y.Z && git branch -D riveter-go-sync
   ```

6. Verify (also warms the module proxy):

   ```bash
   GOPROXY=proxy.golang.org go list -m github.com/riveterhq/riveter-go@vX.Y.Z
   ```
