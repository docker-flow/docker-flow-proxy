resource "digitalocean_ssh_key" "proxy" {
  name = "proxy-key"
  public_key = "${file("proxy-key.pub")}"
}

resource "digitalocean_floating_ip" "docker_1" {
  droplet_id = "${digitalocean_droplet.swarm-manager-1.id}"
  region = "${var.swarm_region}"
}

output "floating_ip_1" {
  value = "${digitalocean_floating_ip.docker_1.ip_address}"
}
