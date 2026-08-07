import { AlertTriangle } from "lucide-react";

export function ErrorState({ message, onRetry }: { message?: string; onRetry?: () => void }) {
  return (
    <div className="flex-1 flex items-center justify-center p-grid-margin">
      <div className="w-full max-w-md text-center flex flex-col items-center">
        <div className="mb-6 w-16 h-16 bg-error-container/20 border border-error-container/40 flex items-center justify-center rounded-xl text-error">
          <AlertTriangle size={32} />
        </div>
        <h2 className="font-headline-md text-headline-md text-on-background mb-2">Something went wrong</h2>
        <p className="font-body-base text-body-base text-on-surface-variant mb-6">
          {message ?? "Couldn't reach the server. Check your connection and try again."}
        </p>
        {onRetry && (
          <button
            onClick={onRetry}
            className="px-6 py-2.5 bg-primary-container text-primary font-bold border border-primary rounded hover:bg-primary hover:text-on-primary transition-all"
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}
