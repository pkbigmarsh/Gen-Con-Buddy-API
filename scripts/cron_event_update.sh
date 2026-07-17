#!/bin/bash
# Production cron entrypoint for the GCB Event Cron Railway service.
#
# Runs two phases on each fire:
#   1. data update — re-ingest the Gen Con catalog, hydrating BGG data from the
#      mapping the previous run left on the volume.
#   2. data bgg — regenerate that mapping, but only when it is older than the age
#      guard, so the BGG ranks dump is pulled at most once a day.
#
# Events are the critical data; BGG enrichment is a bonus. A phase-1 failure fails
# the run; a phase-2 failure only warns and exits 0, leaving the previous mapping
# intact. See docs/adr/0002-bgg-mapping-lives-on-the-cron-volume.md.
#
# For the local testing entrypoint (keeps the catalog on disk, uses the .zip
# format), see scripts/event_fetch_and_update.sh instead.
set -uo pipefail

VOLUME_PATH="${RAILWAY_VOLUME_MOUNT_PATH:-/events}"
MAPPING="${VOLUME_PATH}/bgg_mapping.json"
DOWNLOAD_URL="${DOWNLOAD_URL:-https://www.gencon.com/downloads/events.xlsx}"
MAPPING_MAX_AGE_MIN=1200 # 20h — regenerate once a day, self-healing if a run is skipped

mkdir -p "${VOLUME_PATH}"

# Phase 1: ingest, hydrating from the previous run's mapping (absent on a cold volume).
mapping_args=()
if [ -s "${MAPPING}" ]; then
  mapping_args=(--bgg-mapping "${MAPPING}")
else
  echo "no mapping at ${MAPPING}; events will have no bggId this run"
fi

bin/gcb data update --download_url "${DOWNLOAD_URL}" "${mapping_args[@]}"
update_status=$?
if [ ${update_status} -ne 0 ]; then
  echo "data update failed (${update_status})"
  exit ${update_status}
fi

# Phase 2: refresh the mapping at most once a day.
if [ -s "${MAPPING}" ] && [ -z "$(find "${MAPPING}" -mmin +${MAPPING_MAX_AGE_MIN})" ]; then
  echo "mapping is fresh; skipping bgg regeneration"
  exit 0
fi

# Phase 2 is best-effort. Events are the critical data and are already committed;
# BGG enrichment is a bonus, so a failure here must not fail the run.
# Write to a temp file and rename, so a failed run cannot truncate a good mapping.
bin/gcb data bgg --output "${MAPPING}.tmp"
bgg_status=$?
if [ ${bgg_status} -ne 0 ]; then
  echo "WARNING: bgg mapping refresh failed (${bgg_status}); events are indexed and the previous mapping is intact; retrying next run"
  rm -f "${MAPPING}.tmp"
  exit 0
fi

mv "${MAPPING}.tmp" "${MAPPING}"
echo "mapping refreshed at ${MAPPING}"
exit 0
