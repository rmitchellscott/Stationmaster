import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
// Removed unused dialog imports
import {
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import {
  SquareIcon,
  ColumnsIcon,
  RowsIcon,
  Grid2x2Icon,
  LayersIcon,
  AlertTriangle,
  CheckCircle,
  Code2,
  Database,
  Globe,
  Webhook,
  Plus,
  Trash2,
  Eye,
  Shield,
  Loader2,
  HelpCircle,
} from "lucide-react";
import { LiquidEditor } from "./LiquidEditor";
import { PluginPreview } from "./PluginPreview";
import { PrivatePluginHelp } from "./PrivatePluginHelp";
import { FormFieldBuilder } from "./FormFieldBuilder";

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
  polling_config?: PollingConfig;
  form_fields?: FormFieldConfig;
  sample_data?: any;
  version: string;
  remove_bleed_margin?: boolean;
  enable_dark_mode?: boolean;
}

interface PollingConfig {
  urls: URLConfig[];
  interval: number;
  timeout: number;
  max_size: number;
  user_agent: string;
  retry_count: number;
}

interface URLConfig {
  url: string;
  method: string;
  headers: string; // Headers in TRMNL format: key=value&key2=value2
  body: string;    // JSON body for POST requests
}

interface FormFieldConfig {
  yaml?: string; // YAML string for form field definitions
}

interface LayoutTab {
  id: 'shared' | 'full' | 'half_vertical' | 'half_horizontal' | 'quadrant';
  label: string;
  icon: React.ReactNode;
  description: string;
}

interface PrivatePluginCreatorProps {
  plugin?: PrivatePlugin;
  onSave: (plugin: PrivatePlugin) => void;
  onCancel: () => void;
  saving?: boolean;
}

const layoutTabs: LayoutTab[] = [
  {
    id: 'full',
    label: 'Full Screen',
    icon: <SquareIcon className="h-4 w-4" />,
    description: 'Full 800x480 display'
  },
  {
    id: 'half_horizontal',
    label: 'Half Horizontal',
    icon: <RowsIcon className="h-4 w-4" />,
    description: 'Top or bottom half (800x240)'
  },
  {
    id: 'half_vertical',
    label: 'Half Vertical',
    icon: <ColumnsIcon className="h-4 w-4" />,
    description: 'Left or right half (400x480)'
  },
  {
    id: 'quadrant',
    label: 'Quadrant',
    icon: <Grid2x2Icon className="h-4 w-4" />,
    description: 'Quarter screen (400x240)'
  },
  {
    id: 'shared',
    label: 'Shared Markup',
    icon: <LayersIcon className="h-4 w-4" />,
    description: 'Markup prepended to all layouts'
  }
];

export function PrivatePluginCreator({ 
  plugin, 
  onSave, 
  onCancel,
  saving = false
}: PrivatePluginCreatorProps) {
  const { t } = useTranslation();
  
  // Form state
  const [formData, setFormData] = useState<PrivatePlugin>({
    name: '',
    description: '',
    markup_full: '',
    markup_half_vert: '',
    markup_half_horiz: '',
    markup_quadrant: '',
    shared_markup: '',
    data_strategy: 'webhook',
    version: '1.0.0',
    remove_bleed_margin: false,
    enable_dark_mode: false,
  });

  // UI state
  const [activeLayoutTab, setActiveLayoutTab] = useState<string>('full');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [validating, setValidating] = useState(false);
  const [validationResults, setValidationResults] = useState<{
    valid: boolean;
    message: string;
    warnings: string[];
    errors: string[];
  } | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [showHelp, setShowHelp] = useState(false);

  // Polling configuration state
  const [pollingUrls, setPollingUrls] = useState<URLConfig[]>([]);

  // Form field state
  const [formFieldsYAML, setFormFieldsYAML] = useState<string>('');
  const [formFieldsValid, setFormFieldsValid] = useState<boolean>(true);
  const [formFieldsErrors, setFormFieldsErrors] = useState<string[]>([]);
  
  // Static data state
  const [staticDataJSON, setStaticDataJSON] = useState<string>('{}');

  // Helper function to convert headers to TRMNL string format
  const convertHeadersToString = (headers: any): string => {
    if (typeof headers === 'string') {
      return headers;
    }
    if (typeof headers === 'object' && headers !== null) {
      // Convert object to string format for backward compatibility
      return Object.entries(headers)
        .map(([key, value]) => `${key}=${value}`)
        .join('&');
    }
    return '';
  };

  // Initialize form with existing plugin data
  useEffect(() => {
    if (plugin) {
      setFormData(plugin);
      if (plugin.polling_config?.urls) {
        // Convert headers to string format
        const convertedUrls = plugin.polling_config.urls.map(url => ({
          ...url,
          headers: convertHeadersToString(url.headers)
        }));
        setPollingUrls(convertedUrls);
      }
      if (plugin.form_fields?.yaml) {
        setFormFieldsYAML(plugin.form_fields.yaml);
      }
      if (plugin.sample_data) {
        setStaticDataJSON(JSON.stringify(plugin.sample_data, null, 2));
      }
    } else {
      // Reset form for new plugin
      setFormData({
        name: '',
        description: '',
        markup_full: '',
        markup_half_vert: '',
        markup_half_horiz: '',
        markup_quadrant: '',
        shared_markup: '',
        data_strategy: 'webhook',
        version: '1.0.0',
      });
      setPollingUrls([]);
      setFormFieldsYAML('');
      setFormFieldsErrors([]);
      setFormFieldsValid(true);
      setStaticDataJSON('{}');
    }
  }, [plugin]);

  const handleInputChange = (field: keyof PrivatePlugin, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleLayoutMarkupChange = (layoutId: string, value: string) => {
    const field = layoutId === 'shared' ? 'shared_markup' : 
                  layoutId === 'full' ? 'markup_full' :
                  layoutId === 'half_vertical' ? 'markup_half_vert' :
                  layoutId === 'half_horizontal' ? 'markup_half_horiz' :
                  'markup_quadrant';
    
    handleInputChange(field as keyof PrivatePlugin, value);
  };

  const getLayoutMarkup = (layoutId: string) => {
    switch (layoutId) {
      case 'shared': return formData.shared_markup;
      case 'full': return formData.markup_full;
      case 'half_vertical': return formData.markup_half_vert;
      case 'half_horizontal': return formData.markup_half_horiz;
      case 'quadrant': return formData.markup_quadrant;
      default: return '';
    }
  };

  const addPollingURL = () => {
    setPollingUrls(prev => [...prev, {
      url: '',
      method: 'GET',
      headers: '',
      body: ''
    }]);
  };


  const removePollingURL = (index: number) => {
    setPollingUrls(prev => prev.filter((_, i) => i !== index));
  };

  const updatePollingURL = (index: number, field: keyof URLConfig, value: any) => {
    setPollingUrls(prev => prev.map((url, i) => 
      i === index ? { ...url, [field]: value } : url
    ));
  };

  const validateTemplates = async () => {
    try {
      setValidating(true);
      setError(null);
      setValidationResults(null);

      const response = await fetch('/api/plugin-definitions/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          name: formData.name,
          description: formData.description,
          markup_full: formData.markup_full,
          markup_half_vert: formData.markup_half_vert,
          markup_half_horiz: formData.markup_half_horiz,
          markup_quadrant: formData.markup_quadrant,
          shared_markup: formData.shared_markup,
          data_strategy: formData.data_strategy,
          polling_config: formData.polling_config,
          form_fields: formData.form_fields,
          version: formData.version,
          plugin_type: 'private',
        }),
      });

      if (response.ok) {
        const result = await response.json();
        setValidationResults(result);
        if (result.valid) {
          setSuccess('Templates validated successfully!');
        }
      } else {
        const errorData = await response.json();
        setError(errorData.error || 'Failed to validate templates');
      }
    } catch (error) {
      setError('Network error occurred during validation');
    } finally {
      setValidating(false);
    }
  };

  const handleSave = async () => {
    try {
      setError(null);

      // Validate required fields
      if (!formData.name.trim()) {
        throw new Error("Plugin name is required");
      }

      if (!formData.markup_full.trim()) {
        throw new Error("At least the full screen layout template is required");
      }

      // Validate form fields if provided
      if (!formFieldsValid && formFieldsYAML.trim()) {
        throw new Error(`Form fields validation failed: ${formFieldsErrors.join(', ')}`);
      }

      // Prepare form data with polling config and form fields
      const submitData = { ...formData };
      
      if (formData.data_strategy === 'polling' && pollingUrls.length > 0) {
        submitData.polling_config = {
          urls: pollingUrls,
          interval: 300, // 5 minutes default
          timeout: 10,
          max_size: 1024 * 1024, // 1MB
          user_agent: "TRMNL-Private-Plugin/1.0",
          retry_count: 2
        };
      }

      // Include form fields only if there's actual YAML content
      // Send null for empty forms to be consistent with backend normalization
      const trimmedYAML = formFieldsYAML.trim();
      if (trimmedYAML) {
        submitData.form_fields = {
          yaml: trimmedYAML
        };
      } else {
        submitData.form_fields = null;
      }

      // Include static data if provided (for static data strategy)
      const trimmedStaticJSON = staticDataJSON.trim();
      if (trimmedStaticJSON && trimmedStaticJSON !== '{}') {
        try {
          const parsedStaticData = JSON.parse(trimmedStaticJSON);
          submitData.sample_data = parsedStaticData;
          console.log('[UI] Submitting static data:', parsedStaticData);
        } catch (e) {
          throw new Error(`Invalid JSON in Static Data field: ${e.message}`);
        }
      } else {
        submitData.sample_data = null;
        console.log('[UI] No static data provided, setting to null');
      }

      onSave(submitData);
      
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save plugin");
    }
  };

  const copyTemplate = (fromLayout: string, toLayout: string) => {
    const sourceMarkup = getLayoutMarkup(fromLayout);
    handleLayoutMarkupChange(toLayout, sourceMarkup);
  };

  const renderDataStrategyConfig = () => {
    switch (formData.data_strategy) {
      case 'webhook':
        return (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Webhook className="h-4 w-4" />
              Webhook Configuration
            </h3>
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                Webhook URLs are now available after creating plugin instances. 
                Go to Plugin Management → Plugin Instances to view webhook URLs for each instance.
              </AlertDescription>
            </Alert>
          </div>
        );

      case 'polling':
        return (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Globe className="h-4 w-4" />
              Polling Configuration
            </h3>
            {pollingUrls.map((urlConfig, index) => {
              const isValidUrl = !urlConfig.url || /^https?:\/\//.test(urlConfig.url);
              
              return (
                <Card key={index}>
                  <CardHeader className="pb-4">
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-base">URL {index + 1}</CardTitle>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => removePollingURL(index)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-6">
                        <div className="space-y-4">
                          <div>
                            <Label htmlFor={`url-${index}`}>URL *</Label>
                            <Input
                              id={`url-${index}`}
                              value={urlConfig.url}
                              onChange={(e) => updatePollingURL(index, 'url', e.target.value)}
                              placeholder="https://api.example.com/data"
                              className={`mt-2 ${!isValidUrl ? 'border-destructive' : ''}`}
                            />
                            {!isValidUrl && (
                              <p className="text-xs text-destructive mt-1">
                                URL must start with http:// or https://
                              </p>
                            )}
                          </div>

                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                              <Label htmlFor={`method-${index}`}>Method</Label>
                              <Select
                                value={urlConfig.method}
                                onValueChange={(value) => updatePollingURL(index, 'method', value)}
                              >
                                <SelectTrigger id={`method-${index}`} className="mt-2">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="GET">GET</SelectItem>
                                  <SelectItem value="POST">POST</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>
                            <div>
                              <Label htmlFor={`headers-${index}`}>Headers</Label>
                              <Input
                                id={`headers-${index}`}
                                value={urlConfig.headers}
                                onChange={(e) => updatePollingURL(index, 'headers', e.target.value)}
                                placeholder="authorization=bearer {{ api_key }}&content-type=application/json"
                                className="mt-2"
                              />
                              <p className="text-xs text-muted-foreground mt-1">
                                Format: key=value&key2=value2. Use {`{{ form_field_name }}`} for form field substitution.
                              </p>
                            </div>
                          </div>

                          {urlConfig.method === 'POST' && (
                            <div>
                              <Label htmlFor={`body-${index}`}>Request Body</Label>
                              <Textarea
                                id={`body-${index}`}
                                value={urlConfig.body}
                                onChange={(e) => updatePollingURL(index, 'body', e.target.value)}
                                placeholder='{"api_key": "{{ api_key }}", "query": "{{ search_term }}"}'
                                className="mt-2"
                                rows={3}
                              />
                              <p className="text-xs text-muted-foreground mt-1">
                                JSON body. Use {`{{ form_field_name }}`} to include form field values.
                              </p>
                            </div>
                          )}
                        </div>
                    </CardContent>
                  </Card>
                );
              })}
            <Button variant="outline" onClick={addPollingURL}>
              <Plus className="h-4 w-4 mr-2" />
              Add URL
            </Button>
          </div>
        );

      case 'static':
        return (
          <div className="space-y-4">
            <h3 className="text-lg font-semibold flex items-center gap-2">
              <Database className="h-4 w-4" />
              Static Configuration
            </h3>
            <div>
              <Label>Data Source</Label>
              <p className="text-sm text-muted-foreground mt-1">
                Uses only form fields and TRMNL device/user data - no external data sources
              </p>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  // Main content component
  const renderMainContent = () => (
    <>
      <div className="flex items-center justify-between mb-8">
        <div>
          <p className="text-sm text-muted-foreground">
            Create custom plugins using Liquid templates and TRMNL's design framework
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowHelp(true)}
          className="shrink-0"
        >
          <HelpCircle className="h-4 w-4 mr-2" />
          Help
        </Button>
      </div>

      <div className="space-y-8">
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

        {validationResults && !validationResults.valid && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              <div className="space-y-2">
                <div className="font-medium">Validation Failed</div>
                {validationResults.errors.map((error, index) => (
                  <div key={index} className="text-sm">• {error}</div>
                ))}
              </div>
            </AlertDescription>
          </Alert>
        )}

        {validationResults && validationResults.warnings.length > 0 && (
          <Alert variant="default">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              <div className="space-y-2">
                <div className="font-medium">Validation Warnings</div>
                {validationResults.warnings.map((warning, index) => (
                  <div key={index} className="text-sm">• {warning}</div>
                ))}
              </div>
            </AlertDescription>
          </Alert>
        )}

        {/* Basic Information */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold">Basic Information</h3>
          <div>
            <Label htmlFor="name">Plugin Name</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="My Awesome Plugin"
              className="mt-2"
            />
          </div>
          <div>
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              placeholder="Describe what your plugin does..."
              className="mt-2"
              rows={3}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="version">Version</Label>
              <Input
                id="version"
                value={formData.version}
                onChange={(e) => handleInputChange('version', e.target.value)}
                placeholder="1.0.0"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="data_strategy">Data Strategy</Label>
              <Select
                value={formData.data_strategy}
                onValueChange={(value) => handleInputChange('data_strategy', value as any)}
              >
                <SelectTrigger className="mt-2">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="webhook">Webhook</SelectItem>
                  <SelectItem value="polling">Polling</SelectItem>
                  <SelectItem value="static">Static</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>

        <Separator />

        {/* Data Strategy Configuration */}
        {renderDataStrategyConfig()}

        <Separator />

        {/* Layout Templates */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Code2 className="h-4 w-4" />
            Layout Templates
          </h3>
            <Tabs value={activeLayoutTab} onValueChange={setActiveLayoutTab}>
              <TabsList className="grid w-full grid-cols-5">
                {layoutTabs.map((tab) => (
                  <TabsTrigger key={tab.id} value={tab.id} className="flex items-center gap-2">
                    {tab.icon}
                    <span className="hidden sm:inline">{tab.label}</span>
                  </TabsTrigger>
                ))}
              </TabsList>

              {layoutTabs.map((tab) => (
                <TabsContent key={tab.id} value={tab.id} className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="font-semibold">{tab.label}</h3>
                      <p className="text-sm text-muted-foreground">{tab.description}</p>
                    </div>
                    <div className="flex gap-2">
                      {tab.id !== 'shared' && (
                        <Select onValueChange={(from) => copyTemplate(from, tab.id)}>
                          <SelectTrigger className="w-40">
                            <SelectValue placeholder="Copy from..." />
                          </SelectTrigger>
                          <SelectContent>
                            {layoutTabs
                              .filter(t => t.id !== tab.id)
                              .map(t => (
                                <SelectItem key={t.id} value={t.id}>
                                  {t.label}
                                </SelectItem>
                              ))}
                          </SelectContent>
                        </Select>
                      )}
                      <Button variant="outline" size="sm" onClick={() => setShowPreview(true)}>
                        <Eye className="h-4 w-4 mr-2" />
                        Preview
                      </Button>
                    </div>
                  </div>

                  <LiquidEditor
                    key={`${tab.id}-${plugin?.id || 'new'}`}
                    value={getLayoutMarkup(tab.id)}
                    onChange={(value) => handleLayoutMarkupChange(tab.id, value)}
                    placeholder={`Enter Liquid template for ${tab.label.toLowerCase()}...`}
                    height="400px"
                  />

                  <div className="text-xs text-muted-foreground">
                    <p>Available variables: data.*, trmnl.user.*, trmnl.device.*, layout.*</p>
                    <p>Uses TRMNL framework CSS classes. Templates are automatically wrapped with proper layout structure.</p>
                  </div>
                </TabsContent>
              ))}
            </Tabs>
        </div>

        <Separator />

        {/* Screen Options */}
        <div className="space-y-6">
          <h3 className="text-lg font-semibold">Screen Options</h3>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label htmlFor="remove-bleed-margin">Remove Bleed Margin</Label>
              <p className="text-sm text-muted-foreground">
                Removes default padding and margins, allowing content to extend to screen edges
              </p>
            </div>
            <Switch
              id="remove-bleed-margin"
              checked={formData.remove_bleed_margin || false}
              onCheckedChange={(checked) => handleInputChange('remove_bleed_margin', checked)}
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label htmlFor="enable-dark-mode">Enable Dark Mode</Label>
              <p className="text-sm text-muted-foreground">
                Inverts black/white pixels. Will modify entire screen. Add class "image" to img tags as needed.
              </p>
            </div>
            <Switch
              id="enable-dark-mode"
              checked={formData.enable_dark_mode || false}
              onCheckedChange={(checked) => handleInputChange('enable_dark_mode', checked)}
            />
          </div>
        </div>

        <Separator />

        {/* Plugin Settings Form */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Plugin Settings Form
          </h3>
          <p className="text-sm text-muted-foreground">
            Define form fields that users will fill out when configuring instances of your plugin. 
            These values will be available in your templates as merge variables.
          </p>
          
          {formFieldsErrors.length > 0 && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                Form fields validation failed. Please fix the errors before saving.
              </AlertDescription>
            </Alert>
          )}
          
          <FormFieldBuilder
            value={formFieldsYAML}
            onChange={setFormFieldsYAML}
            onValidationChange={(isValid: boolean, errors: string[]) => {
              setFormFieldsValid(isValid);
              setFormFieldsErrors(errors);
            }}
          />
        </div>

        {/* Static Data */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Database className="h-4 w-4" />
            Static Data
          </h3>
          <p className="text-sm text-muted-foreground">
            Define static data as JSON that will be available in your templates. 
            This is separate from form fields and contains fixed data for your plugin.
          </p>
          
          <div className="space-y-2">
            <Label htmlFor="static-data">Static Data (JSON)</Label>
            <Textarea
              id="static-data"
              value={staticDataJSON}
              onChange={(e) => setStaticDataJSON(e.target.value)}
              placeholder='{\n  "facts": [\n    "Sample fact 1",\n    "Sample fact 2"\n  ],\n  "config": {\n    "title": "My Plugin"\n  }\n}'
              rows={8}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              Provide JSON data that will be merged with form field values in your templates.
            </p>
          </div>
        </div>
      </div>

      {/* Action buttons */}
      <Separator className="my-6" />
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={onCancel} disabled={saving || validating}>
          Cancel
        </Button>
        <Button 
          variant="outline" 
          onClick={validateTemplates} 
          disabled={saving || validating}
          className="text-blue-600 border-blue-600 hover:bg-blue-50"
        >
          {validating ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin mr-2" />
              Validating...
            </>
          ) : (
            <>
              <Shield className="h-4 w-4 mr-2" />
              Validate Templates
            </>
          )}
        </Button>
        <Button onClick={handleSave} disabled={saving || validating}>
          {saving ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin mr-2" />
              {plugin ? "Updating..." : "Creating..."}
            </>
          ) : (
            plugin ? "Update Plugin" : "Create Plugin"
          )}
        </Button>
      </div>
    </>
  );

  return (
    <>
      {renderMainContent()}
      
      {/* Plugin Preview Dialog */}
      <PluginPreview
        plugin={formData}
        isOpen={showPreview}
        onClose={() => setShowPreview(false)}
      />

      {/* Help Dialog */}
      <PrivatePluginHelp
        isOpen={showHelp}
        onClose={() => setShowHelp(false)}
      />
    </>
  );
}
