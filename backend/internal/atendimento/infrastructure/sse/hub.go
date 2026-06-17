package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"flowpay/internal/atendimento/domain"
)

// Hub faz o broadcast de eventos de domínio para os clientes conectados via SSE.
// Implementa application.Publicador.
type Hub struct {
	mu       sync.Mutex
	clientes map[chan domain.Evento]struct{}
}

func NovoHub() *Hub {
	return &Hub{clientes: make(map[chan domain.Evento]struct{})}
}

// Publicar envia o evento a todos os inscritos sem bloquear (descarta se o cliente estiver lento).
func (h *Hub) Publicar(e domain.Evento) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clientes {
		select {
		case ch <- e:
		default:
		}
	}
}

func (h *Hub) inscrever() chan domain.Evento {
	ch := make(chan domain.Evento, 16)
	h.mu.Lock()
	h.clientes[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) desinscrever(ch chan domain.Evento) {
	h.mu.Lock()
	delete(h.clientes, ch)
	h.mu.Unlock()
	close(ch)
}

// Handler é o endpoint SSE. Mantém a conexão aberta transmitindo eventos.
func (h *Hub) Handler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := h.inscrever()
	defer h.desinscrever(ch)

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	fmt.Fprint(w, ": conectado\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case e := <-ch:
			payload, err := json.Marshal(e)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Tipo, payload)
			flusher.Flush()
		}
	}
}
