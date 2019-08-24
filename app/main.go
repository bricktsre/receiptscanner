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
	"math"
	"time"
	"github.com/bricktsre/receiptscanner"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"cloud.google.com/go/storage"
	//"google.golang.org/appengine"
	//"google.golang.org/appengine/datastore"
	//"google.golang.org/appengine/log"
	vision "cloud.google.com/go/vision/apiv1"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
	uuid "github.com/gofrs/uuid"
)

var (
	uploadTmpl = parseTemplate("upload.html")
	editTmpl = parseTemplate("receipt_edit.html")
	listTmpl = parseTemplate("receipt_list.html")
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
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("templates/static/"))))

	r.Methods("GET").Path("/upload").Handler(appHandler(uploadHandler))
	r.Methods("GET").Path("/edit/{id:[0-9]+}").Handler(appHandler(editHandler))
	r.Methods("GET").Path("/list").Handler(appHandler(listHandler))

	r.Methods("POST").Path("/process_image").Handler(appHandler(imageProcessingHandler))
	r.Methods("POST").Path("/update_receipt").Handler(appHandler(receiptUpdateHandler))
	r.Methods("POST").Path("/delete/{id:[0-9]+}").Handler(appHandler(deleteHandler))

	// The following handlers are defined in auth.go and used in the
	r.Methods("GET").Path("/login").
		Handler(appHandler(loginHandler))
	r.Methods("POST").Path("/logout").
		Handler(appHandler(logoutHandler))
	r.Methods("GET").Path("/oauth2callback").
		Handler(appHandler(oauthCallbackHandler))
	
	r.Methods("GET").Path("/_ah/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) *appError{
	return uploadTmpl.Execute(w,r,nil)
}

func editHandler(w http.ResponseWriter, r *http.Request) *appError{
	receipt, err := receiptFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}
	return editTmpl.Execute(w,r,receipt)
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

func listHandler(w http.ResponseWriter, r *http.Request) *appError {
	user := profileFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/list", http.StatusFound)
	}

	receipts, err := receiptscanner.DB.ListReceiptsByUser(user.ID)
	if err != nil {
		return appErrorf(err, "could not list receipts: %v", err)
	}

	return listTmpl.Execute(w, r, receipts)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) *appError {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return appErrorf(err, "bad receipt id: %v", err)
	}
	if err = receiptscanner.DB.DeleteReceipt(id); err != nil {
		return appErrorf(err, "could not delete book: %v", err)
	}
	http.Redirect(w, r, "/list", http.StatusFound)
	return nil
}

func receiptUpdateHandler(w http.ResponseWriter, r *http.Request) *appError {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		return appErrorf(err, "could not parse float from form stringg: %v", err)
	}
	receipt,err := receiptscanner.DB.GetReceipt(id)
	if err != nil {
		return appErrorf(err, "could not get receipt from database: %v", err)
	}
	
	receipt.Business = r.FormValue("business")
	receipt.UserID = r.FormValue("userid")

	temp_date, err := time.Parse("01/02/2006", r.FormValue("date"))
	if err != nil {
		return appErrorf(err, "could not parse date from form string: %v", err)
	}
	receipt.SetDate(temp_date)

	receipt.Subtotal, err = strconv.ParseFloat(r.FormValue("subtotal"),64)
	if err != nil {
		return appErrorf(err, "could not parse float from form string: %v", err)
	}
	
	receipt.Tax, err = strconv.ParseFloat(r.FormValue("tax"),64)
	if err != nil {
		return appErrorf(err, "could not parse float from form string: %v", err)
	}
	
	receipt.Total, err = strconv.ParseFloat(r.FormValue("total"),64)
	if err != nil {
		return appErrorf(err, "could not parse float from form string: %v", err)
	}
	
	if err := receiptscanner.DB.UpdateReceipt(receipt); err != nil {
		return appErrorf(err, "could not update receipt in database: %v",err)
	}	
	http.Redirect(w, r, fmt.Sprint("/list"), http.StatusFound)
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
	http.Redirect(w, r, fmt.Sprintf("/edit/%d", id), http.StatusFound)
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

	subtotal, err := txt_annotation.GetSubTotal()
	if err != nil {
		return nil, fmt.Errorf("Could not get subtotal: %v", err)
	}
	receipt.Subtotal = subtotal
	if (subtotal != 0 && total != 0){
		receipt.Tax = math.Round((receipt.Total - receipt.Subtotal)*100)/100
	}
	
	date, err := txt_annotation.GetDate()
	if err != nil {
		return nil, fmt.Errorf("Cloud not get date: %v:", err)
	}
	receipt.SetDate(date)
	
	user := profileFromSession(r)
	if user != nil {
		receipt.UserID = user.ID
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
func getTextFromImage(file string) (*TextAnnotation, error){
	ctx:= context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	
	image := vision.NewImageFromURI(file)
	image_context := &pb.ImageContext{LanguageHints: []string{"en"}}
	annotations, err := client.DetectDocumentText(ctx, image, image_context)
	if err != nil {
		return nil, err
	}

	return &TextAnnotation{Annotation: annotations}, nil
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
