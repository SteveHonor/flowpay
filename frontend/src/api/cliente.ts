export interface SolicitacaoAtivaView {
  id: string;
  tempo_na_fila_seg?: number;
}

export interface AtendenteView {
  id: string;
  nome: string;
  ativos: number;
  capacidade: number;
  pausado: boolean;
  solicitacoes_ativas: SolicitacaoAtivaView[];
}

export interface TimeView {
  nome: "CARTOES" | "EMPRESTIMOS" | "OUTROS";
  atendentes: AtendenteView[];
  tamanho_fila: number;
  em_atendimento: number;
}

export interface DashboardView {
  times: TimeView[];
  atualizado_em: string;
}

export type TipoEvento =
  | "solicitacao_recebida"
  | "solicitacao_atribuida"
  | "solicitacao_enfileirada"
  | "atendimento_finalizado"
  | "atendente_pausado"
  | "atendente_retomado"
  | "atendente_removido";

export interface Evento {
  tipo: TipoEvento;
  solicitacao_id?: string;
  time: string;
  atendente_id?: string;
  atendente_nome?: string;
  posicao_fila?: number;
  ocorreu: string;
  duracao_atendimento_seg?: number;
  tempo_na_fila_seg?: number;
}

const BASE = "/api";

export async function buscarDashboard(): Promise<DashboardView> {
  const r = await fetch(`${BASE}/dashboard`);
  if (!r.ok) throw new Error("falha ao buscar dashboard");
  return r.json();
}

export async function criarSolicitacao(assunto: string): Promise<{ id: string }> {
  const r = await fetch(`${BASE}/solicitacoes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ assunto }),
  });
  if (!r.ok) throw new Error("falha ao criar solicitação");
  return r.json();
}

export async function adicionarAtendente(nome: string, time: string): Promise<{ id: string }> {
  const r = await fetch(`${BASE}/atendentes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ nome, time }),
  });
  if (!r.ok) throw new Error("falha ao adicionar atendente");
  return r.json();
}

export async function pausarAtendente(id: string): Promise<void> {
  const r = await fetch(`${BASE}/atendentes/${id}/pausar`, { method: "POST" });
  if (!r.ok && r.status !== 204) throw new Error("falha ao pausar atendente");
}

export async function retomarAtendente(id: string): Promise<void> {
  const r = await fetch(`${BASE}/atendentes/${id}/retomar`, { method: "POST" });
  if (!r.ok && r.status !== 204) throw new Error("falha ao retomar atendente");
}

export async function removerAtendente(id: string): Promise<void> {
  const r = await fetch(`${BASE}/atendentes/${id}`, { method: "DELETE" });
  if (!r.ok && r.status !== 204) throw new Error("falha ao remover atendente");
}

export async function finalizarAtendimento(id: string): Promise<void> {
  const r = await fetch(`${BASE}/solicitacoes/${id}/finalizar`, { method: "POST" });
  if (!r.ok && r.status !== 204) throw new Error("falha ao finalizar");
}

export function streamEventos(onEvento: (e: Evento) => void): () => void {
  const es = new EventSource(`${BASE}/eventos`);
  const tipos: TipoEvento[] = [
    "solicitacao_recebida",
    "solicitacao_atribuida",
    "solicitacao_enfileirada",
    "atendimento_finalizado",
    "atendente_pausado",
    "atendente_retomado",
    "atendente_removido",
  ];
  tipos.forEach((tipo) =>
    es.addEventListener(tipo, (ev) => onEvento(JSON.parse((ev as MessageEvent).data)))
  );
  return () => es.close();
}
