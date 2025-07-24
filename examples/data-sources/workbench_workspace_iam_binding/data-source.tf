data "workbench_workspace_iam_binding" "my_iam_binding" {
  workspace_id = data.workbench_workspace.my_workspace.id
  role         = "WRITER"
}

output "workspace_writers" {
  value = data.workbench_workspace_iam_binding.my_iam_binding.members
}
