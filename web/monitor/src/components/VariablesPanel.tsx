interface VariablesPanelProps {
  variables: Record<string, any>;
}

const styles = {
  variable: {
    padding: '10px',
    marginBottom: '8px',
    borderRadius: '4px',
    backgroundColor: '#f5f5f5',
  },
  variableName: {
    fontSize: '12px',
    fontWeight: 500,
    color: '#666',
  },
  variableValue: {
    fontSize: '14px',
    color: '#1a1a2e',
    marginTop: '4px',
    wordBreak: 'break-word' as const,
  },
};

export default function VariablesPanel({ variables }: VariablesPanelProps) {
  const variableEntries = Object.entries(variables || {});

  if (variableEntries.length === 0) {
    return (
      <div style={{ color: '#999', fontSize: '14px' }}>
        No variables
      </div>
    );
  }

  const formatValue = (value: any): string => {
    if (value === null || value === undefined) {
      return 'null';
    }
    if (typeof value === 'object') {
      return JSON.stringify(value);
    }
    return String(value);
  };

  return (
    <div>
      {variableEntries.map(([name, value]) => (
        <div key={name} style={styles.variable}>
          <div style={styles.variableName}>{name}</div>
          <div style={styles.variableValue}>{formatValue(value)}</div>
        </div>
      ))}
    </div>
  );
}
