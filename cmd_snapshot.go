package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create and restore project snapshots",
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create [--output file.tar.gz]",
	Short: "Create a compressed snapshot of the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = fmt.Sprintf("%s-snapshot-%d.tar.gz", projectName, os.Getpid())
		}

		dataDir, _ := getDataDir()
		projectPath := filepath.Join(dataDir, projectName)

		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			return fmt.Errorf("project %s not found", projectName)
		}

		file, err := os.Create(output)
		if err != nil {
			return err
		}
		defer file.Close()

		gw := gzip.NewWriter(file)
		defer gw.Close()
		tw := tar.NewWriter(gw)
		defer tw.Close()

		err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(filepath.Dir(projectPath), path)
			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(tw, f)
				return err
			}
			return nil
		})

		if err != nil {
			return err
		}

		fmt.Printf("✅ Snapshot created: %s\n", output)
		return nil
	},
}

var snapshotRestoreCmd = &cobra.Command{
	Use:   "restore <file.tar.gz>",
	Short: "Restore a project from a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotFile := args[0]
		
		file, err := os.Open(snapshotFile)
		if err != nil {
			return err
		}
		defer file.Close()

		gr, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gr.Close()
		tr := tar.NewReader(gr)

		dataDir, _ := getDataDir()

		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			target := filepath.Join(dataDir, header.Name)
			
			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			case tar.TypeReg:
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				if _, err := io.Copy(f, tr); err != nil {
					f.Close()
					return err
				}
				f.Close()
			}
		}

		fmt.Printf("✅ Snapshot restored to: %s\n", dataDir)
		return nil
	},
}

func init() {
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotRestoreCmd)
	snapshotCreateCmd.Flags().StringP("output", "o", "", "Output filename")
	snapshotCreateCmd.Flags().StringP("project", "p", "", "Project name")
}
