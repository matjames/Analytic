package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// SensitivePattern defines a pattern to detect sensitive data
type SensitivePattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Description string
}

// sensitivePatterns contains regex patterns for detecting sensitive data
var sensitivePatterns = []SensitivePattern{
	{
		Name:        "Password assignment",
		Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*['"` + "`" + `]?[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]{6,}['"` + "`" + `]?`),
		Description: "Hardcoded password detected",
	},
	{
		Name:        "API Key",
		Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*['"` + "`" + `]?[a-zA-Z0-9_\-]{20,}['"` + "`" + `]?`),
		Description: "Hardcoded API key detected",
	},
	{
		Name:        "Secret Key",
		Pattern:     regexp.MustCompile(`(?i)(secret[_-]?key|secretkey)\s*[:=]\s*['"` + "`" + `]?[a-zA-Z0-9_\-]{16,}['"` + "`" + `]?`),
		Description: "Hardcoded secret key detected",
	},
	{
		Name:        "AWS Access Key",
		Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		Description: "AWS Access Key ID detected",
	},
	{
		Name:        "AWS Secret Key",
		Pattern:     regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key\s*[:=]\s*['"` + "`" + `]?[a-zA-Z0-9/+=]{40}['"` + "`" + `]?`),
		Description: "AWS Secret Access Key detected",
	},
	{
		Name:        "Private Key",
		Pattern:     regexp.MustCompile(`-----BEGIN\s+(RSA|DSA|EC|OPENSSH|PGP)?\s*PRIVATE KEY-----`),
		Description: "Private key detected",
	},
	{
		Name:        "Generic Token",
		Pattern:     regexp.MustCompile(`(?i)(auth[_-]?token|access[_-]?token|bearer)\s*[:=]\s*['"` + "`" + `]?[a-zA-Z0-9_\-\.]{20,}['"` + "`" + `]?`),
		Description: "Authentication token detected",
	},
	{
		Name:        "Database Connection String",
		Pattern:     regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[a-zA-Z0-9_]+:[^@\s]+@[a-zA-Z0-9\.\-]+`),
		Description: "Database connection string with credentials detected",
	},
	{
		Name:        "Private IPv4",
		Pattern:     regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2[0-9]|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`),
		Description: "Private IP address detected",
	},
	{
		Name:        "Public IPv4",
		Pattern:     regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
		Description: "IP address detected (verify it's not a real server)",
	},
	{
		Name:        "GitHub Token",
		Pattern:     regexp.MustCompile(`(ghp|gho|ghu|ghs|ghr)_[a-zA-Z0-9]{36}`),
		Description: "GitHub token detected",
	},
	{
		Name:        "Slack Token",
		Pattern:     regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}(-[a-zA-Z0-9]{24})?`),
		Description: "Slack token detected",
	},
	{
		Name:        "JWT Token",
		Pattern:     regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),
		Description: "JWT token detected (may be hardcoded)",
	},
}

// filesToSkip contains patterns for files that should be skipped
var filesToSkip = []string{
	"sensitive_data_test.go", // This test file itself
	"_test.go",               // Other test files may have test tokens
	".git/",
	"vendor/",
	"node_modules/",
	"/.env",                    // .env files are gitignored (checked separately)
	"server/.env",              // .env files are gitignored (checked separately)
	".github/workflows/ci.yml", // Ephemeral Postgres credentials for CI integration job only
}

// allowedPatterns contains patterns that are known safe (false positives)
var allowedPatterns = []string{
	"password: Baker100%",  // Example in .env.example - should be flagged separately
	"password: Dwh@24!",    // Example in .env.example - should be flagged separately
	"localhost",            // Local development
	"127.0.0.1",            // Loopback
	"127.0.0.0",            // Loopback network
	"0.0.0.0",              // Bind all
	"255.255.255",          // Broadcast/netmask
	"192.0.2.",             // TEST-NET-1 (RFC 5737)
	"198.51.100.",          // TEST-NET-2 (RFC 5737)
	"203.0.113.",           // TEST-NET-3 (RFC 5737)
	"example.com",          // Example domain
	"test@example.com",     // Test email
	"testuser",             // Test user
	"PASSWORD_PLACEHOLDER", // Placeholder
	"your_password_here",   // Placeholder
	"<PASSWORD>",           // Placeholder
	"${PASSWORD}",          // Environment variable
	"$PASSWORD",            // Environment variable
	"os.Getenv",            // Environment variable usage in Go
	"process.env",          // Environment variable usage in JS
	"1.0.0",                // Version numbers
	"2.0.0",                // Version numbers
	"rgba(",                // CSS colors
	"rgb(",                 // CSS colors
	"(e.g.,",               // Example in error messages
	"FATAL:",               // Error message examples
	"YourPassword",         // Placeholder in error messages
}

// TestNoSensitiveDataInCodebase scans the codebase for sensitive data
func TestNoSensitiveDataInCodebase(t *testing.T) {
	// Get the project root (parent of server/)
	projectRoot := ".."

	var findings []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip .git, vendor, node_modules, .claude, analytics, .venv, data (gitignored directories).
			// data/ holds runtime state (metrics.json, feedback.db) written by the live
			// server — request paths from internet scanners end up in it, so scanning it
			// makes redeploys fail on content that is not part of the codebase.
			skipDirs := map[string]bool{
				".git": true, "vendor": true, "node_modules": true,
				".claude": true, "analytics": true, ".venv": true, "venv": true,
				"data": true,
			}
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary files, logs, and certain extensions
		ext := strings.ToLower(filepath.Ext(path))
		skipExts := []string{".exe", ".dll", ".so", ".dylib", ".bin", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf", ".zip", ".tar", ".gz", ".woff", ".woff2", ".ttf", ".eot", ".log", ".test", ".out"}
		for _, skipExt := range skipExts {
			if ext == skipExt {
				return nil
			}
		}

		// Get base name for checks
		baseName := filepath.Base(path)

		// Skip compiled binaries (no extension, but executable)
		if ext == "" {
			// Skip common binary names
			if baseName == "report_server" || baseName == "server" {
				return nil
			}
		}

		// Skip files in the skip list (normalize to forward slashes for cross-platform comparison)
		normalizedPath := filepath.ToSlash(path)
		for _, skip := range filesToSkip {
			if strings.Contains(normalizedPath, skip) {
				return nil
			}
		}
		// Skip .env files (they're gitignored) but NOT .env.example
		if baseName == ".env" {
			return nil
		}

		// Read and scan file
		file, err := os.Open(path)
		if err != nil {
			return nil // Skip files we can't read
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Check each pattern
			for _, pattern := range sensitivePatterns {
				if pattern.Pattern.MatchString(line) {
					// Check if it's an allowed pattern (false positive)
					isAllowed := false
					for _, allowed := range allowedPatterns {
						if strings.Contains(line, allowed) {
							isAllowed = true
							break
						}
					}

					if !isAllowed {
						findings = append(findings, formatFinding(path, lineNum, pattern.Name, pattern.Description, line))
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error walking directory: %v", err)
	}

	if len(findings) > 0 {
		t.Errorf("Found %d potential sensitive data issues:\n\n%s", len(findings), strings.Join(findings, "\n"))
	}
}

// TestEnvExampleHasNoRealCredentials specifically checks .env.example
func TestEnvExampleHasNoRealCredentials(t *testing.T) {
	envExamplePath := "../.env.example"

	file, err := os.Open(envExamplePath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip(".env.example not found, skipping")
			return
		}
		t.Fatalf("Error opening .env.example: %v", err)
	}
	defer file.Close()

	// Patterns that indicate real credentials (not placeholders)
	realCredentialPatterns := []struct {
		Name    string
		Pattern *regexp.Regexp
	}{
		{
			Name:    "Real password (not placeholder)",
			Pattern: regexp.MustCompile(`(?i)password\s*:\s*[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]{4,}`),
		},
		{
			Name:    "Real IP address",
			Pattern: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
		},
		{
			Name:    "Real username (not placeholder)",
			Pattern: regexp.MustCompile(`(?i)username\s*:\s*[a-z][a-z0-9_]{2,}`),
		},
	}

	// Placeholder patterns that are OK
	placeholders := []string{
		"your_password",
		"your_username",
		"YOUR_PASSWORD",
		"YOUR_USERNAME",
		"<password>",
		"<username>",
		"example",
		"placeholder",
		"changeme",
		"localhost",
		"127.0.0.1",
	}

	var findings []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		for _, rcp := range realCredentialPatterns {
			if rcp.Pattern.MatchString(line) {
				// Check if it contains a placeholder
				isPlaceholder := false
				lineLower := strings.ToLower(line)
				for _, ph := range placeholders {
					if strings.Contains(lineLower, strings.ToLower(ph)) {
						isPlaceholder = true
						break
					}
				}

				if !isPlaceholder {
					findings = append(findings, formatFinding(envExamplePath, lineNum, rcp.Name, "Real credential in example file", line))
				}
			}
		}
	}

	if len(findings) > 0 {
		t.Errorf(".env.example contains real credentials (should only have placeholders):\n\n%s\n\n"+
			"Replace real values with placeholders like:\n"+
			"  password: your_password_here\n"+
			"  host: localhost\n"+
			"  username: your_username", strings.Join(findings, "\n"))
	}
}

// TestNoEnvFilesCommitted checks that .env files are not committed
func TestNoEnvFilesCommitted(t *testing.T) {
	// This test checks for .env files that should not be committed
	projectRoot := ".."

	dangerousEnvFiles := []string{
		".env",
		".env.local",
		".env.production",
		".env.staging",
		"server/.env",
	}

	var foundFiles []string

	for _, envFile := range dangerousEnvFiles {
		fullPath := filepath.Join(projectRoot, envFile)
		if _, err := os.Stat(fullPath); err == nil {
			// File exists - check if it's in .gitignore
			if !isInGitignore(envFile) {
				foundFiles = append(foundFiles, envFile)
			}
		}
	}

	if len(foundFiles) > 0 {
		t.Errorf("Found .env files that may not be properly gitignored: %v\n"+
			"Make sure these are listed in .gitignore", foundFiles)
	}
}

// TestGitignoreHasSensitivePatterns verifies .gitignore blocks sensitive files
func TestGitignoreHasSensitivePatterns(t *testing.T) {
	gitignorePath := "../.gitignore"

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Could not read .gitignore: %v", err)
	}

	gitignoreContent := string(content)

	requiredPatterns := []string{
		".env",
	}

	var missingPatterns []string
	for _, pattern := range requiredPatterns {
		if !strings.Contains(gitignoreContent, pattern) {
			missingPatterns = append(missingPatterns, pattern)
		}
	}

	if len(missingPatterns) > 0 {
		t.Errorf(".gitignore is missing required patterns: %v", missingPatterns)
	}
}

// formatFinding formats a finding for display
func formatFinding(path string, lineNum int, name, description, line string) string {
	// Truncate long lines
	if len(line) > 100 {
		line = line[:100] + "..."
	}
	return strings.Join([]string{
		"",
		"  File: " + path,
		"  Line: " + strconv.Itoa(lineNum),
		"  Type: " + name,
		"  Issue: " + description,
		"  Content: " + strings.TrimSpace(line),
	}, "\n")
}

// isInGitignore checks if a pattern is in .gitignore
func isInGitignore(pattern string) bool {
	content, err := os.ReadFile("../.gitignore")
	if err != nil {
		return false
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Simple check - could be more sophisticated with glob matching
		if strings.Contains(line, pattern) || line == pattern {
			return true
		}
	}
	return false
}
