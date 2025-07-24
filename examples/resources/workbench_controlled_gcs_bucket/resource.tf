import {
  to = workbench_controlled_gcs_bucket.my_gcs_bucket
  id = "workspaces/12345678-9012-3456-7890-123456789012/controlled_gcs_buckets/12345678-9012-3456-7890-123456789210"
}

resource "workbench_controlled_gcs_bucket" "my_gcs_bucket" {
  workspace_id = data.workbench_workspace.my_workspace.id
  name         = "my_tf_bucket"
  bucket_name  = "my-tf-bucket-vwb-project-id"
  description  = "this is a tf managed bucket"
}

output "workspace_gcs_bucket" {
  value = workbench_controlled_gcs_bucket.my_gcs_bucket
}
