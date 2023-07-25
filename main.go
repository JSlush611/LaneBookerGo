package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.Handle("/", http.HandlerFunc(HomePage))
	http.Handle("/book", http.HandlerFunc(Progress))
	http.Handle("/ok.png", http.HandlerFunc(serveImage))

	log.Println("Starting serverr on port 8080...")

	http.ListenAndServe(":8080", nil)
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func Progress(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	booker := &Booker{}
	booker.Username = r.FormValue("email")
	booker.Password = r.FormValue("password")

	var err error
	booker.Lane, err = strconv.Atoi(r.FormValue("lane"))
	if err != nil {
		panic(err)
	}

	RawSTime := r.FormValue("stime")
	booker.STime, err = time.Parse("15", RawSTime)
	if err != nil {
		panic(err)
	}

	if r.FormValue("halfHourSelected") == "true" {
		booker.STime = booker.STime.Add(30 * time.Minute)
	}

	booker.TOD = r.FormValue("tod")
	booker.Month = r.FormValue("month")
	booker.Day = r.FormValue("day")

	setupBooker(booker)

	booker.LaneId = booker.LaneToTrin[booker.Lane]

	now := time.Now()
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), 4, 45, 20, 0, now.Location())
	//targetTime := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 20, 0, now.Location())

	if now.After(targetTime) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	duration := targetTime.Sub(now)

	go func() {
		fmt.Println("SLEEPING for ", duration)

		//time.Sleep(duration)

		startTime := time.Now()

		fmt.Println("STARTING AT ", startTime.Format("15:04:05"))
		booker.PerformLogin()
		booker.PrepareBooking()
		booker.CompleteBooking()

		duration := time.Since(startTime)
		fmt.Println("Booking Completed in ", duration, " seconds")
	}()

	http.ServeFile(w, r, "initiate.html")
}
