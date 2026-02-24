package main

import (
	"testing"
)

func TestValidateProjectV11(t *testing.T) {
	tests := []struct {
		name    string
		project *ProjectV11
		wantErr bool
	}{
		{
			name: "valid project",
			project: &ProjectV11{
				SchemaVersion: "1.1",
				Tasks: []TaskV11{
					{ID: "t-1", Status: "TODO"},
					{ID: "t-2", Status: "DONE", DependsOn: []string{"t-1"}},
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported schema version",
			project: &ProjectV11{
				SchemaVersion: "1.0",
				Tasks:         []TaskV11{},
			},
			wantErr: true,
		},
		{
			name: "duplicate task id",
			project: &ProjectV11{
				SchemaVersion: "1.1",
				Tasks: []TaskV11{
					{ID: "t-1", Status: "TODO"},
					{ID: "t-1", Status: "DONE"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			project: &ProjectV11{
				SchemaVersion: "1.1",
				Tasks: []TaskV11{
					{ID: "t-1", Status: "INVALID"},
				},
			},
			wantErr: true,
		},
		{
			name: "non-existent dependency",
			project: &ProjectV11{
				SchemaVersion: "1.1",
				Tasks: []TaskV11{
					{ID: "t-1", Status: "TODO", DependsOn: []string{"t-non-existent"}},
				},
			},
			wantErr: true,
		},
		{
			name: "dependency cycle",
			project: &ProjectV11{
				SchemaVersion: "1.1",
				Tasks: []TaskV11{
					{ID: "t-1", Status: "TODO", DependsOn: []string{"t-2"}},
					{ID: "t-2", Status: "TODO", DependsOn: []string{"t-1"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProjectV11(tt.project); (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectV11() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
