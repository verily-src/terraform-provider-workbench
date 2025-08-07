resource "workbench_folder" "tf_folder" {
  workspace_id = workbench_workspace.example_workspace.id
  display_name = "terraform-example-folder"
  description  = "test folder"
}

output "my_folder" {
  value = workbench_folder.tf_folder
}