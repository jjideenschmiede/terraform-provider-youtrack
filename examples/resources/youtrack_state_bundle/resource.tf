resource "youtrack_state_bundle" "support_states" {
  name = "Support States"
  values = [
    { name = "Open", color_id = "5" },
    { name = "In Progress", color_id = "9" },
    { name = "Waiting for Customer" },
    { name = "Solved", is_resolved = true },
    { name = "Rejected", is_resolved = true, archived = true },
  ]
}
