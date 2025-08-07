data "workbench_workspace" "my_workspace" {
  id = "12345678-9012-3456-7890-123456789012"
}

output "my_workspace" {
  value = data.workbench_workspace.my_workspace
}

