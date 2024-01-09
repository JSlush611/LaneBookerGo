package main

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
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

func StoreLaneBooking(ctx context.Context, data *BookingData) (*firestore.DocumentRef, error) {
	docRef, _, err := firestoreClient.Collection("laneJobs").Add(ctx, data)
	return docRef, err
}

func RetrieveLaneBooking(ctx context.Context, id string) (*Booker, error) {
	docRef := firestoreClient.Collection("laneJobs").Doc(id)
	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var booking Booker
	err = docSnapshot.DataTo(&booking)
	return &booking, err
}
