package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/PagerDuty/go-pagerduty"
)

func incidents() []pagerduty.Incident {
	offset := 0
	incidents := []pagerduty.Incident{}
	var aerr pagerduty.APIError

	more := true
	for more {
		i, err := listIncidents(offset)

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
	return incidents
}

// listIncidents calls the pagerduty ListIncident API endpoint for a configurable time period
func listIncidents(offset int) (*pagerduty.ListIncidentsResponse, error) {
	ctx := context.Background()
	resp, err := client.ListIncidentsWithContext(ctx, pagerduty.ListIncidentsOptions{
		Limit:     100,
		Offset:    uint(offset),
		Since:     fmt.Sprintf("%s-%s-01T00:00:00", startYear, startMonth),
		Until:     fmt.Sprintf("%s-%s-01T00:00:00", endYear, endMonth),
		Urgencies: []string{"high"},
		TeamIDs:   teamIDs,
		TimeZone:  "UTC",
	})

	return resp, err
}

func responders(id string) []string {
	offset := 0
	logEntries := []pagerduty.LogEntry{}
	var aerr pagerduty.APIError

	more := true
	for more {
		l, err := listIncidentLogEntries(offset, id)

		// handle api errors
		if errors.As(err, &aerr) {
			if aerr.RateLimited() {
				fmt.Println("rate limited")
				os.Exit(1)
			}

			fmt.Println("unknown status code:", aerr.StatusCode)
			os.Exit(1)
		}

		logEntries = append(logEntries, l.LogEntries...)

		if !l.More {
			more = false
		}

		offset += 100
	}

	users := map[string]bool{}
	for _, v := range logEntries {
		// TODO: remove this hardcoded (bot) user id
		if v.User.ID != "" && v.User.ID != "PJX59OJ" {
			if _, ok := users[v.User.ID]; !ok {
				users[v.User.ID] = true
			}
		}
	}

	ids := []string{}
	for k := range users {
		ids = append(ids, k)
	}

	return ids
}

// listIncidentLogEntries calls the Incident Log API endpoint and returns
func listIncidentLogEntries(offset int, id string) (*pagerduty.ListIncidentLogEntriesResponse, error) {
	ctx := context.Background()
	resp, err := client.ListIncidentLogEntriesWithContext(ctx, id, pagerduty.ListIncidentLogEntriesOptions{
		Limit:    100,
		Offset:   uint(offset),
		Since:    fmt.Sprintf("%s-%s-01T00:00:00", startYear, startMonth),
		Until:    fmt.Sprintf("%s-%s-01T00:00:00", endYear, endMonth),
		TimeZone: "UTC",
	})

	return resp, err
}

// getUserTimeZone calls the PagerDuty Users Endpoint and returns a users timezone
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

// getUserName calls the PagerDuty Users API endpoint and returns the name of a user
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

// getUserTeam calls the PagerDuty Users API endpoint and returns the team9s) associated with a user
// func getUserTeam(id string) []string {
// 	ctx := context.Background()
// 	resp, err := client.GetUserWithContext(ctx, id, pagerduty.GetUserOptions{})

// 	var aerr pagerduty.APIError

// 	if errors.As(err, &aerr) {
// 		if aerr.RateLimited() {
// 			fmt.Println("rate limited")
// 			os.Exit(1)
// 		}

// 		if aerr.NotFound() {
// 			return nil
// 		} else {
// 			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
// 			os.Exit(1)
// 		}
// 	}
// 	teams := []string{}

// 	for _, t := range resp.Teams {
// 		teams = append(teams, t.Summary)
// 	}
// 	return teams
// }

// getIncidentEscalationPolicy calls the PagerDuty Users API endpoint and returns the team9s) associated with a user
func getIncidentEscalationPolicy(id string) string {
	ctx := context.Background()
	resp, err := client.GetEscalationPolicyWithContext(ctx, id, &pagerduty.GetEscalationPolicyOptions{})

	var aerr pagerduty.APIError

	if errors.As(err, &aerr) {
		if aerr.RateLimited() {
			fmt.Println("rate limited")
			os.Exit(1)
		}

		if aerr.NotFound() {
			return ""
		} else {
			fmt.Println("unknown status code when getting username:", aerr.StatusCode)
			os.Exit(1)
		}
	}

	return resp.Summary
}
