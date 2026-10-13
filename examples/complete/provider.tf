terraform {
  required_providers {
    nobus = {
      source  = "Nkponani/nobus"
      version = ">= 0.1.0"
    }
  }
}

provider "nobus" {
  # Configuration is loaded from environment variables:
  # NOBUS_API_KEY, NOBUS_API_SECRET, NOBUS_PROJECT_ID, NOBUS_AVAILABILITY_ZONE, NOBUS_API_URL (optional)
}
