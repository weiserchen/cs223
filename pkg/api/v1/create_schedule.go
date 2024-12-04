package v1

import (
	"context"
	"fmt"
	"math/rand"
	"txchain/pkg/router"
)

func createSchedule(cfg *router.Config, scheduleLen int, proportion []int) ([]string, error) {
	eventCount, err := pgInstance.getEventCount(cfg.Ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to query events: %w", err)
	}
	participantCount, err := pgInstance.getParticipantCount(cfg.Ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to query participants: %w", err)
	}
	var schedule []string
	total := 0
	for i := range len(proportion) {
		total += proportion[i]
	}
	for len(schedule) < scheduleLen {
		random := rand.Intn(total) + 1
		if random < proportion[0] {
			schedule = append(schedule, "Create")
			HandleTxCreateEvent(cfg)
			eventCount++
		} else if random < proportion[0]+proportion[1] {
			schedule = append(schedule, "Update")
			HandleTxUpdateEvent(cfg)
		} else if random < proportion[0]+proportion[1]+proportion[2] {
			if eventCount > 0 {
				schedule = append(schedule, "Delete")
				HandleTxDeleteEvent(cfg)
				eventCount--
			}
		} else if random < proportion[0]+proportion[1]+proportion[2]+proportion[3] {
			schedule = append(schedule, "Join")
			HandleTxJoinEvent(cfg)
		} else {
			if participantCount > 0 {
				schedule = append(schedule, "Leave")
				HandleTxLeaveEvent(cfg)
			}
		}
	}
	return schedule, nil
}

func (pg *postgres) getEventCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM Events`
	var count int
	err := pg.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("unable to query events: %w", err)
	}
	return count, nil
}

func (pg *postgres) getParticipantCount(ctx context.Context) (int, error) {
	query := `SELECT SUM(array_length(participants, 1)) FROM Events WHERE participants IS NOT NULL;`
	var participantCount int
	err := pg.db.QueryRow(ctx, query).Scan(&participantCount)
	if err != nil {
		return 0, fmt.Errorf("unable to query total participants: %w", err)
	}
	return participantCount, nil
}
