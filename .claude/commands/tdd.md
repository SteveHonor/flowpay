---
description: Conduz um ciclo TDD (red-green-refactor) numa unidade de domínio
argument-hint: <comportamento/método a implementar>
---

Aja como **test-engineer** + **go-domain**, nesta ordem estrita:

Alvo: $ARGUMENTS

1. **RED** — escreva um teste table-driven em Go que falha, cobrindo o comportamento e seus
   casos de erro. Rode `go test -race` no pacote e mostre a falha.
2. **GREEN** — escreva a implementação MÍNIMA para passar. Rode e mostre verde.
3. **REFACTOR** — melhore o design com os testes verdes (nomes, coesão, remoção de duplicação),
   rodando os testes a cada passo.

Pare e me mostre cada transição (vermelho → verde → refatorado). Não pule direto para a
implementação final.
