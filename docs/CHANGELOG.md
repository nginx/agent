
2.18.0
=============
2022-09-06

* Fixed publish variable in release branch workflow (e5037c1b)
* Updating changelog (c4ca68f4)
* Updating changelog (f59779c8)
* Moved packaging logic into a different Makefile (#34) (a5fdcaf0)
* Fix release workflow (#33) (c72810ee)
* Add release branch workflow (#30) (1646c3ee)
* feat: new NMS ACM dimensions for advanced metrics (#24) (a51d3d89)
* mitigate click jacking in example html (#28) (934bb58a)
* fixing certs to be seeded by sha256 (#26) (32d34d26)
* Updated build-docker make target to use pkgs.nginx.com (#27) (5d04c77b)
* Merge pull request #25 from nginx/update-commander-plugin-unit-tests (25d5766d)
* Updated commander plugin unit tests (3e75fca0)
* added features key to config to list features for NGINX Agent (#21) (87f8d802)
* Merge pull request #23 from nginx/update-unstable-unit-tests (f3c7384b)
* Updated unstable unit tests (4de4ea02)
* Update release workflow (13b02e4f)
* Update release workflow (6976682c)
* Update release workflow (cdea9397)
* Update release workflow (0a3a4a81)
* Update release workflow (04ce7be0)
* Update release workflow (e5c014f5)
* Update release workflow (a66708fa)
* Update release workflow (b9c5f621)
* Update release workflow (3750a193)
* Update release workflow (2313cd84)
* Update release workflow (76a6517a)
* Update release workflow (4a1fa039)
* Add release workflow (1cba8135)

2.17.0
=============
2022-08-22

* Improve metrics logging for network io and nginx workers (#16) (9149ae8f)
* added a fix for the example server crashing when NGINX is not running. Removed the waitgroup in comms.go Close, as if no active waitgroup can cause a negative count. (#17) (855e28d0)
* Update the default keep alive params for the grpc client (#20) (baf3723c)
* Install test cleanup (#18) (bcb8b792)
* Merge pull request #15 from nginx/Agent-readme-blog (8b81933d)
* docs: readme and blog (4da532c6)
* The NGINX Agent is a lightweight piece of software that can be installed next to NGINX Open Source (OSS) and/or NGINX Plus (2074d371)


