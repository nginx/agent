# SELinux

https://www.redhat.com/en/topics/linux/what-is-selinux

## Prerequisites
```
sudo yum install policycoreutils-devel rpm-build
```

## Enable SELinux
To enable SELinux, update the file `/etc/selinux/config` by setting `SELINUX=enforcing`. Then reboot the machine for the change to take affect.

To validate that SELinux is enabled run the following command:
```
sestatus
```
The output should look something like this:
```
SELinux status:                 enabled
SELinuxfs mount:                /sys/fs/selinux
SELinux root directory:         /etc/selinux
Loaded policy name:             targeted
Current mode:                   enforcing
Mode from config file:          enforcing
Policy MLS status:              enabled
Policy deny_unknown status:     allowed
Max kernel policy version:      31
```
To install the nginx-agent policy run the following commands:
```
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
sudo /usr/sbin/load_policy
sudo restorecon -R /usr/bin/nginx-agent
sudo restorecon -R /var/log/nginx-agent
sudo restorecon -R /etc/nginx-agent
```

## Updating existing policy
Copy the folder `scripts/selinux` over to your rhel server.
Then run the following command to update the policy:
```
sudo ./nginx_agent.sh --update
```

## Debugging
* To check for policy violation look at the file `/var/log/audit/audit.log`
* To check if NGINX Agent is confined by selinux: `ps -efZ | grep nginx-agent`
* For debugging nginx selinux issues refer to this nginx blog: https://www.nginx.com/blog/using-nginx-plus-with-selinux

## References
* https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/using_selinux/writing-a-custom-selinux-policy_using-selinux