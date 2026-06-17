package http

type criarSolicitacaoReq struct {
	Assunto string `json:"assunto"`
}

type criarSolicitacaoResp struct {
	ID string `json:"id"`
}

type adicionarAtendenteReq struct {
	Nome string `json:"nome"`
	Time string `json:"time"`
}

type adicionarAtendenteResp struct {
	ID string `json:"id"`
}

type erroResp struct {
	Erro string `json:"erro"`
}
