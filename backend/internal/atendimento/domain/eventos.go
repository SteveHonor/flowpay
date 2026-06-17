package domain

import "time"

type TipoEvento string

const (
	EvSolicitacaoRecebida    TipoEvento = "solicitacao_recebida"
	EvSolicitacaoAtribuida   TipoEvento = "solicitacao_atribuida"
	EvSolicitacaoEnfileirada TipoEvento = "solicitacao_enfileirada"
	EvAtendimentoFinalizado  TipoEvento = "atendimento_finalizado"
	EvAtendentePausado       TipoEvento = "atendente_pausado"
	EvAtendenteRetomado      TipoEvento = "atendente_retomado"
	EvAtendenteRemovido      TipoEvento = "atendente_removido"
)

// Evento de domínio. Estrutura única e serializável para transmissão via SSE.
type Evento struct {
	Tipo          TipoEvento `json:"tipo"`
	SolicitacaoID string     `json:"solicitacao_id,omitempty"`
	Time          NomeTime   `json:"time"`
	AtendenteID   string     `json:"atendente_id,omitempty"`
	AtendenteNome string     `json:"atendente_nome,omitempty"`
	PosicaoFila   int        `json:"posicao_fila,omitempty"`
	Ocorreu       time.Time  `json:"ocorreu"`
	// Métricas de tempo — presentes apenas nos eventos relevantes.
	DuracaoAtendimentoSeg *int64 `json:"duracao_atendimento_seg,omitempty"` // AtendimentoFinalizado
	TempoNaFilaSeg        *int64 `json:"tempo_na_fila_seg,omitempty"`        // SolicitacaoAtribuida (da fila) + AtendimentoFinalizado (se veio da fila)
}
