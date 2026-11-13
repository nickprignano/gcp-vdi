# `gcp-vdi` — Architecture & ADR Specification

> **Status:** Locked for Implementation  
> **Scope:** Open-source, Terraform-first, tiered VDI platform on Google Cloud with RustDesk-first low-latency remoting and optional on-prem hybrid node support.

---

## 1. Design Philosophy & Core Protocol Strategy

`gcp-vdi` is a **progressive, composable VDI platform**. A solo developer or homelab user can deploy **Tier 1 (`lab`)** for near-\$0/month using Google Cloud's Always-Free `e2-micro` tier and Spot VMs, while teams and enterprises can opt into **Tier 2 (`team`)** and **Tier 3 (`enterprise`)** capabilities—without rewriting the core mental model or carrying enterprise bloat into Tier 1.

### Core Protocol Strategy
1. **Primary (Native, Low-Latency): Self-Hosted RustDesk (`hbbs` + `hbbr`)**
   - Enforces asymmetric keypair authentication (`-k _`) so only clients possessing the public key (`id_ed25519.pub`) can register or relay through the server.
   - Desktops **never expose inbound RDP/VNC ports** to the internet (outbound-only registration to `hbbs`/`hbbr`).
   - Works uniformly across **Linux, Windows, GPU VMs, and on-prem hosts** (e.g., Proxmox VMs, workstations).
2. **Fallback (Browser-Only): Optional Apache Guacamole**
   - Disabled by default in Tier 1 (`enable_browser_fallback = false`) to keep the `e2-micro` gateway lean (`<200 MB` RAM).
   - Opt-in (`guacd` + `guacamole`) for locked-down corporate/travel devices where a native RustDesk client cannot be installed.

---

## 2. Tiered Architecture Overview

### Tier 1 — Lab / Near-\$0 (`lab`)

Designed to stay within the GCP Always-Free envelope for the control plane and pay-per-minute Spot pricing only while a desktop is actively used.

```text
User RustDesk Client ──TCP 21115-21119 / UDP 21116 (-k _)──▶ [e2-micro Gateway: us-central1/east1/west1]
User Browser ──────────HTTPS (443)─────────────────────────▶   ├── Caddy (Auto-TLS via domain or sslip.io)
                                                               ├── oauth2-proxy (Google OAuth, email allowlist)
                                                               ├── gcp-vdi-broker (Go binary, systemd)
                                                               ├── hbbs + hbbr (RustDesk ID & Relay, -k _)
                                                               └── [Optional] guacd + guacamole (disabled by default)
                                                                      │
                                                                      ├── Compute Engine API (Spot VM create/delete)
                                                                      └── GCS API (gs://<bucket>/users/<email_hash>/)
                                                                      │
                                                                      ▼ (Outbound-only registration; NO inbound ports)
                                                             [Spot VM: e2-standard-4]
                                                               ├── Ubuntu + XFCE + RustDesk client (auto-registered)
                                                               ├── Ephemeral per-boot password reported to broker
                                                               ├── Ephemeral boot disk (auto-delete)
                                                               ├── rclone sync (boot, 5m timer, Spot SIGTERM)
                                                               └── Idle reaper (self-delete after N min idle)
```

- **Network:** Single VPC, single subnet, ephemeral external IP on gateway and desktop (zero Cloud Load Balancer, zero Cloud NAT). VPC firewall **denies all inbound** to desktops except IAP SSH (`35.235.240.0/20`) and internal `3389` from gateway *only if* `enable_browser_fallback = true`. Gateway allows `443`/`80` (Caddy) and `21115-21119/tcp` + `21116/udp` (`hbbs`/`hbbr`).
- **State:** Single GCS bucket with per-user prefixes (`gs://<bucket>/users/<sha256_email>/`) and local SQLite state on the gateway's 30 GB Always-Free `pd-standard` disk.
- **Auth & Connect UX:** User logs into the Caddy/OAuth2-protected web portal (`https://<ip>.sslip.io`), clicks **Wake Desktop**, and receives the RustDesk ID + broker-provisioned ephemeral per-boot password (or `rustdesk://<id>` deep link).

---

### Tier 2 — Team / Pro (`team`)

Designed for small-to-medium teams needing Google Workspace group authorization, ~15s VM wake with persistent disks, golden images, and hybrid on-prem node support.

- **Auth & Access:** Google Workspace / Cloud Identity group-based RBAC (verified by `oauth2-proxy` + Go broker).
- **Control Plane:** Optional **Cloud Run** deployment for `gcp-vdi-broker` (scale-to-zero, HTTPS) alongside a dedicated `rustdesk` relay VM (`e2-small`/`e2-medium`).
- **Compute & Storage:** Persistent per-user `pd-balanced` disks attached to start/stop VMs for ~15s wake times without losing local state.
- **Hybrid On-Prem Nodes:** On-prem Linux, Windows, or Proxmox VMs install the RustDesk client configured with the team's `hbbs`/`hbbr` host and `-k _` public key, registering into the same OAuth-gated device directory.
- **Golden Images & CI/CD:** Packer templates built via **GitHub Actions + Workload Identity Federation** (primary) with optional **Cloud Build** triggers.

---

### Tier 3 — Enterprise (`enterprise`)

Designed for regulated or large-scale environments requiring private-only desktops, centralized ingress, KMS encryption, and auditability.

- **Ingress & Zero Trust:** External HTTPS Load Balancer + **Identity-Aware Proxy (IAP)** + **Cloud Armor** WAF for the control plane and optional Guacamole web fallback.
- **Dedicated / HA RustDesk Relay:** High-availability `hbbs`/`hbbr` MIG behind a Network Load Balancer, with **opt-in RustDesk Server Pro** (`enable_rustdesk_pro = true`) for native OIDC, centralized address book, 2FA, and relay audit logs.
- **Network & Tenancy:** **Multi-project Shared VPC** (Host project owns VPC, LB, Cloud NAT, Cloud DNS, Cloud Armor; Service project(s) own VDI VMs, KMS keys, disks, images), with a `single_project_mode = true` fallback for sandbox evaluation.
- **Egress & Hybrid:** Private-only desktops (zero external IPs) behind **Cloud NAT** and private DNS. On-prem nodes connect via `-k _` public `hbbs`/`hbbr` or optional **Tailscale / Cloud VPN**.
- **Control-Plane State & Compliance:** Persistent **Firestore** (default) or **Cloud SQL** (PostgreSQL) backend; **CMEK** (Cloud KMS) encryption; **VPC Service Controls (VPC-SC)** compatibility; **Cloud Logging / OpenTelemetry** lifecycle & session audit trail.
- **Enterprise Add-on Modules (Opt-in):**
  - **Warm Pools:** Pre-booted standby VMs for instant (<5s) login assignment.
  - **GPU Workstations:** `g2-standard-*` (NVIDIA L4) or `n1-standard-*` + T4 (Spot or On-Demand) with hardware-accelerated RustDesk encoding.
  - **Windows BYOL / Sole-Tenant Nodes:** Sole-tenant node groups with affinity labels and BYOL image import.

---

## 3. Architectural Decision Records (ADRs)

### ADR-001: Go for the Lifecycle Broker (`gcp-vdi-broker`)
- **Context:** Tier 1 runs on an `e2-micro` (1 GB RAM) alongside `hbbs`, `hbbr`, Caddy, `oauth2-proxy`, and optional Guacamole. Tier 2/3 optionally run the broker on Cloud Run.
- **Decision:** Build the lifecycle broker in **Go** as a single static binary.
- **Consequences:** `~15 MB` RSS footprint leaves ample headroom on `e2-micro` even when optional Guacamole is enabled, and compiles to a `distroless` Cloud Run container with `<150 ms` cold starts.

### ADR-002: RustDesk (`hbbs` + `hbbr`, `-k _`) Primary, Guacamole Optional Fallback
- **Context:** Traditional RDP/VNC gateways require opening inbound `3389`/`5900` across VPCs and struggle with high-FPS/GPU/on-prem workloads.
- **Decision:** Standardize on self-hosted **RustDesk (`hbbs` + `hbbr`)** with mandatory keypair enforcement (`-k _`) as the primary protocol across all tiers, keeping **Apache Guacamole** as an opt-in browser fallback (disabled by default in Tier 1).
- **Consequences:** Zero inbound ports on desktop VMs, uniform UX across Linux/Windows/GPU/on-prem, and ultra-low memory usage on Tier 1 `e2-micro`.

### ADR-003: RustDesk OSS Default with Tier 3 Pro Opt-In
- **Context:** RustDesk OSS lacks built-in OIDC and a multi-user web address book; RustDesk Server Pro provides these natively but requires a commercial license above 10 devices.
- **Decision:** Default to **RustDesk OSS** across all three tiers, using `oauth2-proxy` + `gcp-vdi-broker` as the OAuth-gated device directory and wake portal. Expose **RustDesk Server Pro** as an opt-in Tier 3 configuration (`enable_rustdesk_pro = true`).
- **Consequences:** 100% open-source, zero-license-cost deployments in Tier 1–3 by default, with a clean upgrade path for enterprises that license RustDesk Pro.

### ADR-004: Broker-Managed Ephemeral Per-Boot Password
- **Context:** Ephemeral Spot VMs in Tier 1 need secure, per-session RustDesk credentials without static shared passwords.
- **Decision:** On boot, the desktop cloud-init/startup script generates a cryptographically random password, sets it via `rustdesk --password`, and registers the `{vm_name, rustdesk_id, ephemeral_password}` tuple with `gcp-vdi-broker` over an IAM-authenticated/internal call. Only the OAuth-authenticated owner can retrieve it from the broker UI.
- **Consequences:** Zero static desktop passwords in Terraform state, metadata, or GCS.

### ADR-005: Public `hbbs`/`hbbr` (`-k _`) for Hybrid On-Prem Nodes, Optional Tailscale/VPN
- **Context:** Tier 2/3 users may want on-prem Linux, Windows, or Proxmox nodes accessible alongside GCP desktops.
- **Decision:** Allow on-prem nodes to register directly to the GCP-hosted `hbbs`/`hbbr` endpoints (`21115-21119/tcp`, `21116/udp`) protected by `-k _` keypair enforcement, with **Tailscale / Cloud VPN** as an opt-in Tier 3 module for strict private-relay requirements.
- **Consequences:** Zero-config hybrid onboarding (install RustDesk + drop `hbbs` host & public key) while preserving an air-gapped option for regulated enterprises.

### ADR-006: GitHub Actions + WIF Primary CI/CD, Optional Cloud Build
- **Decision:** Use **GitHub Actions + Workload Identity Federation** as the primary CI/CD pipeline for Terraform validation and Packer golden-image builds, with optional **Cloud Build** triggers for all-GCP environments.

### ADR-007: Multi-Project Shared VPC Default for Tier 3 with Single-Project Fallback
- **Decision:** Design Tier 3 around **Host + Service project Shared VPC**, controlled via `enable_shared_vpc` / `host_project_id` variables that fall back to single-project mode for sandbox evaluation.

### ADR-008: `sslip.io` Zero-Config TLS Fallback & Single GCS Bucket Prefixing (Tier 1)
- **Decision:** Default Tier 1 Caddy TLS to `<gateway-ip>.sslip.io` when `domain = ""`, and store all user home directories in a single GCS bucket under `gs://<bucket>/users/<sha256_email>/`.

### ADR-009: Automated Operations ("Zero-Ops" Design)
- **Context:** Self-managed VDI typically accumulates ops toil from OS patching, image updates, and orphaned cloud resources.
- **Decision:** Automate day-2 operations out of the box:
  1. **Self-Updating Gateway:** `unattended-upgrades` + scheduled container refresh timer on Tier 1; Container-Optimized OS / MIG autohealing + serverless Cloud Run broker in Tier 2/3.
  2. **Image-Family Rolling Updates:** Weekly scheduled Packer builds (with RDP/RustDesk smoke tests) publish to GCE **Image Families** and prune old images; desktops always boot from the family `latest`.
  3. **Broker Garbage Collection:** Built-in background sweeper reconciles GCE instances/disks against active state to clean up orphaned Spot VMs and stale disks, paired with the desktop idle reaper and Spot `SIGTERM` `rclone` flush.
  4. **Built-In Observability:** Pre-wired Cloud Monitoring uptime checks, structured Cloud Logging, and alert hooks in Terraform.

---

## 4. Per-Tier Cost Model (Estimated Monthly USD, `us-central1`)

> **Assumptions:**
> - **Active usage:** 40 hrs/month for Lab; 160 hrs/month (full-time, 8h × 20d) for Team/Enterprise.
> - **Desktop baseline:** `e2-standard-4` (4 vCPU, 16 GB RAM) — Spot `~$0.040/hr`, On-Demand `~$0.134/hr`.
> - **Shared infra** (Gateway/Relay, LB, Cloud NAT, Cloud Armor, DB) is amortized per user (10 users in Team, 50 users in Enterprise).
> - Prices are illustrative `us-central1` list estimates; actuals vary by CUDs, egress, and region.

| Cost Component | Tier 1 — Lab (1 user) | Tier 2 — Team (10 users, /user) | Tier 3 — Enterprise (50 users, /user) |
| :--- | :--- | :--- | :--- |
| **Control Plane + RustDesk Relay** | `$0.00` (Always-Free `e2-micro` + 30 GB `pd-standard`) | `~$2.50` (`e2-small` relay `$14` + Cloud Run `~$1` + DNS, ÷ 10) | `~$1.60` (NLB/HTTPS LB `$25` + Armor `$6` + NAT `$32` + Relay MIG `$14` + Firestore/SQL `$5`, ÷ 50) |
| **User Storage (Idle)** | `~$0.40` (20 GB GCS Standard @ `$0.02/GB`) | `~$5.00` (50 GB `pd-balanced` @ `$0.10/GB`) | `~$10.00` (100 GB `pd-balanced` + CMEK snapshots) |
| **Idle Total / User / Month** | **`~$0.40`** | **`~$7.50`** | **`~$11.60`** |
| **Active Compute (Linux)** | `~$1.60` (40 hrs Spot `e2-standard-4` @ `$0.04/hr`) | `~$6.40` (160 hrs Spot) to `~$21.44` (160 hrs On-Demand) | `~$21.44` (160 hrs On-Demand `e2-standard-4`) |
| **Active Total / User / Month (Linux)** | **`~$2.00`** *(40h Spot)* | **`~$13.90`** *(Spot)* / **`~$28.94`** *(On-Demand)* | **`~$33.04`** *(On-Demand)* |
| **On-Prem Hybrid Node** | `$0.00` (uses existing `hbbs`/`hbbr`) | `$0.00` compute (relay traffic only) | `$0.00` compute (relay traffic only) |
| **Optional: GPU Add-on (`g2-standard-4` L4)** | N/A | `+$40–$50` *(160h Spot L4)* | `+$113.60` *(160h On-Demand L4)* |
| **Optional: Windows SPLA / BYOL** | `+$7.36` *(40h SPLA)* | `+$29.44` *(160h SPLA)* | `~$50–$90` *(Sole-Tenant node amortized across 25–40 VMs)* |
| **Optional: Warm Pool (2 standby VMs)** | N/A | N/A | `+$3.85/user` *(2 × `e2-standard-4` On-Demand ÷ 50 users)* |

---

## 5. Modular Terraform Layout

```text
gcp-vdi/
├── docs/
│   └── architecture.md              # This specification & ADR document
├── broker/                          # Go lifecycle broker + RustDesk directory + orphan sweeper
│   ├── cmd/broker/main.go
│   ├── internal/
│   │   ├── api/                     # HTTP handlers (wake, stop, status, password handoff, device register)
│   │   ├── compute/                 # GCE instance lifecycle, warm pool & orphan sweeper
│   │   ├── auth/                    # OAuth2 / IAP / Workspace group verification
│   │   └── store/                   # SQLite (Tier 1) / Firestore / Postgres (Tier 3)
│   ├── go.mod
│   └── Dockerfile                   # Distroless multi-stage build for Cloud Run
├── packer/                          # Golden image definitions (Tier 2+)
│   ├── linux-xfce/                  # Ubuntu + XFCE + RustDesk (-k _) + rclone + optional xrdp
│   ├── linux-gpu/                   # Ubuntu + NVIDIA drivers + RustDesk hardware encoding
│   └── windows-rdp/                 # Windows Server/11 + RustDesk + Sysprep
├── .github/workflows/               # CI: terraform fmt/validate/tflint + Packer WIF weekly builds
├── modules/
│   ├── network/                     # VPC, subnets, firewall (deny-all desktop inbound, 21115-21119 gateway), Shared VPC, NAT, DNS
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── versions.tf
│   ├── state/                       # GCS home-dir bucket, optional CMEK KMS keys, Firestore / Cloud SQL
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── versions.tf
│   ├── rustdesk/                    # hbbs + hbbr (-k _ keypair Secret/GCS), optional HA MIG/NLB, optional Pro license
│   │   ├── main.tf
│   │   ├── templates/               # docker-compose / systemd unit for hbbs + hbbr
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── versions.tf
│   ├── broker/                      # IAM SA, systemd unit (Tier 1) OR Cloud Run service (Tier 2/3)
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── versions.tf
│   ├── desktop-image/               # Instance templates, Spot/On-Demand/GPU/Sole-Tenant, RustDesk auto-reg + rclone + reaper
│   │   ├── main.tf
│   │   ├── scripts/                 # rustdesk-boot.sh, rclone-sync.sh, idle-reaper.sh, windows-init.ps1
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── versions.tf
│   └── gateway/                     # e2-micro/MIG gateway, Caddy, oauth2-proxy, optional Guacamole, optional LB+IAP+Armor
│       ├── main.tf
│       ├── templates/               # cloud-init, Caddyfile, guacamole.properties
│       ├── variables.tf
│       ├── outputs.tf
│       └── versions.tf
└── examples/
    ├── lab/                         # Tier 1: Always-Free e2-micro (hbbs/hbbr + Caddy + oauth2-proxy + broker) + Spot + GCS rclone
    │   ├── main.tf
    │   ├── variables.tf
    │   ├── outputs.tf
    │   └── terraform.tfvars.example
    ├── team/                        # Tier 2: Workspace groups + Cloud Run broker + persistent pd-balanced + hybrid on-prem nodes
    │   ├── main.tf
    │   ├── variables.tf
    │   ├── outputs.tf
    │   └── terraform.tfvars.example
    └── enterprise/                  # Tier 3: Shared VPC + LB/IAP/Armor + HA RustDesk/Pro + NAT + CMEK + Firestore + Warm Pool/GPU/Windows
        ├── main.tf
        ├── variables.tf
        ├── outputs.tf
        └── terraform.tfvars.example
```

### Module Dependency Graph & Key Contracts

```text
  [state] ──────────────────────────────────────┐
     │ (bucket, kms_key, db_dsn)                │
     ▼                                          ▼
  [network] ──▶ [rustdesk] ──▶ [desktop-image] ──▶ [broker] ──▶ [gateway]
   (vpc,         (hbbs_host,    (instance_          (broker_     (gateway_url,
    subnet,       pub_key)       template,           url,         lb_ip)
    nat)                         sa_email)           sa_email)
```

- **`modules/network`**: Outputs `network_id`, `subnetwork_id`, `host_project_id`, `service_project_id`, `desktop_network_tag`, `gateway_network_tag`. Enforces deny-all inbound on desktops (plus IAP SSH `35.235.240.0/20`) and opens `21115-21119/tcp` + `21116/udp` on the RustDesk relay/gateway.
- **`modules/state`**: Outputs `home_bucket_name`, `kms_key_id`, `state_backend_uri`.
- **`modules/rustdesk`**: Outputs `hbbs_endpoint`, `hbbr_endpoint`, `rustdesk_public_key`, `compose_snippet` (embedded into `gateway` in Tier 1, or standalone MIG/NLB in Tier 2/3).
- **`modules/desktop-image`**: Outputs `instance_template_self_link`, `desktop_service_account_email`. Auto-configures RustDesk (`-k _` public key + `hbbs` host + ephemeral per-boot password registration to broker).
- **`modules/broker`**: Outputs `broker_url`, `broker_service_account_email`. Deploys as a Go systemd unit on `gateway` (`deployment_mode = "gateway"`) or on **Cloud Run** (`deployment_mode = "cloud_run"`).
- **`modules/gateway`**: Outputs `gateway_url`, `gateway_ip`, `gateway_service_account_email`. Deploys `e2-micro` (Tier 1) or MIG + External HTTPS LB + IAP + Cloud Armor (Tier 3), with `enable_browser_fallback = false` by default.
