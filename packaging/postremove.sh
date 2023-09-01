#!/bin/sh -e
# shellcheck source=/dev/null

if grep -q "debian" /etc/os-release; then
	. /usr/share/debconf/confmodule
	db_purge
fi

if command -v semodule >/dev/null 2>&1; then
	semodule -l | grep -q sshportal && semodule -r sshportal
fi

systemctl is-active --quiet sshportal && systemctl stop sshportal
rm -f /etc/logrotate.d/sshportal || true
[ -d /etc/systemd/system/sshportal.service.d ] && rm -rf /etc/systemd/system/sshportal.service.d

grep -q sshportal /etc/passwd && userdel sshportal
systemctl daemon-reload
systemctl reset-failed

