import React, { useEffect, useState } from 'react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

type AuthState = 'loading' | 'anonymous' | 'authenticated';

const ReportPage: React.FC = () => {
  const [authState, setAuthState] = useState<AuthState>('loading');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<unknown>(null);

  useEffect(() => {
    let cancelled = false;

    const checkSession = async () => {
      try {
        const response = await fetch(`${API_URL}/auth/me`, {
          credentials: 'include',
        });
        if (cancelled) return;
        setAuthState(response.ok ? 'authenticated' : 'anonymous');
      } catch {
        if (!cancelled) setAuthState('anonymous');
      }
    };

    void checkSession();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = () => {
    window.location.href = `${API_URL}/auth/login`;
  };

  const logout = async () => {
    setError(null);
    try {
      const response = await fetch(`${API_URL}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
      const data = (await response.json().catch(() => null)) as
        | { logout_url?: string }
        | null;
      setReport(null);
      setAuthState('anonymous');
      // End Keycloak SSO session, otherwise next Login skips password/OTP.
      if (data?.logout_url) {
        window.location.href = data.logout_url;
        return;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Logout failed');
    }
  };

  const downloadReport = async () => {
    try {
      setLoading(true);
      setError(null);
      setReport(null);

      const response = await fetch(`${API_URL}/reports`, {
        credentials: 'include',
      });

      if (response.status === 401) {
        setAuthState('anonymous');
        setError('Session expired. Please log in again.');
        return;
      }

      if (!response.ok) {
        throw new Error(`Request failed: ${response.status}`);
      }

      const data = await response.json();
      setReport(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  if (authState === 'loading') {
    return <div>Loading...</div>;
  }

  if (authState === 'anonymous') {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
        <button
          onClick={login}
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
        >
          Login
        </button>
        {error && (
          <div className="mt-4 p-4 bg-red-100 text-red-700 rounded">{error}</div>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-100">
      <div className="p-8 bg-white rounded-lg shadow-md min-w-[320px]">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold">Usage Reports</h1>
          <button
            onClick={logout}
            className="px-3 py-1 text-sm text-gray-600 border border-gray-300 rounded hover:bg-gray-50"
          >
            Logout
          </button>
        </div>

        <button
          onClick={downloadReport}
          disabled={loading}
          className={`px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 ${
            loading ? 'opacity-50 cursor-not-allowed' : ''
          }`}
        >
          {loading ? 'Generating Report...' : 'Download Report'}
        </button>

        {error && (
          <div className="mt-4 p-4 bg-red-100 text-red-700 rounded">{error}</div>
        )}

        {report != null && (
          <pre className="mt-4 p-4 bg-gray-50 text-sm overflow-auto rounded">
            {JSON.stringify(report, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
};

export default ReportPage;
