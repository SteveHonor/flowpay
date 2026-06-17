package application_test

import (
	"context"
	"sync"
	"testing"

	"flowpay/internal/atendimento/application"
	"flowpay/internal/atendimento/domain"
	"flowpay/internal/atendimento/infrastructure/memoria"
)

// pubDescarta ignora todos os eventos — usado quando só interessa o estado do Distribuidor.
type pubDescarta struct{}

func (pubDescarta) Publicar(domain.Evento) {}

// pubCaptura acumula eventos publicados — usado para verificar o conteúdo dos eventos emitidos.
type pubCaptura struct {
	mu     sync.Mutex
	eventos []domain.Evento
}

func (p *pubCaptura) Publicar(e domain.Evento) {
	p.mu.Lock()
	p.eventos = append(p.eventos, e)
	p.mu.Unlock()
}

func (p *pubCaptura) porTipo(tipo domain.TipoEvento) []domain.Evento {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []domain.Evento
	for _, e := range p.eventos {
		if e.Tipo == tipo {
			out = append(out, e)
		}
	}
	return out
}

func novoDistribuidor(times ...*domain.Time) *application.Distribuidor {
	return application.NovoDistribuidor(memoria.NovoRepositorio(), pubDescarta{}, times...)
}

func novoDistribuidorCaptura(pub *pubCaptura, times ...*domain.Time) *application.Distribuidor {
	return application.NovoDistribuidor(memoria.NovoRepositorio(), pub, times...)
}

func timesBasicos() []*domain.Time {
	return []*domain.Time{
		domain.NovoTime(domain.TimeCartoes,
			domain.NovoAtendente("c1", "Ana", domain.TimeCartoes),
			domain.NovoAtendente("c2", "Bruno", domain.TimeCartoes),
		),
		domain.NovoTime(domain.TimeEmprestimos,
			domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos),
		),
		domain.NovoTime(domain.TimeOutros,
			domain.NovoAtendente("o1", "Elena", domain.TimeOutros),
		),
	}
}

// Cenário BDD: Registrar solicitação roteia ao time correto e aparece no snapshot.
func TestDistribuidor_RegistraSolicitacao(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(timesBasicos()...)

	id, err := d.RegistrarSolicitacao(ctx, "problema_cartao")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id == "" {
		t.Fatal("esperava ID gerado, veio vazio")
	}

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	if cartoes.EmAtendimento != 1 {
		t.Errorf("esperava 1 em atendimento no time Cartões, veio %d", cartoes.EmAtendimento)
	}
}

// Cenário BDD: Balancear distribuição ao atendente com menor carga.
// Com Ana(0) e Bruno(0), duas solicitações devem ser distribuídas 1 cada.
func TestDistribuidor_BalanceiaEntreAtendentes(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(timesBasicos()...)

	d.RegistrarSolicitacao(ctx, "problema_cartao")
	d.RegistrarSolicitacao(ctx, "problema_cartao")

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	for _, a := range cartoes.Atendentes {
		if a.Ativos != 1 {
			t.Errorf("atendente %q deveria ter 1 ativo após balanceamento, veio %d", a.Nome, a.Ativos)
		}
	}
}

// Cenário BDD: Enfileirar quando todos os atendentes estão lotados.
func TestDistribuidor_EnfileiraQuandoTodosLotados(t *testing.T) {
	ctx := context.Background()
	// Time com apenas um atendente (capacidade 3).
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	for i := 0; i < 3; i++ {
		d.RegistrarSolicitacao(ctx, "problema_cartao")
	}
	d.RegistrarSolicitacao(ctx, "problema_cartao") // 4ª deve enfileirar

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	if cartoes.TamanhoFila != 1 {
		t.Errorf("esperava fila 1, veio %d", cartoes.TamanhoFila)
	}
	if cartoes.EmAtendimento != 3 {
		t.Errorf("esperava 3 em atendimento, veio %d", cartoes.EmAtendimento)
	}
}

// Cenário BDD: Finalizar puxa a próxima da fila (FIFO).
func TestDistribuidor_FinalizarPuxaDaFila(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	var ids []string
	for i := 0; i < 3; i++ {
		id, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")
		ids = append(ids, id)
	}
	d.RegistrarSolicitacao(ctx, "problema_cartao") // enfileirada

	if err := d.FinalizarAtendimento(ctx, ids[0]); err != nil {
		t.Fatalf("erro ao finalizar: %v", err)
	}

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	if cartoes.TamanhoFila != 0 {
		t.Errorf("esperava fila vazia após finalizar, veio %d", cartoes.TamanhoFila)
	}
	if cartoes.EmAtendimento != 3 {
		t.Errorf("esperava 3 em atendimento após puxar da fila, veio %d", cartoes.EmAtendimento)
	}
}

// Cenário BDD: Finalizar solicitação inexistente retorna erro de domínio.
func TestDistribuidor_FinalizarInexistente(t *testing.T) {
	d := novoDistribuidor(timesBasicos()...)
	if err := d.FinalizarAtendimento(context.Background(), "sol_nao_existe"); err != domain.ErrSolicitacaoNaoEncontrada {
		t.Errorf("esperava ErrSolicitacaoNaoEncontrada, veio %v", err)
	}
}

// Cenário BDD: Adicionar atendente ao time em tempo de execução — aparece no snapshot.
func TestDistribuidor_AdicionarAtendente_AparecNoSnapshot(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	id, err := d.AdicionarAtendente(ctx, "Bruno", domain.TimeCartoes)
	if err != nil {
		t.Fatalf("erro ao adicionar atendente: %v", err)
	}
	if id == "" {
		t.Fatal("esperava ID gerado para novo atendente")
	}

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	encontrou := false
	for _, a := range cartoes.Atendentes {
		if a.Nome == "Bruno" {
			encontrou = true
		}
	}
	if !encontrou {
		t.Error("Bruno não encontrado no snapshot após AdicionarAtendente")
	}
}

// Cenário BDD: Novo atendente participa do balanceamento imediatamente.
// Ana(lotada) → adiciona Bruno → nova solicitação vai para Bruno.
func TestDistribuidor_AdicionarAtendente_RecebeNovasSolicitacoes(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	// Lota Ana.
	for i := 0; i < 3; i++ {
		d.RegistrarSolicitacao(ctx, "problema_cartao")
	}

	d.AdicionarAtendente(ctx, "Bruno", domain.TimeCartoes)
	sidNovo, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	var brunoView *application.AtendenteView
	for i := range cartoes.Atendentes {
		if cartoes.Atendentes[i].Nome == "Bruno" {
			brunoView = &cartoes.Atendentes[i]
			break
		}
	}
	if brunoView == nil {
		t.Fatal("Bruno não encontrado no snapshot")
	}
	if brunoView.Ativos != 1 {
		t.Errorf("Bruno deveria ter 1 ativo, veio %d", brunoView.Ativos)
	}
	encontrou := false
	for _, s := range brunoView.SolicitacoesAtivas {
		if s.ID == sidNovo {
			encontrou = true
		}
	}
	if !encontrou {
		t.Errorf("solicitação %q não está em SolicitacoesAtivas de Bruno: %v", sidNovo, brunoView.SolicitacoesAtivas)
	}
}

// Cenário BDD: AdicionarAtendente com time inválido retorna erro de domínio.
func TestDistribuidor_AdicionarAtendenteTimeInvalido(t *testing.T) {
	d := novoDistribuidor(timesBasicos()...)
	if _, err := d.AdicionarAtendente(context.Background(), "X", "INEXISTENTE"); err != domain.ErrTimeNaoEncontrado {
		t.Errorf("esperava ErrTimeNaoEncontrado, veio %v", err)
	}
}

// Cenário BDD: Snapshot expõe solicitações ativas por atendente e remove ao finalizar.
func TestDistribuidor_SnapshotSolicitacoesAtivas(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	sid, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")

	snap := d.Snapshot()
	ana := timeViewDe(snap, domain.TimeCartoes).Atendentes[0]
	if len(ana.SolicitacoesAtivas) != 1 || ana.SolicitacoesAtivas[0].ID != sid {
		t.Errorf("esperava [{ID:%q}] em SolicitacoesAtivas, veio %v", sid, ana.SolicitacoesAtivas)
	}

	d.FinalizarAtendimento(ctx, sid)

	snap = d.Snapshot()
	ana = timeViewDe(snap, domain.TimeCartoes).Atendentes[0]
	if len(ana.SolicitacoesAtivas) != 0 {
		t.Errorf("esperava SolicitacoesAtivas vazio após finalizar, veio %v", ana.SolicitacoesAtivas)
	}
}

// Cenário BDD: Registrar duração do atendimento ao finalizar.
// O evento atendimento_finalizado deve carregar DuracaoAtendimentoSeg >= 0.
func TestDistribuidor_FinalizarEmiteDuracaoNoEvento(t *testing.T) {
	ctx := context.Background()
	pub := &pubCaptura{}
	d := novoDistribuidorCaptura(pub,
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	sid, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")
	if err := d.FinalizarAtendimento(ctx, sid); err != nil {
		t.Fatalf("erro ao finalizar: %v", err)
	}

	finalizados := pub.porTipo(domain.EvAtendimentoFinalizado)
	if len(finalizados) != 1 {
		t.Fatalf("esperava 1 evento atendimento_finalizado, veio %d", len(finalizados))
	}
	ev := finalizados[0]
	if ev.DuracaoAtendimentoSeg == nil {
		t.Fatal("DuracaoAtendimentoSeg não deve ser nil no evento atendimento_finalizado")
	}
	if *ev.DuracaoAtendimentoSeg < 0 {
		t.Errorf("DuracaoAtendimentoSeg não pode ser negativo, veio %d", *ev.DuracaoAtendimentoSeg)
	}
}

// Cenário BDD: Registrar tempo na fila ao puxar solicitação enfileirada.
// O evento solicitacao_atribuida (vinda da fila) deve carregar TempoNaFilaSeg.
// O evento atendimento_finalizado do atendimento anterior NÃO deve ter TempoNaFilaSeg.
func TestDistribuidor_FilaEmiteTempoEsperaNoEvento(t *testing.T) {
	ctx := context.Background()
	pub := &pubCaptura{}
	d := novoDistribuidorCaptura(pub,
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	// Lota Ana e enfileira s4.
	var primeiros []string
	for i := 0; i < 3; i++ {
		id, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")
		primeiros = append(primeiros, id)
	}
	d.RegistrarSolicitacao(ctx, "problema_cartao") // enfileirada

	// Finalizar s1: libera vaga e puxa s4 da fila.
	if err := d.FinalizarAtendimento(ctx, primeiros[0]); err != nil {
		t.Fatalf("erro ao finalizar: %v", err)
	}

	// O evento atendimento_finalizado de s1 NÃO deve ter TempoNaFilaSeg (s1 não foi enfileirada).
	finalizados := pub.porTipo(domain.EvAtendimentoFinalizado)
	if len(finalizados) != 1 {
		t.Fatalf("esperava 1 atendimento_finalizado, veio %d", len(finalizados))
	}
	if finalizados[0].TempoNaFilaSeg != nil {
		t.Errorf("s1 não foi enfileirada: TempoNaFilaSeg deveria ser nil, veio %v", finalizados[0].TempoNaFilaSeg)
	}

	// O evento solicitacao_atribuida de s4 (puxada da fila) DEVE ter TempoNaFilaSeg.
	atribuidas := pub.porTipo(domain.EvSolicitacaoAtribuida)
	var daFila *domain.Evento
	for i := range atribuidas {
		if atribuidas[i].TempoNaFilaSeg != nil {
			daFila = &atribuidas[i]
			break
		}
	}
	if daFila == nil {
		t.Fatal("esperava ao menos um evento solicitacao_atribuida com TempoNaFilaSeg (s4 veio da fila)")
	}
	if *daFila.TempoNaFilaSeg < 0 {
		t.Errorf("TempoNaFilaSeg não pode ser negativo, veio %d", *daFila.TempoNaFilaSeg)
	}
}

// Cenário BDD: Adicionar atendente puxa da fila imediatamente.
func TestDistribuidor_AdicionarAtendente_PuxaFila(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)
	// Lota Ana e enfileira s4.
	for i := 0; i < 3; i++ {
		d.RegistrarSolicitacao(ctx, "problema_cartao")
	}
	sidFila, _ := d.RegistrarSolicitacao(ctx, "problema_cartao") // enfileirada

	snap := d.Snapshot()
	if timeViewDe(snap, domain.TimeCartoes).TamanhoFila != 1 {
		t.Fatalf("pré-condição: esperava fila 1")
	}

	d.AdicionarAtendente(ctx, "Bruno", domain.TimeCartoes)

	snap = d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	if cartoes.TamanhoFila != 0 {
		t.Errorf("esperava fila 0 após adicionar Bruno, veio %d", cartoes.TamanhoFila)
	}
	var brunoView *application.AtendenteView
	for i := range cartoes.Atendentes {
		if cartoes.Atendentes[i].Nome == "Bruno" {
			brunoView = &cartoes.Atendentes[i]
			break
		}
	}
	if brunoView == nil {
		t.Fatal("Bruno não encontrado no snapshot")
	}
	encontrou := false
	for _, s := range brunoView.SolicitacoesAtivas {
		if s.ID == sidFila {
			encontrou = true
		}
	}
	if !encontrou {
		t.Errorf("esperava solicitação da fila (%q) em Bruno, veio %v", sidFila, brunoView.SolicitacoesAtivas)
	}
}

// Cenário BDD: Pausar atendente — não recebe novas solicitações, aparece como pausado no snapshot.
// Ana(0 ativos) é pausada; Bruno recebe a nova solicitação. Finalizar o ativo de Bruno funciona normalmente.
func TestDistribuidor_PausarAtendente(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(timesBasicos()...)

	// Registra uma solicitação → vai para Ana ou Bruno (menor carga).
	sid, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")

	// Descobre qual atendente ficou com 0 ativos (o outro é quem vai ser pausado).
	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	var atendenteVazio string
	for _, a := range cartoes.Atendentes {
		if a.Ativos == 0 {
			atendenteVazio = a.ID
			break
		}
	}
	if atendenteVazio == "" {
		t.Fatal("nenhum atendente com 0 ativos — pré-condição inválida")
	}

	// Pausa o atendente sem ativos (invariante: só pausa com 0 ativos).
	if err := d.PausarAtendente(ctx, atendenteVazio); err != nil {
		t.Fatalf("erro ao pausar atendente sem ativos: %v", err)
	}

	snap = d.Snapshot()
	cartoes = timeViewDe(snap, domain.TimeCartoes)
	for _, a := range cartoes.Atendentes {
		if a.ID == atendenteVazio && !a.Pausado {
			t.Errorf("esperava Pausado=true no snapshot após pausar, veio false")
		}
	}

	// Finalizar o atendimento do atendente ativo ainda funciona.
	if err := d.FinalizarAtendimento(ctx, sid); err != nil {
		t.Errorf("não deveria falhar ao finalizar atendimento: %v", err)
	}
}

// Cenário BDD: Pausar atendente com atendimentos ativos é bloqueado.
func TestDistribuidor_PausarAtendenteComAtivos_Bloqueado(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	d.RegistrarSolicitacao(ctx, "problema_cartao") // c1 → 1 ativo

	if err := d.PausarAtendente(ctx, "c1"); err != domain.ErrAtendenteComAtivos {
		t.Errorf("esperava ErrAtendenteComAtivos ao pausar com ativo, veio %v", err)
	}

	// Verifica que o atendente não foi pausado.
	for _, a := range timeViewDe(d.Snapshot(), domain.TimeCartoes).Atendentes {
		if a.ID == "c1" && a.Pausado {
			t.Error("Ana não deveria estar pausada após tentativa bloqueada")
		}
	}
}

// Cenário BDD: Retomar atendente pausado puxa da fila.
func TestDistribuidor_RetomarAtendente_PuxaFila(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)

	if err := d.PausarAtendente(ctx, "c1"); err != nil {
		t.Fatalf("erro ao pausar: %v", err)
	}

	// Com Ana pausada, solicitação vai para fila.
	sidFila, _ := d.RegistrarSolicitacao(ctx, "problema_cartao")
	if timeViewDe(d.Snapshot(), domain.TimeCartoes).TamanhoFila != 1 {
		t.Fatalf("pré-condição: esperava fila 1 com Ana pausada")
	}

	if err := d.RetomarAtendente(ctx, "c1"); err != nil {
		t.Fatalf("erro ao retomar: %v", err)
	}

	snap := d.Snapshot()
	cartoes := timeViewDe(snap, domain.TimeCartoes)
	if cartoes.TamanhoFila != 0 {
		t.Errorf("esperava fila 0 após retomar Ana, veio %d", cartoes.TamanhoFila)
	}
	ana := cartoes.Atendentes[0]
	if ana.Pausado {
		t.Error("Ana não deveria estar pausada após retomar")
	}
	encontrou := false
	for _, s := range ana.SolicitacoesAtivas {
		if s.ID == sidFila {
			encontrou = true
		}
	}
	if !encontrou {
		t.Errorf("esperava solicitação da fila em Ana após retomar, veio %v", ana.SolicitacoesAtivas)
	}
}

// Cenário BDD: Remover atendente sem ativos — some do snapshot.
func TestDistribuidor_RemoverAtendente(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(timesBasicos()...)

	if err := d.RemoverAtendente(ctx, "c2"); err != nil {
		t.Fatalf("erro ao remover Bruno: %v", err)
	}

	cartoes := timeViewDe(d.Snapshot(), domain.TimeCartoes)
	for _, a := range cartoes.Atendentes {
		if a.ID == "c2" {
			t.Error("Bruno ainda aparece no snapshot após remoção")
		}
	}
}

// Cenário BDD: Remover atendente com ativos retorna ErrAtendenteComAtivos.
func TestDistribuidor_RemoverAtendenteComAtivos_Bloqueado(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(timesBasicos()...)
	d.RegistrarSolicitacao(ctx, "problema_cartao")

	// Determina qual atendente ficou com o ativo.
	cartoes := timeViewDe(d.Snapshot(), domain.TimeCartoes)
	var comAtivo string
	for _, a := range cartoes.Atendentes {
		if a.Ativos > 0 {
			comAtivo = a.ID
			break
		}
	}
	if comAtivo == "" {
		t.Fatal("nenhum atendente com ativo")
	}
	if err := d.RemoverAtendente(ctx, comAtivo); err != domain.ErrAtendenteComAtivos {
		t.Errorf("esperava ErrAtendenteComAtivos, veio %v", err)
	}
}

// Pausar/retomar/remover atendente inexistente retorna ErrAtendenteNaoEncontrado.
func TestDistribuidor_GerenciarAtendenteInexistente(t *testing.T) {
	ctx := context.Background()
	d := novoDistribuidor(timesBasicos()...)

	if err := d.PausarAtendente(ctx, "nao_existe"); err != domain.ErrAtendenteNaoEncontrado {
		t.Errorf("pausar: esperava ErrAtendenteNaoEncontrado, veio %v", err)
	}
	if err := d.RetomarAtendente(ctx, "nao_existe"); err != domain.ErrAtendenteNaoEncontrado {
		t.Errorf("retomar: esperava ErrAtendenteNaoEncontrado, veio %v", err)
	}
	if err := d.RemoverAtendente(ctx, "nao_existe"); err != domain.ErrAtendenteNaoEncontrado {
		t.Errorf("remover: esperava ErrAtendenteNaoEncontrado, veio %v", err)
	}
}

func timeViewDe(snap application.DashboardView, nome domain.NomeTime) application.TimeView {
	for _, tv := range snap.Times {
		if tv.Nome == nome {
			return tv
		}
	}
	return application.TimeView{}
}
