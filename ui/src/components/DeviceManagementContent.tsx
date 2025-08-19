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
import { Switch } from "@/components/ui/switch";
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
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Monitor,
  Edit,
  Trash2,
  Battery,
  BatteryFull,
  BatteryMedium,
  BatteryLow,
  BatteryWarning,
  Wifi,
  WifiOff,
  AlertTriangle,
  CheckCircle,
  FileText,
  ChevronDown,
} from "lucide-react";

interface DeviceModel {
  id: string;
  model_name: string;
  display_name: string;
  description?: string;
  screen_width: number;
  screen_height: number;
  color_depth: number;
  bit_depth: number;
  has_wifi: boolean;
  has_battery: boolean;
  has_buttons: number;
  capabilities?: string;
  min_firmware?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface Device {
  id: string;
  user_id?: string;
  mac_address: string;
  friendly_id: string;
  name?: string;
  model_name?: string;
  manual_model_override?: boolean;
  reported_model_name?: string;
  api_key: string;
  is_claimed: boolean;
  firmware_version?: string;
  battery_voltage?: number;
  rssi?: number;
  refresh_rate: number;
  allow_firmware_updates?: boolean;
  last_seen?: string;
  is_active: boolean;
  is_sharable?: boolean;
  mirror_source_id?: string;
  mirror_synced_at?: string;
  sleep_enabled?: boolean;
  sleep_start_time?: string;
  sleep_end_time?: string;
  sleep_show_screen?: boolean;
  created_at: string;
  updated_at: string;
  device_model?: DeviceModel;
}

interface DeviceLog {
  id: string;
  device_id: string;
  log_data: string;
  timestamp: string;
  created_at: string;
}

interface DeviceManagementContentProps {
  onUpdate?: () => void;
}

export function DeviceManagementContent({ onUpdate }: DeviceManagementContentProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Device claiming
  const [showClaimDialog, setShowClaimDialog] = useState(false);
  const [claimLoading, setClaimLoading] = useState(false);
  const [claimError, setClaimError] = useState<string | null>(null);
  const [friendlyId, setFriendlyId] = useState("");
  const [deviceName, setDeviceName] = useState("");

  // Device editing
  const [editDevice, setEditDevice] = useState<Device | null>(null);
  const [editDeviceName, setEditDeviceName] = useState("");
  const [editRefreshRate, setEditRefreshRate] = useState("");
  const [editAllowFirmwareUpdates, setEditAllowFirmwareUpdates] = useState(true);
  const [editModelName, setEditModelName] = useState<string>("");
  const [editIsSharable, setEditIsSharable] = useState(false);
  const [deviceModels, setDeviceModels] = useState<DeviceModel[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);

  // Sleep mode settings
  const [editSleepEnabled, setEditSleepEnabled] = useState(false);
  const [editSleepStartTime, setEditSleepStartTime] = useState("");
  const [editSleepEndTime, setEditSleepEndTime] = useState("");
  const [editSleepShowScreen, setEditSleepShowScreen] = useState(true);

  // Device deletion
  const [deleteDevice, setDeleteDevice] = useState<Device | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Model override confirmation
  const [showModelOverrideDialog, setShowModelOverrideDialog] = useState(false);
  const [pendingModelOverride, setPendingModelOverride] = useState<string>("");
  
  // Developer section collapsible state
  const [isDeveloperSectionOpen, setIsDeveloperSectionOpen] = useState(false);

  // Device logs
  const [logsDevice, setLogsDevice] = useState<Device | null>(null);
  const [deviceLogs, setDeviceLogs] = useState<DeviceLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [totalLogsCount, setTotalLogsCount] = useState(0);
  const [logsOffset, setLogsOffset] = useState(0);
  const logsLimit = 50;

  // Mirroring
  const [showMirrorDialog, setShowMirrorDialog] = useState(false);
  const [mirrorSourceFriendlyId, setMirrorSourceFriendlyId] = useState("");
  const [mirrorLoading, setMirrorLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);

  useEffect(() => {
    fetchDevices();
  }, []);

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
      setClaimError("Please fill in all fields");
      return;
    }

    try {
      setClaimLoading(true);
      setClaimError(null);

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
        setClaimError(null);
        await fetchDevices();
        onUpdate?.(); // Notify parent component to refresh
      } else {
        const errorData = await response.json();
        setClaimError(errorData.error || "Failed to claim device");
      }
    } catch (error) {
      setClaimError("Network error occurred");
    } finally {
      setClaimLoading(false);
    }
  };

  const fetchDeviceModels = async () => {
    try {
      setModelsLoading(true);
      const response = await fetch("/api/devices/models", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setDeviceModels(data.models || []);
      } else {
        console.error("Failed to fetch device models");
        setDeviceModels([]); // Set empty array on error
      }
    } catch (error) {
      console.error("Error fetching device models:", error);
      setDeviceModels([]); // Set empty array on error
    } finally {
      setModelsLoading(false);
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

      const requestBody: any = {
        name: editDeviceName.trim(),
        refresh_rate: refreshRate,
        allow_firmware_updates: editAllowFirmwareUpdates,
        is_sharable: editIsSharable,
        sleep_enabled: editSleepEnabled,
        sleep_start_time: editSleepStartTime,
        sleep_end_time: editSleepEndTime,
        sleep_show_screen: editSleepShowScreen,
      };


      // Add model name if it has changed from the original
      const originalModelName = editDevice.model_name || "none";
      if (editModelName !== originalModelName) {
        requestBody.model_name = editModelName === "none" ? null : editModelName;
      }

      const response = await fetch(`/api/devices/${editDevice.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(requestBody),
      });

      if (response.ok) {
        setSuccessMessage("Device updated successfully!");
        setEditDevice(null);
        await fetchDevices();
        onUpdate?.(); // Notify parent component to refresh device list
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update device");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const clearModelOverride = async () => {
    if (!editDevice) return;

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
          refresh_rate: parseInt(editRefreshRate),
          allow_firmware_updates: editAllowFirmwareUpdates,
          clear_model_override: true,
        }),
      });

      if (response.ok) {
        setSuccessMessage("Device model override cleared!");
        setEditDevice(null);
        await fetchDevices();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to clear model override");
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
        setSuccessMessage("Device unlinked successfully!");
        setDeleteDevice(null);
        await fetchDevices();
        onUpdate?.(); // Notify parent component to refresh
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to unlink device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setDeleteLoading(false);
    }
  };

  const openEditDialog = (device: Device) => {
    try {
      setEditDevice(device);
      setEditDeviceName(device.name || "");
      setEditRefreshRate(device.refresh_rate.toString());
      setEditAllowFirmwareUpdates(device.allow_firmware_updates ?? true);
      setEditModelName(device.model_name || "none");
      setEditIsSharable(device.is_sharable ?? false);
      
      // Initialize sleep mode settings
      setEditSleepEnabled(device.sleep_enabled ?? false);
      setEditSleepStartTime(device.sleep_start_time || "22:00");
      setEditSleepEndTime(device.sleep_end_time || "06:00");
      setEditSleepShowScreen(device.sleep_show_screen ?? true);
      
      // Fetch device models when opening edit dialog
      if (deviceModels.length === 0) {
        fetchDeviceModels().catch(error => {
          console.error("Failed to fetch device models:", error);
        });
      }
    } catch (error) {
      console.error("Error opening edit dialog:", error);
      setError("Failed to open device edit dialog");
    }
  };

  const hasDeviceChanges = () => {
    if (!editDevice) return false;
    return (
      editDeviceName.trim() !== (editDevice.name || "") ||
      editRefreshRate !== editDevice.refresh_rate.toString() ||
      editAllowFirmwareUpdates !== (editDevice.allow_firmware_updates ?? true) ||
      editModelName !== (editDevice.model_name || "none") ||
      editIsSharable !== (editDevice.is_sharable ?? false) ||
      editSleepEnabled !== (editDevice.sleep_enabled ?? false) ||
      editSleepStartTime !== (editDevice.sleep_start_time || "22:00") ||
      editSleepEndTime !== (editDevice.sleep_end_time || "06:00") ||
      editSleepShowScreen !== (editDevice.sleep_show_screen ?? true)
    );
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
    if (voltage >= 4.2) return 100;
    if (voltage <= 2.75) return 0;
    
    if (voltage >= 4.0) return Math.round(90 + ((voltage - 4.0) / (4.2 - 4.0)) * 10);
    if (voltage >= 3.7) return Math.round(75 + ((voltage - 3.7) / (4.0 - 3.7)) * 15);
    if (voltage >= 3.4) return Math.round(50 + ((voltage - 3.4) / (3.7 - 3.4)) * 25);
    if (voltage >= 3.0) return Math.round(25 + ((voltage - 3.0) / (3.4 - 3.0)) * 25);
    return Math.round((voltage - 2.75) / (3.0 - 2.75) * 25);
  };

  const getSignalQuality = (rssi: number): { quality: string; strength: number; color: string } => {
    if (rssi > -50) return { quality: "Excellent", strength: 5, color: "" };
    if (rssi > -60) return { quality: "Good", strength: 4, color: "" };
    if (rssi > -70) return { quality: "Fair", strength: 3, color: "" };
    if (rssi > -80) return { quality: "Poor", strength: 2, color: "text-destructive" };
    return { quality: "Very Poor", strength: 1, color: "text-destructive" };
  };

  const getStatusBadge = (device: Device) => {
    const status = getDeviceStatus(device);
    
    switch (status) {
      case "online":
        return <Badge variant="outline">Online</Badge>;
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
      icon = <BatteryFull className="h-4 w-4" />;
      color = "";
    } else if (percentage > 50) {
      icon = <BatteryMedium className="h-4 w-4" />;
      color = "";
    } else if (percentage > 25) {
      icon = <BatteryLow className="h-4 w-4 text-destructive" />;
      color = "text-destructive";
    } else {
      icon = <BatteryWarning className="h-4 w-4 text-destructive" />;
      color = "text-destructive";
    }
    
    return {
      icon,
      text: `${percentage}%`,
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

  const getModelDisplayName = (modelName: string) => {
    const model = deviceModels.find(m => m.model_name === modelName);
    return model ? model.display_name : modelName;
  };

  const handleModelChange = (value: string) => {
    if (!editDevice) return;
    
    // If user selects a different model than what device reports, show confirmation
    if (value !== "none" && editDevice.reported_model_name && value !== editDevice.reported_model_name) {
      setPendingModelOverride(value);
      setShowModelOverrideDialog(true);
    } else {
      setEditModelName(value);
    }
  };

  const confirmModelOverride = () => {
    setEditModelName(pendingModelOverride);
    setShowModelOverrideDialog(false);
    setPendingModelOverride("");
  };

  const cancelModelOverride = () => {
    setShowModelOverrideDialog(false);
    setPendingModelOverride("");
  };

  const mirrorDevice = async () => {
    if (!editDevice || !mirrorSourceFriendlyId.trim()) {
      setError("Please enter a device ID");
      return;
    }

    try {
      setMirrorLoading(true);
      setError(null);

      const response = await fetch(`/api/devices/${editDevice.id}/mirror`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          source_friendly_id: mirrorSourceFriendlyId.trim().toUpperCase(),
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setSuccessMessage(data.message);
        setShowMirrorDialog(false);
        setMirrorSourceFriendlyId("");
        await fetchDevices();
        
        // Update the editDevice state to reflect the new mirroring status
        const updatedDevices = await fetch("/api/devices", {
          credentials: "include",
        });
        if (updatedDevices.ok) {
          const deviceData = await updatedDevices.json();
          const updatedDevice = deviceData.devices.find((d: Device) => d.id === editDevice.id);
          if (updatedDevice) {
            setEditDevice(updatedDevice);
          }
        }
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to mirror device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setMirrorLoading(false);
    }
  };

  const syncMirror = async (device: Device) => {
    try {
      setSyncLoading(true);
      setError(null);

      const response = await fetch(`/api/devices/${device.id}/sync-mirror`, {
        method: "POST",
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setSuccessMessage(data.message);
        await fetchDevices();
        
        // Update the editDevice state to reflect the new sync timestamp
        if (editDevice && editDevice.id === device.id) {
          const updatedDevices = await fetch("/api/devices", {
            credentials: "include",
          });
          if (updatedDevices.ok) {
            const deviceData = await updatedDevices.json();
            const updatedDevice = deviceData.devices.find((d: Device) => d.id === device.id);
            if (updatedDevice) {
              setEditDevice(updatedDevice);
            }
          }
        }
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to sync device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setSyncLoading(false);
    }
  };

  const unmirrorDevice = async (device: Device) => {
    try {
      setError(null);

      const response = await fetch(`/api/devices/${device.id}/unmirror`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setSuccessMessage(data.message);
        await fetchDevices();
        
        // Update the editDevice state to reflect the removed mirroring status
        if (editDevice && editDevice.id === device.id) {
          const updatedDevices = await fetch("/api/devices", {
            credentials: "include",
          });
          if (updatedDevices.ok) {
            const deviceData = await updatedDevices.json();
            const updatedDevice = deviceData.devices.find((d: Device) => d.id === device.id);
            if (updatedDevice) {
              setEditDevice(updatedDevice);
            }
          }
        }
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to unmirror device");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  return (
    <>
      {error && (
        <Alert variant="destructive" className="mb-4">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {successMessage && (
        <Alert className="mb-4">
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>{successMessage}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <h3 className="text-lg font-semibold">Your Devices</h3>
                    <Button 
            onClick={() => setShowClaimDialog(true)}
          >
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
                Claim Device
              </Button>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent>
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
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{device.name || "Unnamed Device"}</span>
                            {device.is_sharable && (
                              <Badge variant="secondary" className="text-xs">
                                Sharable
                              </Badge>
                            )}
                            {device.mirror_source_id && (
                              <Badge variant="outline" className="text-xs">
                                Mirroring
                              </Badge>
                            )}
                          </div>
                          <div className="text-sm text-muted-foreground">
                            ID: {device.friendly_id}
                            {device.mirror_source_id && device.mirror_synced_at && (
                              <span className="ml-2">
                                • Last sync: {new Date(device.mirror_synced_at).toLocaleDateString()}
                              </span>
                            )}
                          </div>
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
                            <TooltipContent>Unlink Device</TooltipContent>
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

      {/* Claim Device Dialog */}
      <Dialog open={showClaimDialog} onOpenChange={(open) => {
        setShowClaimDialog(open);
        if (!open) {
          setClaimError(null);
          setFriendlyId("");
          setDeviceName("");
        }
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Claim Device</DialogTitle>
            <DialogDescription>
              Enter your TRMNL device's friendly ID or MAC address to claim it to your account.
            </DialogDescription>
          </DialogHeader>
          
          {claimError && (
            <Alert variant="destructive" className="items-center">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{claimError}</AlertDescription>
            </Alert>
          )}
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="friendly-id">Device Friendly ID or MAC Address</Label>
              <Input
                id="friendly-id"
                value={friendlyId}
                onChange={(e) => setFriendlyId(e.target.value)}
                placeholder="e.g., 917F0B or AA:BB:CC:DD:EE:FF"
                className="mt-2"
              />
              <p className="text-sm text-muted-foreground mt-1">
                Enter either the 6-character ID from your device's setup screen or its MAC address
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
        <DialogContent className="sm:max-w-md mobile-dialog-content overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
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
              <Label className="text-sm">Device Model</Label>
              <div className="mt-1 text-sm">
                {editDevice?.device_model?.display_name || editDevice?.reported_model_name || editDevice?.model_name || "Unknown"}
              </div>
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
            <div>
              <div className="flex items-center space-x-2">
                <Switch
                  id="edit-allow-firmware-updates"
                  checked={editAllowFirmwareUpdates}
                  onCheckedChange={setEditAllowFirmwareUpdates}
                />
                <Label htmlFor="edit-allow-firmware-updates">
                  Allow automatic firmware updates
                </Label>
              </div>
              <p className="text-sm text-muted-foreground mt-1">
                When enabled, device will automatically update to the latest firmware
              </p>
            </div>

            {/* Sleep Mode Section */}
            <div className="pt-3 border-t border-border/50 space-y-4">
              <div>
                <Label className="text-sm font-medium">Sleep Mode</Label>
                <div className="mt-2 space-y-3">
                  <div className="flex items-center space-x-2">
                    <Switch
                      id="edit-sleep-enabled"
                      checked={editSleepEnabled}
                      onCheckedChange={setEditSleepEnabled}
                    />
                    <Label htmlFor="edit-sleep-enabled">
                      Enable sleep mode
                    </Label>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Reduce screen refreshes during inactive periods to save battery
                  </p>

                  {editSleepEnabled && (
                    <div className="space-y-3 pl-6 border-l-2 border-border">
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <Label htmlFor="edit-sleep-start-time" className="text-sm">Start Time</Label>
                          <Input
                            id="edit-sleep-start-time"
                            type="time"
                            value={editSleepStartTime}
                            onChange={(e) => setEditSleepStartTime(e.target.value)}
                            className="mt-1"
                          />
                        </div>
                        <div>
                          <Label htmlFor="edit-sleep-end-time" className="text-sm">End Time</Label>
                          <Input
                            id="edit-sleep-end-time"
                            type="time"
                            value={editSleepEndTime}
                            onChange={(e) => setEditSleepEndTime(e.target.value)}
                            className="mt-1"
                          />
                        </div>
                      </div>


                      <div className="flex items-center space-x-2">
                        <Switch
                          id="edit-sleep-show-screen"
                          checked={editSleepShowScreen}
                          onCheckedChange={setEditSleepShowScreen}
                        />
                        <Label htmlFor="edit-sleep-show-screen" className="text-sm">
                          Show sleep screen
                        </Label>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        When enabled, display sleep image instead of last content during sleep period
                      </p>

                      {editSleepStartTime && editSleepEndTime && (
                        <div className="p-3 bg-muted/30 rounded-md">
                          <div className="text-xs font-medium text-muted-foreground">Schedule Preview</div>
                          <div className="text-sm mt-1">
                            Sleep from <span className="font-mono">{editSleepStartTime}</span> to{" "}
                            <span className="font-mono">{editSleepEndTime}</span> (in your timezone)
                          </div>
                          {editSleepStartTime > editSleepEndTime && (
                            <div className="text-xs text-muted-foreground mt-1">
                              This schedule crosses midnight
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Visibility Section */}
            <div className="pt-3 border-t border-border/50 space-y-4">
              <div>
                <Label className="text-sm font-medium">Visibility</Label>
                <div className="mt-2">
                  <div className="flex items-center space-x-2">
                    <Switch
                      id="edit-is-sharable"
                      checked={editIsSharable}
                      onCheckedChange={setEditIsSharable}
                    />
                    <Label htmlFor="edit-is-sharable">
                      Sharable
                    </Label>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">
                    Allow other devices to mirror this device's playlist
                  </p>
                  {editIsSharable && editDevice && (
                    <div className="mt-3 p-3 bg-muted/30 rounded-md">
                      <Label className="text-xs text-muted-foreground">Device ID for Mirroring</Label>
                      <div className="mt-1 font-mono text-lg font-bold text-primary">
                        {editDevice.friendly_id}
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        Share this 6-digit ID with others who want to mirror this device
                      </p>
                    </div>
                  )}
                </div>
              </div>

              {/* Mirroring Section */}
              <div>
                <Label className="text-sm font-medium">Mirroring</Label>
                <div className="mt-2 space-y-3">
                  {editDevice?.mirror_source_id ? (
                    <div className="space-y-2">
                      <div className="p-3 bg-blue-50 dark:bg-blue-950/30 rounded-md">
                        <div className="flex items-center justify-between">
                          <div>
                            <div className="text-sm font-medium">Currently Mirroring</div>
                            <div className="text-xs text-muted-foreground">
                              Last synced: {editDevice.mirror_synced_at ? new Date(editDevice.mirror_synced_at).toLocaleString() : "Never"}
                            </div>
                          </div>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => editDevice && syncMirror(editDevice)}
                          disabled={syncLoading}
                        >
                          {syncLoading ? "Syncing..." : "Sync Screens"}
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => editDevice && unmirrorDevice(editDevice)}
                        >
                          Stop Mirroring
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setShowMirrorDialog(true)}
                      >
                        Mirror Another Device
                      </Button>
                      <p className="text-xs text-muted-foreground mt-1">
                        Copy playlist from another sharable device
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
            
            {/* Developer Settings Collapsible Section */}
            <Collapsible 
              open={isDeveloperSectionOpen} 
              onOpenChange={setIsDeveloperSectionOpen}
            >
              <CollapsibleTrigger asChild>
                <Button
                  variant="ghost"
                  className="flex items-center justify-between w-full p-0 h-auto font-normal"
                >
                  <span className="text-sm font-medium">Developer Settings</span>
                  <ChevronDown 
                    className={`h-4 w-4 transition-transform duration-200 ${
                      isDeveloperSectionOpen ? 'transform rotate-180' : ''
                    }`} 
                  />
                </Button>
              </CollapsibleTrigger>
              
              <CollapsibleContent className="space-y-4 pt-4">
                <div>
                  <Label htmlFor="edit-model-name">Model Override</Label>
                  
                  <Select value={editModelName} onValueChange={handleModelChange}>
                    <SelectTrigger className="mt-2">
                      <SelectValue placeholder={modelsLoading ? "Loading models..." : "Select a model (optional)"} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">
                        {editDevice?.reported_model_name 
                          ? `Use Device Model (${getModelDisplayName(editDevice.reported_model_name)})`
                          : "Auto-Detect"}
                      </SelectItem>
                      {(deviceModels || []).map((model) => (
                        <SelectItem key={model.model_name} value={model.model_name}>
                          {model.display_name}
                          {model.screen_width && model.screen_height && (
                            <span className="text-muted-foreground ml-2">
                              {model.screen_width}×{model.screen_height}
                            </span>
                          )}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-sm text-muted-foreground mt-1">
                    Override only if your device is incorrectly reporting its model.
                  </p>

                  {/* Model Specifications */}
                  {(() => {
                    const selectedModel = editModelName === "none" 
                      ? deviceModels.find(m => m.model_name === (editDevice?.reported_model_name || editDevice?.model_name))
                      : deviceModels.find(m => m.model_name === editModelName);
                    return selectedModel && (
                      <div className="mt-3 pt-3 border-t border-border/50">
                        <div className="space-y-2">
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-muted-foreground">Display:</span>
                            <span className="font-medium">
                              {selectedModel.screen_width} × {selectedModel.screen_height} ({selectedModel.bit_depth}-bit)
                            </span>
                          </div>
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-muted-foreground">Features:</span>
                            <div className="flex items-center gap-3 text-xs text-muted-foreground">
                              {selectedModel.has_wifi && (
                                <span className="flex items-center gap-1">
                                  <Wifi className="h-3 w-3" />
                                  WiFi
                                </span>
                              )}
                              {selectedModel.has_battery && (
                                <span className="flex items-center gap-1">
                                  <Battery className="h-3 w-3" />
                                  Battery
                                </span>
                              )}
                              {selectedModel.has_buttons > 0 && (
                                <span>
                                  {selectedModel.has_buttons} Button{selectedModel.has_buttons > 1 ? 's' : ''}
                                </span>
                              )}
                            </div>
                          </div>
                          {selectedModel.description && (
                            <div className="text-xs text-muted-foreground pt-1">
                              {selectedModel.description}
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })()}
                </div>
                
                <div className="pt-3 border-t border-border/50 space-y-3">
                  <div>
                    <Label htmlFor="edit-mac-address" className="text-xs text-muted-foreground">MAC Address</Label>
                    <Input
                      id="edit-mac-address"
                      type="text"
                      value={editDevice?.mac_address || ""}
                      readOnly
                      className="mt-1 font-mono text-sm bg-muted/30"
                    />
                  </div>
                  
                  <div>
                    <Label htmlFor="edit-api-key" className="text-xs text-muted-foreground">API Key</Label>
                    <Input
                      id="edit-api-key"
                      type="text"
                      value={editDevice?.api_key || ""}
                      readOnly
                      className="mt-1 font-mono text-sm bg-muted/30"
                    />
                  </div>
                </div>
              </CollapsibleContent>
            </Collapsible>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDevice(null)}>
              Cancel
            </Button>
            <Button
              onClick={updateDevice}
              disabled={!editDeviceName.trim() || !hasDeviceChanges()}
            >
              Update Device
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Unlink Device Dialog */}
      <Dialog open={!!deleteDevice} onOpenChange={() => setDeleteDevice(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Unlink Device
            </DialogTitle>
            <DialogDescription>
              Are you sure you want to unlink "{deleteDevice?.name || 'this device'}"? This will remove all associated playlists and make the device available for reclaiming. You can claim it again later if needed.
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
              {deleteLoading ? "Unlinking..." : "Unlink Device"}
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
                              <div className="max-w-lg">
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

      {/* Model Override Confirmation Dialog */}
      <AlertDialog open={showModelOverrideDialog} onOpenChange={setShowModelOverrideDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Override Device Model?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {editDevice && (
                <div className="space-y-2">
                  <p>
                    Your device reports as <span className="font-medium">{getModelDisplayName(editDevice.reported_model_name || "")}</span>.
                  </p>
                  <p>
                    You're trying to set it to <span className="font-medium">{getModelDisplayName(pendingModelOverride)}</span>.
                  </p>
                  <p className="text-destructive">
                    This may cause display issues if the models have different screen sizes or capabilities.
                  </p>
                  <p>
                    Only override if your device is incorrectly reporting its model.
                  </p>
                </div>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={cancelModelOverride}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmModelOverride}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Override Model
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Mirror Device Dialog */}
      <Dialog open={showMirrorDialog} onOpenChange={setShowMirrorDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Mirror Another Device</DialogTitle>
            <DialogDescription>
              Enter the 6-digit Device ID of the sharable device you want to mirror.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="mirror-source-id">Source Device ID</Label>
              <Input
                id="mirror-source-id"
                value={mirrorSourceFriendlyId}
                onChange={(e) => setMirrorSourceFriendlyId(e.target.value.toUpperCase())}
                placeholder="e.g., 917F0B"
                className="mt-2"
                maxLength={6}
              />
              <p className="text-sm text-muted-foreground mt-1">
                This device must be marked as "Sharable" by its owner
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowMirrorDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={mirrorDevice}
              disabled={mirrorLoading || !mirrorSourceFriendlyId.trim()}
            >
              {mirrorLoading ? "Mirroring..." : "Mirror Device"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
