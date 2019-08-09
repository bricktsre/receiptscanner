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
	Annotations []*pb.TextAnnotation
	SubtotalIndex int
	TotalIndex int 
}

func(t *TextAnnotation) GetTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)/A/d*(t?otal|to?tal|tot?al|tota?l|total?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	total, err := getNumberFromAnnotation(t, regex)
	if err != nil {
		return 0.0, err
	}
	log.Print(fmt.Sprintf("%v",total))
	return total, nil
}

func(t *TextAnnotation) GetSubTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)/A/d*(s?ubtotal|su?btotal|sub?total|subt?otal|subto?tal|subtot?al|subtota?l|subtotal?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	subtotal, err := getNumberFromAnnotation(t, regex)
	if err != nil {
		return 0.0, err
	}
	log.Print(fmt.Sprintf("%v", subtotal))
	return subtotal, nil
}

func getNumberFromAnnotation(t *TextAnnotation, rx *regexp.Regexp) (float64, error) {
	page  := t.GetPages()
	blocks := page.GetBlocks
	for i:=(len(blocks)-1); i >= 0; i-- {
		paragraph := blocks[i].GetParagraphs()
		para_string := paragraph.String()
		if index := rx.FindIndex(para_string); index != -1{
			floatrx, err := regexp.Compile(`/d*/D/d*`)
			if err != nil {
				return 0.0, errors.New("Could not generate float regular expression")
			}
			float_string := floatrx.FindString(para_string)
			return 0.0, nil
		}
	}
	
	return 0.0, errors.New("Could not find regular expression in given text")
}
