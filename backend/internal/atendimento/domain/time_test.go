package domain

import "testing"

func assunto(t *testing.T, codigo string) Assunto {
	t.Helper()
	a, err := NovoAssunto(codigo)
	if err != nil {
		t.Fatalf("assunto %q inválido: %v", codigo, err)
	}
	return a
}

// Cenário: atribuir solicitação a atendente livre do time correto.
func TestTime_AtribuiAAtendenteLivre(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana)

	s := NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao))
	eventos, err := tm.Receber(s)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(eventos) != 1 || eventos[0].Tipo != EvSolicitacaoAtribuida {
		t.Fatalf("esperava 1 evento atribuída, veio %+v", eventos)
	}
	if eventos[0].AtendenteID != "a1" {
		t.Errorf("esperava atribuição a Ana, veio %q", eventos[0].AtendenteID)
	}
	if ana.Ativos() != 1 {
		t.Errorf("esperava Ana com 1 ativo, veio %d", ana.Ativos())
	}
}

// Cenário: respeitar o limite de 3 atendimentos simultâneos.
func TestTime_RespeitaCapacidadeMaxima(t *testing.T) {
	bruno := NovoAtendente("a1", "Bruno", TimeCartoes)
	for i, id := range []string{"s1", "s2", "s3"} {
		if err := bruno.Atribuir(id, nil); err != nil {
			t.Fatalf("atribuição %d falhou: %v", i, err)
		}
	}
	if bruno.TemVaga() {
		t.Fatal("Bruno não deveria ter vaga com 3 ativos")
	}
	if err := bruno.Atribuir("s4", nil); err != ErrAtendenteLotado {
		t.Errorf("esperava ErrAtendenteLotado, veio %v", err)
	}
}

// Cenário: enfileirar quando todos os atendentes do time estão lotados.
func TestTime_EnfileiraQuandoTodosLotados(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	ana.Atribuir("x1", nil)
	ana.Atribuir("x2", nil)
	ana.Atribuir("x3", nil)
	tm := NovoTime(TimeCartoes, ana)

	eventos, err := tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(eventos) != 1 || eventos[0].Tipo != EvSolicitacaoEnfileirada {
		t.Fatalf("esperava evento enfileirada, veio %+v", eventos)
	}
	if tm.TamanhoFila() != 1 {
		t.Errorf("esperava fila com 1, veio %d", tm.TamanhoFila())
	}
}

// Cenário: distribuir da fila assim que um atendente fica livre.
func TestTime_DistribuiDaFilaAoLiberar(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana)

	// Ana enche (3) e a 4ª vai pra fila.
	for _, id := range []string{"s1", "s2", "s3"} {
		tm.Receber(NovaSolicitacao(id, assunto(t, AssuntoProblemaCartao)))
	}
	tm.Receber(NovaSolicitacao("s4", assunto(t, AssuntoProblemaCartao)))
	if tm.TamanhoFila() != 1 {
		t.Fatalf("pré-condição: esperava fila 1, veio %d", tm.TamanhoFila())
	}

	eventos, err := tm.Finalizar("s1")
	if err != nil {
		t.Fatalf("erro ao finalizar: %v", err)
	}
	// Espera: finalizado(s1) + atribuída(s4) ao mesmo atendente.
	if len(eventos) != 2 {
		t.Fatalf("esperava 2 eventos, veio %+v", eventos)
	}
	if eventos[0].Tipo != EvAtendimentoFinalizado || eventos[1].Tipo != EvSolicitacaoAtribuida {
		t.Fatalf("ordem/tipos inesperados: %+v", eventos)
	}
	if eventos[1].SolicitacaoID != "s4" {
		t.Errorf("esperava s4 atribuída, veio %q", eventos[1].SolicitacaoID)
	}
	if tm.TamanhoFila() != 0 {
		t.Errorf("esperava fila vazia, veio %d", tm.TamanhoFila())
	}
}

func TestTime_FinalizarSolicitacaoInexistente(t *testing.T) {
	tm := NovoTime(TimeCartoes, NovoAtendente("a1", "Ana", TimeCartoes))
	if _, err := tm.Finalizar("inexistente"); err != ErrSolicitacaoNaoEncontrada {
		t.Errorf("esperava ErrSolicitacaoNaoEncontrada, veio %v", err)
	}
}

// Cenário BDD: Finalizar emite DuracaoAtendimentoSeg no evento.
func TestTime_FinalizarEmiteDuracao(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana)
	tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))

	eventos, err := tm.Finalizar("s1")
	if err != nil {
		t.Fatalf("erro ao finalizar: %v", err)
	}
	ev := eventos[0]
	if ev.DuracaoAtendimentoSeg == nil {
		t.Fatal("esperava DuracaoAtendimentoSeg != nil no evento AtendimentoFinalizado")
	}
	if *ev.DuracaoAtendimentoSeg < 0 {
		t.Errorf("DuracaoAtendimentoSeg não pode ser negativo, veio %d", *ev.DuracaoAtendimentoSeg)
	}
}

// Cenário BDD: Solicitação que veio da fila deve emitir TempoNaFilaSeg ao ser atribuída.
func TestTime_AtribuirDaFilaEmiteTempoNaFila(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana)

	// Lota e enfileira s4.
	for _, id := range []string{"s1", "s2", "s3"} {
		tm.Receber(NovaSolicitacao(id, assunto(t, AssuntoProblemaCartao)))
	}
	tm.Receber(NovaSolicitacao("s4", assunto(t, AssuntoProblemaCartao)))

	// Ao finalizar s1, s4 é puxada da fila e deve ter TempoNaFilaSeg.
	eventos, err := tm.Finalizar("s1")
	if err != nil {
		t.Fatalf("erro ao finalizar: %v", err)
	}
	atribuida := eventos[1] // SolicitacaoAtribuida para s4
	if atribuida.TempoNaFilaSeg == nil {
		t.Fatal("esperava TempoNaFilaSeg != nil em SolicitacaoAtribuida vinda da fila")
	}
	if *atribuida.TempoNaFilaSeg < 0 {
		t.Errorf("TempoNaFilaSeg não pode ser negativo, veio %d", *atribuida.TempoNaFilaSeg)
	}

	// O evento de finalização também deve carregar TempoNaFilaSeg (s4 veio da fila e então foi finalizada).
	// Mas neste teste s4 ainda não foi finalizada; o TempoNaFila fica no AtendimentoFinalizado quando s4 for finalizada.
	finalizado := eventos[0]
	if finalizado.DuracaoAtendimentoSeg == nil {
		t.Fatal("esperava DuracaoAtendimentoSeg no AtendimentoFinalizado de s1")
	}
}

// Cenário BDD: Balancear distribuição ao atendente com menor carga.
// Com Ana(2 ativos) e Bruno(0), a nova solicitação deve ir para Bruno.
func TestTime_BalanceiaParaAtendenteMenorCarga(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	ana.Atribuir("x1", nil)
	ana.Atribuir("x2", nil)
	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana, bruno)

	eventos, err := tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(eventos) != 1 || eventos[0].Tipo != EvSolicitacaoAtribuida {
		t.Fatalf("esperava 1 evento atribuída, veio %+v", eventos)
	}
	if eventos[0].AtendenteID != "a2" {
		t.Errorf("esperava atribuição a Bruno (menor carga), veio %q", eventos[0].AtendenteID)
	}
	if ana.Ativos() != 2 {
		t.Errorf("Ana deveria permanecer com 2 ativos, veio %d", ana.Ativos())
	}
	if bruno.Ativos() != 1 {
		t.Errorf("Bruno deveria ter 1 ativo, veio %d", bruno.Ativos())
	}
}

// Cenário BDD: Balancear entre atendentes com carga igual — escolhe o de menor carga.
// Ana(1), Bruno(1), Carlos(0): deve ir para Carlos.
func TestTime_BalanceiaComCargaIgual(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	ana.Atribuir("x1", nil)
	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	bruno.Atribuir("x2", nil)
	carlos := NovoAtendente("a3", "Carlos", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana, bruno, carlos)

	eventos, err := tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if eventos[0].AtendenteID != "a3" {
		t.Errorf("esperava atribuição a Carlos (carga 0), veio %q", eventos[0].AtendenteID)
	}
}

// Cenário BDD: Adicionar atendente ao time em tempo de execução.
// Ana(lotada) → adiciona Bruno → nova solicitação vai para Bruno.
func TestTime_AdicionarAtendenteRecebeSolicitacao(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	for _, id := range []string{"s1", "s2", "s3"} {
		ana.Atribuir(id, nil)
	}
	tm := NovoTime(TimeCartoes, ana)

	// Sem Bruno: deve enfileirar.
	evs, _ := tm.Receber(NovaSolicitacao("s4", assunto(t, AssuntoProblemaCartao)))
	if evs[0].Tipo != EvSolicitacaoEnfileirada {
		t.Fatalf("pré-condição: esperava enfileirada sem Bruno, veio %+v", evs)
	}

	// Adiciona Bruno dinamicamente.
	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	tm.AdicionarAtendente(bruno)

	// Nova solicitação deve ser atribuída a Bruno (carga 0).
	evs2, err := tm.Receber(NovaSolicitacao("s5", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if evs2[0].Tipo != EvSolicitacaoAtribuida || evs2[0].AtendenteID != "a2" {
		t.Errorf("esperava s5 atribuída a Bruno, veio %+v", evs2)
	}
}

// Cenário BDD: Novo atendente participa do balanceamento imediatamente.
// Ana(2 ativos) → adiciona Bruno(0) → próxima vai para Bruno.
func TestTime_NovoAtendenteParticipaNoBalaenceamento(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	ana.Atribuir("x1", nil)
	ana.Atribuir("x2", nil)
	tm := NovoTime(TimeCartoes, ana)

	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	tm.AdicionarAtendente(bruno)

	evs, err := tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if evs[0].AtendenteID != "a2" {
		t.Errorf("esperava atribuição a Bruno (menor carga), veio %q", evs[0].AtendenteID)
	}
}

// Cenário BDD: Adicionar atendente puxa da fila imediatamente.
// Ana(lotada) + s4 na fila → adiciona Bruno → s4 deve ser atribuída a Bruno sem nova chegada.
func TestTime_AdicionarAtendentePuxaFila(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	for _, id := range []string{"s1", "s2", "s3"} {
		ana.Atribuir(id, nil)
	}
	tm := NovoTime(TimeCartoes, ana)
	tm.Receber(NovaSolicitacao("s4", assunto(t, AssuntoProblemaCartao)))
	if tm.TamanhoFila() != 1 {
		t.Fatalf("pré-condição: esperava fila 1, veio %d", tm.TamanhoFila())
	}

	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	evs := tm.AdicionarAtendente(bruno)

	if len(evs) != 1 || evs[0].Tipo != EvSolicitacaoAtribuida {
		t.Fatalf("esperava 1 evento atribuída ao adicionar Bruno, veio %+v", evs)
	}
	if evs[0].AtendenteID != "a2" {
		t.Errorf("esperava s4 atribuída a Bruno, veio %q", evs[0].AtendenteID)
	}
	if tm.TamanhoFila() != 0 {
		t.Errorf("esperava fila vazia após adicionar Bruno, veio %d", tm.TamanhoFila())
	}
}

// Cenário BDD: Atendente pausado não recebe novas solicitações.
// Ana(0 ativos, pausada) e Bruno(1 ativo) → nova solicitação vai para Bruno.
func TestTime_PausarAtendenteImpede(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)   // sem ativos → pode pausar
	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	bruno.Atribuir("x1", nil)
	tm := NovoTime(TimeCartoes, ana, bruno)

	if _, err := tm.PausarAtendente("a1"); err != nil {
		t.Fatalf("erro ao pausar: %v", err)
	}

	evs, err := tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if evs[0].AtendenteID != "a2" {
		t.Errorf("esperava atribuição a Bruno (Ana pausada), veio %q", evs[0].AtendenteID)
	}
}

// Cenário BDD: Pausar atendente com atendimentos ativos é bloqueado.
func TestTime_PausarAtendenteComAtivosEBloqueado(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	ana.Atribuir("s1", nil)
	tm := NovoTime(TimeCartoes, ana)

	if _, err := tm.PausarAtendente("a1"); err != ErrAtendenteComAtivos {
		t.Errorf("esperava ErrAtendenteComAtivos ao pausar com ativo, veio %v", err)
	}
	if ana.Pausado() {
		t.Error("Ana não deveria estar pausada após tentativa bloqueada")
	}
}

// Cenário BDD: Atendente pausado como único do time → solicitação enfileirada.
func TestTime_PausarUnicoAtendenteEnfileira(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana)

	tm.PausarAtendente("a1")
	evs, err := tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(evs) != 1 || evs[0].Tipo != EvSolicitacaoEnfileirada {
		t.Fatalf("esperava enfileirada com Ana pausada, veio %+v", evs)
	}
}

// Cenário BDD: Retomar atendente pausado puxa da fila.
func TestTime_RetomarAtendentePuxaFila(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	tm := NovoTime(TimeCartoes, ana)

	tm.PausarAtendente("a1")
	tm.Receber(NovaSolicitacao("s1", assunto(t, AssuntoProblemaCartao))) // vai para fila
	if tm.TamanhoFila() != 1 {
		t.Fatalf("pré-condição: esperava fila 1, veio %d", tm.TamanhoFila())
	}

	evs, err := tm.RetomarAtendente("a1")
	if err != nil {
		t.Fatalf("erro ao retomar: %v", err)
	}
	// Espera: EvAtendenteRetomado + EvSolicitacaoAtribuida
	if len(evs) < 2 {
		t.Fatalf("esperava >=2 eventos ao retomar, veio %+v", evs)
	}
	if evs[0].Tipo != EvAtendenteRetomado {
		t.Errorf("1º evento deveria ser EvAtendenteRetomado, veio %v", evs[0].Tipo)
	}
	if evs[1].Tipo != EvSolicitacaoAtribuida || evs[1].AtendenteID != "a1" {
		t.Errorf("2º evento deveria ser atribuída a Ana, veio %+v", evs[1])
	}
	if tm.TamanhoFila() != 0 {
		t.Errorf("esperava fila vazia após retomar, veio %d", tm.TamanhoFila())
	}
}

// Cenário BDD: Remover atendente sem ativos tem sucesso.
func TestTime_RemoverAtendenteSemAtivos(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	bruno := NovoAtendente("a2", "Bruno", TimeCartoes)
	ana.Atribuir("x1", nil)
	tm := NovoTime(TimeCartoes, ana, bruno)

	ev, err := tm.RemoverAtendente("a2")
	if err != nil {
		t.Fatalf("erro inesperado ao remover Bruno: %v", err)
	}
	if ev.Tipo != EvAtendenteRemovido || ev.AtendenteID != "a2" {
		t.Errorf("esperava EvAtendenteRemovido de Bruno, veio %+v", ev)
	}
	if len(tm.Atendentes()) != 1 || tm.Atendentes()[0].ID != "a1" {
		t.Errorf("esperava apenas Ana no time após remoção, veio %v", tm.Atendentes())
	}
}

// Cenário BDD: Remover atendente com ativos é bloqueado.
func TestTime_RemoverAtendenteComAtivosEBloqueado(t *testing.T) {
	ana := NovoAtendente("a1", "Ana", TimeCartoes)
	ana.Atribuir("s1", nil)
	tm := NovoTime(TimeCartoes, ana)

	if _, err := tm.RemoverAtendente("a1"); err != ErrAtendenteComAtivos {
		t.Errorf("esperava ErrAtendenteComAtivos, veio %v", err)
	}
}

// PausarAtendente com ID inexistente retorna ErrAtendenteNaoEncontrado.
func TestTime_PausarAtendenteInexistente(t *testing.T) {
	tm := NovoTime(TimeCartoes)
	if _, err := tm.PausarAtendente("inexistente"); err != ErrAtendenteNaoEncontrado {
		t.Errorf("esperava ErrAtendenteNaoEncontrado, veio %v", err)
	}
}

// RemoverAtendente com ID inexistente retorna ErrAtendenteNaoEncontrado.
func TestTime_RemoverAtendenteInexistente(t *testing.T) {
	tm := NovoTime(TimeCartoes)
	if _, err := tm.RemoverAtendente("inexistente"); err != ErrAtendenteNaoEncontrado {
		t.Errorf("esperava ErrAtendenteNaoEncontrado, veio %v", err)
	}
}
