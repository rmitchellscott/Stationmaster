import React, { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/components/AuthProvider";
import { useConfig } from "@/components/ConfigProvider";
import { UserDeleteDialog } from "@/components/UserDeleteDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";
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
  Activity,
  Users,
  Monitor,
  Puzzle,
  Download,
  Settings as SettingsIcon,
  Database,
  ArrowLeft,
  LayoutDashboard,
  Mail,
  Server,
  Key,
  Eye,
  AlertTriangle,
  Wifi,
  WifiOff,
  Unlink,
  Loader2,
  Battery,
  BatteryFull,
  BatteryMedium,
  BatteryLow,
  BatteryWarning,
  CheckCircle,
  XCircle,
  Edit,
  Trash2,
  ArrowRightLeft,
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
} from "lucide-react";
import {
  calculateBatteryPercentage,
  getBatteryDisplay,
  getSignalDisplay,
} from "@/utils/deviceHelpers";

interface User {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
  is_active: boolean;
  created_at: string;
  last_login?: string;
  rmapi_paired?: boolean;
}

// Sort types for tables
type UserSortColumn = 'username' | 'email' | 'role' | 'status' | 'created' | 'last_login';
type DeviceSortColumn = 'id' | 'name' | 'user' | 'model' | 'status' | 'last_seen';
type PluginSortColumn = 'name' | 'type' | 'version' | 'author' | 'status' | 'created';
type SortOrder = 'asc' | 'desc';

interface UserSortState {
  column: UserSortColumn;
  order: SortOrder;
}

interface DeviceSortState {
  column: DeviceSortColumn;
  order: SortOrder;
}

interface PluginSortState {
  column: PluginSortColumn;
  order: SortOrder;
}

interface SystemStatus {
  database: {
    total_users: number;
    active_users: number;
    admin_users: number;
    documents: number;
    active_sessions: number;
    api_keys: {
      total: number;
      active: number;
      expired: number;
      recently_used: number;
    };
  };
  smtp: {
    configured: boolean;
    status: string;
  };
  auth: {
    oidc_enabled: boolean;
    proxy_auth_enabled: boolean;
  };
  settings: {
    registration_enabled: string;
    registration_enabled_locked: boolean;
    max_api_keys_per_user: string;
    session_timeout_hours: string;
    site_url?: string;
  };
  mode: string;
  dry_run: boolean;
}

interface BackupJob {
  id: string;
  status: string;
  created_at: string;
  completed_at?: string;
  file_size?: number;
  error_message?: string;
}

interface RestoreUpload {
  id: string;
  filename: string;
  uploaded_at: string;
  file_size: number;
}

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
  api_key: string;
  is_claimed: boolean;
  firmware_version?: string;
  battery_voltage?: number;
  rssi?: number;
  refresh_rate: number;
  last_seen?: string;
  last_playlist_index: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  user?: User;
  device_model?: DeviceModel;
}

interface DeviceStats {
  total_devices: number;
  claimed_devices: number;
  unclaimed_devices: number;
  active_devices: number;
  inactive_devices: number;
  recent_registrations: number;
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

interface PluginStats {
  total_plugins: number;
  active_plugins: number;
  total_plugin_instances: number;
  active_plugin_instances: number;
}

interface FirmwareVersion {
  id: string;
  version: string;
  release_notes: string;
  download_url: string;
  file_size: number;
  is_latest: boolean;
  download_status?: string;
  created_at: string;
}

interface FirmwareStats {
  firmware_versions: {
    total: number;
    downloaded: number;
  };
  device_models: {
    total: number;
  };
  update_settings: {
    enabled: number;
    disabled: number;
  };
  firmware_distribution: Array<{
    version: string;
    count: number;
  }>;
}

/**
 * Compare two semver version strings
 * Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
 * Returns null if either version is invalid
 */
function compareSemver(v1: string | null | undefined, v2: string | null | undefined): number | null {
  if (!v1 || !v2) return null;
  
  // Remove 'v' prefix if present
  const clean1 = v1.replace(/^v/, '');
  const clean2 = v2.replace(/^v/, '');
  
  const parts1 = clean1.split('.').map(Number);
  const parts2 = clean2.split('.').map(Number);
  
  // Ensure both have 3 parts (major.minor.patch)
  while (parts1.length < 3) parts1.push(0);
  while (parts2.length < 3) parts2.push(0);
  
  for (let i = 0; i < 3; i++) {
    if (isNaN(parts1[i]) || isNaN(parts2[i])) return null;
    if (parts1[i] > parts2[i]) return 1;
    if (parts1[i] < parts2[i]) return -1;
  }
  
  return 0;
}

export function AdminPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { user } = useAuth();
  const { config } = useConfig();
  
  // State for system status
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [testingSMTP, setTestingSMTP] = useState(false);
  const [smtpTestResult, setSmtpTestResult] = useState<'working' | 'failed' | null>(null);
  
  // User management state
  const [users, setUsers] = useState<User[]>([]);
  
  // Users table sorting state with localStorage persistence
  const [userSortState, setUserSortState] = useState<UserSortState>(() => {
    try {
      const saved = localStorage.getItem('adminUsersTableSort');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.column && ['username', 'email', 'role', 'status', 'created', 'last_login'].includes(parsed.column) &&
            parsed.order && ['asc', 'desc'].includes(parsed.order)) {
          return parsed;
        }
      }
    } catch (e) {
      // Invalid localStorage data, fall back to default
    }
    return { column: 'username', order: 'asc' };
  });
  const [creatingUser, setCreatingUser] = useState(false);
  const [resettingPassword, setResettingPassword] = useState(false);
  const [deleting, setDeleting] = useState(false);
  
  // User creation form
  const [newUsername, setNewUsername] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  
  // Dialog states
  const [resetPasswordDialog, setResetPasswordDialog] = useState<{
    isOpen: boolean;
    user: User | null;
  }>({ isOpen: false, user: null });
  const [deleteUserDialog, setDeleteUserDialog] = useState<{
    isOpen: boolean;
    user: User | null;
  }>({ isOpen: false, user: null });
  const [viewUser, setViewUser] = useState<User | null>(null);
  const [newPasswordValue, setNewPasswordValue] = useState("");
  const [deleteFromDetails, setDeleteFromDetails] = useState(false);
  
  // Error state
  const [error, setError] = useState<string | null>(null);
  const [backupError, setBackupError] = useState<string | null>(null);
  const [siteUrl, setSiteUrl] = useState("");
  const [registrationEnabled, setRegistrationEnabled] = useState(false);
  const [registrationLocked, setRegistrationLocked] = useState(false);
  const [webhookRateLimit, setWebhookRateLimit] = useState(30);
  const [webhookMaxSizeKB, setWebhookMaxSizeKB] = useState(5);
  
  // Device management state
  const [devices, setDevices] = useState<Device[]>([]);
  
  // Devices table sorting state with localStorage persistence
  const [deviceSortState, setDeviceSortState] = useState<DeviceSortState>(() => {
    try {
      const saved = localStorage.getItem('adminDevicesTableSort');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.column && ['id', 'name', 'user', 'model', 'status', 'last_seen'].includes(parsed.column) &&
            parsed.order && ['asc', 'desc'].includes(parsed.order)) {
          return parsed;
        }
      }
    } catch (e) {
      // Invalid localStorage data, fall back to default
    }
    return { column: 'id', order: 'asc' };
  });
  const [deviceStats, setDeviceStats] = useState<DeviceStats | null>(null);
  const [deviceModels, setDeviceModels] = useState<DeviceModel[]>([]);
  const [viewDevice, setViewDevice] = useState<Device | null>(null);
  const [unlinkingDevice, setUnlinkingDevice] = useState<string | null>(null);
  const [showDeviceModels, setShowDeviceModels] = useState(false);
  
  // Backup/restore state
  const [backupJobs, setBackupJobs] = useState<BackupJob[]>([]);
  const [restoreUploads, setRestoreUploads] = useState<RestoreUpload[]>([]);
  const [deleteBackupDialog, setDeleteBackupDialog] = useState<{
    isOpen: boolean;
    job: BackupJob | null;
  }>({ isOpen: false, job: null });
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [restoreConfirmDialog, setRestoreConfirmDialog] = useState<{
    isOpen: boolean;
    upload: RestoreUpload | null;
  }>({ isOpen: false, upload: null });
  const [backupCounts, setBackupCounts] = useState<{
    users: number;
    api_keys: number;
    documents: number;
  } | null>(null);
  const [backupVersion, setBackupVersion] = useState<string | null>(null);
  const [versionInfo, setVersionInfo] = useState<{
    version: string;
    git_commit: string;
    build_date: string;
    go_version: string;
  } | null>(null);
  const [uploadProgress, setUploadProgress] = useState<number>(0);
  const [uploadPhase, setUploadPhase] = useState<'idle' | 'uploading' | 'extracting' | 'validating'>('idle');
  const [downloadingJobId, setDownloadingJobId] = useState<string | null>(null);
  const [restorePerformed, setRestorePerformed] = useState(false);
  const [creatingBackup, setCreatingBackup] = useState(false);
  const [restoringBackup, setRestoringBackup] = useState(false);
  const [analyzingBackup, setAnalyzingBackup] = useState(false);
  
  // Plugin management state
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  
  // Plugins table sorting state with localStorage persistence
  const [pluginSortState, setPluginSortState] = useState<PluginSortState>(() => {
    try {
      const saved = localStorage.getItem('adminPluginsTableSort');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.column && ['name', 'type', 'version', 'author', 'status', 'created'].includes(parsed.column) &&
            parsed.order && ['asc', 'desc'].includes(parsed.order)) {
          return parsed;
        }
      }
    } catch (e) {
      // Invalid localStorage data, fall back to default
    }
    return { column: 'name', order: 'asc' };
  });
  const [pluginStats, setPluginStats] = useState<PluginStats | null>(null);
  const [viewPlugin, setViewPlugin] = useState<Plugin | null>(null);
  const [showCreatePluginDialog, setShowCreatePluginDialog] = useState(false);
  const [editPlugin, setEditPlugin] = useState<Plugin | null>(null);
  const [deletePlugin, setDeletePlugin] = useState<Plugin | null>(null);
  const [creatingPlugin, setCreatingPlugin] = useState(false);
  const [deletingPlugin, setDeletingPlugin] = useState(false);
  
  // Plugin form state
  const [pluginName, setPluginName] = useState("");
  const [pluginType, setPluginType] = useState("");
  const [pluginDescription, setPluginDescription] = useState("");
  const [pluginConfigSchema, setPluginConfigSchema] = useState("");
  const [pluginVersion, setPluginVersion] = useState("");
  const [pluginAuthor, setPluginAuthor] = useState("");

  // Firmware management state
  const [firmwareVersions, setFirmwareVersions] = useState<FirmwareVersion[]>([]);
  const [firmwareStats, setFirmwareStats] = useState<FirmwareStats | null>(null);
  const [firmwareMode, setFirmwareMode] = useState<string>('proxy');
  const [firmwarePolling, setFirmwarePolling] = useState(false);
  const [modelPolling, setModelPolling] = useState(false);
  const [deleteFirmwareDialog, setDeleteFirmwareDialog] = useState<{
    isOpen: boolean;
    version: FirmwareVersion | null;
  }>({ isOpen: false, version: null });
  const [deletingFirmware, setDeletingFirmware] = useState(false);
  
  // Sort handlers and effects
  const handleUserSort = (column: UserSortColumn) => {
    setUserSortState(prevState => ({
      column,
      order: prevState.column === column && prevState.order === 'asc' ? 'desc' : 'asc'
    }));
  };

  const handleDeviceSort = (column: DeviceSortColumn) => {
    setDeviceSortState(prevState => ({
      column,
      order: prevState.column === column && prevState.order === 'asc' ? 'desc' : 'asc'
    }));
  };

  const handlePluginSort = (column: PluginSortColumn) => {
    setPluginSortState(prevState => ({
      column,
      order: prevState.column === column && prevState.order === 'asc' ? 'desc' : 'asc'
    }));
  };


  // Save sort states to localStorage whenever they change
  useEffect(() => {
    try {
      localStorage.setItem('adminUsersTableSort', JSON.stringify(userSortState));
    } catch (e) {
      // Ignore localStorage errors
    }
  }, [userSortState]);

  useEffect(() => {
    try {
      localStorage.setItem('adminDevicesTableSort', JSON.stringify(deviceSortState));
    } catch (e) {
      // Ignore localStorage errors
    }
  }, [deviceSortState]);

  useEffect(() => {
    try {
      localStorage.setItem('adminPluginsTableSort', JSON.stringify(pluginSortState));
    } catch (e) {
      // Ignore localStorage errors
    }
  }, [pluginSortState]);

  // Sorted arrays
  const sortedUsers = React.useMemo(() => {
    const sorted = [...users].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      switch (userSortState.column) {
        case 'username':
          aValue = a.username?.toLowerCase() || '';
          bValue = b.username?.toLowerCase() || '';
          break;
        case 'email':
          aValue = a.email?.toLowerCase() || '';
          bValue = b.email?.toLowerCase() || '';
          break;
        case 'role':
          aValue = a.is_admin ? 1 : 0;
          bValue = b.is_admin ? 1 : 0;
          break;
        case 'status':
          aValue = a.is_active ? 1 : 0;
          bValue = b.is_active ? 1 : 0;
          break;
        case 'created':
          aValue = new Date(a.created_at).getTime();
          bValue = new Date(b.created_at).getTime();
          break;
        case 'last_login':
          aValue = a.last_login ? new Date(a.last_login).getTime() : 0;
          bValue = b.last_login ? new Date(b.last_login).getTime() : 0;
          break;
        default:
          return 0;
      }

      if (aValue < bValue) {
        return userSortState.order === 'asc' ? -1 : 1;
      }
      if (aValue > bValue) {
        return userSortState.order === 'asc' ? 1 : -1;
      }
      return 0;
    });

    return sorted;
  }, [users, userSortState]);

  const sortedDevices = React.useMemo(() => {
    const sorted = [...devices].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      switch (deviceSortState.column) {
        case 'id':
          aValue = a.friendly_id?.toLowerCase() || '';
          bValue = b.friendly_id?.toLowerCase() || '';
          break;
        case 'name':
          aValue = a.name?.toLowerCase() || '';
          bValue = b.name?.toLowerCase() || '';
          break;
        case 'user':
          aValue = a.user?.username?.toLowerCase() || '';
          bValue = b.user?.username?.toLowerCase() || '';
          break;
        case 'model':
          aValue = a.device_model?.display_name?.toLowerCase() || '';
          bValue = b.device_model?.display_name?.toLowerCase() || '';
          break;
        case 'status':
          aValue = a.is_claimed ? (a.is_active ? 2 : 1) : 0;
          bValue = b.is_claimed ? (b.is_active ? 2 : 1) : 0;
          break;
        case 'last_seen':
          aValue = a.last_seen ? new Date(a.last_seen).getTime() : 0;
          bValue = b.last_seen ? new Date(b.last_seen).getTime() : 0;
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

  const sortedPlugins = React.useMemo(() => {
    const sorted = [...plugins].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      switch (pluginSortState.column) {
        case 'name':
          aValue = a.name?.toLowerCase() || '';
          bValue = b.name?.toLowerCase() || '';
          break;
        case 'type':
          aValue = a.type?.toLowerCase() || '';
          bValue = b.type?.toLowerCase() || '';
          break;
        case 'version':
          aValue = a.version?.toLowerCase() || '';
          bValue = b.version?.toLowerCase() || '';
          break;
        case 'author':
          aValue = a.author?.toLowerCase() || '';
          bValue = b.author?.toLowerCase() || '';
          break;
        case 'status':
          aValue = a.is_active ? 1 : 0;
          bValue = b.is_active ? 1 : 0;
          break;
        case 'created':
          aValue = new Date(a.created_at).getTime();
          bValue = new Date(b.created_at).getTime();
          break;
        default:
          return 0;
      }

      if (aValue < bValue) {
        return pluginSortState.order === 'asc' ? -1 : 1;
      }
      if (aValue > bValue) {
        return pluginSortState.order === 'asc' ? 1 : -1;
      }
      return 0;
    });

    return sorted;
  }, [plugins, pluginSortState]);

  // Detect browser timezone for new users
  const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

  // Fetch system status
  const fetchSystemStatus = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/status", {
        credentials: "include",
      });
      if (response.ok) {
        const status = await response.json();
        setSystemStatus(status);
        setRegistrationEnabled(status.settings.registration_enabled === "true");
        setRegistrationLocked(status.settings.registration_enabled_locked || false);
        setSiteUrl(status.settings.site_url || "");
        setWebhookRateLimit(parseInt(status.settings.webhook_rate_limit_per_hour) || 30);
        setWebhookMaxSizeKB(parseInt(status.settings.webhook_max_request_size_kb) || 5);
      }
    } catch (error) {
      console.error("Failed to fetch system status:", error);
    }
  }, []);

  const updateSystemSetting = async (key: string, value: string) => {
    try {
      const response = await fetch("/api/admin/settings", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({ [key]: value }),
      });
      if (!response.ok) {
        throw new Error("Failed to update setting");
      }
    } catch (error) {
      console.error("Failed to update system setting:", error);
      setError("Failed to update setting");
    }
  };
  
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };
  
  const formatDateOnly = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const formatDateTime = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };
  
  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };
  
  const fetchVersionInfo = async () => {
    try {
      const response = await fetch("/api/version", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setVersionInfo(data);
      }
    } catch (error) {
      console.error("Failed to fetch version info:", error);
    }
  };
  
  const fetchBackupJobs = async () => {
    try {
      const response = await fetch("/api/admin/backup-jobs", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setBackupJobs(data.jobs || []);
      }
    } catch (error) {
      console.error("Failed to fetch backup jobs:", error);
    }
  };
  
  const fetchRestoreUploads = async () => {
    try {
      const response = await fetch("/api/admin/restore/uploads", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setRestoreUploads(data.uploads || []);
      }
    } catch (error) {
      console.error("Failed to fetch restore uploads:", error);
    }
  };

  // Test SMTP configuration
  const testSMTP = async () => {
    try {
      setTestingSMTP(true);
      setSmtpTestResult(null);
      const response = await fetch("/api/admin/test-smtp", {
        method: "POST",
        credentials: "include",
      });
      const result = await response.json();
      if (response.ok) {
        setSmtpTestResult('working');
        // Revert back to default status after 3 seconds
        setTimeout(() => setSmtpTestResult(null), 3000);
      } else {
        setSmtpTestResult('failed');
        setTimeout(() => setSmtpTestResult(null), 3000);
      }
    } catch (error) {
      setSmtpTestResult('failed');
      setTimeout(() => setSmtpTestResult(null), 3000);
    } finally {
      setTestingSMTP(false);
    }
  };

  const handleCreateBackupJob = async () => {
    try {
      console.log("Starting backup job creation...");
      setCreatingBackup(true);
      setBackupError(null);
      
      const response = await fetch("/api/admin/backup-job", {
        method: "POST",
        credentials: "include",
      });
      
      console.log("Backup job response status:", response.status);
      console.log("Backup job response headers:", response.headers);
      
      if (response.ok) {
        const data = await response.json();
        console.log("Backup job created successfully:", data);
        await fetchBackupJobs();
      } else {
        console.error("Backup job creation failed with status:", response.status);
        try {
          const errorData = await response.json();
          console.error("Error response data:", errorData);
          const errorMessage = errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : 
                              errorData.error || errorData.message || 
                              `Failed to create backup (HTTP ${response.status})`;
          setBackupError(errorMessage);
        } catch (parseError) {
          console.error("Failed to parse error response:", parseError);
          setBackupError(`Failed to create backup (HTTP ${response.status})`);
        }
      }
    } catch (error) {
      console.error("Network error creating backup job:", error);
      setBackupError("Network error: Failed to create backup");
    } finally {
      setCreatingBackup(false);
    }
  };

  const handleDownloadBackupJob = async (jobId: string) => {
    try {
      setDownloadingJobId(jobId);
      const response = await fetch(`/api/admin/backup-job/${jobId}/download`, {
        credentials: "include",
      });
      if (response.ok) {
        const contentDisposition = response.headers.get("Content-Disposition");
        let filename = "backup.tar.gz";
        if (contentDisposition) {
          const matches = contentDisposition.match(/filename=([^;]+)/);
          if (matches && matches[1]) {
            filename = matches[1].replace(/"/g, '');
          }
        }
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
      } else {
        const errorData = await response.json();
        setBackupError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : "Failed to download backup");
      }
    } catch (error) {
      setBackupError("Failed to download backup");
    } finally {
      setDownloadingJobId(null);
    }
  };

  const openDeleteBackupDialog = (job: BackupJob) => {
    setDeleteBackupDialog({ isOpen: true, job });
  };

  const closeDeleteBackupDialog = () => {
    setDeleteBackupDialog({ isOpen: false, job: null });
  };

  const confirmDeleteBackupJob = async () => {
    if (!deleteBackupDialog.job) return;
    try {
      setDeleting(true);
      const response = await fetch(`/api/admin/backup-job/${deleteBackupDialog.job.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (response.ok) {
        await fetchBackupJobs();
        closeDeleteBackupDialog();
      } else {
        const errorData = await response.json();
        setBackupError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : "Failed to delete backup");
      }
    } catch (error) {
      setBackupError("Failed to delete backup");
    } finally {
      setDeleting(false);
    }
  };

  const uploadRestoreFileWithProgress = async (formData: FormData): Promise<Response> => {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable) {
          const percentComplete = (event.loaded / event.total) * 100;
          setUploadProgress(percentComplete);
        }
      });
      
      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve({
            ok: true,
            status: xhr.status,
            json: () => Promise.resolve(JSON.parse(xhr.responseText))
          } as Response);
        } else {
          reject(new Error(`HTTP ${xhr.status}: ${xhr.statusText}`));
        }
      });
      
      xhr.addEventListener('error', () => {
        reject(new Error('Network error'));
      });
      
      xhr.open('POST', '/api/admin/restore/upload');
      xhr.withCredentials = true;
      xhr.send(formData);
    });
  };

  const handleRestoreFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    
    const isTarGz = file.name.endsWith('.tar.gz') || file.name.endsWith('.tgz') || 
                   file.type === 'application/gzip' || file.type === 'application/x-gzip' ||
                   file.type === 'application/x-tar' || file.type === 'application/x-compressed-tar';
    
    if (!isTarGz) {
      setBackupError("Invalid file type. Please select a .tar.gz backup file: " + `"${file.name}"`);
      event.target.value = "";
      return;
    }
    
    try {
      setBackupError(null);
      setUploadProgress(0);
      setUploadPhase('idle');
      const formData = new FormData();
      formData.append("backup_file", file);
      setUploadPhase('uploading');
      const response = await uploadRestoreFileWithProgress(formData);
      
      if (response.ok) {
        const responseData = await response.json();
        const uploadId = responseData.upload_id;
        
        setUploadPhase('extracting');
        setUploadProgress(100);
        
        if (uploadId) {
          try {
            setUploadPhase('validating');
            
            const analyzeResponse = await fetch(`/api/admin/restore/uploads/${uploadId}/analyze`, {
              method: "POST",
              credentials: "include",
            });
            
            if (analyzeResponse.ok) {
              const analyzeData = await analyzeResponse.json();
              setBackupCounts(analyzeData.counts);
              setBackupVersion(analyzeData.version);
            }
          } catch (error) {
            console.error("Failed to analyze backup during upload:", error);
          }
        }
        
        await fetchRestoreUploads();
        
        setUploadProgress(0);
        setUploadPhase('idle');
      } else {
        const errorData = await response.json();
        setBackupError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : "Failed to upload restore file");
        setUploadProgress(0);
        setUploadPhase('idle');
      }
    } catch (error) {
      setBackupError("Failed to upload restore file");
      setUploadProgress(0);
      setUploadPhase('idle');
    }
    event.target.value = "";
  };

  const handleRestoreUpload = async (upload: RestoreUpload) => {
    setRestoreConfirmDialog({ isOpen: true, upload });
  };

  const closeRestoreConfirmDialog = () => {
    setRestoreConfirmDialog({ isOpen: false, upload: null });
  };

  const confirmDatabaseRestore = async () => {
    const upload = restoreConfirmDialog.upload;
    if (!upload) return;
    
    try {
      setRestoringBackup(true);
      setBackupError(null);
      const response = await fetch("/api/admin/restore", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({ upload_id: upload.id }),
      });
      
      if (response.ok) {
        setRestoreConfirmDialog({ isOpen: false, upload: null });
        setBackupError(null);
        const filename = restoreConfirmDialog.upload?.filename || 'backup file';
        setRestorePerformed(true);
        
        try {
          await fetchSystemStatus();
          await fetchUsers();
          await fetchBackupJobs();
          await fetchRestoreUploads();
        } catch (error) {
          return;
        }
      } else {
        const result = await response.json();
        setBackupError(result.error_type ? t(`admin.errors.${result.error_type}`) : "Failed to restore backup");
      }
    } catch (error) {
      setBackupError("Restore error: " + error.message);
    } finally {
      setRestoringBackup(false);
    }
  };

  const cancelRestoreUpload = async () => {
    const upload = restoreConfirmDialog.upload;
    if (!upload) return;
    setRestoreUploads([]);
    closeRestoreConfirmDialog();
    try {
      const response = await fetch(`/api/admin/restore/uploads/${upload.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!response.ok) {
        const errorData = await response.json();
        setBackupError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : "Failed to cancel restore");
      }
      await fetchRestoreUploads();
    } catch (error) {
      setBackupError("Failed to cancel restore");
      await fetchRestoreUploads();
    }
  };

  const getBackupStatusButton = (job: BackupJob) => {
    if (job.status === "completed" || job.status === "failed") {
      return (
        <div className="flex gap-2">
          {job.status === "completed" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleDownloadBackupJob(job.id)}
              disabled={downloadingJobId === job.id}
              className="bg-background"
            >
              {downloadingJobId === job.id ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="hidden sm:inline ml-2">Downloading...</span>
                </>
              ) : (
                <>
                  <Download className="h-4 w-4 sm:hidden" />
                  <span className="hidden sm:inline">Download</span>
                </>
              )}
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => openDeleteBackupDialog(job)}
            disabled={deleting}
            className="bg-background text-destructive hover:text-destructive"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      );
    }
    return null;
  };

  // Fetch users
  const fetchUsers = useCallback(async () => {
    try {
      const response = await fetch("/api/users", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setUsers(data.users);
      }
    } catch (error) {
      console.error("Failed to fetch users:", error);
    }
  }, []);

  // Fetch devices
  const fetchDevices = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/devices", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setDevices(data.devices || []);
      }
    } catch (error) {
      console.error("Failed to fetch devices:", error);
    }
  }, []);

  // Fetch device statistics
  const fetchDeviceStats = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/devices/stats", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setDeviceStats(data);
      }
    } catch (error) {
      console.error("Failed to fetch device stats:", error);
    }
  }, []);

  // Fetch device models
  const fetchDeviceModels = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/device-models", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setDeviceModels(data.device_models || []);
      }
    } catch (error) {
      console.error("Failed to fetch device models:", error);
    }
  }, []);

  // Create new user
  const createUser = async () => {
    try {
      setCreatingUser(true);
      setError(null);
      const response = await fetch("/api/auth/register", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          username: newUsername,
          email: newEmail,
          password: newPassword,
          timezone: browserTimezone,
        }),
      });
      if (response.ok) {
        setNewUsername("");
        setNewEmail("");
        setNewPassword("");
        await fetchUsers();
        await fetchSystemStatus();
      } else {
        const errorData = await response.json();
        setError(errorData.error ? t(errorData.error) : t("admin.errors.create_user"));
      }
    } catch (error) {
      setError(t("admin.errors.create_user"));
    } finally {
      setCreatingUser(false);
    }
  };

  // Toggle user active status
  const toggleUserStatus = async (userId: string, isActive: boolean) => {
    try {
      const endpoint = isActive ? "activate" : "deactivate";
      const response = await fetch(`/api/users/${userId}/${endpoint}`, {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        await fetchUsers();
        if (viewUser && viewUser.id === userId) {
          setViewUser(prev => prev ? { ...prev, is_active: isActive } : null);
        }
      }
    } catch (error) {
      console.error("Failed to toggle user status:", error);
    }
  };

  // Toggle admin status
  const toggleAdminStatus = async (userId: string, makeAdmin: boolean) => {
    try {
      const endpoint = makeAdmin ? "promote" : "demote";
      const response = await fetch(`/api/users/${userId}/${endpoint}`, {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        await fetchUsers();
        if (viewUser && viewUser.id === userId) {
          setViewUser(prev => prev ? { ...prev, is_admin: makeAdmin } : null);
        }
      }
    } catch (error) {
      console.error("Failed to toggle admin status:", error);
    }
  };

  // Reset password dialog functions
  const openResetPasswordDialog = (user: User) => {
    setResetPasswordDialog({ isOpen: true, user });
    setNewPasswordValue("");
    setError(null);
  };

  const closeResetPasswordDialog = () => {
    setResetPasswordDialog({ isOpen: false, user: null });
    setNewPasswordValue("");
    setError(null);
  };

  const confirmResetPassword = async () => {
    if (!resetPasswordDialog.user || !newPasswordValue) return;
    if (newPasswordValue.length < 8) {
      setError(t("admin.errors.password_length"));
      return;
    }
    try {
      setResettingPassword(true);
      const response = await fetch(
        `/api/users/${resetPasswordDialog.user.id}/reset-password`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({ new_password: newPasswordValue }),
        },
      );
      if (response.ok) {
        closeResetPasswordDialog();
      } else {
        const errorData = await response.json();
        setError(errorData.error ? t(errorData.error) : t("admin.errors.reset_password"));
      }
    } catch (error) {
      setError(t("admin.errors.reset_password"));
    } finally {
      setResettingPassword(false);
    }
  };

  // Delete user dialog functions
  const openDeleteUserDialog = (user: User) => {
    setDeleteUserDialog({ isOpen: true, user });
  };

  const closeDeleteUserDialog = () => {
    const wasFromDetails = deleteFromDetails;
    const userToRestore = deleteUserDialog.user;
    setDeleteUserDialog({ isOpen: false, user: null });
    setDeleteFromDetails(false);
    if (wasFromDetails && userToRestore) {
      setViewUser(userToRestore);
    }
  };

  const confirmDeleteUser = async () => {
    if (!deleteUserDialog.user) return;
    try {
      setDeleting(true);
      const response = await fetch(`/api/users/${deleteUserDialog.user.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (response.ok) {
        await fetchUsers();
        await fetchSystemStatus();
        setDeleteUserDialog({ isOpen: false, user: null });
        setDeleteFromDetails(false);
        setViewUser(null);
      } else {
        const errorData = await response.json();
        setError(errorData.error ? t(errorData.error) : t("admin.errors.delete_user"));
      }
    } catch (error) {
      setError(t("admin.errors.delete_user"));
    } finally {
      setDeleting(false);
    }
  };

  // Device management functions
  const unlinkDevice = async (deviceId: string) => {
    try {
      setUnlinkingDevice(deviceId);
      setError(null);
      const response = await fetch(`/api/admin/devices/${deviceId}/unlink`, {
        method: "DELETE",
        credentials: "include",
      });
      if (response.ok) {
        await fetchDevices();
        await fetchDeviceStats();
        setViewDevice(null);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to unlink device");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setUnlinkingDevice(null);
    }
  };

  // Plugin management functions
  const fetchPlugins = useCallback(async () => {
    try {
      const response = await fetch("/api/plugin-definitions", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPlugins(data.plugins || []);
      }
    } catch (error) {
      console.error("Failed to fetch plugins:", error);
    }
  }, []);

  const fetchPluginStats = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/plugins/stats", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPluginStats(data);
      }
    } catch (error) {
      console.error("Failed to fetch plugin stats:", error);
    }
  }, []);

  const createPlugin = async () => {
    if (!pluginName.trim()) {
      setError("Please fill in required fields");
      return;
    }
    try {
      setCreatingPlugin(true);
      setError(null);
      const response = await fetch("/api/admin/plugins", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: pluginName.trim(),
          type: pluginType.trim(),
          description: pluginDescription.trim(),
          config_schema: pluginConfigSchema.trim(),
          version: pluginVersion.trim(),
          author: pluginAuthor.trim(),
        }),
      });
      if (response.ok) {
        setShowCreatePluginDialog(false);
        resetPluginForm();
        await fetchPlugins();
        await fetchPluginStats();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to create plugin");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setCreatingPlugin(false);
    }
  };

  const updatePlugin = async () => {
    if (!editPlugin) return;
    try {
      setError(null);
      const response = await fetch(`/api/admin/plugins/${editPlugin.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: pluginName.trim(),
          type: pluginType.trim(),
          description: pluginDescription.trim(),
          config_schema: pluginConfigSchema.trim(),
          version: pluginVersion.trim(),
          author: pluginAuthor.trim(),
          is_active: editPlugin.is_active,
        }),
      });
      if (response.ok) {
        setEditPlugin(null);
        resetPluginForm();
        await fetchPlugins();
        await fetchPluginStats();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update plugin");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const confirmDeletePlugin = async () => {
    if (!deletePlugin) return;
    try {
      setDeletingPlugin(true);
      setError(null);
      const response = await fetch(`/api/admin/plugins/${deletePlugin.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (response.ok) {
        setDeletePlugin(null);
        await fetchPlugins();
        await fetchPluginStats();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete plugin");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setDeletingPlugin(false);
    }
  };

  const togglePluginStatus = async (plugin: Plugin) => {
    try {
      setError(null);
      const response = await fetch(`/api/admin/plugins/${plugin.id}/toggle`, {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        await fetchPlugins();
        await fetchPluginStats();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update plugin status");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  const openEditPluginDialog = (plugin: Plugin) => {
    setEditPlugin(plugin);
    setPluginName(plugin.name);
    setPluginType(plugin.type);
    setPluginDescription(plugin.description);
    setPluginConfigSchema(plugin.config_schema);
    setPluginVersion(plugin.version);
    setPluginAuthor(plugin.author || "");
  };

  const resetPluginForm = () => {
    setPluginName("");
    setPluginType("");
    setPluginDescription("");
    setPluginConfigSchema("");
    setPluginVersion("");
    setPluginAuthor("");
  };

  const hasPluginChanges = () => {
    if (!editPlugin) return false;
    return (
      pluginName.trim() !== editPlugin.name ||
      pluginType.trim() !== editPlugin.type ||
      pluginDescription.trim() !== editPlugin.description ||
      pluginConfigSchema.trim() !== editPlugin.config_schema ||
      pluginVersion.trim() !== editPlugin.version ||
      pluginAuthor.trim() !== (editPlugin.author || "")
    );
  };

  // Firmware management functions
  const fetchFirmwareVersions = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/firmware/versions", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setFirmwareVersions(data.firmware_versions || []);
      }
    } catch (error) {
      console.error("Failed to fetch firmware versions:", error);
    }
  }, []);

  const fetchFirmwareStats = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/firmware/stats", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setFirmwareStats(data);
      }
    } catch (error) {
      console.error("Failed to fetch firmware stats:", error);
    }
  }, []);

  const fetchFirmwareMode = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/firmware/mode", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setFirmwareMode(data.firmware_mode || 'proxy');
      }
    } catch (error) {
      console.error("Failed to fetch firmware mode:", error);
    }
  }, []);

  const triggerFirmwarePoll = async () => {
    try {
      setFirmwarePolling(true);
      const response = await fetch("/api/admin/firmware/poll", {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        await fetchFirmwareVersions();
        await fetchFirmwareStats();
      } else {
        setError("Failed to trigger firmware poll");
      }
    } catch (error) {
      console.error("Failed to trigger firmware poll:", error);
      setError("Failed to trigger firmware poll");
    } finally {
      setFirmwarePolling(false);
    }
  };

  const retryFirmwareDownload = async (versionId: string, version: string) => {
    try {
      const response = await fetch(`/api/admin/firmware/versions/${versionId}/retry`, {
        method: "POST",
        credentials: "include",
      });
      
      if (response.ok) {
        // You could add a success message here if you have that state
      } else {
        const data = await response.json();
        setError(data.error || "Failed to start firmware download");
      }
    } catch (error) {
      console.error("Failed to retry firmware download:", error);
      setError("Failed to start firmware download");
    }
  };

  const openDeleteFirmwareDialog = (version: FirmwareVersion) => {
    setDeleteFirmwareDialog({ isOpen: true, version });
  };

  const closeDeleteFirmwareDialog = () => {
    setDeleteFirmwareDialog({ isOpen: false, version: null });
  };

  const confirmDeleteFirmware = async () => {
    if (!deleteFirmwareDialog.version) return;
    try {
      setDeletingFirmware(true);
      const response = await fetch(`/api/admin/firmware/versions/${deleteFirmwareDialog.version.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      
      if (response.ok) {
        await fetchFirmwareVersions();
        await fetchFirmwareStats();
        closeDeleteFirmwareDialog();
      } else {
        const data = await response.json();
        setError(data.error || "Failed to delete firmware version");
      }
    } catch (error) {
      console.error("Failed to delete firmware version:", error);
      setError("Failed to delete firmware version");
    } finally {
      setDeletingFirmware(false);
    }
  };

  const triggerModelPoll = async () => {
    try {
      setModelPolling(true);
      const response = await fetch("/api/admin/models/poll", {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        await fetchDeviceModels();
        await fetchFirmwareStats();
      } else {
        setError("Failed to trigger model poll");
      }
    } catch (error) {
      console.error("Failed to trigger model poll:", error);
      setError("Failed to trigger model poll");
    } finally {
      setModelPolling(false);
    }
  };

  // SMTP status helper functions
  const getSMTPStatusColor = (status: string) => {
    // If we have a test result, show that instead
    if (smtpTestResult === 'working') {
      return "default";
    }
    if (smtpTestResult === 'failed') {
      return "destructive";
    }
    
    // Otherwise show configuration status
    switch (status) {
      case "configured":
        return "secondary";
      case "not_configured":
        return "secondary";
      default:
        return "secondary";
    }
  };

  const getSMTPStatusText = (status: string) => {
    // If we have a test result, show that instead
    if (smtpTestResult === 'working') {
      return t("admin.status.working");
    }
    if (smtpTestResult === 'failed') {
      return t("admin.status.failed");
    }
    
    // Otherwise show configuration status
    switch (status) {
      case "configured":
        return t("admin.status.configured");
      case "not_configured":
        return t("admin.status.not_configured");
      default:
        return t("admin.status.unknown");
    }
  };

  // User helper functions
  const getUserStatusBadge = (user: User) => {
    if (!user.is_active) {
      return (
        <Badge variant="secondary" className="min-w-16 max-w-32 justify-center text-center whitespace-nowrap">
          {t("admin.status.inactive")}
        </Badge>
      );
    }
    
    if (user.rmapi_paired) {
      return (
        <Badge variant="default" className="min-w-16 max-w-32 justify-center text-center whitespace-nowrap">
          {t("admin.status.paired")}
        </Badge>
      );
    }
    
    return (
      <Badge variant="outline" className="min-w-16 max-w-32 justify-center text-center whitespace-nowrap">
        {t("admin.status.active")}
      </Badge>
    );
  };

  const isCurrentUser = (checkUser: User) => {
    return user && checkUser.id === user.id;
  };

  // Auto-refresh firmware versions when downloads are in progress
  useEffect(() => {
    const hasDownloadingVersions = firmwareVersions.some(
      version => version.download_status === 'downloading'
    );
    
    if (hasDownloadingVersions) {
      const interval = setInterval(() => {
        fetchFirmwareVersions();
      }, 2000); // Poll every 2 seconds
      
      return () => clearInterval(interval);
    }
  }, [firmwareVersions, fetchFirmwareVersions]);

  // Fetch system status and users on mount
  useEffect(() => {
    fetchSystemStatus();
    fetchUsers();
    fetchDevices();
    fetchDeviceStats();
    fetchDeviceModels();
    fetchPlugins();
    fetchPluginStats();
    fetchFirmwareVersions();
    fetchFirmwareStats();
    fetchFirmwareMode();
    fetchBackupJobs();
    fetchRestoreUploads();
    fetchVersionInfo();
  }, [fetchSystemStatus, fetchUsers, fetchDevices, fetchDeviceStats, fetchDeviceModels, fetchPlugins, fetchPluginStats, fetchFirmwareVersions, fetchFirmwareStats, fetchFirmwareMode]);
  
  // Polling for active backup jobs
  useEffect(() => {
    let interval: NodeJS.Timeout;
    const hasActiveJobs = 
      backupJobs.some(job => job.status === 'running' || job.status === 'pending');
      
    if (hasActiveJobs) {
      interval = setInterval(() => {
        fetchBackupJobs();
      }, 2000);
    }

    return () => {
      if (interval) clearInterval(interval);
    };
  }, [backupJobs]);

  if (!user?.is_admin) {
    return (
      <div className="bg-background pt-0 pb-8 px-0 sm:px-8">
        <div className="max-w-6xl mx-0 sm:mx-auto space-y-6">
          <div className="text-center">
            <h1 className="text-2xl font-bold mb-4">Access Denied</h1>
            <p className="text-muted-foreground">You don't have permission to access this page.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      {/* Sticky Header */}
      <div className="sticky top-0 z-40 border-b bg-background">
        <div className="container mx-auto px-4 py-4 space-y-4">
          {/* Breadcrumb */}
          <div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/')}
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              Back to Dashboard
            </Button>
          </div>
          
          {/* Title */}
          <div>
            <h1 className="text-2xl font-semibold">{t("admin.title")}</h1>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="container mx-auto px-4 py-6 space-y-6">
            <Tabs defaultValue="overview" className="w-full">
              <TabsList className="w-full">
                <TabsTrigger value="overview">
                  <Activity className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">{t("admin.tabs.overview")}</span>
                </TabsTrigger>
                <TabsTrigger value="users">
                  <Users className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">{t("admin.tabs.users")}</span>
                </TabsTrigger>
                <TabsTrigger value="devices">
                  <Monitor className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">Devices</span>
                </TabsTrigger>
                <TabsTrigger value="plugins">
                  <Puzzle className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">Plugins</span>
                </TabsTrigger>
                <TabsTrigger value="firmware">
                  <Download className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">Firmware</span>
                </TabsTrigger>
                <TabsTrigger value="settings">
                  <SettingsIcon className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">{t("admin.tabs.settings")}</span>
                </TabsTrigger>
                <TabsTrigger value="system">
                  <Database className="h-4 w-4" />
                  <span className="hidden sm:inline ml-1.5">{t("admin.tabs.system")}</span>
                </TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="mt-6">
                {systemStatus ? (
                  <>
                    <div className="grid grid-cols-2 md:grid-cols-2 lg:grid-cols-4 gap-4">
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm font-medium">
                            {t("admin.cards.total_users")}
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">
                            {systemStatus.database.total_users}
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {systemStatus.database.active_users} {t("admin.status.active")}
                          </p>
                        </CardContent>
                      </Card>
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm font-medium">
                            {t("admin.cards.api_keys")}
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">
                            {systemStatus.database.api_keys.total}
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {systemStatus.database.api_keys.active} {t("admin.status.active")}
                          </p>
                        </CardContent>
                      </Card>
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm font-medium">
                            {t("admin.cards.documents")}
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">
                            {systemStatus.database.documents}
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {t("admin.descriptions.total_uploaded")}
                          </p>
                        </CardContent>
                      </Card>
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm font-medium">
                            {t("admin.cards.active_sessions")}
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">
                            {systemStatus.database.active_sessions}
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {t("admin.descriptions.current_sessions")}
                          </p>
                        </CardContent>
                      </Card>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Card>
                        <CardHeader>
                          <CardTitle className="flex items-center gap-2">
                            <Mail className="h-5 w-5" />
                            {t("admin.cards.smtp_status")}
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="flex flex-col sm:flex-row items-start sm:items-center sm:justify-between gap-2">
                            <Badge
                              variant={getSMTPStatusColor(systemStatus.smtp.status)}
                            >
                              {getSMTPStatusText(systemStatus.smtp.status)}
                            </Badge>
                            <Button 
                              size="sm" 
                              onClick={testSMTP} 
                              disabled={testingSMTP || !systemStatus.smtp.configured} 
                              className="w-full sm:w-auto"
                            >
                              {t("admin.actions.test_smtp")}
                            </Button>
                          </div>
                        </CardContent>
                      </Card>
                      <Card>
                        <CardHeader>
                          <CardTitle className="flex items-center gap-2">
                            <Server className="h-5 w-5" />
                            {t("admin.cards.system_mode")}
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          <div className="flex flex-wrap gap-2 items-start">
                            <Badge variant="secondary">{t("admin.badges.multi_user")}</Badge>
                            {systemStatus.dry_run && (
                              <Badge variant="default">{t("admin.badges.dry_run")}</Badge>
                            )}
                            {systemStatus.auth?.oidc_enabled && (
                              <Badge variant="secondary">{t("admin.badges.oidc_enabled")}</Badge>
                            )}
                            {systemStatus.auth?.proxy_auth_enabled && (
                              <Badge variant="secondary">{t("admin.badges.proxy_auth")}</Badge>
                            )}
                          </div>
                        </CardContent>
                      </Card>
                    </div>
                  </>
                ) : (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <Activity className="h-5 w-5" />
                        System Overview
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-center py-8 text-muted-foreground">
                        <Activity className="h-12 w-12 mx-auto mb-4 opacity-50" />
                        <h3 className="text-lg font-medium mb-2">Loading system status...</h3>
                        <p>Please wait while we fetch the system information.</p>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </TabsContent>

              <TabsContent value="users" className="mt-6">
                <div className="space-y-4">
                  {error && (
                    <Card className="border-destructive">
                      <CardContent className="pt-6">
                        <div className="flex items-center gap-2 text-destructive">
                          <AlertTriangle className="h-4 w-4" />
                          <span>{error}</span>
                        </div>
                      </CardContent>
                    </Card>
                  )}
                  <Card>
                    <CardHeader>
                      <CardTitle>{t("admin.cards.create_new_user")}</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div>
                          <Label htmlFor="new-username">{t("admin.labels.username")}</Label>
                          <Input
                            id="new-username"
                            value={newUsername}
                            onChange={(e) => setNewUsername(e.target.value)}
                            placeholder={t("admin.placeholders.username")}
                            className="mt-2"
                            minLength={3}
                            maxLength={50}
                            pattern="^[a-zA-Z0-9][a-zA-Z0-9_-]{2,49}$"
                            title={t("register.username_help")}
                          />
                        </div>
                        <div>
                          <Label htmlFor="new-email">{t("admin.labels.email")}</Label>
                          <Input
                            id="new-email"
                            type="email"
                            value={newEmail}
                            onChange={(e) => setNewEmail(e.target.value)}
                            placeholder={t("admin.placeholders.email")}
                            className="mt-2"
                          />
                        </div>
                        <div>
                          <Label htmlFor="new-password">{t("admin.labels.password")}</Label>
                          <Input
                            id="new-password"
                            type="password"
                            value={newPassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter' && newUsername && newEmail && newPassword && !creatingUser) {
                                createUser();
                              }
                            }}
                            placeholder={t("admin.placeholders.password")}
                            className="mt-2"
                          />
                        </div>
                      </div>
                      <div className="flex flex-col sm:flex-row sm:justify-end">
                        <Button
                          onClick={createUser}
                          className="w-full sm:w-auto"
                          disabled={
                            creatingUser || !newUsername || !newEmail || !newPassword
                          }
                        >
                          {creatingUser ? t("register.creating") : t("admin.actions.create_user")}
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>{t("admin.counts.users", {count: users.length})}</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <Table className="w-full table-fixed lg:table-auto">
                          <TableHeader>
                            <TableRow>
                              <TableHead 
                                className="cursor-pointer hover:bg-muted/50 select-none"
                                onClick={() => handleUserSort('username')}
                              >
                                <div className="flex items-center gap-1">
                                  {t("admin.labels.username")}
                                  {userSortState.column === 'username' ? (
                                    userSortState.order === 'asc' ? (
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
                                onClick={() => handleUserSort('email')}
                              >
                                <div className="flex items-center gap-1">
                                  {t("admin.labels.email")}
                                  {userSortState.column === 'email' ? (
                                    userSortState.order === 'asc' ? (
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
                                className="hidden lg:table-cell text-center cursor-pointer hover:bg-muted/50 select-none"
                                onClick={() => handleUserSort('role')}
                              >
                                <div className="flex items-center justify-center gap-1">
                                  {t("admin.labels.role")}
                                  {userSortState.column === 'role' ? (
                                    userSortState.order === 'asc' ? (
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
                                className="hidden lg:table-cell text-center cursor-pointer hover:bg-muted/50 select-none"
                                onClick={() => handleUserSort('status')}
                              >
                                <div className="flex items-center justify-center gap-1">
                                  {t("admin.labels.status")}
                                  {userSortState.column === 'status' ? (
                                    userSortState.order === 'asc' ? (
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
                                onClick={() => handleUserSort('created')}
                              >
                                <div className="flex items-center gap-1">
                                  {t("admin.labels.created")}
                                  {userSortState.column === 'created' ? (
                                    userSortState.order === 'asc' ? (
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
                                onClick={() => handleUserSort('last_login')}
                              >
                                <div className="flex items-center gap-1">
                                  {t("admin.labels.last_login")}
                                  {userSortState.column === 'last_login' ? (
                                    userSortState.order === 'asc' ? (
                                      <ChevronUp className="h-4 w-4" />
                                    ) : (
                                      <ChevronDown className="h-4 w-4" />
                                    )
                                  ) : (
                                    <ChevronsUpDown className="h-4 w-4 opacity-50" />
                                  )}
                                </div>
                              </TableHead>
                              <TableHead>{t("admin.labels.actions")}</TableHead>
                            </TableRow>
                          </TableHeader>
                        <TableBody>
                          {sortedUsers.map((user) => (
                            <TableRow key={user.id}>
                              <TableCell className="font-medium">
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <div className="truncate cursor-default" title={user.username}>
                                      {user.username}
                                    </div>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    {t('admin.labels.user_id')}: {user.id}
                                  </TooltipContent>
                                </Tooltip>
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">{user.email}</TableCell>
                              <TableCell className="hidden lg:table-cell text-center">
                                {config?.oidcGroupBasedAdmin ? (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <Badge
                                        variant={user.is_admin ? "default" : "outline"}
                                        className="min-w-14 max-w-24 justify-center cursor-default text-center whitespace-nowrap"
                                      >
                                        {user.is_admin ? t("admin.roles.admin") : t("admin.roles.user")}
                                      </Badge>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      {t("admin.tooltips.oidc_managed")}
                                    </TooltipContent>
                                  </Tooltip>
                                ) : (
                                  <Badge
                                    variant={user.is_admin ? "default" : "outline"}
                                    className="min-w-14 max-w-24 justify-center text-center whitespace-nowrap"
                                  >
                                    {user.is_admin ? t("admin.roles.admin") : t("admin.roles.user")}
                                  </Badge>
                                )}
                              </TableCell>
                              <TableCell className="hidden lg:table-cell text-center">
                                {getUserStatusBadge(user)}
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span className="cursor-default">{formatDateOnly(user.created_at)}</span>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    {formatDateTime(user.created_at)}
                                  </TooltipContent>
                                </Tooltip>
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                {user.last_login ? (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="cursor-default">{formatDateOnly(user.last_login)}</span>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      {formatDateTime(user.last_login)}
                                    </TooltipContent>
                                  </Tooltip>
                                ) : (
                                  t("admin.never")
                                )}
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-2 flex-wrap">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    className="lg:hidden w-full sm:w-auto"
                                    onClick={() => setViewUser(user)}
                                  >
                                    {t('admin.actions.details', 'Details')}
                                  </Button>
                                  <Popover>
                                    <PopoverTrigger asChild>
                                      <Button
                                        size="sm"
                                        variant="outline"
                                        className="hidden lg:inline-flex"
                                      >
                                        {t("admin.actions.modify")}
                                      </Button>
                                    </PopoverTrigger>
                                    <PopoverContent className="w-auto p-2" align="end">
                                      <div className="flex flex-col gap-2">
                                        <Button
                                          size="sm"
                                          variant="outline"
                                          onClick={() => openResetPasswordDialog(user)}
                                          className="w-full justify-start"
                                        >
                                          {t("admin.actions.reset_password")}
                                        </Button>
                                        {!isCurrentUser(user) && (
                                          <>
                                            {!config?.oidcGroupBasedAdmin && (
                                              <Button
                                                size="sm"
                                                variant="outline"
                                                onClick={() =>
                                                  toggleAdminStatus(user.id, !user.is_admin)
                                                }
                                                className="w-full justify-start"
                                              >
                                                {user.is_admin ? t("admin.actions.make_user") : t("admin.actions.make_admin")}
                                              </Button>
                                            )}
                                            <Button
                                              size="sm"
                                              variant="outline"
                                              onClick={() =>
                                                toggleUserStatus(user.id, !user.is_active)
                                              }
                                              className="w-full justify-start"
                                            >
                                              {user.is_active ? t("admin.actions.deactivate") : t("admin.actions.activate")}
                                            </Button>
                                          </>
                                        )}
                                      </div>
                                    </PopoverContent>
                                  </Popover>
                                  {!isCurrentUser(user) && (
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => openDeleteUserDialog(user)}
                                      className="hidden lg:inline-flex whitespace-nowrap"
                                    >
                                      {t("admin.actions.delete")}
                                    </Button>
                                  )}
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="devices" className="mt-6">
                <div className="space-y-4">
                  {/* Device Statistics Cards */}
                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Total Devices</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {deviceStats?.total_devices || 0}
                        </div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Claimed</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold text-emerald-600">
                          {deviceStats?.claimed_devices || 0}
                        </div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Unclaimed</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold text-amber-600">
                          {deviceStats?.unclaimed_devices || 0}
                        </div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Active</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold text-blue-600">
                          {deviceStats?.active_devices || 0}
                        </div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Recent</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {deviceStats?.recent_registrations || 0}
                        </div>
                        <p className="text-xs text-muted-foreground">Last 7 days</p>
                      </CardContent>
                    </Card>
                  </div>
                  {/* Device List */}
                  <Card>
                    <CardHeader>
                      <CardTitle>All Devices ({devices.length})</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead 
                              className="cursor-pointer hover:bg-muted/50 select-none"
                              onClick={() => handleDeviceSort('id')}
                            >
                              <div className="flex items-center gap-1">
                                ID
                                {deviceSortState.column === 'id' ? (
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
                              onClick={() => handleDeviceSort('name')}
                            >
                              <div className="flex items-center gap-1">
                                Name
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
                              className="hidden md:table-cell cursor-pointer hover:bg-muted/50 select-none"
                              onClick={() => handleDeviceSort('user')}
                            >
                              <div className="flex items-center gap-1">
                                Owner
                                {deviceSortState.column === 'user' ? (
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
                              onClick={() => handleDeviceSort('model')}
                            >
                              <div className="flex items-center gap-1">
                                Model
                                {deviceSortState.column === 'model' ? (
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
                            <TableHead className="hidden lg:table-cell">MAC Address</TableHead>
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
                            <TableHead>Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {sortedDevices.map((device) => (
                            <TableRow key={device.id}>
                              <TableCell className="font-medium">
                                <code className="text-sm">{device.friendly_id}</code>
                              </TableCell>
                              <TableCell>
                                {device.name || <span className="text-muted-foreground">Unnamed</span>}
                              </TableCell>
                              <TableCell className="hidden md:table-cell">
                                {device.user_id ? (
                                  <span className="text-sm">
                                    {users.find(u => u.id === device.user_id)?.username || "Unknown User"}
                                  </span>
                                ) : (
                                  <span className="text-muted-foreground">Unclaimed</span>
                                )}
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                {device.device_model?.display_name || (
                                  <span className="text-muted-foreground">Unknown</span>
                                )}
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                <code className="text-xs">{device.mac_address}</code>
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-2">
                                  {device.is_claimed ? (
                                    <Badge variant="default" className="flex items-center gap-1">
                                      <Wifi className="h-3 w-3" />
                                      Claimed
                                    </Badge>
                                  ) : (
                                    <Badge variant="secondary" className="flex items-center gap-1">
                                      <WifiOff className="h-3 w-3" />
                                      Unclaimed
                                    </Badge>
                                  )}
                                  {device.is_active ? (
                                    <Badge variant="outline" className="flex items-center gap-1">
                                      <Eye className="h-3 w-3" />
                                      Active
                                    </Badge>
                                  ) : (
                                    <Badge variant="secondary">Inactive</Badge>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                {device.last_seen ? (
                                  formatDate(device.last_seen)
                                ) : (
                                  <span className="text-muted-foreground">Never</span>
                                )}
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-2">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setViewDevice(device)}
                                  >
                                    <Eye className="h-4 w-4" />
                                  </Button>
                                  {device.is_claimed && (
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => unlinkDevice(device.id)}
                                      disabled={unlinkingDevice === device.id}
                                    >
                                      {unlinkingDevice === device.id ? (
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                      ) : (
                                        <Unlink className="h-4 w-4" />
                                      )}
                                    </Button>
                                  )}
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="plugins" className="mt-6">
                <div className="space-y-4">
                  {/* Plugin Statistics Cards */}
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">System Plugins</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {pluginStats?.total_plugins || 0}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {pluginStats?.active_plugins || 0} active
                        </p>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">User Instances</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {pluginStats?.total_plugin_instances || 0}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {pluginStats?.active_plugin_instances || 0} active
                        </p>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Avg per User</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {systemStatus ? Math.round((pluginStats?.total_plugin_instances || 0) / Math.max(systemStatus.database.total_users, 1) * 10) / 10 : 0}
                        </div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Most Popular</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-sm font-bold">
                          {plugins.find(p => p.name)?.name || "N/A"}
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                  {/* Create Plugin Button */}
                  <div className="flex justify-end">
                    <Button 
                      onClick={() => setShowCreatePluginDialog(true)}
                    >
                      Create Plugin
                    </Button>
                  </div>
                  {/* Plugin List */}
                  <Card>
                    <CardHeader>
                      <CardTitle>System Plugins ({plugins.length})</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead 
                              className="cursor-pointer hover:bg-muted/50 select-none"
                              onClick={() => handlePluginSort('name')}
                            >
                              <div className="flex items-center gap-1">
                                Name
                                {pluginSortState.column === 'name' ? (
                                  pluginSortState.order === 'asc' ? (
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
                              onClick={() => handlePluginSort('type')}
                            >
                              <div className="flex items-center gap-1">
                                Type
                                {pluginSortState.column === 'type' ? (
                                  pluginSortState.order === 'asc' ? (
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
                              className="hidden md:table-cell cursor-pointer hover:bg-muted/50 select-none"
                              onClick={() => handlePluginSort('version')}
                            >
                              <div className="flex items-center gap-1">
                                Version
                                {pluginSortState.column === 'version' ? (
                                  pluginSortState.order === 'asc' ? (
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
                              onClick={() => handlePluginSort('author')}
                            >
                              <div className="flex items-center gap-1">
                                Author
                                {pluginSortState.column === 'author' ? (
                                  pluginSortState.order === 'asc' ? (
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
                              onClick={() => handlePluginSort('status')}
                            >
                              <div className="flex items-center gap-1">
                                Status
                                {pluginSortState.column === 'status' ? (
                                  pluginSortState.order === 'asc' ? (
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
                              onClick={() => handlePluginSort('created')}
                            >
                              <div className="flex items-center gap-1">
                                Created
                                {pluginSortState.column === 'created' ? (
                                  pluginSortState.order === 'asc' ? (
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
                          {sortedPlugins.map((plugin) => (
                            <TableRow key={plugin.id}>
                              <TableCell className="font-medium">{plugin.name}</TableCell>
                              <TableCell>{plugin.type}</TableCell>
                              <TableCell className="hidden md:table-cell">
                                {plugin.version || <span className="text-muted-foreground">N/A</span>}
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                {plugin.author || <span className="text-muted-foreground">Unknown</span>}
                              </TableCell>
                              <TableCell>
                                <Button
                                  size="sm"
                                  variant="ghost"
                                  onClick={() => togglePluginStatus(plugin)}
                                  className="p-0 h-auto"
                                >
                                  {plugin.is_active ? (
                                    <Badge variant="default" className="flex items-center gap-1">
                                      <CheckCircle className="h-3 w-3" />
                                      Active
                                    </Badge>
                                  ) : (
                                    <Badge variant="secondary" className="flex items-center gap-1">
                                      <XCircle className="h-3 w-3" />
                                      Inactive
                                    </Badge>
                                  )}
                                </Button>
                              </TableCell>
                              <TableCell className="hidden lg:table-cell">
                                {formatDate(plugin.created_at)}
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-2">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setViewPlugin(plugin)}
                                  >
                                    <Eye className="h-4 w-4" />
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => openEditPluginDialog(plugin)}
                                  >
                                    <Edit className="h-4 w-4" />
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setDeletePlugin(plugin)}
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
                </div>
              </TabsContent>

              <TabsContent value="firmware" className="mt-6">
                <div className="space-y-4">
                  {/* Firmware Statistics Cards */}
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Firmware Versions</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {firmwareStats?.firmware_versions.total || 0}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {firmwareStats?.firmware_versions.downloaded || 0} downloaded
                        </p>
                      </CardContent>
                    </Card>
                    <Card 
                      className="cursor-pointer hover:bg-accent/50 transition-colors"
                      onClick={() => setShowDeviceModels(true)}
                    >
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Device Models</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {firmwareStats?.device_models.total || 0}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Device types
                        </p>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Update Settings</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {firmwareStats?.update_settings.enabled || 0}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {firmwareStats?.update_settings.disabled || 0} disabled
                        </p>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Latest Version</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">
                          {firmwareVersions.find(v => v.is_latest)?.version || 'N/A'}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Available
                        </p>
                      </CardContent>
                    </Card>
                  </div>
                  {/* Manual Polling Controls */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Manual Polling</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-muted-foreground mb-4">
                        Trigger manual checks for firmware updates and device models
                      </p>
                      <div className="flex gap-2">
                      <Button
                        onClick={triggerFirmwarePoll}
                        disabled={firmwarePolling}
                        variant="outline"
                      >
                        {firmwarePolling ? (
                          <Loader2 className="h-4 w-4 animate-spin mr-2" />
                        ) : (
                          <Download className="h-4 w-4 mr-2" />
                        )}
                        {firmwarePolling ? 'Polling...' : 'Poll Firmware'}
                      </Button>
                      <Button
                        onClick={triggerModelPoll}
                        disabled={modelPolling}
                        variant="outline"
                      >
                        {modelPolling ? (
                          <Loader2 className="h-4 w-4 animate-spin mr-2" />
                        ) : (
                          <Monitor className="h-4 w-4 mr-2" />
                        )}
                        {modelPolling ? 'Polling...' : 'Poll Models'}
                      </Button>
                      </div>
                    </CardContent>
                  </Card>
                  {/* Firmware Versions Table */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Firmware Versions</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-muted-foreground mb-4">
                        Available firmware versions for TRMNL devices
                      </p>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Version</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead className="hidden md:table-cell">Size</TableHead>
                            <TableHead className="hidden lg:table-cell">Notes</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {firmwareVersions.map((version) => (
                            <TableRow key={version.id}>
                              <TableCell className="font-medium">
                                {version.version}
                                {version.is_latest && (
                                  <Badge variant="secondary" className="ml-2">Latest</Badge>
                                )}
                              </TableCell>
                              <TableCell>
                                {version.download_status === 'downloaded' && (
                                  <Badge variant="default" className="flex items-center gap-1 w-fit">
                                    <CheckCircle className="w-3 h-3" />
                                    Downloaded
                                  </Badge>
                                )}
                                {version.download_status === 'downloading' && (
                                  <Badge variant="secondary" className="flex items-center gap-1 w-fit">
                                    <Loader2 className="w-3 h-3 animate-spin" />
                                    Downloading
                                  </Badge>
                                )}
                                {version.download_status === 'failed' && (
                                  <Badge variant="destructive" className="flex items-center gap-1 w-fit">
                                    <XCircle className="w-3 h-3" />
                                    Failed
                                  </Badge>
                                )}
                                {(!version.download_status || version.download_status === 'pending') && (
                                  firmwareMode === 'proxy' ? (
                                    <Badge variant="secondary" className="flex items-center gap-1 w-fit">
                                      <ArrowRightLeft className="w-3 h-3" />
                                      Proxying
                                    </Badge>
                                  ) : (
                                    <Badge variant="outline">Pending</Badge>
                                  )
                                )}
                              </TableCell>
                              <TableCell className="hidden md:table-cell">
                                {version.file_size ? `${(version.file_size / 1024 / 1024).toFixed(1)} MB` : 'Unknown'}
                              </TableCell>
                              <TableCell className="hidden lg:table-cell max-w-xs truncate">
                                {version.release_notes || 'No release notes available'}
                              </TableCell>
                              <TableCell className="text-right">
                                <div className="flex items-center justify-end gap-2">
                                  {(version.download_status === 'failed' || version.download_status === 'pending') && firmwareMode !== 'proxy' && (
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      onClick={() => retryFirmwareDownload(version.id, version.version)}
                                      title={version.download_status === 'failed' ? 'Retry download' : 'Download now'}
                                    >
                                      <Download className="w-4 h-4" />
                                    </Button>
                                  )}
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => openDeleteFirmwareDialog(version)}
                                  >
                                    <Trash2 className="w-4 h-4" />
                                  </Button>
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                      {firmwareVersions.length === 0 && (
                        <div className="text-center py-8 text-muted-foreground">
                          No firmware versions available. Try polling for updates.
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="settings" className="mt-6">
                <div className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Site Configuration</CardTitle>
                      <p className="text-sm text-muted-foreground">Configure the public URL for your instance</p>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <Label htmlFor="site-url">Site URL</Label>
                        <Input
                          id="site-url"
                          placeholder="https://stationmaster.example.com"
                          value={siteUrl}
                          onChange={(e) => setSiteUrl(e.target.value)}
                          onBlur={() => {
                            const trimmedUrl = siteUrl.trim();
                            if (trimmedUrl && !trimmedUrl.match(/^https?:\/\//)) {
                              setError("Site URL must start with http:// or https://");
                              return;
                            }
                            updateSystemSetting("site_url", trimmedUrl);
                          }}
                          className="max-w-md"
                        />
                        <p className="text-sm text-muted-foreground">
                          Full URL including protocol (http/https) used for firmware downloads and other external links.
                          Leave empty to use relative URLs.
                        </p>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>User Management</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="flex items-center justify-between">
                        <div className="space-y-1">
                          <Label htmlFor="registration-enabled">
                            Enable Registration
                          </Label>
                          <p className="text-sm text-muted-foreground">
                            {registrationLocked 
                              ? "Set via environment variable" 
                              : "Allow new users to register accounts"}
                          </p>
                        </div>
                        <div className="relative">
                          <Switch
                            id="registration-enabled"
                            checked={registrationEnabled}
                            disabled={registrationLocked}
                            onCheckedChange={(checked) => {
                              if (!registrationLocked) {
                                setRegistrationEnabled(checked);
                                updateSystemSetting(
                                  "registration_enabled",
                                  checked.toString(),
                                );
                              }
                            }}
                          />
                          {registrationLocked && (
                            <div className="absolute inset-0 cursor-not-allowed" title="Set via environment variable" />
                          )}
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>Webhook Settings</CardTitle>
                      <p className="text-sm text-muted-foreground">Configure rate limiting and request size limits for private plugin webhooks</p>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label htmlFor="webhook-rate-limit">Rate Limit (per hour)</Label>
                          <Input
                            id="webhook-rate-limit"
                            type="number"
                            min="1"
                            max="1000"
                            value={webhookRateLimit}
                            onChange={(e) => setWebhookRateLimit(parseInt(e.target.value) || 30)}
                            onBlur={() => {
                              if (webhookRateLimit < 1 || webhookRateLimit > 1000) {
                                setError("Rate limit must be between 1 and 1000");
                                setWebhookRateLimit(30);
                                return;
                              }
                              updateSystemSetting("webhook_rate_limit_per_hour", webhookRateLimit.toString());
                            }}
                            className="max-w-xs"
                          />
                          <p className="text-sm text-muted-foreground">
                            Maximum webhook requests per user per hour
                          </p>
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="webhook-max-size">Max Request Size (KB)</Label>
                          <Input
                            id="webhook-max-size"
                            type="number"
                            min="1"
                            max="100"
                            value={webhookMaxSizeKB}
                            onChange={(e) => setWebhookMaxSizeKB(parseInt(e.target.value) || 5)}
                            onBlur={() => {
                              if (webhookMaxSizeKB < 1 || webhookMaxSizeKB > 100) {
                                setError("Max request size must be between 1 and 100 KB");
                                setWebhookMaxSizeKB(5);
                                return;
                              }
                              updateSystemSetting("webhook_max_request_size_kb", webhookMaxSizeKB.toString());
                            }}
                            className="max-w-xs"
                          />
                          <p className="text-sm text-muted-foreground">
                            Maximum webhook payload size in kilobytes
                          </p>
                        </div>
                      </div>
                      <div className="text-sm text-muted-foreground mt-4">
                        <p>
                          <strong>Rate Limiting:</strong> Each user can send webhook requests at the specified rate per hour across all their private plugins.
                        </p>
                        <p className="mt-1">
                          <strong>Request Size:</strong> Individual webhook payloads cannot exceed this size limit.
                        </p>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="system" className="mt-6">
                <div className="space-y-4">
                  {backupError && (
                    <Card className="border-destructive">
                      <CardContent className="pt-6">
                        <div className="flex items-center gap-2 text-destructive">
                          <AlertTriangle className="h-4 w-4" />
                          <span>{backupError}</span>
                        </div>
                      </CardContent>
                    </Card>
                  )}
                  <Card>
                    <CardHeader>
                      <CardTitle>{t("admin.cards.backup_restore")}</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <Button
                          variant="outline"
                          size="lg"
                          onClick={handleCreateBackupJob}
                          disabled={creatingBackup}
                          className="w-full"
                        >
                          {creatingBackup ? t("admin.backup.creating_job") : t("admin.backup.create_backup")}
                        </Button>
                        
                        {restoreUploads.length === 0 ? (
                          <div className="w-full">
                            <input
                              type="file"
                              ref={fileInputRef}
                              onChange={handleRestoreFileSelect}
                              accept=".tar.gz,.tgz,application/gzip,application/x-gzip,application/x-tar,application/x-compressed-tar"
                              style={{ display: "none" }}
                            />
                            <Button
                              variant="outline"
                              size="lg"
                              onClick={() => fileInputRef.current?.click()}
                              disabled={analyzingBackup || uploadPhase !== 'idle'}
                              className="w-full"
                            >
                              {uploadPhase === 'uploading' ? t("admin.loading_states.uploading") : 
                               uploadPhase === 'extracting' ? t("admin.loading_states.extracting") :
                               uploadPhase === 'validating' ? t("admin.loading_states.validating") :
                               t("admin.actions.upload_restore")}
                            </Button>
                            {uploadPhase === 'uploading' && (
                              <Progress
                                value={uploadProgress}
                                durationMs={200}
                                className="mt-2"
                              />
                            )}
                          </div>
                        ) : (
                          <Button
                            variant="default"
                            size="lg"
                            onClick={() => handleRestoreUpload(restoreUploads[0])}
                            disabled={restoringBackup}
                            className="w-full"
                          >
                            {restoringBackup ? t("admin.loading_states.restoring") : t("admin.actions.restore_backup")}
                          </Button>
                        )}
                      </div>
                      
                      <div className="mt-6">
                        <h4 className="text-sm font-medium mb-3">Recent Backup Jobs</h4>
                        <div className="space-y-2">
                          {backupJobs.length > 0 ? (
                            backupJobs
                              .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
                              .slice(0, 8)
                              .map((job) => (
                                <div key={job.id} className="flex items-center justify-between p-3 border rounded-lg">
                                  <div className="text-sm flex-1 min-w-0">
                                    <div className="font-medium">
                                      {formatDate(job.created_at)}
                                    </div>
                                    <div className="text-muted-foreground">
                                      {job.status === "failed" ? "Failed" :
                                       job.file_size ? formatFileSize(job.file_size) : 
                                       (job.status === "running" || job.status === "pending") ? 
                                       (job.status === "pending" ? "Pending" : "Running") : 
                                       null}
                                    </div>
                                    {job.error_message && (
                                      <div className="text-xs text-destructive mt-1 break-words">
                                        {job.error_message}
                                      </div>
                                    )}
                                  </div>
                                  {getBackupStatusButton(job)}
                                </div>
                              ))
                          ) : (
                            <div className="p-4 border border-dashed rounded-lg text-center text-muted-foreground">
                              <Database className="h-8 w-8 mx-auto mb-2 opacity-50" />
                              <p className="text-sm">No backup jobs found</p>
                              <p className="text-xs mt-1">Create a backup job to get started</p>
                            </div>
                          )}
                        </div>
                      </div>
                      <div className="text-sm text-muted-foreground space-y-2">
                        <p>
                          <strong>{t("admin.backup.create_backup")}:</strong> {t("admin.backup.background_description")}
                        </p>
                        <p>
                          <strong>{t("admin.actions.upload_restore")}:</strong> {t("admin.descriptions.upload_restore_description")}
                        </p>
                        <div className="flex items-center gap-1 text-muted-foreground">
                          <AlertTriangle className="h-4 w-4" />
                          <span>{t("admin.descriptions.restore_warning")}</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                  {versionInfo && (
                    <div className="flex justify-center mt-8 pt-4 border-t">
                      <div className="text-center text-sm text-muted-foreground">
                        <span>Stationmaster {versionInfo.version} • </span>
                        <a 
                          href="https://github.com/rmitchellscott/stationmaster" 
                          target="_blank" 
                          rel="noopener noreferrer"
                          className="text-muted-foreground hover:underline"
                        >
                          GitHub
                        </a>
                      </div>
                    </div>
                  )}
                </div>
              </TabsContent>
            </Tabs>
      </div>

      {/* Password Reset Dialog */}
      <Dialog
        open={resetPasswordDialog.isOpen}
        onOpenChange={closeResetPasswordDialog}
      >
        <DialogContent className="sm:max-w-md max-h-[60vh] overflow-y-auto max-sm:top-[20vh] md:top-[33%]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Key className="h-5 w-5" />
              {t("admin.dialogs.reset_password_title")}
            </DialogTitle>
            <DialogDescription>
              {t("admin.dialogs.reset_password_description")}
              <strong>{resetPasswordDialog.user?.username}</strong>
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="new-password">{t("admin.labels.new_password")}</Label>
              <Input
                id="new-password"
                type="password"
                value={newPasswordValue}
                onChange={(e) => setNewPasswordValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newPasswordValue.length >= 8 && !resettingPassword) {
                    confirmResetPassword();
                  }
                }}
                placeholder={t("admin.placeholders.new_password")}
                className="mt-2"
                minLength={8}
              />
              <p className="text-xs text-muted-foreground mt-1">
                {t("admin.descriptions.password_requirements")}
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeResetPasswordDialog}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={confirmResetPassword}
              disabled={resettingPassword || newPasswordValue.length < 8}
            >
              {resettingPassword ? t("admin.loading_states.resetting") : t("admin.actions.reset_password")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete User Dialog */}
      <UserDeleteDialog
        isOpen={deleteUserDialog.isOpen}
        onClose={closeDeleteUserDialog}
        onConfirm={confirmDeleteUser}
        user={deleteUserDialog.user}
        isCurrentUser={false}
        loading={deleting}
      />

      {/* User Details Dialog (Mobile) */}
      <Dialog open={!!viewUser} onOpenChange={() => setViewUser(null)}>
        <DialogContent className="sm:max-w-md max-h-[60vh] overflow-y-auto max-sm:top-[20vh] md:top-[33%]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Users className="h-5 w-5" />
              {t('admin.dialogs.user_details')}
            </DialogTitle>
          </DialogHeader>
          {viewUser && (
            <div className="space-y-3">
              <div className="text-sm space-y-2">
                <p>
                  <strong>{t('admin.labels.username')}:</strong> {viewUser.username}
                </p>
                <p>
                  <strong>{t('admin.labels.email')}:</strong> {viewUser.email}
                </p>
                <p>
                  <strong>{t('admin.labels.role')}:</strong>{' '}
                  {viewUser.is_admin ? t('admin.roles.admin') : t('admin.roles.user')}
                </p>
                <p>
                  <strong>{t('admin.labels.status')}:</strong>{' '}
                  {viewUser.is_active
                    ? viewUser.rmapi_paired
                      ? t('admin.status.paired')
                      : t('admin.status.active')
                    : t('admin.status.unpaired')}
                </p>
                <p>
                  <strong>{t('admin.labels.created')}:</strong> {formatDate(viewUser.created_at)}
                </p>
                <p>
                  <strong>{t('admin.labels.last_login')}:</strong>{' '}
                  {viewUser.last_login ? formatDate(viewUser.last_login) : t('admin.never')}
                </p>
              </div>
              <DialogFooter className="lg:hidden flex-col sm:flex-row gap-2">
                <Popover>
                  <PopoverTrigger asChild>
                    <Button
                      size="sm"
                      variant="outline"
                      className="w-full sm:w-auto sm:flex-1"
                    >
                      {t("admin.actions.modify")}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-auto p-2" align="end">
                    <div className="flex flex-col gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          openResetPasswordDialog(viewUser);
                          setViewUser(null);
                        }}
                        className="w-full justify-start"
                      >
                        {t("admin.actions.reset_password")}
                      </Button>
                      {!isCurrentUser(viewUser) && (
                        <>
                          {!config?.oidcGroupBasedAdmin && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                toggleAdminStatus(viewUser.id, !viewUser.is_admin);
                              }}
                              className="w-full justify-start"
                            >
                              {viewUser.is_admin ? t("admin.actions.make_user") : t("admin.actions.make_admin")}
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              toggleUserStatus(viewUser.id, !viewUser.is_active);
                            }}
                            className="w-full justify-start"
                          >
                            {viewUser.is_active ? t("admin.actions.deactivate") : t("admin.actions.activate")}
                          </Button>
                        </>
                      )}
                    </div>
                  </PopoverContent>
                </Popover>
                {!isCurrentUser(viewUser) && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setDeleteFromDetails(true);
                      openDeleteUserDialog(viewUser);
                      setViewUser(null);
                    }}
                    className="w-full sm:w-auto sm:flex-1"
                  >
                    {t("admin.actions.delete")}
                  </Button>
                )}
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Device Details Dialog */}
      <Dialog open={!!viewDevice} onOpenChange={() => setViewDevice(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Device Details</DialogTitle>
          </DialogHeader>
          {viewDevice && (
            <div className="space-y-2 text-sm">
              <p>
                <strong>Friendly ID:</strong> <code>{viewDevice.friendly_id}</code>
              </p>
              <p>
                <strong>Name:</strong> {viewDevice.name || "Unnamed"}
              </p>
              <p>
                <strong>MAC Address:</strong> <code>{viewDevice.mac_address}</code>
              </p>
              <p>
                <strong>Model:</strong> {viewDevice.device_model?.display_name || "Unknown"}
              </p>
              <p>
                <strong>Owner:</strong>{" "}
                {viewDevice.user_id ? 
                  users.find(u => u.id === viewDevice.user_id)?.username || "Unknown User" : 
                  "Unclaimed"
                }
              </p>
              <p>
                <strong>Status:</strong>{" "}
                <span className={viewDevice.is_claimed ? "text-emerald-600" : "text-amber-600"}>
                  {viewDevice.is_claimed ? "Claimed" : "Unclaimed"}
                </span>
              </p>
              <p>
                <strong>Active:</strong> {viewDevice.is_active ? "Yes" : "No"}
              </p>
              <p>
                <strong>Firmware:</strong> {viewDevice.firmware_version || "Unknown"}
              </p>
              <p className="flex items-center gap-2">
                <strong>Battery:</strong>
                {(() => {
                  const battery = getBatteryDisplay(viewDevice.battery_voltage);
                  return (
                    <span className="flex items-center gap-1">
                      {battery.icon}
                      <span className={battery.color}>{battery.text}</span>
                    </span>
                  );
                })()}
              </p>
              <p className="flex items-center gap-2">
                <strong>Signal:</strong>
                {(() => {
                  const signal = getSignalDisplay(viewDevice.rssi);
                  return (
                    <span className="flex items-center gap-1">
                      {signal.icon}
                      <span className={signal.color}>{signal.text}</span>
                    </span>
                  );
                })()}
              </p>
              <p>
                <strong>Refresh Rate:</strong> {viewDevice.refresh_rate}s
              </p>
              <p>
                <strong>Last Seen:</strong>{" "}
                {viewDevice.last_seen ? formatDate(viewDevice.last_seen) : "Never"}
              </p>
              <p>
                <strong>Created:</strong> {formatDate(viewDevice.created_at)}
              </p>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Device Models Dialog */}
      <Dialog open={showDeviceModels} onOpenChange={setShowDeviceModels}>
        <DialogContent className="sm:max-w-4xl max-h-[85vh] overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
          <DialogHeader>
            <DialogTitle>Device Models</DialogTitle>
            <DialogDescription>
              View all device models synced from the TRMNL API
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-[60vh] overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Model Name</TableHead>
                  <TableHead>Display Name</TableHead>
                  <TableHead>Screen Size</TableHead>
                  <TableHead>Color Depth</TableHead>
                  <TableHead>Features</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deviceModels.map((model) => (
                  <TableRow key={model.id}>
                    <TableCell>
                      <code className="text-xs bg-muted px-2 py-1 rounded">
                        {model.model_name}
                      </code>
                    </TableCell>
                    <TableCell className="font-medium">
                      {model.display_name}
                    </TableCell>
                    <TableCell>
                      {model.screen_width} × {model.screen_height}
                    </TableCell>
                    <TableCell>
                      {model.bit_depth === 1 ? "Monochrome" : 
                       model.bit_depth === 8 ? "Grayscale" : 
                       `${model.bit_depth}-bit Color`}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {model.has_wifi && (
                          <Badge variant="secondary" className="text-xs">WiFi</Badge>
                        )}
                        {model.has_battery && (
                          <Badge variant="secondary" className="text-xs">Battery</Badge>
                        )}
                        {model.has_buttons > 0 && (
                          <Badge variant="secondary" className="text-xs">
                            {model.has_buttons} Button{model.has_buttons > 1 ? 's' : ''}
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {deviceModels.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground">
                      No device models found. Try triggering a manual model poll.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>

      {/* Plugin Details Dialog */}
      <Dialog open={!!viewPlugin} onOpenChange={() => setViewPlugin(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Plugin Details</DialogTitle>
          </DialogHeader>
          {viewPlugin && (
            <div className="space-y-2 text-sm">
              <p>
                <strong>Name:</strong> {viewPlugin.name}
              </p>
              <p>
                <strong>Description:</strong> {viewPlugin.description || "No description"}
              </p>
              <p>
                <strong>Version:</strong> {viewPlugin.version || "N/A"}
              </p>
              <p>
                <strong>Author:</strong> {viewPlugin.author || "Unknown"}
              </p>
              <p>
                <strong>Status:</strong>{" "}
                <span className={viewPlugin.is_active ? "text-emerald-600" : "text-amber-600"}>
                  {viewPlugin.is_active ? "Active" : "Inactive"}
                </span>
              </p>
              <p>
                <strong>Created:</strong> {formatDate(viewPlugin.created_at)}
              </p>
              <p>
                <strong>Updated:</strong> {formatDate(viewPlugin.updated_at)}
              </p>
              {viewPlugin.config_schema && (
                <div>
                  <strong>Config Schema:</strong>
                  <pre className="mt-2 p-2 bg-muted rounded text-xs overflow-auto">
                    {viewPlugin.config_schema}
                  </pre>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Create Plugin Dialog */}
      <Dialog open={showCreatePluginDialog} onOpenChange={setShowCreatePluginDialog}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Create New Plugin</DialogTitle>
            <DialogDescription>
              Add a new plugin to the system. This will be available for users to create instances of.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="plugin-name">Name</Label>
              <Input
                id="plugin-name"
                value={pluginName}
                onChange={(e) => setPluginName(e.target.value)}
                placeholder="e.g., Weather Display"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="plugin-description">Description</Label>
              <Input
                id="plugin-description"
                value={pluginDescription}
                onChange={(e) => setPluginDescription(e.target.value)}
                placeholder="Brief description of the plugin"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="plugin-version">Version</Label>
              <Input
                id="plugin-version"
                value={pluginVersion}
                onChange={(e) => setPluginVersion(e.target.value)}
                placeholder="e.g., 1.0.0"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="plugin-author">Author</Label>
              <Input
                id="plugin-author"
                value={pluginAuthor}
                onChange={(e) => setPluginAuthor(e.target.value)}
                placeholder="Plugin author name"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="plugin-config-schema">Configuration Schema (JSON)</Label>
              <textarea
                id="plugin-config-schema"
                value={pluginConfigSchema}
                onChange={(e) => setPluginConfigSchema(e.target.value)}
                placeholder='{"api_key": {"type": "string", "required": true}}'
                className="mt-2 w-full p-2 border rounded text-sm font-mono"
                rows={4}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreatePluginDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={createPlugin}
              disabled={creatingPlugin || !pluginName.trim()}
            >
              {creatingPlugin ? "Creating..." : "Create Plugin"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Plugin Dialog */}
      <Dialog open={!!editPlugin} onOpenChange={() => setEditPlugin(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Plugin</DialogTitle>
            <DialogDescription>
              Update plugin information and configuration.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="edit-plugin-name">Name</Label>
              <Input
                id="edit-plugin-name"
                value={pluginName}
                onChange={(e) => setPluginName(e.target.value)}
                placeholder="e.g., Weather Display"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="edit-plugin-description">Description</Label>
              <Input
                id="edit-plugin-description"
                value={pluginDescription}
                onChange={(e) => setPluginDescription(e.target.value)}
                placeholder="Brief description of the plugin"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="edit-plugin-version">Version</Label>
              <Input
                id="edit-plugin-version"
                value={pluginVersion}
                onChange={(e) => setPluginVersion(e.target.value)}
                placeholder="e.g., 1.0.0"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="edit-plugin-author">Author</Label>
              <Input
                id="edit-plugin-author"
                value={pluginAuthor}
                onChange={(e) => setPluginAuthor(e.target.value)}
                placeholder="Plugin author name"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="edit-plugin-config-schema">Configuration Schema (JSON)</Label>
              <textarea
                id="edit-plugin-config-schema"
                value={pluginConfigSchema}
                onChange={(e) => setPluginConfigSchema(e.target.value)}
                placeholder='{"api_key": {"type": "string", "required": true}}'
                className="mt-2 w-full p-2 border rounded text-sm font-mono"
                rows={4}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditPlugin(null)}>
              Cancel
            </Button>
            <Button
              onClick={updatePlugin}
              disabled={!pluginName.trim() || !hasPluginChanges()}
            >
              Update Plugin
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Plugin Confirmation Dialog */}
      <AlertDialog
        open={!!deletePlugin}
        onOpenChange={() => setDeletePlugin(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Plugin
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deletePlugin?.name}"? This will also deactivate all user instances of this plugin. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deletingPlugin}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDeletePlugin}
              disabled={deletingPlugin}
              variant="destructive"
            >
              {deletingPlugin ? "Deleting..." : "Delete Plugin"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Firmware Delete Dialog */}
      <AlertDialog 
        open={deleteFirmwareDialog.isOpen}
        onOpenChange={closeDeleteFirmwareDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Firmware Version
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete firmware version "{deleteFirmwareDialog.version?.version}"? This will also remove the downloaded file. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deletingFirmware}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDeleteFirmware}
              disabled={deletingFirmware}
              variant="destructive"
            >
              {deletingFirmware ? "Deleting..." : "Delete Firmware"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Database Restore Confirmation Dialog */}
      <AlertDialog
        open={restoreConfirmDialog.isOpen}
        onOpenChange={closeRestoreConfirmDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-muted-foreground" />
              {t("admin.dialogs.confirm_restore_title")}
            </AlertDialogTitle>
            <AlertDialogDescription className="space-y-3">
              <p>
                {t("admin.dialogs.confirm_restore_description")}
                <strong>{restoreConfirmDialog.upload?.filename}</strong>
              </p>
              {backupCounts && (
                <div className="bg-muted p-3 rounded-md">
                  <div className="grid grid-cols-3 gap-4 text-sm mb-4">
                    <div className="text-center">
                      <div className="font-semibold text-lg">{backupCounts.users}</div>
                      <div className="text-muted-foreground">{t("admin.labels.users")}</div>
                    </div>
                    <div className="text-center">
                      <div className="font-semibold text-lg">{backupCounts.api_keys}</div>
                      <div className="text-muted-foreground">{t("admin.labels.api_keys")}</div>
                    </div>
                    <div className="text-center">
                      <div className="font-semibold text-lg">{backupCounts.documents}</div>
                      <div className="text-muted-foreground">{t("admin.labels.documents")}</div>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div className="text-center">
                      <div className="font-semibold text-lg">
                        {backupVersion ? backupVersion.replace(/^v/, '') : t("admin.dialogs.version_unknown")}
                      </div>
                      <div className="text-muted-foreground">{t("admin.dialogs.backup_version")}</div>
                    </div>
                    <div className="text-center">
                      <div className="font-semibold text-lg">
                        {versionInfo?.version ? versionInfo.version.replace(/^v/, '') : t("admin.dialogs.version_unknown")}
                      </div>
                      <div className="text-muted-foreground">{t("admin.dialogs.current_version")}</div>
                    </div>
                  </div>
                </div>
              )}
              
              {(() => {
                const comparison = compareSemver(backupVersion, versionInfo?.version);
                if (comparison === 1) {
                  return (
                    <div className="bg-destructive/10 border border-destructive/20 p-3 rounded-md">
                      <div className="flex items-start gap-2">
                        <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
                        <div className="text-sm text-destructive">
                          <strong>Warning:</strong> This backup is from a newer version of Stationmaster ({backupVersion}) than your current version ({versionInfo?.version}). Restoring may cause compatibility issues.
                        </div>
                      </div>
                    </div>
                  );
                } else if (comparison === -1) {
                  return (
                    <div className="bg-blue-50 border border-blue-200 p-3 rounded-md">
                      <div className="text-sm text-blue-800">
                        <strong>Note:</strong> This backup is from an older version of Stationmaster ({backupVersion}). The restore process will automatically migrate the data to your current version ({versionInfo?.version}).
                      </div>
                    </div>
                  );
                }
                return null;
              })()}
              
              <p className="text-destructive font-medium">
                {t("admin.dialogs.restore_warning_text")}
              </p>
              <ul className="list-disc list-outside text-sm space-y-1 ml-8 sm:ml-9">
                <li>{t("admin.dialogs.restore_warning_items.accounts")}</li>
                <li>{t("admin.dialogs.restore_warning_items.api_keys")}</li>
                <li>{t("admin.dialogs.restore_warning_items.documents")}</li>
                <li>{t("admin.dialogs.restore_warning_items.pairing")}</li>
                <li>{t("admin.dialogs.restore_warning_items.folders")}</li>
                <li>{t("admin.dialogs.restore_warning_items.settings")}</li>
                <li>{t("admin.dialogs.restore_warning_items.backups")}</li>
              </ul>
              <p className="text-destructive font-medium">
                {t("admin.dialogs.restore_final_warning")}
              </p>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel 
              disabled={restoringBackup}
              onClick={cancelRestoreUpload}
            >
              {t("admin.actions.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDatabaseRestore}
              disabled={restoringBackup}
              variant="destructive"
            >
              {restoringBackup ? t("admin.loading_states.restoring") : t("admin.actions.restore_database")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Backup Job Dialog */}
      <AlertDialog
        open={deleteBackupDialog.isOpen}
        onOpenChange={closeDeleteBackupDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-muted-foreground" />
              {t("admin.backup.delete_backup_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("admin.backup.delete_backup_description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>
              {t("admin.actions.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDeleteBackupJob}
              disabled={deleting}
              variant="destructive"
            >
              {deleting ? t("admin.loading_states.deleting") : t("admin.actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}