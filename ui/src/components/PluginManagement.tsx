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
  Puzzle,
  Plus,
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
    if (!confirm("Are you sure you want to delete this plugin instance?")) {
      return;
    }

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
        <Button onClick={() => setShowAddDialog(true)} className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
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
              <Plus className="h-4 w-4 mr-2" />
              Add Your First Plugin
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Plugin</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {userPlugins.map((userPlugin) => (
                  <TableRow key={userPlugin.id}>
                    <TableCell className="font-medium">
                      {userPlugin.name}
                    </TableCell>
                    <TableCell>
                      <div>
                        <div className="font-medium">
                          {userPlugin.plugin?.name || "Unknown Plugin"}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {userPlugin.plugin?.type || "unknown"}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      {userPlugin.is_active ? (
                        <Badge variant="outline">Active</Badge>
                      ) : (
                        <Badge variant="secondary">Inactive</Badge>
                      )}
                    </TableCell>
                    <TableCell>
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
                            setEditInstanceSettings(userPlugin.settings || {});
                            setShowEditDialog(true);
                          }}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => deleteUserPlugin(userPlugin.id)}
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
        <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Add Plugin Instance</DialogTitle>
            <DialogDescription>
              Select a plugin and configure an instance for your device.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6">
            <div>
              <Label>Select Plugin</Label>
              <Select
                value={selectedPlugin?.id || ""}
                onValueChange={(value) => {
                  const plugin = plugins.find(p => p.id === value);
                  setSelectedPlugin(plugin || null);
                  setInstanceSettings({});
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Choose a plugin..." />
                </SelectTrigger>
                <SelectContent>
                  {plugins.map((plugin) => (
                    <SelectItem key={plugin.id} value={plugin.id}>
                      <div>
                        <div className="font-medium">{plugin.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {plugin.description}
                        </div>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {selectedPlugin && (
              <>
                <div>
                  <Label htmlFor="instanceName">Instance Name</Label>
                  <Input
                    id="instanceName"
                    placeholder={`My ${selectedPlugin.name}`}
                    value={instanceName}
                    onChange={(e) => setInstanceName(e.target.value)}
                  />
                </div>

                <div>
                  <Label>Plugin Configuration</Label>
                  {renderSettingsForm(selectedPlugin, instanceSettings, (key, value) => {
                    setInstanceSettings(prev => ({ ...prev, [key]: value }));
                  })}
                </div>
              </>
            )}
          </div>

          <DialogFooter>
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
            <Button
              onClick={createUserPlugin}
              disabled={!selectedPlugin || !instanceName.trim() || createLoading}
            >
              {createLoading ? "Creating..." : "Create Instance"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Plugin Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Plugin Instance</DialogTitle>
            <DialogDescription>
              Update the settings for "{editUserPlugin?.name}".
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
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
              <div>
                <Label className="mb-2 block">Plugin Settings</Label>
                {renderSettingsForm(
                  editUserPlugin.plugin,
                  editInstanceSettings,
                  (key: string, value: any) => {
                    setEditInstanceSettings(prev => ({ ...prev, [key]: value }));
                  }
                )}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={updatePluginInstance}
              disabled={updateLoading || !editInstanceName.trim()}
            >
              {updateLoading ? "Updating..." : "Update Instance"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}