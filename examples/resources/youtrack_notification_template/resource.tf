resource "youtrack_notification_template" "templates" {
  for_each = fileset("${path.module}/templates", "*.ftl")
  id       = trimsuffix(each.value, ".ftl")
  content  = file("${path.module}/templates/${each.value}")
}
