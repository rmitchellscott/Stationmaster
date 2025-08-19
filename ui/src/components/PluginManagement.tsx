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
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Separator } from "@/components/ui/separator";
import {
  Puzzle,
  Edit,
  Trash2,
  Settings as SettingsIcon,
  AlertTriangle,
  CheckCircle,
} from "lucide-react";

interface Plugin {
  id: string;
  name: string;
  type: string;
  description: string;
  config_schema: string;
  version: string;
  author: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface UserPlugin {
  id: string;
  user_id: string;
  plugin_id: string;
  name: string;
  settings: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  plugin: Plugin;
  is_used_in_playlists: boolean;
}

interface PluginManagementProps {
  selectedDeviceId: string;
  onUpdate?: () => void;
}

export function PluginManagement({ selectedDeviceId, onUpdate }: PluginManagementProps) {
  const { t } = useTranslation();
  const [userPlugins, setUserPlugins] = useState<UserPlugin[]>([]);
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Add plugin dialog
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null);
  const [instanceName, setInstanceName] = useState("");
  const [instanceSettings, setInstanceSettings] = useState<Record<string, any>>({});
  const [createLoading, setCreateLoading] = useState(false);

  // Edit plugin dialog
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editUserPlugin, setEditUserPlugin] = useState<UserPlugin | null>(null);
  const [editInstanceName, setEditInstanceName] = useState("");
  const [editInstanceSettings, setEditInstanceSettings] = useState<Record<string, any>>({});
  const [updateLoading, setUpdateLoading] = useState(false);

  // Delete confirmation dialog
  const [deletePluginDialog, setDeletePluginDialog] = useState<{
    isOpen: boolean;
    plugin: UserPlugin | null;
  }>({ isOpen: false, plugin: null });

  const fetchUserPlugins = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/user-plugins", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setUserPlugins(data.user_plugins || []);
      } else {
        setError("Failed to fetch user plugins");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const fetchPlugins = async () => {
    try {
      const response = await fetch("/api/plugins", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPlugins(data.plugins || []);
      }
    } catch (error) {
      console.error("Failed to fetch plugins:", error);
    }
  };

  const createUserPlugin = async () => {
    if (!selectedPlugin || !instanceName.trim()) {
      setError("Please provide a name for the plugin instance");
      return;
    }

    try {
      setCreateLoading(true);
      setError(null);

      const response = await fetch("/api/user-plugins", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          plugin_id: selectedPlugin.id,
          name: instanceName.trim(),
          settings: instanceSettings,
        }),
      });

      if (response.ok) {
        setSuccessMessage("Plugin instance created successfully!");
        setShowAddDialog(false);
        setSelectedPlugin(null);
        setInstanceName("");
        setInstanceSettings({});
        await fetchUserPlugins();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to create plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setCreateLoading(false);
    }
  };

  const hasPluginInstanceChanges = () => {
    if (!editUserPlugin) return false;
    
    // Parse original settings
    let originalSettings = {};
    try {
      originalSettings = editUserPlugin.settings ? JSON.parse(editUserPlugin.settings) : {};
    } catch (e) {
      originalSettings = {};
    }
    
    return (
      editInstanceName.trim() !== editUserPlugin.name ||
      JSON.stringify(editInstanceSettings) !== JSON.stringify(originalSettings)
    );
  };

  const updatePluginInstance = async () => {
    if (!editUserPlugin || !editInstanceName.trim()) {
      setError("Please provide a name for the plugin instance");
      return;
    }

    try {
      setUpdateLoading(true);
      setError(null);

      const response = await fetch(`/api/user-plugins/${editUserPlugin.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: editInstanceName.trim(),
          settings: editInstanceSettings,
        }),
      });

      if (response.ok) {
        setSuccessMessage("Plugin instance updated successfully!");
        setShowEditDialog(false);
        setEditUserPlugin(null);
        setEditInstanceName("");
        setEditInstanceSettings({});
        await fetchUserPlugins();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setUpdateLoading(false);
    }
  };

  const deleteUserPlugin = async (userPluginId: string) => {
    try {
      setError(null);
      const response = await fetch(`/api/user-plugins/${userPluginId}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        setSuccessMessage("Plugin instance deleted successfully!");
        await fetchUserPlugins();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  useEffect(() => {
    fetchUserPlugins();
    fetchPlugins();
  }, [selectedDeviceId]);

  useEffect(() => {
    if (successMessage) {
      const timer = setTimeout(() => setSuccessMessage(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [successMessage]);

  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error]);

  const renderSettingsForm = (plugin: Plugin, settings: Record<string, any>, onChange: (key: string, value: any) => void) => {
    let schema;
    try {
      schema = JSON.parse(plugin.config_schema);
    } catch (e) {
      return <div className="text-muted-foreground">Invalid schema configuration</div>;
    }

    const properties = schema.properties || {};

    return (
      <div className="space-y-4">
        {Object.keys(properties).map((key) => {
          const prop = properties[key];
          const value = settings[key] || prop.default || "";

          if (prop.type === "string") {
            return (
              <div key={key}>
                <Label htmlFor={key}>{prop.title || key}</Label>
                <Input
                  id={key}
                  placeholder={prop.placeholder || ""}
                  value={value}
                  onChange={(e) => onChange(key, e.target.value)}
                  className="mt-2"
                />
                {prop.description && (
                  <p className="text-sm text-muted-foreground mt-1">
                    {prop.description}
                  </p>
                )}
              </div>
            );
          }

          if (prop.type === "integer" || prop.type === "number") {
            return (
              <div key={key}>
                <Label htmlFor={key}>{prop.title || key}</Label>
                <Input
                  id={key}
                  type="number"
                  placeholder={prop.placeholder || ""}
                  value={value}
                  onChange={(e) => onChange(key, prop.type === "integer" ? parseInt(e.target.value) || 0 : parseFloat(e.target.value) || 0)}
                  className="mt-2"
                />
                {prop.description && (
                  <p className="text-sm text-muted-foreground mt-1">
                    {prop.description}
                  </p>
                )}
              </div>
            );
          }

          return null;
        })}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {successMessage && (
        <Alert>
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>{successMessage}</AlertDescription>
        </Alert>
      )}

      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-semibold">Plugin Instances</h3>
          <p className="text-muted-foreground">
            Manage your plugin instances for the selected device
          </p>
        </div>
        <Button onClick={() => setShowAddDialog(true)}>
          Add Plugin
        </Button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="text-muted-foreground">Loading plugins...</div>
        </div>
      ) : userPlugins.length === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <Puzzle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Plugin Instances</h3>
            <p className="text-muted-foreground mb-4">
              Create plugin instances to display content on your device.
            </p>
            <Button onClick={() => setShowAddDialog(true)}>
              Add Your First Plugin
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent>
            <Table className="w-full table-fixed lg:table-auto">
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead className="hidden lg:table-cell">Plugin</TableHead>
                  <TableHead className="hidden md:table-cell">Status</TableHead>
                  <TableHead className="hidden lg:table-cell">Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {userPlugins.map((userPlugin) => (
                  <TableRow key={userPlugin.id}>
                    <TableCell className="font-medium">
                      <div>
                        <div>{userPlugin.name}</div>
                        <div className="text-sm text-muted-foreground lg:hidden">
                          {userPlugin.plugin?.name || "Unknown Plugin"} • {userPlugin.is_used_in_playlists ? "Active" : "Unused"}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="hidden lg:table-cell">
                      <div className="font-medium">
                        {userPlugin.plugin?.name || "Unknown Plugin"}
                      </div>
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
                      {userPlugin.is_used_in_playlists ? (
                        <Badge variant="outline">Active</Badge>
                      ) : (
                        <Badge variant="secondary">Unused</Badge>
                      )}
                    </TableCell>
                    <TableCell className="hidden lg:table-cell">
                      {new Date(userPlugin.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center gap-2 justify-end">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setEditUserPlugin(userPlugin);
                            setEditInstanceName(userPlugin.name);
                            
                            // Parse settings from JSON string to object
                            let parsedSettings = {};
                            try {
                              if (userPlugin.settings && typeof userPlugin.settings === 'string') {
                                parsedSettings = JSON.parse(userPlugin.settings);
                              } else if (userPlugin.settings && typeof userPlugin.settings === 'object') {
                                parsedSettings = userPlugin.settings;
                              }
                            } catch (e) {
                              console.error('Failed to parse plugin settings:', e);
                              parsedSettings = {};
                            }
                            
                            setEditInstanceSettings(parsedSettings);
                            setShowEditDialog(true);
                          }}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setDeletePluginDialog({ isOpen: true, plugin: userPlugin })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Add Plugin Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent 
          className="sm:max-w-5xl max-h-[70vh] overflow-hidden flex flex-col mobile-dialog-content !top-[0vh] !translate-y-0 sm:!top-[6vh]"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader className="pb-3">
            <DialogTitle>Add Plugin Instance</DialogTitle>
            <DialogDescription>
              {selectedPlugin ? `Configure your ${selectedPlugin.name} instance` : "Select a plugin to create an instance for your device"}
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto">
            <div className="space-y-4">
              {!selectedPlugin ? (
                <div>
                  <div className="mb-3">
                    <Label className="text-base font-semibold">Available Plugins</Label>
                  </div>
                  {loading ? (
                    <div className="text-center py-6">Loading available plugins...</div>
                  ) : plugins.length === 0 ? (
                    <Card>
                      <CardContent className="text-center py-6">
                        <Puzzle className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
                        <h3 className="text-base font-semibold mb-2">No Plugins Available</h3>
                        <p className="text-sm text-muted-foreground">
                          No plugins have been installed by the administrator yet.
                        </p>
                      </CardContent>
                    </Card>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-3">
                      {plugins.map((plugin) => (
                        <Card key={plugin.id} className="flex flex-col">
                          <CardHeader className="pb-2">
                            <CardTitle className="flex items-start justify-between gap-2">
                              <div className="min-w-0 flex-1">
                                <div className="text-base font-semibold truncate">{plugin.name}</div>
                                <div className="text-xs text-muted-foreground">
                                  v{plugin.version} by {plugin.author}
                                </div>
                              </div>
                              <Badge variant="outline" className="flex-shrink-0 text-xs">{plugin.type}</Badge>
                            </CardTitle>
                          </CardHeader>
                          <CardContent className="flex flex-col flex-grow pt-0">
                            <div className="flex-grow mb-3">
                              <p className="text-xs text-muted-foreground line-clamp-2">
                                {plugin.description}
                              </p>
                            </div>
                            <Button
                              onClick={() => {
                                setSelectedPlugin(plugin);
                                setInstanceName(`My ${plugin.name}`);
                                
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
                              }}
                              size="sm"
                              className="w-full mt-auto"
                            >
                              Create Instance
                            </Button>
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  )}
                </div>
              ) : (
                <>
                  <div className="mb-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setSelectedPlugin(null);
                        setInstanceName("");
                        setInstanceSettings({});
                      }}
                      className="mb-2"
                    >
                      ← Back to Plugin Selection
                    </Button>
                    <div className="flex items-center gap-3 p-2 bg-muted/50 rounded-lg">
                      <div className="flex-1">
                        <div className="font-semibold text-sm">{selectedPlugin.name}</div>
                        <div className="text-xs text-muted-foreground">
                          v{selectedPlugin.version} by {selectedPlugin.author}
                        </div>
                      </div>
                      <Badge variant="outline" className="text-xs">{selectedPlugin.type}</Badge>
                    </div>
                  </div>

                  <div>
                    <Label htmlFor="instanceName" className="text-sm">Instance Name</Label>
                    <Input
                      id="instanceName"
                      placeholder={`My ${selectedPlugin.name}`}
                      value={instanceName}
                      onChange={(e) => setInstanceName(e.target.value)}
                      className="mt-1"
                    />
                  </div>

                  <div>
                    <Label className="text-sm">Plugin Configuration</Label>
                    <div className="mt-1">
                      {renderSettingsForm(selectedPlugin, instanceSettings, (key, value) => {
                        setInstanceSettings(prev => ({ ...prev, [key]: value }));
                      })}
                    </div>
                  </div>
                </>
              )}
            </div>
          </div>

          <DialogFooter className="pt-3">
            <Button
              variant="outline"
              onClick={() => {
                setShowAddDialog(false);
                setSelectedPlugin(null);
                setInstanceName("");
                setInstanceSettings({});
              }}
            >
              Cancel
            </Button>
            {selectedPlugin && (
              <Button
                onClick={createUserPlugin}
                disabled={!instanceName.trim() || createLoading}
              >
                {createLoading ? "Creating..." : "Create Instance"}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Plugin Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent 
          className="sm:max-w-2xl max-h-[80vh] overflow-y-auto"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>Edit Plugin Instance</DialogTitle>
            <DialogDescription>
              Update the settings for "{editUserPlugin?.name}".
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6">
            <div>
              <Label htmlFor="edit-instance-name">Instance Name</Label>
              <Input
                id="edit-instance-name"
                value={editInstanceName}
                onChange={(e) => setEditInstanceName(e.target.value)}
                placeholder="Enter instance name"
                className="mt-2"
              />
            </div>

            {editUserPlugin?.plugin && (
              <>
                <Separator />
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base flex items-center gap-2">
                      <SettingsIcon className="h-4 w-4" />
                      Plugin Configuration
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-0">
                    {renderSettingsForm(
                      editUserPlugin.plugin,
                      editInstanceSettings,
                      (key: string, value: any) => {
                        setEditInstanceSettings(prev => ({ ...prev, [key]: value }));
                      }
                    )}
                  </CardContent>
                </Card>
              </>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={updatePluginInstance}
              disabled={updateLoading || !editInstanceName.trim() || !hasPluginInstanceChanges()}
            >
              {updateLoading ? "Updating..." : "Update Instance"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Plugin Instance Confirmation Dialog */}
      <AlertDialog
        open={deletePluginDialog.isOpen}
        onOpenChange={(open) => {
          if (!open) {
            setDeletePluginDialog({ isOpen: false, plugin: null });
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Plugin Instance
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the plugin instance "{deletePluginDialog.plugin?.name}"?
              <br /><br />
              This will:
              <ul className="list-disc list-outside ml-6 mt-2 space-y-1">
                <li>Permanently delete this plugin instance and its settings</li>
                <li>Remove it from any playlists it's currently in</li>
                <li>Stop displaying this content on devices</li>
              </ul>
              <br />
              <strong className="text-destructive">
                This action cannot be undone.
              </strong>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setDeletePluginDialog({ isOpen: false, plugin: null })}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={async () => {
                if (deletePluginDialog.plugin) {
                  await deleteUserPlugin(deletePluginDialog.plugin.id);
                  setDeletePluginDialog({ isOpen: false, plugin: null });
                }
              }}
            >
              Delete Instance
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
