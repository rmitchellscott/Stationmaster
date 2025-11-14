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
   * Initiate OAuth connection flow using popup window
   */
  connectProvider(provider: string): Promise<boolean> {
    return new Promise((resolve, reject) => {
      // Popup configuration
      const width = 500;
      const height = 600;
      const left = window.screenX + (window.outerWidth - width) / 2;
      const top = window.screenY + (window.outerHeight - height) / 2;
      
      // Open popup window for OAuth flow
      const popup = window.open(
        `/api/oauth/${provider}/auth`,
        'oauth_popup',
        `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes`
      );
      
      if (!popup) {
        // Popup blocked - fallback to redirect
        window.location.href = `/api/oauth/${provider}/auth`;
        reject(new Error('Popup blocked, falling back to redirect'));
        return;
      }
      
      // Listen for messages from popup
      const messageListener = (event: MessageEvent) => {
        // Security: Check origin
        if (event.origin !== window.location.origin) {
          return;
        }
        
        if (event.data.type === 'OAUTH_SUCCESS' && event.data.provider === provider) {
          // Success - cleanup and resolve
          window.removeEventListener('message', messageListener);
          popup.close();
          resolve(true);
        } else if (event.data.type === 'OAUTH_ERROR') {
          // Error - cleanup and reject
          window.removeEventListener('message', messageListener);
          popup.close();
          reject(new Error(event.data.error || 'OAuth failed'));
        }
      };
      
      window.addEventListener('message', messageListener);
      
      // Handle popup closed by user
      const checkClosed = setInterval(() => {
        if (popup.closed) {
          clearInterval(checkClosed);
          window.removeEventListener('message', messageListener);
          reject(new Error('OAuth popup closed by user'));
        }
      }, 1000);
    });
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