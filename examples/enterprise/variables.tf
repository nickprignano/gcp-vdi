variable "service_project_id" {
  description = "Service project ID for VDI workloads."
  type        = string
}

variable "host_project_id" {
  description = "Host project ID for Shared VPC (optional when enable_shared_vpc = false)."
  type        = string
  default     = ""
}

variable "enable_shared_vpc" {
  description = "Enable Host/Service Shared VPC (set false for single-project sandbox testing)."
  type        = bool
  default     = true
}

variable "region" {
  description = "GCP region."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone."
  type        = string
  default     = "us-central1-a"
}

variable "domain" {
  description = "Enterprise VDI domain."
  type        = string
}

variable "enable_cloud_dns" {
  description = "Manage a Cloud DNS A record for `domain` natively in GCP (~$0.20/mo per zone). Leave false for external DNS such as Cloudflare (DNS-only)."
  type        = bool
  default     = false
}

variable "cloud_dns_zone_name" {
  description = "Optional existing Cloud DNS managed zone name. If empty and `enable_cloud_dns = true`, a new zone is created."
  type        = string
  default     = ""
}

variable "cloud_dns_zone_domain" {
  description = "Optional DNS zone domain (e.g., corp.example.com) when creating a new Cloud DNS zone. Defaults to `domain`."
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
  description = "Allowed user emails."
  type        = list(string)
}

variable "broker_register_secret" {
  description = "Shared bearer secret for device registration."
  type        = string
  sensitive   = true
}

variable "enable_rustdesk_pro" {
  description = "Opt-in to RustDesk Server Pro."
  type        = bool
  default     = false
}

variable "os_family" {
  description = "linux or windows."
  type        = string
  default     = "linux"
}

variable "desktop_source_image" {
  description = "Packer golden image family."
  type        = string
  default     = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
}

variable "desktop_machine_type" {
  description = "Enterprise desktop machine type."
  type        = string
  default     = "e2-standard-4"
}

variable "gpu_type" {
  description = "Optional GPU type (e.g., nvidia-tesla-t4)."
  type        = string
  default     = null
}

variable "sole_tenant_node_group" {
  description = "Optional Sole-Tenant node group for Windows BYOL."
  type        = string
  default     = null
}

variable "enable_browser_fallback" {
  description = "Enable optional Apache Guacamole browser fallback."
  type        = bool
  default     = false
}
