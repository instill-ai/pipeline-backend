package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"gorm.io/gorm/clause"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type transpiler struct {
	filter filtering.Filter
}

// Transpile executes the transpilation on the filter
func (t *transpiler) Transpile() (*clause.Expr, error) {
	if t.filter.CheckedExpr == nil {
		return nil, nil
	}
	expr, err := t.transpileExpr(t.filter.CheckedExpr.Expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

func (t *transpiler) transpileExpr(e *expr.Expr) (*clause.Expr, error) {
	switch e.ExprKind.(type) {
	case *expr.Expr_CallExpr:
		return t.transpileCallExpr(e)
	case *expr.Expr_IdentExpr:
		return t.transpileIdentExpr(e)
	case *expr.Expr_ConstExpr:
		return t.transpileConstExpr(e)
	case *expr.Expr_SelectExpr:
		return t.transpileSelectExpr(e)
	default:
		return nil, fmt.Errorf("unsupported expr: %v", e)
	}
}

func (t *transpiler) transpileConstExpr(e *expr.Expr) (*clause.Expr, error) {
	switch kind := e.GetConstExpr().ConstantKind.(type) {
	case *expr.Constant_BoolValue:
		return &clause.Expr{Vars: []interface{}{kind.BoolValue}}, nil
	case *expr.Constant_DoubleValue:
		return &clause.Expr{Vars: []interface{}{kind.DoubleValue}}, nil
	case *expr.Constant_Int64Value:
		return &clause.Expr{Vars: []interface{}{kind.Int64Value}}, nil
	case *expr.Constant_StringValue:
		return &clause.Expr{Vars: []interface{}{kind.StringValue}}, nil
	case *expr.Constant_Uint64Value:
		return &clause.Expr{Vars: []interface{}{kind.Uint64Value}}, nil

	default:
		return nil, fmt.Errorf("unsupported const expr: %v", kind)
	}
}

func (t *transpiler) transpileCallExpr(e *expr.Expr) (*clause.Expr, error) {
	switch e.GetCallExpr().Function {
	case filtering.FunctionHas:
		return t.transpileHasCallExpr(e)
	case filtering.FunctionEquals:
		return t.transpileComparisonCallExpr(e, clause.Eq{})
	case filtering.FunctionNotEquals:
		return t.transpileComparisonCallExpr(e, clause.Neq{})
	case filtering.FunctionLessThan:
		return t.transpileComparisonCallExpr(e, clause.Lt{})
	case filtering.FunctionLessEquals:
		return t.transpileComparisonCallExpr(e, clause.Lte{})
	case filtering.FunctionGreaterThan:
		return t.transpileComparisonCallExpr(e, clause.Gt{})
	case filtering.FunctionGreaterEquals:
		return t.transpileComparisonCallExpr(e, clause.Gte{})
	case filtering.FunctionAnd:
		return t.transpileBinaryLogicalCallExpr(e, clause.AndConditions{})
	case filtering.FunctionOr:
		return t.transpileBinaryLogicalCallExpr(e, clause.OrConditions{})
	case filtering.FunctionNot:
		return t.transpileNotCallExpr(e)
	case filtering.FunctionTimestamp:
		return t.transpileTimestampCallExpr(e)
	default:
		return nil, fmt.Errorf("unsupported function call: %s", e.GetCallExpr().Function)
	}
}

func (t *transpiler) transpileIdentExpr(e *expr.Expr) (*clause.Expr, error) {

	identExpr := e.GetIdentExpr()
	identType, ok := t.filter.CheckedExpr.TypeMap[e.Id]
	if !ok {
		return nil, fmt.Errorf("unknown type of ident expr %d", e.Id)
	}
	if messageType := identType.GetMessageType(); messageType != "" {
		if enumType, err := protoregistry.GlobalTypes.FindEnumByName(protoreflect.FullName(messageType)); err == nil {
			if enumValue := enumType.Descriptor().Values().ByName(protoreflect.Name(identExpr.Name)); enumValue != nil {
				// TODO: Configurable support for string literals.
				return &clause.Expr{
					Vars:               []interface{}{enumValue.Name()},
					WithoutParentheses: true,
				}, nil
			}
		}
	}

	// Tapping into the transpiler for the integration list filter. `featured`
	// is a boolean property in the API but in the database it translates to a
	// score to order the items in the list.
	if identExpr.Name == "featured" {
		return &clause.Expr{
			SQL:                "feature_score > 0",
			WithoutParentheses: true,
		}, nil
	}

	return &clause.Expr{
		SQL:                strcase.ToSnake(identExpr.Name),
		Vars:               nil,
		WithoutParentheses: true,
	}, nil
}

func (t *transpiler) transpileSelectExpr(e *expr.Expr) (*clause.Expr, error) {
	selectExpr := e.GetSelectExpr()
	operand, err := t.transpileExpr(selectExpr.Operand)
	if err != nil {
		return nil, err
	}
	return &clause.Expr{
		SQL:                fmt.Sprintf("%s ->> '%s'", operand.SQL, selectExpr.Field),
		Vars:               nil,
		WithoutParentheses: true,
	}, nil
}

func (t *transpiler) transpileNotCallExpr(e *expr.Expr) (*clause.Expr, error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 1 {
		return nil, fmt.Errorf(
			"unexpected number of arguments to `%s` expression: %d",
			filtering.FunctionNot,
			len(callExpr.Args),
		)
	}
	rhsExpr, err := t.transpileExpr(callExpr.Args[0])
	if err != nil {
		return nil, err
	}
	return &clause.Expr{
		SQL:                fmt.Sprintf("NOT %s", rhsExpr.SQL),
		WithoutParentheses: true,
	}, nil
}

func (t *transpiler) transpileComparisonCallExpr(e *expr.Expr, op interface{}) (*clause.Expr, error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 2 {
		return nil, fmt.Errorf(
			"unexpected number of arguments to `%s`: %d",
			callExpr.GetFunction(),
			len(callExpr.Args),
		)
	}

	ident, err := t.transpileExpr(callExpr.Args[0])
	if err != nil {
		return nil, err
	}

	con, err := t.transpileExpr(callExpr.Args[1])
	if err != nil {
		return nil, err
	}

	var sql string
	var vars []interface{}

	// TODO: we should remove the hardcode table prefix here.
	// Add "pipeline." prefix to prevent ambiguous since tag table also has the two columns.
	if ident.SQL == "create_time" || ident.SQL == "update_time" {
		ident.SQL = "pipeline." + ident.SQL
	}

	switch op.(type) {
	case clause.Eq:
		switch ident.SQL {
		// TODO we should support wildcards without special filter names
		case "q":
			sql = "((SIMILARITY(id, ?) > 0.2) OR (SIMILARITY(namespace_id, ?) > 0.2) OR (LOWER(id) LIKE LOWER(?)) OR (LOWER(namespace_id) LIKE LOWER(?)))"
			vars = append(vars, con.Vars[0], con.Vars[0], fmt.Sprintf("%%%s%%", con.Vars[0]), fmt.Sprintf("%%%s%%", con.Vars[0]))
		case "q_title":
			sql = "((SIMILARITY(title, ?) > 0.2) OR (LOWER(title) LIKE LOWER(?)))"
			vars = append(vars, con.Vars[0], fmt.Sprintf("%%%s%%", con.Vars[0]))
		case "q_integration":
			val := con.Vars[0]
			likeVal := fmt.Sprintf("%%%s%%", val)

			sql = "((SIMILARITY(title, ?) > 0.2) OR (LOWER(title) LIKE LOWER(?)) OR (SIMILARITY(vendor, ?) > 0.2) OR (LOWER(vendor) LIKE LOWER(?)))"
			vars = append(vars, val, likeVal, val, likeVal)
		case "q_connection":
			val := con.Vars[0]
			likeVal := fmt.Sprintf("%%%s%%", val)

			sql = "((SIMILARITY(connection.id, ?) > 0.2) OR (LOWER(connection.id) LIKE LOWER(?)) OR " +
				"(SIMILARITY(component_definition_index.title, ?) > 0.2) OR (LOWER(component_definition_index.title) LIKE LOWER(?)) OR" +
				"(SIMILARITY(component_definition_index.vendor, ?) > 0.2) OR (LOWER(component_definition_index.vendor) LIKE LOWER(?)))"
			vars = append(vars, val, likeVal, val, likeVal, val, likeVal)
		case "integration_id":
			sql = "component_definition_index.id = ?"
			vars = append(vars, con.Vars...)
		case "tag":
			sql = "tag.tag_name = ?"
			vars = append(vars, strings.ToLower(con.Vars[0].(string)))
		default:
			sql = fmt.Sprintf("%s = ?", ident.SQL)
			vars = append(vars, con.Vars...)
		}
	case clause.Neq:
		sql = fmt.Sprintf("%s <> ?", ident.SQL)
		vars = append(vars, con.Vars...)
	case clause.Lt:
		sql = fmt.Sprintf("%s < ?", ident.SQL)
		vars = append(vars, con.Vars...)
	case clause.Lte:
		sql = fmt.Sprintf("%s <= ?", ident.SQL)
		vars = append(vars, con.Vars...)
	case clause.Gt:
		sql = fmt.Sprintf("%s > ?", ident.SQL)
		vars = append(vars, con.Vars...)
	case clause.Gte:
		sql = fmt.Sprintf("%s >= ?", ident.SQL)
		vars = append(vars, con.Vars...)
	}

	return &clause.Expr{
		SQL:                sql,
		Vars:               vars,
		WithoutParentheses: true,
	}, nil
}

func (t *transpiler) transpileBinaryLogicalCallExpr(e *expr.Expr, op clause.Expression) (*clause.Expr, error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 2 {
		return nil, fmt.Errorf(
			"unexpected number of arguments to `%s`: %d",
			callExpr.GetFunction(),
			len(callExpr.Args),
		)
	}
	lhsExpr, err := t.transpileExpr(callExpr.Args[0])
	if err != nil {
		return nil, err
	}
	rhsExpr, err := t.transpileExpr(callExpr.Args[1])
	if err != nil {
		return nil, err
	}

	var sql string
	switch op.(type) {
	case clause.AndConditions:
		sql = fmt.Sprintf("%s AND %s", lhsExpr.SQL, rhsExpr.SQL)
	case clause.OrConditions:
		sql = fmt.Sprintf("%s OR %s", lhsExpr.SQL, rhsExpr.SQL)
	}

	return &clause.Expr{
		SQL:                sql,
		Vars:               append(lhsExpr.Vars, rhsExpr.Vars...),
		WithoutParentheses: true,
	}, nil
}

func (t *transpiler) transpileHasCallExpr(e *expr.Expr) (*clause.Expr, error) {
	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 2 {
		return nil, fmt.Errorf("unexpected number of arguments to `in` expression: %d", len(callExpr.Args))
	}

	if callExpr.Args[1].GetConstExpr() == nil {
		return nil, fmt.Errorf("TODO: add support for transpiling `:` where RHS is other than Const")
	}

	switch callExpr.Args[0].ExprKind.(type) {
	case *expr.Expr_IdentExpr:
		identExpr := callExpr.Args[0]
		constExpr := callExpr.Args[1]
		identType, ok := t.filter.CheckedExpr.TypeMap[callExpr.Args[0].Id]
		if !ok {
			return nil, fmt.Errorf("unknown type of ident expr %d", e.Id)
		}
		switch {
		// Repeated primitives:
		// > Repeated fields query to see if the repeated structure contains a matching element.
		case identType.GetListType().GetElemType().GetPrimitive() != expr.Type_PRIMITIVE_TYPE_UNSPECIFIED:
			iden, err := t.transpileIdentExpr(identExpr)
			if err != nil {
				return nil, err
			}
			con, err := t.transpileConstExpr(constExpr)
			if err != nil {
				return nil, err
			}
			return &clause.Expr{
				SQL:                fmt.Sprintf("? = ANY(%s)", iden.SQL),
				Vars:               con.Vars,
				WithoutParentheses: false,
			}, nil
		default:
			return nil, fmt.Errorf("TODO: add support for transpiling `:` on other types than repeated primitives")
		}
	case *expr.Expr_SelectExpr:
		operand := callExpr.Args[0].GetSelectExpr().Operand
		field := callExpr.Args[0].GetSelectExpr().Field
		constExpr := callExpr.Args[1]

		switch operand.ExprKind.(type) {
		case *expr.Expr_IdentExpr:
			iden, err := t.transpileIdentExpr(operand)
			if err != nil {
				return nil, err
			}
			con, err := t.transpileConstExpr(constExpr)
			if err != nil {
				return nil, err
			}
			con.Vars[0] = "%\"" + con.Vars[0].(string) + "\"%"

			return &clause.Expr{
				SQL:                fmt.Sprintf("%s ->> '%s' LIKE ?", iden.SQL, field),
				Vars:               con.Vars,
				WithoutParentheses: false,
			}, nil
		case *expr.Expr_SelectExpr:

			selectExpr := operand.GetSelectExpr()
			operand, err := t.transpileExpr(selectExpr.Operand)
			if err != nil {
				return nil, err
			}
			if err != nil {
				return nil, err
			}
			con, err := t.transpileConstExpr(constExpr)
			if err != nil {
				return nil, err
			}
			con.Vars[0] = "%\"" + field + "\": \"" + con.Vars[0].(string) + "\"%"

			return &clause.Expr{
				SQL:                fmt.Sprintf("%s ->> '%s' LIKE ?", operand.SQL, selectExpr.Field),
				Vars:               con.Vars,
				WithoutParentheses: false,
			}, nil
		default:
			return nil, fmt.Errorf("TODO: add support for more complicated transpiling")
		}

	default:
		return nil, fmt.Errorf("TODO: add support for transpiling `:` where LHS is other than Ident and Select")
	}

}

func (t *transpiler) transpileTimestampCallExpr(e *expr.Expr) (*clause.Expr, error) {

	callExpr := e.GetCallExpr()
	if len(callExpr.Args) != 1 {
		return nil, fmt.Errorf(
			"unexpected number of arguments to `%s`: %d", callExpr.Function, len(callExpr.Args),
		)
	}
	constArg, ok := callExpr.Args[0].ExprKind.(*expr.Expr_ConstExpr)
	if !ok {
		return nil, fmt.Errorf("expected constant string arg to %s", callExpr.Function)
	}
	stringArg, ok := constArg.ConstExpr.ConstantKind.(*expr.Constant_StringValue)
	if !ok {
		return nil, fmt.Errorf("expected constant string arg to %s", callExpr.Function)
	}
	timeArg, err := time.Parse(time.RFC3339, stringArg.StringValue)
	if err != nil {
		return nil, fmt.Errorf("invalid string arg to %s: %w", callExpr.Function, err)
	}
	return &clause.Expr{
		Vars:               []interface{}{timeArg},
		WithoutParentheses: true,
	}, nil
}
