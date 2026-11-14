locals {
  image_hbbs = var.enable_rustdesk_pro ? "rustdesk/rustdesk-server-pro:latest" : "rustdesk/rustdesk-server:latest"
  image_hbbr = var.enable_rustdesk_pro ? "rustdesk/rustdesk-server-pro:latest" : "rustdesk/rustdesk-server:latest"
}

resource "google_compute_address" "relay" {
  count   = var.deployment_mode == "dedicated" ? 1 : 0
  project = var.project_id
  name    = "gcp-vdi-rustdesk-relay-ip"
  region  = var.region
}

resource "google_service_account" "relay" {
  count        = var.deployment_mode == "dedicated" ? 1 : 0
  project      = var.project_id
  account_id   = "gcp-vdi-rustdesk-relay"
  display_name = "gcp-vdi RustDesk Relay SA"
}

resource "google_compute_instance" "relay" {
  count        = var.deployment_mode == "dedicated" ? 1 : 0
  project      = var.project_id
  name         = "gcp-vdi-rustdesk-relay"
  machine_type = var.machine_type
  zone         = var.zone
  tags         = [var.gateway_tag]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 20
      type  = "pd-balanced"
    }
  }

  network_interface {
    subnetwork = var.subnetwork_id
    access_config {
      nat_ip = google_compute_address.relay[0].address
    }
  }

  service_account {
    email  = google_service_account.relay[0].email
    scopes = ["cloud-platform"]
  }

  metadata_startup_script = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail
    apt-get update -qq
    apt-get install -y -qq docker.io unattended-upgrades
    apt-get install -y -qq docker-compose-v2 || apt-get install -y -qq docker-compose-plugin || true
    mkdir -p /opt/rustdesk /var/lib/rustdesk
    cat <<'EOF' > /opt/rustdesk/docker-compose.yml
    ${templatefile("${path.module}/templates/docker-compose.yml.tftpl", {
  image_hbbs = local.image_hbbs
  image_hbbr = local.image_hbbr
  relay_host = google_compute_address.relay[0].address
})}
    EOF
    docker compose -f /opt/rustdesk/docker-compose.yml up -d
  EOT
}
