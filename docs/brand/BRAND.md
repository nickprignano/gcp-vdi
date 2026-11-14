# `gcp-vdi` Visual Brand & XFCE Desktop Specification

Visual identity, color tokens, typography, repository integration snippets, and Ubuntu [XFCE](https://www.xfce.org/) desktop theme guidelines for **[`gcp-vdi`](https://github.com/nickprignano/gcp-vdi)** — a [Terraform](https://registry.terraform.io/providers/hashicorp/google/latest/docs)-first, tiered cloud VDI platform on Google Cloud ([Spot VMs](https://docs.cloud.google.com/compute/docs/instances/spot), 1-click Google OAuth + [Apache Guacamole](https://guacamole.apache.org/doc/gug/) browser SSO, self-hosted [RustDesk](https://rustdesk.com/docs/en/self-host/) relay, and [Cloud Storage FUSE (`gcsfuse`)](https://docs.cloud.google.com/storage/docs/cloud-storage-fuse/overview) home-directory persistence).

## 1. Brand Mark & Asset Inventory

The `gcp-vdi` mark is built on an integer `64x64` geometric grid (`4u = 1px` at `16x16`, `2u = 1px` at `32x32`) combining three visual primitives:

- **Cloud Canopy (`#3b82f6`)**: Geometric three-lobe cloud silhouette representing tiered Google Cloud infrastructure managed as code.
- **VDI Viewport (`#0b1120` / `#111827`)**: Crisp floating desktop window separated from the cloud canopy by a `3.8u` negative-space knockout moat so the silhouette never blurs at `16px–32px`.
- **Session Prompt & Ready Node (`#f3f4f6` + `#10b981`)**: Terminal chevron (`>`) paired with an emerald underscore and status indicator matching the web portal's `READY` instance state.

All standalone wordmark SVGs convert glyphs into pure vector `<path>` outlines so GitHub `README.md` embeds render identically across Linux, macOS, and Windows without external font requests.

| Asset File | Dimensions / Format | Primary Placement |
| :--- | :--- | :--- |
| `gcp-vdi-icon.svg` / `gcp-vdi-icon.png` | `64x64` viewBox / `512x512` RGBA PNG | Standalone transparent mark for docs, launcher icons, and avatars |
| `gcp-vdi-wordmark-dark.svg` / `.png` | Vector / `1074x324` RGBA PNG | Web portal dark surfaces and GitHub dark-mode `README.md` |
| `gcp-vdi-wordmark-light.svg` / `.png` | Vector / `1074x324` RGBA PNG | Light-mode documentation and GitHub light-mode `README.md` |
| `portal-header-badge.svg` / `.png` | `64x64` viewBox / `256x256` RGBA PNG | Web portal `<header>` navbar brand tile (`#111827` fill, `#1f2937` border) |
| `favicon.svg` / `favicon.ico` | Vector / `16x16`, `32x32`, `48x48` ICO | Web portal browser tab icon (dark-slate squircle tile for light/dark tab contrast) |
| `wallpaper-dark.svg` / `1920x1080` / `2560x1440` | Vector + FHD & QHD RGB PNG | Default Ubuntu XFCE desktop wallpaper (low-contrast left icon zone) |
| `wallpaper-light.svg` / `1920x1080` / `2560x1440` | Vector + FHD & QHD RGB PNG | Optional daylight Ubuntu XFCE desktop wallpaper |
| `github-social-banner.svg` / `-1280x640.png` | `1280x640` (2:1 OpenGraph) | GitHub repository Social Preview (`Settings -> General -> Social preview`) & README hero |

## 2. Color System & Contrast Tokens

### Dark-First Portal & Desktop Palette (Default)

| Role | Token | Hex | RGB | WCAG Contrast (vs `#0b1120`) | Usage |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Canvas Background | `--vdi-bg` | `#0b1120` | `11, 17, 32` | — | Portal page background, terminal viewport, dark wallpaper base |
| Card Surface | `--vdi-surface` | `#111827` | `17, 24, 39` | `1.06:1` (elevation) | Instance tier cards, header badge, modal surfaces |
| Subtle Border | `--vdi-border` | `#1f2937` | `31, 41, 55` | `1.36:1` (structural) | `1px` card hairlines, segmented controls, grid lines |
| Primary Accent | `--vdi-primary` | `#3b82f6` | `59, 130, 246` | `5.12:1` (AA) | Primary SSO CTA buttons, active tier border, cloud mark, wordmark hyphen |
| Accent Highlight | `--vdi-primary-hi` | `#60a5fa` | `96, 165, 250` | `7.64:1` (AAA) | Hover states, inline links, cloud canopy top gradient stop |
| Ready / Success | `--vdi-ready` | `#10b981` | `16, 185, 129` | `7.42:1` (AAA) | `READY` VM badge, active session pulse, mounted GCS volume status |
| Primary Text | `--vdi-text` | `#f3f4f6` | `243, 244, 246` | `17.11:1` (AAA) | Headings, primary UI labels, dark-mode wordmark (`16.12:1` on `#111827` cards) |
| Muted Text | `--vdi-text-muted` | `#9ca3af` | `156, 163, 175` | `7.42:1` (AAA) | Machine types (`n2-standard-8`), zone metadata, secondary hints |

### Light-Mode Companion Palette

| Role | Token | Hex | RGB | WCAG Contrast (vs `#f8fafc`) | Usage |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Daylight Canvas | `--vdi-bg-light` | `#f8fafc` | `248, 250, 252` | — | Light wallpaper base, light documentation background |
| Daylight Card | `--vdi-surface-light` | `#ffffff` | `255, 255, 255` | `1.04:1` | Elevated cards on `#f8fafc` / `#f1f5f9` |
| Daylight Border | `--vdi-border-light` | `#cbd5e1` | `203, 213, 225` | `1.48:1` | Crisp card borders and wallpaper grid lines |
| Primary Accent (Light) | `--vdi-primary-light` | `#2563eb` | `37, 99, 235` | `4.93:1` (AA) | High-contrast electric blue for light backgrounds |
| Ready (Light) | `--vdi-ready-light` | `#059669` | `5, 150, 105` | `4.54:1` (AA) | Accessible `READY` status text/border on light surfaces |
| Primary Text (Light) | `--vdi-text-light` | `#0f172a` | `15, 23, 42` | `17.06:1` (AAA) | Light-mode wordmark and primary text |

### Drop-In Portal CSS Custom Properties

```css
:root {
  --vdi-bg: #0b1120;
  --vdi-surface: #111827;
  --vdi-border: #1f2937;
  --vdi-primary: #3b82f6;
  --vdi-primary-hover: #60a5fa;
  --vdi-ready: #10b981;
  --vdi-ready-tint: rgba(16, 185, 129, 0.14);
  --vdi-text: #f3f4f6;
  --vdi-text-muted: #9ca3af;

  --vdi-font-sans: 'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --vdi-font-display: 'Inter Display', 'Space Grotesk', 'Inter', system-ui, sans-serif;
  --vdi-font-mono: 'JetBrains Mono', ui-monospace, 'SFMono-Regular', Menlo, Consolas, monospace;
}
```

## 3. Typography Specification

All three typefaces use the permissive **SIL Open Font License 1.1**, allowing local self-hosting in the web portal (no third-party CDN leaks under strict Content Security Policies) and pre-installation on Ubuntu XFCE images:

| Role | Recommended Family | Weights | Upstream Source & Package | Usage |
| :--- | :--- | :--- | :--- | :--- |
| UI & Portal Body | **[Inter](https://github.com/rsms/inter/)** ([Google Fonts](https://fonts.google.com/specimen/Inter)) | `400`, `500`, `600`, `700` | `fonts-inter` (Ubuntu APT) | Web portal controls, XFCE GTK system UI (`Inter Regular 10`), badges, tables |
| Brand & Display | **[Inter Display](https://github.com/rsms/inter/)** or **[Space Grotesk](https://github.com/floriankarsten/space-grotesk)** ([Google Fonts](https://fonts.google.com/specimen/Space+Grotesk)) | `600`, `700` | Bundled `.woff2` / SVG `<path>` | `gcp-vdi` lowercase wordmark, portal hero headers, marketing banners |
| Terminal & Code | **[JetBrains Mono](https://github.com/jetbrains/jetbrainsmono)** ([Google Fonts](https://fonts.google.com/specimen/JetBrains+Mono)) | `400`, `500`, `700` | `fonts-jetbrains-mono` (Ubuntu APT) | XFCE4 Terminal (`JetBrains Mono Regular 10.5`), VM hostnames, regions, RustDesk IDs, HCL snippets |

## 4. Ubuntu XFCE Desktop Theme & Icon Pairing

To make ephemeral Ubuntu XFCE sessions feel cohesive with the `#0b1120` / `#111827` / `#3b82f6` web portal immediately upon 1-click [Apache Guacamole](https://guacamole.apache.org/doc/gug/) or [RustDesk](https://rustdesk.com/docs/en/self-host/) login — while keeping memory overhead tiny and remote-desktop frame encoding fast (zero blur shaders or heavy animations):

### Recommended Primary Pairing (Zero External Build Needed on Ubuntu 22.04 / 24.04 LTS)

- **GTK3/GTK2 & XFWM4 Window Theme**: **`Arc-Dark`** via **[Arc Theme](https://github.com/arc-design/arc-theme)** (`apt install arc-theme`, actively maintained upstream at `jnsh/arc-theme`). Its flat cool-slate chrome (`#2f343f` / `#383c4a`) and `#5294e2` blue selection highlight pair cleanly with `#0b1120` wallpapers and compress cleanly over Guacamole WebSocket/RDP/VNC and RustDesk codecs.
  - *Light-mode counterpart*: **`Arc`** (included in the same `arc-theme` package) paired with `wallpaper-light`.
  - *Optional ultra-deep slate alternative*: **[Qogir-Dark](https://github.com/vinceliuice/qogir-theme)** (`vinceliuice/Qogir-theme`), which uses sharper titlebar geometry and deeper slate surfaces closer to `#111827`.
- **Icon Theme**: **`Papirus-Dark`** via **[Papirus Icon Theme](https://github.com/PapirusDevelopmentTeam/papirus-icon-theme)** (`apt install papirus-icon-theme`), customized with **[papirus-folders](https://github.com/PapirusDevelopmentTeam/papirus-folders)** set to `-C blue` (`Papirus-Dark` with electric-blue folders matching `#3b82f6`).
  - *Light-mode counterpart*: **`Papirus-Light`** (`papirus-folders -C blue --theme Papirus-Light`).
- **Cursor Theme**: **`Bibata-Modern-Classic`** (dark) / **`Bibata-Modern-Ice`** (light) via **[Bibata Cursor](https://github.com/ful1e5/Bibata_Cursor)** (`apt install bibata-cursor-theme` on Ubuntu 24.04+).
- **XFCE Compositor Tuning for VDI**: Keep window-shadow and menu opacity compositing **disabled** (`/general/use_compositing = false` in `xfwm4`) so remote frames stay razor-sharp with minimal bandwidth and CPU usage on [Spot VMs](https://docs.cloud.google.com/compute/docs/instances/spot).

### XFCE Terminal Palette (`~/.config/xfce4/terminal/terminalrc`)

```ini
[Configuration]
FontName=JetBrains Mono 10.5
ColorForeground=#f3f4f6
ColorBackground=#0b1120
ColorCursor=#10b981
ColorCursorUseDefault=FALSE
ColorSelection=#1f2937
ColorSelectionUseDefault=FALSE
ColorPalette=#111827;#ef4444;#10b981;#f59e0b;#3b82f6;#a855f7;#06b6d4;#e5e7eb;#4b5563;#f87171;#34d399;#fbbf24;#60a5fa;#c084fc;#22d3ee;#f9fafb
```

### Automated Golden-Image / Startup Provisioning Snippet

Drop this snippet into your Packer build or Terraform startup script (uses [Xfconf-query](https://docs.xfce.org/xfce/xfconf/xfconf-query) and system-wide XDG defaults so persistent [Cloud Storage FUSE](https://docs.cloud.google.com/storage/docs/cloud-storage-fuse/overview) `/home` directories inherit the brand defaults automatically):

```bash
#!/usr/bin/env bash
set -euo pipefail

# 1. Install lightweight XFCE theme, icon, cursor, and font packages
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y --no-install-recommends \
  arc-theme \
  papirus-icon-theme \
  fonts-inter \
  fonts-jetbrains-mono

# Optional: tint Papirus folders to electric blue (#3b82f6 match)
if command -v papirus-folders >/dev/null 2>&1; then
  papirus-folders -C blue --theme Papirus-Dark
fi

# 2. Install gcp-vdi wallpapers & launcher icon system-wide
install -d -m 0755 /usr/share/backgrounds/gcp-vdi /usr/share/pixmaps
install -m 0644 wallpaper-dark-2560x1440.png /usr/share/backgrounds/gcp-vdi/wallpaper-dark.png
install -m 0644 wallpaper-light-2560x1440.png /usr/share/backgrounds/gcp-vdi/wallpaper-light.png
install -m 0644 gcp-vdi-icon.svg /usr/share/pixmaps/gcp-vdi.svg

# 3. Apply session defaults when running inside an active XFCE user session
if command -v xfconf-query >/dev/null 2>&1 && [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
  xfconf-query -c xsettings -p /Net/ThemeName -n -t string -s "Arc-Dark"
  xfconf-query -c xsettings -p /Net/IconThemeName -n -t string -s "Papirus-Dark"
  xfconf-query -c xsettings -p /Gtk/FontName -n -t string -s "Inter 10"
  xfconf-query -c xsettings -p /Gtk/MonospaceFontName -n -t string -s "JetBrains Mono 10"
  xfconf-query -c xfwm4 -p /general/theme -n -t string -s "Arc-Dark"
  xfconf-query -c xfwm4 -p /general/title_font -n -t string -s "Inter SemiBold 10"
  xfconf-query -c xfwm4 -p /general/use_compositing -n -t bool -s false

  for prop in $(xfconf-query -c xfce4-desktop -l | grep "last-image$" || true); do
    xfconf-query -c xfce4-desktop -p "$prop" -s "/usr/share/backgrounds/gcp-vdi/wallpaper-dark.png"
  done
fi
```

## 5. GitHub `README.md` & Web Portal Integration Snippets

### Adaptive Dark/Light Wordmark in `README.md`

```html
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/brand/logo/gcp-vdi-wordmark-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="docs/brand/logo/gcp-vdi-wordmark-light.svg">
    <img alt="gcp-vdi — Terraform-first Cloud VDI on Google Cloud" src="docs/brand/logo/gcp-vdi-wordmark-dark.svg" width="320">
  </picture>
</p>
```

### Web Portal `<head>` Favicon & Header Badge

```html
<!-- Portal <head> -->
<link rel="icon" href="/favicon.ico" sizes="48x48">
<link rel="icon" href="/favicon.svg" type="image/svg+xml">

<!-- Portal <header> brand link -->
<a href="/" class="inline-flex items-center gap-2.5 text-[#f3f4f6] font-semibold tracking-tight">
  <img src="/portal-header-badge.svg" alt="" width="36" height="36" />
  <span>gcp<span class="text-[#3b82f6]">-</span>vdi</span>
</a>
```
