// all environment variables including default values put here

package pkg

const (
	Logo = "🦞"
	// AppName is the name of the app
	AppName = "SylastraClaws"

	// DefaultConfigDir is the default config directory name under
	// $XDG_CONFIG_HOME (fallback ~/.config).
	DefaultConfigDir = "sylastraclaws"

	// DeprecatedPicoClawHome is the legacy config directory name.
	// Used only as a fallback when neither XDG nor SYLASTRACLAWS_HOME is set.
	DeprecatedPicoClawHome = ".picoclaw"

	WorkspaceName = "workspace"
)
