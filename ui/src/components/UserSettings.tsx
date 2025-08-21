import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/components/AuthProvider";
import { useConfig } from "@/components/ConfigProvider";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandItem,
} from "@/components/ui/command";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Settings,
  Key,
  User,
  UserCog,
  Save,

  Copy,
  Eye,
  EyeOff,
  CheckCircle,
  XCircle,
  Clock,
  AlertTriangle,
} from "lucide-react";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  is_active: boolean;
  last_used?: string;
  expires_at?: string;
  created_at: string;
}

interface UserSettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export function UserSettings({ isOpen, onClose }: UserSettingsProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { user, isLoading: userDataLoading, refetchAuth: refetch } = useAuth();
  const { config } = useConfig();

  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCopiedText, setShowCopiedText] = useState(false);
  const [maxApiKeys, setMaxApiKeys] = useState<number>(10);

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [timezone, setTimezone] = useState("");
  const [timezoneOpen, setTimezoneOpen] = useState(false);
  
  // Original values for change tracking
  const [originalValues, setOriginalValues] = useState({
    username: "",
    email: "",
    timezone: ""
  });

  // Grouped timezone data for better UX
  const getGroupedTimezones = () => {
    const commonTimezones = [
      'UTC',
      'America/New_York',
      'America/Chicago', 
      'America/Denver',
      'America/Los_Angeles',
      'Europe/London',
      'Europe/Paris',
      'Europe/Berlin',
      'Asia/Tokyo',
      'Asia/Shanghai',
      'Australia/Sydney'
    ];

    const allTimezones = Intl.supportedValuesOf('timeZone');
    const now = new Date();
    
    const formatTimezone = (tz: string) => {
      try {
        const time = now.toLocaleTimeString('en-US', {
          timeZone: tz,
          hour: 'numeric',
          minute: '2-digit',
          hour12: true
        });
        const abbr = now.toLocaleDateString('en', {
          timeZone: tz,
          timeZoneName: 'short'
        }).split(', ').pop() || '';
        
        return {
          value: tz,
          label: tz.replace(/_/g, ' '),
          time: time,
          abbr: abbr
        };
      } catch {
        return {
          value: tz,
          label: tz.replace(/_/g, ' '),
          time: '',
          abbr: ''
        };
      }
    };

    const commonFormatted = commonTimezones.map(formatTimezone);
    const otherTimezones = allTimezones
      .filter(tz => !commonTimezones.includes(tz))
      .map(formatTimezone)
      .sort((a, b) => a.label.localeCompare(b.label));

    return {
      common: commonFormatted,
      other: otherTimezones
    };
  };

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [deletePassword, setDeletePassword] = useState("");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyExpiry, setNewKeyExpiry] = useState("never");
  const [showNewKey, setShowNewKey] = useState<string | null>(null);

  const [deleteAccountDialog, setDeleteAccountDialog] = useState(false);

  const [deleteKeyDialog, setDeleteKeyDialog] = useState<{
    isOpen: boolean;
    key: APIKey | null;
  }>({ isOpen: false, key: null });

  const [viewKey, setViewKey] = useState<APIKey | null>(null);
  const [deleteFromDetails, setDeleteFromDetails] = useState(false);

  const handleClose = () => {
    setError(null);
    setShowCopiedText(false);
    onClose();
  };

  useEffect(() => {
    if (isOpen) {
      fetchAPIKeys();
      if (user) {
        setUsername(user.username);
        setEmail(user.email);
        // Use user's timezone if available, otherwise default to browser timezone
        setTimezone(user.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone);
      }
    }
  }, [isOpen, user]);

  // Update form fields when user data changes
  useEffect(() => {
    if (user) {
      setUsername(user.username);
      setEmail(user.email);
      const userTimezone = user.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
      setTimezone(userTimezone);
      
      setOriginalValues({
        username: user.username,
        email: user.email,
        timezone: userTimezone
      });
    }
  }, [user]);

  // Listen for logout event to clear sensitive state
  useEffect(() => {
    const handleLogout = () => {
      // Clear all sensitive state
      setApiKeys([]);
      setError(null);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setDeletePassword("");
      setDeleteConfirmation("");
      setNewKeyName("");
      setNewKeyExpiry("");
      setShowNewKey(null);
      setViewKey(null);
      
      // Close any open dialogs
      setDeleteAccountDialog(false);
      setDeleteKeyDialog({ isOpen: false, key: null });
      
      // Reset form state to defaults
      setUsername("");
      setEmail("");
      setTimezone("");
      
      // Reset original values
      setOriginalValues({
        username: "",
        email: "",
        timezone: ""
      });
    };

    window.addEventListener('logout', handleLogout);

    return () => {
      window.removeEventListener('logout', handleLogout);
    };
  }, []);

  const fetchAPIKeys = async () => {
    try {
      const response = await fetch("/api/api-keys", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        // Handle both old and new response formats for backward compatibility
        if (Array.isArray(data)) {
          // Old format: direct array of API keys
          setApiKeys(data);
        } else {
          // New format: object with api_keys, max_api_keys, etc.
          setApiKeys(data.api_keys || []);
          if (data.max_api_keys) {
            setMaxApiKeys(data.max_api_keys);
          }
        }
      }
    } catch (error) {
      console.error("Failed to fetch API keys:", error);
    }
  };

  // Check if there are unsaved changes
  const hasChanges = () => {
    return (
      (!config?.oidcEnabled && username !== originalValues.username) ||
      (!config?.oidcEnabled && email !== originalValues.email) ||
      timezone !== originalValues.timezone
    );
  };

  const updateProfile = async () => {
    try {
      setSaving(true);
      setError(null);

      const response = await fetch("/api/profile", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          ...(config?.oidcEnabled ? {} : { username, email }),
          timezone
        }),
      });

      if (response.ok) {
        // Update original values to reflect the saved state
        setOriginalValues({
          username,
          email,
          timezone
        });
        
        // Refetch user data to ensure we have the latest from server
        // This happens after updating originalValues to prevent flash
        setTimeout(() => {
          refetch();
        }, 100);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update profile");
      }
    } catch (error) {
      setError(t("settings.errors.update_profile"));
    } finally {
      setSaving(false);
    }
  };

  const updatePassword = async () => {
    if (newPassword !== confirmPassword) {
      setError(t("settings.errors.passwords_mismatch"));
      return;
    }

    try {
      setSaving(true);
      setError(null);

      const response = await fetch("/api/profile/password", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });

      if (response.ok) {
        setCurrentPassword("");
        setNewPassword("");
        setConfirmPassword("");
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to update password");
      }
    } catch (error) {
      setError(t("settings.errors.update_password"));
    } finally {
      setSaving(false);
    }
  };

  const createAPIKey = async () => {
    try {
      setSaving(true);
      setError(null);

      const body: any = { name: newKeyName };
      if (newKeyExpiry && newKeyExpiry !== "" && newKeyExpiry !== "never") {
        let expiryDate: Date;
        const now = new Date();
        switch (newKeyExpiry) {
          case "1week":
            expiryDate = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
            break;
          case "1month":
            expiryDate = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
            break;
          case "3months":
            expiryDate = new Date(now.getTime() + 90 * 24 * 60 * 60 * 1000);
            break;
          case "1year":
            expiryDate = new Date(now.getTime() + 365 * 24 * 60 * 60 * 1000);
            break;
          default:
            expiryDate = new Date(newKeyExpiry); // fallback for date strings
        }
        body.expires_at = Math.floor(expiryDate.getTime() / 1000);
      }

      const response = await fetch("/api/api-keys", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(body),
      });

      if (response.ok) {
        const newKey = await response.json();
        setShowNewKey(newKey.api_key);
        setNewKeyName("");
        setNewKeyExpiry("never");
        await fetchAPIKeys();
      } else {
        const errorData = await response.json();
        const errorMessage = errorData.error || "Failed to create API key";
        
        // Check for specific API key limit error and use i18n message
        if (errorMessage === "Maximum number of API keys reached" || 
            errorMessage.includes("maximum number of API keys") || 
            errorMessage.includes("reached")) {
          setError(t('settings.messages.api_key_limit_reached', { maxKeys: maxApiKeys }));
        } else {
          setError(errorMessage);
        }
      }
    } catch (error) {
      setError(t("settings.errors.create_api_key"));
    } finally {
      setSaving(false);
    }
  };

  const openDeleteKeyDialog = (key: APIKey) => {
    setDeleteKeyDialog({ isOpen: true, key });
  };

  const closeDeleteKeyDialog = () => {
    const wasFromDetails = deleteFromDetails;
    const keyToRestore = deleteKeyDialog.key;
    setDeleteKeyDialog({ isOpen: false, key: null });
    setDeleteFromDetails(false);
    if (wasFromDetails && keyToRestore) {
      setViewKey(keyToRestore);
    }
  };

  const confirmDeleteAPIKey = async () => {
    if (!deleteKeyDialog.key) return;

    try {
      const response = await fetch(`/api/api-keys/${deleteKeyDialog.key.id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        await fetchAPIKeys();
        setDeleteKeyDialog({ isOpen: false, key: null });
        setDeleteFromDetails(false);
        setViewKey(null);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete API key");
      }
    } catch (error) {
      setError(t("settings.errors.delete_api_key"));
    }
  };

  const openDeleteAccountDialog = () => {
    setDeletePassword("");
    setDeleteConfirmation("");
    setDeleteAccountDialog(true);
  };

  const closeDeleteAccountDialog = () => {
    setDeleteAccountDialog(false);
    setDeletePassword("");
    setDeleteConfirmation("");
  };

  const confirmDeleteAccount = async () => {
    try {
      setSaving(true);
      setError(null);

      const response = await fetch("/api/profile", {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          current_password: deletePassword,
          confirmation: deleteConfirmation,
        }),
      });

      if (response.ok) {
        // Account deleted successfully, redirect to login
        navigate("/");
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete account");
      }
    } catch (error) {
      setError(t("settings.errors.delete_account"));
    } finally {
      setSaving(false);
      setDeleteAccountDialog(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setShowCopiedText(true);
    setTimeout(() => setShowCopiedText(false), 3000);
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

  const canDeleteAccount = deletePassword && deleteConfirmation === t('settings.placeholders.delete_confirm');

  if (userDataLoading) {
    return (
      <Dialog open={isOpen} onOpenChange={handleClose}>
        <DialogContent className="max-w-4xl">
          <div className="flex items-center justify-center p-8">{t('settings.loading_states.loading')}</div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <>
      <Dialog open={isOpen} onOpenChange={handleClose}>
        <DialogContent className="max-w-7xl mobile-dialog-content sm:max-w-7xl overflow-y-auto !top-[0vh] !translate-y-0 sm:!top-[6vh]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            {t("settings.title")}
            </DialogTitle>
            <DialogDescription>
            {t("settings.description")}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="bg-destructive/10 border border-destructive/20 rounded-md p-3 text-destructive">
              {error}
            </div>
          )}


          <Tabs defaultValue="profile">
            <TabsList className="w-full">
              <TabsTrigger value="profile">
                <User className="h-4 w-4" />
                <span className="ml-1.5">{t("settings.tabs.profile")}</span>
              </TabsTrigger>
              <TabsTrigger value="account">
                <UserCog className="h-4 w-4" />
                <span className="ml-1.5">{t("settings.tabs.account")}</span>
              </TabsTrigger>

              {/* Temporarily hidden
              <TabsTrigger value="api-keys">
                <Key className="h-4 w-4" />
                <span className="ml-1.5">{t("settings.tabs.api_keys")}</span>
              </TabsTrigger>
              */}
            </TabsList>

            <TabsContent value="profile">
              <Card>
                <CardHeader>
                  <CardTitle>{t("settings.cards.profile_information")}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="username">{t("settings.labels.username")}</Label>
                      {config?.oidcEnabled ? (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Input
                              id="username"
                              value={username}
                              readOnly
                              className="mt-2 bg-muted text-muted-foreground cursor-default"
                            />
                          </TooltipTrigger>
                          <TooltipContent>
                            {t("admin.tooltips.oidc_managed")}
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        <Input
                          id="username"
                          value={username}
                          disabled
                          className="mt-2"
                        />
                      )}
                    </div>
                    <div>
                      <Label htmlFor="email">{t("settings.labels.email")}</Label>
                      {config?.oidcEnabled ? (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Input
                              id="email"
                              type="email"
                              value={email}
                              readOnly
                              className="mt-2 bg-muted text-muted-foreground cursor-default"
                            />
                          </TooltipTrigger>
                          <TooltipContent>
                            {t("admin.tooltips.oidc_managed")}
                          </TooltipContent>
                        </Tooltip>
                      ) : (
                        <Input
                          id="email"
                          type="email"
                          value={email}
                          onChange={(e) => setEmail(e.target.value)}
                          className="mt-2"
                        />
                      )}
                    </div>

                    <div>
                      <Label htmlFor="timezone">Timezone</Label>
                      <Popover modal={true} open={timezoneOpen} onOpenChange={setTimezoneOpen}>
                        <PopoverTrigger asChild>
                          <Button
                            variant="outline"
                            role="combobox"
                            aria-expanded={timezoneOpen}
                            className="mt-2 w-full justify-between"
                          >
                            {timezone ? 
                              `${timezone.replace(/_/g, ' ')} ${(() => {
                                try {
                                  const now = new Date();
                                  const time = now.toLocaleTimeString('en-US', {
                                    timeZone: timezone,
                                    hour: 'numeric',
                                    minute: '2-digit',
                                    hour12: true
                                  });
                                  return `(${time})`;
                                } catch {
                                  return '';
                                }
                              })()} ` : 
                              "Select timezone..."
                            }
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent 
                          className="w-96 p-0 mobile-dialog-content sm:w-96 !top-[0vh] !translate-y-0 sm:!top-auto sm:!translate-y-0"
                          side="bottom"
                          align="start"
                          onOpenAutoFocus={(e) => {
                            // Prevent auto-focus issues on iOS
                            if (typeof navigator !== 'undefined' && /iPad|iPhone|iPod/.test(navigator.userAgent)) {
                              e.preventDefault();
                            }
                          }}
                        >
                          <Command>
                            <CommandInput placeholder="Search timezones..." className="h-8" />
                            <CommandList>
                              <CommandEmpty>No timezone found.</CommandEmpty>
                              {(() => {
                                const { common, other } = getGroupedTimezones();
                                return (
                                  <>
                                    {/* Common timezones with star indicator */}
                                    {common.map((tz) => (
                                      <CommandItem
                                        key={tz.value}
                                        value={tz.label}
                                        onSelect={() => {
                                          setTimezone(tz.value);
                                          setTimezoneOpen(false);
                                        }}
                                        className="cursor-pointer"
                                        style={{ pointerEvents: 'auto' }}
                                      >
                                        <div className="flex items-center justify-between w-full">
                                          <span>{tz.label}</span>
                                          <span className="text-sm text-muted-foreground">
                                            {tz.time} {tz.abbr}
                                          </span>
                                        </div>
                                      </CommandItem>
                                    ))}
                                    {/* All other timezones */}
                                    {other.map((tz) => (
                                      <CommandItem
                                        key={tz.value}
                                        value={tz.label}
                                        onSelect={() => {
                                          setTimezone(tz.value);
                                          setTimezoneOpen(false);
                                        }}
                                        className="cursor-pointer"
                                        style={{ pointerEvents: 'auto' }}
                                      >
                                        <div className="flex items-center justify-between w-full">
                                          <span>{tz.label}</span>
                                          <span className="text-sm text-muted-foreground">
                                            {tz.time} {tz.abbr}
                                          </span>
                                        </div>
                                      </CommandItem>
                                    ))}
                                  </>
                                );
                              })()}
                            </CommandList>
                          </Command>
                        </PopoverContent>
                      </Popover>
                      <p className="text-xs text-muted-foreground mt-1">
                        Used for schedule displays and timezone-aware features.
                      </p>
                    </div>
                  </div>

                  <div className="flex flex-col sm:flex-row sm:justify-end">
                    <Button onClick={updateProfile} disabled={saving || !hasChanges()} className="w-full sm:w-auto">
                      {saving ? t('settings.loading_states.saving') : t('settings.buttons.save_changes')}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="account">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <Card>
                  <CardHeader>
                    <CardTitle>{t("settings.cards.change_password")}</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div>
                      <Label htmlFor="current-password">{t("settings.labels.current_password")}</Label>
                      <Input
                        id="current-password"
                        type="password"
                        value={currentPassword}
                        onChange={(e) => setCurrentPassword(e.target.value)}
                        placeholder={t('settings.placeholders.old_password')}
                        className="mt-2"
                      />
                    </div>

                    <div>
                      <Label htmlFor="new-password">{t("settings.labels.new_password")}</Label>
                      <Input
                        id="new-password"
                        type="password"
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        placeholder={t('settings.placeholders.new_password')}
                        className="mt-2"
                      />
                    </div>

                    <div>
                      <Label htmlFor="confirm-password">{t("settings.labels.confirm_new_password")}</Label>
                      <Input
                        id="confirm-password"
                        type="password"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        placeholder={t('settings.placeholders.new_password')}
                        className="mt-2"
                      />
                    </div>

                    <div className="flex flex-col sm:flex-row sm:justify-end">
                      <Button
                        onClick={updatePassword}
                        className="w-full sm:w-auto"
                        disabled={
                          saving ||
                          !currentPassword ||
                          !newPassword ||
                          !confirmPassword
                        }
                      >
                        {saving ? t('settings.loading_states.updating') : t('settings.buttons.update_password')}
                      </Button>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>
                      {t("settings.cards.delete_account")}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-muted-foreground mb-4">
                      {t("settings.messages.delete_warning_intro")}
                    </p>
                    <ul className="text-sm text-muted-foreground list-disc list-outside ml-6 sm:ml-5 space-y-1 mb-4">
                      <li>{t("settings.delete_warnings.api_keys")}</li>
                      <li>{t("settings.delete_warnings.profile")}</li>
                    </ul>
                    <div className="flex flex-col sm:flex-row sm:justify-end">
                      <Button
                        variant="outline"
                        onClick={openDeleteAccountDialog}
                        className="w-full sm:w-auto"
                      >
                        {t('settings.buttons.delete_my_account')}
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>

            <TabsContent value="api-keys">
              <div className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle>{t("settings.cards.create_api_key")}</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <Label htmlFor="key-name">{t("settings.labels.api_key_name")}</Label>
                        <Input
                          id="key-name"
                          value={newKeyName}
                          onChange={(e) => setNewKeyName(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && newKeyName && !saving && apiKeys.length < maxApiKeys) {
                              e.preventDefault();
                              createAPIKey();
                            }
                          }}
                          placeholder={t('settings.placeholders.api_key')}
                          className="mt-2"
                        />
                        {apiKeys.length < maxApiKeys ? (
                          <p className="text-sm text-muted-foreground mt-1 ml-1">
                            {t('settings.messages.api_keys_remaining', { 
                              remaining: maxApiKeys - apiKeys.length, 
                              maxKeys: maxApiKeys 
                            })}
                          </p>
                        ) : (
                          <p className="text-sm text-destructive mt-1 ml-1">
                            {t('settings.messages.api_key_limit_reached', { maxKeys: maxApiKeys })}
                          </p>
                        )}
                      </div>
                      <div>
                        <Label htmlFor="key-expiry">{t("settings.labels.expires")}</Label>
                        <Select
                          value={newKeyExpiry}
                          onValueChange={setNewKeyExpiry}
                        >
                          <SelectTrigger className="mt-2 w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="never">{t('settings.never')}</SelectItem>
                            <SelectItem value="1week">{t("settings.expiry_options.one_week")}</SelectItem>
                            <SelectItem value="1month">{t("settings.expiry_options.one_month")}</SelectItem>
                            <SelectItem value="3months">{t("settings.expiry_options.three_months")}</SelectItem>
                            <SelectItem value="1year">{t("settings.expiry_options.one_year")}</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    <div className="flex flex-col sm:flex-row sm:justify-end">
                      <Button
                        onClick={createAPIKey}
                        disabled={saving || !newKeyName || apiKeys.length >= maxApiKeys}
                        className="w-full sm:w-auto"
                      >
                        {t("settings.actions.create_api_key")}
                      </Button>
                    </div>
                  </CardContent>
                </Card>

                {showNewKey && (
                  <Card className="border-primary/20 bg-primary/10">
                    <CardHeader>
                      <CardTitle className="text-primary">
                        {t("settings.messages.api_key_created_title")}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm text-primary/80 mb-2">
                        {t("settings.messages.api_key_created_message")}
                      </p>
                      <div className="flex flex-col sm:flex-row sm:items-center gap-2 p-2 bg-card rounded border">
                        <code className="block min-w-0 w-full sm:flex-1 font-mono text-sm break-all">
                          {showNewKey}
                        </code>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => copyToClipboard(showNewKey)}
                          className="shrink-0"
                        >
                          {showCopiedText ? t('settings.tooltips.api_key_copied') : <Copy className="h-4 w-4" />}
                        </Button>
                      </div>
                      <div className="flex justify-end mt-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setShowNewKey(null)}
                          className="w-full sm:w-auto"
                        >
                          {t("settings.actions.got_it")}
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                )}

                <Card>
                  <CardHeader>
                    <CardTitle>{t("settings.cards.your_api_keys")}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    {apiKeys.length === 0 ? (
                      <p className="text-center text-muted-foreground py-4">
                        {t("settings.messages.no_api_keys")}
                      </p>
                    ) : (
                      <Table className="w-full table-fixed lg:table-auto">
                        <TableHeader>
                          <TableRow>
                          <TableHead className="w-auto lg:w-[200px]">{t("settings.labels.api_key_name")}</TableHead>
                          <TableHead className="hidden lg:table-cell w-32">{t("settings.labels.key_preview")}</TableHead>
                          <TableHead className="hidden lg:table-cell text-center">{t("settings.labels.status")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("settings.labels.created")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("settings.labels.last_used")}</TableHead>
                          <TableHead className="hidden lg:table-cell">{t("settings.labels.expires")}</TableHead>
                          <TableHead></TableHead>
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
                                <TableCell className="hidden lg:table-cell w-32">
                                  <code className="text-sm">
                                    {key.key_prefix}...
                                  </code>
                                </TableCell>
                                <TableCell className="hidden lg:table-cell text-center">
                                  <Badge
                                    variant={
                                      status === "active"
                                        ? "default"
                                        : status === "expired"
                                          ? "destructive"
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
                                    t('settings.never')
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
                                    t('settings.never')
                                  )}
                                </TableCell>
                                <TableCell>
                                  <div className="flex flex-col sm:flex-row gap-2">
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      className="lg:hidden w-full sm:w-auto"
                                      onClick={() => setViewKey(key)}
                                    >
                                      {t('settings.actions.details', 'Details')}
                                    </Button>
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => openDeleteKeyDialog(key)}
                                      className="hidden lg:inline-flex w-full sm:w-auto"
                                    >
                                      {t("admin.actions.delete")}
                                    </Button>
                                  </div>
                                </TableCell>
                              </TableRow>
                            );
                          })}
                        </TableBody>
                      </Table>
                    )}
                  </CardContent>
                </Card>
              </div>
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>

      {/* Delete Account Dialog */}
      <AlertDialog open={deleteAccountDialog} onOpenChange={closeDeleteAccountDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Account
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("settings.dialogs.delete_account_confirmation")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="delete-password">Current Password</Label>
              <Input
                id="delete-password"
                type="password"
                value={deletePassword}
                onChange={(e) => setDeletePassword(e.target.value)}
                placeholder={t('settings.placeholders.current_password')}
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="delete-confirmation">{t("settings.dialogs.delete_account_type_confirm", {confirmText: t('settings.placeholders.delete_confirm')})}</Label>
              <Input
                id="delete-confirmation"
                value={deleteConfirmation}
                onChange={(e) => setDeleteConfirmation(e.target.value)}
                placeholder={t('settings.placeholders.delete_confirm')}
                className="mt-1"
              />
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={closeDeleteAccountDialog} disabled={saving}>
              {t("settings.actions.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDeleteAccount}
              disabled={saving || !canDeleteAccount}
              variant="destructive"
            >
              {saving ? t('settings.loading_states.deleting') : t('settings.buttons.delete_my_account')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete API Key Dialog */}
      <AlertDialog
        open={deleteKeyDialog.isOpen}
        onOpenChange={closeDeleteKeyDialog}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              {t("settings.dialogs.delete_api_key_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("settings.dialogs.delete_api_key_message")}{" "}
              <strong>"{deleteKeyDialog.key?.name}"</strong>?
              <br />
              <br />
              {t("settings.dialogs.delete_api_key_consequences_title")}
              <ul className="list-disc list-outside ml-6 sm:ml-5 mt-2 space-y-1">
                <li>{t("settings.dialogs.delete_api_key_consequences.revoke")}</li>
                <li>{t("settings.dialogs.delete_api_key_consequences.stop_apps")}</li>
                <li>{t("settings.dialogs.delete_api_key_consequences.remove")}</li>
              </ul>
              <br />
              <strong className="text-destructive">
                {t("settings.dialogs.cannot_undo")}
              </strong>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={closeDeleteKeyDialog}>
              {t("settings.actions.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmDeleteAPIKey}
              variant="destructive"
            >
              {t("settings.actions.delete_api_key")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* API Key Details Dialog */}
      <Dialog open={!!viewKey} onOpenChange={() => setViewKey(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{viewKey?.name}</DialogTitle>
          </DialogHeader>
          {viewKey && (
            <>
              <div className="space-y-2 text-sm">
                <p>
                  <strong>{t('settings.labels.key_preview')}:</strong>{' '}
                  <code>{viewKey.key_prefix}...</code>
                </p>
                <p>
                  <strong>{t('settings.labels.status')}:</strong>{' '}
                  {t(`settings.status.${getKeyStatus(viewKey)}`)}
                </p>
                <p>
                  <strong>{t('settings.labels.created')}:</strong>{' '}
                  {formatDate(viewKey.created_at)}
                </p>
                <p>
                  <strong>{t('settings.labels.last_used')}:</strong>{' '}
                  {viewKey.last_used ? formatDate(viewKey.last_used) : t('settings.never')}
                </p>
                <p>
                  <strong>{t('settings.labels.expires')}:</strong>{' '}
                  {viewKey.expires_at ? formatDate(viewKey.expires_at) : t('settings.never')}
                </p>
              </div>
              <div className="lg:hidden flex flex-col gap-2 pt-4 border-t">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setDeleteFromDetails(true);
                    openDeleteKeyDialog(viewKey);
                    setViewKey(null);
                  }}
                  className="w-full sm:w-auto"
                >
                  {t("admin.actions.delete")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setViewKey(null)}
                  className="w-full sm:w-auto"
                >
                  {t("settings.actions.cancel")}
                </Button>
              </div>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
