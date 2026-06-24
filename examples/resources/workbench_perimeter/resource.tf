
resource "workbench_perimeter" "my_perimeter" {
  resource_id = "acme-research"

  owners = [
    "admin-group@example.com",
  ]

  users = [
    "automations-ci-sa@terra-solutions-9g0o-admin.iam.gserviceaccount.com",
    "research-team@example.com",
  ]

  sync_google_group = true
}

output "my_perimeter" {
  value = workbench_perimeter.my_perimeter
}
