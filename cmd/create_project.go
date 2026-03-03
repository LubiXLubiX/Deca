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

var (
	createProjectTemplateURL string
)

var createProjectCmd = &cobra.Command{
	Use:   "create-project [name]",
	Short: "Create a new LubiX project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		repoURL := createProjectTemplateURL
		if strings.TrimSpace(repoURL) == "" {
			repoURL = "https://github.com/LubiXLubiX/LubiXLubix/archive/refs/heads/main.zip"
		}
		tempZip := "deca-template.zip"
		tempDir, err := os.MkdirTemp("", "deca-template-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		fmt.Printf("[+] [Deca] Creating project '%s'...\n", projectName)
		fmt.Println("[+] Downloading template...")

		if err := downloadFile(tempZip, repoURL); err != nil {
			return err
		}
		defer os.Remove(tempZip)

		fmt.Println("[+] Extracting...")
		if err := unzip(tempZip, tempDir); err != nil {
			return err
		}

		extractedFolder, err := findSingleRootDir(tempDir)
		if err != nil {
			return err
		}

		if _, err := os.Stat(projectName); err == nil {
			return fmt.Errorf("destination already exists: %s", projectName)
		}
		if err := os.Rename(extractedFolder, projectName); err != nil {
			return err
		}

		fmt.Printf("\n[OK] Project '%s' created successfully\n", projectName)
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  deca lubix serve")
		return nil
	},
}

func init() {
	createProjectCmd.Flags().StringVar(&createProjectTemplateURL, "template", "", "Template ZIP URL (optional)")
}

func findSingleRootDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var roots []string
	for _, e := range entries {
		if e.IsDir() {
			roots = append(roots, filepath.Join(dir, e.Name()))
		}
	}
	if len(roots) == 1 {
		return roots[0], nil
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("template folder not found")
	}
	return "", fmt.Errorf("template extraction produced multiple root folders")
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to download template: %s", resp.Status)
	}
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
		cleanName := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip path: %s", f.Name)
		}
		fpath := filepath.Join(dest, cleanName)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}
