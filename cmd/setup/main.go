package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
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
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter new module name (e.g., github.com/user/my-app) [default: %s]: ", currentModule)
	newModule, _ := reader.ReadString('\n')
	newModule = strings.TrimSpace(newModule)

	if newModule == "" {
		newModule = currentModule
	}

	if newModule == templateModule {
		// If user didn't change anything and it's still the template name, ask again strictly or just proceed?
		// Let's warn.
		fmt.Println("Warning: You are keeping the template module name.")
	}

	fmt.Printf("\nReplacing '%s' (and '%s') -> '%s'...\n", currentModule, templateModule, newModule)

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

	// 4. Cleanup
	fmt.Println("\nCleaning up...")
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

	fmt.Println("\n✅ Setup complete!")
	fmt.Println("Next steps:")
	fmt.Println("  1. go mod tidy")
	fmt.Println("  2. go run github.com/a-h/templ/cmd/templ@latest generate")
	fmt.Println("  3. go build -o app cmd/server/main.go")
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
