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
  description = "GCP zone."
  type        = string
  default     = "us-central1-a"
}

variable "domain" {
  description = "Team portal domain (leave empty for sslip.io)."
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
  description = "Allowed Google Workspace / user email addresses."
  type        = list(string)
}

variable "broker_register_secret" {
  description = "Shared bearer secret for desktop & on-prem device registration."
  type        = string
  sensitive   = true
}

variable "desktop_source_image" {
  description = "Packer golden image family or base Ubuntu image."
  type        = string
  default     = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
}

variable "desktop_machine_type" {
  description = "Team desktop machine type."
  type        = string
  default     = "e2-standard-4"
}

variable "enable_browser_fallback" {
  description = "Enable optional Apache Guacamole browser fallback."
  type        = bool
  default     = false
}
