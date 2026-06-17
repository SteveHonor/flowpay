.PHONY: up down build tdd test-domain test-app test-http bdd front-test fmt vet deps swagger

# ── Detect docker compose command (plugin vs legacy standalone) ───────────────
COMPOSE := $(shell docker compose version > /dev/null 2>&1 && echo "docker compose" || echo "docker-compose")

# ── Docker runners ────────────────────────────────────────────────────────────
# alpine: build/fmt/vet — sem CGO, imagem enxuta
# golang:1.22: testes com -race — exige CGO (gcc via Debian)
# GO_FULL: monta o projeto inteiro para o BDD acessar ../../features
GO      = docker run --rm -v "$(CURDIR)/backend:/src" -w /src golang:1.22-alpine sh -c
GO_TEST = docker run --rm -v "$(CURDIR)/backend:/src" -w /src golang:1.22 sh -c
GO_FULL = docker run --rm -v "$(CURDIR):/app" -w /app/backend golang:1.22 sh -c
NODE    = docker run --rm -e NO_UPDATE_NOTIFIER=1 -v "$(CURDIR)/frontend:/src" -w /src node:22-alpine sh -c

# ── Infra ─────────────────────────────────────────────────────────────────────
up:           ## sobe tudo (postgres + api + web) via docker compose
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

# ── Go ────────────────────────────────────────────────────────────────────────
deps:         ## baixa/atualiza dependências Go e regenera go.sum
	$(GO) "go mod download && go mod tidy"

build:        ## compila o backend
	$(GO) "go mod download && go build ./..."

fmt:          ## formata o código Go
	$(GO) "gofmt -w ."

vet:          ## análise estática Go
	$(GO) "go mod download && go vet ./..."

swagger:      ## alias para deps — use após mudar go.mod
	$(MAKE) deps

# ── Testes Go ─────────────────────────────────────────────────────────────────
tdd:          ## todos os testes do backend (race detector)
	$(GO_TEST) "go mod download && go test -race -v ./..."

test-domain:  ## só o domínio — invariantes do aggregate Time
	$(GO_TEST) "go mod download && go test -race -v ./internal/atendimento/domain/..."

test-app:     ## só a camada de aplicação — casos de uso do Distribuidor
	$(GO_TEST) "go mod download && go test -race -v ./internal/atendimento/application/..."

test-http:    ## só os testes de integração HTTP — status codes, JSON, erros → HTTP
	$(GO_TEST) "go mod download && go test -race -v ./internal/atendimento/infrastructure/http/..."

bdd:          ## cenários BDD Gherkin/godog — exercita o aggregate diretamente
	$(GO_FULL) "go mod download && go test -tags bdd -v -count=1 ./features/..."

# ── Frontend ──────────────────────────────────────────────────────────────────
front-test:   ## testes de componentes React (Vitest)
	$(NODE) "npm ci && npm test"
