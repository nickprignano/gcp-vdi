output "hbbs_endpoint" {
  description = "RustDesk hbbs ID server host/IP."
  value       = var.deployment_mode == "dedicated" ? google_compute_address.relay[0].address : var.relay_host
}

output "hbbr_endpoint" {
  description = "RustDesk hbbr relay server host/IP."
  value       = var.deployment_mode == "dedicated" ? google_compute_address.relay[0].address : var.relay_host
}

output "compose_yaml" {
  description = "Rendered docker-compose.yml for embedding hbbs/hbbr on the gateway (Tier 1)."
  value = templatefile("${path.module}/templates/docker-compose.yml.tftpl", {
    image_hbbs = local.image_hbbs
    image_hbbr = local.image_hbbr
    relay_host = var.deployment_mode == "dedicated" ? google_compute_address.relay[0].address : var.relay_host
  })
}
