data "workbench_group_iam_policy" "my_group_iam_policy" {
  group        = data.workbench_group.my_group.group_name
  organization = data.workbench_group.my_group.organization_id
}

output "group_iams" {
  value = data.workbench_group_iam_policy.my_group_iam_policy.iams
}