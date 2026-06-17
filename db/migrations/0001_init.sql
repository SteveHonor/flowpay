-- +goose Up
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

CREATE INDEX IF NOT EXISTS idx_eventos_ocorreu ON eventos (ocorreu DESC);

-- +goose Down
DROP TABLE IF EXISTS eventos;
DROP TABLE IF EXISTS solicitacoes;
