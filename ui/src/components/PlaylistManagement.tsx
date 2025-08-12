import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/components/AuthProvider";
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
import { Checkbox } from "@/components/ui/checkbox";
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
  PlayCircle,
  Plus,
  Edit,
  Trash2,
  Star,
  Clock,
  ArrowUp,
  ArrowDown,
  Calendar,
  CheckCircle,
  AlertTriangle,
  Settings as SettingsIcon,
} from "lucide-react";

interface Device {
  id: string;
  user_id?: string;
  mac_address: string;
  friendly_id: string;
  name?: string;
  api_key: string;
  is_claimed: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface Plugin {
  id: string;
  name: string;
  type: string;
  description: string;
  config_schema: string;
  version: string;
  author?: string;
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

interface Schedule {
  id: string;
  playlist_item_id: string;
  name?: string;
  day_mask: number;
  start_time: string;
  end_time: string;
  timezone: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
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
  schedules?: Schedule[];
}

interface Playlist {
  id: string;
  user_id: string;
  device_id: string;
  name: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
  device?: Device;
  playlist_items?: PlaylistItem[];
}

interface PlaylistManagementProps {
  isOpen: boolean;
  onClose: () => void;
}

export function PlaylistManagement({ isOpen, onClose }: PlaylistManagementProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [userPlugins, setUserPlugins] = useState<UserPlugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Playlist creation
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [playlistName, setPlaylistName] = useState("");
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");
  const [isDefaultPlaylist, setIsDefaultPlaylist] = useState(false);

  // Playlist editing
  const [editPlaylist, setEditPlaylist] = useState<Playlist | null>(null);
  const [editPlaylistName, setEditPlaylistName] = useState("");
  const [editIsDefaultPlaylist, setEditIsDefaultPlaylist] = useState(false);

  // Playlist deletion
  const [deletePlaylist, setDeletePlaylist] = useState<Playlist | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Selected playlist for detailed view
  const [selectedPlaylist, setSelectedPlaylist] = useState<Playlist | null>(null);
  const [playlistItems, setPlaylistItems] = useState<PlaylistItem[]>([]);

  // Add item to playlist
  const [showAddItemDialog, setShowAddItemDialog] = useState(false);
  const [selectedUserPluginId, setSelectedUserPluginId] = useState<string>("");
  const [itemImportance, setItemImportance] = useState(0);
  const [durationOverride, setDurationOverride] = useState("");

  useEffect(() => {
    if (isOpen) {
      fetchPlaylists();
      fetchDevices();
      fetchUserPlugins();
    }
  }, [isOpen]);

  useEffect(() => {
    // Clear success/error messages after 5 seconds
    if (successMessage || error) {
      const timer = setTimeout(() => {
        setSuccessMessage(null);
        setError(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [successMessage, error]);

  const fetchPlaylists = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch("/api/playlists", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setPlaylists(data.playlists || []);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to fetch playlists");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const fetchDevices = async () => {
    try {
      const response = await fetch("/api/devices", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setDevices(data.devices || []);
      }
    } catch (error) {
      console.error("Failed to fetch devices:", error);
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

  const fetchPlaylistItems = async (playlistId: string) => {
    try {
      const response = await fetch(`/api/playlists/${playlistId}`, {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setPlaylistItems(data.items || []);
      }
    } catch (error) {
      console.error("Failed to fetch playlist items:", error);
    }
  };

  const createPlaylist = async () => {
    if (!playlistName.trim() || !selectedDeviceId) {
      setError("Please fill in all fields");
      return;
    }

    try {
      setCreateLoading(true);
      setError(null);

      const response = await fetch("/api/playlists", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          device_id: selectedDeviceId,
          name: playlistName.trim(),
          is_default: isDefaultPlaylist,
        }),
      });

      if (response.ok) {
        setSuccessMessage("Playlist created successfully!");
        setShowCreateDialog(false);
        setPlaylistName("");
        setSelectedDeviceId("");
        setIsDefaultPlaylist(false);
        await fetchPlaylists();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to create playlist");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setCreateLoading(false);
    }
  };

  const updatePlaylist = async () => {
    if (!editPlaylist || !editPlaylistName.trim()) {
      setError("Please fill in all fields");
      return;
    }

    try {
      setError(null);

      const response = await fetch(`/api/playlists/${editPlaylist.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: editPlaylistName.trim(),
          is_default: editIsDefaultPlaylist,
        }),
      });

      if (response.ok) {
        setSuccessMessage("Playlist updated successfully!");
        setEditPlaylist(null);
        await fetchPlaylists();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update playlist");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const confirmDeletePlaylist = async () => {
    if (!deletePlaylist) return;

    try {
      setDeleteLoading(true);
      setError(null);

      const response = await fetch(`/api/playlists/${deletePlaylist.id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        setSuccessMessage("Playlist deleted successfully!");
        setDeletePlaylist(null);
        if (selectedPlaylist?.id === deletePlaylist.id) {
          setSelectedPlaylist(null);
          setPlaylistItems([]);
        }
        await fetchPlaylists();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete playlist");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setDeleteLoading(false);
    }
  };

  const addItemToPlaylist = async () => {
    if (!selectedPlaylist || !selectedUserPluginId) {
      setError("Please select a plugin");
      return;
    }

    try {
      setError(null);

      const body: any = {
        user_plugin_id: selectedUserPluginId,
        importance: itemImportance,
      };

      if (durationOverride.trim()) {
        const duration = parseInt(durationOverride);
        if (!isNaN(duration) && duration > 0) {
          body.duration_override = duration;
        }
      }

      const response = await fetch(`/api/playlists/${selectedPlaylist.id}/items`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(body),
      });

      if (response.ok) {
        setSuccessMessage("Item added to playlist successfully!");
        setShowAddItemDialog(false);
        setSelectedUserPluginId("");
        setItemImportance(0);
        setDurationOverride("");
        await fetchPlaylistItems(selectedPlaylist.id);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to add item to playlist");
      }
    } catch (error) {
      setError("Network error occurred");
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
        if (selectedPlaylist) {
          await fetchPlaylistItems(selectedPlaylist.id);
        }
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to remove item from playlist");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const openEditDialog = (playlist: Playlist) => {
    setEditPlaylist(playlist);
    setEditPlaylistName(playlist.name);
    setEditIsDefaultPlaylist(playlist.is_default);
  };

  const selectPlaylist = (playlist: Playlist) => {
    setSelectedPlaylist(playlist);
    fetchPlaylistItems(playlist.id);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getDeviceName = (deviceId: string) => {
    const device = devices.find(d => d.id === deviceId);
    return device?.name || device?.friendly_id || "Unknown Device";
  };

  const getAvailableUserPlugins = () => {
    if (!selectedPlaylist) return userPlugins;
    
    const usedPluginIds = playlistItems.map(item => item.user_plugin_id);
    return userPlugins.filter(plugin => !usedPluginIds.includes(plugin.id));
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-7xl mobile-dialog-content sm:max-w-7xl overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <PlayCircle className="h-5 w-5" />
            Playlist Management
          </DialogTitle>
          <DialogDescription>
            Manage your device playlists, add plugins, and configure schedules.
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

        <Tabs defaultValue="playlists" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="playlists">Playlists</TabsTrigger>
            <TabsTrigger value="items" disabled={!selectedPlaylist}>
              Items {selectedPlaylist && `(${selectedPlaylist.name})`}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="playlists" className="space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Your Playlists</h3>
              <Button
                onClick={() => setShowCreateDialog(true)}
                className="flex items-center gap-2"
                disabled={devices.length === 0}
              >
                <Plus className="h-4 w-4" />
                Create Playlist
              </Button>
            </div>

            {devices.length === 0 && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  You need to claim at least one device before creating playlists.
                </AlertDescription>
              </Alert>
            )}

            {loading ? (
              <div className="text-center py-8">Loading playlists...</div>
            ) : playlists.length === 0 ? (
              <Card>
                <CardContent className="text-center py-8">
                  <PlayCircle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <h3 className="text-lg font-semibold mb-2">No Playlists Created</h3>
                  <p className="text-muted-foreground mb-4">
                    Create your first playlist to organize content for your devices.
                  </p>
                  <Button 
                    onClick={() => setShowCreateDialog(true)}
                    disabled={devices.length === 0}
                  >
                    <Plus className="h-4 w-4 mr-2" />
                    Create Playlist
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {playlists.map((playlist) => (
                  <Card 
                    key={playlist.id} 
                    className={`cursor-pointer transition-colors hover:bg-accent ${
                      selectedPlaylist?.id === playlist.id ? 'ring-2 ring-primary' : ''
                    }`}
                    onClick={() => selectPlaylist(playlist)}
                  >
                    <CardHeader className="pb-3">
                      <CardTitle className="flex items-start justify-between gap-3">
                        <div className="flex items-center gap-2">
                          <span className="text-base">{playlist.name}</span>
                          {playlist.is_default && (
                            <Badge variant="secondary" className="bg-yellow-100 text-yellow-800">
                              <Star className="h-3 w-3 mr-1" />
                              Default
                            </Badge>
                          )}
                        </div>
                        <div className="flex gap-1">
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation();
                              openEditDialog(playlist);
                            }}
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation();
                              setDeletePlaylist(playlist);
                            }}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <div className="space-y-2 text-sm text-muted-foreground">
                        <div>Device: {getDeviceName(playlist.device_id)}</div>
                        <div>Items: {playlist.playlist_items?.length || 0}</div>
                        <div>Created: {new Date(playlist.created_at).toLocaleDateString()}</div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>

          <TabsContent value="items" className="space-y-4">
            {selectedPlaylist && (
              <>
                <div className="flex justify-between items-center">
                  <div>
                    <h3 className="text-lg font-semibold">{selectedPlaylist.name} - Items</h3>
                    <p className="text-muted-foreground">
                      Device: {getDeviceName(selectedPlaylist.device_id)}
                    </p>
                  </div>
                  <Button
                    onClick={() => setShowAddItemDialog(true)}
                    className="flex items-center gap-2"
                    disabled={getAvailableUserPlugins().length === 0}
                  >
                    <Plus className="h-4 w-4" />
                    Add Item
                  </Button>
                </div>

                {getAvailableUserPlugins().length === 0 && (
                  <Alert>
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription>
                      No available plugins to add. Create plugin instances first or all plugins are already in this playlist.
                    </AlertDescription>
                  </Alert>
                )}

                {playlistItems.length === 0 ? (
                  <Card>
                    <CardContent className="text-center py-8">
                      <SettingsIcon className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                      <h3 className="text-lg font-semibold mb-2">No Items in Playlist</h3>
                      <p className="text-muted-foreground mb-4">
                        Add plugin instances to this playlist to display content on your device.
                      </p>
                      <Button 
                        onClick={() => setShowAddItemDialog(true)}
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
                            <TableHead>Instance Name</TableHead>
                            <TableHead className="hidden sm:table-cell">Importance</TableHead>
                            <TableHead className="hidden sm:table-cell">Duration</TableHead>
                            <TableHead className="hidden lg:table-cell">Schedules</TableHead>
                            <TableHead>Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {playlistItems
                            .sort((a, b) => a.order_index - b.order_index)
                            .map((item) => (
                              <TableRow key={item.id}>
                                <TableCell>
                                  <Badge variant="outline">#{item.order_index}</Badge>
                                </TableCell>
                                <TableCell>
                                  <div>
                                    <div className="font-medium">
                                      {item.user_plugin?.plugin?.name || "Unknown Plugin"}
                                    </div>
                                    <div className="text-sm text-muted-foreground">
                                      {item.user_plugin?.plugin?.type || "unknown"}
                                    </div>
                                  </div>
                                </TableCell>
                                <TableCell>
                                  {item.user_plugin?.name || "Unnamed Instance"}
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
                                      onClick={() => removePlaylistItem(item.id)}
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
              </>
            )}
          </TabsContent>
        </Tabs>
      </DialogContent>

      {/* Create Playlist Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create Playlist</DialogTitle>
            <DialogDescription>
              Create a new playlist for one of your devices.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="playlist-name">Playlist Name</Label>
              <Input
                id="playlist-name"
                value={playlistName}
                onChange={(e) => setPlaylistName(e.target.value)}
                placeholder="e.g., Morning News"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="device-select">Device</Label>
              <Select value={selectedDeviceId} onValueChange={setSelectedDeviceId}>
                <SelectTrigger className="mt-2">
                  <SelectValue placeholder="Select a device" />
                </SelectTrigger>
                <SelectContent>
                  {devices.map((device) => (
                    <SelectItem key={device.id} value={device.id}>
                      {device.name || device.friendly_id} ({device.friendly_id})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="is-default"
                checked={isDefaultPlaylist}
                onCheckedChange={(checked) => setIsDefaultPlaylist(checked as boolean)}
              />
              <Label htmlFor="is-default">Set as default playlist for this device</Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={createPlaylist}
              disabled={createLoading || !playlistName.trim() || !selectedDeviceId}
            >
              {createLoading ? "Creating..." : "Create Playlist"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Playlist Dialog */}
      <Dialog open={!!editPlaylist} onOpenChange={() => setEditPlaylist(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Playlist</DialogTitle>
            <DialogDescription>
              Update playlist settings and configuration.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="edit-playlist-name">Playlist Name</Label>
              <Input
                id="edit-playlist-name"
                value={editPlaylistName}
                onChange={(e) => setEditPlaylistName(e.target.value)}
                placeholder="e.g., Morning News"
                className="mt-2"
              />
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="edit-is-default"
                checked={editIsDefaultPlaylist}
                onCheckedChange={(checked) => setEditIsDefaultPlaylist(checked as boolean)}
              />
              <Label htmlFor="edit-is-default">Set as default playlist for this device</Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditPlaylist(null)}>
              Cancel
            </Button>
            <Button
              onClick={updatePlaylist}
              disabled={!editPlaylistName.trim()}
            >
              Update Playlist
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Playlist Dialog */}
      <Dialog open={!!deletePlaylist} onOpenChange={() => setDeletePlaylist(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Playlist
            </DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deletePlaylist?.name}"? This will remove all items and schedules. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDeletePlaylist(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDeletePlaylist}
              disabled={deleteLoading}
            >
              {deleteLoading ? "Deleting..." : "Delete Playlist"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add Item Dialog */}
      <Dialog open={showAddItemDialog} onOpenChange={setShowAddItemDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add Item to Playlist</DialogTitle>
            <DialogDescription>
              Add a plugin instance to this playlist.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="plugin-select">Plugin Instance</Label>
              <Select value={selectedUserPluginId} onValueChange={setSelectedUserPluginId}>
                <SelectTrigger className="mt-2">
                  <SelectValue placeholder="Select a plugin instance" />
                </SelectTrigger>
                <SelectContent>
                  {getAvailableUserPlugins().map((userPlugin) => (
                    <SelectItem key={userPlugin.id} value={userPlugin.id}>
                      {userPlugin.name} ({userPlugin.plugin.name})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="importance-select">Importance</Label>
              <Select value={itemImportance.toString()} onValueChange={(value) => setItemImportance(parseInt(value))}>
                <SelectTrigger className="mt-2">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">Normal</SelectItem>
                  <SelectItem value="1">Important</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="duration-override">Duration Override (seconds)</Label>
              <Input
                id="duration-override"
                type="number"
                value={durationOverride}
                onChange={(e) => setDurationOverride(e.target.value)}
                placeholder="Optional - leave empty for default"
                className="mt-2"
              />
              <p className="text-sm text-muted-foreground mt-1">
                Override the default display duration for this item
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddItemDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={addItemToPlaylist}
              disabled={!selectedUserPluginId}
            >
              Add Item
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}