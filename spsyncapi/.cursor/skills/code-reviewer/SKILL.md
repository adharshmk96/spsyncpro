---
name: code-reviewer
description: Reviews code changes for objective completeness, edge-case coverage, test coverage per code path, and security issues. Use when reviewing pull requests, diffs, commits, or when the user asks for a code review, security review, or test coverage check.
---

# Code Reviewer

Perform a structured review of the change set (PR, branch diff, or files the user specifies). Read the actual code and tests before concluding.

## Review workflow

1. **Scope** — Identify changed files, entry points, and dependencies (handlers, services, storage, config).
2. **Objective** — Summarize what the change is trying to accomplish in one or two sentences.
3. **Completeness** — Confirm the stated objective is fully implemented; flag missing pieces, dead code, or scope creep.
4. **Edge cases** — List inputs, states, and failure modes; mark each as handled, missing, or N/A.
5. **Tests** — Map branches/paths to tests; flag untested paths and suggest concrete cases.
6. **Security** — Run the security checklist below on touched surfaces.

## Required checks

The review must explicitly address:

- summarize the objective and validate completeness
- validate all edge cases are covered
- ensure test cases exists for all code path
- look for security vulnerabilities

## Completeness

| Question | Action if no |
|----------|----------------|
| Does behavior match the stated goal? | List gaps with file/line references |
| Are errors returned/handled consistently with the rest of the package? | Note inconsistency |
| Are config, migrations, and docs updated when behavior or API changes? | Call out omissions |
| Is logging appropriate (no secrets, useful context on failures)? | Suggest fixes |

## Edge cases

For each public function, handler, and critical branch:

- **Inputs**: empty, nil, zero, max length, invalid types, malformed encoding
- **AuthZ**: missing/invalid token, wrong tenant/user, expired credentials
- **Concurrency**: double submit, partial failure, idempotency
- **External deps**: timeout, 4xx/5xx, empty response, partial data
- **Persistence**: not found, duplicate key, transaction rollback

Mark each as **covered** (code + test), **handled untested**, or **gap**.

## Test coverage (all code paths)

1. Enumerate branches: `if`/`switch`/`return err`/early exits, loops, goroutines, defer paths.
2. For each path, cite an existing test or state **missing**.
3. Prefer table-driven tests for input variants; one test per error path when behavior differs.
4. Distinguish **unit** (logic, mocks) vs **integration** (DB, HTTP) and say what is still needed.

Do not claim full coverage without naming tests or explaining why a path is unreachable.

## Security checklist

Focus on what the diff touches:

| Area | Look for |
|------|----------|
| Input | Unvalidated query/body/headers; injection (SQL, command, template, path traversal) |
| AuthN/AuthZ | Missing checks on new routes; IDOR; privilege escalation; JWT/session misuse |
| Secrets | Hardcoded keys; logs or errors leaking tokens/passwords/PII |
| Crypto | Weak algorithms; static IVs; passwords not hashed with appropriate cost |
| HTTP | CSRF on state-changing cookie auth; open redirects; overly permissive CORS |
| Data | Mass assignment; sensitive fields in JSON responses; insecure deserialization |
| Resources | Unbounded payloads; missing rate limits; SSRF on outbound URLs |
| Dependencies | New risky packages; outdated libs with known CVEs (note if not verifiable locally) |

Severity: **Critical** (exploit before merge), **High**, **Medium**, **Low**, **Informational**.

## Output format

Use this structure in the review comment:

```markdown
## Objective
[What the change intends to do]

## Completeness
[Met / partial / not met — bullets with file references]

## Edge cases
| Case | Status | Notes |
|------|--------|-------|
| ... | covered / gap / N/A | ... |

## Test coverage
| Path / behavior | Test | Status |
|-----------------|------|--------|
| ... | `Test...` or — | covered / missing |

## Security
[Findings by severity, or "No issues identified" with brief rationale]

## Verdict
[Approve / approve with nits / request changes — one line why]

## Action items
- [ ] ...
```

## Principles

- Cite code with `startLine:endLine:filepath` when pointing at issues.
- Prefer specific, actionable feedback over generic advice.
- Separate **must fix** from **should consider**.
- If context is insufficient (no tests in diff, unclear product goal), state assumptions and what to verify.

## Additional resources

- Example reviews: [examples.md](examples.md)
