# Bad Grouping Flag — Eval Viewer

**Date:** 2026-04-25

## Problem

Some eval rows group GenCon events that don't belong together (events for different games merged under one `game_system` + `rules_edition` key). These rows can't be given a `correct_bgg_id`, but leaving them blank is ambiguous — blank currently means "not yet graded."

The goal is a third, visually distinct state that marks a row as ungradeable due to bad grouping, supports filtering for bulk review, and exports cleanly in the CSV for downstream analysis.

## Approach

Store the sentinel string `"BAD_GROUPING"` in `correct_bgg_id` (both in localStorage and the exported CSV). This reuses the existing storage and export mechanism with no structural changes, and is self-documenting for anyone filtering the CSV.

## Row States

| State | Symbol | Color | `correct_bgg_id` value |
|---|---|---|---|
| Ungraded | `○` | muted | `""` (empty) |
| Bad grouping | `⚠` | orange | `"BAD_GROUPING"` |
| Graded | `✓` | green | a BGG numeric ID |

## UI Changes

### Table row
The scored column displays `⚠` in orange for bad-grouping rows, visually distinct from `○` (ungraded) and `✓` (graded).

### Expand panel
- A **"Flag as bad grouping"** button appears alongside the existing BGG ID input.
- When a row is already flagged, the input area is replaced with a "Bad grouping" label and a **"clear"** button to unflag it.
- Clicking the flag button sets `correct_bgg_id = "BAD_GROUPING"` and hides the text input.

### Sidebar progress
- The scored count excludes bad-grouping rows.
- A separate counter below shows the bad-grouping count so graders always know the full picture.

### Filter dropdown
A **"Bad grouping"** option is added to filter the table to only flagged rows.

## Export

No structural change. The exported CSV `correct_bgg_id` column will contain `"BAD_GROUPING"` for flagged rows. Downstream accuracy analysis should filter `correct_bgg_id != "BAD_GROUPING"` before computing match rates.

## Out of Scope

- Fixing the upstream grouping logic (separate concern).
- Any automated detection of bad groupings.
