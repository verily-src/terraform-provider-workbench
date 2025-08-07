data "workbench_group_iam_binding" "my_iam_binding" {
  group        = data.workbench_group.my_group.group_name
  organization = data.workbench_group.my_group.organization_id
  role         = "ADMIN"
}

output "group_admin_member" {
  value = data.workbench_group_iam_binding.my_iam_binding.principals
}
