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
	// Cached token (written by login, read by all commands)
	Token     string `yaml:"token,omitempty"`
	ExpiresAt string `yaml:"expires_at,omitempty"`
}

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
	// Env vars override config file
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
	return time.Now().Before(exp.Add(-30 * time.Second)) // 30s buffer
}

func (c *CLIConfig) ensureToken() string {
	if c.tokenValid() {
		return c.Token
	}
	// Auto-refresh if credentials are saved
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
		return c.Token // expired but try anyway
	}
	return ""
}

func requestToken(server, clientID, secret, scope string) (token, expiresAt string) {
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

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := loadConfig()

	switch os.Args[1] {
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
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`oauth4os — CLI for OAuth 2.0 proxy for OpenSearch

Usage: oauth4os <command> [options]

Commands:
  login            Authenticate and cache token
  logout           Clear cached credentials
  create-token     Create a scoped access token
  revoke-token     Revoke an access token
  list-tokens      List active tokens
  inspect-token    Show token details
  status           Check proxy health
  config           Show current configuration

Config file: ~/.oauth4os.yaml (or OAUTH4OS_CONFIG env)

Environment:
  OAUTH4OS_SERVER  Proxy URL (overrides config)
  OAUTH4OS_TOKEN   Bearer token (overrides cached token)
  OAUTH4OS_CONFIG  Config file path`)
}

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
	if err := cfg.save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", err)
	}

	fmt.Printf("Authenticated as %s\n", clientID)
	fmt.Printf("Token cached at %s\n", configPath())
	fmt.Printf("Expires: %s\n", exp)
}

func cmdLogout(cfg *CLIConfig) {
	cfg.Token = ""
	cfg.ExpiresAt = ""
	cfg.ClientSecret = ""
	cfg.save()
	fmt.Println("Credentials cleared.")
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

	fmt.Printf("Token:      %s\n", result["access_token"])
	fmt.Printf("Scope:      %s\n", result["scope"])
	fmt.Printf("Expires in: %.0fs\n", result["expires_in"].(float64))
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
		fmt.Println("Token revoked.")
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

	if len(tokens) == 0 {
		fmt.Println("No active tokens.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCLIENT\tSCOPES\tEXPIRES")
	for _, t := range tokens {
		scopes := ""
		if s, ok := t["scopes"].([]interface{}); ok {
			parts := make([]string, len(s))
			for i, v := range s {
				parts[i] = fmt.Sprint(v)
			}
			scopes = strings.Join(parts, " ")
		}
		expires := ""
		if e, ok := t["expires_at"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, e); err == nil {
				expires = parsed.Format("2006-01-02 15:04")
			}
		}
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(token)
}

func cmdStatus(cfg *CLIConfig) {
	resp, err := http.Get(cfg.Server + "/health")
	if err != nil {
		fatal("Proxy unreachable at %s: %v", cfg.Server, err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("Server:  %s\n", cfg.Server)
	fmt.Printf("Status:  %s\n", result["status"])
	fmt.Printf("Version: %s\n", result["version"])
	if cfg.tokenValid() {
		fmt.Printf("Token:   cached (expires %s)\n", cfg.ExpiresAt)
	} else if cfg.Token != "" {
		fmt.Printf("Token:   expired\n")
	} else {
		fmt.Printf("Token:   not logged in\n")
	}
}

func cmdConfig(cfg *CLIConfig) {
	fmt.Printf("Config:  %s\n", configPath())
	fmt.Printf("Server:  %s\n", cfg.Server)
	fmt.Printf("Client:  %s\n", cfg.ClientID)
	fmt.Printf("Scope:   %s\n", cfg.DefaultScope)
	if cfg.tokenValid() {
		fmt.Printf("Token:   cached (expires %s)\n", cfg.ExpiresAt)
	} else {
		fmt.Printf("Token:   none\n")
	}
}

// --- helpers ---

func addAuth(req *http.Request, cfg *CLIConfig) {
	tok := cfg.ensureToken()
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
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
