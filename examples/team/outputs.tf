output "gateway_ip" {
  description = "Static external IP of the gateway (point your Cloudflare DNS-only A record here)."
  value       = module.gateway.gateway_ip
}

output "gateway_url" {
  description = "Team VDI portal URL."
  value       = module.gateway.gateway_url
}

output "oauth_redirect_uri" {
  description = "Add this Redirect URI to your Google OAuth2 Web Client."
  value       = module.gateway.oauth_redirect_uri
}

output "cloud_dns_name_servers" {
  description = "Cloud DNS name servers (populated when enable_cloud_dns = true)."
  value       = module.gateway.cloud_dns_name_servers
}

output "rustdesk_relay_endpoint" {
  description = "Dedicated RustDesk hbbs/hbbr endpoint (use this on on-prem hybrid nodes)."
  value       = module.rustdesk.hbbs_endpoint
}

output "cloud_run_broker_url" {
  description = "Cloud Run broker service URL."
  value       = module.broker.broker_url
}
