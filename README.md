pagerdy off hour reporter
--

![S](https://github.com/imup-io/client/workflows/CodeQL/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/phanyzewski/pd-off-hour-reporter.svg)](https://pkg.go.dev/github.com/phanyzewski/pd-off-hour-reporter)

## On-Call Health Check

PagerDuty Sleep Reporter is a supplemental analytics package that allows you to get a better insight into your on-call team's health.
The default behavior of the app is to look at a single month at a time, but generally you can pull a quarter's worth of data from the API
before running into validation errors with pagerduty's API.

## Requirements

A pager duty api token set in your environment.  PAGERDUTY_API_TOKEN [generating an api key](https://support.pagerduty.com/docs/api-access-keys#section-generate-a-general-access-rest-api-key)

## Optional Configuration

additional command-line arguments are as follows

```sh
$ ./pd-off-hour-reporter -h
Usage of ./pd-off-hour-reporter:
  -end-month string
     ending month (default current month)
  -end-year string
     scoped year for analytics (default current year)
  -start-month string
     starting month (default previous month)
  -start-year string
     scoped year for analytics (default current year)
 -team-ids string
  comma separated list of pager duty team ids (default does not filter to any specific team)
```

## Generating a report

install the app

```sh
go install github.com/phanyzewski/pd-off-hour-reporter@latest
```

set your PAGERDUTY_API_TOKEN in your environment, and run the app for a period of time.

example, to generate a report for the month of January of 2023 for your teams

```sh
./pd-off-hour-reporter -start-year 2023 -start-month 01 -end-month 0 --team-ids UUID1,UUID2
```
