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
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Puzzle,
  Plus,
  Edit,
  Trash2,
  Settings as SettingsIcon,
  AlertTriangle,
  CheckCircle,
  Copy,
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
}

interface PluginManagementProps {
  isOpen: boolean;
  onClose: () => void;
}

export function PluginManagement({ isOpen, onClose }: PluginManagementProps) {
  const { t } = useTranslation();
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [userPlugins, setUserPlugins] = useState<UserPlugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Plugin instance creation
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null);
  const [instanceName, setInstanceName] = useState("");
  const [instanceSettings, setInstanceSettings] = useState<Record<string, any>>({});

  // Plugin instance editing
  const [editUserPlugin, setEditUserPlugin] = useState<UserPlugin | null>(null);
  const [editInstanceName, setEditInstanceName] = useState("");
  const [editInstanceSettings, setEditInstanceSettings] = useState<Record<string, any>>({});

  // Plugin instance deletion
  const [deleteUserPlugin, setDeleteUserPlugin] = useState<UserPlugin | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      fetchPlugins();
      fetchUserPlugins();
    }
  }, [isOpen]);

  const fetchPlugins = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch("/api/plugins", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setPlugins(data.plugins || []);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to fetch plugins");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const fetchUserPlugins = async () => {
    try {
      const response = await fetch("/api/user-plugins", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setUserPlugins(data.user_plugins || []);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to fetch user plugins");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const openCreateDialog = (plugin: Plugin) => {
    setSelectedPlugin(plugin);
    setInstanceName(`${plugin.name} Instance`);
    
    // Parse default settings from schema
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
    
    setShowCreateDialog(true);
  };

  const createUserPlugin = async () => {
    if (!selectedPlugin || !instanceName.trim()) {
      setError("Please fill in all required fields");
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
        setShowCreateDialog(false);
        setSelectedPlugin(null);
        setInstanceName("");
        setInstanceSettings({});
        await fetchUserPlugins();
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

  const openEditDialog = (userPlugin: UserPlugin) => {
    setEditUserPlugin(userPlugin);
    setEditInstanceName(userPlugin.name);
    
    try {
      setEditInstanceSettings(userPlugin.settings ? JSON.parse(userPlugin.settings) : {});
    } catch (e) {
      setEditInstanceSettings({});
    }
  };

  const updateUserPlugin = async () => {
    if (!editUserPlugin || !editInstanceName.trim()) {
      setError("Please fill in all required fields");
      return;
    }

    try {
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
        setEditUserPlugin(null);
        setEditInstanceName("");
        setEditInstanceSettings({});
        await fetchUserPlugins();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const confirmDeleteUserPlugin = async () => {
    if (!deleteUserPlugin) return;

    try {
      setDeleteLoading(true);
      setError(null);

      const response = await fetch(`/api/user-plugins/${deleteUserPlugin.id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        setSuccessMessage("Plugin instance deleted successfully!");
        setDeleteUserPlugin(null);
        await fetchUserPlugins();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setDeleteLoading(false);
    }
  };

  const copyUserPlugin = async (userPlugin: UserPlugin) => {
    try {
      setError(null);

      const response = await fetch("/api/user-plugins", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          plugin_id: userPlugin.plugin_id,
          name: `${userPlugin.name} (Copy)`,
          settings: userPlugin.settings ? JSON.parse(userPlugin.settings) : {},
        }),
      });

      if (response.ok) {
        setSuccessMessage("Plugin instance copied successfully!");
        await fetchUserPlugins();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to copy plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const renderSettingsForm = (
    schema: string | null,
    settings: Record<string, any>,
    onSettingsChange: (settings: Record<string, any>) => void
  ) => {
    if (!schema) {
      return (
        <div>
          <Label>Settings (JSON)</Label>
          <Textarea
            value={JSON.stringify(settings, null, 2)}
            onChange={(e) => {
              try {
                onSettingsChange(JSON.parse(e.target.value));
              } catch (err) {
                // Invalid JSON, don't update
              }
            }}
            placeholder="{}"
            className="mt-2 font-mono"
            rows={6}
          />
        </div>
      );
    }

    try {
      const parsedSchema = JSON.parse(schema);
      
      if (!parsedSchema.properties) {
        return renderSettingsForm(null, settings, onSettingsChange);
      }

      return (
        <div className="space-y-4">
          {Object.keys(parsedSchema.properties).map((key) => {
            const property = parsedSchema.properties[key];
            const value = settings[key] || property.default || "";

            const updateSetting = (newValue: any) => {
              onSettingsChange({
                ...settings,
                [key]: newValue,
              });
            };

            switch (property.type) {
              case "string":
                if (property.enum) {
                  return (
                    <div key={key}>
                      <Label htmlFor={key}>{property.title || key}</Label>
                      <Select value={value} onValueChange={updateSetting}>
                        <SelectTrigger className="mt-2">
                          <SelectValue placeholder="Select an option" />
                        </SelectTrigger>
                        <SelectContent>
                          {property.enum.map((option: string) => (
                            <SelectItem key={option} value={option}>
                              {option}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {property.description && (
                        <p className="text-sm text-gray-600 mt-1">{property.description}</p>
                      )}
                    </div>
                  );
                } else if (property.format === "textarea") {
                  return (
                    <div key={key}>
                      <Label htmlFor={key}>{property.title || key}</Label>
                      <Textarea
                        id={key}
                        value={value}
                        onChange={(e) => updateSetting(e.target.value)}
                        placeholder={property.placeholder}
                        className="mt-2"
                        rows={3}
                      />
                      {property.description && (
                        <p className="text-sm text-gray-600 mt-1">{property.description}</p>
                      )}
                    </div>
                  );
                } else {
                  return (
                    <div key={key}>
                      <Label htmlFor={key}>{property.title || key}</Label>
                      <Input
                        id={key}
                        type={property.format === "password" ? "password" : "text"}
                        value={value}
                        onChange={(e) => updateSetting(e.target.value)}
                        placeholder={property.placeholder}
                        className="mt-2"
                      />
                      {property.description && (
                        <p className="text-sm text-gray-600 mt-1">{property.description}</p>
                      )}
                    </div>
                  );
                }
              
              case "number":
              case "integer":
                return (
                  <div key={key}>
                    <Label htmlFor={key}>{property.title || key}</Label>
                    <Input
                      id={key}
                      type="number"
                      min={property.minimum}
                      max={property.maximum}
                      step={property.type === "integer" ? 1 : "any"}
                      value={value}
                      onChange={(e) => updateSetting(property.type === "integer" 
                        ? parseInt(e.target.value) || 0
                        : parseFloat(e.target.value) || 0
                      )}
                      className="mt-2"
                    />
                    {property.description && (
                      <p className="text-sm text-gray-600 mt-1">{property.description}</p>
                    )}
                  </div>
                );
              
              case "boolean":
                return (
                  <div key={key} className="flex items-center space-x-2">
                    <input
                      id={key}
                      type="checkbox"
                      checked={!!value}
                      onChange={(e) => updateSetting(e.target.checked)}
                      className="rounded border-gray-300"
                    />
                    <Label htmlFor={key}>{property.title || key}</Label>
                    {property.description && (
                      <p className="text-sm text-gray-600">{property.description}</p>
                    )}
                  </div>
                );
              
              default:
                return (
                  <div key={key}>
                    <Label>{property.title || key}</Label>
                    <Textarea
                      value={JSON.stringify(value, null, 2)}
                      onChange={(e) => {
                        try {
                          updateSetting(JSON.parse(e.target.value));
                        } catch (err) {
                          // Invalid JSON
                        }
                      }}
                      className="mt-2 font-mono"
                      rows={3}
                    />
                    {property.description && (
                      <p className="text-sm text-gray-600 mt-1">{property.description}</p>
                    )}
                  </div>
                );
            }
          })}
        </div>
      );
    } catch (e) {
      return renderSettingsForm(null, settings, onSettingsChange);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Puzzle className="h-5 w-5" />
            Plugin Management
          </DialogTitle>
          <DialogDescription>
            Manage your plugins and create instances with custom settings.
          </DialogDescription>
        </DialogHeader>

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

        <Tabs defaultValue="instances" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="instances">My Plugin Instances</TabsTrigger>
            <TabsTrigger value="available">Available Plugins</TabsTrigger>
          </TabsList>
          
          <TabsContent value="instances" className="space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Plugin Instances</h3>
            </div>

            {loading ? (
              <div className="text-center py-8">Loading plugin instances...</div>
            ) : userPlugins.length === 0 ? (
              <Card>
                <CardContent className="text-center py-8">
                  <Puzzle className="h-12 w-12 mx-auto text-gray-400 mb-4" />
                  <h3 className="text-lg font-semibold mb-2">No Plugin Instances</h3>
                  <p className="text-gray-600 mb-4">
                    Create your first plugin instance to get started.
                  </p>
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardContent className="p-0">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Instance Name</TableHead>
                        <TableHead>Plugin</TableHead>
                        <TableHead className="hidden sm:table-cell">Type</TableHead>
                        <TableHead className="hidden sm:table-cell">Status</TableHead>
                        <TableHead className="hidden lg:table-cell">Created</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {userPlugins.map((userPlugin) => (
                        <TableRow key={userPlugin.id}>
                          <TableCell>
                            <div className="font-medium">{userPlugin.name}</div>
                          </TableCell>
                          <TableCell>
                            <div>
                              <div className="font-medium">{userPlugin.plugin.name}</div>
                              <div className="text-sm text-gray-600">v{userPlugin.plugin.version}</div>
                            </div>
                          </TableCell>
                          <TableCell className="hidden sm:table-cell">
                            <Badge variant="outline">{userPlugin.plugin.type}</Badge>
                          </TableCell>
                          <TableCell className="hidden sm:table-cell">
                            {userPlugin.is_active ? (
                              <Badge variant="default">Active</Badge>
                            ) : (
                              <Badge variant="secondary">Inactive</Badge>
                            )}
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-default">
                                  {new Date(userPlugin.created_at).toLocaleDateString()}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                {formatDate(userPlugin.created_at)}
                              </TooltipContent>
                            </Tooltip>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => openEditDialog(userPlugin)}
                              >
                                <Edit className="h-4 w-4" />
                              </Button>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => copyUserPlugin(userPlugin)}
                              >
                                <Copy className="h-4 w-4" />
                              </Button>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => setDeleteUserPlugin(userPlugin)}
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
          </TabsContent>

          <TabsContent value="available" className="space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Available Plugins</h3>
            </div>

            {loading ? (
              <div className="text-center py-8">Loading available plugins...</div>
            ) : plugins.length === 0 ? (
              <Card>
                <CardContent className="text-center py-8">
                  <Puzzle className="h-12 w-12 mx-auto text-gray-400 mb-4" />
                  <h3 className="text-lg font-semibold mb-2">No Plugins Available</h3>
                  <p className="text-gray-600">
                    No plugins have been installed by the administrator yet.
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {plugins.map((plugin) => (
                  <Card key={plugin.id}>
                    <CardHeader>
                      <CardTitle className="flex items-center justify-between">
                        <div>
                          <div className="text-lg">{plugin.name}</div>
                          <div className="text-sm font-normal text-gray-600">
                            v{plugin.version} by {plugin.author}
                          </div>
                        </div>
                        <Badge variant="outline">{plugin.type}</Badge>
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-gray-600 mb-4">{plugin.description}</p>
                      <Button
                        onClick={() => openCreateDialog(plugin)}
                        className="w-full"
                      >
                        <Plus className="h-4 w-4 mr-2" />
                        Create Instance
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>
        </Tabs>
      </DialogContent>

      {/* Create Instance Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create Plugin Instance</DialogTitle>
            <DialogDescription>
              Create a new instance of "{selectedPlugin?.name}" with custom settings.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-6">
            <div>
              <Label htmlFor="instance-name">Instance Name</Label>
              <Input
                id="instance-name"
                value={instanceName}
                onChange={(e) => setInstanceName(e.target.value)}
                placeholder="My Plugin Instance"
                className="mt-2"
              />
            </div>

            {selectedPlugin && (
              <div>
                <Label>Plugin Settings</Label>
                <div className="mt-2 p-4 border rounded-lg">
                  {renderSettingsForm(
                    selectedPlugin.config_schema,
                    instanceSettings,
                    setInstanceSettings
                  )}
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={createUserPlugin}
              disabled={createLoading || !instanceName.trim()}
            >
              {createLoading ? "Creating..." : "Create Instance"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Instance Dialog */}
      <Dialog open={!!editUserPlugin} onOpenChange={() => setEditUserPlugin(null)}>
        <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
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
                className="mt-2"
              />
            </div>

            {editUserPlugin && (
              <div>
                <Label>Plugin Settings</Label>
                <div className="mt-2 p-4 border rounded-lg">
                  {renderSettingsForm(
                    editUserPlugin.plugin.config_schema,
                    editInstanceSettings,
                    setEditInstanceSettings
                  )}
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditUserPlugin(null)}>
              Cancel
            </Button>
            <Button
              onClick={updateUserPlugin}
              disabled={!editInstanceName.trim()}
            >
              Update Instance
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Instance Dialog */}
      <Dialog open={!!deleteUserPlugin} onOpenChange={() => setDeleteUserPlugin(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Plugin Instance
            </DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deleteUserPlugin?.name}"? This will remove it from all playlists. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteUserPlugin(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDeleteUserPlugin}
              disabled={deleteLoading}
            >
              {deleteLoading ? "Deleting..." : "Delete Instance"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}