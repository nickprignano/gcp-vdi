output "gateway_url" {
  description = "Enterprise VDI portal URL."
  value       = module.gateway.gateway_url
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
