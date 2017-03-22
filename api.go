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
	"strconv"
)

type Metric struct {
	ID              string   `json:"id,omitempty"`
	Url             string   `json:"url,omitempty"`
	Origin          string   `json:"origin,omitempty"`
	Dest            string   `json:"destino,omitempty"`
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

	reqCountByDest = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "qtty_reqs_counter",
		Help:      "Requests by Dest",
	}, []string{"destino", "status"})
)

func init() {
	prometheus.MustRegister(reqQttyStatus200)
	prometheus.MustRegister(reqQttyStatus400)
	prometheus.MustRegister(reqQttyStatus500)
	prometheus.MustRegister(reqCountByDest)
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
	router.HandleFunc("/metrics/create", CreateMetricEndpoint).Methods("POST")
	fmt.Println("Server Running: ", *sHost + ":" + *sPort)
	log.Fatal(http.ListenAndServe(*sHost + ":" + *sPort, router))
}

func SearchKeyToCounter(current string, next string, status_start int, status_end int) {
	fmt.Println("----------------------------------------------")
	timeline := elastic.NewTermsAggregation().Field("destino.keyword")
	searchResult, err := client.Search().
		Index("metrics").
		Query(elastic.NewMatchAllQuery()).
		Size(0).
		Aggregation("destinos", timeline).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		panic(err)
	}

	agg, found := searchResult.Aggregations.Terms("destinos")

	if !found {
		log.Fatalf("we should have a terms aggregation called %q", "timeline")
	}
	var t string

	if status_start == 0{
		t = "Total";
	}else{
		t = strconv.Itoa(status_start)
	}

	fmt.Println(t)

	for _, dest := range agg.Buckets {
		dest := dest.Key.(string)
		reqCountByDest.WithLabelValues(dest, t).Set(GetMetricsByDest(dest, current, next, status_start, status_end))
	}
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
	current := time.Now().Local().Add(time.Duration(-10) * time.Second).Format("20060102150405")
	next := time.Now().Local().Add(time.Duration(seconds) * time.Second).Format("20060102150405")

	fmt.Println("Current: " + current)
	fmt.Println("Next:    " + next)

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

	rawQuery200 := elastic.NewRawStringQuery(fmt.Sprintf(query, current, next, 200, 299))
	searchResult200, err := client.Search().
		Query(rawQuery200).
		Index("metrics").
		Pretty(true).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	SearchKeyToCounter(current, next, 200, 299)
	reqQttyStatus200.Set(float64(searchResult200.Hits.TotalHits))

	rawQuery400 := elastic.NewRawStringQuery(fmt.Sprintf(query, current, next, 400, 499))
	searchResult400, err := client.Search().
		Query(rawQuery400).
		Index("metrics").
		Pretty(true).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	SearchKeyToCounter(current, next, 400, 499)
	reqQttyStatus400.Set(float64(searchResult400.Hits.TotalHits))

	rawQuery500 := elastic.NewRawStringQuery(fmt.Sprintf(query, current, next, 500, 999))
	searchResult500, err := client.Search().
		Query(rawQuery500).
		Index("metrics").
		Pretty(true).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	SearchKeyToCounter(current, next, 500, 999)
	reqQttyStatus500.Set(float64(searchResult500.Hits.TotalHits))

	SearchKeyToCounter(current, next, 0, 999)

	promhttp.Handler().ServeHTTP(w, req)
}

func GetMetricsByDest(dest string, current string, next string, status_start int, status_end int) float64 {
	query := `
	{
	    "bool": {
		"must": [
		    {
			"term": {
			    "destino": {
				"value": "%s"
			    }
			}
            	    },
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

	rawQuery := elastic.NewRawStringQuery(fmt.Sprintf(query, dest, current, next, status_start, status_end))

	searchResult, err := client.Search().
		Query(rawQuery).
		Index("metrics").
		Pretty(true).
		Do(context.Background())
	if err != nil {
		panic(err)
	}

	return float64(searchResult.Hits.TotalHits)
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
