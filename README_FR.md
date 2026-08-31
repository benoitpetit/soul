<div align="center">
  <img src="./logo.png" alt="SOUL Logo" width="800">

  # SOUL
  ### System for Observed Unique Legacy

  **Système de Préservation d'Identité pour Agents LLM**

  [![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
  [![Version](https://img.shields.io/badge/Version-0.0.6-blue?style=flat-square)]()
  [![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen?style=flat-square)]()

  *100% Local • Déterministe • Identité Versionnée • Agnostique au Modèle*

  [Changelog](CHANGELOG.md) • [Skill](SKILL.md) • [English](README.md) • [Intégration MIRA](https://github.com/benoitpetit/mira)

</div>

---

## Table des matières

- [Relation avec MIRA](#relation-avec-mira)
- [Pourquoi SOUL ?](#pourquoi-soul-)
- [Architecture](#architecture)
- [Modèle d'identité](#modèle-didentité)
- [Schéma de base de données](#schéma-de-base-de-données)
- [Installation](#installation)
- [Configuration](#configuration)
- [Utilisation CLI](#utilisation-cli)
- [Outils MCP](#outils-mcp)
- [Déploiement](#déploiement)
- [Détection de dérive](#détection-de-dérive)
- [Tests](#tests)
- [Module](#module)
- [Changelog](CHANGELOG.md)

---

## Relation avec MIRA

| Aspect | Détail |
|--------|--------|
| **Dépendance** | Aucune — SOUL compile et s'exécute sans MIRA |
| **Intégration** | Serveur MCP standalone, ou intégré dans MIRA (binaire unique, 17 outils) |
| **Base de données** | SOUL ajoute des tables `soul_*` à `.mira/mira.db` |
| **Accès croisé** | SOUL peut lire la table `verbatim` de MIRA pour enrichir le contexte identitaire |
| **Déploiement** | Standalone via stdio JSON-RPC, ou intégré dans le processus MIRA |

SOUL est **optionnel**. Un client peut se connecter à MIRA uniquement, à SOUL uniquement, ou aux deux.

### Intégration embarquée (MIRA + SOUL)

MIRA peut intégrer SOUL en un binaire unique avec 17 outils MCP :

```bash
./mira --config config.yaml --with-soul
```

Lorsqu'il est intégré, SOUL partage la connexion SQLite de MIRA (`ownsDB = false`). Si l'initialisation de SOUL échoue, MIRA continue avec ses 9 outils.

---

## Pourquoi SOUL ?

Les agents LLM perdent leur personnalité entre les sessions et lors des changements de modèle :

```
L'utilisateur parle à "Claude-3-Assistant" pendant 6 mois.
L'agent a développé une personnalité unique : empathique, analytique,
avec un humour subtil et une préférence pour les analogies.

Le modèle passe à GPT-4. MIRA se souvient de tous les faits.
Mais l'agent répond maintenant différemment :
- Plus formel, moins chaleureux
- Plus d'analogies
- Ne reconnaît plus les blagues de l'utilisateur
- A "oublié" comment réagir aux frustrations

L'utilisateur a l'impression de parler à un ÉTRANGER.
```

SOUL résout ce problème en :

1. **Capturant** les traits de personnalité, le profil vocal, le style de communication, les valeurs et le ton émotionnel
2. **Stockant** des instantanés d'identité versionnés dans la base de données partagée
3. **Rappelant** un prompt d'identité structuré pour l'injection dans le contexte LLM
4. **Détectant** la dérive identitaire et alertant lors d'un changement significatif
5. **Gérant** les changements de modèle en générant un prompt de renforcement

---

## Architecture

```
soul/
├── cmd/soul/main.go              # Point d'entrée CLI + dispatcheur MCP
├── config.example.yaml           # Référence de configuration
├── internal/
│   ├── app/
│   │   ├── app.go                # Racine de composition
│   │   └── config_loader.go      # Chargement config YAML
│   ├── domain/
│   │   ├── entities/             # IdentitySnapshot, PersonalityTrait, VoiceProfile...
│   │   └── valueobjects/         # SoulQuery, DriftReport, ModelSwap...
│   ├── usecases/
│   │   └── interactors/          # Capture, Recall, Drift, Swap, Evolution, Merge, Update
│   ├── adapters/
│   │   ├── sqlite/storage.go     # Stockage SQLite (partagé avec MIRA)
│   │   ├── composition/service.go # Composeur de prompt d'identité
│   │   ├── drift/detector.go     # Algorithme de détection de dérive
│   │   ├── embedder/service.go   # Embedder d'identité 13 dimensions
│   │   ├── extraction/service.go # Extraction de traits depuis les conversations
│   │   └── modelswap/handler.go  # Logique de changement et fusion de modèle
│   └── interfaces/
│       └── mcp/server.go         # Serveur MCP (8 outils, stdio JSON-RPC)
```

**Architecture hexagonale** — le domaine n'importe jamais les adaptateurs. Toutes les dépendances externes circulent vers l'intérieur à travers les ports.

---

## Modèle d'identité

Un `IdentitySnapshot` contient :

- **PersonalityTraits** — traits nommés avec catégorie, intensité (0–1), confiance (0–1), nombre d'évidences
- **VoiceProfile** — formalité, verbosité, richesse du vocabulaire, usage des métaphores
- **CommunicationStyle** — directitude, empathie, humour, fréquence des questions, usage des exemples
- **BehavioralSignature** — modèles de réponse, style de raisonnement, gestion des erreurs
- **ValueSystem** — positions éthiques, priorités, limites
- **EmotionalTone** — valence de base, excitation, expressivité

Catégories de traits : `cognitive`, `emotional`, `social`, `epistemic`, `expressive`, `ethical`

---

## Schéma de base de données

SOUL ajoute ces tables à la base de données SQLite partagée :

| Table | Rôle |
|-------|------|
| `soul_identities` | Instantanés d'identité versionnés par agent |
| `soul_traits` | Traits de personnalité agrégés avec confiance |
| `soul_observations` | Observations brutes extraites des conversations |
| `soul_diffs` | Diffs d'évolution entre versions consécutives |
| `soul_model_swaps` | Historique des transitions de modèle |
| `soul_mira_links` | Liens entre instantanés d'identité et mémoires MIRA |

---

## Installation

### Prérequis

- Go 1.23+
- GCC (pour la compilation CGo de `go-sqlite3`)

### Compilation

```bash
git clone https://github.com/benoitpetit/soul
cd soul
go build -o soul ./cmd/soul
```

### Exécution

```bash
./soul help
```

---

## Configuration

Copiez `config.example.yaml` pour configurer SOUL :

```bash
cp config.example.yaml soul.yaml
```

Paramètres clés :

```yaml
soul:
  storage:
    path: ".mira/mira.db"       # Doit correspondre au chemin de la base MIRA

  drift_detection:
    threshold: 0.3               # 30% de changement déclenche une alerte
    window_size: 10

  recall:
    default_budget_tokens: 1000
    enrich_with_mira_memories: false  # Active l'enrichissement avec les mémoires MIRA (nécessite l'intégration MIRA)
    max_mira_memories: 5
```

---

## Utilisation CLI

### Capturer l'identité depuis une conversation

```bash
soul capture \
  --agent mon-agent \
  --conversation conversation.txt \
  --model claude-3-sonnet
```

### Rappeler l'identité pour injection dans le contexte LLM

```bash
soul recall --agent mon-agent --budget 800
```

La sortie est le prompt d'identité prêt à coller dans un message système.

### Vérifier la dérive identitaire

```bash
soul drift --agent mon-agent --window 10
```

### Gérer un changement de modèle

```bash
soul swap --agent mon-agent --from gpt-4 --to claude-3-sonnet
```

Génère un prompt de renforcement à injecter dans le premier message du nouveau modèle.

### Afficher le statut d'identité

```bash
soul status --agent mon-agent
```

### Afficher l'historique d'évolution

```bash
soul history --agent mon-agent --limit 20
```

### Démarrer le serveur MCP

```bash
soul mcp --storage .mira/mira.db
```

---

## Outils MCP

SOUL expose **8 outils MCP** via stdio JSON-RPC :

| Outil | Description |
|-------|-------------|
| `soul_capture` | Capturer l'identité depuis une conversation |
| `soul_recall` | Rappeler le prompt d'identité pour injection LLM |
| `soul_drift` | Analyser la dérive identitaire |
| `soul_swap` | Gérer le changement de modèle et générer le prompt de renforcement |
| `soul_status` | Obtenir le statut d'identité actuel |
| `soul_history` | Obtenir l'historique d'évolution d'identité |
| `soul_update` | Mettre à jour l'identité via directive en langage naturel (FR/EN) |
| `soul_patch` | Appliquer un patch structuré et explicite à l'identité |

---

## Déploiement

### Option 1 : Intégré dans MIRA (recommandé)

SOUL est **opt-in** dans MIRA. Par défaut, MIRA fonctionne seul (9 outils). Pour activer SOUL :

```bash
# Activer SOUL via le flag CLI
./mira --config config.yaml --with-soul
```

Ou activer SOUL via la configuration :

```yaml
soul:
  enabled: true
```

Quand il est activé, les 8 outils SOUL sont enregistrés aux côtés des 9 outils MIRA (17 au total).

### Option 2 : SOUL standalone

SOUL peut fonctionner comme un serveur MCP séparé, avec ou sans MIRA :

```bash
# Standalone avec sa propre base de données
soul mcp --storage /path/to/soul.db

# Standalone partageant la base de MIRA
soul mcp --storage /path/to/.mira/mira.db
```

### Option 3 : MIRA et SOUL comme serveurs séparés

```bash
# Terminal 1 — MIRA (SOUL désactivé par défaut)
./mira --config /path/to/mira/config.yaml

# Terminal 2 — SOUL (partage la base de MIRA)
./soul mcp --storage /path/to/.mira/mira.db
```

Les deux fonctionnent comme des processus de serveur MCP séparés enregistrés dans la configuration de votre client MCP.

### Configuration client MCP

**OpenCode / b0p :**

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
| MIRA uniquement | 9 (`mira_*`) |
| SOUL standalone | 8 (`soul_*`) |
| MIRA + SOUL (serveurs séparés) | 17 (`mira_*` + `soul_*`) |
| MIRA avec SOUL intégré (binaire unique) | 17 (`mira_*` + `soul_*`) |

Les noms d'outils ne rentrent jamais en collision — MIRA utilise le préfixe `mira_`, SOUL utilise le préfixe `soul_`.

---

## Détection de dérive

SOUL calcule la dérive en comparant l'instantané actuel contre N versions précédentes :

- Distance par dimension sur les 6 dimensions : profil vocal, traits de personnalité, système de valeurs, ton émotionnel, style de communication, signature comportementale
- Score moyen de `DriftScore` à travers les dimensions
- Alerte quand `DriftScore > threshold` (défaut : 0.3)

Action recommandée quand la dérive est significative : injecter le prompt de renforcement de `soul_recall` ou `soul_swap` dans le prochain contexte.

---

## Tests

```bash
go test ./... -count=1
```

Tous les packages passent avec une base de données SQLite en mémoire. L'absence de tables MIRA est gérée gracieusement (requêtes de repli, résultats vides plutôt qu'erreurs).

---

## Module

```
github.com/benoitpetit/soul
```

**Dépôt :** https://github.com/benoitpetit/soul

Go 1.23.2 — SQLite via `mattn/go-sqlite3` — MCP via `mark3labs/mcp-go v0.2.0`

---

## Changelog

Voir [CHANGELOG.md](CHANGELOG.md) pour l'historique complet des releases.
