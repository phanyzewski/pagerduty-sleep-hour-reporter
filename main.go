package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// June 2022
// Total Pages for all teams: 86
// Total Sleep Hour Alerts for all teams: map[June:17]
// Total Off Hour Alerts for all teams: map[June:12]

// july 2022
// Total Pages for all teams: 47
// Total Sleep Hour Alerts for all teams: map[July:2]
// Total Off Hour Alerts for all teams: map[July:3]
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
		Since:     "2020-01-01T00:00:00-05:00",
		Until:     "2020-04-01T00:00:00-05:00",
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

	i, err := client.ListIncidentLogEntriesWithContext(context.Background(), id, pagerduty.ListIncidentLogEntriesOptions{Limit: 100, Offset: 0, Total: true})
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
			// os.Exit(1)
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
		// fmt.Println("current offset: ", offset)
		// fmt.Println("current incidents: ", len(i.Incidents))
		// fmt.Println("total incidents: ", len(incidents))

		if !i.More {
			more = false
		}

		offset += 100
	}

	offHourAlerts := map[string]int{}
	sleepHourAlerts := map[string]int{}
	// alertsForMonth := map[string]int{}
	sleepPeople := map[string]int{}
	dedupe := map[string]bool{}
	for _, i := range incidents {
		// tz, ok := getUserTimeZone(i.LastStatusChangeBy.ID)
		// assign a default timezone for API integrations, bots and unregistered users

		ids := incidentResponders(i.ID)
		if len(ids) < 1 {
			ids = []string{i.LastStatusChangeBy.ID}
		}

		for _, v := range ids {
			if !isService(v) {
				name := getUserName(v)
				// fmt.Println("responder: ", name)
				tz, ok := getUserTimeZone(v)
				if !ok {
					// fmt.Printf("No time zone for bot: %s, using America/Denver\n", name)
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
				if yes := sleepHours(t); yes {
					// dedupe incidents that happen on the same day/hour for a single incident
					dupeKey := fmt.Sprintf("%v%v%v", i.LastStatusChangeBy.ID, t.YearDay(), t.Hour())
					if _, ok := dedupe[dupeKey]; !ok {
						dedupe[dupeKey] = true
						_, week := t.ISOWeek()

						key := fmt.Sprintf("%v%v", name, week)
						sleepPeople[key] += 1
						fmt.Println()
						fmt.Println("page originated at: ", t)
						fmt.Printf("sleep interruption: %s %v \n", name, i.Summary)

						// debug
						// fmt.Println()
						// fmt.Println("Parsed time: ", t)
						// fmt.Printf("Initial Responder: %v\n", i.FirstTriggerLogEntry)
						// fmt.Printf("Final Responder: %v\n", i.LastStatusChangeBy.Summary)
						// fmt.Printf("Incident ID: %v\n", i.ID)
						// fmt.Printf("Sleeping Hour Alert Detected!: %s\n", i.Description)

						if sleepPeople[key] > 1 {
							fmt.Printf("%s: Sleep interruption SLO violation week: %v Number of interruptions: %v\n", name, week, sleepPeople[key])
						}

						sleepHourAlerts[t.Month().String()] += 1
					}
				} else if no := businessHours(t); no {
					dupeKey := fmt.Sprintf("%v%v%v", i.LastStatusChangeBy.ID, t.YearDay(), t.Hour())
					if _, ok := dedupe[dupeKey]; !ok {
						dedupe[dupeKey] = true
						offHourAlerts[t.Month().String()] += 1
					}
				}
			} else {
				fmt.Println("service name: ", getServiceName(v))
			}

			// alertsForMonth[t.Month().String()] += 1
		}
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
