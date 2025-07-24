import {
  to = workbench_workspace_iam_policy.my_iam_policy
  id = "workspaces/12345678-9012-3456-7890-123456789012/roles"
}

resource "workbench_workspace_iam_policy" "my_iam_policy" {
  workspace_id = data.workbench_workspace.my_workspace.id
  # Any members added or removed externally to any role will be overwritten
  iams = [
    {
      role = "OWNER"
      members = [
        "user1@example.com",
      ]
    },
    {
      role = "READER"
      members = [
        "user2@example.com",
        "user3@example.com",
      ]
    },
    {
      role    = "WRITER"
      members = ["user4@example.com"]
    },
  ]
}

output "workspace_members" {
  value = workbench_workspace_iam_policy.my_iam_policy.iams
}
