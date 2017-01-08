resource "digitalocean_droplet" "swarm-manager" {
  image = "${var.swarm_snapshot_id}"
  size = "${var.swarm_instance_size}"
  count = "${var.swarm_managers}"
  name = "${format("swarm-manager-%02d", (count.index + 1))}"
  region = "${var.swarm_region}"
  private_networking = true
  ssh_keys = [
    "${digitalocean_ssh_key.proxy.id}"
  ]
  connection {
    user = "root"
    private_key = "${file("proxy-key")}"
    agent = false
  }
  provisioner "remote-exec" {
    inline = [
      "if ${var.swarm_init}; then docker swarm init --advertise-addr ${self.ipv4_address_private}; fi",
      "if ! ${var.swarm_init}; then docker swarm join --token ${var.swarm_manager_token} --advertise-addr ${self.ipv4_address_private} ${var.swarm_manager_ip}:2377; fi"
    ]
  }
}

resource "digitalocean_droplet" "swarm-worker" {
  image = "${var.swarm_snapshot_id}"
  size = "${var.swarm_instance_size}"
  count = "${var.swarm_workers}"
  name = "${format("swarm-worker-%02d", (count.index + 1))}"
  region = "${var.swarm_region}"
  private_networking = true
  ssh_keys = [
    "${digitalocean_ssh_key.proxy.id}"
  ]
  connection {
    user = "root"
    private_key = "${file("proxy-key")}"
    agent = false
  }
  provisioner "remote-exec" {
    inline = [
      "docker swarm join --token ${var.swarm_worker_token} --advertise-addr ${self.ipv4_address_private} ${var.swarm_manager_ip}:2377"
    ]
  }
}

output "swarm_manager_1_public_ip" {
  value = "${digitalocean_droplet.swarm-manager.0.ipv4_address}"
}

output "swarm_manager_1_private_ip" {
  value = "${digitalocean_droplet.swarm-manager.0.ipv4_address_private}"
}

output "swarm_manager_2_public_ip" {
  value = "${digitalocean_droplet.swarm-manager.1.ipv4_address}"
}

output "swarm_manager_2_private_ip" {
  value = "${digitalocean_droplet.swarm-manager.1.ipv4_address_private}"
}

output "swarm_manager_3_public_ip" {
  value = "${digitalocean_droplet.swarm-manager.2.ipv4_address}"
}

output "swarm_manager_3_private_ip" {
  value = "${digitalocean_droplet.swarm-manager.2.ipv4_address_private}"
}