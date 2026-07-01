import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Outlet } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { EmailVerifiedGuard } from './components/auth/EmailVerifiedGuard';
import { OnboardingGuard } from './components/auth/OnboardingGuard';
import { AppLayout } from './components/layout/AppLayout';
import { LoadingScreen } from './components/ui/LoadingScreen';
import { DashboardPage } from './pages/DashboardPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { PersonListPage } from './pages/PersonListPage';
import { PersonCreatePage } from './pages/PersonCreatePage';
import { PersonDetailPage } from './pages/PersonDetailPage';
import { PersonEditPage } from './pages/PersonEditPage';
import { ProfileCompletionPage } from './pages/ProfileCompletionPage';
import { VolunteerAgreementPage } from './pages/VolunteerAgreementPage';
import { EmailVerificationPendingPage } from './pages/EmailVerificationPendingPage';
import { CampusListPage } from './pages/CampusListPage';
import { CampusFormPage } from './pages/CampusFormPage';
import { TriageListPage } from './pages/TriageListPage';
import { TriageCreatePage } from './pages/TriageCreatePage';
import { TriageDetailPage } from './pages/TriageDetailPage';
import { AttendanceListPage } from './pages/AttendanceListPage';
import { AttendanceCreatePage } from './pages/AttendanceCreatePage';
import { AttendanceDetailPage } from './pages/AttendanceDetailPage';
import { ReportsPage } from './pages/ReportsPage';
import { SyncConflictsPage } from './pages/SyncConflictsPage';

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
      <Routes>
        {/* Email verification — requires auth only (no email guard) */}
        <Route
          element={
            <ProtectedRoute>
              <Outlet />
            </ProtectedRoute>
          }
        >
          <Route
            path="email-verification"
            element={<EmailVerificationPendingPage />}
          />
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
          <Route
            path="complete-profile"
            element={<ProfileCompletionPage />}
          />
          <Route
            path="volunteer-agreement"
            element={<VolunteerAgreementPage />}
          />
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
          <Route path="reports" element={<ReportsPage />} />
          <Route path="sync/conflicts" element={<SyncConflictsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
