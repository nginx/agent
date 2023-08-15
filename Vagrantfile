$script = <<-SCRIPT
echo "********************** Check SELinux **********************"
sestatus
echo ""
echo ""
echo "********************** NGINX Install **********************"
sudo yum install -y yum-utils
sudo touch /etc/yum.repos.d/nginx.repo
sudo chmod 666 /etc/yum.repos.d/nginx.repo
sudo printf '[nginx-stable]\nname=nginx stable repo\nbaseurl=http://nginx.org/packages/centos/\$%s/\$%s/\ngpgcheck=1\nenabled=1\ngpgkey=https://nginx.org/keys/nginx_signing.key\nmodule_hotfixes=true\n\n[nginx-mainline]\nname=nginx mainline repo\nbaseurl=http://nginx.org/packages/mainline/centos/\$%s/\$%s/\ngpgcheck=1\nenabled=0\ngpgkey=https://nginx.org/keys/nginx_signing.key\nmodule_hotfixes=true' \"releasever\" \"basearch\" \"releasever\" \"basearch\" > /etc/yum.repos.d/nginx.repo
sudo cat /etc/yum.repos.d/nginx.repo
sudo yum update -y
sudo yum install -y nginx
sudo nginx
curl -I 127.0.0.1
echo ""
echo ""
echo "********************** Agent Install **********************"
sudo yum localinstall -y /home/vagrant/build/nginx-agent--SNAPSHOT-$COMMIT.rpm
echo ""
echo "********************** Install Agent Policy **********************"
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
sudo /usr/sbin/load_policy
sudo restorecon -R /usr/bin/nginx-agent
sudo restorecon -R /var/log/nginx-agent
sudo restorecon -R /etc/nginx-agent
sudo systemctl start nginx-agent
echo ""
echo "********************** Check Logs **********************"
sudo cat /var/log/audit/audit.log | grep nginx-agent
echo ""
echo "********************** Check Stuff **********************"
sudo semodule -lfull | grep "nginx_agent"
ps -efZ | grep nginx-agent
ps auxZ | grep nginx-agent
sudo ausearch -m AVC,USER_AVC,SELINUX_ERR,USER_SELINUX_ERR -ts recent --input-logs
SCRIPT

Vagrant.configure("2") do |config|
    config.vm.box = "generic/rhel8"
    config.vm.hostname = 'test'
    config.vm.network :private_network, ip: "192.168.56.3"

    config.vm.synced_folder "./build", "/home/vagrant/build"
    config.vm.provision "shell", inline: $script, env: {"COMMIT" => ENV['COMMIT']}
  end