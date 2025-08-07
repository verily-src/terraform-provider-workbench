data "workbench_workspace_iam_policy" "my_iam_policy" {
  workspace_id = data.workbench_workspace.my_workspace.id
}

output "workspace_members" {
  value = data.workbench_workspace_iam_policy.my_iam_policy.iams
}
