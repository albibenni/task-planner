package main

import (
	"fmt"
	"io"
)

func listSchedules(writer io.Writer, findAll func() ([]plan, error)) error {
	schedules, err := findAll()
	if err != nil {
		return err
	}
	for _, schedule := range schedules {
		fmt.Fprintf(writer, "%s — %s to %s\n", schedule.Content, schedule.StartDate.Format("2006-01-02"), schedule.EndDate.Format("2006-01-02"))
	}
	return nil
}
