resource "youtrack_tag" "needs_triage" {
  name                   = "needs triage"
  color_id               = "12"
  untag_on_resolve       = true
  visible_for_group_id   = youtrack_user_group.support_agents.id
  updateable_by_group_id = youtrack_user_group.support_agents.id
}
