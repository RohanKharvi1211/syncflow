import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuthStore } from '@shared/store/authStore';

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { setUser, setToken } = useAuthStore();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    const error = searchParams.get('error');
    const success = searchParams.get('success');
    const provider = searchParams.get('provider');
    const email = searchParams.get('email');
    const token = searchParams.get('token');
    const companyId = searchParams.get('company_id');
    const companyName = searchParams.get('company_name');

    if (error) {
      setStatus('error');
      setMessage(decodeURIComponent(error));
      return;
    }

    if (success === 'true' && token && email && companyId) {
      setStatus('success');
      setMessage(`Successfully authenticated with ${provider || 'account'}`);
      
      // Store token
      setToken(token);
      
      // Create user object from URL params
      const user = {
        id: '', // Will be set from token or fetched
        company_id: companyId,
        email: decodeURIComponent(email),
        company_name: companyName ? decodeURIComponent(companyName) : undefined,
        role: 'user' as const,
        is_active: true,
      };
      
      setUser(user);
      
      // Redirect to dashboard for the company
      setTimeout(() => {
        navigate('/dashboard');
      }, 1500);
      return;
    }

    // If no error or success, this might be the initial callback
    // Backend should have processed it and redirected with success/error
    setStatus('loading');
  }, [searchParams, navigate, setUser, setToken]);

  if (status === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Completing authentication...</p>
        </div>
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
          <div className="text-center">
            <div className="text-red-600 text-5xl mb-4">✕</div>
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Authentication Failed</h2>
            <p className="text-gray-600 mb-6">{message}</p>
            <button
              onClick={() => navigate('/login')}
              className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
            >
              Back to Login
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
        <div className="text-center">
          <div className="text-green-600 text-5xl mb-4">✓</div>
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Authentication Successful</h2>
          <p className="text-gray-600 mb-6">Redirecting to dashboard...</p>
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto"></div>
        </div>
      </div>
    </div>
  );
}

