# SELinux

https://www.redhat.com/en/topics/linux/what-is-selinux

# Table of Contents
- [Prerequisites](#prerequisites)
- [Enable SELinux](#enable-selinux)
- [Install NGINX Agent Policy](#install-nginx-agent-policy)
- [Updating existing policy](#updating-existing-policy)
- [Troubleshooting](#troubleshooting)
    - [Policy version does not match](#policy-version-does-not-match)
    - [Unknown Type](#unknown-type)
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


## Install NGINX Agent Policy
To install the nginx-agent policy run the following commands:
```
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
sudo /usr/sbin/load_policy
sudo restorecon -R /usr/bin/nginx-agent
sudo restorecon -R /var/log/nginx-agent
sudo restorecon -R /etc/nginx-agent
```

## Updating existing policy
Check for errors by using the `ausearch` command:
```
sudo ausearch -m AVC,USER_AVC,SELINUX_ERR,USER_SELINUX_ERR --raw -se nginx_agent -ts recent
```
Generate new rule based on the errors by using `audit2allow`:
```
sudo ausearch -m AVC,USER_AVC,SELINUX_ERR,USER_SELINUX_ERR --raw -se nms -ts recent | audit2allow
```

Update the `scripts/selinux/nginx_agent.te` file with the output from the `audit2allow` command.

Copy the `scripts/selinux/nginx_agent.te` file to a Centos 7 machine and build a new `nginx_agent.pp` file by running the following command:
```
make -f /usr/share/selinux/devel/Makefile nginx_agent.pp
```
**[NOTE: The policy has to be built on a Centos 7 machine. If it is built on a different OS like RHEL 8/9 then we will encounter this issue [Policy version does not match](#policy-version-does-not-match) when installing it on an older OS like Centos 7. Even if the `audit2allow` command was run on a RHEL 8/9 machine the updates to the policy need to be made on a Centos 7 machine.]**

Install the policy by following the steps here [Install NGINX Agent Policy](#install-nginx-agent-policy)

Then create a PR with the changes made to the `nginx_agent.te` and `nginx_agent.pp` files.

## Troubleshooting
### Policy version does not match
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
this usually means that the policy file was built on a newer environment than isn't complicate with the environment the policy is being installed on.

To resolve this issue the policy file needs to be rebuilt on a Centos 7 environment. See [Updating existing policy](#updating-existing-policy) for instruction on how to rebuild a policy file.

### Unknown Type
If running the command
```
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
```
results in the following error
```
/usr/bin/checkmodule:  loading policy configuration from tmp/nginx_agent.tmp
nginx_agent.te:52:ERROR 'unknown type bin_t' at token ';' on line 4301:
```
that means that the type is unknown and needs to be added to the require block in the `nginx_agent.te` file like this:
```
require {
    bin_t
}
```

## Debugging
* To check for policy violation look at the file `/var/log/audit/audit.log`
* To check if NGINX Agent is confined by selinux: `ps -efZ | grep nginx-agent`
* For debugging nginx selinux issues refer to this nginx blog: https://www.nginx.com/blog/using-nginx-plus-with-selinux

## References
* https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/using_selinux/writing-a-custom-selinux-policy_using-selinux
