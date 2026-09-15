package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

func checkPastSchedules() error {
	all, err := plans()
	if err != nil {
		return err
	}
	reportPastSchedules(os.Stdout, all, time.Now())
	return nil
}

func reportPastSchedules(writer io.Writer, all []plan, today time.Time) {
	past := make([]plan, 0)
	for _, schedule := range all {
		if scheduleIsPast(schedule.EndDate, today) {
			past = append(past, schedule)
		}
	}
	if len(past) == 0 {
		fmt.Fprintln(writer, "No past schedules remain in Supabase.")
		return
	}
	sort.Slice(past, func(i, j int) bool {
		return past[i].EndDate.Before(past[j].EndDate)
	})
	noun, verb := "schedules", "remain"
	if len(past) == 1 {
		noun, verb = "schedule", "remains"
	}
	fmt.Fprintf(writer, "%d past %s %s in Supabase:\n", len(past), noun, verb)
	for _, schedule := range past {
		fmt.Fprintf(writer, "  %q — %s to %s\n", schedule.Content, schedule.StartDate.Format("02 Jan 2006"), schedule.EndDate.Format("02 Jan 2006"))
	}
	fmt.Fprintln(writer, "Run `task-planner delete old` to review and remove past schedules from Supabase only. Todoist tasks will stay unchanged. Use `task-planner delete` if you also want to remove their Todoist tasks.")
}
