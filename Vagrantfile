# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

Vagrant.configure(2) do |config|
  config.vm.box = "debian/jessie64"
  config.vm.network "private_network", type: "dhcp"
  # config.vm.network "private_network", ip: "192.168.100.22"

  # This is to be run provision.sh
  config.vm.synced_folder ".", "/vagrant", id: "vagrant-root",
    :nfs => true,
    :nfs_udp => false,
    :mount_options  => ['nolock,tcp,actimeo=2,rw,fsc,async']

  # Building go source needs to have project in GOPATH
  config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/atelierdisko/hoi", id: "hoi-src",
    :nfs => true,
    :nfs_udp => false,
    :mount_options  => ['nolock,tcp,actimeo=2,rw,fsc,async']

  config.vm.provision :shell, path: "provision.sh"
end
