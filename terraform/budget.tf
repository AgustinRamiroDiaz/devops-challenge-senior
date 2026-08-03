resource "google_billing_budget" "simple_time_service" {
  billing_account = local.billing_account_id
  display_name    = "SimpleTimeService monthly budget"

  budget_filter {
    projects        = ["projects/${google_project.simple_time_service.number}"]
    calendar_period = "MONTH"
  }

  amount {
    specified_amount {
      # Omitting currency_code uses the billing account's currency.
      units = tostring(var.monthly_budget_amount)
    }
  }

  threshold_rules {
    threshold_percent = 0.5
  }

  threshold_rules {
    threshold_percent = 0.8
  }

  threshold_rules {
    threshold_percent = 1.0
  }

  threshold_rules {
    threshold_percent = 1.0
    spend_basis       = "FORECASTED_SPEND"
  }

  all_updates_rule {
    enable_project_level_recipients  = true
    monitoring_notification_channels = []
  }

  depends_on = [google_project_service.required]
}
