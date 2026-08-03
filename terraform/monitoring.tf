resource "google_monitoring_dashboard" "simple_time_service" {
  project = google_project.simple_time_service.project_id

  dashboard_json = jsonencode({
    displayName = "SimpleTimeService"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          xPos   = 0
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "Request rate"
            scorecard = {
              timeSeriesQuery = {
                prometheusQuery = "sum(rate(simple_time_requests_total[5m]))"
              }
              sparkChartView = {
                sparkChartType = "SPARK_LINE"
              }
            }
          }
        },
        {
          xPos   = 4
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "5xx error rate (%)"
            scorecard = {
              timeSeriesQuery = {
                prometheusQuery = "100 * (sum(rate(simple_time_requests_total{http_response_status_code=~\"5..\"}[5m])) or vector(0)) / clamp_min((sum(rate(simple_time_requests_total[5m])) or vector(0)), 0.001)"
              }
              sparkChartView = {
                sparkChartType = "SPARK_LINE"
              }
              thresholds = [
                {
                  value     = 1
                  color     = "YELLOW"
                  direction = "ABOVE"
                },
                {
                  value     = 5
                  color     = "RED"
                  direction = "ABOVE"
                }
              ]
            }
          }
        },
        {
          xPos   = 8
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "p95 latency (ms)"
            scorecard = {
              timeSeriesQuery = {
                prometheusQuery = "histogram_quantile(0.95, sum by (le) (rate(simple_time_request_duration_ms_milliseconds_bucket[5m])))"
              }
              sparkChartView = {
                sparkChartType = "SPARK_LINE"
              }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 4
          width  = 6
          height = 5
          widget = {
            title = "Request rate by route"
            xyChart = {
              dataSets = [
                {
                  timeSeriesQuery = {
                    prometheusQuery = "sum by (http_route) (rate(simple_time_requests_total[5m]))"
                  }
                  plotType   = "LINE"
                  targetAxis = "Y1"
                }
              ]
              yAxis = {
                label = "requests/s"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 6
          yPos   = 4
          width  = 6
          height = 5
          widget = {
            title = "Request rate by status code"
            xyChart = {
              dataSets = [
                {
                  timeSeriesQuery = {
                    prometheusQuery = "sum by (http_response_status_code) (rate(simple_time_requests_total[5m]))"
                  }
                  plotType   = "STACKED_BAR"
                  targetAxis = "Y1"
                }
              ]
              yAxis = {
                label = "requests/s"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 9
          width  = 12
          height = 5
          widget = {
            title = "Request latency percentiles"
            xyChart = {
              dataSets = [
                {
                  legendTemplate = "p50"
                  timeSeriesQuery = {
                    prometheusQuery = "histogram_quantile(0.50, sum by (le) (rate(simple_time_request_duration_ms_milliseconds_bucket[5m])))"
                  }
                  plotType   = "LINE"
                  targetAxis = "Y1"
                },
                {
                  legendTemplate = "p95"
                  timeSeriesQuery = {
                    prometheusQuery = "histogram_quantile(0.95, sum by (le) (rate(simple_time_request_duration_ms_milliseconds_bucket[5m])))"
                  }
                  plotType   = "LINE"
                  targetAxis = "Y1"
                },
                {
                  legendTemplate = "p99"
                  timeSeriesQuery = {
                    prometheusQuery = "histogram_quantile(0.99, sum by (le) (rate(simple_time_request_duration_ms_milliseconds_bucket[5m])))"
                  }
                  plotType   = "LINE"
                  targetAxis = "Y1"
                }
              ]
              yAxis = {
                label = "milliseconds"
                scale = "LINEAR"
              }
              chartOptions = {
                mode = "COLOR"
              }
            }
          }
        }
      ]
    }
  })

  depends_on = [google_project_service.required]
}
