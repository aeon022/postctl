#!/bin/bash
# postctl Setup Utility

# Terminal-Farben definieren
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # Keine Farbe

echo -e "${BLUE}${BOLD}=========================================${NC}"
echo -e "${BLUE}${BOLD}        postctl Setup & Installation      ${NC}"
echo -e "${BLUE}${BOLD}=========================================${NC}"
echo ""

# 1. Go Installation prüfen
echo -e "${BLUE}[1/4] Prüfe Go-Installation...${NC}"
# GOTOOLCHAIN=local verhindert, dass go bei einer zu alten lokalen Version
# still versucht, die in go.mod geforderte Toolchain aus dem Netz nachzuladen —
# genau das führte auf WSL/Ubuntu zu einem endlosen, meldungslosen Hänger direkt
# an dieser Stelle (github.com/aeon022/postctl/issues/1), sobald Proxy/Firewall
# den Download blockierten. Mit "local" schlägt eine zu alte Version stattdessen
# sofort mit einer klaren Fehlermeldung fehl.
export GOTOOLCHAIN=local
if ! command -v go &> /dev/null; then
    echo -e "${RED}Fehler: Go ist nicht installiert!${NC}"
    echo -e "Bitte installiere Go zuerst über Homebrew:"
    echo -e "  ${YELLOW}brew install go${NC}"
    exit 1
else
    INSTALLED_GO=$(go env GOVERSION | sed 's/^go//')
    REQUIRED_GO=$(grep -m1 '^go ' go.mod | awk '{print $2}')
    NEWEST=$(printf '%s\n%s\n' "$INSTALLED_GO" "$REQUIRED_GO" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)
    if [ "$NEWEST" != "$INSTALLED_GO" ]; then
        echo -e "${RED}Fehler: Installierte Go-Version (go$INSTALLED_GO) ist älter als benötigt (go$REQUIRED_GO).${NC}"
        echo -e "Bitte Go aktualisieren, z.B. über Homebrew:"
        echo -e "  ${YELLOW}brew upgrade go${NC}"
        exit 1
    fi
    echo -e "${GREEN}✔ Go gefunden:${NC} go$INSTALLED_GO"
fi
echo ""

# 2. Abhängigkeiten herunterladen
echo -e "${BLUE}[2/4] Lade Go-Abhängigkeiten herunter...${NC}"
if go mod download; then
    echo -e "${GREEN}✔ Abhängigkeiten erfolgreich geladen.${NC}"
else
    echo -e "${RED}Fehler beim Herunterladen der Abhängigkeiten!${NC}"
    exit 1
fi
echo ""

# 3. Konfigurationsverzeichnis erstellen
echo -e "${BLUE}[3/4] Bereite Konfiguration vor...${NC}"
CONFIG_DIR="$HOME/.config/postctl"
if [ ! -d "$CONFIG_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    echo -e "${GREEN}✔ Verzeichnis erstellt:${NC} $CONFIG_DIR"
else
    echo -e "${GREEN}✔ Konfigurationsverzeichnis existiert bereits.${NC}"
fi
echo ""

# 4. Binary kompilieren & installieren
echo -e "${BLUE}[4/4] Kompiliere und installiere postctl...${NC}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$INSTALL_DIR"
if go build -o "$INSTALL_DIR/postctl" .; then
    echo -e "${GREEN}✔ postctl erfolgreich kompiliert und installiert in:${NC} $INSTALL_DIR/postctl"
    echo -e "Du kannst die App ab sofort überall mit dem Befehl ${GREEN}${BOLD}postctl tui${NC} starten."
else
    echo -e "${RED}Fehler beim Kompilieren von postctl!${NC}"
    exit 1
fi
echo ""

# ── MCP: register in ~/.claude.json ───────────────────────────────────────────
CLAUDE_JSON="$HOME/.claude.json"
if command -v python3 &>/dev/null; then
  python3 - "$CLAUDE_JSON" "$INSTALL_DIR/postctl" <<'PYEOF'
import json, sys, os

claude_json = sys.argv[1]
binary_path = sys.argv[2]

data = {}
if os.path.exists(claude_json):
    with open(claude_json) as f:
        try:
            data = json.load(f)
        except Exception:
            pass

data.setdefault("mcpServers", {})
data["mcpServers"]["postctl"] = {
    "command": binary_path,
    "args": ["mcp"]
}

with open(claude_json, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print("✓ MCP server registered in ~/.claude.json")
print("  Restart Claude Code to activate postctl MCP tools")
PYEOF
else
  echo "  To enable MCP (Claude Code integration), add to ~/.claude.json:"
  echo "  \"mcpServers\": { \"postctl\": { \"command\": \"$INSTALL_DIR/postctl\", \"args\": [\"mcp\"] } }"
fi

echo ""
echo -e "${GREEN}${BOLD}Setup erfolgreich abgeschlossen! 🎉${NC}"
echo -e "Tipp: Richte deine API-Schlüssel ein mit:"
echo -e "  ${YELLOW}postctl config set <key> <value>${NC}"
