package main

import (
	"fmt"
	"net/http"
	"log"
	"os"
	"io"
	"encoding/json"
	"context"
	"errors"
	"path"
	//"strconv"
	"../../receiptscanner"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"cloud.google.com/go/storage"
	//"google.golang.org/appengine"
	//"google.golang.org/appengine/datastore"
	//"google.golang.org/appengine/log"
	vision "cloud.google.com/go/vision/apiv1"
	uuid "github.com/gofrs/uuid"
)

var (
	uploadTmpl = parseTemplate("upload.html")
	resultTmpl = parseTemplate("result.html")
)


func main() {
	port := os.Getenv("Port")
	if port == "" {
		port = "8080"
	}
	registerHandlers()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func registerHandlers() {
	r := mux.NewRouter()

	r.Handle("/", http.RedirectHandler("/upload", http.StatusFound))

	r.Methods("GET").Path("/upload").Handler(appHandler(uploadHandler))
	r.Methods("GET").Path("/result").Handler(appHandler(resultHandler))

	r.Methods("POST").Path("/process_image").Handler(appHandler(imageProcessingHandler))

	r.Methods("GET").Path("/_ah/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) *appError{
	return uploadTmpl.Execute(w,r,nil)
}

func resultHandler(w http.ResponseWriter, r *http.Request) *appError{
	var receipt receiptscanner.Receipt
	if r.Body == nil {
		return appErrorf(errors.New("Receipt Data was not sent"),"",nil)
	}
	
	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {
		return appErrorf(errors.New("Receipt Data could not be parsed from http request"),"",nil)
	}
	return resultTmpl.Execute(w,r,receipt)
}

func imageProcessingHandler(w http.ResponseWriter, r *http.Request) *appError{
	receipt, err := receiptFromForm(r)
	if err != nil {
		return appErrorf(err, "could not parse book from form: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/result", receipt), http.StatusFound)
	return nil	
}

// creates a receipt object from the uploaded image file
func receiptFromForm(r *http.Request) (*receiptscanner.Receipt, error) {
	imageURL, err := uploadFileFromForm(r)
	if err != nil {
		return nil, fmt.Errorf("could not upload file: %v", err)
	}
	var receipt receiptscanner.Receipt
	receipt.URL = imageURL
	
	text, err := getTextFromImage(receipt.URL)
	if err != nil {
		return nil, fmt.Errorf("could not get text from image: %v", err)
	}
	for i,v := range text {
		log.Print(fmt.Sprintf("%v: %v", i, v)) 
	}
	return &receipt, nil
}

// uploadFileFromForm uploads a file if it's present in the "image" form field.
func uploadFileFromForm(r *http.Request) (url string, err error) {
        f, fh, err := r.FormFile("image")
        if err == http.ErrMissingFile {
                return "", nil
        }
        if err != nil {
                return "", err
        }

        if receiptscanner.StorageBucket == nil {
                return "", errors.New("storage bucket is missing - check config.go")
        }

        // random filename, retaining existing extension.
        name := uuid.Must(uuid.NewV4()).String() + path.Ext(fh.Filename)

        ctx := context.Background()
        w := receiptscanner.StorageBucket.Object(name).NewWriter(ctx)

        // Warning: storage.AllUsers gives public read access to anyone.
        w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
        w.ContentType = fh.Header.Get("Content-Type")

        // Entries are immutable, be aggressive about caching (1 day).
        w.CacheControl = "public, max-age=86400"

        if _, err := io.Copy(w, f); err != nil {
                return "", err
        }
        if err := w.Close(); err != nil {
                return "", err
        }

        const publicURL = "https://storage.googleapis.com/%s/%s"
        return fmt.Sprintf(publicURL, receiptscanner.StorageBucketName, name), nil
}

//Uses google ML Vision API to detect text in the uploaded image
func getTextFromImage(file string) ([]string, error){
	ctx:= context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	
	image := vision.NewImageFromURI(file)
	annotations, err := client.DetectTexts(ctx, image, nil, 50)
	if err != nil {
		return nil, err
	}
		
	var text []string
	for _, annotation := range annotations {
		text = append(text, annotation.Description)
	}
	return text, nil
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
	Error   error
	Message string
	Code    int
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error.
		log.Printf("Handler error: status code: %d, message: %s, underlying err: %#v",
			e.Code, e.Message, e.Error)

		http.Error(w, e.Message, e.Code)
	}
}

func appErrorf(err error, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    500,
	}
}
