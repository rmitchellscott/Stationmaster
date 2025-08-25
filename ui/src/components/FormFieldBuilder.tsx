import React, { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  AlertTriangle,
  HelpCircle,
} from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";

const EXAMPLE_YAML = `# Example form fields for a weather plugin
- keyname: api_key
  field_type: string
  name: API Key
  description: Find this at weather-api.com/settings
  placeholder: abc123xyz
  optional: false

- keyname: location
  field_type: string
  name: Location
  description: City name or coordinates
  placeholder: New York
  optional: false

- keyname: temperature_unit
  field_type: select
  name: Temperature Unit
  description: Choose temperature display unit
  optional: true
  default: fahrenheit
  options:
    - Fahrenheit: fahrenheit
    - Celsius: celsius`;

interface FormFieldBuilderProps {
  value: string; // YAML string
  onChange: (yaml: string) => void;
  onValidationChange: (isValid: boolean, errors: string[]) => void;
}

export function FormFieldBuilder({ value, onChange, onValidationChange }: FormFieldBuilderProps) {
  const [errors, setErrors] = useState<string[]>([]);
  const [showExample, setShowExample] = useState(false);

  // Validate YAML on change
  useEffect(() => {
    validateYAML(value);
  }, [value]);

  const validateYAML = (yamlContent: string) => {
    const newErrors: string[] = [];
    
    if (!yamlContent.trim()) {
      setErrors([]);
      onValidationChange(true, []);
      return;
    }

    try {
      // Basic YAML validation - check for proper structure
      const lines = yamlContent.split('\n');
      let fieldCount = 0;
      let currentFieldHasName = false;
      let currentFieldHasKeyname = false;
      let currentFieldHasType = false;
      
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) continue;
        
        if (trimmed.startsWith('- keyname:')) {
          if (fieldCount > 0 && (!currentFieldHasName || !currentFieldHasKeyname || !currentFieldHasType)) {
            newErrors.push(`Field ${fieldCount}: Missing required fields (keyname, field_type, name)`);
          }
          fieldCount++;
          currentFieldHasKeyname = !!trimmed.replace('- keyname:', '').trim();
          currentFieldHasName = false;
          currentFieldHasType = false;
        } else if (trimmed.startsWith('name:')) {
          currentFieldHasName = !!trimmed.replace('name:', '').trim();
        } else if (trimmed.startsWith('field_type:')) {
          currentFieldHasType = !!trimmed.replace('field_type:', '').trim();
        }
      }
      
      // Check last field
      if (fieldCount > 0 && (!currentFieldHasName || !currentFieldHasKeyname || !currentFieldHasType)) {
        newErrors.push(`Field ${fieldCount}: Missing required fields (keyname, field_type, name)`);
      }
      
    } catch (error) {
      newErrors.push(`Invalid YAML format: ${error}`);
    }
    
    setErrors(newErrors);
    onValidationChange(newErrors.length === 0, newErrors);
  };

  const handleYAMLChange = (newValue: string) => {
    onChange(newValue);
  };

  const insertExample = () => {
    onChange(EXAMPLE_YAML);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Form Fields (YAML)</h3>
        <div className="flex gap-2">
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => setShowExample(!showExample)}
          >
            <HelpCircle className="h-4 w-4 mr-2" />
            {showExample ? 'Hide' : 'Show'} Example
          </Button>
          <Button 
            variant="outline" 
            size="sm"
            onClick={insertExample}
          >
            Insert Example
          </Button>
        </div>
      </div>

      <p className="text-sm text-muted-foreground">
        Define form fields in YAML format. These values will be available in templates as {`{{ field_keyname }}`}.
      </p>

      {showExample && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Example YAML Format</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-xs bg-muted p-3 rounded overflow-x-auto whitespace-pre-wrap">
              {EXAMPLE_YAML}
            </pre>
          </CardContent>
        </Card>
      )}

      {errors.length > 0 && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <div className="space-y-1">
              <div className="font-medium">YAML validation errors:</div>
              {errors.map((error, index) => (
                <div key={index} className="text-sm">• {error}</div>
              ))}
            </div>
          </AlertDescription>
        </Alert>
      )}

      <div>
        <Label htmlFor="form-fields-yaml">YAML Configuration</Label>
        <Textarea
          id="form-fields-yaml"
          value={value}
          onChange={(e) => handleYAMLChange(e.target.value)}
          placeholder={`# Define your form fields here
# Example:
- keyname: team
  field_type: select
  name: Team
  options:
    - Boston Bruins: BOS
    - New York Rangers: NYR
  default: BOS`}
          className="mt-2 font-mono text-sm"
          rows={12}
        />
        <p className="text-xs text-muted-foreground mt-2">
          Supported field types: string, text, number, password, url, date, time, select, code, copyable
        </p>
      </div>
    </div>
  );
}