package domain

import "errors"

var (
	ErrAssuntoInvalido          = errors.New("assunto inválido")
	ErrAtendenteLotado          = errors.New("atendente já está com capacidade máxima")
	ErrTimeNaoEncontrado        = errors.New("time não encontrado")
	ErrSolicitacaoNaoEncontrada = errors.New("solicitação em atendimento não encontrada")
	ErrAtendenteNaoEncontrado   = errors.New("atendente não encontrado")
	ErrAtendenteComAtivos       = errors.New("atendente possui atendimentos ativos")
)
