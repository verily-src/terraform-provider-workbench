resource "workbench_workspace_iam_binding" "my_iam_binding" {
  workspace_id = data.workbench_workspace.my_workspace.id
  role         = "READER"
  # Any members added or removed externally to the READER role will be overwritten
  members = [
    "user1@example.com",
    "user2@example.com",
  ]
}

output "workspace_readers" {
  value = workbench_workspace_iam_binding.my_iam_binding.members
}
