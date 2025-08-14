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

data "workbench_workspace" "my_workspace" {
  id = "12345678-9012-3456-7890-123456789012"
}
