#!/bin/bash
#
# Copyright 2016 Atelier Disko. All rights reserved.
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

apt-get update

apt-get install -y nginx

sudo debconf-set-selections <<< 'mysql-server mysql-server/root_password password vagrant'
sudo debconf-set-selections <<< 'mysql-server mysql-server/root_password_again password vagrant'
apt-get install -y mariadb-server mariadb-client

apt-get install -y php5-fpm

