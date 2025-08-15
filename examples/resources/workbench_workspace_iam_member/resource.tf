resource "workbench_workspace_iam_member" "my_workspace_member" {
  workspace_id = data.workbench_workspace.my_workspace.id
  role         = "WRITER"
  # The WRITER role will be granted to these members, but other WRTIERs will NOT
  # be removed
  members = [
    "user3@example.com",
    "user4@example.com",
  ]
}
