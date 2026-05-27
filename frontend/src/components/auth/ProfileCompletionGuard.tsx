import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';

interface ProfileCompletionGuardProps {
  children: ReactNode;
}

export function ProfileCompletionGuard({
  children,
}: ProfileCompletionGuardProps) {
  const { personId, isAuthenticated, roles } = useAuth();
  const location = useLocation();

  // Admin users skip profile completion (created by admin, not self-registered)
  const isAdmin = roles.includes('ADMIN');

  if (isAuthenticated && !personId && !isAdmin) {
    if (location.pathname !== '/complete-profile') {
      return <Navigate to="/complete-profile" replace />;
    }
  }

  return <>{children}</>;
}
