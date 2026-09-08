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

- Next.js 15.5+ App Router, React 19, strict TypeScript (`noImplicitAny: true`).
- Follow the [Frontend Design and Engineering Guidelines](FRONTEND_GUIDELINES.md) for task-based UX, semantic tokens, reusable patterns, accessibility and review gates. Rules apply to frontend work regardless of editor or AI provider.
- Use the [screen specification template](design/SCREEN_SPEC_TEMPLATE.md) for new flows and the [Phase 3 design plan](FRONTEND_PHASE3_DESIGN.md) for dependencies. Architectural decisions follow [docs/adr/](adr/).
- UI components built with **shadcn/ui** and **Tailwind CSS**.
- Linting, type verification, and production build checks:
  ```bash
  cd frontend
  npm run lint
  npx tsc --noEmit
  npm run build
  ```

---

## 4. GitHub Project Management & Workflow

To maintain a professional, organized, and auditable repository, we strictly use GitHub Milestones, Issues, and Scoped Labels.

### 4.1 Custom Labels (Scoped Labels)
We use scoped labels to clearly define the domain, type, and priority of an issue. When creating issues, apply at least one label from each category.

**Domain Labels (Modules):**
- `domain: pos` (Color: `#1D76DB` - Blue)
- `domain: supply-chain` (Color: `#0E8A16` - Dark Green)
- `domain: sharia-ledger` (Color: `#FBCA04` - Gold)
- `domain: ai-auditor` (Color: `#5319E7` - Purple)
- `domain: infrastructure` (Color: `#5C6F7B` - Gray)

**Type & Priority Labels:**
- `type: bug` (Color: `#D73A4A` - Red)
- `type: feature` (Color: `#A2EEEF` - Cyan/Mint)
- `type: tech-debt` (Color: `#F9D0C4` - Pale Pink)
- `type: docs` (Color: `#C5DEF5` - Light Blue)
- `priority: critical` (Color: `#B60205` - Dark Red)
- `priority: high` (Color: `#D93F0B` - Orange)

**Compliance Labels (Crucial):**
- `compliance: halal` (Color: `#176332` - Forest Green)
- `compliance: sharia` (Color: `#D4C500` - Dark Gold)

### 4.2 Standard Task Workflow
Follow this exact sequence when contributing:

1. **Check Milestones:** View the current active Milestone (e.g., `Phase 1: Foundation`).
2. **Select/Create Issue:** Pick an unassigned issue from the Milestone, or create a new one using the standard Issue Templates (`.github/ISSUE_TEMPLATE`).
3. **Assign Labels:** Apply the relevant `domain:`, `type:`, and `priority:` labels to the issue.
4. **Create Branch:** Create a branch strictly following the naming convention (e.g., `feature/12-setup-postgres-schema`).
5. **Commit Code:** Use Conventional Commits (e.g., `feat(infrastructure): setup postgres connection pool`).
6. **Open Pull Request:** Use the provided PR template, link the issue (`Fixes #12`), and request a review.

---

## 5. Pull Request Checklist

Before submitting a Pull Request, ensure the following criteria are satisfied:

1. [ ] **Unit & Integration Tests:** All new and existing tests pass locally (`go test ./...`, `pytest`).
2. [ ] **Linters & Formatting:** Zero lint warnings across Go, Python, and TypeScript codebases.
3. [ ] **Idempotency & Concurrency:** Any new mutating API endpoint implements `Idempotency-Key` handling.
4. [ ] **Sharia Verification:** Changes affecting inventory, procurement, or ledger conform to Halal validity and double-entry invariants.
5. [ ] **Documentation:** Relevant markdown documents in `/docs` updated to reflect API or schema alterations.

---

*Tenet Commerce — Open Source Engineering Guidelines v1.0.0*
