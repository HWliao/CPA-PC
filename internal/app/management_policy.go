package app

import (
	"os"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	"gopkg.in/yaml.v3"
)

func managementPolicyFromFile(configPath string) func() (string, bool, error) {
	return func() (string, bool, error) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return "", false, err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return "", false, err
		}
		if len(doc.Content) == 0 {
			return pcconfig.DefaultManagementKey, false, nil
		}
		root := doc.Content[0]
		management := mappingValue(root, "management")
		legacy := mappingValue(root, "remote-management")
		key := mappingValue(management, "secret-key")
		v8Key := key != nil
		if key == nil {
			key = mappingValue(legacy, "secret-key")
		}
		secret := pcconfig.DefaultManagementKey
		if key != nil {
			if err := key.Decode(&secret); err != nil {
				return "", false, err
			}
			if secret == "" && !v8Key {
				secret = pcconfig.DefaultManagementKey
			}
		}
		remoteNode := mappingValue(management, "allow-remote")
		if remoteNode == nil {
			remoteNode = mappingValue(legacy, "allow-remote")
		}
		var remote bool
		if remoteNode != nil {
			if err := remoteNode.Decode(&remote); err != nil {
				return "", false, err
			}
		}
		return secret, remote, nil
	}
}
