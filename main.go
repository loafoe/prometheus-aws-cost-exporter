package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	costGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aws_cost",
			Help: "AWS cost in USD",
		},
		[]string{"service", "region"},
	)
)

func init() {
	prometheus.MustRegister(costGauge)
}

func fetchCostData() {
	sess := session.Must(session.NewSession())
	svc := costexplorer.New(sess)

	end := time.Now().Format("2006-01-02")
	start := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
		Granularity: aws.String("DAILY"),
		Metrics:     []*string{aws.String("UnblendedCost")},
		GroupBy: []*costexplorer.GroupDefinition{
			{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("SERVICE"),
			},
			{
				Type: aws.String("DIMENSION"),
				Key:  aws.String("REGION"),
			},
		},
	}

	result, err := svc.GetCostAndUsage(input)
	if err != nil {
		log.Fatalf("Failed to get cost data: %v", err)
	}

	costGauge.Reset()
	for _, group := range result.ResultsByTime[0].Groups {
		service := *group.Keys[0]
		region := *group.Keys[1]
		amount := *group.Metrics["UnblendedCost"].Amount
		cost, err := strconv.ParseFloat(amount, 64)
		if err != nil {
			log.Printf("Failed to parse cost for service %s: %v", service, err)
			continue
		}
		costGauge.WithLabelValues(service, region).Set(cost)
	}
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		for {
			fetchCostData()
			time.Sleep(24 * time.Hour)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
