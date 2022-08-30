package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

//
//
// Since:     "2021-08-01T00:00:00-05:00",
// Until:     "2022-01-01T00:00:00-05:00",
//
// Total Pages for all teams: 607
// Total Sleep Hour Alerts for all teams: map[August:19 December:7 November:5 October:13 September:13]
// Total Off Hour Alerts for all teams: map[August:5 December:2 November:2 October:7 September:6]

// Since:     "2022-01-01T00:00:00-05:00",
// Until:     "2022-07-01T00:00:00-05:00",
//
// Total Pages for all teams: 683
// Total Sleep Hour Alerts for all teams: map[April:15 February:11 January:8 June:13 March:15 May:27]
// Total Off Hour Alerts for all teams: map[April:2 February:1 January:2 June:6 March:4 May:8]

// Since:     "2022-07-01T00:00:00-05:00",
// Until:     "2022-09-01T00:00:00-05:00",
//
// Total Pages for all teams: 121
// Total Sleep Hour Alerts for all teams: map[August:4 July:2]
// Total Off Hour Alerts for all teams: map[August:2]
var client *pagerduty.Client

func main() {
	// data := pagerduty.AnalyticsData{}
	token := os.Getenv("PAGERDUTY_API_TOKEN")
	if len(token) < 1 {
		fmt.Println("env PAGERDUTY_API_TOKEN is required")
		os.Exit(1)
	}
	client = pagerduty.NewClient(token)

	listIncidents()
}

func BeginningOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 0, -date.Day()+1)
}

func EndOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 1, -date.Day())
}

func listIncidentResponse(offset int) (*pagerduty.ListIncidentsResponse, error) {
	ctx := context.Background()
	resp, err := client.ListIncidentsWithContext(ctx, pagerduty.ListIncidentsOptions{
		Limit:     100,
		Offset:    uint(offset),
		Since:     "2022-01-01T00:00:00-05:00",
		Until:     "2022-07-01T00:00:00-05:00",
		Urgencies: []string{"high"},
		// dev on-call https://teamsnap.pagerduty.com/teams/PTBNXW0/users
		// infra on-call https://teamsnap.pagerduty.com/teams/PTV792K/users
		TeamIDs: []string{"PTV792K", "PTBNXW0"},

		TimeZone: "UTC",
	})

	return resp, err
}

// func incidentLog(id string) {
// 	i, err := client.ListIncidentLogEntriesWithContext(context.Background(), id, pagerduty.ListIncidentLogEntriesOptions{})
// 	var aerr pagerduty.APIError

// 	// handle api errors
// 	if errors.As(err, &aerr) {
// 		if aerr.RateLimited() {
// 			fmt.Println("rate limited")
// 			os.Exit(1)
// 		}

// 		fmt.Println("unknown status code:", aerr.StatusCode)
// 		os.Exit(1)
// 	}

// 	for _, v := range i.LogEntries {

// 		fmt.Printf("log: %+v\n", v)
// 	}
// }

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
		// fmt.Println("current offset: ", offset)
		// fmt.Println("current incidents: ", len(i.Incidents))
		// fmt.Println("total incidents: ", len(incidents))

		if !i.More {
			more = false
		}

		offset += 100
	}

	bots := []string{"Platform Datadog Alerts", "Datadog Ops", "Ops Low Priority Alerts", "AWS", "Ghost Inspector Payments", "Hoyle", "SNAPI"}
	offHourAlerts := map[string]int{}
	sleepHourAlerts := map[string]int{}
	alertsForMonth := map[string]int{}
	sleepPeople := map[string]int{}
	dupe := 0
	dedupe := map[string]bool{}
	for _, i := range incidents {
		tz, ok := getUserTimeZone(i.LastStatusChangeBy.ID)
		if !ok {
			for _, v := range bots {
				if i.LastStatusChangeBy.Summary == v {
					// fmt.Printf("No time zone for bot: %s, using America/Denver\n", i.LastStatusChangeBy.Summary)
					tz = "America/Denver"
					break
				}
			}
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
		if yes := sleepHours(t); yes {
			// dedupe incidents that happen on the same day/hour/minute for a single responder
			dupeKey := fmt.Sprintf("%v%v%v", i.LastStatusChangeBy.ID, t.YearDay(), t.Hour())
			// dupeKey := fmt.Sprintf("KEY: %v", i.IncidentKey)
			if _, ok := dedupe[dupeKey]; !ok {
				dedupe[dupeKey] = true
				_, week := t.ISOWeek()
				key := fmt.Sprintf("%v%v", i.LastStatusChangeBy.ID, week)
				sleepPeople[key] += 1

				if sleepPeople[key] > 1 {
					name, _ := getUserName(i.LastStatusChangeBy.ID)
					fmt.Printf("ALERT %s sleep interruption SLO violation for week: %v Number of interruptions: %v\n", name, week, sleepPeople[key])
				} else {
					fmt.Printf("sleep interruption: %v \n", i.Summary)
				}
			} else {
				// fmt.Printf("DUPE: %s, YearDay: %v Hour: %v Minute: %v \n", dupeKey, t.YearDay(), t.Hour(), t.Minute())
				dupe += 1
				continue

			}

			// debug
			// fmt.Println()
			// fmt.Println("Parsed time: ", t)
			// fmt.Printf("Initial Responder: %v\n", i.FirstTriggerLogEntry)
			// fmt.Printf("Final Responder: %v\n", i.LastStatusChangeBy.Summary)
			// fmt.Printf("Incident ID: %v\n", i.ID)
			// fmt.Printf("Sleeping Hour Alert Detected!: %s\n", i.Description)

			sleepHourAlerts[t.Month().String()] += 1
		}

		if no := businessHours(t); no {
			dupeKey := fmt.Sprintf("%v%v%v", i.LastStatusChangeBy.ID, t.YearDay(), t.Hour())
			// dupeKey := fmt.Sprintf("KEY: %v", i.IncidentKey)
			if _, ok := dedupe[dupeKey]; !ok {
				dedupe[dupeKey] = true
			} else {
				// fmt.Printf("DUPE: %s, YearDay: %v Hour: %v Minute: %v \n", dupeKey, t.YearDay(), t.Hour(), t.Minute())
				dupe += 1
				continue
			}

			// fmt.Println()
			// fmt.Println("Parsed time: ", t)
			// fmt.Printf("Responder: %v\n", i.LastStatusChangeBy.Summary)
			// fmt.Printf("Incident ID: %v\n", i.ID)
			// fmt.Printf("Off Hour Alert Detected!: %s\n", i.Description)

			offHourAlerts[t.Month().String()] += 1
		}

		alertsForMonth[t.Month().String()] += 1
	}

	fmt.Printf("Total Pages for all teams: %v\n", len(incidents))
	fmt.Printf("Total Sleep Hour Alerts for all teams: %v\n", sleepHourAlerts)
	fmt.Printf("Total Off Hour Alerts for all teams: %v\n", offHourAlerts)
	// fmt.Printf("Total Sleep Hour Alerts for infra: %v\n", infraSleepHourAlerts)
	// fmt.Printf("Total Sleep Hour Alerts for dev: %v\n", devSleepHourAlerts)
}

func getUserTimeZone(id string) (string, bool) {
	ctx := context.Background()
	resp, err := client.GetUserWithContext(ctx, id, pagerduty.GetUserOptions{})

	var aerr pagerduty.APIError

	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			return "", false
		} else {
			fmt.Println("unknown status code getting user time zone:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	return resp.Timezone, true
}

func getUserName(id string) (string, string) {
	ctx := context.Background()
	resp, err := client.GetUserWithContext(ctx, id, pagerduty.GetUserOptions{})

	var aerr pagerduty.APIError

	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			return "disabled-user", ""
		} else {
			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	return resp.Name, resp.Email
}

func sleepHours(dt time.Time) bool {
	// - Sleep Hours: 10pm-8am every day, based on the user’s time zone.
	eod := 21
	bod := 8
	if dt.Hour() > eod || dt.Hour() < bod {
		return true
	}

	return false
}

func businessHours(dt time.Time) bool {
	// - Business Hours: 8am-6pm Mon-Fri, based on the user’s time zone.
	bod := 8
	eod := 19
	// check day of week
	if dt.Weekday() == time.Saturday || dt.Weekday() == time.Sunday {
		return false
	}

	// check hour of day
	if dt.Hour() >= bod && eod < dt.Hour() {
		return true
	}

	return false
}
