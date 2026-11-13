output "gateway_url" {
  description = "Team VDI portal URL."
  value       = module.gateway.gateway_url
}

output "rustdesk_relay_endpoint" {
  description = "Dedicated RustDesk hbbs/hbbr endpoint (use this on on-prem hybrid nodes)."
  value       = module.rustdesk.hbbs_endpoint
}

output "cloud_run_broker_url" {
  description = "Cloud Run broker service URL."
  value       = module.broker.broker_url
}
