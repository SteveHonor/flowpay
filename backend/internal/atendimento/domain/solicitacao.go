package domain

import "time"

type Status string

const (
	StatusAguardando    Status = "AGUARDANDO"
	StatusEmAtendimento Status = "EM_ATENDIMENTO"
	StatusFinalizada    Status = "FINALIZADA"
)

// Solicitacao é o pedido de atendimento de um cliente.
type Solicitacao struct {
	ID            string
	Assunto       Assunto
	Time          NomeTime
	Status        Status
	AtendenteID   string
	CriadaEm      time.Time
	EnfileiradaEm *time.Time // preenchido quando a solicitação entra na fila
}

func NovaSolicitacao(id string, assunto Assunto) Solicitacao {
	return Solicitacao{
		ID:       id,
		Assunto:  assunto,
		Time:     assunto.Time(),
		Status:   StatusAguardando,
		CriadaEm: time.Now().UTC(),
	}
}
