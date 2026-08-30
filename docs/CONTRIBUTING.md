# Contributing to Tenet Commerce
## Open Source Engineering & Development Guidelines

---

Thank you for your interest in contributing to **Tenet Commerce**. As a flagship enterprise platform combining cloud-native software architecture with strict Sharia compliance, we enforce rigorous engineering standards.

---

## 1. Development Principles & Code of Conduct

- **Transactional Determinism:** Every state-mutating operation must guarantee idempotency and ACID transactional safety.
- **Zero-Tolerance Compliance:** Sharia rules (Halal validity, double-entry ledger balance, Zakat mathematical formulas) are hard constraints, not optional configurations.
- **Explainable Code:** Clean, idiomatic code with explicit error handling. Avoid hidden side-effects or untyped data transformations.

---

## 2. Git Workflow & Branching Strategy

We follow the **GitFlow** branching model:

- `main`: Production-ready, stable releases.
- `develop`: Integration branch for active sprint features.
- `feature/<issue-number>-<short-description>`: Feature branches cut from `develop`.
- `fix/<issue-number>-<short-description>`: Bug fixes cut from `develop` or hotfixes from `main`.

### Conventional Commits
All commit messages must adhere to the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <short summary>

[optional body explaining rationale]

[optional footer(s) e.g. Fixes #123]
```

**Allowed Types:**
- `feat`: New feature or domain capability.
- `fix`: Bug fix in business logic, API, or UI.
- `docs`: Documentation updates or additions.
- `refactor`: Code restructuring without behavioral changes.
- `perf`: Performance optimization.
- `test`: Adding or updating test suites.
- `chore`: Build scripts, dependencies, or CI/CD workflow updates.

---

## 3. Engineering & Style Standards

### 3.1 Go (Backend Engine)
- Follow standard Go formatting via `gofmt` and `goimports`.
- All code must pass `golangci-lint` without warnings:
  ```bash
  golangci-lint run ./...
  ```
- Explicit error handling: Never ignore returned errors (`_ = fn()`).
- Use structured logging (e.g., `slog` or `zap`) with contextual fields (`tenant_id`, `request_id`, `idempotency_key`).

### 3.2 Python (AI Auditor Worker)
- Use **Python 3.12+** with explicit type annotations on all functions.
- Format and lint using `ruff`:
  ```bash
  ruff check ai-worker/
  ruff format ai-worker/
  ```
- Type verification using `mypy`:
  ```bash
  mypy --strict ai-worker/
  ```

### 3.3 TypeScript & Next.js (Frontend Client)
- Strict TypeScript (`noImplicitAny: true`).
- UI components built with **shadcn/ui** and **Tailwind CSS**.
- Linting and type verification:
  ```bash
  cd frontend
  npm run lint
  npx tsc --noEmit
  ```

---

## 4. Pull Request Checklist

Before submitting a Pull Request, ensure the following criteria are satisfied:

1. [ ] **Unit & Integration Tests:** All new and existing tests pass locally (`go test ./...`, `pytest`).
2. [ ] **Linters & Formatting:** Zero lint warnings across Go, Python, and TypeScript codebases.
3. [ ] **Idempotency & Concurrency:** Any new mutating API endpoint implements `Idempotency-Key` handling.
4. [ ] **Sharia Verification:** Changes affecting inventory, procurement, or ledger conform to Halal validity and double-entry invariants.
5. [ ] **Documentation:** Relevant markdown documents in `/docs` updated to reflect API or schema alterations.

---

*Tenet Commerce — Open Source Engineering Guidelines v1.0.0*
