<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/brand/logo/gcp-vdi-wordmark-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="docs/brand/logo/gcp-vdi-wordmark-light.svg">
    <img alt="gcp-vdi — Terraform-first Cloud VDI on Google Cloud" src="docs/brand/logo/gcp-vdi-wordmark-dark.svg" width="320">
  </picture>
</p>

Open-source, Terraform-first, tiered VDI platform on Google Cloud — powered by **self-hosted RustDesk (`hbbs` + `hbbr`, `-k _` keypair enforcement)** with optional Apache Guacamole browser fallback and hybrid on-prem node support.

See [docs/architecture.md](docs/architecture.md) for the full architecture, ADRs, and per-tier cost model, and [docs/brand/BRAND.md](docs/brand/BRAND.md) for the visual brand and XFCE desktop specification.

## Tiers

1. **Tier 1 — Lab (`examples/lab`)**: Always-Free `e2-micro` gateway (`hbbs`/`hbbr` + Caddy auto-TLS + `oauth2-proxy` + Go broker) + Spot `e2-standard-4` desktops with GCS `rclone` home-dir sync (`~$0.40/mo` idle, `~$2.00/mo` @ 40h active).
2. **Tier 2 — Team (`examples/team`)**: Dedicated RustDesk relay, Cloud Run broker, persistent `pd-balanced` disks, Packer golden images, and on-prem hybrid node registration.
3. **Tier 3 — Enterprise (`examples/enterprise`)**: Shared VPC (Host + Service project), Cloud Armor WAF + IAP, Cloud NAT private-only desktops, CMEK, Firestore, and opt-in RustDesk Pro / GPU / Windows BYOL.

## Quickstart (Tier 1 Lab)

```bash
cd examples/lab
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your project_id, OAuth client, and allowed_emails
terraform init
terraform apply
```

After `terraform apply`, add the printed `oauth_redirect_uri` to your Google Cloud OAuth2 Web Client, then open `gateway_url` in your browser.
