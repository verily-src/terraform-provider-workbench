import {
  to = workbench_group_iam_member.my_iam_member
  id = "organizations/12345678-9012-3456-7890-123456789012/groups/my-group/roles"
}

resource "workbench_group_iam_member" "my_iam_member" {
  group        = data.workbench_group.my_group.group_name
  organization = data.workbench_group.my_group.organization_id
  # Any members added or removed externally is ignored
  role = "MEMBER"
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

output "group_members" {
  value = workbench_group_iam_member.my_iam_member.principals
}
