package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"flowpay/internal/atendimento/domain"
)

// Distribuidor é o coração da aplicação: mantém o estado vivo dos times em memória
// (autoridade para o tempo real e a concorrência) e orquestra persistência + publicação.
//
// Toda mutação passa pelo mutex, serializando o acesso aos aggregates. O domínio
// permanece livre de concorrência; o controle de acesso é responsabilidade desta camada.
type Distribuidor struct {
	mu             sync.Mutex
	times          map[domain.NomeTime]*domain.Time
	timePorID      map[string]domain.NomeTime // solicitacaoID → time
	atendentePorID map[string]domain.NomeTime // atendenteID → time
	repo           domain.Repositorio
	pub            Publicador
}

func NovoDistribuidor(repo domain.Repositorio, pub Publicador, times ...*domain.Time) *Distribuidor {
	d := &Distribuidor{
		times:          make(map[domain.NomeTime]*domain.Time),
		timePorID:      make(map[string]domain.NomeTime),
		atendentePorID: make(map[string]domain.NomeTime),
		repo:           repo,
		pub:            pub,
	}
	for _, t := range times {
		d.times[t.Nome] = t
		for _, a := range t.Atendentes() {
			d.atendentePorID[a.ID] = t.Nome
		}
	}
	return d
}

// RegistrarSolicitacao cria uma solicitação, roteia para o time correto e aplica a
// política de distribuição. Retorna o ID gerado.
func (d *Distribuidor) RegistrarSolicitacao(ctx context.Context, codigoAssunto string) (string, error) {
	assunto, err := domain.NovoAssunto(codigoAssunto)
	if err != nil {
		return "", err
	}
	s := domain.NovaSolicitacao(novoID(), assunto)

	d.mu.Lock()
	t, ok := d.times[s.Time]
	if !ok {
		d.mu.Unlock()
		return "", domain.ErrTimeNaoEncontrado
	}
	d.timePorID[s.ID] = s.Time
	eventos, err := t.Receber(s)
	d.mu.Unlock()
	if err != nil {
		return "", err
	}

	recebida := domain.Evento{
		Tipo:          domain.EvSolicitacaoRecebida,
		SolicitacaoID: s.ID,
		Time:          s.Time,
		Ocorreu:       s.CriadaEm,
	}
	_ = d.repo.SalvarSolicitacao(ctx, s)
	d.emitir(ctx, append([]domain.Evento{recebida}, eventos...)...)
	return s.ID, nil
}

// FinalizarAtendimento encerra um atendimento e dispara a redistribuição da fila.
func (d *Distribuidor) FinalizarAtendimento(ctx context.Context, solicitacaoID string) error {
	d.mu.Lock()
	nome, ok := d.timePorID[solicitacaoID]
	if !ok {
		d.mu.Unlock()
		return domain.ErrSolicitacaoNaoEncontrada
	}
	eventos, err := d.times[nome].Finalizar(solicitacaoID)
	if err == nil {
		delete(d.timePorID, solicitacaoID)
		for _, e := range eventos {
			if e.Tipo == domain.EvSolicitacaoAtribuida {
				d.timePorID[e.SolicitacaoID] = nome
			}
		}
	}
	d.mu.Unlock()
	if err != nil {
		return err
	}
	d.emitir(ctx, eventos...)
	return nil
}

// AdicionarAtendente cria um novo atendente, o insere no time e puxa da fila imediatamente.
func (d *Distribuidor) AdicionarAtendente(ctx context.Context, nome string, nomeTime domain.NomeTime) (string, error) {
	d.mu.Lock()
	t, ok := d.times[nomeTime]
	if !ok {
		d.mu.Unlock()
		return "", domain.ErrTimeNaoEncontrado
	}
	id := "ate_" + novoIDBytes(4)
	a := domain.NovoAtendente(id, nome, nomeTime)
	eventos := t.AdicionarAtendente(a)
	d.atendentePorID[id] = nomeTime
	for _, e := range eventos {
		if e.Tipo == domain.EvSolicitacaoAtribuida {
			d.timePorID[e.SolicitacaoID] = nomeTime
		}
	}
	d.mu.Unlock()
	d.emitir(ctx, eventos...)
	return id, nil
}

// PausarAtendente suspende o atendente: ele mantém os ativos mas não recebe novas.
func (d *Distribuidor) PausarAtendente(ctx context.Context, id string) error {
	d.mu.Lock()
	nome, ok := d.atendentePorID[id]
	if !ok {
		d.mu.Unlock()
		return domain.ErrAtendenteNaoEncontrado
	}
	ev, err := d.times[nome].PausarAtendente(id)
	d.mu.Unlock()
	if err != nil {
		return err
	}
	d.emitir(ctx, ev)
	return nil
}

// RetomarAtendente reativa o atendente e puxa da fila até encher a capacidade.
func (d *Distribuidor) RetomarAtendente(ctx context.Context, id string) error {
	d.mu.Lock()
	nome, ok := d.atendentePorID[id]
	if !ok {
		d.mu.Unlock()
		return domain.ErrAtendenteNaoEncontrado
	}
	eventos, err := d.times[nome].RetomarAtendente(id)
	if err == nil {
		for _, e := range eventos {
			if e.Tipo == domain.EvSolicitacaoAtribuida {
				d.timePorID[e.SolicitacaoID] = nome
			}
		}
	}
	d.mu.Unlock()
	if err != nil {
		return err
	}
	d.emitir(ctx, eventos...)
	return nil
}

// RemoverAtendente remove o atendente do time. Falha com ErrAtendenteComAtivos se tiver ativos.
func (d *Distribuidor) RemoverAtendente(ctx context.Context, id string) error {
	d.mu.Lock()
	nome, ok := d.atendentePorID[id]
	if !ok {
		d.mu.Unlock()
		return domain.ErrAtendenteNaoEncontrado
	}
	ev, err := d.times[nome].RemoverAtendente(id)
	if err == nil {
		delete(d.atendentePorID, id)
	}
	d.mu.Unlock()
	if err != nil {
		return err
	}
	d.emitir(ctx, ev)
	return nil
}

// SeedTimes carrega times iniciais sem emitir eventos de domínio. É idempotente:
// retorna false (e não faz nada) se qualquer time já tiver atendentes.
func (d *Distribuidor) SeedTimes(times ...*domain.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, t := range d.times {
		if len(t.Atendentes()) > 0 {
			return false
		}
	}
	for _, t := range times {
		d.times[t.Nome] = t
		for _, a := range t.Atendentes() {
			d.atendentePorID[a.ID] = t.Nome
		}
	}
	return true
}

// Snapshot devolve a projeção de leitura atual para o dashboard.
func (d *Distribuidor) Snapshot() DashboardView {
	d.mu.Lock()
	defer d.mu.Unlock()

	ordem := []domain.NomeTime{domain.TimeCartoes, domain.TimeEmprestimos, domain.TimeOutros}
	view := DashboardView{AtualizadoEm: time.Now().UTC(), Times: []TimeView{}}
	for _, nome := range ordem {
		t, ok := d.times[nome]
		if !ok {
			continue
		}
		tv := TimeView{Nome: nome, TamanhoFila: t.TamanhoFila(), Atendentes: []AtendenteView{}}
		for _, a := range t.Atendentes() {
			ativas := make([]SolicitacaoAtivaView, 0, a.Ativos())
			for _, info := range a.InfoSolicitacoesAtivas() {
				ativas = append(ativas, SolicitacaoAtivaView{
					ID:             info.ID,
					TempoNaFilaSeg: info.TempoNaFilaSeg,
				})
			}
			tv.Atendentes = append(tv.Atendentes, AtendenteView{
				ID:                 a.ID,
				Nome:               a.Nome,
				Ativos:             a.Ativos(),
				Capacidade:         domain.CapacidadeMaxima,
				Pausado:            a.Pausado(),
				SolicitacoesAtivas: ativas,
			})
			tv.EmAtendimento += a.Ativos()
		}
		view.Times = append(view.Times, tv)
	}
	return view
}

// Historico devolve os eventos persistidos mais recentes.
func (d *Distribuidor) Historico(ctx context.Context, limite int) ([]domain.Evento, error) {
	return d.repo.HistoricoEventos(ctx, limite)
}

func (d *Distribuidor) emitir(ctx context.Context, eventos ...domain.Evento) {
	for _, e := range eventos {
		_ = d.repo.RegistrarEvento(ctx, e)
		d.pub.Publicar(e)
	}
}

func novoID() string {
	return "sol_" + novoIDBytes(8)
}

func novoIDBytes(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
