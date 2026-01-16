import { Outlet } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useUIStore } from "@/stores";
import { Sidebar } from "./Sidebar";
import { Header } from "./Header";

export function MainLayout() {
  const { sidebarCollapsed } = useUIStore();

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
          <Outlet />
        </div>
      </main>
    </div>
  );
}
