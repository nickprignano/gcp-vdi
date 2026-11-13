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
  description = "GCP zone where desktop VMs are launched."
  type        = string
  default     = "us-central1-a"
}

variable "deployment_mode" {
  description = "gateway (systemd/container on Tier 1 e2-micro) or cloud_run (Tier 2/3 serverless)."
  type        = string
  default     = "gateway"

  validation {
    condition     = contains(["gateway", "cloud_run"], var.deployment_mode)
    error_message = "deployment_mode must be 'gateway' or 'cloud_run'."
  }
}

variable "instance_template_self_link" {
  description = "Self-link of the desktop GCE instance template."
  type        = string
}

variable "desktop_service_account_email" {
  description = "Service account email used by desktop VMs (for iam.serviceAccountUser grant)."
  type        = string
}

variable "rustdesk_host" {
  description = "RustDesk hbbs/hbbr host or IP."
  type        = string
}

variable "broker_image" {
  description = "Container image for gcp-vdi-broker (used in cloud_run mode)."
  type        = string
  default     = "ghcr.io/nickprignano/gcp-vdi-broker:latest"
}

variable "enable_firestore" {
  description = "Grant roles/datastore.user for Firestore state (Tier 3)."
  type        = bool
  default     = false
}

variable "enable_browser_fallback" {
  description = "Whether Guacamole browser fallback is enabled."
  type        = bool
  default     = false
}
