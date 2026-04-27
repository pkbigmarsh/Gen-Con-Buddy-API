# BGG Eval Viewer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone HTML viewer for `bgg_eval.csv` that shows matcher comparison, Gen Con event data, and supports manual `correct_bgg_id` scoring with localStorage persistence and CSV export.

**Architecture:** A Python embed script (`cmd/evalbgg/embed_data.py`) reads `bgg_eval.csv` and `data.csv`, JSON-encodes them, and injects them into `cmd/evalbgg/viewer_template.html`, writing the output to `bgg_eval_viewer.html` at the repo root. All UI logic is vanilla JS inside the template. No external dependencies.

**Tech Stack:** Python 3 stdlib (`csv`, `json`, `os`), vanilla HTML/CSS/JS, `localStorage` for persistence, `Blob` + `URL.createObjectURL` for CSV export.

---

## File Map

| File | Responsibility |
|------|---------------|
| `cmd/evalbgg/embed_data.py` | Reads CSVs → injects JSON into template → writes `bgg_eval_viewer.html` |
| `cmd/evalbgg/viewer_template.html` | Full app HTML/CSS/JS with `/*EVAL_DATA_PLACEHOLDER*/` and `/*EVENTS_DATA_PLACEHOLDER*/` |
| `bgg_eval_viewer.html` | Generated output at repo root — gitignored, regenerated from data |

---

## Task 1: Embed script and gitignore

**Files:**
- Create: `cmd/evalbgg/embed_data.py`
- Modify: `.gitignore`

- [ ] **Step 1: Create `cmd/evalbgg/embed_data.py`**

```python
#!/usr/bin/env python3
"""Inject bgg_eval.csv and data.csv (BGM rows) as JSON into viewer_template.html."""

import csv
import json
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT  = os.path.join(SCRIPT_DIR, '..', '..')

BGM_FIELDS = [
    'Game ID', 'Title', 'Short Description', 'Event Type',
    'Game System', 'Rules Edition', 'Minimum Players', 'Maximum Players',
    'Age Required', 'Experience Required', 'Start Date & Time', 'Duration',
    'Location', 'Room Name', 'Table Number', 'GM Names',
    'Website', 'Cost $', 'Tickets Available', 'Tournament?',
]

def read_eval_csv(path):
    rows = []
    with open(path, newline='', encoding='utf-8') as f:
        for row in csv.DictReader(f):
            rows.append(dict(row))
    return rows

def read_events_index(path):
    """Returns dict: 'GameSystem||RulesEdition' -> [event, ...]"""
    index = {}
    with open(path, newline='', encoding='cp1252') as f:
        for row in csv.DictReader(f):
            if not row.get('Event Type', '').startswith('BGM'):
                continue
            event = {k: row.get(k, '').strip() for k in BGM_FIELDS}
            key = row.get('Game System', '').strip() + '||' + row.get('Rules Edition', '').strip()
            index.setdefault(key, []).append(event)
    return index

def main():
    eval_path     = os.path.join(REPO_ROOT, 'bgg_eval.csv')
    events_path   = os.path.join(REPO_ROOT, 'data.csv')
    template_path = os.path.join(SCRIPT_DIR, 'viewer_template.html')
    output_path   = os.path.join(REPO_ROOT, 'bgg_eval_viewer.html')

    print('Reading eval CSV...')
    eval_data = read_eval_csv(eval_path)
    print(f'  {len(eval_data)} combos')

    print('Reading Gen Con events CSV...')
    events_index = read_events_index(events_path)
    total = sum(len(v) for v in events_index.values())
    print(f'  {total} BGM events across {len(events_index)} combos')

    with open(template_path, encoding='utf-8') as f:
        html = f.read()

    html = html.replace('/*EVAL_DATA_PLACEHOLDER*/',   json.dumps(eval_data))
    html = html.replace('/*EVENTS_DATA_PLACEHOLDER*/', json.dumps(events_index))

    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(html)

    size_kb = os.path.getsize(output_path) // 1024
    print(f'Done. {output_path} ({size_kb} KB)')

if __name__ == '__main__':
    main()
```

- [ ] **Step 2: Add `bgg_eval_viewer.html` to `.gitignore`**

Append to `.gitignore`:

```
bgg_eval_viewer.html
```

- [ ] **Step 3: Verify the script can be imported without error (template doesn't exist yet)**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
python3 -c "import cmd.evalbgg.embed_data" 2>&1 || python3 cmd/evalbgg/embed_data.py 2>&1 | head -5
```

Expected: either "Reading eval CSV..." (if it runs) or a clean import. It will fail on missing template — that's fine.

- [ ] **Step 4: Commit**

```bash
git add cmd/evalbgg/embed_data.py .gitignore
git commit -m "feat(evalbgg): add embed script and gitignore viewer output"
```

---

## Task 2: Template skeleton — layout, CSS, static sidebar

**Files:**
- Create: `cmd/evalbgg/viewer_template.html`

- [ ] **Step 1: Create `cmd/evalbgg/viewer_template.html`**

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>BGG Eval Viewer</title>
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
:root {
  --bg:      #0d0d0d;
  --surface: #111111;
  --surface2:#1a1a1a;
  --border:  #222222;
  --border2: #1e1e1e;
  --text:    #e0e0e0;
  --muted:   #555555;
  --accent:  #7c5cbf;
  --accent2: #9b7fd4;
  --green:   #2ecc71;
  --orange:  #e67e22;
  --red:     #e74c3c;
  --blue:    #3498db;
}
body { font-family: system-ui, -apple-system, sans-serif; background: var(--bg); color: var(--text); font-size: 13px; }
a { color: var(--accent2); text-decoration: none; }
a:hover { text-decoration: underline; }

/* ── App shell ───────────────────────────────────────────── */
.app { display: grid; grid-template-columns: 260px 1fr; height: 100vh; overflow: hidden; }

/* ── Sidebar ─────────────────────────────────────────────── */
.sidebar {
  background: var(--surface);
  border-right: 1px solid var(--border);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
}
.sidebar h1 { font-size: 14px; font-weight: 700; color: #fff; letter-spacing: .05em; }
.sidebar .sub { font-size: 11px; color: var(--muted); margin-top: 2px; }

.stat-block { background: var(--surface2); border-radius: 6px; padding: 12px; }
.stat-block .lbl { font-size: 10px; text-transform: uppercase; letter-spacing: .08em; color: var(--muted); margin-bottom: 8px; }

/* scored progress */
.score-big { font-size: 26px; font-weight: 700; color: var(--accent); }
.score-denom { font-size: 13px; color: var(--muted); }
.prog-bg { background: var(--border); border-radius: 3px; height: 5px; margin-top: 8px; }
.prog-fill { background: var(--accent); border-radius: 3px; height: 5px; width: 0%; transition: width .3s; }

/* agreement chart */
#agree-chart { width: 100%; margin-top: 4px; }

/* match rate bars */
.mr-row { display: flex; align-items: center; gap: 5px; margin-bottom: 5px; }
.mr-name { font-size: 10px; color: var(--muted); width: 132px; flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mr-bg { flex: 1; background: var(--border); border-radius: 2px; height: 4px; }
.mr-bar { height: 4px; border-radius: 2px; }
.mr-pct { font-size: 10px; color: var(--muted); width: 28px; text-align: right; }

/* export button */
.export-btn {
  margin-top: auto;
  background: var(--accent); color: #fff; border: none;
  border-radius: 6px; padding: 10px; font-size: 12px; font-weight: 600;
  cursor: pointer; width: 100%;
}
.export-btn:hover { background: var(--accent2); }

/* ── Main area ───────────────────────────────────────────── */
.main { display: flex; flex-direction: column; overflow: hidden; }

/* toolbar */
.toolbar {
  background: var(--surface); border-bottom: 1px solid var(--border);
  padding: 10px 16px; display: flex; align-items: center; gap: 10px; flex-shrink: 0;
}
.toolbar input[type=text] {
  background: var(--surface2); border: 1px solid var(--border2);
  border-radius: 5px; color: var(--text); padding: 6px 10px; font-size: 12px; width: 210px;
}
.toolbar select {
  background: var(--surface2); border: 1px solid var(--border2);
  border-radius: 5px; color: var(--text); padding: 6px 8px; font-size: 12px;
}
.row-count { font-size: 11px; color: var(--muted); }

/* table */
.table-wrap { flex: 1; overflow-y: auto; }
table { width: 100%; border-collapse: collapse; }
thead th {
  background: var(--surface); position: sticky; top: 0; z-index: 2;
  padding: 8px 12px; text-align: left; font-size: 11px;
  text-transform: uppercase; letter-spacing: .06em; color: var(--muted);
  border-bottom: 1px solid var(--border2); cursor: pointer; user-select: none;
}
thead th:hover { color: var(--text); }
thead th.sort-asc::after  { content: ' ↑'; }
thead th.sort-desc::after { content: ' ↓'; }
tbody tr { border-bottom: 1px solid #161616; cursor: pointer; transition: background .1s; }
tbody tr:hover { background: #181818; }
tbody tr.expanded { background: #12122a; }
tbody td { padding: 9px 12px; font-size: 12px; vertical-align: middle; }

/* agreement pills */
.pill { display: inline-block; border-radius: 10px; padding: 2px 8px; font-size: 11px; font-weight: 600; min-width: 28px; text-align: center; }
.pill-red  { background: #3d1515; color: var(--red); }
.pill-ora  { background: #3d2a10; color: var(--orange); }
.pill-grn  { background: #1a3a1a; color: var(--green); }
.pill-blu  { background: #1a2a3a; color: var(--blue); }
.scored-yes { color: var(--green); }
.scored-no  { color: #333; }

/* ── Expanded row panel ──────────────────────────────────── */
.expand-td { padding: 0 !important; }
.expand-panel { background: #0b0b1a; border-bottom: 2px solid var(--accent); }

/* tabs */
.etabs { display: flex; border-bottom: 1px solid var(--border); }
.etab {
  padding: 10px 16px; font-size: 12px; color: var(--muted);
  cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px;
}
.etab:hover { color: var(--text); }
.etab.active { color: var(--accent2); border-bottom-color: var(--accent); }
.etab-body { padding: 16px; }

/* ── Matcher cards ───────────────────────────────────────── */
.matcher-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(185px, 1fr)); gap: 8px; margin-bottom: 14px; }
.mc {
  background: #111828; border: 1px solid #1e2840; border-radius: 6px;
  padding: 10px; font-size: 11px;
}
.mc.consensus { border-color: var(--accent); background: #170f2e; }
.mc .mc-name  { color: var(--muted); font-size: 10px; margin-bottom: 3px; }
.mc .mc-votes { color: var(--accent2); font-size: 10px; }
.mc .mc-game  { color: var(--text); font-weight: 600; margin-bottom: 2px; word-break: break-word; }
.mc .mc-link  { font-size: 10px; }
.mc .mc-score { color: var(--green); font-size: 10px; }
.mc .mc-none  { color: #333; font-style: italic; }
.use-btn {
  margin-top: 6px; background: #7c5cbf22; border: 1px solid #7c5cbf55;
  color: var(--accent2); border-radius: 4px; padding: 3px 8px;
  font-size: 10px; cursor: pointer; width: 100%;
}
.use-btn:hover { background: #7c5cbf44; }

/* score row */
.score-row { display: flex; align-items: center; gap: 10px; background: #111828; border-radius: 6px; padding: 10px; }
.score-row label { font-size: 11px; color: var(--muted); }
.score-row input {
  background: var(--bg); border: 1px solid var(--border);
  border-radius: 4px; color: var(--text); padding: 5px 10px; font-size: 12px; width: 130px;
}
.score-row input:focus { outline: none; border-color: var(--accent); }
.clear-btn { background: none; border: 1px solid var(--border); color: var(--muted); border-radius: 4px; padding: 5px 10px; font-size: 11px; cursor: pointer; }
.clear-btn:hover { color: var(--text); }
.save-ok { font-size: 11px; color: var(--green); display: none; }

/* ── Events table ────────────────────────────────────────── */
.events-scroll { max-height: 280px; overflow-y: auto; border: 1px solid var(--border2); border-radius: 6px; }
.ev-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.ev-table th {
  background: var(--surface); position: sticky; top: 0;
  padding: 6px 10px; text-align: left; font-size: 10px;
  text-transform: uppercase; letter-spacing: .06em; color: var(--muted);
  border-bottom: 1px solid var(--border2);
}
.ev-table td { padding: 7px 10px; border-bottom: 1px solid #141414; vertical-align: top; }
.ev-table tr:hover td { background: #141420; }
.ev-id { font-family: monospace; font-size: 11px; color: var(--muted); }
.ev-desc { max-width: 260px; color: #888; }
.ev-loc { color: #888; }
.badge { display: inline-block; border-radius: 3px; padding: 1px 5px; font-size: 10px; margin-right: 2px; }
.badge-tourn { background: #2a1a1a; color: var(--red); }
.badge-exp   { background: #2a2a10; color: var(--orange); }
.badge-misc  { background: #1a2a1a; color: var(--green); }
</style>
</head>
<body>
<div class="app">

  <!-- ── Sidebar ── -->
  <div class="sidebar">
    <div>
      <h1>BGG Eval</h1>
      <div class="sub" id="sb-subtitle">Loading…</div>
    </div>

    <div class="stat-block">
      <div class="lbl">Scored</div>
      <span class="score-big" id="sb-scored">0</span><span class="score-denom"> / <span id="sb-total">—</span></span>
      <div class="prog-bg"><div class="prog-fill" id="sb-prog"></div></div>
    </div>

    <div class="stat-block">
      <div class="lbl">Agreement distribution</div>
      <svg id="agree-chart" viewBox="0 0 220 72"></svg>
    </div>

    <div class="stat-block">
      <div class="lbl">Match rate by matcher</div>
      <div id="match-rate-bars"></div>
    </div>

    <button class="export-btn" onclick="exportCSV()">⬇ Export scored CSV</button>
  </div>

  <!-- ── Main ── -->
  <div class="main">
    <div class="toolbar">
      <input type="text" id="search-input" placeholder="🔍 Search game system…" oninput="applyFilters()" />
      <select id="filter-select" onchange="applyFilters()">
        <option value="lte7">Agreement ≤ 7 (uncertain)</option>
        <option value="lte10">Agreement ≤ 10</option>
        <option value="all">All rows</option>
        <option value="unscored">Unscored only</option>
        <option value="scored">Scored only</option>
      </select>
      <span class="row-count" id="row-count"></span>
    </div>

    <div class="table-wrap">
      <table id="main-table">
        <thead>
          <tr>
            <th onclick="sortBy('game_system')">Game System</th>
            <th onclick="sortBy('rules_edition')">Edition</th>
            <th onclick="sortBy('event_count')" style="width:65px">Events</th>
            <th onclick="sortBy('agreement_count')" style="width:70px">Agree</th>
            <th>Consensus match</th>
            <th style="width:36px">✓</th>
          </tr>
        </thead>
        <tbody id="table-body"></tbody>
      </table>
    </div>
  </div>
</div>

<script>
// ── Embedded data (injected by embed_data.py) ────────────────
const EVAL_DATA   = /*EVAL_DATA_PLACEHOLDER*/;
const EVENTS_DATA = /*EVENTS_DATA_PLACEHOLDER*/;

// ── Constants ────────────────────────────────────────────────
const MATCHERS = [
  'exact-system-rank','fuzzy-system-rank','fuzzy-system-rated','token-system-rank',
  'exact-always-edition-rank','fuzzy-always-edition-rank','token-always-edition-rank',
  'exact-smart-edition-rank','fuzzy-smart-edition-rank','fuzzy-smart-edition-rated','token-smart-edition-rank',
  'fuzzy-title-rank',
  'exact-title-derived-always-rank','fuzzy-title-derived-always-rank',
  'exact-title-derived-smart-rank','fuzzy-title-derived-smart-rank','fuzzy-title-derived-smart-rated','token-title-derived-smart-rank',
];
const STORAGE_KEY = 'bgg-eval-scores';
const BGG_URL = id => `https://boardgamegeek.com/boardgame/${id}`;

// ── State ────────────────────────────────────────────────────
let scores       = {};   // { 'GameSystem||Edition': bggId }
let filteredRows = [];   // current visible rows (indices into EVAL_DATA)
let sortKey      = 'agreement_count';
let sortDir      = 1;    // 1 = asc, -1 = desc
let expandedIdx  = null; // EVAL_DATA index of expanded row

// ── Persistence ──────────────────────────────────────────────
function loadScores() {
  try { scores = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}'); } catch { scores = {}; }
}
function saveScore(key, value) {
  if (value) scores[key] = value; else delete scores[key];
  localStorage.setItem(STORAGE_KEY, JSON.stringify(scores));
  updateScoredProgress();
}
function rowKey(row) { return row.game_system + '||' + row.rules_edition; }

// ── Agreement pill ───────────────────────────────────────────
function pillClass(n) {
  n = parseInt(n);
  if (n <= 5)  return 'pill-red';
  if (n <= 9)  return 'pill-ora';
  if (n <= 12) return 'pill-grn';
  return 'pill-blu';
}

// ── BGG link helper ──────────────────────────────────────────
function bggLink(id, label) {
  if (!id) return '';
  return `<a href="${BGG_URL(id)}" target="_blank" rel="noopener">${label || '↗ #' + id}</a>`;
}

// ── Filtering ────────────────────────────────────────────────
function applyFilters() {
  const q    = document.getElementById('search-input').value.toLowerCase();
  const mode = document.getElementById('filter-select').value;
  filteredRows = EVAL_DATA
    .map((row, i) => ({ row, i }))
    .filter(({ row, i }) => {
      if (q && !row.game_system.toLowerCase().includes(q)) return false;
      const agree = parseInt(row.agreement_count);
      const key   = rowKey(row);
      const isScored = !!scores[key];
      if (mode === 'lte7'     && agree > 7)    return false;
      if (mode === 'lte10'    && agree > 10)   return false;
      if (mode === 'unscored' && isScored)     return false;
      if (mode === 'scored'   && !isScored)    return false;
      return true;
    });
  sortRows();
  renderTable();
}

// ── Sorting ──────────────────────────────────────────────────
function sortBy(key) {
  if (sortKey === key) sortDir *= -1; else { sortKey = key; sortDir = 1; }
  document.querySelectorAll('thead th').forEach(th => {
    th.classList.remove('sort-asc', 'sort-desc');
  });
  const colIndex = ['game_system','rules_edition','event_count','agreement_count'].indexOf(key);
  if (colIndex >= 0) {
    const th = document.querySelectorAll('thead th')[colIndex];
    th.classList.add(sortDir === 1 ? 'sort-asc' : 'sort-desc');
  }
  sortRows();
  renderTable();
}
function sortRows() {
  filteredRows.sort((a, b) => {
    let av = a.row[sortKey], bv = b.row[sortKey];
    if (!isNaN(av) && !isNaN(bv)) { av = parseFloat(av); bv = parseFloat(bv); }
    if (av < bv) return -sortDir;
    if (av > bv) return  sortDir;
    return 0;
  });
}

// ── Table rendering ──────────────────────────────────────────
function renderTable() {
  document.getElementById('row-count').textContent = `Showing ${filteredRows.length} rows`;
  const tbody = document.getElementById('table-body');
  tbody.innerHTML = '';

  filteredRows.forEach(({ row, i }) => {
    const key      = rowKey(row);
    const isScored = !!scores[key];
    const agree    = parseInt(row.agreement_count);
    const isOpen   = expandedIdx === i;

    // consensus cell
    let consensus = '<span style="color:var(--muted)">split — no consensus</span>';
    if (row.consensus_id) {
      consensus = `${row.consensus_name || ''} ${bggLink(row.consensus_id)}`;
    }

    // main row
    const tr = document.createElement('tr');
    if (isOpen) tr.classList.add('expanded');
    tr.innerHTML = `
      <td><strong>${esc(row.game_system)}</strong></td>
      <td>${esc(row.rules_edition)}</td>
      <td>${row.event_count}</td>
      <td><span class="pill ${pillClass(agree)}">${agree}</span></td>
      <td>${consensus}</td>
      <td class="${isScored ? 'scored-yes' : 'scored-no'}">${isScored ? '✓' : '○'}</td>
    `;
    tr.addEventListener('click', () => toggleExpand(i, tr));
    tbody.appendChild(tr);

    // expand row (hidden by default)
    const expandTr = document.createElement('tr');
    expandTr.style.display = isOpen ? '' : 'none';
    const expandTd = document.createElement('td');
    expandTd.colSpan = 6;
    expandTd.className = 'expand-td';
    expandTd.innerHTML = isOpen ? buildExpandPanel(row, i) : '';
    expandTr.appendChild(expandTd);
    tbody.appendChild(expandTr);
  });
}

function esc(str) {
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

// ── Row expansion ────────────────────────────────────────────
function toggleExpand(dataIdx, clickedTr) {
  if (expandedIdx === dataIdx) {
    expandedIdx = null;
  } else {
    expandedIdx = dataIdx;
  }
  renderTable();
  if (expandedIdx !== null) {
    // scroll expanded row into view after render
    setTimeout(() => {
      const expanded = document.querySelector('tr.expanded');
      if (expanded) expanded.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }, 50);
  }
}

function buildExpandPanel(row, dataIdx) {
  const key = rowKey(row);
  const savedId = scores[key] || '';
  const eventsForCombo = EVENTS_DATA[key] || [];
  const infoTag = row.edition_informative === 'true'
    ? `<span style="margin-left:8px;font-size:10px;background:#7c5cbf33;color:var(--accent2);border-radius:4px;padding:2px 6px;">informative edition</span>` : '';
  return `
<div class="expand-panel">
  <div class="etabs">
    <div class="etab active" onclick="switchTab(this,'matchers-${dataIdx}','events-${dataIdx}')">🎯 Matchers &amp; Scoring</div>
    <div class="etab" onclick="switchTab(this,'events-${dataIdx}','matchers-${dataIdx}')">📋 Gen Con Events (${eventsForCombo.length})</div>
  </div>
  <div class="etab-body" id="matchers-${dataIdx}">
    <div style="font-size:13px;font-weight:600;color:#fff;margin-bottom:12px;">
      ${esc(row.game_system)} · ${esc(row.rules_edition)}${infoTag}
      <span style="margin-left:8px;font-size:10px;background:#1a3a1a33;color:var(--green);border-radius:4px;padding:2px 6px;">
        Rep title: "${esc(row.representative_title)}"
      </span>
    </div>
    ${buildMatcherGrid(row, dataIdx)}
    <div class="score-row">
      <label>correct_bgg_id</label>
      <input type="text" id="score-${dataIdx}" value="${esc(savedId)}" placeholder="BGG ID…"
             oninput="onScoreInput(${dataIdx}, '${esc(key).replace(/'/g, "\\'")}', this.value)" />
      <button class="clear-btn" onclick="clearScore(${dataIdx}, '${esc(key).replace(/'/g, "\\'")}')">clear</button>
      <span class="save-ok" id="save-ok-${dataIdx}">✓ saved</span>
    </div>
  </div>
  <div class="etab-body" id="events-${dataIdx}" style="display:none">
    ${buildEventsTable(eventsForCombo)}
  </div>
</div>`;
}

function switchTab(el, showId, hideId) {
  el.closest('.etabs').querySelectorAll('.etab').forEach(t => t.classList.remove('active'));
  el.classList.add('active');
  document.getElementById(showId).style.display = '';
  document.getElementById(hideId).style.display = 'none';
}

// ── Matcher cards ────────────────────────────────────────────
function buildMatcherGrid(row, dataIdx) {
  // Find consensus: which BGG ID do most matchers return?
  const votes = {};
  MATCHERS.forEach(m => {
    const id = row[m + '_id'];
    if (id) votes[id] = (votes[id] || 0) + 1;
  });
  const topId = Object.keys(votes).sort((a, b) => votes[b] - votes[a])[0];
  const topVotes = topId ? votes[topId] : 0;

  const cards = MATCHERS.map(m => {
    const id    = row[m + '_id'];
    const name  = row[m + '_name'];
    const score = row[m + '_score'];
    const isCon = id && id === topId && topVotes > 1;
    const key   = rowKey(row);

    if (!id) {
      return `<div class="mc"><div class="mc-name">${m}</div><div class="mc-none">no match</div></div>`;
    }
    const voteLabel = isCon ? `<span class="mc-votes">· ${topVotes} agree</span>` : '';
    const scoreLabel = score ? `<div class="mc-score">score: ${score}</div>` : '';
    return `
<div class="mc${isCon ? ' consensus' : ''}">
  <div class="mc-name">${m} ${voteLabel}</div>
  <div class="mc-game">${esc(name)}</div>
  <div class="mc-link">${bggLink(id, '↗ BGG #' + id)}</div>
  ${scoreLabel}
  <button class="use-btn" onclick="useId(${dataIdx},'${esc(key).replace(/'/g,"\\'")}','${id}')">✓ Use this ID</button>
</div>`;
  }).join('');
  return `<div class="matcher-grid">${cards}</div>`;
}

function useId(dataIdx, key, id) {
  document.getElementById('score-' + dataIdx).value = id;
  saveScore(key, id);
  const ok = document.getElementById('save-ok-' + dataIdx);
  ok.style.display = 'inline';
}

function onScoreInput(dataIdx, key, value) {
  saveScore(key, value.trim());
  const ok = document.getElementById('save-ok-' + dataIdx);
  ok.style.display = value.trim() ? 'inline' : 'none';
}

function clearScore(dataIdx, key) {
  document.getElementById('score-' + dataIdx).value = '';
  saveScore(key, '');
  document.getElementById('save-ok-' + dataIdx).style.display = 'none';
}

// ── Events table ─────────────────────────────────────────────
function buildEventsTable(events) {
  if (!events.length) return '<p style="color:var(--muted);font-size:12px;">No events found for this combo.</p>';
  const rows = events.map(e => {
    const players = (e['Minimum Players'] && e['Maximum Players'])
      ? `${e['Minimum Players']}–${e['Maximum Players']}` : (e['Minimum Players'] || '—');
    const loc = [e['Room Name'], e['Table Number'] ? 'Tbl ' + e['Table Number'] : ''].filter(Boolean).join(' · ');
    const badges = [
      e['Tournament?'] === 'Yes' ? '<span class="badge badge-tourn">Tournament</span>' : '',
      e['Experience Required'] && e['Experience Required'] !== 'None'
        ? `<span class="badge badge-exp">${esc(e['Experience Required'])}</span>` : '',
    ].filter(Boolean).join('');
    return `<tr>
      <td class="ev-id">${esc(e['Game ID'])}</td>
      <td>${esc(e['Title'])}</td>
      <td class="ev-desc">${esc(e['Short Description'])}</td>
      <td>${players}</td>
      <td>${esc(e['Start Date & Time'])}</td>
      <td>${esc(e['Duration'])}</td>
      <td class="ev-loc">${esc(loc)}</td>
      <td>${esc(e['GM Names'])}</td>
      <td>${badges}</td>
    </tr>`;
  }).join('');
  return `
<div class="events-scroll">
  <table class="ev-table">
    <thead><tr>
      <th>Game ID</th><th>Title</th><th>Description</th><th>Players</th>
      <th>Date &amp; Time</th><th>Duration</th><th>Location</th><th>GM</th><th>Tags</th>
    </tr></thead>
    <tbody>${rows}</tbody>
  </table>
</div>`;
}

// ── Sidebar: live stats ───────────────────────────────────────
function updateScoredProgress() {
  const total   = EVAL_DATA.length;
  const scored  = EVAL_DATA.filter(r => !!scores[rowKey(r)]).length;
  document.getElementById('sb-scored').textContent = scored;
  document.getElementById('sb-total').textContent  = total;
  document.getElementById('sb-prog').style.width   = (scored / total * 100).toFixed(1) + '%';
}

function renderAgreeChart() {
  const counts = {};
  EVAL_DATA.forEach(r => {
    const n = parseInt(r.agreement_count);
    counts[n] = (counts[n] || 0) + 1;
  });
  const max = Math.max(...Object.values(counts));
  const barW = 13, gap = 2, h = 57, totalH = 72;
  const keys = Array.from({length: 13}, (_, i) => i + 3);
  const colors = ['#e74c3c','#e74c3c','#e67e22','#e67e22','#f39c12','#2ecc71','#2ecc71','#27ae60','#27ae60','#1abc9c','#1abc9c','#3498db','#2980b9'];
  let svg = `<g transform="translate(10,5)">`;
  keys.forEach((k, idx) => {
    const cnt = counts[k] || 0;
    const barH = cnt ? Math.max(2, (cnt / max) * h) : 0;
    const x = idx * (barW + gap);
    const y = h - barH;
    svg += `<rect x="${x}" y="${y}" width="${barW}" height="${barH}" fill="${colors[idx]}" opacity=".85" rx="1"/>`;
    if (idx === 0 || idx === 6 || idx === 12) {
      svg += `<text x="${x + barW/2}" y="${h+10}" text-anchor="middle" font-size="7" fill="#444">${k}</text>`;
    }
  });
  svg += `<line x1="0" y1="${h}" x2="${keys.length*(barW+gap)-gap}" y2="${h}" stroke="#333" stroke-width="1"/>`;
  svg += `</g>`;
  document.getElementById('agree-chart').innerHTML = svg;
}

function renderMatchRateBars() {
  const total = EVAL_DATA.length;
  const bars = MATCHERS.map(m => {
    const matched = EVAL_DATA.filter(r => r[m + '_id'] !== '').length;
    const pct = matched / total;
    const color = pct >= 0.8 ? '#27ae60' : pct >= 0.3 ? '#f39c12' : '#e74c3c';
    return `<div class="mr-row">
      <span class="mr-name" title="${m}">${m}</span>
      <div class="mr-bg"><div class="mr-bar" style="width:${(pct*100).toFixed(0)}%;background:${color}"></div></div>
      <span class="mr-pct">${(pct*100).toFixed(0)}%</span>
    </div>`;
  }).join('');
  document.getElementById('match-rate-bars').innerHTML = bars;
}

// ── Export CSV ───────────────────────────────────────────────
function exportCSV() {
  const headers = Object.keys(EVAL_DATA[0]);
  const rows = EVAL_DATA.map(row => {
    const out = { ...row };
    const key = rowKey(row);
    out.correct_bgg_id = scores[key] || '';
    return headers.map(h => {
      const v = String(out[h] ?? '');
      return v.includes(',') || v.includes('"') || v.includes('\n')
        ? '"' + v.replace(/"/g, '""') + '"' : v;
    }).join(',');
  });
  const csv = [headers.join(','), ...rows].join('\n');
  const blob = new Blob([csv], { type: 'text/csv' });
  const url  = URL.createObjectURL(blob);
  const a    = document.createElement('a');
  a.href = url; a.download = 'bgg_eval_scored.csv'; a.click();
  URL.revokeObjectURL(url);
}

// ── Init ─────────────────────────────────────────────────────
function init() {
  loadScores();
  document.getElementById('sb-subtitle').textContent = `${EVAL_DATA.length} combos · 18 matchers`;
  updateScoredProgress();
  renderAgreeChart();
  renderMatchRateBars();
  applyFilters(); // also calls renderTable
}

init();
</script>
</body>
</html>
```

- [ ] **Step 2: Run the embed script to generate the viewer**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
python3 cmd/evalbgg/embed_data.py
```

Expected output:
```
Reading eval CSV...
  865 combos
Reading Gen Con events CSV...
  7457 BGM events across 865 combos
Reading template...
Writing .../bgg_eval_viewer.html...
Done. .../bgg_eval_viewer.html (XXXX KB)
```

- [ ] **Step 3: Open in browser and verify**

```bash
xdg-open bgg_eval_viewer.html 2>/dev/null || open bgg_eval_viewer.html 2>/dev/null || echo "Open bgg_eval_viewer.html in your browser"
```

Check:
- Two-column layout renders (sidebar left, main right)
- Sidebar shows "865 combos · 18 matchers", scored progress "0 / 865", agreement chart, match rate bars
- Table shows rows sorted by agreement ascending with colored pills
- Search input filters rows, filter dropdown changes row set
- Expanding a row shows the Matchers tab with 18 cards and BGG links
- "Gen Con Events" tab shows the events table for that combo
- "Use this ID" populates the score input
- Typing in the score input and reloading the page preserves the value (localStorage)
- Export button downloads `bgg_eval_scored.csv`

- [ ] **Step 4: Commit**

```bash
git add cmd/evalbgg/viewer_template.html
git commit -m "feat(evalbgg): add eval viewer template (standalone HTML)"
```

---

## Task 3: Run embed script and add regeneration instructions

**Files:**
- Modify: `cmd/evalbgg/embed_data.py` — add a `--help` usage string

This task verifies the full pipeline works end-to-end and documents the regeneration workflow.

- [ ] **Step 1: Add usage comment to top of `embed_data.py`**

Replace the existing docstring at the top of `cmd/evalbgg/embed_data.py`:

```python
#!/usr/bin/env python3
"""
Inject bgg_eval.csv and data.csv (BGM rows) as JSON into viewer_template.html,
writing bgg_eval_viewer.html at the repo root.

Usage:
  python3 cmd/evalbgg/embed_data.py

Inputs (relative to repo root):
  bgg_eval.csv          — output of cmd/evalbgg/evalbgg binary
  data.csv              — Gen Con events CSV (Windows-1252 encoding)
  cmd/evalbgg/viewer_template.html — HTML/JS template

Output:
  bgg_eval_viewer.html  — standalone viewer, open directly in browser
"""
```

- [ ] **Step 2: Run embed script and confirm file size is reasonable**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
python3 cmd/evalbgg/embed_data.py
ls -lh bgg_eval_viewer.html
```

Expected: file between 4 MB and 10 MB.

- [ ] **Step 3: Spot-check the generated file**

```bash
grep -c '"game_system"' bgg_eval_viewer.html
grep -c 'boardgamegeek' bgg_eval_viewer.html
```

First command: should print `1` (the data is one JS array literal, `game_system` appears once per EVAL_DATA key — but the key is in the header, so it appears 865 times in the JSON). Actually let's check differently:

```bash
python3 -c "
import json, re
with open('bgg_eval_viewer.html') as f:
    html = f.read()
m = re.search(r'const EVAL_DATA\s*=\s*(\[.*?\]);', html, re.DOTALL)
data = json.loads(m.group(1))
print('EVAL_DATA rows:', len(data))
m2 = re.search(r'const EVENTS_DATA\s*=\s*(\{.*?\});', html, re.DOTALL)
events = json.loads(m2.group(1))
print('EVENTS_DATA combos:', len(events))
"
```

Expected:
```
EVAL_DATA rows: 865
EVENTS_DATA combos: 865
```

- [ ] **Step 4: Commit**

```bash
git add cmd/evalbgg/embed_data.py
git commit -m "docs(evalbgg): add usage comment to embed script"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Covered by |
|-----------------|-----------|
| Standalone HTML, no external deps | Task 2 template — no CDN links, pure vanilla |
| EVAL_DATA embedded as JS const | Task 1 embed script + Task 2 template |
| EVENTS_DATA embedded, indexed by `system\|\|edition` | Task 1 `read_events_index()` |
| Two-column layout (sidebar + main) | Task 2 CSS `.app` grid |
| Sidebar: scored progress + bar | Task 2 `updateScoredProgress()` |
| Sidebar: agreement distribution chart | Task 2 `renderAgreeChart()` |
| Sidebar: match rate bars | Task 2 `renderMatchRateBars()` |
| Toolbar: search, filter dropdown, row count | Task 2 toolbar HTML + `applyFilters()` |
| Table: sortable, color-coded pills | Task 2 `sortBy()`, `pillClass()` |
| Consensus cell: name + BGG link | Task 2 `renderTable()` consensus cell |
| Row expansion (one at a time) | Task 2 `toggleExpand()` |
| Tab 1: 18 matcher cards | Task 2 `buildMatcherGrid()` |
| Matcher cards: BGG link to `boardgamegeek.com/boardgame/{id}` | Task 2 `bggLink()` |
| Consensus card highlighted (purple) | Task 2 `.mc.consensus` CSS + `isCon` logic |
| "Use this ID" button | Task 2 `useId()` |
| Score input with auto-save | Task 2 `onScoreInput()` + `saveScore()` |
| localStorage persistence | Task 2 `loadScores()` / `saveScore()` |
| Clear button | Task 2 `clearScore()` |
| Tab 2: Gen Con events table | Task 2 `buildEventsTable()` |
| Events: all 20 BGM fields | Task 1 `BGM_FIELDS` list |
| Events: Tournament/Experience badges | Task 2 badge logic in `buildEventsTable()` |
| Export: download `bgg_eval_scored.csv` | Task 2 `exportCSV()` |
| Export: merges localStorage scores | Task 2 `exportCSV()` reads `scores` |
| `bgg_eval_viewer.html` gitignored | Task 1 `.gitignore` |
| Python embed script | Task 1 |
| Template at `cmd/evalbgg/viewer_template.html` | Task 2 |

All spec requirements covered. No placeholders or gaps found.

**Placeholder scan:** No TBD/TODO in either task. All code blocks are complete and self-contained. ✓

**Type consistency:** `rowKey()` returns `system + '||' + edition` in JS, matching the Python `key` format in `read_events_index()`. `MATCHERS` array matches exact column prefixes in `bgg_eval.csv`. `scores` object structure matches `saveScore`/`loadScores`. ✓
