package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"
)

// CLIConfig is persisted at ~/.oauth4os.yaml
type CLIConfig struct {
	Server       string `yaml:"server"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	DefaultScope string `yaml:"default_scope"`
	Token        string `yaml:"token,omitempty"`
	ExpiresAt    string `yaml:"expires_at,omitempty"`
}

var outputFmt = "table" // global --output flag

func configPath() string {
	if p := os.Getenv("OAUTH4OS_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".oauth4os.yaml")
}

func loadConfig() *CLIConfig {
	cfg := &CLIConfig{Server: "http://localhost:8443", DefaultScope: "admin"}
	data, err := os.ReadFile(configPath())
	if err == nil {
		yaml.Unmarshal(data, cfg)
	}
	if s := os.Getenv("OAUTH4OS_SERVER"); s != "" {
		cfg.Server = s
	}
	if t := os.Getenv("OAUTH4OS_TOKEN"); t != "" {
		cfg.Token = t
	}
	return cfg
}

func (c *CLIConfig) save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func (c *CLIConfig) tokenValid() bool {
	if c.Token == "" || c.ExpiresAt == "" {
		return false
	}
	exp, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Now().Before(exp.Add(-30 * time.Second))
}

func (c *CLIConfig) ensureToken() string {
	if c.tokenValid() {
		return c.Token
	}
	if c.ClientID != "" && c.ClientSecret != "" {
		tok, exp := requestToken(c.Server, c.ClientID, c.ClientSecret, c.DefaultScope)
		if tok != "" {
			c.Token = tok
			c.ExpiresAt = exp
			c.save()
			fmt.Fprintf(os.Stderr, "Token auto-refreshed.\n")
			return tok
		}
	}
	if c.Token != "" {
		return c.Token
	}
	return ""
}

func requestToken(server, clientID, secret, scope string) (string, string) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {secret},
		"scope":         {scope},
	}
	resp, err := http.PostForm(server+"/oauth/token", data)
	if err != nil || resp.StatusCode != 200 {
		return "", ""
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	tok, _ := result["access_token"].(string)
	expiresIn, _ := result["expires_in"].(float64)
	exp := time.Now().Add(time.Duration(expiresIn) * time.Second).Format(time.RFC3339)
	return tok, exp
}

// parseFlags extracts --output flag and returns remaining args.
func parseFlags() []string {
	var remaining []string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if (args[i] == "--output" || args[i] == "-o") && i+1 < len(args) {
			outputFmt = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--output=") {
			outputFmt = strings.TrimPrefix(args[i], "--output=")
		} else if strings.HasPrefix(args[i], "-o=") {
			outputFmt = strings.TrimPrefix(args[i], "-o=")
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return remaining
}

var commands = []string{"login", "logout", "create-token", "revoke-token", "list-tokens", "inspect-token", "status", "config", "completion"}

func main() {
	args := parseFlags()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// Override os.Args for positional arg helpers
	os.Args = append([]string{os.Args[0]}, args...)
	cfg := loadConfig()

	switch args[0] {
	case "login":
		cmdLogin(cfg)
	case "create-token":
		cmdCreateToken(cfg)
	case "revoke-token":
		cmdRevokeToken(cfg)
	case "list-tokens":
		cmdListTokens(cfg)
	case "inspect-token":
		cmdInspectToken(cfg)
	case "status":
		cmdStatus(cfg)
	case "config":
		cmdConfig(cfg)
	case "logout":
		cmdLogout(cfg)
	case "completion":
		cmdCompletion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`oauth4os — CLI for OAuth 2.0 proxy for OpenSearch

Usage: oauth4os [--output json|table|yaml] <command> [args]

Commands:
  login            Authenticate and cache token
  logout           Clear cached credentials
  create-token     Create a scoped access token
  revoke-token     Revoke an access token
  list-tokens      List active tokens
  inspect-token    Show token details
  status           Check proxy health
  config           Show current configuration
  completion       Generate shell completions (bash|zsh|fish)

Flags:
  -o, --output     Output format: table (default), json, yaml

Config: ~/.oauth4os.yaml (or OAUTH4OS_CONFIG env)

Shell completions:
  eval "$(oauth4os completion bash)"
  eval "$(oauth4os completion zsh)"
  oauth4os completion fish | source`)
}

// --- output helpers ---

func output(data interface{}) {
	switch outputFmt {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(data)
	case "yaml":
		enc := yaml.NewEncoder(os.Stdout)
		enc.Encode(data)
	default:
		// table handled per-command
	}
}

func outputKV(pairs [][2]string) {
	switch outputFmt {
	case "json", "yaml":
		m := make(map[string]string)
		for _, p := range pairs {
			m[p[0]] = p[1]
		}
		output(m)
	default:
		for _, p := range pairs {
			fmt.Printf("%-9s%s\n", p[0]+":", p[1])
		}
	}
}

// --- commands ---

func cmdLogin(cfg *CLIConfig) {
	clientID := flagOrPrompt(2, "Client ID")
	clientSecret := flagOrPrompt(3, "Client Secret")
	scope := flagOrDefault(4, cfg.DefaultScope)

	tok, exp := requestToken(cfg.Server, clientID, clientSecret, scope)
	if tok == "" {
		fatal("Login failed — check credentials and server (%s)", cfg.Server)
	}

	cfg.ClientID = clientID
	cfg.ClientSecret = clientSecret
	cfg.DefaultScope = scope
	cfg.Token = tok
	cfg.ExpiresAt = exp
	cfg.save()

	outputKV([][2]string{
		{"Client", clientID},
		{"Token", tok},
		{"Expires", exp},
		{"Config", configPath()},
	})
}

func cmdLogout(cfg *CLIConfig) {
	cfg.Token = ""
	cfg.ExpiresAt = ""
	cfg.ClientSecret = ""
	cfg.save()
	outputKV([][2]string{{"Status", "credentials cleared"}})
}

func cmdCreateToken(cfg *CLIConfig) {
	clientID := flagOrDefault(2, cfg.ClientID)
	scope := flagOrDefault(3, cfg.DefaultScope)
	if clientID == "" {
		clientID = flagOrPrompt(2, "Client ID")
	}

	data := url.Values{
		"grant_type": {"client_credentials"},
		"client_id":  {clientID},
		"scope":      {scope},
	}
	resp, err := http.PostForm(cfg.Server+"/oauth/token", data)
	if err != nil {
		fatal("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if resp.StatusCode != 200 {
		fatal("Token creation failed: %v", result)
	}

	outputKV([][2]string{
		{"Token", fmt.Sprint(result["access_token"])},
		{"Scope", fmt.Sprint(result["scope"])},
		{"Expires", fmt.Sprintf("%.0fs", result["expires_in"].(float64))},
	})
}

func cmdRevokeToken(cfg *CLIConfig) {
	tokenID := flagOrPrompt(2, "Token ID")
	req, _ := http.NewRequest("DELETE", cfg.Server+"/oauth/token/"+tokenID, nil)
	addAuth(req, cfg)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatal("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		outputKV([][2]string{{"Status", "revoked"}, {"Token", tokenID}})
	} else {
		body, _ := io.ReadAll(resp.Body)
		fatal("Revoke failed (%d): %s", resp.StatusCode, body)
	}
}

func cmdListTokens(cfg *CLIConfig) {
	req, _ := http.NewRequest("GET", cfg.Server+"/oauth/tokens", nil)
	addAuth(req, cfg)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatal("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	var tokens []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokens)

	if outputFmt == "json" || outputFmt == "yaml" {
		output(tokens)
		return
	}

	if len(tokens) == 0 {
		fmt.Println("No active tokens.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCLIENT\tSCOPES\tEXPIRES")
	for _, t := range tokens {
		scopes := joinScopes(t["scopes"])
		expires := fmtExpiry(t["expires_at"])
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t["id"], t["client_id"], scopes, expires)
	}
	w.Flush()
}

func cmdInspectToken(cfg *CLIConfig) {
	tokenID := flagOrPrompt(2, "Token ID")
	req, _ := http.NewRequest("GET", cfg.Server+"/oauth/token/"+tokenID, nil)
	addAuth(req, cfg)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatal("Connection failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		fatal("Token not found: %s", tokenID)
	}
	var token map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&token)
	output(token) // always structured output for inspect
}

func cmdStatus(cfg *CLIConfig) {
	resp, err := http.Get(cfg.Server + "/health")
	if err != nil {
		fatal("Proxy unreachable at %s: %v", cfg.Server, err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	tokenState := "not logged in"
	if cfg.tokenValid() {
		tokenState = "cached (expires " + cfg.ExpiresAt + ")"
	} else if cfg.Token != "" {
		tokenState = "expired"
	}

	outputKV([][2]string{
		{"Server", cfg.Server},
		{"Status", fmt.Sprint(result["status"])},
		{"Version", fmt.Sprint(result["version"])},
		{"Token", tokenState},
	})
}

func cmdConfig(cfg *CLIConfig) {
	tokenState := "none"
	if cfg.tokenValid() {
		tokenState = "cached (expires " + cfg.ExpiresAt + ")"
	}
	outputKV([][2]string{
		{"Config", configPath()},
		{"Server", cfg.Server},
		{"Client", cfg.ClientID},
		{"Scope", cfg.DefaultScope},
		{"Token", tokenState},
	})
}

// --- shell completions ---

func cmdCompletion() {
	shell := flagOrDefault(2, "bash")
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fatal("Unsupported shell: %s (use bash, zsh, or fish)", shell)
	}
}

var bashCompletion = `_oauth4os() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local cmds="login logout create-token revoke-token list-tokens inspect-token status config completion help"
    local flags="--output -o --help -h"
    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$cmds" -- "$cur"))
    elif [[ "${COMP_WORDS[COMP_CWORD-1]}" == "--output" || "${COMP_WORDS[COMP_CWORD-1]}" == "-o" ]]; then
        COMPREPLY=($(compgen -W "table json yaml" -- "$cur"))
    elif [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W "$flags" -- "$cur"))
    elif [[ "${COMP_WORDS[1]}" == "completion" ]]; then
        COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
    fi
}
complete -F _oauth4os oauth4os
`

var zshCompletion = `#compdef oauth4os
_oauth4os() {
    local -a commands=(
        'login:Authenticate and cache token'
        'logout:Clear cached credentials'
        'create-token:Create a scoped access token'
        'revoke-token:Revoke an access token'
        'list-tokens:List active tokens'
        'inspect-token:Show token details'
        'status:Check proxy health'
        'config:Show current configuration'
        'completion:Generate shell completions'
        'help:Show help'
    )
    _arguments \
        '(-o --output)'{-o,--output}'[Output format]:format:(table json yaml)' \
        '1:command:->cmds' \
        '*::arg:->args'
    case $state in
        cmds) _describe 'command' commands ;;
        args)
            case $words[1] in
                completion) _values 'shell' bash zsh fish ;;
            esac ;;
    esac
}
_oauth4os
`

var fishCompletion = `complete -c oauth4os -f
complete -c oauth4os -n '__fish_use_subcommand' -a login -d 'Authenticate and cache token'
complete -c oauth4os -n '__fish_use_subcommand' -a logout -d 'Clear cached credentials'
complete -c oauth4os -n '__fish_use_subcommand' -a create-token -d 'Create a scoped access token'
complete -c oauth4os -n '__fish_use_subcommand' -a revoke-token -d 'Revoke an access token'
complete -c oauth4os -n '__fish_use_subcommand' -a list-tokens -d 'List active tokens'
complete -c oauth4os -n '__fish_use_subcommand' -a inspect-token -d 'Show token details'
complete -c oauth4os -n '__fish_use_subcommand' -a status -d 'Check proxy health'
complete -c oauth4os -n '__fish_use_subcommand' -a config -d 'Show current configuration'
complete -c oauth4os -n '__fish_use_subcommand' -a completion -d 'Generate shell completions'
complete -c oauth4os -n '__fish_use_subcommand' -a help -d 'Show help'
complete -c oauth4os -l output -s o -d 'Output format' -ra 'table json yaml'
complete -c oauth4os -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`

// --- helpers ---

func addAuth(req *http.Request, cfg *CLIConfig) {
	if tok := cfg.ensureToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

func joinScopes(v interface{}) string {
	if s, ok := v.([]interface{}); ok {
		parts := make([]string, len(s))
		for i, x := range s {
			parts[i] = fmt.Sprint(x)
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func fmtExpiry(v interface{}) string {
	if e, ok := v.(string); ok {
		if parsed, err := time.Parse(time.RFC3339, e); err == nil {
			return parsed.Format("2006-01-02 15:04")
		}
	}
	return ""
}

func flagOrPrompt(idx int, label string) string {
	if idx < len(os.Args) {
		return os.Args[idx]
	}
	fmt.Printf("%s: ", label)
	var val string
	fmt.Scanln(&val)
	return val
}

func flagOrDefault(idx int, def string) string {
	if idx < len(os.Args) {
		return os.Args[idx]
	}
	return def
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
