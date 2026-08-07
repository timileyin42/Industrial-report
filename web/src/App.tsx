import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { router } from "./routes";

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        {/* Fixed behind every page — see index.css's .app-backdrop comment
            for why this is a CSS gradient standing in for a real photo. */}
        <div className="app-backdrop" aria-hidden="true" />
        <RouterProvider router={router} />
      </AuthProvider>
    </QueryClientProvider>
  );
}
