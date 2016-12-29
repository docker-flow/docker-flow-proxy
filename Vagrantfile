# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  if (/cygwin|mswin|mingw|bccwin|wince|emx/ =~ RUBY_PLATFORM) != nil
    config.vm.synced_folder ".", "/vagrant", mount_options: ["dmode=700,fmode=600"]
  else
    config.vm.synced_folder ".", "/vagrant"
  end
  config.vm.define "proxy" do |d|
    d.vm.box = "ubuntu/trusty64"
    d.vm.hostname = "proxy"
    d.vm.network "private_network", ip: "10.100.198.200"
    d.vm.provision :shell, path: "scripts/bootstrap_ansible.sh"
    d.vm.provision :shell, inline: "PYTHONUNBUFFERED=1 ansible-playbook /vagrant/ansible/proxy.yml -c local"
    d.vm.provision :shell, inline: "PYTHONUNBUFFERED=1 ansible-playbook /vagrant/ansible/swarm.yml -i /vagrant/ansible/hosts/prod"
    d.vm.provider "virtualbox" do |v|
      v.memory = 1024
    end
  end
  config.vm.define "services" do |d|
    d.vm.box = "ubuntu/trusty64"
    d.vm.hostname = "services"
    d.vm.network "private_network", ip: "10.100.198.201"
    d.vm.provider "virtualbox" do |v|
      v.memory = 1024
    end
  end
  config.vm.define "jenkins" do |d|
    d.vm.box = "ubuntu/trusty64"
    d.vm.hostname = "jenkins"
    d.vm.network "private_network", ip: "10.100.199.200"
    d.vm.provision :shell, path: "scripts/bootstrap_ansible.sh"
    d.vm.provision :shell, inline: "PYTHONUNBUFFERED=1 ansible-playbook /vagrant/ansible/jenkins.yml -c local"
    d.vm.provision :shell, inline: "PYTHONUNBUFFERED=1 ansible-playbook /vagrant/ansible/swarm.yml -i /vagrant/ansible/hosts/prod --extra-vars 'consul_ip=10.100.199.200'"
    d.vm.provider "virtualbox" do |v|
      v.memory = 1024
    end
  end
  config.vm.define "swarm-manager-1" do |d|
    d.vm.box = "https://cloud-images.ubuntu.com/xenial/current/xenial-server-cloudimg-amd64-vagrant.box"
    d.vm.hostname = "swarm-manager-1"
    d.vm.network "private_network", ip: "10.100.192.200"
    d.vm.provision :shell, path: "scripts/bootstrap-vagrant-swarm.sh"
    d.vm.provision :shell, inline: "sudo docker swarm init --advertise-addr 10.100.192.200"
    d.vm.provision :shell, inline: "docker swarm join-token -q worker >/vagrant/worker-token"
    d.vm.provider "virtualbox" do |v|
      v.memory = 1024
    end
  end
  (1..2).each do |i|
    config.vm.define "swarm-worker-#{i}" do |d|
      d.vm.box = "https://cloud-images.ubuntu.com/xenial/current/xenial-server-cloudimg-amd64-vagrant.box"
      d.vm.hostname = "swarm-worker-#{i}"
      d.vm.network "private_network", ip: "10.100.192.20#{i}"
      d.vm.provision :shell, path: "scripts/bootstrap-vagrant-swarm.sh"
      d.vm.provision :shell, inline: "docker swarm join --token $(cat /vagrant/worker-token) --advertise-addr 10.100.192.20#{i} 10.100.192.200:2377"
      d.vm.provider "virtualbox" do |v|
        v.memory = 1024
      end
    end
  end
  if Vagrant.has_plugin?("vagrant-cachier")
    config.cache.scope = :box
  end
end