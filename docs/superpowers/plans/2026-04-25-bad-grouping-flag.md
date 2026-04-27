# Bad Grouping Flag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Flag as bad grouping" state to the eval viewer so rows with mismatched GenCon events can be marked and reviewed separately, distinct from both ungraded (blank) and graded (BGG ID) rows.

**Architecture:** Single-file HTML app (`bgg_eval_viewer.html`). The sentinel string `"BAD_GROUPING"` is stored in the `scores` object (localStorage) as the `correct_bgg_id` value. All five changes — CSS, sidebar, filter, table row, expand panel — are independent edits to that one file, done task by task.

**Tech Stack:** Vanilla HTML/CSS/JS, localStorage for persistence.

---

### Task 1: CSS — add bad-grouping styles

**Files:**
- Modify: `bgg_eval_viewer.html` (~line 155, inside `<style>` block, after `.save-ok` rule)

- [ ] **Step 1: Add CSS rules**

  Open `bgg_eval_viewer.html`. Find the `.save-ok` rule (around line 161):
  ```css
  .save-ok { font-size: 11px; color: var(--green); display: none; }
  ```
  Insert immediately after it:
  ```css
  .scored-bad  { color: var(--orange); }
  .flag-btn    { background: none; border: 1px solid var(--orange); color: var(--orange); border-radius: 4px; padding: 5px 10px; font-size: 11px; cursor: pointer; }
  .flag-btn:hover { background: #2a1a0a; }
  .bad-grouping-row { display: flex; align-items: center; gap: 10px; background: #1a1208; border-radius: 6px; padding: 10px; }
  .bad-grouping-label { font-size: 11px; color: var(--orange); font-weight: 600; }
  ```

- [ ] **Step 2: Verify in browser**

  Open `bgg_eval_viewer.html` in a browser. No visual change expected yet — just confirm the page still loads without console errors.

- [ ] **Step 3: Commit**
  ```bash
  git add bgg_eval_viewer.html
  git commit -m "style(eval): add bad-grouping CSS states"
  ```

---

### Task 2: Sidebar — add bad-grouping counter

**Files:**
- Modify: `bgg_eval_viewer.html` — sidebar HTML (~line 193) and `updateScoredProgress()` (~line 530)

- [ ] **Step 1: Add bad-grouping count to sidebar HTML**

  Find the scored stat block (~line 193–196):
  ```html
      <div class="stat-block">
        <div class="lbl">Scored</div>
        <span class="score-big" id="sb-scored">0</span><span class="score-denom"> / <span id="sb-total">—</span></span>
        <div class="prog-bg"><div class="prog-fill" id="sb-prog"></div></div>
      </div>
  ```
  Replace it with:
  ```html
      <div class="stat-block">
        <div class="lbl">Scored</div>
        <span class="score-big" id="sb-scored">0</span><span class="score-denom"> / <span id="sb-total">—</span></span>
        <div class="prog-bg"><div class="prog-fill" id="sb-prog"></div></div>
        <div style="margin-top:6px;font-size:11px;color:var(--muted)">
          <span id="sb-bad-count" style="color:var(--orange)">0</span> bad grouping
        </div>
      </div>
  ```

- [ ] **Step 2: Update `updateScoredProgress()` to compute bad count and exclude bad from scored**

  Find `updateScoredProgress()` (~line 530):
  ```javascript
  function updateScoredProgress() {
    const total  = EVAL_DATA.length;
    const scored = EVAL_DATA.filter(r => !!scores[rowKey(r)]).length;
    document.getElementById('sb-scored').textContent = scored;
    document.getElementById('sb-total').textContent  = total;
    document.getElementById('sb-prog').style.width   = (scored / total * 100).toFixed(1) + '%';
  }
  ```
  Replace with:
  ```javascript
  function updateScoredProgress() {
    const total  = EVAL_DATA.length;
    const bad    = EVAL_DATA.filter(r => scores[rowKey(r)] === 'BAD_GROUPING').length;
    const scored = EVAL_DATA.filter(r => !!scores[rowKey(r)] && scores[rowKey(r)] !== 'BAD_GROUPING').length;
    document.getElementById('sb-scored').textContent    = scored;
    document.getElementById('sb-total').textContent     = total;
    document.getElementById('sb-prog').style.width      = (scored / total * 100).toFixed(1) + '%';
    document.getElementById('sb-bad-count').textContent = bad;
  }
  ```

- [ ] **Step 3: Verify in browser**

  Reload the page. The sidebar should show "0 bad grouping" below the progress bar. Scored count should be unchanged (no rows flagged yet).

- [ ] **Step 4: Commit**
  ```bash
  git add bgg_eval_viewer.html
  git commit -m "feat(eval): add bad-grouping counter to sidebar"
  ```

---

### Task 3: Filter dropdown — add bad-grouping option and fix isScored logic

**Files:**
- Modify: `bgg_eval_viewer.html` — filter `<select>` (~line 219) and `applyFilters()` (~line 295)

- [ ] **Step 1: Add option to filter dropdown**

  Find the filter select options (~line 218–220):
  ```html
        <option value="unscored">Unscored only</option>
        <option value="scored">Scored only</option>
  ```
  Replace with:
  ```html
        <option value="unscored">Unscored only</option>
        <option value="scored">Scored only</option>
        <option value="bad-grouping">Bad grouping</option>
  ```

- [ ] **Step 2: Update `applyFilters()` to handle new mode and fix isScored/isBad**

  Find the filter predicate inside `applyFilters()` (~line 299–310):
  ```javascript
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
  ```
  Replace with:
  ```javascript
      .filter(({ row, i }) => {
        if (q && !row.game_system.toLowerCase().includes(q)) return false;
        const agree    = parseInt(row.agreement_count);
        const key      = rowKey(row);
        const isBad    = scores[key] === 'BAD_GROUPING';
        const isScored = !!scores[key] && !isBad;
        if (mode === 'lte7'         && agree > 7)  return false;
        if (mode === 'lte10'        && agree > 10) return false;
        if (mode === 'unscored'     && (isScored || isBad)) return false;
        if (mode === 'scored'       && !isScored)  return false;
        if (mode === 'bad-grouping' && !isBad)     return false;
        return true;
      });
  ```

- [ ] **Step 3: Verify in browser**

  Reload. The filter dropdown should show "Bad grouping" as an option. Selecting it should show 0 rows (none flagged yet). "Unscored only" should behave the same as before.

- [ ] **Step 4: Commit**
  ```bash
  git add bgg_eval_viewer.html
  git commit -m "feat(eval): add bad-grouping filter option"
  ```

---

### Task 4: Table row — third state rendering

**Files:**
- Modify: `bgg_eval_viewer.html` — `renderTable()` (~line 344)

- [ ] **Step 1: Update renderTable to show ⚠ for bad-grouping rows**

  Find the scored cell logic inside `renderTable()` (~line 344–365):
  ```javascript
    const key      = rowKey(row);
    const isScored = !!scores[key];
    const agree    = parseInt(row.agreement_count);
    const isOpen   = expandedIdx === i;
  ```
  Replace with:
  ```javascript
    const key      = rowKey(row);
    const isBad    = scores[key] === 'BAD_GROUPING';
    const isScored = !!scores[key] && !isBad;
    const agree    = parseInt(row.agreement_count);
    const isOpen   = expandedIdx === i;
  ```

  Then find the scored `<td>` inside `tr.innerHTML` (~line 365):
  ```javascript
        <td class="${isScored ? 'scored-yes' : 'scored-no'}">${isScored ? '✓' : '○'}</td>
  ```
  Replace with:
  ```javascript
        <td class="${isBad ? 'scored-bad' : isScored ? 'scored-yes' : 'scored-no'}">${isBad ? '⚠' : isScored ? '✓' : '○'}</td>
  ```

- [ ] **Step 2: Verify in browser**

  Reload. No rows show ⚠ yet (none flagged). Graded rows still show ✓, ungraded show ○.

- [ ] **Step 3: Commit**
  ```bash
  git add bgg_eval_viewer.html
  git commit -m "feat(eval): render bad-grouping state in table rows"
  ```

---

### Task 5: Expand panel — flag button, flagged state, and JS functions

**Files:**
- Modify: `bgg_eval_viewer.html` — `buildExpandPanel()` (~line 397), scoring functions (~line 470)

- [ ] **Step 1: Fix `savedId` and add `isBad` in `buildExpandPanel()`**

  Find the top of `buildExpandPanel()` (~line 397):
  ```javascript
  function buildExpandPanel(row, dataIdx) {
    const key = rowKey(row);
    const savedId = scores[key] || '';
  ```
  Replace with:
  ```javascript
  function buildExpandPanel(row, dataIdx) {
    const key     = rowKey(row);
    const isBad   = scores[key] === 'BAD_GROUPING';
    const savedId = isBad ? '' : (scores[key] || '');
  ```

- [ ] **Step 2: Replace the score-row with dual-state UI in `buildExpandPanel()`**

  Find the score-row block inside the returned template string (~line 418–424):
  ```html
      <div class="score-row">
        <label>correct_bgg_id</label>
        <input type="text" id="score-${dataIdx}" value="${esc(savedId)}" placeholder="BGG ID…"
               oninput="onScoreInput(${dataIdx}, '${escapedKey}', this.value)" />
        <button class="clear-btn" onclick="clearScore(${dataIdx}, '${escapedKey}')">clear</button>
        <span class="save-ok" id="save-ok-${dataIdx}">✓ saved</span>
      </div>
  ```
  Replace with:
  ```html
      <div class="score-row" id="score-input-${dataIdx}" ${isBad ? 'style="display:none"' : ''}>
        <label>correct_bgg_id</label>
        <input type="text" id="score-${dataIdx}" value="${esc(savedId)}" placeholder="BGG ID…"
               oninput="onScoreInput(${dataIdx}, '${escapedKey}', this.value)" />
        <button class="clear-btn" onclick="clearScore(${dataIdx}, '${escapedKey}')">clear</button>
        <span class="save-ok" id="save-ok-${dataIdx}">✓ saved</span>
        <button class="flag-btn" onclick="flagBadGrouping(${dataIdx}, '${escapedKey}')">⚠ Flag as bad grouping</button>
      </div>
      <div class="bad-grouping-row" id="bad-grouping-row-${dataIdx}" ${isBad ? '' : 'style="display:none"'}>
        <span class="bad-grouping-label">⚠ Flagged: bad grouping</span>
        <button class="clear-btn" onclick="clearFlag(${dataIdx}, '${escapedKey}')">clear flag</button>
      </div>
  ```

- [ ] **Step 3: Add `flagBadGrouping()` and `clearFlag()` functions**

  Find `clearScore()` (~line 487):
  ```javascript
  function clearScore(dataIdx, key) {
    document.getElementById('score-' + dataIdx).value = '';
    saveScore(key, '');
    document.getElementById('save-ok-' + dataIdx).style.display = 'none';
  }
  ```
  Add two new functions immediately after it:
  ```javascript
  function flagBadGrouping(dataIdx, key) {
    document.getElementById('score-' + dataIdx).value = '';
    document.getElementById('save-ok-' + dataIdx).style.display = 'none';
    saveScore(key, 'BAD_GROUPING');
    document.getElementById('score-input-' + dataIdx).style.display = 'none';
    document.getElementById('bad-grouping-row-' + dataIdx).style.display = '';
  }

  function clearFlag(dataIdx, key) {
    saveScore(key, '');
    document.getElementById('score-input-' + dataIdx).style.display = '';
    document.getElementById('bad-grouping-row-' + dataIdx).style.display = 'none';
  }
  ```

- [ ] **Step 4: Verify the full feature in browser**

  Reload. Expand any row:
  - The score-row should show an orange "⚠ Flag as bad grouping" button alongside the BGG ID input.
  - Click it: the input row hides, "⚠ Flagged: bad grouping" appears.
  - The table row for that entry now shows `⚠` in orange.
  - The sidebar shows "1 bad grouping".
  - The scored count does not include the flagged row.
  - Click "clear flag": reverts to input state.
  - Select "Bad grouping" from the filter: only the flagged row appears.
  - Reload the page: flag persists (stored in localStorage).

- [ ] **Step 5: Commit**
  ```bash
  git add bgg_eval_viewer.html
  git commit -m "feat(eval): add flag-as-bad-grouping to expand panel"
  ```
