<div align="center">

# Tenet Commerce
### Multi-Tenant Enterprise POS & Halal Supply Chain Platform

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.12+-3776AB?style=flat&logo=python)](https://www.python.org/)
[![Next.js](https://img.shields.io/badge/Next.js-14+-000000?style=flat&logo=next.js)](https://nextjs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io/)
[![Architecture](https://img.shields.io/badge/Architecture-Modular_Monolith-success)](docs/ARCHITECTURE.md)
[![Compliance](https://img.shields.io/badge/Sharia-AAOIFI_Compliant-gold)](docs/SHARIA_COMPLIANCE.md)

<p align="center">
  <b>A cloud-native, offline-first enterprise retail engine converging multi-tenant POS, automated Halal certificate verification, double-entry Sharia ledgering, and continuous AI auditing.</b>
</p>

[Documentation](#-documentation-index) • [Key Features](#-key-features) • [System Architecture](#-system-architecture) • [Tech Stack](#-technology-stack) • [Quick Start](#-quick-start) • [Roadmap](#-roadmap)

---

</div>

## 📌 Overview

**Tenet Commerce** is an enterprise-grade retail platform engineered for high-throughput retail operations subject to strict **Sharia compliance and Halal regulatory standards**. 

Traditional enterprise POS systems treat supply chain compliance and financial ledgering as disconnected afterthoughts. Tenet Commerce natively bridges this gap through four tightly integrated core domains:

1. **High-Velocity Offline-First POS:** Design goal of sub-300ms checkout latency (not yet benchmarked) with client-side IndexedDB caching and background Service Worker synchronization.
2. **Deterministic Transaction Engine:** Redis-backed idempotency key verification (preventing double charges) coupled with dual-layer distributed inventory locking.
3. **Halal Supply Chain Governance:** Automated hard-validation on Halal Certificate validity across Suppliers, Purchase Orders, Goods Receipts, and Stock Transfers.
4. **Sharia Financial Ledger & Applied AI:** Real-time double-entry journal, dynamic Zakat Tijarah (Trade Zakat) calculation, and an asynchronous AI Reasoning Engine executing weekly continuous audits for financial anomalies.

---

## 🏛️ System Architecture

Tenet Commerce employs a **Modular Monolith** architecture with **Schema-per-Tenant Logical Isolation** in PostgreSQL. This ensures zero data leakage across enterprise tenants while maintaining minimal operational overhead.

```mermaid
flowchart TB
    subgraph CLIENT["🖥️ Frontend Client (Next.js 14 + shadcn/ui)"]
        POS["POS Cashier UI"]
        SW["Service Worker<br/>(Background Sync)"]
        IDB[("IndexedDB<br/>Offline Queue")]
        POS <--> SW
        SW <--> IDB
    end

    subgraph INGRESS["🔐 API Gateway Layer"]
        AUTH["JWT Authenticator"]
        RBAC["RBAC Guard"]
        TENANT["Tenant Resolver"]
        IDEMP["Idempotency Filter"]
        AUTH --> RBAC --> TENANT --> IDEMP
    end

    subgraph REDIS["⚡ Redis Cluster"]
        IK[("Idempotency Store<br/>TTL: 24h")]
        LOCK[("Distributed Locks<br/>Redlock")]
    end

    subgraph MONOLITH["⚙️ Core Engine (Go 1.22 Modular Monolith)"]
        POS_MOD["POS & Checkout Module"]
        SC_MOD["Halal Supply Chain Module"]
        LEDGER_MOD["Double-Entry Ledger Module"]
        ZAKAT_CALC["Zakat Tijarah Engine"]
        
        POS_MOD --> LEDGER_MOD
        SC_MOD --> LEDGER_MOD
        LEDGER_MOD --> ZAKAT_CALC
    end

    subgraph DB["🗄️ PostgreSQL 16 (Schema-per-Tenant)"]
        PUBLIC[("public schema<br/>(Tenant Registry & Global Config)")]
        T1[("tenant_corp_a<br/>(Products, Orders, Ledger)")]
        T2[("tenant_corp_b<br/>(Products, Orders, Ledger)")]
    end

    subgraph AI["🤖 Continuous Sharia Auditor (Python 3.12 Worker)"]
        AUDIT_CRON["Weekly Cron Scheduler"]
        ANOMALY["Statistical Anomaly Detector"]
        SHARIA_RULES["Sharia Rule Reasoner"]
        
        AUDIT_CRON --> ANOMALY --> SHARIA_RULES
    end

    %% Flow Connections
    POS -- "HTTPS POST + Idempotency-Key" --> AUTH
    SW -- "Replay Queue" --> AUTH
    IDEMP <--> IK
    POS_MOD <--> LOCK
    
    TENANT -. "Search Path Routing" .-> DB
    POS_MOD --> T1
    SC_MOD --> T1
    LEDGER_MOD --> T1

    AI -- "Read-Only Analysis" --> T1
    AI -- "Read-Only Analysis" --> T2
    SHARIA_RULES -- "Persist Audit Reports" --> T1
```

---

## ✨ Key Features

| Domain | Capabilities |
|---|---|
| **POS & Checkout** | • Sub-second barcode/SKU lookup<br/>• Offline-first cart persistence and receipt generation<br/>• Background sync with deterministic transaction replay |
| **Transaction Engine** | • Guaranteed idempotency via `Idempotency-Key` headers<br/>• Redis distributed lock on SKU mutations<br/>• PostgreSQL `SELECT ... FOR UPDATE` row-level locking |
| **Halal Supply Chain** | • Supplier Halal Certificate tracking (MUI, BPJPH, JAKIM, etc.)<br/>• **Hard-block validation:** prevents PO, Goods Receipt, or transfers from expired certs<br/>• Complete traceability from PO to store shelf |
| **Sharia Ledger** | • Automated balanced double-entry journal for all sales and receipts<br/>• Real-time Trial Balance and Chart of Accounts<br/>• Dynamic **Zakat Tijarah** (2.5% Net Working Capital) calculation |
| **Continuous AI Auditor** | • Asynchronous Python worker running scheduled weekly audits<br/>• Heuristic anomaly detection (Benford's law, off-hours sales, round-number clustering)<br/>• Categorized severity alerts (`INFO`, `WARNING`, `CRITICAL`) |

---

## 💻 Technology Stack

```
tenet-commerce
├── Backend API      : Go 1.22+ (Chi / Gin Router, pgx, SQLC)
├── AI Worker        : Python 3.12+ (Pydantic, Polars, Scikit-learn)
├── Frontend UI      : Next.js 14 (App Router, TypeScript, Tailwind CSS, shadcn/ui)
├── Database         : PostgreSQL 16 (Schema-per-Tenant Isolation)
├── Cache & Locks    : Redis 7 (Idempotency Key Store & Distributed Locks)
├── Containerization : Docker & Docker Compose
└── CI/CD            : GitHub Actions (Lint, Test, Security Scan, Multi-arch Build)
```

---

## 📚 Documentation Index

Our documentation is structured for enterprise architects, security auditors, and developers:

- 📋 [**Product Requirements Document (PRD)**](docs/PRD.md): Executive summary, user stories, Gherkin acceptance criteria, and NFRs.
- 🏗️ [**Technical Architecture Deep-Dive**](docs/ARCHITECTURE.md): Multi-tenancy isolation, distributed locking, idempotency mechanics, and offline sync lifecycle.
- 🗄️ [**Database Schema & DDL Specification**](docs/DATABASE_SCHEMA.md): Complete PostgreSQL DDL, schema-per-tenant migration strategy, and double-entry ledger models.
- 🔌 [**REST API Specification**](docs/API_SPECIFICATION.md): Comprehensive endpoint contracts, request/response payloads, headers, and error dictionary.
- ⚖️ [**Sharia Compliance & AI Auditor**](docs/SHARIA_COMPLIANCE.md): Halal certification governance, Zakat Tijarah formula, and continuous audit heuristics.
- 🗺️ [**8-Week Implementation Roadmap**](docs/ROADMAP.md): Detailed sprint breakdowns, milestone deliverables, and risk mitigation strategies.
- 🤝 [**Contributing Guidelines**](docs/CONTRIBUTING.md): Engineering standards, branching models, conventional commits, and local setup.

---

## 🚀 Quick Start

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) (v24.0+)
- [Docker Compose](https://docs.docker.com/compose/) (v2.20+)
- [Go](https://go.dev/doc/install) (v1.22+) *(optional for local binary build)*
- [Node.js](https://nodejs.org/) (v20 LTS) *(optional for frontend development)*

### 1. Clone the Repository
```bash
git clone https://github.com/your-org/tenet-commerce.git
cd tenet-commerce
```

### 2. Configure Environment
```bash
cp .env.example .env
```

### 3. Launch Local Development Stack
```bash
docker compose up -d --build
```

### 4. Verify Services
- **Backend API Gateway:** [http://localhost:8080/healthz](http://localhost:8080/healthz)
- **POS Web Client:** [http://localhost:3000](http://localhost:3000)
- **PostgreSQL Database:** `localhost:5432` (`postgres` / `postgres`)
- **Redis Cache & Lock:** `localhost:6379`

---

## 🗺️ Roadmap & Sprint Progress

The MVP is structured as an **8-Week Fast-Track Sprint**:

- [x] **Phase 1: Foundation (Weeks 1-2):** Architecture, Multi-Tenant Postgres, JWT Auth & RBAC
- [ ] **Phase 2: Core APIs (Weeks 3-4):** POS Engine, Redis Idempotency, Halal Supply Chain & Ledger
- [ ] **Phase 3: Frontend & Offline-First (Weeks 5-6):** Next.js POS UI, IndexedDB Sync & Dashboards
- [ ] **Phase 4: AI Auditor & Production Readiness (Weeks 7-8):** Python Worker, Zakat Engine, E2E Testing

*See [docs/ROADMAP.md](docs/ROADMAP.md) for full phase details and milestone deliverables.*

---

## 🔒 Security & Data Privacy

- **Tenant Isolation:** Hard separation at the database layer via PostgreSQL schemas (`tenant_{uuid}`).
- **Data Protection:** All passwords hashed with `Argon2id`; JWT access tokens expire in 15 minutes.
- **Idempotent Safety:** Redis-guarded write operations guarantee zero double-charging or duplicate inventory decrements.

---

## 📄 License

This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.
