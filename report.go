package main

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
)

type alertReport struct {
	startDay time.Time
	endDay   time.Time

	alerts         []alert
	alertTotal     int
	offHourTotal   int
	sleepHourTotal int
}

type responder struct {
	name      string
	offHour   int
	sleepHour int

	// these fields should be calculated
	// alertTotal     int
	// offHourTotal   int
	// sleepHourTotal int
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

func (r *alertReport) sleepHourAlertsByResponder() map[string]int {
	res := map[string]int{}
	for _, a := range r.alerts {
		for _, p := range a.responders {
			if p.sleepHour < 1 {
				continue
			}

			if _, ok := res[p.name]; !ok {
				res[p.name] = 1
			} else {
				res[p.name] += 1
			}
		}
	}

	return res
}

func (r *alertReport) offHourAlertsByResponder() map[string]int {
	res := map[string]int{}
	for _, a := range r.alerts {
		for _, p := range a.responders {
			if p.offHour < 1 {
				continue
			}

			if _, ok := res[p.name]; !ok {
				res[p.name] = 1
			} else {
				res[p.name] += 1
			}
		}
	}

	return res
}

func (r *alertReport) emit() {

	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetHeader([]string{"Alert", "Description", "Responder", "Off Hour Alert", "Sleep Hour Alert"})
	tw.SetFooter([]string{"", "", fmt.Sprintf("Alerts: %v", r.alertTotal), fmt.Sprintf("Off Hours: %v", r.offHourTotal), fmt.Sprintf("Sleep Hours: %v", r.sleepHourTotal)})
	for _, a := range r.alerts {
		for _, v := range a.responders {
			if v.offHour > 0 || v.sleepHour > 0 {
				tw.Append([]string{a.id, a.desc, v.name, fmt.Sprint(v.offHour), fmt.Sprint(v.sleepHour)})
			}
		}
	}

	stw := tablewriter.NewWriter(os.Stdout)
	stw.SetHeader([]string{"Name", "Sleep Hour Alerts"})
	for k, v := range r.sleepHourAlertsByResponder() {
		stw.Append([]string{k, fmt.Sprint(v)})
	}

	fmt.Println("Off Hour Alerts")
	otw := tablewriter.NewWriter(os.Stdout)
	otw.SetHeader([]string{"Name", "Sleep Hour Alerts"})
	for k, v := range r.offHourAlertsByResponder() {
		stw.Append([]string{k, fmt.Sprint(v)})
	}

	fmt.Println("All Alerts")
	tw.Render()
	fmt.Println("Off Hour Alerts")
	otw.Render()
	fmt.Println("Sleep Hour Alerts")
	stw.Render()
}
