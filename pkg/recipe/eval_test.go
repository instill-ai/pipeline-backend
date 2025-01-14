package recipe

import (
	"go/ast"
	"go/token"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func TestSanitizePath(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple property access",
			input:    "${foo}",
			expected: "memory[\"foo\"]",
		},
		{
			name:     "nested property access",
			input:    "${foo.bar}",
			expected: "memory[\"foo\"][\"bar\"]",
		},
		{
			name:     "array index access",
			input:    "${foo[0]}",
			expected: "memory[\"foo\"][0]",
		},
		{
			name:     "nested array index access",
			input:    "${foo.bar[0]}",
			expected: "memory[\"foo\"][\"bar\"][0]",
		},
		{
			name:     "multiple expressions",
			input:    "${foo} && ${bar}",
			expected: "memory[\"foo\"] && memory[\"bar\"]",
		},
		{
			name:     "complex nested access",
			input:    "${foo.bar[0].baz}",
			expected: "memory[\"foo\"][\"bar\"][0][\"baz\"]",
		},
		{
			name:     "direct property access",
			input:    "${foo.bar}",
			expected: "memory[\"foo\"][\"bar\"]",
		},
		{
			name:     "direct array access",
			input:    "${foo[0]}",
			expected: "memory[\"foo\"][0]",
		},
		{
			name:     "mixed access",
			input:    "${foo.bar} && ${baz.qux}",
			expected: "memory[\"foo\"][\"bar\"] && memory[\"baz\"][\"qux\"]",
		},
		{
			name:     "multiple array indices",
			input:    "${foo[0][1]}",
			expected: "memory[\"foo\"][0][1]",
		},
		{
			name:     "whitespace in path",
			input:    "${  foo.bar  }",
			expected: "memory[\"foo\"][\"bar\"]",
		},
		{
			name:     "empty path",
			input:    "${}",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "numeric property names",
			input:    "${foo.123.bar}",
			expected: "memory[\"foo\"][\"123\"][\"bar\"]",
		},
		{
			name:     "special characters in property names",
			input:    "${foo.bar-baz.qux}",
			expected: "memory[\"foo\"][\"bar-baz\"][\"qux\"]",
		},
		{
			name:     "logical operators",
			input:    "${foo} && ${bar} || ${baz}",
			expected: "memory[\"foo\"] && memory[\"bar\"] || memory[\"baz\"]",
		},
		{
			name:     "comparison operators",
			input:    "${foo} > ${bar}",
			expected: "memory[\"foo\"] > memory[\"bar\"]",
		},
		{
			name:     "nested expressions with operators",
			input:    "${foo.bar[0]} == ${baz.qux[1]}",
			expected: "memory[\"foo\"][\"bar\"][0] == memory[\"baz\"][\"qux\"][1]",
		},
		{
			name:     "multiple nested arrays",
			input:    "${foo[0][1][2]}",
			expected: "memory[\"foo\"][0][1][2]",
		},
		{
			name:     "complex logical expression",
			input:    "${foo} && (${bar} || ${baz}) && !${qux}",
			expected: "memory[\"foo\"] && (memory[\"bar\"] || memory[\"baz\"]) && !memory[\"qux\"]",
		},
		{
			name:     "multiple comparison operators",
			input:    "${foo} > ${bar} && ${baz} < ${qux}",
			expected: "memory[\"foo\"] > memory[\"bar\"] && memory[\"baz\"] < memory[\"qux\"]",
		},
		{
			name:     "deeply nested properties",
			input:    "${a.b.c.d.e.f}",
			expected: "memory[\"a\"][\"b\"][\"c\"][\"d\"][\"e\"][\"f\"]",
		},
		{
			name:     "mixed array and property access",
			input:    "${foo[0].bar.baz[1].qux}",
			expected: "memory[\"foo\"][0][\"bar\"][\"baz\"][1][\"qux\"]",
		},
		{
			name:     "string concatenation",
			input:    "${foo}${bar} == ${baz}",
			expected: "concat(memory[\"foo\"], memory[\"bar\"]) == memory[\"baz\"]",
		},
		{
			name:     "arithmetic operators",
			input:    "${foo} + ${bar} * ${baz}",
			expected: "memory[\"foo\"] + memory[\"bar\"] * memory[\"baz\"]",
		},
		{
			name:     "nested arithmetic",
			input:    "(${foo} + ${bar}) * ${baz}",
			expected: "(memory[\"foo\"] + memory[\"bar\"]) * memory[\"baz\"]",
		},
		{
			name:     "modulo operator",
			input:    "${foo} % ${bar}",
			expected: "memory[\"foo\"] % memory[\"bar\"]",
		},
		{
			name:     "bitwise operators",
			input:    "${foo} & ${bar} | ${baz}",
			expected: "memory[\"foo\"] & memory[\"bar\"] | memory[\"baz\"]",
		},
		{
			name:     "complex arithmetic expression",
			input:    "${a} * (${b} + ${c}) / ${d} - ${e}",
			expected: "memory[\"a\"] * (memory[\"b\"] + memory[\"c\"]) / memory[\"d\"] - memory[\"e\"]",
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			result, err := sanitizePath(tt.input)
			if tt.wantErr {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}
			c.Assert(err, qt.IsNil)
			c.Assert(result, qt.Equals, tt.expected)
		})
	}
}
func TestEval(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name     string
		input    string
		data     map[string]any
		expected bool
		wantErr  bool
	}{
		{
			name:     "simple boolean",
			input:    "${foo}",
			data:     map[string]any{"foo": true},
			expected: true,
		},
		{
			name:     "nested boolean",
			input:    "${foo.bar}",
			data:     map[string]any{"foo": map[string]any{"bar": true}},
			expected: true,
		},
		{
			name:     "array access",
			input:    "${foo[0]}",
			data:     map[string]any{"foo": []any{true}},
			expected: true,
		},
		{
			name:     "complex nested access",
			input:    "${foo.bar[0].baz}",
			data:     map[string]any{"foo": map[string]any{"bar": []any{map[string]any{"baz": true}}}},
			expected: true,
		},
		{
			name:     "logical AND",
			input:    "${foo} && ${bar}",
			data:     map[string]any{"foo": true, "bar": true},
			expected: true,
		},
		{
			name:     "logical OR",
			input:    "${foo} || ${bar}",
			data:     map[string]any{"foo": false, "bar": true},
			expected: true,
		},
		{
			name:     "mixed operators",
			input:    "${foo} && (${bar} || ${baz})",
			data:     map[string]any{"foo": true, "bar": false, "baz": true},
			expected: true,
		},
		{
			name:    "invalid path",
			input:   "${nonexistent}",
			data:    map[string]any{},
			wantErr: true,
		},
		{
			name:     "numeric comparison less than",
			input:    "${foo} < ${bar}",
			data:     map[string]any{"foo": 5, "bar": 10},
			expected: true,
		},
		{
			name:     "numeric comparison greater than",
			input:    "${foo} > ${bar}",
			data:     map[string]any{"foo": 10, "bar": 5},
			expected: true,
		},
		{
			name:     "numeric comparison less than or equal",
			input:    "${foo} <= ${bar}",
			data:     map[string]any{"foo": 5, "bar": 5},
			expected: true,
		},
		{
			name:     "numeric comparison greater than or equal",
			input:    "${foo} >= ${bar}",
			data:     map[string]any{"foo": 5, "bar": 5},
			expected: true,
		},
		{
			name:     "equality comparison",
			input:    "${foo} == ${bar}",
			data:     map[string]any{"foo": "test", "bar": "test"},
			expected: true,
		},
		{
			name:     "inequality comparison",
			input:    "${foo} != ${bar}",
			data:     map[string]any{"foo": "test1", "bar": "test2"},
			expected: true,
		},
		{
			name:     "mixed numeric types comparison",
			input:    "${foo} < ${bar}",
			data:     map[string]any{"foo": 5, "bar": 5.5},
			expected: true,
		},
		{
			name:  "complex nested comparison",
			input: "${foo.bar[0]} == ${baz.qux[1]}",
			data: map[string]any{
				"foo": map[string]any{"bar": []any{"test"}},
				"baz": map[string]any{"qux": []any{"other", "test"}},
			},
			expected: true,
		},
		{
			name:     "unary not operator",
			input:    "!${foo}",
			data:     map[string]any{"foo": false},
			expected: true,
		},
		{
			name:     "unary minus operator",
			input:    "-${foo} > ${bar}",
			data:     map[string]any{"foo": 5, "bar": -10},
			expected: true,
		},
		{
			name:     "complex boolean expression",
			input:    "${a} && (${b} || ${c}) && !${d}",
			data:     map[string]any{"a": true, "b": false, "c": true, "d": false},
			expected: true,
		},
		{
			name:     "nested array comparison",
			input:    "${foo[0][1]} == ${bar[1][0]}",
			data:     map[string]any{"foo": []any{[]any{1, 2}}, "bar": []any{[]any{3}, []any{2}}},
			expected: true,
		},
		{
			name:     "string concatenation comparison",
			input:    "${foo}${bar} == ${baz}",
			data:     map[string]any{"foo": "hello", "bar": "world", "baz": "helloworld"},
			expected: true,
		},
		{
			name:     "multiple comparisons",
			input:    "${a} > ${b} && ${c} < ${d} && ${e} == ${f}",
			data:     map[string]any{"a": 10, "b": 5, "c": 3, "d": 4, "e": "test", "f": "test"},
			expected: true,
		},
		{
			name:     "arithmetic addition",
			input:    "${a} + ${b} == ${c}",
			data:     map[string]any{"a": 5, "b": 3, "c": 8},
			expected: true,
		},
		{
			name:     "arithmetic multiplication",
			input:    "${a} * ${b} == ${c}",
			data:     map[string]any{"a": 4, "b": 3, "c": 12},
			expected: true,
		},
		{
			name:     "arithmetic division",
			input:    "${a} / ${b} == ${c}",
			data:     map[string]any{"a": 10, "b": 2, "c": 5},
			expected: true,
		},
		{
			name:     "arithmetic modulo",
			input:    "${a} % ${b} == ${c}",
			data:     map[string]any{"a": 7, "b": 4, "c": 3},
			expected: true,
		},
		{
			name:     "complex arithmetic",
			input:    "(${a} + ${b}) * ${c} == ${d}",
			data:     map[string]any{"a": 2, "b": 3, "c": 4, "d": 20},
			expected: true,
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			val, err := data.NewValue(tt.data)
			c.Assert(err, qt.IsNil)

			result, err := Eval(tt.input, val)
			if tt.wantErr {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}
			c.Assert(err, qt.IsNil)
			c.Assert(result, qt.Equals, tt.expected)
		})
	}
}

func TestEvalExpr(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name    string
		expr    ast.Expr
		value   map[string]any
		want    any
		wantErr bool
	}{
		{
			name: "basic literal int",
			expr: &ast.BasicLit{
				Kind:  token.INT,
				Value: "42",
			},
			want: int(42),
		},
		{
			name: "basic literal float",
			expr: &ast.BasicLit{
				Kind:  token.FLOAT,
				Value: "3.14",
			},
			want: float64(3.14),
		},
		{
			name: "basic literal string",
			expr: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"test"`,
			},
			want: "test",
		},
		{
			name: "identifier true",
			expr: &ast.Ident{
				Name: "true",
			},
			want: true,
		},
		{
			name: "identifier false",
			expr: &ast.Ident{
				Name: "false",
			},
			want: false,
		},
		{
			name: "map selector",
			expr: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "memory"},
				Sel: &ast.Ident{Name: "foo"},
			},
			value: map[string]any{
				"memory": map[string]any{"foo": "bar"},
			},
			want: "bar",
		},
		{
			name: "array index",
			expr: &ast.IndexExpr{
				X: &ast.Ident{Name: "memory"},
				Index: &ast.BasicLit{
					Kind:  token.INT,
					Value: "1",
				},
			},
			value: map[string]any{
				"memory": []any{"foo", "bar"},
			},
			want: "bar",
		},
		{
			name: "map index",
			expr: &ast.IndexExpr{
				X: &ast.Ident{Name: "memory"},
				Index: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"foo"`,
				},
			},
			value: map[string]any{
				"memory": map[string]any{"foo": "bar"},
			},
			want: "bar",
		},
		{
			name: "binary expr equals",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
				Op: token.EQL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			want: true,
		},
		{
			name: "binary expr not equals",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
				Op: token.NEQ,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
			},
			want: true,
		},
		{
			name: "binary expr less than",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
				Op: token.LSS,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
			},
			want: true,
		},
		{
			name: "binary expr greater than",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "2"},
				Op: token.GTR,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			want: true,
		},
		{
			name: "binary expr less than or equal",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
				Op: token.LEQ,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			want: true,
		},
		{
			name: "binary expr greater than or equal",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "2"},
				Op: token.GEQ,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			want: true,
		},
		{
			name: "logical AND",
			expr: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "true"},
				Op: token.LAND,
				Y:  &ast.Ident{Name: "true"},
			},
			want: true,
		},
		{
			name: "logical OR",
			expr: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "true"},
				Op: token.LOR,
				Y:  &ast.Ident{Name: "false"},
			},
			want: true,
		},
		{
			name: "unary NOT",
			expr: &ast.UnaryExpr{
				Op: token.NOT,
				X:  &ast.Ident{Name: "false"},
			},
			want: true,
		},
		{
			name: "unary minus",
			expr: &ast.UnaryExpr{
				Op: token.SUB,
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			want: int(-1),
		},
		{
			name: "parenthesized expression",
			expr: &ast.ParenExpr{
				X: &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
					Op: token.EQL,
					Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
				},
			},
			want: true,
		},
		{
			name: "nested map selector",
			expr: &ast.SelectorExpr{
				X: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "memory"},
					Sel: &ast.Ident{Name: "foo"},
				},
				Sel: &ast.Ident{Name: "bar"},
			},
			value: map[string]any{
				"memory": map[string]any{
					"foo": map[string]any{
						"bar": "baz",
					},
				},
			},
			want: "baz",
		},
		{
			name: "complex binary expression",
			expr: &ast.BinaryExpr{
				X: &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
					Op: token.ADD,
					Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
				},
				Op: token.EQL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "3"},
			},
			want: true,
		},
		{
			name: "nested array index",
			expr: &ast.IndexExpr{
				X: &ast.IndexExpr{
					X: &ast.Ident{Name: "memory"},
					Index: &ast.BasicLit{
						Kind:  token.INT,
						Value: "0",
					},
				},
				Index: &ast.BasicLit{
					Kind:  token.INT,
					Value: "1",
				},
			},
			value: map[string]any{
				"memory": []any{[]any{"foo", "bar"}},
			},
			want: "bar",
		},
		{
			name: "binary expr addition",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "5"},
				Op: token.ADD,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "3"},
			},
			want: int(8),
		},
		{
			name: "binary expr multiplication",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "4"},
				Op: token.MUL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "3"},
			},
			want: int(12),
		},
		{
			name: "binary expr division",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "10"},
				Op: token.QUO,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "2"},
			},
			want: int(5),
		},
		{
			name: "binary expr modulo",
			expr: &ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "7"},
				Op: token.REM,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "4"},
			},
			want: int(3),
		},
		{
			name: "complex arithmetic expression",
			expr: &ast.BinaryExpr{
				X: &ast.ParenExpr{
					X: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.INT, Value: "2"},
						Op: token.ADD,
						Y:  &ast.BasicLit{Kind: token.INT, Value: "3"},
					},
				},
				Op: token.MUL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "4"},
			},
			want: int(20),
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			got, err := evalExpr(tt.expr, tt.value)
			if tt.wantErr {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}
			c.Assert(err, qt.IsNil)
			c.Assert(got, qt.DeepEquals, tt.want)
		})
	}
}
