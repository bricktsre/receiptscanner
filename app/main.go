package main

import (
	"fmt"
	"net/http"
	"log"
	"os"
	"io"
	"context"
	"errors"
	"path"
	"strconv"
	"../../receiptscanner"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"cloud.google.com/go/storage"
	//"google.golang.org/appengine"
	//"google.golang.org/appengine/datastore"
	//"google.golang.org/appengine/log"
	//vision "cloud.google.com/go/vision/apiv1"
	vision "google.golang.org/genproto/googleapis/cloud/vision/v1"
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
	r.Methods("GET").Path("/result/{id:[0-9]+}").Handler(appHandler(resultHandler))

	r.Methods("POST").Path("/process_image").Handler(appHandler(imageProcessingHandler))
	r.Methods("POST").Path("/update_receipt").Handler(appHandler(receiptUpdateHandler))

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
	receipt, err := receiptFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}
	return resultTmpl.Execute(w,r,receipt)
}

func receiptFromRequest(r *http.Request) (*receiptscanner.Receipt, error) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad receipt id: %v", err)
	}
	receipt, err := receiptscanner.DB.GetReceipt(id)
	if err != nil {
		return nil, fmt.Errorf("could not get receipt from datastore: %v", err)
	}
	return receipt, nil
}

func receiptUpdateHandler(r *http.Request) *appError {
	receipt, err := receiptscanner.DB.GetReceipt(r.FormFile("id"))
	if err != nil {
		return appErrorf(err, "could not get receipt from database: %v", err)
	}
	receipt.Total = r.FormFile("total")
	receipt.Subtotal = r.FormFile("subtotal")
	receipt.Tax = r.FormFile("tax")
	receipt.Total = r.FormFile("total")
	id, err := receiptscanner.DB.UpdateReceipt(receipt) 
	if err != nil {
		return appErrorf(err, "could not update receipt in database: %v",err)
	}
	receipt.ID = id
	return nil
}

func imageProcessingHandler(w http.ResponseWriter, r *http.Request) *appError{
	receipt, err := receiptFromForm(r)
	if err != nil {
		return appErrorf(err, "could not parse receipt from form: %v", err)
	}
	id, err := receiptscanner.DB.AddReceipt(receipt)
	if err != nil {
		return appErrorf(err, "could not add reciept to database: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/result/%d", id), http.StatusFound)
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
	
	txt_annotation, err := getTextFromImage(receipt.URL)
	if err != nil {
		return nil, fmt.Errorf("could not get text from image: %v", err)
	}
	
	total, err := txt_annotation.GetTotal()
	if err != nil {
		return nil, fmt.Errorf("Could not get total: %v", err)
	}
	receipt.Total = total
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
func getTextFromImage(file string) (*TextAnnotation, error){
	ctx:= context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	
	image := vision.NewImageFromURI(file)
	annotations, err := client.DetectDocumentTexts(ctx, image, nil)
	if err != nil {
		return nil, err
	}

	return &TextAnnotation{Annotations: annotations}, nil
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
