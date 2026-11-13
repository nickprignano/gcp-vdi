variable "project_id" {
  description = "Service project ID where VDI compute resources live."
  type        = string
}

variable "host_project_id" {
  description = "Host project ID for Shared VPC (defaults to project_id when enable_shared_vpc is false)."
  type        = string
  default     = ""
}

variable "enable_shared_vpc" {
  description = "Enable Shared VPC (Tier 3). Requires Org/Folder Shared VPC admin permissions."
  type        = bool
  default     = false
}

variable "network_name" {
  description = "Name of the VPC network."
  type        = string
  default     = "gcp-vdi-vpc"
}

variable "region" {
  description = "GCP region for the subnet and optional Cloud NAT."
  type        = string
  default     = "us-central1"
}

variable "subnet_cidr" {
  description = "CIDR block for the VDI subnet."
  type        = string
  default     = "10.42.0.0/20"
}

variable "enable_cloud_nat" {
  description = "Provision Cloud Router + Cloud NAT for private-only desktops (Tier 3)."
  type        = bool
  default     = false
}

variable "enable_browser_fallback" {
  description = "Allow internal TCP 3389 from gateway to desktops for optional Guacamole fallback."
  type        = bool
  default     = false
}

variable "gateway_tag" {
  description = "Network tag for gateway / RustDesk relay instances."
  type        = string
  default     = "gcp-vdi-gateway"
}

variable "desktop_tag" {
  description = "Network tag for VDI desktop instances."
  type        = string
  default     = "gcp-vdi-desktop"
}
