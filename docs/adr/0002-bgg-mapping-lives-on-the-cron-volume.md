# The BGG mapping lives on the cron service's persistent volume

ADR-0001 chose the matcher that produces `bgg_mapping.json` and assumed the file would be committed to the repo. That premise no longer holds: `data bgg` now builds the mapping by reading every indexed `GameSystem` + `RulesEdition` combo out of OpenSearch and matching each against the BGG corpus (see ADR-0003). It is regenerated state, not source. That read also makes the pipeline inherently two-pass — the mapping can only be produced *after* Events are indexed, and can only be consumed by a *later* `data update`. This ADR decides where that regenerated file lives, and supersedes ADR-0001's committed-mapping premise.

We write it to the Railway persistent volume already attached to the `GCB Event Cron` service, at `${RAILWAY_VOLUME_MOUNT_PATH}/bgg_mapping.json`. A single cron service runs both phases on each 6-hourly fire: `data update` hydrates from the mapping the previous run left behind, then `data bgg` regenerates it — but only when the existing file is more than 20 hours old, since the BGG ranks dump only refreshes once a day. Each run enriches from the last run's mapping and refreshes it for the next, with no human in the loop.

This is a stepping stone, not an end state. The mapping is relational data — a combo keyed to a BGG game, with rank and rating attributes — being kept in a denormalized JSON blob because a blob is what the volume affords. We expect to move it to **Postgres** once that database exists in the project, at which point the file, the volume, and the age guard all go away and the two phases become free to run as separate services on independent schedules.

## Considered Options

**Commit the mapping to the repo** (what ADR-0001 originally assumed): maximally reproducible and auditable, and needs no credentials in Railway. Rejected because regeneration requires a human running `data bgg` against a populated index and opening a PR. New GameSystem combos appear throughout the Gen Con season, so enrichment would silently decay between manual refreshes.

**A separate BGG cron service on a daily schedule, sharing the volume:** the natural shape given the two phases have genuinely different cadences. Not possible on the platform — Railway volumes are single-attach ("Each service can only have a single volume"), so a second service has no way to hand the file to the event cron via disk. The age guard in the cron script exists purely to recover this daily cadence within one service.

**A Railway bucket (S3-compatible object storage):** would decouple the two services and survive the move off the volume. Rejected as premature — it adds infrastructure, S3 credentials on both services, and an SDK dependency, all to solve a sharing problem we do not have while the phases live in one service. If we needed sharing before Postgres arrives, this is the option to revisit.

**A dedicated OpenSearch index:** both services already talk to the same Bonsai cluster, so it is shared state we get for free, with no volume and no new secrets. Rejected because it stores relational data in a search index to dodge a scheduling constraint we can live with today. Postgres is where this data is headed; a bespoke OpenSearch index would be a migration we would have to undo.

## Consequences

On a cold volume the first run indexes Events with no `bggId`, and the second run (~6 hours later) fills them in. A brand-new `GameSystem` + `RulesEdition` combo is likewise enriched one cycle late — it must be indexed before `data bgg` can see it. Both are acceptable for data that changes on a convention's timescale.

The mapping is unversioned and lives only on the volume. There is no history of what mapped to what on a given day, no way to reproduce a past run's enrichment, and a lost volume means a cold start rather than a lost artifact. `data bgg` writes to a temporary file and renames it, because `loadBGGMapping` only *warns* on an unreadable mapping — a truncated write would silently strip `bggId` from every Event for a full cycle rather than failing loudly.

Enrichment is best-effort and never fails the pull. Events are the critical data; BGG rank and rating are a bonus. A `data bgg` failure therefore logs a warning and exits 0, leaving the previous mapping intact — only a `data update` failure marks the run failed. This keeps a red run meaning "the pull is broken" rather than "the bonus didn't land", which is what makes a red run worth reacting to at all. The cost is that a sustained enrichment outage is invisible in Railway's run status and shows up only as a warning in the logs; the mapping's mtime is the guard, so a failed refresh retries on the next fire without intervention.

The 20-hour threshold is deliberate rather than 24: with runs at 00/06/12/18 UTC, a mapping written at 06:00 is 24 hours old at the next day's 06:00 run and regenerates there, so the refresh holds a stable daily hour while still self-healing if a run is skipped or fails.

`bggRank` and `bggAvgRating` are not in `EventJsonCmpIgnoredFields`, and BGG average ratings drift daily. Each nightly refresh will therefore make the following `data update` diff a large number of Events as "updated" from rating noise alone, polluting the ChangeLog — which is meant to record what Gen Con changed. This was accepted knowingly to keep the change infra-only; adding `/bggRank` and `/bggAvgRating` to the ignore list is tracked in [issue #56](https://github.com/pkbigmarsh/Gen-Con-Buddy-API/issues/56).
