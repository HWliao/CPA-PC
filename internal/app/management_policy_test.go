package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagementPolicyFollowsV8ConfigChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	policy := managementPolicyFromFile(path)
	for _, tc := range []struct {
		content string
		key     string
		remote  bool
	}{
		{"remote-management: {secret-key: legacy-key, allow-remote: false}\n", "legacy-key", false},
		{"config-version: 8\nmanagement: {secret-key: next-key, allow-remote: true}\n", "next-key", true},
		{"config-version: 8\nmanagement: {secret-key: \"\", allow-remote: false}\n", "", false},
		{"remote-management: {secret-key: \"\"}\n", "123456", false},
	} {
		if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
			t.Fatal(err)
		}
		key, remote, err := policy()
		if err != nil || key != tc.key || remote != tc.remote {
			t.Fatalf("policy(%q) = %q, %t, %v", tc.content, key, remote, err)
		}
	}
}
