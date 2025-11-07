resource "workbench_group_iam_policy" "my_iam_policy" {
  group        = "mytestgroup"
  organization = "12345678-9012-3456-7890-123456789012"
  # Any members added or removed externally to any role will be overwritten
  iams = [
    {
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
            group        = "mygroup2"
            organization = "~myorg2"
          }
        }
      ]
    },
    {
      role = "ADMIN"
      principals = [
        {
          user = "admin@example.com"
        },
      ]
    },
    {
      role = "SUPPORT"
      principals = [
        {
          user = "support@example.com"
        },
      ]
    },
  ]
}

output "group_members" {
  value = workbench_group_iam_policy.my_iam_policy.iams
}
