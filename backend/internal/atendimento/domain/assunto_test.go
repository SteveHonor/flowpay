package domain

import "testing"

func TestAssunto_RoteamentoParaTime(t *testing.T) {
	casos := []struct {
		codigo   string
		esperado NomeTime
	}{
		{AssuntoProblemaCartao, TimeCartoes},
		{AssuntoContratacaoEmprestimo, TimeEmprestimos},
		{"atualizacao_cadastral", TimeOutros},
		{"qualquer_outra_coisa", TimeOutros},
	}
	for _, c := range casos {
		a, err := NovoAssunto(c.codigo)
		if err != nil {
			t.Fatalf("assunto %q: %v", c.codigo, err)
		}
		if got := a.Time(); got != c.esperado {
			t.Errorf("assunto %q: esperava %q, veio %q", c.codigo, c.esperado, got)
		}
	}
}

func TestNovoAssunto_Invalido(t *testing.T) {
	if _, err := NovoAssunto("   "); err != ErrAssuntoInvalido {
		t.Errorf("esperava ErrAssuntoInvalido, veio %v", err)
	}
}
