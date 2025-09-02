import React, { useState, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
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
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { DatePicker } from "@/components/ui/date-picker";
import { TimePicker } from "@/components/ui/time-picker";
import { DateTimePicker } from "@/components/ui/datetime-picker";
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
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Puzzle,
  Edit,
  Trash2,
  Copy,
  Settings as SettingsIcon,
  AlertTriangle,
  CheckCircle,
  RefreshCw,
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  Layers,
  PlayCircle,
} from "lucide-react";
import { PrivatePluginList } from "./PrivatePluginList";
import { PluginPreview } from "./PluginPreview";
import { LiquidEditor } from "./LiquidEditor";
import { PrivatePluginHelp } from "./PrivatePluginHelp";
import { AddPluginDropdown } from "./AddPluginDropdown";
import { MashupSlotGrid } from "./MashupSlotGrid";
import { getMashupLayoutGrid } from "./MashupLayoutGrid";
import { MashupLayout, MashupSlotInfo, AvailablePluginInstance, mashupService } from "@/services/mashupService";

interface Plugin {
  id: string;
  name: string;
  type: string;
  description: string;
  config_schema: string;
  version: string;
  author: string;
  is_active: boolean;
  requires_processing: boolean;
  status: string; // "available", "unavailable", "error"
  data_strategy?: string;
  created_at: string;
  updated_at: string;
}

interface PluginInstance {
  id: string;
  user_id: string;
  plugin_id: string;
  name: string;
  settings: string;
  refresh_interval: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  plugin: Plugin;
  is_used_in_playlists: boolean;
  needs_config_update: boolean;
  last_schema_version: number;
}

interface RefreshRateOption {
  label: string;
  value: number;
  default?: boolean;
}

interface PluginManagementProps {
  selectedDeviceId: string;
  onUpdate?: () => void;
}

type SortColumn = 'name' | 'plugin' | 'status' | 'created';
type SortOrder = 'asc' | 'desc';

interface SortState {
  column: SortColumn;
  order: SortOrder;
}

export function PluginManagement({ selectedDeviceId, onUpdate }: PluginManagementProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [pluginInstances, setPluginInstances] = useState<PluginInstance[]>([]);
  const [mashupLayoutCache, setMashupLayoutCache] = useState<Record<string, string>>({});
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [refreshRateOptions, setRefreshRateOptions] = useState<RefreshRateOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Sorting state with localStorage persistence
  const [sortState, setSortState] = useState<SortState>(() => {
    try {
      const saved = localStorage.getItem('pluginInstanceTableSort');
      if (saved) {
        const parsed = JSON.parse(saved);
        // Validate the parsed data
        if (parsed.column && ['name', 'plugin', 'status', 'created'].includes(parsed.column) &&
            parsed.order && ['asc', 'desc'].includes(parsed.order)) {
          return parsed;
        }
      }
    } catch (e) {
      // Invalid localStorage data, fall back to default
    }
    return { column: 'name', order: 'asc' };
  });

  // Add plugin dialog
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null);
  const [instanceName, setInstanceName] = useState("");
  const [instanceSettings, setInstanceSettings] = useState<Record<string, any>>({});
  const [instanceRefreshRate, setInstanceRefreshRate] = useState<number>(86400); // Default to daily
  const [createLoading, setCreateLoading] = useState(false);
  
  // Add dialog specific alerts
  const [createDialogError, setCreateDialogError] = useState<string | null>(null);

  // Edit plugin dialog
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editPluginInstance, setEditPluginInstance] = useState<PluginInstance | null>(null);
  const [editMashupLayout, setEditMashupLayout] = useState<string | null>(null);
  const [editMashupLayoutLoading, setEditMashupLayoutLoading] = useState(false);
  const [editMashupSlots, setEditMashupSlots] = useState<MashupSlotInfo[]>([]);
  const [editMashupAssignments, setEditMashupAssignments] = useState<Record<string, string>>({});
  const [editOriginalMashupAssignments, setEditOriginalMashupAssignments] = useState<Record<string, string>>({});
  const [editAvailablePlugins, setEditAvailablePlugins] = useState<AvailablePluginInstance[]>([]);
  const [editInstanceName, setEditInstanceName] = useState("");
  const [editInstanceSettings, setEditInstanceSettings] = useState<Record<string, any>>({});
  const [editInstanceRefreshRate, setEditInstanceRefreshRate] = useState<number>(86400);
  const [updateLoading, setUpdateLoading] = useState(false);
  const [forceRefreshLoading, setForceRefreshLoading] = useState(false);
  
  // Edit dialog specific alerts
  const [editDialogError, setEditDialogError] = useState<string | null>(null);
  const [editDialogSuccess, setEditDialogSuccess] = useState<string | null>(null);
  
  // Schema diff state
  const [schemaDiff, setSchemaDiff] = useState<any>(null);
  const [schemaDiffLoading, setSchemaDiffLoading] = useState(false);

  // Delete confirmation dialog
  const [deletePluginDialog, setDeletePluginDialog] = useState<{
    isOpen: boolean;
    plugin: PluginInstance | null;
  }>({ isOpen: false, plugin: null });

  // Private plugin management state
  const [previewingPrivatePlugin, setPreviewingPrivatePlugin] = useState<any | null>(null);

  // Pending edit state for handling navigation timing issues
  const [pendingEditInstanceId, setPendingEditInstanceId] = useState<string | null>(null);


  // Add plugin dialog mode state
  const [creationMode, setCreationMode] = useState<'plugin' | 'mashup'>('plugin');
  const [selectedMashupLayout, setSelectedMashupLayout] = useState<MashupLayout | null>(null);
  
  // Mashup creation state
  const [mashupSlots, setMashupSlots] = useState<MashupSlotInfo[]>([]);
  const [availablePluginInstances, setAvailablePluginInstances] = useState<AvailablePluginInstance[]>([]);
  const [mashupAssignments, setMashupAssignments] = useState<Record<string, string>>({});
  const [mashupDescription, setMashupDescription] = useState("");

  // Get active subtab from URL query parameters
  const activeTab = (searchParams.get('subtab') as 'instances' | 'private') || 'instances';

  // Handle subtab change by updating URL query parameters
  const handleSubTabChange = (subtab: string) => {
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set('subtab', subtab);
    // Ensure main tab is set to plugins
    newSearchParams.set('tab', 'plugins');
    setSearchParams(newSearchParams);
    
    // Refresh plugins when switching to instances tab to catch any newly created private plugins
    if (subtab === 'instances') {
      fetchPlugins();
    }
  };

  // Helper function to generate instance webhook URL
  const generateInstanceWebhookURL = (instanceId: string): string => {
    return `${window.location.origin}/api/webhooks/instance/${instanceId}`;
  };

  // Helper function to copy webhook URL to clipboard
  const copyWebhookUrl = async (instanceId: string) => {
    try {
      const webhookURL = generateInstanceWebhookURL(instanceId);
      await navigator.clipboard.writeText(webhookURL);
      setSuccessMessage("Webhook URL copied to clipboard!");
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (error) {
      setError("Failed to copy webhook URL");
    }
  };

  const fetchPluginInstances = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/plugin-instances", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        const instances = data.plugin_instances || [];
        setPluginInstances(instances);
        
        // Load layouts for mashup instances
        const mashupInstances = instances.filter((instance: PluginInstance) => 
          instance.plugin?.type === 'mashup'
        );
        
        // Load layouts in parallel for all mashup instances
        if (mashupInstances.length > 0) {
          mashupInstances.forEach((instance: PluginInstance) => {
            loadMashupLayoutForTable(instance.id);
          });
        }
      } else {
        setError("Failed to fetch plugin instances");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const fetchPlugins = async () => {
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
  };

  const fetchRefreshRateOptions = async () => {
    try {
      const response = await fetch("/api/plugin-definitions/refresh-rate-options", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setRefreshRateOptions(data.refresh_rate_options || []);
      }
    } catch (error) {
      console.error("Failed to fetch refresh rate options:", error);
    }
  };

  const loadMashupData = async (layout: MashupLayout) => {
    try {
      const [slotsData, availablePlugins] = await Promise.all([
        mashupService.getLayoutSlots(layout.id),
        mashupService.getAvailablePluginInstances()
      ]);
      
      setMashupSlots(slotsData.slots);
      setAvailablePluginInstances(availablePlugins);
      setMashupAssignments({});
    } catch (error) {
      console.error("Failed to load mashup data:", error);
      setCreateDialogError("Failed to load mashup configuration");
    }
  };

  const makeFriendlyError = (errorMessage: string) => {
    return errorMessage
      .replace(/image_url/g, 'Image URL')
      .replace(/endpoint_url/g, 'Endpoint URL')
      .replace(/validation failed: /, '');
  };

  const createMashupInstance = async () => {
    if (!selectedMashupLayout || !instanceName.trim()) {
      setCreateDialogError("Please provide a name for the mashup instance");
      return;
    }

    try {
      setCreateLoading(true);
      setCreateDialogError(null);

      // Create the mashup definition
      const mashupResponse = await mashupService.createMashup({
        name: instanceName.trim(),
        description: mashupDescription.trim() || undefined,
        layout: selectedMashupLayout.id,
      });

      if (!mashupResponse || !mashupResponse.mashup || !mashupResponse.mashup.id) {
        throw new Error('Invalid response from mashup creation - missing mashup ID');
      }

      const mashupDefinitionId = mashupResponse.mashup.id;

      // Create a plugin instance from the definition
      const instanceResponse = await fetch("/api/plugin-instances", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          definition_id: mashupDefinitionId,
          definition_type: "private",
          name: instanceName.trim(),
          settings: {},
          refresh_interval: 3600, // Default 1 hour, will be updated based on children
        }),
      });

      if (!instanceResponse.ok) {
        const errorData = await instanceResponse.json();
        throw new Error(`Failed to create mashup instance: ${errorData.error || instanceResponse.statusText}`);
      }

      const instanceData = await instanceResponse.json();
      if (!instanceData || !instanceData.instance || !instanceData.instance.id) {
        throw new Error('Invalid response from instance creation - missing instance ID');
      }

      const mashupInstanceId = instanceData.instance.id;

      // Assign children if any
      if (Object.keys(mashupAssignments).length > 0) {
        await mashupService.assignChildren(mashupInstanceId, mashupAssignments);
      }

      setSuccessMessage("Mashup created successfully!");
      setShowAddDialog(false);
      resetAddDialogState();
      await fetchPluginInstances();
      onUpdate?.();

    } catch (error) {
      setCreateDialogError(error instanceof Error ? error.message : "Failed to create mashup");
    } finally {
      setCreateLoading(false);
    }
  };

  const createPluginInstance = async () => {
    if (creationMode === 'mashup') {
      return createMashupInstance();
    }

    if (!selectedPlugin || !instanceName.trim()) {
      setCreateDialogError("Please provide a name for the plugin instance");
      return;
    }

    try {
      setCreateLoading(true);
      setCreateDialogError(null);

      const requestBody: any = {
        definition_id: selectedPlugin.id,
        definition_type: selectedPlugin.type, // "system" or "private"
        name: instanceName.trim(),
        settings: instanceSettings,
      };

      // Only include refresh_interval for plugins that require processing
      if (selectedPlugin.requires_processing) {
        requestBody.refresh_interval = instanceRefreshRate;
      }

      const response = await fetch("/api/plugin-instances", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(requestBody),
      });

      if (response.ok) {
        setSuccessMessage("Plugin instance created successfully!");
        setShowAddDialog(false);
        resetAddDialogState();
        await fetchPluginInstances();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        const friendlyError = makeFriendlyError(errorData.details || errorData.error || "Failed to create plugin instance");
        setCreateDialogError(friendlyError);
      }
    } catch (error) {
      setCreateDialogError("Network error occurred");
    } finally {
      setCreateLoading(false);
    }
  };

  // Validation function to check for unassigned mashup slots
  const hasUnassignedSlots = () => {
    if (creationMode !== 'mashup' || !mashupSlots.length) return false;
    
    // Check if any slot is unassigned (empty or "__none__")
    return mashupSlots.some(slot => {
      const assignment = mashupAssignments[slot.position];
      return !assignment || assignment === "__none__";
    });
  };

  const resetAddDialogState = () => {
    setSelectedPlugin(null);
    setInstanceName("");
    setInstanceSettings({});
    setInstanceRefreshRate(86400);
    setCreateDialogError(null);
    setCreationMode('plugin');
    setSelectedMashupLayout(null);
    setMashupSlots([]);
    setAvailablePluginInstances([]);
    setMashupAssignments({});
    setMashupDescription("");
  };

  const hasPluginInstanceChanges = () => {
    if (!editPluginInstance) return false;
    
    // Parse original settings
    let originalSettings = {};
    try {
      originalSettings = editPluginInstance.settings ? JSON.parse(editPluginInstance.settings) : {};
    } catch (e) {
      originalSettings = {};
    }
    
    // Check for refresh rate changes (only for plugins that require processing)
    const hasRefreshRateChanged = editPluginInstance.plugin?.requires_processing && 
      editInstanceRefreshRate !== editPluginInstance.refresh_interval;
    
    return (
      editInstanceName.trim() !== editPluginInstance.name ||
      JSON.stringify(editInstanceSettings) !== JSON.stringify(originalSettings) ||
      hasRefreshRateChanged ||
      hasMashupAssignmentChanges()
    );
  };

  const hasMashupAssignmentChanges = (): boolean => {
    if (!editPluginInstance || editPluginInstance.plugin?.type !== 'mashup') return false;
    
    return JSON.stringify(editMashupAssignments) !== JSON.stringify(editOriginalMashupAssignments);
  };

  const loadMashupLayoutForTable = async (instanceId: string): Promise<string | null> => {
    // Return cached layout if available
    if (mashupLayoutCache[instanceId]) {
      return mashupLayoutCache[instanceId];
    }

    try {
      const mashupData = await mashupService.getChildren(instanceId);
      const layout = mashupData.layout;
      
      // Cache the layout
      setMashupLayoutCache(prev => ({ ...prev, [instanceId]: layout }));
      
      return layout;
    } catch (error) {
      console.error('Failed to load mashup layout for table:', error);
      return null;
    }
  };

  // Fetch schema diff for an instance that needs config updates
  const fetchSchemaDiff = async (instanceId: string) => {
    setSchemaDiffLoading(true);
    setSchemaDiff(null);
    
    try {
      const response = await fetch(`/api/plugin-instances/${instanceId}/schema-diff`, {
        credentials: "include",
      });
      
      if (response.ok) {
        const diff = await response.json();
        setSchemaDiff(diff);
      } else {
        console.error("Failed to fetch schema diff");
      }
    } catch (error) {
      console.error("Error fetching schema diff:", error);
    } finally {
      setSchemaDiffLoading(false);
    }
  };

  const updatePluginInstance = async () => {
    if (!editPluginInstance || !editInstanceName.trim()) {
      setError("Please provide a name for the plugin instance");
      return;
    }

    try {
      setUpdateLoading(true);
      setError(null);

      const requestBody: any = {
        name: editInstanceName.trim(),
        settings: editInstanceSettings,
      };

      // Only include refresh_interval for plugins that require processing
      if (editPluginInstance.plugin?.requires_processing) {
        requestBody.refresh_interval = editInstanceRefreshRate;
      }

      const response = await fetch(`/api/plugin-instances/${editPluginInstance.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(requestBody),
      });

      if (response.ok) {
        // Handle mashup slot assignments if this is a mashup
        let shouldForceRefresh = false;
        if (editPluginInstance.plugin?.type === 'mashup' && hasMashupAssignmentChanges()) {
          try {
            await mashupService.assignChildren(editPluginInstance.id, editMashupAssignments);
            shouldForceRefresh = true; // Only refresh when slots change
          } catch (assignError) {
            console.error("Failed to assign mashup children:", assignError);
            setEditDialogError("Plugin updated but failed to assign slot changes");
            return;
          }
        }

        // Force refresh if slot assignments changed
        if (shouldForceRefresh) {
          try {
            await fetch(`/api/plugin-instances/${editPluginInstance.id}/force-refresh`, {
              method: "POST",
              credentials: "include",
            });
          } catch (refreshError) {
            console.error("Failed to force refresh after slot update:", refreshError);
            // Don't fail the whole operation if refresh fails
          }
        }

        setSuccessMessage("Plugin instance updated successfully!");
        setShowEditDialog(false);
        setEditPluginInstance(null);
        setEditInstanceName("");
        setEditInstanceSettings({});
        setEditInstanceRefreshRate(86400);
        setEditMashupLayout(null);
        setEditMashupSlots([]);
        setEditMashupAssignments({});
        setEditOriginalMashupAssignments({});
        setEditAvailablePlugins([]);
        await fetchPluginInstances();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        const friendlyError = makeFriendlyError(errorData.details || errorData.error || "Failed to update plugin instance");
        setEditDialogError(friendlyError);
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
    } finally {
      setUpdateLoading(false);
    }
  };

  const forceRefreshPluginInstance = async () => {
    if (!editPluginInstance) {
      setEditDialogError("No plugin selected");
      return;
    }

    try {
      setForceRefreshLoading(true);
      setEditDialogError(null);
      setEditDialogSuccess(null);

      const response = await fetch(`/api/plugin-instances/${editPluginInstance.id}/force-refresh`, {
        method: "POST",
        credentials: "include",
      });

      if (response.ok) {
        setEditDialogSuccess("Content refresh started! Your plugin will update shortly.");
      } else {
        const errorData = await response.json();
        setEditDialogError(errorData.error || "Failed to force refresh plugin");
      }
    } catch (error) {
      setEditDialogError("Network error occurred");
    } finally {
      setForceRefreshLoading(false);
    }
  };

  const deletePluginInstance = async (userPluginId: string) => {
    try {
      setError(null);
      const response = await fetch(`/api/plugin-instances/${userPluginId}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        await fetchPluginInstances();
        onUpdate?.();
      } else {
        const errorData = await response.json();
        setError(errorData.details || errorData.error || "Failed to delete plugin instance");
      }
    } catch (error) {
      setError("Network error occurred");
    }
  };

  useEffect(() => {
    fetchPluginInstances();
    fetchPlugins();
    fetchRefreshRateOptions();
  }, [selectedDeviceId]);

  // Refresh plugins when window gains focus (catches returns from private plugin editing)
  useEffect(() => {
    const handleFocus = () => {
      fetchPlugins();
    };

    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, []);

  // Load mashup data when mashup layout is selected
  useEffect(() => {
    if (creationMode === 'mashup' && selectedMashupLayout) {
      loadMashupData(selectedMashupLayout);
    }
  }, [creationMode, selectedMashupLayout]);

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

  useEffect(() => {
    if (editDialogSuccess) {
      const timer = setTimeout(() => setEditDialogSuccess(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [editDialogSuccess]);

  // Save sort state to localStorage whenever it changes
  useEffect(() => {
    try {
      localStorage.setItem('pluginInstanceTableSort', JSON.stringify(sortState));
    } catch (e) {
      // Ignore localStorage errors
    }
  }, [sortState]);

  // Helper function to load mashup layout info for editing
  const loadEditMashupLayout = async (instanceId: string) => {
    console.log('🔄 loadEditMashupLayout called with instanceId:', instanceId);
    setEditMashupLayoutLoading(true);
    try {
      console.log('📡 Fetching mashup children data...');
      const mashupData = await mashupService.getChildren(instanceId);
      console.log('✅ Mashup data received:', mashupData);
      console.log('🎯 Layout extracted:', mashupData.layout);
      console.log('🔍 Slots extracted:', mashupData.slots);
      console.log('📋 Assignments extracted:', mashupData.assignments);
      
      setEditMashupLayout(mashupData.layout);
      setEditMashupSlots(mashupData.slots || []);
      
      // Convert assignments from MashupChild objects to plugin instance IDs
      const assignmentMap: Record<string, string> = {};
      if (mashupData.assignments) {
        console.log('🔍 Processing assignments:', mashupData.assignments);
        for (const [slotPosition, child] of Object.entries(mashupData.assignments)) {
          console.log(`  - Slot ${slotPosition}:`, child);
          if (child && child.instance_id) {
            assignmentMap[slotPosition] = child.instance_id;
            console.log(`    ✅ Mapped to instance ID: ${child.instance_id}`);
          }
        }
      }
      console.log('📊 Final assignment map:', assignmentMap);
      setEditMashupAssignments(assignmentMap);
      setEditOriginalMashupAssignments({...assignmentMap}); // Store original for change detection
      
      // Load available plugins to display names
      console.log('📡 Loading available plugins for edit modal...');
      const availablePlugins = await mashupService.getAvailablePluginInstances();
      console.log('✅ Available plugins loaded:', availablePlugins);
      setEditAvailablePlugins(availablePlugins);
    } catch (error) {
      console.error('❌ Error loading mashup layout:', error);
      setEditMashupLayout(null);
      setEditMashupSlots([]);
      setEditMashupAssignments({});
      setEditOriginalMashupAssignments({});
      setEditAvailablePlugins([]);
    } finally {
      setEditMashupLayoutLoading(false);
    }
  };

  // Helper function to open edit dialog for an instance
  const openEditDialog = (instanceToEdit: PluginInstance) => {
    console.log('🚀 openEditDialog called with instance:', instanceToEdit);
    console.log('🔍 Instance plugin object:', instanceToEdit.plugin);
    console.log('🏷️ Plugin type:', instanceToEdit.plugin?.type);
    console.log('📋 Plugin name:', instanceToEdit.plugin?.name);
    
    setEditPluginInstance(instanceToEdit);
    setEditInstanceName(instanceToEdit.name);
    
    // Parse settings from JSON string to object
    let parsedSettings = {};
    try {
      if (instanceToEdit.settings && typeof instanceToEdit.settings === 'string') {
        parsedSettings = JSON.parse(instanceToEdit.settings);
      } else if (instanceToEdit.settings && typeof instanceToEdit.settings === 'object') {
        parsedSettings = instanceToEdit.settings;
      }
    } catch (e) {
      console.error("Error parsing plugin settings:", e);
      parsedSettings = {};
    }
    
    setEditInstanceSettings(parsedSettings);
    setEditInstanceRefreshRate(instanceToEdit.refresh_interval || 86400);
    
    // Clear dialog-specific alerts when opening
    setEditDialogError(null);
    setEditDialogSuccess(null);
    
    // If it's a mashup, load the layout info
    const isMashup = instanceToEdit.plugin?.type === 'mashup';
    console.log('🤔 Is mashup check:', isMashup);
    if (isMashup) {
      console.log('✅ Detected mashup - loading layout info');
      loadEditMashupLayout(instanceToEdit.id);
    } else {
      console.log('❌ Not a mashup - clearing layout');
      setEditMashupLayout(null);
      setEditMashupSlots([]);
      setEditMashupAssignments({});
      setEditOriginalMashupAssignments({});
      setEditAvailablePlugins([]);
    }
    
    // Fetch schema diff if instance needs config update
    if (instanceToEdit.needs_config_update) {
      fetchSchemaDiff(instanceToEdit.id);
    } else {
      setSchemaDiff(null);
    }
    
    setShowEditDialog(true);
  };

  // Handle auto-opening edit dialog from URL parameter - Check for edit parameter
  useEffect(() => {
    const editInstanceId = searchParams.get('edit');
    
    if (editInstanceId) {
      if (pluginInstances.length > 0) {
        // Instances are loaded, try to find and open the edit dialog
        const instanceToEdit = pluginInstances.find(instance => instance.id === editInstanceId);
        if (instanceToEdit) {
          openEditDialog(instanceToEdit);
          setPendingEditInstanceId(null); // Clear pending since we handled it
          
          // Clear the URL parameter
          const newSearchParams = new URLSearchParams(searchParams);
          newSearchParams.delete('edit');
          setSearchParams(newSearchParams);
        }
      } else {
        // Instances not loaded yet, store the pending edit ID
        setPendingEditInstanceId(editInstanceId);
      }
    }
  }, [searchParams, pluginInstances, setSearchParams]);

  // Handle pending edit when plugin instances load
  useEffect(() => {
    if (pendingEditInstanceId && pluginInstances.length > 0) {
      const instanceToEdit = pluginInstances.find(instance => instance.id === pendingEditInstanceId);
      if (instanceToEdit) {
        openEditDialog(instanceToEdit);
        setPendingEditInstanceId(null);
        
        // Clear the URL parameter
        const newSearchParams = new URLSearchParams(searchParams);
        newSearchParams.delete('edit');
        setSearchParams(newSearchParams);
      }
    }
  }, [pluginInstances, pendingEditInstanceId, searchParams, setSearchParams]);

  // Handle auto-opening create dialog from URL parameter - Check for create action
  useEffect(() => {
    const action = searchParams.get('action');
    const pluginId = searchParams.get('pluginId');
    
    if (action === 'create' && pluginId && plugins.length > 0) {
      // Find the plugin by ID
      const selectedPlugin = plugins.find(plugin => plugin.id === pluginId);
      if (selectedPlugin) {
        // Open the create dialog with the selected plugin
        setSelectedPlugin(selectedPlugin);
        setInstanceName(selectedPlugin.name);
        setCreateDialogError(null);
        setCreationMode('plugin');
        setSelectedMashupLayout(null);
        
        // Set up default settings
        try {
          if (selectedPlugin.config_schema) {
            const schema = JSON.parse(selectedPlugin.config_schema);
            const defaults: Record<string, any> = {};
            
            if (schema.properties) {
              Object.keys(schema.properties).forEach(key => {
                const property = schema.properties[key];
                if (property.default !== undefined) {
                  defaults[key] = property.default;
                }
              });
            }
            
            setInstanceSettings(defaults);
          } else {
            setInstanceSettings({});
          }
        } catch (e) {
          setInstanceSettings({});
        }
        
        setShowAddDialog(true);
        
        // Clear the URL parameters
        const newSearchParams = new URLSearchParams(searchParams);
        newSearchParams.delete('action');
        newSearchParams.delete('pluginId');
        setSearchParams(newSearchParams);
      }
    }
  }, [searchParams, plugins, setSearchParams]);

  // Debug: Log when editPluginInstance changes
  useEffect(() => {
    console.log('🔄 editPluginInstance state changed:', editPluginInstance);
    if (editPluginInstance) {
      console.log('  - Instance ID:', editPluginInstance.id);
      console.log('  - Instance name:', editPluginInstance.name);
      console.log('  - Plugin type:', editPluginInstance.plugin?.type);
    }
  }, [editPluginInstance]);

  // Debug: Log when editMashupLayout changes
  useEffect(() => {
    console.log('🗂️ editMashupLayout state changed:', editMashupLayout);
  }, [editMashupLayout]);

  // Sort function
  const handleSort = (column: SortColumn) => {
    setSortState(prevState => ({
      column,
      order: prevState.column === column && prevState.order === 'asc' ? 'desc' : 'asc'
    }));
  };

  // Sort the pluginInstances array based on current sort state
  const sortedPluginInstances = React.useMemo(() => {
    const sorted = [...pluginInstances].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      switch (sortState.column) {
        case 'name':
          aValue = a.name?.toLowerCase() || '';
          bValue = b.name?.toLowerCase() || '';
          break;
        case 'plugin':
          aValue = a.plugin?.name?.toLowerCase() || '';
          bValue = b.plugin?.name?.toLowerCase() || '';
          break;
        case 'status':
          // Priority system: Unavailable (4) > Update Config (3) > Active (2) > Unused (1)
          const getStatusPriority = (instance: PluginInstance) => {
            if (instance.plugin?.status === 'unavailable') return 4;
            if (instance.needs_config_update) return 3;
            if (instance.is_used_in_playlists) return 2;
            return 1;
          };
          aValue = getStatusPriority(a);
          bValue = getStatusPriority(b);
          break;
        case 'created':
          aValue = new Date(a.created_at).getTime();
          bValue = new Date(b.created_at).getTime();
          break;
        default:
          return 0;
      }

      if (aValue < bValue) {
        return sortState.order === 'asc' ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortState.order === 'asc' ? 1 : -1;
      }
      return 0;
    });

    return sorted;
  }, [pluginInstances, sortState]);

  // Private plugin handlers
  const handleCreatePrivatePlugin = () => {
    navigate('/plugins/private/edit');
  };

  const handleEditPrivatePlugin = (plugin: any) => {
    navigate(`/plugins/private/edit?pluginId=${plugin.id}`);
  };

  const handlePreviewPrivatePlugin = (plugin: any) => {
    setPreviewingPrivatePlugin(plugin);
  };


  // Helper function to check if a plugin has configuration fields
  const hasConfigurationFields = (plugin: Plugin): boolean => {
    try {
      const schema = JSON.parse(plugin.config_schema);
      const properties = schema.properties || {};
      return Object.keys(properties).length > 0;
    } catch (e) {
      return false;
    }
  };

  const renderSettingsForm = (plugin: Plugin, settings: Record<string, any>, onChange: (key: string, value: any) => void) => {
    let schema;
    try {
      schema = JSON.parse(plugin.config_schema);
    } catch (e) {
      return <div className="text-muted-foreground">Invalid schema configuration</div>;
    }

    const properties = schema.properties || {};

    return (
      <div className="space-y-4">
        {Object.keys(properties).map((key) => {
          const prop = properties[key];
          const value = settings[key] || prop.default || "";

          // Handle enum (select dropdown) FIRST - before string type
          if (prop.enum && Array.isArray(prop.enum)) {
            const enumNames = prop.enumNames && Array.isArray(prop.enumNames) ? prop.enumNames : prop.enum;
            return (
              <div key={key}>
                <Label htmlFor={key}>{prop.title || key}</Label>
                <Select value={value} onValueChange={(val) => onChange(key, val)}>
                  <SelectTrigger className="mt-2">
                    <SelectValue placeholder={prop.placeholder || "Select an option"} />
                  </SelectTrigger>
                  <SelectContent>
                    {prop.enum.map((option, index) => (
                      <SelectItem key={option} value={option}>
                        {enumNames[index] || option}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {prop.description && (
                  <p className="text-sm text-muted-foreground mt-1">
                    {prop.description}
                  </p>
                )}
              </div>
            );
          }

          if (prop.type === "string") {
            // Handle different string formats
            if (prop.format === "date") {
              const dateValue = value ? new Date(value) : undefined;
              return (
                <div key={key}>
                  <Label htmlFor={key}>{prop.title || key}</Label>
                  <DatePicker
                    date={dateValue}
                    onDateChange={(date) => onChange(key, date ? date.toISOString().split('T')[0] : "")}
                    placeholder={prop.placeholder || "Select date"}
                    className="mt-2"
                  />
                  {prop.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {prop.description}
                    </p>
                  )}
                </div>
              );
            }

            if (prop.format === "time") {
              return (
                <div key={key}>
                  <Label htmlFor={key}>{prop.title || key}</Label>
                  <TimePicker
                    value={value}
                    onChange={(time) => onChange(key, time)}
                    placeholder={prop.placeholder || "HH:MM"}
                    className="mt-2"
                  />
                  {prop.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {prop.description}
                    </p>
                  )}
                </div>
              );
            }

            if (prop.format === "date-time") {
              return (
                <div key={key}>
                  <Label htmlFor={key}>{prop.title || key}</Label>
                  <DateTimePicker
                    value={value}
                    onChange={(datetime) => onChange(key, datetime)}
                    placeholder={prop.placeholder || "Select date and time"}
                    className="mt-2"
                  />
                  {prop.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {prop.description}
                    </p>
                  )}
                </div>
              );
            }

            if (prop.format === "password") {
              return (
                <div key={key}>
                  <Label htmlFor={key}>{prop.title || key}</Label>
                  <Input
                    id={key}
                    type="password"
                    placeholder={prop.placeholder || ""}
                    value={value}
                    onChange={(e) => onChange(key, e.target.value)}
                    className="mt-2"
                  />
                  {prop.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {prop.description}
                    </p>
                  )}
                </div>
              );
            }

            if (prop.format === "uri") {
              return (
                <div key={key}>
                  <Label htmlFor={key}>{prop.title || key}</Label>
                  <Input
                    id={key}
                    type="url"
                    placeholder={prop.placeholder || "https://example.com"}
                    value={value}
                    onChange={(e) => onChange(key, e.target.value)}
                    className="mt-2"
                  />
                  {prop.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {prop.description}
                    </p>
                  )}
                </div>
              );
            }

            // Check if it should be a textarea (long text)
            if (prop.maxLength && prop.maxLength > 200) {
              return (
                <div key={key}>
                  <Label htmlFor={key}>{prop.title || key}</Label>
                  <Textarea
                    id={key}
                    placeholder={prop.placeholder || ""}
                    value={value}
                    onChange={(e) => onChange(key, e.target.value)}
                    className="mt-2"
                    rows={4}
                  />
                  {prop.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {prop.description}
                    </p>
                  )}
                </div>
              );
            }

            // Default string input
            return (
              <div key={key}>
                <Label htmlFor={key}>{prop.title || key}</Label>
                <Input
                  id={key}
                  placeholder={prop.placeholder || ""}
                  value={value}
                  onChange={(e) => onChange(key, e.target.value)}
                  className="mt-2"
                />
                {prop.description && (
                  <p className="text-sm text-muted-foreground mt-1">
                    {prop.description}
                  </p>
                )}
              </div>
            );
          }


          if (prop.type === "integer" || prop.type === "number") {
            const numValue = typeof value === 'number' ? value : (prop.default || 0);
            const isInvalid = (prop.minimum !== undefined && numValue < prop.minimum) ||
                              (prop.maximum !== undefined && numValue > prop.maximum);
            
            return (
              <div key={key}>
                <Label htmlFor={key}>{prop.title || key}</Label>
                <Input
                  id={key}
                  type="number"
                  min={prop.minimum}
                  max={prop.maximum}
                  step={prop.type === "integer" ? 1 : "any"}
                  placeholder={prop.placeholder || ""}
                  value={value}
                  onChange={(e) => onChange(key, prop.type === "integer" ? parseInt(e.target.value) || 0 : parseFloat(e.target.value) || 0)}
                  className={`mt-2 ${isInvalid ? 'border-red-500' : ''}`}
                />
                {isInvalid && (
                  <p className="text-sm text-red-500 mt-1">
                    Value must be between {prop.minimum || 'any'} and {prop.maximum || 'any'}
                  </p>
                )}
                {prop.description && (
                  <p className="text-sm text-muted-foreground mt-1">
                    {prop.description}
                    {(prop.minimum !== undefined || prop.maximum !== undefined) && (
                      <span className="text-xs block">
                        Range: {prop.minimum || 'any'} - {prop.maximum || 'any'}
                      </span>
                    )}
                  </p>
                )}
              </div>
            );
          }

          if (prop.type === "boolean") {
            const boolValue = typeof value === 'boolean' ? value : prop.default || false;
            return (
              <div key={key} className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label htmlFor={key}>{prop.title || key}</Label>
                    {prop.description && (
                      <p className="text-sm text-muted-foreground">
                        {prop.description}
                      </p>
                    )}
                  </div>
                  <Switch
                    id={key}
                    checked={boolValue}
                    onCheckedChange={(checked) => onChange(key, checked)}
                  />
                </div>
              </div>
            );
          }

          return null;
        })}
      </div>
    );
  };


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
          <h3 className="text-lg font-semibold">Plugin Management</h3>
          <p className="text-muted-foreground">
            Manage plugin instances and create private plugins
          </p>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={handleSubTabChange}>
        <TabsList>
          <TabsTrigger value="instances">Plugin Instances</TabsTrigger>
          <TabsTrigger value="private">Private Plugins</TabsTrigger>
        </TabsList>

        <Card>
          <CardContent>
            <TabsContent value="instances" className="space-y-4">
          <div className="flex justify-between items-center">
            <div>
              <h3 className="text-lg font-semibold">Plugin Instances</h3>
              <p className="text-muted-foreground">
                Manage your plugin instances for all devices
              </p>
            </div>
            <AddPluginDropdown
              onPluginSelect={() => {
                setCreationMode('plugin');
                setSelectedMashupLayout(null);
                setShowAddDialog(true);
              }}
              onMashupSelect={(layout) => {
                setCreationMode('mashup');
                setSelectedMashupLayout(layout);
                setShowAddDialog(true);
              }}
            />
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="text-muted-foreground">Loading plugins...</div>
            </div>
          ) : pluginInstances.length === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <Puzzle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Plugin Instances</h3>
            <p className="text-muted-foreground mb-4">
              Create plugin instances to display content on your device.
            </p>
            <AddPluginDropdown
              onPluginSelect={() => {
                setCreationMode('plugin');
                setSelectedMashupLayout(null);
                setShowAddDialog(true);
              }}
              onMashupSelect={(layout) => {
                setCreationMode('mashup');
                setSelectedMashupLayout(layout);
                setShowAddDialog(true);
              }}
            />
          </CardContent>
        </Card>
      ) : (
        <Table className="w-full table-fixed lg:table-auto">
              <TableHeader>
                <TableRow>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50 select-none"
                    onClick={() => handleSort('name')}
                  >
                    <div className="flex items-center gap-1">
                      Name
                      {sortState.column === 'name' ? (
                        sortState.order === 'asc' ? (
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
                    onClick={() => handleSort('plugin')}
                  >
                    <div className="flex items-center gap-1">
                      Plugin
                      {sortState.column === 'plugin' ? (
                        sortState.order === 'asc' ? (
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
                    onClick={() => handleSort('status')}
                  >
                    <div className="flex items-center gap-1">
                      Status
                      {sortState.column === 'status' ? (
                        sortState.order === 'asc' ? (
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
                    onClick={() => handleSort('created')}
                  >
                    <div className="flex items-center gap-1">
                      Created
                      {sortState.column === 'created' ? (
                        sortState.order === 'asc' ? (
                          <ChevronUp className="h-4 w-4" />
                        ) : (
                          <ChevronDown className="h-4 w-4" />
                        )
                      ) : (
                        <ChevronsUpDown className="h-4 w-4 opacity-50" />
                      )}
                    </div>
                  </TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedPluginInstances.map((userPlugin) => (
                  <TableRow 
                    key={userPlugin.id}
                    className=""
                  >
                    <TableCell className="font-medium">
                      <div>
                        <div>{userPlugin.name}</div>
                        <div className="text-sm lg:hidden flex items-center gap-2">
                          <span className="text-muted-foreground flex items-center gap-2">
                            {userPlugin.plugin?.type === "mashup" ? (
                              <>
                                <span>Mashup</span>
                                {mashupLayoutCache[userPlugin.id] && (
                                  <div className="inline-flex">
                                    {getMashupLayoutGrid(mashupLayoutCache[userPlugin.id], 'tiny', 'subtle')}
                                  </div>
                                )}
                              </>
                            ) : (userPlugin.plugin?.name || "Unknown Plugin")}
                          </span>
                          {userPlugin.plugin?.status === 'unavailable' ? (
                            <Badge variant="secondary" className="text-xs">
                              Unavailable
                            </Badge>
                          ) : userPlugin.needs_config_update ? (
                            <Badge 
                              variant="destructive" 
                              className={`text-xs ${userPlugin.plugin?.status !== 'unavailable' ? 'cursor-pointer hover:bg-destructive/80' : 'opacity-60 cursor-not-allowed'}`}
                              onClick={userPlugin.plugin?.status !== 'unavailable' ? () => openEditDialog(userPlugin) : undefined}
                            >
                              Update Config
                            </Badge>
                          ) : (
                            <span className="text-muted-foreground">• {userPlugin.is_used_in_playlists ? "Active" : "Unused"}</span>
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="hidden lg:table-cell">
                      <div className="flex items-center gap-2">
                        {userPlugin.plugin?.type === "mashup" ? (
                          <>
                            <span>Mashup</span>
                            {mashupLayoutCache[userPlugin.id] && (
                              <div className="inline-flex">
                                {getMashupLayoutGrid(mashupLayoutCache[userPlugin.id], 'tiny', 'subtle')}
                              </div>
                            )}
                          </>
                        ) : (userPlugin.plugin?.name || "Unknown Plugin")}
                      </div>
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
                      <div className="flex gap-1 flex-wrap">
                        {userPlugin.plugin?.status === 'unavailable' ? (
                          <Badge variant="secondary">Unavailable</Badge>
                        ) : userPlugin.needs_config_update ? (
                          <Badge 
                            variant="destructive" 
                            className={userPlugin.plugin?.status !== 'unavailable' ? 'cursor-pointer hover:bg-destructive/80' : 'opacity-60 cursor-not-allowed'}
                            onClick={userPlugin.plugin?.status !== 'unavailable' ? () => openEditDialog(userPlugin) : undefined}
                          >
                            Update Config
                          </Badge>
                        ) : userPlugin.is_used_in_playlists ? (
                          <Badge variant="outline">Active</Badge>
                        ) : (
                          <Badge variant="secondary">Unused</Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="hidden lg:table-cell">
                      {new Date(userPlugin.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center gap-2 justify-end">
                        {userPlugin.plugin?.data_strategy === 'webhook' && (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => copyWebhookUrl(userPlugin.id)}
                              >
                                <Copy className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Copy Webhook URL</TooltipContent>
                          </Tooltip>
                        )}
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => openEditDialog(userPlugin)}
                              disabled={userPlugin.plugin?.status === 'unavailable'}
                            >
                              <Edit className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>
                            {userPlugin.plugin?.status === 'unavailable' 
                              ? "Plugin unavailable - cannot edit settings"
                              : "Edit plugin instance"
                            }
                          </TooltipContent>
                        </Tooltip>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setDeletePluginDialog({ isOpen: true, plugin: userPlugin })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
      )}

      {/* Add Plugin Dialog */}
      <Dialog open={showAddDialog} onOpenChange={(open) => {
        setShowAddDialog(open);
        if (!open) {
          resetAddDialogState();
        }
      }}>
        <DialogContent 
          className="sm:max-w-5xl max-h-[70vh] overflow-hidden flex flex-col mobile-dialog-content !top-[0vh] !translate-y-0 sm:!top-[6vh]"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader className="pb-3">
            {creationMode === 'mashup' && selectedMashupLayout ? (
              <div className="flex items-start gap-4">
                <div className="flex-shrink-0">
                  {getMashupLayoutGrid(selectedMashupLayout.id, 'small')}
                </div>
                <div className="flex-1">
                  <DialogTitle>Create Mashup</DialogTitle>
                  <DialogDescription>
                    Create a mashup using the {selectedMashupLayout.name} layout
                  </DialogDescription>
                </div>
              </div>
            ) : (
              <>
                <DialogTitle>Add Plugin Instance</DialogTitle>
                <DialogDescription>
                  {selectedPlugin ? `Configure your ${selectedPlugin.name} instance` : "Select a plugin to create an instance for your device"}
                </DialogDescription>
              </>
            )}
          </DialogHeader>

          <div className="flex-1 overflow-y-auto">
            <div className="space-y-4">
              {createDialogError && (
                <Alert variant="destructive">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>{createDialogError}</AlertDescription>
                </Alert>
              )}

              {creationMode === 'mashup' && selectedMashupLayout ? (
                <>
                  {/* Mashup instance name */}
                  <div>
                    <Label htmlFor="mashup-instanceName" className="text-sm">Mashup Name</Label>
                    <Input
                      id="mashup-instanceName"
                      placeholder={`My ${selectedMashupLayout.name} Mashup`}
                      value={instanceName}
                      onChange={(e) => setInstanceName(e.target.value)}
                      className="mt-1"
                    />
                  </div>

                  {/* Optional description */}
                  <div>
                    <Label htmlFor="mashup-description" className="text-sm">Description (Optional)</Label>
                    <Textarea
                      id="mashup-description"
                      placeholder="Describe what this mashup displays..."
                      value={mashupDescription}
                      onChange={(e) => setMashupDescription(e.target.value)}
                      className="mt-1"
                      rows={2}
                    />
                  </div>

                  {/* Slot assignments */}
                  {mashupSlots.length > 0 && (
                    <div>
                      <Label className="text-sm">Plugin Assignments</Label>
                      <div className="mt-2">
                        <MashupSlotGrid
                          layout={selectedMashupLayout.id}
                          slots={mashupSlots}
                          availablePlugins={availablePluginInstances}
                          assignments={mashupAssignments}
                          onAssignmentsChange={setMashupAssignments}
                        />
                      </div>
                    </div>
                  )}
                </>
              ) : creationMode === 'plugin' && !selectedPlugin ? (
                <div>
                  <div className="mb-3">
                    <Label className="text-base font-semibold">Available Plugins</Label>
                  </div>
                  {loading ? (
                    <div className="text-center py-6">Loading available plugins...</div>
                  ) : plugins.length === 0 ? (
                    <Card>
                      <CardContent className="text-center py-6">
                        <Puzzle className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
                        <h3 className="text-base font-semibold mb-2">No Plugins Available</h3>
                        <p className="text-sm text-muted-foreground">
                          No plugins have been installed by the administrator yet.
                        </p>
                      </CardContent>
                    </Card>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-3">
                      {plugins.map((plugin) => (
                        <Card key={plugin.id} className="flex flex-col">
                          <CardHeader className="pb-2">
                            <CardTitle className="flex items-start justify-between gap-2">
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-2 mb-1">
                                  <div className="text-base font-semibold truncate">{plugin.name}</div>
                                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs ${
                                    plugin.type === 'system' 
                                      ? 'bg-blue-100 text-blue-800' 
                                      : 'bg-purple-100 text-purple-800'
                                  }`}>
                                    {plugin.type === 'system' ? 'System' : 'Private'}
                                  </span>
                                </div>
                                <div className="text-xs text-muted-foreground">
                                  v{plugin.version} by {plugin.author}
                                  {plugin.instance_count !== undefined && (
                                    <span className="ml-2">• {plugin.instance_count} instance{plugin.instance_count !== 1 ? 's' : ''}</span>
                                  )}
                                </div>
                              </div>
                            </CardTitle>
                          </CardHeader>
                          <CardContent className="flex flex-col flex-grow pt-0">
                            <div className="flex-grow mb-3">
                              <p className="text-xs text-muted-foreground line-clamp-2">
                                {plugin.description}
                              </p>
                            </div>
                            <Button
                              onClick={() => {
                                setSelectedPlugin(plugin);
                                setInstanceName(plugin.name);
                                setCreateDialogError(null);
                                
                                try {
                                  if (plugin.config_schema) {
                                    const schema = JSON.parse(plugin.config_schema);
                                    const defaults: Record<string, any> = {};
                                    
                                    if (schema.properties) {
                                      Object.keys(schema.properties).forEach(key => {
                                        const property = schema.properties[key];
                                        if (property.default !== undefined) {
                                          defaults[key] = property.default;
                                        }
                                      });
                                    }
                                    
                                    setInstanceSettings(defaults);
                                  } else {
                                    setInstanceSettings({});
                                  }
                                } catch (e) {
                                  setInstanceSettings({});
                                }
                              }}
                              size="sm"
                              className="w-full mt-auto"
                            >
                              Create Instance
                            </Button>
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  )}
                </div>
              ) : (
                <>
                  <div className="mb-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        resetAddDialogState();
                      }}
                      className="mb-2"
                    >
                      ← Back to Plugin Selection
                    </Button>
                    <div className="flex items-center gap-3 p-2 bg-muted/50 rounded-lg">
                      <div className="flex-1">
                        <div className="font-semibold text-sm">{selectedPlugin.name}</div>
                        <div className="text-xs text-muted-foreground">
                          v{selectedPlugin.version} by {selectedPlugin.author}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div>
                    <Label htmlFor="instanceName" className="text-sm">Instance Name</Label>
                    <Input
                      id="instanceName"
                      placeholder={selectedPlugin.name}
                      value={instanceName}
                      onChange={(e) => setInstanceName(e.target.value)}
                      className="mt-1"
                    />
                  </div>

                  {selectedPlugin.requires_processing && (
                    <div>
                      <Label htmlFor="instanceRefreshRate" className="text-sm">Refresh Rate</Label>
                      <Select
                        value={instanceRefreshRate.toString()}
                        onValueChange={(value) => setInstanceRefreshRate(Number(value))}
                      >
                        <SelectTrigger className="mt-1">
                          <SelectValue placeholder="Select refresh rate" />
                        </SelectTrigger>
                        <SelectContent>
                          {refreshRateOptions.map((option) => (
                            <SelectItem key={option.value} value={option.value.toString()}>
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  {hasConfigurationFields(selectedPlugin) && (
                    <div>
                      <Label className="text-sm">Plugin Configuration</Label>
                      <div className="mt-1">
                        {renderSettingsForm(selectedPlugin, instanceSettings, (key, value) => {
                          setInstanceSettings(prev => ({ ...prev, [key]: value }));
                        })}
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>

          <DialogFooter className="pt-3">
            <Button
              variant="outline"
              onClick={() => {
                setShowAddDialog(false);
                resetAddDialogState();
              }}
            >
              Cancel
            </Button>
            {(selectedPlugin || (creationMode === 'mashup' && selectedMashupLayout)) && (
              <Button
                onClick={createPluginInstance}
                disabled={
                  !instanceName.trim() || 
                  createLoading || 
                  (creationMode === 'mashup' && hasUnassignedSlots())
                }
              >
                {createLoading 
                  ? (creationMode === 'mashup' ? "Creating Mashup..." : "Creating Instance...") 
                  : (creationMode === 'mashup' ? "Create Mashup" : "Create Instance")
                }
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Plugin Dialog */}
      <Dialog open={showEditDialog} onOpenChange={(open) => {
        setShowEditDialog(open);
        if (!open) {
          // Clear dialog-specific alerts when closing
          setEditDialogError(null);
          setEditDialogSuccess(null);
        }
      }}>
        <DialogContent 
          className="sm:max-w-2xl max-h-[80vh] overflow-y-auto mobile-dialog-content !top-[0vh] !translate-y-0 sm:!top-[6vh]"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            {(() => {
              const pluginType = editPluginInstance?.plugin?.type;
              const layoutId = editMashupLayout;
              const isMashup = pluginType === 'mashup';
              const hasLayout = !!layoutId;
              const isLoading = editMashupLayoutLoading;
              const showMashupHeader = isMashup && (hasLayout || isLoading);
              
              console.log('🎨 Edit dialog header render:');
              console.log('  - Plugin type:', pluginType);
              console.log('  - Layout ID:', layoutId);
              console.log('  - Is mashup:', isMashup);
              console.log('  - Has layout:', hasLayout);
              console.log('  - Is loading:', isLoading);
              console.log('  - Show mashup header:', showMashupHeader);
              
              return showMashupHeader ? (
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0">
                    {isLoading ? (
                      <div className="h-10 w-16 border border-dashed border-muted-foreground/30 rounded flex items-center justify-center bg-muted animate-pulse">
                        <div className="text-xs text-muted-foreground">...</div>
                      </div>
                    ) : (
                      getMashupLayoutGrid(layoutId, 'small')
                    )}
                  </div>
                  <div className="flex-1">
                    <DialogTitle>Edit Mashup Instance</DialogTitle>
                    <DialogDescription>
                      Update the settings for "{editPluginInstance?.name}".
                    </DialogDescription>
                  </div>
                </div>
              ) : (
                <>
                  <DialogTitle>Edit Plugin Instance</DialogTitle>
                  <DialogDescription>
                    Update the settings for "{editPluginInstance?.name}".
                  </DialogDescription>
                </>
              );
            })()}
          </DialogHeader>
          
          {/* Schema diff warning banner */}
          {editPluginInstance?.needs_config_update && schemaDiff?.needs_update && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                <strong>Configuration Update Required</strong>
                <br />
                {schemaDiff.message || "This plugin's form has changed. Please review and update your settings."}
              </AlertDescription>
            </Alert>
          )}

          <div className="space-y-6">
            {editDialogError && (
              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{editDialogError}</AlertDescription>
              </Alert>
            )}

            {editDialogSuccess && (
              <Alert>
                <CheckCircle className="h-4 w-4" />
                <AlertDescription>{editDialogSuccess}</AlertDescription>
              </Alert>
            )}

            <div>
              <Label htmlFor="edit-instance-name">Instance Name</Label>
              <Input
                id="edit-instance-name"
                value={editInstanceName}
                onChange={(e) => setEditInstanceName(e.target.value)}
                placeholder="Enter instance name"
                className="mt-2"
              />
            </div>

            {editPluginInstance?.plugin?.data_strategy === 'webhook' && (
              <div>
                <Label>Webhook URL</Label>
                <div className="flex gap-2 mt-2">
                  <Input
                    value={generateInstanceWebhookURL(editPluginInstance.id)}
                    readOnly
                    className="font-mono text-sm"
                  />
                  <Button
                    variant="outline"
                    onClick={() => copyWebhookUrl(editPluginInstance.id)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground mt-1">
                  Use this URL to send webhook data to your plugin instance.
                </p>
              </div>
            )}

            {editPluginInstance?.plugin?.requires_processing && (
              <div>
                <Label htmlFor="edit-instance-refresh-rate">Refresh Rate</Label>
                <div className="flex gap-2 mt-2">
                  <Select
                    value={editInstanceRefreshRate.toString()}
                    onValueChange={(value) => setEditInstanceRefreshRate(Number(value))}
                    disabled={editPluginInstance?.plugin?.type === 'mashup'}
                  >
                    <SelectTrigger className={editPluginInstance?.plugin?.type === 'mashup' ? 'opacity-60' : ''}>
                      <SelectValue placeholder="Select refresh rate" />
                    </SelectTrigger>
                    <SelectContent>
                      {refreshRateOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value.toString()}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Button
                    variant="outline"
                    onClick={forceRefreshPluginInstance}
                    disabled={forceRefreshLoading}
                    className="gap-2"
                  >
                    <RefreshCw className={`h-4 w-4 ${forceRefreshLoading ? "animate-spin" : ""}`} />
                    {forceRefreshLoading ? "Refreshing..." : "Force Refresh"}
                  </Button>
                </div>
                {editPluginInstance?.plugin?.type === 'mashup' && (
                  <p className="text-xs text-muted-foreground mt-2">
                    Refresh rate determined by assigned plugins
                  </p>
                )}
              </div>
            )}

            {/* Mashup slot assignments - read only */}
            {editPluginInstance?.plugin?.type === 'mashup' && editMashupLayout && editMashupSlots.length > 0 && (
              <div>
                <Label className="text-sm">Plugin Assignments</Label>
                <div className="mt-2">
                  <MashupSlotGrid
                    layout={editMashupLayout}
                    slots={editMashupSlots}
                    availablePlugins={editAvailablePlugins}
                    assignments={editMashupAssignments}
                    onAssignmentsChange={setEditMashupAssignments}
                  />
                </div>
              </div>
            )}

            {editPluginInstance?.plugin && hasConfigurationFields(editPluginInstance.plugin) && (
              <>
                <Separator />
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base flex items-center gap-2">
                      <SettingsIcon className="h-4 w-4" />
                      Plugin Configuration
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-0">
                    {renderSettingsForm(
                      editPluginInstance.plugin,
                      editInstanceSettings,
                      (key: string, value: any) => {
                        setEditInstanceSettings(prev => ({ ...prev, [key]: value }));
                      }
                    )}
                  </CardContent>
                </Card>
              </>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => {
              setShowEditDialog(false);
              setEditDialogError(null);
              setEditDialogSuccess(null);
            }}>
              Cancel
            </Button>
            <Button
              onClick={updatePluginInstance}
              disabled={updateLoading || !editInstanceName.trim() || (!hasPluginInstanceChanges() && !editPluginInstance?.needs_config_update)}
            >
              {updateLoading ? "Updating..." : "Update Instance"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Plugin Instance Confirmation Dialog */}
      <AlertDialog
        open={deletePluginDialog.isOpen}
        onOpenChange={(open) => {
          if (!open) {
            setDeletePluginDialog({ isOpen: false, plugin: null });
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Delete Plugin Instance
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the plugin instance "{deletePluginDialog.plugin?.name}"?
              <br /><br />
              This will:
              <ul className="list-disc list-outside ml-6 mt-2 space-y-1">
                <li>Permanently delete this plugin instance and its settings</li>
                <li>Remove it from any playlists it's currently in</li>
                <li>Stop displaying this content on devices</li>
              </ul>
              <br />
              <strong className="text-destructive">
                This action cannot be undone.
              </strong>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setDeletePluginDialog({ isOpen: false, plugin: null })}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={async () => {
                if (deletePluginDialog.plugin) {
                  await deletePluginInstance(deletePluginDialog.plugin.id);
                  setDeletePluginDialog({ isOpen: false, plugin: null });
                }
              }}
            >
              Delete Instance
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
        </TabsContent>

        <TabsContent value="private" className="space-y-4">
          <PrivatePluginList
            onCreatePlugin={handleCreatePrivatePlugin}
            onEditPlugin={handleEditPrivatePlugin}
            onPreviewPlugin={handlePreviewPrivatePlugin}
          />
        </TabsContent>

          </CardContent>
        </Card>
      </Tabs>


      {/* Private Plugin Preview Dialog */}
      {/* {previewingPrivatePlugin && (
        <PluginPreview
          plugin={previewingPrivatePlugin}
          isOpen={Boolean(previewingPrivatePlugin)}
          onClose={() => setPreviewingPrivatePlugin(null)}
        />
      )} */}

    </div>
  );
}
