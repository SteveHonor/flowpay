// Testes do adapter HTTP: verificam status codes, serialização JSON e tradução de erros de domínio.
// Regras de negócio (distribuição, balanceamento, fila) são testadas em application/distribuidor_test.go.
package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"flowpay/internal/atendimento/application"
	"flowpay/internal/atendimento/domain"
	web "flowpay/internal/atendimento/infrastructure/http"
	"flowpay/internal/atendimento/infrastructure/memoria"
	"flowpay/internal/atendimento/infrastructure/sse"
)

func novoServidor(t *testing.T) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := sse.NovoHub()
	dist := application.NovoDistribuidor(memoria.NovoRepositorio(), hub,
		domain.NovoTime(domain.TimeCartoes, domain.NovoAtendente("c1", "Ana", domain.TimeCartoes)),
		domain.NovoTime(domain.TimeEmprestimos, domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos)),
		domain.NovoTime(domain.TimeOutros, domain.NovoAtendente("o1", "Elena", domain.TimeOutros)),
	)
	// Origens abertas e rate limit generoso nos testes de integração.
	rl := web.NovoRateLimiter(1000, 10000)
	return httptest.NewServer(web.NovoRouter(web.NovosHandlers(dist, log), hub.Handler, log, []string{"*"}, rl))
}

// POST /api/solicitacoes → 201 com ID no corpo.
func TestHTTP_CriarSolicitacao_Retorna201EID(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	id := postJSON(t, srv.URL+"/api/solicitacoes", map[string]string{"assunto": "problema_cartao"}, http.StatusCreated)
	if id == "" {
		t.Error("esperava campo 'id' no corpo da resposta")
	}
}

// POST /api/solicitacoes com assunto inválido → 400.
func TestHTTP_CriarSolicitacaoAssuntoInvalido_Retorna400(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postJSONStatus(t, srv.URL+"/api/solicitacoes", map[string]string{"assunto": ""}, http.StatusBadRequest)
}

// POST /api/solicitacoes/{id}/finalizar → 204.
func TestHTTP_FinalizarAtendimento_Retorna204(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	id := postJSON(t, srv.URL+"/api/solicitacoes", map[string]string{"assunto": "problema_cartao"}, http.StatusCreated)
	postStatus(t, srv.URL+"/api/solicitacoes/"+id+"/finalizar", http.StatusNoContent)
}

// POST /api/solicitacoes/{id}/finalizar com ID inexistente → 404.
func TestHTTP_FinalizarInexistente_Retorna404(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postStatus(t, srv.URL+"/api/solicitacoes/sol_inexistente/finalizar", http.StatusNotFound)
}

// GET /api/dashboard → 200 com estrutura times[].
func TestHTTP_Dashboard_RetornaEstrutura(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", resp.StatusCode)
	}
	var v application.DashboardView
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("corpo não é JSON válido: %v", err)
	}
	if len(v.Times) != 3 {
		t.Errorf("esperava 3 times no snapshot, veio %d", len(v.Times))
	}
}

// POST /api/atendentes → 201 com ID no corpo.
func TestHTTP_AdicionarAtendente_Retorna201EID(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	id := postJSON(t, srv.URL+"/api/atendentes", map[string]string{"nome": "Bruno", "time": "CARTOES"}, http.StatusCreated)
	if id == "" {
		t.Error("esperava campo 'id' no corpo da resposta")
	}
}

// POST /api/atendentes com time inexistente → 422 (ErrTimeNaoEncontrado → UnprocessableEntity).
func TestHTTP_AdicionarAtendenteTimeInvalido_Retorna422(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postJSONStatus(t, srv.URL+"/api/atendentes", map[string]string{"nome": "X", "time": "INEXISTENTE"}, http.StatusUnprocessableEntity)
}

// POST /api/atendentes sem nome → 400 (validação do adapter).
func TestHTTP_AdicionarAtendenteSemNome_Retorna400(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postJSONStatus(t, srv.URL+"/api/atendentes", map[string]string{"nome": "", "time": "CARTOES"}, http.StatusBadRequest)
}

// POST /api/atendentes/{id}/pausar → 204.
func TestHTTP_PausarAtendente_Retorna204(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postStatus(t, srv.URL+"/api/atendentes/c1/pausar", http.StatusNoContent)
}

// POST /api/atendentes/{id}/retomar → 204.
func TestHTTP_RetomarAtendente_Retorna204(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postStatus(t, srv.URL+"/api/atendentes/c1/pausar", http.StatusNoContent)
	postStatus(t, srv.URL+"/api/atendentes/c1/retomar", http.StatusNoContent)
}

// DELETE /api/atendentes/{id} sem ativos → 204.
func TestHTTP_RemoverAtendente_Retorna204(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	deleteStatus(t, srv.URL+"/api/atendentes/c1", http.StatusNoContent)
}

// DELETE /api/atendentes/{id} com ativos → 409.
func TestHTTP_RemoverAtendenteComAtivos_Retorna409(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postJSON(t, srv.URL+"/api/solicitacoes", map[string]string{"assunto": "problema_cartao"}, http.StatusCreated)

	// Determina quem ficou com o ativo via snapshot.
	resp, err := http.Get(srv.URL + "/api/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var snap application.DashboardView
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("erro ao decodificar snapshot: %v", err)
	}
	var comAtivo string
	for _, tv := range snap.Times {
		if tv.Nome == domain.TimeCartoes {
			for _, a := range tv.Atendentes {
				if a.Ativos > 0 {
					comAtivo = a.ID
				}
			}
		}
	}
	if comAtivo == "" {
		t.Fatal("nenhum atendente com ativo encontrado no snapshot")
	}
	deleteStatus(t, srv.URL+"/api/atendentes/"+comAtivo, http.StatusConflict)
}

// POST /api/atendentes/{id}/pausar com atendente que tem ativos → 409.
func TestHTTP_PausarAtendenteComAtivos_Retorna409(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	// c1 (Ana) recebe a solicitação → fica com 1 ativo.
	postJSON(t, srv.URL+"/api/solicitacoes", map[string]string{"assunto": "problema_cartao"}, http.StatusCreated)

	postStatus(t, srv.URL+"/api/atendentes/c1/pausar", http.StatusConflict)
}

// POST /api/atendentes/{id}/pausar com ID inexistente → 404.
func TestHTTP_PausarAtendenteInexistente_Retorna404(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	postStatus(t, srv.URL+"/api/atendentes/inexistente/pausar", http.StatusNotFound)
}

// GET /api/health → 200.
func TestHTTP_Health_Retorna200(t *testing.T) {
	srv := novoServidor(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", resp.StatusCode)
	}
}

// --- helpers ---

// postJSON faz POST com corpo JSON, verifica o status esperado e retorna o campo "id" do corpo.
func postJSON(t *testing.T, url string, body map[string]string, statusEsperado int) string {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != statusEsperado {
		t.Fatalf("POST %s: esperava %d, veio %d", url, statusEsperado, resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.ID
}

// postJSONStatus faz POST com corpo JSON e verifica apenas o status — sem retorno.
func postJSONStatus(t *testing.T, url string, body map[string]string, statusEsperado int) {
	t.Helper()
	postJSON(t, url, body, statusEsperado)
}

// postStatus faz POST sem corpo e verifica o status.
func postStatus(t *testing.T, url string, statusEsperado int) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != statusEsperado {
		t.Fatalf("POST %s: esperava %d, veio %d", url, statusEsperado, resp.StatusCode)
	}
}

// deleteStatus faz DELETE sem corpo e verifica o status.
func deleteStatus(t *testing.T, url string, statusEsperado int) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != statusEsperado {
		t.Fatalf("DELETE %s: esperava %d, veio %d", url, statusEsperado, resp.StatusCode)
	}
}
