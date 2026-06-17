---
description: Modela um conceito de domínio (entidade/VO/aggregate/evento) seguindo o DDD do projeto
argument-hint: <conceito a modelar>
---

Aja como o subagent **go-domain**.

Conceito: $ARGUMENTS

1. Declare a categoria tática (Value Object, Entidade, Aggregate Root ou Evento) e justifique.
2. Liste as invariantes que esse conceito carrega e onde serão garantidas.
3. Proponha primeiro o teste de unidade que prova as invariantes (chame /tdd se for o caso).
4. Implemente em `internal/atendimento/domain/`, Go puro, erros como sentinelas, comportamento
   junto dos dados.

Se o conceito quebrar os limites do aggregate Time, sinalize antes de codar.
