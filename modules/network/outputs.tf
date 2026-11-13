output "network_id" {
  description = "Self-link / ID of the VPC network."
  value       = google_compute_network.this.id
}

output "subnetwork_id" {
  description = "Self-link / ID of the VDI subnetwork."
  value       = google_compute_subnetwork.this.id
}

output "network_project_id" {
  description = "Project ID owning the VPC network."
  value       = local.network_project_id
}

output "gateway_tag" {
  description = "Network tag applied to gateway/relay VMs."
  value       = var.gateway_tag
}

output "desktop_tag" {
  description = "Network tag applied to desktop VMs."
  value       = var.desktop_tag
}
