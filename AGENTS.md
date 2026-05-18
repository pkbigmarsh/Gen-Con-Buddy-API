## Setup

After cloning, install the git hooks:

```bash
bash scripts/install-hooks.sh
```

Requires `golangci-lint` and `goimports` on your PATH:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
```

## Build & test

```bash
go build -ldflags="-w -s" -o ./bin/gcb .
go test ./...
```

## Local dev prerequisites

OpenSearch must be running. Quickstart:

```bash
docker run -d --name gcb-os-docker \
    -p 9200:9200 -p 9600:9600 \
    -e "discovery.type=single-node" \
    -e "OPENSEARCH_INITIAL_ADMIN_PASSWORD={password}" \
    opensearchproject/opensearch:latest
```

`data.csv` (Gen Con event catalog) and `boardgames_ranks.csv` (BGG rankings) are not committed — they are large source files. Do not attempt `data init` or `data update` without them present. `data.csv` is available from `https://www.gencon.com/downloads/events.xlsx`.

## Code style

The pre-commit hook runs `golangci-lint` automatically (see `.golangci.yml`). To fix formatting manually:

```bash
gofmt -w .
goimports -w -local github.com/gencon_buddy_api .
```

- Always add a newline after a closing brace — the only exception is when it is immediately followed by another closing brace.
- Multi-item slice literals: one item per line (vertical, trailing comma on last item).

## Go patterns

- **Table-driven tests.** Use a `tests := []struct{...}` table; avoid repetitive individual test functions.
- **Pre-allocate slices when length is known.** Use `make([]T, len(source))` and assign by index (`s[i] = ...`). Only use `append` when the final length is not known in advance.
- **No anonymous structs.** If a struct is used more than once or belongs to an exported API, declare and export it — don't inline `struct{...}` at the call site.

## Error handling

- **Return after every error response.** After calling `WriteHeader` or writing an error body, always `return`. Never fall through to a success path.
- **Invalid or empty params → 400.** Don't silently `break` or `continue` on bad input that should be rejected. Return a `400 Bad Request` with a descriptive message.
- **Check OpenSearch bulk response bodies.** `resp.IsError()` only checks the HTTP status. Always inspect the response body for `"errors": true` to catch per-document failures.

## API design

- **One generic endpoint, not one per field.** Use a path parameter (e.g. `/api/events/facets/{field}`) rather than a separate endpoint per field.
- **Sortable and faceted text fields need a `.keyword` subfield** in `event_index_template.json`. Facet and sort queries must target the `.keyword` subfield; full-text `match` queries target the base field.
- **Don't switch a field from `match` to `term` without a product reason.** Default to full-text `match`. Only move to exact `term`/keyword search when there is an explicit reason.
- **After editing `event_index_template.json`, re-run `data init`** to apply the updated schema. The template is not applied retroactively to an existing index.

## Data handling

- **Trim whitespace** from values during CSV ingestion and from search query param values before building queries.
- **The Gen Con event file is Windows-1252 encoded.** Always decode it through a Windows-1252 → UTF-8 decoder (`golang.org/x/text/encoding/charmap`) before reading. Never read it as raw UTF-8.
- **Embed the IANA timezone database.** Import `_ "time/tzdata"` so timezone lookups work in scratch-based container images.

## Naming

- **`Parse` prefix for parsing functions.** Functions that parse a string into a typed value should be named `ParseX`, not `NewXFromString`.
- **Generic names for shared types.** Name response and DTO types by their shape, not their first use case (e.g. `KeywordFacet`, not `GameSystemFacet`).
- **Constants near their flags.** Declare a constant in the same file as the CLI flag it corresponds to.

## Test data

- **No real PII in fixtures.** Do not commit real email addresses or real names in files under `testdata/`.

## Misc

- **No AI artifact references in code comments.** Don't reference local Claude conversation docs or design files in comments. Either summarize the decision inline or open a GitHub issue and reference the issue number.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI)

### Domain docs

Single-context repo — one `CONTEXT.md` + `docs/adr/` at the root. See `docs/agents/domain.md`.

### Related repos

- **Frontend:** `github.com/myasonik/Gen-Con-Buddy` — React 19 / TypeScript SPA. The primary consumer of this API.
