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
		Since:     "2022-09-01T00:00:00-05:00",
		Until:     "2022-10-01T00:00:00-05:00",
		Urgencies: []string{"high"},
		// dev on-call https://teamsnap.pagerduty.com/teams/PTBNXW0/users
		// infra on-call https://teamsnap.pagerduty.com/teams/PTV792K/users
		TeamIDs: []string{"PTV792K", "PTBNXW0"},

		TimeZone: "UTC",
	})

	return resp, err
}

// for incidents resolved by an api integration, get paged humans
func incidentResponders(id string) []string {
	blockList := []string{"PCNS2G8", "PHGM5IU", "P0MFEHL"}
	for _, v := range blockList {
		if id == v {
			return []string{}
		}
	}

	i, err := client.ListIncidentLogEntriesWithContext(context.Background(), id,
		pagerduty.ListIncidentLogEntriesOptions{
			Limit:  100,
			Offset: 0,
			Total:  true,
		},
	)
	var aerr pagerduty.APIError

	// handle api errors
	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			fmt.Printf("missing log entries for: %s\n", id)
		} else {
			fmt.Printf("unknown status code ListIncidentLogEntriesWithContext: %v, %v\n", id, aerr.StatusCode)
			return []string{}
		}
	}

	if i.Total > 100 {
		fmt.Println("warn, more than 100 log entries for incident", id)
		fmt.Println("count: ", i.Total)
	}

	users := map[string]bool{}
	for _, v := range i.LogEntries {
		if v.User.ID != "" {
			if _, ok := users[v.User.ID]; !ok {
				users[v.User.ID] = true
			}
		}
	}

	ids := []string{}
	for k, _ := range users {
		ids = append(ids, k)
	}

	return ids
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
	sleepPeople := map[string]int{}
	for _, i := range incidents {
		chars := min(40, len(i.Summary))
		alert := alert{id: i.ID, desc: i.Summary[:chars], responders: map[string]responder{}}
		ids := incidentResponders(i.ID)
		if len(ids) < 1 {
			ids = []string{i.LastStatusChangeBy.ID}
		}

		for _, v := range ids {
			// don't count stats for bots
			if !isService(v) {
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

					// stub in the beginning on sleep week SLOs
					_, week := t.ISOWeek()
					key := fmt.Sprintf("%v%v", name, week)
					sleepPeople[key] += 1

					if sleepPeople[key] > 1 {
						fmt.Printf("%s: Sleep interruption SLO violation week: %v Number of interruptions: %v\n", name, week, sleepPeople[key])
					}

					report.sleepHourTotal += 1
				} else if no := isBusinessHours(t); no {
					person.offHour += 1
					report.offHourTotal += 1
				}

				alert.responders[name] = person
			} else {
				fmt.Println("service name: ", getServiceName(v))
			}
		}

		report.alerts = append(report.alerts, alert)
	}

	report.alertTotal = len(incidents)
	report.emit()
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

func getUserName(id string) string {
	ctx := context.Background()
	resp, err := client.GetUserWithContext(ctx, id, pagerduty.GetUserOptions{})

	var aerr pagerduty.APIError

	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			return id
		} else {
			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	return resp.Name
}

func getServiceName(id string) string {
	ctx := context.Background()
	resp, err := client.GetServiceWithContext(ctx, id, &pagerduty.GetServiceOptions{})

	var aerr pagerduty.APIError

	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			return id
		} else {
			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	return resp.Name
}

func isService(id string) bool {
	ctx := context.Background()
	_, err := client.GetServiceWithContext(ctx, id, &pagerduty.GetServiceOptions{})

	var aerr pagerduty.APIError

	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			return false
		} else {
			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	return true
}

func isSleepHours(dt time.Time) bool {
	// - Sleep Hours: 10pm-8am every day, based on the user’s time zone.
	eod := 21
	bod := 8
	if dt.Hour() > eod || dt.Hour() < bod {
		return true
	}

	return false
}

func isBusinessHours(dt time.Time) bool {
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
