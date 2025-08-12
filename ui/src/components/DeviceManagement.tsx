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
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Monitor,
  Plus,
  Edit,
  Trash2,
  Battery,
  Wifi,
  Calendar,
  AlertTriangle,
  CheckCircle,
  Clock,
  Settings as SettingsIcon,
} from "lucide-react";

interface Device {
  id: string;
  user_id: string;
  device_id: string;
  friendly_name: string;
  api_key: string;
  firmware_version?: string;
  battery_voltage?: number;
  rssi?: number;
  refresh_rate: number;
  last_seen?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface DeviceManagementProps {
  isOpen: boolean;
  onClose: () => void;
}

export function DeviceManagement({ isOpen, onClose }: DeviceManagementProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Device creation
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [deviceId, setDeviceId] = useState("");
  const [friendlyName, setFriendlyName] = useState("");

  // Device editing
  const [editDevice, setEditDevice] = useState<Device | null>(null);
  const [editFriendlyName, setEditFriendlyName] = useState("");
  const [editRefreshRate, setEditRefreshRate] = useState("");

  // Device deletion
  const [deleteDevice, setDeleteDevice] = useState<Device | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      fetchDevices();
    }
  }, [isOpen]);

  const fetchDevices = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch("/api/devices", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setDevices(data.devices || []);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to fetch devices");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const createDevice = async () => {
    if (!deviceId.trim() || !friendlyName.trim()) {
      setError("Please fill in all fields");
      return;
    }

    try {
      setCreateLoading(true);
      setError(null);

      const response = await fetch("/api/devices", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          device_id: deviceId.trim(),
          friendly_name: friendlyName.trim(),
        }),
      });

      if (response.ok) {
        setSuccessMessage("Device linked successfully!");
        setShowCreateDialog(false);
        setDeviceId("");
        setFriendlyName("");
        await fetchDevices();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to create device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setCreateLoading(false);
    }
  };

  const updateDevice = async () => {
    if (!editDevice || !editFriendlyName.trim()) {
      setError("Please fill in all fields");
      return;
    }

    const refreshRate = parseInt(editRefreshRate);
    if (isNaN(refreshRate) || refreshRate < 60 || refreshRate > 86400) {
      setError("Refresh rate must be between 60 and 86400 seconds");
      return;
    }

    try {
      setError(null);

      const response = await fetch(`/api/devices/${editDevice.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          friendly_name: editFriendlyName.trim(),
          refresh_rate: refreshRate,
        }),
      });

      if (response.ok) {
        setSuccessMessage("Device updated successfully!");
        setEditDevice(null);
        await fetchDevices();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update device");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const confirmDeleteDevice = async () => {
    if (!deleteDevice) return;

    try {
      setDeleteLoading(true);
      setError(null);

      const response = await fetch(`/api/devices/${deleteDevice.id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        setSuccessMessage("Device deleted successfully!");
        setDeleteDevice(null);
        await fetchDevices();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setDeleteLoading(false);
    }
  };

  const openEditDialog = (device: Device) => {
    setEditDevice(device);
    setEditFriendlyName(device.friendly_name);
    setEditRefreshRate(device.refresh_rate.toString());
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const formatLastSeen = (lastSeenString?: string) => {
    if (!lastSeenString) return "Never";
    
    const lastSeen = new Date(lastSeenString);
    const now = new Date();
    const diffMs = now.getTime() - lastSeen.getTime();
    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    
    if (diffMinutes < 5) return "Just now";
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}h ago`;
    return `${Math.floor(diffMinutes / 1440)}d ago`;
  };

  const getDeviceStatus = (device: Device) => {
    if (!device.is_active) return "inactive";
    if (!device.last_seen) return "never_connected";
    
    const lastSeen = new Date(device.last_seen);
    const now = new Date();
    const diffMinutes = (now.getTime() - lastSeen.getTime()) / (1000 * 60);
    
    if (diffMinutes < 60) return "online";
    if (diffMinutes < 1440) return "recently_online";
    return "offline";
  };

  const getStatusBadge = (device: Device) => {
    const status = getDeviceStatus(device);
    
    switch (status) {
      case "online":
        return <Badge variant="default" className="bg-green-500">Online</Badge>;
      case "recently_online":
        return <Badge variant="secondary">Recently Online</Badge>;
      case "offline":
        return <Badge variant="secondary">Offline</Badge>;
      case "inactive":
        return <Badge variant="secondary">Inactive</Badge>;
      default:
        return <Badge variant="outline">Never Connected</Badge>;
    }
  };

  const getBatteryIcon = (voltage?: number) => {
    if (!voltage) return <Battery className="h-4 w-4 text-gray-400" />;
    
    if (voltage > 3.8) return <Battery className="h-4 w-4 text-green-500" />;
    if (voltage > 3.4) return <Battery className="h-4 w-4 text-yellow-500" />;
    return <Battery className="h-4 w-4 text-red-500" />;
  };

  const getRSSIIcon = (rssi?: number) => {
    if (!rssi) return <Wifi className="h-4 w-4 text-gray-400" />;
    
    if (rssi > -50) return <Wifi className="h-4 w-4 text-green-500" />;
    if (rssi > -70) return <Wifi className="h-4 w-4 text-yellow-500" />;
    return <Wifi className="h-4 w-4 text-red-500" />;
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Monitor className="h-5 w-5" />
            Device Management
          </DialogTitle>
          <DialogDescription>
            Manage your TRMNL devices, view status, and configure settings.
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

        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-semibold">Your Devices</h3>
            <Button
              onClick={() => setShowCreateDialog(true)}
              className="flex items-center gap-2"
            >
              <Plus className="h-4 w-4" />
              Link Device
            </Button>
          </div>

          {loading ? (
            <div className="text-center py-8">Loading devices...</div>
          ) : devices.length === 0 ? (
            <Card>
              <CardContent className="text-center py-8">
                <Monitor className="h-12 w-12 mx-auto text-gray-400 mb-4" />
                <h3 className="text-lg font-semibold mb-2">No Devices Linked</h3>
                <p className="text-gray-600 mb-4">
                  Link your first TRMNL device to get started.
                </p>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Link Device
                </Button>
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Device</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="hidden sm:table-cell">Firmware</TableHead>
                      <TableHead className="hidden sm:table-cell">Battery</TableHead>
                      <TableHead className="hidden sm:table-cell">Signal</TableHead>
                      <TableHead className="hidden lg:table-cell">Last Seen</TableHead>
                      <TableHead className="hidden lg:table-cell">Created</TableHead>
                      <TableHead>Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {devices.map((device) => (
                      <TableRow key={device.id}>
                        <TableCell>
                          <div>
                            <div className="font-medium">{device.friendly_name}</div>
                            <div className="text-sm text-gray-600">{device.device_id}</div>
                          </div>
                        </TableCell>
                        <TableCell>{getStatusBadge(device)}</TableCell>
                        <TableCell className="hidden sm:table-cell">
                          {device.firmware_version || "Unknown"}
                        </TableCell>
                        <TableCell className="hidden sm:table-cell">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div className="flex items-center gap-1">
                                {getBatteryIcon(device.battery_voltage)}
                                <span className="text-sm">
                                  {device.battery_voltage ? `${device.battery_voltage.toFixed(1)}V` : "N/A"}
                                </span>
                              </div>
                            </TooltipTrigger>
                            <TooltipContent>
                              Battery Voltage: {device.battery_voltage ? `${device.battery_voltage}V` : "Unknown"}
                            </TooltipContent>
                          </Tooltip>
                        </TableCell>
                        <TableCell className="hidden sm:table-cell">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div className="flex items-center gap-1">
                                {getRSSIIcon(device.rssi)}
                                <span className="text-sm">
                                  {device.rssi ? `${device.rssi}dBm` : "N/A"}
                                </span>
                              </div>
                            </TooltipTrigger>
                            <TooltipContent>
                              Signal Strength: {device.rssi ? `${device.rssi}dBm` : "Unknown"}
                            </TooltipContent>
                          </Tooltip>
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="cursor-default">
                                {formatLastSeen(device.last_seen)}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent>
                              {device.last_seen ? formatDate(device.last_seen) : "Never connected"}
                            </TooltipContent>
                          </Tooltip>
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="cursor-default">
                                {new Date(device.created_at).toLocaleDateString()}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent>
                              {formatDate(device.created_at)}
                            </TooltipContent>
                          </Tooltip>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => openEditDialog(device)}
                            >
                              <Edit className="h-4 w-4" />
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => setDeleteDevice(device)}
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
        </div>
      </DialogContent>

      {/* Create Device Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Link New Device</DialogTitle>
            <DialogDescription>
              Enter your TRMNL device information to link it to your account.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="device-id">Device ID / MAC Address</Label>
              <Input
                id="device-id"
                value={deviceId}
                onChange={(e) => setDeviceId(e.target.value)}
                placeholder="e.g., AA:BB:CC:DD:EE:FF"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="friendly-name">Friendly Name</Label>
              <Input
                id="friendly-name"
                value={friendlyName}
                onChange={(e) => setFriendlyName(e.target.value)}
                placeholder="e.g., Kitchen Display"
                className="mt-2"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={createDevice}
              disabled={createLoading || !deviceId.trim() || !friendlyName.trim()}
            >
              {createLoading ? "Linking..." : "Link Device"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Device Dialog */}
      <Dialog open={!!editDevice} onOpenChange={() => setEditDevice(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Device</DialogTitle>
            <DialogDescription>
              Update device settings and configuration.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="edit-friendly-name">Friendly Name</Label>
              <Input
                id="edit-friendly-name"
                value={editFriendlyName}
                onChange={(e) => setEditFriendlyName(e.target.value)}
                placeholder="e.g., Kitchen Display"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="edit-refresh-rate">Refresh Rate (seconds)</Label>
              <Input
                id="edit-refresh-rate"
                type="number"
                min="60"
                max="86400"
                value={editRefreshRate}
                onChange={(e) => setEditRefreshRate(e.target.value)}
                className="mt-2"
              />
              <p className="text-sm text-gray-600 mt-1">
                How often the device should check for new content (60-86400 seconds)
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDevice(null)}>
              Cancel
            </Button>
            <Button
              onClick={updateDevice}
              disabled={!editFriendlyName.trim()}
            >
              Update Device
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Device Dialog */}
      <Dialog open={!!deleteDevice} onOpenChange={() => setDeleteDevice(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Device
            </DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deleteDevice?.friendly_name}"? This will remove all associated playlists and settings. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDevice(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDeleteDevice}
              disabled={deleteLoading}
            >
              {deleteLoading ? "Deleting..." : "Delete Device"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}