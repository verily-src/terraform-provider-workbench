resource "workbench_data_collection" "tf_datacollection_in_vpc" {
  display_name    = "My Data Collection in VPC"
  user_facing_id  = "my-data-collection"
  pod_id          = "12345678-9012-3456-7890-123456789012"
  organization_id = "12345678-9012-3456-0987-210987654321"
  description     = "terraform-managed data collection"
  policies = [
    {
      namespace = "terra"
      name      = "exfil-perimeter-constraint"
      additional_data = [
        {
          key   = "perimeter-id"
          value = "my-vpc-perimeter"
        }
      ]
    }
  ]
  location          = "us-east1"
  support_email     = "support@example.com"
  organization_name = "my-org"
  therapeutic_tags  = ["dermatology", "immunology"]
}

output "my_datacollection" {
  value = workbench_data_collection.tf_datacollection_in_vpc
}