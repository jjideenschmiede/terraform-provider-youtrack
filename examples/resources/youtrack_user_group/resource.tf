data "youtrack_user_group" "all_users" {
  name = "All Users"
}

resource "youtrack_user_group" "support_agents" {
  name               = "Support Agents"
  successor_group_id = data.youtrack_user_group.all_users.id
}
