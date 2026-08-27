import { Component, type ErrorInfo, type ReactNode } from "react";
import { Config } from "@/base/config";
import { debugError } from "./Logger";

interface ErrorHandlerProps {
  children?: ReactNode;
  /** Rendered in place of the subtree once an error has been caught. */
  onErrorComponent?: ReactNode;
  /** Identifies the boundary in logs. */
  componentName: string;
}

interface ErrorHandlerState {
  hasError: boolean;
}

/**
 * Error boundary. Keeps one broken subtree from taking down the whole page and
 * reports the failure through the app logger.
 */
export default class ErrorHandler extends Component<
  ErrorHandlerProps,
  ErrorHandlerState
> {
  state: ErrorHandlerState = { hasError: false };

  static getDerivedStateFromError(): ErrorHandlerState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    if (Config.logErrors) {
      debugError(`[${this.props.componentName}]`, error, errorInfo);
    }
  }

  render(): ReactNode {
    const { hasError } = this.state;
    const { children, onErrorComponent } = this.props;

    if (!hasError) {
      return children;
    }

    return (
      onErrorComponent ?? (
        <div className="p-6 text-muted-foreground text-sm">
          Something went wrong while rendering this section.
        </div>
      )
    );
  }
}
