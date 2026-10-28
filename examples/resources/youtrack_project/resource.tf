data "youtrack_user" "owner" {
  login = "jane.doe"
}

resource "youtrack_project" "support" {
  name             = "Support"
  short_name       = "SUP"
  leader_id        = data.youtrack_user.owner.id
  description      = "Customer support tickets"
  reply_to_email   = "support@example.com"
  project_template = "kanban"
}
