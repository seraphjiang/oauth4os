package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

const defaultServer = "http://localhost:8443"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	server := os.Getenv("OAUTH4OS_SERVER")
	if server == "" {
		server = defaultServer
	}

	switch os.Args[1] {
	case "login":
		cmdLogin(server)
	case "create-token":
		cmdCreateToken(server)
	case "revoke-token":
		cmdRevokeToken(server)
	case "list-tokens":
		cmdListTokens(server)
	case "inspect-token":
		cmdInspectToken(server)
	case "status":
		cmdStatus(server)
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
  login            Authenticate and save credentials
  create-token     Create a scoped access token
  revoke-token     Revoke an access token
  list-tokens      List active tokens
  inspect-token    Show token details
  status           Check proxy health

Environment:
  OAUTH4OS_SERVER  Proxy URL (default: http://localhost:8443)
  OAUTH4OS_TOKEN   Bearer token for authenticated commands`)
}

func cmdLogin(server string) {
	clientID := flagOrPrompt(2, "Client ID")
	clientSecret := flagOrPrompt(3, "Client Secret")
	scope := flagOrDefault(4, "admin")

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {scope},
	}

	resp, err := http.PostForm(server+"/oauth/token", data)
	if err != nil {
		fatal("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		fatal("Login failed: %v", result)
	}

	token := result["access_token"].(string)
	fmt.Printf("Authenticated. Token: %s\n", token)
	fmt.Printf("Expires in: %.0fs\n", result["expires_in"].(float64))
	fmt.Printf("\nExport to use:\n  export OAUTH4OS_TOKEN=%s\n", token)
}

func cmdCreateToken(server string) {
	clientID := flagOrPrompt(2, "Client ID")
	scope := flagOrDefault(3, "read:*")

	data := url.Values{
		"grant_type": {"client_credentials"},
		"client_id":  {clientID},
		"scope":      {scope},
	}

	resp, err := http.PostForm(server+"/oauth/token", data)
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

func cmdRevokeToken(server string) {
	tokenID := flagOrPrompt(2, "Token ID")

	req, _ := http.NewRequest("DELETE", server+"/oauth/token/"+tokenID, nil)
	addAuth(req)
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

func cmdListTokens(server string) {
	req, _ := http.NewRequest("GET", server+"/oauth/tokens", nil)
	addAuth(req)
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

func cmdInspectToken(server string) {
	tokenID := flagOrPrompt(2, "Token ID")

	req, _ := http.NewRequest("GET", server+"/oauth/token/"+tokenID, nil)
	addAuth(req)
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

func cmdStatus(server string) {
	resp, err := http.Get(server + "/health")
	if err != nil {
		fatal("Proxy unreachable: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("Server:  %s\n", server)
	fmt.Printf("Status:  %s\n", result["status"])
	fmt.Printf("Version: %s\n", result["version"])
}

// --- helpers ---

func addAuth(req *http.Request) {
	if tok := os.Getenv("OAUTH4OS_TOKEN"); tok != "" {
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
