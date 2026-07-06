import { type ReactNode, useEffect } from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { useOnboardingStatus } from '../../hooks/useOnboardingStatus';
import { useAuthStore } from '../../store/authStore';

interface OnboardingGuardProps {
  children: ReactNode;
}

export function OnboardingGuard({ children }: OnboardingGuardProps) {
  const { roles } = useAuth();
  const { status, loading, error } = useOnboardingStatus();
  const setCampusId = useAuthStore((s) => s.setCampusId);
  const setCampusTimezone = useAuthStore((s) => s.setCampusTimezone);

  const isAdmin = roles.includes('ADMIN');

  // Set campus (and its timezone) in auth store when resolved from backend
  useEffect(() => {
    if (status?.campus_id) {
      setCampusId(status.campus_id);
    }
    if (status?.campus_timezone) {
      setCampusTimezone(status.campus_timezone);
    }
  }, [status?.campus_id, status?.campus_timezone, setCampusId, setCampusTimezone]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  // On error, allow through — backend middleware will enforce
  if (error || !status) {
    return <>{children}</>;
  }

  if (!isAdmin && (status.needs_profile_completion || status.needs_campus_assignment)) {
    return <Navigate to="/complete-profile" replace />;
  }

  if (status.needs_agreement) {
    return <Navigate to="/volunteer-agreement" replace />;
  }

  return <>{children}</>;
}
