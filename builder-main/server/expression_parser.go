package main

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token in an expression
type TokenType int

const (
	TokenIdentifier TokenType = iota
	TokenPlus
	TokenMinus
	TokenMultiply
	TokenDivide
	TokenLeftParen
	TokenRightParen
	TokenEOF
	TokenError
)

// Token represents a single token in an expression
type Token struct {
	Type  TokenType
	Value string
}

// ExprNode is the interface for AST nodes
type ExprNode interface {
	ToSQL(wrapFunc func(string) string) string
}

// BinaryOpNode represents a binary operation (a + b, a * b, etc.)
type BinaryOpNode struct {
	Left     ExprNode
	Operator string
	Right    ExprNode
}

// IdentifierNode represents a column name
type IdentifierNode struct {
	Name string
}

// UnaryOpNode represents a unary operation (-a)
type UnaryOpNode struct {
	Operator string
	Operand  ExprNode
}

// Implement ExprNode for BinaryOpNode
func (n *BinaryOpNode) ToSQL(wrapFunc func(string) string) string {
	leftSQL := n.Left.ToSQL(wrapFunc)
	rightSQL := n.Right.ToSQL(wrapFunc)
	return fmt.Sprintf("(%s %s %s)", leftSQL, n.Operator, rightSQL)
}

// Implement ExprNode for IdentifierNode
func (n *IdentifierNode) ToSQL(wrapFunc func(string) string) string {
	return wrapFunc(n.Name)
}

// Implement ExprNode for UnaryOpNode
func (n *UnaryOpNode) ToSQL(wrapFunc func(string) string) string {
	operandSQL := n.Operand.ToSQL(wrapFunc)
	return fmt.Sprintf("(%s%s)", n.Operator, operandSQL)
}

// ParseColumnExpression parses a column expression and returns SQL with wrapped columns
func ParseColumnExpression(expr string, tableName string) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", fmt.Errorf("empty expression")
	}

	tokens, err := tokenize(expr)
	if err != nil {
		return "", err
	}

	if len(tokens) == 0 || (len(tokens) == 1 && tokens[0].Type == TokenEOF) {
		return "", fmt.Errorf("empty expression")
	}

	pos := 0
	ast, err := parseExpression(tokens, &pos)
	if err != nil {
		return "", err
	}

	// Verify we consumed all tokens (except EOF)
	if pos < len(tokens)-1 || (pos < len(tokens) && tokens[pos].Type != TokenEOF) {
		return "", fmt.Errorf("unexpected tokens after expression")
	}

	// Generate SQL with proper column wrapping
	wrapFunc := func(columnName string) string {
		return wrapSingleColumn(columnName)
	}

	return ast.ToSQL(wrapFunc), nil
}

// tokenize converts an expression string into tokens
func tokenize(input string) ([]Token, error) {
	var tokens []Token
	i := 0

	for i < len(input) {
		// Skip whitespace
		if unicode.IsSpace(rune(input[i])) {
			i++
			continue
		}

		// Single-character tokens
		switch input[i] {
		case '+':
			tokens = append(tokens, Token{Type: TokenPlus, Value: "+"})
			i++
		case '-':
			// Distinguish hyphenated column names (Re-attendance, 097b-IC04) from subtraction
			// Convention: subtraction operators have whitespace before them; hyphens in column names don't
			prevIsSpace := i == 0 || unicode.IsSpace(rune(input[i-1]))
			if prevIsSpace {
				tokens = append(tokens, Token{Type: TokenMinus, Value: "-"})
				i++
			} else if i+1 < len(input) && (unicode.IsLetter(rune(input[i+1])) || unicode.IsDigit(rune(input[i+1])) || input[i+1] == '_') {
				// Hyphen followed by alphanumeric: part of column name — backtrack into last identifier
				// Pop the last identifier token and extend it
				if len(tokens) > 0 && tokens[len(tokens)-1].Type == TokenIdentifier {
					prev := tokens[len(tokens)-1].Value
					i++ // skip the hyphen
					start := i
					for i < len(input) && (unicode.IsLetter(rune(input[i])) || unicode.IsDigit(rune(input[i])) || input[i] == '_' || input[i] == '-') {
						i++
					}
					// Trim any trailing hyphens (a hyphen at end would be an error)
					extended := prev + "-" + input[start:i]
					tokens[len(tokens)-1].Value = strings.TrimRight(extended, "-")
				} else {
					tokens = append(tokens, Token{Type: TokenMinus, Value: "-"})
					i++
				}
			} else {
				tokens = append(tokens, Token{Type: TokenMinus, Value: "-"})
				i++
			}
		case '*':
			tokens = append(tokens, Token{Type: TokenMultiply, Value: "*"})
			i++
		case '/':
			tokens = append(tokens, Token{Type: TokenDivide, Value: "/"})
			i++
		case '(':
			tokens = append(tokens, Token{Type: TokenLeftParen, Value: "("})
			i++
		case ')':
			tokens = append(tokens, Token{Type: TokenRightParen, Value: ")"})
			i++
		default:
			// Must be an identifier (column name)
			if unicode.IsLetter(rune(input[i])) || input[i] == '_' {
				start := i
				for i < len(input) && (unicode.IsLetter(rune(input[i])) || unicode.IsDigit(rune(input[i])) || input[i] == '_') {
					i++
				}
				tokens = append(tokens, Token{Type: TokenIdentifier, Value: input[start:i]})
			} else {
				return nil, fmt.Errorf("unexpected character '%c' at position %d", input[i], i)
			}
		}
	}

	tokens = append(tokens, Token{Type: TokenEOF, Value: ""})
	return tokens, nil
}

// parseExpression parses addition and subtraction (lowest precedence)
// expression := term (('+' | '-') term)*
func parseExpression(tokens []Token, pos *int) (ExprNode, error) {
	left, err := parseTerm(tokens, pos)
	if err != nil {
		return nil, err
	}

	for *pos < len(tokens) {
		token := tokens[*pos]
		if token.Type == TokenPlus || token.Type == TokenMinus {
			op := token.Value
			*pos++
			right, err := parseTerm(tokens, pos)
			if err != nil {
				return nil, err
			}
			left = &BinaryOpNode{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}

	return left, nil
}

// parseTerm parses multiplication and division (higher precedence)
// term := factor (('*' | '/') factor)*
func parseTerm(tokens []Token, pos *int) (ExprNode, error) {
	left, err := parseFactor(tokens, pos)
	if err != nil {
		return nil, err
	}

	for *pos < len(tokens) {
		token := tokens[*pos]
		if token.Type == TokenMultiply || token.Type == TokenDivide {
			op := token.Value
			*pos++
			right, err := parseFactor(tokens, pos)
			if err != nil {
				return nil, err
			}
			left = &BinaryOpNode{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}

	return left, nil
}

// parseFactor parses parentheses, unary operators, and identifiers (highest precedence)
// factor := '(' expression ')' | unary_op factor | identifier
func parseFactor(tokens []Token, pos *int) (ExprNode, error) {
	if *pos >= len(tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	token := tokens[*pos]

	// Handle parentheses
	if token.Type == TokenLeftParen {
		*pos++
		expr, err := parseExpression(tokens, pos)
		if err != nil {
			return nil, err
		}
		if *pos >= len(tokens) || tokens[*pos].Type != TokenRightParen {
			return nil, fmt.Errorf("expected ')' but got EOF")
		}
		*pos++
		return expr, nil
	}

	// Handle unary minus
	if token.Type == TokenMinus {
		*pos++
		operand, err := parseFactor(tokens, pos)
		if err != nil {
			return nil, err
		}
		return &UnaryOpNode{Operator: "-", Operand: operand}, nil
	}

	// Handle identifier
	if token.Type == TokenIdentifier {
		*pos++
		return &IdentifierNode{Name: token.Value}, nil
	}

	return nil, fmt.Errorf("expected identifier or '(' but got '%s'", token.Value)
}

// wrapSingleColumn wraps a single column name with COALESCE(CAST(NULLIF(...)))
func wrapSingleColumn(columnName string) string {
	quotedColumn := quoteIdentifier(columnName)
	return fmt.Sprintf("COALESCE(CAST(NULLIF(TRIM(CAST(%s AS TEXT)), '') AS NUMERIC), 0)", quotedColumn)
}
