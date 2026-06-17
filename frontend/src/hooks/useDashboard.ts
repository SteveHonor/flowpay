import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { buscarDashboard, streamEventos, type Evento } from "../api/cliente";

export function useDashboard() {
  return useQuery({
    queryKey: ["dashboard"],
    queryFn: buscarDashboard,
    refetchInterval: 15000,
  });
}

// useEventos abre uma única conexão SSE e mantém dois acumuladores independentes:
// - feed: janela rolling dos últimos N eventos (todos os tipos), para o painel ao vivo
// - finalizados: acumulador sem limite de atendimento_finalizado, nunca perde eventos
export function useEventos(limite = 12) {
  const qc = useQueryClient();
  const [feed, setFeed] = useState<Evento[]>([]);
  const [finalizados, setFinalizados] = useState<Evento[]>([]);
  const limiteRef = useRef(limite);

  useEffect(() => {
    const cancelar = streamEventos((e) => {
      setFeed((atual) => [e, ...atual].slice(0, limiteRef.current));
      if (e.tipo === "atendimento_finalizado") {
        setFinalizados((atual) => [e, ...atual]);
      }
      qc.invalidateQueries({ queryKey: ["dashboard"] });
    });
    return cancelar;
  }, [qc]);

  return { feed, finalizados };
}
