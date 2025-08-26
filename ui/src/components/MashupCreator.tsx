import React, { useState } from "react";
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
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { 
  AlertTriangle, 
  CheckCircle,
  Grid2x2,
  Loader2,
} from "lucide-react";
import { MashupLayoutPicker, MashupLayout, mashupLayouts } from "./MashupLayoutPicker";

interface MashupCreatorProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (mashupId: string) => void;
}

interface MashupFormData {
  name: string;
  description: string;
  layout: string;
}

export const MashupCreator: React.FC<MashupCreatorProps> = ({
  isOpen,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [formData, setFormData] = useState<MashupFormData>({
    name: "",
    description: "",
    layout: "",
  });
  const [selectedLayout, setSelectedLayout] = useState<MashupLayout | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const resetForm = () => {
    setFormData({
      name: "",
      description: "",
      layout: "",
    });
    setSelectedLayout(null);
    setError(null);
    setSuccess(false);
    setIsCreating(false);
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const handleLayoutSelect = (layout: MashupLayout) => {
    setSelectedLayout(layout);
    setFormData(prev => ({ ...prev, layout: layout.id }));
    setError(null);
  };

  const validateForm = (): string | null => {
    if (!formData.name.trim()) {
      return "Mashup name is required";
    }
    if (formData.name.trim().length < 3) {
      return "Mashup name must be at least 3 characters long";
    }
    if (!formData.layout) {
      return "Please select a layout for your mashup";
    }
    return null;
  };

  const createMashup = async () => {
    const validationError = validateForm();
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsCreating(true);
    setError(null);

    try {
      const response = await fetch("/api/mashups", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: formData.name.trim(),
          description: formData.description.trim(),
          mashup_layout: formData.layout,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        setSuccess(true);
        
        // Show success state briefly, then close
        setTimeout(() => {
          handleClose();
          onSuccess?.(result.id);
        }, 1500);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to create mashup");
      }
    } catch (error) {
      setError("Network error occurred. Please try again.");
    } finally {
      setIsCreating(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMashup();
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Grid2x2 className="h-5 w-5" />
            Create Mashup
          </DialogTitle>
          <DialogDescription>
            Create a new mashup to combine multiple private plugins in a grid layout
          </DialogDescription>
        </DialogHeader>

        {success ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <CheckCircle className="h-12 w-12 text-green-500 mb-4" />
            <h3 className="text-lg font-semibold mb-2">Mashup Created!</h3>
            <p className="text-muted-foreground">
              Your mashup has been created successfully. You can now create instances from it.
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-6">
            {error && (
              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {/* Basic Information */}
            <div className="space-y-4">
              <div>
                <Label htmlFor="mashup-name" className="text-base font-medium">
                  Mashup Details
                </Label>
                <p className="text-sm text-muted-foreground mt-1">
                  Give your mashup a name and description
                </p>
              </div>

              <div className="space-y-4">
                <div>
                  <Label htmlFor="mashup-name">Name *</Label>
                  <Input
                    id="mashup-name"
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                    placeholder="e.g., Dashboard Overview, Weather & News"
                    className="mt-2"
                    disabled={isCreating}
                    maxLength={100}
                  />
                </div>

                <div>
                  <Label htmlFor="mashup-description">Description</Label>
                  <Textarea
                    id="mashup-description"
                    value={formData.description}
                    onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                    placeholder="Optional description of what this mashup displays"
                    className="mt-2 resize-none"
                    rows={3}
                    disabled={isCreating}
                    maxLength={500}
                  />
                </div>
              </div>
            </div>

            <Separator />

            {/* Layout Selection */}
            <MashupLayoutPicker
              selectedLayout={formData.layout}
              onLayoutSelect={handleLayoutSelect}
            />

            {/* Layout Summary */}
            {selectedLayout && (
              <div className="bg-muted/50 rounded-lg p-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/10 rounded-lg">
                    {selectedLayout.icon}
                  </div>
                  <div>
                    <h4 className="font-medium">{selectedLayout.name}</h4>
                    <p className="text-sm text-muted-foreground">
                      {selectedLayout.description} • {selectedLayout.positions.length} plugin slots
                    </p>
                  </div>
                </div>
              </div>
            )}
          </form>
        )}

        {!success && (
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={handleClose}
              disabled={isCreating}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              onClick={handleSubmit}
              disabled={isCreating || !formData.name.trim() || !formData.layout}
              className="min-w-[100px]"
            >
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Mashup"
              )}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
};