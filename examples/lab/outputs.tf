output "gateway_ip" {
  description = "Static external IP of the gateway (point your Cloudflare DNS-only A record here)."
  value       = module.gateway.gateway_ip
}

output "gateway_url" {
  description = "HTTPS portal URL."
  value       = module.gateway.gateway_url
}

output "cloud_dns_name_servers" {
  description = "Cloud DNS name servers (populated when enable_cloud_dns = true)."
  value       = module.gateway.cloud_dns_name_servers
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
