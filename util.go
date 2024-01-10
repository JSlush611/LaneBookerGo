package main

import (
	"fmt"
	"time"
)

func setupBooker(booker *Booker) error {
	initializeLaneMappings(booker)
	calculateBookingTimes(booker)

	if err := booker.NewClient(); err != nil {
		return fmt.Errorf("error initializing new client: %w", err)
	}

	if err := booker.GetInitialCookies(); err != nil {
		return fmt.Errorf("error getting initial cookies: %w", err)
	}

	booker.FormatLoginWebsite()
	return nil
}

func initializeLaneMappings(booker *Booker) {
	booker.LaneToTrin = map[int]int{
		1:  100000301, // OUTSIDE
		2:  100000302,
		3:  100000303,
		91: 100000212, // INSIDE 30 minutes
		92: 100000213,
		93: 100000237, // INSIDE 60 minutes
		94: 100000215,
		95: 100000216,
		96: 100000217,
	}
}

func calculateBookingTimes(booker *Booker) {
	booker.STime2 = booker.STime.Add(30 * time.Minute)
	booker.ETime = booker.STime2
	booker.ETime2 = booker.STime2.Add(30 * time.Minute)
}
