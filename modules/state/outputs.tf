output "home_bucket_name" {
  description = "GCS bucket name for per-user rclone home-dir sync."
  value       = google_storage_bucket.homes.name
}

output "kms_key_id" {
  description = "Cloud KMS crypto key ID when enable_cmek is true, otherwise null."
  value       = var.enable_cmek ? google_kms_crypto_key.this[0].id : null
}

output "state_backend_uri" {
  description = "Broker state backend URI."
  value       = var.enable_firestore ? "firestore://${var.project_id}/(default)" : "file:///var/lib/gcp-vdi/state.json"
}
