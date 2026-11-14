output "gateway_ip" {
  description = "Static external IP of the gateway (point your Cloudflare DNS-only A record here)."
  value       = module.gateway.gateway_ip
}

output "gateway_url" {
  description = "Enterprise VDI portal URL."
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
  description = "Dedicated RustDesk hbbs/hbbr endpoint."
  value       = module.rustdesk.hbbs_endpoint
}

output "kms_key_id" {
  description = "CMEK Cloud KMS key ID."
  value       = module.state.kms_key_id
}

output "state_backend_uri" {
  description = "Firestore state URI."
  value       = module.state.state_backend_uri
}
