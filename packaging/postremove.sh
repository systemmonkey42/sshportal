#!/bin/sh

systemctl is-active --quiet sshportal && systemctl stop sshportal
rm -f /etc/cron.d/sshportal
systemctl daemon-reload
systemctl reset-failed