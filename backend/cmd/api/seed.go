package main

import (
	"log/slog"

	"flowpay/internal/atendimento/application"
	"flowpay/internal/atendimento/domain"
)

// aplicarSeedInicial popula os 3 times com atendentes de exemplo apenas na primeira
// chamada do processo. Chamadas subsequentes (ex: endpoint de admin) são no-op.
func aplicarSeedInicial(d *application.Distribuidor, log *slog.Logger) {
	times := []*domain.Time{
		domain.NovoTime(domain.TimeCartoes,
			domain.NovoAtendente("c1", "Ana", domain.TimeCartoes),
			domain.NovoAtendente("c2", "Bruno", domain.TimeCartoes),
		),
		domain.NovoTime(domain.TimeEmprestimos,
			domain.NovoAtendente("e1", "Carla", domain.TimeEmprestimos),
			domain.NovoAtendente("e2", "Diego", domain.TimeEmprestimos),
		),
		domain.NovoTime(domain.TimeOutros,
			domain.NovoAtendente("o1", "Elena", domain.TimeOutros),
			domain.NovoAtendente("o2", "Felipe", domain.TimeOutros),
		),
	}
	if d.SeedTimes(times...) {
		log.Info("seed inicial aplicado", "atendentes", 6)
	} else {
		log.Info("seed ignorado: times já possuem atendentes")
	}
}
