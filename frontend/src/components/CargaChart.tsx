import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { DashboardView } from "../api/cliente";

const ROTULOS: Record<string, string> = {
  CARTOES: "Cartões",
  EMPRESTIMOS: "Empréstimos",
  OUTROS: "Outros",
};

export function CargaChart({ dashboard }: { dashboard: DashboardView }) {
  const dados = dashboard.times.map((t) => ({
    time: ROTULOS[t.nome] ?? t.nome,
    "Em atendimento": t.em_atendimento,
    "Na fila": t.tamanho_fila,
  }));

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <h3 className="mb-4 text-lg font-medium text-slate-800">Carga por time</h3>
      <ResponsiveContainer width="100%" height={240}>
        <BarChart data={dados}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
          <XAxis dataKey="time" tick={{ fontSize: 13 }} />
          <YAxis allowDecimals={false} tick={{ fontSize: 13 }} />
          <Tooltip />
          <Legend />
          <Bar dataKey="Em atendimento" fill="#6366f1" radius={[4, 4, 0, 0]} />
          <Bar dataKey="Na fila" fill="#f59e0b" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
