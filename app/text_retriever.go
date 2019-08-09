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
	SubtotalIndex int
	TotalIndex int 
}

func(t *TextAnnotation) GetTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)/A/d*(t?otal|to?tal|tot?al|tota?l|total?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	b, temp, err := getTextBoundingPoly(t, regex)
	t.TotalIndex = temp
	if err != nil {
		return 0.0, err
	}
	log.Print(fmt.Sprintf("%v",b))
	return 0.0, nil
}

func(t *TextAnnotation) GetSubTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)/A/d*(s?ubtotal|su?btotal|sub?total|subt?otal|subto?tal|subtot?al|subtota?l|subtotal?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	b, temp, err := getTextBoundingPoly(t, regex)
	t.SubtotalIndex = temp
	if err != nil {
		return 0.0, err
	}
	log.Print(fmt.Sprintf("%v",b))
	return 0.0, nil
}

func getTextBoundingPoly(t *TextAnnotation, rx *regexp.Regexp) (*pb.BoundingPoly, int, error) {
	for i,v := range t.Annotations {
		log.Print(fmt.Sprintf("%v",v))
		if tmp := rx.FindString(v.Description); tmp != "" && len(v.Description) < 10 {
			return v.BoundingPoly, i, nil
		}
	}
	return nil, 0, errors.New("Could not find total in given text")
}
