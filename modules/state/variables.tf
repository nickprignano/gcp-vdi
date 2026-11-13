variable "project_id" {
  description = "GCP project ID for state resources."
  type        = string
}

variable "region" {
  description = "GCP region for bucket, KMS keyring, and Firestore."
  type        = string
  default     = "us-central1"
}

variable "name_prefix" {
  description = "Prefix for state resource names."
  type        = string
  default     = "gcp-vdi"
}

variable "enable_cmek" {
  description = "Provision a Cloud KMS key for CMEK encryption (Tier 3)."
  type        = bool
  default     = false
}

variable "enable_firestore" {
  description = "Provision a Firestore Native database for distributed broker state (Tier 3)."
  type        = bool
  default     = false
}

variable "force_destroy_bucket" {
  description = "Allow Terraform destroy to delete non-empty GCS home-dir bucket (recommended false in prod)."
  type        = bool
  default     = false
}
