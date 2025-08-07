terraform {
  required_providers {
    workbench = {
      source = "registry.terraform.io/verily-src/workbench"
    }
  }
}

provider "workbench" {
  host = "https://workbench.verily.com"
}
