# SOUL - System for Observed Unique Legacy
# Makefile pour le build, test et déploiement

.PHONY: build test clean install dev lint help

# Variables
BINARY_NAME=soul
BUILD_DIR=./build
CMD_DIR=./cmd/soul
GO=go
GOFLAGS=-v

# Couleurs pour le output
BLUE=\033[36m
GREEN=\033[32m
YELLOW=\033[33m
RED=\033[31m
RESET=\033[0m

## help: Affiche cette aide
help:
	@echo "${BLUE}SOUL - System for Observed Unique Legacy${RESET}"
	@echo ""
	@echo "${GREEN}Usage:${RESET}"
	@echo "  make ${YELLOW}<target>${RESET}"
	@echo ""
	@echo "${GREEN}Targets:${RESET}"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${YELLOW}%-15s${RESET} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: Compile le binaire SOUL
build:
	@echo "${BLUE}Building SOUL...${RESET}"
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "${GREEN}Build complete: $(BUILD_DIR)/$(BINARY_NAME)${RESET}"

## test: Lance les tests
test:
	@echo "${BLUE}Running tests...${RESET}"
	$(GO) test -v ./...
	@echo "${GREEN}Tests complete${RESET}"

## test-coverage: Lance les tests avec couverture
test-coverage:
	@echo "${BLUE}Running tests with coverage...${RESET}"
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${RESET}"

## clean: Nettoie les fichiers de build
clean:
	@echo "${BLUE}Cleaning...${RESET}"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "${GREEN}Clean complete${RESET}"

## install: Installe SOUL dans $GOPATH/bin
install: build
	@echo "${BLUE}Installing SOUL...${RESET}"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/ 2>/dev/null || cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "${GREEN}SOUL installed${RESET}"

## dev: Mode développement avec hot reload
dev:
	@echo "${BLUE}Starting SOUL in dev mode...${RESET}"
	@air -c .air.toml 2>/dev/null || $(GO) run $(CMD_DIR) help

## lint: Lance le linter
golint:
	@echo "${BLUE}Running linter...${RESET}"
	@golangci-lint run ./... 2>/dev/null || echo "${YELLOW}golangci-lint not installed${RESET}"

## fmt: Formate le code
fmt:
	@echo "${BLUE}Formatting code...${RESET}"
	$(GO) fmt ./...
	@echo "${GREEN}Formatting complete${RESET}"

## vet: Analyse statique du code
vet:
	@echo "${BLUE}Running go vet...${RESET}"
	$(GO) vet ./...
	@echo "${GREEN}Vet complete${RESET}"

## deps: Télécharge les dépendances
deps:
	@echo "${BLUE}Downloading dependencies...${RESET}"
	$(GO) mod download
	$(GO) mod tidy
	@echo "${GREEN}Dependencies ready${RESET}"

## migrate: Crée les tables dans la base SQLite (pour init)
migrate:
	@echo "${BLUE}Initializing SOUL database...${RESET}"
	@echo "${YELLOW}Note: Tables are created automatically on first run${RESET}"
	@echo "${GREEN}Database ready${RESET}"

## capture-example: Exemple de capture d'identité
capture-example: build
	@echo "${BLUE}Running capture example...${RESET}"
	@echo "This is a sample conversation with my assistant. The assistant is very analytical and empathetic. It always tries to understand the root cause of problems. It uses humor occasionally to lighten the mood." > /tmp/soul_example_convo.txt
	$(BUILD_DIR)/$(BINARY_NAME) capture --agent example-agent --conversation /tmp/soul_example_convo.txt --model gpt-4

## recall-example: Exemple de recall d'identité
recall-example: build
	@echo "${BLUE}Running recall example...${RESET}"
	$(BUILD_DIR)/$(BINARY_NAME) recall --agent example-agent --budget 500

## status-example: Exemple de statut
status-example: build
	@echo "${BLUE}Running status example...${RESET}"
	$(BUILD_DIR)/$(BINARY_NAME) status --agent example-agent

## all: Build + test + lint
all: deps fmt vet build test
	@echo "${GREEN}All tasks complete!${RESET}"
