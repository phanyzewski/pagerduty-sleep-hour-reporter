package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/PagerDuty/go-pagerduty"
)

var client *pagerduty.Client

// listIncidentResponse calls the pagerduty ListIncident API endpoint for a configurable time period
func listIncidentResponse(offset int) (*pagerduty.ListIncidentsResponse, error) {
	ctx := context.Background()
	resp, err := client.ListIncidentsWithContext(ctx, pagerduty.ListIncidentsOptions{
		Limit:     100,
		Offset:    uint(offset),
		Since:     fmt.Sprintf("%s-%s-01T00:00:00", startYear, "01"),
		Until:     fmt.Sprintf("%s-%s-01T00:00:00", endYear, "02"),
		Urgencies: []string{"high"},
		TeamIDs:   teamIDs,

		TimeZone: "UTC",
	})

	return resp, err
}

// incidentResponders calls the Incident Log API endpoint and returns
// a list of users who were paged
func incidentResponders(id string) []string {
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
	for k := range users {
		ids = append(ids, k)
	}

	return ids
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
