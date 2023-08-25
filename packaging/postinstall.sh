#!/bin/sh

mkdir -p /var/log/sshportal/audit
echo "0 1 * * 0 root find /var/log/sshportal/session/ -ctime +365 -type f -delete" > /etc/cron.d/sshportal
systemctl daemon-reload
systemctl enable sshportal
systemctl start sshportal
