import { type ReactNode, useState, useEffect } from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { getPersonAgreement } from '../../api/persons';

interface AgreementGuardProps {
  children: ReactNode;
}

/**
 * Blocks access for volunteers who have not accepted the volunteer agreement.
 * Redirects to /volunteer-agreement page.
 * ADMIN users and non-volunteers bypass this guard.
 */
export function AgreementGuard({ children }: AgreementGuardProps) {
  const { personId, roles } = useAuth();
  const [checking, setChecking] = useState(true);
  const [needsAgreement, setNeedsAgreement] = useState(false);

  const isAdmin = roles.includes('ADMIN');
  const isVolunteer = roles.includes('VOLUNTEER');

  useEffect(() => {
    // Skip check for admins and non-volunteers
    if (isAdmin || !isVolunteer || !personId) {
      setChecking(false);
      return;
    }

    getPersonAgreement(personId)
      .then(({ agreements }) => {
        const hasAccepted = agreements.some((a) => a.status === 'ACCEPTED');
        setNeedsAgreement(!hasAccepted);
      })
      .catch(() => {
        // On error, don't block — let the backend middleware handle it
        setNeedsAgreement(false);
      })
      .finally(() => setChecking(false));
  }, [personId, isAdmin, isVolunteer]);

  if (checking) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (needsAgreement) {
    return <Navigate to="/volunteer-agreement" replace />;
  }

  return <>{children}</>;
}
