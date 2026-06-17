package http

import (
	"log/slog"
	"net/http"
	"time"
)

// NovoRouter monta as rotas REST + SSE + docs e aplica os middlewares em camadas:
// logRequest → cors → rateLimiter → mux
func NovoRouter(h *Handlers, sseHandler http.HandlerFunc, log *slog.Logger, origens []string, rl *RateLimiter) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("POST /api/atendentes", h.AdicionarAtendente)
	mux.HandleFunc("POST /api/atendentes/{id}/pausar", h.PausarAtendente)
	mux.HandleFunc("POST /api/atendentes/{id}/retomar", h.RetomarAtendente)
	mux.HandleFunc("DELETE /api/atendentes/{id}", h.RemoverAtendente)
	mux.HandleFunc("POST /api/solicitacoes", h.CriarSolicitacao)
	mux.HandleFunc("POST /api/solicitacoes/{id}/finalizar", h.FinalizarAtendimento)
	mux.HandleFunc("GET /api/dashboard", h.Dashboard)
	mux.HandleFunc("GET /api/historico", h.Historico)
	mux.HandleFunc("GET /api/eventos", sseHandler)

	// Documentação interativa — spec embutida no binário, UI via http-swagger.
	// Excluída do rate limiting (ver middleware.go).
	mux.HandleFunc("GET /api/docs/openapi.yaml", specHandler)
	mux.Handle("/api/docs/", swaggerUIHandler)

	return logRequest(log, cors(origens)(rl.Middleware(mux)))
}

func logRequest(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		next.ServeHTTP(w, r)
		log.Info("requisição",
			"metodo", r.Method, "rota", r.URL.Path, "duracao", time.Since(inicio).String())
	})
}
