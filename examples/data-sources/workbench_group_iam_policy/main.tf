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

data "workbench_group" "my_group" {
  group_name                  = "my-org-group"
  organization_user_facing_id = "my-org"
}
