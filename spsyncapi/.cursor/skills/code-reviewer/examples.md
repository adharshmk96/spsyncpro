# Code review examples

## Example 1: Auth handler change

**Objective**: Add refresh-token rotation on `/auth/refresh`.

**Completeness**: Partial — rotation updates DB but old token is not invalidated on concurrent refresh.

**Edge cases**

| Case | Status | Notes |
|------|--------|-------|
| Expired refresh token | covered | `TestRefresh_Expired` |
| Reused refresh token after rotation | gap | No test; race allows two valid sessions |
| Missing `Authorization` header | covered | 401 in handler |

**Test coverage**

| Path | Test | Status |
|------|------|--------|
| Happy path refresh | `TestRefresh_OK` | covered |
| Invalid signature | `TestRefresh_InvalidJWT` | covered |
| DB error on save | — | missing |

**Security**: **High** — reused refresh token not rejected (session fixation risk). Request changes.

---

## Example 2: Small refactor (no behavior change)

**Objective**: Extract validation from handler into service (no API change).

**Completeness**: Met — call sites unchanged, behavior preserved.

**Edge cases**: N/A for new behavior; existing tests should still pass — confirm `go test ./...`.

**Test coverage**: No new branches; existing `TestCreateUser_*` still apply. **covered** (by existing suite).

**Security**: No new attack surface.

**Verdict**: Approve with nits — run full package tests before merge.
