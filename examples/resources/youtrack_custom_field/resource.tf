resource "youtrack_custom_field" "customer" {
  name                       = "Customer"
  type                       = "enum[1]"
  is_displayed_in_issue_list = true
}
