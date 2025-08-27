import React from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MashupSlotInfo, AvailablePluginInstance } from "@/services/mashupService";

interface MashupSlotGridProps {
  layout: string;
  slots: MashupSlotInfo[];
  availablePlugins: AvailablePluginInstance[];
  assignments: Record<string, string>; // slot position -> plugin instance id
  onAssignmentsChange: (assignments: Record<string, string>) => void;
  validationErrors?: Record<string, string>;
  disabled?: boolean;
}

// Define grid area mapping for each layout based on slot position
const getGridAreaForSlot = (layout: string, slotPosition: string): string => {
  const layoutGridMaps: Record<string, Record<string, string>> = {
    "1Lx1R": {
      "left": "1 / 1 / 3 / 2",
      "right": "1 / 2 / 3 / 3"
    },
    "1Tx1B": {
      "top": "1 / 1 / 2 / 3", 
      "bottom": "2 / 1 / 3 / 3"
    },
    "2x2": {
      "q1": "1 / 1 / 2 / 2",
      "q2": "1 / 2 / 2 / 3", 
      "q3": "2 / 1 / 3 / 2",
      "q4": "2 / 2 / 3 / 3"
    },
    "1Lx2R": {
      "left": "1 / 1 / 3 / 2",
      "right-top": "1 / 2 / 2 / 3",
      "right-bottom": "2 / 2 / 3 / 3"
    },
    "2Lx1R": {
      "left-top": "1 / 1 / 2 / 2",
      "left-bottom": "2 / 1 / 3 / 2", 
      "right": "1 / 2 / 3 / 3"
    },
    "2Tx1B": {
      "top-left": "1 / 1 / 2 / 2",
      "top-right": "1 / 2 / 2 / 3",
      "bottom": "2 / 1 / 3 / 3"
    },
    "1Tx2B": {
      "top": "1 / 1 / 2 / 3",
      "bottom-left": "2 / 1 / 3 / 2", 
      "bottom-right": "2 / 2 / 3 / 3"
    }
  };

  const layoutMap = layoutGridMaps[layout];
  if (!layoutMap) {
    console.warn(`No grid mapping found for layout: ${layout}`);
    return "1 / 1 / 2 / 2"; // fallback
  }


  // Try exact match first
  if (layoutMap[slotPosition]) {
    return layoutMap[slotPosition];
  }

  // Try partial matching for variations in API naming
  for (const [key, gridArea] of Object.entries(layoutMap)) {
    if (slotPosition.includes(key) || key.includes(slotPosition)) {
      return gridArea;
    }
  }

  console.warn(`No grid area mapping found for slot: "${slotPosition}" in layout: ${layout}`);
  return "1 / 1 / 2 / 2"; // fallback
};


export const MashupSlotGrid: React.FC<MashupSlotGridProps> = ({
  layout,
  slots,
  availablePlugins,
  assignments,
  onAssignmentsChange,
  validationErrors = {},
  disabled = false
}) => {
  
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

  const handleAssignPlugin = (slotPosition: string, pluginId: string) => {
    const newAssignments = { ...assignments };
    if (pluginId && pluginId !== "" && pluginId !== "__none__") {
      newAssignments[slotPosition] = pluginId;
    } else {
      delete newAssignments[slotPosition];
    }
    onAssignmentsChange(newAssignments);
  };


  const renderSlotSelector = (slot: MashupSlotInfo) => {
    const slotPosition = slot.position;
    const assignedPluginId = assignments[slotPosition];
    const availableForThisSlot = getAvailablePluginsForSlot(slotPosition);
    const hasError = validationErrors[slotPosition];
    const gridArea = getGridAreaForSlot(layout, slotPosition);

    return (
      <div 
        key={slot.position}
        className={hasError ? 'ring-2 ring-red-300' : ''}
        style={{ gridArea }}
      >
        <div className="space-y-2">
          <div className="text-xs font-medium text-muted-foreground">
            {slot.display_name}
          </div>
          <Select 
            value={assignedPluginId || "__none__"} 
            onValueChange={(value) => handleAssignPlugin(slotPosition, value)}
            disabled={disabled}
          >
            <SelectTrigger 
              className={`w-full ${hasError ? 'border-red-300' : ''}`}
              aria-label={`Select plugin for ${slot.display_name}`}
            >
              <SelectValue placeholder="Choose plugin..." />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">None</SelectItem>
              {availableForThisSlot.map((plugin) => (
                <SelectItem key={plugin.id} value={plugin.id}>
                  {plugin.name}
                </SelectItem>
              ))}
              {availableForThisSlot.length === 0 && (
                <SelectItem value="__no_plugins__" disabled>
                  No available plugins
                </SelectItem>
              )}
            </SelectContent>
          </Select>
          {hasError && (
            <div className="text-xs text-red-600">{hasError}</div>
          )}
        </div>
      </div>
    );
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

  if (slots.length === 0) {
    return (
      <div className="space-y-4">
        <div className="text-sm text-muted-foreground">
          No slots available for layout: {layout}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* 2x2 Grid */}
      <div className="grid grid-cols-2 grid-rows-2 gap-4 min-h-[160px]">
        {slots.map((slot) => renderSlotSelector(slot))}
      </div>
      
      {/* Assignment Summary */}
      {Object.keys(assignments).length > 0 && (
        <div className="text-xs text-muted-foreground">
          {Object.keys(assignments).length} of {slots.length} slots assigned
        </div>
      )}
    </div>
  );
};