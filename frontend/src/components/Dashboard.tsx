import { useDashboard, useEventos } from "../hooks/useDashboard";
import { CargaChart } from "./CargaChart";
import { FeedEventos } from "./FeedEventos";
import { FinalizadosCard } from "./FinalizadosCard";
import { Simulador } from "./Simulador";
import { TimeCard } from "./TimeCard";

function Disclaimer() {
  return (
    <div className="mb-6 rounded-xl border border-indigo-100 bg-indigo-50 px-4 py-3 text-sm text-indigo-900">
      <span className="font-semibold">Demo interativo</span>
      {" — "}
      simule chegadas de clientes e veja a distribuição automática por time, o limite de
      3 atendimentos por atendente e a fila FIFO em ação. Nos cards abaixo você pode
      <span className="font-medium"> adicionar · pausar · retomar · remover</span> atendentes
      e <span className="font-medium">finalizar</span> atendimentos em tempo real.
    </div>
  );
}

export function Dashboard() {
  const { data, isLoading, isError } = useDashboard();
  const { feed, finalizados } = useEventos();

  return (
    <div className="mx-auto max-w-6xl px-4 py-6 sm:px-6 sm:py-8">
      {/* Cabeçalho */}
      <header className="mb-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-slate-900 sm:text-2xl">
              FlowPay — Central de Atendimentos
            </h1>
            <p className="mt-0.5 text-sm text-slate-500">
              Distribuição e monitoramento de atendimentos em tempo real
            </p>
          </div>
          <Simulador />
        </div>
      </header>

      <Disclaimer />

      {/* Estado de carregamento e erro */}
      {isLoading && !data && (
        <p className="text-sm text-slate-400">Carregando…</p>
      )}
      {/* Só mostra erro quando não há dados — falhas em background refetch não interrompem o dashboard */}
      {isError && !data && (
        <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
          Falha ao carregar o dashboard. Verifique se a API está acessível e recarregue a página.
        </div>
      )}

      {data && (
        <div className="space-y-6">
          {/* Times */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {(data.times ?? []).map((t) => (
              <TimeCard key={t.nome} time={t} />
            ))}
          </div>

          {/* Gráfico + Feed */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <CargaChart dashboard={data} />
            <FeedEventos eventos={feed} />
          </div>

          {/* Atendimentos finalizados */}
          <FinalizadosCard finalizados={finalizados} />
        </div>
      )}
    </div>
  );
}
