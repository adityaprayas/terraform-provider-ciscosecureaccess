# Virtual appliances must be imported, not created.
# First import an existing virtual appliance by its origin_id:
# terraform import ciscosecureaccess_virtual_appliance.example 12345

resource "ciscosecureaccess_virtual_appliance" "example" {
  origin_id = 12345
  site_id   = 101
}
