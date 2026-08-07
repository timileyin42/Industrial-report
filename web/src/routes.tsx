import { createBrowserRouter } from "react-router-dom";
import { RequireAuth } from "./auth/RequireAuth";
import { RequireRole } from "./auth/RequireRole";
import { AppLayout } from "./components/layout/AppLayout";
import { HomePage } from "./pages/landing/HomePage";
import { FeaturesPage } from "./pages/landing/FeaturesPage";
import { SolutionsPage } from "./pages/landing/SolutionsPage";
import { CompanyPage } from "./pages/landing/CompanyPage";
import { LoginPage } from "./pages/LoginPage";
import { FleetDashboardPage } from "./pages/FleetDashboardPage";
import { SitesListPage } from "./pages/SitesListPage";
import { AddSitePage } from "./pages/AddSitePage";
import { SiteDetailPage } from "./pages/SiteDetailPage";
import { DevicesListPage } from "./pages/DevicesListPage";
import { RegisterDevicePage } from "./pages/RegisterDevicePage";
import { SiteAnalyticsPage } from "./pages/SiteAnalyticsPage";
import { FleetAnalyticsPage } from "./pages/FleetAnalyticsPage";
import { EmissionFactorSettingsPage } from "./pages/EmissionFactorSettingsPage";
import { FleetHealthPage } from "./pages/FleetHealthPage";
import { AuditLogPage } from "./pages/AuditLogPage";
import { IngestionAuditPage } from "./pages/IngestionAuditPage";
import { InviteUserPage } from "./pages/InviteUserPage";
import { AcceptInvitePage } from "./pages/AcceptInvitePage";
import { ForgotPasswordPage } from "./pages/ForgotPasswordPage";
import { ResetPasswordPage } from "./pages/ResetPasswordPage";

export const router = createBrowserRouter([
  // Public marketing site
  { path: "/", element: <HomePage /> },
  { path: "/features", element: <FeaturesPage /> },
  { path: "/solutions", element: <SolutionsPage /> },
  { path: "/company", element: <CompanyPage /> },
  { path: "/login", element: <LoginPage /> },
  { path: "/accept-invite", element: <AcceptInvitePage /> },
  { path: "/forgot-password", element: <ForgotPasswordPage /> },
  { path: "/reset-password", element: <ResetPasswordPage /> },

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
              { path: "analytics", element: <FleetAnalyticsPage /> },
              { path: "settings/emissions", element: <EmissionFactorSettingsPage /> },
              { path: "fleet-health", element: <FleetHealthPage /> },
              { path: "audit", element: <AuditLogPage /> },
              { path: "users/invite", element: <InviteUserPage /> },
            ],
          },
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
