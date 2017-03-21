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

var client *elastic.Client
var err error

var (
	reqQttyStatus200 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:  "qtty_reqs_counter_status_200",
		Help: "Total 200 Requests Count",
	})
	reqQttyStatus400 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:  "qtty_reqs_counter_status_400",
		Help: "Total 400 Requests Count",
	})
	reqQttyStatus500 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:  "qtty_reqs_counter_status_500",
		Help: "Total 500 Requests Count",
	})
)

func init() {
	prometheus.MustRegister(reqQttyStatus200)
	prometheus.MustRegister(reqQttyStatus400)
	prometheus.MustRegister(reqQttyStatus500)
}

func main() {

	sHost = flag.String("host", "0.0.0.0", "Server Host")
	sPort = flag.String("port", "8300", "Server Port")
	eHost = flag.String("ehost", "127.0.0.1", "ElasticSearch Host")
	ePort = flag.String("eport", "9200", "ElasticSearch Port")

	flag.Parse()

	reqQttyStatus200.Set(0)
	reqQttyStatus400.Set(0)
	reqQttyStatus500.Set(0)

	client, err = elastic.NewClient(elastic.SetSniff(false), elastic.SetURL("http://" + *eHost + ":" + *ePort))
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/metrics", HandleProme)
	router.HandleFunc("/metrics/all", GetMetricsEndpoint).Methods("GET")
	router.HandleFunc("/metrics/create", CreateMetricEndpoint).Methods("POST")
	fmt.Println("Server Running: ", *sHost + ":" + *sPort)
	log.Fatal(http.ListenAndServe(*sHost + ":" + *sPort, router))
}

func HandleProme(w http.ResponseWriter, req *http.Request) {

	ctx := context.Background()

	exists, err := client.IndexExists("metrics").Do(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		_, err = client.CreateIndex("metrics").Do(ctx)
		if err != nil {
			panic(err)
		}
	}

	seconds := 10
	current := time.Now().Local().Add(time.Duration(-10) * time.Second)
	next := time.Now().Local().Add(time.Duration(seconds) * time.Second)

	fmt.Println("Current: " + current.Format("20060102150405"))
	fmt.Println("Next:    " + next.Format("20060102150405"))

	query := `
	{
	    "bool": {
		"must": [
		    {
		        "range": {
			    "Datetime": {
			       "format": "yyyyMMddHHmmss",
			       "from": "%s",
			       "to": "%s"
			    }
		        }
		    },
		    {
		        "range": {
			    "ResponseHeaders.status": {
			       "from": %d,
			       "to": %d
			   }
		        }
		    }
		]
	    }
  	 }`

	rawQuery200 := elastic.NewRawStringQuery(fmt.Sprintf(query, current.Format("20060102150405"), next.Format("20060102150405"), 200, 299))
	searchResult200, err := client.Search().
		Query(rawQuery200).
		Index("metrics").
		Pretty(true).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	if searchResult200.Hits.TotalHits > 0 {
		reqQttyStatus200.Set(float64(searchResult200.Hits.TotalHits))
		fmt.Printf("Total 200: %d", searchResult200.Hits.TotalHits)
		fmt.Println()
	} else {
		reqQttyStatus200.Set(0)
	}

	rawQuery400 := elastic.NewRawStringQuery(fmt.Sprintf(query, current.Format("20060102150405"), next.Format("20060102150405"), 400, 500))
	searchResult400, err := client.Search().
		Query(rawQuery400).
		Index("metrics").
		Pretty(true).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	if searchResult400.Hits.TotalHits > 0 {
		reqQttyStatus400.Set(float64(searchResult400.Hits.TotalHits))
		fmt.Printf("Total 400: %d", searchResult400.Hits.TotalHits)
		fmt.Println()
	} else {
		reqQttyStatus400.Set(0)
	}

	rawQuery500 := elastic.NewRawStringQuery(fmt.Sprintf(query, current.Format("20060102150405"), next.Format("20060102150405"), 500, 999))
	searchResult500, err := client.Search().
		Query(rawQuery500).
		Index("metrics").
		Pretty(true).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	if searchResult500.Hits.TotalHits > 0 {
		reqQttyStatus500.Set(float64(searchResult500.Hits.TotalHits))
		fmt.Printf("Total 500: %d", searchResult500.Hits.TotalHits)
		fmt.Println()
	} else {
		reqQttyStatus500.Set(0)
	}

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
	ctx := context.Background()

	exists, err := client.IndexExists("metrics").Do(ctx)
	if err != nil {
		panic(err)
	}
	if !exists {
		_, err = client.CreateIndex("metrics").Do(ctx)
		if err != nil {
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
		panic(err)
	}
}
