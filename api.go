package main

import (
	"encoding/json"
	"log"
	"net/http"
	"context"
	"fmt"
	"flag"
	"github.com/gorilla/mux"
	elastic "gopkg.in/olivere/elastic.v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

type Metric struct {
	ID              string   `json:"id,omitempty"`
	Url             string   `json:"url,omitempty"`
	Origin          string   `json:"origin,omitempty"`
	Time            string   `json:"time,omitempty"`
	RequestHeaders  RequestHeaders
	ResponseHeaders ResponseHeaders
	Datetime        string
}

type RequestHeaders struct {
	Accept string `json:"accept,omitempty"`
	Host   string `json:"host,omitempty"`
}

type ResponseHeaders struct {
	Status int `json:"status,omitempty"`
	Data   int `json:"date,omitempty"`
}

var metrics []Metric

var sHost *string
var sPort *string
var eHost *string
var ePort *string

var (
	reqQtty = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:  "qtty_reqs_counter",
		Help: "Total Requests Count",
	})
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(reqQtty)
}

func main() {

	sHost = flag.String("host", "0.0.0.0", "Server Host")
	sPort = flag.String("port", "8300", "Server Port")
	eHost = flag.String("ehost", "127.0.0.1", "ElasticSearch Host")
	ePort = flag.String("eport", "9200", "ElasticSearch Port")

	flag.Parse()

	// Create a context
	ctx := context.Background()

	// Create a client
	client, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL("http://" + *eHost + ":" + *ePort))
	if err != nil {
		// Handle error
		panic(err)
	}

	exists, err := client.IndexExists("metrics").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create an index
		_, err = client.CreateIndex("metrics").Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
	}
	// Search with a term query
	searchResult, err := client.Search().
		Index("metrics").// search in index "twitter"
		From(0).// take documents 0-9
		Pretty(true).// pretty print request and response JSON
		Do(ctx)             // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	// searchResult is of type SearchResult and returns hits, suggestions,
	// and all kinds of other information from Elasticsearch.
	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)

	// Here's how you iterate through results with full control over each step.
	if searchResult.Hits.TotalHits > 0 {
		fmt.Printf("Found a total of %d metrics\n", searchResult.Hits.TotalHits)

		// Iterate through results
		for _, hit := range searchResult.Hits.Hits {
			// hit.Index contains the name of the index

			// Deserialize hit.Source into a Tweet (could also be just a map[string]interface{}).
			var t Metric
			err := json.Unmarshal(*hit.Source, &t)
			if err != nil {
				// Deserialization failed
			}
			t.ID = hit.Id
			metrics = append(metrics, t)
		}
	} else {
		// No hits
		fmt.Print("Found no metric\n")
	}
	reqQtty.Set(0)

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.

	router := mux.NewRouter()
	router.HandleFunc("/metrics", HandleProme)
	router.HandleFunc("/metrics/all", GetMetricsEndpoint).Methods("GET")
	router.HandleFunc("/metrics/create", CreateMetricEndpoint).Methods("POST")
	fmt.Println("Server Running: ", *sHost + ":" + *sPort)
	log.Fatal(http.ListenAndServe(*sHost + ":" + *sPort, router))
}

func HandleProme(w http.ResponseWriter, req *http.Request) {

	fmt.Println("Teste")
	// Create a context
	ctx := context.Background()

	// Create a client
	client, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL("http://" + *eHost + ":" + *ePort))
	if err != nil {
		// Handle error
		panic(err)
	}

	exists, err := client.IndexExists("metrics").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create an index
		_, err = client.CreateIndex("metrics").Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
	}

	seconds := 10
	current := time.Now().Local().Add(time.Duration(-10) * time.Second)
	next := time.Now().Local().Add(time.Duration(seconds) * time.Second)

	fmt.Println("Current: " + current.Format("20060102150405"))
	fmt.Println("Next:    " + next.Format("20060102150405"))

	q := elastic.NewRangeQuery("Datetime").
		Format("yyyyMMddHHmmss").
		From(current.Format("20060102150405")).
		To(next.Format("20060102150405"))

	q = q.QueryName("my_query")

	// Search with a term query
	searchResult, err := client.Search().
		Query(q).
		Index("metrics").// search in index "twitter"
		From(0).// take documents 0-9
		Pretty(true).// pretty print request and response JSON
		Do(ctx)             // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	// searchResult is of type SearchResult and returns hits, suggestions,
	// and all kinds of other information from Elasticsearch.
	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)

	// Here's how you iterate through results with full control over each step.
	if searchResult.Hits.TotalHits > 0 {
		fmt.Printf("Found a total of %d metrics\n", searchResult.Hits.TotalHits)

		// Iterate through results
		for _, hit := range searchResult.Hits.Hits {
			// hit.Index contains the name of the index

			// Deserialize hit.Source into a Tweet (could also be just a map[string]interface{}).
			var t Metric
			err := json.Unmarshal(*hit.Source, &t)
			if err != nil {
				// Deserialization failed
			}
			t.ID = hit.Id
			metrics = append(metrics, t)
		}
	} else {
		// No hits
		fmt.Print("Found no metric\n")
	}
	reqQtty.Set(float64(searchResult.Hits.TotalHits))

	promhttp.Handler().ServeHTTP(w, req)

}

func GetMetricsEndpoint(w http.ResponseWriter, req *http.Request) {
	//log.Print("New Request" + req.URL.String())
	//json.NewEncoder(w).Encode(metrics)
}

func CreateMetricEndpoint(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Create Metric")
	params := mux.Vars(req)
	var metric Metric
	_ = json.NewDecoder(req.Body).Decode(&metric)
	current_time := time.Now().Local()
	metric.ID = params["id"]
	metric.Datetime = current_time.Format("20060102150405")
	metrics = append(metrics, metric)
	//json.NewEncoder(w).Encode(metrics)

	// Create a context
	ctx := context.Background()

	// Create a client
	client, err := elastic.NewClient(elastic.SetSniff(false), elastic.SetURL("http://" + *eHost + ":" + *ePort))
	if err != nil {
		// Handle error
		panic(err)
	}

	exists, err := client.IndexExists("metrics").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create an index
		_, err = client.CreateIndex("metrics").Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
	}
	_, err = client.Index().
		Index("metrics").
		Type("metric").
		BodyJson(metric).
		Refresh("true").
		Do(context.Background())

	if err != nil {
		// Handle error
		panic(err)
	}

	seconds := 10
	current := time.Now().Local().Add(time.Duration(-10) * time.Second)
	next := time.Now().Local().Add(time.Duration(seconds) * time.Second)

	fmt.Println("Current: " + current.Format("20060102150405"))
	fmt.Println("Next:    " + next.Format("20060102150405"))

	q := elastic.NewRangeQuery("Datetime").
		Format("yyyyMMddHHmmss").
		From(current.Format("20060102150405")).
		To(next.Format("20060102150405"))

	q = q.QueryName("my_query")

	// Search with a term query
	searchResult, err := client.Search().
		Query(q).
		Index("metrics").// search in index "twitter"
		From(0).// take documents 0-9
		Pretty(true).// pretty print request and response JSON
		Do(ctx)             // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	// searchResult is of type SearchResult and returns hits, suggestions,
	// and all kinds of other information from Elasticsearch.
	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)

	// Here's how you iterate through results with full control over each step.
	if searchResult.Hits.TotalHits > 0 {
		fmt.Printf("Found a total of %d metrics\n", searchResult.Hits.TotalHits)

		// Iterate through results
		for _, hit := range searchResult.Hits.Hits {
			// hit.Index contains the name of the index

			// Deserialize hit.Source into a Tweet (could also be just a map[string]interface{}).
			var t Metric
			err := json.Unmarshal(*hit.Source, &t)
			if err != nil {
				// Deserialization failed
			}
			t.ID = hit.Id
			metrics = append(metrics, t)
		}
	} else {
		// No hits
		fmt.Print("Found no metric\n")
	}
	//reqQtty.Inc()
	reqQtty.Set(float64(searchResult.Hits.TotalHits))
}
