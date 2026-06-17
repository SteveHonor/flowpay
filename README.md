# FlowPay — Central de Atendimentos

Sistema de **distribuição e monitoramento de atendimentos em tempo real**. Roteia cada
solicitação para o time correto, respeita o limite de atendimentos simultâneos por atendente,
enfileira o excedente e puxa a fila (FIFO) assim que uma vaga abre — tudo sem polling.

---

## Como rodar

**Pré-requisito único: Docker.**

```bash
docker compose up --build
```

| Serviço | URL |
|---------|-----|
| Dashboard | http://localhost:3000 |
| API REST | http://localhost:8080 |
| Swagger UI | http://localhost:8080/api/docs/ |
| Spec OpenAPI | http://localhost:8080/api/docs/openapi.yaml |

A API espera o Postgres ficar saudável (`healthcheck`) antes de iniciar.
Sem banco, sobe com repositório em memória: `cd backend && go run ./cmd/api`.

### Variáveis de ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `DATABASE_URL` | *(vazio — usa memória)* | DSN Postgres |
| `PORT` | `8080` | Porta da API |
| `ALLOWED_ORIGINS` | `*` | Origens CORS (vírgula-separadas) |
| `RATE_LIMIT_RPS` | `10` | Requisições/segundo por IP |
| `RATE_LIMIT_BURST` | `30` | Burst máximo do token bucket |

### Comandos Makefile

```bash
make deps          # resolve go.sum via Docker (necessário na 1ª vez)
make tdd           # todos os testes Go, race detector
make test-domain   # só o domínio
make test-app      # só a camada de aplicação
make test-http     # só os testes de integração HTTP
make bdd           # cenários Gherkin/godog
make front-test    # Vitest (componentes React)
make build         # compila o binário Go
make fmt           # gofmt
make vet           # go vet
```

Todos os targets rodam via Docker — nenhum Go ou Node instalado localmente é necessário.

---

## Regras de negócio

- 3 times: **Cartões** (`problema_cartao`), **Empréstimos** (`contratacao_emprestimo`), **Outros Assuntos** (qualquer outro assunto).
- Cada atendente atende no **máximo 3 simultâneos** (`CAPACIDADE_MAXIMA`).
- Nova solicitação → atribuída ao atendente com **menor carga** (balanceamento por mínimo de ativos).
- Time lotado → solicitação vai para a **fila FIFO**. Vaga liberada → próxima da fila é atribuída automaticamente.
- Atendentes podem ser **adicionados dinamicamente**: participam do balanceamento e puxam a fila imediatamente.
- Atendentes podem ser **pausados**: só quando não há ativos (HTTP 409 caso contrário). Pausados não recebem novas solicitações. Ao **retomar**, puxam a fila até a capacidade máxima.
- Atendentes podem ser **removidos** sem ativos ativos (HTTP 409 caso contrário).
- Ao finalizar, o evento carrega `duracao_atendimento_seg` e, se passou pela fila, `tempo_na_fila_seg`.

---

## Por que Go (e não Java)?

A vaga cita Java, mas o desafio deixou a stack livre. A escolha de Go foi técnica e deliberada:

| Critério | Go | Java |
|----------|----|------|
| **Concorrência** | Goroutines + canais nativos — modelo CSP, custo de memória ~2 KB/goroutine | Threads ou Virtual Threads (Java 21+) — maior overhead por unidade |
| **Tempo real (SSE)** | `net/http` da stdlib suporta streaming sem framework | Precisa de Servlet async ou WebFlux |
| **Binário** | Single static binary (~10 MB), ideal para imagem distroless | Precisa de JVM (~200 MB+) |
| **Ruído de infraestrutura** | Stdlib cobre HTTP, JSON, SQL, logs, embed — DDD sem ruído | Spring/Quarkus adicionam camadas que ofuscam o domínio |
| **Race detector** | `go test -race` nativo, detecta data races em tempo de teste | Ferramentas externas (Thread Sanitizer) menos integradas |

O problema central é **concorrência + tempo real**. Goroutines + SSE da stdlib resolvem isso
com o mínimo de código, deixando o domínio no centro — exatamente o que o DDD busca.

---

## Arquitetura

### Padrões arquiteturais

#### DDD — Domain-Driven Design

O projeto implementa DDD de forma rigorosa dentro de um único bounded context: **Atendimento**.

**Linguagem Ubíqua em pt-BR — escolha intencional.** Todas as entidades, value objects,
eventos e métodos do domínio usam português brasileiro. Isso não é estilístico: é DDD.
A linguagem ubíqua existe para **eliminar a tradução** entre o vocabulário do negócio e o
código. Quando o PO fala "solicitação aguardando na fila do time de Cartões", o código diz
exatamente `time.Receber(solicitacao)` e o evento retornado é `EvSolicitacaoEnfileirada`.
Não há glossário mental para manter.

```
Linguagem do negócio          Código (domínio Go)
─────────────────────         ──────────────────────────────
"solicitação"             →   type Solicitacao struct
"time de Cartões"         →   TimeCartoes NomeTime = "CARTOES"
"atendente lotado"        →   ErrAtendenteLotado
"enfileirar"              →   EvSolicitacaoEnfileirada
"puxa da fila"            →   t.puxarFila(a)
"pausar atendente"        →   tm.PausarAtendente(id)
```

**Aggregate Root — `domain.Time`**

`Time` é o aggregate root. Ele concentra os atendentes e a fila e é o **único** lugar onde
as invariantes são garantidas. Nenhum serviço de aplicação externo pode violar as regras:

```
domain.Time
├── Receber(Solicitacao) → atribui OU enfileira
├── Finalizar(id) → libera vaga + puxa fila (FIFO)
├── AdicionarAtendente(a) → insere + puxa fila
├── PausarAtendente(id) → falha com ErrAtendenteComAtivos se há ativos
├── RetomarAtendente(id) → reativa + puxa fila
└── RemoverAtendente(id) → falha com ErrAtendenteComAtivos se há ativos
```

**Value Object — `Assunto`**

`Assunto` encapsula o roteamento assunto → time. A regra de negócio vive no VO,
não num switch espalhado pela aplicação.

**Erros de domínio como sentinelas**

```go
var (
    ErrAssuntoInvalido          = errors.New("assunto inválido")
    ErrSolicitacaoNaoEncontrada = errors.New("solicitação em atendimento não encontrada")
    ErrAtendenteNaoEncontrado   = errors.New("atendente não encontrado")
    ErrAtendenteComAtivos       = errors.New("atendente possui atendimentos ativos")
)
```

Erros são sentinelas (`var Err... = errors.New(...)`), nunca strings soltas. Os handlers HTTP
usam `errors.Is()` para traduzir domínio → status code (404, 409, 422) sem acoplar o domínio
ao HTTP.

**Eventos de domínio**

Cada operação do aggregate retorna eventos (`[]Evento`). Esses eventos alimentam dois
consumidores: o repositório Postgres (log de auditoria) e o hub SSE (tempo real para o
dashboard). O aggregate não sabe quem consome — ele só produz.

```
SolicitacaoRecebida → SolicitacaoAtribuida | SolicitacaoEnfileirada
AtendimentoFinalizado (com duracao_atendimento_seg + tempo_na_fila_seg)
AtendentePausado | AtendenteRetomado | AtendenteRemovido
```

---

#### Arquitetura Hexagonal (Ports & Adapters)

As dependências apontam sempre **para dentro**. O domínio é Go puro, sem nenhum import de infra.

```
┌─────────────────────────────────────────────────────────┐
│  Infrastructure (adapters)                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────┐  │
│  │ HTTP     │  │ Postgres │  │ Memória  │  │  SSE  │  │
│  │ handlers │  │ reposit. │  │ reposit. │  │  hub  │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬───┘  │
│       │              │              │             │      │
│  ┌────▼──────────────▼──────────────▼─────────────▼──┐  │
│  │  Application (casos de uso)                       │  │
│  │  Distribuidor — orquestra Times em memória        │  │
│  │  Portas: Repositorio (port) + Publicador (port)   │  │
│  └─────────────────────┬─────────────────────────────┘  │
│                        │                                 │
│  ┌─────────────────────▼─────────────────────────────┐  │
│  │  Domain (núcleo puro)                             │  │
│  │  Time · Atendente · Solicitacao · Assunto         │  │
│  │  Eventos · Erros · Repositorio (interface/port)   │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

**Ports** são interfaces Go no pacote `domain`:

```go
// domain/repositorio.go — PORT de persistência
type Repositorio interface {
    SalvarSolicitacao(ctx context.Context, s Solicitacao) error
    RegistrarEvento(ctx context.Context, e Evento) error
    HistoricoEventos(ctx context.Context, limite int) ([]Evento, error)
}

// application/portas.go — PORT de tempo real
type Publicador interface {
    Publicar(e Evento)
}
```

**Adapters** implementam os ports na infra: `infrastructure/postgres`, `infrastructure/memoria`,
`infrastructure/sse`. Trocáveis sem tocar no domínio ou na aplicação.

---

### Design Patterns

| Pattern | Onde | Como se manifesta |
|---------|------|-------------------|
| **Aggregate Root** | `domain.Time` | Ponto único de entrada para todas as mutações; garante as invariantes |
| **Value Object** | `domain.Assunto` | Imutável, sem identidade, encapsula roteamento assunto→time |
| **Domain Events** | `domain.Evento` | Aggregate retorna `[]Evento`; desacopla produção de consumo |
| **Repository** | `domain.Repositorio` | Interface (port) no domínio; implementações concretas na infra |
| **Port & Adapter** | `Repositorio` + `Publicador` | Domínio define a interface, infra implementa |
| **Sentinel Errors** | `domain/erros.go` | `var ErrX = errors.New(...)` + `errors.Is()` nos handlers |
| **Composition Root** | `cmd/api/main.go` | Todo o wiring (DI manual) em um único ponto |
| **Middleware Chain** | `infrastructure/http` | CORS → Rate Limiter → Router; cada middleware faz uma coisa |
| **Hub/Broker** | `infrastructure/sse/hub.go` | Fan-out de eventos para múltiplos clientes SSE via canais |
| **In-Memory State** | `application.Distribuidor` | Estado vivo dos times em memória (autoridade para concorrência) |
| **Mutex Guard** | `application.Distribuidor` | `sync.RWMutex` protege o mapa de times; race detector valida |
| **Token Bucket** | `infrastructure/http/middleware.go` | Rate limiter por IP implementado com stdlib pura |
| **Embed FS** | `infrastructure/http/swagger.go` | `//go:embed docs/openapi.yaml` — spec no binário, zero arquivo externo |
| **Facade** | `application.Distribuidor` | Expõe casos de uso simples ocultando a complexidade do aggregate |
| **Table-Driven Tests** | `*_test.go` | Casos de borda como tabela; evita duplicação de lógica de teste |
| **Fake (Test Double)** | `application/*_test.go` | Ports fake in-memory nos testes de aplicação; sem mock gerado |

---

## Stack técnica

### Backend — Go 1.22

| Componente | Escolha | Motivo |
|------------|---------|--------|
| HTTP router | `net/http` ServeMux (stdlib) | Go 1.22 roteia por método + path params nativamente |
| Tempo real | SSE via `net/http` (stdlib) | Push server→client sem WebSocket; `EventSource` é suficiente |
| Banco | `database/sql` + `lib/pq` | Driver puro, zero deps transitivas |
| Logs | `log/slog` (stdlib) | Structured logging sem deps desde Go 1.21 |
| Migrations | `goose` | Schema idempotente; suporte a down migrations |
| Docs API | `swaggo/http-swagger` | Swagger UI embutida, spec YAML via `//go:embed` |
| BDD runner | `github.com/cucumber/godog` | Executa `.feature` Gherkin contra o aggregate real |
| DI | Manual (composition root) | Wiring explícito; sem magic, sem reflexão |

**Dependências externas: 3.** Nada mais. `chi` (router), `pgx`/`sqlc` (DB type-safe) e
frameworks de DI são alternativas drop-in; optei pela stdlib para um build sem fricção e
para deixar o domínio no centro da atenção.

### Frontend — React + Vite + TypeScript

| Componente | Biblioteca |
|------------|-----------|
| Estado servidor | TanStack Query |
| Tempo real | `EventSource` nativo → patch no cache do Query |
| UI components | shadcn/ui + Tailwind CSS |
| Gráficos | Recharts |
| Testes de componente | Vitest + Testing Library |

### Infraestrutura

| Componente | Escolha |
|------------|---------|
| Containerização | Docker multi-stage (backend → `gcr.io/distroless/static`; frontend → nginx) |
| Orquestração local | docker-compose com `healthcheck` no Postgres |
| Banco de dados | PostgreSQL 16 |
| Proxy frontend | nginx com `proxy_buffering off` para SSE |

---

## Segurança

### CORS

Middleware configurável via `ALLOWED_ORIGINS`. Suporta dois modos:

- **Wildcard (`*`)**: dev/demo — qualquer origem aceita.
- **Lista explícita**: modo produção — valida o header `Origin` e reflete apenas origens
  conhecidas. Adiciona `Vary: Origin` para que caches HTTP não confundam respostas entre
  origens distintas. Preflight cacheado por 24h (`Access-Control-Max-Age: 86400`).

### Rate Limiter

**Token bucket por IP**, implementado com `sync.Mutex` + `map` da stdlib — zero deps extras.

- 10 req/s com burst de 30 (configurável via env).
- Responde `429 Too Many Requests` com header `Retry-After`.
- **Exclusões**: `/api/eventos` (SSE — conexão persistente, não é request repetida),
  `/api/docs/` (assets estáticos), `OPTIONS` (preflight CORS não é uso da API).
- Goroutine de limpeza remove buckets inativos a cada 5 min (sem memory leak).

---

## Documentação da API — OpenAPI 3.0 / Swagger

A spec OpenAPI 3.0 é escrita à mão em YAML (`infrastructure/http/docs/openapi.yaml`)
e embutida no binário via `//go:embed`. Não há geração de código — a spec é a fonte
da verdade e evolui junto com os handlers.

```go
//go:embed docs/openapi.yaml
var specFS embed.FS
```

A UI Swagger é servida por `swaggo/http-swagger` apontando para o endpoint da spec
embutida. Isso garante que a doc e o binário são a mesma versão — sem dessincronização.

**Endpoints documentados:** todos os 11 endpoints com schemas de request/response,
respostas de erro reutilizáveis (400, 404, 409) e enumerações (`NomeTime`, `TipoEvento`).

| Rota | Descrição |
|------|-----------|
| `POST /api/solicitacoes` | cria solicitação → `201 { "id" }` |
| `POST /api/solicitacoes/{id}/finalizar` | encerra atendimento → `204` |
| `POST /api/atendentes` | adiciona atendente → `201 { "id" }` |
| `POST /api/atendentes/{id}/pausar` | pausa (sem ativos → `204`; com ativos → `409`) |
| `POST /api/atendentes/{id}/retomar` | retoma pausado, puxa fila → `204` |
| `DELETE /api/atendentes/{id}` | remove sem ativos → `204`; com ativos → `409` |
| `GET /api/dashboard` | snapshot com `pausado`, `solicitacoes_ativas[]` |
| `GET /api/eventos` | stream SSE com `duracao_atendimento_seg` e `tempo_na_fila_seg` |
| `GET /api/historico?limite=N` | últimos N eventos persistidos |
| `GET /api/health` | health check |
| `GET /api/docs/` | Swagger UI interativo |

---

## Migrations

Migrations gerenciadas com **goose**, aplicadas de forma idempotente na inicialização da API.

```
db/migrations/
├── 001_criar_solicitacoes.sql
├── 002_criar_eventos.sql
└── ...
```

Cada migration tem `-- +goose Up` e `-- +goose Down`, permitindo rollback seguro.
O adapter Postgres aplica `goose.Up()` no startup — sem script manual, sem estado fora do repo.

---

## Metodologia de desenvolvimento

### TDD — Test-Driven Development

O workflow seguido foi estritamente **red → green → refactor**:

1. Escreve o teste que falha.
2. Implementa o mínimo para passar.
3. Refatora mantendo o verde.

Os testes seguem a hierarquia de camadas do DDD:

| Camada | Arquivo | O que verifica |
|--------|---------|----------------|
| **Domínio** | `domain/*_test.go` | Invariantes do aggregate: capacidade, balanceamento, fila FIFO, pausar/retomar/remover com `ErrAtendenteComAtivos`, duração e tempo de fila |
| **Aplicação** | `application/distribuidor_test.go` | Casos de uso com ports fake in-memory: registro, finalização, snapshot, métricas de tempo, 409 para pausa/remoção com ativos |
| **HTTP** | `infrastructure/http/integration_test.go` | Status codes, shape do JSON, tradução domínio → HTTP (400/404/409/422), CORS, rate limiter |
| **Componentes** | `frontend/src/components/*.test.tsx` | `TimeCard`: carga, badge fila, badge pausado, botão Pausar desabilitado com ativos. `FinalizadosCard`: agrupamento, duração, ID curto |

```bash
make tdd          # todos os testes Go com race detector
make test-domain  # ciclo TDD rápido no domínio
make test-app     # ciclo TDD na aplicação
make test-http    # ciclo TDD na camada HTTP
```

**`go test -race`** em todos os targets que rodam Go — o race detector pega data races em
tempo de teste, não em produção.

### BDD — Behavior-Driven Development

Os cenários Gherkin em `features/distribuicao.feature` descrevem o comportamento de negócio
**em português**, antes do código existir. São a ponte entre o requisito e o teste:

```gherkin
# language: pt
Cenário: Pausar atendente com atendimentos ativos é bloqueado
  Dado um atendente "Ana" no time "Cartões" com 1 atendimento ativo
  Quando se tenta pausar "Ana"
  Então a operação falha com erro "atendente possui atendimentos ativos"
  E "Ana" permanece não pausada
```

Os steps Go (`backend/features/steps_test.go`) exercitam o **aggregate `Time` diretamente**
— sem HTTP, sem banco, sem Distribuidor. Isso garante que o BDD testa o domínio puro,
não a pilha inteira.

```bash
make bdd   # 19 cenários, godog pretty format
```

O BDD usa build tag `//go:build bdd`, isolado do `make tdd` (que monta só `backend/`).
O `make bdd` usa o runner `GO_FULL` que monta o projeto inteiro e permite que os steps
acessem `../../features` a partir do pacote de teste.

**Cobertura BDD:** distribuição direta, respeito à capacidade, enfileiramento FIFO, pull
automático ao liberar vaga, roteamento de assunto desconhecido, balanceamento por carga,
adicionar atendente em tempo de execução, snapshot de ativos, métricas de duração e fila,
pausar/retomar com invariante de ativos, remoção, pull imediato ao adicionar atendente.

---

## Claude Code como copiloto de desenvolvimento

Todo o desenvolvimento foi feito com **Claude Code** (CLI da Anthropic) como copiloto,
configurado como Tech Lead sênior especializado em Go, DDD e engenharia de software.

A configuração em `.claude/` faz parte do repositório:

| Arquivo/Dir | Propósito |
|-------------|-----------|
| `CLAUDE.md` | Contexto do projeto: stack, arquitetura, linguagem ubíqua, invariantes, regras inegociáveis |
| `.claude/agents/` | Sub-agentes especializados: `go-domain`, `go-backend`, `react-dashboard`, `test-engineer`, `code-reviewer` |
| `.claude/commands/` | Slash commands: `/tdd`, `/bdd`, `/domain`, `/review`, `/migration` |
| `.claude/hooks/` | Hooks automáticos: `gofmt` e `go vet` antes de cada edição Go |

O fluxo de trabalho com o copiloto seguiu as mesmas regras de qualquer engenheiro sênior
no projeto: BDD primeiro (cenário Gherkin), TDD no domínio (red→green→refactor), só depois
sobe para aplicação e infra. Nenhum mock de dados em código de produção. Nenhum atalho
que violasse as invariantes de arquitetura.

O copiloto foi especialmente útil para:
- Manter a consistência da linguagem ubíqua em todas as camadas
- Garantir que a regra de camadas (domínio puro, sem imports de infra) não fosse violada
- Escrever os step definitions BDD em sincronia com os cenários Gherkin
- Implementar o rate limiter com stdlib pura (sem deps adicionais)
- Estruturar os testes table-driven de forma consistente entre todas as camadas

---

## Estrutura de arquivos

```
flowpay/
├── backend/
│   ├── cmd/api/
│   │   ├── main.go          # composition root — wiring de tudo
│   │   └── seed.go          # dados iniciais (atendentes de exemplo)
│   ├── features/            # runner BDD (godog) — build tag: bdd
│   │   ├── bdd_test.go
│   │   └── steps_test.go
│   ├── db/migrations/       # goose migrations
│   └── internal/atendimento/
│       ├── domain/          # NÚCLEO — Go puro, zero deps externas
│       │   ├── time.go      # aggregate root
│       │   ├── atendente.go # entidade
│       │   ├── solicitacao.go
│       │   ├── assunto.go   # value object + roteamento
│       │   ├── eventos.go   # eventos de domínio
│       │   ├── erros.go     # erros sentinela
│       │   └── repositorio.go # port (interface)
│       ├── application/
│       │   ├── distribuidor.go  # orquestra Times em memória (mutex)
│       │   └── portas.go        # port Publicador
│       └── infrastructure/
│           ├── http/
│           │   ├── router.go
│           │   ├── handlers.go
│           │   ├── dto.go
│           │   ├── middleware.go  # CORS + rate limiter
│           │   ├── swagger.go     # embed + Swagger UI
│           │   └── docs/openapi.yaml
│           ├── sse/hub.go         # broker SSE (fan-out via canais)
│           ├── postgres/          # adapter Repositorio (Postgres)
│           └── memoria/           # adapter Repositorio (in-memory)
├── features/
│   └── distribuicao.feature  # cenários BDD em pt-BR (Gherkin)
├── frontend/
│   └── src/
│       ├── components/       # TimeCard, FinalizadosCard, EventFeed
│       └── hooks/            # useSSE, useEventSource
├── docker-compose.yml
├── Makefile
└── .env.example
```
