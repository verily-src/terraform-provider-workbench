terraform {
  required_providers {
    workbench = {
      source  = "verily-src/workbench"
      version = ">= 0.0.1"
    }
  }
}

provider "workbench" {
  host = "https://workbench.verily.com"
}