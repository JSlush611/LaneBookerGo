package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", http.HandlerFunc(HomePage))
	http.Handle("/book", http.HandlerFunc(Progress))
	http.Handle("/smileyface.png", http.HandlerFunc(serveImage))

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

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), 19, 40, 0, 0, location)

	if now.After(targetTime) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	duration := targetTime.Sub(now)

	go func() {
		log.Println("NOW SLEEPING FOR ", duration)

		time.Sleep(duration)

		startTime := time.Now()

		log.Println("NOW STARTING AT ", startTime.Format("15:04:05"))
		booker.PerformLogin()
		booker.PrepareBooking()
		booker.CompleteBooking()

		duration := time.Since(startTime)
		log.Println("Booking Completed in ", duration, " seconds")
	}()

	http.ServeFile(w, r, "static/initiate.html")
}
