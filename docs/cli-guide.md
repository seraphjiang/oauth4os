# oauth4os-demo CLI Guide

Interactive CLI for the oauth4os OAuth 2.0 proxy. Requires `curl` and `jq`.

## Install

```bash
curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
```

## Commands (23)

### Authentication

| Command | Description |
|---------|-------------|
| `login` | Authenticate via client credentials, cache token |
| `logout` | Clear cached token |
| `token` | Show raw cached token |
| `whoami` | Decode JWT payload |
| `profile` | Formatted view: client, scopes, expiry countdown |

```bash
oauth4os-demo login
oauth4os-demo profile
# 🔐 Token Profile
#   Client:  demo-cli
#   Scopes:  read:logs-*
#   TTL:     3542s remaining
```

### Search & Query

| Command | Description |
|---------|-------------|
| `search <kql>` | Search logs with KQL syntax |
| `sql <query>` | Run OpenSearch SQL query |
| `tail [service]` | Live tail, poll every 2s |
| `watch <kql>` | Alert on new KQL matches, poll every 5s |

```bash
oauth4os-demo search 'level:ERROR'
oauth4os-demo search 'service:payment AND latency_ms:>500'
oauth4os-demo search 'service:auth* AND NOT level:INFO'
oauth4os-demo sql 'SELECT service, count(*) FROM logs-demo GROUP BY service'
oauth4os-demo tail payment
oauth4os-demo watch 'level:FATAL'
```

### KQL Syntax

| Pattern | Meaning |
|---------|---------|
| `field:value` | Exact match |
| `field:>N` | Greater than |
| `field:<N` / `>=N` / `<=N` | Range comparisons |
| `field:val*` | Wildcard |
| `A AND B` | Both must match |
| `A OR B` | Either matches |
| `NOT A` | Exclude |

### Analytics & Monitoring

| Command | Description |
|---------|-------------|
| `stats` | Error counts, avg latency, top errors by service |
| `dashboard` | Full-screen TUI dashboard (htop for logs) |
| `top` | Real-time top consumers (requests, clients, scopes) |
| `diff <r1> <r2>` | Compare time ranges (today/yesterday/1h/24h/7d) |
| `latency` | Upstream latency, throughput, error rate |
| `alerts` | Alert status from proxy metrics |
| `audit [n]` | Show last n admin audit events |
| `services` | List indexed services |
| `indices` | List OpenSearch indices |
| `status` | Check proxy health |

```bash
oauth4os-demo stats
oauth4os-demo dashboard        # press q to quit
oauth4os-demo top              # press q to quit
oauth4os-demo diff today yesterday
oauth4os-demo diff 1h 24h
```

### Data Export

| Command | Description |
|---------|-------------|
| `export <kql> -f csv\|json -o <file>` | Export results to file |

```bash
oauth4os-demo export 'level:ERROR' -f csv -o errors.csv
oauth4os-demo export 'service:payment' -f json -o payment.json
```

### History & Bookmarks

| Command | Description |
|---------|-------------|
| `history` | Show last 50 queries |
| `bookmark save <name> <kql>` | Save a query bookmark |
| `bookmark run <name>` | Run a saved bookmark |
| `bookmark delete <name>` | Delete a bookmark |
| `bookmark list` | List all bookmarks |

```bash
oauth4os-demo bookmark save errors 'level:ERROR'
oauth4os-demo bookmark save slow 'latency_ms:>1000'
oauth4os-demo bookmark run errors
```

### Configuration

| Command | Description |
|---------|-------------|
| `config show` | Show current settings |
| `config set <key> <value>` | Set a config value (proxy, index, format) |
| `config get <key>` | Get a config value |
| `config reset` | Reset to defaults |
| `alias add <name> <query>` | Create command alias |
| `alias run <name>` | Run an alias |
| `alias list` | List aliases |
| `env` | Diagnostic dump (paths, deps, connectivity) |

```bash
oauth4os-demo config set proxy https://my-proxy:8443
oauth4os-demo config set index my-logs-*
oauth4os-demo alias add errors 'level:ERROR'
oauth4os-demo alias run errors
oauth4os-demo env
```

### Shell Integration

| Command | Description |
|---------|-------------|
| `completion bash` | Generate bash completions |
| `completion zsh` | Generate zsh completions |
| `install-man` | Install man page to system |

```bash
eval "$(oauth4os-demo completion bash)"   # add to ~/.bashrc
eval "$(oauth4os-demo completion zsh)"    # add to ~/.zshrc
sudo oauth4os-demo install-man
man oauth4os-demo
```

## Pipe Support

When stdout is not a terminal or `--json` is passed, commands output raw JSON:

```bash
oauth4os-demo --json status | jq .version
oauth4os-demo --json latency | jq .latency_ms
oauth4os-demo --json env | jq .proxy_reachable
oauth4os-demo --json profile | jq .scope
oauth4os-demo search 'level:ERROR' | jq '.[] | .service' | sort -u
oauth4os-demo search 'service:payment' | jq length
oauth4os-demo search 'latency_ms:>500' | jq '.[] | .latency_ms' | sort -n
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OAUTH4OS_PROXY` | `https://f5cmk2hxwx.us-west-2.awsapprunner.com` | Proxy URL |
| `OAUTH4OS_INDEX` | `logs-*` | Default search index |
| `OAUTH4OS_FORMAT` | `text` | Output format |
| `WATCH_INTERVAL` | `5` | Seconds between watch polls |

## Files

| Path | Purpose |
|------|---------|
| `~/.oauth4os/config` | Persistent settings |
| `~/.oauth4os/token` | Cached access token |
| `~/.oauth4os/aliases` | Command aliases |
| `~/.oauth4os-history` | Query history (last 50) |
| `~/.oauth4os-bookmarks` | Saved bookmarks |
