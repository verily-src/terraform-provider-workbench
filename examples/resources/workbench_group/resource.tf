
resource "workbench_group" "my_group" {
  group_name                  = "my-group"
  organization_user_facing_id = "my-org"
  require_grant_reason        = false
  description                 = "terraform managed group"
}

output "my_group" {
  value = workbench_group.my_group
}
