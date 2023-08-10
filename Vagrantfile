$script = <<-SCRIPT
echo "********************** Check **********************"
sestatus
echo "********************** Agent Install **********************"
sudo yum localinstall -y /home/vagrant/build/nginx-agent-2.27.1-SNAPSHOT-$COMMIT.rpm
sudo systemctl start nginx-agent
echo "********************** Install Agent Policy **********************"
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
sudo /usr/sbin/load_policy
sudo restorecon -R /usr/bin/nginx-agent;
sudo restorecon -R /var/log/nginx-agent;
sudo restorecon -R /etc/nginx-agent;
echo "********************** Check Logs **********************"
sudo cat /var/log/audit/audit.log | grep nginx-agent
echo "********************** Check Stuff **********************"
sudo semodule -lfull | grep "nginx_agent"
ps -efZ | grep nginx-agent
ps auxZ | grep nginx-agent
sudo ausearch -m AVC,USER_AVC,SELINUX_ERR,USER_SELINUX_ERR -ts recent
SCRIPT

Vagrant.configure("2") do |config|
    config.vm.box = "generic/rhel8"
    config.vm.hostname = 'test'
    config.vm.network :private_network, ip: "192.168.56.3"

    config.vm.synced_folder "./build", "/home/vagrant/build"
    config.vm.provision "shell", inline: $script, env: {"COMMIT" => ENV['COMMIT']}
  end