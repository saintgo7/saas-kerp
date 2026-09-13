import { Outlet, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useUIStore } from "@/stores";
import { ErrorBoundary } from "@/components/common";
import { Sidebar } from "./Sidebar";
import { Header } from "./Header";

export function MainLayout() {
  const { sidebarCollapsed } = useUIStore();
  const location = useLocation();

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <Header />
      <main
        className={cn(
          "pt-16 min-h-screen transition-all duration-300",
          sidebarCollapsed ? "lg:ml-16" : "lg:ml-64"
        )}
      >
        <div className="p-6">
          {/* Keyed by route so navigating away clears a crashed page's state,
              and so a page-level crash keeps the shell (sidebar/header) alive. */}
          <ErrorBoundary key={location.pathname}>
            <Outlet />
          </ErrorBoundary>
        </div>
      </main>
    </div>
  );
}
