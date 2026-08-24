# ClickUp Terraform Provider — Project Notes

## Local development commands

This project uses `mise`. The common tasks are defined in `mise.toml`:

- Build: `mise run build` (or `go build -v ./...`)
- Unit tests: `mise run test` (or `go test -v -cover -timeout=120s -parallel=10 ./...`)
- Acceptance tests: `mise run testacc` (requires `TF_ACC=1`)
- Lint: `mise run lint` (or `golangci-lint run ./...`)
- Format: `mise run fmt` (or `gofmt -s -w -e .`)
- Generate: `mise run generate`

## Conventional Commits and PR workflow

- Use Conventional Commits with a scope: `feat(scope):`, `fix(scope):`, `test(scope):`, `refactor(scope):`, `chore(scope):`, `docs(scope):`.
- Keep each commit to one logical, reviewable change.
- Stack PRs on top of each other for large features:
  - Branch `pr-1-...` from `main`.
  - Branch `pr-2-...` from `pr-1-...`, etc.
  - Open each PR against its base branch. Merge from the bottom up.
- Rebase the next branch after its parent merges. Use force-with-lease only on feature branches, never on `main`.
- Do not rewrite shared history without explicit confirmation.

## E2E testing

- Default e2e target is a mock ClickUp HTTP server using `net/http/httptest` and `terraform-plugin-testing`.
- Run: `TF_ACC=1 go test ./...`
- Keep a parallel live-ClickUp acceptance suite for spec-vs-reality checks. Use a dedicated test workspace, unique resource names, and cleanup.
- Add an acceptance test for every new resource/data source before the implementation.

## Roadmap to full provider

1. **PR-1: Fix broken Phase 2 resources + e2e harness**
   - Add `terraform-plugin-testing`, `AGENTS.md`, and a mock `httptest` server.
   - Acceptance tests for existing hand-written and one generated resource.
   - Fix `goal` read response wrapping and `owners` mapping, `user_group` create `members` handling, `view.filters.fields` shape, `space_tag` update color keys, and any other spec-vs-reality issues.
2. **PR-2: Evaluate and integrate a generated Go client**
   - Generate `internal/clickupapi` from a cleaned OpenAPI 3.1 spec (`ClickUp_PUBLIC_API_V2.prepared.json`) using `oapi-codegen` or `ogen`. Spike both; choose the one that needs the least hand-patching.
   - Add client unit tests.
   - Refactor `clickupclient.Client` into a thin wrapper around the generated client (or keep the existing interface and delegate).
   - **Spike findings (2026-08-24):** oapi-codegen v1.16.3 fails on OpenAPI 3.1
     `null` type (`unhandled Schema type: null`). ogen v1.24.0 fails on the raw
     spec (`multiple types not supported: [string, number]`) and only works on
     the prepared spec with `ignore_not_implemented: ["all"]`, producing 252K
     lines across 21 files with 138 of 413 operations skipped and critical
     schemas (views, custom_fields values) falling back to `any`. Neither
     generator eliminates `prepare_openapi_spec.py` — both still need it for
     type-array/null/path-local-`$ref` normalization (the V2 spec has 0
     component schemas and 79 path-local `$ref`s). ogen adds heavy deps
     (opentelemetry, go-faster/*). Conclusion: a generated typed client is not
     a win for the current dynamic/generic architecture (126-line
     `clickupclient` returning `[]byte` + `genericResource` decoding to
     `map[string]any`). It only pays off as part of a full rewrite to typed
     per-resource implementations (PR-2+PR-3 combined). If proceeding, ogen is
     the only viable option; oapi-codegen would need its own 3.1 patching
     layer on top of the existing spec prep.
3. **PR-3: Complete remaining resources and data sources**
   - Use `tools/audit_coverage.py` to drive coverage of the 57 missing endpoints.
   - Add generic and hand-written resources, with acceptance tests for each.
4. **PR-4: Documentation, CI, and full e2e validation**
   - Run `tfplugindocs generate`, add examples.
   - GitHub Actions: build, unit tests, lint, mock acceptance tests, and optional live-ClickUp job.
   - Final full `TF_ACC` run.

## Notes

- The ClickUp OpenAPI spec is `3.1.0`, not "v2". Any client generator must support OpenAPI 3.1 or be preceded by a conversion step.
- The generated client is a low-level HTTP package only. It does not replace Terraform-specific response mapping and schema work.
- `tools/audit_coverage.py <ClickUp_PUBLIC_API_V2.prepared.json>` is the source of truth for endpoint coverage.
