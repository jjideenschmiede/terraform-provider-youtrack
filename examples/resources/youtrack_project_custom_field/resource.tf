resource "youtrack_project_custom_field" "customer" {
  project_id       = youtrack_project.support.id
  field_id         = youtrack_custom_field.customer.id
  bundle_id        = youtrack_enum_bundle.customers.id
  can_be_empty     = true
  empty_field_text = "No customer"
}
