<div align="center">
  <img src="./logo.png" alt="SOUL Logo" width="800">
  
# SOUL - System for Observed Unique Legacy

[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Version](https://img.shields.io/badge/Version-0.0.2-blue?style=flat-square)]()

**SOUL** (System for Observed Unique Legacy) est une **extension de preservation d'identite pour les agents LLM**. Elle capture, stocke et recall (rappelle) la personnalite, la voix et les valeurs des agents IA a travers les sessions et les changements de modele.

Là où [**MIRA**](https://github.com/benoitpetit/mira) repond *"Qu'est-ce que l'agent sait ?"*, SOUL repond *"Qui est l'agent ?"*
</div>

## Relation avec MIRA

| Aspect | Detail |
|--------|--------|
| **Dependance** | Aucune - SOUL compile et s'execute sans MIRA |
| **Integration** | Peut fonctionner en standalone (serveur MCP separe) ou integre dans MIRA (binaire unique, 16 outils) |
| **Base de donnees** | SOUL ajoute des tables `soul_*` a `.mira/mira.db` |
| **Acces croise** | SOUL peut lire la table `verbatim` de MIRA pour enrichir le contexte identitaire |
| **Deploiement** | Standalone via stdio JSON-RPC, ou integre dans le processus MIRA |

SOUL est **optionnel**. Un client peut se connecter a MIRA seulement, SOUL seulement, ou les deux.

### Integration integree (MIRA + SOUL)

MIRA peut integrer SOUL comme un binaire unique avec 16 outils MCP :

```bash
# MIRA avec SOUL integre - binaire unique, 16 outils
./mira --config config.yaml --with-soul
```

Lorsqu'il est integre, SOUL partage la connexion SQLite de MIRA (`ownsDB = false`). Si l'initialisation de SOUL echoue, MIRA continue avec ses 8 outils.

---

## Pourquoi SOUL ?

Les agents LLM perdent leur personnalite entre les sessions et lors des changements de modele :

```
L'utilisateur parle a "Claude-3-Assistant" pendant 6 mois.
L'agent a developpe une personnalite unique : empathique, analytique,
avec un humour subtil et une preference pour les analogies.

Le modele passe a GPT-4. MIRA se souvient de tous les faits.
Mais l'agent repond maintenant differemment :
- Plus formel, moins chaleureux
- Plus d'analogies
- Ne reconnait plus les blagues de l'utilisateur
- A "oublie" comment reaghir aux frustrations

L'utilisateur a l'impression de parler a un ETRANGER.
```

SOUL resout ce probleme en :
1. **Capturant** les traits de personnalite, le profil vocal, le style de communication, les valeurs et le ton emotionnel
2. **Stockant** des instantanes d'identite versionnes dans la base de donnees partagee
3. **Rappelant** un prompt d'identite structure pour l'injection dans le contexte LLM
4. **Detectant** la derive identitaire et alertant lorsqu'un changement significatif se produit
5. **Gerant** les changements de modele en generant un prompt de renforcement

---

## Architecture

```
soul/
├── cmd/soul/main.go              # Point d'entree CLI + dispatcheur MCP
├── config.example.yaml           # Reference de configuration
├── internal/
│   ├── app/
│   │   ├── app.go                # Racine de composition
│   │   └── config_loader.go      # Chargement config YAML
│   ├── domain/
│   │   ├── entities/             # IdentitySnapshot, PersonalityTrait, VoiceProfile...
│   │   └── valueobjects/         # SoulQuery, DriftReport, ModelSwap...
│   ├── usecases/
│   │   └── interactors/          # Capture, Recall, Drift, Swap, Evolution, Merge
│   ├── adapters/
│   │   ├── sqlite/storage.go     # Stockage SQLite (partage avec MIRA)
│   │   ├── composition/service.go # Composeur de prompt d'identite
│   │   ├── drift/detector.go     # Algorithme de detection de derive
│   │   ├── embedder/service.go   # Embedder d'identite 13 dimensions
│   │   ├── extraction/service.go # Extraction de traits depuis conversations
│   │   └── modelswap/handler.go  # Logique de changement et fusion de modele
│   └── interfaces/
│       └── mcp/server.go         # Serveur MCP (8 outils, stdio JSON-RPC)
```

**Architecture hexagonale** - le domaine n'importe jamais les adaptateurs. Toutes les dependances externes circulent vers l'interieur a travers les ports.

---

## Modele d'identite

Un `IdentitySnapshot` contient :

- **PersonalityTraits** - Traits nommes avec categorie, intensite (0-1), confiance (0-1), compte d'evidences
- **VoiceProfile** - Formalite, verbosite, richesse du vocabulaire, usage des metaphores
- **CommunicationStyle** - Directitude, empathie, humour, frequence des questions, usage des exemples
- **BehavioralSignature** - Modeles de reponse, style de raisonnement, gestion des erreurs
- **ValueSystem** - Positions ethiques, priorites, limites
- **EmotionalTone** - Valence de base, excitation, expressivite

Categories de traits : `cognitive`, `emotional`, `social`, `epistemic`, `expressive`, `ethical`

---

## Schema de base de donnees

SOUL ajoute ces tables a la base de donnees SQLite partagee :

| Table | But |
|-------|-----|
| `soul_identities` | Instantanes d'identite versions par agent |
| `soul_traits` | Traits de personnalite agreges avec confiance |
| `soul_observations` | Observations brutes extraites des conversations |
| `soul_diffs` | Diffs d'evolution entre versions consecutives |
| `soul_model_swaps` | Historique des transitions de modele |
| `soul_mira_links` | Liens entre instantanes d'identite et memoires MIRA |

---

## Installation

### Prerequis

- Go 1.23+
- GCC (pour la compilation CGo de `go-sqlite3`)

### Construction

```bash
git clone https://github.com/benoitpetit/soul
cd soul
go build -o soul ./cmd/soul
```

### Execution

```bash
./soul help
```

---

## Configuration

Copiez `config.example.yaml` pour configurer SOUL :

```bash
cp config.example.yaml soul.yaml
```

Parametres cles :

```yaml
soul:
  storage:
    path: ".mira/mira.db"     # Doit correspondre au chemin de la base MIRA

  drift_detection:
    threshold: 0.3             # 30% de changement declenche une alerte
    window_size: 10

  recall:
    default_budget_tokens: 1000
    enrich_with_mira_memories: true
    max_mira_memories: 5
```

---

## Utilisation CLI

### Capturer l'identite depuis une conversation

```bash
soul capture \
  --agent mon-agent \
  --conversation conversation.txt \
  --model claude-3-sonnet
```

### Rappeler l'identite pour injection dans le contexte LLM

```bash
soul recall --agent mon-agent --budget 800
```

La sortie est le prompt d'identite pret a coller dans un message systeme.

### Verifier la derive identitaire

```bash
soul drift --agent mon-agent --window 10
```

### Gerer un changement de modele

```bash
soul swap --agent mon-agent --from gpt-4 --to claude-3-sonnet
```

Genere un prompt de renforcement a injecter dans le premier message du nouveau modele.

### Afficher le statut d'identite

```bash
soul status --agent mon-agent
```

### Afficher l'historique d'evolution

```bash
soul history --agent mon-agent --limit 20
```

### Demarrer le serveur MCP

```bash
soul mcp --storage .mira/mira.db
```

---

## Outils MCP

SOUL expose **8 outils MCP** via stdio JSON-RPC :

| Outil | Description |
|-------|-------------|
| `soul_capture` | Capturer l'identite depuis une conversation |
| `soul_recall` | Rappeler le prompt d'identite pour injection LLM |
| `soul_drift` | Analyser la derive identitaire |
| `soul_swap` | Gerer le changement de modele + generer le prompt de renforcement |
| `soul_status` | Obtenir le statut d'identite actuel |
| `soul_history` | Obtenir l'historique d'evolution d'identite |
| `soul_update` | Mettre a jour l'identite via directive en langage naturel (FR/EN) |
| `soul_patch` | Appliquer un patch structure et explicite a l'identite |

---

## Deploiement

### Option 1 : Integre dans MIRA (recommande)

SOUL est **opt-in** dans MIRA. Par defaut, MIRA fonctionne seul (8 outils). Pour activer SOUL :

```bash
# Activer SOUL via le flag CLI
./mira --config config.yaml --with-soul

# Ou activer SOUL via la configuration
```yaml
soul:
  enabled: true
```

Quand il est active, les 8 outils SOUL sont enregistres aux cotes des 8 outils MIRA (16 au total).

### Option 2 : SOUL standalone

SOUL peut fonctionner comme un serveur MCP separe, avec ou sans MIRA :

```bash
# Standalone avec sa propre base de donnees
soul mcp --storage /path/to/soul.db

# Standalone partageant la base de MIRA
soul mcp --storage /path/to/.mira/mira.db
```

### Option 3 : MIRA et SOUL comme serveurs separes

```bash
# Terminal 1 - MIRA (SOUL desactive par defaut)
./mira --config /path/to/mira/config.yaml

# Terminal 2 - SOUL (partage la base de MIRA)
./soul mcp --storage /path/to/.mira/mira.db
```

Les deux fonctionnent comme des processus de serveur MCP separes enregistres dans la configuration de votre client MCP.

### Configuration client MCP

**b0p :**
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

**Claude Desktop :**
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

### Nombre d'outils

| Configuration | Outils disponibles |
|---------------|-------------------|
| MIRA seulement | 8 (`mira_*`) |
| SOUL standalone | 8 (`soul_*`) |
| MIRA + SOUL (serveurs separes) | 16 (`mira_*` + `soul_*`) |
| MIRA avec SOUL integre (binaire unique) | 16 (`mira_*` + `soul_*`) |

Les noms d'outils ne rentrent jamais en collision - MIRA utilise le prefixe `mira_`, SOUL utilise le prefixe `soul_`.

---

## Detection de derive

SOUL calcule la derive en comparant l'instantane actuel contre N versions precedentes :

- Distance par dimension : profil vocal, traits de personnalite, systeme de valeurs, ton emotionnel
- Score moyen de `DriftScore` a travers les dimensions
- Alerte quand `DriftScore > threshold` (defaut : 0.3)

Action recommandee quand la derive est significative : injecter le prompt de renforcement de `soul_recall` ou `soul_swap` dans le prochain contexte.

---

## Tests

```bash
go test ./... -count=1
```

Tous les packages passent avec une base de donnees SQLite en memoire. L'absence de tables MIRA est geree gracefully (requetes de repli, resultats vides au lieu d'erreurs).

---

## Module

```
github.com/benoitpetit/soul
```

**Depot :** https://github.com/benoitpetit/soul

Go 1.23.2 - SQLite via `mattn/go-sqlite3` - MCP via `mark3labs/mcp-go v0.2.0`
