package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"flowpay/internal/atendimento/application"
	"flowpay/internal/atendimento/domain"
	web "flowpay/internal/atendimento/infrastructure/http"
	"flowpay/internal/atendimento/infrastructure/memoria"
	"flowpay/internal/atendimento/infrastructure/postgres"
	"flowpay/internal/atendimento/infrastructure/sse"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	repo, fechar := abrirRepositorio(log)
	defer fechar()

	hub := sse.NovoHub()
	dist := application.NovoDistribuidor(repo, hub)
	aplicarSeedInicial(dist, log)
	handlers := web.NovosHandlers(dist, log)

	origens := origensPermitidas()
	rl := novoRateLimiter()
	log.Info("segurança", "origens_permitidas", origens, "rate_limit_rps", rl)

	router := web.NovoRouter(handlers, hub.Handler, log, origens, rl)

	srv := &http.Server{
		Addr:              ":" + porta(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		// sem WriteTimeout: o endpoint SSE mantém a conexão aberta indefinidamente.
	}

	go func() {
		log.Info("servidor iniciado", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("falha no servidor", "erro", err)
			os.Exit(1)
		}
	}()

	parar := make(chan os.Signal, 1)
	signal.Notify(parar, syscall.SIGINT, syscall.SIGTERM)
	<-parar

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Info("encerrando servidor")
	_ = srv.Shutdown(ctx)
}

func abrirRepositorio(log *slog.Logger) (domain.Repositorio, func()) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Info("usando repositório em memória (sem DATABASE_URL)")
		return memoria.NovoRepositorio(), func() {}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pg, err := postgres.Conectar(ctx, dsn)
	if err != nil {
		log.Warn("falha ao conectar no Postgres, caindo para memória", "erro", err)
		return memoria.NovoRepositorio(), func() {}
	}
	log.Info("usando repositório Postgres")
	return pg, func() { _ = pg.Close() }
}

// origensPermitidas lê ALLOWED_ORIGINS (vírgula-separado).
// Padrão: "*" (qualquer origem — adequado para dev/demo).
// Em produção, configure com a URL real: ALLOWED_ORIGINS=https://app.flowpay.com.br
func origensPermitidas() []string {
	v := os.Getenv("ALLOWED_ORIGINS")
	if v == "" {
		return []string{"*"}
	}
	var out []string
	for _, o := range strings.Split(v, ",") {
		if s := strings.TrimSpace(o); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// novoRateLimiter lê RATE_LIMIT_RPS e RATE_LIMIT_BURST do ambiente.
// Padrão: 10 req/s com burst de 30 — adequado para uso interativo.
func novoRateLimiter() *web.RateLimiter {
	rps := envInt("RATE_LIMIT_RPS", 10)
	burst := envInt("RATE_LIMIT_BURST", 30)
	return web.NovoRateLimiter(rps, burst)
}

func envInt(key string, padrao int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return padrao
}

func porta() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
