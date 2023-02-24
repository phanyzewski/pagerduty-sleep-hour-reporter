package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

type alertReport struct {
	alerts         []alert
	alertTotal     int
	offHourTotal   int
	sleepHourTotal int
}

type responder struct {
	name      string
	offHour   int
	sleepHour int
}

type alert struct {
	id         string
	desc       string
	responders map[string]responder
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (r *alertReport) emit() {
	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetHeader([]string{"Alert", "Description", "Responder", "Off Hour Alert", "Sleep Hour Alert"})
	tw.SetFooter([]string{"", "", fmt.Sprintf("Sleep Hours: %v", r.sleepHourTotal), fmt.Sprintf("Off Hours: %v", r.offHourTotal), fmt.Sprintf("Alerts: %v", r.alertTotal)})
	for _, a := range r.alerts {
		for _, v := range a.responders {
			if v.sleepHour > 0 {
				tw.Append([]string{a.id, a.desc, v.name, fmt.Sprint(v.offHour), fmt.Sprint(v.sleepHour)})
			}
		}
	}

	fmt.Printf("All Alerts from: %v to: %v\n", startMonth, endMonth)
	tw.Render()
}
