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
  PlayCircle,
  Plus,
  Edit,
  Trash2,
  Star,
  Clock,
  Calendar,
  CheckCircle,
  AlertTriangle,
  Eye,
  EyeOff,
} from "lucide-react";
import { Device } from "@/utils/deviceHelpers";

interface UserPlugin {
  id: string;
  user_id: string;
  plugin_id: string;
  name: string;
  settings: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  plugin: {
    id: string;
    name: string;
    type: string;
    description: string;
  };
}

interface PlaylistItem {
  id: string;
  playlist_id: string;
  user_plugin_id: string;
  order_index: number;
  is_visible: boolean;
  importance: number;
  duration_override?: number;
  created_at: string;
  updated_at: string;
  user_plugin?: UserPlugin;
  schedules?: any[];
}

interface PlaylistManagementProps {
  selectedDeviceId: string;
  devices: Device[];
  onUpdate?: () => void;
}

export function PlaylistManagement({ selectedDeviceId, devices, onUpdate }: PlaylistManagementProps) {
  const { t } = useTranslation();
  const [playlistItems, setPlaylistItems] = useState<PlaylistItem[]>([]);
  const [userPlugins, setUserPlugins] = useState<UserPlugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Add item dialog
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [selectedUserPlugin, setSelectedUserPlugin] = useState<UserPlugin | null>(null);
  const [addLoading, setAddLoading] = useState(false);

  const fetchPlaylistItems = async () => {
    if (!selectedDeviceId) return;
    
    try {
      setLoading(true);
      // Get the default playlist for this device
      const playlistResponse = await fetch(`/api/playlists?device_id=${selectedDeviceId}`, {
        credentials: "include",
      });
      
      if (playlistResponse.ok) {
        const playlistData = await playlistResponse.json();
        const defaultPlaylist = playlistData.playlists?.find((p: any) => p.is_default);
        
        if (defaultPlaylist) {
          const itemsResponse = await fetch(`/api/playlists/${defaultPlaylist.id}`, {
            credentials: "include",
          });
          if (itemsResponse.ok) {
            const itemsData = await itemsResponse.json();
            setPlaylistItems(itemsData.items || []);
          }
        } else {
          // Create default playlist if it doesn't exist
          await createDefaultPlaylist();
        }
      }
    } catch (error) {
      console.error("Failed to fetch playlist items:", error);
      setError("Failed to fetch playlist items");
    } finally {
      setLoading(false);
    }
  };

  const createDefaultPlaylist = async () => {
    try {
      const response = await fetch("/api/playlists", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: "Default Playlist",
          device_id: selectedDeviceId,
          is_default: true,
        }),
      });

      if (response.ok) {
        await fetchPlaylistItems();
      }
    } catch (error) {
      console.error("Failed to create default playlist:", error);
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
      }
    } catch (error) {
      console.error("Failed to fetch user plugins:", error);
    }
  };

  const addPlaylistItem = async () => {
    if (!selectedUserPlugin || !selectedDeviceId) return;

    try {
      setAddLoading(true);
      setError(null);

      // Get the default playlist for this device
      const playlistResponse = await fetch(`/api/playlists?device_id=${selectedDeviceId}`, {
        credentials: "include",
      });
      
      if (playlistResponse.ok) {
        const playlistData = await playlistResponse.json();
        const defaultPlaylist = playlistData.playlists?.find((p: any) => p.is_default);
        
        if (defaultPlaylist) {
          const response = await fetch(`/api/playlists/${defaultPlaylist.id}/items`, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            credentials: "include",
            body: JSON.stringify({
              user_plugin_id: selectedUserPlugin.id,
            }),
          });

          if (response.ok) {
            setSuccessMessage("Item added to playlist successfully!");
            setShowAddDialog(false);
            setSelectedUserPlugin(null);
            await fetchPlaylistItems();
            onUpdate?.();
          } else {
            const errorData = await response.json();
            setError(errorData.error || "Failed to add item to playlist");
          }
        }
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setAddLoading(false);
    }
  };

  const removePlaylistItem = async (itemId: string) => {
    try {
      setError(null);
      const response = await fetch(`/api/playlists/items/${itemId}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (response.ok) {
        setSuccessMessage("Item removed from playlist successfully!");
        await fetchPlaylistItems();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to remove item from playlist");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const toggleItemVisibility = async (item: PlaylistItem) => {
    try {
      setError(null);
      const response = await fetch(`/api/playlists/items/${item.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          is_visible: !item.is_visible,
        }),
      });
      if (response.ok) {
        await fetchPlaylistItems();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update item visibility");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const getAvailableUserPlugins = () => {
    const usedPluginIds = playlistItems.map(item => item.user_plugin_id);
    return userPlugins.filter(plugin => !usedPluginIds.includes(plugin.id));
  };

  useEffect(() => {
    if (selectedDeviceId) {
      fetchPlaylistItems();
      fetchUserPlugins();
    }
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
          <h3 className="text-lg font-semibold">Playlist Items</h3>
          <p className="text-muted-foreground">
            Manage content rotation for the selected device
          </p>
        </div>
        <Button
          onClick={() => setShowAddDialog(true)}
          className="flex items-center gap-2"
          disabled={getAvailableUserPlugins().length === 0}
        >
          <Plus className="h-4 w-4" />
          Add Item
        </Button>
      </div>

      {getAvailableUserPlugins().length === 0 && playlistItems.length === 0 && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            No plugin instances available to add. Create plugin instances first in the Plugins tab.
          </AlertDescription>
        </Alert>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="text-muted-foreground">Loading playlist items...</div>
        </div>
      ) : playlistItems.length === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <PlayCircle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Playlist Items</h3>
            <p className="text-muted-foreground mb-4">
              Add plugin instances to this playlist to display content on your device.
            </p>
            <Button 
              onClick={() => setShowAddDialog(true)}
              disabled={getAvailableUserPlugins().length === 0}
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Item
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Order</TableHead>
                  <TableHead>Plugin</TableHead>
                  <TableHead className="hidden sm:table-cell">Status</TableHead>
                  <TableHead className="hidden sm:table-cell">Importance</TableHead>
                  <TableHead className="hidden sm:table-cell">Duration</TableHead>
                  <TableHead className="hidden lg:table-cell">Schedules</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {playlistItems
                  .sort((a, b) => a.order_index - b.order_index)
                  .map((item) => {
                    const device = devices.find((d) => d.id === selectedDeviceId);
                    
                    // Get the list of visible items in order (same filtering as backend)
                    const visibleItems = playlistItems
                      .filter(i => i.is_visible)
                      .sort((a, b) => a.order_index - b.order_index);
                    
                    // Check if this item is the currently showing one
                    const currentlyShowingItem = device && visibleItems[device.last_playlist_index];
                    const isCurrentlyShowing = currentlyShowingItem && currentlyShowingItem.id === item.id;
                    
                    return (
                    <TableRow key={item.id}>
                      <TableCell>
                        <Badge variant="outline">#{item.order_index}</Badge>
                      </TableCell>
                      <TableCell>
                        <div>
                          <div className="font-medium">
                            {item.user_plugin?.name || "Unnamed Instance"}
                          </div>
                          <div className="text-sm text-muted-foreground">
                            {item.user_plugin?.plugin?.name || "Unknown Plugin"}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="hidden sm:table-cell">
                        {isCurrentlyShowing ? (
                          <Badge variant="default">
                            <PlayCircle className="h-3 w-3 mr-1" />
                            Now Showing
                          </Badge>
                        ) : item.is_visible ? (
                          <Badge variant="outline">
                            <Eye className="h-3 w-3 mr-1" />
                            Visible
                          </Badge>
                        ) : (
                          <Badge variant="secondary">
                            <EyeOff className="h-3 w-3 mr-1" />
                            Hidden
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="hidden sm:table-cell">
                        {item.importance === 1 ? (
                          <Badge variant="secondary" className="bg-orange-100 text-orange-800">
                            <Star className="h-3 w-3 mr-1" />
                            Important
                          </Badge>
                        ) : (
                          <Badge variant="outline">Normal</Badge>
                        )}
                      </TableCell>
                      <TableCell className="hidden sm:table-cell">
                        {item.duration_override ? `${item.duration_override}s` : "Default"}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {item.schedules?.length || 0} schedule(s)
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => toggleItemVisibility(item)}
                            title={item.is_visible ? "Hide from device" : "Show on device"}
                          >
                            {item.is_visible ? (
                              <Eye className="h-4 w-4" />
                            ) : (
                              <EyeOff className="h-4 w-4" />
                            )}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => removePlaylistItem(item.id)}
                            title="Remove from playlist"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    );
                  })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Add Item Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add Playlist Item</DialogTitle>
            <DialogDescription>
              Select a plugin instance to add to the playlist.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label>Select Plugin Instance</Label>
              <Select
                value={selectedUserPlugin?.id || ""}
                onValueChange={(value) => {
                  const plugin = userPlugins.find(p => p.id === value);
                  setSelectedUserPlugin(plugin || null);
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Choose a plugin instance..." />
                </SelectTrigger>
                <SelectContent>
                  {getAvailableUserPlugins().map((userPlugin) => (
                    <SelectItem key={userPlugin.id} value={userPlugin.id}>
                      <div>
                        <div className="font-medium">{userPlugin.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {userPlugin.plugin?.name || "Unknown Plugin"}
                        </div>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowAddDialog(false);
                setSelectedUserPlugin(null);
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={addPlaylistItem}
              disabled={!selectedUserPlugin || addLoading}
            >
              {addLoading ? "Adding..." : "Add Item"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}