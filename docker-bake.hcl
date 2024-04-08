# ref: https://docs.docker.com/build/bake/reference/

# ref: https://github.com/docker/metadata-action?tab=readme-ov-file#bake-definition
target "docker-metadata-action" {
  tags = ["goldpinger:latest"]
}

group "default" {
  targets = ["linux-simple"]
}

group "ci" {
  targets = ["linux-simple", "linux-vendor", "windows-nanoserver-ltsc2019", "windows-nanoserver-ltsc2022"]
}

target "linux-simple" {
  inherits = ["docker-metadata-action"]
  tags = "${formatlist("%s-linux", target.docker-metadata-action.tags)}"
  platforms = ["linux/amd64", "linux/arm64"]
  target = "simple"
}

target "linux-vendor" {
  inherits = ["docker-metadata-action"]
  tags = "${formatlist("%s-vendor", target.docker-metadata-action.tags)}"
  platforms = ["linux/amd64", "linux/arm64"]
  target = "vendor"
}

target "windows-nanoserver-ltsc2019" {
  inherits = ["docker-metadata-action"]
  tags = "${formatlist("%s-windows-ltsc2019", target.docker-metadata-action.tags)}"

  platforms = ["windows/amd64"]

  target = "windows"
  args = {
    WINDOWS_BASE_IMAGE = "mcr.microsoft.com/windows/nanoserver:ltsc2019"
  }
}

target "windows-nanoserver-ltsc2022" {
  inherits = ["docker-metadata-action"]
  tags = "${formatlist("%s-windows-ltsc2022", target.docker-metadata-action.tags)}"

  platforms = ["windows/amd64"]

  target = "windows"
  args = {
    WINDOWS_BASE_IMAGE = "mcr.microsoft.com/windows/nanoserver:ltsc2022"
  }
}
