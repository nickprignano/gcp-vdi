variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "GCP region (us-central1, us-east1, or us-west1 for Always-Free e2-micro)."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone for the gateway VM."
  type        = string
  default     = "us-central1-a"
}

variable "subnetwork_id" {
  description = "Subnetwork self-link."
  type        = string
}

variable "gateway_tag" {
  description = "Network tag for gateway firewall rules."
  type        = string
  default     = "gcp-vdi-gateway"
}

variable "machine_type" {
  description = "Gateway machine type (e2-micro for Tier 1 Always-Free)."
  type        = string
  default     = "e2-micro"
}

variable "boot_disk_size_gb" {
  description = "Gateway boot disk size in GB (30 GB pd-standard for Always-Free)."
  type        = number
  default     = 30
}

variable "boot_disk_type" {
  description = "Gateway boot disk type (pd-standard or pd-balanced)."
  type        = string
  default     = "pd-standard"
}

variable "domain" {
  description = "Custom domain for Caddy auto-TLS. Leave empty to use <gateway-ip>.sslip.io."
  type        = string
  default     = ""
}

variable "oauth_client_id" {
  description = "Google OAuth2 Client ID for oauth2-proxy."
  type        = string
}

variable "oauth_client_secret" {
  description = "Google OAuth2 Client Secret for oauth2-proxy."
  type        = string
  sensitive   = true
}

variable "allowed_emails" {
  description = "List of allowed Google email addresses for Tier 1 access."
  type        = list(string)
  default     = []
}

variable "home_bucket_name" {
  description = "GCS bucket name for publishing id_ed25519.pub."
  type        = string
}

variable "embed_rustdesk" {
  description = "Run hbbs/hbbr on the gateway (true in Tier 1, false in Tier 2/3 dedicated relay)."
  type        = bool
  default     = true
}

variable "rustdesk_compose_yaml" {
  description = "Rendered docker-compose.yml snippet from modules/rustdesk."
  type        = string
  default     = ""
}

variable "broker_upstream_url" {
  description = "Upstream URL for the broker (http://127.0.0.1:8080 in Tier 1 or Cloud Run URI in Tier 2/3)."
  type        = string
  default     = "http://127.0.0.1:8080"
}

variable "broker_image" {
  description = "Container image for gcp-vdi-broker when embedded on gateway."
  type        = string
  default     = "ghcr.io/nickprignano/gcp-vdi-broker:latest"
}

variable "broker_register_secret" {
  description = "Shared bearer secret for desktop-to-broker registration."
  type        = string
  sensitive   = true
}

variable "instance_template_self_link" {
  description = "Desktop instance template self-link for embedded broker."
  type        = string
}

variable "desktop_service_account_email" {
  description = "Desktop SA email so gateway SA can act as it when embedded broker runs on gateway."
  type        = string
}

variable "enable_browser_fallback" {
  description = "Enable optional Apache Guacamole (guacd + guacamole) container on gateway."
  type        = bool
  default     = false
}

variable "enable_enterprise_ingress" {
  description = "Provision Cloud Armor WAF + External HTTPS LB + IAP (Tier 3)."
  type        = bool
  default     = false
}

variable "allocate_gateway_ip" {
  description = "Set false when passing a pre-allocated gateway_ip_address."
  type        = bool
  default     = true
}

variable "gateway_ip_address" {
  description = "Optional pre-allocated static IP address for the gateway (breaks cycle with desktop-image)."
  type        = string
  default     = ""
}
