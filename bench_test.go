package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func BenchmarkLoadProject100Tasks(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench-load-*")
	defer os.RemoveAll(tmpDir)
	
	projectName := "bench-proj"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	tasks := make([]TaskV11, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = TaskV11{
			ID:     fmt.Sprintf("t-%d", i),
			Name:   fmt.Sprintf("Task %d", i),
			Status: "PENDING",
		}
	}
	
	v11 := ProjectV11{
		SchemaVersion: "1.1",
		Tasks:         tasks,
	}
	data, _ := yaml.Marshal(v11)
	os.WriteFile(filepath.Join(projectPath, "project.yaml"), data, 0644)

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pdm.LoadProjectV11(projectName)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAppendEvent1000(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench-append-*")
	defer os.RemoveAll(tmpDir)
	
	projectName := "bench-append"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	v11 := ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{Name: "bench"},
		Tasks:   []TaskV11{},
		Events:  []Event{},
	}
	data, _ := yaml.Marshal(v11)
	os.WriteFile(filepath.Join(projectPath, "project.yaml"), data, 0644)

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := Event{
			Timestamp: time.Now(),
			Type:      "BENCHMARK",
			Actor:     "bench",
		}
		err := pdm.AppendEvent(projectName, event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDependencyValidation(b *testing.B) {
	tasks := make([]TaskV11, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = TaskV11{
			ID:     fmt.Sprintf("t-%d", i),
			Status: "PENDING",
		}
		if i > 0 {
			tasks[i].DependsOn = []string{fmt.Sprintf("t-%d", i-1)}
		}
	}
	
	project := &ProjectV11{
		SchemaVersion: "1.1",
		Tasks:         tasks,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateProjectV11(project)
		if err != nil {
			b.Fatal(err)
		}
	}
}
