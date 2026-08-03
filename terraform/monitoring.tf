resource "google_monitoring_dashboard" "simple_time_service" {
  project = google_project.simple_time_service.project_id

  dashboard_json = jsonencode({
    displayName = "SimpleTimeService"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          width  = 12
          height = 2
          widget = {
            title = "How to read this dashboard"
            text = {
              format  = "MARKDOWN"
              content = <<-EOT
                **Application metrics exported by the OpenTelemetry sidecar**

                - Rates use a trailing **5-minute window**.
                - Latency measures **Go HTTP handler execution**.
                - Latency excludes client, network, and load-balancer time.
              EOT
            }
          }
        },
        {
          yPos   = 2
          width  = 4
          height = 4
          widget = {
            title = "Request rate — trailing 5-minute average"
            scorecard = {
              timeSeriesQuery = {
                prometheusQuery = "sum(rate(simple_time_requests_total[5m]))"
                unitOverride    = "{request}/s"
              }
              sparkChartView = {
                sparkChartType = "SPARK_LINE"
              }
            }
          }
        },
        {
          xPos   = 4
          yPos   = 2
          width  = 4
          height = 4
          widget = {
            title = "5xx error rate — share of requests"
            scorecard = {
              timeSeriesQuery = {
                prometheusQuery = "100 * (sum(rate(simple_time_requests_total{http_response_status_code=~\"5..\"}[5m])) or vector(0)) / clamp_min((sum(rate(simple_time_requests_total[5m])) or vector(0)), 0.001)"
                unitOverride    = "%"
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
          yPos   = 2
          width  = 4
          height = 4
          widget = {
            title = "p95 handler latency — trailing 5 minutes"
            scorecard = {
              timeSeriesQuery = {
                prometheusQuery = "1000 * histogram_quantile(0.95, sum by (le) (rate(http_server_request_duration_seconds_bucket[5m])))"
                unitOverride    = "ms"
              }
              sparkChartView = {
                sparkChartType = "SPARK_LINE"
              }
            }
          }
        },
        {
          yPos   = 6
          width  = 6
          height = 5
          widget = {
            title = "Request rate by route — traffic distribution"
            xyChart = {
              dataSets = [
                {
                  timeSeriesQuery = {
                    prometheusQuery = "sum by (http_route) (rate(simple_time_requests_total[5m]))"
                    unitOverride    = "{request}/s"
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
          yPos   = 6
          width  = 6
          height = 5
          widget = {
            title = "Request rate by status code — response outcomes"
            xyChart = {
              dataSets = [
                {
                  timeSeriesQuery = {
                    prometheusQuery = "sum by (http_response_status_code) (rate(simple_time_requests_total[5m]))"
                    unitOverride    = "{request}/s"
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
          yPos   = 11
          width  = 12
          height = 5
          widget = {
            title = "Handler latency percentiles — response-time distribution"
            xyChart = {
              dataSets = [
                {
                  legendTemplate = "p50"
                  timeSeriesQuery = {
                    prometheusQuery = "1000 * histogram_quantile(0.50, sum by (le) (rate(http_server_request_duration_seconds_bucket[5m])))"
                    unitOverride    = "ms"
                  }
                  plotType   = "LINE"
                  targetAxis = "Y1"
                },
                {
                  legendTemplate = "p95"
                  timeSeriesQuery = {
                    prometheusQuery = "1000 * histogram_quantile(0.95, sum by (le) (rate(http_server_request_duration_seconds_bucket[5m])))"
                    unitOverride    = "ms"
                  }
                  plotType   = "LINE"
                  targetAxis = "Y1"
                },
                {
                  legendTemplate = "p99"
                  timeSeriesQuery = {
                    prometheusQuery = "1000 * histogram_quantile(0.99, sum by (le) (rate(http_server_request_duration_seconds_bucket[5m])))"
                    unitOverride    = "ms"
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
