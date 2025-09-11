// OAuth Service for managing OAuth connections with external services

export interface OAuthConnection {
  provider: string;
  connected: boolean;
  connected_at?: string;
  scopes?: string[];
}

export interface OAuthConfig {
  provider: string;
  auth_url: string;
  token_url: string;
  scopes: string[];
  client_id_env: string;
  client_secret_env: string;
}

class OAuthService {
  /**
   * Check if a user has connected a specific OAuth provider
   */
  async checkConnectionStatus(provider: string): Promise<OAuthConnection> {
    try {
      const response = await fetch(`/api/oauth/${provider}/status`, {
        credentials: 'include',
      });

      if (response.ok) {
        return await response.json();
      }

      // If 404, provider is not connected
      if (response.status === 404) {
        return {
          provider,
          connected: false,
        };
      }

      throw new Error('Failed to check OAuth connection status');
    } catch (error) {
      console.error('Error checking OAuth status:', error);
      return {
        provider,
        connected: false,
      };
    }
  }

  /**
   * Initiate OAuth connection flow
   */
  connectProvider(provider: string): void {
    // Store current location for return after OAuth flow
    sessionStorage.setItem('oauth_return_url', window.location.pathname);
    sessionStorage.setItem('oauth_provider', provider);
    
    // Redirect to OAuth endpoint
    window.location.href = `/api/oauth/${provider}/auth`;
  }

  /**
   * Disconnect OAuth provider
   */
  async disconnectProvider(provider: string): Promise<boolean> {
    try {
      const response = await fetch(`/api/oauth/${provider}/disconnect`, {
        method: 'DELETE',
        credentials: 'include',
      });

      return response.ok;
    } catch (error) {
      console.error('Error disconnecting OAuth provider:', error);
      return false;
    }
  }

  /**
   * Handle OAuth return from provider
   */
  handleOAuthReturn(): { success: boolean; provider?: string; error?: string } {
    const urlParams = new URLSearchParams(window.location.search);
    
    // Check for success
    const successProvider = urlParams.get('oauth_success');
    if (successProvider) {
      // Clean up URL
      const newUrl = window.location.pathname;
      window.history.replaceState({}, document.title, newUrl);
      
      return {
        success: true,
        provider: successProvider,
      };
    }

    // Check for error
    const errorProvider = urlParams.get('oauth_error');
    const errorMessage = urlParams.get('error');
    if (errorProvider) {
      // Clean up URL
      const newUrl = window.location.pathname;
      window.history.replaceState({}, document.title, newUrl);
      
      return {
        success: false,
        provider: errorProvider,
        error: errorMessage || 'OAuth connection failed',
      };
    }

    return { success: false };
  }

  /**
   * Get stored return URL after OAuth flow
   */
  getReturnUrl(): string | null {
    const url = sessionStorage.getItem('oauth_return_url');
    sessionStorage.removeItem('oauth_return_url');
    return url;
  }

  /**
   * Get stored provider from OAuth flow
   */
  getStoredProvider(): string | null {
    const provider = sessionStorage.getItem('oauth_provider');
    sessionStorage.removeItem('oauth_provider');
    return provider;
  }

  /**
   * Get friendly provider name for display
   */
  getProviderDisplayName(provider: string): string {
    const displayNames: Record<string, string> = {
      google: 'Google',
      todoist: 'Todoist',
      github: 'GitHub',
      shopify: 'Shopify',
    };
    
    return displayNames[provider] || provider.charAt(0).toUpperCase() + provider.slice(1);
  }

  /**
   * Get provider icon/logo (could be expanded with actual icons)
   */
  getProviderIcon(provider: string): string {
    const icons: Record<string, string> = {
      google: '🔍',
      todoist: '✅',
      github: '🐙',
      shopify: '🛍️',
    };
    
    return icons[provider] || '🔗';
  }
}

export const oauthService = new OAuthService();