// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"os"
	"path/filepath"

	"github.com/conglinyizhi/SylastraClaws/pkg"
)

// Runtime environment variable keys for the SylastraClaws process.
// These control the location of files and binaries at runtime and are read
// directly via os.Getenv / os.LookupEnv. All SylastraClaws-specific keys use
// the SYLASTRACLAWS_ prefix. Reference these constants instead of inline
// string literals to keep all supported knobs visible in one place and to
// prevent typos.
const (
	// EnvHome overrides the base directory for all SylastraClaws data
	// (config, workspace, skills, auth store, …).
	// Default: $XDG_CONFIG_HOME/sylastraclaws (fallback ~/.config/sylastraclaws).
	// Legacy fallback: ~/.picoclaw if none of the above exist.
	EnvHome = "SYLASTRACLAWS_HOME"

	// EnvConfig overrides the full path to the JSON config file.
	// Default: $SYLASTRACLAWS_HOME/config.json
	EnvConfig = "SYLASTRACLAWS_CONFIG"

	// EnvBuiltinSkills overrides the directory from which built-in
	// skills are loaded.
	// Default: <cwd>/skills
	EnvBuiltinSkills = "SYLASTRACLAWS_BUILTIN_SKILLS"

	// EnvBinary overrides the path to the SylastraClaws executable.
	// Used by the web launcher when spawning the gateway subprocess.
	// Default: resolved from the same directory as the current executable.
	EnvBinary = "SYLASTRACLAWS_BINARY"

	// EnvGatewayHost overrides the host address for the gateway server.
	// Default: "localhost"
	EnvGatewayHost = "SYLASTRACLAWS_GATEWAY_HOST"
)

// GetHome returns the base directory for all SylastraClaws data.
// Resolution order:
//  1. $SYLASTRACLAWS_HOME (explicit override)
//  2. $XDG_CONFIG_HOME/sylastraclaws  (XDG spec)
//  3. ~/.config/sylastraclaws         (XDG fallback)
//  4. .                               (last resort)
func GetHome() string {
	// 1. Explicit override
	if h := os.Getenv(EnvHome); h != "" {
		return h
	}

	// 2. XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, pkg.DefaultConfigDir)
	}

	// 3. XDG fallback: ~/.config/sylastraclaws
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config", pkg.DefaultConfigDir)
	}

	return "."
}
