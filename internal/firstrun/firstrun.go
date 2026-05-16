// Package firstrun implements the --first-run flag for interactive setup.
// Users pass comma-separated values in any order:
//
//	sylastraclaws --first-run "sk-ant-xxx,claude-sonnet-4-20250514,https://api.anthropic.com"
//	sylastraclaws --first-run "sk-xxx,gpt-4o"
//	sylastraclaws --first-run "https://my-proxy.com/v1,sk-xxx,deepseek-chat"
package firstrun

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conglinyizhi/SylastraClaws/internal"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
	"github.com/conglinyizhi/SylastraClaws/pkg/providers"
)

// ErrNotFirstRun is returned when the flag value is empty, so callers can skip silently.
var ErrNotFirstRun = errors.New("not first run")

// Run parses the --first-run value, detects inputs, probes the endpoint,
// writes the config file, and exits with 0 on success.
func Run(rawValue string) error {
	if rawValue == "" {
		return ErrNotFirstRun
	}

	fields := splitAndClean(rawValue)
	if len(fields) < 2 {
		fmt.Println("Usage: sylastraclaws --first-run \"<api_key>,<model_name>[,<base_url>]\"")
		fmt.Println("  Fields can be in any order. At least api_key and model_name are required.")
		return nil
	}

	detected := detectFields(fields)
	if detected.APIKey == "" {
		fmt.Println("Error: could not identify an API key in the input.")
		fmt.Println("  API keys typically start with 'sk-', 'sk-ant-', or are long random strings.")
		return nil
	}
	if detected.ModelName == "" {
		fmt.Println("Error: could not identify a model name in the input.")
		fmt.Println("  Common examples: gpt-4o, claude-sonnet-4-20250514, deepseek-chat, gemini-2.5-flash")
		return nil
	}

	// Resolve protocol from model name + base URL
	protocol := resolveProtocol(detected.ModelName, detected.BaseURL)
	defaultBase := providers.DefaultAPIBaseForProtocol(protocol)
	baseURL := detected.BaseURL
	if baseURL == "" {
		baseURL = defaultBase
	}

	// Build a temporary ModelConfig for probing
	mc := &config.ModelConfig{
		ModelName: detected.ModelName,
		Model:     detected.ModelName,
		Provider:  protocol,
		APIBase:   baseURL,
	}
	mc.SetAPIKey(detected.APIKey)

	// Display what we're about to do
	fmt.Println("Detected inputs:")
	fmt.Printf("  API Key:    %s\n", truncateEnd(detected.APIKey, 6, 4))
	fmt.Printf("  Model:      %s\n", detected.ModelName)
	fmt.Printf("  Provider:   %s\n", protocol)
	fmt.Printf("  Base URL:   %s\n", baseURL)
	fmt.Println()
	fmt.Println("Testing connection...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	provider, modelID, err := providers.CreateProviderFromConfig(mc)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		fmt.Println()
		fmt.Println("If you're using a custom endpoint, try:")
		fmt.Printf("  sylastraclaws --first-run \"<api_key>,<model_name>,<base_url>\"\n")
		return nil
	}

	resp, err := provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: "Reply with exactly one word: ok"},
	}, nil, modelID, nil)
	if err != nil {
		fmt.Printf("Connection test failed: %v\n", err)
		fmt.Println()
		suggestions := []string{}
		if baseURL != "" {
			suggestions = append(suggestions,
				fmt.Sprintf("Check that the base URL is correct: %s", baseURL),
				"Some providers use /v1, /v1beta, or no version suffix",
			)
		}
		suggestions = append(suggestions, "Check that the API key is valid and has credits")
		fmt.Println("Suggestions:")
		for _, s := range suggestions {
			fmt.Printf("  - %s\n", s)
		}
		return nil
	}

	_ = resp

	fmt.Println("✓ Connection successful! Writing configuration...")

	// Build the full config
	cfg := config.DefaultConfig()
	if cfg.ModelList == nil {
		cfg.ModelList = config.SecureModelList{}
	}
	modelEntry := &config.ModelConfig{
		ModelName: detected.ModelName,
		Provider:  protocol,
		Model:     detected.ModelName,
		APIBase:   baseURL,
		Enabled:   true,
	}
	modelEntry.SetAPIKey(detected.APIKey)
	cfg.ModelList = append(cfg.ModelList, modelEntry)

	// Set defaults to use this model
	cfg.Agents.Defaults.ModelName = detected.ModelName
	cfg.Agents.Defaults.Provider = protocol

	// Write config
	configPath := internal.GetConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		return nil
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return nil
	}

	fmt.Println()
	fmt.Println("✓ Configuration complete!")
	fmt.Println()
	fmt.Printf("  Config written to: %s\n", configPath)
	fmt.Println()
	fmt.Println("Next step:")
	fmt.Println("  Start sylastraclaws without --first-run to begin chatting:")
	fmt.Println("    sylastraclaws agent")
	fmt.Println()
	fmt.Println("  Or if you use the TUI:")
	fmt.Println("    sylastraclaws")

	return nil
}

// detection represents the three possible field classes.
type detection struct {
	APIKey    string
	ModelName string
	BaseURL   string
}

// detectFields classifies each input field by its content.
func detectFields(fields []string) detection {
	var d detection

	for _, f := range fields {
		lower := strings.ToLower(f)

		switch {
		case isURL(f):
			d.BaseURL = f

		case isAPIKey(f, lower):
			if d.APIKey == "" {
				d.APIKey = f
			}

		case isModelName(f, lower):
			if d.ModelName == "" {
				d.ModelName = f
			}

		default:
			// Ambiguous — fallback heuristics
			if len(f) > 40 && d.APIKey == "" {
				d.APIKey = f
			} else if strings.ContainsAny(f, ".-") && d.ModelName == "" {
				d.ModelName = f
			} else if strings.HasPrefix(lower, "http") && d.BaseURL == "" {
				d.BaseURL = f
			} else if d.APIKey == "" {
				d.APIKey = f
			} else if d.ModelName == "" {
				d.ModelName = f
			}
		}
	}

	return d
}

// isURL checks if a string looks like a URL.
func isURL(s string) bool {
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	return false
}

// isAPIKey checks if a string looks like an API key.
func isAPIKey(s, lower string) bool {
	prefixes := []string{
		"sk-", "sk-ant-", "sk-or-", "fk", "gsk_", "pplx-",
		"xai-", "nvapi-", "wb_", "pat-", "ghp_",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}

	// Heuristic: long random-looking strings
	if len(s) >= 32 && !strings.ContainsAny(s, " \t\n") &&
		!strings.HasPrefix(lower, "http") &&
		!strings.ContainsAny(s, "/?&=#") {
		hasUpper := false
		hasLower := false
		hasDigit := false
		for _, c := range s {
			switch {
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= '0' && c <= '9':
				hasDigit = true
			}
		}
		if hasLower && (hasDigit || hasUpper) {
			return true
		}
	}

	return false
}

// isModelName checks if a string looks like a model name.
func isModelName(s, lower string) bool {
	modelSignals := []string{
		"gpt", "claude", "gemini", "deepseek", "qwen", "glm",
		"llama", "mistral", "mixtral", "phi", "command", "dbrx",
		"yi-", "step", "minimax", "spark", "moonshot", "ernie",
		"o1-", "o3-", "nova", "solar", "aya", "jamba",
	}

	for _, signal := range modelSignals {
		if strings.Contains(lower, signal) {
			return true
		}
	}

	// "provider/model"
	if strings.Contains(s, "/") && !strings.HasPrefix(lower, "http") {
		return true
	}

	// Has hyphens but no "://" + reasonable length
	if !strings.Contains(s, "://") && len(s) > 3 && len(s) < 80 && strings.Count(s, "-") >= 1 {
		return true
	}

	return false
}

// resolveProtocol determines the provider protocol from model name and base URL.
func resolveProtocol(modelName, baseURL string) string {
	name := strings.ToLower(modelName)

	// Anthropic
	if strings.HasPrefix(name, "claude") || strings.Contains(name, "anthropic") {
		if baseURL != "" && !strings.Contains(strings.ToLower(baseURL), "anthropic.com") {
			return "openai" // custom base → OpenAI-compatible
		}
		return "anthropic"
	}

	// Gemini
	if strings.HasPrefix(name, "gemini") || strings.HasPrefix(modelName, "models/") {
		return "gemini"
	}

	// DeepSeek
	if strings.Contains(name, "deepseek") {
		return "deepseek"
	}

	// Known providers by model prefix
	knownProviders := map[string]string{
		"qwen":     "qwen-portal",
		"glm-":     "zhipu",
		"ernie":    "zhipu",
		"moonshot": "moonshot",
		"minimax":  "minimax",
		"nova":     "novita",
	}
	for prefix, p := range knownProviders {
		if strings.HasPrefix(name, prefix) {
			return p
		}
	}

	// Try base URL matching
	if baseURL != "" {
		base := strings.ToLower(baseURL)
		pairs := []struct {
			sub     string
			proto string
		}{
			{"openai.com", "openai"},
			{"deepseek.com", "deepseek"},
			{"anthropic.com", "anthropic"},
			{"generativelanguage.googleapis.com", "gemini"},
			{"dashscope.aliyuncs.com", "qwen-portal"},
			{"dashscope-intl.aliyuncs.com", "qwen-intl"},
			{"bigmodel.cn", "zhipu"},
			{"ark.cn-beijing.volces.com", "volcengine"},
			{"moonshot.cn", "moonshot"},
			{"minimaxi.com", "minimax"},
			{"novita.ai", "novita"},
			{"modelscope.cn", "modelscope"},
			{"groq.com", "groq"},
			{"openrouter.ai", "openrouter"},
			{"cerebras", "cerebras"},
			{"mistral", "mistral"},
			{"ollama", "ollama"},
		}
		for _, pair := range pairs {
			if strings.Contains(base, pair.sub) {
				return pair.proto
			}
		}
		// Localhost → OpenAI-compatible
		if strings.Contains(base, "localhost") || strings.Contains(base, "127.0.0.1") {
			return "openai"
		}
	}

	return "openai"
}

// splitAndClean splits a comma-separated string and trims whitespace.
// Handles both ASCII and fullwidth commas.
func splitAndClean(raw string) []string {
	if raw == "" {
		return nil
	}
	s := strings.ReplaceAll(raw, "，", ",")     // Chinese comma
	s = strings.ReplaceAll(s, "\uff0c", ",")    // Fullwidth comma (U+FF0C)
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// truncateEnd returns s with the middle replaced by "…" for display.
func truncateEnd(s string, leading, trailing int) string {
	if len(s) <= leading+trailing+3 {
		return s
	}
	if leading > len(s) {
		leading = len(s)
	}
	if trailing > len(s) {
		trailing = 0
	}
	return s[:leading] + "…" + s[len(s)-trailing:]
}
