import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
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
  Copy,
  Eye,
  Shield,
  Loader2,
  HelpCircle,
} from "lucide-react";
import { LiquidEditor } from "./LiquidEditor";
import { PluginPreview } from "./PluginPreview";
import { PrivatePluginHelp } from "./PrivatePluginHelp";

interface PrivatePlugin {
  id?: string;
  name: string;
  description: string;
  markup_full: string;
  markup_half_vert: string;
  markup_half_horiz: string;
  markup_quadrant: string;
  shared_markup: string;
  data_strategy: 'webhook' | 'polling' | 'merge';
  polling_config?: PollingConfig;
  form_fields?: FormFieldConfig;
  version: string;
  webhook_url?: string;
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
  headers: Record<string, string>;
  method: string;
  body: string;
  key: string;
}

interface FormFieldConfig {
  type: string;
  properties: Record<string, any>;
}

interface LayoutTab {
  id: 'shared' | 'full' | 'half_vertical' | 'half_horizontal' | 'quadrant';
  label: string;
  icon: React.ReactNode;
  description: string;
}

interface PrivatePluginCreatorProps {
  plugin?: PrivatePlugin;
  isOpen: boolean;
  onClose: () => void;
  onSave: (plugin: PrivatePlugin) => void;
  onCancel: () => void;
}

const layoutTabs: LayoutTab[] = [
  {
    id: 'shared',
    label: 'Shared Markup',
    icon: <LayersIcon className="h-4 w-4" />,
    description: 'Markup prepended to all layouts'
  },
  {
    id: 'full',
    label: 'Full Screen',
    icon: <SquareIcon className="h-4 w-4" />,
    description: 'Full 800x480 display'
  },
  {
    id: 'half_vertical',
    label: 'Half Vertical',
    icon: <ColumnsIcon className="h-4 w-4" />,
    description: 'Left or right half (400x480)'
  },
  {
    id: 'half_horizontal',
    label: 'Half Horizontal',
    icon: <RowsIcon className="h-4 w-4" />,
    description: 'Top or bottom half (800x240)'
  },
  {
    id: 'quadrant',
    label: 'Quadrant',
    icon: <Grid2x2Icon className="h-4 w-4" />,
    description: 'Quarter screen (400x240)'
  }
];

export function PrivatePluginCreator({ 
  plugin, 
  isOpen, 
  onClose, 
  onSave, 
  onCancel 
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
  });

  // UI state
  const [activeLayoutTab, setActiveLayoutTab] = useState<string>('shared');
  const [loading, setLoading] = useState(false);
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

  // Initialize form with existing plugin data
  useEffect(() => {
    if (plugin) {
      setFormData(plugin);
      if (plugin.polling_config?.urls) {
        setPollingUrls(plugin.polling_config.urls);
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
      headers: {},
      method: 'GET',
      body: '',
      key: ''
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

      const response = await fetch('/api/private-plugins/validate', {
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
      setLoading(true);
      setError(null);

      // Validate required fields
      if (!formData.name.trim()) {
        throw new Error("Plugin name is required");
      }

      if (!formData.markup_full.trim()) {
        throw new Error("At least the full screen layout template is required");
      }

      // Prepare form data with polling config if needed
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

      onSave(submitData);
      setSuccess("Private plugin saved successfully!");
      
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save plugin");
    } finally {
      setLoading(false);
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
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Webhook className="h-4 w-4" />
                Webhook Configuration
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <Label>Webhook URL</Label>
                  <div className="flex gap-2 mt-2">
                    <Input 
                      value={formData.webhook_url || "Will be generated after saving"}
                      readOnly
                      className="bg-muted"
                    />
                    <Button variant="outline" size="sm">
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">
                    POST JSON data to this URL (max 2KB)
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        );

      case 'polling':
        return (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Globe className="h-4 w-4" />
                Polling Configuration
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {pollingUrls.map((urlConfig, index) => (
                  <div key={index} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <Label>URL {index + 1}</Label>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => removePollingURL(index)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <Label>URL</Label>
                        <Input
                          value={urlConfig.url}
                          onChange={(e) => updatePollingURL(index, 'url', e.target.value)}
                          placeholder="https://api.example.com/data"
                          className="mt-1"
                        />
                      </div>
                      <div>
                        <Label>Data Key</Label>
                        <Input
                          value={urlConfig.key}
                          onChange={(e) => updatePollingURL(index, 'key', e.target.value)}
                          placeholder="api_data"
                          className="mt-1"
                        />
                      </div>
                      <div>
                        <Label>Method</Label>
                        <Select
                          value={urlConfig.method}
                          onValueChange={(value) => updatePollingURL(index, 'method', value)}
                        >
                          <SelectTrigger className="mt-1">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="GET">GET</SelectItem>
                            <SelectItem value="POST">POST</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  </div>
                ))}
                <Button variant="outline" onClick={addPollingURL}>
                  <Plus className="h-4 w-4 mr-2" />
                  Add URL
                </Button>
              </div>
            </CardContent>
          </Card>
        );

      case 'merge':
        return (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Database className="h-4 w-4" />
                Plugin Merge Configuration
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <Label>Source Plugins</Label>
                  <p className="text-sm text-muted-foreground mt-1">
                    Coming soon: Select other plugins to merge data from
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        );

      default:
        return null;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle>
                {plugin ? 'Edit Private Plugin' : 'Create Private Plugin'}
              </DialogTitle>
              <DialogDescription>
                Create custom plugins using Liquid templates and TRMNL's design framework
              </DialogDescription>
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
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          <div className="space-y-6">
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
            <Card>
              <CardHeader>
                <CardTitle>Basic Information</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
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
                        <SelectItem value="merge">Plugin Merge</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Data Strategy Configuration */}
            {renderDataStrategyConfig()}

            {/* Layout Templates */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Code2 className="h-4 w-4" />
                  Layout Templates
                </CardTitle>
              </CardHeader>
              <CardContent>
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
                        value={getLayoutMarkup(tab.id)}
                        onChange={(value) => handleLayoutMarkupChange(tab.id, value)}
                        placeholder={`Enter Liquid template for ${tab.label.toLowerCase()}...`}
                        height="400px"
                      />

                      <div className="text-xs text-muted-foreground">
                        <p>Available variables: `data.*`, `trmnl.user.*`, `trmnl.device.*`, `layout.*`, `instance_id`</p>
                        <p>Uses TRMNL framework CSS classes. Templates are automatically wrapped with proper layout structure.</p>
                      </div>
                    </TabsContent>
                  ))}
                </Tabs>
              </CardContent>
            </Card>
          </div>
        </div>

        <Separator />

        <DialogFooter>
          <Button variant="outline" onClick={onCancel} disabled={loading || validating}>
            Cancel
          </Button>
          <Button 
            variant="outline" 
            onClick={validateTemplates} 
            disabled={loading || validating}
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
          <Button onClick={handleSave} disabled={loading || validating}>
            {loading ? "Saving..." : (plugin ? "Update Plugin" : "Create Plugin")}
          </Button>
        </DialogFooter>
      </DialogContent>

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
    </Dialog>
  );
}