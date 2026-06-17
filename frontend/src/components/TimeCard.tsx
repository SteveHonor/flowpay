import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  adicionarAtendente,
  finalizarAtendimento,
  pausarAtendente,
  retomarAtendente,
  removerAtendente,
  type TimeView,
} from "../api/cliente";

const ROTULOS: Record<TimeView["nome"], string> = {
  CARTOES: "Cartões",
  EMPRESTIMOS: "Empréstimos",
  OUTROS: "Outros Assuntos",
};

function idCurto(id: string) {
  return id.replace(/^sol_/, "").slice(0, 8);
}

function formatarDuracao(seg: number): string {
  if (seg < 60) return `${seg}s`;
  const min = Math.floor(seg / 60);
  const s = seg % 60;
  return s > 0 ? `${min}m ${s}s` : `${min}m`;
}

export function TimeCard({ time }: { time: TimeView }) {
  const qc = useQueryClient();
  const [nomeNovo, setNomeNovo] = useState("");
  const [adicionando, setAdicionando] = useState(false);
  const [carregando, setCarregando] = useState(false);
  const [finalizando, setFinalizando] = useState<string | null>(null);
  const [pausando, setPausando] = useState<string | null>(null);
  const [removendo, setRemoving] = useState<string | null>(null);

  async function handleFinalizar(solicitacaoId: string) {
    setFinalizando(solicitacaoId);
    try {
      await finalizarAtendimento(solicitacaoId);
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    } finally {
      setFinalizando(null);
    }
  }

  async function handlePausar(atendenteId: string) {
    setPausando(atendenteId);
    try {
      await pausarAtendente(atendenteId);
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    } finally {
      setPausando(null);
    }
  }

  async function handleRetomar(atendenteId: string) {
    setPausando(atendenteId);
    try {
      await retomarAtendente(atendenteId);
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    } finally {
      setPausando(null);
    }
  }

  async function handleRemover(atendenteId: string) {
    setRemoving(atendenteId);
    try {
      await removerAtendente(atendenteId);
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    } finally {
      setRemoving(null);
    }
  }

  async function handleAdicionar(e: React.FormEvent) {
    e.preventDefault();
    const nome = nomeNovo.trim();
    if (!nome) return;
    setCarregando(true);
    try {
      await adicionarAtendente(nome, time.nome);
      setNomeNovo("");
      setAdicionando(false);
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    } finally {
      setCarregando(false);
    }
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      {/* Cabeçalho do time */}
      <div className="mb-4">
        <h3 className="text-base font-semibold text-slate-800">{ROTULOS[time.nome]}</h3>
        <div className="mt-1.5 flex items-center gap-2 text-sm">
          <span className="rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700 whitespace-nowrap">
            {time.em_atendimento} em atendimento
          </span>
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap ${
              time.tamanho_fila > 0
                ? "bg-amber-50 text-amber-700"
                : "bg-slate-100 text-slate-500"
            }`}
          >
            fila: {time.tamanho_fila}
          </span>
        </div>
      </div>

      {/* Atendentes */}
      <ul className="space-y-4">
        {time.atendentes.map((a) => (
          <li
            key={a.id}
            className={`transition-opacity ${a.pausado ? "opacity-60" : ""}`}
          >
            {/* Nome + estado + ações */}
            <div className="mb-1 flex items-center justify-between gap-2">
              <div className="flex min-w-0 items-center gap-1.5">
                <span className={`truncate text-sm font-medium ${a.pausado ? "text-slate-400" : "text-slate-700"}`}>
                  {a.nome}
                </span>
                {a.pausado && (
                  <span className="shrink-0 rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium text-slate-500">
                    pausado
                  </span>
                )}
              </div>
              <div className="flex shrink-0 items-center gap-1.5">
                <span className="tabular-nums text-sm text-slate-500">
                  {a.ativos}/{a.capacidade}
                </span>
                {/* Pausar / Retomar */}
                <button
                  onClick={() => a.pausado ? handleRetomar(a.id) : handlePausar(a.id)}
                  disabled={pausando === a.id || (!a.pausado && a.ativos > 0)}
                  title={!a.pausado && a.ativos > 0 ? "Finalize os atendimentos antes de pausar" : undefined}
                  className={`rounded px-2 py-0.5 text-[11px] font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-30 ${
                    a.pausado
                      ? "bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                      : "bg-slate-100 text-slate-500 hover:bg-slate-200"
                  }`}
                >
                  {pausando === a.id ? "…" : a.pausado ? "Retomar" : "Pausar"}
                </button>
                {/* Remover */}
                <button
                  onClick={() => handleRemover(a.id)}
                  disabled={a.ativos > 0 || removendo === a.id}
                  title={a.ativos > 0 ? "Finalize os atendimentos ativos antes de remover" : "Remover atendente"}
                  className="rounded px-1.5 py-0.5 text-[11px] text-rose-400 hover:bg-rose-50 hover:text-rose-600 disabled:cursor-not-allowed disabled:opacity-25 transition-colors"
                >
                  {removendo === a.id ? "…" : "✕"}
                </button>
              </div>
            </div>

            {/* Barra de carga */}
            <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100">
              <div
                className={`h-full rounded-full transition-all ${
                  a.pausado ? "bg-slate-300" : a.ativos >= a.capacidade ? "bg-rose-500" : "bg-indigo-500"
                }`}
                style={{ width: `${(a.ativos / a.capacidade) * 100}%` }}
              />
            </div>

            {/* Solicitações ativas */}
            {a.solicitacoes_ativas.length > 0 && (
              <ul className="mt-2 space-y-1">
                {a.solicitacoes_ativas.map((s) => (
                  <li
                    key={s.id}
                    className="flex items-center justify-between gap-2 rounded-md bg-slate-50 px-2 py-1"
                  >
                    <div className="flex min-w-0 items-center gap-1.5">
                      <span className="font-mono text-xs text-slate-400">{idCurto(s.id)}…</span>
                      {s.tempo_na_fila_seg != null && (
                        <span className="shrink-0 rounded-full bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-700">
                          aguardou {formatarDuracao(s.tempo_na_fila_seg)} na fila
                        </span>
                      )}
                    </div>
                    <button
                      onClick={() => handleFinalizar(s.id)}
                      disabled={finalizando === s.id}
                      className="shrink-0 rounded bg-rose-50 px-2 py-0.5 text-xs font-medium text-rose-600 hover:bg-rose-100 disabled:opacity-50"
                    >
                      {finalizando === s.id ? "…" : "Finalizar"}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </li>
        ))}
      </ul>

      {/* Adicionar atendente */}
      <div className="mt-4 border-t border-slate-100 pt-3">
        {adicionando ? (
          <form onSubmit={handleAdicionar} className="flex gap-2">
            <input
              autoFocus
              type="text"
              placeholder="Nome do atendente"
              value={nomeNovo}
              onChange={(e) => setNomeNovo(e.target.value)}
              className="min-w-0 flex-1 rounded border border-slate-200 px-2 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-indigo-400"
            />
            <button
              type="submit"
              disabled={carregando || !nomeNovo.trim()}
              className="rounded bg-indigo-600 px-3 py-1 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {carregando ? "…" : "Adicionar"}
            </button>
            <button
              type="button"
              onClick={() => { setAdicionando(false); setNomeNovo(""); }}
              className="rounded border border-slate-200 px-2 py-1 text-xs text-slate-500 hover:bg-slate-50"
            >
              ✕
            </button>
          </form>
        ) : (
          <button
            onClick={() => setAdicionando(true)}
            className="w-full rounded-lg border border-dashed border-slate-300 py-1.5 text-xs font-medium text-slate-400 hover:border-indigo-400 hover:text-indigo-600 transition-colors"
          >
            + Adicionar atendente
          </button>
        )}
      </div>
    </div>
  );
}
