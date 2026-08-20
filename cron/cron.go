package cron

import (
	"API/api/services"
	"context"
	"log"
	"time"
)

func StartCronJobs(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Default().Printf("running cron: deleting collected productions")
				services.DeleteCollectedProductions()
			case <-ctx.Done():
				log.Default().Printf("cron jobs stopped")
				return
			}
		}
	}()
}
