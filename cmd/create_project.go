package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var createProjectCmd = &cobra.Command{
	Use:   "create-project [name]",
	Short: "Create a new LubiX project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		repoURL := "https://github.com/LubiXLubiX/LubiXLubiX/archive/refs/heads/main.zip"
		tempZip := "lubix-template.zip"

		fmt.Printf("🚀 Creating project '%s'...\n", projectName)
		fmt.Println("📥 Downloading template from GitHub...")

		// 1. Download ZIP
		if err := downloadFile(tempZip, repoURL); err != nil {
			return fmt.Errorf("failed to download template: %w", err)
		}
		defer os.Remove(tempZip)

		// 2. Extract ZIP
		fmt.Println("📦 Extracting files...")
		if err := unzip(tempZip, "."); err != nil {
			return fmt.Errorf("failed to unzip template: %w", err)
		}

		// 3. Rename/Move folder
		// GitHub ZIP extracts to <repo>-<branch>
		files, _ := os.ReadDir(".")
		var extractedFolder string
		for _, f := range files {
			if f.IsDir() && strings.HasPrefix(f.Name(), "LubiXLubiX-") {
				extractedFolder = f.Name()
				break
			}
		}

		if extractedFolder == "" {
			return fmt.Errorf("could not find extracted template folder (expected prefix LubiXLubiX-)")
		}

		if err := os.Rename(extractedFolder, projectName); err != nil {
			return fmt.Errorf("failed to rename folder to %s: %w", projectName, err)
		}

		fmt.Printf("\n✨ Project '%s' created successfully!\n", projectName)
		fmt.Println("------------------------------------")
		fmt.Printf("Next steps:\n")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Printf("  composer install\n")
		fmt.Printf("  npm install\n")
		fmt.Printf("  lubix run dev\n")
		fmt.Println("------------------------------------")

		return nil
	},
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
