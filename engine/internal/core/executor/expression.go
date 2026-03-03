// internal/core/executor/expression.go
package executor

import (
	"fmt"
	"strconv"
	"strings"
)

// ExpressionEvaluator evaluates BPMN expressions
type ExpressionEvaluator struct {
	variables map[string]interface{}
}

func NewExpressionEvaluator(vars map[string]interface{}) *ExpressionEvaluator {
	return &ExpressionEvaluator{variables: vars}
}

// Evaluate evaluates a condition expression
func (e *ExpressionEvaluator) Evaluate(expression string) (bool, error) {
	expression = strings.TrimSpace(expression)
	expression = strings.TrimPrefix(expression, "${")
	expression = strings.TrimSuffix(expression, "}")

	// Сравнение: ==, !=, >, <, >=, <=
	if strings.Contains(expression, "==") {
		return e.evaluateEquals(expression)
	}
	if strings.Contains(expression, "!=") {
		return e.evaluateNotEquals(expression)
	}
	if strings.Contains(expression, ">=") {
		return e.evaluateGreaterOrEqual(expression)
	}
	if strings.Contains(expression, "<=") {
		return e.evaluateLessOrEqual(expression)
	}
	if strings.Contains(expression, ">") {
		return e.evaluateGreater(expression)
	}
	if strings.Contains(expression, "<") {
		return e.evaluateLess(expression)
	}

	// Просто переменная - truthy check
	return e.evaluateTruthy(expression), nil
}

func (e *ExpressionEvaluator) evaluateEquals(expr string) (bool, error) {
	parts := strings.SplitN(expr, "==", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	leftVal := e.getValue(left)
	rightVal := e.parseValue(right)

	return fmt.Sprintf("%v", leftVal) == fmt.Sprintf("%v", rightVal), nil
}

func (e *ExpressionEvaluator) evaluateNotEquals(expr string) (bool, error) {
	parts := strings.SplitN(expr, "!=", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	leftVal := e.getValue(left)
	rightVal := e.parseValue(right)

	return fmt.Sprintf("%v", leftVal) != fmt.Sprintf("%v", rightVal), nil
}

func (e *ExpressionEvaluator) evaluateGreater(expr string) (bool, error) {
	parts := strings.SplitN(expr, ">", 2)
	leftNum, err := e.toNumber(strings.TrimSpace(parts[0]))
	if err != nil {
		return false, err
	}
	rightNum, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return false, err
	}
	return leftNum > rightNum, nil
}

func (e *ExpressionEvaluator) evaluateLess(expr string) (bool, error) {
	parts := strings.SplitN(expr, "<", 2)
	leftNum, err := e.toNumber(strings.TrimSpace(parts[0]))
	if err != nil {
		return false, err
	}
	rightNum, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return false, err
	}
	return leftNum < rightNum, nil
}

func (e *ExpressionEvaluator) evaluateGreaterOrEqual(expr string) (bool, error) {
	parts := strings.SplitN(expr, ">=", 2)
	leftNum, err := e.toNumber(strings.TrimSpace(parts[0]))
	if err != nil {
		return false, err
	}
	rightNum, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return false, err
	}
	return leftNum >= rightNum, nil
}

func (e *ExpressionEvaluator) evaluateLessOrEqual(expr string) (bool, error) {
	parts := strings.SplitN(expr, "<=", 2)
	leftNum, err := e.toNumber(strings.TrimSpace(parts[0]))
	if err != nil {
		return false, err
	}
	rightNum, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return false, err
	}
	return leftNum <= rightNum, nil
}

func (e *ExpressionEvaluator) evaluateTruthy(name string) bool {
	val, exists := e.variables[name]
	if !exists {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return val != nil
	}
}

func (e *ExpressionEvaluator) getValue(name string) interface{} {
	if val, exists := e.variables[name]; exists {
		return val
	}
	return nil
}

func (e *ExpressionEvaluator) parseValue(s string) interface{} {
	s = strings.TrimSpace(s)

	// Boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Number
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}

	// String (с кавычками)
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return strings.Trim(s, `"`)
	}
	if strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) {
		return strings.Trim(s, `'`)
	}

	// Переменная
	return e.getValue(s)
}

func (e *ExpressionEvaluator) toNumber(name string) (float64, error) {
	val := e.getValue(name)
	if val == nil {
		return 0, fmt.Errorf("variable %s not found", name)
	}

	switch v := val.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", val)
	}
}
