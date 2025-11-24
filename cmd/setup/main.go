package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const templateModule = "github.com/naozine/project_crud_with_auth_tmpl"

func main() {
	fmt.Println("🚀 GOTH Stack Template Setup Tool")
	fmt.Println("--------------------------------")

	// 1. Detect current module name
	currentModule, err := getCurrentModuleName()
	if err != nil {
		fmt.Printf("Error detecting current module name: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current module name: %s\n", currentModule)

	// 2. Ask for new module name
	defaultModule := suggestModuleName()
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter new module name (e.g., github.com/user/my-app) [default: %s]: ", defaultModule)
	newModule, _ := reader.ReadString('\n')
	newModule = strings.TrimSpace(newModule)

	if newModule == "" {
		newModule = defaultModule
	}

	if newModule == templateModule {
		fmt.Println("Warning: You are keeping the template module name.")
	}

	fmt.Printf("\nReplacing '%s' (and '%s') -> '%s'\n\n", currentModule, templateModule, newModule)

	// 3. Walk and Replace
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".github" { // Skip .git, .idea, etc. but keep .github
				return filepath.SkipDir
			}
			if d.Name() == "tmp" || d.Name() == "bin" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
		}

		// Skip the setup tool itself (optional, but good to keep clean)
		// logic: we still want to replace content inside it if needed, but mostly we focus on code.

		if !d.IsDir() {
			// Check file extension
			ext := filepath.Ext(path)
			validExts := map[string]bool{
				".go": true, ".mod": true, ".sum": true,
				".md": true, ".yaml": true, ".yml": true,
				".toml": true, ".json": true, ".sql": true,
				".templ": true, ".html": true, ".css": true, ".js": true,
				".gitignore": true, ".bypass_emails": true,
			}

			// Also check for specific files without extensions
			if validExts[ext] || d.Name() == "Dockerfile" || d.Name() == "Makefile" {
				// Replace both currentModule AND templateModule
				if err := replaceInFile(path, []string{currentModule, templateModule}, newModule); err != nil {
					fmt.Printf("Failed to process %s: %v\n", path, err)
				}
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directories: %v\n", err)
		os.Exit(1)
	}

	// 4. Cleanup and create essential files
	fmt.Println("\nCleaning up and creating essential files...")
	filesToRemove := []string{
		"go.sum",
		"app.db",
		"magiclink.db",
		"app", // binary
		"tmp", // directory
	}

	for _, f := range filesToRemove {
		os.RemoveAll(f)
		fmt.Printf("Removed: %s\n", f)
	}

	// Create .bypass_emails if it doesn't exist
	bypassFilePath := ".bypass_emails"
	if _, err := os.Stat(bypassFilePath); os.IsNotExist(err) {
		content := `# Add email addresses here (one per line) to bypass email sending in development.
# Example:
# test@example.com
`
		err = os.WriteFile(bypassFilePath, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Warning: Failed to create %s: %v\n", bypassFilePath, err)
		} else {
			fmt.Printf("Created: %s\n", bypassFilePath)
		}
	} else {
		fmt.Printf("Found existing: %s\n", bypassFilePath)
	}

	// Create .env file if it doesn't exist
	envFilePath := ".env"
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		content := `# Application Server Address
SERVER_ADDR="http://localhost:8080"

# SMTP Configuration for Magic Link (Gmail example)
# Replace with your actual SMTP server details
# SMTP_HOST="smtp.gmail.com"
# SMTP_PORT="587"
# SMTP_USERNAME="your-email@gmail.com"
# SMTP_PASSWORD="your-app-password"
# SMTP_FROM="your-email@gmail.com"
# SMTP_FROM_NAME="Your App Name"

# Optional: Development bypass emails for magic link testing (e.g. test@example.com)
# DEV_BYPASS_EMAILS_FILE=".bypass_emails"
`
		err = os.WriteFile(envFilePath, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Warning: Failed to create %s: %v\n", envFilePath, err)
		} else {
			fmt.Printf("Created: %s\n", envFilePath)
		}
	} else {
		fmt.Printf("Found existing: %s\n", envFilePath)
	}

	fmt.Println("\n✅ Setup complete!")
	fmt.Println("Next steps:")
	fmt.Println("  1. go run github.com/a-h/templ/cmd/templ@latest generate")
	fmt.Println("  2. go mod tidy")
	fmt.Println("  3. go build -o app cmd/server/main.go")
}

func suggestModuleName() string {
	// 1. Get current directory name
	wd, _ := os.Getwd()
	dirName := filepath.Base(wd)

	// 2. Try to get git user name
	gitUser := getGitConfig("github.user")
	if gitUser == "" {
		gitUser = getGitConfig("user.name")
	}

	// Clean up git user (remove spaces, lowercase)
	gitUser = strings.ToLower(strings.ReplaceAll(gitUser, " ", ""))

	if gitUser == "" {
		// Fallback to OS user
		gitUser = os.Getenv("USER")
	}

	if gitUser == "" {
		gitUser = "user"
	}

	return fmt.Sprintf("github.com/%s/%s", gitUser, dirName)
}

func getGitConfig(key string) string {
	cmd := exec.Command("git", "config", "--get", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getCurrentModuleName() (string, error) {
	f, err := os.Open("go.mod")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module name not found in go.mod")
}

func replaceInFile(path string, olds []string, new string) error {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	strContent := string(content)
	newContent := strContent

	// Replace all occurrences of all old strings
	for _, old := range olds {
		if old != "" && old != new {
			newContent = strings.ReplaceAll(newContent, old, new)
		}
	}

	// If changed, write back
	if newContent != strContent {
		// Preserve permission
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		err = os.WriteFile(path, []byte(newContent), info.Mode())
		if err != nil {
			return err
		}
		fmt.Printf("Updated: %s\n", path)
	}

	return nil
}
