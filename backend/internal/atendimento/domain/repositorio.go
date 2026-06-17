package domain

import "context"

// Repositorio é o PORT de persistência. A infraestrutura fornece adapters
// (em memória, Postgres). O domínio só conhece esta interface.
type Repositorio interface {
	SalvarSolicitacao(ctx context.Context, s Solicitacao) error
	RegistrarEvento(ctx context.Context, e Evento) error
	HistoricoEventos(ctx context.Context, limite int) ([]Evento, error)
}
