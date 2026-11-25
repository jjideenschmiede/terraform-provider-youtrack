resource "youtrack_issue_link_type" "blocks" {
  name             = "Blocks"
  source_to_target = "blocks"
  target_to_source = "is blocked by"
  directed         = true
}
