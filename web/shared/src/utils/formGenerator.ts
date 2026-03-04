/**
 * Utility for generating form fields from JSON Schema
 */

export interface FormField {
  id: string;
  label: string;
  type: 'text' | 'select' | 'textarea' | 'number' | 'boolean' | 'date' | 'email' | 'password';
  required?: boolean;
  readonly?: boolean;
  defaultValue?: any;
  options?: Array<{ value: string; label: string }>;
  rows?: number;
  placeholder?: string;
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  description?: string;
}

// JSON Schema types (simplified)
interface JSONSchema7 {
  type?: string | string[];
  properties?: Record<string, any>;
  required?: string[];
  title?: string;
  description?: string;
  default?: any;
  readOnly?: boolean;
  enum?: any[];
  enumNames?: string[];
  format?: string;
  minLength?: number;
  maxLength?: number;
  minimum?: number;
  maximum?: number;
  pattern?: string;
  items?: JSONSchema7;
}

/**
 * Service action with schema
 */
export interface ActionSchema {
  input: JSONSchema7;
  output: JSONSchema7;
}

/**
 * Mapping parameter for BPMN element
 */
export interface MappingParameter {
  name: string;
  type: string;
  required: boolean;
  description?: string;
  defaultValue?: any;
  source?: string; // for output mapping
  target?: string; // for input mapping
}

/**
 * Generated mapping for a service action
 */
export interface ServiceMapping {
  serviceName: string;
  actionName: string;
  inputParameters: MappingParameter[];
  outputParameters: MappingParameter[];
}

/**
 * Generate input/output parameters from JSON Schema
 */
export function generateMappingFromSchema(schema: JSONSchema7): MappingParameter[] {
  if (!schema.properties) return [];

  return Object.entries(schema.properties).map(([key, prop]: [string, any]) => ({
    name: key,
    type: prop.type || 'string',
    required: schema.required?.includes(key) || false,
    description: prop.description,
    defaultValue: prop.default,
  }));
}

/**
 * Generate complete service mapping from action schema
 */
export function generateServiceMapping(
  serviceName: string,
  actionName: string,
  actionSchema?: ActionSchema
): ServiceMapping {
  const inputParameters = actionSchema?.input 
    ? generateMappingFromSchema(actionSchema.input)
    : [];

  const outputParameters = actionSchema?.output
    ? generateMappingFromSchema(actionSchema.output)
    : [];

  return {
    serviceName,
    actionName,
    inputParameters,
    outputParameters,
  };
}

/**
 * Convert mapping to BPMN extension elements format
 */
export function mappingToBpmnExtension(mapping: ServiceMapping): Record<string, any> {
  return {
    serviceName: mapping.serviceName,
    actionName: mapping.actionName,
    inputParameters: mapping.inputParameters.map(p => ({
      name: p.name,
      type: p.type,
      required: p.required,
      defaultValue: p.defaultValue,
      target: p.target,
    })),
    outputParameters: mapping.outputParameters.map(p => ({
      name: p.name,
      type: p.type,
      source: p.source,
    })),
  };
}

/**
 * Parse BPMN extension elements to mapping
 */
export function bpmnExtensionToMapping(extension: Record<string, any>): ServiceMapping | null {
  if (!extension || !extension.serviceName) return null;

  return {
    serviceName: extension.serviceName,
    actionName: extension.actionName || '',
    inputParameters: (extension.inputParameters || []).map((p: any) => ({
      name: p.name,
      type: p.type,
      required: p.required || false,
      defaultValue: p.defaultValue,
      target: p.target,
    })),
    outputParameters: (extension.outputParameters || []).map((p: any) => ({
      name: p.name,
      type: p.type,
      source: p.source,
    })),
  };
}

/**
 * Generate form fields from JSON Schema
 */
export function generateFormFields(schema: JSONSchema7, variables?: Record<string, any>): FormField[] {
  if (!schema.properties) return [];

  return Object.entries(schema.properties).map(([key, prop]: [string, any]) => {
    const field: FormField = {
      id: key,
      label: prop.title || key,
      type: mapJsonSchemaType(prop.type, prop.format),
      required: schema.required?.includes(key),
      readonly: prop.readOnly || false,
      defaultValue: variables?.[key] ?? prop.default,
      description: prop.description,
      placeholder: prop.placeholder,
      minLength: prop.minLength,
      maxLength: prop.maxLength,
      min: prop.minimum,
      max: prop.maximum,
      pattern: prop.pattern,
    };

    // Handle enum for select fields
    if (field.type === 'select' && prop.enum) {
      field.options = prop.enum.map((val: any, index: number) => ({
        value: String(val),
        label: prop.enumNames?.[index] || String(val),
      }));
    }

    // Handle textarea for large text
    if (field.type === 'textarea' || (prop.maxLength && prop.maxLength > 200)) {
      field.rows = prop.maxLength ? Math.min(Math.ceil(prop.maxLength / 80), 10) : 4;
    }

    return field;
  });
}

/**
 * Map JSON Schema type to form field type
 */
function mapJsonSchemaType(type?: string | string[], format?: string): FormField['type'] {
  const t = Array.isArray(type) ? type[0] : type;
  
  switch (t) {
    case 'string':
      if (format === 'email') return 'email';
      if (format === 'password') return 'password';
      if (format === 'date') return 'date';
      return 'text';
    case 'number':
    case 'integer':
      return 'number';
    case 'boolean':
      return 'boolean';
    default:
      return 'text';
  }
}

/**
 * Validate form data against JSON Schema
 */
export function validateFormData(data: Record<string, any>, schema: JSONSchema7): {
  isValid: boolean;
  errors: Record<string, string>;
} {
  const errors: Record<string, string> = {};

  // Check required fields
  if (schema.required) {
    for (const field of schema.required) {
      if (data[field] === undefined || data[field] === null || data[field] === '') {
        errors[field] = 'This field is required';
      }
    }
  }

  // Validate field constraints
  if (schema.properties) {
    for (const [key, prop] of Object.entries(schema.properties)) {
      const value = data[key];
      if (value === undefined || value === null) continue;

      // Min length
      if (prop.minLength && typeof value === 'string' && value.length < prop.minLength) {
        errors[key] = `Minimum length is ${prop.minLength} characters`;
      }

      // Max length
      if (prop.maxLength && typeof value === 'string' && value.length > prop.maxLength) {
        errors[key] = `Maximum length is ${prop.maxLength} characters`;
      }

      // Pattern
      if (prop.pattern && typeof value === 'string') {
        const regex = new RegExp(prop.pattern);
        if (!regex.test(value)) {
          errors[key] = `Invalid format`;
        }
      }

      // Minimum
      if (prop.minimum !== undefined && typeof value === 'number' && value < prop.minimum) {
        errors[key] = `Minimum value is ${prop.minimum}`;
      }

      // Maximum
      if (prop.maximum !== undefined && typeof value === 'number' && value > prop.maximum) {
        errors[key] = `Maximum value is ${prop.maximum}`;
      }
    }
  }

  return {
    isValid: Object.keys(errors).length === 0,
    errors,
  };
}

/**
 * Convert form data to variables format
 */
export function formDataToVariables(formData: Record<string, any>): Record<string, any> {
  const variables: Record<string, any> = {};
  
  for (const [key, value] of Object.entries(formData)) {
    if (value !== undefined && value !== null && value !== '') {
      variables[key] = value;
    }
  }
  
  return variables;
}

export default {
  generateFormFields,
  validateFormData,
  formDataToVariables,
  generateMappingFromSchema,
  generateServiceMapping,
  mappingToBpmnExtension,
  bpmnExtensionToMapping,
};
