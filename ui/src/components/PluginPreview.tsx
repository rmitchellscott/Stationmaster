import React, { useState, useEffect, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import {
  SquareIcon,
  ColumnsIcon,
  RowsIcon,
  Grid2x2Icon,
  RefreshCw,
  Download,
  AlertTriangle,
  Loader2,
  Eye,
  Code2,
} from "lucide-react";

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
  author?: string;
  webhook_token?: string;
  remove_bleed_margin?: boolean;
  enable_dark_mode?: boolean;
}

interface PluginPreviewProps {
  plugin: PrivatePlugin;
  isOpen: boolean;
  onClose: () => void;
}

interface LayoutOption {
  id: 'full' | 'half_vertical' | 'half_horizontal' | 'quadrant';
  label: string;
  icon: React.ReactNode;
  width: number;
  height: number;
}

const layoutOptions: LayoutOption[] = [
  {
    id: 'full',
    label: 'Full Screen',
    icon: <SquareIcon className="h-4 w-4" />,
    width: 800,
    height: 480,
  },
  {
    id: 'half_vertical',
    label: 'Half Vertical',
    icon: <ColumnsIcon className="h-4 w-4" />,
    width: 400,
    height: 480,
  },
  {
    id: 'half_horizontal',
    label: 'Half Horizontal',
    icon: <RowsIcon className="h-4 w-4" />,
    width: 800,
    height: 240,
  },
  {
    id: 'quadrant',
    label: 'Quadrant',
    icon: <Grid2x2Icon className="h-4 w-4" />,
    width: 400,
    height: 240,
  },
];

const sampleData = {
  // Sample webhook/polling data
  weather: {
    temperature: "72°F",
    condition: "Sunny",
    humidity: "45%",
    wind: "8 mph"
  },
  news: {
    headline: "Sample News Headline",
    summary: "This is a sample news summary for preview purposes."
  },
  calendar: {
    next_event: "Team Meeting",
    time: "2:00 PM",
    location: "Conference Room A"
  },
  // TRMNL context data
  user: {
    first_name: "John",
    email: "john@example.com"
  },
  device: {
    name: "My TRMNL",
    width: 800,
    height: 480
  },
  timestamp: new Date().toLocaleString()
};

export function PluginPreview({ plugin, isOpen, onClose }: PluginPreviewProps) {
  const [selectedLayout, setSelectedLayout] = useState<string>('full');
  const [customData, setCustomData] = useState(JSON.stringify(sampleData, null, 2));
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'preview' | 'data'>('preview');
  const [savingSampleData, setSavingSampleData] = useState(false);
  const [sampleDataSaved, setSampleDataSaved] = useState(false);
  
  const iframeRef = useRef<HTMLIFrameElement>(null);

  // Get the current layout option
  const currentLayout = layoutOptions.find(l => l.id === selectedLayout) || layoutOptions[0];

  // Initialize sample data from plugin when plugin changes
  useEffect(() => {
    if (plugin) {
      console.log('[PluginPreview] Loading plugin:', plugin.name, 'sample_data:', plugin.sample_data);
      if (plugin.sample_data) {
        console.log('[PluginPreview] Using plugin sample data');
        setCustomData(JSON.stringify(plugin.sample_data, null, 2));
      } else {
        console.log('[PluginPreview] Using default sample data');
        setCustomData(JSON.stringify(sampleData, null, 2));
      }
      setSampleDataSaved(false);
    }
  }, [plugin]);

  // Generate preview when plugin, layout, or data changes
  useEffect(() => {
    if (isOpen && plugin) {
      generatePreview();
    }
  }, [isOpen, plugin, selectedLayout, customData]);

  const generatePreview = async () => {
    try {
      setLoading(true);
      setError(null);

      // Parse custom data
      let parsedData;
      try {
        parsedData = JSON.parse(customData);
      } catch (e) {
        throw new Error("Invalid JSON in sample data");
      }

      // Get the appropriate template for the selected layout
      let template = '';
      switch (selectedLayout) {
        case 'full':
          template = plugin.markup_full;
          break;
        case 'half_vertical':
          template = plugin.markup_half_vert;
          break;
        case 'half_horizontal':
          template = plugin.markup_half_horiz;
          break;
        case 'quadrant':
          template = plugin.markup_quadrant;
          break;
        default:
          template = plugin.markup_full;
      }

      if (!template.trim()) {
        throw new Error(`No template defined for ${currentLayout.label} layout`);
      }

      // Create test plugin data
      const testPlugin = {
        ...plugin,
        polling_config: plugin.polling_config || {},
        form_fields: plugin.form_fields || {},
      };

      // Call the test API endpoint
      const response = await fetch('/api/plugin-definitions/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          plugin: testPlugin,
          layout: selectedLayout,
          sample_data: parsedData,
          device_width: currentLayout.width,
          device_height: currentLayout.height,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      
      if (result.preview_url) {
        setPreviewUrl(result.preview_url);
      } else {
        throw new Error("No preview URL returned from server");
      }

    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to generate preview");
      setPreviewUrl(null);
    } finally {
      setLoading(false);
    }
  };

  const refreshPreview = () => {
    generatePreview();
  };

  const downloadPreview = () => {
    if (previewUrl && previewUrl.startsWith('data:')) {
      // Create download link for base64 image
      const link = document.createElement('a');
      link.href = previewUrl;
      link.download = `${plugin.name}_${selectedLayout}_preview.png`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
  };

  const resetSampleData = () => {
    setCustomData(JSON.stringify(sampleData, null, 2));
    setSampleDataSaved(false);
  };

  const saveSampleData = async () => {
    if (!plugin.id) {
      setError("Plugin must be saved before sample data can be saved");
      return;
    }

    try {
      console.log('[PluginPreview] Starting to save sample data for plugin:', plugin.id);
      setSavingSampleData(true);
      setError(null);

      // Parse custom data to validate it
      let parsedData;
      try {
        parsedData = JSON.parse(customData);
        console.log('[PluginPreview] Parsed sample data:', parsedData);
      } catch (e) {
        throw new Error("Invalid JSON in sample data");
      }

      // Update the plugin with sample data
      const response = await fetch(`/api/plugin-definitions/${plugin.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          name: plugin.name,
          description: plugin.description,
          version: plugin.version,
          author: plugin.author || '',
          markup_full: plugin.markup_full,
          markup_half_vert: plugin.markup_half_vert,
          markup_half_horiz: plugin.markup_half_horiz,
          markup_quadrant: plugin.markup_quadrant,
          shared_markup: plugin.shared_markup,
          data_strategy: plugin.data_strategy,
          polling_config: plugin.polling_config,
          form_fields: plugin.form_fields,
          sample_data: parsedData,
          remove_bleed_margin: plugin.remove_bleed_margin || false,
          enable_dark_mode: plugin.enable_dark_mode || false,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        console.error('[PluginPreview] Save failed:', response.status, errorData);
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }

      const responseData = await response.json();
      console.log('[PluginPreview] Save successful:', responseData);
      
      setSampleDataSaved(true);
      // Update the plugin object in memory
      plugin.sample_data = parsedData;
      console.log('[PluginPreview] Updated plugin object with sample data');
      
    } catch (err) {
      console.error('[PluginPreview] Save error:', err);
      setError(err instanceof Error ? err.message : "Failed to save sample data");
    } finally {
      setSavingSampleData(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent 
        className="sm:max-w-lg md:max-w-2xl lg:max-w-4xl xl:max-w-5xl max-h-[85vh] mobile-dialog-content overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh] flex flex-col"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Eye className="h-5 w-5" />
            Preview: {plugin.name}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-hidden">
          <Tabs value={activeTab} onValueChange={(tab) => setActiveTab(tab as any)}>
            <div className="flex justify-between items-center mb-4">
              <TabsList>
                <TabsTrigger value="preview">Preview</TabsTrigger>
                <TabsTrigger value="data">Sample Data</TabsTrigger>
              </TabsList>

              <div className="flex items-center gap-2">
                <Label>Layout:</Label>
                <Select value={selectedLayout} onValueChange={setSelectedLayout}>
                  <SelectTrigger className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {layoutOptions.map((layout) => (
                      <SelectItem key={layout.id} value={layout.id}>
                        <div className="flex items-center gap-2">
                          {layout.icon}
                          {layout.label}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                <Button variant="outline" size="sm" onClick={refreshPreview} disabled={loading}>
                  {loading ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4" />
                  )}
                </Button>

                {previewUrl && (
                  <Button variant="outline" size="sm" onClick={downloadPreview}>
                    <Download className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>

            <TabsContent value="preview" className="space-y-4">
              {error && (
                <Alert variant="destructive">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    <span>{currentLayout.label} Layout Preview</span>
                    <span className="text-sm font-normal text-muted-foreground">
                      {currentLayout.width} × {currentLayout.height}px
                    </span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex justify-center p-4 bg-muted/20 rounded-lg">
                    {loading ? (
                      <div className="flex items-center justify-center" style={{ 
                        width: Math.min(currentLayout.width, 600), 
                        height: Math.min(currentLayout.height, 360) 
                      }}>
                        <div className="text-center">
                          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-2" />
                          <p className="text-sm text-muted-foreground">Generating preview...</p>
                        </div>
                      </div>
                    ) : previewUrl ? (
                      <div 
                        className="border border-gray-300 bg-white"
                        style={{ 
                          width: Math.min(currentLayout.width, 600), 
                          height: Math.min(currentLayout.height, 360) 
                        }}
                      >
                        <img
                          src={previewUrl}
                          alt="Plugin Preview"
                          className="w-full h-full object-contain"
                          style={{ imageRendering: 'pixelated' }}
                        />
                      </div>
                    ) : (
                      <div className="flex items-center justify-center text-muted-foreground" style={{ 
                        width: Math.min(currentLayout.width, 600), 
                        height: Math.min(currentLayout.height, 360) 
                      }}>
                        <div className="text-center">
                          <Eye className="h-8 w-8 mx-auto mb-2" />
                          <p>Click Refresh to generate preview</p>
                        </div>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="data" className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Code2 className="h-4 w-4" />
                      Sample Data
                      {sampleDataSaved && (
                        <span className="text-xs text-green-600 font-normal">Saved</span>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <Button 
                        variant="outline" 
                        size="sm" 
                        onClick={saveSampleData}
                        disabled={savingSampleData}
                      >
                        {savingSampleData ? (
                          <>
                            <Loader2 className="h-4 w-4 animate-spin mr-2" />
                            Saving...
                          </>
                        ) : (
                          'Save Sample Data'
                        )}
                      </Button>
                      <Button variant="outline" size="sm" onClick={resetSampleData}>
                        Reset to Default
                      </Button>
                    </div>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <Label>JSON Data (will be available as `data.*` in templates)</Label>
                    <Textarea
                      value={customData}
                      onChange={(e) => {
                        setCustomData(e.target.value);
                        setSampleDataSaved(false);
                      }}
                      className="font-mono text-sm min-h-[300px]"
                      placeholder="Enter JSON data for testing templates..."
                    />
                    <div className="text-xs text-muted-foreground">
                      <p>This data will be available in your templates as Liquid variables.</p>
                      <p>Example: `{"{"} "weather": {"{"} "temp": "72°F" {"}"} {"}"}` → `{"{"}{" "} data.weather.temp {"}"}`</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </DialogContent>
    </Dialog>
  );
}