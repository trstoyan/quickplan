package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check project health and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, _ := getDataDir()
		projectManager := NewProjectDataManager(dataDir, NewVersionManager(version))

		if globalJSON {
			report := map[string]interface{}{
				"project":  projectName,
				"lock":     map[string]interface{}{"status": "OK"},
				"schema":   map[string]interface{}{"status": "OK"},
				"deps":     map[string]interface{}{"status": "OK"},
				"registry": map[string]interface{}{"status": "OK"},
			}

			// 1. Lock
			stale, lock, err := projectManager.IsLockStale(projectName)
			if err != nil {
				report["lock"] = map[string]string{"status": "OK", "message": "No active lock"}
			} else if stale {
				report["lock"] = map[string]interface{}{"status": "WARN", "message": "Stale lock found", "pid": lock.PID}
			} else {
				report["lock"] = map[string]interface{}{"status": "LOCKED", "message": "Active lock found", "pid": lock.PID}
			}

			// 2. Schema
			v11, err := projectManager.LoadProjectV11(projectName)
			if err == nil {
				if err := ValidateProjectV11(v11); err != nil {
					report["schema"] = map[string]interface{}{"status": "ERROR", "message": fmt.Sprintf("Invalid v1.1: %v", err)}
				} else {
					report["schema"] = map[string]string{"status": "OK", "version": "v1.1"}
				}
			} else if os.IsNotExist(err) {
				legacy, err := projectManager.LoadProjectData(projectName)
				if err != nil {
					report["schema"] = map[string]interface{}{"status": "ERROR", "message": fmt.Sprintf("Load failed: %v", err)}
				} else {
					report["schema"] = map[string]string{"status": "OK", "version": legacy.Version}
				}
			} else {
				report["schema"] = map[string]interface{}{"status": "ERROR", "message": fmt.Sprintf("Load failed: %v", err)}
			}

			// 3. Deps
			views, _, err := projectManager.GetTaskViews(projectName)
			if err == nil {
				taskIDs := make(map[string]bool)
				for _, v := range views {
					taskIDs[v.ID] = true
				}
				orphans := 0
				for _, v := range views {
					for _, dep := range v.DependsOn {
						if !taskIDs[dep] {
							orphans++
						}
					}
				}
				if orphans > 0 {
					report["deps"] = map[string]interface{}{"status": "WARN", "orphans": orphans}
				} else {
					report["deps"] = map[string]string{"status": "OK", "message": "No orphans"}
				}
			} else {
				report["deps"] = map[string]string{"status": "SKIPPED", "message": "Load failed"}
			}

			// 4. Registry
			registryURL := os.Getenv("QUICKPLAN_REGISTRY_URL")
			if registryURL == "" {
				registryURL = "http://localhost:8081"
			}
			client := http.Client{Timeout: 2 * time.Second}
			req, reqErr := http.NewRequest(http.MethodGet, registryURL+"/api/v1/info", nil)
			if reqErr != nil {
				report["registry"] = map[string]interface{}{"status": "ERROR", "url": registryURL, "message": reqErr.Error()}
				jsonOut, _ := json.Marshal(report)
				fmt.Println(string(jsonOut))
				return nil
			}
			applyWebAuth(req)
			resp, err := client.Do(req)
			if err != nil {
				report["registry"] = map[string]interface{}{"status": "ERROR", "url": registryURL, "message": "Unreachable"}
			} else {
				defer resp.Body.Close()
				var info struct {
					Status string `json:"status"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
					report["registry"] = map[string]interface{}{"status": "OK", "url": registryURL, "remote_status": info.Status}
				} else {
					report["registry"] = map[string]interface{}{"status": "WARN", "url": registryURL, "message": "Malformed response"}
				}
			}

			jsonOut, _ := json.Marshal(report)
			fmt.Println(string(jsonOut))
			return nil
		}

		fmt.Printf("🏥 Quick Plan Doctor: Checking project '%s'...\n", projectName)

		// 1. Check Lock Status
		fmt.Print("  [1/4] Lock status: ")
		stale, lock, err := projectManager.IsLockStale(projectName)
		if err != nil {
			fmt.Println("✅ No active lock")
		} else if stale {
			fmt.Printf("⚠️  STALE (held by PID %d on %s)\n", lock.PID, lock.Host)
		} else {
			fmt.Printf("🔒 LOCKED (held by PID %d on %s)\n", lock.PID, lock.Host)
		}

		// 2. Check Schema Validity
		fmt.Print("  [2/4] Schema validity: ")
		v11, err := projectManager.LoadProjectV11(projectName)
		if err == nil {
			if err := ValidateProjectV11(v11); err != nil {
				fmt.Printf("❌ Invalid v1.1: %v\n", err)
			} else {
				fmt.Println("✅ Valid v1.1 project.yaml")
			}
		} else if os.IsNotExist(err) {
			// Check legacy
			legacy, err := projectManager.LoadProjectData(projectName)
			if err != nil {
				fmt.Printf("❌ Error loading legacy tasks.yaml: %v\n", err)
			} else {
				fmt.Printf("✅ Valid legacy tasks.yaml (v%s)\n", legacy.Version)
			}
		} else {
			fmt.Printf("❌ Error loading project: %v\n", err)
		}

		// 3. Check Orphan Dependencies
		fmt.Print("  [3/4] Orphan dependencies: ")
		views, _, err := projectManager.GetTaskViews(projectName)
		if err == nil {
			taskIDs := make(map[string]bool)
			for _, v := range views {
				taskIDs[v.ID] = true
			}

			orphans := 0
			for _, v := range views {
				for _, dep := range v.DependsOn {
					if !taskIDs[dep] {
						orphans++
					}
				}
			}
			if orphans > 0 {
				fmt.Printf("⚠️  Found %d orphaned dependencies\n", orphans)
			} else {
				fmt.Println("✅ None")
			}
		} else {
			fmt.Println("SKIPPED (load failed)")
		}

		// 4. Check Registry Connectivity
		fmt.Print("  [4/4] Registry status: ")
		registryURL := os.Getenv("QUICKPLAN_REGISTRY_URL")
		if registryURL == "" {
			registryURL = "http://localhost:8081"
		}

		client := http.Client{Timeout: 2 * time.Second}
		req, reqErr := http.NewRequest(http.MethodGet, registryURL+"/api/v1/info", nil)
		if reqErr != nil {
			fmt.Printf("❌ %v\n", reqErr)
			return nil
		}
		applyWebAuth(req)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ UNREACHABLE (%s)\n", registryURL)
		} else {
			defer resp.Body.Close()
			var info struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&info); err == nil {
				fmt.Printf("✅ %s (%s)\n", strings.ToUpper(info.Status), registryURL)
			} else {
				fmt.Printf("⚠️  CONNECTED but malformed response (%s)\n", registryURL)
			}
		}

		return nil

	},
}

func init() {
	doctorCmd.Flags().StringP("project", "p", "", "Project name")
}
