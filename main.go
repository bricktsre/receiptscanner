package main

import (
	"fmt"
	"html/template"
	"net/http"
	"log"
	"os"
	"io"
	"encoding/json"
	"context"
	"errors"
	"path"
	"strconv"

	"github.com/gorilla/handles"
	"github.com/gorilla/mux"
	
	"google.golang.org/appengine"
	"google.golang.org/appenginge/datastore"
	"google.golang.org/appengine/log"
	vision "cloud.google.com/go/vision/apiv1"
)

type Reciept struct {
	URL string
	ID int
	Business string
	SubTotal float64
	Tax float64
	Total float64
}

func main() {
	port := os.Getenv("Port")
	if port == "" {
		port = "8080"
	}
	registerHandlers()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func registerHandlers() {
	r := mux.Newrouter()

	r.Methods("GET").Path("/").Handler(appHandler(indexHandler))
	r.Methods("POST").Path("/process_image").Handler(appHandler(imageProcessingHandler))

	r.Methods("GET").Path("/_ah/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
}

func imageProcessingHandler(w http.ResponseWriter, r *http.Request) {

}

func getTextFromImage(file string) ([]string, error{
	ctx:= context.Backgrount()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	
	image := vision.NewImageFromURI(file)
	annotations, err := client.DetectTexts(ctx, image, nil, 10)
	if err != nil {
		return err
	}
		
	var text []string
	for _, annotation := range annotations {
		text = append(text, annotation.Description)
	}
	return text, nil
}
