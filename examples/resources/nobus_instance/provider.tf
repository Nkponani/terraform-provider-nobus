terraform {
  required_providers {
    nobus = {
      source = "nobus-cloud/nobus"
    }
  }
}

# Configuration is read from NOBUS_API_KEY, NOBUS_PROJECT_ID, and optionally
# NOBUS_AVAILABILITY_ZONE. This keeps credentials out of this example.
provider "nobus" {}
