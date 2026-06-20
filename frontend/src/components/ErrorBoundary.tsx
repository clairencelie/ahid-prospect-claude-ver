import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

// ErrorBoundary keeps one crashing page from white-screening the whole app.
// React unmounts the entire tree on an unhandled render error, so without a
// boundary a single bad component blanks every screen including the nav.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Page crashed:", error, info);
  }

  // Reset so navigating away from the broken page recovers.
  reset = () => this.setState({ error: null });

  componentDidUpdate(prevProps: Props) {
    if (prevProps.children !== this.props.children && this.state.error) {
      this.reset();
    }
  }

  render() {
    if (this.state.error) {
      return (
        <div className="card" style={{ borderColor: "#c0392b" }}>
          <h3 style={{ marginTop: 0, color: "#c0392b" }}>Something went wrong on this page</h3>
          <p style={{ fontSize: 13 }}>{this.state.error.message}</p>
          <button className="btn secondary" onClick={this.reset}>
            Dismiss
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
