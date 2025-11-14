import { useState, useEffect } from 'react';
import { oauthService, OAuthConnection } from '@/services/oauthService';

interface UseOAuthStatusResult {
  connection: OAuthConnection | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * React hook for managing OAuth connection status
 * @param provider - The OAuth provider name (e.g., 'google', 'todoist')
 */
export function useOAuthStatus(provider: string | undefined): UseOAuthStatusResult {
  const [connection, setConnection] = useState<OAuthConnection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = async () => {
    if (!provider) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const status = await oauthService.checkConnectionStatus(provider);
      setConnection(status);
    } catch (err) {
      setError('Failed to check OAuth connection status');
      setConnection({ provider, connected: false });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, [provider]);

  // Listen for window focus to refresh status (user might have completed OAuth in another tab)
  useEffect(() => {
    const handleFocus = () => {
      if (provider) {
        fetchStatus();
      }
    };

    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, [provider]);

  return {
    connection,
    loading,
    error,
    refetch: fetchStatus,
  };
}

/**
 * React hook for managing multiple OAuth providers' status
 * @param providers - Array of OAuth provider names
 */
export function useMultipleOAuthStatus(providers: string[]): Record<string, UseOAuthStatusResult> {
  const [statuses, setStatuses] = useState<Record<string, UseOAuthStatusResult>>({});

  useEffect(() => {
    const newStatuses: Record<string, UseOAuthStatusResult> = {};
    
    providers.forEach(provider => {
      // This is a simplified version - in production, you'd want to use individual hooks
      // or a more sophisticated state management approach
      newStatuses[provider] = {
        connection: null,
        loading: true,
        error: null,
        refetch: async () => {
          try {
            const status = await oauthService.checkConnectionStatus(provider);
            setStatuses(prev => ({
              ...prev,
              [provider]: {
                ...prev[provider],
                connection: status,
                loading: false,
                error: null,
              }
            }));
          } catch (err) {
            setStatuses(prev => ({
              ...prev,
              [provider]: {
                ...prev[provider],
                connection: { provider, connected: false },
                loading: false,
                error: 'Failed to check status',
              }
            }));
          }
        }
      };
    });

    setStatuses(newStatuses);

    // Initial fetch for all providers
    providers.forEach(async provider => {
      try {
        const status = await oauthService.checkConnectionStatus(provider);
        setStatuses(prev => ({
          ...prev,
          [provider]: {
            ...prev[provider],
            connection: status,
            loading: false,
          }
        }));
      } catch (err) {
        setStatuses(prev => ({
          ...prev,
          [provider]: {
            ...prev[provider],
            connection: { provider, connected: false },
            loading: false,
            error: 'Failed to check status',
          }
        }));
      }
    });
  }, [providers.join(',')]);

  return statuses;
}