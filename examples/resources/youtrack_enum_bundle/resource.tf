resource "youtrack_enum_bundle" "customers" {
  name = "Customers"
  values = [
    { name = "ACME", color_id = "20" },
    { name = "Globex" },
    { name = "Initech", archived = true },
  ]
}
