package domain

import "time"

// CapacidadeMaxima é o número máximo de atendimentos simultâneos por atendente.
const CapacidadeMaxima = 3

// registroAtendimento guarda o instante de início e, se a solicitação veio da fila,
// quando ela entrou na fila — para calcular durações ao finalizar.
type registroAtendimento struct {
	inicio        time.Time
	enfileiradaEm *time.Time
}

// Atendente pertence a um Time e atende no máximo CapacidadeMaxima solicitações ao mesmo tempo.
type Atendente struct {
	ID            string
	Nome          string
	Time          NomeTime
	pausado       bool
	emAtendimento map[string]registroAtendimento
}

func NovoAtendente(id, nome string, time NomeTime) *Atendente {
	return &Atendente{
		ID:            id,
		Nome:          nome,
		Time:          time,
		emAtendimento: make(map[string]registroAtendimento),
	}
}

func (a *Atendente) Ativos() int { return len(a.emAtendimento) }

func (a *Atendente) TemVaga() bool { return a.Ativos() < CapacidadeMaxima }

// EstaDisponivel indica se o atendente pode receber novas solicitações:
// não está pausado E ainda tem vaga de capacidade.
func (a *Atendente) EstaDisponivel() bool { return !a.pausado && a.TemVaga() }

func (a *Atendente) Pausado() bool { return a.pausado }

func (a *Atendente) Pausar()  { a.pausado = true }
func (a *Atendente) Retomar() { a.pausado = false }

func (a *Atendente) Atende(solicitacaoID string) bool {
	_, ok := a.emAtendimento[solicitacaoID]
	return ok
}

func (a *Atendente) SolicitacoesAtivas() []string {
	ids := make([]string, 0, len(a.emAtendimento))
	for id := range a.emAtendimento {
		ids = append(ids, id)
	}
	return ids
}

// InfoAtiva é a projeção de uma solicitação ativa, incluindo tempo de fila se aplicável.
type InfoAtiva struct {
	ID             string
	TempoNaFilaSeg *int64 // nil se a solicitação não passou pela fila
}

// InfoSolicitacoesAtivas retorna as solicitações ativas com metadados de fila,
// permitindo que a camada de aplicação projete o snapshot com informação de espera.
func (a *Atendente) InfoSolicitacoesAtivas() []InfoAtiva {
	result := make([]InfoAtiva, 0, len(a.emAtendimento))
	for id, reg := range a.emAtendimento {
		info := InfoAtiva{ID: id}
		if reg.enfileiradaEm != nil {
			seg := int64(reg.inicio.Sub(*reg.enfileiradaEm).Seconds())
			info.TempoNaFilaSeg = &seg
		}
		result = append(result, info)
	}
	return result
}

// Atribuir associa uma solicitação ao atendente, registrando o instante de início e,
// opcionalmente, quando ela ficou na fila (para cálculo posterior do tempo de espera).
func (a *Atendente) Atribuir(solicitacaoID string, enfileiradaEm *time.Time) error {
	if !a.TemVaga() {
		return ErrAtendenteLotado
	}
	a.emAtendimento[solicitacaoID] = registroAtendimento{
		inicio:        time.Now().UTC(),
		enfileiradaEm: enfileiradaEm,
	}
	return nil
}

// Liberar remove a solicitação e devolve:
//   - duracao: tempo total de atendimento
//   - tempoNaFila: tempo que ficou na fila antes de ser atendida (nil se não foi enfileirada)
//   - ok: false se o atendente não atendia essa solicitação
func (a *Atendente) Liberar(solicitacaoID string) (duracao time.Duration, tempoNaFila *time.Duration, ok bool) {
	reg, ok := a.emAtendimento[solicitacaoID]
	if !ok {
		return 0, nil, false
	}
	delete(a.emAtendimento, solicitacaoID)
	duracao = time.Since(reg.inicio)
	if reg.enfileiradaEm != nil {
		d := reg.inicio.Sub(*reg.enfileiradaEm)
		tempoNaFila = &d
	}
	return duracao, tempoNaFila, true
}
