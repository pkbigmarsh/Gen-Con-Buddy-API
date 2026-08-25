# BGG combos are sourced from the event index, making enrichment a two-pass pipeline

`data bgg` builds its list of `GameSystem` + `RulesEdition` combos by paginating the event index with `search_after` (`generateBggMapping`, `cmd/data/bgg.go`), not by parsing the Gen Con event file. The matched values are then denormalized onto Event documents at ingest — `HydrateBGG` applies them per-Event during parse (`internal/event/hydrator.go`), writing `bggId`, `bggRank`, and `bggAvgRating` onto each document rather than joining against BGG at query time.

Together these make enrichment inherently two-pass: Events must be indexed before `data bgg` can see their combos, and a *later* `data update` must run to attach the results. `data bgg` cannot meaningfully run first.

**This ADR is written after the fact and partly reconstructs its own reasoning.** The decision was not made deliberately in one place. PR #33 built the opposite design — `data match-bgg --gencon <csv> --bgg <csv> --output bgg_mapping.json`, reading combos from the event CSV, with the mapping committed to the repo for reproducibility. That is the design ADR-0001 describes. PR #53 (`5c1c1e8`), whose stated purpose was auto-fetching the BGG ranks dump, silently changed the combo source to the event index and renamed the command to `data bgg`. Neither the PR body nor the commit message mentions either change. The rationale below is inferred from the code and the constraints; the consequences are observed. Treat the reasoning as reconstructed, not recorded.

## Considered Options

**Parse combos from the Gen Con event file** — the original PR #33 design, whose implementation still exists as `bgg.LoadGenConCombos` in `internal/bgg/load.go`, now referenced by nothing but its own tests. It keeps `data bgg` standalone: no OpenSearch dependency, no ordering constraint, one pass, and a mapping static enough to commit. It does not survive production. The scheduled cron streams the events file from Gen Con over HTTP directly into the reader (`setupEventReaderFromDownload`, `cmd/data/update.go`) — no file is ever written to disk, so there is nothing for a file-parsing combo loader to read without downloading the catalog a second time. It also duplicates the parsing and BGG-eligible-type filtering that the event reader already does, and scopes combos to one file rather than to everything actually indexed.

**Join BGG data at query time in the API** — no denormalization, no ingest ordering, and mapping changes would appear instantly with no re-ingest. Rejected on two counts. It puts a lookup on the hot search path and requires the mapping to be available to the `GCB_API` service, which today needs no BGG knowledge at all. More decisively, for the API to sort or facet on `bggRank` — the feature the enrichment exists to serve — OpenSearch needs that value present in each document, which a query-time join cannot provide. (The API does not yet expose bgg fields for search, sort, or facet; that registration is tracked separately. The point is architectural: denormalizing at ingest is what makes it possible at all.)

**Hydrate the index in a second pass** — let `data bgg` bulk-update `bggId` onto documents in place, rather than producing a file for a later `data update` to consume. Rejected because `data update` rewrites every Event on each run to refresh its ChangeLog stamp, hydrating from whatever mapping it was given; an out-of-band pass would have its writes clobbered on the next pull unless `data update` also carried the mapping — which is the current design, with an extra moving part.

## Consequences

The ordering is load-bearing and easy to get wrong silently. Running `data bgg` against a stale or empty index yields a nearly empty mapping rather than an error — commit `69878f6` fixed exactly this, where `bgg` running before `init` produced 2 combos instead of thousands. `data bgg` misses are logged and omitted, so a bad ordering degrades to "no enrichment" rather than failing loudly.

Enrichment is always one cycle behind for combos the index has not seen yet, and `data bgg` cannot run offline or against a fresh environment — it needs a populated cluster. This is the root constraint behind ADR-0002: because the mapping is derived from live index state and consumed by a subsequent pass, it is generated state rather than a source artifact, which is what invalidated ADR-0001's committed-mapping premise. ADR-0001 was not wrong when written; PR #53 changed the architecture out from under it without updating it.

Because BGG values live on Event documents, BGG rank and rating drift is indistinguishable from a Gen Con edit at diff time — the ChangeLog noise recorded in ADR-0002 is a direct consequence of denormalizing at ingest, not an independent problem.

`bgg.LoadGenConCombos` and its tests are dead code describing an architecture the project no longer has. They should be deleted; leaving them is how the next reader concludes the CSV path is still supported.
