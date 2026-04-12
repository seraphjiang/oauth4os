#!/bin/bash
# oauth4os Hive Watchdog — runs every 5 min via cron
# Pokes idle agents to pick up backlog work

export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
SESSION="o11y"
LOG="/tmp/oauth4os-watchdog.log"

echo "[watchdog] $(date '+%Y-%m-%d %H:%M:%S') running" >> "$LOG"

STATUS=$(kiro-hive status 2>/dev/null)
if [ $? -ne 0 ]; then
    echo "[watchdog] hive not available, skipping" >> "$LOG"
    exit 0
fi

# Check each agent — if idle, poke them
for agent in sde devops index-eng auth-eng query-eng frontend ai-lead test; do
    LINE=$(echo "$STATUS" | grep "  $agent:")
    if echo "$LINE" | grep -q "status=idle"; then
        echo "[watchdog] $agent is idle, poking" >> "$LOG"
        kiro-hive tell "$agent" "[WATCHDOG] You've been idle. Pick up work from the backlog. If your assigned tasks are done, grab from: OpenTelemetry export, webhook notifications, admin UI, Docker Compose with Prometheus+Grafana, Helm chart integration test, i18n consent screen, rate limit by API key, canary deployment. Or: find bugs, write tests, improve docs, optimize code. Report what you picked up." >> "$LOG" 2>&1
    else
        echo "[watchdog] $agent is active" >> "$LOG"
    fi
done

echo "[watchdog] cycle complete" >> "$LOG"
