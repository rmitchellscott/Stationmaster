import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  SquareIcon,
  ColumnsIcon,
  RowsIcon,
  Grid2x2Icon,
  AlertTriangle,
  CheckCircle,
  Edit,
  Trash2,
  Plus,
  Eye,
  Code2,
  Webhook,
  Globe,
  Database,
  RefreshCw,
  Download,
  Upload,
} from "lucide-react";

interface PrivatePlugin {
  id: string;
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
  version: string;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

interface PrivatePluginListProps {
  onCreatePlugin: () => void;
  onEditPlugin: (plugin: PrivatePlugin) => void;
  onPreviewPlugin: (plugin: PrivatePlugin) => void;
}

export function PrivatePluginList({ 
  onCreatePlugin, 
  onEditPlugin, 
  onPreviewPlugin 
}: PrivatePluginListProps) {
  const { t } = useTranslation();
  const [plugins, setPlugins] = useState<PrivatePlugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Delete confirmation dialog state
  const [deleteDialog, setDeleteDialog] = useState<{
    isOpen: boolean;
    plugin: PrivatePlugin | null;
  }>({ isOpen: false, plugin: null });

  // Import dialog state
  const [importDialog, setImportDialog] = useState<{
    isOpen: boolean;
    isUploading: boolean;
  }>({ isOpen: false, isUploading: false });

  const fetchPrivatePlugins = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/plugin-definitions?plugin_type=private", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setPlugins(data.plugins || []);
      } else {
        setError("Failed to fetch private plugins");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const deletePrivatePlugin = async (pluginId: string) => {
    try {
      setError(null);
      const response = await fetch(`/api/plugin-definitions/${pluginId}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        setSuccess("Private plugin deleted successfully");
        await fetchPrivatePlugins();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete private plugin");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };


  const exportPlugin = async (pluginId: string) => {
    try {
      setError(null);
      const response = await fetch(`/api/plugin-definitions/${pluginId}/export`, {
        method: "GET",
        credentials: "include",
      });

      if (response.ok) {
        // Get filename from Content-Disposition header or create a default
        const contentDisposition = response.headers.get('Content-Disposition');
        let filename = 'plugin.zip';
        if (contentDisposition) {
          const filenameMatch = contentDisposition.match(/filename="([^"]+)"/);
          if (filenameMatch) {
            filename = filenameMatch[1];
          }
        }

        // Create blob and download
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);

        setSuccess("Plugin exported successfully!");
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to export plugin");
      }
    } catch (error) {
      setError("Network error occurred while exporting plugin");
    }
  };

  const importPlugin = async (file: File) => {
    try {
      setError(null);
      setImportDialog(prev => ({ ...prev, isUploading: true }));

      const formData = new FormData();
      formData.append('file', file);

      const response = await fetch('/api/plugin-definitions/import', {
        method: 'POST',
        credentials: 'include',
        body: formData,
      });

      if (response.ok) {
        const data = await response.json();
        setSuccess(`Plugin "${data.plugin.name}" imported successfully!`);
        setImportDialog({ isOpen: false, isUploading: false });
        await fetchPrivatePlugins(); // Refresh the list
      } else {
        const errorData = await response.json();
        let errorMessage = errorData.error || "Failed to import plugin";
        
        // Add detailed error information if available
        if (errorData.details) {
          errorMessage += ": " + errorData.details;
        }
        
        // Add validation errors if available
        if (errorData.validation_errors && errorData.validation_errors.length > 0) {
          errorMessage += "\n\nValidation errors:\n" + errorData.validation_errors.join("\n");
        }
        
        // Add validation warnings if available
        if (errorData.validation_warnings && errorData.validation_warnings.length > 0) {
          errorMessage += "\n\nValidation warnings:\n" + errorData.validation_warnings.join("\n");
        }
        
        setError(errorMessage);
      }
    } catch (error) {
      setError("Network error occurred while importing plugin");
    } finally {
      setImportDialog(prev => ({ ...prev, isUploading: false }));
    }
  };

  const handleImportFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      if (file.type === 'application/zip' || file.name.endsWith('.zip')) {
        importPlugin(file);
      } else {
        setError("Please select a valid ZIP file");
      }
    }
  };

  const getDataStrategyIcon = (strategy: string) => {
    switch (strategy) {
      case 'webhook':
        return <Webhook className="h-4 w-4" />;
      case 'polling':
        return <Globe className="h-4 w-4" />;
      case 'static':
        return <Database className="h-4 w-4" />;
      default:
        return <Code2 className="h-4 w-4" />;
    }
  };

  const getDataStrategyBadge = (strategy: string) => {
    // Handle null/undefined strategy
    const safeStrategy = strategy || 'webhook';
    
    return (
      <Badge variant="outline">
        <div className="flex items-center gap-1">
          {getDataStrategyIcon(safeStrategy)}
          {safeStrategy.charAt(0).toUpperCase() + safeStrategy.slice(1)}
        </div>
      </Badge>
    );
  };

  const getAvailableLayouts = (plugin: PrivatePlugin) => {
    const layouts = [];
    if (plugin.markup_full) layouts.push('Full');
    if (plugin.markup_half_vert) layouts.push('Half V');
    if (plugin.markup_half_horiz) layouts.push('Half H');
    if (plugin.markup_quadrant) layouts.push('Quad');
    return layouts;
  };

  useEffect(() => {
    fetchPrivatePlugins();
  }, []);

  useEffect(() => {
    if (success) {
      const timer = setTimeout(() => setSuccess(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [success]);

  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error]);

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <div className="whitespace-pre-wrap">{error}</div>
          </AlertDescription>
        </Alert>
      )}

      {success && (
        <Alert>
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}

      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-semibold">Private Plugins</h3>
          <p className="text-muted-foreground">
            Custom plugins you've created with Liquid templates
          </p>
        </div>
        <div className="flex gap-2">
          <div className="relative">
            <Input
              id="import-file"
              type="file"
              accept=".zip"
              onChange={handleImportFileChange}
              className="sr-only"
              disabled={importDialog.isUploading}
            />
            <Button 
              variant="outline" 
              onClick={() => document.getElementById('import-file')?.click()}
              disabled={importDialog.isUploading}
            >
              <Upload className="h-4 w-4 mr-2" />
              {importDialog.isUploading ? "Importing..." : "Import"}
            </Button>
          </div>
          <Button onClick={onCreatePlugin}>
            <Plus className="h-4 w-4 mr-2" />
            Create Private Plugin
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="text-muted-foreground">Loading private plugins...</div>
        </div>
      ) : plugins.length === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <Code2 className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Private Plugins</h3>
            <p className="text-muted-foreground mb-4">
              Create your first private plugin using Liquid templates and TRMNL's design framework, or import an existing TRMNL plugin.
            </p>
            <div className="flex gap-2 justify-center">
              <div className="relative">
                <Input
                  id="import-file-empty"
                  type="file"
                  accept=".zip"
                  onChange={handleImportFileChange}
                  className="sr-only"
                  disabled={importDialog.isUploading}
                />
                <Button 
                  variant="outline" 
                  onClick={() => document.getElementById('import-file-empty')?.click()}
                  disabled={importDialog.isUploading}
                >
                  <Upload className="h-4 w-4 mr-2" />
                  {importDialog.isUploading ? "Importing..." : "Import"}
                </Button>
              </div>
              <Button onClick={onCreatePlugin}>
                <Plus className="h-4 w-4 mr-2" />
                Create Your First Private Plugin
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Data Strategy</TableHead>
                  <TableHead>Layouts</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {plugins.map((plugin) => (
                  <TableRow key={plugin.id}>
                    <TableCell>
                      <div>
                        <div className="font-medium">{plugin.name}</div>
                        {plugin.description && (
                          <div className="text-sm text-muted-foreground line-clamp-1">
                            {plugin.description}
                          </div>
                        )}
                        {plugin.is_published && (
                          <Badge variant="secondary" className="mt-1">
                            Published
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {getDataStrategyBadge(plugin.data_strategy)}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {getAvailableLayouts(plugin).map((layout) => (
                          <Badge key={layout} variant="outline" className="text-xs">
                            {layout}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell>
                      v{plugin.version}
                    </TableCell>
                    <TableCell>
                      {new Date(plugin.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center gap-2 justify-end">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => exportPlugin(plugin.id)}
                            >
                              <Download className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Export</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => onEditPlugin(plugin)}
                            >
                              <Edit className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Edit Plugin</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => setDeleteDialog({ isOpen: true, plugin })}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Delete Plugin</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={deleteDialog.isOpen}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteDialog({ isOpen: false, plugin: null });
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Private Plugin
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deleteDialog.plugin?.name}"?
              <br /><br />
              This will:
              <ul className="list-disc list-outside ml-6 mt-2 space-y-1">
                <li>Permanently delete the plugin and all its templates</li>
                <li>Remove any existing plugin instances using this plugin</li>
                <li>Stop displaying content on devices</li>
              </ul>
              <br />
              <strong className="text-destructive">
                This action cannot be undone.
              </strong>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setDeleteDialog({ isOpen: false, plugin: null })}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={async () => {
                if (deleteDialog.plugin) {
                  await deletePrivatePlugin(deleteDialog.plugin.id);
                  setDeleteDialog({ isOpen: false, plugin: null });
                }
              }}
            >
              Delete Plugin
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
