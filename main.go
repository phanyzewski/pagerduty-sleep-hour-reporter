package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

var client *pagerduty.Client

func main() {
	// data := pagerduty.AnalyticsData{}
	token := os.Getenv("PAGERDUTY_API_TOKEN")
	if len(token) < 1 {
		fmt.Println("env PAGERDUTY_API_TOKEN is required")
		os.Exit(1)
	}
	client = pagerduty.NewClient(token)

	// offHourReport()
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

	sleepHourAlerts := map[string]int{}
	devSleepHourAlerts := map[string]int{}
	infraSleepHourAlerts := map[string]int{}
	alertsForMonth := map[string]int{}
	sleepPeople := map[string]int{}
	dupe := 0
	dedupe := map[string]bool{}
	for _, i := range incidents {
		tz, ok := getUserTimeZone(i.LastStatusChangeBy.ID)
		if !ok {
			// fmt.Printf("No time zone found, likely a service responder not a user: %v\n", i.LastStatusChangeBy.Summary)
			continue
		}

		userZone, err := time.LoadLocation(tz)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var t time.Time
		if t, err = time.ParseInLocation(time.RFC3339, i.CreatedAt, userZone); err != nil {
			fmt.Printf("bad parse: %s\n", err)
			os.Exit(1)
		}

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
				}
			} else {
				// fmt.Printf("DUPE: %s, YearDay: %v Hour: %v Minute: %v \n", dupeKey, t.YearDay(), t.Hour(), t.Minute())
				dupe += 1
				continue

			}

			// fmt.Printf("Time in responders zone: %v\n", t.Hour())
			// fmt.Printf("Responder: %v\n", i.LastStatusChangeBy.Summary)
			// fmt.Printf("Incident ID: %v\n", i.ID)
			// fmt.Printf("Sleeping Hour Alert Detected!: %s\n", i.Description)

			for _, v := range i.Teams {
				// infra on-call https://teamsnap.pagerduty.com/teams/PTV792K/users
				if v.ID == "PTV792K" {
					infraSleepHourAlerts[t.Month().String()] += 1
				}
				// dev on-call https://teamsnap.pagerduty.com/teams/PTBNXW0/users
				if v.ID == "PTBNXW0" {
					devSleepHourAlerts[t.Month().String()] += 1
				}
			}

			sleepHourAlerts[t.Month().String()] += 1
		}

		alertsForMonth[t.Month().String()] += 1
	}

	// fmt.Printf("Total Off Hour Alerts for all teams: %v\n", offHourAlerts)

	fmt.Printf("Total Pages for all teams: %v\n", len(incidents))
	fmt.Printf("Total Sleep Hour Alerts for all teams: %v\n", sleepHourAlerts)
	fmt.Printf("Total Sleep Hour Alerts for infra: %v\n", infraSleepHourAlerts)
	fmt.Printf("Total Sleep Hour Alerts for dev: %v\n", devSleepHourAlerts)

	fmt.Println("dupes: ", dupe)
	// fmt.Printf("Total Business Hour Alerts for all teams: %v\n", businessHourAlerts)
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
			// fmt.Printf("cannot find timezone for %s\n", id)
			return "", false
		} else {
			fmt.Println("unknown status code getting user time zone:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	// fmt.Printf("timezone: %s\n", resp.Timezone)

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
			// fmt.Printf("cannot find timezone for %s\n", id)
			return "disabled-user", ""
		} else {
			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	// fmt.Printf("timezone: %s\n", resp.Timezone)

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

// func offHourReport() {
// 	// https://developer.pagerduty.com/api-reference/c2d493e995071-get-raw-data-multiple-incidents
// 	// 	{
// 	//   "filters": {
// 	//     "created_at_start": "2021-01-01T00:00:00-05:00",
// 	//     "created_at_end": "2021-01-31T00:00:00-05:00",
// 	//     "urgency": "high",
// 	//     "major": true,
// 	//     "team_ids": [
// 	//       "PGVXG6U",
// 	//       "PNVU4U4"
// 	//     ],
// 	//     "service_ids": [
// 	//       "PQVUB8D",
// 	//       "PU2D9X3"
// 	//     ],
// 	//     "priority_names": [
// 	//       "P1",
// 	//       "P2"
// 	//     ]
// 	//   },
// 	//   "limit": 20,
// 	//   "order": "desc",
// 	//   "order_by": "created_at",
// 	//   "time_zone": "Etc/UTC"
// 	// }

// 	ctx := context.Background()
// 	// resp, err := client.GetAggregatedServiceData(ctx, pagerduty.AnalyticsRequest{
// 	resp, err := client.GetAggregatedTeamData(ctx, pagerduty.AnalyticsRequest{
// 		Filters: &pagerduty.AnalyticsFilter{
// 			CreatedAtStart: "2022-06-01T00:00:00-05:00",
// 			CreatedAtEnd:   "2022-07-01T00:00:00-05:00",
// 			Urgency:        "high",
// 			// Major:          false,
// 			// dev on-call https://teamsnap.pagerduty.com/teams/PTBNXW0/users
// 			// infra on-call https://teamsnap.pagerduty.com/teams/PTV792K/users
// 			TeamIDs: []string{"PTV792K", "PTBNXW0"},
// 			// ServiceIDs: []string{"P6YWFZ0"},
// 		},
// 		TimeZone: "MST",
// 	})

// 	// resp, err := client.Listin

// 	var aerr pagerduty.APIError

// 	if errors.As(err, &aerr) {
// 		if aerr.RateLimited() {
// 			fmt.Println("rate limited")
// 			os.Exit(1)
// 		}

// 		fmt.Println("unknown status code:", aerr.StatusCode)

// 		os.Exit(1)
// 	}

// 	incidentCount := 0
// 	offHourAlerts := 0
// 	sleepHourAlerts := 0
// 	businessHourAlerts := 0
// 	for _, d := range resp.Data {
// 		fmt.Printf("%+v\n", d)

// 		fmt.Printf("Off Hour Alerts: %v\n", d.TotalOffHourInterruptions)
// 		fmt.Printf("Sleep Hour Alerts: %v\n", d.TotalSleepHourInterruptions)
// 		fmt.Printf("Business Hour Alerts: %v\n", d.TotalBusinessHourInterruptions)
// 		fmt.Printf("Incident Count: %v\n", d.TotalIncidentCount)
// 		offHourAlerts += d.TotalOffHourInterruptions
// 		sleepHourAlerts += d.TotalSleepHourInterruptions
// 		incidentCount += d.TotalIncidentCount
// 		businessHourAlerts += d.TotalBusinessHourInterruptions
// 	}

// 	fmt.Printf("Total Pages for all teams: %v\n", incidentCount)
// 	fmt.Printf("Total Off Hour Alerts for all teams: %v\n", offHourAlerts)
// 	fmt.Printf("Total Sleep Hour Alerts for all teams: %v\n", sleepHourAlerts)
// 	fmt.Printf("Total Business Hour Alerts for all teams: %v\n", businessHourAlerts)
// }
