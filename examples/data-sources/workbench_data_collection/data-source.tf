data "workbench_data_collection" "my_data_collection" {
  id = "12345678-9012-3456-7890-123456789012"
}

output "my_data_collection" {
  value = data.workbench_data_collection.my_data_collection
}