package main

import (
	"log"
	elastic "gopkg.in/olivere/elastic.v5"
)

func main() {
	_, err := elastic.NewClient(elastic.SetSniff(false),elastic.SetURL("http://172.20.253.99:9200"))
	if err != nil {
		log.Fatalf("Connect failed: %v", err)
	}
	log.Print("Connected")
}