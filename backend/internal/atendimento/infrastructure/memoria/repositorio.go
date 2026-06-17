package memoria

import (
	"context"
	"sync"

	"flowpay/internal/atendimento/domain"
)

// Repositorio em memória. Adapter padrão quando não há banco configurado.
type Repositorio struct {
	mu           sync.Mutex
	solicitacoes map[string]domain.Solicitacao
	eventos      []domain.Evento
}

func NovoRepositorio() *Repositorio {
	return &Repositorio{solicitacoes: make(map[string]domain.Solicitacao)}
}

func (r *Repositorio) SalvarSolicitacao(_ context.Context, s domain.Solicitacao) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.solicitacoes[s.ID] = s
	return nil
}

func (r *Repositorio) RegistrarEvento(_ context.Context, e domain.Evento) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.eventos = append(r.eventos, e)
	return nil
}

func (r *Repositorio) HistoricoEventos(_ context.Context, limite int) ([]domain.Evento, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(r.eventos)
	if limite > 0 && limite < n {
		out := make([]domain.Evento, limite)
		copy(out, r.eventos[n-limite:])
		return out, nil
	}
	out := make([]domain.Evento, n)
	copy(out, r.eventos)
	return out, nil
}
