import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/components/AuthProvider";
import { useConfig } from "@/components/ConfigProvider";

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
import { Separator } from "@/components/ui/separator";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  Settings,
  User,
  UserCog,
  AlertTriangle,
  ArrowLeft,
  LayoutDashboard,
} from "lucide-react";


export function UserSettingsPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { user, isLoading: userDataLoading, refetchAuth: refetch } = useAuth();
  const { config } = useConfig();

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [timezone, setTimezone] = useState("");
  const [timezoneOpen, setTimezoneOpen] = useState(false);
  const [locale, setLocale] = useState("");
  const [localeOpen, setLocaleOpen] = useState(false);
  
  const [originalValues, setOriginalValues] = useState({
    username: "",
    email: "",
    firstName: "",
    lastName: "",
    timezone: "",
    locale: ""
  });

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

  const getCommonLocales = () => {
    return [
      { value: 'en-US', label: 'English (United States)' },
      { value: 'en-GB', label: 'English (United Kingdom)' },
      { value: 'fr-FR', label: 'French (France)' },
      { value: 'de-DE', label: 'German (Germany)' },
      { value: 'es-ES', label: 'Spanish (Spain)' },
      { value: 'it-IT', label: 'Italian (Italy)' },
      { value: 'pt-BR', label: 'Portuguese (Brazil)' },
      { value: 'ru-RU', label: 'Russian (Russia)' },
      { value: 'ja-JP', label: 'Japanese (Japan)' },
      { value: 'ko-KR', label: 'Korean (South Korea)' },
      { value: 'zh-CN', label: 'Chinese (Simplified)' },
      { value: 'zh-TW', label: 'Chinese (Traditional)' },
      { value: 'ar-SA', label: 'Arabic (Saudi Arabia)' },
      { value: 'hi-IN', label: 'Hindi (India)' },
      { value: 'th-TH', label: 'Thai (Thailand)' },
      { value: 'vi-VN', label: 'Vietnamese (Vietnam)' },
      { value: 'nl-NL', label: 'Dutch (Netherlands)' },
      { value: 'sv-SE', label: 'Swedish (Sweden)' },
      { value: 'no-NO', label: 'Norwegian (Norway)' },
      { value: 'da-DK', label: 'Danish (Denmark)' },
    ];
  };

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [deletePassword, setDeletePassword] = useState("");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  const [deleteAccountDialog, setDeleteAccountDialog] = useState(false);


  useEffect(() => {
    if (user) {
      setUsername(user.username);
      setEmail(user.email);
      setFirstName(user.first_name || "");
      setLastName(user.last_name || "");
      const userTimezone = user.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
      setTimezone(userTimezone);
      const userLocale = user.locale || navigator.language || 'en-US';
      setLocale(userLocale);
      
      setOriginalValues({
        username: user.username,
        email: user.email,
        firstName: user.first_name || "",
        lastName: user.last_name || "",
        timezone: userTimezone,
        locale: userLocale
      });
    }
  }, [user]);

  useEffect(() => {
    const handleLogout = () => {
      setError(null);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setDeletePassword("");
      setDeleteConfirmation("");
      
      setDeleteAccountDialog(false);
      
      setUsername("");
      setEmail("");
      setFirstName("");
      setLastName("");
      setTimezone("");
      setLocale("");
      
      setOriginalValues({
        username: "",
        email: "",
        firstName: "",
        lastName: "",
        timezone: "",
        locale: ""
      });
    };

    window.addEventListener('logout', handleLogout);

    return () => {
      window.removeEventListener('logout', handleLogout);
    };
  }, []);


  const hasChanges = () => {
    return (
      (!config?.oidcEnabled && username !== originalValues.username) ||
      (!config?.oidcEnabled && email !== originalValues.email) ||
      firstName !== originalValues.firstName ||
      lastName !== originalValues.lastName ||
      timezone !== originalValues.timezone ||
      locale !== originalValues.locale
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
          first_name: firstName,
          last_name: lastName,
          timezone,
          locale
        }),
      });

      if (response.ok) {
        setOriginalValues({
          username,
          email,
          firstName,
          lastName,
          timezone,
          locale
        });
        
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


  const canDeleteAccount = deletePassword && deleteConfirmation === t('settings.placeholders.delete_confirm');

  if (userDataLoading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center p-8">{t('settings.loading_states.loading')}</div>
      </div>
    );
  }

  return (
    <>
      <div className="bg-background pt-0 pb-8 px-0 sm:px-8">
        <div className="max-w-6xl mx-0 sm:mx-auto space-y-6">

        {error && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-md p-3 text-destructive">
            {error}
          </div>
        )}

        <Card>
          <CardHeader>
            <div>
              <button 
                onClick={() => navigate(-1)} 
                className="text-sm text-muted-foreground hover:text-foreground inline-flex items-center gap-1 mb-1"
              >
                <ArrowLeft className="h-3 w-3" />
                Back to Dashboard
              </button>
              <CardTitle className="flex items-center gap-2 text-2xl">
                <Settings className="h-5 w-5" />
                {t("settings.title")}
              </CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="profile" className="w-full">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="profile">
                  <User className="h-4 w-4" />
                  <span className="ml-1.5">{t("settings.tabs.profile")}</span>
                </TabsTrigger>
                <TabsTrigger value="account">
                  <UserCog className="h-4 w-4" />
                  <span className="ml-1.5">{t("settings.tabs.account")}</span>
                </TabsTrigger>
              </TabsList>

              <TabsContent value="profile" className="mt-6">
                <div className="space-y-6">
                  <h3 className="text-lg font-semibold">{t("settings.cards.profile_information")}</h3>
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
                    <Label htmlFor="firstName">First Name</Label>
                    <Input
                      id="firstName"
                      value={firstName}
                      onChange={(e) => setFirstName(e.target.value)}
                      className="mt-2"
                      placeholder="Enter your first name"
                    />
                  </div>
                  <div>
                    <Label htmlFor="lastName">Last Name</Label>
                    <Input
                      id="lastName"
                      value={lastName}
                      onChange={(e) => setLastName(e.target.value)}
                      className="mt-2"
                      placeholder="Enter your last name"
                    />
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

                  <div>
                    <Label htmlFor="locale">Locale</Label>
                    <Popover modal={true} open={localeOpen} onOpenChange={setLocaleOpen}>
                      <PopoverTrigger asChild>
                        <Button
                          variant="outline"
                          role="combobox"
                          aria-expanded={localeOpen}
                          className="mt-2 w-full justify-between"
                        >
                          {locale ? 
                            (() => {
                              const localeOption = getCommonLocales().find(l => l.value === locale);
                              return localeOption ? localeOption.label : locale;
                            })() : 
                            "Select locale..."
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
                          <CommandInput placeholder="Search locales..." className="h-8" />
                          <CommandList>
                            <CommandEmpty>No locale found.</CommandEmpty>
                            {getCommonLocales().map((localeOption) => (
                              <CommandItem
                                key={localeOption.value}
                                value={localeOption.label}
                                onSelect={() => {
                                  setLocale(localeOption.value);
                                  setLocaleOpen(false);
                                }}
                                className="cursor-pointer"
                                style={{ pointerEvents: 'auto' }}
                              >
                                <div className="flex items-center justify-between w-full">
                                  <span>{localeOption.label}</span>
                                  <span className="text-sm text-muted-foreground">
                                    {localeOption.value}
                                  </span>
                                </div>
                              </CommandItem>
                            ))}
                          </CommandList>
                        </Command>
                      </PopoverContent>
                    </Popover>
                    <p className="text-xs text-muted-foreground mt-1">
                      Used for date/time formatting and localization.
                    </p>
                  </div>
                </div>

                <div className="flex flex-col sm:flex-row sm:justify-end">
                  <Button onClick={updateProfile} disabled={saving || !hasChanges()} className="w-full sm:w-auto">
                    {saving ? t('settings.loading_states.saving') : t('settings.buttons.save_changes')}
                  </Button>
                </div>
                </div>
              </TabsContent>

              <TabsContent value="account" className="mt-6">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 lg:divide-x lg:divide-border">
                <div className="space-y-4 lg:pr-6">
                  <h3 className="text-lg font-semibold">{t("settings.cards.change_password")}</h3>
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
                </div>

                <div className="space-y-4 flex flex-col h-full lg:pl-6">
                  <Separator className="lg:hidden mb-4" />
                  <h3 className="text-lg font-semibold">{t("settings.cards.delete_account")}</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    {t("settings.messages.delete_warning_intro")}
                  </p>
                  <ul className="text-sm text-muted-foreground list-disc list-outside ml-6 sm:ml-5 space-y-1 mb-4">
                    <li>{t("settings.delete_warnings.api_keys")}</li>
                    <li>{t("settings.delete_warnings.profile")}</li>
                  </ul>
                  <div className="flex flex-col sm:flex-row sm:justify-end md:mt-auto">
                    <Button
                      variant="outline"
                      onClick={openDeleteAccountDialog}
                      className="w-full sm:w-auto"
                    >
                      {t('settings.buttons.delete_my_account')}
                    </Button>
                  </div>
                </div>
                </div>
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>
        </div>
      </div>

      {/* Alert Dialogs */}
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

    </>
  );
}