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
if ! command -v go &> /dev/null; then
    echo -e "${RED}Fehler: Go ist nicht installiert!${NC}"
    echo -e "Bitte installiere Go zuerst über Homebrew:"
    echo -e "  ${YELLOW}brew install go${NC}"
    exit 1
else
    GO_VERSION=$(go version)
    echo -e "${GREEN}✔ Go gefunden:${NC} $GO_VERSION"
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

# 4. Binary kompilieren
echo -e "${BLUE}[4/4] Kompiliere postctl...${NC}"
if go build -o postctl .; then
    echo -e "${GREEN}✔ postctl erfolgreich kompiliert.${NC}"
else
    echo -e "${RED}Fehler beim Kompilieren von postctl!${NC}"
    exit 1
fi
echo ""

# 5. Optionaler globaler Installations-Schritt
echo -e "${BLUE}${BOLD}Globale Installation:${NC}"
echo -e "Möchtest du postctl nach ${YELLOW}/usr/local/bin${NC} kopieren, damit du es von überall aus starten kannst?"
read -p "Installieren? (y/n): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Kopiere Binary nach /usr/local/bin... (sudo benötigt)${NC}"
    if sudo cp postctl /usr/local/bin/postctl; then
        echo -e "${GREEN}✔ Erfolgreich global installiert!${NC}"
        echo -e "Du kannst die App ab sofort überall mit dem Befehl ${GREEN}${BOLD}postctl tui${NC} starten."
    else
        echo -e "${RED}Fehler beim Kopieren des Binaries. Du kannst es lokal ausführen:${NC}"
        echo -e "  ${YELLOW}./postctl tui${NC}"
    fi
else
    echo -e "${YELLOW}Lokale Ausführung gewählt.${NC}"
    echo -e "Du kannst die TUI lokal über diesen Befehl starten:"
    echo -e "  ${GREEN}${BOLD}./postctl tui${NC}"
fi

echo ""
echo -e "${GREEN}${BOLD}Setup erfolgreich abgeschlossen! 🎉${NC}"
echo -e "Tipp: Richte deine API-Schlüssel ein mit:"
echo -e "  ${YELLOW}./postctl config set <key> <value>${NC} (oder global: ${YELLOW}postctl config set ...${NC})"
