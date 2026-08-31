# Changelog

All notable changes to SOUL will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.6] - 2026-04-24

### Changed
- Maintenance release — version alignment with MIRA 0.4.6/0.4.7 integration cycle.

## [0.0.5] - 2026-04-24

### Changed
- Maintenance release — internal cleanup and dependency alignment.

## [0.0.4] - 2026-04-24

### Added
- **Unified Embedded Configuration**: `NewApplicationWithDBAndConfig` allows MIRA to pass a full `SoulConfig` when embedding SOUL. Embedded mode now supports the same tuning options as standalone mode: drift threshold, recall budget, extraction confidence, model-swap reinforcement, and evolution history tuning.
- **Public API Expansion**: Exported `soul.Config` struct and `soul.DefaultConfig()` function as aliases for external modules that embed SOUL as a library.
- **Prepublish Script**: Added `scripts/prepublish.sh` for automated version bump, build, test, and benchmark workflow.

## [0.0.3] - 2026-04-17

### Added
- Initial stable release of SOUL.
- MCP server with 8 `soul_*` tools: `soul_capture_identity`, `soul_recall_identity`, `soul_detect_drift`, `soul_swap_model`, `soul_get_evolution`, `soul_merge_identities`, `soul_update_identity`, `soul_enrich_with_mira_memories`.
- Identity capture and versioned storage with agent/model context.
- Drift detection across 6 dimensions: vocabulary, tone, values, style, confidence, consistency.
- Model swap handling with identity preservation across LLM transitions.
- Evolution tracking with full history and summarization support.
- Identity merge for combining profiles from multiple agents or sessions.
- Cross-access to MIRA's `verbatim` table for memory-enriched identity context.
- Hexagonal architecture with clean separation: domain, use cases, ports, adapters, interfaces.
- SQLite storage sharing MIRA's database (adds `soul_*` tables).
- Standalone mode via stdio JSON-RPC, or embedded in MIRA process.

---

[0.0.6]: https://github.com/benoitpetit/soul/releases/tag/v0.0.6
[0.0.5]: https://github.com/benoitpetit/soul/releases/tag/v0.0.5
[0.0.4]: https://github.com/benoitpetit/soul/releases/tag/v0.0.4
[0.0.3]: https://github.com/benoitpetit/soul/releases/tag/v0.0.3
