import { Component, type ReactNode } from "react";

/**
 * If a page crashes while rendering (a bug, unexpected data), React would
 * otherwise show a blank screen. This catches the error and shows a message
 * instead. It's a class component because React only offers error
 * boundaries as classes.
 */
export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: unknown) {
    console.error("Page crashed:", error);
  }

  render() {
    if (!this.state.failed) return this.props.children;
    return (
      <div className="py-24 text-center">
        <h1 className="text-2xl font-bold">Something went wrong on this page</h1>
        <p className="mt-2 text-muted">Please reload. If it keeps happening, try another page.</p>
        <button
          onClick={() => window.location.reload()}
          className="mt-6 rounded-md bg-cta px-5 py-2 font-semibold text-page hover:bg-cta-hover"
        >
          Reload
        </button>
      </div>
    );
  }
}
