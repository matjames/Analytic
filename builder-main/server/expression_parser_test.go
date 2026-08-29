package main

import (
	"strings"
	"testing"
)

// ============== Tokenizer Tests ==============

func TestTokenizeSimpleAddition(t *testing.T) {
	tokens, err := tokenize("a + b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 4 { // a, +, b, EOF
		t.Errorf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != TokenIdentifier || tokens[0].Value != "a" {
		t.Errorf("expected first token to be identifier 'a', got %v", tokens[0])
	}
	if tokens[1].Type != TokenPlus {
		t.Errorf("expected second token to be '+', got %v", tokens[1])
	}
	if tokens[2].Type != TokenIdentifier || tokens[2].Value != "b" {
		t.Errorf("expected third token to be identifier 'b', got %v", tokens[2])
	}
	if tokens[3].Type != TokenEOF {
		t.Errorf("expected last token to be EOF, got %v", tokens[3])
	}
}

func TestTokenizeAllOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{"a + b", []TokenType{TokenIdentifier, TokenPlus, TokenIdentifier, TokenEOF}},
		{"a - b", []TokenType{TokenIdentifier, TokenMinus, TokenIdentifier, TokenEOF}},
		{"a * b", []TokenType{TokenIdentifier, TokenMultiply, TokenIdentifier, TokenEOF}},
		{"a / b", []TokenType{TokenIdentifier, TokenDivide, TokenIdentifier, TokenEOF}},
		{"(a)", []TokenType{TokenLeftParen, TokenIdentifier, TokenRightParen, TokenEOF}},
	}

	for _, test := range tests {
		tokens, err := tokenize(test.input)
		if err != nil {
			t.Errorf("tokenize(%q) returned error: %v", test.input, err)
			continue
		}
		if len(tokens) != len(test.expected) {
			t.Errorf("tokenize(%q): expected %d tokens, got %d", test.input, len(test.expected), len(tokens))
			continue
		}
		for i, expectedType := range test.expected {
			if tokens[i].Type != expectedType {
				t.Errorf("tokenize(%q): token %d expected type %v, got %v", test.input, i, expectedType, tokens[i].Type)
			}
		}
	}
}

func TestTokenizeWhitespace(t *testing.T) {
	tests := []string{
		"a+b",
		"a + b",
		"a  +  b",
		"  a + b  ",
	}

	for _, input := range tests {
		tokens, err := tokenize(input)
		if err != nil {
			t.Errorf("tokenize(%q) returned error: %v", input, err)
			continue
		}
		// All should produce same token structure: identifier, operator, identifier, EOF
		if len(tokens) != 4 {
			t.Errorf("tokenize(%q): expected 4 tokens, got %d", input, len(tokens))
		}
	}
}

func TestTokenizeComplexExpression(t *testing.T) {
	tokens, err := tokenize("(a + b) * c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []TokenType{
		TokenLeftParen, TokenIdentifier, TokenPlus, TokenIdentifier, TokenRightParen,
		TokenMultiply, TokenIdentifier, TokenEOF,
	}
	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, expectedType := range expected {
		if i >= len(tokens) {
			break
		}
		if tokens[i].Type != expectedType {
			t.Errorf("token %d: expected %v, got %v", i, expectedType, tokens[i].Type)
		}
	}
}

func TestTokenizeUnderscoreAndDigits(t *testing.T) {
	tokens, err := tokenize("col_1 + col_2_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Value != "col_1" {
		t.Errorf("expected 'col_1', got '%s'", tokens[0].Value)
	}
	if tokens[2].Value != "col_2_test" {
		t.Errorf("expected 'col_2_test', got '%s'", tokens[2].Value)
	}
}

func TestTokenizeErrors(t *testing.T) {
	tests := []string{
		"a & b",      // Invalid character &
		"a $ b",      // Invalid character $
		"a # b",      // Invalid character #
		"@invalid",   // Invalid character @
	}

	for _, input := range tests {
		_, err := tokenize(input)
		if err == nil {
			t.Errorf("tokenize(%q) expected error, got nil", input)
		}
	}
}

// ============== Parser Tests ==============

func TestParseSimpleExpression(t *testing.T) {
	tests := []struct {
		input string
		want  string // Expected SQL pattern
	}{
		{"a", "COALESCE"},
		{"a + b", "(COALESCE"},
		{"a - b", "(COALESCE"},
		{"a * b", "(COALESCE"},
		{"a / b", "(COALESCE"},
	}

	for _, test := range tests {
		sql, err := ParseColumnExpression(test.input, "test_table")
		if err != nil {
			t.Errorf("ParseColumnExpression(%q) returned error: %v", test.input, err)
			continue
		}
		if !strings.Contains(sql, test.want) {
			t.Errorf("ParseColumnExpression(%q) expected to contain '%s', got: %s", test.input, test.want, sql)
		}
	}
}

func TestParsePrecedence(t *testing.T) {
	// Test that a + b * c wraps as (a + (b * c))
	// The SQL should have parentheses: (...a...) + (...(...b...) * (...c...)...)
	sql, err := ParseColumnExpression("a + b * c", "test_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify structure: should NOT have b*c wrapped together without a +
	// Count parentheses to ensure proper nesting
	openCount := strings.Count(sql, "(")
	closeCount := strings.Count(sql, ")")
	if openCount != closeCount {
		t.Errorf("unbalanced parentheses in SQL: %s", sql)
	}

	// The multiplication should happen before addition in precedence
	// This is hard to verify without parsing, but we check the structure exists
	if !strings.Contains(sql, "*") {
		t.Errorf("expected multiplication operator in SQL: %s", sql)
	}
}

func TestParseParentheses(t *testing.T) {
	// (a + b) * c should wrap (a + b) together
	sql, err := ParseColumnExpression("(a + b) * c", "test_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just verify it generates valid SQL
	if !strings.Contains(sql, "*") {
		t.Errorf("expected multiplication in SQL: %s", sql)
	}
}

func TestParseUnaryMinus(t *testing.T) {
	sql, err := ParseColumnExpression("-a + b", "test_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "-") {
		t.Errorf("expected unary minus in SQL: %s", sql)
	}
}

func TestParseComplexExpression(t *testing.T) {
	sql, err := ParseColumnExpression("(a + b) / c - d * e", "test_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all operators are present
	if !strings.Contains(sql, "+") {
		t.Errorf("expected + in SQL: %s", sql)
	}
	if !strings.Contains(sql, "/") {
		t.Errorf("expected / in SQL: %s", sql)
	}
	if !strings.Contains(sql, "-") {
		t.Errorf("expected - in SQL: %s", sql)
	}
	if !strings.Contains(sql, "*") {
		t.Errorf("expected * in SQL: %s", sql)
	}
}

// ============== SQL Generation Tests ==============

func TestSQLWrappingForSimpleColumn(t *testing.T) {
	sql, err := ParseColumnExpression("num_fever", "cht_form_097b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it contains the wrapping
	if !strings.Contains(sql, "COALESCE") {
		t.Errorf("expected COALESCE in SQL: %s", sql)
	}
	if !strings.Contains(sql, "CAST") {
		t.Errorf("expected CAST in SQL: %s", sql)
	}
	if !strings.Contains(sql, "NULLIF") {
		t.Errorf("expected NULLIF in SQL: %s", sql)
	}
	if !strings.Contains(sql, "TRIM") {
		t.Errorf("expected TRIM in SQL: %s", sql)
	}
	if !strings.Contains(sql, "NUMERIC") {
		t.Errorf("expected NUMERIC in SQL: %s", sql)
	}
}

func TestSQLWrappingForMultipleColumns(t *testing.T) {
	sql, err := ParseColumnExpression("num_fever_male + num_fever_female", "cht_form_097b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count how many COALESCE blocks (one per column)
	coalesceCount := strings.Count(sql, "COALESCE")
	if coalesceCount < 2 {
		t.Errorf("expected at least 2 COALESCE blocks for 2 columns, got %d: %s", coalesceCount, sql)
	}

	// Verify both column names are present (quoted)
	if !strings.Contains(sql, "num_fever_male") {
		t.Errorf("expected num_fever_male in SQL: %s", sql)
	}
	if !strings.Contains(sql, "num_fever_female") {
		t.Errorf("expected num_fever_female in SQL: %s", sql)
	}
}

func TestSQLAllOperators(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"a + b", "+"},
		{"a - b", "-"},
		{"a * b", "*"},
		{"a / b", "/"},
	}

	for _, test := range tests {
		sql, err := ParseColumnExpression(test.input, "test_table")
		if err != nil {
			t.Errorf("ParseColumnExpression(%q) returned error: %v", test.input, err)
			continue
		}
		if !strings.Contains(sql, test.operator) {
			t.Errorf("expected '%s' in SQL for %q: %s", test.operator, test.input, sql)
		}
	}
}

// ============== Error Handling Tests ==============

func TestParseErrorEmptyExpression(t *testing.T) {
	_, err := ParseColumnExpression("", "test_table")
	if err == nil {
		t.Errorf("expected error for empty expression")
	}
}

func TestParseErrorUnexpectedToken(t *testing.T) {
	_, err := ParseColumnExpression("a & b", "test_table")
	if err == nil {
		t.Errorf("expected error for invalid character")
	}
}

func TestParseErrorUnbalancedParentheses(t *testing.T) {
	tests := []string{
		"(a + b",
		"a + b)",
		"((a + b)",
	}

	for _, input := range tests {
		_, err := ParseColumnExpression(input, "test_table")
		if err == nil {
			t.Errorf("expected error for unbalanced parentheses: %q", input)
		}
	}
}

func TestParseErrorIncompleteExpression(t *testing.T) {
	tests := []string{
		"a +",
		"* b",
		"a / ",
	}

	for _, input := range tests {
		_, err := ParseColumnExpression(input, "test_table")
		if err == nil {
			t.Errorf("expected error for incomplete expression: %q", input)
		}
	}
}

// ============== Backward Compatibility Tests ==============

func TestBackwardCompatibilityAdditionOnly(t *testing.T) {
	// Test all the addition-only expressions that currently work
	tests := []string{
		"col1 + col2",
		"col1 + col2 + col3",
		"num_fever_u5_male + num_fever_u5_female",
		"a + b + c + d + e + f",
	}

	for _, input := range tests {
		sql, err := ParseColumnExpression(input, "cht_form_097b")
		if err != nil {
			t.Errorf("backward compat test failed for %q: %v", input, err)
			continue
		}

		// Verify SQL is generated
		if sql == "" {
			t.Errorf("empty SQL generated for %q", input)
		}

		// Verify columns are wrapped
		if !strings.Contains(sql, "COALESCE") {
			t.Errorf("no COALESCE wrapping for %q: %s", input, sql)
		}
	}
}

// ============== Integration-Style Tests ==============

func TestParseAndSQLGeneration(t *testing.T) {
	tests := []struct {
		input string
		check func(sql string) bool
	}{
		{
			"a + b",
			func(sql string) bool {
				return strings.Contains(sql, "+") && strings.Count(sql, "COALESCE") >= 2
			},
		},
		{
			"a * b + c",
			func(sql string) bool {
				return strings.Contains(sql, "*") && strings.Contains(sql, "+")
			},
		},
		{
			"(a + b) * c",
			func(sql string) bool {
				return strings.Contains(sql, "*") && strings.Count(sql, "COALESCE") >= 3
			},
		},
	}

	for _, test := range tests {
		sql, err := ParseColumnExpression(test.input, "test_table")
		if err != nil {
			t.Errorf("ParseColumnExpression(%q) failed: %v", test.input, err)
			continue
		}
		if !test.check(sql) {
			t.Errorf("check failed for %q: SQL = %s", test.input, sql)
		}
	}
}

// ============== Benchmark Tests ==============

func BenchmarkParseSimpleExpression(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseColumnExpression("a + b", "test_table")
	}
}

func BenchmarkParseComplexExpression(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseColumnExpression("(a + b) * c / d - e", "test_table")
	}
}

// ============== Real-World YAML Examples ==============

// TestEchisDistrictReportBODMAS tests the complex expression from estat-district-report.yaml
func TestEchisDistrictReportBODMAS(t *testing.T) {
	// This is the actual BODMAS expression from the YAML file
	expr := "(num_fever_u5_male + num_fever_u5_female) + (num_malaria_u5_male + num_malaria_u5_female) + (num_diarrhea_u5_male + num_diarrhea_u5_female) + (num_u5_with_pneumonia_and_recieved_amoxicilin_male + num_u5_with_pneumonia_and_recieved_amoxicilin_female)"

	sql, err := ParseColumnExpression(expr, "report.cht_form_097b")
	if err != nil {
		t.Fatalf("ParseColumnExpression failed for estat report expression: %v", err)
	}

	// Verify the SQL was generated
	if sql == "" {
		t.Error("empty SQL generated")
	}

	// Verify all operators are present
	if !strings.Contains(sql, "+") {
		t.Errorf("missing + operators in SQL: %s", sql)
	}

	// Verify columns are wrapped
	if !strings.Contains(sql, "COALESCE") {
		t.Errorf("missing COALESCE wrapping: %s", sql)
	}

	// Verify at least one column name is present
	if !strings.Contains(sql, "num_fever_u5_male") {
		t.Errorf("column names not preserved: %s", sql)
	}

	t.Log("✅ estat-district-report BODMAS expression parsed successfully")
}

// TestTreatmentCoverageCalculation tests division in calculate formulas
func TestTreatmentCoverageCalculation(t *testing.T) {
	// This mimics: (treated / cases * 100)
	// The parser handles this in the calculate formula, but we test the aggregation columns work
	expr := "num_diarrhea_u5_male_treated_with_ors + num_diarrhea_u5_female_treated_with_ors"

	sql, err := ParseColumnExpression(expr, "report.cht_form_097b")
	if err != nil {
		t.Fatalf("ParseColumnExpression failed: %v", err)
	}

	if !strings.Contains(sql, "+") {
		t.Errorf("missing + operator in SQL: %s", sql)
	}

	t.Log("✅ Treatment coverage aggregation expression parsed successfully")
}

// ============== Fuzz Tests ==============
// Run with: go test -fuzz=FuzzTokenize -fuzztime=30s ./server
// Run with: go test -fuzz=FuzzParseColumnExpression -fuzztime=30s ./server

// FuzzTokenize tests that the tokenizer never panics on arbitrary input
func FuzzTokenize(f *testing.F) {
	// Seed corpus with valid expressions
	f.Add("a + b")
	f.Add("a - b")
	f.Add("a * b")
	f.Add("a / b")
	f.Add("(a + b)")
	f.Add("(a + b) * c")
	f.Add("a + b * c - d / e")
	f.Add("col_1 + col_2")
	f.Add("num_fever_u5_male + num_fever_u5_female")
	f.Add("-a + b")
	f.Add("((a + b) * (c - d))")

	// Seed corpus with edge cases
	f.Add("")
	f.Add(" ")
	f.Add("   ")
	f.Add("(")
	f.Add(")")
	f.Add("((")
	f.Add("))")
	f.Add("+++")
	f.Add("---")
	f.Add("***")
	f.Add("///")
	f.Add("a +")
	f.Add("+ b")
	f.Add("a b")
	f.Add("123")
	f.Add("a123")
	f.Add("_underscore")
	f.Add("CamelCase")

	// Seed corpus with potentially problematic characters
	f.Add("a & b")
	f.Add("a | b")
	f.Add("a $ b")
	f.Add("a @ b")
	f.Add("a # b")
	f.Add("a ! b")
	f.Add("a % b")
	f.Add("a ^ b")
	f.Add("a \\ b")
	f.Add("a \" b")
	f.Add("a ' b")
	f.Add("a ; b")
	f.Add("a : b")
	f.Add("a , b")
	f.Add("a . b")
	f.Add("a < b")
	f.Add("a > b")
	f.Add("a = b")
	f.Add("a ? b")
	f.Add("a ` b")
	f.Add("a ~ b")
	f.Add("a [ b")
	f.Add("a ] b")
	f.Add("a { b")
	f.Add("a } b")

	// Seed with unicode and special strings
	f.Add("日本語")
	f.Add("émoji")
	f.Add("a\nb")
	f.Add("a\tb")
	f.Add("a\rb")
	f.Add("a\x00b")

	f.Fuzz(func(t *testing.T, input string) {
		// The tokenizer should never panic, regardless of input
		// It may return an error, which is fine
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("tokenize panicked on input %q: %v", input, r)
			}
		}()

		tokens, err := tokenize(input)

		// If no error, verify basic token invariants
		if err == nil && len(tokens) > 0 {
			// Last token should always be EOF
			lastToken := tokens[len(tokens)-1]
			if lastToken.Type != TokenEOF {
				t.Errorf("last token should be EOF for input %q, got %v", input, lastToken.Type)
			}
		}
	})
}

// FuzzParseColumnExpression tests that the parser never panics on arbitrary input
func FuzzParseColumnExpression(f *testing.F) {
	// Seed corpus with valid expressions that should parse successfully
	f.Add("a + b", "test_table")
	f.Add("a - b", "test_table")
	f.Add("a * b", "test_table")
	f.Add("a / b", "test_table")
	f.Add("(a + b)", "test_table")
	f.Add("(a + b) * c", "test_table")
	f.Add("a + b * c", "test_table")
	f.Add("a + b - c * d / e", "test_table")
	f.Add("col_1 + col_2 + col_3", "test_table")
	f.Add("-a", "test_table")
	f.Add("-a + b", "test_table")
	f.Add("-(a + b)", "test_table")
	f.Add("((a))", "test_table")
	f.Add("(((a + b)))", "test_table")

	// Real-world expressions from YAML configs
	f.Add("num_fever_u5_male + num_fever_u5_female", "report.cht_form_097b")
	f.Add("(num_fever_u5_male + num_fever_u5_female) + (num_malaria_u5_male + num_malaria_u5_female)", "report.cht_form_097b")
	f.Add("num_diarrhea_u5_male_treated_with_ors + num_diarrhea_u5_female_treated_with_ors", "report.cht_form_097b")

	// Edge cases that should return errors (but not panic)
	f.Add("", "test_table")
	f.Add(" ", "test_table")
	f.Add("(", "test_table")
	f.Add(")", "test_table")
	f.Add("()", "test_table")
	f.Add("((a + b)", "test_table")
	f.Add("(a + b))", "test_table")
	f.Add("a +", "test_table")
	f.Add("+ b", "test_table")
	f.Add("* b", "test_table")
	f.Add("a + + b", "test_table")
	f.Add("a b", "test_table")
	f.Add("a & b", "test_table")
	f.Add("a | b", "test_table")

	// SQL injection attempts (should be rejected, not cause issues)
	f.Add("a; DROP TABLE users", "test_table")
	f.Add("a' OR '1'='1", "test_table")
	f.Add("a--comment", "test_table")
	f.Add("a/*comment*/b", "test_table")
	f.Add("1; DELETE FROM users WHERE 1=1", "test_table")

	// Various table names
	f.Add("col", "")
	f.Add("col", "public.table")
	f.Add("col", "schema.table_name")
	f.Add("col", "a.b.c.d")
	f.Add("col", "table with spaces")
	f.Add("col", "table\"with\"quotes")

	f.Fuzz(func(t *testing.T, expr string, tableName string) {
		// The parser should never panic, regardless of input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseColumnExpression panicked on expr=%q, table=%q: %v", expr, tableName, r)
			}
		}()

		sql, err := ParseColumnExpression(expr, tableName)

		// If parsing succeeded, verify basic SQL invariants
		if err == nil {
			// Result should not be empty for non-empty valid input
			if sql == "" && expr != "" {
				// Check if it's a valid expression that should produce output
				hasValidChar := false
				for _, c := range expr {
					if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
						hasValidChar = true
						break
					}
				}
				if hasValidChar {
					t.Logf("Empty SQL for seemingly valid expr: %q", expr)
				}
			}

			// SQL should not contain raw user input that could be dangerous
			// (The parser wraps columns in COALESCE/CAST, so raw identifiers should be quoted)
			if sql != "" && !strings.Contains(sql, "COALESCE") {
				t.Logf("SQL without COALESCE wrapper for expr=%q: %s", expr, sql)
			}
		}
	})
}

// FuzzParseColumnExpressionStress tests with longer/more complex expressions
func FuzzParseColumnExpressionStress(f *testing.F) {
	// Generate some longer seed expressions
	f.Add("a + b + c + d + e + f + g + h + i + j")
	f.Add("((((a + b) + c) + d) + e)")
	f.Add("a * b + c * d + e * f + g * h")
	f.Add("(a + b) * (c + d) * (e + f) * (g + h)")
	f.Add("a / b / c / d / e")
	f.Add("-a - -b - -c - -d")
	f.Add("(((((a)))))")
	f.Add("col_with_very_long_name_that_goes_on_and_on + another_extremely_long_column_name")

	f.Fuzz(func(t *testing.T, expr string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseColumnExpression panicked on expr=%q: %v", expr, r)
			}
		}()

		// Just ensure it doesn't panic or hang
		// Set a reasonable length limit to avoid memory issues
		if len(expr) > 10000 {
			return
		}

		ParseColumnExpression(expr, "test_table")
	})
}

// FuzzTokenizeAndParse tests the full pipeline: tokenize then parse
func FuzzTokenizeAndParse(f *testing.F) {
	f.Add("a + b * c")
	f.Add("(a - b) / c")
	f.Add("x + y + z")

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 1000 {
			return // Limit input size
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Pipeline panicked on input %q: %v", input, r)
			}
		}()

		// First tokenize
		tokens, tokenErr := tokenize(input)

		// If tokenization succeeded, try parsing
		if tokenErr == nil && len(tokens) > 0 {
			// Parsing should also not panic
			ParseColumnExpression(input, "test_table")
		}
	})
}
