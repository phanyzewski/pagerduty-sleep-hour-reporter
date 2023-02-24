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

## Generating a Report

install the app

```sh
go install github.com/phanyzewski/pd-off-hour-reporter@latest
```

set your PAGERDUTY_API_TOKEN in your environment, and run the app for a period of time.

example, to generate a report for the month of January of 2023 for your teams

```sh
PAGERDUTY_API_TOKEN=token ./pd-off-hour-reporter -start-year 2023 -start-month 01 -end-month 0 --team-ids UUID1,UUID2
```

## Optional Configuration

additional command-line arguments are as follows, note that all dates assume a numeric representation

```sh
$ ./pd-off-hour-reporter --help
Usage of ./pd-off-hour-reporter:
  -end-month string
     ending numeric representation of ending month, eg 03 (default current month)
  -end-year string
     ending numeric representation of year, eg 2023 (default current years)
  -start-month string
     starting numeric representation of month, eg 12 (default last month)
  -start-year string
     starting numeric representation of year, eg 2022 (default current years)
  -team-ids string
     comma separated string of pager duty team ids

```
