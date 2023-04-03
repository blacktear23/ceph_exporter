#!/bin/bash

install_packages() {
    mkdir -p /opt/ceph_exporter
    cp * /opt/ceph_exporter
    cp /opt/ceph_exporter/ceph_exporter.service /etc/systemd/system
    systemctl daemon-reload
}

start_service() {
    systemctl start ceph_exporter
    systemctl enable ceph_exporter
}

install_crontab() {
    if crontab -l | grep "ceph_exporter.push.sh >"; then
        echo Already installed
    else
        echo Install cron job
        temp_file=crontab-push
        crontab -l > ${temp_file}
        echo "* * * * * /opt/ceph_exporter/push.sh > /dev/null 2>&1" >> ${temp_file}
        crontab $temp_file
        rm $temp_file
    fi
}

echo Install Packages
install_packages

echo Start Service
start_service

echo Install Crontab
install_crontab
