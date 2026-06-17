import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Dashboard } from "./components/Dashboard";

const qc = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 5000,
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <div className="min-h-screen bg-gradient-to-b from-slate-50 to-slate-100">
        <Dashboard />
      </div>
    </QueryClientProvider>
  );
}
