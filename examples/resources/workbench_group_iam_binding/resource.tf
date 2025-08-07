import {
  to = workbench_group_iam_binding.my_iam_binding
  id = "organizations/12345678-9012-3456-7890-123456789012/groups/mytestgroup/roles"
}

resource "workbench_group_iam_binding" "my_iam_binding" {
  group        = data.workbench_group.my_group.group_name
  organization = data.workbench_group.my_group.organization_id
  role         = "MEMBER"
  principals = [
    {
      user = "alice@example.com"
    },
    {
      user = "bob@example.com"
    },
    {
      group = {
        group        = data.workbench_group.another_group.group_name
        organization = data.workbench_group.another_group.organization_id
      }
    }
  ]
}

output "group_bindings" {
  value = workbench_group_iam_binding.my_iam_binding.principals
}
