output "gateway_url" {
  description = "HTTPS portal URL."
  value       = module.gateway.gateway_url
}

output "oauth_redirect_uri" {
  description = "Add this Redirect URI to your Google OAuth2 Web Client."
  value       = module.gateway.oauth_redirect_uri
}

output "rustdesk_endpoint" {
  description = "RustDesk hbbs/hbbr host."
  value       = module.rustdesk.hbbs_endpoint
}

output "home_bucket_name" {
  description = "GCS bucket storing synced user home directories and RustDesk pubkey."
  value       = module.state.home_bucket_name
}
