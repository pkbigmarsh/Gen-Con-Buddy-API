# Remove Premier Event from SpecialCategory enum

**Issue:** pkbigmarsh/Gen-Con-Buddy-API#51

## Summary

`Premier Event` no longer appears in Gen Con's source data. Remove it from the `Category` enum, its validation, its search term mapping, and all references in tests and documentation. No backward compatibility handling — treat it as if it never existed.

## Changes

**`internal/event/enums.go`**
- Delete `Premier Category = "Premier Event"` constant
- Remove `Premier` from the `ValidateCategory` switch
- Remove the `"premier"` case from `CategoryFromSearchTerm`

**`internal/event/search_test.go`**
- Update the `"specialCategory multi value"` test: change value from `"official,premier"` to `"none,official"` and update the expected query to match

**`CONTEXT.md`**
- Update `SpecialCategory` definition to reference only `Gen Con Presents`
- Close the open question about `Premier Event`

## Out of scope

No OpenSearch index migration needed — the field is freeform string storage; existing documents with `"Premier Event"` remain in the index but are simply never matched by any query (no active events carry the value).
