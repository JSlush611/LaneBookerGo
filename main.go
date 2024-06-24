package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	username = "rip-mindbody"
	password = "1_benfolds"
)

// Middleware for HTTP Basic Authentication
func basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow App Engine cron service
		if r.Header.Get("X-AppEngine-Cron") == "true" {
			next.ServeHTTP(w, r)

			return
		}

		user, pass, ok := r.BasicAuth()

		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	http.Handle("/", basicAuthMiddleware(http.HandlerFunc(HomePage)))
	http.Handle("/saveBooking", basicAuthMiddleware(http.HandlerFunc(SaveBookingHandler)))
	http.Handle("/processBookings", basicAuthMiddleware(http.HandlerFunc(TriggerSendBookRequestsHandler)))
	http.Handle("/book", basicAuthMiddleware(http.HandlerFunc(Book)))

	// Routes without Basic Auth Middleware
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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

func conductBooking(booker *Booker) {
	go func() {
		startTime := time.Now()
		log.Println("Booking started at", startTime.Format("15:04:05"))

		err := booker.PerformLogin()
		if err != nil {
			log.Printf("Error during login: %v", err)

			return
		}

		err = booker.PrepareBooking()
		if err != nil {
			log.Printf("Error during booking preparation: %v", err)

			return
		}

		err = booker.CompleteBooking()
		if err != nil {
			log.Printf("Error during booking completion: %v", err)

			return
		}

		elapsed := time.Since(startTime)
		log.Println("Booking attempted in", elapsed, "seconds")
	}()
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

	conductBooking(booker)

	http.ServeFile(w, r, "static/scheduled.html")
}
