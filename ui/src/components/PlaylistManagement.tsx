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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
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
  const { user } = useAuth();
  const [playlistItems, setPlaylistItems] = useState<PlaylistItem[]>([]);

  // Get user's timezone or fall back to browser timezone
  const getUserTimezone = () => {
    return user?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
  };

  // Convert local time (HH:MM) to UTC time for database storage
  const convertLocalTimeToUTC = (localTime: string): string => {
    const timezone = getUserTimezone();
    const today = new Date().toISOString().split('T')[0]; // Today in YYYY-MM-DD
    
    // Parse the time input as if it's in the user's timezone
    const [hours, minutes] = localTime.split(':').map(Number);
    
    // Create a date in user's timezone (using a reference date to handle DST correctly)
    const localDate = new Date();
    localDate.setFullYear(parseInt(today.split('-')[0]));
    localDate.setMonth(parseInt(today.split('-')[1]) - 1); // Month is 0-indexed
    localDate.setDate(parseInt(today.split('-')[2]));
    localDate.setHours(hours, minutes, 0, 0);
    
    // Convert to UTC
    const utcTime = localDate.toISOString().substring(11, 19); // Extract HH:MM:SS
    return utcTime;
  };

  // Convert UTC time (HH:MM:SS) from database to local time for display
  const convertUTCTimeToLocal = (utcTime: string): string => {
    const today = new Date().toISOString().split('T')[0]; // Today in YYYY-MM-DD
    const utcDateTime = `${today}T${utcTime}Z`;
    const utcDate = new Date(utcDateTime);
    
    // Convert to local time using the browser's timezone
    const localTime = utcDate.toLocaleTimeString('en-GB', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit'
    });
    
    return localTime; // Returns HH:MM
  };
  const [userPlugins, setUserPlugins] = useState<UserPlugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Add item dialog
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [selectedUserPlugin, setSelectedUserPlugin] = useState<UserPlugin | null>(null);
  const [addLoading, setAddLoading] = useState(false);

  // Edit item state (now used in schedule dialog)
  const [editImportance, setEditImportance] = useState<number>(0);
  const [editDurationOverride, setEditDurationOverride] = useState<string>("");

  // Schedule management dialog
  const [showScheduleDialog, setShowScheduleDialog] = useState(false);
  const [scheduleItem, setScheduleItem] = useState<PlaylistItem | null>(null);
  const [schedules, setSchedules] = useState<any[]>([]);
  const [scheduleLoading, setScheduleLoading] = useState(false);

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

  const openScheduleDialog = async (item: PlaylistItem) => {
    setScheduleItem(item);
    
    // Convert UTC times from database to local times for display
    const schedulesWithLocalTimes = (item.schedules || []).map(schedule => ({
      ...schedule,
      start_time: convertUTCTimeToLocal(schedule.start_time) + ":00", // Add seconds for UI
      end_time: convertUTCTimeToLocal(schedule.end_time) + ":00", // Add seconds for UI
    }));
    
    setSchedules(schedulesWithLocalTimes);
    
    // Also load the edit data for importance and duration
    setEditImportance(item.importance);
    setEditDurationOverride(item.duration_override ? item.duration_override.toString() : "");
    setShowScheduleDialog(true);
  };

  const saveSchedules = async () => {
    if (!scheduleItem) return;

    try {
      setScheduleLoading(true);
      setError(null);

      // First, delete all existing schedules
      const existingSchedules = scheduleItem.schedules || [];
      for (const existingSchedule of existingSchedules) {
        if (existingSchedule.id && !existingSchedule.id.startsWith('temp-')) {
          await fetch(`/api/playlists/schedules/${existingSchedule.id}`, {
            method: "DELETE",
            credentials: "include",
          });
        }
      }

      // Then create all new schedules
      for (const schedule of schedules) {
        if (!schedule.is_active) continue; // Skip inactive schedules

        const scheduleData = {
          name: schedule.name || "Unnamed Schedule",
          day_mask: schedule.day_mask,
          start_time: convertLocalTimeToUTC(schedule.start_time.substring(0, 5)), // Convert to UTC
          end_time: convertLocalTimeToUTC(schedule.end_time.substring(0, 5)), // Convert to UTC
          timezone: "UTC",
        };

        const response = await fetch(`/api/playlists/items/${scheduleItem.id}/schedules`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify(scheduleData),
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.error || "Failed to save schedule");
        }
      }

      // Update playlist item settings (importance and duration)
      if (editImportance !== scheduleItem.importance || 
          editDurationOverride !== (scheduleItem.duration_override ? scheduleItem.duration_override.toString() : "")) {
        
        const updateData: any = {
          importance: editImportance,
        };

        // Only include duration_override if it's a valid number or null
        if (editDurationOverride.trim() === "") {
          updateData.duration_override = null;
        } else {
          const duration = parseInt(editDurationOverride);
          if (!isNaN(duration) && duration > 0) {
            updateData.duration_override = duration;
          }
        }

        const itemUpdateResponse = await fetch(`/api/playlists/items/${scheduleItem.id}`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify(updateData),
        });

        if (!itemUpdateResponse.ok) {
          const errorData = await itemUpdateResponse.json();
          throw new Error(errorData.error || "Failed to update playlist item settings");
        }
      }

      setSuccessMessage("Schedules and settings saved successfully!");
      setShowScheduleDialog(false);
      setScheduleItem(null);
      setSchedules([]);
      await fetchPlaylistItems(); // Refresh to get updated schedules
      onUpdate?.();
    } catch (error) {
      setError(error instanceof Error ? error.message : "Network error occurred");
    } finally {
      setScheduleLoading(false);
    }
  };

  const formatDuration = (seconds: number): string => {
    if (seconds < 60) {
      return `${seconds}s`;
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = seconds % 60;
      if (remainingSeconds === 0) {
        return `${minutes}m`;
      } else {
        return `${minutes}m ${remainingSeconds}s`;
      }
    } else {
      const hours = Math.floor(seconds / 3600);
      const remainingMinutes = Math.floor((seconds % 3600) / 60);
      const remainingSeconds = seconds % 60;
      
      let result = `${hours}h`;
      if (remainingMinutes > 0) {
        result += ` ${remainingMinutes}m`;
      }
      if (remainingSeconds > 0) {
        result += ` ${remainingSeconds}s`;
      }
      return result;
    }
  };

  const formatScheduleSummary = (schedules: any[]): string => {
    if (!schedules || schedules.length === 0) {
      return "Always active";
    }

    const activSchedules = schedules.filter(s => s.is_active);
    if (activSchedules.length === 0) {
      return "No active schedules";
    }

    // For display purposes, show the first schedule's summary
    const schedule = activSchedules[0];
    const dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    const selectedDays = [];

    for (let i = 0; i < 7; i++) {
      if (schedule.day_mask & (1 << i)) {
        selectedDays.push(dayNames[i]);
      }
    }

    const dayText = selectedDays.length === 7 ? "Daily" : 
                   selectedDays.length === 5 && !selectedDays.includes("Sat") && !selectedDays.includes("Sun") ? "Mon-Fri" :
                   selectedDays.length === 2 && selectedDays.includes("Sat") && selectedDays.includes("Sun") ? "Weekends" :
                   selectedDays.join(", ");

    // Convert UTC times from database to local times for display
    const startTimeLocal = schedule.start_time ? convertUTCTimeToLocal(schedule.start_time) : "09:00";
    const endTimeLocal = schedule.end_time ? convertUTCTimeToLocal(schedule.end_time) : "17:00";
    
    // Convert to 12-hour format for display
    const formatTime12 = (time24: string) => {
      const [hours, minutes] = time24.split(':');
      const hour = parseInt(hours);
      const ampm = hour >= 12 ? 'PM' : 'AM';
      const hour12 = hour % 12 || 12;
      return `${hour12}:${minutes}${ampm}`;
    };

    const timeRange = `${formatTime12(startTimeLocal)} - ${formatTime12(endTimeLocal)}`;
    
    return activSchedules.length > 1 ? 
      `${dayText} ${timeRange} +${activSchedules.length - 1} more` :
      `${dayText} ${timeRange}`;
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
          disabled={getAvailableUserPlugins().length === 0}
        >
          {getAvailableUserPlugins().length === 0 ? "All Plugins Added" : "Add Item"}
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
              {getAvailableUserPlugins().length === 0 ? "All Plugins Added" : "Add Item"}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent>
            <Table className="w-full table-fixed lg:table-auto">
              <TableHeader>
                <TableRow>
                  <TableHead>Order</TableHead>
                  <TableHead>Plugin</TableHead>
                  <TableHead className="hidden md:table-cell">Status</TableHead>
                  <TableHead className="hidden lg:table-cell">Importance</TableHead>
                  <TableHead className="hidden lg:table-cell">Duration</TableHead>
                  <TableHead className="hidden lg:table-cell">Schedules</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
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
                        <div className="text-sm text-muted-foreground">#{item.order_index}</div>
                      </TableCell>
                      <TableCell>
                        <div>
                          <div className="font-medium">
                            {item.user_plugin?.name || "Unnamed Instance"}
                          </div>
                          <div className="text-sm text-muted-foreground">
                            {item.user_plugin?.plugin?.name || "Unknown Plugin"}
                          </div>
                          <div className="text-xs text-muted-foreground md:hidden mt-1">
                            {isCurrentlyShowing ? "Now Showing" : item.is_visible ? "Visible" : "Hidden"} • {item.importance === 1 ? "Important" : "Normal"}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
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
                      <TableCell className="hidden lg:table-cell">
                        {item.importance === 1 ? (
                          <Badge variant="secondary" className="bg-orange-100 text-orange-800">
                            <Star className="h-3 w-3 mr-1" />
                            Important
                          </Badge>
                        ) : (
                          <Badge variant="outline">Normal</Badge>
                        )}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {item.duration_override ? formatDuration(item.duration_override) : "Default"}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        <div className="text-sm">
                          {formatScheduleSummary(item.schedules || [])}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center gap-2 justify-end">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => openScheduleDialog(item)}
                            title="Manage schedules & settings"
                          >
                            <Calendar className="h-4 w-4" />
                          </Button>
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

      {/* Schedule Management Dialog */}
      <Dialog open={showScheduleDialog} onOpenChange={setShowScheduleDialog}>
        <DialogContent className="sm:max-w-4xl max-h-[85vh] overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
          <DialogHeader>
            <DialogTitle>Manage Schedules & Settings</DialogTitle>
            <DialogDescription>
              Configure schedules and settings for "{scheduleItem?.user_plugin?.name}". 
              Multiple schedules can be created for different times and days.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-6">
            {/* Playlist Item Settings */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Playlist Item Settings</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor="importance">Importance Level</Label>
                    <Select
                      value={editImportance.toString()}
                      onValueChange={(value) => setEditImportance(parseInt(value))}
                    >
                      <SelectTrigger className="mt-2">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="0">Normal</SelectItem>
                        <SelectItem value="1">Important</SelectItem>
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground mt-1">
                      Important items are prioritized in the rotation.
                    </p>
                  </div>

                  <div>
                    <Label htmlFor="duration-override">Duration Override (seconds)</Label>
                    <Input
                      id="duration-override"
                      type="number"
                      min="60"
                      placeholder="Leave empty for device default"
                      value={editDurationOverride}
                      onChange={(e) => setEditDurationOverride(e.target.value)}
                      className="mt-2"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Override the device's default refresh rate for this item only. 
                      Leave empty to use the device's configured refresh rate.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
            {schedules.length === 0 ? (
              <Card>
                <CardContent className="text-center py-8">
                  <Calendar className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <h3 className="text-lg font-semibold mb-2">No Custom Schedules</h3>
                  <p className="text-muted-foreground mb-4">
                    This item will display during all active hours using your device's default refresh rate.
                  </p>
                  <Button 
                    onClick={() => {
                      // Add a default schedule (times are in local time for UI display)
                      const defaultSchedule = {
                        id: `temp-${Date.now()}`,
                        name: "Daily Schedule",
                        day_mask: 127, // All days (1+2+4+8+16+32+64)
                        start_time: "09:00:00", // Local time for display
                        end_time: "17:00:00", // Local time for display
                        is_active: true
                      };
                      setSchedules([defaultSchedule]);
                    }}
                  >
                    Create First Schedule
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-4">
                {schedules.map((schedule, index) => (
                  <Card key={schedule.id || index}>
                    <CardHeader className="pb-3">
                      <div className="flex items-center justify-between">
                        <CardTitle className="text-base">
                          {schedule.name || `Schedule ${index + 1}`}
                        </CardTitle>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSchedules(schedules.filter((_, i) => i !== index));
                          }}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <Label>Schedule Name</Label>
                          <Input
                            value={schedule.name || ""}
                            onChange={(e) => {
                              const updated = [...schedules];
                              updated[index] = { ...updated[index], name: e.target.value };
                              setSchedules(updated);
                            }}
                            placeholder="e.g., Work Hours, Weekend Only"
                            className="mt-2"
                          />
                        </div>

                        <div>
                          <Label>Days of Week</Label>
                          <div className="mt-2 grid grid-cols-7 gap-2">
                            {[
                              { name: "Sun", bit: 1 },
                              { name: "Mon", bit: 2 },
                              { name: "Tue", bit: 4 },
                              { name: "Wed", bit: 8 },
                              { name: "Thu", bit: 16 },
                              { name: "Fri", bit: 32 },
                              { name: "Sat", bit: 64 }
                            ].map(({ name, bit }) => (
                              <div key={name} className="flex flex-col items-center">
                                <Checkbox
                                  id={`${schedule.id}-${name}`}
                                  checked={(schedule.day_mask & bit) > 0}
                                  onCheckedChange={(checked) => {
                                    const updated = [...schedules];
                                    if (checked) {
                                      updated[index] = { ...updated[index], day_mask: schedule.day_mask | bit };
                                    } else {
                                      updated[index] = { ...updated[index], day_mask: schedule.day_mask & ~bit };
                                    }
                                    setSchedules(updated);
                                  }}
                                />
                                <Label 
                                  htmlFor={`${schedule.id}-${name}`}
                                  className="text-xs mt-1 cursor-pointer"
                                >
                                  {name}
                                </Label>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div className="flex flex-col gap-3">
                          <Label htmlFor={`start-time-${schedule.id}`} className="px-1">
                            Start Time
                          </Label>
                          <Input
                            type="time"
                            id={`start-time-${schedule.id}`}
                            value={schedule.start_time?.substring(0, 5) || "09:00"}
                            onChange={(e) => {
                              const updated = [...schedules];
                              updated[index] = { ...updated[index], start_time: `${e.target.value}:00` };
                              setSchedules(updated);
                            }}
                            className="bg-background appearance-none [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none"
                          />
                        </div>
                        <div className="flex flex-col gap-3">
                          <Label htmlFor={`end-time-${schedule.id}`} className="px-1">
                            End Time
                          </Label>
                          <Input
                            type="time"
                            id={`end-time-${schedule.id}`}
                            value={schedule.end_time?.substring(0, 5) || "17:00"}
                            onChange={(e) => {
                              const updated = [...schedules];
                              updated[index] = { ...updated[index], end_time: `${e.target.value}:00` };
                              setSchedules(updated);
                            }}
                            className="bg-background appearance-none [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none"
                          />
                        </div>
                      </div>

                      <div className="flex items-center space-x-2">
                        <Switch
                          id={`active-${schedule.id}`}
                          checked={schedule.is_active}
                          onCheckedChange={(checked) => {
                            const updated = [...schedules];
                            updated[index] = { ...updated[index], is_active: checked };
                            setSchedules(updated);
                          }}
                        />
                        <Label htmlFor={`active-${schedule.id}`}>
                          Schedule Active
                        </Label>
                      </div>
                    </CardContent>
                  </Card>
                ))}
                
                <Button
                  variant="outline"
                  onClick={() => {
                    const newSchedule = {
                      id: `temp-${Date.now()}`,
                      name: `Schedule ${schedules.length + 1}`,
                      day_mask: 127, // All days (1+2+4+8+16+32+64)
                      start_time: "09:00:00", // Local time for display
                      end_time: "17:00:00", // Local time for display
                      is_active: true
                    };
                    setSchedules([...schedules, newSchedule]);
                  }}
                >
                  + Add Another Schedule
                </Button>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowScheduleDialog(false);
                setScheduleItem(null);
                setSchedules([]);
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={saveSchedules}
              disabled={scheduleLoading}
            >
              {scheduleLoading ? "Saving..." : "Save Schedules & Settings"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
