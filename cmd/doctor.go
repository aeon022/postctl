package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aeon022/missionctl-core/doctor"
	"github.com/aeon022/postctl/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check config and database health",
	Run: func(cmd *cobra.Command, args []string) {
		profile := config.ActiveProfile
		if profile == "" {
			profile = "default"
		}
		checks := []doctor.Check{
			{Label: "Profile", OK: true, Detail: profile},
			doctor.CheckSQLite("Database", config.GetDBPath(), "posts"),
			doctor.CheckDataDir("Data directory", config.GetDBPath(), config.Shared()),
		}
		if c, ok := instagramTokenCheck(config.ActiveConfig.Instagram.API, config.ActiveConfig.Instagram.AccessToken != "", config.ActiveConfig.Instagram.TokenIssued, time.Now()); ok {
			checks = append(checks, c)
		}
		if !doctor.PrintReport(checks) {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// instagramTokenCheck reports the age of the Instagram-Login token. Tokens last 60 days; the scheduler
// daemon refreshes them from day 30 on, so an age beyond 45 days means the refresh is not happening
// (daemon not running or Instagram rejected it) and the token expires soon. ok=false: no such token.
func instagramTokenCheck(api string, hasToken bool, issued string, now time.Time) (doctor.Check, bool) {
	if api != "instagram" || !hasToken || issued == "" {
		return doctor.Check{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", issued, time.Local)
	if err != nil {
		return doctor.Check{Label: "Instagram token", OK: false, Detail: fmt.Sprintf("token_issued %q is not a date (YYYY-MM-DD)", issued)}, true
	}
	age := int(now.Sub(t).Hours() / 24)
	left := 60 - age
	switch {
	case left <= 0:
		return doctor.Check{Label: "Instagram token", OK: false, Detail: fmt.Sprintf("issued %s — expired; run `postctl auth instagram`", issued)}, true
	case age > 45:
		return doctor.Check{Label: "Instagram token", OK: false, Detail: fmt.Sprintf("issued %s, %d days left — the daemon should have refreshed it from day 30; is `postctl daemon` running?", issued, left)}, true
	}
	return doctor.Check{Label: "Instagram token", OK: true, Detail: fmt.Sprintf("issued %s, %d days old; refreshed automatically by the scheduler from day 30", issued, age)}, true
}
