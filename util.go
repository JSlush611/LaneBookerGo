package main

import (
	"io"
	"net/http"
	"os"
	"time"
)

func setupBooker(booker *Booker) {
	booker.LaneToTrin = make(map[int]int)

	// OUTSIDE
	booker.LaneToTrin[1] = 100000301
	booker.LaneToTrin[2] = 100000302
	booker.LaneToTrin[3] = 100000303

	// INSIDE 30 minutes
	booker.LaneToTrin[91] = 100000212
	booker.LaneToTrin[92] = 100000213

	// INSIDE 60 minutes
	booker.LaneToTrin[93] = 100000237
	booker.LaneToTrin[94] = 100000215
	booker.LaneToTrin[95] = 100000216
	booker.LaneToTrin[96] = 100000217

	booker.STime2 = booker.STime.Add(30 * time.Minute)
	booker.ETime = booker.STime2
	booker.ETime2 = booker.STime2.Add(30 * time.Minute)

	booker.NewClient()
	booker.GetInitialCookies()
	booker.FormatLoginWebsite()

}

func serveImage(w http.ResponseWriter, r *http.Request) {
	imageFile, err := os.Open("ok.png")
	if err != nil {
		http.Error(w, "Failed to open image file", http.StatusInternalServerError)
		return
	}
	defer imageFile.Close()

	w.Header().Set("Content-Type", "image/png")

	_, err = io.Copy(w, imageFile)
	if err != nil {
		http.Error(w, "Failed to write image data", http.StatusInternalServerError)
		return
	}
}
