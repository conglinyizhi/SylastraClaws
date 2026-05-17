package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conglinyizhi/SylastraClaws/pkg/config"
)

func TestGetConfigPath(t *testing.T) {
	origHome := os.Getenv(config.EnvHome)
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		os.Setenv(config.EnvHome, origHome)
		os.Setenv("XDG_CONFIG_HOME", origXDG)
	})
	os.Unsetenv(config.EnvHome)
	os.Unsetenv("XDG_CONFIG_HOME")

	// On Go 1.26+, os.UserHomeDir is cached, so t.Setenv("HOME") won't apply.
	// Instead, set SYLASTRACLAWS_HOME to test the config path resolution.
	t.Setenv(config.EnvHome, "/tmp/sylastraclaws-home")

	got := GetConfigPath()
	want := filepath.Join("/tmp/sylastraclaws-home", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithSYLASTRACLAWS_HOME(t *testing.T) {
	t.Setenv(config.EnvHome, "/custom/sylastraclaws")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/custom/sylastraclaws", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithSYLASTRACLAWS_CONFIG(t *testing.T) {
	t.Setenv("SYLASTRACLAWS_CONFIG", "/custom/config.json")
	t.Setenv(config.EnvHome, "/custom/sylastraclaws")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := "/custom/config.json"

	assert.Equal(t, want, got)
}

func TestGetConfigPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific HOME behavior varies; run on windows")
	}

	testUserProfilePath := `C:\Users\Test`
	t.Setenv("USERPROFILE", testUserProfilePath)

	got := GetConfigPath()
	want := filepath.Join(testUserProfilePath, ".picoclaw", "config.json")

	require.True(t, strings.EqualFold(got, want), "GetConfigPath() = %q, want %q", got, want)
}
