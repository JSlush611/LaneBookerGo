package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var firestoreClient *firestore.Client

func initFirestore() {
	ctx := context.Background()

	sa := option.WithCredentialsFile("mindbody-go-07f79bd519ea.json")
	client, err := firestore.NewClient(ctx, "mindbody-go", sa)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	firestoreClient = client
}

func StoreLaneBooking(ctx context.Context, booker *Booker) error {
	if firestoreClient == nil {
		return fmt.Errorf("firestore client is not initialized")
	}

	data := map[string]interface{}{
		"username":         booker.Username,
		"password":         booker.Password,
		"lane":             booker.Lane,
		"rawSTime":         booker.RawSTime,
		"day":              booker.Day,
		"month":            booker.Month,
		"tod":              booker.TOD,
		"halfHourSelected": booker.halfHourSelected,
	}

	_, _, err := firestoreClient.Collection("laneJobs").Add(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to add form data to Firestore: %v", err)
	}

	return nil
}

func SendBookRequests(ctx context.Context) error {
	if firestoreClient == nil {
		return fmt.Errorf("firestore client is not initialized")
	}

	iter := firestoreClient.Collection("laneJobs").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return fmt.Errorf("failed to iterate documents: %v", err)
		}

		formData := url.Values{}
		formData.Set("email", doc.Data()["username"].(string))
		formData.Set("password", doc.Data()["password"].(string))
		formData.Set("lane", fmt.Sprint(doc.Data()["lane"]))
		formData.Set("stime", doc.Data()["rawSTime"].(string))
		formData.Set("tod", doc.Data()["tod"].(string))
		formData.Set("month", doc.Data()["month"].(string))
		formData.Set("day", doc.Data()["day"].(string))

		if doc.Data()["halfHourSelected"].(bool) {
			formData.Set("halfHourSelected", "true")
		} else {
			formData.Set("halfHourSelected", "false")
		}

		// https://mindbody-go.uc.r.appspot.com/book
		// http://localhost:8080/book
		fmt.Println("SENDING REQUEST WITH ", formData.Encode())
		req, err := http.NewRequest("POST", "https://mindbody-go.uc.r.appspot.com/book", strings.NewReader(formData.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()
	}

	return nil
}

func TriggerSendBookRequestsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	go func() {
		err := SendBookRequests(ctx)
		if err != nil {
			log.Printf("Error sending book requests: %v", err)
		}
	}()
	fmt.Fprintln(w, "Send book requests initiated")
}
