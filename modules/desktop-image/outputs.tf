output "instance_template_self_link" {
  description = "Self-link of the desktop GCE instance template."
  value       = google_compute_instance_template.desktop.self_link
}

output "desktop_service_account_email" {
  description = "Service account email attached to desktop VMs."
  value       = google_service_account.desktop.email
}
