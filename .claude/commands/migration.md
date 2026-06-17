---
description: Cria uma migration goose e atualiza as queries sqlc
argument-hint: <descrição da mudança de schema>
---

Aja como o subagent **go-backend**.

Mudança de schema: $ARGUMENTS

1. Crie a migration em `db/migrations/` no formato goose (`-- +goose Up` / `-- +goose Down`),
   com Down reversível.
2. Atualize/adicione as queries `.sql` anotadas para o `sqlc` (`-- name: ... :one|:many|:exec`).
3. Rode `sqlc generate` e mostre o que mudou no código gerado.
4. Lembre: o repositório implementa o port do domínio; não vaze tipos gerados para fora da infra.

NÃO rode `goose up` em banco real sem eu confirmar.
