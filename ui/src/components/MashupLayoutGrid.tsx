import React from "react";

// Mashup layout grid representations
export const getMashupLayoutGrid = (layoutId: string, size: 'tiny' | 'small' | 'normal' = 'normal', style: 'subtle' | 'prominent' = 'subtle') => {
  const backgroundClass = style === 'prominent' ? 'bg-primary/30' : 'bg-muted';
  const baseClasses = `border border-dashed border-muted-foreground/30 rounded flex items-center justify-center ${backgroundClass}`;
  
  let dimensions, gap;
  if (size === 'tiny') {
    dimensions = 'h-6 w-10';
    gap = 'gap-0.5';
  } else if (size === 'small') {
    dimensions = 'h-10 w-16';
    gap = 'gap-0.5';
  } else {
    dimensions = 'h-16 w-24';
    gap = 'gap-1';
  }
  
  switch (layoutId) {
    case "1Lx1R": // Left | Right
      return (
        <div className={`grid grid-cols-2 ${gap} ${dimensions}`}>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
        </div>
      );
    case "1Tx1B": // Top / Bottom
      return (
        <div className={`grid grid-rows-2 ${gap} ${dimensions}`}>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
        </div>
      );
    case "1Lx2R": // Left | Top-Right / Bottom-Right
      return (
        <div className={`grid grid-cols-2 ${gap} ${dimensions}`}>
          <div className={baseClasses}></div>
          <div className={`grid grid-rows-2 ${gap}`}>
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
        </div>
      );
    case "2Lx1R": // Left-Top / Left-Bottom | Right
      return (
        <div className={`grid grid-cols-2 ${gap} ${dimensions}`}>
          <div className={`grid grid-rows-2 ${gap}`}>
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
          <div className={baseClasses}></div>
        </div>
      );
    case "2Tx1B": // Top-Left | Top-Right / Bottom
      return (
        <div className={`grid grid-rows-2 ${gap} ${dimensions}`}>
          <div className={`grid grid-cols-2 ${gap}`}>
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
          <div className={baseClasses}></div>
        </div>
      );
    case "1Tx2B": // Top / Bottom-Left | Bottom-Right
      return (
        <div className={`grid grid-rows-2 ${gap} ${dimensions}`}>
          <div className={baseClasses}></div>
          <div className={`grid grid-cols-2 ${gap}`}>
            <div className={baseClasses}></div>
            <div className={baseClasses}></div>
          </div>
        </div>
      );
    case "2x2": // Quadrant grid
      return (
        <div className={`grid grid-cols-2 grid-rows-2 ${gap} ${dimensions}`}>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
          <div className={baseClasses}></div>
        </div>
      );
    default:
      console.warn('Unknown layout ID:', layoutId);
      return <div className={`${baseClasses} ${dimensions}`}>?</div>;
  }
};

// Single plugin layout representation
export const getSinglePluginGrid = (style: 'subtle' | 'prominent' = 'subtle') => {
  const backgroundClass = style === 'prominent' ? 'bg-primary/30' : 'bg-muted';
  const baseClasses = `border border-dashed border-muted-foreground/30 rounded text-xs flex items-center justify-center text-muted-foreground/60 font-medium ${backgroundClass}`;
  return <div className={`${baseClasses} h-16 w-24`}></div>;
};