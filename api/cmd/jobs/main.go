// Command jobs inspects the dedicated broker. Retrying requires an explicit job ID.
package main

import (
	"classgame-api/internal/modules/attachments"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"classgame-api/internal/platform/redisqueue"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("jobs", flag.ContinueOnError)
	id := flags.String("id", "", "read one job's state")
	classID := flags.String("enqueue-attachment-class", "", "explicitly enqueue a persisted published attachment revision")
	missionID := flags.String("enqueue-attachment-mission", "", "mission ID for explicit attachment recovery")
	revision := flags.Int64("enqueue-attachment-revision", 0, "immutable revision for explicit attachment recovery")
	retry := flags.String("retry", "", "explicitly retry this stored archived/retry job")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	enqueueAttachment := *classID != "" || *missionID != "" || *revision != 0
	actions := 0
	for _, set := range []bool{*id != "", *retry != "", enqueueAttachment} {
		if set {
			actions++
		}
	}
	if flags.NArg() != 0 || actions > 1 {
		return fmt.Errorf("use either --id or --retry, without positional arguments")
	}
	job := attachments.Job{ClassID: *classID, MissionID: *missionID, Revision: *revision}
	if enqueueAttachment {
		if err := job.Validate(); err != nil {
			return fmt.Errorf("attachment recovery requires class, mission and positive revision")
		}
	}
	queue := os.Getenv("TASKS_QUEUE")
	if queue == "" {
		queue = "classgame-background"
	}
	client, err := redisqueue.Connect(context.Background(), redisqueue.Options{URL: os.Getenv("TASKS_REDIS_URL"), Queue: queue, Production: os.Getenv("APP_ENV") == "production"})
	if err != nil {
		return err
	}
	defer client.Close()
	if enqueueAttachment {
		if err := attachments.NewTaskScheduler(client.Destination(redisqueue.Attachments), time.Second).Enqueue(context.Background(), job); err != nil {
			return fmt.Errorf("attachment enqueue failed: %w", err)
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"id": attachments.TaskID(job), "action": "attachment enqueue requested"})
	}
	if *retry != "" {
		if err := client.Retry(*retry); err != nil {
			return fmt.Errorf("job retry failed; verify its state and ID")
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"id": *retry, "action": "retry requested"})
	}
	if *id != "" {
		state, err := client.Status(*id)
		if err != nil {
			return fmt.Errorf("job not found or broker unavailable")
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"id": *id, "state": state})
	}
	stats, err := client.Stats()
	if err != nil {
		return fmt.Errorf("queue not found or broker unavailable")
	}
	return json.NewEncoder(os.Stdout).Encode(stats)
}
