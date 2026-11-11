#!/bin/sh
set -eu
sed -i "s/^#\?PermitRootLogin.*/PermitRootLogin yes/" /etc/ssh/sshd_config
systemctl restart ssh
