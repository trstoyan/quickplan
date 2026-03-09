package main

import (
	"strings"
	"testing"
)

func TestRemoteCommandSurfaceAvoidsInternalProductNames(t *testing.T) {
	for name, value := range map[string]string{
		"sync short":    syncCmd.Short,
		"push short":    pushCmd.Short,
		"pull short":    pullCmd.Short,
		"verify short":  verifyCmd.Short,
		"monitor flag":  monitorCmd.Flag("url").Usage,
		"doctor short":  doctorCmd.Short,
	} {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "quickplan.sh") || strings.Contains(lower, "quickplan-web") {
			t.Fatalf("%s leaks internal naming: %q", name, value)
		}
	}
}
