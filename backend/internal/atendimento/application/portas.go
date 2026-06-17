package application

import (
	"time"

	"flowpay/internal/atendimento/domain"
)

// Publicador é o PORT de saída usado para emitir eventos em tempo real.
// A infraestrutura (hub SSE) implementa esta interface; a aplicação só a conhece.
type Publicador interface {
	Publicar(domain.Evento)
}

// SolicitacaoAtivaView é a projeção de uma solicitação em atendimento,
// incluindo tempo de espera na fila quando a solicitação passou pela fila.
type SolicitacaoAtivaView struct {
	ID             string `json:"id"`
	TempoNaFilaSeg *int64 `json:"tempo_na_fila_seg,omitempty"`
}

// AtendenteView é a projeção de leitura de um atendente para o dashboard.
type AtendenteView struct {
	ID                 string                 `json:"id"`
	Nome               string                 `json:"nome"`
	Ativos             int                    `json:"ativos"`
	Capacidade         int                    `json:"capacidade"`
	Pausado            bool                   `json:"pausado"`
	SolicitacoesAtivas []SolicitacaoAtivaView `json:"solicitacoes_ativas"`
}

type TimeView struct {
	Nome          domain.NomeTime `json:"nome"`
	Atendentes    []AtendenteView `json:"atendentes"`
	TamanhoFila   int             `json:"tamanho_fila"`
	EmAtendimento int             `json:"em_atendimento"`
}

type DashboardView struct {
	Times        []TimeView `json:"times"`
	AtualizadoEm time.Time  `json:"atualizado_em"`
}
