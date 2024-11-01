package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

var (
	startYear  string
	endYear    string
	startMonth string
	endMonth   string
	teamIDs    []string

	setupFlags sync.Once
	client     *pagerduty.Client
)

func main() {
	token := os.Getenv("PAGERDUTY_API_TOKEN")
	if len(token) < 1 {
		fmt.Println("env PAGERDUTY_API_TOKEN is required")
		os.Exit(1)
	}

	var teams *string
	setupFlags.Do(func() {
		flag.StringVar(&startYear, "start-year", fmt.Sprintf("%v", time.Now().Year()), "starting numeric representation of year, eg 2022")
		flag.StringVar(&endYear, "end-year", fmt.Sprintf("%v", time.Now().Year()), "ending numeric representation of year, eg 2023")
		flag.StringVar(&startMonth, "start-month", fmt.Sprintf("%d", time.Now().Month()-1), "starting numeric representation of month, eg 12")
		flag.StringVar(&endMonth, "end-month", fmt.Sprintf("%d", time.Now().Month()), "ending numeric representation of ending month, eg 03")
		teams = flag.String("team-ids", "", "comma separated string of pager duty team ids")

		flag.Parse()
	})

	teamIDs = strings.Split(*teams, ",")
	client = pagerduty.NewClient(token)
	generateSleepHourReport()
}

func generateSleepHourReport() {
	incidents := incidents()
	report := &alertReport{}

	for _, i := range incidents {
		policy := getIncidentEscalationPolicy(i.EscalationPolicy.ID)
		chars := min(128, len(i.Summary))
		alert := alert{
			ID:         i.ID,
			Desc:       i.Summary[:chars],
			Responders: map[string]responder{},
			URL:        i.HTMLURL,
		}

		ids := responders(i.ID)
		if len(ids) < 1 {
			ids = []string{i.LastStatusChangeBy.ID}
		}

		for _, v := range ids {
			name := getUserName(v)
			tz, ok := getUserTimeZone(v)

			// if a user has no time zone use MT
			if !ok {
				tz = "America/Denver"
			}

			userZone, err := time.LoadLocation(tz)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			var utc time.Time
			if utc, err = time.Parse(time.RFC3339, i.CreatedAt); err != nil {
				fmt.Printf("bad parse: %s\n", err)
				os.Exit(1)
			}

			t := utc.In(userZone)

			var person responder
			if _, ok := alert.Responders[name]; !ok {
				person = responder{Name: name, EscalationPolicy: policy}
			} else {
				person = alert.Responders[name]
			}

			if yes := isSleepHours(t); yes {
				person.SleepHour += 1
				report.SleepHourTotal += 1
			} else if yes := isOffHours(t); yes {
				person.OffHour += 1
				report.OffHourTotal += 1
			}

			alert.DateTime = t
			alert.Responders[name] = person
		}

		report.Alerts = append(report.Alerts, alert)
	}

	report.AlertTotal = len(incidents)
	report.emitCsv()
}
