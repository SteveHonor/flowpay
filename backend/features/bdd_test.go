//go:build bdd

package features_test

import (
	"testing"

	"github.com/cucumber/godog"
)

// TestBDD executa todos os cenários Gherkin de features/distribuicao.feature
// contra o aggregate Time do domínio (sem HTTP, sem banco).
// Rode com: go test -v ./features/... (ou make bdd)
func TestBDD(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: inicializarCenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features"},
			Strict:   true,
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("um ou mais cenários BDD falharam — veja saída acima")
	}
}
