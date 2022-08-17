# vim: sw=4:ts=4:et


%define relabel_files() \
restorecon -R /usr/bin/nginx-agent; \
restorecon -R /var/log/nginx-agent; \
restorecon -R /etc/nginx-agent; \

%define selinux_policyver 3.13.1-268

Name:   nginx_agent_selinux
Version:	1.0
Release:	1%{?dist}
Summary:	SELinux policy module for nginx_agent

Group:	System Environment/Base
License:	GPLv2+
# This is an example. You will need to change it.
URL:		http://HOSTNAME
Source0:	nginx_agent.pp
Source1:	nginx_agent.if
Source2:	nginx_agent_selinux.8


Requires: policycoreutils, libselinux-utils
Requires(post): selinux-policy-base >= %{selinux_policyver}, policycoreutils
Requires(postun): policycoreutils
Requires(post): nginx-agent
BuildArch: noarch

%description
This package installs and sets up the  SELinux policy security module for nginx_agent.

%install
install -d %{buildroot}%{_datadir}/selinux/packages
install -m 644 %{SOURCE0} %{buildroot}%{_datadir}/selinux/packages
install -d %{buildroot}%{_datadir}/selinux/devel/include/contrib
install -m 644 %{SOURCE1} %{buildroot}%{_datadir}/selinux/devel/include/contrib/
install -d %{buildroot}%{_mandir}/man8/
install -m 644 %{SOURCE2} %{buildroot}%{_mandir}/man8/nginx_agent_selinux.8
install -d %{buildroot}/etc/selinux/targeted/contexts/users/


%post
semodule -n -i %{_datadir}/selinux/packages/nginx_agent.pp
if /usr/sbin/selinuxenabled ; then
    /usr/sbin/load_policy
    %relabel_files

fi;
exit 0

%postun
if [ $1 -eq 0 ]; then
    semodule -n -r nginx_agent
    if /usr/sbin/selinuxenabled ; then
       /usr/sbin/load_policy
       %relabel_files

    fi;
fi;
exit 0

%files
%attr(0600,root,root) %{_datadir}/selinux/packages/nginx_agent.pp
%{_datadir}/selinux/devel/include/contrib/nginx_agent.if
%{_mandir}/man8/nginx_agent_selinux.8.*


%changelog
* Mon Nov 15 2021 YOUR NAME <YOUR@EMAILADDRESS> 1.0-1
- Initial version
