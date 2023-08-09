Vagrant.configure("2") do |config|
    config.vm.box = "generic/rhel8"
    config.vm.hostname = 'test'
    config.vm.network :private_network, ip: "192.168.56.3"

    config.vm.synced_folder "/build", "/home"
  end