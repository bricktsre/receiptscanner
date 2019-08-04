package receiptscanner

import (
	"context"
	//"errors"
	"log"
	//"os"
	
	"cloud.google.com/go/storage"
	"cloud.google.com/go/datastore"
)

var (
	DB ReceiptDatabase	

	StorageBucket *storage.BucketHandle
	StorageBucketName string
)
	

func init() {
	var err error	

	DB, err = configureDatastoreDB("receiptscanner-0")
	if err != nil {
		log.Fatal(err)
	}

	StorageBucketName = "receiptscanner-0.appspot.com"
	StorageBucket, err = configureStorage(StorageBucketName)
	if err != nil {
		log.Fatal(err)
	}
}

func configureDatastoreDB(projectID string) (ReceiptDatabase, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return newDatastoreDB(client)
}

func configureStorage(bucketID string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucketID), nil
}
