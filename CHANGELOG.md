## 0.0.5 (June 24, 2026)

- Added support for `workbench_perimeter`

## 0.0.4 (January 22, 2026)

- Added import functionality to `workbench_group` resource.
- Added support for `workbench_workspace` data source lookup with `user_facing_id`.

## 0.0.3 (November 6, 2025)

- Support impersonate_service_account for workbench provider configuration.
- Minor docs update for resource import.

## 0.0.2 (August 13, 2025)

Update provider config to support using ID token for authentication with workbench backend.

## 0.0.1 (July 23, 2025)

Initial Release

### 🚀 New Features

- Workspace Support
  - Added `workbench_workspace` resource for managing provisioned workspaces.
  - Added `workbench_workspace` data source to fetch existing workspace details.
  - Added `workbench_workspace_iam_policy`, `workbench_workspace_iam_member`, and
    `workbench_workspace_iam_binding` resources for managing IAM on workspaces.
  - Added `workbench_workspace_iam_policy`, `workbench_workspace_iam_binding` data source to fetch
    existing iam memberships.

- Group Support
  - Added `workbench_group` resource for managing provisioned groups.
  - Added `workbench_group` data source to fetch existing group details.
  - Added `workbench_group_iam_policy`, `workbench_group_iam_member`, and
    `workbench_group_iam_binding` respirces for managing IAM on groups.
  - Added `workbench_group_iam_policy`, `workbench_group_iam_binding` data source to fetch existing
    iam memberships.

- Data Collection Support
  - Added `workbench_data_collection` resource for managing provisioned data collections.
  - Added `workbench_data_collection` data source to fetch existing data collection details.
  - Added `workbench_data_collection_version` resource for managing data collections versioning and
    publishing.
  - Added `workbench_data_collection_version` data source to fetch existing versions details.

- Workspace folder Support
  - Added `workbench_folder` resource for managing workspace folders.
  - Added `workbench_folder` data source to fetch existing folder details.

- GCS bucket Support
  - Added `workbench_gcs_bucket` resource for managing GCS buckets.
  - Added `workbench_gcs_bucket` data source for fetching existing buckets details.
