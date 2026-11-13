resource "random_id" "cookie_secret" {
  byte_length = 16
}

resource "google_compute_address" "gateway" {
  count   = var.gateway_ip_address == "" ? 1 : 0
  project = var.project_id
  name    = "gcp-vdi-gateway-ip"
  region  = var.region
}

locals {
  effective_ip = var.gateway_ip_address != "" ? var.gateway_ip_address : google_compute_address.gateway[0].address
  site_host    = var.domain != "" ? var.domain : "${local.effective_ip}.sslip.io"
  caddyfile = templatefile("${path.module}/templates/Caddyfile.tftpl", {
    site_host       = local.site_host
    broker_upstream = var.broker_upstream_url
  })
}

resource "google_service_account" "gateway" {
  project      = var.project_id
  account_id   = "gcp-vdi-gateway"
  display_name = "gcp-vdi Gateway SA"
}

resource "google_project_iam_member" "gateway_compute" {
  project = var.project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:${google_service_account.gateway.email}"
}

resource "google_service_account_iam_member" "gateway_act_as_desktop" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/${var.desktop_service_account_email}"
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.gateway.email}"
}

resource "google_storage_bucket_iam_member" "gateway_pubkey" {
  bucket = var.home_bucket_name
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.gateway.email}"
}

resource "google_compute_instance" "gateway" {
  project      = var.project_id
  name         = "gcp-vdi-gateway"
  machine_type = var.machine_type
  zone         = var.zone
  tags         = [var.gateway_tag]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
      size  = var.boot_disk_size_gb
      type  = var.boot_disk_type
    }
  }

  network_interface {
    subnetwork = var.subnetwork_id
    access_config {
      nat_ip = local.effective_ip
    }
  }

  service_account {
    email  = google_service_account.gateway.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    enable-oslogin = "TRUE"
    startup-script = templatefile("${path.module}/templates/gateway-startup.sh.tftpl", {
      site_host                   = local.site_host
      caddyfile                   = local.caddyfile
      allowed_emails_text         = join("\n", var.allowed_emails)
      oauth_client_id             = var.oauth_client_id
      oauth_client_secret         = var.oauth_client_secret
      cookie_secret               = random_id.cookie_secret.b64_url
      broker_upstream             = var.broker_upstream_url
      broker_image                = var.broker_image
      project_id                  = var.project_id
      zone                        = var.zone
      instance_template_self_link = var.instance_template_self_link
      broker_register_secret      = var.broker_register_secret
      enable_browser_fallback     = tostring(var.enable_browser_fallback)
      embed_rustdesk              = tostring(var.embed_rustdesk)
      rustdesk_compose_yaml       = var.rustdesk_compose_yaml
      home_bucket_name            = var.home_bucket_name
    })
  }
}

# Tier 3 Enterprise WAF (Cloud Armor) when enable_enterprise_ingress = true
resource "google_compute_security_policy" "waf" {
  count   = var.enable_enterprise_ingress ? 1 : 0
  project = var.project_id
  name    = "gcp-vdi-cloud-armor"

  rule {
    action   = "throttle"
    priority = 1000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      rate_limit_threshold {
        count        = 120
        interval_sec = 60
      }
    }
  }

  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
  }
}
