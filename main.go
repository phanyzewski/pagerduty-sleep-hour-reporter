package main

import (
	"errors"
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
)

func main() {
	token := os.Getenv("PAGERDUTY_API_TOKEN")
	if len(token) < 1 {
		fmt.Println("env PAGERDUTY_API_TOKEN is required")
		os.Exit(1)
	}

	var teams *string
	setupFlags.Do(func() {
		flag.StringVar(&startYear, "start-year", fmt.Sprintf("%v", time.Now().Year()), "scoped year for analytics")
		flag.StringVar(&endYear, "end-year", fmt.Sprintf("%v", time.Now().Year()), "scoped year for analytics")
		flag.StringVar(&startMonth, "start-month", fmt.Sprintf("%d", time.Now().Month()-1), "starting month")
		flag.StringVar(&endMonth, "end-month", fmt.Sprintf("%d", time.Now().Month()), "ending month")
		teams = flag.String("team-ids", "", "comma separated string of pager duty team ids")

		flag.Parse()
	})

	teamIDs = strings.Split(*teams, ",")

	client = pagerduty.NewClient(token)

	listIncidents()
}

func listIncidents() {
	offset := 0
	incidents := []pagerduty.Incident{}
	var aerr pagerduty.APIError

	more := true
	for more {
		i, err := listIncidentResponse(offset)

		// handle api errors
		if errors.As(err, &aerr) {
			if aerr.RateLimited() {
				fmt.Println("rate limited")
				os.Exit(1)
			}

			fmt.Println("unknown status code:", aerr.StatusCode)
			os.Exit(1)
		}

		incidents = append(incidents, i.Incidents...)

		if !i.More {
			more = false
		}

		offset += 100
	}

	report := &alertReport{}
	for _, i := range incidents {
		chars := min(64, len(i.Summary))
		alert := alert{id: i.ID, desc: i.Summary[:chars], responders: map[string]responder{}}
		ids := incidentResponders(i.ID)
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
			if _, ok := alert.responders[name]; !ok {
				person = responder{name: name}
			} else {
				person = alert.responders[name]
			}

			if yes := isSleepHours(t); yes {
				person.sleepHour += 1
				report.sleepHourTotal += 1
			} else if yes := isOffHours(t); yes {
				person.offHour += 1
				report.offHourTotal += 1
			}

			alert.responders[name] = person
		}

		report.alerts = append(report.alerts, alert)
	}

	report.alertTotal = len(incidents)
	report.emit()
}
