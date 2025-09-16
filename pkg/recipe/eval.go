package recipe

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// Eval evaluates a path expression against a Value object and returns the
// result. The path can contain references to fields in the Value using ${}
// syntax. For example: "${memory.foo} > 5" or "${memory.array[0]}"
//
// TODO: Eval is currently used for evaluating condition fields in recipes, but
// will be extended to support evaluating all values in pipeline recipes in the
// future.
func Eval(path string, value format.Value) (any, error) {
	path, err := sanitizePath(path)
	if err != nil {
		return false, err
	}
	expr, err := parser.ParseExpr(path)
	if err != nil {
		return false, err
	}
	jsonValue, err := value.ToJSONValue()
	if err != nil {
		return false, fmt.Errorf("eval: %w", err)
	}
	conditionMemory := map[string]any{
		"memory": jsonValue,
	}
	res, err := evalExpr(expr, conditionMemory)
	if err != nil {
		return false, err
	}
	return res, nil
}

// evalExpr evaluates an AST expression node against a map of values.
// It supports:
// - Function calls (currently only concat)
// - Unary operators (!, -)
// - Binary operators (+, -, *, /, %, ==, !=, <, >, <=, >=, &&, ||)
// - Parentheses
// - Field selection (.)
// - Array/map indexing ([])
// - Basic literals (int, float, string)
// - Identifiers (variables)
func evalExpr(expr ast.Expr, value map[string]any) (any, error) {

	switch e := expr.(type) {
	case *ast.CallExpr:
		// Handle concat function
		if ident, ok := e.Fun.(*ast.Ident); ok && ident.Name == "concat" {
			if len(e.Args) < 2 {
				return nil, fmt.Errorf("concat requires at least 2 arguments")
			}

			var result string
			for i, arg := range e.Args {
				val, err := evalExpr(arg, value)
				if err != nil {
					return nil, err
				}

				str, ok := val.(string)
				if !ok {
					return nil, fmt.Errorf("concat argument %d must be string", i+1)
				}

				result += str
			}
			return result, nil
		}
		return nil, fmt.Errorf("unknown function: %v", e.Fun)

	case *ast.UnaryExpr:
		xRes, err := evalExpr(e.X, value)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case token.NOT: // !
			switch xVal := xRes.(type) {
			case bool:
				return !xVal, nil
			}
		case token.SUB: // -
			switch xVal := xRes.(type) {
			case int:
				return -xVal, nil
			case float64:
				return -xVal, nil
			}
		}

	case *ast.BinaryExpr:
		xRes, err := evalExpr(e.X, value)
		if err != nil {
			return nil, err
		}
		yRes, err := evalExpr(e.Y, value)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case token.LAND: // &&
			xBool, ok := xRes.(bool)
			if !ok {
				return nil, fmt.Errorf("left operand must be boolean")
			}
			yBool, ok := yRes.(bool)
			if !ok {
				return nil, fmt.Errorf("right operand must be boolean")
			}
			return xBool && yBool, nil

		case token.LOR: // ||
			xBool, ok := xRes.(bool)
			if !ok {
				return nil, fmt.Errorf("left operand must be boolean")
			}
			yBool, ok := yRes.(bool)
			if !ok {
				return nil, fmt.Errorf("right operand must be boolean")
			}
			return xBool || yBool, nil

		case token.EQL: // ==
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x == y, nil
				case float64:
					return float64(x) == y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x == float64(y), nil
				case float64:
					return x == y, nil
				}
			}
			return xRes == yRes, nil

		case token.NEQ: // !=
			return xRes != yRes, nil

		case token.LSS: // <
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x < y, nil
				case float64:
					return float64(x) < y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x < float64(y), nil
				case float64:
					return x < y, nil
				}
			}

		case token.GTR: // >
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x > y, nil
				case float64:
					return float64(x) > y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x > float64(y), nil
				case float64:
					return x > y, nil
				}
			}

		case token.LEQ: // <=
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x <= y, nil
				case float64:
					return float64(x) <= y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x <= float64(y), nil
				case float64:
					return x <= y, nil
				}
			}

		case token.GEQ: // >=
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x >= y, nil
				case float64:
					return float64(x) >= y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x >= float64(y), nil
				case float64:
					return x >= y, nil
				}
			}

		case token.ADD: // +
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x + y, nil
				case float64:
					return float64(x) + y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x + float64(y), nil
				case float64:
					return x + y, nil
				}
			case string:
				switch y := yRes.(type) {
				case string:
					return x + y, nil
				}
			}

		case token.SUB: // -
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x - y, nil
				case float64:
					return float64(x) - y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x - float64(y), nil
				case float64:
					return x - y, nil
				}
			}

		case token.MUL: // *
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					return x * y, nil
				case float64:
					return float64(x) * y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					return x * float64(y), nil
				case float64:
					return x * y, nil
				}
			}

		case token.QUO: // /
			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					if y == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return x / y, nil
				case float64:
					if y == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return float64(x) / y, nil
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					if y == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return x / float64(y), nil
				case float64:
					if y == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return x / y, nil
				}
			}

		case token.REM: // %
			var xInt int
			var yInt int

			switch x := xRes.(type) {
			case int:
				switch y := yRes.(type) {
				case int:
					if y == 0 {
						return nil, fmt.Errorf("modulo by zero")
					}
					xInt = x
					yInt = y
				case float64:
					if math.Mod(y, 1.0) == 0 {
						xInt = x
						yInt = int(y)
					} else {
						return nil, fmt.Errorf("right operand must be integer")
					}
				}
			case float64:
				switch y := yRes.(type) {
				case int:
					if math.Mod(x, 1.0) == 0 {
						xInt = int(x)
						yInt = y
					} else {
						return nil, fmt.Errorf("left operand must be integer")
					}
				case float64:
					if math.Mod(x, 1.0) == 0 {
						xInt = int(x)
					} else {
						return nil, fmt.Errorf("left operand must be integer")
					}
					if math.Mod(y, 1.0) == 0 {
						yInt = int(y)
					} else {
						return nil, fmt.Errorf("right operand must be integer")
					}
				}
			}
			return xInt % yInt, nil
		default:
			return nil, fmt.Errorf("unknown operator: %v", e.Op)
		}

	case *ast.ParenExpr:
		return evalExpr(e.X, value)

	case *ast.SelectorExpr:
		v, err := evalExpr(e.X, value)
		if err != nil {
			return nil, err
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot select from non-map value")
		}
		val, exists := m[e.Sel.Name]
		if !exists {
			return nil, fmt.Errorf("undefined field: %s", e.Sel.Name)
		}
		return val, nil

	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			i, err := strconv.ParseInt(e.Value, 10, 32)
			if err != nil {
				return nil, err
			}
			return int(i), nil
		case token.FLOAT:
			return strconv.ParseFloat(e.Value, 64)
		case token.STRING:
			return e.Value[1 : len(e.Value)-1], nil
		}
		return e.Value, nil

	case *ast.Ident:
		if e.Name == "true" {
			return true, nil
		}
		if e.Name == "false" {
			return false, nil
		}
		if e.Name == "nil" {
			return nil, nil
		}
		if e.Name == "null" {
			return nil, nil
		}
		if e.Name == "undefined" {
			return nil, nil
		}
		val, ok := value[e.Name]
		if !ok {
			return nil, fmt.Errorf("undefined variable: %s", e.Name)
		}
		return val, nil

	case *ast.IndexExpr:
		v, err := evalExpr(e.X, value)
		if err != nil {
			return nil, err
		}

		switch idx := e.Index.(type) {
		case *ast.BasicLit:
			if idx.Kind == token.INT {
				i, err := strconv.Atoi(idx.Value)
				if err != nil {
					return nil, err
				}
				arr, ok := v.([]any)
				if !ok {
					return nil, fmt.Errorf("cannot index non-array value")
				}
				if i < 0 || i >= len(arr) {
					return nil, fmt.Errorf("array index out of bounds")
				}
				return arr[i], nil
			}
			if idx.Kind == token.STRING {
				key := idx.Value[1 : len(idx.Value)-1]
				m, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("cannot index non-map value")
				}
				val, exists := m[key]
				if !exists {
					return nil, fmt.Errorf("undefined field: %s", key)
				}
				return val, nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported expression type: %T", expr)
}

// segment represents a part of a path expression, which can be either a literal
// string value or a reference to a field using ${} syntax
type segment struct {
	Value       string
	IsReference bool
}

// sanitizePath converts a path expression with ${} references into a valid Go
// expression that can be parsed. It handles:
//   - Converting dot notation to bracket notation (foo.bar ->
//     memory["foo"]["bar"])
//   - Handling array indices (foo[0] -> memory["foo"][0])
//   - Concatenating adjacent string literals and references using the concat()
//     function
func sanitizePath(path string) (string, error) {
	var segments []segment

	for {
		// Find next ${} expression
		start := strings.Index(path, "${")
		if start == -1 {
			// No more ${} expressions, add remaining text if any
			if path != "" {
				segments = append(segments, segment{Value: path, IsReference: false})
			}
			break
		}

		// Add text before ${} if any
		if start > 0 {
			segments = append(segments, segment{Value: path[:start], IsReference: false})
		}

		// Find closing }
		end := strings.Index(path[start:], "}")
		if end == -1 {
			return "", fmt.Errorf("unclosed ${} expression")
		}
		end += start

		// Extract and validate expression
		expr := strings.TrimSpace(path[start+2 : end])
		if expr == "" {
			return "", fmt.Errorf("empty expression in ${}")
		}

		// Convert dot notation to bracket notation
		converted := "memory"
		parts := strings.Split(expr, ".")
		for _, part := range parts {
			if idx := strings.Index(part, "["); idx != -1 {
				// Handle array indexing
				key := part[:idx]
				index := part[idx:]
				if key != "" {
					converted += "[\"" + key + "\"]"
				}
				converted += index
			} else {
				converted += "[\"" + part + "\"]"
			}
		}

		segments = append(segments, segment{Value: converted, IsReference: true})
		path = path[end+1:]
	}

	// Build final result string
	var result string
	var consecutiveRefs []string

	for i, seg := range segments {
		if seg.IsReference {
			consecutiveRefs = append(consecutiveRefs, seg.Value)

			// Check if this is the last segment or next segment is not a reference
			if i == len(segments)-1 || !segments[i+1].IsReference {
				if len(consecutiveRefs) > 1 {
					// Multiple consecutive refs - use concat
					result += "concat(" + strings.Join(consecutiveRefs, ", ") + ")"
				} else {
					// Single ref - use directly
					result += consecutiveRefs[0]
				}
				consecutiveRefs = nil
			}
		} else {
			result += seg.Value
		}
	}

	return result, nil
}
