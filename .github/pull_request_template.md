## Related issue / Change request
<!-- Specify the issue number: Closes #XX or a link to the discussion this PR addresses. If no issue exists, briefly describe the problem you are solving. -->

## Type of change
<!-- Put an 'x' in the boxes that apply. -->
- [ ] ✨ New feature (non-breaking change)
- [ ] 🐛 Bug fix
- [ ] 🏗️ Refactoring
- [ ] 📝 Documentation or comments
- [ ] 🤖 CI / Build configuration

## What this change does
<!-- Describe *what* is changing and, most importantly, *why* you chose this approach. For financial operations, explaining the reasoning is critical. -->

## Financial & logical validations
<!-- ⚠️ This is the most important section for CapitalFlow! Ensure your changes do not break financial logic. -->
- [ ] Calculation accuracy is preserved (especially for money operations).
- [ ] Edge cases are tested (e.g., negative balances or transaction cancellations).
- [ ] Unit tests are written for new or modified business rules.

## How to test
<!-- Provide step-by-step instructions to verify your work. -->
1. Run migrations: ...
2. Call the API endpoint: ...
3. Expected result: ...

## Additional Go‑project checks
- [ ] Code is formatted (`gofmt` or `goimports`).
- [ ] No unnecessary dependencies (check `go.mod` / `go.sum`).
- [ ] Unit tests are written/updated; `go test ./...` passes.
- [ ] No resource leaks (memory, file descriptors, goroutines).
- [ ] Logs are added with the appropriate level (Debug, Info, Error).

## Security
- [ ] Changes do not introduce new attack vectors (SQL injection, XSS, data leaks).
- [ ] Sensitive data (passwords, tokens, keys) does not appear in logs or API responses.

## Screenshots (for WebUI changes)
<!-- If your PR affects the web interface, add screenshots or GIFs. -->

## Author's checklist
- [ ] I have read [CONTRIBUTING.md](https://github.com/Sunriseex/CapitalFlow/blob/master/docs/CONTRIBUTING.md) (if it exists).
- [ ] I have run all checks locally: linters, tests, build.
- [ ] I have added an entry to `CHANGELOG.md` (if changes are user‑visible).

## Additional context
<!-- Any other information that helps understand the changes: links to external resources, complex architectural decisions, etc. -->