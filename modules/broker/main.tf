resource "random_password" "register_secret" {
  length  = 32
  special = false
}

resource "google_service_account" "broker" {
  project      = var.project_id
  account_id   = "gcp-vdi-broker"
  display_name = "gcp-vdi Lifecycle Broker SA"
}

resource "google_project_iam_member" "broker_compute" {
  project = var.project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:${google_service_account.broker.email}"
}

resource "google_service_account_iam_member" "broker_act_as_desktop" {
  service_account_id = "projects/${var.project_id}/serviceAccounts/${var.desktop_service_account_email}"
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.broker.email}"
}

resource "google_project_iam_member" "broker_firestore" {
  count   = var.enable_firestore ? 1 : 0
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.broker.email}"
}

resource "google_cloud_run_v2_service" "broker" {
  count    = var.deployment_mode == "cloud_run" ? 1 : 0
  project  = var.project_id
  name     = "gcp-vdi-broker"
  location = var.region

  template {
    service_account = google_service_account.broker.email
    scaling {
      min_instance_count = 0
      max_instance_count = 5
    }
    containers {
      image = var.broker_image
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "GCP_ZONE"
        value = var.zone
      }
      env {
        name  = "GCE_INSTANCE_TEMPLATE"
        value = var.instance_template_self_link
      }
      env {
        name  = "RUSTDESK_HOST"
        value = var.rustdesk_host
      }
      env {
        name  = "BROKER_REGISTER_SECRET"
        value = random_password.register_secret.result
      }
      env {
        name  = "ENABLE_BROWSER_FALLBACK"
        value = tostring(var.enable_browser_fallback)
      }
    }
  }
}
