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
		tempZip := "deca-template.zip"

		fmt.Printf("🚀 [Deca] Creating project '%s'...\n", projectName)
		fmt.Println("📥 Downloading template...")

		if err := downloadFile(tempZip, repoURL); err != nil {
			return err
		}
		defer os.Remove(tempZip)

		fmt.Println("📦 Extracting...")
		if err := unzip(tempZip, "."); err != nil {
			return err
		}

		files, _ := os.ReadDir(".")
		var extractedFolder string
		for _, f := range files {
			if f.IsDir() && strings.HasPrefix(f.Name(), "LubiXLubiX-") {
				extractedFolder = f.Name()
				break
			}
		}

		if extractedFolder == "" {
			return fmt.Errorf("template folder not found")
		}

		os.Rename(extractedFolder, projectName)

		fmt.Printf("\n✨ Project '%s' created successfully!\n", projectName)
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  deca lubix serve")
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
