data "nobus_networks" "all" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
}

resource "nobus_network" "app" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = "${var.name_prefix}-net"
}

resource "nobus_subnet" "app" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = "${var.name_prefix}-subnet"
  network_id        = nobus_network.app.id
  cidr              = "192.168.10.0/24"
  ip_version        = 4
  enable_dhcp       = true
}

resource "nobus_security_group" "app" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = "${var.name_prefix}-sg"
  description       = "Security group for example resources"
}

resource "nobus_security_group_rule" "ssh" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  security_group_id = nobus_security_group.app.id
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 22
  port_range_max    = 22
  remote_ip_prefix  = "0.0.0.0/0"
}

resource "nobus_security_group_rule" "icmp" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  security_group_id = nobus_security_group.app.id
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "icmp"
  remote_ip_prefix  = "0.0.0.0/0"
}

resource "nobus_instance" "app" {
  project_id               = var.project_id
  availability_zone        = var.availability_zone
  name                     = "${var.name_prefix}-vm"
  image                    = var.image
  flavor                   = var.flavor
  key_name                 = var.key_name
  network_id               = nobus_network.app.id
  security_groups          = [nobus_security_group.app.name]
  volume_size              = 20
  delete_volume_on_destroy = true
}

resource "nobus_floating_ip" "app" {
  project_id          = var.project_id
  availability_zone   = var.availability_zone
  floating_network_id = data.nobus_networks.all.ids[0]
  port_id             = null
}

resource "nobus_volume" "data" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  name              = "${var.name_prefix}-vol"
  size              = 50
}

resource "nobus_volume_attachment" "app_data" {
  project_id        = var.project_id
  availability_zone = var.availability_zone
  instance_id       = nobus_instance.app.id
  volume_id         = nobus_volume.data.id
}
