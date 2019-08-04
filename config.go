package receiptscanner

import (
	"context"
	//"errors"
	"log"
	//"os"
	
	"cloud.google.com/go/storage"
)

var (
	StorageBucket *storage.BucketHandle
	StorageBucketName string
)
	

func init() {
	var err error	

	StorageBucketName = "receiptscanner-0.appspot.com"
	StorageBucket, err = configureStorage(StorageBucketName)
	if err != nil {
		log.Fatal(err)
	}
}

func configureStorage(bucketID string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucketID), nil
}
