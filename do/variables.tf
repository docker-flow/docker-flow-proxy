variable "swarm_manager_token" {
  default = ""
}
variable "swarm_worker_token" {
  default = ""
}
variable "swarm_snapshot_id" {
  default = "unknown"
}
variable "swarm_manager_ip" {
  default = ""
}
variable "swarm_managers" {
  default = 3
}
variable "swarm_workers" {
  default = 0
}
variable "swarm_region" {
  default = "sfo2"
}
variable "swarm_instance_size" {
  default = "1gb"
}
variable "swarm_init" {
  default = false
}
