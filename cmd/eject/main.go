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
	fmt.Println("🚀 GOTH Stack Template Eject Tool (一括置換方式)")
	fmt.Println("------------------------------------------------")
	fmt.Println("このツールはテンプレートのimportパスを完全に置き換え、")
	fmt.Println("独立したプロジェクトにします。")
	fmt.Println()
	fmt.Println("⚠️  注意: eject後はテンプレートからのマージが困難になります。")
	fmt.Println()

	// 1. Detect current module name
	currentModule, err := getCurrentModuleName()
	if err != nil {
		fmt.Printf("Error detecting current module name: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current module name: %s\n", currentModule)

	// Check if this is a fresh template (no replace directive, module is template)
	hasReplace := hasReplaceDirective()

	// 2. Ask for new module name
	var defaultModule string
	if hasReplace {
		// Already initialized with replace, use current module as default
		defaultModule = currentModule
	} else if currentModule == templateModule {
		// Fresh template
		defaultModule = suggestModuleName()
	} else {
		// Already has different module name
		defaultModule = currentModule
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\nEnter new module name (e.g., github.com/user/my-app) [default: %s]: ", defaultModule)
	newModule, _ := reader.ReadString('\n')
	newModule = strings.TrimSpace(newModule)

	if newModule == "" {
		newModule = defaultModule
	}

	if newModule == templateModule {
		fmt.Println("Error: 新しいモジュール名はテンプレートと異なる必要があります。")
		os.Exit(1)
	}

	// Confirmation
	fmt.Println()
	fmt.Println("以下の置換を行います:")
	fmt.Printf("  - %s -> %s\n", templateModule, newModule)
	if currentModule != templateModule && currentModule != newModule {
		fmt.Printf("  - %s -> %s\n", currentModule, newModule)
	}
	fmt.Println()
	fmt.Print("続行しますか? (y/N): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}

	fmt.Printf("\nReplacing module paths...\n\n")

	// 3. Walk and Replace
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".github" {
				return filepath.SkipDir
			}
			if d.Name() == "tmp" || d.Name() == "bin" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
		}

		if !d.IsDir() {
			ext := filepath.Ext(path)
			validExts := map[string]bool{
				".go": true, ".mod": true, ".sum": true,
				".md": true, ".yaml": true, ".yml": true,
				".toml": true, ".json": true, ".sql": true,
				".templ": true, ".html": true, ".css": true, ".js": true,
				".gitignore": true, ".bypass_emails": true,
			}

			if validExts[ext] || d.Name() == "Dockerfile" || d.Name() == "Makefile" {
				oldStrings := []string{templateModule}
				if currentModule != templateModule && currentModule != newModule {
					oldStrings = append(oldStrings, currentModule)
				}
				if err := replaceInFile(path, oldStrings, newModule); err != nil {
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

	// 4. Remove replace directive from go.mod if exists
	if hasReplace {
		if err := removeReplaceDirective(); err != nil {
			fmt.Printf("Warning: Failed to remove replace directive: %v\n", err)
		} else {
			fmt.Println("Removed replace directive from go.mod")
		}
	}

	// 5. Cleanup and create essential files
	fmt.Println("\nCleaning up and creating essential files...")
	filesToRemove := []string{
		"go.sum",
		"app.db",
		"magiclink.db",
		"app",  // binary
		"tmp",  // directory
		".git", // テンプレートのgit履歴を削除
	}

	for _, f := range filesToRemove {
		if err := os.RemoveAll(f); err == nil {
			if _, statErr := os.Stat(f); os.IsNotExist(statErr) {
				fmt.Printf("Removed: %s\n", f)
			}
		}
	}

	// Create .bypass_emails if it doesn't exist
	createBypassEmailsFile()

	// Create .env file if it doesn't exist
	createEnvFile()

	// 6. Initialize new git repository
	fmt.Println("\nInitializing new git repository...")
	if err := initGitRepo(); err != nil {
		fmt.Printf("Warning: Failed to initialize git repository: %v\n", err)
		fmt.Println("You may need to run 'git init' manually.")
	} else {
		fmt.Println("Git repository initialized.")
	}

	fmt.Println("\n✅ Eject complete!")
	fmt.Printf("プロジェクトは '%s' として独立しました。\n", newModule)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. go run github.com/a-h/templ/cmd/templ@latest generate")
	fmt.Println("  2. go mod tidy")
	fmt.Println("  3. go build -o app cmd/server/main.go")
}

func suggestModuleName() string {
	wd, _ := os.Getwd()
	dirName := filepath.Base(wd)

	gitUser := getGitConfig("github.user")
	if gitUser == "" {
		gitUser = getGitConfig("user.name")
	}

	gitUser = strings.ToLower(strings.ReplaceAll(gitUser, " ", ""))

	if gitUser == "" {
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

func hasReplaceDirective() bool {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "replace "+templateModule)
}

func removeReplaceDirective() error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		// Skip replace directive line
		if strings.HasPrefix(strings.TrimSpace(line), "replace "+templateModule) {
			skipNext = false
			continue
		}
		// Skip empty lines right before replace (cleanup)
		if strings.TrimSpace(line) == "" && skipNext {
			continue
		}
		newLines = append(newLines, line)
		skipNext = strings.TrimSpace(line) == ""
	}

	return os.WriteFile("go.mod", []byte(strings.Join(newLines, "\n")), 0644)
}

func initGitRepo() error {
	cmd := exec.Command("git", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func replaceInFile(path string, olds []string, new string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	strContent := string(content)
	newContent := strContent

	for _, old := range olds {
		if old != "" && old != new {
			newContent = strings.ReplaceAll(newContent, old, new)
		}
	}

	if newContent != strContent {
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

func createBypassEmailsFile() {
	bypassFilePath := ".bypass_emails"
	if _, err := os.Stat(bypassFilePath); os.IsNotExist(err) {
		content := `# Add email addresses here (one per line) to bypass email sending in development.
# Example:
# test@example.com
`
		if err := os.WriteFile(bypassFilePath, []byte(content), 0644); err != nil {
			fmt.Printf("Warning: Failed to create %s: %v\n", bypassFilePath, err)
		} else {
			fmt.Printf("Created: %s\n", bypassFilePath)
		}
	} else {
		fmt.Printf("Found existing: %s\n", bypassFilePath)
	}
}

func createEnvFile() {
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
		if err := os.WriteFile(envFilePath, []byte(content), 0644); err != nil {
			fmt.Printf("Warning: Failed to create %s: %v\n", envFilePath, err)
		} else {
			fmt.Printf("Created: %s\n", envFilePath)
		}
	} else {
		fmt.Printf("Found existing: %s\n", envFilePath)
	}
}
