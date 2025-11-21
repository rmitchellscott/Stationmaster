import React, { useState, useEffect } from "react";
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
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  Layers, 
  ArrowLeft, 
  Save, 
  AlertCircle,
  CheckCircle2,
  Loader2 
} from "lucide-react";
import { MashupLayoutSelector } from "./MashupLayoutSelector";
import { MashupSlotAssigner } from "./MashupSlotAssigner";
import { 
  mashupService, 
  MashupLayout, 
  MashupSlotInfo, 
  AvailablePluginInstance 
} from "@/services/mashupService";

interface MashupCreatorProps {
  open: boolean;
  onClose: () => void;
  onSuccess?: (mashupId: string) => void;
}

interface ValidationErrors {
  name?: string;
  layout?: string;
  assignments?: Record<string, string>;
}

export const MashupCreator: React.FC<MashupCreatorProps> = ({
  open,
  onClose,
  onSuccess
}) => {
  const { t } = useTranslation();
  
  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedLayout, setSelectedLayout] = useState<MashupLayout | null>(null);
  const [assignments, setAssignments] = useState<Record<string, string>>({});
  
  // Data state
  const [layouts, setLayouts] = useState<MashupLayout[]>([]);
  const [slots, setSlots] = useState<MashupSlotInfo[]>([]);
  const [availablePlugins, setAvailablePlugins] = useState<AvailablePluginInstance[]>([]);
  
  // UI state
  const [currentStep, setCurrentStep] = useState<"layout" | "assignments" | "details">("layout");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({});
  
  // Load data on mount
  useEffect(() => {
    if (open) {
      loadData();
    } else {
      // Reset form when closed
      resetForm();
    }
  }, [open]);
  
  // Load slots when layout changes
  useEffect(() => {
    if (selectedLayout) {
      loadSlots(selectedLayout.id);
    }
  }, [selectedLayout]);

  const resetForm = () => {
    setName("");
    setDescription("");
    setSelectedLayout(null);
    setAssignments({});
    setSlots([]);
    setCurrentStep("layout");
    setError(null);
    setValidationErrors({});
  };

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [layoutsData, pluginsData] = await Promise.all([
        mashupService.getAvailableLayouts(),
        mashupService.getAvailablePluginInstances()
      ]);
      
      setLayouts(layoutsData);
      setAvailablePlugins(pluginsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const loadSlots = async (layoutId: string) => {
    try {
      const slotsData = await mashupService.getLayoutSlots(layoutId);
      setSlots(slotsData.slots);
      // Clear assignments when layout changes
      setAssignments({});
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load layout slots");
    }
  };

  const validateForm = (): boolean => {
    const errors: ValidationErrors = {};
    
    // Validate name
    if (!name.trim()) {
      errors.name = "Name is required";
    } else if (name.length > 255) {
      errors.name = "Name must be less than 255 characters";
    }
    
    // Validate layout selection
    if (!selectedLayout) {
      errors.layout = "Please select a layout";
    }
    
    // Validate assignments (optional - mashups can be saved without all slots filled)
    const assignmentErrors: Record<string, string> = {};
    const assignedPluginIds = Object.values(assignments);
    const duplicates = assignedPluginIds.filter((id, index) => assignedPluginIds.indexOf(id) !== index);
    
    if (duplicates.length > 0) {
      Object.keys(assignments).forEach(slotPosition => {
        if (duplicates.includes(assignments[slotPosition])) {
          assignmentErrors[slotPosition] = "Plugin is assigned to multiple slots";
        }
      });
    }
    
    if (Object.keys(assignmentErrors).length > 0) {
      errors.assignments = assignmentErrors;
    }
    
    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleLayoutSelect = (layout: MashupLayout) => {
    setSelectedLayout(layout);
    setCurrentStep("assignments");
  };

  const handleNext = () => {
    if (currentStep === "assignments") {
      setCurrentStep("details");
    }
  };

  const handleBack = () => {
    if (currentStep === "details") {
      setCurrentStep("assignments");
    } else if (currentStep === "assignments") {
      setCurrentStep("layout");
    }
  };

  const handleSave = async () => {
    if (!validateForm()) {
      return;
    }
    
    try {
      setSaving(true);
      setError(null);
      
      // Validate required data before making API calls
      if (!selectedLayout || !selectedLayout.id) {
        throw new Error('No layout selected');
      }
      
      // Create the mashup definition
      const response = await mashupService.createMashup({
        name: name.trim(),
        description: description.trim() || undefined,
        layout: selectedLayout.id,
      });
      
      if (!response || !response.mashup || !response.mashup.id) {
        console.error('Invalid mashup creation response:', response);
        throw new Error('Invalid response from mashup creation - missing mashup ID');
      }
      
      const mashupDefinitionId = response.mashup.id;
      
      // Create a plugin instance from the definition
      const instanceResponse = await fetch("/api/plugin-instances", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          definition_id: mashupDefinitionId,
          definition_type: "private",
          name: name.trim(),
          settings: {},
          refresh_interval: 3600, // Default 1 hour, will be updated based on children
        }),
      });
      
      if (!instanceResponse.ok) {
        const errorText = await instanceResponse.text();
        console.error('Instance creation failed:', {
          status: instanceResponse.status,
          statusText: instanceResponse.statusText,
          body: errorText,
          requestBody: {
            definition_id: mashupDefinitionId,
            definition_type: "private",
            name: name.trim(),
            settings: {},
            refresh_interval: 3600
          }
        });
        throw new Error(`Failed to create mashup instance: ${instanceResponse.status} ${instanceResponse.statusText}`);
      }
      
      const instanceData = await instanceResponse.json();

      if (!instanceData || !instanceData.instance || !instanceData.instance.id) {
        console.error('Invalid instance creation response:', instanceData);
        throw new Error('Invalid response from instance creation - missing instance ID');
      }

      const mashupInstanceId = instanceData.instance.id;

      // Assign children if any
      if (Object.keys(assignments).length > 0) {
        await mashupService.assignChildren(mashupInstanceId, assignments);
      }
      
      // Success - close dialog and notify parent
      onSuccess?.(mashupInstanceId);
      onClose();
      
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create mashup");
    } finally {
      setSaving(false);
    }
  };

  const canProceed = () => {
    switch (currentStep) {
      case "layout":
        return selectedLayout !== null;
      case "assignments":
        return true; // Can proceed with or without assignments
      case "details":
        return name.trim().length > 0;
      default:
        return false;
    }
  };

  const getStepTitle = () => {
    switch (currentStep) {
      case "layout":
        return "Choose Layout";
      case "assignments":
        return "Assign Plugins";
      case "details":
        return "Mashup Details";
      default:
        return "Create Mashup";
    }
  };

  if (loading) {
    return (
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className="max-w-4xl">
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 animate-spin mr-2" />
            Loading...
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <Layers className="w-5 h-5 text-primary" />
            <DialogTitle>{getStepTitle()}</DialogTitle>
          </div>
          <DialogDescription>
            Create a mashup by combining multiple private plugins into a single layout.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-6">
          {/* Step 1: Layout Selection */}
          {currentStep === "layout" && (
            <div className="space-y-4">
              <div className="text-center">
                <h3 className="text-lg font-semibold mb-2">Select a Layout</h3>
                <p className="text-sm text-muted-foreground">
                  Choose how you want to arrange your plugins on the screen.
                </p>
              </div>
              <MashupLayoutSelector
                layouts={layouts}
                selectedLayout={selectedLayout?.id || null}
                onLayoutSelect={handleLayoutSelect}
              />
            </div>
          )}

          {/* Step 2: Plugin Assignment */}
          {currentStep === "assignments" && selectedLayout && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold">Assign Plugins to Slots</h3>
                  <p className="text-sm text-muted-foreground">
                    Select plugins for each mashup slot using the dropdowns. You can leave slots empty.
                  </p>
                </div>
                <div className="text-sm text-muted-foreground">
                  Layout: <span className="font-medium">{selectedLayout.name}</span>
                </div>
              </div>
              
              {availablePlugins.length === 0 ? (
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    No private plugin instances available. Create some private plugins first to use in your mashup.
                  </AlertDescription>
                </Alert>
              ) : (
                <MashupSlotAssigner
                  layout={selectedLayout?.id || ''}
                  slots={slots}
                  availablePlugins={availablePlugins || []}
                  assignments={assignments}
                  onAssignmentsChange={setAssignments}
                  validationErrors={validationErrors.assignments}
                />
              )}
            </div>
          )}

          {/* Step 3: Details */}
          {currentStep === "details" && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold mb-2">Mashup Details</h3>
                <p className="text-sm text-muted-foreground">
                  Give your mashup a name and optional description.
                </p>
              </div>
              
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Configuration</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="name">
                      Name <span className="text-red-500">*</span>
                    </Label>
                    <Input
                      id="name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="My Awesome Mashup"
                      className={validationErrors.name ? "border-red-300" : ""}
                    />
                    {validationErrors.name && (
                      <p className="text-sm text-red-600">{validationErrors.name}</p>
                    )}
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="description">Description</Label>
                    <Textarea
                      id="description"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="Describe what this mashup shows..."
                      rows={3}
                    />
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <h4 className="font-medium">Summary</h4>
                    <div className="text-sm space-y-1">
                      <p><span className="font-medium">Layout:</span> {selectedLayout?.name}</p>
                      <p><span className="font-medium">Slots:</span> {slots.length}</p>
                      <p><span className="font-medium">Assigned:</span> {Object.keys(assignments).length}</p>
                      {Object.keys(assignments).length > 0 && (
                        <p><span className="font-medium">Refresh rate:</span> {
                          (() => {
                            const assignedPlugins = Object.values(assignments)
                              .map(id => availablePlugins.find(p => p.id === id))
                              .filter(Boolean) as AvailablePluginInstance[];
                            
                            if (assignedPlugins.length === 0) return "Not set";
                            
                            const minRate = Math.min(...assignedPlugins.map(p => p.refresh_interval));
                            return minRate >= 3600 
                              ? `${Math.floor(minRate / 3600)}h` 
                              : minRate >= 60
                              ? `${Math.floor(minRate / 60)}m`
                              : `${minRate}s`;
                          })()
                        }</p>
                      )}
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>

        <DialogFooter className="flex items-center justify-between">
          <div>
            {currentStep !== "layout" && (
              <Button variant="ghost" onClick={handleBack}>
                <ArrowLeft className="w-4 h-4 mr-2" />
                Back
              </Button>
            )}
          </div>
          
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            
            {currentStep === "details" ? (
              <Button onClick={handleSave} disabled={saving || !canProceed()}>
                {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                <Save className="w-4 h-4 mr-2" />
                Create Mashup
              </Button>
            ) : currentStep === "assignments" ? (
              <Button onClick={handleNext} disabled={!canProceed()}>
                Next: Details
              </Button>
            ) : null}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};