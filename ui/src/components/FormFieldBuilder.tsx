import React, { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import {
  Plus,
  Trash2,
  Move,
  Eye,
  Code2,
  AlertTriangle,
  CheckCircle,
} from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";

// TRMNL supported field types
const FIELD_TYPES = [
  { value: "string", label: "Text Input", description: "Single line text input" },
  { value: "text", label: "Text Area", description: "Multi-line text input" },
  { value: "number", label: "Number", description: "Numeric input" },
  { value: "password", label: "Password", description: "Password input field" },
  { value: "url", label: "URL", description: "URL input with validation" },
  { value: "author_bio", label: "Author Bio", description: "Author biography field" },
  { value: "date", label: "Date", description: "Date picker" },
  { value: "time", label: "Time", description: "Time picker" },
  { value: "time_zone", label: "Time Zone", description: "Timezone selector" },
  { value: "select", label: "Select", description: "Dropdown selection" },
  { value: "code", label: "Code", description: "Code editor" },
  { value: "copyable", label: "Copyable", description: "Read-only copyable field" },
  { value: "copyable_webhook_url", label: "Copyable Webhook URL", description: "Webhook URL field" },
  { value: "xhrSelect", label: "XHR Select", description: "Dynamic options from API" },
  { value: "xhrSelectSearch", label: "XHR Select Search", description: "Searchable dynamic select" },
];

interface FormFieldOption {
  label: string;
  value: string;
}

interface FormField {
  keyname: string;
  field_type: string;
  name: string;
  description: string;
  optional: boolean;
  default: string | number | boolean | null;
  placeholder: string;
  help_text: string;
  options: FormFieldOption[];
  validation: Record<string, any>;
}

interface FormFieldBuilderProps {
  value: string; // YAML string
  onChange: (yaml: string) => void;
  onValidationChange: (isValid: boolean, errors: string[]) => void;
}

export function FormFieldBuilder({ value, onChange, onValidationChange }: FormFieldBuilderProps) {
  const [fields, setFields] = useState<FormField[]>([]);
  const [errors, setErrors] = useState<string[]>([]);
  const [previewMode, setPreviewMode] = useState(false);

  // Parse YAML on mount and value changes
  useEffect(() => {
    if (value.trim()) {
      try {
        // Simple YAML parser for our structure
        parseYAMLToFields(value);
      } catch (error) {
        setErrors([`Invalid YAML: ${error}`]);
        onValidationChange(false, [`Invalid YAML: ${error}`]);
      }
    } else {
      setFields([]);
      setErrors([]);
      onValidationChange(true, []);
    }
  }, [value]);

  // Generate YAML when fields change
  useEffect(() => {
    const yaml = generateYAML(fields);
    onChange(yaml);
    validateFields();
  }, [fields]);

  const parseYAMLToFields = (yaml: string) => {
    // Simple YAML parsing - in a real implementation, use a proper YAML library
    try {
      const lines = yaml.split('\n');
      const parsedFields: FormField[] = [];
      let currentField: Partial<FormField> | null = null;
      let inOptions = false;
      let currentOptions: FormFieldOption[] = [];

      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) continue;

        if (trimmed.startsWith('- keyname:')) {
          if (currentField) {
            if (inOptions) {
              currentField.options = currentOptions;
            }
            parsedFields.push(currentField as FormField);
          }
          currentField = {
            keyname: trimmed.replace('- keyname:', '').trim(),
            options: [],
            validation: {},
          };
          inOptions = false;
          currentOptions = [];
        } else if (currentField && trimmed.includes(':')) {
          const [key, ...valueParts] = trimmed.split(':');
          const value = valueParts.join(':').trim();

          switch (key.trim()) {
            case 'field_type':
              currentField.field_type = value;
              break;
            case 'name':
              currentField.name = value;
              break;
            case 'description':
              currentField.description = value;
              break;
            case 'optional':
              currentField.optional = value === 'true';
              break;
            case 'default':
              currentField.default = value;
              break;
            case 'placeholder':
              currentField.placeholder = value;
              break;
            case 'help_text':
              currentField.help_text = value;
              break;
            case 'options':
              inOptions = true;
              break;
          }
        } else if (inOptions && trimmed.startsWith('- label:')) {
          const label = trimmed.replace('- label:', '').trim();
          currentOptions.push({ label, value: '' });
        } else if (inOptions && trimmed.startsWith('value:') && currentOptions.length > 0) {
          const value = trimmed.replace('value:', '').trim();
          currentOptions[currentOptions.length - 1].value = value;
        }
      }

      if (currentField) {
        if (inOptions) {
          currentField.options = currentOptions;
        }
        parsedFields.push(currentField as FormField);
      }

      setFields(parsedFields);
      setErrors([]);
    } catch (error) {
      throw new Error(`Failed to parse YAML: ${error}`);
    }
  };

  const generateYAML = (fields: FormField[]): string => {
    if (fields.length === 0) return '';

    const yamlLines: string[] = [];

    fields.forEach((field) => {
      yamlLines.push(`- keyname: ${field.keyname}`);
      yamlLines.push(`  field_type: ${field.field_type}`);
      yamlLines.push(`  name: ${field.name}`);
      if (field.description) yamlLines.push(`  description: ${field.description}`);
      if (field.optional) yamlLines.push(`  optional: true`);
      if (field.default !== null && field.default !== undefined && field.default !== '') {
        yamlLines.push(`  default: ${field.default}`);
      }
      if (field.placeholder) yamlLines.push(`  placeholder: ${field.placeholder}`);
      if (field.help_text) yamlLines.push(`  help_text: ${field.help_text}`);
      
      if (field.options && field.options.length > 0) {
        yamlLines.push(`  options:`);
        field.options.forEach((option) => {
          yamlLines.push(`    - label: ${option.label}`);
          yamlLines.push(`      value: ${option.value}`);
        });
      }
      yamlLines.push(''); // Empty line between fields
    });

    return yamlLines.join('\n');
  };

  const validateFields = () => {
    const newErrors: string[] = [];
    const usedKeys = new Set<string>();

    fields.forEach((field, index) => {
      if (!field.keyname?.trim()) {
        newErrors.push(`Field ${index + 1}: keyname is required`);
      } else if (usedKeys.has(field.keyname)) {
        newErrors.push(`Field ${index + 1}: keyname "${field.keyname}" is already used`);
      } else {
        usedKeys.add(field.keyname);
      }

      if (!field.field_type?.trim()) {
        newErrors.push(`Field ${index + 1}: field_type is required`);
      }

      if (!field.name?.trim()) {
        newErrors.push(`Field ${index + 1}: name is required`);
      }

      if (field.field_type === 'select' && (!field.options || field.options.length === 0)) {
        newErrors.push(`Field ${index + 1}: select field requires at least one option`);
      }
    });

    setErrors(newErrors);
    onValidationChange(newErrors.length === 0, newErrors);
  };

  const addField = () => {
    const newField: FormField = {
      keyname: `field_${fields.length + 1}`,
      field_type: 'string',
      name: `Field ${fields.length + 1}`,
      description: '',
      optional: false,
      default: null,
      placeholder: '',
      help_text: '',
      options: [],
      validation: {},
    };
    setFields([...fields, newField]);
  };

  const removeField = (index: number) => {
    setFields(fields.filter((_, i) => i !== index));
  };

  const updateField = (index: number, updates: Partial<FormField>) => {
    const newFields = [...fields];
    newFields[index] = { ...newFields[index], ...updates };
    setFields(newFields);
  };

  const moveField = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= fields.length) return;

    const newFields = [...fields];
    [newFields[index], newFields[newIndex]] = [newFields[newIndex], newFields[index]];
    setFields(newFields);
  };

  const addOption = (fieldIndex: number) => {
    const newFields = [...fields];
    if (!newFields[fieldIndex].options) {
      newFields[fieldIndex].options = [];
    }
    newFields[fieldIndex].options.push({ label: '', value: '' });
    setFields(newFields);
  };

  const removeOption = (fieldIndex: number, optionIndex: number) => {
    const newFields = [...fields];
    newFields[fieldIndex].options = newFields[fieldIndex].options.filter(
      (_, i) => i !== optionIndex
    );
    setFields(newFields);
  };

  const updateOption = (fieldIndex: number, optionIndex: number, updates: Partial<FormFieldOption>) => {
    const newFields = [...fields];
    newFields[fieldIndex].options[optionIndex] = {
      ...newFields[fieldIndex].options[optionIndex],
      ...updates,
    };
    setFields(newFields);
  };

  if (previewMode) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium">Form Field Preview</h3>
          <Button variant="outline" onClick={() => setPreviewMode(false)}>
            <Code2 className="h-4 w-4 mr-2" />
            Back to Editor
          </Button>
        </div>
        
        {fields.length === 0 ? (
          <p className="text-muted-foreground">No form fields defined</p>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>Plugin Settings Form</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {fields.map((field) => (
                <div key={field.keyname} className="space-y-2">
                  <Label htmlFor={field.keyname}>
                    {field.name}
                    {!field.optional && <span className="text-destructive ml-1">*</span>}
                  </Label>
                  {field.description && (
                    <p className="text-sm text-muted-foreground">{field.description}</p>
                  )}
                  {renderPreviewField(field)}
                  {field.help_text && (
                    <p className="text-xs text-muted-foreground">{field.help_text}</p>
                  )}
                </div>
              ))}
            </CardContent>
          </Card>
        )}
      </div>
    );
  }

  const renderPreviewField = (field: FormField) => {
    switch (field.field_type) {
      case 'text':
      case 'code':
        return (
          <Textarea
            id={field.keyname}
            placeholder={field.placeholder}
            defaultValue={field.default as string}
            disabled
          />
        );
      case 'number':
        return (
          <Input
            id={field.keyname}
            type="number"
            placeholder={field.placeholder}
            defaultValue={field.default as number}
            disabled
          />
        );
      case 'password':
        return (
          <Input
            id={field.keyname}
            type="password"
            placeholder={field.placeholder}
            disabled
          />
        );
      case 'date':
        return (
          <Input
            id={field.keyname}
            type="date"
            defaultValue={field.default as string}
            disabled
          />
        );
      case 'time':
        return (
          <Input
            id={field.keyname}
            type="time"
            defaultValue={field.default as string}
            disabled
          />
        );
      case 'select':
        return (
          <Select disabled>
            <SelectTrigger>
              <SelectValue placeholder="Select an option..." />
            </SelectTrigger>
            <SelectContent>
              {field.options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        );
      default:
        return (
          <Input
            id={field.keyname}
            placeholder={field.placeholder}
            defaultValue={field.default as string}
            disabled
          />
        );
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Form Fields Builder</h3>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setPreviewMode(true)}>
            <Eye className="h-4 w-4 mr-2" />
            Preview
          </Button>
          <Button variant="outline" onClick={addField}>
            <Plus className="h-4 w-4 mr-2" />
            Add Field
          </Button>
        </div>
      </div>

      {errors.length > 0 && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            <div className="space-y-1">
              <div className="font-medium">Form field validation errors:</div>
              {errors.map((error, index) => (
                <div key={index} className="text-sm">• {error}</div>
              ))}
            </div>
          </AlertDescription>
        </Alert>
      )}

      {fields.length === 0 ? (
        <Card>
          <CardContent className="flex items-center justify-center py-8">
            <div className="text-center">
              <p className="text-muted-foreground mb-4">No form fields defined</p>
              <Button onClick={addField}>
                <Plus className="h-4 w-4 mr-2" />
                Add Your First Field
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {fields.map((field, index) => (
            <Card key={index}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">
                    {field.name || `Field ${index + 1}`}
                    {!field.optional && <span className="text-destructive ml-1">*</span>}
                  </CardTitle>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => moveField(index, 'up')}
                      disabled={index === 0}
                    >
                      <Move className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => moveField(index, 'down')}
                      disabled={index === fields.length - 1}
                    >
                      <Move className="h-4 w-4 rotate-180" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeField(index)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor={`keyname-${index}`}>Key Name *</Label>
                    <Input
                      id={`keyname-${index}`}
                      value={field.keyname}
                      onChange={(e) => updateField(index, { keyname: e.target.value })}
                      placeholder="field_key"
                      className="mt-1"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Used in templates as {`{{ ${field.keyname} }}`}
                    </p>
                  </div>
                  <div>
                    <Label htmlFor={`field-type-${index}`}>Field Type *</Label>
                    <Select
                      value={field.field_type}
                      onValueChange={(value) => updateField(index, { field_type: value })}
                    >
                      <SelectTrigger className="mt-1">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {FIELD_TYPES.map((type) => (
                          <SelectItem key={type.value} value={type.value}>
                            <div>
                              <div className="font-medium">{type.label}</div>
                              <div className="text-xs text-muted-foreground">{type.description}</div>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor={`name-${index}`}>Display Name *</Label>
                    <Input
                      id={`name-${index}`}
                      value={field.name}
                      onChange={(e) => updateField(index, { name: e.target.value })}
                      placeholder="Field Name"
                      className="mt-1"
                    />
                  </div>
                  <div>
                    <Label htmlFor={`description-${index}`}>Description</Label>
                    <Input
                      id={`description-${index}`}
                      value={field.description}
                      onChange={(e) => updateField(index, { description: e.target.value })}
                      placeholder="Field description"
                      className="mt-1"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor={`placeholder-${index}`}>Placeholder</Label>
                    <Input
                      id={`placeholder-${index}`}
                      value={field.placeholder}
                      onChange={(e) => updateField(index, { placeholder: e.target.value })}
                      placeholder="Placeholder text"
                      className="mt-1"
                    />
                  </div>
                  <div>
                    <Label htmlFor={`default-${index}`}>Default Value</Label>
                    <Input
                      id={`default-${index}`}
                      value={field.default as string}
                      onChange={(e) => updateField(index, { default: e.target.value })}
                      placeholder="Default value"
                      className="mt-1"
                    />
                  </div>
                </div>

                <div>
                  <Label htmlFor={`help-text-${index}`}>Help Text</Label>
                  <Textarea
                    id={`help-text-${index}`}
                    value={field.help_text}
                    onChange={(e) => updateField(index, { help_text: e.target.value })}
                    placeholder="Additional help text for users"
                    className="mt-1"
                    rows={2}
                  />
                </div>

                <div className="flex items-center space-x-2">
                  <Checkbox
                    id={`optional-${index}`}
                    checked={field.optional}
                    onCheckedChange={(checked) => updateField(index, { optional: !!checked })}
                  />
                  <Label htmlFor={`optional-${index}`}>Optional field</Label>
                </div>

                {field.field_type === 'select' && (
                  <div>
                    <Separator className="my-4" />
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <Label>Options</Label>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => addOption(index)}
                        >
                          <Plus className="h-4 w-4 mr-2" />
                          Add Option
                        </Button>
                      </div>
                      {field.options.map((option, optionIndex) => (
                        <div key={optionIndex} className="flex gap-2">
                          <Input
                            placeholder="Label"
                            value={option.label}
                            onChange={(e) =>
                              updateOption(index, optionIndex, { label: e.target.value })
                            }
                          />
                          <Input
                            placeholder="Value"
                            value={option.value}
                            onChange={(e) =>
                              updateOption(index, optionIndex, { value: e.target.value })
                            }
                          />
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => removeOption(index, optionIndex)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}