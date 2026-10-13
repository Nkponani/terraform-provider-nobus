variable "project_id" {
  description = "Nobus project ID"
  type        = string
  default     = null
}

variable "availability_zone" {
  description = "Nobus availability zone"
  type        = string
  default     = null
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "example"
}

variable "key_name" {
  description = "SSH key name to use for instances"
  type        = string
}

variable "image" {
  description = "Image name/ID"
  type        = string
  default     = "ubuntu-22-04"
}

variable "flavor" {
  description = "Flavor name/ID"
  type        = string
  default     = "small"
}
