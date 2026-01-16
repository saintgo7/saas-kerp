import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  Bell,
  Search,
  User,
  LogOut,
  Settings,
  Building2,
  Moon,
  Sun,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore, useUIStore } from "@/stores";
import { Button } from "@/components/ui";

export function Header() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const { theme, setTheme, sidebarCollapsed } = useUIStore();
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [showSearch, setShowSearch] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);

  // Close user menu on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        userMenuRef.current &&
        !userMenuRef.current.contains(event.target as Node)
      ) {
        setShowUserMenu(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleLogout = async () => {
    await logout();
    navigate("/login");
  };

  const toggleTheme = () => {
    setTheme(theme === "dark" ? "light" : "dark");
  };

  return (
    <header
      className={cn(
        "fixed top-0 right-0 z-30 h-16 bg-background border-b transition-all duration-300",
        sidebarCollapsed ? "left-16" : "left-64",
        "lg:left-16",
        !sidebarCollapsed && "lg:left-64"
      )}
    >
      <div className="flex items-center justify-between h-full px-4">
        {/* Search */}
        <div className="flex items-center flex-1">
          <div className="relative w-full max-w-md">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="검색..."
              className={cn(
                "w-full h-9 pl-10 pr-4 rounded-lg border bg-muted/50 text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all",
                showSearch ? "opacity-100" : "opacity-0 lg:opacity-100"
              )}
              onFocus={() => setShowSearch(true)}
              onBlur={() => setShowSearch(false)}
            />
          </div>
        </div>

        {/* Right side */}
        <div className="flex items-center space-x-2">
          {/* Theme toggle */}
          <Button variant="ghost" size="icon" onClick={toggleTheme}>
            {theme === "dark" ? (
              <Sun className="h-5 w-5" />
            ) : (
              <Moon className="h-5 w-5" />
            )}
          </Button>

          {/* Notifications */}
          <Button variant="ghost" size="icon" className="relative">
            <Bell className="h-5 w-5" />
            <span className="absolute top-1 right-1 w-2 h-2 bg-destructive rounded-full" />
          </Button>

          {/* User menu */}
          <div className="relative" ref={userMenuRef}>
            <button
              onClick={() => setShowUserMenu(!showUserMenu)}
              className="flex items-center space-x-2 p-2 rounded-lg hover:bg-muted transition-colors"
            >
              <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                <User className="h-4 w-4 text-primary" />
              </div>
              <div className="hidden md:block text-left">
                <p className="text-sm font-medium">{user?.name || "사용자"}</p>
                <p className="text-xs text-muted-foreground">
                  {user?.company?.name || "회사명"}
                </p>
              </div>
            </button>

            {/* Dropdown menu */}
            {showUserMenu && (
              <div className="absolute right-0 mt-2 w-56 bg-popover border rounded-lg shadow-lg py-1 z-50">
                <div className="px-4 py-2 border-b">
                  <p className="text-sm font-medium">{user?.name}</p>
                  <p className="text-xs text-muted-foreground">{user?.email}</p>
                </div>

                <div className="py-1">
                  <button
                    onClick={() => {
                      setShowUserMenu(false);
                      navigate("/settings/company");
                    }}
                    className="flex items-center w-full px-4 py-2 text-sm hover:bg-muted transition-colors"
                  >
                    <Building2 className="h-4 w-4 mr-3" />
                    회사정보
                  </button>
                  <button
                    onClick={() => {
                      setShowUserMenu(false);
                      navigate("/settings/profile");
                    }}
                    className="flex items-center w-full px-4 py-2 text-sm hover:bg-muted transition-colors"
                  >
                    <User className="h-4 w-4 mr-3" />
                    내 프로필
                  </button>
                  <button
                    onClick={() => {
                      setShowUserMenu(false);
                      navigate("/settings");
                    }}
                    className="flex items-center w-full px-4 py-2 text-sm hover:bg-muted transition-colors"
                  >
                    <Settings className="h-4 w-4 mr-3" />
                    설정
                  </button>
                </div>

                <div className="border-t py-1">
                  <button
                    onClick={handleLogout}
                    className="flex items-center w-full px-4 py-2 text-sm text-destructive hover:bg-muted transition-colors"
                  >
                    <LogOut className="h-4 w-4 mr-3" />
                    로그아웃
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
