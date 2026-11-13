output "broker_service_account_email" {
  description = "Service account email for the broker."
  value       = google_service_account.broker.email
}

output "broker_url" {
  description = "Broker URL (Cloud Run URI or localhost:8080 on gateway)."
  value       = var.deployment_mode == "cloud_run" ? google_cloud_run_v2_service.broker[0].uri : "http://127.0.0.1:8080"
}

output "register_secret" {
  description = "Shared bearer secret for desktop-to-broker registration."
  value       = random_password.register_secret.result
  sensitive   = true
}
