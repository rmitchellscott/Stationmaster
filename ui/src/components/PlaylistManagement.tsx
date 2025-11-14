import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/components/AuthProvider";
import { useDeviceEvents } from "@/hooks/useDeviceEvents";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
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
import { getMashupLayoutGrid } from "./MashupLayoutGrid";
import { mashupService } from "@/services/mashupService";
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
  GripVertical,
  Moon,
  Layers,
  CircleMinus,
} from "lucide-react";
import { Device, isDeviceCurrentlySleeping } from "@/utils/deviceHelpers";

interface PluginInstance {
  id: string;
  user_id: string;
  plugin_id: string;
  name: string;
  settings: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  needs_config_update: boolean;
  last_schema_version: number;
  plugin: {
    id: string;
    name: string;
    type: string;
    description: string;
    status: string; // "available", "unavailable", "error"
  };
}

interface PlaylistItem {
  id: string;
  playlist_id: string;
  plugin_instance_id: string;
  order_index: number;
  is_visible: boolean;
  importance: boolean;
  duration_override?: number;
  created_at: string;
  updated_at: string;
  plugin_instance?: PluginInstance;
  schedules?: any[];
  is_sleep_mode?: boolean; // Virtual field for sleep mode items
  sleep_schedule_text?: string; // Schedule text for sleep mode items
  skip_display?: boolean; // Skip display flag from TRMNL_SKIP_DISPLAY
}

interface PlaylistManagementProps {
  selectedDeviceId: string;
  devices: Device[];
  onUpdate?: () => void;
}

// Helper functions
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

// Helper function to check if an item is currently active based on schedules
const isItemCurrentlyActive = (item: PlaylistItem, userTimezone: string): boolean => {
  if (!item.is_visible) return false;
  if (!item.schedules || item.schedules.length === 0) return true; // No schedules means always active

  const now = new Date();

  // Get current day in user's timezone (not browser timezone)
  const currentDayInTz = new Intl.DateTimeFormat('en-US', {
    timeZone: userTimezone,
    weekday: 'short'
  }).format(now);
  const dayMap: Record<string, number> = {
    'Sun': 0, 'Mon': 1, 'Tue': 2, 'Wed': 3, 'Thu': 4, 'Fri': 5, 'Sat': 6
  };
  const currentDay = dayMap[currentDayInTz] ?? 0;

  const currentTime = now.toLocaleTimeString('en-US', {
    hour12: false,
    timeZone: userTimezone,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });

  return item.schedules.some(schedule => {
    if (!schedule.is_active) return false;

    // Check day mask
    const dayBit = 1 << currentDay;
    const dayMatch = (schedule.day_mask & dayBit) !== 0;
    if (!dayMatch) return false;

    // Check time range
    if (schedule.end_time < schedule.start_time) {
      // Overnight schedule
      return currentTime >= schedule.start_time || currentTime <= schedule.end_time;
    } else {
      // Normal schedule
      return currentTime >= schedule.start_time && currentTime <= schedule.end_time;
    }
  });
};

const formatScheduleSummary = (schedules: any[], userTimezone?: string): string => {
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

  // Convert schedule times to user's timezone for display
  // Handles both old UTC schedules and new timezone-aware schedules
  const convertScheduleTimeToLocal = (timeStr: string, scheduleTimezone: string): string => {
    // If schedule is already in user's timezone, no conversion needed
    if (scheduleTimezone && scheduleTimezone === userTimezone) {
      return timeStr.substring(0, 5); // Just return HH:MM
    }

    // Convert from schedule's timezone to user's timezone
    const today = new Date().toISOString().split('T')[0];
    const scheduleDateTime = scheduleTimezone === "UTC"
      ? `${today}T${timeStr}Z`
      : `${today}T${timeStr}`;
    const scheduleDate = new Date(scheduleDateTime);

    const localTime = scheduleDate.toLocaleTimeString('en-GB', {
      timeZone: userTimezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
      hour12: false,
      hour: '2-digit',
      minute: '2-digit'
    });

    return localTime;
  };

  const startTimeLocal = schedule.start_time ? convertScheduleTimeToLocal(schedule.start_time, schedule.timezone) : "09:00";
  const endTimeLocal = schedule.end_time ? convertScheduleTimeToLocal(schedule.end_time, schedule.timezone) : "17:00";
  
  // Convert to 12-hour format for display
  const formatTime12 = (time24: string) => {
    const [hours, minutes] = time24.split(':');
    const hour = parseInt(hours);
    const ampm = hour >= 12 ? 'PM' : 'AM';
    const hour12 = hour % 12 || 12;
    return `${hour12}:${minutes}${ampm}`;
  };

  const timeRange = `${formatTime12(startTimeLocal)} - ${formatTime12(endTimeLocal)}`;
  
  // Get timezone abbreviation for display
  const timezoneAbbr = userTimezone ? 
    new Intl.DateTimeFormat('en', { timeZoneName: 'short', timeZone: userTimezone })
      .formatToParts(new Date())
      .find(part => part.type === 'timeZoneName')?.value || '' : '';
  
  const timeWithTz = timezoneAbbr ? `${timeRange} ${timezoneAbbr}` : timeRange;
  
  return activSchedules.length > 1 ? 
    `${dayText} ${timeWithTz} +${activSchedules.length - 1} more` :
    `${dayText} ${timeWithTz}`;
};

// Create a virtual sleep mode item for display in the playlist
const createSleepModeItem = (sleepConfig: any, userTimezone: string): PlaylistItem => {
  const formatTime12 = (time24: string) => {
    if (!time24) return "";
    const [hours, minutes] = time24.split(':');
    const hour = parseInt(hours);
    const ampm = hour >= 12 ? 'PM' : 'AM';
    const hour12 = hour % 12 || 12;
    return `${hour12}:${minutes}${ampm}`;
  };

  const startTime = formatTime12(sleepConfig.start_time);
  const endTime = formatTime12(sleepConfig.end_time);
  const scheduleText = `${startTime} - ${endTime}`;
  
  // Get timezone abbreviation for display
  const timezoneAbbr = userTimezone ? 
    new Intl.DateTimeFormat('en', { timeZoneName: 'short', timeZone: userTimezone })
      .formatToParts(new Date())
      .find(part => part.type === 'timeZoneName')?.value || '' : '';
  
  const fullScheduleText = timezoneAbbr ? `${scheduleText} ${timezoneAbbr}` : scheduleText;

  return {
    id: 'virtual-sleep-mode',
    playlist_id: '',
    plugin_instance_id: '',
    order_index: -1, 
    is_visible: true,
    importance: false,
    created_at: '',
    updated_at: '',
    is_sleep_mode: true,
    sleep_schedule_text: fullScheduleText, 
    plugin_instance: {
      id: 'sleep-mode',
      user_id: '',
      plugin_id: '',
      name: 'Sleep Mode',
      settings: '',
      is_active: true,
      created_at: '',
      updated_at: '',
      plugin_definition: {
        id: 'sleep-mode',
        name: 'Device Setting', 
        type: 'system',
        description: 'Device sleep schedule',
      }
    }
  };
};

interface SortableTableRowProps {
  item: PlaylistItem;
  index: number;
  devices: Device[];
  selectedDeviceId: string;
  allPlaylistItems: PlaylistItem[];
  currentlyShowingItemId: string | null;
  currentItemChanged: boolean;
  userTimezone: string;
  timeTravelMode: boolean;
  timeTravelActiveItems: any[];
  deviceEvents: any; // DeviceEventsHookResult type
  playlistMashupLayoutCache: Record<string, string>;
  onScheduleClick: (item: PlaylistItem) => void;
  onVisibilityToggle: (item: PlaylistItem) => void;
  onRemove: (itemId: string) => void;
}

function SortableTableRow({ 
  item, 
  index, 
  devices, 
  selectedDeviceId, 
  allPlaylistItems,
  currentlyShowingItemId,
  currentItemChanged,
  userTimezone,
  timeTravelMode,
  timeTravelActiveItems,
  deviceEvents,
  playlistMashupLayoutCache,
  onScheduleClick, 
  onVisibilityToggle, 
  onRemove,
}: SortableTableRowProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    WebkitUserSelect: 'none' as const,
    userSelect: 'none' as const,
    WebkitTouchCallout: 'none' as const,
  };

  // Check if sleep screen is showing (using hybrid approach)
  const selectedDevice = devices.find(d => d.id === selectedDeviceId);
  const sseCurrentlySleeping = deviceEvents?.sleepConfig?.currently_sleeping;
  const fallbackCurrentlySleeping = selectedDevice ? isDeviceCurrentlySleeping(selectedDevice, userTimezone) : false;
  const currentlySleeping = sseCurrentlySleeping !== undefined ? sseCurrentlySleeping : fallbackCurrentlySleeping;
  
  // Sleep screen is only "active" (showing "Now Showing") when:
  // 1. Device is currently sleeping AND sleep screen is enabled
  // 2. AND the sleep screen is actually being served (from SSE data)
  const isSleepScreenActive = !timeTravelMode && currentlySleeping && selectedDevice?.sleep_show_screen && deviceEvents?.sleepConfig?.sleep_screen_served;
  
  // Use SSE-provided currently showing item ID for real-time updates (only in live mode)
  // Logic: "Now Showing" appears on whatever is actually displayed on the device
  const isCurrentlyShowing = !timeTravelMode && (
    item.is_sleep_mode 
      ? isSleepScreenActive  // Sleep mode shows "Now Showing" only when sleep screen is actually displayed
      : (currentlyShowingItemId === item.id && !isSleepScreenActive)  // Regular items show "Now Showing" when they're current AND sleep screen is not active
  );
  const isChangingToCurrent = isCurrentlyShowing && currentItemChanged;
  
  // Check if item is currently active based on schedules
  // In time travel mode, check if item is in the time travel active items list
  const isActive = timeTravelMode 
    ? timeTravelActiveItems.some(activeItem => activeItem.id === item.id)
    : isItemCurrentlyActive(item, userTimezone);


  // Create styling classes
  const animationClasses = [
    'sortable-row',
    'transition-transform duration-300', // Only transition transform for drag-and-drop
    isDragging ? 'relative z-50' : '',
    !item.is_visible ? 'opacity-60' : '',
    !isActive && item.is_visible ? 'opacity-75' : '',
    item.is_sleep_mode ? 'bg-muted/50' : '', // Special styling for sleep mode
  ].filter(Boolean).join(' ');

  return (
    <TableRow 
      ref={setNodeRef} 
      style={style} 
      className={`${animationClasses} ${
        item.plugin_instance?.plugin?.status === 'unavailable' ? 'opacity-70 bg-muted/30' : 
        item.plugin_instance?.needs_config_update ? 'opacity-60 bg-muted/30' : ''
      }`}
    >
      <TableCell>
        <div className="flex items-center gap-2">
          {item.is_sleep_mode ? (
            <div className="p-2 text-muted-foreground min-h-[44px] min-w-[44px] flex items-center justify-center">
              <Moon className="h-5 w-5" />
            </div>
          ) : (
            <button
              className="cursor-grab active:cursor-grabbing p-2 text-muted-foreground hover:text-foreground touch-manipulation min-h-[44px] min-w-[44px] flex items-center justify-center"
              style={{
                touchAction: 'none',
                WebkitUserSelect: 'none',
                userSelect: 'none',
                WebkitTouchCallout: 'none',
              }}
              {...attributes}
              {...listeners}
            >
              <GripVertical className="h-5 w-5" />
            </button>
          )}
          <div className="text-sm text-muted-foreground">
            {item.is_sleep_mode ? '' : `#${index + 1}`}
          </div>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-between h-full min-h-[3rem]">
          <div>
            <div className="font-medium">
              {item.plugin_instance?.name || "Unnamed Instance"}
            </div>
            <div className="text-sm text-muted-foreground">
              {item.plugin_instance?.plugin_definition?.is_mashup === true
                ? "Mashup" 
                : item.plugin_instance?.plugin_definition?.name || "Unknown Plugin"}
            </div>
          </div>
          {item.plugin_instance?.plugin_definition?.is_mashup === true && item.plugin_instance?.id && playlistMashupLayoutCache[item.plugin_instance.id] && (
            <div className="flex items-center">
              {getMashupLayoutGrid(playlistMashupLayoutCache[item.plugin_instance.id], 'tiny', 'subtle')}
            </div>
          )}
        </div>
        <div className="text-xs md:hidden mt-1">
            <span className="flex items-center gap-1">
              {item.plugin_instance?.plugin?.status === 'unavailable' ? (
                <Badge variant="secondary" className="text-xs !opacity-100 relative z-10">
                  Unavailable
                </Badge>
              ) : item.plugin_instance?.needs_config_update ? (
                <Badge 
                  variant="destructive" 
                  className="text-xs cursor-pointer hover:bg-destructive/80 !opacity-100 relative z-10"
                  onClick={() => {
                    // Navigate to plugin management and open edit for this instance
                    navigate(`/?tab=plugins&subtab=instances&edit=${item.plugin_instance_id}`);
                  }}
                >
                  Update Config
                </Badge>
              ) : !item.is_visible ? (
                <Badge variant="secondary" className="text-xs !opacity-100 relative z-10">
                  <EyeOff className="h-3 w-3 mr-1" />
                  Hidden
                </Badge>
              ) : item.skip_display ? (
                <Badge variant="secondary" className="text-xs !opacity-100 relative z-10">
                  <CircleMinus className="h-3 w-3 mr-1" />
                  Skipped
                </Badge>
              ) : (
                <>
                  {isCurrentlyShowing ? (
                    <PlayCircle className="h-3 w-3" />
                  ) : isActive ? (
                    <Eye className="h-3 w-3" />
                  ) : (
                    <EyeOff className="h-3 w-3" />
                  )}
                  <span className="text-muted-foreground">
                    {isCurrentlyShowing ? "Now Showing" : isActive ? "Active" : "Scheduled"} • {item.importance ? "Important" : "Normal"}
                  </span>
                </>
              )}
            </span>
          </div>
      </TableCell>
      <TableCell className="hidden md:table-cell">
        {item.plugin_instance?.plugin?.status === 'unavailable' ? (
          // Unavailable takes highest priority
          <Badge variant="secondary" className="!opacity-100 relative z-10">
            Unavailable
          </Badge>
        ) : item.plugin_instance?.needs_config_update ? (
          // Config update takes second priority
          <Badge 
            variant="destructive"
            className="cursor-pointer hover:bg-destructive/80 !opacity-100 relative z-10"
            onClick={() => {
              // Navigate to plugin management and open edit for this instance
              navigate(`/?tab=plugins&subtab=instances&edit=${item.plugin_instance_id}`);
            }}
          >
            Update Config
          </Badge>
        ) : !item.is_visible ? (
          // Hidden takes third priority
          <Badge variant="secondary" className="!opacity-100 relative z-10">
            <EyeOff className="h-3 w-3 mr-1" />
            Hidden
          </Badge>
        ) : item.skip_display ? (
          // Skip display takes fourth priority
          <Badge variant="secondary" className="!opacity-100 relative z-10">
            <CircleMinus className="h-3 w-3 mr-1" />
            Skipped
          </Badge>
        ) : item.is_sleep_mode ? (
          // Special status logic for sleep mode items
          timeTravelMode ? (
            // In time travel mode, just show if sleep would be active at that time
            isActive ? (
              <Badge variant="outline">
                <Moon className="h-3 w-3 mr-1" />
                Active
              </Badge>
            ) : (
              <Badge variant="secondary">
                <Clock className="h-3 w-3 mr-1" />
                Scheduled
              </Badge>
            )
          ) : (
            // In live mode, use hybrid approach for current status
            (() => {
              if (currentlySleeping) {
                // Only show "Now Showing" if sleep screen is actually being served
                if (selectedDevice?.sleep_show_screen && deviceEvents?.sleepConfig?.sleep_screen_served) {
                  return (
                    <Badge variant="default">
                      <PlayCircle className="h-3 w-3 mr-1" />
                      Now Showing
                    </Badge>
                  );
                } else {
                  // Sleep mode is active but sleep screen not yet served - show "Active"
                  return (
                    <Badge variant="outline">
                      <Moon className="h-3 w-3 mr-1" />
                      Active
                    </Badge>
                  );
                }
              } else {
                return (
                  <Badge variant="secondary">
                    <Clock className="h-3 w-3 mr-1" />
                    Scheduled
                  </Badge>
                );
              }
            })()
          )
        ) : (
          // Regular playlist item status logic
          isCurrentlyShowing ? (
            <Badge variant="default">
              <PlayCircle className="h-3 w-3 mr-1" />
              Now Showing
            </Badge>
          ) : isActive ? (
            <Badge variant="outline">
              <Eye className="h-3 w-3 mr-1" />
              Active
            </Badge>
          ) : (
            <Badge variant="secondary">
              <EyeOff className="h-3 w-3 mr-1" />
              Scheduled
            </Badge>
          )
        )}
      </TableCell>
      <TableCell className="hidden lg:table-cell">
        {item.is_sleep_mode ? (
          // Empty for sleep mode items
          <></>
        ) : (
          item.importance ? (
            <Badge variant="default">
              <Star className="h-3 w-3 mr-1" />
              Important
            </Badge>
          ) : (
            <Badge variant="outline">Normal</Badge>
          )
        )}
      </TableCell>
      <TableCell className="hidden lg:table-cell">
        {item.is_sleep_mode ? (
          // Empty for sleep mode items
          <></>
        ) : (
          item.duration_override ? formatDuration(item.duration_override) : "Default"
        )}
      </TableCell>
      <TableCell className="hidden lg:table-cell">
        <div className="text-sm">
          {item.is_sleep_mode 
            ? item.sleep_schedule_text || 'Always active'
            : formatScheduleSummary(item.schedules || [], userTimezone)}
        </div>
      </TableCell>
      <TableCell className="text-right">
        {item.is_sleep_mode ? (
          // No actions for sleep mode items
          <div className="flex items-center gap-2 justify-end">
            {/* Empty - no actions for sleep mode */}
          </div>
        ) : (
          <div className="flex items-center gap-2 justify-end">
            <Button
              size="sm"
              variant="outline"
              onClick={() => onScheduleClick(item)}
              title="Manage schedules & settings"
            >
              <Calendar className="h-4 w-4" />
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => onVisibilityToggle(item)}
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
              onClick={() => onRemove(item.id)}
              title="Remove from playlist"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        )}
      </TableCell>
    </TableRow>
  );
}

export function PlaylistManagement({ selectedDeviceId, devices, onUpdate }: PlaylistManagementProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [playlistItems, setPlaylistItems] = useState<PlaylistItem[]>([]);
  const [playlistMashupLayoutCache, setPlaylistMashupLayoutCache] = useState<Record<string, string>>({});
  const [selectorMashupLayoutCache, setSelectorMashupLayoutCache] = useState<Record<string, string>>({});
  const [selectorLayoutsLoading, setSelectorLayoutsLoading] = useState(false);
  
  // Use SSE hook for real-time device events
  const deviceEvents = useDeviceEvents(selectedDeviceId);

  // Time travel state
  const [timeTravelMode, setTimeTravelMode] = useState(false);
  const [timeTravelDate, setTimeTravelDate] = useState<string>("");
  const [timeTravelTime, setTimeTravelTime] = useState<string>("");
  const [timeTravelActiveItems, setTimeTravelActiveItems] = useState<any[]>([]);
  const [timeTravelCurrentIndex, setTimeTravelCurrentIndex] = useState<number | null>(null);

  // Get user's timezone or fall back to browser timezone
  const getUserTimezone = () => {
    return user?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone;
  };

  // Fetch active items for a specific time (time travel)
  const fetchActiveItemsForTime = async (targetTime: Date) => {
    if (!selectedDeviceId) return;

    try {
      const response = await fetch(
        `/api/devices/${selectedDeviceId}/active-items?at=${targetTime.toISOString()}`,
        { credentials: "include" }
      );

      if (response.ok) {
        const data = await response.json();
        setTimeTravelActiveItems(data.active_items || []);
        setTimeTravelCurrentIndex(data.current_index);
      } else {
        console.error("Failed to fetch active items for time:", response.status);
        setTimeTravelActiveItems([]);
        setTimeTravelCurrentIndex(null);
      }
    } catch (error) {
      console.error("Error fetching active items for time:", error);
      setTimeTravelActiveItems([]);
      setTimeTravelCurrentIndex(null);
    }
  };

  // Time travel helper functions
  const enableTimeTravel = () => {
    const now = new Date();
    setTimeTravelDate(now.toISOString().split('T')[0]); // YYYY-MM-DD
    setTimeTravelTime(now.toTimeString().substring(0, 5)); // HH:MM
    setTimeTravelMode(true);
    fetchActiveItemsForTime(now);
  };

  const disableTimeTravel = () => {
    setTimeTravelMode(false);
    setTimeTravelDate("");
    setTimeTravelTime("");
    setTimeTravelActiveItems([]);
    setTimeTravelCurrentIndex(null);
  };

  const handleTimeTravelChange = () => {
    if (!timeTravelDate || !timeTravelTime) return;
    
    // Combine date and time and treat as local time in user's timezone
    const targetDateTime = new Date(`${timeTravelDate}T${timeTravelTime}:00`);
    fetchActiveItemsForTime(targetDateTime);
  };

  // Get the currently showing item ID (either from SSE or time travel)
  const getCurrentlyShowingItemId = () => {
    if (timeTravelMode) {
      if (timeTravelCurrentIndex !== null && timeTravelCurrentIndex >= 0 && timeTravelCurrentIndex < timeTravelActiveItems.length) {
        return timeTravelActiveItems[timeTravelCurrentIndex]?.id || null;
      }
      return null;
    }
    return deviceEvents.currentItem?.id || null;
  };

  // Check if device would be sleeping at the time travel time
  const isDeviceSleepingInTimeTravel = () => {
    if (!timeTravelMode || !timeTravelDate || !timeTravelTime) return false;
    
    const selectedDevice = devices.find(d => d.id === selectedDeviceId);
    if (!selectedDevice?.sleep_enabled) return false;
    
    // Parse the target time
    const targetDateTime = new Date(`${timeTravelDate}T${timeTravelTime}:00`);
    const userTimezone = getUserTimezone();
    
    // Get time in user's timezone
    const timeInTz = targetDateTime.toLocaleTimeString('en-US', { 
      hour12: false, 
      timeZone: userTimezone,
      hour: '2-digit',
      minute: '2-digit'
    });
    
    const [targetHours, targetMinutes] = timeInTz.split(':').map(Number);
    const targetTimeMinutes = targetHours * 60 + targetMinutes;
    
    // Parse sleep times
    const [startHours, startMinutes] = (selectedDevice.sleep_start_time || '22:00').split(':').map(Number);
    const [endHours, endMinutes] = (selectedDevice.sleep_end_time || '06:00').split(':').map(Number);
    
    const sleepStartMinutes = startHours * 60 + startMinutes;
    const sleepEndMinutes = endHours * 60 + endMinutes;
    
    // Handle overnight sleep periods
    if (sleepStartMinutes > sleepEndMinutes) {
      return targetTimeMinutes >= sleepStartMinutes || targetTimeMinutes <= sleepEndMinutes;
    } else {
      return targetTimeMinutes >= sleepStartMinutes && targetTimeMinutes <= sleepEndMinutes;
    }
  };

  // Get display items including sleep mode item when appropriate
  const getDisplayItems = (): PlaylistItem[] => {
    const sortedItems = [...playlistItems].sort((a, b) => a.order_index - b.order_index);
    
    // Always check the actual selected device for sleep configuration
    const selectedDevice = devices.find(d => d.id === selectedDeviceId);
    
    if (!selectedDevice?.sleep_enabled) {
      return sortedItems;
    }

    let sleepConfig;
    
    if (timeTravelMode) {
      // In time travel mode, calculate sleep status for specific time
      sleepConfig = {
        enabled: selectedDevice.sleep_enabled,
        start_time: selectedDevice.sleep_start_time || '22:00',
        end_time: selectedDevice.sleep_end_time || '06:00',
        show_screen: selectedDevice.sleep_show_screen ?? true,
        currently_sleeping: isDeviceSleepingInTimeTravel(),
      };
    } else {
      // In live mode, use device settings for config, SSE for real-time status (with fallback)
      const sseCurrentlySleeping = deviceEvents.sleepConfig?.currently_sleeping;
      const fallbackCurrentlySleeping = isDeviceCurrentlySleeping(selectedDevice, getUserTimezone());
      
      sleepConfig = {
        enabled: selectedDevice.sleep_enabled,
        start_time: selectedDevice.sleep_start_time || '22:00',
        end_time: selectedDevice.sleep_end_time || '06:00',
        show_screen: selectedDevice.sleep_show_screen ?? true,
        // Use SSE data when available, fallback to local calculation
        currently_sleeping: sseCurrentlySleeping !== undefined ? sseCurrentlySleeping : fallbackCurrentlySleeping,
      };
    }

    // Create sleep mode item
    const sleepModeItem = createSleepModeItem(sleepConfig, getUserTimezone());
    
    // Add sleep mode item at the beginning
    return [sleepModeItem, ...sortedItems];
  };

  // Convert schedule time from its stored timezone to user's timezone for display
  // Handles both old UTC schedules and new timezone-aware schedules
  const convertScheduleTimeToLocal = (timeStr: string, scheduleTimezone: string): string => {
    const userTimezone = getUserTimezone();

    // If schedule is already in user's timezone, no conversion needed
    if (scheduleTimezone && scheduleTimezone === userTimezone) {
      return timeStr.substring(0, 5); // Just return HH:MM
    }

    // Convert from schedule's timezone to user's timezone
    const today = new Date().toISOString().split('T')[0];
    const scheduleDateTime = scheduleTimezone === "UTC"
      ? `${today}T${timeStr}Z`
      : `${today}T${timeStr}`;
    const scheduleDate = new Date(scheduleDateTime);

    const localTime = scheduleDate.toLocaleTimeString('en-GB', {
      timeZone: userTimezone,
      hour12: false,
      hour: '2-digit',
      minute: '2-digit'
    });

    return localTime; // Returns HH:MM
  };
  const [pluginInstances, setPluginInstances] = useState<PluginInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Add item dialog
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [selectedPluginInstance, setSelectedPluginInstance] = useState<PluginInstance | null>(null);
  const [addLoading, setAddLoading] = useState(false);

  // Edit item state (now used in schedule dialog)
  const [editImportance, setEditImportance] = useState<boolean>(false);
  const [editDurationOverride, setEditDurationOverride] = useState<string>("");
  const [editDurationMode, setEditDurationMode] = useState<'default' | 'custom'>('default');

  // Delete confirmation dialog
  const [deleteItemDialog, setDeleteItemDialog] = useState<{
    isOpen: boolean;
    item: PlaylistItem | null;
  }>({ isOpen: false, item: null });

  // Schedule management dialog
  const [showScheduleDialog, setShowScheduleDialog] = useState(false);
  const [scheduleItem, setScheduleItem] = useState<PlaylistItem | null>(null);
  const [schedules, setSchedules] = useState<any[]>([]);
  const [originalSchedules, setOriginalSchedules] = useState<any[]>([]);
  const [scheduleLoading, setScheduleLoading] = useState(false);

  // Drag and drop sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // Minimum drag distance before activation
      }
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 250, // Delay to differentiate from scroll
        tolerance: 5, // Pixel tolerance for touch
      }
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const reorderPlaylistItems = async (newOrder: PlaylistItem[]): Promise<boolean> => {
    try {
      setError(null);
      
      // Get the default playlist for this device
      const playlistResponse = await fetch(`/api/playlists?device_id=${selectedDeviceId}`, {
        credentials: "include",
      });
      
      if (playlistResponse.ok) {
        const playlistData = await playlistResponse.json();
        const defaultPlaylist = playlistData.playlists?.find((p: any) => p.is_default);
        
        if (defaultPlaylist) {
          // Extract item IDs in the new order
          const itemIds = newOrder.map(item => item.id);
          
          const response = await fetch(`/api/playlists/${defaultPlaylist.id}/reorder-array`, {
            method: "PUT",
            headers: {
              "Content-Type": "application/json",
            },
            credentials: "include",
            body: JSON.stringify({
              item_ids: itemIds,
            }),
          });

          if (response.ok) {
            // Update order_index values in the newOrder array to match new positions
            const updatedOrder = newOrder.map((item, index) => ({
              ...item,
              order_index: index + 1
            }));
            
            // Update local state to reflect new order with updated order_index
            setPlaylistItems(updatedOrder);
            return true;
          } else {
            const errorData = await response.json();
            setError(errorData.error || "Failed to reorder playlist items");
            return false;
          }
        }
      }
      return false;
    } catch (error) {
      setError("Network error occurred");
      return false;
    }
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;

    if (active.id !== over?.id) {
      // Get current sorted items
      const sortedItems = [...playlistItems].sort((a, b) => a.order_index - b.order_index);
      const oldIndex = sortedItems.findIndex((item) => item.id === active.id);
      const newIndex = sortedItems.findIndex((item) => item.id === over!.id);

      if (oldIndex !== -1 && newIndex !== -1) {
        const newOrder = arrayMove(sortedItems, oldIndex, newIndex);
        
        // Optimistically update the UI immediately
        setPlaylistItems(newOrder.map((item, index) => ({
          ...item,
          order_index: index + 1
        })));
        
        // Update the order in the backend
        const success = await reorderPlaylistItems(newOrder);
        
        if (success) {
          // Backend update succeeded, trigger any parent updates after a brief delay
          setTimeout(() => {
            onUpdate?.();
          }, 100);
        } else {
          // Backend update failed, revert to original order
          setPlaylistItems(sortedItems);
        }
      }
    }
  };

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
            const items = itemsData.items || [];
            setPlaylistItems(items);
            
            // Load layouts for mashup items
            const mashupItems = items.filter((item: PlaylistItem) => 
              item.plugin_instance?.plugin_definition?.is_mashup === true
            );
            
            // Load layouts in parallel for all mashup items
            if (mashupItems.length > 0) {
              mashupItems.forEach((item: PlaylistItem) => {
                if (item.plugin_instance?.id) {
                  loadPlaylistMashupLayout(item.plugin_instance.id);
                }
              });
            }
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

  const loadPlaylistMashupLayout = async (instanceId: string): Promise<string | null> => {
    // Return cached layout if available
    if (playlistMashupLayoutCache[instanceId]) {
      return playlistMashupLayoutCache[instanceId];
    }

    try {
      const mashupData = await mashupService.getChildren(instanceId);
      const layout = mashupData.layout;
      
      // Cache the layout
      setPlaylistMashupLayoutCache(prev => ({ ...prev, [instanceId]: layout }));
      
      return layout;
    } catch (error) {
      console.error('Failed to load mashup layout for playlist:', error);
      return null;
    }
  };

  const loadSelectorMashupLayouts = async () => {
    setSelectorLayoutsLoading(true);
    
    const availableInstances = getAvailablePluginInstances();
    console.log('Available instances for selector:', availableInstances);
    
    // Debug: Compare data for instances that should be mashups
    availableInstances.forEach(instance => {
      if (instance.name.toLowerCase().includes('f1') || instance.name.toLowerCase().includes('formula')) {
        console.group(`🔍 DEBUG: Analyzing ${instance.name}`);
        
        const selectorIsMashup = instance.plugin_definition?.is_mashup;
        console.log('Selector data - is_mashup:', selectorIsMashup);
        console.log('Selector data - full plugin_definition:', instance.plugin_definition);
        
        // Compare with playlist version if it exists
        const playlistItem = playlistItems.find(pi => pi.plugin_instance_id === instance.id);
        if (playlistItem) {
          const playlistIsMashup = playlistItem.plugin_instance?.plugin_definition?.is_mashup;
          console.log('Playlist data - is_mashup:', playlistIsMashup);
          console.log('Playlist data - full plugin_definition:', playlistItem.plugin_instance?.plugin_definition);
          
          if (selectorIsMashup !== playlistIsMashup) {
            console.error('❌ DATA MISMATCH FOUND!');
            console.log('Selector has:', selectorIsMashup);
            console.log('Playlist has:', playlistIsMashup);
          }
        }
        
        console.groupEnd();
      }
    });

    const mashupInstances = availableInstances.filter(instance => {
      return instance.plugin_definition?.is_mashup === true;
    });
    
    console.log('Detected mashup instances:', mashupInstances);

    for (const instance of mashupInstances) {
      // Skip if already cached
      if (selectorMashupLayoutCache[instance.id]) {
        console.log(`Layout already cached for ${instance.name}`);
        continue;
      }

      try {
        console.log(`Loading layout for mashup instance: ${instance.name}`);
        const mashupData = await mashupService.getChildren(instance.id);
        const layout = mashupData.layout;
        console.log(`Loaded layout for ${instance.name}:`, layout);
        
        // Cache the layout
        setSelectorMashupLayoutCache(prev => ({ ...prev, [instance.id]: layout }));
      } catch (error) {
        console.error(`Failed to load mashup layout for selector ${instance.name}:`, error);
      }
    }
    
    setSelectorLayoutsLoading(false);
  };

  const fetchPluginInstances = async () => {
    try {
      const response = await fetch("/api/plugin-instances", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPluginInstances(data.plugin_instances || []);
      }
    } catch (error) {
      console.error("Failed to fetch user plugins:", error);
    }
  };

  const addPlaylistItem = async () => {
    if (!selectedPluginInstance || !selectedDeviceId) return;

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
              plugin_instance_id: selectedPluginInstance.id,
            }),
          });

          if (response.ok) {
            setShowAddDialog(false);
            setSelectedPluginInstance(null);
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
      console.log(`[removePlaylistItem] Attempting to delete playlist item: ${itemId}`);
      setError(null);
      
      const response = await fetch(`/api/playlists/items/${itemId}`, {
        method: "DELETE",
        credentials: "include",
      });
      
      console.log(`[removePlaylistItem] Response status: ${response.status}`);
      
      if (response.ok) {
        console.log(`[removePlaylistItem] Successfully deleted item: ${itemId}`);
        await fetchPlaylistItems();
        onUpdate?.();
      } else {
        console.error(`[removePlaylistItem] Failed to delete item: ${itemId}, status: ${response.status}`);
        try {
          const errorData = await response.json();
          console.error(`[removePlaylistItem] Error details:`, errorData);
          setError(errorData.error || "Failed to remove item from playlist");
        } catch (parseError) {
          console.error(`[removePlaylistItem] Failed to parse error response:`, parseError);
          setError(`Failed to remove item from playlist (HTTP ${response.status})`);
        }
      }
    } catch (error) {
      console.error(`[removePlaylistItem] Network error:`, error);
      setError("Network error occurred while removing item");
    }
  };

  const toggleItemVisibility = async (item: PlaylistItem) => {
    try {
      setError(null);
      
      // Optimistic UI update - update local state immediately
      const updatedItems = playlistItems.map(playlistItem => 
        playlistItem.id === item.id 
          ? { ...playlistItem, is_visible: !playlistItem.is_visible }
          : playlistItem
      );
      setPlaylistItems(updatedItems);
      
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
      
      if (!response.ok) {
        // Revert optimistic update on failure
        setPlaylistItems(playlistItems);
        const errorData = await response.json();
        setError(errorData.error || "Failed to update item visibility");
      }
      // Note: No need to fetch fresh data on success since we already updated optimistically
    } catch (error) {
      // Revert optimistic update on network error
      setPlaylistItems(playlistItems);
      setError("Network error occurred");
    }
  };

  const openScheduleDialog = async (item: PlaylistItem) => {
    setScheduleItem(item);
    
    // Convert schedule times to user's timezone for display
    const schedulesWithLocalTimes = (item.schedules || []).map(schedule => ({
      ...schedule,
      start_time: convertScheduleTimeToLocal(schedule.start_time, schedule.timezone) + ":00", // Add seconds for UI
      end_time: convertScheduleTimeToLocal(schedule.end_time, schedule.timezone) + ":00", // Add seconds for UI
      is_active: schedule.is_active !== undefined ? schedule.is_active : true, // Ensure is_active is set
    }));
    
    setSchedules(schedulesWithLocalTimes);
    setOriginalSchedules(JSON.parse(JSON.stringify(schedulesWithLocalTimes))); // Deep copy for comparison
    
    // Also load the edit data for importance and duration
    setEditImportance(item.importance);
    setEditDurationOverride(item.duration_override ? item.duration_override.toString() : "");
    setEditDurationMode(item.duration_override ? 'custom' : 'default');
    setShowScheduleDialog(true);
  };

  const hasScheduleChanges = () => {
    if (!scheduleItem) return false;
    
    // Check if importance or duration changed
    const importanceChanged = editImportance !== scheduleItem.importance;
    
    // Check duration changes by comparing the final values that would be saved
    const originalDurationValue = scheduleItem.duration_override;
    let newDurationValue: number | null;
    
    if (editDurationMode === 'default') {
      newDurationValue = null;
    } else {
      const duration = parseInt(editDurationOverride);
      newDurationValue = (!isNaN(duration) && duration > 0) ? duration : null;
    }
    
    const durationChanged = newDurationValue !== originalDurationValue;
    
    // Check if schedules changed
    const schedulesChanged = JSON.stringify(schedules) !== JSON.stringify(originalSchedules);
    
    return importanceChanged || durationChanged || schedulesChanged;
  };

  const saveSchedules = async () => {
    if (!scheduleItem) return;

    // Warn if user doesn't have timezone configured
    if (!user?.timezone) {
      const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      const confirm = window.confirm(
        `Your timezone preference is not set. Times will be interpreted as ${userTimezone}. ` +
        `Would you like to continue? (You can set your timezone in User Settings)`
      );
      if (!confirm) return;
    }

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
        const scheduleData = {
          name: schedule.name || "Unnamed Schedule",
          day_mask: schedule.day_mask,
          start_time: schedule.start_time.substring(0, 8), // Keep as local time HH:MM:SS
          end_time: schedule.end_time.substring(0, 8), // Keep as local time HH:MM:SS
          timezone: getUserTimezone(), // Use user's actual timezone
          is_active: schedule.is_active,
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
      // Check if importance changed or if duration value would change
      const importanceChanged = editImportance !== scheduleItem.importance;
      
      const originalDurationValue = scheduleItem.duration_override;
      let newDurationValue: number | null;
      if (editDurationMode === 'default') {
        newDurationValue = null;
      } else {
        const duration = parseInt(editDurationOverride);
        newDurationValue = (!isNaN(duration) && duration > 0) ? duration : null;
      }
      const durationChanged = newDurationValue !== originalDurationValue;
      
      if (importanceChanged || durationChanged) {
        
        const updateData: any = {
          importance: editImportance,
        };

        // Handle duration override based on mode
        if (editDurationMode === 'default') {
          updateData.duration_override = null;
        } else {
          // Custom mode - validate the input
          const duration = parseInt(editDurationOverride);
          if (!isNaN(duration) && duration > 0) {
            updateData.duration_override = duration;
          } else {
            // Invalid input in custom mode - set to null
            updateData.duration_override = null;
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

      setShowScheduleDialog(false);
      setScheduleItem(null);
      setSchedules([]);
      setOriginalSchedules([]);
      await fetchPlaylistItems(); // Refresh to get updated schedules
      onUpdate?.();
    } catch (error) {
      setError(error instanceof Error ? error.message : "Network error occurred");
    } finally {
      setScheduleLoading(false);
    }
  };

  const getAvailablePluginInstances = () => {
    const usedPluginIds = playlistItems.map(item => item.plugin_instance_id);
    return pluginInstances.filter(plugin => 
      !usedPluginIds.includes(plugin.id) && !plugin.needs_config_update
    );
  };

  useEffect(() => {
    if (selectedDeviceId) {
      fetchPlaylistItems();
      fetchPluginInstances();
    }
  }, [selectedDeviceId]);

  // Debug log when selector mashup cache updates
  useEffect(() => {
    console.log('Selector mashup layout cache updated:', selectorMashupLayoutCache);
  }, [selectorMashupLayoutCache]);


  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error]);

  // Update time travel data when date/time changes
  useEffect(() => {
    if (timeTravelMode && timeTravelDate && timeTravelTime) {
      handleTimeTravelChange();
    }
  }, [timeTravelDate, timeTravelTime, timeTravelMode]);

  // Listen for skip display updates via SSE and refresh playlist items
  useEffect(() => {
    if (deviceEvents.lastEvent?.type === 'playlist_item_skip_updated') {
      // Refresh playlist items when skip display status changes
      fetchPlaylistItems();
    }
  }, [deviceEvents.lastEvent]);


  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}




      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-semibold">
            Playlist Items
            {timeTravelMode && (
              <Badge variant="secondary" className="ml-2">
                Time Travel Mode
              </Badge>
            )}
          </h3>
          <p className="text-muted-foreground">
            {timeTravelMode 
              ? `Viewing playlist state at ${timeTravelDate} ${timeTravelTime}`
              : "Manage content rotation for the selected device"
            }
          </p>
        </div>
        <div className="flex items-center gap-2">
          {timeTravelMode ? (
            <>
              <div className="flex items-center gap-2">
                <Input
                  type="date"
                  value={timeTravelDate}
                  onChange={(e) => setTimeTravelDate(e.target.value)}
                  className="w-auto"
                />
                <Input
                  type="time"
                  value={timeTravelTime}
                  onChange={(e) => setTimeTravelTime(e.target.value)}
                  className="w-auto"
                />
              </div>
              <Button
                variant="default"
                onClick={disableTimeTravel}
              >
                Return to Now
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="outline"
                onClick={enableTimeTravel}
                className="flex items-center gap-2"
              >
                <Clock className="h-4 w-4" />
                Time Travel
              </Button>
              <Button
                onClick={() => {
                  setShowAddDialog(true);
                  loadSelectorMashupLayouts();
                }}
                disabled={getAvailablePluginInstances().length === 0}
              >
                {getAvailablePluginInstances().length === 0 ? "All Plugins Added" : "Add Item"}
              </Button>
            </>
          )}
        </div>
      </div>

      {getAvailablePluginInstances().length === 0 && playlistItems.length === 0 && (
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
              onClick={() => {
                setShowAddDialog(true);
                loadSelectorMashupLayouts();
              }}
              disabled={getAvailablePluginInstances().length === 0}
            >
              {getAvailablePluginInstances().length === 0 ? "All Plugins Added" : "Add Item"}
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
              <DndContext
                sensors={timeTravelMode ? [] : sensors}
                collisionDetection={closestCenter}
                onDragEnd={timeTravelMode ? undefined : handleDragEnd}
                modifiers={[restrictToVerticalAxis]}
              >
                <TableBody>
                  <SortableContext
                    items={playlistItems.sort((a, b) => a.order_index - b.order_index).map(item => item.id)}
                    strategy={verticalListSortingStrategy}
                  >
                    {getDisplayItems().map((item, index) => {
                      // Calculate proper index for non-sleep-mode items
                      const displayIndex = item.is_sleep_mode ? 0 : index;
                      const playlistIndex = item.is_sleep_mode ? -1 : 
                        (getDisplayItems()[0]?.is_sleep_mode ? index - 1 : index);
                      
                      return (
                        <SortableTableRow
                          key={item.id}
                          item={item}
                          index={playlistIndex}
                        devices={devices}
                        selectedDeviceId={selectedDeviceId}
                        allPlaylistItems={playlistItems}
                        currentlyShowingItemId={getCurrentlyShowingItemId()}
                        currentItemChanged={deviceEvents.currentItemChanged}
                        userTimezone={getUserTimezone()}
                        timeTravelMode={timeTravelMode}
                        timeTravelActiveItems={timeTravelActiveItems}
                        deviceEvents={deviceEvents}
                        playlistMashupLayoutCache={playlistMashupLayoutCache}
                        onScheduleClick={openScheduleDialog}
                        onVisibilityToggle={toggleItemVisibility}
                        onRemove={(itemId) => {
                          const item = playlistItems.find(p => p.id === itemId);
                          if (item) {
                            setDeleteItemDialog({ isOpen: true, item });
                          }
                        }}
                        />
                      );
                    })}
                  </SortableContext>
                </TableBody>
              </DndContext>
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
                value={selectedPluginInstance?.id || ""}
                onValueChange={(value) => {
                  const plugin = pluginInstances.find(p => p.id === value);
                  setSelectedPluginInstance(plugin || null);
                }}
              >
                <SelectTrigger className="mt-2">
                  <SelectValue placeholder="Choose a plugin instance..." />
                </SelectTrigger>
                <SelectContent>
                  {selectorLayoutsLoading ? (
                    <div className="py-2 px-3 text-sm text-muted-foreground">
                      Loading layouts...
                    </div>
                  ) : (
                    getAvailablePluginInstances().map((userPlugin) => {
                      const isMashup = userPlugin.plugin_definition?.is_mashup === true;
                      const hasLayout = !!selectorMashupLayoutCache[userPlugin.id];
                      const layoutId = selectorMashupLayoutCache[userPlugin.id];
                      
                      console.log(`Rendering SelectItem for ${userPlugin.name}: isMashup=${isMashup}, hasLayout=${hasLayout}, layoutId=${layoutId}`);
                      
                      return (
                        <SelectItem key={userPlugin.id} value={userPlugin.id}>
                          <div className="flex items-center justify-between w-full">
                            <span>{userPlugin.name}</span>
                            {isMashup && hasLayout && (
                              <div className="ml-2">
                                {getMashupLayoutGrid(layoutId, 'tiny', 'subtle')}
                              </div>
                            )}
                          </div>
                        </SelectItem>
                      );
                    })
                  )}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowAddDialog(false);
                setSelectedPluginInstance(null);
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={addPlaylistItem}
              disabled={!selectedPluginInstance || addLoading}
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
              Configure schedules and settings for "{scheduleItem?.plugin_instance?.name}". 
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
                <div className="space-y-6">
                  <div>
                    <div className="flex items-center justify-between">
                      <Label htmlFor="importance">Important</Label>
                      <Switch
                        id="importance"
                        checked={editImportance}
                        onCheckedChange={setEditImportance}
                      />
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      Only show this screen and other important screens during this schedule.
                    </p>
                  </div>

                  <div>
                    <Label htmlFor="duration-mode">Duration</Label>
                    <div className="mt-2 grid grid-cols-2 gap-3">
                      <Select
                        value={editDurationMode}
                        onValueChange={(value: 'default' | 'custom') => {
                          setEditDurationMode(value);
                          if (value === 'default') {
                            setEditDurationOverride("");
                          }
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="default">Default</SelectItem>
                          <SelectItem value="custom">Custom</SelectItem>
                        </SelectContent>
                      </Select>
                      
                      {editDurationMode === 'custom' && (
                        <Input
                          id="duration-override"
                          type="number"
                          min="60"
                          placeholder="Duration in seconds"
                          value={editDurationOverride}
                          onChange={(e) => setEditDurationOverride(e.target.value)}
                        />
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {editDurationMode === 'default' 
                        ? "Uses plugin default or device refresh rate."
                        : "Override the refresh rate for this item. Takes precedence over plugin-suggested rates."
                      }
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
                setOriginalSchedules([]);
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={saveSchedules}
              disabled={scheduleLoading || !hasScheduleChanges()}
            >
              {scheduleLoading ? "Saving..." : "Save Schedules & Settings"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Playlist Item Confirmation Dialog */}
      <AlertDialog
        open={deleteItemDialog.isOpen}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteItemDialog({ isOpen: false, item: null });
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Remove Playlist Item
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove "{deleteItemDialog.item?.plugin_instance?.name || 'this item'}" from the playlist?
              <br /><br />
              This will:
              <ul className="list-disc list-outside ml-6 mt-2 space-y-1">
                <li>Remove the item from the device's playlist</li>
                <li>Delete all associated schedules</li>
                <li>Stop displaying this content on the device</li>
              </ul>
              <br />
              <strong className="text-destructive">
                This action cannot be undone.
              </strong>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setDeleteItemDialog({ isOpen: false, item: null })}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={async () => {
                if (deleteItemDialog.item) {
                  await removePlaylistItem(deleteItemDialog.item.id);
                  setDeleteItemDialog({ isOpen: false, item: null });
                }
              }}
            >
              Remove Item
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
