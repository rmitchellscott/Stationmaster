import React, { useState, useEffect } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { 
  CheckCircle, 
  AlertTriangle, 
  Loader2,
  ExternalLink,
  RefreshCw
} from 'lucide-react';
import { oauthService, OAuthConnection as OAuthConnectionType, OAuthConfig } from '@/services/oauthService';
import { format } from 'date-fns';

interface OAuthConnectionProps {
  oauthConfig: OAuthConfig;
  onConnectionChange?: (connected: boolean) => void;
  className?: string;
}

export function OAuthConnection({ 
  oauthConfig, 
  onConnectionChange,
  className = '' 
}: OAuthConnectionProps) {
  const [connection, setConnection] = useState<OAuthConnectionType | null>(null);
  const [loading, setLoading] = useState(true);
  const [connecting, setConnecting] = useState(false);
  const [disconnecting, setDisconnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    checkConnectionStatus();
  }, [oauthConfig.provider]);

  const checkConnectionStatus = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const status = await oauthService.checkConnectionStatus(oauthConfig.provider);
      setConnection(status);
      onConnectionChange?.(status.connected);
    } catch (err) {
      setError('Failed to check connection status');
      setConnection({ provider: oauthConfig.provider, connected: false });
    } finally {
      setLoading(false);
    }
  };

  const handleConnect = () => {
    setConnecting(true);
    oauthService.connectProvider(oauthConfig.provider);
    // Note: This will redirect to OAuth provider, so loading state will persist until page unloads
  };

  const handleDisconnect = async () => {
    if (!confirm(`Are you sure you want to disconnect from ${oauthService.getProviderDisplayName(oauthConfig.provider)}? You'll need to reconnect to use this plugin.`)) {
      return;
    }

    setDisconnecting(true);
    setError(null);
    
    try {
      const success = await oauthService.disconnectProvider(oauthConfig.provider);
      if (success) {
        setConnection({ ...connection!, connected: false, connected_at: undefined, scopes: undefined });
        onConnectionChange?.(false);
      } else {
        setError('Failed to disconnect. Please try again.');
      }
    } catch (err) {
      setError('Failed to disconnect. Please try again.');
    } finally {
      setDisconnecting(false);
    }
  };

  const providerName = oauthService.getProviderDisplayName(oauthConfig.provider);
  const providerIcon = oauthService.getProviderIcon(oauthConfig.provider);

  if (loading) {
    return (
      <Card className={className}>
        <CardContent className="pt-6">
          <div className="flex items-center gap-3">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div className="flex-1">
              <Skeleton className="h-4 w-32 mb-2" />
              <Skeleton className="h-3 w-48" />
            </div>
            <Skeleton className="h-10 w-28" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Alert className={`border-red-200 ${className}`}>
        <AlertTriangle className="h-4 w-4 text-red-600" />
        <AlertDescription className="flex items-center justify-between">
          <span>{error}</span>
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={checkConnectionStatus}
            disabled={loading}
          >
            <RefreshCw className="h-4 w-4" />
            Retry
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  if (!connection?.connected) {
    return (
      <Card className={`border-amber-200 bg-amber-50/50 ${className}`}>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100">
              <AlertTriangle className="h-5 w-5 text-amber-600" />
            </div>
            <div className="flex-1">
              <p className="font-medium text-gray-900">
                {providerIcon} {providerName} Not Connected
              </p>
              <p className="text-sm text-gray-600">
                Connect your {providerName} account to use this plugin
              </p>
            </div>
            <Button 
              onClick={handleConnect}
              disabled={connecting}
              className="gap-2"
            >
              {connecting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Connecting...
                </>
              ) : (
                <>
                  Connect to {providerName}
                  <ExternalLink className="h-4 w-4" />
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={`border-green-200 bg-green-50/50 ${className}`}>
      <CardContent className="pt-6">
        <div className="flex items-center gap-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-100">
            <CheckCircle className="h-5 w-5 text-green-600" />
          </div>
          <div className="flex-1">
            <p className="font-medium text-gray-900">
              {providerIcon} {providerName} Connected
            </p>
            <p className="text-sm text-gray-600">
              {connection.connected_at && (
                <>Connected on {format(new Date(connection.connected_at), 'MMM d, yyyy')}</>
              )}
              {connection.scopes && connection.scopes.length > 0 && (
                <> • {connection.scopes.length} permission{connection.scopes.length > 1 ? 's' : ''} granted</>
              )}
            </p>
          </div>
          <Button 
            variant="outline"
            onClick={handleDisconnect}
            disabled={disconnecting}
            className="text-gray-600 hover:text-red-600 hover:border-red-300"
          >
            {disconnecting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                Disconnecting...
              </>
            ) : (
              'Disconnect'
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}