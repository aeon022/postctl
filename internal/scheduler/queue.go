package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// GetNextQueueSlot sucht den nächsten freien Veröffentlichungs-Slot für eine Plattform
func GetNextQueueSlot(ctx context.Context, s *store.SQLiteStore, platform string) (time.Time, error) {
	slots := config.ActiveConfig.Scheduler.Slots
	if len(slots) == 0 {
		slots = []string{"Mon 09:00", "Wed 14:00", "Fri 17:30"}
	}

	type parsedSlot struct {
		weekday time.Weekday
		hour    int
		minute  int
	}

	var parsedSlots []parsedSlot
	weekdayMap := map[string]time.Weekday{
		"mon": time.Monday,
		"tue": time.Tuesday,
		"wed": time.Wednesday,
		"thu": time.Thursday,
		"fri": time.Friday,
		"sat": time.Saturday,
		"sun": time.Sunday,
	}

	for _, slotStr := range slots {
		parts := strings.Fields(strings.ToLower(slotStr))
		if len(parts) != 2 {
			continue
		}
		weekday, ok := weekdayMap[parts[0]]
		if !ok {
			// Kurze Formate wie "mo", "di" etc. unterstützen, falls der Nutzer Deutsch eingibt
			deMap := map[string]time.Weekday{
				"mo": time.Monday,
				"di": time.Tuesday,
				"mi": time.Wednesday,
				"do": time.Thursday,
				"fr": time.Friday,
				"sa": time.Saturday,
				"so": time.Sunday,
			}
			weekday, ok = deMap[parts[0]]
			if !ok {
				continue
			}
		}
		var hour, minute int
		_, err := fmt.Sscanf(parts[1], "%d:%d", &hour, &minute)
		if err != nil {
			continue
		}
		parsedSlots = append(parsedSlots, parsedSlot{
			weekday: weekday,
			hour:    hour,
			minute:  minute,
		})
	}

	if len(parsedSlots) == 0 {
		return time.Time{}, fmt.Errorf("no valid scheduler slots configured (expected format: 'Mon 09:00')")
	}

	searchStart := time.Now()

	for iteration := 0; iteration < 365; iteration++ {
		var nextSlot time.Time
		for _, ps := range parsedSlots {
			occurrence := nextOccurrenceOfSlot(searchStart, ps.weekday, ps.hour, ps.minute)
			if nextSlot.IsZero() || occurrence.Before(nextSlot) {
				nextSlot = occurrence
			}
		}

		// Prüfen, ob für diese Plattform bereits ein Post in diesem Slot (+/- 2 Minuten) geplant ist
		posts, err := s.ListPosts(ctx, platform, models.StatusScheduled, "")
		if err != nil {
			return time.Time{}, err
		}

		slotAvailable := true
		for _, p := range posts {
			if p.ScheduledAt != nil {
				diff := p.ScheduledAt.Sub(nextSlot)
				if diff < 0 {
					diff = -diff
				}
				if diff < 2*time.Minute {
					slotAvailable = false
					break
				}
			}
		}

		if slotAvailable {
			return nextSlot, nil
		}

		searchStart = nextSlot
	}

	return time.Time{}, fmt.Errorf("failed to find an available queue slot within 365 attempts")
}

func nextOccurrenceOfSlot(t time.Time, weekday time.Weekday, hour, minute int) time.Time {
	// Erstelle ein Datum am gleichen Tag wie t, aber mit der Stunde/Minute des Slots
	candidate := time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, t.Location())

	daysDiff := int(weekday) - int(t.Weekday())
	if daysDiff < 0 {
		daysDiff += 7
	}

	candidate = candidate.AddDate(0, 0, daysDiff)

	// Falls der Kandidat in der Vergangenheit liegt, plane ihn für nächste Woche
	if !candidate.After(t) {
		candidate = candidate.AddDate(0, 0, 7)
	}

	return candidate
}
