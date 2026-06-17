import type { Evento } from "../api/cliente";

function hora(iso: string) {
  return new Date(iso).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
}

function formatarDuracao(seg: number): string {
  if (seg < 60) return `${seg}s`;
  const min = Math.floor(seg / 60);
  const s = seg % 60;
  return s > 0 ? `${min}m ${s}s` : `${min}m`;
}

function idCurto(id: string) {
  return id.replace(/^sol_/, "").slice(0, 8);
}

function agruparPorAtendente(finalizados: Evento[]): Map<string, Evento[]> {
  const grupos = new Map<string, Evento[]>();
  for (const e of finalizados) {
    const chave = e.atendente_nome ?? "Sem atendente";
    const lista = grupos.get(chave) ?? [];
    lista.push(e);
    grupos.set(chave, lista);
  }
  return grupos;
}

export function FinalizadosCard({ finalizados }: { finalizados: Evento[] }) {
  const grupos = agruparPorAtendente(finalizados);

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-base font-semibold text-slate-800">Atendimentos finalizados</h3>
        <span className="rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700">
          {finalizados.length}
        </span>
      </div>

      {finalizados.length === 0 ? (
        <p className="text-sm text-slate-400">Nenhum atendimento finalizado ainda.</p>
      ) : (
        <div className="space-y-5">
          {[...grupos.entries()].map(([atendente, eventos]) => (
            <div key={atendente}>
              {/* Cabeçalho do atendente */}
              <div className="mb-2 flex items-center gap-2">
                <span className="text-sm font-medium text-slate-700">{atendente}</span>
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">
                  {eventos.length}
                </span>
              </div>

              <ul className="space-y-2 border-l-2 border-slate-100 pl-3">
                {eventos.map((e, i) => (
                  <li
                    key={`${e.solicitacao_id}-${i}`}
                    className="rounded-lg bg-slate-50 p-2.5"
                  >
                    {/* ID + hora */}
                    <div className="mb-1.5 flex items-center justify-between gap-2">
                      <span className="font-mono text-[11px] text-slate-400">
                        {idCurto(e.solicitacao_id!)}…
                      </span>
                      <span className="shrink-0 text-[11px] text-slate-400">{hora(e.ocorreu)}</span>
                    </div>

                    {/* Métricas com texto legível */}
                    <div className="flex flex-col gap-1">
                      {e.duracao_atendimento_seg != null && (
                        <div className="flex items-center gap-1.5 text-xs text-indigo-600">
                          <svg className="h-3 w-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
                          </svg>
                          <span>em atendimento por <strong>{formatarDuracao(e.duracao_atendimento_seg)}</strong></span>
                        </div>
                      )}
                      {e.tempo_na_fila_seg != null && (
                        <div className="flex items-center gap-1.5 text-xs text-amber-600">
                          <svg className="h-3 w-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
                          </svg>
                          <span>ficou <strong>{formatarDuracao(e.tempo_na_fila_seg)}</strong> na fila</span>
                        </div>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
