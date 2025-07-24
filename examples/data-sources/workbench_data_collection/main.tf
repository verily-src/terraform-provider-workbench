terraform {
  required_providers {
    workbench = {
      source = "hashicorp.com/local/workbench"
    }
  }
}

provider "workbench" {
  host = "https://workbench.verily.com"
}