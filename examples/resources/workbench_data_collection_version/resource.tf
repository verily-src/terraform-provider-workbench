resource "workbench_data_collection_version" "first_version" {
  workspace_id = workbench_data_collection.example_datacollection.id
  display_name = "version 1"
  description  = "first version"
  published    = false
}

output "first_version" {
  value = workbench_data_collection_version.first_version
}