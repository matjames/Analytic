import React from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
}

export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(): Partial<ErrorBoundaryState> {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
    this.setState({
      error,
      errorInfo,
    });
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback || (
          <div className="min-h-screen bg-gradient-to-br from-red-50 to-red-100 flex items-center justify-center px-4">
            <div className="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
              <div className="flex items-center justify-center w-12 h-12 bg-red-100 rounded-full mx-auto">
                <AlertCircle className="w-6 h-6 text-red-600" />
              </div>

              <h2 className="mt-4 text-center text-xl font-bold text-gray-900">
                Something went wrong
              </h2>

              <p className="mt-2 text-center text-gray-600 text-sm">
                An unexpected error occurred. Please try again or contact
                support.
              </p>

              {process.env.NODE_ENV === 'development' && this.state.error && (
                <div className="mt-4 p-4 bg-gray-100 rounded-lg">
                  <p className="text-xs font-mono text-gray-700 whitespace-pre-wrap break-words">
                    {this.state.error.toString()}
                  </p>
                </div>
              )}

              <div className="mt-6 space-y-2">
                <button
                  onClick={this.handleReset}
                  className="w-full flex items-center justify-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
                >
                  <RefreshCw className="w-4 h-4" />
                  <span>Try Again</span>
                </button>
                <button
                  onClick={() => (window.location.href = '/')}
                  className="w-full px-4 py-2 bg-gray-200 text-gray-900 rounded-lg hover:bg-gray-300 transition-colors font-medium"
                >
                  Go to Home
                </button>
              </div>
            </div>
          </div>
        )
      );
    }

    return this.props.children;
  }
}
