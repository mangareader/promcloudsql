package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	cpuUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudsql_cpu_utilization",
			Help: "Cloud SQL CPU utilization",
		},
		[]string{"instance"},
	)
)

func init() {
	prometheus.MustRegister(cpuUtilization)
}

func fetchMetrics(projectID string) {
	ctx := context.Background()

	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Monitoring client: %v", err)
	}
	defer client.Close()

	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: `metric.type="cloudsql.googleapis.com/database/cpu/utilization"`,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}

	it := client.ListTimeSeries(ctx, req)
	for {
		resp, err := it.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			log.Printf("Error fetching time series: %v", err)
			break
		}

		// Get instance name
		instance := resp.Resource.Labels["database_id"]
		// Get latest point
		if len(resp.Points) > 0 {
			val := resp.Points[0].Value.GetDoubleValue()
			cpuUtilization.WithLabelValues(instance).Set(val)
		}
	}
}

func main() {
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		log.Fatal("GCP_PROJECT environment variable not set")
	}

	// Scrape every 60 seconds
	go func() {
		for {
			fetchMetrics(projectID)
			time.Sleep(60 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Exporter running at :8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
