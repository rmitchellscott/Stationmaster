import React, { useState, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PageCard, PageCardContent, PageCardHeader, PageCardTitle } from "@/components/ui/page-card";
import { Button } from "@/components/ui/button";
import {
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import {
  ArrowLeft,
  Settings,
  AlertTriangle,
  CheckCircle,
} from "lucide-react";
import { PrivatePluginCreator } from "./PrivatePluginCreator";

interface PrivatePlugin {
  id?: string;
  name: string;
  description: string;
  markup_full: string;
  markup_half_vert: string;
  markup_half_horiz: string;
  markup_quadrant: string;
  shared_markup: string;
  data_strategy: 'webhook' | 'polling' | 'static';
  polling_config?: any;
  form_fields?: any;
  sample_data?: any;
  version: string;
  remove_bleed_margin?: boolean;
  enable_dark_mode?: boolean;
}

export function PrivatePluginEditorPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  
  const [plugin, setPlugin] = useState<PrivatePlugin | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const pluginId = searchParams.get('pluginId');
  const isEditing = Boolean(pluginId);

  // Fetch plugin data if editing
  useEffect(() => {
    const fetchPlugin = async () => {
      if (!pluginId) return;
      
      try {
        setLoading(true);
        setError(null);
        
        const response = await fetch(`/api/plugin-definitions/${pluginId}`, {
          credentials: 'include',
        });
        
        if (response.ok) {
          const data = await response.json();
          console.log('[PrivatePluginEditorPage] Fetched plugin data:', data);
          const pluginData = data.plugin_definition || data;
          console.log('[PrivatePluginEditorPage] Setting plugin:', pluginData);
          console.log('[PrivatePluginEditorPage] Plugin sample_data:', pluginData.sample_data);
          setPlugin(pluginData);
        } else {
          const errorData = await response.json();
          setError(errorData.error || 'Failed to fetch plugin');
        }
      } catch (error) {
        setError('Network error occurred');
      } finally {
        setLoading(false);
      }
    };

    fetchPlugin();
  }, [pluginId]);

  const handleSavePlugin = async (pluginData: PrivatePlugin) => {
    try {
      setSaving(true);
      setError(null);
      setSuccess(null);
      
      const url = isEditing ? `/api/plugin-definitions/${pluginId}` : '/api/plugin-definitions';
      const method = isEditing ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          ...pluginData,
          plugin_type: 'private'
        }),
      });

      if (response.ok) {
        setSuccess(`Private plugin ${isEditing ? 'updated' : 'created'} successfully!`);
        navigateBackToPlugins();
      } else {
        const errorData = await response.json();
        setError(errorData.error || `Failed to ${isEditing ? 'update' : 'create'} private plugin`);
      }
    } catch (error) {
      setError('Network error occurred');
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    navigateBackToPlugins();
  };

  const navigateBackToPlugins = () => {
    // Navigate back to homepage with proper tab and subtab query parameters
    navigate('/?tab=plugins&subtab=private');
  };

  if (loading) {
    return (
      <div className="bg-background pt-0 pb-8 px-0 sm:px-8">
        <div className="max-w-6xl mx-0 sm:mx-auto space-y-6">
          <PageCard>
            <PageCardContent className="flex items-center justify-center py-8">
              <div className="text-muted-foreground">Loading plugin...</div>
            </PageCardContent>
          </PageCard>
        </div>
      </div>
    );
  }
  
  return (
    <div className="bg-background pt-0 pb-8 px-0 sm:px-8">
      <div className="max-w-6xl mx-0 sm:mx-auto space-y-6">
        
        {error && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {success && (
          <Alert>
            <CheckCircle className="h-4 w-4" />
            <AlertDescription>{success}</AlertDescription>
          </Alert>
        )}

        <PageCard>
          <PageCardHeader>
            <div>
              <button 
                onClick={navigateBackToPlugins}
                className="text-sm text-muted-foreground hover:text-foreground inline-flex items-center gap-1 mb-1"
              >
                <ArrowLeft className="h-3 w-3" />
                Back to Plugin Management
              </button>
              <PageCardTitle className="flex items-center gap-2 text-2xl">
                <Settings className="h-5 w-5" />
                {isEditing ? 'Edit Private Plugin' : 'Create Private Plugin'}
              </PageCardTitle>
            </div>
          </PageCardHeader>
          <PageCardContent>
            <PrivatePluginCreator
              plugin={plugin}
              onSave={handleSavePlugin}
              onCancel={handleCancel}
              saving={saving}
            />
          </PageCardContent>
        </PageCard>
      </div>
    </div>
  );
}
