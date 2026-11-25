resource "youtrack_saved_query" "open_support_tickets" {
  name                 = "Open support tickets"
  query                = "project: SUP #Unresolved"
  visible_for_group_id = youtrack_user_group.support_agents.id
}
