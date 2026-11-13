resource "google_service_account" "desktop" {
  project      = var.project_id
  account_id   = "gcp-vdi-desktop"
  display_name = "gcp-vdi Desktop VM SA"
}

# Scoped GCS access for rclone home-dir sync and RustDesk pubkey fetch
resource "google_storage_bucket_iam_member" "desktop_homes" {
  bucket = var.home_bucket_name
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.desktop.email}"
}

resource "google_project_iam_member" "desktop_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.desktop.email}"
}

locals {
  startup_script = templatefile("${path.module}/scripts/linux-startup.sh.tftpl", {
    home_bucket_name        = var.home_bucket_name
    rustdesk_host           = var.rustdesk_host
    broker_register_url     = var.broker_register_url
    broker_register_secret  = var.broker_register_secret
    idle_timeout_minutes    = var.idle_timeout_minutes
    enable_browser_fallback = var.enable_browser_fallback
  })
}

resource "google_compute_instance_template" "desktop" {
  project      = var.project_id
  name_prefix  = "gcp-vdi-${var.os_family}-"
  machine_type = var.machine_type
  region       = var.region
  tags         = [var.desktop_tag]

  labels = {
    managed-by = "gcp-vdi"
    os-family  = var.os_family
  }

  disk {
    source_image = var.source_image
    auto_delete  = true
    boot         = true
    disk_size_gb = var.boot_disk_size_gb
    disk_type    = var.boot_disk_type

    dynamic "disk_encryption_key" {
      for_each = var.kms_key_id != null ? [1] : []
      content {
        kms_key_self_link = var.kms_key_id
      }
    }
  }

  network_interface {
    subnetwork = var.subnetwork_id
    dynamic "access_config" {
      for_each = var.assign_public_ip ? [1] : []
      content {}
    }
  }

  scheduling {
    preemptible                 = var.provisioning_model == "SPOT"
    automatic_restart           = var.provisioning_model == "STANDARD"
    provisioning_model          = var.provisioning_model
    instance_termination_action = var.provisioning_model == "SPOT" ? "DELETE" : null
    on_host_maintenance         = var.gpu_type != null ? "TERMINATE" : (var.provisioning_model == "SPOT" ? "TERMINATE" : "MIGRATE")

    dynamic "node_affinities" {
      for_each = var.sole_tenant_node_group != null ? [1] : []
      content {
        key      = "compute.googleapis.com/node-group-name"
        operator = "IN"
        values   = [var.sole_tenant_node_group]
      }
    }
  }

  dynamic "guest_accelerator" {
    for_each = var.gpu_type != null ? [1] : []
    content {
      type  = var.gpu_type
      count = var.gpu_count
    }
  }

  service_account {
    email  = google_service_account.desktop.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    enable-oslogin  = "TRUE"
    startup-script  = var.os_family == "linux" ? local.startup_script : null
    shutdown-script = var.os_family == "linux" ? "/usr/local/bin/vdi-sync.sh || true" : null
  }

  lifecycle {
    create_before_destroy = true
  }
}
