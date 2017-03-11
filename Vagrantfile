# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

Vagrant.configure(2) do |config|
  config.vm.box = "debian/jessie64"
  config.vm.network "private_network", type: "dhcp"
  # config.vm.network "private_network", ip: "192.168.100.22"

  config.vm.provider "virtualbox" do |vb|
    vb.cpus = `sysctl -n hw.ncpu`.to_i
    vb.memory = `sysctl -n hw.memsize`.to_i / 1024 / 1024 / 4
    vb.customize ["modifyvm", :id, "--cpuexecutioncap", "50"]
    vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
    vb.customize ["modifyvm", :id, "--natdnsproxy1", "on"]
  end

  config.vm.synced_folder ".", "/vagrant", id: "vagrant-root", 
    :nfs => true, 
    :nfs_udp => false, 
    :mount_options  => ['nolock,tcp,actimeo=2,rw,fsc,async']

  config.vm.provision :shell, path: "provision.sh"
end
