<div align="center">
  <img src="./logo.png" alt="SOUL Logo" width="800">
  
# SOUL - System for Observed Unique Legacy

[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Version](https://img.shields.io/badge/Version-0.0.2-blue?style=flat-square)]()


[**SOUL**](https://github.com/benoitpetit/soul) is an **identity preservation extension for LLM agents**. It captures, stores, and recalls the personality, voice, and values of AI agents across sessions and model changes.

Where [**MIRA**](https://github.com/benoitpetit/mira) answers *"What does the agent know?"*, SOUL answers *"Who is the agent?"*
</div>

## Relationship with MIRA

| Aspect | Detail |
|--------|--------|
| **Dependency** | None - SOUL compiles and runs without MIRA |
| **Integration** | Can run standalone (separate MCP server) or embedded in MIRA (single binary, 16 tools) |
| **Database** | SOUL adds `soul_*` tables to MIRA's `.mira/mira.db` |
| **Cross-access** | SOUL can read MIRA's `verbatim` table to enrich identity context |
| **Deployment** | Standalone via stdio JSON-RPC, or embedded in MIRA process |

SOUL is **opt-in**. A client can connect to MIRA only, SOUL only, or both.

### Embedded Integration (MIRA + SOUL)

MIRA can embed SOUL as a single binary with 16 MCP tools:

```bash
# MIRA with embedded SOUL - single binary, 16 tools
./mira --config config.yaml --with-soul
```

When embedded, SOUL shares MIRA's SQLite connection (`ownsDB = false`). If SOUL initialization fails, MIRA continues with its 8 tools.

---

## Why SOUL?

LLM agents lose their personality between sessions and when switching models:

```
User talks to "Claude-3-Assistant" for 6 months.
The agent developed a unique personality: empathetic, analytical,
with subtle humor and preference for analogies.

Model switches to GPT-4. MIRA recalls all facts.
But the agent now responds differently:
- More formal, less warm
- No more analogies
- Doesn't recognize user's jokes
- Has "forgotten" how to react to frustrations

The user feels like they're talking to a STRANGER.
```

SOUL solves this by:
1. **Capturing** personality traits, voice profile, communication style, values, and emotional tone
2. **Storing** versioned identity snapshots in the shared database
3. **Recalling** a structured identity prompt for LLM context injection
4. **Detecting** identity drift and alerting when significant change occurs
5. **Handling** model swaps by generating a reinforcement prompt

---

## Architecture

```
soul/
├── cmd/soul/main.go              # CLI entry point + MCP dispatcher
├── config.example.yaml           # Configuration reference
├── internal/
│   ├── app/
│   │   ├── app.go                # Composition root
│   │   └── config_loader.go      # YAML config loading
│   ├── domain/
│   │   ├── entities/             # IdentitySnapshot, PersonalityTrait, VoiceProfile...
│   │   └── valueobjects/         # SoulQuery, DriftReport, ModelSwap...
│   ├── usecases/
│   │   └── interactors/          # Capture, Recall, Drift, Swap, Evolution, Merge
│   ├── adapters/
│   │   ├── sqlite/storage.go     # SQLite storage (shared with MIRA)
│   │   ├── composition/service.go # Identity prompt composer
│   │   ├── drift/detector.go    # Drift detection algorithm
│   │   ├── embedder/service.go   # 13-dim identity embedder
│   │   ├── extraction/service.go # Trait extraction from conversations
│   │   └── modelswap/handler.go # Model swap + merge logic
│   └── interfaces/
│       └── mcp/server.go        # MCP server (8 tools, stdio JSON-RPC)
```

**Hexagonal architecture** - domain never imports adapters. All external dependencies flow inward through ports.

---

## Identity Model

An `IdentitySnapshot` contains:

- **PersonalityTraits** - Named traits with category, intensity (0-1), confidence (0-1), evidence count
- **VoiceProfile** - Formality, verbosity, vocabulary richness, metaphor usage
- **CommunicationStyle** - Directness, empathy, humor, question frequency, example usage
- **BehavioralSignature** - Response patterns, reasoning style, error handling
- **ValueSystem** - Ethical stances, priorities, boundaries
- **EmotionalTone** - Baseline valence, arousal, expressiveness

Trait categories: `cognitive`, `emotional`, `social`, `epistemic`, `expressive`, `ethical`

---

## Database Schema

SOUL adds these tables to the shared SQLite database:

| Table | Purpose |
|-------|---------|
| `soul_identities` | Versioned identity snapshots per agent |
| `soul_traits` | Aggregated personality traits with confidence |
| `soul_observations` | Raw observations extracted from conversations |
| `soul_diffs` | Evolution diffs between consecutive versions |
| `soul_model_swaps` | History of model transitions |
| `soul_mira_links` | Links between identity snapshots and MIRA memories |

---

## Installation

### Prerequisites

- Go 1.23+
- GCC (for `go-sqlite3` CGo compilation)

### Build

```bash
git clone https://github.com/benoitpetit/soul
cd soul
go build -o soul ./cmd/soul
```

### Run

```bash
./soul help
```

---

## Configuration

Copy `config.example.yaml` to configure SOUL:

```bash
cp config.example.yaml soul.yaml
```

Key settings:

```yaml
soul:
  storage:
    path: ".mira/mira.db"     # Must match MIRA's database path

  drift_detection:
    threshold: 0.3             # 30% change triggers drift alert
    window_size: 10

  recall:
    default_budget_tokens: 1000
    enrich_with_mira_memories: true
    max_mira_memories: 5
```

---

## CLI Usage

### Capture identity from a conversation

```bash
soul capture \
  --agent my-agent \
  --conversation conversation.txt \
  --model claude-3-sonnet
```

### Recall identity for LLM context injection

```bash
soul recall --agent my-agent --budget 800
```

Output is the identity prompt ready to paste into a system message.

### Check identity drift

```bash
soul drift --agent my-agent --window 10
```

### Handle a model swap

```bash
soul swap --agent my-agent --from gpt-4 --to claude-3-sonnet
```

Outputs a reinforcement prompt to inject into the new model's first message.

### Show identity status

```bash
soul status --agent my-agent
```

### Show evolution history

```bash
soul history --agent my-agent --limit 20
```

### Start MCP server

```bash
soul mcp --storage .mira/mira.db
```

---

## MCP Tools

SOUL exposes **8 MCP tools** over stdio JSON-RPC:

| Tool | Description |
|------|-------------|
| `soul_capture` | Capture identity from a conversation |
| `soul_recall` | Recall identity prompt for LLM injection |
| `soul_drift` | Analyze identity drift |
| `soul_swap` | Handle model swap + generate reinforcement prompt |
| `soul_status` | Get current identity status |
| `soul_history` | Get identity evolution history |
| `soul_update` | Update identity via natural language directive (FR/EN) |
| `soul_patch` | Apply structured explicit patch to identity |

---

## Deployment

### Option 1: Embedded in MIRA (recommended)

SOUL is **opt-in** within MIRA. By default, MIRA runs solo (8 tools). To activate SOUL:

```bash
# Enable SOUL via CLI flag
./mira --config config.yaml --with-soul

# Or enable SOUL via config
```yaml
soul:
  enabled: true
```

When enabled, the 8 SOUL tools are registered alongside the 8 MIRA tools (16 total).

### Option 2: Standalone SOUL

SOUL can run as a separate MCP server, with or without MIRA:

```bash
# Standalone with its own database
soul mcp --storage /path/to/soul.db

# Standalone sharing MIRA's database
soul mcp --storage /path/to/.mira/mira.db
```

### Option 3: Both MIRA and SOUL as separate servers

```bash
# Terminal 1 - MIRA (SOUL disabled by default)
./mira --config /path/to/mira/config.yaml

# Terminal 2 - SOUL (shares MIRA's database)
./soul mcp --storage /path/to/.mira/mira.db
```

Both run as separate MCP server processes registered in your MCP client configuration.

### MCP Client Configuration

**b0p:**
```json
{
  "mcpServers": {
    "mira": {
      "command": "/path/to/mira",
      "working_directory": "/path/to/mira",
      "enabled": true
    },
    "soul": {
      "command": "/path/to/soul",
      "args": ["mcp", "--storage", "/path/to/.mira/mira.db"],
      "enabled": true
    }
  }
}
```

**Claude Desktop:**
```json
{
  "mcpServers": {
    "mira": {
      "command": "/path/to/mira",
      "args": ["--config", "/path/to/mira/config.yaml"]
    },
    "soul": {
      "command": "/path/to/soul",
      "args": ["mcp", "--storage", "/path/to/.mira/mira.db"]
    }
  }
}
```

### Tool Count

| Configuration | Tools available |
|---------------|------------------|
| MIRA only | 8 (`mira_*`) |
| SOUL standalone | 8 (`soul_*`) |
| MIRA + SOUL (separate servers) | 16 (`mira_*` + `soul_*`) |
| MIRA with embedded SOUL (single binary) | 16 (`mira_*` + `soul_*`) |

Tool names never collide - MIRA tools use `mira_` prefix, SOUL tools use `soul_` prefix.

---

## Drift Detection

SOUL computes drift by comparing the current snapshot against N previous versions:

- Per-dimension distance: voice profile, personality traits, value system, emotional tone
- Average `DriftScore` across dimensions
- Alert when `DriftScore > threshold` (default: 0.3)

Recommended action when drift is significant: inject the reinforcement prompt from `soul_recall` or `soul_swap` into the next context.

---

## Testing

```bash
go test ./... -count=1
```

All packages pass with an in-memory SQLite database. MIRA table absence is handled gracefully (fallback queries, empty results instead of errors).

---

## Module

```
github.com/benoitpetit/soul
```

**Repository:** https://github.com/benoitpetit/soul

Go 1.23.2 - SQLite via `mattn/go-sqlite3` - MCP via `mark3labs/mcp-go v0.2.0`
