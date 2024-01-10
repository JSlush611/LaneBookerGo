package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	//http.Handle("/", http.HandlerFunc(HomePage))
	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/saveBooking", http.HandlerFunc(SaveBookingHandler))
	http.Handle("/book", http.HandlerFunc(Book))
	http.Handle("/processBookings", http.HandlerFunc(TriggerSendBookRequestsHandler))

	initFirestore()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func SaveBookingHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	booker, err := initializeBookerFromRequest(r)
	if err != nil {
		log.Printf("Error initializing booker: %v", err)
		http.Error(w, "Error processing booking request", http.StatusBadRequest)
		return
	}

	err = StoreLaneBooking(r.Context(), booker)
	if err != nil {
		log.Printf("Error saving booking to database: %v", err)
		http.Error(w, "Error saving booking to database", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, "static/scheduled.html")
}

func initializeBookerFromRequest(r *http.Request) (*Booker, error) {
	booker := &Booker{}
	var err error

	booker.Username = r.FormValue("email")
	booker.Password = r.FormValue("password")

	booker.Lane, err = strconv.Atoi(r.FormValue("lane"))
	if err != nil {
		return nil, fmt.Errorf("invalid lane number: %v", err)
	}

	booker.RawSTime = r.FormValue("stime")

	booker.STime, err = time.Parse("15", booker.RawSTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %v", err)
	}

	if r.FormValue("halfHourSelected") == "true" {
		booker.halfHourSelected = true
		booker.STime = booker.STime.Add(30 * time.Minute)
	}

	booker.TOD = r.FormValue("tod")
	booker.Month = r.FormValue("month")
	booker.Day = r.FormValue("day")

	return booker, nil
}

func calculateBookingTime(booker *Booker) (time.Duration, error) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return 0, err
	}

	now := time.Now().In(loc)
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), 20, 45, 0, 0, loc)
	var duration time.Duration

	if now.Before(targetTime) {
		duration = targetTime.Sub(now)
	} else {
		targetTime = targetTime.Add(24*time.Hour - 1*time.Nanosecond)
		duration = targetTime.Sub(now)
	}

	return duration, nil
}

func conductBooking(booker *Booker, duration time.Duration) {
	go func(dur time.Duration) {
		log.Println("NOW SLEEPING FOR ", dur)

		//time.Sleep(dur)

		startTime := time.Now()
		log.Println("NOW STARTING AT ", startTime.Format("15:04:05"))

		booker.PerformLogin()
		booker.PrepareBooking()
		booker.CompleteBooking()

		elapsed := time.Since(startTime)
		log.Println("Booking Completed in ", elapsed, " seconds")
	}(duration)
}

func Book(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	booker, err := initializeBookerFromRequest(r)
	if err != nil {
		log.Printf("Error initializing booker: %v", err)
		http.Error(w, "Error processing booking request", http.StatusBadRequest)

		return
	}

	setupBooker(booker)

	booker.LaneId = booker.LaneToTrin[booker.Lane]

	duration, err := calculateBookingTime(booker)
	if err != nil {
		log.Fatal(err)
	}

	conductBooking(booker, duration)

	http.ServeFile(w, r, "static/scheduled.html")
}
