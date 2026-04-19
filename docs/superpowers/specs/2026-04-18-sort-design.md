# Sort Feature Design

**Date:** 2026-04-18

## Overview

Add sort support to the `/api/events/search` endpoint. The `sort` query parameter accepts a single `{field}.{asc|desc}` value. When omitted, results default to ascending `startDateTime` order.

## API

**Query parameter:** `sort` (single value, format: `{fieldName}.{asc|desc}`)

Examples:
- `sort=startDateTime.asc` — earliest events first (same as default)
- `sort=cost.desc` — most expensive first
- `sort=title.asc` — alphabetical by title

**Default behavior:** If `sort` is not provided, the response is sorted by `startDateTime` ascending.

**Validation errors (400):**
- More than one `sort` param provided
- Field name not in the known field set
- `filter` used as sort field (it is a virtual search-only field, not stored in OpenSearch)
- Direction is not `asc` or `desc`

## Changes

### `internal/event/search.go`

Add `SortField Field` and `SortDir string` to `SearchRequest`. Zero values mean "use default sort."

Add `NewSortFromString(s string) (Field, string, error)`:
1. Split on `.` — expect exactly 2 parts
2. Validate field via `FieldFromString`
3. Reject `Filter` field
4. Validate direction is `"asc"` or `"desc"`
5. Return parsed `(Field, dir, nil)` or a descriptive error

### `internal/api/event_handler.go`

Replace `case "sort": // TODO lol` with:
1. Reject if `len(values) > 1` → 400
2. Call `NewSortFromString(values[0])`
3. On error → 400 with detail message
4. On success → set `searchReq.SortField` and `searchReq.SortDir`

### `internal/event/repo.go`

In `Search`, build a sort clause before executing the OpenSearch request:

- If `req.SortField == ""` → use `[{ "startDateTime": { "order": "asc" } }]`
- Otherwise → use `req.SortField` (appending `.keyword` for text-type fields) with `req.SortDir`

**Text fields that require `.keyword` suffix for sorting:**
Group, Title, ShortDescription, LongDescription, GameSystem, RulesEdition, MaterialsProvided, MaterialsRequiredDetails, GMNames, Tournament, Location, RoomName, TableNumber, Prize, RulesComplexity

All other field types (keyword, numeric, date, double) sort on the field name directly.

Append the sort clause to `searchBody` in all cases (replacing the implicit OpenSearch relevance default).
