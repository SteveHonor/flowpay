import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TimeCard } from "./TimeCard";
import type { TimeView } from "../api/cliente";

function renderWithQuery(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

const time: TimeView = {
  nome: "CARTOES",
  tamanho_fila: 2,
  em_atendimento: 4,
  atendentes: [
    {
      id: "c1", nome: "Ana", ativos: 3, capacidade: 3, pausado: false,
      solicitacoes_ativas: [
        { id: "s1" },
        { id: "s2", tempo_na_fila_seg: 45 },
        { id: "s3" },
      ],
    },
    { id: "c2", nome: "Bruno", ativos: 1, capacidade: 3, pausado: true, solicitacoes_ativas: [{ id: "s4" }] },
  ],
};

describe("TimeCard", () => {
  it("mostra o rótulo do time, a fila e a carga dos atendentes", () => {
    renderWithQuery(<TimeCard time={time} />);
    expect(screen.getByText("Cartões")).toBeInTheDocument();
    expect(screen.getByText("fila: 2")).toBeInTheDocument();
    expect(screen.getByText("4 em atendimento")).toBeInTheDocument();
    expect(screen.getByText("3/3")).toBeInTheDocument();
  });

  it("exibe badge de fila apenas para solicitações que vieram da fila", () => {
    renderWithQuery(<TimeCard time={time} />);
    // s2 tem tempo_na_fila_seg=45 → deve mostrar "aguardou 45s na fila"
    expect(screen.getByText("aguardou 45s na fila")).toBeInTheDocument();
    // s1 e s3 não têm tempo_na_fila_seg → não devem aparecer badges extras
    expect(screen.getAllByText(/aguardou .+ na fila/).length).toBe(1);
  });

  it("exibe badge 'pausado' apenas para atendentes pausados", () => {
    renderWithQuery(<TimeCard time={time} />);
    // Bruno está pausado → badge "pausado" aparece
    expect(screen.getByText("pausado")).toBeInTheDocument();
    // Ana não está pausada → só um badge de "pausado" no total
    expect(screen.getAllByText("pausado").length).toBe(1);
  });

  it("exibe botão 'Retomar' para atendente pausado e 'Pausar' para ativo", () => {
    renderWithQuery(<TimeCard time={time} />);
    expect(screen.getByText("Retomar")).toBeInTheDocument();
    expect(screen.getByText("Pausar")).toBeInTheDocument();
  });

  it("botão 'Pausar' fica desabilitado quando o atendente tem atendimentos ativos", () => {
    renderWithQuery(<TimeCard time={time} />);
    // Ana tem ativos=3 → Pausar deve estar desabilitado (invariante: só pausa com 0 ativos)
    const pausar = screen.getByText("Pausar").closest("button");
    expect(pausar).toBeDisabled();
  });

  it("botão 'Retomar' fica habilitado independentemente dos ativos do atendente pausado", () => {
    renderWithQuery(<TimeCard time={time} />);
    // Bruno está pausado com 1 ativo → Retomar deve estar habilitado
    const retomar = screen.getByText("Retomar").closest("button");
    expect(retomar).not.toBeDisabled();
  });
});
