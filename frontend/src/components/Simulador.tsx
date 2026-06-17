import { useQueryClient } from "@tanstack/react-query";
import { criarSolicitacao } from "../api/cliente";

const ASSUNTOS = [
  { codigo: "problema_cartao", rotulo: "Problema com cartão" },
  { codigo: "contratacao_emprestimo", rotulo: "Contratar empréstimo" },
  { codigo: "atualizacao_cadastral", rotulo: "Outro assunto" },
];

export function Simulador() {
  const qc = useQueryClient();

  async function simular(codigo: string) {
    await criarSolicitacao(codigo);
    qc.invalidateQueries({ queryKey: ["dashboard"] });
  }

  return (
    <div className="shrink-0">
      <p className="mb-1.5 text-[11px] font-semibold uppercase tracking-wide text-slate-400">
        Simular chegada
      </p>
      <div className="flex flex-wrap gap-2">
        {ASSUNTOS.map((a) => (
          <button
            key={a.codigo}
            onClick={() => simular(a.codigo)}
            className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 active:scale-95 transition-transform"
          >
            {a.rotulo}
          </button>
        ))}
      </div>
    </div>
  );
}
