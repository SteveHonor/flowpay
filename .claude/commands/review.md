---
description: Revisa o diff staged contra as regras de DDD, camadas e testes antes do commit
allowed-tools: Bash(git diff *), Bash(git status), Bash(go test *), Read, Grep, Glob
---

Aja como o subagent **code-reviewer**.

Diff a revisar:

```
!`git diff --staged`
```

Aplique o checklist completo (DDD/camadas, testes/TDD-BDD, qualidade). Rode `go test -race ./...`
se fizer sentido. Saída com 🔴 bloqueadores, 🟡 sugestões, 🟢 elogios, citando arquivo e linha.
Não edite o código.
