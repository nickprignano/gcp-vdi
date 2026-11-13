resource "random_id" "suffix" {
  byte_length = 3
}

resource "google_kms_key_ring" "this" {
  count    = var.enable_cmek ? 1 : 0
  project  = var.project_id
  name     = "${var.name_prefix}-keyring-${random_id.suffix.hex}"
  location = var.region
}

resource "google_kms_crypto_key" "this" {
  count           = var.enable_cmek ? 1 : 0
  name            = "${var.name_prefix}-cmek"
  key_ring        = google_kms_key_ring.this[0].id
  rotation_period = "7776000s" # 90 days
}

data "google_storage_project_service_account" "gcs" {
  count   = var.enable_cmek ? 1 : 0
  project = var.project_id
}

resource "google_kms_crypto_key_iam_member" "gcs_cmek" {
  count         = var.enable_cmek ? 1 : 0
  crypto_key_id = google_kms_crypto_key.this[0].id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${data.google_storage_project_service_account.gcs[0].email_address}"
}

resource "google_storage_bucket" "homes" {
  project                     = var.project_id
  name                        = "${var.name_prefix}-homes-${random_id.suffix.hex}"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = var.force_destroy_bucket

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      num_newer_versions = 3
    }
    action {
      type = "Delete"
    }
  }

  dynamic "encryption" {
    for_each = var.enable_cmek ? [1] : []
    content {
      default_kms_key_name = google_kms_crypto_key.this[0].id
    }
  }

  depends_on = [google_kms_crypto_key_iam_member.gcs_cmek]
}

resource "google_firestore_database" "this" {
  count       = var.enable_firestore ? 1 : 0
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
}
