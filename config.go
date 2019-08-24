package receiptscanner

import (
	"context"
	//"errors"
	"log"
	"os"
	
	"cloud.google.com/go/storage"
	"cloud.google.com/go/datastore"

	"github.com/gorilla/sessions"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	DB ReceiptDatabase	
	OAuthConfig *oauth2.Config	

	StorageBucket *storage.BucketHandle
	StorageBucketName string

	SessionStore sessions.Store
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

	OAuthConfig = configureOAuthClient("1039960761298-5qu3sukbi9qj1nc3mormicj2dqjjc4ub.apps.googleusercontent.com", "TCStpnJA-_Pzw6PghCkZr9AV")

	cookieStore := sessions.NewCookieStore([]byte("hard-2-guess-string5"))
	cookieStore.Options = &sessions.Options{
		HttpOnly: true,
	}
	SessionStore = cookieStore
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

func configureOAuthClient(clientID, clientSecret string) *oauth2.Config {
	redirectURL := os.Getenv("OAUTH2_CALLBACK")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/oauth2callback"
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}
