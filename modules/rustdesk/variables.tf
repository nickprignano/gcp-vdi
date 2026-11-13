variable "project_id" {
  description = "GCP project ID."
  type        = string
}

variable "region" {
  description = "GCP region."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone for dedicated relay VM (when deployment_mode = dedicated)."
  type        = string
  default     = "us-central1-a"
}

variable "deployment_mode" {
  description = "embedded (Tier 1 on e2-micro gateway) or dedicated (Tier 2/3 standalone relay VM)."
  type        = string
  default     = "embedded"

  validation {
    condition     = contains(["embedded", "dedicated"], var.deployment_mode)
    error_message = "deployment_mode must be 'embedded' or 'dedicated'."
  }
}

variable "enable_rustdesk_pro" {
  description = "Use RustDesk Server Pro image (Tier 3 opt-in; requires commercial license)."
  type        = bool
  default     = false
}

variable "relay_host" {
  description = "Public hostname or IP for hbbs/hbbr (used when deployment_mode = embedded; overridden by dedicated IP otherwise)."
  type        = string
  default     = ""
}

variable "subnetwork_id" {
  description = "Subnetwork self-link for dedicated relay VM."
  type        = string
  default     = ""
}

variable "gateway_tag" {
  description = "Network tag for firewall rules."
  type        = string
  default     = "gcp-vdi-gateway"
}

variable "machine_type" {
  description = "Machine type for dedicated relay VM."
  type        = string
  default     = "e2-small"
}
