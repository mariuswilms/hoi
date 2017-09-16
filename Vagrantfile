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

  config.vm.provision :shell, inline: <<SCRIPT
apt-get update

# Install extras as config files use more_headers module.
apt-get install -y nginx nginx-extras

sudo debconf-set-selections <<< 'mysql-server mysql-server/root_password password vagrant'
sudo debconf-set-selections <<< 'mysql-server mysql-server/root_password_again password vagrant'
apt-get install -y mariadb-server mariadb-client

apt-get install -y php5-fpm

# Needed by wokers, as an examplaric After= dependency for service units.
apt-get install -y beanstalkd

# Install build tools and get a more recent Go version than the provided 1.3, so
# we can easily build and run system tests inside the VM.
wget --no-verbose https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.8.linux-amd64.tar.gz
echo "export GOPATH=/home/vagrant/go" >> /etc/bash.bashrc
echo "export PATH=\$PATH:/usr/local/go/bin" >> /etc/profile
sed -i -e "s|Defaults\tsecure_path|# Defaults\tsecure_path|g" /etc/sudoers
apt-get install -y make
SCRIPT
end
