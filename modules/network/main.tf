locals {
  network_project_id = var.enable_shared_vpc && var.host_project_id != "" ? var.host_project_id : var.project_id
}

resource "google_compute_network" "this" {
  project                 = local.network_project_id
  name                    = var.network_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "this" {
  project                  = local.network_project_id
  name                     = "${var.network_name}-${var.region}"
  ip_cidr_range            = var.subnet_cidr
  region                   = var.region
  network                  = google_compute_network.this.id
  private_ip_google_access = true
}

resource "google_compute_shared_vpc_host_project" "host" {
  count   = var.enable_shared_vpc ? 1 : 0
  project = local.network_project_id
}

resource "google_compute_shared_vpc_service_project" "service" {
  count           = var.enable_shared_vpc ? 1 : 0
  host_project    = google_compute_shared_vpc_host_project.host[0].project
  service_project = var.project_id
}

# IAP-only SSH across all VDI instances (35.235.240.0/20)
resource "google_compute_firewall" "allow_iap_ssh" {
  project = local.network_project_id
  name    = "${var.network_name}-allow-iap-ssh"
  network = google_compute_network.this.id

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["35.235.240.0/20"]
  target_tags   = [var.gateway_tag, var.desktop_tag]
}

# Gateway HTTPS/HTTP (Caddy auto-TLS)
resource "google_compute_firewall" "allow_gateway_https" {
  project = local.network_project_id
  name    = "${var.network_name}-allow-gateway-https"
  network = google_compute_network.this.id

  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = [var.gateway_tag]
}

# RustDesk hbbs/hbbr ports on gateway/relay (protected by -k _ keypair enforcement)
resource "google_compute_firewall" "allow_rustdesk_relay" {
  project = local.network_project_id
  name    = "${var.network_name}-allow-rustdesk"
  network = google_compute_network.this.id

  allow {
    protocol = "tcp"
    ports    = ["21115-21119"]
  }

  allow {
    protocol = "udp"
    ports    = ["21116"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = [var.gateway_tag]
}

# Optional internal RDP from gateway to desktop when Guacamole browser fallback is enabled
resource "google_compute_firewall" "allow_gateway_to_desktop_rdp" {
  count   = var.enable_browser_fallback ? 1 : 0
  project = local.network_project_id
  name    = "${var.network_name}-allow-guac-rdp"
  network = google_compute_network.this.id

  allow {
    protocol = "tcp"
    ports    = ["3389"]
  }

  source_tags = [var.gateway_tag]
  target_tags = [var.desktop_tag]
}

# Explicit deny-all inbound on desktops (lowest priority after IAP & optional Guac)
resource "google_compute_firewall" "deny_desktop_inbound" {
  project  = local.network_project_id
  name     = "${var.network_name}-deny-desktop-inbound"
  network  = google_compute_network.this.id
  priority = 65534

  deny {
    protocol = "all"
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = [var.desktop_tag]
}

# Optional Cloud NAT for private-only desktops (Tier 3)
resource "google_compute_router" "nat" {
  count   = var.enable_cloud_nat ? 1 : 0
  project = local.network_project_id
  name    = "${var.network_name}-router"
  region  = var.region
  network = google_compute_network.this.id
}

resource "google_compute_router_nat" "nat" {
  count                              = var.enable_cloud_nat ? 1 : 0
  project                            = local.network_project_id
  name                               = "${var.network_name}-nat"
  router                             = google_compute_router.nat[0].name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}
