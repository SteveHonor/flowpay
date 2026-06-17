package domain

import "strings"

// NomeTime identifica um time de atendimento.
type NomeTime string

const (
	TimeCartoes     NomeTime = "CARTOES"
	TimeEmprestimos NomeTime = "EMPRESTIMOS"
	TimeOutros      NomeTime = "OUTROS"
)

// Códigos canônicos de assunto aceitos pela API.
const (
	AssuntoProblemaCartao        = "problema_cartao"
	AssuntoContratacaoEmprestimo = "contratacao_emprestimo"
)

// Assunto é um value object: imutável e auto-validado.
type Assunto struct {
	codigo string
}

// NovoAssunto valida e cria um Assunto. Assunto vazio é inválido;
// qualquer outro texto é aceito e roteado para o time "Outros Assuntos".
func NovoAssunto(codigo string) (Assunto, error) {
	c := strings.TrimSpace(strings.ToLower(codigo))
	if c == "" {
		return Assunto{}, ErrAssuntoInvalido
	}
	return Assunto{codigo: c}, nil
}

func (a Assunto) Codigo() string { return a.codigo }

// Time aplica a política de roteamento Assunto -> Time.
func (a Assunto) Time() NomeTime {
	switch a.codigo {
	case AssuntoProblemaCartao:
		return TimeCartoes
	case AssuntoContratacaoEmprestimo:
		return TimeEmprestimos
	default:
		return TimeOutros
	}
}
