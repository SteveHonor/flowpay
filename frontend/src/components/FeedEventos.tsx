import type { Evento } from "../api/cliente";

const ROTULO_EVENTO: Record<Evento["tipo"], string> = {
  solicitacao_recebida: "Recebida",
  solicitacao_atribuida: "Atribuída",
  solicitacao_enfileirada: "Enfileirada",
  atendimento_finalizado: "Finalizada",
  atendente_pausado: "Pausado",
  atendente_retomado: "Retomado",
  atendente_removido: "Removido",
};

const COR: Record<Evento["tipo"], string> = {
  solicitacao_recebida: "bg-slate-100 text-slate-600",
  solicitacao_atribuida: "bg-indigo-50 text-indigo-700",
  solicitacao_enfileirada: "bg-amber-50 text-amber-700",
  atendimento_finalizado: "bg-emerald-50 text-emerald-700",
  atendente_pausado: "bg-slate-100 text-slate-500",
  atendente_retomado: "bg-emerald-50 text-emerald-600",
  atendente_removido: "bg-rose-50 text-rose-600",
};

function hora(iso: string) {
  return new Date(iso).toLocaleTimeString("pt-BR");
}

function descricao(e: Evento): string {
  if (e.atendente_nome && !e.solicitacao_id) return e.atendente_nome;
  if (e.atendente_nome && e.solicitacao_id) return `${e.atendente_nome} · ${e.solicitacao_id}`;
  return e.solicitacao_id ?? "";
}

export function FeedEventos({ eventos }: { eventos: Evento[] }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <h3 className="mb-4 text-lg font-medium text-slate-800">Eventos em tempo real</h3>
      {eventos.length === 0 ? (
        <p className="text-sm text-slate-400">Aguardando atividade…</p>
      ) : (
        <ul className="space-y-2">
          {eventos.map((e, i) => (
            <li
              key={`${e.solicitacao_id ?? e.atendente_id}-${e.tipo}-${i}`}
              className="flex items-center justify-between gap-3 text-sm"
            >
              <div className="flex min-w-0 items-center gap-2">
                <span className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium ${COR[e.tipo]}`}>
                  {ROTULO_EVENTO[e.tipo]}
                </span>
                <span className="truncate font-mono text-xs text-slate-400">
                  {descricao(e)}
                </span>
              </div>
              <span className="shrink-0 text-xs text-slate-400">{hora(e.ocorreu)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
