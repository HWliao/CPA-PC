package app

import "runtime/debug"

const (
	cliProxyAPIModulePath      = "github.com/router-for-me/CLIProxyAPI/v7"
	fallbackCLIProxyAPIVersion = "v7.1.19"
)

func resolveCLIProxyAPIVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackCLIProxyAPIVersion
	}

	for _, dep := range info.Deps {
		if dep.Path != cliProxyAPIModulePath {
			continue
		}
		version := dep.Version
		if dep.Replace != nil && dep.Replace.Version != "" {
			version = dep.Replace.Version
		}
		if version != "" && version != "(devel)" {
			return version
		}
	}

	return fallbackCLIProxyAPIVersion
}
