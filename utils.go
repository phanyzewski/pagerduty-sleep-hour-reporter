package main

import "time"

func isSleepHours(dt time.Time) bool {
	// - Sleep Hours: 10pm-8am every day, based on the user’s time zone.
	eod := 21
	bod := 8
	if dt.Hour() > eod || dt.Hour() < bod {
		return true
	}

	return false
}

func isOffHours(dt time.Time) bool {
	// - Business Hours: 8am-6pm Mon-Fri, based on the user’s time zone.
	bod := 8
	eod := 19
	// check day of week
	if dt.Weekday() == time.Saturday || dt.Weekday() == time.Sunday {
		return true
	}

	// check hour of day
	if dt.Hour() >= bod && dt.Hour() < eod {
		// fmt.Printf("its business time %v\n", dt.Hour())
		return false
	}

	return true
}
