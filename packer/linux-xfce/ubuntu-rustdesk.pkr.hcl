packer {
  required_plugins {
    googlecompute = {
      source  = "github.com/hashicorp/googlecompute"
      version = ">= 1.1.4"
    }
  }
}

variable "project_id" {
  type = string
}

variable "zone" {
  type    = string
  default = "us-central1-a"
}

source "googlecompute" "linux_xfce" {
  project_id          = var.project_id
  source_image_family = "ubuntu-2404-lts-amd64"
  zone                = var.zone
  image_name          = "gcp-vdi-linux-xfce-{{timestamp}}"
  image_family        = "gcp-vdi-linux-xfce"
  ssh_username        = "packer"
  machine_type        = "e2-standard-4"
  use_iap             = true
  omit_external_ip    = false
}

build {
  sources = ["source.googlecompute.linux_xfce"]

  provisioner "file" {
    source      = "docs/brand/wallpapers/wallpaper-dark-2560x1440.png"
    destination = "/tmp/wallpaper-dark.png"
  }

  provisioner "file" {
    source      = "docs/brand/wallpapers/wallpaper-light-2560x1440.png"
    destination = "/tmp/wallpaper-light.png"
  }

  provisioner "file" {
    source      = "docs/brand/logo/gcp-vdi-icon.svg"
    destination = "/tmp/gcp-vdi.svg"
  }

  provisioner "shell" {
    inline = [
      "sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq",
      "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq xfce4 xfce4-goodies arc-theme papirus-icon-theme fonts-inter fonts-jetbrains-mono dbus-x11 rclone curl jq",
      "sudo install -d -m 0755 /usr/share/backgrounds/gcp-vdi /usr/share/pixmaps",
      "sudo install -m 0644 /tmp/wallpaper-dark.png /usr/share/backgrounds/gcp-vdi/wallpaper-dark.png",
      "sudo install -m 0644 /tmp/wallpaper-light.png /usr/share/backgrounds/gcp-vdi/wallpaper-light.png",
      "sudo install -m 0644 /tmp/gcp-vdi.svg /usr/share/pixmaps/gcp-vdi.svg",
      "rm -f /tmp/wallpaper-dark.png /tmp/wallpaper-light.png /tmp/gcp-vdi.svg",
      "curl -fsSL -o /tmp/rustdesk.deb https://github.com/rustdesk/rustdesk/releases/download/1.3.2/rustdesk-1.3.2-x86_64.deb",
      "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq /tmp/rustdesk.deb || sudo apt-get install -f -y -qq",
      "rm -f /tmp/rustdesk.deb"
    ]
  }
}
