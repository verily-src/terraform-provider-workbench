data "workbench_group" "my_group" {
  group_name = "my-global-group"
}

data "workbench_group" "my_org_group" {
  group_name                  = "my-org-group"
  organization_user_facing_id = "my-org"
}

output "my_group" {
  value = data.workbench_group.my_group
}

output "my_org_group" {
  value = data.workbench_group.my_org_group
}
