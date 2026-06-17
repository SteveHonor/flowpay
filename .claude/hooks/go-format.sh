#!/usr/bin/env bash
# PostToolUse hook (matcher Edit|Write).
# Recebe JSON do Claude Code via stdin. Só age em arquivos .go.
# gofmt -w no arquivo + go vet no pacote. Não bloqueia o fluxo (exit 0).
set -euo pipefail

input="$(cat)"
file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // .tool_input.path // empty')"

[[ -z "$file" || "$file" != *.go ]] && exit 0
[[ ! -f "$file" ]] && exit 0

gofmt -w "$file" 2>/dev/null || true
command -v goimports >/dev/null 2>&1 && goimports -w "$file" 2>/dev/null || true

# vet só do pacote do arquivo, silencioso; avisos vão pro stderr (não bloqueia)
pkg_dir="$(dirname "$file")"
( cd "$pkg_dir" && go vet ./ 2>&1 ) || true

exit 0
