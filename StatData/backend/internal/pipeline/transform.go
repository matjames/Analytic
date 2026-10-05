package pipeline

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
)

// cloneRecord copies one row so stage executors never mutate shared state.
func cloneRecord(rec map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{}, len(rec)+1)
	for k, v := range rec {
		clone[k] = v
	}
	return clone
}

// decodeOperations reads the "operations" list from a stage config. Each
// entry is a map describing one real per-record transformation step.
func decodeOperations(config map[string]interface{}) ([]map[string]interface{}, error) {
	raw, ok := config["operations"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("transform stage has no operations configured")
	}
	ops := make([]map[string]interface{}, 0, len(raw))
	for i, item := range raw {
		op, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("operation %d is not an object", i)
		}
		ops = append(ops, op)
	}
	return ops, nil
}

// decodeStringList reads a string list from a stage config.
func decodeStringList(config map[string]interface{}, key string) ([]string, error) {
	raw, ok := config[key].([]interface{})
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("no '%s' list configured", key)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// hashValue hashes a scalar value (SHA-256 hex) for anonymization stages.
func hashValue(v interface{}) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(normalizeScalar(v))))
}

// normalizeScalar renders any JSON scalar for comparison and hashing.
func normalizeScalar(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", v)
}

// applyTransformOperations applies one ordered operation list to a single
// record. The returned bool reports whether the record survives filters.
// Supported operations: filter, rename, remove_fields, set_constant,
// cast_number. Unknown operation types fail the record rather than passing it
// silently.
func applyTransformOperations(rec map[string]interface{}, ops []map[string]interface{}) (bool, map[string]interface{}, error) {
	out := cloneRecord(rec)
	for _, op := range ops {
		opType, _ := op["type"].(string)
		switch opType {
		case "filter":
			field, _ := op["field"].(string)
			filterOp, _ := op["op"].(string)
			if !evaluateFilter(out, field, filterOp, op["value"]) {
				return false, nil, nil
			}
		case "rename":
			from, _ := op["from"].(string)
			to, _ := op["to"].(string)
			if from == "" || to == "" {
				return false, nil, fmt.Errorf("rename requires 'from' and 'to'")
			}
			if v, ok := out[from]; ok {
				out[to] = v
				delete(out, from)
			}
		case "remove_fields":
			fields, err := opStringList(op, "fields")
			if err != nil {
				return false, nil, err
			}
			for _, f := range fields {
				delete(out, f)
			}
		case "set_constant":
			field, _ := op["field"].(string)
			if field == "" {
				return false, nil, fmt.Errorf("set_constant requires a 'field'")
			}
			out[field] = op["value"]
		case "cast_number":
			field, _ := op["field"].(string)
			if field == "" {
				return false, nil, fmt.Errorf("cast_number requires a 'field'")
			}
			v, ok := out[field]
			if !ok || v == nil {
				return false, nil, nil // reject rows with missing values
			}
			num, ok := toFloat64(v)
			if !ok {
				return false, nil, nil // reject rows with uncastable values
			}
			out[field] = num
		default:
			return false, nil, fmt.Errorf("unknown operation type %q", opType)
		}
	}
	return true, out, nil
}

// evaluateFilter compares a record field against a configured value.
// Unknown comparators reject rather than guess.
func evaluateFilter(rec map[string]interface{}, field, filterOp string, value interface{}) bool {
	v, exists := rec[field]
	switch filterOp {
	case "not_null":
		return exists && v != nil && v != ""
	case "eq":
		return exists && normalizeScalar(v) == normalizeScalar(value)
	case "neq":
		return !exists || normalizeScalar(v) != normalizeScalar(value)
	case "gt", "gte", "lt", "lte":
		a, okA := toFloat64(v)
		b, okB := toFloat64(value)
		if !okA || !okB {
			return false
		}
		switch filterOp {
		case "gt":
			return a > b
		case "gte":
			return a >= b
		case "lt":
			return a < b
		case "lte":
			return a <= b
		}
		return false
	default:
		return false
	}
}

// opStringList reads a per-operation string list.
func opStringList(op map[string]interface{}, key string) ([]string, error) {
	raw, ok := op[key].([]interface{})
	if !ok {
		return nil, fmt.Errorf("operation requires a '%s' list", key)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// toFloat64 converts JSON-decoded numbers (and numeric strings) to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}