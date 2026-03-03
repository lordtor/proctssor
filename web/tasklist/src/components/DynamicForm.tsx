interface FormField {
  name: string;
  type: 'text' | 'textarea' | 'select' | 'boolean' | 'number' | 'date';
  label: string;
  required?: boolean;
  options?: string[];
}

interface FormSchema {
  fields: FormField[];
}

interface DynamicFormProps {
  schema?: FormSchema;
  values: Record<string, any>;
  onChange: (values: Record<string, any>) => void;
  readOnly?: boolean;
}

const styles = {
  form: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '20px',
  },
  field: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '8px',
  },
  label: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
  },
  required: {
    color: '#e74c3c',
    marginLeft: '4px',
  },
  input: {
    padding: '10px 12px',
    border: '1px solid #ddd',
    borderRadius: '4px',
    fontSize: '14px',
    outline: 'none',
    transition: 'border-color 0.2s',
  },
  textarea: {
    padding: '10px 12px',
    border: '1px solid #ddd',
    borderRadius: '4px',
    fontSize: '14px',
    outline: 'none',
    minHeight: '100px',
    resize: 'vertical' as const,
    fontFamily: 'inherit',
  },
  select: {
    padding: '10px 12px',
    border: '1px solid #ddd',
    borderRadius: '4px',
    fontSize: '14px',
    outline: 'none',
    backgroundColor: '#fff',
  },
  checkbox: {
    width: '20px',
    height: '20px',
    cursor: 'pointer',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    fontSize: '14px',
    color: '#1a1a2e',
    cursor: 'pointer',
  },
};

export default function DynamicForm({ schema, values, onChange, readOnly = false }: DynamicFormProps) {
  if (!schema || !schema.fields || schema.fields.length === 0) {
    return (
      <div style={{ color: '#999', fontSize: '14px' }}>
        No form fields defined for this task
      </div>
    );
  }

  const handleChange = (name: string, value: any) => {
    onChange({ ...values, [name]: value });
  };

  return (
    <div style={styles.form}>
      {schema.fields.map((field) => (
        <div key={field.name} style={styles.field}>
          {field.type === 'boolean' ? (
            <label style={styles.checkboxLabel}>
              <input
                type="checkbox"
                checked={!!values[field.name]}
                onChange={(e) => handleChange(field.name, e.target.checked)}
                disabled={readOnly}
                style={styles.checkbox}
              />
              {field.label}
              {field.required && <span style={styles.required}>*</span>}
            </label>
          ) : (
            <>
              <label style={styles.label}>
                {field.label}
                {field.required && <span style={styles.required}>*</span>}
              </label>
              
              {field.type === 'textarea' ? (
                <textarea
                  value={values[field.name] || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  disabled={readOnly}
                  style={styles.textarea}
                />
              ) : field.type === 'select' ? (
                <select
                  value={values[field.name] || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  disabled={readOnly}
                  style={styles.select}
                >
                  <option value="">Select...</option>
                  {field.options?.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              ) : field.type === 'date' ? (
                <input
                  type="date"
                  value={values[field.name] || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  disabled={readOnly}
                  style={styles.input}
                />
              ) : field.type === 'number' ? (
                <input
                  type="number"
                  value={values[field.name] || ''}
                  onChange={(e) => handleChange(field.name, parseFloat(e.target.value))}
                  disabled={readOnly}
                  style={styles.input}
                />
              ) : (
                <input
                  type="text"
                  value={values[field.name] || ''}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  disabled={readOnly}
                  style={styles.input}
                />
              )}
            </>
          )}
        </div>
      ))}
    </div>
  );
}
