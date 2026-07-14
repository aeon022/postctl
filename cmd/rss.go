package cmd

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var rssCmd = &cobra.Command{
	Use:   "rss",
	Short: "Manage and import posts from RSS feeds",
	Long:  `Configure RSS feeds and import new articles as drafts automatically.`,
}

var rssAddCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Add a new RSS feed URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := strings.TrimSpace(args[0])
		if url == "" {
			cmd.Println("❌ Feed URL darf nicht leer sein.")
			return
		}

		// Check if already exists
		for _, f := range config.ActiveConfig.RSSFeeds {
			if f == url {
				cmd.Printf("➖ Feed %s ist bereits registriert.\n", url)
				return
			}
		}

		config.ActiveConfig.RSSFeeds = append(config.ActiveConfig.RSSFeeds, url)
		if err := config.SaveConfig(); err != nil {
			cmd.Printf("❌ Fehler beim Speichern der Konfiguration: %v\n", err)
			return
		}

		cmd.Printf("✅ Feed %s erfolgreich hinzugefügt.\n", url)
	},
}

var rssListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured RSS feeds",
	Run: func(cmd *cobra.Command, args []string) {
		feeds := config.ActiveConfig.RSSFeeds
		if len(feeds) == 0 {
			cmd.Println("Keine RSS-Feeds konfiguriert. Füge einen hinzu mit: postctl rss add <url>")
			return
		}

		cmd.Println("Registrierte RSS-Feeds:")
		for i, f := range feeds {
			cmd.Printf("  %d) %s\n", i+1, f)
		}
	},
}

var rssRemoveCmd = &cobra.Command{
	Use:   "remove [url]",
	Short: "Remove a configured RSS feed",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := strings.TrimSpace(args[0])
		feeds := config.ActiveConfig.RSSFeeds
		newFeeds := []string{}
		found := false

		for _, f := range feeds {
			if f == url {
				found = true
				continue
			}
			newFeeds = append(newFeeds, f)
		}

		if !found {
			cmd.Printf("❌ Feed %s wurde nicht gefunden.\n", url)
			return
		}

		config.ActiveConfig.RSSFeeds = newFeeds
		if err := config.SaveConfig(); err != nil {
			cmd.Printf("❌ Fehler beim Speichern der Konfiguration: %v\n", err)
			return
		}

		cmd.Printf("✅ Feed %s erfolgreich entfernt.\n", url)
	},
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type RSSFeed struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

var rssImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Fetch RSS feeds and import new articles as drafts",
	Run: func(cmd *cobra.Command, args []string) {
		feeds := config.ActiveConfig.RSSFeeds
		if len(feeds) == 0 {
			cmd.Println("Keine RSS-Feeds konfiguriert. Füge einen hinzu mit: postctl rss add <url>")
			return
		}

		ctx := context.Background()
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			cmd.Printf("❌ Fehler beim Öffnen der Datenbank: %v\n", err)
			return
		}
		defer s.Close()

		client := &http.Client{Timeout: 15 * time.Second}
		importedCount := 0

		cmd.Println("Scanne RSS-Feeds nach neuen Beiträgen...")
		cmd.Println("========================================")

		for _, feedURL := range feeds {
			cmd.Printf("Lade Feed %s...\n", feedURL)
			req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
			if err != nil {
				cmd.Printf("  ❌ Fehler beim Erstellen des Requests: %v\n", err)
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				cmd.Printf("  ❌ Netzwerkfehler beim Laden des Feeds: %v\n", err)
				continue
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				cmd.Printf("  ❌ Fehler beim Lesen des Feeds: %v\n", err)
				continue
			}

			var feed RSSFeed
			if err := xml.Unmarshal(bodyBytes, &feed); err != nil {
				cmd.Printf("  ❌ Fehler beim Parsen des RSS-XMLs: %v\n", err)
				continue
			}

			cmd.Printf("  Gefunden: %d Beiträge in '%s'\n", len(feed.Channel.Items), feed.Channel.Title)

			for _, item := range feed.Channel.Items {
				hasher := md5.New()
				hasher.Write([]byte(item.Link))
				hash := hex.EncodeToString(hasher.Sum(nil))
				postID := fmt.Sprintf("rss-%s", hash[:12])

				_, err := s.GetPost(ctx, postID)
				if err == nil {
					continue
				}

				post := &models.Post{
					ID:         postID,
					Platform:   models.PlatformTwitter,
					Type:       "post",
					Status:     models.StatusDraft,
					Title:      item.Title,
					Body:       item.Title + "\n\n" + item.Description + "\n\n" + item.Link,
					SourceFile: feedURL,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				if err := s.SavePost(ctx, post); err != nil {
					cmd.Printf("  ❌ Fehler beim Speichern von Beitrag '%s': %v\n", item.Title, err)
					continue
				}

				cmd.Printf("  ✨ Neu importiert: %s\n", item.Title)
				importedCount++
			}
		}

		cmd.Println("========================================")
		cmd.Printf("Fertig! %d neue Beiträge erfolgreich als Drafts importiert.\n", importedCount)
	},
}

func init() {
	rssCmd.AddCommand(rssAddCmd)
	rssCmd.AddCommand(rssListCmd)
	rssCmd.AddCommand(rssRemoveCmd)
	rssCmd.AddCommand(rssImportCmd)
	rootCmd.AddCommand(rssCmd)
}
