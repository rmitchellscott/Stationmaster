import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PageCard, PageCardContent, PageCardHeader, PageCardTitle } from "@/components/ui/page-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import { 
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
} from "@/components/ui/command";
import {
  ArrowLeft,
  Puzzle,
  Search,
  Settings as SettingsIcon,
  AlertTriangle,
  CheckCircle,
} from "lucide-react";
import { OAuthConnection } from "@/components/OAuthConnection";
import { useOAuthStatus } from "@/hooks/useOAuthStatus";

interface Plugin {
  id: string;
  name: string;
  type: string;
  description: string;
  author: string;
  version: string;
  config_schema: string;
  requires_processing: boolean;
  status: string; // "available", "unavailable", "error"
  oauth_config?: {
    provider: string;
    auth_url: string;
    token_url: string;
    scopes: string[];
    client_id_env: string;
    client_secret_env: string;
  };
}

interface RefreshRateOption {
  value: number;
  label: string;
}

type PluginType = 'all' | 'system' | 'private';

export function AddPluginPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [filteredPlugins, setFilteredPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedType, setSelectedType] = useState<PluginType>('all');
  const [expandedPlugin, setExpandedPlugin] = useState<Plugin | null>(null);
  const [refreshRateOptions, setRefreshRateOptions] = useState<RefreshRateOption[]>([]);

  // Get OAuth status for expanded plugin
  const { connection: oauthConnection } = useOAuthStatus(expandedPlugin?.oauth_config?.provider);

  // OAuth tokens state - stored separately from connection status
  const [oauthTokens, setOauthTokens] = useState<Record<string, any> | null>(null);

  // Form state for instance creation
  const [instanceName, setInstanceName] = useState("");
  const [instanceRefreshRate, setInstanceRefreshRate] = useState<number>(86400);
  const [instanceSettings, setInstanceSettings] = useState<Record<string, any>>({});
  const [createLoading, setCreateLoading] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [createSuccess, setCreateSuccess] = useState<string | null>(null);

  // Dynamic field state
  const [dynamicFieldOptions, setDynamicFieldOptions] = useState<Record<string, any[]>>({});
  const [dynamicFieldsLoading, setDynamicFieldsLoading] = useState<Record<string, boolean>>({});

  // Fetch dynamic field options from the plugin service
  const fetchDynamicFieldOptions = async (
    pluginId: string,
    fieldName: string,
    oauthTokens: Record<string, any> = {}
  ) => {
    const fieldKey = `${pluginId}.${fieldName}`;
    const apiUrl = `/api/plugins/${pluginId}/options/${fieldName}`;

    console.log('[DEBUG] fetchDynamicFieldOptions called:', {
      pluginId,
      fieldName,
      fieldKey,
      apiUrl,
      hasTokens: !!oauthTokens,
      tokenKeys: Object.keys(oauthTokens || {}),
      tokens: oauthTokens
    });

    try {
      setDynamicFieldsLoading(prev => ({ ...prev, [fieldKey]: true }));

      const requestBody = { oauth_tokens: oauthTokens };
      console.log('[DEBUG] Sending API request:', {
        url: apiUrl,
        method: 'POST',
        body: requestBody
      });

      const response = await fetch(apiUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(requestBody)
      });

      console.log('[DEBUG] API response:', {
        ok: response.ok,
        status: response.status,
        statusText: response.statusText
      });

      if (response.ok) {
        const data = await response.json();
        console.log('[DEBUG] API response data:', data);
        console.log('[DEBUG] Options received:', data.data?.options);

        setDynamicFieldOptions(prev => {
          const updated = { ...prev, [fieldKey]: data.data?.options || [] };
          console.log('[DEBUG] Updated dynamicFieldOptions state:', updated);
          return updated;
        });
      } else {
        const errorText = await response.text();
        console.error('[DEBUG] Failed to fetch dynamic field options:', {
          status: response.status,
          statusText: response.statusText,
          errorBody: errorText
        });
        setDynamicFieldOptions(prev => ({ ...prev, [fieldKey]: [] }));
      }
    } catch (error) {
      console.error('[DEBUG] Error fetching dynamic field options:', error);
      setDynamicFieldOptions(prev => ({ ...prev, [fieldKey]: [] }));
    } finally {
      setDynamicFieldsLoading(prev => ({ ...prev, [fieldKey]: false }));
    }
  };

  // Fetch plugins
  const fetchPlugins = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/plugin-definitions", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPlugins(data.plugins || []);
      }
    } catch (error) {
      console.error("Failed to fetch plugins:", error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch refresh rate options
  const fetchRefreshRateOptions = async () => {
    try {
      const response = await fetch("/api/plugin-definitions/refresh-rate-options", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setRefreshRateOptions(data.refresh_rate_options || []);
      }
    } catch (error) {
      console.error("Failed to fetch refresh rate options:", error);
    }
  };

  useEffect(() => {
    fetchPlugins();
    fetchRefreshRateOptions();
  }, []);

  // Fetch dynamic fields when plugin is expanded and OAuth is connected
  useEffect(() => {
    console.log('[DEBUG] OAuth useEffect triggered:', {
      hasExpandedPlugin: !!expandedPlugin,
      pluginId: expandedPlugin?.id,
      pluginType: expandedPlugin?.type,
      pluginName: expandedPlugin?.name,
      oauthConnected: oauthConnection?.connected,
      hasStoredTokens: !!oauthTokens,
      storedTokenKeys: Object.keys(oauthTokens || {}),
      fullOauthConnection: oauthConnection,
      oauthTokens: oauthTokens
    });

    // Try to fetch if OAuth is connected, even without tokens (backend might have them in session)
    if (expandedPlugin && oauthConnection?.connected) {
      try {
        const schema = JSON.parse(expandedPlugin.config_schema);
        const properties = schema.properties || {};

        console.log('[DEBUG] Plugin schema parsed:', {
          pluginId: expandedPlugin.id,
          propertyKeys: Object.keys(properties),
          fullProperties: properties
        });

        Object.keys(properties).forEach(key => {
          const prop = properties[key];
          const dynamicSourceField = prop.dynamicSource || prop.dynamic_source;

          console.log(`[DEBUG] Checking property ${key}:`, {
            isDynamic: prop.dynamic,
            dynamicSource: prop.dynamicSource,
            dynamic_source: prop.dynamic_source,
            dynamicSourceField,
            fullProp: prop
          });

          if (prop.dynamic && dynamicSourceField) {
            // Check if we haven't already fetched this field
            const fieldKey = `${expandedPlugin.id}.${dynamicSourceField}`;
            const alreadyFetched = !!dynamicFieldOptions[fieldKey];
            const currentlyLoading = !!dynamicFieldsLoading[fieldKey];

            console.log(`[DEBUG] Dynamic field ${key} status:`, {
              fieldKey,
              alreadyFetched,
              currentlyLoading,
              willFetch: !alreadyFetched && !currentlyLoading
            });

            if (!alreadyFetched && !currentlyLoading) {
              // For external plugins, the API might expect a different identifier
              // Use the identifier field from plugin metadata
              const pluginIdentifier = expandedPlugin.identifier || expandedPlugin.id;

              console.log(`[DEBUG] Fetching dynamic options for ${key} (tokens: ${!!oauthTokens}, identifier: ${pluginIdentifier})`);
              // Pass tokens if available, otherwise empty object - backend might have them in session
              fetchDynamicFieldOptions(pluginIdentifier, dynamicSourceField, oauthTokens || {});
            }
          }
        });
      } catch (e) {
        console.error('[DEBUG] Error parsing plugin schema for dynamic fields:', e);
        console.error('[DEBUG] Plugin config_schema:', expandedPlugin.config_schema);
      }
    }
  }, [expandedPlugin, oauthConnection?.connected, oauthTokens]);

  // Filter and search plugins
  useEffect(() => {
    let filtered = plugins;

    // Filter out unavailable plugins
    filtered = filtered.filter(plugin => plugin.status !== 'unavailable');

    // Filter by type
    if (selectedType !== 'all') {
      if (selectedType === 'system') {
        // Include both system and external plugins in "Native" category
        filtered = filtered.filter(plugin => plugin.type === 'system' || plugin.type === 'external');
      } else {
        filtered = filtered.filter(plugin => plugin.type === selectedType);
      }
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(plugin => 
        plugin.name.toLowerCase().includes(query) ||
        plugin.description.toLowerCase().includes(query) ||
        plugin.author.toLowerCase().includes(query)
      );
    }

    setFilteredPlugins(filtered);
  }, [plugins, searchQuery, selectedType]);

  const handleCreateInstance = async (plugin: Plugin) => {
    try {
      setCreateLoading(true);
      setCreateError(null);
      setCreateSuccess(null);

      const response = await fetch('/api/plugin-instances', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          definition_id: plugin.id,
          definition_type: plugin.type,
          name: instanceName.trim(),
          refresh_interval: plugin.requires_processing ? instanceRefreshRate : undefined,
          settings: instanceSettings,
        }),
      });

      if (response.ok) {
        setCreateSuccess('Plugin instance created successfully!');
        
        // Close modal and navigate back after a short delay
        setTimeout(() => {
          handleCloseExpanded();
          navigate('/?tab=plugins&subtab=instances');
        }, 1500);
      } else {
        const errorData = await response.json();
        setCreateError(errorData.error || 'Failed to create plugin instance');
      }
    } catch (error) {
      setCreateError('Network error occurred while creating instance');
    } finally {
      setCreateLoading(false);
    }
  };

  const handleCardClick = (plugin: Plugin) => {
    setExpandedPlugin(plugin);
    
    // Initialize form state
    setInstanceName(plugin.name);
    setInstanceRefreshRate(86400); // Default refresh rate (daily)
    setCreateError(null);
    setCreateSuccess(null);
    
    // Initialize plugin settings from schema defaults
    try {
      if (plugin.config_schema) {
        const schema = JSON.parse(plugin.config_schema);
        const defaults: Record<string, any> = {};
        
        if (schema.properties) {
          Object.keys(schema.properties).forEach(key => {
            const property = schema.properties[key];
            if (property.default !== undefined) {
              defaults[key] = property.default;
            }
          });
        }
        
        setInstanceSettings(defaults);
      } else {
        setInstanceSettings({});
      }
    } catch (e) {
      setInstanceSettings({});
    }
  };

  const handleCloseExpanded = () => {
    setExpandedPlugin(null);
  };

  // Helper function to check if a plugin has configuration fields
  const hasConfigurationFields = (plugin: Plugin): boolean => {
    try {
      const schema = JSON.parse(plugin.config_schema);
      const properties = schema.properties || {};
      return Object.keys(properties).length > 0;
    } catch (e) {
      return false;
    }
  };

  // Helper function to validate required configuration fields
  const validateRequiredConfigFields = (plugin: Plugin, settings: Record<string, any>): boolean => {
    try {
      const schema = JSON.parse(plugin.config_schema);
      const required = schema.required || [];
      
      return required.every(fieldName => {
        const fieldValue = settings[fieldName];
        return fieldValue !== undefined && fieldValue !== null && 
               (typeof fieldValue !== 'string' || fieldValue.trim() !== '');
      });
    } catch (e) {
      return true; // If schema parsing fails, assume valid
    }
  };

  const renderSettingsForm = (plugin: Plugin, settings: Record<string, any>, onChange: (key: string, value: any) => void) => {
    let schema;
    try {
      schema = JSON.parse(plugin.config_schema);
    } catch (e) {
      return <div className="text-muted-foreground">Invalid schema configuration</div>;
    }
    
    const properties = schema.properties || {};
    const required = schema.required || [];
    
    return (
      <div className="space-y-4">
        {Object.keys(properties).map((key) => {
          const prop = properties[key];
          const value = settings[key] || prop.default || "";
          const isRequired = required.includes(key);
          
          // Handle dynamic select fields FIRST
          if (prop.dynamic && (prop.dynamicSource || prop.dynamic_source)) {
            const dynamicSourceField = prop.dynamicSource || prop.dynamic_source;
            // Use the same identifier logic as during storage
            const pluginIdentifier = expandedPlugin.identifier || expandedPlugin.id;
            const fieldKey = `${pluginIdentifier}.${dynamicSourceField}`;
            const options = dynamicFieldOptions[fieldKey] || [];
            const isLoading = dynamicFieldsLoading[fieldKey] || false;

            console.log(`[DEBUG] Rendering dynamic field ${key}:`, {
              pluginIdentifier,
              dynamicSourceField,
              fieldKey,
              optionsCount: options.length,
              allKeys: Object.keys(dynamicFieldOptions),
              isLoading
            });

            // Check if OAuth is required but not connected
            if ((prop.dependsOn === 'oauth_connected' || prop.depends_on === 'oauth_connected')) {
              // For now, show placeholder - OAuth status will be handled by connection listener
              if (options.length === 0 && !isLoading) {
                return (
                  <div key={key}>
                    <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                      {prop.title || key}
                      {isRequired && <span className="text-red-500 ml-1">*</span>}
                    </Label>
                    <Select disabled>
                      <SelectTrigger className="mt-2">
                        <SelectValue placeholder="Connect to OAuth first to load options" />
                      </SelectTrigger>
                    </Select>
                    {prop.description && (
                      <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                    )}
                  </div>
                );
              }
            }

            // Render dynamic select with fetched options
            return (
              <div key={key}>
                <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                  {prop.title || key}
                  {isRequired && <span className="text-red-500 ml-1">*</span>}
                </Label>
                <Select
                  value={Array.isArray(value) ? value[0] : value}
                  onValueChange={(val) => onChange(key, prop.multiple ? [val] : val)}
                  disabled={isLoading}
                >
                  <SelectTrigger className="mt-2">
                    <SelectValue placeholder={isLoading ? "Loading..." : (prop.placeholder || "Select option")} />
                  </SelectTrigger>
                  <SelectContent>
                    {options.map((option: any) => {
                      // Handle different option formats
                      const optionValue = typeof option === 'object' ? (option.value || Object.values(option)[0]) : option;
                      const optionLabel = typeof option === 'object' ? (option.label || Object.keys(option)[0]) : option;

                      return (
                        <SelectItem key={optionValue} value={optionValue}>
                          {optionLabel}
                        </SelectItem>
                      );
                    })}
                  </SelectContent>
                </Select>
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                )}
              </div>
            );
          }

          // Handle enum (select dropdown) SECOND - after dynamic fields
          if (prop.enum && Array.isArray(prop.enum)) {
            const enumNames = prop.enumNames && Array.isArray(prop.enumNames) ? prop.enumNames : prop.enum;
            return (
              <div key={key}>
                <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                  {prop.title || key}
                  {isRequired && <span className="text-red-500 ml-1">*</span>}
                </Label>
                <Select value={value} onValueChange={(val) => onChange(key, val)}>
                  <SelectTrigger className="mt-2">
                    <SelectValue placeholder={prop.description || "Select an option"} />
                  </SelectTrigger>
                  <SelectContent>
                    {prop.enum.map((option: any, index: number) => (
                      <SelectItem key={option} value={option}>
                        {enumNames[index] || option}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                )}
              </div>
            );
          }
          
          // Handle boolean type
          if (prop.type === "boolean") {
            return (
              <div key={key}>
                <div className="flex items-center space-x-2 mt-2">
                  <input
                    type="checkbox"
                    id={key}
                    checked={value || false}
                    onChange={(e) => onChange(key, e.target.checked)}
                    className="rounded border-gray-300"
                  />
                  <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                    {prop.title || key}
                    {isRequired && <span className="text-red-500 ml-1">*</span>}
                  </Label>
                </div>
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                )}
              </div>
            );
          }
          
          // Handle number type
          if (prop.type === "number" || prop.type === "integer") {
            return (
              <div key={key}>
                <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                  {prop.title || key}
                  {isRequired && <span className="text-red-500 ml-1">*</span>}
                </Label>
                <Input
                  id={key}
                  type="number"
                  value={value}
                  onChange={(e) => onChange(key, prop.type === "integer" ? parseInt(e.target.value) || 0 : parseFloat(e.target.value) || 0)}
                  placeholder={prop.description}
                  className="mt-2"
                  min={prop.minimum}
                  max={prop.maximum}
                  required={isRequired}
                />
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                )}
              </div>
            );
          }
          
          // Handle textarea type
          if (prop.format === "textarea") {
            return (
              <div key={key}>
                <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                  {prop.title || key}
                  {isRequired && <span className="text-red-500 ml-1">*</span>}
                </Label>
                <textarea
                  id={key}
                  value={value}
                  onChange={(e) => onChange(key, e.target.value)}
                  placeholder={prop.placeholder || prop.description}
                  className="mt-2 min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  required={isRequired}
                  rows={4}
                />
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                )}
              </div>
            );
          }

          // Handle date type
          if (prop.type === "string" && prop.format === "date") {
            return (
              <div key={key}>
                <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                  {prop.title || key}
                  {isRequired && <span className="text-red-500 ml-1">*</span>}
                </Label>
                <Input
                  id={key}
                  type="date"
                  value={value}
                  onChange={(e) => onChange(key, e.target.value)}
                  className="mt-2"
                  required={isRequired}
                />
                {prop.description && (
                  <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
                )}
              </div>
            );
          }
          
          // Default to string type
          return (
            <div key={key}>
              <Label htmlFor={key} className={isRequired ? "font-medium" : ""}>
                {prop.title || key}
                {isRequired && <span className="text-red-500 ml-1">*</span>}
              </Label>
              <Input
                id={key}
                value={value}
                onChange={(e) => onChange(key, e.target.value)}
                placeholder={prop.description}
                className="mt-2"
                required={isRequired}
              />
              {prop.description && (
                <p className="text-xs text-muted-foreground mt-1">{prop.description}</p>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  // Handle escape key to close expanded view
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && expandedPlugin) {
        handleCloseExpanded();
      }
    };

    if (expandedPlugin) {
      document.addEventListener('keydown', handleKeyDown);
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = 'unset';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [expandedPlugin]);

  const getPluginTypeBadge = (type: string) => {
    if (type === 'system' || type === 'external') {
      return <Badge variant="outline">Native</Badge>;
    }
    return <Badge variant="outline">Private</Badge>;
  };

  return (
    <div className="min-h-screen">
      {/* Sticky Header */}
      <div className="sticky top-0 z-40 border-b bg-background">
        <div className="container mx-auto px-4 py-4 space-y-4">
          {/* Breadcrumb */}
          <div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/?tab=plugins&subtab=instances')}
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              Back to Plugin Management
            </Button>
          </div>
          
          {/* Title and Subtitle */}
          <div>
            <h1 className="text-2xl font-semibold">Add Plugin Instance</h1>
            <p className="text-muted-foreground">Choose a plugin to create an instance</p>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="container mx-auto px-4 py-6 space-y-6">
          {/* Search and Filter Controls */}
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search plugins by name, description, or author..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={selectedType === 'all' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedType('all')}
                className="w-[60px]"
              >
                All
              </Button>
              <Button
                variant={selectedType === 'system' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedType('system')}
                className="w-[70px]"
              >
                Native
              </Button>
              <Button
                variant={selectedType === 'private' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedType('private')}
                className="w-[70px]"
              >
                Private
              </Button>
            </div>
          </div>

          {/* Plugin Grid */}
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-muted-foreground">Loading plugins...</div>
            </div>
          ) : filteredPlugins.length === 0 ? (
            <Card className="py-12">
              <CardContent className="text-center">
                <Puzzle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">
                  {searchQuery || selectedType !== 'all' ? 'No matching plugins' : 'No plugins available'}
                </h3>
                <p className="text-muted-foreground">
                  {searchQuery || selectedType !== 'all' 
                    ? 'Try adjusting your search terms or filters'
                    : 'No plugins have been installed yet'
                  }
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
              {filteredPlugins.map((plugin) => (
                <Card 
                  key={plugin.id} 
                  className="cursor-pointer hover:shadow-md transition-all duration-200 hover:scale-[1.02]"
                  onClick={() => handleCardClick(plugin)}
                >
                  <CardContent className="px-3">
                    <div className="space-y-1">
                      {/* Line 1: Title + Badge on same line */}
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="text-base font-semibold leading-tight flex-1 min-w-0">
                          {plugin.name}
                        </h3>
                        {getPluginTypeBadge(plugin.type)}
                      </div>
                      
                      {/* Line 2: Author */}
                      <p className="text-sm text-muted-foreground">
                        by {plugin.author}
                      </p>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>

      {/* Expanded Plugin Modal */}
      {expandedPlugin && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          {/* Backdrop */}
          <div 
            className="absolute inset-0 bg-background/80 backdrop-blur-sm"
            onClick={handleCloseExpanded}
          />
          
          {/* Expanded Card */}
          <Card className="relative z-10 w-full max-w-4xl max-h-[90vh] overflow-y-auto mx-auto animate-in fade-in-0 zoom-in-95 duration-300">
            <CardHeader className="pb-4">
              <div className="flex items-start justify-between gap-4">
                <CardTitle className="text-xl font-semibold">
                  {expandedPlugin.name}
                </CardTitle>
                {getPluginTypeBadge(expandedPlugin.type)}
              </div>
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">
                  by {expandedPlugin.author}
                </p>
                <p className="text-xs text-muted-foreground">
                  Version {expandedPlugin.version}
                </p>
              </div>
            </CardHeader>
            
            <CardContent className="space-y-6">
              {/* Error and Success Messages */}
              {createError && (
                <Alert variant="destructive">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>{createError}</AlertDescription>
                </Alert>
              )}

              {createSuccess && (
                <Alert>
                  <CheckCircle className="h-4 w-4" />
                  <AlertDescription>{createSuccess}</AlertDescription>
                </Alert>
              )}

              {/* Description */}
              <div>
                <div 
                  className="text-sm text-muted-foreground leading-relaxed"
                  dangerouslySetInnerHTML={{
                    __html: expandedPlugin.description || "No description available."
                  }}
                />
              </div>

              {/* OAuth Connection */}
              {expandedPlugin.oauth_config && (
                <>
                  <Separator />
                  <div>
                    <h3 className="text-base font-semibold mb-3">
                      Authentication
                    </h3>
                    <OAuthConnection
                      oauthConfig={expandedPlugin.oauth_config}
                      onConnectionChange={(connected, tokens) => {
                        console.log('[DEBUG] OAuthConnection onConnectionChange:', {
                          connected,
                          hasTokens: !!tokens,
                          tokenKeys: Object.keys(tokens || {}),
                          tokens
                        });

                        if (connected && tokens) {
                          // Store tokens in state
                          setOauthTokens(tokens);

                          // Fetch dynamic field options when OAuth connects
                          const schema = JSON.parse(expandedPlugin.config_schema);
                          const properties = schema.properties || {};

                          Object.keys(properties).forEach(key => {
                            const prop = properties[key];
                            const dynamicSourceField = prop.dynamicSource || prop.dynamic_source;
                            if (prop.dynamic && dynamicSourceField) {
                              // Use the identifier field from plugin metadata
                              const pluginIdentifier = expandedPlugin.identifier || expandedPlugin.id;

                              console.log(`[DEBUG] Fetching options for ${key} on OAuth connect (identifier: ${pluginIdentifier})`);
                              fetchDynamicFieldOptions(pluginIdentifier, dynamicSourceField, tokens);
                            }
                          });
                        } else if (!connected) {
                          // Clear tokens when disconnected
                          setOauthTokens(null);
                        }
                      }}
                    />
                  </div>
                </>
              )}

              {/* Instance Name */}
              <div>
                <Label htmlFor="instance-name">Instance Name</Label>
                <Input
                  id="instance-name"
                  value={instanceName}
                  onChange={(e) => setInstanceName(e.target.value)}
                  placeholder="Enter instance name"
                  className="mt-2"
                />
              </div>

              {/* Refresh Rate */}
              {expandedPlugin.requires_processing && (
                <div>
                  <Label htmlFor="refresh-rate">Refresh Rate</Label>
                  <Select 
                    value={instanceRefreshRate.toString()} 
                    onValueChange={(value) => setInstanceRefreshRate(Number(value))}
                  >
                    <SelectTrigger className="mt-2">
                      <SelectValue placeholder="Select refresh rate" />
                    </SelectTrigger>
                    <SelectContent>
                      {refreshRateOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value.toString()}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {/* Plugin Configuration */}
              {hasConfigurationFields(expandedPlugin) && (
                <>
                  <Separator />
                  <div>
                    <h3 className="text-base font-semibold flex items-center gap-2 mb-4">
                      <SettingsIcon className="h-4 w-4" />
                      Plugin Configuration
                    </h3>
                    {renderSettingsForm(expandedPlugin, instanceSettings, (key, value) => {
                      setInstanceSettings(prev => ({ ...prev, [key]: value }));
                    })}
                  </div>
                </>
              )}
              
              <div className="flex gap-3">
                <Button
                  variant="outline"
                  onClick={handleCloseExpanded}
                  className="flex-1"
                  disabled={createLoading}
                >
                  Cancel
                </Button>
                <Button
                  onClick={() => handleCreateInstance(expandedPlugin)}
                  className="flex-1"
                  disabled={
                    createLoading || 
                    !instanceName.trim() || 
                    (expandedPlugin.requires_processing && (!instanceRefreshRate || instanceRefreshRate <= 0)) ||
                    !validateRequiredConfigFields(expandedPlugin, instanceSettings)
                  }
                >
                  {createLoading ? "Creating..." : "Create Instance"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
