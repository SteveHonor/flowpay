package domain

import (
	"slices"
	"time"
)

// Time é o aggregate root. Concentra os atendentes e a fila, e é o ÚNICO
// lugar onde as invariantes de distribuição são garantidas:
//  1. nenhum atendente passa de CapacidadeMaxima simultâneos;
//  2. se ninguém tem vaga, a solicitação é enfileirada;
//  3. ao liberar uma vaga, a próxima da fila (FIFO) é atribuída.
type Time struct {
	Nome       NomeTime
	atendentes []*Atendente
	fila       []Solicitacao
}

func NovoTime(nome NomeTime, atendentes ...*Atendente) *Time {
	return &Time{Nome: nome, atendentes: atendentes}
}

func (t *Time) Atendentes() []*Atendente { return t.atendentes }

func (t *Time) TamanhoFila() int { return len(t.fila) }

// Receber aplica a política de distribuição a uma solicitação destinada a este time.
// Retorna os eventos de domínio resultantes (atribuída OU enfileirada).
func (t *Time) Receber(s Solicitacao) ([]Evento, error) {
	if a := t.atendenteComVaga(); a != nil {
		return t.atribuir(a, s), nil
	}
	agora := time.Now().UTC()
	s.Status = StatusAguardando
	s.EnfileiradaEm = &agora
	t.fila = append(t.fila, s)
	return []Evento{{
		Tipo:          EvSolicitacaoEnfileirada,
		SolicitacaoID: s.ID,
		Time:          t.Nome,
		PosicaoFila:   len(t.fila),
		Ocorreu:       agora,
	}}, nil
}

// Finalizar encerra um atendimento e, se houver fila, puxa a próxima solicitação
// para o atendente que acabou de liberar a vaga.
func (t *Time) Finalizar(solicitacaoID string) ([]Evento, error) {
	a := t.atendenteDe(solicitacaoID)
	if a == nil {
		return nil, ErrSolicitacaoNaoEncontrada
	}
	duracao, tempoNaFila, _ := a.Liberar(solicitacaoID)

	duracaoSeg := int64(duracao.Seconds())
	ev := Evento{
		Tipo:                  EvAtendimentoFinalizado,
		SolicitacaoID:         solicitacaoID,
		Time:                  t.Nome,
		AtendenteID:           a.ID,
		AtendenteNome:         a.Nome,
		Ocorreu:               time.Now().UTC(),
		DuracaoAtendimentoSeg: &duracaoSeg,
	}
	if tempoNaFila != nil {
		seg := int64(tempoNaFila.Seconds())
		ev.TempoNaFilaSeg = &seg
	}
	eventos := []Evento{ev}

	if len(t.fila) > 0 && a.EstaDisponivel() {
		proxima := t.fila[0]
		t.fila = t.fila[1:]
		eventos = append(eventos, t.atribuir(a, proxima)...)
	}
	return eventos, nil
}

// AdicionarAtendente insere um novo atendente no time em tempo de execução
// e imediatamente puxa solicitações da fila enquanto houver capacidade.
func (t *Time) AdicionarAtendente(a *Atendente) []Evento {
	t.atendentes = append(t.atendentes, a)
	return t.puxarFila(a)
}

// PausarAtendente suspende o atendente para novas atribuições.
// Falha com ErrAtendenteComAtivos se o atendente ainda tiver atendimentos em andamento:
// ele deve finalizar os ativos antes de ser pausado.
func (t *Time) PausarAtendente(id string) (Evento, error) {
	a := t.encontrarAtendente(id)
	if a == nil {
		return Evento{}, ErrAtendenteNaoEncontrado
	}
	if a.Ativos() > 0 {
		return Evento{}, ErrAtendenteComAtivos
	}
	a.Pausar()
	return Evento{
		Tipo:          EvAtendentePausado,
		Time:          t.Nome,
		AtendenteID:   a.ID,
		AtendenteNome: a.Nome,
		Ocorreu:       time.Now().UTC(),
	}, nil
}

// RetomarAtendente reativa um atendente pausado e puxa solicitações da fila
// até ele atingir a capacidade máxima ou a fila esvaziar.
func (t *Time) RetomarAtendente(id string) ([]Evento, error) {
	a := t.encontrarAtendente(id)
	if a == nil {
		return nil, ErrAtendenteNaoEncontrado
	}
	a.Retomar()
	eventos := []Evento{{
		Tipo:          EvAtendenteRetomado,
		Time:          t.Nome,
		AtendenteID:   a.ID,
		AtendenteNome: a.Nome,
		Ocorreu:       time.Now().UTC(),
	}}
	eventos = append(eventos, t.puxarFila(a)...)
	return eventos, nil
}

// RemoverAtendente remove o atendente do time. Falha se ele tiver ativos.
func (t *Time) RemoverAtendente(id string) (Evento, error) {
	a := t.encontrarAtendente(id)
	if a == nil {
		return Evento{}, ErrAtendenteNaoEncontrado
	}
	if a.Ativos() > 0 {
		return Evento{}, ErrAtendenteComAtivos
	}
	t.atendentes = slices.DeleteFunc(t.atendentes, func(x *Atendente) bool { return x.ID == id })
	return Evento{
		Tipo:          EvAtendenteRemovido,
		Time:          t.Nome,
		AtendenteID:   a.ID,
		AtendenteNome: a.Nome,
		Ocorreu:       time.Now().UTC(),
	}, nil
}

// atribuir associa a solicitação ao atendente e devolve o evento correspondente.
// Pré-condição: a.EstaDisponivel() == true.
func (t *Time) atribuir(a *Atendente, s Solicitacao) []Evento {
	_ = a.Atribuir(s.ID, s.EnfileiradaEm)
	ev := Evento{
		Tipo:          EvSolicitacaoAtribuida,
		SolicitacaoID: s.ID,
		Time:          t.Nome,
		AtendenteID:   a.ID,
		AtendenteNome: a.Nome,
		Ocorreu:       time.Now().UTC(),
	}
	if s.EnfileiradaEm != nil {
		seg := int64(time.Since(*s.EnfileiradaEm).Seconds())
		ev.TempoNaFilaSeg = &seg
	}
	return []Evento{ev}
}

// puxarFila tira solicitações da fila e atribui ao atendente até ele não ter mais vaga.
func (t *Time) puxarFila(a *Atendente) []Evento {
	var eventos []Evento
	for a.EstaDisponivel() && len(t.fila) > 0 {
		proxima := t.fila[0]
		t.fila = t.fila[1:]
		eventos = append(eventos, t.atribuir(a, proxima)...)
	}
	return eventos
}

// atendenteComVaga retorna o atendente disponível (não pausado, com vaga) de menor carga.
func (t *Time) atendenteComVaga() *Atendente {
	var escolhido *Atendente
	for _, a := range t.atendentes {
		if a.EstaDisponivel() {
			if escolhido == nil || a.Ativos() < escolhido.Ativos() {
				escolhido = a
			}
		}
	}
	return escolhido
}

func (t *Time) atendenteDe(solicitacaoID string) *Atendente {
	for _, a := range t.atendentes {
		if a.Atende(solicitacaoID) {
			return a
		}
	}
	return nil
}

func (t *Time) encontrarAtendente(id string) *Atendente {
	for _, a := range t.atendentes {
		if a.ID == id {
			return a
		}
	}
	return nil
}
