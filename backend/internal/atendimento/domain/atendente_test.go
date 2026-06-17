package domain

import (
	"sort"
	"testing"
	"time"
)

func TestAtendente_SolicitacoesAtivasVazio(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	if ids := a.SolicitacoesAtivas(); len(ids) != 0 {
		t.Errorf("esperava slice vazio, veio %v", ids)
	}
}

func TestAtendente_SolicitacoesAtivasRefleteSomentePendentes(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	a.Atribuir("s1", nil)
	a.Atribuir("s2", nil)
	a.Atribuir("s3", nil)

	ids := a.SolicitacoesAtivas()
	sort.Strings(ids)
	esperado := []string{"s1", "s2", "s3"}

	if len(ids) != len(esperado) {
		t.Fatalf("esperava %d IDs, veio %d: %v", len(esperado), len(ids), ids)
	}
	for i, id := range ids {
		if id != esperado[i] {
			t.Errorf("posição %d: esperava %q, veio %q", i, esperado[i], id)
		}
	}
}

// Cenário BDD: Expor solicitações ativas por atendente no snapshot.
// Após atribuir e liberar, apenas a pendente deve aparecer.
func TestAtendente_SolicitacoesAtivasAposLiberar(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	a.Atribuir("s1", nil)
	a.Atribuir("s2", nil)
	a.Liberar("s1")

	ids := a.SolicitacoesAtivas()
	if len(ids) != 1 || ids[0] != "s2" {
		t.Errorf("esperava [s2], veio %v", ids)
	}
}

// Liberar deve retornar duracao > 0 e ok=true para solicitação conhecida.
func TestAtendente_LiberarRetornaDuracao(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	a.Atribuir("s1", nil)
	duracao, tempoFila, ok := a.Liberar("s1")
	if !ok {
		t.Fatal("esperava ok=true ao liberar solicitação conhecida")
	}
	if duracao < 0 {
		t.Errorf("duracao não pode ser negativa, veio %v", duracao)
	}
	if tempoFila != nil {
		t.Errorf("tempoNaFila deveria ser nil para atribuição direta, veio %v", tempoFila)
	}
}

// Liberar com solicitação que veio da fila deve retornar tempoNaFila != nil.
func TestAtendente_LiberarRetornaTempoNaFilaQuandoEnfileirada(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	agora := time.Now().UTC()
	a.Atribuir("s1", &agora)
	_, tempoFila, ok := a.Liberar("s1")
	if !ok {
		t.Fatal("esperava ok=true")
	}
	if tempoFila == nil {
		t.Fatal("esperava tempoNaFila != nil para solicitação que veio da fila")
	}
	if *tempoFila < 0 {
		t.Errorf("tempoNaFila não pode ser negativo, veio %v", *tempoFila)
	}
}

// Liberar solicitação inexistente deve retornar ok=false.
func TestAtendente_LiberarInexistente(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	_, _, ok := a.Liberar("nao-existe")
	if ok {
		t.Error("esperava ok=false ao liberar solicitação inexistente")
	}
}

// EstaDisponivel retorna false quando pausado, mesmo com vaga.
func TestAtendente_EstaDisponivelQuandoPausado(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	if !a.EstaDisponivel() {
		t.Error("atendente sem ativos e não pausado deveria estar disponível")
	}
	a.Pausar()
	if a.EstaDisponivel() {
		t.Error("atendente pausado não deveria estar disponível")
	}
	a.Retomar()
	if !a.EstaDisponivel() {
		t.Error("atendente retomado deveria estar disponível novamente")
	}
}

// EstaDisponivel retorna false quando lotado (mesmo não pausado).
func TestAtendente_EstaDisponivelQuandoLotado(t *testing.T) {
	a := NovoAtendente("a1", "Ana", TimeCartoes)
	for _, id := range []string{"s1", "s2", "s3"} {
		a.Atribuir(id, nil)
	}
	if a.EstaDisponivel() {
		t.Error("atendente lotado não deveria estar disponível")
	}
}

// Table-driven: garante que Ativos() e SolicitacoesAtivas() são sempre consistentes.
func TestAtendente_AtivosConsistenteComSolicitacoesAtivas(t *testing.T) {
	casos := []struct {
		atribuir []string
		liberar  []string
		esperado int
	}{
		{atribuir: nil, liberar: nil, esperado: 0},
		{atribuir: []string{"s1"}, liberar: nil, esperado: 1},
		{atribuir: []string{"s1", "s2"}, liberar: []string{"s1"}, esperado: 1},
		{atribuir: []string{"s1", "s2", "s3"}, liberar: []string{"s1", "s2", "s3"}, esperado: 0},
	}
	for _, c := range casos {
		a := NovoAtendente("x", "X", TimeCartoes)
		for _, id := range c.atribuir {
			a.Atribuir(id, nil)
		}
		for _, id := range c.liberar {
			a.Liberar(id)
		}
		if got := len(a.SolicitacoesAtivas()); got != c.esperado {
			t.Errorf("atribuir=%v liberar=%v: esperava %d ativos, veio %d",
				c.atribuir, c.liberar, c.esperado, got)
		}
		if a.Ativos() != c.esperado {
			t.Errorf("Ativos() e SolicitacoesAtivas() divergem: Ativos=%d len=%d",
				a.Ativos(), len(a.SolicitacoesAtivas()))
		}
	}
}
