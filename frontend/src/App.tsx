import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from 'react-query';
import { Layout } from '@shared/components/Layout';
import { useAuthStore } from '@shared/store/authStore';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { ConnectionsPage } from './pages/ConnectionsPage';
import { PipelinesPage } from './pages/PipelinesPage';
import { CreatePipelinePage } from './pages/CreatePipelinePage';
import { PipelineDetailsPage } from './pages/PipelineDetailsPage';
import { IntegrationsPage } from './pages/IntegrationsPage';
import { OAuthCallbackPage } from './pages/OAuthCallbackPage';
import { OAuthErrorPage } from './pages/OAuthErrorPage';
import { SelectSheetPage } from './pages/SelectSheetPage';
import { SelectQuickBooksObjectPage } from './pages/SelectQuickBooksObjectPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuthStore();

  // Check localStorage as fallback in case store hasn't updated yet
  const storedUser = localStorage.getItem('user');
  const storedToken = localStorage.getItem('auth_token');
  const hasAuth = isAuthenticated || (storedUser && storedToken);

  if (isLoading && !hasAuth) {
    return <div className="min-h-screen flex items-center justify-center">Loading...</div>;
  }

  if (!hasAuth) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function App() {
  const { initialize } = useAuthStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Layout>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/oauth" element={<OAuthCallbackPage />} />
            <Route path="/oauth-error" element={<OAuthErrorPage />} />
            <Route
              path="/dashboard"
              element={
                <PrivateRoute>
                  <DashboardPage />
                </PrivateRoute>
              }
            />
            <Route
              path="/connections"
              element={
                <PrivateRoute>
                  <ConnectionsPage />
                </PrivateRoute>
              }
            />
            <Route
              path="/connections/select-sheet"
              element={<SelectSheetPage />}
            />
            <Route
              path="/connections/select-quickbooks-object"
              element={<SelectQuickBooksObjectPage />}
            />
            <Route
              path="/pipelines"
              element={
                <PrivateRoute>
                  <PipelinesPage />
                </PrivateRoute>
              }
            />
            <Route
              path="/pipelines/create"
              element={
                <PrivateRoute>
                  <CreatePipelinePage />
                </PrivateRoute>
              }
            />
            <Route
              path="/pipelines/:id"
              element={
                <PrivateRoute>
                  <PipelineDetailsPage />
                </PrivateRoute>
              }
            />
            <Route
              path="/integrations"
              element={
                <PrivateRoute>
                  <IntegrationsPage />
                </PrivateRoute>
              }
            />
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;

