---
description: Escreve um cenário BDD (Gherkin pt + steps godog) a partir de um requisito, antes da implementação
argument-hint: <descrição do comportamento de negócio>
---

Aja como o subagent **test-engineer**.

Requisito: $ARGUMENTS

1. Escreva (ou estenda) o arquivo `.feature` apropriado em `features/`, com `# language: pt`,
   cobrindo o comportamento acima em cenários Dado/Quando/Então. Inclua os casos de borda
   relevantes (lotação, fila, redistribuição).
2. Implemente os steps godog em Go exercitando o **aggregate real** do domínio (sem mockar o
   próprio domínio).
3. Rode `godog ./features` e mostre o resultado VERMELHO (cenário pendente/falhando) — ainda
   não implemente a regra. O vermelho é o ponto de partida.

Não escreva a lógica de produção neste passo. Só o cenário e os steps.
