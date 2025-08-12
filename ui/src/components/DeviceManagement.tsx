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
  BatteryFull,
  BatteryMedium,
  BatteryLow,
  BatteryWarning,
  Wifi,
  WifiOff,
  Calendar,
  AlertTriangle,
  CheckCircle,
  Clock,
  Settings as SettingsIcon,
  FileText,
} from "lucide-react";

interface Device {
  id: string;
  user_id?: string;
  mac_address: string;
  friendly_id: string;
  name?: string;
  api_key: string;
  is_claimed: boolean;
  firmware_version?: string;
  battery_voltage?: number;
  rssi?: number;
  refresh_rate: number;
  last_seen?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface DeviceLog {
  id: string;
  device_id: string;
  log_data: string;
  level: string;
  timestamp: string;
  created_at: string;
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

  // Device claiming
  const [showClaimDialog, setShowClaimDialog] = useState(false);
  const [claimLoading, setClaimLoading] = useState(false);
  const [friendlyId, setFriendlyId] = useState("");
  const [deviceName, setDeviceName] = useState("");

  // Device editing
  const [editDevice, setEditDevice] = useState<Device | null>(null);
  const [editDeviceName, setEditDeviceName] = useState("");
  const [editRefreshRate, setEditRefreshRate] = useState("");

  // Device deletion
  const [deleteDevice, setDeleteDevice] = useState<Device | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Device logs
  const [logsDevice, setLogsDevice] = useState<Device | null>(null);
  const [deviceLogs, setDeviceLogs] = useState<DeviceLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [totalLogsCount, setTotalLogsCount] = useState(0);
  const [logsOffset, setLogsOffset] = useState(0);
  const logsLimit = 50;

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

  const claimDevice = async () => {
    if (!friendlyId.trim() || !deviceName.trim()) {
      setError("Please fill in all fields");
      return;
    }

    try {
      setClaimLoading(true);
      setError(null);

      const response = await fetch("/api/devices/claim", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          friendly_id: friendlyId.trim(),
          name: deviceName.trim(),
        }),
      });

      if (response.ok) {
        setSuccessMessage("Device claimed successfully!");
        setShowClaimDialog(false);
        setFriendlyId("");
        setDeviceName("");
        await fetchDevices();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to claim device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setClaimLoading(false);
    }
  };

  const updateDevice = async () => {
    if (!editDevice || !editDeviceName.trim()) {
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
          name: editDeviceName.trim(),
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
    setEditDeviceName(device.name || "");
    setEditRefreshRate(device.refresh_rate.toString());
  };

  const fetchDeviceLogs = async (device: Device, offset = 0) => {
    try {
      setLogsLoading(true);
      setError(null);

      const response = await fetch(`/api/devices/${device.id}/logs?limit=${logsLimit}&offset=${offset}`, {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setDeviceLogs(data.logs || []);
        setTotalLogsCount(data.total_count || 0);
        setLogsOffset(offset);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to fetch device logs");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLogsLoading(false);
    }
  };

  const openLogsDialog = (device: Device) => {
    setLogsDevice(device);
    setDeviceLogs([]);
    setLogsOffset(0);
    fetchDeviceLogs(device, 0);
  };

  const formatLogData = (logData: string) => {
    try {
      const parsed = JSON.parse(logData);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return logData;
    }
  };

  const getLevelBadgeColor = (level: string) => {
    switch (level.toLowerCase()) {
      case "error":
        return "bg-destructive";
      case "warn":
      case "warning":
        return "bg-amber-600";
      case "debug":
        return "bg-slate-600";
      default:
        return "bg-blue-600";
    }
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

  const calculateBatteryPercentage = (voltage: number): number => {
    // Lithium battery voltage ranges: 2.75V (0%) to 4.2V (100%)
    if (voltage >= 4.2) return 100;
    if (voltage <= 2.75) return 0;
    
    // Linear interpolation between voltage thresholds
    if (voltage > 3.95) return Math.round(75 + ((voltage - 3.95) / (4.2 - 3.95)) * 25);
    if (voltage > 3.55) return Math.round(50 + ((voltage - 3.55) / (3.95 - 3.55)) * 25);
    if (voltage > 3.15) return Math.round(25 + ((voltage - 3.15) / (3.55 - 3.15)) * 25);
    return Math.round((voltage - 2.75) / (3.15 - 2.75) * 25);
  };

  const getSignalQuality = (rssi: number): { quality: string; strength: number; color: string } => {
    if (rssi > -50) return { quality: "Excellent", strength: 5, color: "text-emerald-600" };
    if (rssi > -60) return { quality: "Good", strength: 4, color: "text-emerald-600" };
    if (rssi > -70) return { quality: "Fair", strength: 3, color: "text-amber-600" };
    if (rssi > -80) return { quality: "Poor", strength: 2, color: "text-amber-600" };
    return { quality: "Very Poor", strength: 1, color: "text-destructive" };
  };

  const getStatusBadge = (device: Device) => {
    const status = getDeviceStatus(device);
    
    switch (status) {
      case "online":
        return <Badge variant="default" className="bg-emerald-600">Online</Badge>;
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

  const getBatteryDisplay = (voltage?: number) => {
    if (!voltage) {
      return {
        icon: <Battery className="h-4 w-4 text-muted-foreground" />,
        text: "N/A",
        tooltip: "Battery status unknown"
      };
    }
    
    const percentage = calculateBatteryPercentage(voltage);
    let icon;
    let color;
    
    if (percentage > 75) {
      icon = <BatteryFull className="h-4 w-4 text-emerald-600" />;
      color = "text-emerald-600";
    } else if (percentage > 50) {
      icon = <BatteryMedium className="h-4 w-4 text-emerald-600" />;
      color = "text-emerald-600";
    } else if (percentage > 25) {
      icon = <BatteryLow className="h-4 w-4 text-amber-600" />;
      color = "text-amber-600";
    } else {
      icon = <BatteryWarning className="h-4 w-4 text-destructive" />;
      color = "text-destructive";
    }
    
    return {
      icon,
      text: `${percentage}% (${voltage.toFixed(1)}V)`,
      tooltip: `Battery Level: ${percentage}% (${voltage.toFixed(1)}V)`,
      color
    };
  };

  const getSignalDisplay = (rssi?: number) => {
    if (!rssi) {
      return {
        icon: <WifiOff className="h-4 w-4 text-muted-foreground" />,
        text: "N/A",
        tooltip: "Signal strength unknown"
      };
    }
    
    const { quality, color } = getSignalQuality(rssi);
    
    return {
      icon: <Wifi className={`h-4 w-4 ${color}`} />,
      text: quality,
      tooltip: `Signal Quality: ${quality} (${rssi}dBm)`,
      color
    };
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-7xl mobile-dialog-content sm:max-w-7xl overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
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
              onClick={() => setShowClaimDialog(true)}
              className="flex items-center gap-2"
            >
              <Plus className="h-4 w-4" />
              Claim Device
            </Button>
          </div>

          {loading ? (
            <div className="text-center py-8">Loading devices...</div>
          ) : devices.length === 0 ? (
            <Card>
              <CardContent className="text-center py-8">
                <Monitor className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No Devices Claimed</h3>
                <p className="text-muted-foreground mb-4">
                  Claim your first TRMNL device to get started.
                </p>
                <Button onClick={() => setShowClaimDialog(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Claim Device
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
                            <div className="font-medium">{device.name || "Unnamed Device"}</div>
                            <div className="text-sm text-muted-foreground">ID: {device.friendly_id}</div>
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
                                {(() => {
                                  const battery = getBatteryDisplay(device.battery_voltage);
                                  return (
                                    <>
                                      {battery.icon}
                                      <span className={`text-sm ${battery.color}`}>
                                        {battery.text}
                                      </span>
                                    </>
                                  );
                                })()}
                              </div>
                            </TooltipTrigger>
                            <TooltipContent>
                              {getBatteryDisplay(device.battery_voltage).tooltip}
                            </TooltipContent>
                          </Tooltip>
                        </TableCell>
                        <TableCell className="hidden sm:table-cell">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div className="flex items-center gap-1">
                                {(() => {
                                  const signal = getSignalDisplay(device.rssi);
                                  return (
                                    <>
                                      {signal.icon}
                                      <span className={`text-sm ${signal.color}`}>
                                        {signal.text}
                                      </span>
                                    </>
                                  );
                                })()}
                              </div>
                            </TooltipTrigger>
                            <TooltipContent>
                              {getSignalDisplay(device.rssi).tooltip}
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
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => openLogsDialog(device)}
                                >
                                  <FileText className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>View Logs</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => openEditDialog(device)}
                                >
                                  <Edit className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit Device</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => setDeleteDevice(device)}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Delete Device</TooltipContent>
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
        </div>
      </DialogContent>

      {/* Claim Device Dialog */}
      <Dialog open={showClaimDialog} onOpenChange={setShowClaimDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Claim Device</DialogTitle>
            <DialogDescription>
              Enter your TRMNL device's friendly ID to claim it to your account.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="friendly-id">Device Friendly ID</Label>
              <Input
                id="friendly-id"
                value={friendlyId}
                onChange={(e) => setFriendlyId(e.target.value.toUpperCase())}
                placeholder="e.g., 917F0B"
                className="mt-2"
                maxLength={6}
              />
              <p className="text-sm text-muted-foreground mt-1">
                Find this 6-character ID on your device's setup screen
              </p>
            </div>
            <div>
              <Label htmlFor="device-name">Device Name</Label>
              <Input
                id="device-name"
                value={deviceName}
                onChange={(e) => setDeviceName(e.target.value)}
                placeholder="e.g., Kitchen Display"
                className="mt-2"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowClaimDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={claimDevice}
              disabled={claimLoading || !friendlyId.trim() || !deviceName.trim()}
            >
              {claimLoading ? "Claiming..." : "Claim Device"}
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
              <Label htmlFor="edit-device-name">Device Name</Label>
              <Input
                id="edit-device-name"
                value={editDeviceName}
                onChange={(e) => setEditDeviceName(e.target.value)}
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
              <p className="text-sm text-muted-foreground mt-1">
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
              disabled={!editDeviceName.trim()}
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
              Are you sure you want to delete "{deleteDevice?.name || 'this device'}"? This will remove all associated playlists and settings. This action cannot be undone.
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

      {/* Device Logs Dialog */}
      <Dialog open={!!logsDevice} onOpenChange={() => setLogsDevice(null)}>
        <DialogContent className="max-w-6xl mobile-dialog-content sm:max-w-6xl overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              Device Logs - {logsDevice?.name || "Unnamed Device"}
            </DialogTitle>
            <DialogDescription>
              View logs submitted by your device. Logs are displayed in reverse chronological order.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {logsLoading ? (
              <div className="text-center py-8">Loading logs...</div>
            ) : deviceLogs.length === 0 ? (
              <Card>
                <CardContent className="text-center py-8">
                  <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <h3 className="text-lg font-semibold mb-2">No Logs Found</h3>
                  <p className="text-muted-foreground">
                    This device hasn't submitted any logs yet.
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-4">
                <div className="flex justify-between items-center">
                  <p className="text-sm text-muted-foreground">
                    Showing {logsOffset + 1}-{Math.min(logsOffset + logsLimit, totalLogsCount)} of {totalLogsCount} logs
                  </p>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => fetchDeviceLogs(logsDevice!, Math.max(0, logsOffset - logsLimit))}
                      disabled={logsOffset === 0 || logsLoading}
                    >
                      Previous
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => fetchDeviceLogs(logsDevice!, logsOffset + logsLimit)}
                      disabled={logsOffset + logsLimit >= totalLogsCount || logsLoading}
                    >
                      Next
                    </Button>
                  </div>
                </div>

                <Card>
                  <CardContent className="p-0">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Timestamp</TableHead>
                          <TableHead>Level</TableHead>
                          <TableHead>Log Data</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {deviceLogs.map((log) => (
                          <TableRow key={log.id}>
                            <TableCell>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="cursor-default">
                                    {new Date(log.timestamp).toLocaleString()}
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent>
                                  {formatDate(log.timestamp)}
                                </TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell>
                              <Badge className={getLevelBadgeColor(log.level)}>
                                {log.level.toUpperCase()}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className="max-w-md">
                                <pre className="text-xs bg-muted p-2 rounded overflow-x-auto whitespace-pre-wrap">
                                  {formatLogData(log.log_data)}
                                </pre>
                              </div>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>

                <div className="flex justify-between items-center">
                  <p className="text-sm text-muted-foreground">
                    Showing {logsOffset + 1}-{Math.min(logsOffset + logsLimit, totalLogsCount)} of {totalLogsCount} logs
                  </p>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => fetchDeviceLogs(logsDevice!, Math.max(0, logsOffset - logsLimit))}
                      disabled={logsOffset === 0 || logsLoading}
                    >
                      Previous
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => fetchDeviceLogs(logsDevice!, logsOffset + logsLimit)}
                      disabled={logsOffset + logsLimit >= totalLogsCount || logsLoading}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setLogsDevice(null)}>
              Close
            </Button>
            <Button
              onClick={() => logsDevice && fetchDeviceLogs(logsDevice, logsOffset)}
              disabled={logsLoading}
            >
              {logsLoading ? "Refreshing..." : "Refresh"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}