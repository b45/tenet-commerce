# ADR 007: Public claims require reproducible evidence

Status: publication policy.

Public architecture describes source behavior with code and test references, candidate identity and limitations. Local agent handoffs remain private. Publishing raw internal logs would obscure provenance and can disclose data; publishing only marketing claims would prevent verification. Instead, retain concise diagrams, decisions, sanitized synthetic examples and reproducible commands.

Historical plans remain in Git history and [roadmap](../ROADMAP.md), with future features clearly identified. Offline cash, QRIS settlement, AI and production availability must not be inferred from draft storage, configuration endpoints or container definitions. [Evidence map](../ARCHITECTURE_EVIDENCE.md) lists the current boundary.

Update evidence when code changes. An application test pass is not a Docker-image test, a laptop benchmark is not an SLA, and a merged PR does not satisfy unrelated acceptance criteria. Release publication requires verification tied to the release candidate.
