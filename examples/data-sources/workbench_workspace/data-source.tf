data "workbench_workspace" "my_workspace" {
  id = "12345678-9012-3456-7890-123456789012"
}
data "workbench_workspace" "my_workspace_by_user_facing_id" {
  user_facing_id = "my-workspace"
}

output "my_workspace" {
  value = {
    my_workspace                   = data.workbench_workspace.my_workspace
    my_workspace_by_user_facing_id = data.workbench_workspace.my_workspace_by_user_facing_id
  }
}

