import type { ReactNode } from 'react';

interface AlertProps {
  variant: 'warning' | 'error' | 'info' | 'success';
  children: ReactNode;
  onClose?: () => void;
}

const variantClasses = {
  warning: 'bg-yellow-50 border-yellow-200 text-yellow-800',
  error: 'bg-red-50 border-red-200 text-red-800',
  info: 'bg-blue-50 border-blue-200 text-blue-800',
  success: 'bg-green-50 border-green-200 text-green-800',
};

export function Alert({ variant, children, onClose }: AlertProps) {
  return (
    <div
      className={`rounded-lg border p-4 text-sm ${variantClasses[variant]}`}
      role="alert"
    >
      <div className="flex items-start justify-between">
        <div>{children}</div>
        {onClose && (
          <button
            type="button"
            onClick={onClose}
            className="ml-4 inline-flex shrink-0 rounded-md p-1 hover:opacity-70"
          >
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>
    </div>
  );
}
