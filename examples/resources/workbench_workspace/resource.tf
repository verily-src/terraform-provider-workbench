resource "workbench_workspace" "my_workspace" {
  display_name    = "My Workspace"
  user_facing_id  = "my-workspace"
  pod_id          = "12345678-9012-3456-7890-123456789012"
  organization_id = "23456789-0123-4567-8901-234567890123"
  description     = "terraform-managed"
  policies = [
    {
      namespace = "terra"
      name      = "my-policy-constrant"
      additional_data = [
        {
          key   = "my-policy-key"
          value = "my-policy-data"
        }
      ]
    }
  ]
  location = "us-east1"
}

output "my_workspace" {
  value = workbench_workspace.my_workspace
}
