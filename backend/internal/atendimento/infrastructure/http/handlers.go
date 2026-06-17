package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"flowpay/internal/atendimento/application"
	"flowpay/internal/atendimento/domain"
)

type Handlers struct {
	dist *application.Distribuidor
	log  *slog.Logger
}

func NovosHandlers(dist *application.Distribuidor, log *slog.Logger) *Handlers {
	return &Handlers{dist: dist, log: log}
}

func (h *Handlers) CriarSolicitacao(w http.ResponseWriter, r *http.Request) {
	var req criarSolicitacaoReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		escreverErro(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	id, err := h.dist.RegistrarSolicitacao(r.Context(), req.Assunto)
	if err != nil {
		h.traduzirErro(w, err)
		return
	}
	escreverJSON(w, http.StatusCreated, criarSolicitacaoResp{ID: id})
}

func (h *Handlers) FinalizarAtendimento(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.dist.FinalizarAtendimento(r.Context(), id); err != nil {
		h.traduzirErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	escreverJSON(w, http.StatusOK, h.dist.Snapshot())
}

func (h *Handlers) Historico(w http.ResponseWriter, r *http.Request) {
	limite, _ := strconv.Atoi(r.URL.Query().Get("limite"))
	eventos, err := h.dist.Historico(r.Context(), limite)
	if err != nil {
		h.traduzirErro(w, err)
		return
	}
	if eventos == nil {
		eventos = []domain.Evento{}
	}
	escreverJSON(w, http.StatusOK, eventos)
}

func (h *Handlers) AdicionarAtendente(w http.ResponseWriter, r *http.Request) {
	var req adicionarAtendenteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		escreverErro(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if req.Nome == "" || req.Time == "" {
		escreverErro(w, http.StatusBadRequest, "nome e time são obrigatórios")
		return
	}
	id, err := h.dist.AdicionarAtendente(r.Context(), req.Nome, domain.NomeTime(req.Time))
	if err != nil {
		h.traduzirErro(w, err)
		return
	}
	escreverJSON(w, http.StatusCreated, adicionarAtendenteResp{ID: id})
}

func (h *Handlers) PausarAtendente(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.dist.PausarAtendente(r.Context(), id); err != nil {
		h.traduzirErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) RetomarAtendente(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.dist.RetomarAtendente(r.Context(), id); err != nil {
		h.traduzirErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) RemoverAtendente(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.dist.RemoverAtendente(r.Context(), id); err != nil {
		h.traduzirErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	escreverJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) traduzirErro(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAssuntoInvalido):
		escreverErro(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrSolicitacaoNaoEncontrada),
		errors.Is(err, domain.ErrAtendenteNaoEncontrado):
		escreverErro(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrTimeNaoEncontrado):
		escreverErro(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrAtendenteComAtivos):
		escreverErro(w, http.StatusConflict, err.Error())
	default:
		h.log.Error("erro interno", "erro", err)
		escreverErro(w, http.StatusInternalServerError, "erro interno")
	}
}

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func escreverErro(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, erroResp{Erro: msg})
}
