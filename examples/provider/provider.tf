terraform {
  required_providers {
    nobus = {
      source = "nobus-cloud/nobus"
    }
  }
}

provider "nobus" {
  # Preferred: read the API key from a sensitive variable.
  api_key = var.nobus_api_key

  # Optional. Defaults shown below; each can also be supplied through the
  # matching NOBUS_* environment variable.
  # endpoint         = "https://cloud-api.nobus.io"
  # project_id       = var.nobus_project_id
  # availability_zone = "nobus-wa-az1"
}

variable "nobus_api_key" {
  description = "Nobus API key, sent as a bearer token."
  type        = string
  sensitive   = true
}

variable "nobus_project_id" {
  description = "Nobus project that owns the resources."
  type        = string
}
