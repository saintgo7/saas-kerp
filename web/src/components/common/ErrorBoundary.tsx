import { Component, type ErrorInfo, type ReactNode } from "react";

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: (error: Error, reset: () => void) => ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

/**
 * Catches render-time exceptions so a single bad response cannot blank the
 * whole app. React unmounts the entire tree on an uncaught render error, which
 * previously left the user staring at a white page with no way back.
 */
export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Kept as console.error (not console.log) so it is a real error signal and
    // carries no credentials. Replace with a reporting sink when one exists.
    console.error("Unhandled render error:", error, info.componentStack);
  }

  reset = (): void => {
    this.setState({ error: null });
  };

  render(): ReactNode {
    const { error } = this.state;
    const { children, fallback } = this.props;

    if (!error) return children;

    if (fallback) return fallback(error, this.reset);

    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 p-6 text-center">
        <h1 className="text-xl font-bold">문제가 발생했습니다</h1>
        <p className="max-w-md text-sm text-muted-foreground">
          화면을 표시하는 중 오류가 발생했습니다. 다시 시도해도 같은 문제가
          계속되면 관리자에게 문의하십시오.
        </p>
        <p className="max-w-md break-words rounded bg-muted px-3 py-2 font-mono text-xs text-muted-foreground">
          {error.message}
        </p>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={this.reset}
            className="rounded-md border px-4 py-2 text-sm font-medium hover:bg-muted"
          >
            다시 시도
          </button>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90"
          >
            새로고침
          </button>
        </div>
      </div>
    );
  }
}

export default ErrorBoundary;
