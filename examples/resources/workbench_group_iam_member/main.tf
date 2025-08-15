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

data "workbench_group" "my_group" {
  group_name                  = "my_group"
  organization_user_facing_id = "my_org"
}

data "workbench_group" "another_group" {
  group_name                  = "another_group"
  organization_user_facing_id = "my_org"
}