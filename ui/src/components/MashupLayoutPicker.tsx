import React from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
  ColumnsIcon,
  Grid2x2Icon,
  Rows3Icon,
  Rows2Icon,
} from "lucide-react";

export interface MashupLayout {
  id: string;
  name: string;
  description: string;
  positions: string[];
  icon: React.ReactNode;
  gridTemplate: string;
}

export const mashupLayouts: MashupLayout[] = [
  {
    id: "1L1R",
    name: "Left & Right",
    description: "Two plugins side by side",
    positions: ["left", "right"],
    icon: <ColumnsIcon className="h-6 w-6" />,
    gridTemplate: "grid-cols-2 grid-rows-1",
  },
  {
    id: "2T1B", 
    name: "Two Top, One Bottom",
    description: "Two plugins on top, one on bottom",
    positions: ["top-left", "top-right", "bottom"],
    icon: <Rows2Icon className="h-6 w-6" />,
    gridTemplate: "grid-cols-2 grid-rows-2",
  },
  {
    id: "1T2B",
    name: "One Top, Two Bottom", 
    description: "One plugin on top, two on bottom",
    positions: ["top", "bottom-left", "bottom-right"],
    icon: <Rows3Icon className="h-6 w-6" />,
    gridTemplate: "grid-cols-2 grid-rows-2",
  },
  {
    id: "2x2",
    name: "Four Quadrants",
    description: "Four plugins in a 2x2 grid",
    positions: ["top-left", "top-right", "bottom-left", "bottom-right"],
    icon: <Grid2x2Icon className="h-6 w-6" />,
    gridTemplate: "grid-cols-2 grid-rows-2",
  },
];

interface MashupLayoutPickerProps {
  selectedLayout?: string;
  onLayoutSelect: (layout: MashupLayout) => void;
  className?: string;
}

const GridPreview: React.FC<{ layout: MashupLayout; isSelected: boolean }> = ({ 
  layout, 
  isSelected 
}) => {
  const getGridAreaClass = (position: string, layoutId: string) => {
    const areaMap: { [key: string]: { [pos: string]: string } } = {
      "1L1R": {
        "left": "col-span-1 row-span-1",
        "right": "col-span-1 row-span-1",
      },
      "2T1B": {
        "top-left": "col-span-1 row-span-1",
        "top-right": "col-span-1 row-span-1", 
        "bottom": "col-span-2 row-span-1",
      },
      "1T2B": {
        "top": "col-span-2 row-span-1",
        "bottom-left": "col-span-1 row-span-1",
        "bottom-right": "col-span-1 row-span-1",
      },
      "2x2": {
        "top-left": "col-span-1 row-span-1",
        "top-right": "col-span-1 row-span-1",
        "bottom-left": "col-span-1 row-span-1", 
        "bottom-right": "col-span-1 row-span-1",
      },
    };
    
    return areaMap[layoutId]?.[position] || "col-span-1 row-span-1";
  };

  return (
    <div className="w-full h-24 p-2">
      <div className={`grid ${layout.gridTemplate} gap-1 h-full w-full`}>
        {layout.positions.map((position, index) => (
          <div
            key={position}
            className={`
              ${getGridAreaClass(position, layout.id)}
              ${isSelected 
                ? "bg-primary/20 border border-primary" 
                : "bg-muted border border-muted-foreground/20"
              }
              rounded flex items-center justify-center text-xs font-mono
              transition-colors
            `}
          >
            {index + 1}
          </div>
        ))}
      </div>
    </div>
  );
};

export const MashupLayoutPicker: React.FC<MashupLayoutPickerProps> = ({
  selectedLayout,
  onLayoutSelect,
  className = "",
}) => {
  return (
    <div className={`space-y-4 ${className}`}>
      <div>
        <Label className="text-base font-medium">Choose Layout</Label>
        <p className="text-sm text-muted-foreground mt-1">
          Select how your plugins will be arranged in the mashup
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {mashupLayouts.map((layout) => {
          const isSelected = selectedLayout === layout.id;
          
          return (
            <Card
              key={layout.id}
              className={`cursor-pointer transition-all hover:shadow-md ${
                isSelected
                  ? "ring-2 ring-primary border-primary bg-primary/5"
                  : "border-border hover:border-primary/50"
              }`}
              onClick={() => onLayoutSelect(layout)}
            >
              <CardContent className="p-4">
                <div className="flex items-start gap-3 mb-3">
                  <div className={`
                    p-2 rounded-lg 
                    ${isSelected ? "bg-primary text-primary-foreground" : "bg-muted"}
                  `}>
                    {layout.icon}
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="font-medium text-sm leading-tight">
                      {layout.name}
                    </h3>
                    <p className="text-xs text-muted-foreground mt-1">
                      {layout.description}
                    </p>
                  </div>
                </div>
                
                <GridPreview layout={layout} isSelected={isSelected} />
                
                <div className="mt-3 pt-3 border-t border-border">
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>{layout.positions.length} positions</span>
                    <span className="font-mono">{layout.id}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
};