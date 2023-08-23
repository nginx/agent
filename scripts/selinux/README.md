# SELinux

https://www.redhat.com/en/topics/linux/what-is-selinux

# Table of Contents
- [Prerequisites](#rerequisites)
- [Enable SELinux](#enable-selinux)
- [Updating existing policy](#updating-existing-policy)
- [Known Issues](#known-issues)
- [Debugging](#debugging)
- [References](#references)

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
Copy the folder `scripts/selinux` over to your rhel 8 server.
Then run the following command to update the policy:
```
sudo ./nginx_agent.sh --update
```
Then copy the `nginx_agent.te` and `nginx_agent.pp` files back and create a PR with the changes.

To just rebuild the policy file `nginx_agent.pp` run the following command:
```
sudo ./nginx_agent.sh
```

## Known Issues
If running the command 
```
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
```
results in the following error
```
libsemanage.semanage_pipe_data: Child process /usr/libexec/selinux/hll/pp failed with code: 255. (No such file or directory).
nginx_agent: libsepol.policydb_read: policydb module version 21 does not match my version range 4-19
nginx_agent: libsepol.sepol_module_package_read: invalid module in module package (at section 0)
nginx_agent: Failed to read policy package
libsemanage.semanage_direct_commit: Failed to compile hll files into cil files.
 (No such file or directory).
semodule:  Failed!
```
this usually means that the policy file was built on a rhel 9 environment. To resolve this issue the policy file needs to be rebuilt on a rhel 8 environment. See [Updating existing policy](#updating-existing-policy) for instruction on how to rebuild a policy file.


## Debugging
* To check for policy violation look at the file `/var/log/audit/audit.log`
* To check if NGINX Agent is confined by selinux: `ps -efZ | grep nginx-agent`
* For debugging nginx selinux issues refer to this nginx blog: https://www.nginx.com/blog/using-nginx-plus-with-selinux

## References
* https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/using_selinux/writing-a-custom-selinux-policy_using-selinux
