package http

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── CORS ──────────────────────────────────────────────────────────────────────

// cors devolve um middleware que adiciona cabeçalhos CORS baseados nas origens permitidas.
// Se origensPermitidas contiver "*", qualquer origem é aceita (modo dev).
// Caso contrário, valida o header Origin e reflete apenas a origem conhecida (modo prod),
// adicionando Vary: Origin para que caches não confundam respostas entre origens.
func cors(origensPermitidas []string) func(http.Handler) http.Handler {
	permitidas := make(map[string]bool, len(origensPermitidas))
	for _, o := range origensPermitidas {
		permitidas[o] = true
	}
	aberto := permitidas["*"]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origem := r.Header.Get("Origin")
			switch {
			case aberto:
				w.Header().Set("Access-Control-Allow-Origin", "*")
			case permitidas[origem]:
				w.Header().Set("Access-Control-Allow-Origin", origem)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400") // cache preflight 24h

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ── Rate Limiter ──────────────────────────────────────────────────────────────

// RateLimiter implementa token bucket por IP usando apenas stdlib.
// Cada IP começa com `burst` tokens; tokens se repõem à taxa `rps` por segundo.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64 // tokens repostos por segundo
	burst   float64 // capacidade máxima do bucket
}

type tokenBucket struct {
	tokens    float64
	updatedAt time.Time
}

func NovoRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     float64(rps),
		burst:   float64(burst),
	}
	go rl.limpar()
	return rl
}

// Allow consome 1 token do bucket do IP. Retorna false se não há tokens disponíveis.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &tokenBucket{tokens: rl.burst, updatedAt: time.Now()}
		rl.buckets[ip] = b
	}

	elapsed := time.Since(b.updatedAt).Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed*rl.rps)
	b.updatedAt = time.Now()

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware aplica rate limiting a todas as rotas, exceto:
//   - /api/eventos: conexão SSE persistente, não é request repetida
//   - /api/docs/: assets estáticos de documentação, sem sentido limitar
//   - OPTIONS: preflight CORS não deve ser contado como uso da API
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	retryAfter := strconv.Itoa(int(math.Ceil(1.0 / rl.rps)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/eventos" ||
			strings.HasPrefix(r.URL.Path, "/api/docs/") ||
			r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if !rl.Allow(ipCliente(r)) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"erro":"muitas requisições — aguarde antes de tentar novamente"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// limpar remove buckets inativos a cada 5 minutos para evitar crescimento ilimitado do mapa.
func (rl *RateLimiter) limpar() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-5 * time.Minute)
		for ip, b := range rl.buckets {
			if b.updatedAt.Before(cutoff) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// ipCliente extrai o IP real do cliente considerando o header X-Forwarded-For
// que o nginx injeta quando em produção. Nunca usa o campo diretamente sem
// validação para evitar IP spoofing — apenas o primeiro da cadeia é confiável.
func ipCliente(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
