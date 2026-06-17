// Testes internos do pacote http: acessam diretamente cors() (unexported) e RateLimiter.
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// ── Rate Limiter ──────────────────────────────────────────────────────────────

func TestRateLimiter_PermiteAbaixoDoLimite(t *testing.T) {
	rl := NovoRateLimiter(10, 5)
	for i := range 5 {
		if !rl.Allow("127.0.0.1") {
			t.Fatalf("requisição %d deveria ser permitida (burst=5)", i+1)
		}
	}
}

func TestRateLimiter_BloqueiaAcimaDoLimite(t *testing.T) {
	rl := NovoRateLimiter(10, 3)
	for range 3 {
		rl.Allow("127.0.0.1")
	}
	if rl.Allow("127.0.0.1") {
		t.Error("4ª requisição deveria ser bloqueada (burst=3 esgotado)")
	}
}

func TestRateLimiter_RepoeTokensComTempo(t *testing.T) {
	rl := NovoRateLimiter(100, 1) // 100 tok/s → 1 token a cada 10ms
	rl.Allow("127.0.0.1")        // consome o único token
	if rl.Allow("127.0.0.1") {
		t.Fatal("deveria estar bloqueado antes de repor")
	}
	time.Sleep(20 * time.Millisecond) // aguarda ~2 tokens repostos
	if !rl.Allow("127.0.0.1") {
		t.Error("deveria ser permitido após reposição")
	}
}

func TestRateLimiter_IsolaIPsDiferentes(t *testing.T) {
	rl := NovoRateLimiter(10, 1)
	rl.Allow("192.168.0.1") // esgota burst do IP A
	if !rl.Allow("192.168.0.2") {
		t.Error("IP diferente deve ter seu próprio bucket")
	}
}

func TestRateLimiter_Middleware_Retorna429QuandoBloqueado(t *testing.T) {
	rl := NovoRateLimiter(10, 0) // burst=0 → bloqueia imediatamente
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	rl.Middleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("esperava 429, veio %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("esperava header Retry-After na resposta 429")
	}
}

func TestRateLimiter_Middleware_NaoLimitaSSE(t *testing.T) {
	rl := NovoRateLimiter(10, 0) // bloquearia qualquer outro endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/eventos", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	rl.Middleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/api/eventos não deve ser limitado; veio %d", rr.Code)
	}
}

func TestRateLimiter_Middleware_NaoLimitaOptions(t *testing.T) {
	rl := NovoRateLimiter(10, 0)
	req := httptest.NewRequest(http.MethodOptions, "/api/solicitacoes", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	rl.Middleware(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("preflight OPTIONS não deve ser limitado; veio %d", rr.Code)
	}
}

// ── CORS ──────────────────────────────────────────────────────────────────────

func corsMW(origens []string) http.Handler {
	return cors(origens)(okHandler)
}

func reqComOrigem(metodo, path, origem string) *http.Request {
	r := httptest.NewRequest(metodo, path, nil)
	if origem != "" {
		r.Header.Set("Origin", origem)
	}
	return r
}

func TestCORS_WildcardPermiteQualquerOrigem(t *testing.T) {
	rr := httptest.NewRecorder()
	corsMW([]string{"*"}).ServeHTTP(rr, reqComOrigem(http.MethodOptions, "/api/health", "https://qualquer.com"))
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("esperava '*', veio %q", got)
	}
}

func TestCORS_OrigemEspecificaPermitida(t *testing.T) {
	rr := httptest.NewRecorder()
	corsMW([]string{"https://app.flowpay.com.br"}).ServeHTTP(
		rr, reqComOrigem(http.MethodOptions, "/api/health", "https://app.flowpay.com.br"),
	)
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.flowpay.com.br" {
		t.Errorf("esperava origem refletida, veio %q", got)
	}
	if rr.Header().Get("Vary") == "" {
		t.Error("esperava header Vary quando origem é específica")
	}
}

func TestCORS_OrigemDesconhecidaNaoERefletida(t *testing.T) {
	rr := httptest.NewRecorder()
	corsMW([]string{"https://app.flowpay.com.br"}).ServeHTTP(
		rr, reqComOrigem(http.MethodOptions, "/api/health", "https://atacante.com"),
	)
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origem desconhecida não deveria ser refletida, veio %q", got)
	}
}

func TestCORS_PreflightRetorna204(t *testing.T) {
	rr := httptest.NewRecorder()
	corsMW([]string{"*"}).ServeHTTP(rr, reqComOrigem(http.MethodOptions, "/api/solicitacoes", "https://app.com"))
	if rr.Code != http.StatusNoContent {
		t.Errorf("preflight deveria retornar 204, veio %d", rr.Code)
	}
}

func TestCORS_MaxAgeCacheadoNasPreflight(t *testing.T) {
	rr := httptest.NewRecorder()
	corsMW([]string{"*"}).ServeHTTP(rr, reqComOrigem(http.MethodOptions, "/api/health", "https://app.com"))
	if got := rr.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("esperava Access-Control-Max-Age=86400, veio %q", got)
	}
}
