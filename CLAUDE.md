# FlowPay — Distribuição e Monitoramento de Atendimentos

Projeto de desafio técnico (Tech Lead Fullstack). **Excelência técnica em todas as
camadas** é o critério de avaliação, não velocidade bruta. Cada decisão deve ser
defensável em entrevista.

## Stack

**Backend (Go 1.22+)** — enxuto em dependências (3 deps externas permitidas):

| Dependência | Pacote | Motivo |
|-------------|--------|--------|
| Driver Postgres | `github.com/lib/pq` | única dependência de runtime; `database/sql` da stdlib precisa de driver |
| Runner BDD | `github.com/cucumber/godog` | executa os cenários `.feature` (Gherkin) contra o domínio real; só em teste |
| Swagger/OpenAPI | `github.com/swaggo/swag` + `github.com/swaggo/http-swagger` | gera e serve a UI interativa da API; só na camada `infrastructure/http` |

- HTTP: **`net/http` + ServeMux da stdlib** (Go 1.22 já roteia por método + path params). `chi` é drop-in se quiser, mas a stdlib basta e zera dependência.
- Tempo real: **SSE** (Server-Sent Events) via stdlib — dashboard é stream server→client
- Persistência: PostgreSQL via **`database/sql` + `lib/pq`** (driver puro, sem deps transitivas). `pgx`/`sqlc` são alternativas se quiser type-safety gerada.
- Estado vivo da distribuição: **em memória** (autoridade para o tempo real/concorrência); o Postgres persiste solicitações + log de eventos para histórico.
- Migrations: `goose` (schema também aplicado de forma idempotente pelo adapter)
- Log: `log/slog` (stdlib)
- Testes: **`testing` da stdlib**, table-driven, `go test -race`. **`godog`**/Gherkin é o runner BDD; os cenários do `.feature` estão espelhados 1:1 nos testes do domínio.
- DI: wiring manual no `cmd/api/main.go` (composition root). Sem framework de DI.

**Frontend (React + Vite + TypeScript)**
- Server state: TanStack Query
- Tempo real: `EventSource` (SSE) → patch no cache do Query
- UI: Tailwind + shadcn/ui
- Gráficos: Recharts
- Testes: Vitest + Testing Library (componente), Playwright (E2E/BDD)

## Arquitetura — DDD + Hexagonal

Bounded context único: **Atendimento**. O domínio é Go puro, sem imports de infra.
Dependências apontam para dentro: `infrastructure → application → domain`.

```
backend/
  cmd/api/main.go              # composition root: wiring de tudo
  internal/atendimento/
    domain/                    # Go PURO. Zero deps externas.
      assunto.go               # VO: PROBLEMA_CARTAO | CONTRATACAO_EMPRESTIMO | OUTROS
      atendente.go             # entidade
      time.go                  # AGGREGATE ROOT: dono das invariantes
      solicitacao.go           # entidade
      eventos.go               # eventos de domínio
      erros.go                 # erros de domínio (sentinelas)
      repositorio.go           # PORTS (interfaces)
    application/
      registrar_solicitacao.go # caso de uso
      finalizar_atendimento.go
      consultar_dashboard.go
      distribuidor.go          # orquestra distribuição + publica eventos
    infrastructure/
      postgres/                # ADAPTERS: implementam os ports
      sse/                     # hub/broker SSE
      http/                    # handlers chi, DTOs, router
  features/                    # arquivos .feature (godog)
  db/migrations/               # goose
```

## Linguagem Ubíqua (PT — é a linguagem do negócio FlowPay)

- **Solicitação**: pedido de atendimento. Status: `AGUARDANDO` → `EM_ATENDIMENTO` → `FINALIZADA`.
- **Assunto** (VO): `PROBLEMA_CARTAO`, `CONTRATACAO_EMPRESTIMO`, `OUTROS`.
- **Time** (aggregate root): `CARTOES`, `EMPRESTIMOS`, `OUTROS`. Dono dos atendentes e da fila.
- **Atendente**: pertence a um Time. `CAPACIDADE_MAXIMA = 3` simultâneos.
- **Fila**: por Time, FIFO. Usada quando todos os atendentes estão lotados.

**Roteamento Assunto → Time:** PROBLEMA_CARTAO→CARTOES, CONTRATACAO_EMPRESTIMO→EMPRESTIMOS, resto→OUTROS.

## Invariantes (vivem DENTRO do aggregate Time)

1. Nenhum atendente ultrapassa 3 atendimentos simultâneos.
2. Se nenhum atendente do Time tem vaga → a solicitação é **enfileirada**.
3. Ao finalizar um atendimento, a vaga liberada **puxa o próximo da fila** (FIFO), se houver.

A política de distribuição NÃO mora em service de aplicação espalhado. Mora no Time:
`time.Receber(solicitacao)` e `time.Finalizar(solicitacaoID)` retornam eventos de domínio.

## Eventos de Domínio (alimentam o SSE)

`SolicitacaoRecebida`, `SolicitacaoAtribuida`, `SolicitacaoEnfileirada`, `AtendimentoFinalizado`.

## Workflow obrigatório — TDD + BDD

**Toda feature de domínio segue red → green → refactor.**

1. **BDD primeiro** (regra de negócio): escreva/atualize o `.feature` em Gherkin (`# language: pt`)
   descrevendo o comportamento ANTES do código. Os steps godog exercitam o aggregate real.
2. **TDD no domínio**: teste de unidade table-driven falhando → implementação mínima → refatora.
3. Só depois sobe para application e infra.

**Backend-first, sempre.** Nada de frontend antes do endpoint existir e ter teste passando.

## Regras inegociáveis

- **SEM dados mockados** em código de produção. Mock só em teste, e ainda assim prefira
  fakes/in-memory dos ports a mocks gerados.
- **SEM atalho "só TypeScript"**: o backend Go é a fonte da verdade. O front consome a API real.
- Erros de domínio são sentinelas (`var ErrAtendenteLotado = errors.New(...)`), nunca strings soltas.
- Domínio não importa `chi`, `pgx`, `context` de framework. Só stdlib + tipos próprios.
- Concorrência: o estado vivo do `Time` é protegido por mutex OU serializado por uma goroutine
  dispatcher por time (canais). Escolha uma e seja consistente. Race detector (`go test -race`) sempre.
- Commits pequenos, mensagem no imperativo. Um commit = um passo verde do TDD quando possível.

## Comandos do projeto

```
make test          # go test -race ./... + vitest
make bdd           # godog ./features
make lint          # golangci-lint + eslint
make up            # docker compose up (postgres, redis, api)
make migrate       # goose up
sqlc generate      # regenera queries type-safe
```

## Como me dirigir aqui

- Antes de implementar, confirme em qual camada o código entra (domain/application/infra).
- Se eu pedir algo que viole uma invariante ou a regra de camadas, **aponte o conflito** antes de codar.
- Use os subagents: `go-domain` para modelagem, `go-backend` para infra/HTTP, `test-engineer`
  para TDD/BDD, `react-dashboard` para o front, `code-reviewer` antes de eu commitar.
