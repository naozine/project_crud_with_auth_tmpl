package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const templateModule = "github.com/naozine/project_crud_with_auth_tmpl"

func main() {
	fmt.Println("🚀 GOTH Stack Template Setup Tool (replace方式)")
	fmt.Println("------------------------------------------------")
	fmt.Println("この方式はテンプレートからのマージを容易にします。")
	fmt.Println("完全に独立したプロジェクトにする場合は `go run ./cmd/eject` を使用してください。")
	fmt.Println()

	// 1. Detect current module name
	currentModule, err := getCurrentModuleName()
	if err != nil {
		fmt.Printf("Error detecting current module name: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current module name: %s\n", currentModule)

	// Check if already initialized (has replace directive)
	if hasReplaceDirective() {
		fmt.Println("\n⚠️  このプロジェクトは既にinit済みです（replaceディレクティブが存在します）。")
		fmt.Println("再度初期化する場合は、まずgo.modのreplaceディレクティブを削除してください。")
		os.Exit(1)
	}

	// 2. Ask for new module name
	defaultModule := suggestModuleName()
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

	fmt.Printf("\n新しいモジュール名: %s\n", newModule)
	fmt.Printf("replaceディレクティブ: %s => ./\n\n", templateModule)

	// 3. Update go.mod
	if err := updateGoMod(newModule); err != nil {
		fmt.Printf("Error updating go.mod: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Updated: go.mod")

	// 4. Cleanup and create essential files
	fmt.Println("\nCleaning up and creating essential files...")
	filesToRemove := []string{
		"go.sum",
		"app.db",
		"magiclink.db",
		"app", // binary
		"tmp", // directory
		//".git", // Use this templateでリポジトリを作ってからクローンして使う想定なので、消さない
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

	// 5. Initialize new git repository
	fmt.Println("\nInitializing new git repository...")
	if err := initGitRepo(); err != nil {
		fmt.Printf("Warning: Failed to initialize git repository: %v\n", err)
		fmt.Println("You may need to run 'git init' manually.")
	} else {
		fmt.Println("Git repository initialized.")
	}

	fmt.Println("\n✅ Setup complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. go run github.com/a-h/templ/cmd/templ@latest generate")
	fmt.Println("  2. go mod tidy")
	fmt.Println("  3. go build -o app cmd/server/main.go")
	fmt.Println("\n💡 Tips:")
	fmt.Println("  - テンプレートの更新をマージするには:")
	fmt.Println("      git remote add template https://github.com/naozine/project_crud_with_auth_tmpl.git")
	fmt.Println("      git fetch template")
	fmt.Println("      git merge template/main --allow-unrelated-histories")
	fmt.Println("  - プロジェクトが成熟したら `go run ./cmd/eject` で完全に独立できます")
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

func updateGoMod(newModule string) error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	goVersionLine := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			// Replace module line
			newLines = append(newLines, "module "+newModule)
		} else if strings.HasPrefix(line, "go ") {
			goVersionLine = line
			newLines = append(newLines, line)
		} else {
			newLines = append(newLines, line)
		}
	}

	// Add replace directive after go version line if not exists
	result := strings.Join(newLines, "\n")
	if goVersionLine != "" && !strings.Contains(result, "replace "+templateModule) {
		// Find position after go version line and add replace
		result = strings.Replace(result, goVersionLine, goVersionLine+"\n\nreplace "+templateModule+" => ./", 1)
	}

	return os.WriteFile("go.mod", []byte(result), 0644)
}

func initGitRepo() error {
	cmd := exec.Command("git", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
