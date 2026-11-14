variable "project_id" {
  description = "GCP project ID for desktop instances."
  type        = string
}

variable "region" {
  description = "GCP region."
  type        = string
  default     = "us-central1"
}

variable "subnetwork_id" {
  description = "Subnetwork self-link for desktop VMs."
  type        = string
}

variable "desktop_tag" {
  description = "Network tag applied to desktop VMs."
  type        = string
  default     = "gcp-vdi-desktop"
}

variable "home_bucket_name" {
  description = "GCS bucket name for rclone home-dir sync."
  type        = string
}

variable "rustdesk_host" {
  description = "RustDesk hbbs/hbbr host/IP."
  type        = string
}

variable "broker_register_url" {
  description = "Broker /api/v1/devices/register URL."
  type        = string
}

variable "broker_register_secret" {
  description = "Shared bearer secret for desktop-to-broker registration."
  type        = string
  sensitive   = true
}

variable "os_family" {
  description = "Desktop OS family: linux or windows."
  type        = string
  default     = "linux"

  validation {
    condition     = contains(["linux", "windows"], var.os_family)
    error_message = "os_family must be 'linux' or 'windows'."
  }
}

variable "source_image" {
  description = "Source image or image family (e.g., ubuntu-os-cloud/ubuntu-2404-lts-amd64)."
  type        = string
  default     = "ubuntu-os-cloud/ubuntu-2404-lts-amd64"
}

variable "machine_type" {
  description = "Desktop GCE machine type."
  type        = string
  default     = "e2-standard-4"
}

variable "provisioning_model" {
  description = "SPOT (Tier 1/2) or STANDARD (Tier 2/3)."
  type        = string
  default     = "SPOT"
}

variable "assign_public_ip" {
  description = "Assign ephemeral external IP to desktops (true in Tier 1/2 without Cloud NAT; false in Tier 3)."
  type        = bool
  default     = true
}

variable "boot_disk_size_gb" {
  description = "Boot disk size in GB."
  type        = number
  default     = 40
}

variable "boot_disk_type" {
  description = "Boot disk type (pd-standard or pd-balanced)."
  type        = string
  default     = "pd-balanced"
}

variable "kms_key_id" {
  description = "Optional CMEK key ID for disk encryption (Tier 3)."
  type        = string
  default     = null
}

variable "gpu_type" {
  description = "Optional GPU accelerator type (e.g., nvidia-tesla-t4) for n1/g2 workstations."
  type        = string
  default     = null
}

variable "gpu_count" {
  description = "Number of GPUs when gpu_type is set."
  type        = number
  default     = 1
}

variable "sole_tenant_node_group" {
  description = "Optional Sole-Tenant node group name for Windows BYOL (Tier 3)."
  type        = string
  default     = null
}

variable "idle_timeout_minutes" {
  description = "Minutes of idle before desktop self-deletes/stops."
  type        = number
  default     = 45
}

variable "enable_browser_fallback" {
  description = "Install xrdp on Linux desktops for Guacamole browser access."
  type        = bool
  default     = true
}
