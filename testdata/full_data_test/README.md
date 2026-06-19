# Full-data manual verification

Runs the real `gcb` data pipeline end-to-end against a local OpenSearch:

    fetch-bgg → bgg → init → update

Unlike `../bgg_tests/test_bgg_mappings.sh` (fixture smoke test), this fetches
real BoardGameGeek data and loads a real Gen Con events file.

## Prerequisites

- Local OpenSearch at `http://localhost:9200`, no auth (security plugin
  disabled — see the docker hint the script prints if it is unreachable).
- `BGG_USERNAME` / `BGG_PASSWORD` exported (used by `fetch-bgg`).
- A Gen Con events file (`.xlsx` or `.csv`).

## Run

    BGG_USERNAME=you BGG_PASSWORD=secret \
      testdata/full_data_test/verify_full_pipeline.sh path/to/events.xlsx

All artifacts (compiled binary, fetched ranks CSV, generated mapping, and
per-step logs) are written to the gitignored `./tmp/` directory. Each step's
output goes to `tmp/logs/NN_<step>.log`; the script fails on any `ERR`-level log
line or non-zero exit, and reports `WRN`-level lines without failing.
