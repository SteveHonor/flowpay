package http

import (
	"embed"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

//go:embed docs/openapi.yaml
var specFS embed.FS

// specHandler serve o openapi.yaml embutido no binário.
// Disponível em GET /api/docs/openapi.yaml.
func specHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	data, _ := specFS.ReadFile("docs/openapi.yaml")
	_, _ = w.Write(data)
}

// swaggerUIHandler serve o Swagger UI apontando para a spec embutida.
// Disponível em GET /api/docs/ (e sub-paths de assets).
var swaggerUIHandler = httpSwagger.Handler(
	httpSwagger.URL("/api/docs/openapi.yaml"),
	httpSwagger.DeepLinking(true),
	httpSwagger.DocExpansion("list"),
)
