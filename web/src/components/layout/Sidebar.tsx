import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Calculator,
  FileText,
  Users,
  Package,
  BarChart3,
  Settings,
  ChevronDown,
  ChevronRight,
  Menu,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useUIStore, useAuthStore } from "@/stores";
import { MAIN_MENU } from "@/constants";
import { Logo, LogoIcon } from "@/components/common";
import type { MenuItem, UserRole } from "@/types";

// Icon mapping
const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  LayoutDashboard,
  Calculator,
  FileText,
  Users,
  Package,
  BarChart3,
  Settings,
};

interface SidebarItemProps {
  item: MenuItem;
  collapsed: boolean;
  userRole: UserRole;
}

function SidebarItem({ item, collapsed, userRole }: SidebarItemProps) {
  const location = useLocation();
  const [isOpen, setIsOpen] = useState(false);
  const Icon = item.icon ? iconMap[item.icon] : null;

  // Check if user has access to this menu item
  if (item.roles && !item.roles.includes(userRole)) {
    return null;
  }

  const hasChildren = item.children && item.children.length > 0;
  const isActive = item.path
    ? location.pathname === item.path
    : item.children?.some((child) => location.pathname === child.path);

  // Filter children based on role
  const visibleChildren = item.children?.filter(
    (child) => !child.roles || child.roles.includes(userRole)
  );

  if (hasChildren && collapsed) {
    return (
      <div className="relative group">
        <button
          className={cn(
            "flex items-center justify-center w-full p-3 rounded-lg transition-colors",
            isActive
              ? "bg-primary/10 text-primary"
              : "text-muted-foreground hover:bg-muted hover:text-foreground"
          )}
        >
          {Icon && <Icon className="h-5 w-5" />}
        </button>
        <div className="absolute left-full top-0 ml-2 hidden group-hover:block z-50">
          <div className="bg-popover border rounded-lg shadow-lg py-2 min-w-[180px]">
            <div className="px-3 py-1 font-medium text-sm border-b mb-1">
              {item.label}
            </div>
            {visibleChildren?.map((child) => (
              <Link
                key={child.id}
                to={child.path || "#"}
                className={cn(
                  "block px-3 py-2 text-sm transition-colors",
                  location.pathname === child.path
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                {child.label}
              </Link>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (hasChildren) {
    return (
      <div>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className={cn(
            "flex items-center w-full p-3 rounded-lg transition-colors",
            isActive
              ? "bg-primary/10 text-primary"
              : "text-muted-foreground hover:bg-muted hover:text-foreground"
          )}
        >
          {Icon && <Icon className="h-5 w-5 mr-3" />}
          <span className="flex-1 text-left">{item.label}</span>
          {isOpen ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </button>
        {isOpen && (
          <div className="ml-8 mt-1 space-y-1">
            {visibleChildren?.map((child) => (
              <Link
                key={child.id}
                to={child.path || "#"}
                className={cn(
                  "flex items-center p-2 rounded-lg text-sm transition-colors",
                  location.pathname === child.path
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                {child.label}
                {child.badge && (
                  <span className="ml-auto bg-primary text-primary-foreground text-xs px-2 py-0.5 rounded-full">
                    {child.badge}
                  </span>
                )}
              </Link>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <Link
      to={item.path || "#"}
      className={cn(
        "flex items-center p-3 rounded-lg transition-colors",
        isActive
          ? "bg-primary/10 text-primary"
          : "text-muted-foreground hover:bg-muted hover:text-foreground",
        collapsed && "justify-center"
      )}
    >
      {Icon && <Icon className={cn("h-5 w-5", !collapsed && "mr-3")} />}
      {!collapsed && <span>{item.label}</span>}
    </Link>
  );
}

export function Sidebar() {
  const { sidebarCollapsed, sidebarMobileOpen, toggleSidebar, setSidebarMobileOpen } =
    useUIStore();
  const { user } = useAuthStore();
  const userRole = user?.role || "user";

  return (
    <>
      {/* Mobile overlay */}
      {sidebarMobileOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={() => setSidebarMobileOpen(false)}
        />
      )}

      {/* Mobile toggle button */}
      <button
        onClick={() => setSidebarMobileOpen(true)}
        className="fixed top-4 left-4 z-50 lg:hidden p-2 rounded-lg bg-background border shadow-sm"
      >
        <Menu className="h-5 w-5" />
      </button>

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed top-0 left-0 z-50 h-screen bg-background border-r transition-all duration-300",
          sidebarCollapsed ? "w-16" : "w-64",
          sidebarMobileOpen ? "translate-x-0" : "-translate-x-full",
          "lg:translate-x-0"
        )}
      >
        {/* Logo */}
        <div className="flex items-center justify-between h-16 px-4 border-b">
          {!sidebarCollapsed ? (
            <Link to="/dashboard" className="flex items-center">
              <Logo size="md" />
            </Link>
          ) : (
            <Link to="/dashboard" className="flex items-center justify-center w-full">
              <LogoIcon size="md" />
            </Link>
          )}
          <button
            onClick={() => {
              if (window.innerWidth < 1024) {
                setSidebarMobileOpen(false);
              } else {
                toggleSidebar();
              }
            }}
            className="p-2 rounded-lg hover:bg-muted transition-colors"
          >
            {sidebarMobileOpen ? (
              <X className="h-5 w-5 lg:hidden" />
            ) : (
              <Menu className="h-5 w-5" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="p-3 space-y-1 overflow-y-auto h-[calc(100vh-4rem)]">
          {MAIN_MENU.map((item) => (
            <SidebarItem
              key={item.id}
              item={item}
              collapsed={sidebarCollapsed}
              userRole={userRole}
            />
          ))}
        </nav>
      </aside>
    </>
  );
}
