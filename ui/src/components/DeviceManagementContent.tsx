import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/components/AuthProvider";
import { calculateBatteryPercentage } from "@/utils/deviceHelpers";
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
  ChevronUp,
  ChevronsUpDown,
} from "lucide-react";

interface DeviceModel {
  id: number;
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
  device_model_id?: number;
  manual_model_override?: boolean;
  reported_model_name?: string;
  api_key: string;
  is_claimed: boolean;
  firmware_version?: string;
  target_firmware_version?: string;
  battery_voltage?: number;
  rssi?: number;
  refresh_rate: number;
  allow_firmware_updates?: boolean;
  last_seen?: string;
  is_active: boolean;
  is_shareable?: boolean;
  mirror_source_id?: string;
  mirror_synced_at?: string;
  sleep_enabled?: boolean;
  sleep_start_time?: string;
  sleep_end_time?: string;
  sleep_show_screen?: boolean;
  firmware_update_start_time?: string;
  firmware_update_end_time?: string;
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

interface FirmwareVersion {
  id: string;
  version: string;
  download_url: string;
  is_latest: boolean;
  is_downloaded: boolean;
  released_at: string;
  download_status?: string;
  file_size?: number;
  release_notes?: string;
}

// Sort types for devices table
type DeviceSortColumn = 'name' | 'status' | 'firmware' | 'battery' | 'signal' | 'last_seen' | 'created';
type SortOrder = 'asc' | 'desc';

interface DeviceSortState {
  column: DeviceSortColumn;
  order: SortOrder;
}

interface DeviceManagementContentProps {
  onUpdate?: () => void;
}

export function DeviceManagementContent({ onUpdate }: DeviceManagementContentProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [devices, setDevices] = useState<Device[]>([]);
  
  // Devices table sorting state with localStorage persistence
  const [deviceSortState, setDeviceSortState] = useState<DeviceSortState>(() => {
    try {
      const saved = localStorage.getItem('userSettingsDevicesTableSort');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.column && ['name', 'status', 'firmware', 'battery', 'signal', 'last_seen', 'created'].includes(parsed.column) &&
            parsed.order && ['asc', 'desc'].includes(parsed.order)) {
          return parsed;
        }
      }
    } catch (e) {
      // Invalid localStorage data, fall back to default
    }
    return { column: 'name', order: 'asc' };
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Device claiming
  const [showClaimDialog, setShowClaimDialog] = useState(false);
  const [claimLoading, setClaimLoading] = useState(false);
  const [claimError, setClaimError] = useState<string | null>(null);
  const [friendlyId, setFriendlyId] = useState("");
  const [deviceName, setDeviceName] = useState("");

  // Device importing
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);
  const [importMacAddress, setImportMacAddress] = useState("");
  const [importApiKey, setImportApiKey] = useState("");
  const [importFriendlyId, setImportFriendlyId] = useState("");
  const [importDeviceName, setImportDeviceName] = useState("");
  const [importDeviceModelId, setImportDeviceModelId] = useState<number | undefined>(undefined);

  // Device editing
  const [editDevice, setEditDevice] = useState<Device | null>(null);
  const [editDeviceName, setEditDeviceName] = useState("");
  const [editRefreshRate, setEditRefreshRate] = useState("");
  const [editAllowFirmwareUpdates, setEditAllowFirmwareUpdates] = useState(true);
  const [editModelName, setEditModelName] = useState<string>("");
  const [editIsShareable, setEditIsShareable] = useState(false);
  const [deviceModels, setDeviceModels] = useState<DeviceModel[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [editDialogError, setEditDialogError] = useState<string | null>(null);

  // Sleep mode settings
  const [editSleepEnabled, setEditSleepEnabled] = useState(false);
  const [editSleepStartTime, setEditSleepStartTime] = useState("");
  const [editSleepEndTime, setEditSleepEndTime] = useState("");
  const [editSleepShowScreen, setEditSleepShowScreen] = useState(true);

  // Firmware update schedule settings
  const [editFirmwareUpdateStartTime, setEditFirmwareUpdateStartTime] = useState("00:00");
  const [editFirmwareUpdateEndTime, setEditFirmwareUpdateEndTime] = useState("23:59");
  const [editTargetFirmwareVersion, setEditTargetFirmwareVersion] = useState<string>("latest");

  // Firmware versions for dropdown
  const [firmwareVersions, setFirmwareVersions] = useState<FirmwareVersion[]>([]);
  const [showUnstableVersions, setShowUnstableVersions] = useState(false);

  // Semantic version comparison utility
  const compareSemanticVersions = React.useCallback((a: string, b: string): number => {
    const parseVersion = (v: string) => {
      const parts = v.split('.').map(x => parseInt(x, 10) || 0);
      return { major: parts[0] || 0, minor: parts[1] || 0, patch: parts[2] || 0 };
    };

    const vA = parseVersion(a);
    const vB = parseVersion(b);

    if (vA.major !== vB.major) return vB.major - vA.major;
    if (vA.minor !== vB.minor) return vB.minor - vA.minor;
    return vB.patch - vA.patch;
  }, []);

  // Sort firmware versions by semantic version (descending: newest first)
  const sortedFirmwareVersions = React.useMemo(() => {
    return [...firmwareVersions].sort((a, b) =>
      compareSemanticVersions(a.version, b.version)
    );
  }, [firmwareVersions, compareSemanticVersions]);

  // Filter firmware versions based on unstable checkbox
  const filteredFirmwareVersions = React.useMemo(() => {
    const stableVersion = sortedFirmwareVersions.find(fw => fw.is_latest);
    if (!stableVersion) {
      return sortedFirmwareVersions;
    }

    if (showUnstableVersions) {
      return sortedFirmwareVersions;
    }

    const filtered = sortedFirmwareVersions.filter(fw => {
      const versionComparison = compareSemanticVersions(fw.version, stableVersion.version);
      const shouldInclude = fw.is_latest || versionComparison >= 0;
      return shouldInclude;
    });

    return filtered;
  }, [sortedFirmwareVersions, showUnstableVersions, compareSemanticVersions]);

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
  
  // Sort handler
  const handleDeviceSort = (column: DeviceSortColumn) => {
    setDeviceSortState(prevState => ({
      column,
      order: prevState.column === column && prevState.order === 'asc' ? 'desc' : 'asc'
    }));
  };
  
  // Save sort state to localStorage whenever it changes
  useEffect(() => {
    try {
      localStorage.setItem('userSettingsDevicesTableSort', JSON.stringify(deviceSortState));
    } catch (e) {
      // Ignore localStorage errors
    }
  }, [deviceSortState]);
  
  // Sorted devices array
  const sortedDevices = React.useMemo(() => {
    const sorted = [...devices].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      switch (deviceSortState.column) {
        case 'name':
          aValue = (a.name || 'Unnamed Device')?.toLowerCase() || '';
          bValue = (b.name || 'Unnamed Device')?.toLowerCase() || '';
          break;
        case 'status':
          aValue = getDeviceStatus(a) === 'online' ? 3 : (getDeviceStatus(a) === 'recently_online' ? 2 : (getDeviceStatus(a) === 'offline' ? 1 : 0));
          bValue = getDeviceStatus(b) === 'online' ? 3 : (getDeviceStatus(b) === 'recently_online' ? 2 : (getDeviceStatus(b) === 'offline' ? 1 : 0));
          break;
        case 'firmware':
          aValue = a.firmware_version?.toLowerCase() || '';
          bValue = b.firmware_version?.toLowerCase() || '';
          break;
        case 'battery':
          aValue = a.battery_voltage || 0;
          bValue = b.battery_voltage || 0;
          break;
        case 'signal':
          aValue = a.rssi || -999;
          bValue = b.rssi || -999;
          break;
        case 'last_seen':
          aValue = a.last_seen ? new Date(a.last_seen).getTime() : 0;
          bValue = b.last_seen ? new Date(b.last_seen).getTime() : 0;
          break;
        case 'created':
          aValue = new Date(a.created_at).getTime();
          bValue = new Date(b.created_at).getTime();
          break;
        default:
          return 0;
      }

      if (aValue < bValue) {
        return deviceSortState.order === 'asc' ? -1 : 1;
      }
      if (aValue > bValue) {
        return deviceSortState.order === 'asc' ? 1 : -1;
      }
      return 0;
    });

    return sorted;
  }, [devices, deviceSortState]);

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
        onUpdate?.();
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

  const openImportDialog = () => {
    setShowImportDialog(true);
    if (deviceModels.length === 0) {
      fetchDeviceModels().catch(error => {
        console.error("Failed to fetch device models:", error);
      });
    }
  };

  const importDevice = async () => {
    if (!importMacAddress.trim() || !importApiKey.trim() || !importFriendlyId.trim()) {
      setImportError("MAC address, API key, and friendly ID are required");
      return;
    }

    try {
      setImportLoading(true);
      setImportError(null);

      const requestBody: any = {
        mac_address: importMacAddress.trim(),
        api_key: importApiKey.trim(),
        friendly_id: importFriendlyId.trim().toUpperCase(),
      };

      if (importDeviceName.trim()) {
        requestBody.name = importDeviceName.trim();
      }

      if (importDeviceModelId !== undefined) {
        requestBody.device_model_id = importDeviceModelId;
      }

      const response = await fetch("/api/devices/import", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(requestBody),
      });

      if (response.ok) {
        setSuccessMessage("Device imported successfully!");
        setShowImportDialog(false);
        setImportMacAddress("");
        setImportApiKey("");
        setImportFriendlyId("");
        setImportDeviceName("");
        setImportDeviceModelId(undefined);
        setImportError(null);
        await fetchDevices();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        setImportError(errorData.error || "Failed to import device");
      }
    } catch (error) {
      setImportError("Network error occurred");
    } finally {
      setImportLoading(false);
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
        setDeviceModels([]);
      }
    } catch (error) {
      console.error("Error fetching device models:", error);
      setDeviceModels([]);
    } finally {
      setModelsLoading(false);
    }
  };

  const fetchFirmwareVersions = async () => {
    try {
      const response = await fetch("/api/admin/firmware/versions", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setFirmwareVersions(data.firmware_versions || []);
      } else {
        console.error("Failed to fetch firmware versions");
        setFirmwareVersions([]);
      }
    } catch (error) {
      console.error("Error fetching firmware versions:", error);
      setFirmwareVersions([]);
    }
  };

  const updateDevice = async () => {
    if (!editDevice || !editDeviceName.trim()) {
      setEditDialogError("Please fill in all fields");
      return;
    }

    const refreshRate = parseInt(editRefreshRate);
    if (isNaN(refreshRate) || refreshRate < 60 || refreshRate > 86400) {
      setEditDialogError("Refresh rate must be between 60 and 86400 seconds");
      return;
    }

    try {
      setEditDialogError(null);

      const requestBody: any = {
        name: editDeviceName.trim(),
        refresh_rate: refreshRate,
        allow_firmware_updates: editAllowFirmwareUpdates,
        is_shareable: editIsShareable,
        sleep_enabled: editSleepEnabled,
        sleep_start_time: editSleepStartTime,
        sleep_end_time: editSleepEndTime,
        sleep_show_screen: editSleepShowScreen,
        firmware_update_start_time: editFirmwareUpdateStartTime,
        firmware_update_end_time: editFirmwareUpdateEndTime,
        target_firmware_version: editTargetFirmwareVersion,
      };


      // Add device model ID if model selection has changed from the original
      const originalModelName = editDevice.device_model?.model_name || "none";
      if (editModelName !== originalModelName) {
        if (editModelName === "none") {
          requestBody.device_model_id = 0; // 0 means clear the model
        } else {
          // Find the device model ID by name
          const selectedModel = deviceModels.find(m => m.model_name === editModelName);
          if (selectedModel) {
            requestBody.device_model_id = selectedModel.id;
          }
        }
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
        setEditDialogError(errorData.error || "Failed to update device");
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
    }
  };

  const clearModelOverride = async () => {
    if (!editDevice) return;

    try {
      setEditDialogError(null);

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
        setEditDialogError(errorData.error || "Failed to clear model override");
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
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
      setEditModelName(device.device_model?.model_name || "none");
      setEditIsShareable(device.is_shareable ?? false);
      
      // Initialize sleep mode settings
      setEditSleepEnabled(device.sleep_enabled ?? false);
      setEditSleepStartTime(device.sleep_start_time || "22:00");
      setEditSleepEndTime(device.sleep_end_time || "06:00");
      setEditSleepShowScreen(device.sleep_show_screen ?? true);
      
      // Initialize firmware update schedule settings
      setEditFirmwareUpdateStartTime(device.firmware_update_start_time || "00:00");
      setEditFirmwareUpdateEndTime(device.firmware_update_end_time || "23:59");
      setEditTargetFirmwareVersion(device.target_firmware_version || "latest");

      // Clear any previous dialog errors
      setEditDialogError(null);

      // Fetch device models when opening edit dialog
      if (deviceModels.length === 0) {
        fetchDeviceModels().catch(error => {
          console.error("Failed to fetch device models:", error);
        });
      }

      // Fetch firmware versions when opening edit dialog
      if (firmwareVersions.length === 0) {
        fetchFirmwareVersions().catch(error => {
          console.error("Failed to fetch firmware versions:", error);
        });
      }
    } catch (error) {
      console.error("Error opening edit dialog:", error);
      setEditDialogError("Failed to open device edit dialog");
    }
  };

  const hasDeviceChanges = () => {
    if (!editDevice) return false;
    return (
      editDeviceName.trim() !== (editDevice.name || "") ||
      editRefreshRate !== editDevice.refresh_rate.toString() ||
      editAllowFirmwareUpdates !== (editDevice.allow_firmware_updates ?? true) ||
      editModelName !== (editDevice.device_model?.model_name || "none") ||
      editIsShareable !== (editDevice.is_shareable ?? false) ||
      editSleepEnabled !== (editDevice.sleep_enabled ?? false) ||
      editSleepStartTime !== (editDevice.sleep_start_time || "22:00") ||
      editSleepEndTime !== (editDevice.sleep_end_time || "06:00") ||
      editSleepShowScreen !== (editDevice.sleep_show_screen ?? true) ||
      editFirmwareUpdateStartTime !== (editDevice.firmware_update_start_time || "00:00") ||
      editFirmwareUpdateEndTime !== (editDevice.firmware_update_end_time || "23:59") ||
      editTargetFirmwareVersion !== (editDevice.target_firmware_version || "latest")
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
      icon = <BatteryLow className="h-4 w-4" />;
      color = "";
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
      setEditDialogError("Please enter a device ID");
      return;
    }

    try {
      setMirrorLoading(true);
      setEditDialogError(null);

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
        setEditDialogError(errorData.error || "Failed to mirror device");
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
    } finally {
      setMirrorLoading(false);
    }
  };

  const syncMirror = async (device: Device) => {
    try {
      setSyncLoading(true);
      setEditDialogError(null);

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
        setEditDialogError(errorData.error || "Failed to sync device");
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
    } finally {
      setSyncLoading(false);
    }
  };

  const unmirrorDevice = async (device: Device) => {
    try {
      setEditDialogError(null);

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
        setEditDialogError(errorData.error || "Failed to unmirror device");
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
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
          <div className="flex gap-2">
            <Button
              onClick={openImportDialog}
              variant="outline"
            >
              Import Device
            </Button>
            <Button
              onClick={() => setShowClaimDialog(true)}
            >
              Claim Device
            </Button>
          </div>
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
                    <TableHead 
                      className="cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('name')}
                    >
                      <div className="flex items-center gap-1">
                        Device
                        {deviceSortState.column === 'name' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead 
                      className="cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('status')}
                    >
                      <div className="flex items-center gap-1">
                        Status
                        {deviceSortState.column === 'status' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead 
                      className="hidden sm:table-cell cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('firmware')}
                    >
                      <div className="flex items-center gap-1">
                        Firmware
                        {deviceSortState.column === 'firmware' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead 
                      className="hidden sm:table-cell cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('battery')}
                    >
                      <div className="flex items-center gap-1">
                        Battery
                        {deviceSortState.column === 'battery' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead 
                      className="hidden sm:table-cell cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('signal')}
                    >
                      <div className="flex items-center gap-1">
                        Signal
                        {deviceSortState.column === 'signal' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead 
                      className="hidden lg:table-cell cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('last_seen')}
                    >
                      <div className="flex items-center gap-1">
                        Last Seen
                        {deviceSortState.column === 'last_seen' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead 
                      className="hidden lg:table-cell cursor-pointer hover:bg-muted/50 select-none"
                      onClick={() => handleDeviceSort('created')}
                    >
                      <div className="flex items-center gap-1">
                        Created
                        {deviceSortState.column === 'created' ? (
                          deviceSortState.order === 'asc' ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )
                        ) : (
                          <ChevronsUpDown className="h-4 w-4 opacity-50" />
                        )}
                      </div>
                    </TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sortedDevices.map((device) => (
                    <TableRow key={device.id}>
                      <TableCell>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{device.name || "Unnamed Device"}</span>
                            {device.is_shareable && (
                              <Badge variant="outline" className="text-xs">
                                Shareable
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

      {/* Import Device Dialog */}
      <Dialog open={showImportDialog} onOpenChange={(open) => {
        setShowImportDialog(open);
        if (!open) {
          setImportError(null);
          setImportMacAddress("");
          setImportApiKey("");
          setImportFriendlyId("");
          setImportDeviceName("");
          setImportDeviceModelId(undefined);
        }
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Import Device</DialogTitle>
            <DialogDescription>
              Import an existing device with its credentials. Use this to migrate a device from another instance.
            </DialogDescription>
          </DialogHeader>

          {importError && (
            <Alert variant="destructive" className="items-center">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{importError}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-4">
            <div>
              <Label htmlFor="import-mac">MAC Address *</Label>
              <Input
                id="import-mac"
                value={importMacAddress}
                onChange={(e) => setImportMacAddress(e.target.value)}
                placeholder="AA:BB:CC:DD:EE:FF"
                className="mt-2"
              />
            </div>

            <div>
              <Label htmlFor="import-api-key">API Key *</Label>
              <Input
                id="import-api-key"
                value={importApiKey}
                onChange={(e) => setImportApiKey(e.target.value)}
                placeholder="64-character hexadecimal API key"
                className="mt-2"
              />
            </div>

            <div>
              <Label htmlFor="import-friendly-id">Friendly ID *</Label>
              <Input
                id="import-friendly-id"
                value={importFriendlyId}
                onChange={(e) => setImportFriendlyId(e.target.value.toUpperCase())}
                placeholder="917F0B"
                maxLength={6}
                className="mt-2"
              />
            </div>

            <div>
              <Label htmlFor="import-name">Device Name (optional)</Label>
              <Input
                id="import-name"
                value={importDeviceName}
                onChange={(e) => setImportDeviceName(e.target.value)}
                placeholder="e.g., Kitchen Display"
                className="mt-2"
              />
            </div>

            <div>
              <Label htmlFor="import-model">Device Model (optional)</Label>
              <Select
                value={importDeviceModelId?.toString()}
                onValueChange={(value) => setImportDeviceModelId(parseInt(value))}
              >
                <SelectTrigger className="mt-2">
                  <SelectValue placeholder={modelsLoading ? "Loading models..." : "Auto-detect from device"} />
                </SelectTrigger>
                <SelectContent>
                  {(deviceModels || []).map((model) => (
                    <SelectItem key={model.id} value={model.id.toString()}>
                      {model.display_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowImportDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={importDevice}
              disabled={importLoading || !importMacAddress.trim() || !importApiKey.trim() || !importFriendlyId.trim()}
            >
              {importLoading ? "Importing..." : "Import Device"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Device Dialog */}
      <Dialog open={!!editDevice} onOpenChange={() => {
        setEditDevice(null);
        setEditDialogError(null);
      }}>
        <DialogContent 
          className="sm:max-w-lg md:max-w-2xl lg:max-w-4xl xl:max-w-5xl max-h-[85vh] mobile-dialog-content overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>Edit Device</DialogTitle>
            <DialogDescription>
              Update device settings and configuration.
            </DialogDescription>
          </DialogHeader>
          
          {editDialogError && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{editDialogError}</AlertDescription>
            </Alert>
          )}
          
          <div className="space-y-6 md:grid md:grid-cols-2 md:gap-6 md:space-y-0">
            {/* Left Column - Basic Settings */}
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
                  {editDevice?.device_model?.display_name || editDevice?.reported_model_name || "Unknown"}
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

                {editAllowFirmwareUpdates && (
                  <div className="mt-3 pl-6 border-l-2 border-border space-y-3">
                    <div>
                      <Label className="text-sm font-medium">Firmware Update Schedule</Label>
                      <p className="text-sm text-muted-foreground mt-1">
                        Restrict firmware updates to specific times
                      </p>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <Label htmlFor="edit-firmware-start-time" className="text-sm">Start Time</Label>
                        <Input
                          id="edit-firmware-start-time"
                          type="time"
                          value={editFirmwareUpdateStartTime}
                          onChange={(e) => setEditFirmwareUpdateStartTime(e.target.value)}
                          className="mt-1"
                        />
                      </div>
                      <div>
                        <Label htmlFor="edit-firmware-end-time" className="text-sm">End Time</Label>
                        <Input
                          id="edit-firmware-end-time"
                          type="time"
                          value={editFirmwareUpdateEndTime}
                          onChange={(e) => setEditFirmwareUpdateEndTime(e.target.value)}
                          className="mt-1"
                        />
                      </div>
                    </div>
                    <div>
                      <Label htmlFor="edit-target-firmware" className="text-sm">Target Firmware Version</Label>
                      <Select
                        value={editTargetFirmwareVersion}
                        onValueChange={setEditTargetFirmwareVersion}
                      >
                        <SelectTrigger id="edit-target-firmware" className="mt-1">
                          <SelectValue placeholder="Stable (Auto)" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="latest">Stable (Auto)</SelectItem>
                          {filteredFirmwareVersions.map((fw) => (
                            <SelectItem key={fw.id} value={fw.version}>
                              {fw.version} {fw.is_latest && "(Stable)"}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <div className="flex items-center gap-2 mt-2">
                        <Checkbox
                          id="show-unstable-versions"
                          checked={showUnstableVersions}
                          onCheckedChange={(checked) => setShowUnstableVersions(checked as boolean)}
                        />
                        <Label htmlFor="show-unstable-versions" className="text-sm font-normal cursor-pointer">
                          Show unstable versions
                        </Label>
                      </div>
                      <p className="text-sm text-muted-foreground mt-1">
                        Pin device to a specific firmware version or keep it on stable
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Right Column - Advanced Settings */}
            <div className="space-y-4">
              {/* Sleep Mode Section */}
              <div className="space-y-4">
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
                        id="edit-is-shareable"
                        checked={editIsShareable}
                        onCheckedChange={setEditIsShareable}
                      />
                      <Label htmlFor="edit-is-shareable">
                        Shareable
                      </Label>
                    </div>
                    <p className="text-sm text-muted-foreground mt-1">
                      Allow other devices to mirror this device's playlist
                    </p>
                    {editIsShareable && editDevice && (
                      <div className="mt-3 p-3 bg-muted/50 rounded-md">
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
                        <div className="p-3 bg-muted/30 rounded-md">
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
                          Copy playlist from another shareable device
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
            
            {/* Developer Settings */}
            <div className="md:col-span-2 pt-3 border-t border-border/50">
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
              
              <CollapsibleContent className="space-y-4 pt-4 md:grid md:grid-cols-2 md:gap-6 md:space-y-0">
                {/* Left Column - Model Configuration */}
                <div className="space-y-4">
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
                        ? deviceModels.find(m => m.model_name === (editDevice?.reported_model_name || editDevice?.device_model?.model_name))
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
                </div>
                
                {/* Right Column - Device Credentials */}
                <div className="space-y-4 md:pt-0 pt-3 md:border-t-0 border-t border-border/50">
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
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => {
              setEditDevice(null);
              setEditDialogError(null);
            }}>
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
              Enter the 6-digit Device ID of the shareable device you want to mirror.
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
                This device must be marked as "Shareable" by its owner
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
