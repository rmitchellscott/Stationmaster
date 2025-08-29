import React, { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { 
  Puzzle,
  Clock,
  X,
  AlertCircle,
  CheckCircle2
} from "lucide-react";
import { MashupSlotInfo, AvailablePluginInstance } from "@/services/mashupService";


interface MashupSlotAssignerProps {
  layout: string;
  slots: MashupSlotInfo[];
  availablePlugins: AvailablePluginInstance[];
  assignments: Record<string, string>; // slot position -> plugin instance id
  onAssignmentsChange: (assignments: Record<string, string>) => void;
  validationErrors?: Record<string, string>;
  disabled?: boolean;
}

// Plugin card component for display
const PluginCard: React.FC<{
  plugin: AvailablePluginInstance;
}> = ({ plugin }) => {
  const formatRefreshInterval = (seconds: number) => {
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)}h`;
    if (seconds >= 60) return `${Math.floor(seconds / 60)}m`;
    return `${seconds}s`;
  };

  return (
    <Card className="hover:shadow-sm transition-shadow border-l-4 border-l-primary">
      <CardContent className="p-3">
        <div className="flex items-start gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <Puzzle className="w-4 h-4 text-primary flex-shrink-0" />
              <p className="font-medium text-sm truncate">{plugin.name}</p>
            </div>
            <p className="text-xs text-muted-foreground mb-2 line-clamp-2">
              {plugin.plugin_description}
            </p>
            <div className="flex items-center gap-1">
              <Clock className="w-3 h-3 text-muted-foreground" />
              <span className="text-xs text-muted-foreground">
                {formatRefreshInterval(plugin.refresh_interval)}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

// Slot selector component
const SlotSelector: React.FC<{
  slot: MashupSlotInfo;
  assignedPlugin: AvailablePluginInstance | null;
  availablePlugins: AvailablePluginInstance[];
  onAssignPlugin: (pluginId: string) => void;
  onRemoveAssignment: () => void;
  validationError?: string;
}> = ({ slot, assignedPlugin, availablePlugins, onAssignPlugin, onRemoveAssignment, validationError }) => {
  const formatRefreshInterval = (seconds: number) => {
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)}h`;
    if (seconds >= 60) return `${Math.floor(seconds / 60)}m`;
    return `${seconds}s`;
  };

  return (
    <Card className={`min-h-[140px] transition-all duration-200 ${
      validationError ? 'border-red-300 bg-red-50' : ''
    }`}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">{slot.display_name}</CardTitle>
          <Badge variant="outline" className="text-xs">
            {slot.required_size}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground">Position: {slot.position}</p>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="space-y-3">
          <Select value={assignedPlugin?.id || ""} onValueChange={onAssignPlugin}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select a plugin..." />
            </SelectTrigger>
            <SelectContent>
              {availablePlugins.map((plugin) => (
                <SelectItem key={plugin.id} value={plugin.id}>
                  <div className="flex items-center gap-2">
                    <Puzzle className="w-4 h-4 text-primary flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <span className="font-medium text-sm truncate">{plugin.name}</span>
                      <span className="text-xs text-muted-foreground ml-2">
                        ({formatRefreshInterval(plugin.refresh_interval)})
                      </span>
                    </div>
                  </div>
                </SelectItem>
              ))}
              {availablePlugins.length === 0 && (
                <SelectItem value="" disabled>
                  No available plugins
                </SelectItem>
              )}
            </SelectContent>
          </Select>

          {assignedPlugin && (
            <div className="space-y-2">
              <div className="flex items-start gap-2 p-2 bg-muted rounded">
                <Puzzle className="w-4 h-4 text-primary mt-0.5 flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-sm truncate">{assignedPlugin.name}</p>
                  <p className="text-xs text-muted-foreground line-clamp-1">
                    {assignedPlugin.plugin_description}
                  </p>
                  <div className="flex items-center gap-1 mt-1">
                    <Clock className="w-3 h-3 text-muted-foreground" />
                    <span className="text-xs text-muted-foreground">
                      {formatRefreshInterval(assignedPlugin.refresh_interval)}
                    </span>
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={onRemoveAssignment}
                  className="h-6 w-6 p-0 hover:bg-red-100 hover:text-red-600"
                >
                  <X className="w-3 h-3" />
                </Button>
              </div>
            </div>
          )}
        </div>
        
        {validationError && (
          <div className="flex items-center gap-1 mt-2 text-red-600">
            <AlertCircle className="w-3 h-3" />
            <p className="text-xs">{validationError}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export const MashupSlotAssigner: React.FC<MashupSlotAssignerProps> = ({
  layout,
  slots,
  availablePlugins,
  assignments,
  onAssignmentsChange,
  validationErrors = {},
  disabled = false
}) => {

  // Get assigned plugin instances
  const getAssignedPlugin = (slotPosition: string): AvailablePluginInstance | null => {
    const pluginId = assignments[slotPosition];
    if (!pluginId || !availablePlugins || !Array.isArray(availablePlugins)) return null;
    return availablePlugins.find(p => p && p.id === pluginId) || null;
  };

  // Get available plugins for a specific slot (excluding already assigned ones)
  const getAvailablePluginsForSlot = (currentSlotPosition: string): AvailablePluginInstance[] => {
    if (!availablePlugins || !Array.isArray(availablePlugins)) return [];
    
    const currentAssignedId = assignments[currentSlotPosition];
    const otherAssignedIds = Object.entries(assignments || {})
      .filter(([slotPos]) => slotPos !== currentSlotPosition)
      .map(([, pluginId]) => pluginId);
    
    return availablePlugins.filter(plugin => 
      plugin && plugin.id && !otherAssignedIds.includes(plugin.id)
    );
  };

  // Get unassigned plugins for summary
  const unassignedPlugins = useMemo(() => {
    if (!availablePlugins || !Array.isArray(availablePlugins)) return [];
    
    const assignedIds = Object.values(assignments || {});
    return availablePlugins.filter(plugin => 
      plugin && plugin.id && !assignedIds.includes(plugin.id)
    );
  }, [availablePlugins, assignments]);

  // Calculate minimum refresh rate
  const calculatedRefreshRate = useMemo(() => {
    if (!availablePlugins || !Array.isArray(availablePlugins) || !assignments) return null;
    
    const assignedPlugins = Object.values(assignments)
      .map(id => availablePlugins.find(p => p && p.id === id))
      .filter(Boolean) as AvailablePluginInstance[];
    
    if (assignedPlugins.length === 0) return null;
    
    return Math.min(...assignedPlugins.map(p => p.refresh_interval || 3600));
  }, [assignments, availablePlugins]);

  const formatRefreshRate = (seconds: number) => {
    if (seconds >= 3600) return `${Math.floor(seconds / 3600)}h`;
    if (seconds >= 60) return `${Math.floor(seconds / 60)}m`;
    return `${seconds}s`;
  };

  const handleAssignPlugin = (slotPosition: string, pluginId: string) => {
    const newAssignments = { ...assignments };
    if (pluginId) {
      newAssignments[slotPosition] = pluginId;
    } else {
      delete newAssignments[slotPosition];
    }
    onAssignmentsChange(newAssignments);
  };

  const handleRemoveAssignment = (slotPosition: string) => {
    const newAssignments = { ...assignments };
    delete newAssignments[slotPosition];
    onAssignmentsChange(newAssignments);
  };

  if (disabled) {
    return (
      <div className="space-y-4 opacity-50 pointer-events-none">
        <div className="text-sm text-muted-foreground">
          Plugin assignment is disabled. Please select a layout first.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Mashup refresh rate display */}
      {calculatedRefreshRate && (
        <Card className="bg-green-50 border-green-200">
          <CardContent className="p-4">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-green-600" />
              <span className="text-sm font-medium text-green-800">
                Mashup refresh rate: {formatRefreshRate(calculatedRefreshRate)}
              </span>
              <span className="text-xs text-green-600">
                (minimum of assigned plugins)
              </span>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Available plugins summary */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium">Available Plugins ({(availablePlugins || []).length})</h3>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {(availablePlugins || []).map((plugin) => {
              if (!plugin || !plugin.id) return null;
              
              const isAssigned = Object.values(assignments || {}).includes(plugin.id);
              return (
                <div key={plugin.id} className={isAssigned ? "opacity-50" : ""}>
                  <PluginCard plugin={plugin} />
                  {isAssigned && (
                    <p className="text-xs text-muted-foreground mt-1 ml-3">
                      Assigned to slot
                    </p>
                  )}
                </div>
              );
            })}
            {(availablePlugins || []).length === 0 && (
              <div className="text-center py-8 text-muted-foreground">
                <Puzzle className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p className="text-sm">No plugins available</p>
              </div>
            )}
          </div>
        </div>

        {/* Slot assignments */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium">Mashup Slots ({layout})</h3>
          <div className="grid grid-cols-1 gap-4">
            {slots.map((slot) => (
              <SlotSelector
                key={slot.position}
                slot={slot}
                assignedPlugin={getAssignedPlugin(slot.position)}
                availablePlugins={getAvailablePluginsForSlot(slot.position)}
                onAssignPlugin={(pluginId) => handleAssignPlugin(slot.position, pluginId)}
                onRemoveAssignment={() => handleRemoveAssignment(slot.position)}
                validationError={validationErrors[slot.position]}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};