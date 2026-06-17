//go:build bdd

package features_test

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog"

	"flowpay/internal/atendimento/domain"
)

// bddCtx é o estado compartilhado entre os steps de um único cenário.
// Uma nova instância é criada para cada cenário (via inicializarCenario).
type bddCtx struct {
	times      map[domain.NomeTime]*domain.Time
	atendentes map[string]*domain.Atendente // nome → *Atendente
	eventos    []domain.Evento
	ultimoErro error
	solCounter int
}

func (c *bddCtx) nextSolID() string {
	c.solCounter++
	return fmt.Sprintf("sol_%04d", c.solCounter)
}

func (c *bddCtx) emit(evs []domain.Evento) {
	c.eventos = append(c.eventos, evs...)
}

// resolverNomeTime mapeia o nome PT do time para o tipo de domínio.
func (c *bddCtx) resolverNomeTime(nome string) domain.NomeTime {
	switch nome {
	case "Cartões":
		return domain.TimeCartoes
	case "Empréstimos":
		return domain.TimeEmprestimos
	case "Outros Assuntos":
		return domain.TimeOutros
	}
	return domain.NomeTime(nome)
}

// resolverAssunto mapeia a descrição do cenário para o código de assunto do domínio.
func (c *bddCtx) resolverAssunto(descricao string) string {
	switch descricao {
	case "Problemas com cartão":
		return domain.AssuntoProblemaCartao
	case "Contratação de empréstimo":
		return domain.AssuntoContratacaoEmprestimo
	default:
		// Qualquer outra descrição é tratada como "outros assuntos" pelo domínio.
		return strings.ToLower(strings.ReplaceAll(descricao, " ", "_"))
	}
}

func (c *bddCtx) getTime(nomePT string) *domain.Time {
	return c.times[c.resolverNomeTime(nomePT)]
}

// encontrarAtendente localiza o atendente e o seu time atual.
func (c *bddCtx) encontrarAtendente(nome string) (*domain.Atendente, *domain.Time, error) {
	a, ok := c.atendentes[nome]
	if !ok {
		return nil, nil, fmt.Errorf("atendente %q não encontrado no contexto do cenário", nome)
	}
	for _, tm := range c.times {
		for _, ta := range tm.Atendentes() {
			if ta.ID == a.ID {
				return a, tm, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("atendente %q não está em nenhum time", nome)
}

// ── Background ────────────────────────────────────────────────────────────────

// "o time 'Cartões' responsável por 'Problemas com cartão'"
// Cria o time no contexto; o roteamento assunto→time já está no domínio.
func (c *bddCtx) oTimeResponsavelPor(nomePT, _ string) error {
	nomeTime := c.resolverNomeTime(nomePT)
	c.times[nomeTime] = domain.NovoTime(nomeTime)
	return nil
}

// "o time 'Outros Assuntos' para os demais assuntos" — variação sem segundo argumento.
func (c *bddCtx) oTimeParaOsDemaisAssuntos(nomePT string) error {
	return c.oTimeResponsavelPor(nomePT, "")
}

// ── Given steps ───────────────────────────────────────────────────────────────

func (c *bddCtx) atendenteNoTimeSemAtivos(nome, nomePT string) error {
	tm := c.getTime(nomePT)
	if tm == nil {
		return fmt.Errorf("time %q não configurado (falta o passo de Contexto?)", nomePT)
	}
	id := "ate_" + strings.ToLower(strings.ReplaceAll(nome, " ", "_"))
	a := domain.NovoAtendente(id, nome, c.resolverNomeTime(nomePT))
	c.emit(tm.AdicionarAtendente(a))
	c.atendentes[nome] = a
	return nil
}

func (c *bddCtx) atendenteNoTimeComNAtivos(nome, nomePT string, n int) error {
	tm := c.getTime(nomePT)
	if tm == nil {
		return fmt.Errorf("time %q não configurado", nomePT)
	}
	id := "ate_" + strings.ToLower(strings.ReplaceAll(nome, " ", "_"))
	a := domain.NovoAtendente(id, nome, c.resolverNomeTime(nomePT))
	// Pre-preenche os ativos antes de adicionar ao time para que puxarFila não
	// interfira (atendente já estará na capacidade máxima se n == 3).
	for i := 0; i < n; i++ {
		if err := a.Atribuir(c.nextSolID(), nil); err != nil {
			return fmt.Errorf("erro ao pré-preencher ativo de %q: %w", nome, err)
		}
	}
	c.emit(tm.AdicionarAtendente(a))
	c.atendentes[nome] = a
	return nil
}

// "que todos os atendentes do time 'Cartões' estão com 3 atendimentos ativos"
// Se o time não tiver atendentes (cenário sem Dado prévio), cria um default
// para que o cenário seja exercitado de forma significativa.
func (c *bddCtx) todosOsAtendentesCom3Ativos(nomePT string) error {
	tm := c.getTime(nomePT)
	if tm == nil {
		return fmt.Errorf("time %q não configurado", nomePT)
	}
	if len(tm.Atendentes()) == 0 {
		if err := c.atendenteNoTimeSemAtivos("Atendente", nomePT); err != nil {
			return err
		}
	}
	for _, a := range tm.Atendentes() {
		for a.Ativos() < domain.CapacidadeMaxima {
			if err := a.Atribuir(c.nextSolID(), nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// "uma solicitação aguardando na fila do time 'Cartões'"
// Cria uma solicitação genérica que, com o time lotado/vazio, vai para a fila.
func (c *bddCtx) umaSolicitacaoNaFilaDoTime(nomePT string) error {
	tm := c.getTime(nomePT)
	if tm == nil {
		return fmt.Errorf("time %q não configurado", nomePT)
	}
	assunto, _ := domain.NovoAssunto(domain.AssuntoProblemaCartao)
	s := domain.NovaSolicitacao(c.nextSolID(), assunto)
	evs, err := tm.Receber(s)
	if err != nil {
		return err
	}
	c.emit(evs)
	for _, ev := range evs {
		if ev.Tipo == domain.EvSolicitacaoEnfileirada {
			return nil
		}
	}
	return fmt.Errorf("esperava solicitação enfileirada no time %q, mas foi atribuída", nomePT)
}

// "uma solicitação de 'X' está aguardando na fila" / "foi enfileirada"
func (c *bddCtx) umaSolicitacaoDeNaFila(assuntoPT string) error {
	assunto, err := domain.NovoAssunto(c.resolverAssunto(assuntoPT))
	if err != nil {
		return err
	}
	s := domain.NovaSolicitacao(c.nextSolID(), assunto)
	tm := c.times[s.Time]
	if tm == nil {
		return fmt.Errorf("time %q não configurado para assunto %q", s.Time, assuntoPT)
	}
	evs, err := tm.Receber(s)
	if err != nil {
		return err
	}
	c.emit(evs)
	for _, ev := range evs {
		if ev.Tipo == domain.EvSolicitacaoEnfileirada {
			return nil
		}
	}
	return fmt.Errorf("esperava solicitação enfileirada (assunto %q), mas foi atribuída", assuntoPT)
}

// "o time 'X' com apenas um atendente 'Y' com N atendimentos ativos"
func (c *bddCtx) oTimeComApenas1Atendente(nomePT, nome string, n int) error {
	return c.atendenteNoTimeComNAtivos(nome, nomePT, n)
}

// "uma solicitação de 'X' atribuída a 'Y'"
func (c *bddCtx) umaSolAtribuidaA(assuntoPT, nome string) error {
	assunto, err := domain.NovoAssunto(c.resolverAssunto(assuntoPT))
	if err != nil {
		return err
	}
	s := domain.NovaSolicitacao(c.nextSolID(), assunto)
	tm := c.times[s.Time]
	if tm == nil {
		return fmt.Errorf("time %q não configurado", s.Time)
	}
	evs, err := tm.Receber(s)
	if err != nil {
		return err
	}
	c.emit(evs)
	for _, ev := range evs {
		if ev.Tipo == domain.EvSolicitacaoAtribuida && ev.AtendenteNome == nome {
			return nil
		}
	}
	return fmt.Errorf("solicitação não foi atribuída a %q (veio: %v)", nome, evs)
}

// "'Ana' é pausada"  — usado como Given e como When
func (c *bddCtx) atendenteEPausado(nome string) error {
	a, tm, err := c.encontrarAtendente(nome)
	if err != nil {
		return err
	}
	ev, err := tm.PausarAtendente(a.ID)
	if err != nil {
		return fmt.Errorf("erro ao pausar %q: %w", nome, err)
	}
	c.emit([]domain.Evento{ev})
	return nil
}

// "'Ana' é retomada"
func (c *bddCtx) atendenteERetomado(nome string) error {
	a, tm, err := c.encontrarAtendente(nome)
	if err != nil {
		return err
	}
	evs, err := tm.RetomarAtendente(a.ID)
	if err != nil {
		return fmt.Errorf("erro ao retomar %q: %w", nome, err)
	}
	c.emit(evs)
	return nil
}

// "'Ana' é removido do time 'X'"
func (c *bddCtx) atendenteERemovidoDoTime(nome, _ string) error {
	a, tm, err := c.encontrarAtendente(nome)
	if err != nil {
		return err
	}
	ev, err := tm.RemoverAtendente(a.ID)
	if err != nil {
		return fmt.Errorf("erro ao remover %q: %w", nome, err)
	}
	c.emit([]domain.Evento{ev})
	delete(c.atendentes, nome)
	return nil
}

// ── When steps ────────────────────────────────────────────────────────────────

func (c *bddCtx) chegaUmaSolicitacaoDe(assuntoPT string) error {
	assunto, err := domain.NovoAssunto(c.resolverAssunto(assuntoPT))
	if err != nil {
		return fmt.Errorf("assunto inválido %q: %w", assuntoPT, err)
	}
	s := domain.NovaSolicitacao(c.nextSolID(), assunto)
	tm := c.times[s.Time]
	if tm == nil {
		return fmt.Errorf("time %q não configurado para assunto %q", s.Time, assuntoPT)
	}
	evs, err := tm.Receber(s)
	if err != nil {
		return err
	}
	c.emit(evs)
	return nil
}

// "'Ana' finaliza um/o atendimento"
func (c *bddCtx) xFinalizaAtendimento(nome string) error {
	a, tm, err := c.encontrarAtendente(nome)
	if err != nil {
		return err
	}
	infos := a.InfoSolicitacoesAtivas()
	if len(infos) == 0 {
		return fmt.Errorf("atendente %q não tem atendimentos ativos para finalizar", nome)
	}
	evs, err := tm.Finalizar(infos[0].ID)
	if err != nil {
		return fmt.Errorf("erro ao finalizar atendimento de %q: %w", nome, err)
	}
	c.emit(evs)
	return nil
}

// "um novo atendente 'X' é adicionado ao time 'Y'"
func (c *bddCtx) umNovoAtendenteEAdicionado(nome, nomePT string) error {
	return c.atendenteNoTimeSemAtivos(nome, nomePT)
}

// "um atendente finaliza um atendimento e puxa a solicitação da fila"
func (c *bddCtx) umAtendenteFinalizaEPuxa() error {
	for _, tm := range c.times {
		for _, a := range tm.Atendentes() {
			infos := a.InfoSolicitacoesAtivas()
			if len(infos) > 0 {
				evs, err := tm.Finalizar(infos[0].ID)
				if err != nil {
					return err
				}
				c.emit(evs)
				return nil
			}
		}
	}
	return fmt.Errorf("nenhum atendente com atendimentos ativos encontrado")
}

// "se tenta pausar 'Ana'" — armazena o erro para verificação no Then
func (c *bddCtx) seTentaPausar(nome string) error {
	a, tm, err := c.encontrarAtendente(nome)
	if err != nil {
		return err
	}
	ev, err := tm.PausarAtendente(a.ID)
	c.ultimoErro = err
	if err == nil {
		c.emit([]domain.Evento{ev})
	}
	return nil
}

// "se tenta remover 'Ana'" — armazena o erro para verificação no Then
func (c *bddCtx) seTentaRemover(nome string) error {
	a, tm, err := c.encontrarAtendente(nome)
	if err != nil {
		return err
	}
	ev, err := tm.RemoverAtendente(a.ID)
	c.ultimoErro = err
	if err == nil {
		c.emit([]domain.Evento{ev})
		delete(c.atendentes, nome)
	}
	return nil
}

// ── Then steps ────────────────────────────────────────────────────────────────

// "a solicitação é atribuída a 'X'" / "a próxima solicitação da fila é atribuída a 'X'" /
// "a solicitação da fila é atribuída a 'X'" / "... sem aguardar nova chegada"
func (c *bddCtx) aSolAtribuidaA(nome string) error {
	for i := len(c.eventos) - 1; i >= 0; i-- {
		ev := c.eventos[i]
		if ev.Tipo == domain.EvSolicitacaoAtribuida {
			if ev.AtendenteNome != nome {
				return fmt.Errorf("última solicitação atribuída a %q, esperava %q", ev.AtendenteNome, nome)
			}
			return nil
		}
	}
	return fmt.Errorf("nenhum evento solicitacao_atribuida encontrado")
}

// "a solicitação não é atribuída a 'X'" — verifica que a última destinação não foi X
func (c *bddCtx) aSolNaoAtribuidaA(nome string) error {
	for i := len(c.eventos) - 1; i >= 0; i-- {
		ev := c.eventos[i]
		if ev.Tipo == domain.EvSolicitacaoAtribuida {
			if ev.AtendenteNome == nome {
				return fmt.Errorf("solicitação foi atribuída a %q, mas não deveria ter sido", nome)
			}
			return nil
		}
		if ev.Tipo == domain.EvSolicitacaoEnfileirada {
			return nil // foi enfileirada, não atribuída — correto
		}
	}
	return fmt.Errorf("nenhum evento de atribuição ou enfileiramento encontrado")
}

// "a solicitação é enfileirada no time 'X'"
func (c *bddCtx) aSolEnfileiraNoTime(nomePT string) error {
	nomeTime := c.resolverNomeTime(nomePT)
	for i := len(c.eventos) - 1; i >= 0; i-- {
		ev := c.eventos[i]
		if ev.Tipo == domain.EvSolicitacaoEnfileirada {
			if ev.Time != nomeTime {
				return fmt.Errorf("solicitação enfileirada no time %q, esperava %q", ev.Time, nomeTime)
			}
			return nil
		}
		if ev.Tipo == domain.EvSolicitacaoAtribuida {
			return fmt.Errorf("solicitação foi atribuída (a %q), mas deveria ter sido enfileirada", ev.AtendenteNome)
		}
	}
	return fmt.Errorf("nenhum evento de enfileiramento encontrado")
}

// "'Ana' passa a ter N atendimentos ativos" / "'Ana' continua com N atendimentos ativos"
func (c *bddCtx) xTemNAtivos(nome string, n int) error {
	a, ok := c.atendentes[nome]
	if !ok {
		return fmt.Errorf("atendente %q não encontrado no contexto", nome)
	}
	if a.Ativos() != n {
		return fmt.Errorf("%q tem %d ativos, esperava %d", nome, a.Ativos(), n)
	}
	return nil
}

// "'Ana' continua sem atendimentos ativos"
func (c *bddCtx) xSemAtivos(nome string) error {
	return c.xTemNAtivos(nome, 0)
}

// "o snapshot do time 'Cartões' lista a solicitação ativa de 'Ana'"
func (c *bddCtx) oSnapshotLista(_, nomeAtendente string) error {
	a, ok := c.atendentes[nomeAtendente]
	if !ok {
		return fmt.Errorf("atendente %q não encontrado", nomeAtendente)
	}
	if len(a.InfoSolicitacoesAtivas()) == 0 {
		return fmt.Errorf("atendente %q não tem solicitações ativas no snapshot", nomeAtendente)
	}
	return nil
}

// "o evento 'atendimento_finalizado' contém o campo 'duracao_atendimento_seg' preenchido"
func (c *bddCtx) oEventoContemCampo(tipoEvPT, campo string) error {
	tipo := c.resolverTipoEvento(tipoEvPT)
	for _, ev := range c.eventos {
		if ev.Tipo == tipo {
			return c.verificarCampoPreenchido(ev, campo)
		}
	}
	return fmt.Errorf("evento %q não encontrado nos eventos coletados", tipoEvPT)
}

// "o valor de 'duracao_atendimento_seg' é maior ou igual a zero"
func (c *bddCtx) oValorMaiorOuIgualZero(campo string) error {
	for _, ev := range c.eventos {
		switch campo {
		case "duracao_atendimento_seg":
			if ev.DuracaoAtendimentoSeg != nil {
				if *ev.DuracaoAtendimentoSeg < 0 {
					return fmt.Errorf("duracao_atendimento_seg é negativo: %d", *ev.DuracaoAtendimentoSeg)
				}
				return nil
			}
		case "tempo_na_fila_seg":
			if ev.TempoNaFilaSeg != nil {
				if *ev.TempoNaFilaSeg < 0 {
					return fmt.Errorf("tempo_na_fila_seg é negativo: %d", *ev.TempoNaFilaSeg)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("campo %q não encontrado com valor >= 0", campo)
}

// "o evento 'atendimento_finalizado' do atendimento anterior não contém 'tempo_na_fila_seg'"
func (c *bddCtx) oEventoNaoContem(tipoEvPT, campo string) error {
	tipo := c.resolverTipoEvento(tipoEvPT)
	for _, ev := range c.eventos {
		if ev.Tipo == tipo {
			switch campo {
			case "tempo_na_fila_seg":
				if ev.TempoNaFilaSeg != nil {
					return fmt.Errorf("evento %q contém %s=%d, mas não deveria (sol não foi enfileirada)", tipoEvPT, campo, *ev.TempoNaFilaSeg)
				}
				return nil
			case "duracao_atendimento_seg":
				if ev.DuracaoAtendimentoSeg != nil {
					return fmt.Errorf("evento %q contém %s, mas não deveria", tipoEvPT, campo)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("evento %q não encontrado", tipoEvPT)
}

// "a operação falha com erro 'atendente possui atendimentos ativos'"
func (c *bddCtx) aOperacaoFalha(errMsg string) error {
	if c.ultimoErro == nil {
		return fmt.Errorf("esperava erro %q, mas a operação teve sucesso", errMsg)
	}
	if !strings.Contains(c.ultimoErro.Error(), errMsg) {
		return fmt.Errorf("esperava erro contendo %q, veio %q", errMsg, c.ultimoErro.Error())
	}
	return nil
}

// "'Ana' permanece não pausada"
func (c *bddCtx) xNaoPausada(nome string) error {
	a, ok := c.atendentes[nome]
	if !ok {
		return fmt.Errorf("atendente %q não encontrado", nome)
	}
	if a.Pausado() {
		return fmt.Errorf("%q está pausada, mas deveria permanecer não pausada", nome)
	}
	return nil
}

// "o time 'Cartões' conta com apenas 'Ana' como atendente"
func (c *bddCtx) oTimeContaComApenas(nomePT, nome string) error {
	tm := c.getTime(nomePT)
	if tm == nil {
		return fmt.Errorf("time %q não encontrado", nomePT)
	}
	atendentes := tm.Atendentes()
	if len(atendentes) != 1 {
		return fmt.Errorf("time %q tem %d atendentes, esperava 1", nomePT, len(atendentes))
	}
	if atendentes[0].Nome != nome {
		return fmt.Errorf("único atendente é %q, esperava %q", atendentes[0].Nome, nome)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (c *bddCtx) resolverTipoEvento(pt string) domain.TipoEvento {
	switch pt {
	case "atendimento_finalizado":
		return domain.EvAtendimentoFinalizado
	case "solicitacao_atribuida":
		return domain.EvSolicitacaoAtribuida
	case "solicitacao_enfileirada":
		return domain.EvSolicitacaoEnfileirada
	case "atendente_pausado":
		return domain.EvAtendentePausado
	case "atendente_retomado":
		return domain.EvAtendenteRetomado
	}
	return domain.TipoEvento(pt)
}

func (c *bddCtx) verificarCampoPreenchido(ev domain.Evento, campo string) error {
	switch campo {
	case "duracao_atendimento_seg":
		if ev.DuracaoAtendimentoSeg == nil {
			return fmt.Errorf("campo duracao_atendimento_seg é nil no evento %q", ev.Tipo)
		}
	case "tempo_na_fila_seg":
		if ev.TempoNaFilaSeg == nil {
			return fmt.Errorf("campo tempo_na_fila_seg é nil no evento %q", ev.Tipo)
		}
	default:
		return fmt.Errorf("campo desconhecido para verificação: %q", campo)
	}
	return nil
}

// ── Wiring ────────────────────────────────────────────────────────────────────

func inicializarCenario(ctx *godog.ScenarioContext) {
	c := &bddCtx{
		times:      make(map[domain.NomeTime]*domain.Time),
		atendentes: make(map[string]*domain.Atendente),
	}

	// Contexto (Background) — executado antes de cada cenário
	ctx.Step(`^o time "([^"]*)" responsável por "([^"]*)"$`, c.oTimeResponsavelPor)
	ctx.Step(`^o time "([^"]*)" para os demais assuntos$`, c.oTimeParaOsDemaisAssuntos)

	// Given
	ctx.Step(`^um atendente "([^"]*)" no time "([^"]*)" sem atendimentos$`, c.atendenteNoTimeSemAtivos)
	ctx.Step(`^um atendente "([^"]*)" no time "([^"]*)" com (\d+) atendimentos? ativos?$`, c.atendenteNoTimeComNAtivos)
	ctx.Step(`^um atendente "([^"]*)" do time "([^"]*)" com (\d+) atendimentos ativos$`, c.atendenteNoTimeComNAtivos)
	ctx.Step(`^que todos os atendentes do time "([^"]*)" estão com 3 atendimentos ativos$`, c.todosOsAtendentesCom3Ativos)
	ctx.Step(`^uma solicitação aguardando na fila do time "([^"]*)"$`, c.umaSolicitacaoNaFilaDoTime)
	ctx.Step(`^uma solicitação de "([^"]*)" está aguardando na fila$`, c.umaSolicitacaoDeNaFila)
	ctx.Step(`^uma solicitação de "([^"]*)" foi enfileirada$`, c.umaSolicitacaoDeNaFila)
	ctx.Step(`^o time "([^"]*)" com apenas um atendente "([^"]*)" com (\d+) atendimentos ativos$`, c.oTimeComApenas1Atendente)
	ctx.Step(`^uma solicitação de "([^"]*)" atribuída a "([^"]*)"$`, c.umaSolAtribuidaA)

	// Given + When (mesmo padrão, mesmo handler)
	ctx.Step(`^"([^"]*)" é pausada$`, c.atendenteEPausado)
	ctx.Step(`^"([^"]*)" é retomada$`, c.atendenteERetomado)
	ctx.Step(`^"([^"]*)" é removido do time "([^"]*)"$`, c.atendenteERemovidoDoTime)

	// When
	ctx.Step(`^chega uma solicitação de "([^"]*)"$`, c.chegaUmaSolicitacaoDe)
	ctx.Step(`^"([^"]*)" finaliza (?:um|o) atendimento$`, c.xFinalizaAtendimento)
	ctx.Step(`^um novo atendente "([^"]*)" é adicionado ao time "([^"]*)"$`, c.umNovoAtendenteEAdicionado)
	ctx.Step(`^um atendente finaliza um atendimento e puxa a solicitação da fila$`, c.umAtendenteFinalizaEPuxa)
	ctx.Step(`^se tenta pausar "([^"]*)"$`, c.seTentaPausar)
	ctx.Step(`^se tenta remover "([^"]*)"$`, c.seTentaRemover)

	// Then — atribuição (padrão cobre todas as variações de "solicitação atribuída a X")
	ctx.Step(`^a solicitação (?:da fila )?é atribuída a "([^"]*)"`, c.aSolAtribuidaA)
	ctx.Step(`^a próxima solicitação da fila é atribuída a "([^"]*)"$`, c.aSolAtribuidaA)
	ctx.Step(`^a solicitação da fila é atribuída a "([^"]*)" sem aguardar nova chegada$`, c.aSolAtribuidaA)
	ctx.Step(`^a solicitação é atribuída a "([^"]*)" pois tem menor carga$`, c.aSolAtribuidaA)
	ctx.Step(`^a solicitação não é atribuída a "([^"]*)"$`, c.aSolNaoAtribuidaA)
	ctx.Step(`^a solicitação é enfileirada no time "([^"]*)"$`, c.aSolEnfileiraNoTime)

	// Then — atendentes
	ctx.Step(`^"([^"]*)" passa a ter (\d+) atendimentos? (?:ativo|ativos)$`, c.xTemNAtivos)
	ctx.Step(`^"([^"]*)" continua com (\d+) atendimentos? ativos$`, c.xTemNAtivos)
	ctx.Step(`^"([^"]*)" continua sem atendimentos ativos$`, c.xSemAtivos)
	ctx.Step(`^"([^"]*)" permanece não pausada$`, c.xNaoPausada)
	ctx.Step(`^o time "([^"]*)" conta com apenas "([^"]*)" como atendente$`, c.oTimeContaComApenas)

	// Then — snapshot e eventos
	ctx.Step(`^o snapshot do time "([^"]*)" lista a solicitação ativa de "([^"]*)"$`, c.oSnapshotLista)
	ctx.Step(`^o evento "([^"]*)" contém o campo "([^"]*)" preenchido$`, c.oEventoContemCampo)
	ctx.Step(`^o valor de "([^"]*)" é maior ou igual a zero$`, c.oValorMaiorOuIgualZero)
	ctx.Step(`^o evento "([^"]*)" do atendimento anterior não contém "([^"]*)"$`, c.oEventoNaoContem)

	// Then — erros de negócio
	ctx.Step(`^a operação falha com erro "([^"]*)"$`, c.aOperacaoFalha)
}
