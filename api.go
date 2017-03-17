package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	elastic "gopkg.in/olivere/elastic.v5"
	"context"
)

type Tweet struct {
	User    string                `json:"user"`
	Message string                `json:"message"`
}

type Metric struct {
	ID              string   `json:"id,omitempty"`
	Url             string   `json:"url,omitempty"`
	Origin          string   `json:"origin,omitempty"`
	Time            string   `json:"time,omitempty"`
	RequestHeaders  string   `json:"requestheaders,omitempty"`
	ResponseHeaders string   `json:"responseheaders,omitempty"`
}

var metrics []Metric

func GetMetricsEndpoint(w http.ResponseWriter, req *http.Request) {
	log.Print("New Request" + req.URL.String())

	json.NewEncoder(w).Encode(metrics)
}

func CreateMetricEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	var metric Metric
	_ = json.NewDecoder(req.Body).Decode(&metric)
	metric.ID = params["id"]
	metrics = append(metrics, metric)
	json.NewEncoder(w).Encode(metrics)
}

func main() {

	// Create a context
	ctx := context.Background()

	// Create a client
	client, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL("http://172.20.253.99:9200"))
	if err != nil {
		// Handle error
		panic(err)
	}

	// Create an index
	_, err = client.CreateIndex("twitter").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	tweet := Tweet{User: "olivere", Message: "Take Five"}

	_, err = client.Index().
		Index("twitter").
		Type("tweet").
		Id("5").
		BodyJson(tweet).
		Refresh("true").
		Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}

	router := mux.NewRouter()
	metrics = append(metrics, Metric{ID: "1", Url: "Nic", Origin: "Raboy", Time: "1231", RequestHeaders: "psaodufg", ResponseHeaders: "asdfasdfadsfggfad"})
	metrics = append(metrics, Metric{ID: "2", Url: "Nic", Origin: "Raboy", Time: "1231", RequestHeaders: "psaodufg", ResponseHeaders: "asdfasdfadsfggfad"})
	router.HandleFunc("/metrics", GetMetricsEndpoint).Methods("GET")
	router.HandleFunc("/metrics", CreateMetricEndpoint).Methods("POST")
	log.Fatal(http.ListenAndServe(":12345", router))
}