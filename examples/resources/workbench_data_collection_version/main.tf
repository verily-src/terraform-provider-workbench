terraform {
  required_providers {
    workbench = {
      source = "registry.terraform.io/verily-src/workbench"
    }
  }
}

provider "workbench" {
  host = "https://workbench.verily.com" # replace with your Workbench host
}

resource "workbench_data_collection" "example_datacollection" {
  display_name      = "Terraform Example Data Collection - Version"
  user_facing_id    = "tf-example-dc-version"
  pod_id            = "12345678-9012-3456-7890-123456789012" # replace with your pod ID
  organization_id   = "23456789-0123-4567-8901-234567890123" # replace with your organization ID
  description       = "terraform-managed"
  support_email     = "support@example.com"
  organization_name = "my org"

}
