import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { 
  Grid2x2, 
  Columns2, 
  Rows2, 
  LayoutGrid,
  Split,
  AlignVerticalJustifyCenter,
  AlignHorizontalJustifyCenter
} from "lucide-react";
import { MashupLayout } from "@/services/mashupService";

interface MashupLayoutSelectorProps {
  layouts: MashupLayout[];
  selectedLayout: string | null;
  onLayoutSelect: (layout: MashupLayout) => void;
  disabled?: boolean;
}

// Layout icon mapping
const getLayoutIcon = (layoutId: string) => {
  const iconProps = { className: "w-8 h-8 text-muted-foreground" };
  
  switch (layoutId) {
    case "1Lx1R":
      return <Columns2 {...iconProps} />;
    case "1Tx1B":
      return <Rows2 {...iconProps} />;
    case "1Lx2R":
      return <AlignVerticalJustifyCenter {...iconProps} />;
    case "2Lx1R":
      return <AlignVerticalJustifyCenter {...iconProps} style={{ transform: "rotate(180deg)" }} />;
    case "2Tx1B":
      return <AlignHorizontalJustifyCenter {...iconProps} />;
    case "1Tx2B":
      return <AlignHorizontalJustifyCenter {...iconProps} style={{ transform: "rotate(180deg)" }} />;
    case "2x2":
      return <Grid2x2 {...iconProps} />;
    default:
      return <LayoutGrid {...iconProps} />;
  }
};

// Visual grid representation for each layout
const getLayoutGrid = (layoutId: string) => {
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

export const MashupLayoutSelector: React.FC<MashupLayoutSelectorProps> = ({
  layouts,
  selectedLayout,
  onLayoutSelect,
  disabled = false
}) => {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {layouts.map((layout) => (
          <Card
            key={layout.id}
            className={`cursor-pointer transition-all duration-200 ${
              selectedLayout === layout.id 
                ? "ring-2 ring-primary border-primary bg-primary/5" 
                : "hover:shadow-md hover:border-muted-foreground/30"
            } ${disabled ? "opacity-50 cursor-not-allowed" : ""}`}
            onClick={() => !disabled && onLayoutSelect(layout)}
          >
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium">{layout.name}</CardTitle>
                <Badge variant="secondary" className="text-xs">
                  {layout.slots} slots
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-center">
                {getLayoutGrid(layout.id)}
              </div>
              <div className="flex items-center gap-2">
                {getLayoutIcon(layout.id)}
                <p className="text-xs text-muted-foreground flex-1">
                  {layout.description}
                </p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      {selectedLayout && (
        <div className="text-sm text-muted-foreground">
          Selected: <span className="font-medium">{layouts.find(l => l.id === selectedLayout)?.name}</span>
        </div>
      )}
    </div>
  );
};