import React, { useState, useEffect, useCallback } from "react";
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
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AlertTriangle,
  Grid2x2,
  Plus,
  X,
  Loader2,
  Move,
  Settings,
  RefreshCw,
} from "lucide-react";
import { mashupLayouts, MashupLayout } from "./MashupLayoutPicker";

interface PluginInstance {
  id: string;
  name: string;
  plugin_definition?: {
    name: string;
    type: string;
  };
  refresh_interval: number;
  is_active: boolean;
}

interface MashupChild {
  id: string;
  mashup_instance_id: string;
  child_instance_id: string;
  grid_position: string;
  child_instance: PluginInstance;
}

interface MashupInstance {
  id: string;
  name: string;
  plugin_definition: {
    id: string;
    name: string;
    mashup_layout?: string;
  };
}

interface MashupEditorProps {
  isOpen: boolean;
  onClose: () => void;
  mashupInstance: MashupInstance | null;
}

export const MashupEditor: React.FC<MashupEditorProps> = ({
  isOpen,
  onClose,
  mashupInstance,
}) => {
  const { t } = useTranslation();
  const [children, setChildren] = useState<MashupChild[]>([]);
  const [availablePlugins, setAvailablePlugins] = useState<PluginInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [updatingChildren, setUpdatingChildren] = useState<string[]>([]);

  // Get layout info
  const layout = mashupLayouts.find(
    l => l.id === mashupInstance?.plugin_definition.mashup_layout
  ) || mashupLayouts[0];

  // Fetch mashup children
  const fetchChildren = useCallback(async () => {
    if (!mashupInstance?.id) return;
    
    try {
      setLoading(true);
      setError(null);

      const response = await fetch(`/api/plugin-instances/${mashupInstance.id}/children`, {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setChildren(data.children || []);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to fetch mashup children");
      }
    } catch (error) {
      setError("Network error occurred while fetching children");
    } finally {
      setLoading(false);
    }
  }, [mashupInstance?.id]);

  // Fetch available plugin instances
  const fetchAvailablePlugins = useCallback(async () => {
    try {
      const response = await fetch("/api/plugin-instances", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        // Filter to only private plugins that aren't already in this mashup
        const childIds = new Set(children.map(c => c.child_instance_id));
        const available = (data.plugin_instances || [])
          .filter((instance: PluginInstance) => 
            instance.plugin_definition?.type === "private" &&
            instance.is_active &&
            !childIds.has(instance.id)
          );
        setAvailablePlugins(available);
      }
    } catch (error) {
      // Silently fail for available plugins - not critical
    }
  }, [children]);

  useEffect(() => {
    if (isOpen && mashupInstance) {
      fetchChildren();
    }
  }, [isOpen, mashupInstance, fetchChildren]);

  useEffect(() => {
    if (isOpen) {
      fetchAvailablePlugins();
    }
  }, [isOpen, fetchAvailablePlugins]);

  const getChildAtPosition = (position: string): MashupChild | null => {
    return children.find(child => child.grid_position === position) || null;
  };

  const addChildToPosition = async (position: string, pluginInstanceId: string) => {
    if (!mashupInstance?.id) return;

    setUpdatingChildren(prev => [...prev, position]);
    setError(null);

    try {
      const response = await fetch(`/api/plugin-instances/${mashupInstance.id}/children`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          child_instance_id: pluginInstanceId,
          grid_position: position,
        }),
      });

      if (response.ok) {
        await fetchChildren(); // Refresh children list
        setSuccess(`Plugin added to ${position} position`);
        setTimeout(() => setSuccess(null), 3000);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to add child plugin");
      }
    } catch (error) {
      setError("Network error occurred while adding child plugin");
    } finally {
      setUpdatingChildren(prev => prev.filter(p => p !== position));
    }
  };

  const removeChildFromPosition = async (position: string, childId: string) => {
    if (!mashupInstance?.id) return;

    setUpdatingChildren(prev => [...prev, position]);
    setError(null);

    try {
      const response = await fetch(
        `/api/plugin-instances/${mashupInstance.id}/children/${childId}`,
        {
          method: "DELETE",
          credentials: "include",
        }
      );

      if (response.ok) {
        await fetchChildren(); // Refresh children list
        setSuccess(`Plugin removed from ${position} position`);
        setTimeout(() => setSuccess(null), 3000);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to remove child plugin");
      }
    } catch (error) {
      setError("Network error occurred while removing child plugin");
    } finally {
      setUpdatingChildren(prev => prev.filter(p => p !== position));
    }
  };

  const GridPosition: React.FC<{ position: string; index: number }> = ({ 
    position, 
    index 
  }) => {
    const child = getChildAtPosition(position);
    const isUpdating = updatingChildren.includes(position);
    const [selectedPlugin, setSelectedPlugin] = useState<string>("");

    const handleAddPlugin = () => {
      if (selectedPlugin) {
        addChildToPosition(position, selectedPlugin);
        setSelectedPlugin("");
      }
    };

    return (
      <Card 
        className={`h-32 ${child ? 'bg-primary/5 border-primary/20' : 'border-dashed border-muted-foreground/30'}`}
      >
        <CardContent className="p-3 h-full flex flex-col">
          <div className="flex items-center justify-between mb-2">
            <Badge variant="outline" className="text-xs">
              {position}
            </Badge>
            <span className="text-xs font-mono text-muted-foreground">
              {index + 1}
            </span>
          </div>

          {child ? (
            <div className="flex-1 flex flex-col justify-between">
              <div>
                <h4 className="text-sm font-medium truncate">
                  {child.child_instance.name}
                </h4>
                <p className="text-xs text-muted-foreground">
                  {child.child_instance.plugin_definition?.name || "Unknown Plugin"}
                </p>
              </div>
              <Button
                size="sm"
                variant="outline"
                onClick={() => removeChildFromPosition(position, child.child_instance_id)}
                disabled={isUpdating}
                className="mt-2 h-7"
              >
                {isUpdating ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <X className="h-3 w-3" />
                )}
              </Button>
            </div>
          ) : (
            <div className="flex-1 flex flex-col justify-center items-center gap-2">
              <Select
                value={selectedPlugin}
                onValueChange={setSelectedPlugin}
                disabled={isUpdating}
              >
                <SelectTrigger className="h-7 text-xs">
                  <SelectValue placeholder="Select plugin..." />
                </SelectTrigger>
                <SelectContent>
                  {availablePlugins.map((plugin) => (
                    <SelectItem key={plugin.id} value={plugin.id}>
                      {plugin.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              
              <Button
                size="sm"
                onClick={handleAddPlugin}
                disabled={!selectedPlugin || isUpdating}
                className="h-7"
              >
                {isUpdating ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <Plus className="h-3 w-3" />
                )}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    );
  };

  if (!mashupInstance) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-4xl max-h-[90vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Grid2x2 className="h-5 w-5" />
            Edit Mashup: {mashupInstance.name}
          </DialogTitle>
          <DialogDescription>
            Manage child plugins in your {layout.name.toLowerCase()} mashup layout
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {success && (
            <Alert>
              <AlertDescription>{success}</AlertDescription>
            </Alert>
          )}

          {loading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 animate-spin mr-2" />
              <span>Loading mashup children...</span>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-medium">
                    {layout.name} Layout
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    {layout.description} • {layout.positions.length} positions
                  </p>
                </div>
                <Badge variant="secondary">
                  {children.length} / {layout.positions.length} filled
                </Badge>
              </div>

              <Separator />

              {/* Grid Layout */}
              <div className="space-y-4">
                <h4 className="font-medium">Plugin Positions</h4>
                <div className={`grid ${layout.gridTemplate} gap-4`}>
                  {layout.positions.map((position, index) => (
                    <GridPosition
                      key={position}
                      position={position}
                      index={index}
                    />
                  ))}
                </div>
              </div>

              {/* Available Plugins Info */}
              {availablePlugins.length === 0 && children.length < layout.positions.length && (
                <Alert>
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    No available private plugin instances found. Create some private plugin instances first to add them to your mashup.
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};