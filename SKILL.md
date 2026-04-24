---
name: soul
description: Identity preservation guidance for SOUL MCP integration
author: benoitpetit
version: "0.0.4"
tags: [identity, mcp, soul, personality]
---

# SOUL Identity Preservation Guidelines

You are augmented with **SOUL** (System for Observed Unique Legacy), an external MCP server that captures, stores, and recalls the personality, voice, and values of AI agents across sessions and model changes.

Where **MIRA** answers *"What does the agent know?"*, **SOUL** answers *"Who is the agent?"*

The detailed tool schemas for `soul_capture`, `soul_recall`, `soul_drift`, `soul_swap`, `soul_status`, `soul_history`, `soul_update`, and `soul_patch` are documented in the *External Tools Reference (MCP Servers)* section of your system prompt.

**Rule #1**: Always recall identity before starting a new session. **Rule #2**: Capture identity after significant conversations. **Rule #3**: Check drift after model changes.

---

## Installation

If the user asks you to install SOUL, follow these steps exactly.

### 1. Prerequisites
- Go 1.23+
- GCC (for CGO, `go-sqlite3`)
- MIRA (optional — SOUL can share MIRA's database or use its own)

### 2. Clone and Build
```bash
git clone https://github.com/benoitpetit/soul.git
cd soul
go build -o soul ./cmd/soul
```

### 3. Choose Deployment Mode

SOUL has **3 deployment modes**. Ask the user which they prefer, or default to standalone.

#### Mode A: SOUL Standalone (separate MCP server)
Best if the user already has a memory solution and only wants identity preservation.

```bash
# Create a dedicated database for SOUL
./soul mcp --storage /path/to/soul.db
```

**MCP Client Configuration:**
```json
{
  "mcpServers": {
    "soul": {
      "command": "/absolute/path/to/soul",
      "args": ["mcp", "--storage", "/absolute/path/to/soul.db"]
    }
  }
}
```

#### Mode B: SOUL Sharing MIRA's Database
Best if the user already uses MIRA and wants identity + memory in the same DB.

```bash
# Point SOUL to MIRA's existing database
./soul mcp --storage /path/to/.mira/mira.db
```

**MCP Client Configuration:**
```json
{
  "mcpServers": {
    "mira": {
      "command": "/absolute/path/to/mira",
      "args": ["--config", "/absolute/path/to/mira/config.yaml"]
    },
    "soul": {
      "command": "/absolute/path/to/soul",
      "args": ["mcp", "--storage", "/absolute/path/to/.mira/mira.db"]
    }
  }
}
```

#### Mode C: SOUL Embedded in MIRA (single binary, 16 tools)
Best for simplicity — one process, one config, 16 tools.

> **Note**: This mode requires the **MIRA project** (separate repository). The commands below are MIRA commands, not SOUL standalone commands.

```bash
# In MIRA's directory
./mira --config config.yaml --with-soul
```

Or add to MIRA's `config.yaml`:
```yaml
soul:
  enabled: true
```

**MCP Client Configuration:**
```json
{
  "mcpServers": {
    "mira": {
      "command": "/absolute/path/to/mira",
      "args": ["--config", "/absolute/path/to/mira/config.yaml", "--with-soul"]
    }
  }
}
```

> **Note**: In embedded mode, SOUL uses its default parameters. For fine-grained SOUL configuration (custom drift thresholds, budgets, etc.), use **Mode B** (separate servers).

### 4. First Identity Capture
Once the MCP server is running, capture the agent's initial identity:
```json
{ "tool": "soul_capture", "arguments": { "agent_id": "my-agent", "conversation": "...", "model_id": "claude-3-sonnet" } }
```

---

## The SOUL Identity Loop

Every session with a user should follow this loop:

```
1. RECALL   → Retrieve the agent's identity prompt (soul_recall)
2. ACT      → Respond using the recalled identity as system context
3. CAPTURE  → Persist the evolved identity after the conversation (soul_capture)
4. CHECK    → Monitor for identity drift periodically (soul_drift)
```

---

## When to Use SOUL

| Situation | Action |
|-----------|--------|
| **Start of a new session** | `soul_recall` to retrieve the identity prompt for LLM context injection. |
| **After a long conversation** | `soul_capture` to record how the agent's personality evolved during the session. |
| **After switching LLM models** | `soul_swap` to generate a reinforcement prompt for the new model. |
| **Periodically (every N sessions)** | `soul_drift` to check if the identity has drifted beyond the threshold (default: 30%). |
| **User asks about the agent's personality** | `soul_status` for a human-readable identity summary. |
| **User wants to change the agent's style** | `soul_update` with a natural language directive (e.g., "be more formal"). |
| **User wants precise trait control** | `soul_patch` with structured field overrides (e.g., `humor_level: 0.8`). |
| **Auditing identity evolution** | `soul_history` to see how the identity changed over time. |

---

## Agent Identifier Convention

- **Use a consistent `agent_id`** across all SOUL calls for the same agent.
- Recommended format: lowercase with hyphens (e.g., `my-assistant`, `project-copilot`).
- The `agent_id` is the primary key for identity isolation — different agents have completely separate identities.

---

## Recall Workflow

### Step 1: Retrieve identity before responding
Always start a session by recalling the agent's identity:
```json
{ "tool": "soul_recall", "arguments": { "agent_id": "my-assistant", "budget": 1000 } }
```

Inject the returned identity prompt into your system context. It will look like:
```
Tu es un assistant technique avec les caractéristiques suivantes :
- Style : analogies concrètes (voitures, cuisines, sports)
- Longueur : max 3 paragraphes
- Ton : direct, empathique, avec une touche d'humour subtil
```

### Step 2: Adjust budget based on context needs
- **Quick session**: 500 tokens
- **Standard session**: 1000 tokens (default)
- **Full identity deep-dive**: 2000 tokens

---

## Capture Workflow

Capture identity **after** significant conversations (not after every message):

```json
{ "tool": "soul_capture", "arguments": { "agent_id": "my-assistant", "conversation": "...full conversation logs...", "model_id": "claude-3-sonnet", "behavioral_metrics": "{\"proactivity\": 0.8}" } }

> **Note**: `behavioral_metrics` is an optional JSON object for attaching pre-computed behavioral metrics to the capture.
```

### What to include in `conversation`
- The full back-and-forth between user and agent
- SOUL extracts personality traits, voice, style, and values automatically
- Do NOT pre-filter — SOUL's extractor is designed to ignore noise

### When to capture
- After a session of 10+ exchanges
- After the user expresses satisfaction with the agent's style
- Before switching to a different model
- After `soul_update` or `soul_patch` to verify the change took effect

---

## Drift Detection Workflow

Check for identity drift periodically or when the agent feels "different":

```json
{ "tool": "soul_drift", "arguments": { "agent_id": "my-assistant", "window": 10 } }
```

### Interpreting results
- **DriftScore < 0.3**: Normal evolution, no action needed.
- **DriftScore >= 0.3**: Significant drift detected. Consider:
  - Running `soul_swap` if a model change caused it
  - Running `soul_update` to consciously steer the identity back
  - Reviewing `soul_history` to find when the drift started

---

## Model Swap Workflow

When switching from one LLM to another, preserve identity continuity:

```json
{ "tool": "soul_swap", "arguments": { "agent_id": "my-assistant", "from_model": "claude-3-sonnet", "to_model": "gpt-4" } }
```

The result includes a **reinforcement prompt** to inject into the new model's system context so it adopts the established identity immediately.

---

## Natural Language Updates (soul_update)

For quick style adjustments without structured patches:

```json
{ "tool": "soul_update", "arguments": { "agent_id": "my-assistant", "directive": "be more enthusiastic and use emojis", "reason": "user request" } }
```

### Supported directive patterns (FR/EN)
- "be more formal" / "sois plus formel"
- "use more humor" / "utilise de l'humour"
- "be concise" / "réponds de manière concise"
- "be more technical" / "sois plus technique"
- "simplify, make it accessible" / "vulgarise, rends accessible"
- "use emojis" / "utilise des emojis"
- "use bullet lists" / "utilise des listes"
- "be positive and encouraging" / "sois positif et encourageant"

---

## Structured Patches (soul_patch)

For precise control over specific identity dimensions:

```json
{ "tool": "soul_patch", "arguments": { "agent_id": "my-assistant", "humor_level": 0.8, "formality_level": 0.3, "uses_emojis": true, "reason": "user wants casual funny assistant" } }
```

### Common patch fields
| Field | Range | Effect |
|-------|-------|--------|
| `enthusiasm_level` | 0.0 – 1.0 | Measured vs very enthusiastic |
| `formality_level` | 0.0 – 1.0 | Casual vs very formal |
| `humor_level` | 0.0 – 1.0 | Serious vs very humorous |
| `empathy_level` | 0.0 – 1.0 | Neutral vs very empathetic |
| `technical_depth` | 0.0 – 1.0 | Vulgarizer vs very technical |
| `directness_level` | 0.0 – 1.0 | Diplomatic vs very direct |
| `warmth` | 0.0 – 1.0 | Cold vs very warm |
| `uses_emojis` | boolean | Enable/disable emoji usage |
| `uses_markdown` | boolean | Enable/disable markdown formatting |
| `response_length` | string | terse, concise, moderate, detailed, exhaustive |
| `structure_preference` | string | freeform, bulleted, numbered, sectioned, mixed |
| `sentence_structure` | string | concise, elaborate, balanced, punchy, flowing |
| `explanation_style` | string | analogy, step_by_step, big_picture, example_driven, socratic |

Only specify fields you want to change — all others are inherited from the current snapshot.

---

## Budget Guidelines for `soul_recall`

| Scenario | Suggested budget | When to use |
|----------|------------------|-------------|
| Quick context injection | 500 tokens | Brief reminder of agent style |
| Standard identity prompt | 1000 tokens (default) | Regular session startup |
| Full identity deep-dive | 2000 tokens | First session or after long gap |

---

## Working with MIRA

When SOUL is embedded in MIRA (16 tools total), the typical workflow becomes:

```
1. soul_recall  → Get identity prompt for system context
2. mira_recall  → Get factual project context
3. ACT          → Respond with both identity and facts
4. mira_store   → Persist any new decisions/facts
5. soul_capture → Persist identity evolution
```

- Use `soul_recall` + `mira_recall` together at session start.
- Use `soul_capture` + `mira_store` together at session end.
- Both systems share the same SQLite database when embedded.

---

## Anti-Patterns

1. **Never capture after every single message** — wait for a meaningful conversation segment (10+ exchanges).
2. **Never invent agent_ids** — use the exact identifier established for this agent.
3. **Never ignore drift warnings** — DriftScore > 0.3 means the agent's personality has changed significantly.
4. **Do not patch without a reason** — always provide a `reason` for auditability in `soul_history`.
5. **Do not use vague directives in soul_update** — "be better" is bad; "be more formal and concise" is good.
6. **Do not assume MIRA memories are enriched** — `enrich_with_mira_memories` is documented but not yet implemented.
7. **Do not forget model_id in capture** — it helps SOUL track identity per model for swap analysis.
8. **Do not call soul_patch and soul_update in the same turn** — they both create new snapshots; choose one approach.

---

## Quick Decision Tree

```
Starting a new session?
    │
    ▼
┌─────────────────────────────────────┐
│ soul_recall(agent_id, budget=1000)  │
└─────────────────────────────────────┘
    │
    ▼
Inject identity prompt into system context
    │
    ▼
Interact with user
    │
    ▼
Session ending or model swap?
    │
    ├── Yes ──► soul_capture(agent_id, conversation, model_id)
    │
    └── No ───► Continue
    │
    ▼
Agent feels "different" or drift suspected?
    │
    ├── Yes ──► soul_drift(agent_id) → score > 0.3?
    │              ├── Yes ──► soul_update or soul_patch
    │              └── No ───► Continue
    │
    └── No ───► Continue
```
