variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "Always-Free eligible region (us-central1, us-east1, or us-west1)."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone."
  type        = string
  default     = "us-central1-a"
}

variable "domain" {
  description = "Optional custom domain for Caddy TLS. Leave empty for zero-config sslip.io."
  type        = string
  default     = ""
}

variable "enable_cloud_dns" {
  description = "Manage a Cloud DNS A record for `domain` natively in GCP (~$0.20/mo per zone). Leave false for Cloudflare DNS-only or sslip.io."
  type        = bool
  default     = false
}

variable "cloud_dns_zone_name" {
  description = "Optional existing Cloud DNS managed zone name. If empty and `enable_cloud_dns = true`, a new zone is created."
  type        = string
  default     = ""
}

variable "cloud_dns_zone_domain" {
  description = "Optional DNS zone domain (e.g., example.com) when creating a new Cloud DNS zone. Defaults to `domain`."
  type        = string
  default     = ""
}

variable "oauth_client_id" {
  description = "Google OAuth2 Client ID."
  type        = string
}

variable "oauth_client_secret" {
  description = "Google OAuth2 Client Secret."
  type        = string
  sensitive   = true
}

variable "allowed_emails" {
  description = "Allowed Google email addresses."
  type        = list(string)
}

variable "broker_register_secret" {
  description = "Shared bearer secret for desktop-to-broker registration."
  type        = string
  sensitive   = true
}

variable "desktop_machine_type" {
  description = "Spot VM machine type for desktops."
  type        = string
  default     = "e2-standard-4"
}

variable "enable_browser_fallback" {
  description = "Enable Apache Guacamole browser access."
  type        = bool
  default     = true
}

variable "desktop_hourly_rate_usd" {
  description = "Estimated hourly compute rate in USD for the live cost meter."
  type        = number
  default     = 0.04
}

variable "max_vm_hours" {
  description = "Hard maximum desktop lifetime in hours before unconditional auto-destroy."
  type        = number
  default     = 8
}
