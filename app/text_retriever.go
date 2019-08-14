package main

import (
	"errors"
	"regexp"
	"fmt"
	"log"
	"strconv"
	"strings"

	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
	//vision "cloud.google.com/go/vision/apiv1"
)

type TextAnnotation struct {
	Annotation *pb.TextAnnotation
}

func(t *TextAnnotation) GetTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)\A[^a-z]*(t?otal|to?tal|tot?al|tota?l|total?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	total, err := getNumberFromAnnotation(t, regex)
	if err != nil {
		log.Print(fmt.Sprintf("Could not pull number asscioated with Total from text"))
		return 0.0, err
	}
	return total, nil
}

func(t *TextAnnotation) GetSubTotal() (float64, error) {
	regex,err := regexp.Compile(`(?i)(s?ubtotal|su?btotal|sub?total|subt?otal|subto?tal|subtot?al|subtota?l|subtotal?)`)
	if err != nil {
		return 0.0, fmt.Errorf("Could not generate regular expression: %v", err)
	}
	
	subtotal, err := getNumberFromAnnotation(t, regex)
	if err != nil {
		log.Print(fmt.Sprintf("Could not pull number asscioated with Subtotal from text"))
		return 0.0, nil
	}
	return subtotal, nil
}

func getNumberFromAnnotation(t *TextAnnotation, rx *regexp.Regexp) (float64, error) {
	pages := t.Annotation.GetPages()
	blocks := pages[0].GetBlocks()
	for i:=(len(blocks)-1); i >= 0; i-- {
		for _, paragraph := range blocks[i].GetParagraphs() {
			var para_string strings.Builder
			for _, word := range paragraph.GetWords() {
				for _, symbol := range word.GetSymbols(){
					_, err:= para_string.WriteString(symbol.GetText())
					if err != nil {
						return 0.0, errors.New("Could not add symbol to string builder")
					}
				}
			}
			if index := rx.FindString(para_string.String()); index != ""{
				floatrx, err := regexp.Compile(`\d*\W\d*`)
				if err != nil {
					return 0.0, errors.New("Could not generate float regular expression")
				}
				float_string := floatrx.FindString(para_string.String())
				return strconv.ParseFloat(float_string,64)
			}
		}
	}
	return 0.0, errors.New("Could not find regular expression in given text")
}
