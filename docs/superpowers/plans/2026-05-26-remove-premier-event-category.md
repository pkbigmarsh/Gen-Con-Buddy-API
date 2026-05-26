# Remove Premier Event from SpecialCategory Enum — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove `Premier Event` from the `Category` enum — its constant, validation, search term mapping, tests, and documentation — as if it never existed.

**Architecture:** Three-file change: delete the `Premier` constant and its switch cases from `enums.go`, update one test case in `search_test.go`, and update the domain docs in `CONTEXT.md`. TDD order: write a failing enum test first, then make it pass by removing the constant, then fix the now-broken search test, then update docs.

**Tech Stack:** Go 1.22+, `github.com/stretchr/testify/require`

---

### Task 1: Write a failing test for `ValidateCategory`

**Files:**
- Create: `internal/event/enums_test.go`

There is no existing `enums_test.go`. Create it now with a test that asserts `ValidateCategory("Premier Event")` returns an error. This test must **fail** with the current code because `Premier` is still a valid enum value.

- [ ] **Step 1: Create the test file**

```go
package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCategory_RejectsPremierEvent(t *testing.T) {
	err := ValidateCategory("Premier Event")
	require.Error(t, err, "Premier Event is no longer a valid SpecialCategory")
}
```

- [ ] **Step 2: Run the test and confirm it fails**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
go test ./internal/event/... -run TestValidateCategory_RejectsPremierEvent -v
```

Expected output:
```
--- FAIL: TestValidateCategory_RejectsPremierEvent
    enums_test.go:12: Premier Event is no longer a valid SpecialCategory
FAIL
```

---

### Task 2: Remove `Premier` from `enums.go`

**Files:**
- Modify: `internal/event/enums.go`

Delete the `Premier` constant, remove it from the `ValidateCategory` switch, and remove the `"premier"` case from `CategoryFromSearchTerm`.

- [ ] **Step 1: Delete the `Premier` constant**

In `internal/event/enums.go`, change the `Category` const block from:

```go
const (
	No       Category = "none"
	Official Category = "Gen Con presents"
	Premier  Category = "Premier Event"
)
```

to:

```go
const (
	No       Category = "none"
	Official Category = "Gen Con presents"
)
```

- [ ] **Step 2: Remove `Premier` from `ValidateCategory`**

Change:

```go
func ValidateCategory(v string) error {
	switch Category(v) {
	case No, Official, Premier:
		return nil
	default:
		return fmt.Errorf("invalid value for Category: %s", v)
	}
}
```

to:

```go
func ValidateCategory(v string) error {
	switch Category(v) {
	case No, Official:
		return nil
	default:
		return fmt.Errorf("invalid value for Category: %s", v)
	}
}
```

- [ ] **Step 3: Remove the `"premier"` case from `CategoryFromSearchTerm`**

Change:

```go
func CategoryFromSearchTerm(s string) Category {
	switch s {
	case "none":
		return No
	case "official":
		return Official
	case "premier":
		return Premier
	default:
		return Category("invalid")
	}
}
```

to:

```go
func CategoryFromSearchTerm(s string) Category {
	switch s {
	case "none":
		return No
	case "official":
		return Official
	default:
		return Category("invalid")
	}
}
```

- [ ] **Step 4: Run the new enum test — it should now pass**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
go test ./internal/event/... -run TestValidateCategory_RejectsPremierEvent -v
```

Expected output:
```
--- PASS: TestValidateCategory_RejectsPremierEvent
PASS
```

- [ ] **Step 5: Run the full test suite — expect one failure in search_test.go**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
go test ./internal/event/... -v 2>&1 | grep -E "FAIL|PASS|Error"
```

Expected: all pass except `"specialCategory multi value"` — that test still references `"premier"` and will now produce `"invalid"` instead of `"Premier Event"`. Fix it in the next task.

---

### Task 3: Update the broken search test

**Files:**
- Modify: `internal/event/search_test.go`

The `"specialCategory multi value"` test case currently passes `"official,premier"` and expects `["Gen Con presents", "Premier Event"]`. Replace it with `"none,official"` to cover the multi-value path using the two remaining valid values.

- [ ] **Step 1: Update the test case**

In `internal/event/search_test.go`, change:

```go
{
    name:  "specialCategory multi value",
    field: "specialCategory",
    value: "official,premier",
    wantQuery: map[string]any{
        "terms": map[string]any{"specialCategory": []string{"Gen Con presents", "Premier Event"}},
    },
},
```

to:

```go
{
    name:  "specialCategory multi value",
    field: "specialCategory",
    value: "none,official",
    wantQuery: map[string]any{
        "terms": map[string]any{"specialCategory": []string{"none", "Gen Con presents"}},
    },
},
```

- [ ] **Step 2: Run the full test suite — all should pass**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
go test ./... 2>&1
```

Expected:
```
Go test: 94 passed in 12 packages
```

(or one more if the new enums_test.go adds a test)

- [ ] **Step 3: Commit**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
git add internal/event/enums.go internal/event/enums_test.go internal/event/search_test.go
git commit -m "feat: remove Premier Event from SpecialCategory enum

Premier Event no longer appears in Gen Con's source data.
Remove the constant, validation case, and search term mapping.

Closes #51"
```

---

### Task 4: Update CONTEXT.md

**Files:**
- Modify: `CONTEXT.md`

Two updates:
1. The `SpecialCategory` definition on line 27 references `Premier Event` — remove it.
2. The open question on line 94 asks about the distinction between `Gen Con Presents` and `Premier Event` — remove it (the question is moot now that only one value exists).

- [ ] **Step 1: Update the SpecialCategory definition**

Change line 27 from:

```
A Gen Con-assigned classification (`Gen Con Presents` or `Premier Event`). Meaning is defined by Gen Con; treat as an opaque label from the source data.
```

to:

```
A Gen Con-assigned classification. The only value currently in use is `Gen Con Presents`. Meaning is defined by Gen Con; treat as an opaque label from the source data.
```

- [ ] **Step 2: Remove the open question about Premier Event**

Remove this line from the `## Open questions` section (line 94):

```
- **SpecialCategory**: What do "Gen Con Presents" and "Premier Event" mean to attendees in practice? What distinguishes the two?
```

If removing it leaves the `## Open questions` section empty (no other bullets), remove the entire section header and its preamble paragraph as well. Check the section after removal — if it only contains the header and the "These came up during domain modelling…" sentence with no bullets, delete both.

- [ ] **Step 3: Commit**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
git add CONTEXT.md
git commit -m "docs(context): remove Premier Event from SpecialCategory definition

The value no longer exists in Gen Con's source data.
Closes the open question that asked about its meaning."
```

---

### Task 5: Open the PR

- [ ] **Step 1: Push the branch**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
git push -u origin HEAD
```

- [ ] **Step 2: Create the PR**

```bash
gh pr create \
  --repo pkbigmarsh/Gen-Con-Buddy-API \
  --title "Remove Premier Event from SpecialCategory enum" \
  --body "$(cat <<'EOF'
## Summary

- Removes the `Premier` constant, its `ValidateCategory` case, and its `CategoryFromSearchTerm` mapping from `internal/event/enums.go`
- Updates the `specialCategory multi value` search test to use `none,official` instead of `official,premier`
- Adds `enums_test.go` with a direct test for `ValidateCategory` rejecting the removed value
- Updates `CONTEXT.md` to reflect that `Premier Event` no longer exists in source data

Closes #51

## Test plan

- [ ] `go test ./...` passes with no failures
- [ ] `TestValidateCategory_RejectsPremierEvent` passes (new test, previously would have failed)
- [ ] `specialCategory multi value` search test updated and passing

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
