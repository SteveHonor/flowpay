import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FinalizadosCard } from "./FinalizadosCard";
import type { Evento } from "../api/cliente";

function evento(overrides: Partial<Evento> = {}): Evento {
  return {
    tipo: "atendimento_finalizado",
    solicitacao_id: "sol_aabbccdd1122",
    time: "CARTOES",
    atendente_id: "ate_01",
    atendente_nome: "Ana",
    ocorreu: "2024-01-15T09:25:21Z",
    ...overrides,
  };
}

describe("FinalizadosCard — estado vazio", () => {
  it("exibe mensagem quando não há atendimentos", () => {
    render(<FinalizadosCard finalizados={[]} />);
    expect(screen.getByText(/nenhum atendimento finalizado/i)).toBeInTheDocument();
  });

  it("exibe contador zero", () => {
    render(<FinalizadosCard finalizados={[]} />);
    expect(screen.getByText("0")).toBeInTheDocument();
  });
});

describe("FinalizadosCard — duração e fila", () => {
  it("exibe 'em atendimento por Xs' quando há duração", () => {
    render(<FinalizadosCard finalizados={[evento({ duracao_atendimento_seg: 42 })]} />);
    expect(screen.getByText(/em atendimento por/)).toBeInTheDocument();
    expect(screen.getByText(/42s/)).toBeInTheDocument();
  });

  it("exibe 'ficou Xs na fila' apenas quando há tempo de fila", () => {
    render(
      <FinalizadosCard
        finalizados={[evento({ duracao_atendimento_seg: 10, tempo_na_fila_seg: 90 })]}
      />
    );
    expect(screen.getByText(/ficou/)).toBeInTheDocument();
    expect(screen.getByText(/1m 30s/)).toBeInTheDocument();
    expect(screen.getByText(/na fila/)).toBeInTheDocument();
  });

  it("omite linha de fila quando solicitação não passou pela fila", () => {
    render(<FinalizadosCard finalizados={[evento({ duracao_atendimento_seg: 5 })]} />);
    expect(screen.queryByText(/na fila/)).toBeNull();
  });
});

describe("FinalizadosCard — agrupamento por atendente", () => {
  it("agrupa eventos do mesmo atendente sob uma seção", () => {
    const finalizados = [
      evento({ solicitacao_id: "sol_aabbccdd1111", atendente_nome: "Ana", duracao_atendimento_seg: 4 }),
      evento({ solicitacao_id: "sol_aabbccdd2222", atendente_nome: "Ana", duracao_atendimento_seg: 7 }),
      evento({ solicitacao_id: "sol_aabbccdd3333", atendente_nome: "Bruno", duracao_atendimento_seg: 12 }),
    ];
    render(<FinalizadosCard finalizados={finalizados} />);
    expect(screen.getByText("Ana")).toBeInTheDocument();
    expect(screen.getByText("Bruno")).toBeInTheDocument();
    // contador do grupo Ana = 2
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("exibe o ID curto (8 chars após 'sol_') de cada solicitação", () => {
    render(<FinalizadosCard finalizados={[evento({ solicitacao_id: "sol_aabbccdd1122" })]} />);
    expect(screen.getByText(/aabbccdd…/)).toBeInTheDocument();
  });
});
