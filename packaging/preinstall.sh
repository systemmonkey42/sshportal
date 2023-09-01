#!/bin/sh -e
# shellcheck source=/dev/null

MARIADB_INSTALLED="false"
RET="false"
readonly BYELLOW='\033[1;33m'
readonly BRED='\033[1;31m'
readonly NC='\033[0m'

if grep -q "fedora" /etc/os-release; then
	[ "$1" -ne 1 ] && exit 0 # run this script only on install
	rpm -q mariadb-server >/dev/null 2>&1 && MARIADB_INSTALLED="true"
fi

if grep -q "debian" /etc/os-release; then
	[ "$1" != "install" ] && exit 0 # run this script only on install

	dpkg -s mariadb-server >/dev/null 2>&1 && MARIADB_INSTALLED="true"

	. /usr/share/debconf/confmodule
	db_input high sshportal_mariadb_database || true
	db_go || true
	db_get sshportal_mariadb_database
fi

if [ "$RET" = "true" ] || [ "$SSHPORTAL_MARIADB_SETUP" = "true" ]; then

	if [ "$MARIADB_INSTALLED" = "false" ]; then
		printf "${BRED}%s %s${NC}\n" "ERROR: Please install mariadb-server if you don't want to use Sqlite"
	 	exit 2
	fi

	useradd -rd /nonexistent -s /usr/sbin/nologin sshportal # can't use systemd dynamic user to access the unix socket
	systemctl enable --now mariadb
    mariadb -e "CREATE DATABASE sshportal CHARACTER SET utf8;" || printf "${BYELLOW}%s %s${NC}\n" "WARNING: sshportal database already exists"
	mariadb -e "GRANT ALL on sshportal.* to 'sshportal'@'localhost' identified via unix_socket;"

    if grep -q "Debian" /etc/os-release; then
    	db_go || true
    fi
    touch /tmp/sshportal_mariadb
fi
