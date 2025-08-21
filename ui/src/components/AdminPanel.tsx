import React, { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useConfig } from "@/components/ConfigProvider";
import { useAuth } from "@/components/AuthProvider";
import { UserDeleteDialog } from "@/components/UserDeleteDialog";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
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
import { Progress } from "@/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import {
  Shield,
  Users,
  Key,
  Settings as SettingsIcon,
  Database,

  Edit,
  CheckCircle,
  XCircle,
  Clock,
  Activity,
  Mail,
  Server,
  AlertTriangle,
  Loader2,
  Download,
  Trash2,
  Monitor,
  Puzzle,
  Wifi,
  WifiOff,
  Eye,
  EyeOff,
  Unlink,
  Calendar as CalendarIcon,
  Battery,
  BatteryFull,
  BatteryMedium,
  BatteryLow,
  BatteryWarning,
  ArrowRightLeft,
} from "lucide-react";

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
  
  const parts1 = clean1.split('.').map(p => parseInt(p, 10));
  const parts2 = clean2.split('.').map(p => parseInt(p, 10));
  
  if (parts1.length < 3 || parts2.length < 3 || 
      parts1.some(isNaN) || parts2.some(isNaN)) {
    return null;
  }
  
  for (let i = 0; i < 3; i++) {
    if (parts1[i] > parts2[i]) return 1;
    if (parts1[i] < parts2[i]) return -1;
  }
  
  return 0;
}

interface User {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
  is_active: boolean;
  rmapi_host?: string;
  rmapi_paired: boolean;
  default_rmdir: string;
  created_at: string;
  last_login?: string;
}

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  is_active: boolean;
  last_used?: string;
  expires_at?: string;
  created_at: string;
  user_id: string;
  username: string;
}

interface BackupJob {
  id: string;
  status: string;
  progress: number;
  include_files: boolean;
  include_configs: boolean;
  file_path?: string;
  filename?: string;
  file_size?: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  expires_at?: string;
  created_at: string;
}

interface RestoreUpload {
  id: string;
  filename: string;
  file_size: number;
  status: string;
  expires_at: string;
  created_at: string;
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
  model_name?: string;
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
  user?: User;
  device_model?: DeviceModel;
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

interface DeviceStats {
  total_devices: number;
  claimed_devices: number;
  unclaimed_devices: number;
  active_devices: number;
  inactive_devices: number;
  recent_registrations: number;
}

interface PluginStats {
  total_plugins: number;
  active_plugins: number;
  total_user_plugins: number;
  active_user_plugins: number;
}

interface FirmwareVersion {
  id: string;
  version: string;
  release_notes: string;
  download_url: string;
  file_size: number;
  file_path: string;
  sha256: string;
  is_latest: boolean;
  is_downloaded: boolean;
  download_status?: string;
  download_progress?: number;
  download_error?: string;
  released_at: string;
  created_at: string;
  updated_at: string;
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
  };
  mode: string;
  dry_run: boolean;
}

interface AdminPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AdminPanel({ isOpen, onClose }: AdminPanelProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { user: currentUser } = useAuth();
  const { config } = useConfig();
  const { logout } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [backupJobs, setBackupJobs] = useState<BackupJob[]>([]);
  const [restoreUploads, setRestoreUploads] = useState<RestoreUpload[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [deviceStats, setDeviceStats] = useState<DeviceStats | null>(null);
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [pluginStats, setPluginStats] = useState<PluginStats | null>(null);
  const [firmwareVersions, setFirmwareVersions] = useState<FirmwareVersion[]>([]);
  const [firmwareStats, setFirmwareStats] = useState<FirmwareStats | null>(null);
  const [firmwareMode, setFirmwareMode] = useState<string>('proxy');
  const [deviceModels, setDeviceModels] = useState<DeviceModel[]>([]);
  const [firmwarePolling, setFirmwarePolling] = useState(false);
  const [modelPolling, setModelPolling] = useState(false);
  const [loading, setLoading] = useState(false);
  const [creatingBackup, setCreatingBackup] = useState(false);
  const [restoringBackup, setRestoringBackup] = useState(false);
  const [testingSMTP, setTestingSMTP] = useState(false);
  const [creatingUser, setCreatingUser] = useState(false);
  const [resettingPassword, setResettingPassword] = useState(false);
  const [analyzingBackup, setAnalyzingBackup] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  // SMTP test state
  const [smtpTestResult, setSmtpTestResult] = useState<'working' | 'failed' | null>(null);
  // User creation form
  const [newUsername, setNewUsername] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  
  // Detect browser timezone for new users
  const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

  const [registrationEnabled, setRegistrationEnabled] = useState(false);
  const [registrationLocked, setRegistrationLocked] = useState(false);
  const [maxApiKeys, setMaxApiKeys] = useState("10");
  const [maxApiKeysError, setMaxApiKeysError] = useState<string | null>(null);
  const [siteUrl, setSiteUrl] = useState("");

  const [resetPasswordDialog, setResetPasswordDialog] = useState<{
    isOpen: boolean;
    user: User | null;
  }>({ isOpen: false, user: null });
  const [deleteUserDialog, setDeleteUserDialog] = useState<{
    isOpen: boolean;
    user: User | null;
  }>({ isOpen: false, user: null });
  const [newPasswordValue, setNewPasswordValue] = useState("");
  const [deleting, setDeleting] = useState(false);

  const [viewUser, setViewUser] = useState<User | null>(null);
  const [viewKey, setViewKey] = useState<APIKey | null>(null);
  const [deleteFromDetails, setDeleteFromDetails] = useState(false);

  const [deleteBackupDialog, setDeleteBackupDialog] = useState<{
    isOpen: boolean;
    job: BackupJob | null;
  }>({ isOpen: false, job: null });

  // Database backup/restore
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

  // Device management state
  const [viewDevice, setViewDevice] = useState<Device | null>(null);
  const [unlinkingDevice, setUnlinkingDevice] = useState<string | null>(null);
  
  // Plugin management state
  const [viewPlugin, setViewPlugin] = useState<Plugin | null>(null);
  const [showCreatePluginDialog, setShowCreatePluginDialog] = useState(false);
  const [editPlugin, setEditPlugin] = useState<Plugin | null>(null);
  const [deletePlugin, setDeletePlugin] = useState<Plugin | null>(null);
  const [creatingPlugin, setCreatingPlugin] = useState(false);
  const [deletingPlugin, setDeletingPlugin] = useState(false);
  
  // Firmware delete state
  const [deleteFirmwareDialog, setDeleteFirmwareDialog] = useState<{
    isOpen: boolean;
    version: any | null;
  }>({ isOpen: false, version: null });
  const [deletingFirmware, setDeletingFirmware] = useState(false);
  
  // Device models state
  const [showDeviceModels, setShowDeviceModels] = useState(false);
  
  // Plugin form state
  const [pluginName, setPluginName] = useState("");
  const [pluginType, setPluginType] = useState("");
  const [pluginDescription, setPluginDescription] = useState("");
  const [pluginConfigSchema, setPluginConfigSchema] = useState("");
  const [pluginVersion, setPluginVersion] = useState("");
  const [pluginAuthor, setPluginAuthor] = useState("");

  useEffect(() => {
    if (isOpen) {
      setRestorePerformed(false);
      fetchSystemStatus();
      fetchUsers();
      // fetchAPIKeys();
      fetchBackupJobs();
      fetchRestoreUploads();
      fetchVersionInfo();
      fetchDevices();
      fetchDeviceStats();
      fetchPlugins();
      fetchPluginStats();
      fetchFirmwareVersions();
      fetchFirmwareStats();
      fetchFirmwareMode();
      fetchDeviceModels();
    }
  }, [isOpen]);

  useEffect(() => {
    let interval: NodeJS.Timeout;
    const hasActiveJobs = 
      backupJobs.some(job => job.status === 'running' || job.status === 'pending');
      
    if (isOpen && hasActiveJobs) {
      interval = setInterval(() => {
        fetchBackupJobs();
      }, 2000);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [isOpen, backupJobs]);

  // Listen for logout event to clear sensitive admin state
  useEffect(() => {
    const handleLogout = () => {
      // Clear all sensitive admin state
      setUsers([]);
      setApiKeys([]);
      setSystemStatus(null);
      setBackupJobs([]);
      setRestoreUploads([]);
      setError(null);
      setSuccessMessage(null);
      setSmtpTestResult(null);
      
      // Clear form state
      setNewUsername("");
      setNewEmail("");
      setNewPassword("");
      setRegistrationEnabled(false);
      setRegistrationLocked(false);
      setMaxApiKeys("10");
      setMaxApiKeysError(null);
      setNewPasswordValue("");
      setDeleting(false);
      setBackupCounts({ users: 0, api_keys: 0, documents: 0 });
      setBackupVersion(null);
      setRestorePerformed(false);
      setVersionInfo({ version: "", gitCommit: "", buildDate: "", goVersion: "" });
      setUploadProgress(0);
      setUploadPhase('idle');
      setDownloadingJobId(null);
      
      // Close any open dialogs
      setResetPasswordDialog({ isOpen: false, user: null });
      setDeleteUserDialog({ isOpen: false, user: null });
      setDeleteBackupDialog({ isOpen: false, job: null });
      setRestoreConfirmDialog({ isOpen: false, upload: null });
      setViewUser(null);
      setViewKey(null);
      
      // Reset loading states
      setLoading(false);
      setCreatingBackup(false);
      setRestoringBackup(false);
      setTestingSMTP(false);
      setCreatingUser(false);
      setResettingPassword(false);
      setAnalyzingBackup(false);
    };

    window.addEventListener('logout', handleLogout);

    return () => {
      window.removeEventListener('logout', handleLogout);
    };
  }, []);

  const fetchWithSessionCheck = async (url: string, options?: RequestInit) => {
    const response = await fetch(url, options);
    if (response.status === 401) {
      logout();
      navigate('/');
      throw new Error('Session expired after restore. Please log in again.');
    }
    return response;
  };

  const fetchSystemStatus = async () => {
    try {
      const response = await fetch("/api/admin/status", {
        credentials: "include",
      });

      if (response.ok) {
        const status = await response.json();
        setSystemStatus(status);
        setRegistrationEnabled(status.settings.registration_enabled === "true");
        setRegistrationLocked(status.settings.registration_enabled_locked || false);
        setMaxApiKeys(status.settings.max_api_keys_per_user);
        setSiteUrl(status.settings.site_url || "");
      }
    } catch (error) {
      console.error("Failed to fetch system status:", error);
    }
  };

  const fetchUsers = async () => {
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
  };

  const fetchAPIKeys = async () => {
    try {
      const response = await fetch("/api/admin/api-keys", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setApiKeys(data.api_keys);
      }
    } catch (error) {
      console.error("Failed to fetch API keys:", error);
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

  const fetchDevices = async () => {
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
  };

  const fetchDeviceStats = async () => {
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

  const fetchPluginStats = async () => {
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
  };

  const fetchFirmwareVersions = async () => {
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
  }, [firmwareVersions]);

  const fetchFirmwareStats = async () => {
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
  };

  const fetchFirmwareMode = async () => {
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
  };

  const fetchDeviceModels = async () => {
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
  };

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
        setSuccessMessage(`Firmware ${version} download started`);
      } else {
        const data = await response.json();
        setError(data.error || "Failed to start firmware download");
      }
    } catch (error) {
      console.error("Failed to retry firmware download:", error);
      setError("Failed to start firmware download");
    }
  };

  const openDeleteFirmwareDialog = (version: any) => {
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
        setSuccessMessage("Firmware version deleted successfully");
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

  const openResetPasswordDialog = (user: User) => {
    setResetPasswordDialog({ isOpen: true, user });
    setNewPasswordValue("");
  };

  const closeResetPasswordDialog = () => {
    setResetPasswordDialog({ isOpen: false, user: null });
    setNewPasswordValue("");
  };

  const handleClose = () => {
    setError(null);
    setSuccessMessage(null);
    
    if (restorePerformed) {
      window.location.reload();
    } else {
      onClose();
    }
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
        setError(null);
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

  const updateSystemSetting = async (key: string, value: string) => {
    try {
      const response = await fetch("/api/admin/settings", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({ key, value }),
      });

      if (response.ok) {
        await fetchSystemStatus();
      }
    } catch (error) {
      console.error("Failed to update setting:", error);
    }
  };

  const testSMTP = async () => {
    try {
      setTestingSMTP(true);
      setError(null);
      setSmtpTestResult(null);

      const response = await fetch("/api/admin/test-smtp", {
        method: "POST",
        credentials: "include",
      });

      const result = await response.json();
      if (response.ok) {
        setError(null);
        setSmtpTestResult('working');
        // Revert back to default status after 3 seconds
        setTimeout(() => setSmtpTestResult(null), 3000);
      } else {
        setSmtpTestResult('failed');
        setTimeout(() => setSmtpTestResult(null), 3000);
        setError(result.error || t("admin.errors.smtp_test"));
      }
    } catch (error) {
      setSmtpTestResult('failed');
      setTimeout(() => setSmtpTestResult(null), 3000);
      setError(t("admin.errors.smtp_test_network"));
    } finally {
      setTestingSMTP(false);
    }
  };

  const handleCreateBackupJob = async () => {
    try {
      setCreatingBackup(true);
      setError(null);

      const response = await fetch("/api/admin/backup-job", {
        method: "POST",
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setSuccessMessage(t("admin.messages.backup_job_created"));
        await fetchBackupJobs();
      } else {
        const errorData = await response.json();
        setError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : t("admin.errors.backup_create"));
      }
    } catch (error) {
      setError(t("admin.errors.backup_create"));
    } finally {
      setCreatingBackup(false);
    }
  };


  const handleDownloadBackup = async (jobId: string) => {
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
            filename = matches[1].replace(/"/g, "");
          }
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
      } else {
        const errorData = await response.json();
        setError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : t("admin.errors.backup_download"));
      }
    } catch (error) {
      setError(t("admin.errors.backup_download"));
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

  const confirmDeleteBackup = async () => {
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
        setError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : t("admin.backup.delete_error"));
      }
    } catch (error) {
      setError(t("admin.backup.delete_error"));
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
          const response = new Response(xhr.responseText, {
            status: xhr.status,
            statusText: xhr.statusText,
            headers: {
              'Content-Type': 'application/json',
            },
          });
          resolve(response);
        } else {
          reject(new Error(xhr.responseText || `HTTP ${xhr.status}`));
        }
      });
      
      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });
      
      xhr.open('POST', '/api/admin/restore/upload');
      xhr.withCredentials = true;
      
      xhr.send(formData);
    });
  };

  const analyzeBackupFile = async (file: File) => {
    try {
      setAnalyzingBackup(true);
      setError(null);
      setBackupCounts(null);

      const formData = new FormData();
      formData.append("backup_file", file);

      const response = await fetch("/api/admin/backup/analyze", {
        method: "POST",
        credentials: "include",
        body: formData,
      });

      const result = await response.json();
      if (response.ok && result.valid) {
        setBackupCounts({
          users: result.metadata.user_count,
          api_keys: result.metadata.api_key_count,
          documents: result.metadata.document_count,
        });
        return true;
      } else {
        setError(result.error_type ? t(`admin.errors.${result.error_type}`) : t("admin.errors.backup_invalid"));
        return false;
      }
    } catch (error) {
      setError(t("admin.errors.backup_analyze") + error.message);
      return false;
    } finally {
      setAnalyzingBackup(false);
    }
  };

  const handleRestoreFileSelect = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    if (file) {
      const fileName = file.name.toLowerCase();
      const isTarGz = fileName.endsWith('.tar.gz') || fileName.endsWith('.tgz');
      
      if (!isTarGz) {
        setError(t("admin.errors.backup_file_type") + `"${file.name}"`);
        event.target.value = "";
        return;
      }
      
      try {
        setError(null);
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
              
              const analyzeResult = await analyzeResponse.json();
              if (analyzeResponse.ok && analyzeResult.valid) {
                setBackupCounts({
                  users: analyzeResult.metadata.user_count,
                  api_keys: analyzeResult.metadata.api_key_count,
                  documents: analyzeResult.metadata.document_count,
                });
                setBackupVersion(analyzeResult.metadata.aviary_version || null);
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
          setError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : t("admin.errors.restore_failed"));
          setUploadProgress(0);
          setUploadPhase('idle');
        }
      } catch (error) {
        setError(t("admin.errors.restore_failed"));
        setUploadProgress(0);
        setUploadPhase('idle');
      }
    }
    event.target.value = "";
  };

  const confirmDatabaseRestore = async () => {
    const upload = restoreConfirmDialog.upload;
    if (!upload) return;

    try {
      setRestoringBackup(true);
      setError(null);
      setSuccessMessage(null);

      const response = await fetch("/api/admin/restore", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          upload_id: upload.id,
          overwrite_files: true,
          overwrite_database: true,
          user_ids: []
        }),
      });

      const result = await response.json();
      if (response.ok) {
        setRestoreConfirmDialog({ isOpen: false, upload: null });
        setError(null);
        const filename = restoreConfirmDialog.upload?.filename || 'backup file';
        const message = t("admin.success.backup_restored", { filename });
        setSuccessMessage(message);
        setRestorePerformed(true);
        try {
          await fetchWithSessionCheck("/api/admin/status", { credentials: "include" });
          await fetchWithSessionCheck("/api/users", { credentials: "include" });
          await fetchWithSessionCheck("/api/admin/api-keys", { credentials: "include" });
          await fetchWithSessionCheck("/api/admin/backup-jobs", { credentials: "include" });
          await fetchWithSessionCheck("/api/admin/restore/uploads", { credentials: "include" });
          
          await fetchSystemStatus();
          await fetchUsers();
          // await fetchAPIKeys();
          await fetchBackupJobs();
          await fetchRestoreUploads();
        } catch (error) {
          return;
        }
      } else {
        setError(result.error_type ? t(`admin.errors.${result.error_type}`) : t("admin.errors.restore_failed"));
      }
    } catch (error) {
      setError(t("admin.errors.restore_error") + error.message);
    } finally {
      setRestoringBackup(false);
    }
  };

  const handleRestoreUpload = async (upload: RestoreUpload) => {
    setRestoreConfirmDialog({ isOpen: true, upload });
  };

  const closeRestoreConfirmDialog = () => {
    setRestoreConfirmDialog({ isOpen: false, upload: null });
    setBackupCounts(null);
    setBackupVersion(null);
  };

  const cancelRestoreUpload = async () => {
    const upload = restoreConfirmDialog.upload;
    if (!upload) return;

    // Immediately clear restore uploads for instant UI feedback
    setRestoreUploads([]);
    closeRestoreConfirmDialog();

    try {
      const response = await fetch(`/api/admin/restore/uploads/${upload.id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (!response.ok) {
        const errorData = await response.json();
        setError(errorData.error_type ? t(`admin.errors.${errorData.error_type}`) : t("admin.errors.restore_cancel_failed"));
      }
      // Always refresh to get accurate state
      await fetchRestoreUploads();
    } catch (error) {
      setError(t("admin.errors.restore_cancel_failed"));
      // Re-fetch to restore state in case of error
      await fetchRestoreUploads();
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
        setSuccessMessage("Device unlinked successfully!");
        await fetchDevices();
        await fetchDeviceStats();
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
  const createPlugin = async () => {
    if (!pluginName.trim() || !pluginType.trim()) {
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
        setSuccessMessage("Plugin created successfully!");
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
        setSuccessMessage("Plugin updated successfully!");
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
        setSuccessMessage("Plugin deleted successfully!");
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

      const response = await fetch(`/api/admin/plugins/${plugin.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          ...plugin,
          is_active: !plugin.is_active,
        }),
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

  const resetPluginForm = () => {
    setPluginName("");
    setPluginType("");
    setPluginDescription("");
    setPluginConfigSchema("");
    setPluginVersion("");
    setPluginAuthor("");
  };

  // Device utility functions

  const getSignalQuality = (rssi: number): { quality: string; strength: number; color: string } => {
    if (rssi > -50) return { quality: "Excellent", strength: 5, color: "text-emerald-600" };
    if (rssi > -60) return { quality: "Good", strength: 4, color: "text-emerald-600" };
    if (rssi > -70) return { quality: "Fair", strength: 3, color: "text-amber-600" };
    if (rssi > -80) return { quality: "Poor", strength: 2, color: "text-amber-600" };
    return { quality: "Very Poor", strength: 1, color: "text-destructive" };
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

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const formatDateOnly = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const formatDateTime = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getKeyStatus = (key: APIKey) => {
    if (!key.is_active) return "inactive";
    if (key.expires_at && new Date(key.expires_at) < new Date())
      return "expired";
    return "active";
  };

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


  const getBackupStatusButton = (job: BackupJob) => {
    if (job.status === "completed" || job.status === "failed") {
      return (
        <div className="flex gap-2 shrink-0">
          <Button
            size="sm"
            variant="outline"
            onClick={() => handleDownloadBackup(job.id)}
            disabled={job.status === "failed" || downloadingJobId === job.id}
          >
            {downloadingJobId === job.id ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <>
                <Download className="h-4 w-4 sm:hidden" />
                <span className="hidden sm:inline">{t("admin.backup.download")}</span>
              </>
            )}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => openDeleteBackupDialog(job)}
          >
            <Trash2 className="h-4 w-4 sm:hidden" />
            <span className="hidden sm:inline">{t("admin.actions.delete")}</span>
          </Button>
        </div>
      );
    }

    let content;

    switch (job.status) {
      case "pending":
      case "running":
        content = <Loader2 className="h-4 w-4 animate-spin" />;
        break;
      default:
        content = job.status;
    }

    return (
      <Button
        size="sm"
        variant="secondary"
        disabled
        className="shrink-0"
      >
        {content}
      </Button>
    );
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

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
        <Badge variant="outline" className="min-w-16 max-w-32 justify-center text-center whitespace-nowrap">
          {t("admin.status.paired")}
        </Badge>
      );
    }
    
    return (
      <Badge variant="default" className="min-w-16 max-w-32 justify-center text-center whitespace-nowrap">
        {t("admin.status.unpaired")}
      </Badge>
    );
  };

  const isCurrentUser = (user: User) => {
    return currentUser && user.id === currentUser.id;
  };

  if (!systemStatus) {
    return (
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="max-w-6xl">
          <DialogHeader>
            <DialogTitle>{t("admin.loading_states.loading")}</DialogTitle>
          </DialogHeader>
          <div className="flex items-center justify-center p-8">{t("admin.loading_states.loading")}</div>
        </DialogContent>
      </Dialog>
    );
  }

  const FirmwareManagementTab = () => (
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
          <CardDescription>
            Trigger manual checks for firmware updates and device models
          </CardDescription>
        </CardHeader>
        <CardContent className="flex gap-2">
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
        </CardContent>
      </Card>

      {/* Firmware Versions Table */}
      <Card>
        <CardHeader>
          <CardTitle>Firmware Versions</CardTitle>
          <CardDescription>
            Available firmware versions for TRMNL devices
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Version</TableHead>
                <TableHead>Released</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Size</TableHead>
                <TableHead>Notes</TableHead>
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
                    {new Date(version.released_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    {version.download_status === 'downloading' && (
                      <Badge variant="outline">
                        <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                        Downloading {version.download_progress || 0}%
                      </Badge>
                    )}
                    {version.download_status === 'downloaded' && (
                      <Badge variant="outline">
                        Downloaded
                      </Badge>
                    )}
                    {version.download_status === 'failed' && (
                      <Badge variant="destructive">
                        <XCircle className="w-3 h-3 mr-1" />
                        Failed
                      </Badge>
                    )}
                    {(!version.download_status || version.download_status === 'pending') && (
                      firmwareMode === 'proxy' ? (
                        <Badge variant="secondary">
                          <ArrowRightLeft className="w-3 h-3 mr-1" />
                          Proxying
                        </Badge>
                      ) : (
                        <Badge variant="outline">
                          <Clock className="w-3 h-3 mr-1" />
                          Pending
                        </Badge>
                      )
                    )}
                  </TableCell>
                  <TableCell>
                    {version.file_size > 0 ? 
                      `${(version.file_size / 1024 / 1024).toFixed(2)} MB` : 
                      'Unknown'
                    }
                  </TableCell>
                  <TableCell className="max-w-xs truncate">
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
  );

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-7xl mobile-dialog-content sm:max-w-7xl overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            {t("admin.title")}
          </DialogTitle>
          <DialogDescription>
            {t("admin.description")}
          </DialogDescription>
        </DialogHeader>

        {error && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              {error}
            </AlertDescription>
          </Alert>
        )}

        {successMessage && (
          <Alert>
            <CheckCircle className="h-4 w-4" />
            <AlertDescription>
              {successMessage}
            </AlertDescription>
          </Alert>
        )}

        <Tabs defaultValue="overview">
          <TabsList className="w-full">
            <TabsTrigger value="overview">
              <Activity className="h-4 w-4" />
              <span className="hidden sm:inline ml-1.5">{t("admin.tabs.overview")}</span>
            </TabsTrigger>
            <TabsTrigger value="users">
              <Users className="h-4 w-4" />
              <span className="hidden sm:inline ml-1.5">{t("admin.tabs.users")}</span>
            </TabsTrigger>
            {/* <TabsTrigger value="api-keys">
              <Key className="h-4 w-4" />
              <span className="hidden sm:inline ml-1.5">{t("admin.tabs.api_keys")}</span>
            </TabsTrigger> */}
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

          <TabsContent value="overview">
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
                    <Button size="sm" onClick={testSMTP} disabled={testingSMTP || !systemStatus.smtp.configured} className="w-full sm:w-auto">
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
          </TabsContent>

          <TabsContent value="users">
            <div className="space-y-4">
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
                          <TableHead>{t("admin.labels.username")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.email")}</TableHead>
                          <TableHead className="hidden lg:table-cell text-center">{t("admin.labels.role")}</TableHead>
                          <TableHead className="hidden lg:table-cell text-center">{t("admin.labels.status")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.created")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.last_login")}</TableHead>
                          <TableHead>{t("admin.labels.actions")}</TableHead>
                        </TableRow>
                      </TableHeader>
                    <TableBody>
                      {users.map((user) => (
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

          {/* <TabsContent value="api-keys">
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>{t("admin.counts.all_api_keys", {count: apiKeys.length})}</CardTitle>
                </CardHeader>
                <CardContent>
                    <Table className="w-full table-fixed lg:table-auto">
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t("admin.labels.name")}</TableHead>
                          <TableHead>{t("admin.labels.user")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.key_preview")}</TableHead>
                          <TableHead className="hidden lg:table-cell text-center">{t("admin.labels.status")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.created")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.last_used")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("admin.labels.expires")}</TableHead>
                          <TableHead className="lg:hidden">{t("admin.labels.actions")}</TableHead>
                        </TableRow>
                      </TableHeader>
                  <TableBody>
                    {apiKeys.map((key) => {
                      const status = getKeyStatus(key);
                      return (
                        <TableRow key={key.id}>
                          <TableCell className="font-medium">
                            <div className="truncate" title={key.name}>
                              {key.name}
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="truncate" title={key.username}>
                              {key.username}
                            </div>
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">
                            <code className="text-sm">{key.key_prefix}...</code>
                          </TableCell>
                          <TableCell className="hidden lg:table-cell text-center">
                            <Badge
                              variant={
                                status === "active"
                                  ? "outline"
                                  : status === "expired"
                                    ? "secondary"
                                    : "secondary"
                              }
                              className="min-w-16 max-w-32 justify-center text-center whitespace-nowrap"
                            >
                              {t(`settings.status.${status}`)}
                            </Badge>
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-default">{formatDateOnly(key.created_at)}</span>
                              </TooltipTrigger>
                              <TooltipContent>
                                {formatDateTime(key.created_at)}
                              </TooltipContent>
                            </Tooltip>
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">
                            {key.last_used ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="cursor-default">{formatDateOnly(key.last_used)}</span>
                                </TooltipTrigger>
                                <TooltipContent>
                                  {formatDateTime(key.last_used)}
                                </TooltipContent>
                              </Tooltip>
                            ) : (
                              t("admin.never")
                            )}
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">
                            {key.expires_at ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="cursor-default">{formatDateOnly(key.expires_at)}</span>
                                </TooltipTrigger>
                                <TooltipContent>
                                  {formatDateTime(key.expires_at)}
                                </TooltipContent>
                              </Tooltip>
                            ) : (
                              t("admin.never")
                            )}
                          </TableCell>
                          <TableCell className="lg:hidden">
                            <Button
                              size="sm"
                              variant="outline"
                              className="w-full sm:w-auto"
                              onClick={() => setViewKey(key)}
                            >
                              {t('admin.actions.details', 'Details')}
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
                </CardContent>
              </Card>
            </div>
          </TabsContent> */}

          <TabsContent value="settings">
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>Site Configuration</CardTitle>
                  <CardDescription>Configure the public URL for your instance</CardDescription>
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
                  <CardTitle>{t("admin.cards.user_management")}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="space-y-1">
                      <Label htmlFor="registration-enabled">
                        {t("admin.labels.enable_registration")}
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        {registrationLocked 
                          ? "Set via environment variable" 
                          : t("admin.descriptions.registration_help")}
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
                  <CardTitle>{t("admin.cards.api_key_settings")}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <Label htmlFor="max-api-keys">
                      {t("admin.labels.max_api_keys")}
                    </Label>
                    <Input
                      id="max-api-keys"
                      type="number"
                      min="1"
                      max="100"
                      value={maxApiKeys}
                      onChange={(e) => {
                        const value = e.target.value;
                        setMaxApiKeys(value);
                        
                        if (value === "") {
                          setMaxApiKeysError(null);
                          return;
                        }
                        
                        const numValue = parseInt(value, 10);
                        if (isNaN(numValue) || numValue < 1 || numValue > 100) {
                          setMaxApiKeysError(t("admin.errors.max_api_keys_invalid"));
                        } else {
                          setMaxApiKeysError(null);
                        }
                      }}
                      onBlur={() => {
                        const numValue = parseInt(maxApiKeys, 10);
                        if (!isNaN(numValue) && numValue >= 1 && numValue <= 100) {
                          updateSystemSetting("max_api_keys_per_user", maxApiKeys);
                        }
                      }}
                      className={cn(
                        "mt-2 max-w-[200px]",
                        maxApiKeysError && "border-destructive"
                      )}
                      aria-invalid={!!maxApiKeysError}
                    />
                    <p className="text-sm text-muted-foreground mt-2">
                      {t("admin.descriptions.max_api_keys_help")}
                    </p>
                    {maxApiKeysError && (
                      <p className="text-sm text-destructive mt-1">
                        {maxApiKeysError}
                      </p>
                    )}
                  </div>
                </CardContent>
              </Card>

            </div>
          </TabsContent>

          <TabsContent value="system">
            <div className="space-y-4">
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
                  
                  {backupJobs.length > 0 && (
                    <div className="mt-6">
                      <h4 className="text-sm font-medium mb-3">{t("admin.jobs.recent_jobs")}</h4>
                      <div className="space-y-2">
                        {backupJobs
                          .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
                          .slice(0, 8)
                          .map((job) => (
                            <div key={job.id} className="flex items-center justify-between p-3 border rounded-lg">
                              <div className="text-sm flex-1 min-w-0">
                                <div className="font-medium">
                                  {formatDate(job.created_at)}
                                </div>
                                <div className="text-muted-foreground">
                                  {job.status === "failed" ? t("admin.backup.status.failed") :
                                   job.file_size ? formatFileSize(job.file_size) : 
                                   (job.status === "running" || job.status === "pending") ? 
                                   (job.status === "pending" ? t("admin.backup.status.pending") : t("admin.backup.status.running")) : 
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
                          ))}
                      </div>
                    </div>
                  )}

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


              {/* <Card>
                <CardHeader>
                  <CardTitle>Maintenance</CardTitle>
                </CardHeader>
                <CardContent>
                  <Button variant="outline">
                    <Trash2 className="h-4 w-4 mr-2" />
                    Cleanup Old Data
                  </Button>
                </CardContent>
              </Card> */}

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

          <TabsContent value="devices">
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
                        <TableHead>Friendly ID</TableHead>
                        <TableHead>Name</TableHead>
                        <TableHead className="hidden md:table-cell">Owner</TableHead>
                        <TableHead className="hidden lg:table-cell">Model</TableHead>
                        <TableHead className="hidden lg:table-cell">MAC Address</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="hidden lg:table-cell">Last Seen</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {devices.map((device) => (
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
                            {device.device_model?.display_name || device.model_name || (
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
                                <Badge variant="secondary" className="flex items-center gap-1">
                                  <EyeOff className="h-3 w-3" />
                                  Inactive
                                </Badge>
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

          <TabsContent value="plugins">
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
                      {pluginStats?.total_user_plugins || 0}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {pluginStats?.active_user_plugins || 0} active
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Avg per User</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {systemStatus ? Math.round((pluginStats?.total_user_plugins || 0) / Math.max(systemStatus.database.total_users, 1) * 10) / 10 : 0}
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
                        <TableHead>Name</TableHead>
                        <TableHead>Type</TableHead>
                        <TableHead className="hidden md:table-cell">Version</TableHead>
                        <TableHead className="hidden lg:table-cell">Author</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="hidden lg:table-cell">Created</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {plugins.map((plugin) => (
                        <TableRow key={plugin.id}>
                          <TableCell className="font-medium">{plugin.name}</TableCell>
                          <TableCell>
                            <Badge variant="outline">{plugin.type}</Badge>
                          </TableCell>
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
          <TabsContent value="firmware">
            <FirmwareManagementTab />
          </TabsContent>
        </Tabs>
      </DialogContent>

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
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={closeResetPasswordDialog}>
              {t("admin.actions.cancel")}
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
                        <p className="text-sm text-destructive font-medium">
                          {t("admin.dialogs.version_warning")}
                        </p>
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

      {/* User Details Dialog */}
      <Dialog open={!!viewUser} onOpenChange={(open) => {
        if (!open) setViewUser(null);
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{viewUser?.username}</DialogTitle>
          </DialogHeader>
          {viewUser && (
            <>
              <div className="space-y-2 text-sm">
                <p>
                  <strong>{t('admin.labels.email')}:</strong> {viewUser.email}
                </p>
                <p>
                  <strong>{t('admin.labels.user_id')}:</strong> {viewUser.id}
                </p>
                <p>
                  <strong>{t('admin.labels.role')}:</strong>{' '}
                  {viewUser.is_admin ? t('admin.roles.admin') : t('admin.roles.user')}
                </p>
                <p>
                  <strong>{t('admin.labels.status')}:</strong>{' '}
                  {!viewUser.is_active
                    ? t('admin.status.inactive')
                    : viewUser.rmapi_paired
                    ? t('admin.status.paired')
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
                  <PopoverContent className="w-[calc(100vw-2rem)] sm:w-auto p-2" align="center">
                    <div className="flex flex-col gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          openResetPasswordDialog(viewUser);
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
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* API Key Details Dialog */}
      <Dialog open={!!viewKey} onOpenChange={() => setViewKey(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{viewKey?.name}</DialogTitle>
          </DialogHeader>
          {viewKey && (
            <div className="space-y-2 text-sm">
              <p>
                <strong>{t('admin.labels.user')}:</strong> {viewKey.username}
              </p>
              <p>
                <strong>{t('admin.labels.key_preview')}:</strong>{' '}
                <code>{viewKey.key_prefix}...</code>
              </p>
              <p>
                <strong>{t('admin.labels.status')}:</strong> {t(`settings.status.${getKeyStatus(viewKey)}`)}
              </p>
              <p>
                <strong>{t('admin.labels.created')}:</strong> {formatDate(viewKey.created_at)}
              </p>
              <p>
                <strong>{t('admin.labels.last_used')}:</strong>{' '}
                {viewKey.last_used ? formatDate(viewKey.last_used) : t('admin.never')}
              </p>
              <p>
                <strong>{t('admin.labels.expires')}:</strong>{' '}
                {viewKey.expires_at ? formatDate(viewKey.expires_at) : t('admin.never')}
              </p>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Backup Confirmation Dialog */}
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
              {t("settings.dialogs.cannot_undo")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>
              {t("admin.actions.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDeleteBackup}
              disabled={deleting}
              variant="destructive"
            >
              {deleting ? t("admin.loading_states.deleting") : t("admin.actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
                <strong>Model:</strong> {viewDevice.device_model?.display_name || viewDevice.model_name || "Unknown"}
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
        <DialogContent className="sm:max-w-4xl">
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
                  <TableHead>Status</TableHead>
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
                    <TableCell>
                      <Badge variant={model.is_active ? "default" : "secondary"}>
                        {model.is_active ? "Active" : "Inactive"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
                {deviceModels.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground">
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
            <DialogTitle>{viewPlugin?.name}</DialogTitle>
          </DialogHeader>
          {viewPlugin && (
            <div className="space-y-2 text-sm">
              <p>
                <strong>Type:</strong> <Badge variant="outline">{viewPlugin.type}</Badge>
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
                <span className={viewPlugin.is_active ? "text-green-600" : "text-gray-600"}>
                  {viewPlugin.is_active ? "Active" : "Inactive"}
                </span>
              </p>
              <p>
                <strong>Created:</strong> {formatDate(viewPlugin.created_at)}
              </p>
              {viewPlugin.config_schema && (
                <div>
                  <strong>Config Schema:</strong>
                  <pre className="text-xs bg-gray-100 p-2 rounded mt-1 overflow-auto max-h-32">
                    {JSON.stringify(JSON.parse(viewPlugin.config_schema || "{}"), null, 2)}
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
            <DialogTitle>Create System Plugin</DialogTitle>
            <DialogDescription>
              Create a new system plugin that users can instantiate.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="plugin-name">Name *</Label>
              <Input
                id="plugin-name"
                value={pluginName}
                onChange={(e) => setPluginName(e.target.value)}
                placeholder="e.g., Weather Display"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="plugin-type">Type *</Label>
              <Input
                id="plugin-type"
                value={pluginType}
                onChange={(e) => setPluginType(e.target.value)}
                placeholder="e.g., widget, data-source, utility"
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
              <Label htmlFor="plugin-config-schema">Config Schema (JSON)</Label>
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
              disabled={creatingPlugin || !pluginName.trim() || !pluginType.trim()}
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
              <Label htmlFor="edit-plugin-name">Name *</Label>
              <Input
                id="edit-plugin-name"
                value={pluginName}
                onChange={(e) => setPluginName(e.target.value)}
                placeholder="e.g., Weather Display"
                className="mt-2"
              />
            </div>
            <div>
              <Label htmlFor="edit-plugin-type">Type *</Label>
              <Input
                id="edit-plugin-type"
                value={pluginType}
                onChange={(e) => setPluginType(e.target.value)}
                placeholder="e.g., widget, data-source, utility"
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
              <Label htmlFor="edit-plugin-config-schema">Config Schema (JSON)</Label>
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
              disabled={!pluginName.trim() || !pluginType.trim() || !hasPluginChanges()}
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

    </Dialog>
  );
}
