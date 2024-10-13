// copyfilestotemp.go - simple program to copy the files i need to share into a temp directory without retaining the hierarchy
// this is probably only necessary for me, on the specific setup i'm using

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	dirsToInclude  = []string{"cmd", "core", "deployments", "docs", "pkg", "scripts", "test", "web"}
	filesToInclude = []string{"SPEC.md", "DICEROLL-SPEC.md", "Dockerfile", "start.sh", "go.mod"}
	filesToExclude = []string{".env", ".gitignore", ".env_example"}
)

func main() {
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	tempDir := filepath.Join(baseDir, fmt.Sprintf("temp_copy_%s", time.Now().Format("20060102_150405")))
	err = os.Mkdir(tempDir, 0755)
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return
	}

	fmt.Println("Temporary directory created:", tempDir)

	// Copy specified directories
	for _, dir := range dirsToInclude {
		err := filepath.Walk(filepath.Join(baseDir, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if shouldExclude(info.Name()) {
				fmt.Println("Excluded:", path)
				return nil
			}
			return copyFileToTemp(path, tempDir, dir)
		})
		if err != nil {
			fmt.Println("Error processing directory", dir, ":", err)
		}
	}

	// Copy specified files from base directory
	for _, file := range filesToInclude {
		srcPath := filepath.Join(baseDir, file)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Println("Warning: File not found -", srcPath)
			continue
		}
		err := copyFileToTemp(srcPath, tempDir, "")
		if err != nil {
			fmt.Println("Error copying file", file, ":", err)
		}
	}

	// Generate directory hierarchy file
	err = generateDirectoryHierarchy(baseDir, tempDir)
	if err != nil {
		fmt.Println("Error generating directory hierarchy:", err)
	} else {
		fmt.Println("Directory hierarchy file generated successfully")
	}

	fmt.Println("File copying process completed.")
	fmt.Println("Files have been copied to", tempDir)
}

func shouldExclude(filename string) bool {
	for _, exclude := range filesToExclude {
		if strings.EqualFold(filename, exclude) {
			return true
		}
	}
	return false
}

func copyFileToTemp(src, tempDir, sourceDir string) error {
	fileName := filepath.Base(src)
	dst := filepath.Join(tempDir, fileName)

	// If file already exists, append source directory to the filename
	if _, err := os.Stat(dst); err == nil {
		ext := filepath.Ext(fileName)
		nameWithoutExt := strings.TrimSuffix(fileName, ext)
		newFileName := fmt.Sprintf("%s-%s%s", nameWithoutExt, sourceDir, ext)
		dst = filepath.Join(tempDir, newFileName)
	}

	fmt.Println("Copying", src, "to", dst)

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func generateDirectoryHierarchy(baseDir, tempDir string) error {
	hierarchyFile := filepath.Join(tempDir, "directory_hierarchy.txt")
	file, err := os.Create(hierarchyFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString("Directory Hierarchy:\n\n")
	if err != nil {
		return err
	}

	for _, dir := range dirsToInclude {
		err = filepath.Walk(filepath.Join(baseDir, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			if info.IsDir() {
				_, err = file.WriteString(fmt.Sprintf("%s/\n", relPath))
			} else if !shouldExclude(info.Name()) {
				_, err = file.WriteString(fmt.Sprintf("%s\n", relPath))
			}

			return err
		})
		if err != nil {
			return err
		}
	}

	for _, fileToInclude := range filesToInclude {
		_, err = file.WriteString(fmt.Sprintf("%s\n", fileToInclude))
		if err != nil {
			return err
		}
	}

	return nil
}
