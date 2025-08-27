import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ChevronDown, Puzzle, Grid2x2, Columns2, Rows2 } from "lucide-react";
import { MashupLayout, mashupService } from "@/services/mashupService";

interface AddPluginDropdownProps {
  onPluginSelect: () => void;
  onMashupSelect: (layout: MashupLayout) => void;
  disabled?: boolean;
}

// Single plugin layout representation
const getSinglePluginGrid = () => {
  const baseClasses = "border border-dashed border-muted-foreground/30 rounded text-xs flex items-center justify-center text-muted-foreground/60 font-medium bg-muted/20";
  return <div className={`${baseClasses} h-16 w-20`}>Plugin</div>;
};

// Mashup layout grid representations (reused from MashupLayoutSelector)
const getMashupLayoutGrid = (layoutId: string) => {
  const baseClasses = "border border-dashed border-muted-foreground/30 rounded text-xs flex items-center justify-center text-muted-foreground/60 font-medium";
  
  switch (layoutId) {
    case "1Lx1R": // Left | Right
      return (
        <div className="grid grid-cols-2 gap-1 h-16 w-20">
          <div className={`${baseClasses} bg-muted/20`}>L</div>
          <div className={`${baseClasses} bg-muted/20`}>R</div>
        </div>
      );
    case "1Tx1B": // Top / Bottom
      return (
        <div className="grid grid-rows-2 gap-1 h-16 w-20">
          <div className={`${baseClasses} bg-muted/20`}>T</div>
          <div className={`${baseClasses} bg-muted/20`}>B</div>
        </div>
      );
    case "1Lx2R": // Left | Top-Right / Bottom-Right
      return (
        <div className="grid grid-cols-2 gap-1 h-16 w-20">
          <div className={`${baseClasses} bg-muted/20 row-span-2`}>L</div>
          <div className="grid grid-rows-2 gap-1">
            <div className={`${baseClasses} bg-muted/20`}>RT</div>
            <div className={`${baseClasses} bg-muted/20`}>RB</div>
          </div>
        </div>
      );
    case "2Lx1R": // Left-Top / Left-Bottom | Right
      return (
        <div className="grid grid-cols-2 gap-1 h-16 w-20">
          <div className="grid grid-rows-2 gap-1">
            <div className={`${baseClasses} bg-muted/20`}>LT</div>
            <div className={`${baseClasses} bg-muted/20`}>LB</div>
          </div>
          <div className={`${baseClasses} bg-muted/20 row-span-2`}>R</div>
        </div>
      );
    case "2Tx1B": // Top-Left | Top-Right / Bottom
      return (
        <div className="grid grid-rows-2 gap-1 h-16 w-20">
          <div className="grid grid-cols-2 gap-1">
            <div className={`${baseClasses} bg-muted/20`}>TL</div>
            <div className={`${baseClasses} bg-muted/20`}>TR</div>
          </div>
          <div className={`${baseClasses} bg-muted/20 col-span-2`}>B</div>
        </div>
      );
    case "1Tx2B": // Top / Bottom-Left | Bottom-Right
      return (
        <div className="grid grid-rows-2 gap-1 h-16 w-20">
          <div className={`${baseClasses} bg-muted/20 col-span-2`}>T</div>
          <div className="grid grid-cols-2 gap-1">
            <div className={`${baseClasses} bg-muted/20`}>BL</div>
            <div className={`${baseClasses} bg-muted/20`}>BR</div>
          </div>
        </div>
      );
    case "2x2": // Quadrant grid
      return (
        <div className="grid grid-cols-2 grid-rows-2 gap-1 h-16 w-20">
          <div className={`${baseClasses} bg-muted/20`}>Q1</div>
          <div className={`${baseClasses} bg-muted/20`}>Q2</div>
          <div className={`${baseClasses} bg-muted/20`}>Q3</div>
          <div className={`${baseClasses} bg-muted/20`}>Q4</div>
        </div>
      );
    default:
      return <div className={`${baseClasses} h-16 w-20 bg-muted/20`}>?</div>;
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
          <div className="text-sm font-medium mb-3">Choose Plugin Type</div>
          
          {/* Single Plugin Option */}
          <Card 
            className="mb-3 cursor-pointer transition-all duration-200 hover:shadow-sm hover:border-muted-foreground/30"
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
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium flex items-center gap-2">
                  <Puzzle className="h-4 w-4" />
                  Single Plugin
                </CardTitle>
                <Badge variant="secondary" className="text-xs">
                  1 slot
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="pt-0">
              <div className="flex items-center justify-center mb-2">
                {getSinglePluginGrid()}
              </div>
              <p className="text-xs text-muted-foreground">
                Create a regular plugin instance
              </p>
            </CardContent>
          </Card>

          {/* Mashup Layouts */}
          {loading ? (
            <div className="text-center py-4 text-sm text-muted-foreground">
              Loading layouts...
            </div>
          ) : (
            <div className="space-y-2">
              <div className="text-xs font-medium text-muted-foreground mb-2">
                Mashup Layouts
              </div>
              <div className="max-h-64 overflow-y-auto space-y-2">
                {layouts.map((layout) => (
                  <Card
                    key={layout.id}
                    className="cursor-pointer transition-all duration-200 hover:shadow-sm hover:border-muted-foreground/30"
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
                    <CardHeader className="pb-2">
                      <div className="flex items-center justify-between">
                        <CardTitle className="text-sm font-medium flex items-center gap-2">
                          <Grid2x2 className="h-4 w-4" />
                          {layout.name}
                        </CardTitle>
                        <Badge variant="secondary" className="text-xs">
                          {layout.slots} slots
                        </Badge>
                      </div>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <div className="flex items-center justify-center mb-2">
                        {getMashupLayoutGrid(layout.id)}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {layout.description}
                      </p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};