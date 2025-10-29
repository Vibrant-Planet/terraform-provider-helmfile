terraform {
  required_providers {
    helmfile = {
      source = "Vibrant-Planet/helmfile"
    }
  }
}

provider "helmfile" {}
