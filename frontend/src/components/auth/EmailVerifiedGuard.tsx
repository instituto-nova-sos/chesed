import type { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';

interface EmailVerifiedGuardProps {
  children: ReactNode;
}

export function EmailVerifiedGuard({ children }: EmailVerifiedGuardProps) {
  const { isAuthenticated, emailVerified } = useAuth();

  if (isAuthenticated && !emailVerified) {
    return <Navigate to="/email-verification" replace />;
  }

  return <>{children}</>;
}
