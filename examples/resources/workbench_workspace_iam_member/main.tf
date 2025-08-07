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

data "workbench_workspace" "my_workspace" {
  id = "12345678-9012-3456-7890-123456789012"
}
