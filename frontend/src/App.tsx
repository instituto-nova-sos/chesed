import { useEffect, lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { EmailVerifiedGuard } from './components/auth/EmailVerifiedGuard';
import { OnboardingGuard } from './components/auth/OnboardingGuard';
import { AppLayout } from './components/layout/AppLayout';
import { LoadingScreen } from './components/ui/LoadingScreen';

// Route components are code-split so the initial load ships only the shell +
// the landing route's chunk instead of one monolithic bundle. Pages use named
// exports, so each import is re-mapped to a default for React.lazy.
const DashboardPage = lazy(() =>
  import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })),
);
const NotFoundPage = lazy(() =>
  import('./pages/NotFoundPage').then((m) => ({ default: m.NotFoundPage })),
);
const PersonListPage = lazy(() =>
  import('./pages/PersonListPage').then((m) => ({ default: m.PersonListPage })),
);
const PersonCreatePage = lazy(() =>
  import('./pages/PersonCreatePage').then((m) => ({ default: m.PersonCreatePage })),
);
const PersonDetailPage = lazy(() =>
  import('./pages/PersonDetailPage').then((m) => ({ default: m.PersonDetailPage })),
);
const PersonEditPage = lazy(() =>
  import('./pages/PersonEditPage').then((m) => ({ default: m.PersonEditPage })),
);
const ProfileCompletionPage = lazy(() =>
  import('./pages/ProfileCompletionPage').then((m) => ({
    default: m.ProfileCompletionPage,
  })),
);
const VolunteerAgreementPage = lazy(() =>
  import('./pages/VolunteerAgreementPage').then((m) => ({
    default: m.VolunteerAgreementPage,
  })),
);
const EmailVerificationPendingPage = lazy(() =>
  import('./pages/EmailVerificationPendingPage').then((m) => ({
    default: m.EmailVerificationPendingPage,
  })),
);
const CampusListPage = lazy(() =>
  import('./pages/CampusListPage').then((m) => ({ default: m.CampusListPage })),
);
const CampusFormPage = lazy(() =>
  import('./pages/CampusFormPage').then((m) => ({ default: m.CampusFormPage })),
);
const TriageListPage = lazy(() =>
  import('./pages/TriageListPage').then((m) => ({ default: m.TriageListPage })),
);
const TriageCreatePage = lazy(() =>
  import('./pages/TriageCreatePage').then((m) => ({ default: m.TriageCreatePage })),
);
const TriageDetailPage = lazy(() =>
  import('./pages/TriageDetailPage').then((m) => ({ default: m.TriageDetailPage })),
);
const AttendanceListPage = lazy(() =>
  import('./pages/AttendanceListPage').then((m) => ({
    default: m.AttendanceListPage,
  })),
);
const AttendanceCreatePage = lazy(() =>
  import('./pages/AttendanceCreatePage').then((m) => ({
    default: m.AttendanceCreatePage,
  })),
);
const AttendanceDetailPage = lazy(() =>
  import('./pages/AttendanceDetailPage').then((m) => ({
    default: m.AttendanceDetailPage,
  })),
);
const CampaignListPage = lazy(() =>
  import('./pages/CampaignListPage').then((m) => ({ default: m.CampaignListPage })),
);
const CampaignFormPage = lazy(() =>
  import('./pages/CampaignFormPage').then((m) => ({ default: m.CampaignFormPage })),
);
const CampaignDetailPage = lazy(() =>
  import('./pages/CampaignDetailPage').then((m) => ({
    default: m.CampaignDetailPage,
  })),
);
const ReportsPage = lazy(() =>
  import('./pages/ReportsPage').then((m) => ({ default: m.ReportsPage })),
);
const SyncConflictsPage = lazy(() =>
  import('./pages/SyncConflictsPage').then((m) => ({ default: m.SyncConflictsPage })),
);

export function App() {
  const { initialized, isLoading, initialize } = useAuthStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  if (!initialized || isLoading) {
    return <LoadingScreen />;
  }

  return (
    <BrowserRouter>
      <Suspense fallback={<LoadingScreen />}>
        <Routes>
          {/* Email verification — requires auth only (no email guard) */}
          <Route
            element={
              <ProtectedRoute>
                <Outlet />
              </ProtectedRoute>
            }
          >
            <Route path="email-verification" element={<EmailVerificationPendingPage />} />
          </Route>

          {/* Profile completion + agreement — requires verified email */}
          <Route
            element={
              <ProtectedRoute>
                <EmailVerifiedGuard>
                  <Outlet />
                </EmailVerifiedGuard>
              </ProtectedRoute>
            }
          >
            <Route path="complete-profile" element={<ProfileCompletionPage />} />
            <Route path="volunteer-agreement" element={<VolunteerAgreementPage />} />
          </Route>

          {/* Main app — all guards: verified email + onboarding (profile + agreement) */}
          <Route
            element={
              <ProtectedRoute>
                <EmailVerifiedGuard>
                  <OnboardingGuard>
                    <AppLayout />
                  </OnboardingGuard>
                </EmailVerifiedGuard>
              </ProtectedRoute>
            }
          >
            <Route index element={<DashboardPage />} />
            <Route path="persons" element={<PersonListPage />} />
            <Route path="persons/new" element={<PersonCreatePage />} />
            <Route path="persons/:id" element={<PersonDetailPage />} />
            <Route path="persons/:id/edit" element={<PersonEditPage />} />
            <Route path="campuses" element={<CampusListPage />} />
            <Route path="campuses/new" element={<CampusFormPage />} />
            <Route path="campuses/:id/edit" element={<CampusFormPage />} />
            <Route path="triages" element={<TriageListPage />} />
            <Route path="triages/new" element={<TriageCreatePage />} />
            <Route path="triages/:id" element={<TriageDetailPage />} />
            <Route path="attendances" element={<AttendanceListPage />} />
            <Route path="attendances/new" element={<AttendanceCreatePage />} />
            <Route path="attendances/:id" element={<AttendanceDetailPage />} />
            <Route path="campaigns" element={<CampaignListPage />} />
            <Route path="campaigns/new" element={<CampaignFormPage />} />
            <Route path="campaigns/:id" element={<CampaignDetailPage />} />
            <Route path="campaigns/:id/edit" element={<CampaignFormPage />} />
            <Route path="reports" element={<ReportsPage />} />
            <Route path="sync/conflicts" element={<SyncConflictsPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
