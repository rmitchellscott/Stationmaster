import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ChevronDown, Puzzle, Grid2x2, Columns2, Rows2 } from "lucide-react";
import { MashupLayout, mashupService } from "@/services/mashupService";

interface AddPluginDropdownProps {
  onPluginSelect: () => void;
  onMashupSelect: (layout: MashupLayout) => void;
  disabled?: boolean;
}

// Single plugin layout representation
const getSinglePluginGrid = () => {
  const baseClasses = "border border-dashed border-muted-foreground/30 rounded text-xs flex items-center justify-center text-muted-foreground/60 font-medium bg-muted/60";
  return <div className={`${baseClasses} h-16 w-24`}></div>;
};

// Mashup layout grid representations
const getMashupLayoutGrid = (layoutId: string) => {
  const baseClasses = "border border-dashed border-muted-foreground/30 rounded flex items-center justify-center bg-muted/60";
  
  console.log('Rendering layout:', layoutId); // Debug log
  
  switch (layoutId) {
    case "1Lx1R": // Left | Right
      return (
        <div className="grid grid-cols-2 gap-1 h-16 w-24">
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
        </div>
      );
    case "1Tx1B": // Top / Bottom
      return (
        <div className="grid grid-rows-2 gap-1 h-16 w-24">
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
        </div>
      );
    case "1Lx2R": // Left | Top-Right / Bottom-Right
      return (
        <div className="grid grid-cols-2 gap-1 h-16 w-24">
          <div className={baseClasses}></div>
          <div className="grid grid-rows-2 gap-1">
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
        </div>
      );
    case "2Lx1R": // Left-Top / Left-Bottom | Right
      return (
        <div className="grid grid-cols-2 gap-1 h-16 w-24">
          <div className="grid grid-rows-2 gap-1">
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
          <div className={baseClasses}></div>
        </div>
      );
    case "2Tx1B": // Top-Left | Top-Right / Bottom
      return (
        <div className="grid grid-rows-2 gap-1 h-16 w-24">
          <div className="grid grid-cols-2 gap-1">
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
          <div className={baseClasses}></div>
        </div>
      );
    case "1Tx2B": // Top / Bottom-Left | Bottom-Right
      return (
        <div className="grid grid-rows-2 gap-1 h-16 w-24">
          <div className={baseClasses}></div>
          <div className="grid grid-cols-2 gap-1">
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
        </div>
      );
    case "2x2": // Quadrant grid
      return (
        <div className="grid grid-cols-2 grid-rows-2 gap-1 h-16 w-24">
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
        </div>
      );
    default:
      console.warn('Unknown layout ID:', layoutId); // Debug warning
      return <div className={`${baseClasses} h-16 w-24`}>?</div>;
  }
};

export const AddPluginDropdown: React.FC<AddPluginDropdownProps> = ({
  onPluginSelect,
  onMashupSelect,
  disabled = false
}) => {
  const [open, setOpen] = useState(false);
  const [layouts, setLayouts] = useState<MashupLayout[]>([]);
  const [loading, setLoading] = useState(false);

  // Load mashup layouts when component mounts or popover opens
  useEffect(() => {
    if (open && layouts.length === 0) {
      loadLayouts();
    }
  }, [open, layouts.length]);

  const loadLayouts = async () => {
    try {
      setLoading(true);
      const layoutsData = await mashupService.getAvailableLayouts();
      setLayouts(layoutsData);
    } catch (error) {
      console.error("Failed to load mashup layouts:", error);
    } finally {
      setLoading(false);
    }
  };

  const handlePluginSelect = () => {
    setOpen(false);
    onPluginSelect();
  };

  const handleMashupSelect = (layout: MashupLayout) => {
    setOpen(false);
    onMashupSelect(layout);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button 
          disabled={disabled} 
          className="gap-2"
          aria-haspopup="dialog"
          aria-expanded={open}
          aria-label="Add plugin or mashup"
        >
          Add Plugin Instance
          <ChevronDown className="h-4 w-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent 
        className="w-80 p-0" 
        align="end"
        role="dialog"
        aria-label="Plugin selection menu"
      >
        <div className="p-4">
          {loading ? (
            <div className="text-center py-4 text-sm text-muted-foreground">
              Loading layouts...
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              {/* Single Plugin Option - First in grid */}
              <div 
                className="p-4 cursor-pointer transition-all duration-200 hover:bg-muted/20 rounded-lg border border-dashed border-muted-foreground/30"
                onClick={handlePluginSelect}
                role="button"
                tabIndex={0}
                aria-label="Create single plugin instance"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handlePluginSelect();
                  }
                }}
              >
                <div className="flex items-center justify-center">
                  {getSinglePluginGrid()}
                </div>
              </div>
              
              {/* Ordered Mashup Layouts */}
              {['1Lx1R', '1Lx2R', '2Lx1R', '1Tx1B', '1Tx2B', '2Tx1B', '2x2']
                .map(layoutId => layouts.find(l => l.id === layoutId))
                .filter(Boolean)
                .map((layout) => (
                  <div
                    key={layout.id}
                    className="p-4 cursor-pointer transition-all duration-200 hover:bg-muted/20 rounded-lg border border-dashed border-muted-foreground/30"
                    onClick={() => handleMashupSelect(layout)}
                    role="button"
                    tabIndex={0}
                    aria-label={`Create mashup with ${layout.name} layout`}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        handleMashupSelect(layout);
                      }
                    }}
                  >
                    <div className="flex items-center justify-center">
                      {getMashupLayoutGrid(layout.id)}
                    </div>
                  </div>
                ))
              }
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};