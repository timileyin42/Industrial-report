import { createBrowserRouter } from "react-router-dom";
import { RequireAuth } from "./auth/RequireAuth";
import { RequireRole } from "./auth/RequireRole";
import { AppLayout } from "./components/layout/AppLayout";
import { HomePage } from "./pages/landing/HomePage";
import { FeaturesPage } from "./pages/landing/FeaturesPage";
import { SolutionsPage } from "./pages/landing/SolutionsPage";
import { CompanyPage } from "./pages/landing/CompanyPage";
import { TermsOfServicePage } from "./pages/landing/TermsOfServicePage";
import { PrivacyPolicyPage } from "./pages/landing/PrivacyPolicyPage";
import { SecurityPage } from "./pages/landing/SecurityPage";
import { LoginPage } from "./pages/LoginPage";
import { FleetDashboardPage } from "./pages/FleetDashboardPage";
import { SitesListPage } from "./pages/SitesListPage";
import { AddSitePage } from "./pages/AddSitePage";
import { SiteDetailPage } from "./pages/SiteDetailPage";
import { DevicesListPage } from "./pages/DevicesListPage";
import { RegisterDevicePage } from "./pages/RegisterDevicePage";
import { SiteAnalyticsPage } from "./pages/SiteAnalyticsPage";
import { PerformancePage } from "./pages/PerformancePage";
import { EnergyPage } from "./pages/EnergyPage";
import { EmissionsPage } from "./pages/EmissionsPage";
import { ReportsPage } from "./pages/ReportsPage";
import { MapViewPage } from "./pages/MapViewPage";
import { AlertsPage } from "./pages/AlertsPage";
import { UsersListPage } from "./pages/UsersListPage";
import { CohortsListPage } from "./pages/CohortsListPage";
import { SettingsPage } from "./pages/SettingsPage";
import { HelpPage } from "./pages/HelpPage";
import { EmissionFactorSettingsPage } from "./pages/EmissionFactorSettingsPage";
import { FleetHealthPage } from "./pages/FleetHealthPage";
import { AuditLogPage } from "./pages/AuditLogPage";
import { IngestionAuditPage } from "./pages/IngestionAuditPage";
import { InviteUserPage } from "./pages/InviteUserPage";
import { AcceptInvitePage } from "./pages/AcceptInvitePage";
import { ForgotPasswordPage } from "./pages/ForgotPasswordPage";
import { ResetPasswordPage } from "./pages/ResetPasswordPage";
import { SandboxUploadPage } from "./pages/SandboxUploadPage";
import { SandboxResultsPage } from "./pages/SandboxResultsPage";

export const router = createBrowserRouter([
  // Public marketing site
  { path: "/", element: <HomePage /> },
  { path: "/features", element: <FeaturesPage /> },
  { path: "/solutions", element: <SolutionsPage /> },
  { path: "/company", element: <CompanyPage /> },
  { path: "/terms", element: <TermsOfServicePage /> },
  { path: "/privacy", element: <PrivacyPolicyPage /> },
  { path: "/security", element: <SecurityPage /> },
  { path: "/login", element: <LoginPage /> },
  { path: "/accept-invite", element: <AcceptInvitePage /> },
  { path: "/forgot-password", element: <ForgotPasswordPage /> },
  { path: "/reset-password", element: <ResetPasswordPage /> },
  { path: "/sandbox", element: <SandboxUploadPage /> },
  { path: "/sandbox/:runId", element: <SandboxResultsPage /> },

  // Authenticated dashboard app, under /app
  {
    path: "/app",
    element: <RequireAuth />,
    children: [
      {
        element: <AppLayout />,
        children: [
          {
            element: <RequireRole role="operator" />,
            children: [
              { index: true, element: <FleetDashboardPage /> },
              { path: "sites/new", element: <AddSitePage /> },
              { path: "devices/new", element: <RegisterDevicePage /> },
              { path: "analytics/performance", element: <PerformancePage /> },
              { path: "analytics/energy", element: <EnergyPage /> },
              { path: "analytics/emissions", element: <EmissionsPage /> },
              { path: "reports", element: <ReportsPage /> },
              { path: "map", element: <MapViewPage /> },
              { path: "alerts", element: <AlertsPage /> },
              { path: "cohorts", element: <CohortsListPage /> },
              { path: "users", element: <UsersListPage /> },
              { path: "settings", element: <SettingsPage /> },
              { path: "settings/emissions", element: <EmissionFactorSettingsPage /> },
              { path: "fleet-health", element: <FleetHealthPage /> },
              { path: "audit", element: <AuditLogPage /> },
              { path: "users/invite", element: <InviteUserPage /> },
            ],
          },
          { path: "help", element: <HelpPage /> },
          { path: "sites", element: <SitesListPage /> },
          { path: "sites/:siteId", element: <SiteDetailPage /> },
          { path: "sites/:siteId/analytics", element: <SiteAnalyticsPage /> },
          { path: "devices", element: <DevicesListPage /> },
          { path: "ingestion-log", element: <IngestionAuditPage /> },
        ],
      },
    ],
  },
]);
