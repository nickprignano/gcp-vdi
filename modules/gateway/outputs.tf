output "gateway_ip" {
  description = "Static external IP of the gateway."
  value       = local.effective_ip
}

output "gateway_url" {
  description = "HTTPS portal URL."
  value       = "https://${local.site_host}"
}

output "oauth_redirect_uri" {
  description = "OAuth2 callback URL to register in Google Cloud Console."
  value       = "https://${local.site_host}/oauth2/callback"
}

output "gateway_service_account_email" {
  description = "Service account email attached to the gateway."
  value       = google_service_account.gateway.email
}
