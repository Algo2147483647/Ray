package exprdiff

import (
	"fmt"
	"strconv"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// Derivatives returns symbolic derivatives of source with respect to variables.
// It intentionally supports only the smooth subset of the expression runtime.
func Derivatives(source string, variables ...string) ([]string, bool) {
	tree, err := parser.Parse(source)
	if err != nil || tree == nil || tree.Node == nil {
		return nil, false
	}

	derivatives := make([]string, len(variables))
	for i, variable := range variables {
		derivative, ok := diffExprNode(tree.Node, variable)
		if !ok {
			return nil, false
		}
		derivatives[i] = derivative
	}
	return derivatives, true
}

func diffExprNode(node ast.Node, variable string) (string, bool) {
	switch n := node.(type) {
	case *ast.IntegerNode, *ast.FloatNode:
		return "0", true
	case *ast.IdentifierNode:
		if n.Value == variable {
			return "1", true
		}
		return "0", true
	case *ast.UnaryNode:
		return diffUnaryExprNode(n, variable)
	case *ast.BinaryNode:
		return diffBinaryExprNode(n, variable)
	case *ast.CallNode:
		return diffCallExprNode(n, variable)
	case *ast.BuiltinNode:
		return diffNamedCallExprNode(n.Name, n.Arguments, variable)
	default:
		return "", false
	}
}

func diffUnaryExprNode(node *ast.UnaryNode, variable string) (string, bool) {
	inner, ok := diffExprNode(node.Node, variable)
	if !ok {
		return "", false
	}
	switch node.Operator {
	case "+":
		return inner, true
	case "-":
		return fmt.Sprintf("-(%s)", inner), true
	default:
		return "", false
	}
}

func diffBinaryExprNode(node *ast.BinaryNode, variable string) (string, bool) {
	left := node.Left.String()
	right := node.Right.String()
	dLeft, ok := diffExprNode(node.Left, variable)
	if !ok {
		return "", false
	}
	dRight, ok := diffExprNode(node.Right, variable)
	if !ok {
		return "", false
	}

	switch node.Operator {
	case "+":
		return fmt.Sprintf("(%s) + (%s)", dLeft, dRight), true
	case "-":
		return fmt.Sprintf("(%s) - (%s)", dLeft, dRight), true
	case "*":
		return fmt.Sprintf("(%s)*(%s) + (%s)*(%s)", dLeft, right, left, dRight), true
	case "/":
		return fmt.Sprintf("((%s)*(%s) - (%s)*(%s)) / pow(%s, 2)", dLeft, right, left, dRight, right), true
	case "**", "^":
		exponent, ok := constantExprNodeValue(node.Right)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("(%s)*pow(%s, %s)*(%s)", formatFloat(exponent), left, formatFloat(exponent-1), dLeft), true
	default:
		return "", false
	}
}

func diffCallExprNode(node *ast.CallNode, variable string) (string, bool) {
	callee, ok := node.Callee.(*ast.IdentifierNode)
	if !ok {
		return "", false
	}
	return diffNamedCallExprNode(callee.Value, node.Arguments, variable)
}

func diffNamedCallExprNode(name string, args []ast.Node, variable string) (string, bool) {
	switch name {
	case "abs", "sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh", "exp", "log", "log10", "sqrt":
		if len(args) != 1 {
			return "", false
		}
		arg := args[0].String()
		dArg, ok := diffExprNode(args[0], variable)
		if !ok {
			return "", false
		}
		switch name {
		case "abs":
			return fmt.Sprintf("sign(%s)*(%s)", arg, dArg), true
		case "sin":
			return fmt.Sprintf("cos(%s)*(%s)", arg, dArg), true
		case "cos":
			return fmt.Sprintf("-sin(%s)*(%s)", arg, dArg), true
		case "tan":
			return fmt.Sprintf("(%s) / pow(cos(%s), 2)", dArg, arg), true
		case "asin":
			return fmt.Sprintf("(%s) / sqrt(1 - pow(%s, 2))", dArg, arg), true
		case "acos":
			return fmt.Sprintf("-(%s) / sqrt(1 - pow(%s, 2))", dArg, arg), true
		case "atan":
			return fmt.Sprintf("(%s) / (1 + pow(%s, 2))", dArg, arg), true
		case "sinh":
			return fmt.Sprintf("cosh(%s)*(%s)", arg, dArg), true
		case "cosh":
			return fmt.Sprintf("sinh(%s)*(%s)", arg, dArg), true
		case "tanh":
			return fmt.Sprintf("(%s) / pow(cosh(%s), 2)", dArg, arg), true
		case "exp":
			return fmt.Sprintf("exp(%s)*(%s)", arg, dArg), true
		case "log":
			return fmt.Sprintf("(%s) / (%s)", dArg, arg), true
		case "log10":
			return fmt.Sprintf("(%s) / ((%s)*log(10))", dArg, arg), true
		case "sqrt":
			return fmt.Sprintf("(%s) / (2*sqrt(%s))", dArg, arg), true
		}
	case "pow":
		if len(args) != 2 {
			return "", false
		}
		exponent, ok := constantExprNodeValue(args[1])
		if !ok {
			return "", false
		}
		base := args[0].String()
		dBase, ok := diffExprNode(args[0], variable)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("(%s)*pow(%s, %s)*(%s)", formatFloat(exponent), base, formatFloat(exponent-1), dBase), true
	case "atan2":
		if len(args) != 2 {
			return "", false
		}
		numerator := args[0].String()
		denominator := args[1].String()
		dNumerator, ok := diffExprNode(args[0], variable)
		if !ok {
			return "", false
		}
		dDenominator, ok := diffExprNode(args[1], variable)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("((%s)*(%s) - (%s)*(%s)) / (pow(%s, 2) + pow(%s, 2))", denominator, dNumerator, numerator, dDenominator, numerator, denominator), true
	default:
		return "", false
	}
	return "", false
}

func constantExprNodeValue(node ast.Node) (float64, bool) {
	switch n := node.(type) {
	case *ast.IntegerNode:
		return float64(n.Value), true
	case *ast.FloatNode:
		return n.Value, true
	case *ast.UnaryNode:
		value, ok := constantExprNodeValue(n.Node)
		if !ok {
			return 0, false
		}
		switch n.Operator {
		case "+":
			return value, true
		case "-":
			return -value, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
