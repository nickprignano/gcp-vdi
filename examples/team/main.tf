terraform {
  required_version = ">= 1.6.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.30.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_compute_address" "gateway" {
  project = var.project_id
  name    = "gcp-vdi-team-gateway-ip"
  region  = var.region
}

locals {
  site_host = var.domain != "" ? var.domain : "${google_compute_address.gateway.address}.sslip.io"
}

module "network" {
  source = "../../modules/network"

  project_id              = var.project_id
  region                  = var.region
  network_name            = "gcp-vdi-team"
  enable_browser_fallback = var.enable_browser_fallback
}

module "state" {
  source = "../../modules/state"

  project_id  = var.project_id
  region      = var.region
  name_prefix = "gcp-vdi-team"
}

module "rustdesk" {
  source = "../../modules/rustdesk"

  project_id      = var.project_id
  region          = var.region
  zone            = var.zone
  deployment_mode = "dedicated"
  subnetwork_id   = module.network.subnetwork_id
  gateway_tag     = module.network.gateway_tag
  machine_type    = "e2-small"
}

module "desktop_image" {
  source = "../../modules/desktop-image"

  project_id              = var.project_id
  region                  = var.region
  subnetwork_id           = module.network.subnetwork_id
  desktop_tag             = module.network.desktop_tag
  home_bucket_name        = module.state.home_bucket_name
  rustdesk_host           = module.rustdesk.hbbs_endpoint
  broker_register_url     = "https://${local.site_host}/api/v1/devices/register"
  broker_register_secret  = var.broker_register_secret
  source_image            = var.desktop_source_image
  provisioning_model      = "SPOT"
  assign_public_ip        = true
  machine_type            = var.desktop_machine_type
  enable_browser_fallback = var.enable_browser_fallback
}

module "broker" {
  source = "../../modules/broker"

  project_id                    = var.project_id
  region                        = var.region
  zone                          = var.zone
  deployment_mode               = "cloud_run"
  instance_template_self_link   = module.desktop_image.instance_template_self_link
  desktop_service_account_email = module.desktop_image.desktop_service_account_email
  rustdesk_host                 = module.rustdesk.hbbs_endpoint
  enable_browser_fallback       = var.enable_browser_fallback
}

module "gateway" {
  source = "../../modules/gateway"

  project_id                    = var.project_id
  region                        = var.region
  zone                          = var.zone
  subnetwork_id                 = module.network.subnetwork_id
  gateway_tag                   = module.network.gateway_tag
  gateway_ip_address            = google_compute_address.gateway.address
  machine_type                  = "e2-small"
  boot_disk_type                = "pd-balanced"
  domain                        = var.domain
  oauth_client_id               = var.oauth_client_id
  oauth_client_secret           = var.oauth_client_secret
  allowed_emails                = var.allowed_emails
  home_bucket_name              = module.state.home_bucket_name
  embed_rustdesk                = false
  broker_upstream_url           = module.broker.broker_url
  broker_register_secret        = var.broker_register_secret
  instance_template_self_link   = module.desktop_image.instance_template_self_link
  desktop_service_account_email = module.desktop_image.desktop_service_account_email
  enable_browser_fallback       = var.enable_browser_fallback
}
