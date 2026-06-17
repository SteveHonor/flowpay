package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"flowpay/internal/atendimento/domain"

	_ "github.com/lib/pq"
)

// Repositorio Postgres. Implementa o port domain.Repositorio com database/sql + lib/pq.
type Repositorio struct {
	db *sql.DB
}

// Conectar abre a conexão e garante o schema (idempotente).
func Conectar(ctx context.Context, dsn string) (*Repositorio, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir conexão: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	r := &Repositorio{db: db}
	if err := r.migrar(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Repositorio) Close() error { return r.db.Close() }

func (r *Repositorio) migrar(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS solicitacoes (
    id          TEXT PRIMARY KEY,
    assunto     TEXT NOT NULL,
    time_nome   TEXT NOT NULL,
    status      TEXT NOT NULL,
    criada_em   TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS eventos (
    id              BIGSERIAL PRIMARY KEY,
    tipo            TEXT NOT NULL,
    solicitacao_id  TEXT NOT NULL,
    time_nome       TEXT NOT NULL,
    atendente_id    TEXT,
    atendente_nome  TEXT,
    posicao_fila    INT,
    ocorreu         TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_eventos_ocorreu ON eventos (ocorreu DESC);`
	_, err := r.db.ExecContext(ctx, ddl)
	return err
}

func (r *Repositorio) SalvarSolicitacao(ctx context.Context, s domain.Solicitacao) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO solicitacoes (id, assunto, time_nome, status, criada_em)
		 VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO NOTHING`,
		s.ID, s.Assunto.Codigo(), s.Time, s.Status, s.CriadaEm)
	return err
}

func (r *Repositorio) RegistrarEvento(ctx context.Context, e domain.Evento) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO eventos (tipo, solicitacao_id, time_nome, atendente_id, atendente_nome, posicao_fila, ocorreu)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.Tipo, e.SolicitacaoID, e.Time, nullStr(e.AtendenteID), nullStr(e.AtendenteNome), e.PosicaoFila, e.Ocorreu)
	return err
}

func (r *Repositorio) HistoricoEventos(ctx context.Context, limite int) ([]domain.Evento, error) {
	if limite <= 0 {
		limite = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT tipo, solicitacao_id, time_nome, COALESCE(atendente_id,''), COALESCE(atendente_nome,''),
		        COALESCE(posicao_fila,0), ocorreu
		 FROM eventos ORDER BY ocorreu DESC LIMIT $1`, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventos []domain.Evento
	for rows.Next() {
		var e domain.Evento
		if err := rows.Scan(&e.Tipo, &e.SolicitacaoID, &e.Time, &e.AtendenteID,
			&e.AtendenteNome, &e.PosicaoFila, &e.Ocorreu); err != nil {
			return nil, err
		}
		eventos = append(eventos, e)
	}
	return eventos, rows.Err()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
