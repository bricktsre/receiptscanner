package main

import (
	"errors"
	"regexp"
	"fmt"
	"log"

	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
	//"cloud.google.com/go/vision/apiv1"
)

type TextAnnotation struct {
	Annotations []*pb.EntityAnnotation
	TotalIndex int 
}

func(t *TextAnnotation) GetTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)(t?otal|to?tal|tot?al|tota?l|total?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	b, err := getTextBoundingPoly(t, regex)
	if err != nil {
		return 0.0, err
	}
	log.Print(fmt.Sprintf("%v",b))
	return 0.0, nil
}

func getTextBoundingPoly(t *TextAnnotation, rx *regexp.Regexp) (*pb.BoundingPoly, error) {
	for i,v := range t.Annotations {
		log.Print(fmt.Sprintf("%v",v))
		if tmp := rx.FindString(v.Description); tmp != "" && len(v.Description) < 10 {
			t.TotalIndex = i
			return v.BoundingPoly, nil
		}
	}
	return nil, errors.New("Could not find total in given text")
}
