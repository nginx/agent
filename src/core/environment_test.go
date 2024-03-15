/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

const fedoraOsReleaseInfo = `
NAME=Fedora
VERSION="32 (Workstation Edition)"
ID=fedora
VERSION_ID=32
PRETTY_NAME="Fedora 32 (Workstation Edition)"
ANSI_COLOR="0;38;2;60;110;180"
LOGO=fedora-logo-icon
CPE_NAME="cpe:/o:fedoraproject:fedora:32"
HOME_URL="https://fedoraproject.org/"
DOCUMENTATION_URL="https://docs.fedoraproject.org/en-US/fedora/f32/system-administrators-guide/"
SUPPORT_URL="https://fedoraproject.org/wiki/Communicating_and_getting_help"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
REDHAT_BUGZILLA_PRODUCT="Fedora"
REDHAT_BUGZILLA_PRODUCT_VERSION=32
REDHAT_SUPPORT_PRODUCT="Fedora"
REDHAT_SUPPORT_PRODUCT_VERSION=32
PRIVACY_POLICY_URL="https://fedoraproject.org/wiki/Legal:PrivacyPolicy"
VARIANT="Workstation Edition"
VARIANT_ID=workstation"
`

const ubuntuReleaseInfo = `
NAME="Ubuntu"
VERSION="20.04.5 LTS (Focal Fossa)"
VERSION_ID="20.04"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 20.04.5 LTS"
HOME_URL="https://www.ubuntu.com"
SUPPORT_URL=\"https://help.ubuntu.com/"
BUG_REPORT_URL=\"https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
VERSION_CODENAME=focal
UBUNTU_CODENAME=focal
`

const osReleaseInfoWithNoName = `
VERSION="20.04.5 LTS (Focal Fossa)"
VERSION_ID="20.04"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 20.04.5 LTS"
HOME_URL="https://www.ubuntu.com"
SUPPORT_URL=\"https://help.ubuntu.com/"
BUG_REPORT_URL=\"https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
VERSION_CODENAME=focal
UBUNTU_CODENAME=focal
`

const lscpuInfo1 = `
Architecture:                    aarch64
CPU op-mode(s):                  32-bit, 64-bit
Byte Order:                      Little Endian
CPU(s):                          2
On-line CPU(s) list:             0,1
Thread(s) per core:              1
Core(s) per socket:              2
Socket(s):                       1
NUMA node(s):                    1
Vendor ID:                       ARM
Model:                           3
Model name:                      Cortex-A72
Stepping:                        r0p3
BogoMIPS:                        166.66
L1d cache:                       64 KiB
L1i cache:                       96 KiB
L2 cache:                        2 MiB
NUMA node0 CPU(s):               0,1
Vulnerability Itlb multihit:     Not affected
Vulnerability L1tf:              Not affected
Vulnerability Mds:               Not affected
Vulnerability Meltdown:          Not affected
Vulnerability Mmio stale data:   Not affected
Vulnerability Retbleed:          Not affected
Vulnerability Spec store bypass: Not affected
Vulnerability Spectre v1:        Mitigation; __user pointer sanitization
Vulnerability Spectre v2:        Mitigation; Branch predictor hardening, BHB
Vulnerability Srbds:             Not affected\nVulnerability Tsx async abort:   Not affected
Flags:                           fp asimd evtstrm aes pmull sha1 sha2 crc32 cpuid
`

const lscpuInfo2 = `
L1d cache:                       64 KiB
L1i cache:                       "96 KiB"
L2 cache:                        2 MiB
`

// set at buildtime
var envMountInfo = [10]string{
	`NO CONTAINER ID PRESENT IN MOUNTINFO`,
	`822 773 0:55 / / rw,relatime master:312 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/OVIJO6CZHWIXJDDHZXRECADDI3:/var/lib/docker/overlay2/l/3D3QYHJJTMCK6GLLVY7MMM6K4V:/var/lib/docker/overlay2/l/OKH52ZN3IE727BHLU3G3LEVI6S:/var/lib/docker/overlay2/l/K3BV3TCWQS2WDAY3ZVXO5GIHLQ:/var/lib/docker/overlay2/l/2KZOTUQIESHNC4FHZHYULXIKZ5,upperdir=/var/lib/docker/overlay2/f8c1fa1c3a6eb3731265dc674bf238c60fb594eedc4639cdbefef93ad443f55d/diff,workdir=/var/lib/docker/overlay2/f8c1fa1c3a6eb3731265dc674bf238c60fb594eedc4639cdbefef93ad443f55d/work,xino=off
823 822 0:57 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
824 822 0:58 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
825 824 0:59 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
826 822 0:60 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
827 826 0:61 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
828 827 0:30 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime master:11 - cgroup cgroup rw,xattr,name=systemd
829 827 0:33 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,rdma
830 827 0:34 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/net_cls,net_prio ro,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,net_cls,net_prio
831 827 0:35 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/cpu,cpuacct ro,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,cpu,cpuacct
832 827 0:36 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,devices
833 827 0:37 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,hugetlb
834 827 0:38 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,memory
835 827 0:39 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime master:21 - cgroup cgroup rw,freezer
836 827 0:40 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime master:22 - cgroup cgroup rw,blkio
837 827 0:41 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime master:23 - cgroup cgroup rw,cpuset
838 827 0:42 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime master:24 - cgroup cgroup rw,pids
839 827 0:43 /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4 /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime master:25 - cgroup cgroup rw,perf_event
840 824 0:56 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
841 824 0:62 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
842 822 8:1 /var/lib/docker/containers/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/sda1 rw
843 822 8:1 /var/lib/docker/containers/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4/hostname /etc/hostname rw,relatime - ext4 /dev/sda1 rw
844 822 8:1 /var/lib/docker/containers/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4/hosts /etc/hosts rw,relatime - ext4 /dev/sda1 rw
774 823 0:57 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
775 823 0:57 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
776 823 0:57 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
777 823 0:57 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
778 823 0:57 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
779 823 0:63 / /proc/acpi ro,relatime - tmpfs tmpfs ro
780 823 0:58 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
781 823 0:58 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
782 823 0:58 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
783 823 0:58 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
784 823 0:64 / /proc/scsi ro,relatime - tmpfs tmpfs ro
785 826 0:65 / /sys/firmware ro,relatime - tmpfs tmpfs ro`,
	`648 603 0:41 / / rw,relatime master:304 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/PUPHWIJFCPRLWVUF4FNUZUOCK6:/var/lib/docker/overlay2/l/I2ESYRZNCXSTQZUZADJL535IFQ,upperdir=/var/lib/docker/overlay2/b4a145accf21c673470f76384380e892d599a935e4e066eac9d2761e6c8dd1f3/diff,workdir=/var/lib/docker/overlay2/b4a145accf21c673470f76384380e892d599a935e4e066eac9d2761e6c8dd1f3/work
649 648 0:48 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
650 648 0:50 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
651 650 0:51 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
652 648 0:52 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
653 652 0:27 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw,nsdelegate,memory_recursiveprot
654 650 0:47 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
655 650 0:53 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k,inode64
656 648 8:3 /var/lib/docker/containers/bc22cfd94f0d32e476d3187519ab39bd8af99ca2af1e3e69c6c82e9c157551be/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
657 648 8:3 /var/lib/docker/containers/bc22cfd94f0d32e476d3187519ab39bd8af99ca2af1e3e69c6c82e9c157551be/hostname /etc/hostname rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
658 648 8:3 /var/lib/docker/containers/bc22cfd94f0d32e476d3187519ab39bd8af99ca2af1e3e69c6c82e9c157551be/hosts /etc/hosts rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
604 650 0:51 /0 /dev/console rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
605 649 0:48 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
606 649 0:48 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
607 649 0:48 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
608 649 0:48 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
609 649 0:48 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
610 649 0:54 / /proc/asound ro,relatime - tmpfs tmpfs ro,inode64
611 649 0:55 / /proc/acpi ro,relatime - tmpfs tmpfs ro,inode64
612 649 0:50 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
613 649 0:50 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
614 649 0:50 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
615 649 0:56 / /proc/scsi ro,relatime - tmpfs tmpfs ro,inode64
616 652 0:57 / /sys/firmware ro,relatime - tmpfs tmpfs ro,inode64`,
	`5625 5410 0:525 / / rw,relatime master:1623 - overlay overlay rw,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/589/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/588/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/587/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/586/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/585/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/584/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/583/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/582/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/581/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/580/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/579/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/57/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/592/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/592/work,xino=off
5626 5625 0:526 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
5627 5625 0:527 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
5628 5627 0:528 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
5629 5627 0:516 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
5630 5625 0:521 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
5631 5630 0:529 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
5632 5631 0:30 /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime master:11 - cgroup cgroup rw,xattr,name=systemd
5633 5631 0:33 /kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/net_cls,net_prio ro,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,net_cls,net_prio
5634 5631 0:34 /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,memory
5635 5631 0:35 /kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,freezer
5636 5631 0:36 /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/cpu,cpuacct ro,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,cpu,cpuacct
5637 5631 0:37 /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,devices
5638 5631 0:38 /kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,perf_event
5639 5631 0:39 /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime master:21 - cgroup cgroup rw,pids
5640 5631 0:40 /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime master:22 - cgroup cgroup rw,blkio
5641 5631 0:41 /kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime master:23 - cgroup cgroup rw,cpuset
5642 5631 0:42 /kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80 /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime master:24 - cgroup cgroup rw,hugetlb
5643 5631 0:43 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime master:25 - cgroup cgroup rw,rdma
5644 5627 0:514 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
5645 5625 253:0 /var/lib/kubelet/pods/214f3ba8-4b69-4bdb-a7d5-5ecc73f04ae9/etc-hosts /etc/hosts rw,relatime - ext4 /dev/mapper/ubuntu--vg-ubuntu--lv rw
5646 5627 253:0 /var/lib/kubelet/pods/214f3ba8-4b69-4bdb-a7d5-5ecc73f04ae9/containers/nginx-nim/7de6c2d0 /dev/termination-log rw,relatime - ext4 /dev/mapper/ubuntu--vg-ubuntu--lv rw
5647 5625 253:0 /var/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/d7cb24ec5dede02990283dec30bd1e6ae1f93e3e19b152b708b7e0e133c6baec/hostname /etc/hostname rw,relatime - ext4 /dev/mapper/ubuntu--vg-ubuntu--lv rw
5648 5625 253:0 /var/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/d7cb24ec5dede02990283dec30bd1e6ae1f93e3e19b152b708b7e0e133c6baec/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/mapper/ubuntu--vg-ubuntu--lv rw
5649 5625 253:0 /var/lib/kubelet/pods/214f3ba8-4b69-4bdb-a7d5-5ecc73f04ae9/volumes/kubernetes.io~configmap/nginx-default-conf/..2022_03_26_00_02_23.1074775554/default.conf /etc/nginx/conf.d/default.conf ro,relatime - ext4 /dev/mapper/ubuntu--vg-ubuntu--lv rw
5650 5625 0:513 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw,size=8046268k
5411 5626 0:526 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
5412 5626 0:526 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
5413 5626 0:526 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
5414 5626 0:526 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
5415 5626 0:526 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
5416 5626 0:530 / /proc/acpi ro,relatime - tmpfs tmpfs ro
5417 5626 0:527 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
5418 5626 0:527 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
5419 5626 0:527 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
5420 5626 0:527 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
5421 5626 0:531 / /proc/scsi ro,relatime - tmpfs tmpfs ro
5422 5630 0:532 / /sys/firmware ro,relatime - tmpfs tmpfs ro`,
	`1859 1574 0:466 / / rw,relatime master:300 - overlay overlay rw,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/117/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/116/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/115/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/114/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/113/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/104/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/118/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/118/work
1860 1859 0:468 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
1861 1859 0:469 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1862 1861 0:470 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
1863 1861 0:299 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
1864 1859 0:314 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
1865 1864 0:471 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
1866 1865 0:31 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime - cgroup cpuset rw,cpuset
1867 1865 0:32 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/cpu ro,nosuid,nodev,noexec,relatime - cgroup cpu rw,cpu
1868 1865 0:33 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/cpuacct ro,nosuid,nodev,noexec,relatime - cgroup cpuacct rw,cpuacct
1869 1865 0:34 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime - cgroup blkio rw,blkio
1870 1865 0:35 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime - cgroup memory rw,memory
1871 1865 0:36 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime - cgroup devices rw,devices
1872 1865 0:37 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime - cgroup freezer rw,freezer
1873 1865 0:38 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/net_cls ro,nosuid,nodev,noexec,relatime - cgroup net_cls rw,net_cls
1874 1865 0:39 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime - cgroup perf_event rw,perf_event
1875 1865 0:40 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/net_prio ro,nosuid,nodev,noexec,relatime - cgroup net_prio rw,net_prio
1876 1865 0:41 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime - cgroup hugetlb rw,hugetlb
1877 1865 0:42 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime - cgroup pids rw,pids
1878 1865 0:43 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime - cgroup rdma rw,rdma
1879 1865 0:44 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/pod0635783e-afc4-448c-b3c2-ed3c739eaf39/c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime - cgroup cgroup rw,name=systemd
1880 1859 254:1 /docker/volumes/minikube/_data/lib/kubelet/pods/0635783e-afc4-448c-b3c2-ed3c739eaf39/etc-hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw
1881 1861 254:1 /docker/volumes/minikube/_data/lib/kubelet/pods/0635783e-afc4-448c-b3c2-ed3c739eaf39/containers/hello-agent5/fce2e0f4 /dev/termination-log rw,relatime - ext4 /dev/vda1 rw
1882 1859 254:1 /docker/volumes/minikube/_data/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/282dfef55fa7416b05891d21a5f5fc17779c706cb834de07d9bd707635bce041/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw
1883 1859 254:1 /docker/volumes/minikube/_data/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/282dfef55fa7416b05891d21a5f5fc17779c706cb834de07d9bd707635bce041/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw
1884 1861 0:295 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
1885 1859 0:270 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw,size=12267028k
1575 1860 0:468 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
1604 1860 0:468 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
1605 1860 0:468 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
1606 1860 0:468 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
1607 1860 0:468 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
1608 1860 0:472 / /proc/acpi ro,relatime - tmpfs tmpfs ro
1609 1860 0:469 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1610 1860 0:469 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1611 1860 0:469 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1612 1860 0:469 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1613 1864 0:473 / /sys/firmware ro,relatime - tmpfs tmpfs ro`,
	`1939 1564 0:486 / / rw,relatime master:305 - overlay overlay rw,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/109/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/108/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/107/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/106/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/105/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/104/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/124/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/124/work
1940 1939 0:488 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
1941 1939 0:489 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1942 1941 0:490 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
1943 1941 0:478 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
1944 1939 0:483 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
1945 1944 0:491 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
1946 1945 0:31 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime - cgroup cpuset rw,cpuset
1947 1945 0:32 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/cpu ro,nosuid,nodev,noexec,relatime - cgroup cpu rw,cpu
1948 1945 0:33 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/cpuacct ro,nosuid,nodev,noexec,relatime - cgroup cpuacct rw,cpuacct
1949 1945 0:34 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime - cgroup blkio rw,blkio
1950 1945 0:35 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime - cgroup memory rw,memory
1951 1945 0:36 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime - cgroup devices rw,devices
1952 1945 0:37 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime - cgroup freezer rw,freezer
1953 1945 0:38 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/net_cls ro,nosuid,nodev,noexec,relatime - cgroup net_cls rw,net_cls
1954 1945 0:39 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime - cgroup perf_event rw,perf_event
1955 1945 0:40 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/net_prio ro,nosuid,nodev,noexec,relatime - cgroup net_prio rw,net_prio
1956 1945 0:41 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime - cgroup hugetlb rw,hugetlb
1957 1945 0:42 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime - cgroup pids rw,pids
1958 1945 0:43 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime - cgroup rdma rw,rdma
1959 1945 0:44 /docker/72be2865c435abce6fb5167e6c0604027a41964dba3b920bd34b23762af67fb8/kubepods/besteffort/poddc449d8e-9349-4a00-8f12-fefccdb2f49a/2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime - cgroup cgroup rw,name=systemd
1960 1939 254:1 /docker/volumes/minikube/_data/lib/kubelet/pods/dc449d8e-9349-4a00-8f12-fefccdb2f49a/etc-hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw
1961 1941 254:1 /docker/volumes/minikube/_data/lib/kubelet/pods/dc449d8e-9349-4a00-8f12-fefccdb2f49a/containers/hello-agent/0c6ff58b /dev/termination-log rw,relatime - ext4 /dev/vda1 rw
1962 1939 254:1 /docker/volumes/minikube/_data/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/3bfbcbb0e0ebf3ad7f51f3aaccd4babed90a513e94e0c52d31278777d0f48b9e/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw
1963 1939 254:1 /docker/volumes/minikube/_data/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/3bfbcbb0e0ebf3ad7f51f3aaccd4babed90a513e94e0c52d31278777d0f48b9e/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/vda1 rw
1964 1941 0:475 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
1965 1939 0:465 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw,size=12267028k
1565 1940 0:488 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
1573 1940 0:488 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
1614 1940 0:488 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
1615 1940 0:488 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
1616 1940 0:488 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
1617 1940 0:492 / /proc/acpi ro,relatime - tmpfs tmpfs ro
1618 1940 0:489 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1619 1940 0:489 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1620 1940 0:489 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1621 1940 0:489 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
1622 1944 0:493 / /sys/firmware ro,relatime - tmpfs tmpfs ro`,
	`826 709 0:61 / / rw,nodev,relatime - overlay overlay rw,lowerdir=/var/lib/containers/storage/overlay/l/GV3YTMQ7IJHJXELM6CHUHA5POE:/var/lib/containers/storage/overlay/l/LUOYE3LBGFXFVZ232OXDLHZBML:/var/lib/containers/storage/overlay/l/GBYDEUH4YSOZH77CSJGKD7FJFZ:/var/lib/containers/storage/overlay/l/VSVG2DQQ2BEBO4DN3KLIWULYGS:/var/lib/containers/storage/overlay/l/QDKTJIWHFVLEVB6CGXXR6A4ZWK:/var/lib/containers/storage/overlay/l/POOFWLQ7VH2CVRXJ4FVY5GUKUF,upperdir=/var/lib/containers/storage/overlay/11c606dbc7b58ac436103d998ce48adf3c83fa8d915e87ca313217d279be6082/diff,workdir=/var/lib/containers/storage/overlay/11c606dbc7b58ac436103d998ce48adf3c83fa8d915e87ca313217d279be6082/work,xino=off,metacopy=on
827 826 0:57 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
828 826 0:62 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
829 828 0:63 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
830 828 0:55 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
831 826 0:60 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
832 831 0:64 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
833 832 0:30 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime master:11 - cgroup cgroup rw,xattr,name=systemd
834 832 0:33 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/net_cls,net_prio ro,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,net_cls,net_prio
835 832 0:34 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,pids
836 832 0:35 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,hugetlb
837 832 0:36 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,memory
838 832 0:37 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,rdma
839 832 0:38 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,freezer
840 832 0:39 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime master:21 - cgroup cgroup rw,perf_event
841 832 0:40 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime master:22 - cgroup cgroup rw,blkio
842 832 0:41 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime master:23 - cgroup cgroup rw,cpuset
843 832 0:42 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/cpu,cpuacct ro,nosuid,nodev,noexec,relatime master:24 - cgroup cgroup rw,cpu,cpuacct
844 832 0:43 /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime master:25 - cgroup cgroup rw,devices
845 828 0:54 / /dev/shm rw,nosuid,nodev,noexec,relatime master:253 - tmpfs shm rw,size=65536k
846 826 0:24 /containers/storage/overlay-containers/ba0be90007be48bca767be0a462390ad2c9b0e910608158f79c8d6a984302b7e/userdata/resolv.conf /etc/resolv.conf rw,nosuid,nodev,noexec,relatime master:5 - tmpfs tmpfs rw,size=203524k,mode=755
847 826 0:24 /containers/storage/overlay-containers/ba0be90007be48bca767be0a462390ad2c9b0e910608158f79c8d6a984302b7e/userdata/hostname /etc/hostname rw,nosuid,nodev,noexec,relatime master:5 - tmpfs tmpfs rw,size=203524k,mode=755
710 827 0:57 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
711 827 0:57 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
712 827 0:57 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
713 827 0:57 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
714 827 0:57 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
715 827 0:65 / /proc/acpi ro,relatime - tmpfs tmpfs ro
716 827 0:62 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
717 827 0:62 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
718 827 0:62 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
719 827 0:62 /null /proc/sched_debug rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
720 827 0:66 / /proc/scsi ro,relatime - tmpfs tmpfs ro
721 831 0:67 / /sys/firmware ro,relatime - tmpfs tmpfs ro
741 831 0:68 / /sys/dev ro,relatime - tmpfs tmpfs ro`,
	`688 633 0:95 / / rw,relatime - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/3INAU44LWT5WVNUZGR3CU5KVTJ:/var/lib/docker/overlay2/l/UZ76YP26WVD6B2PGORRVDQKZ4C:/var/lib/docker/overlay2/l/G7LFO56NGQ637KQAPC6LJBQJMX:/var/lib/docker/overlay2/l/3VXOJK66Z2MJ6GAGA33OE2AOHL:/var/lib/docker/overlay2/l/3DZRQPPSQROUJJHWPIEQ2RSF27:/var/lib/docker/overlay2/l/7245IZGQM7NSJVTRWYNVPI5OCE:/var/lib/docker/overlay2/l/4DOOTUH5YC674WDH2TXS5EPJBC,upperdir=/var/lib/docker/overlay2/5b8946c6104e8a128b005d9735ee5371a5a773f59c69fd205cc8d722728f61fb/diff,workdir=/var/lib/docker/overlay2/5b8946c6104e8a128b005d9735ee5371a5a773f59c69fd205cc8d722728f61fb/work
689 688 0:97 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw"
690 688 0:99 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755"
797 690 0:100 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666"
798 688 0:101 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro"
799 798 0:102 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755"
800 799 0:24 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime master:9 - cgroup cgroup rw,xattr,name=systemd"
801 799 0:27 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime master:14 - cgroup cgroup rw,blkio"
802 799 0:28 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/net_cls,net_prio ro,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,net_cls,net_prio"
803 799 0:29 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,devices"
804 799 0:30 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,hugetlb"
805 799 0:31 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,pids"
806 799 0:32 / /sys/fs/cgroup/rdma ro,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,rdma"
807 799 0:33 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/cpu,cpuacct ro,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,cpu,cpuacct"
808 799 0:34 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime master:21 - cgroup cgroup rw,freezer"
839 799 0:35 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime master:22 - cgroup cgroup rw,perf_event"
847 799 0:36 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime master:23 - cgroup cgroup rw,cpuset"
848 799 0:37 /docker/98fb932878c55d7440d70ea973cd034d4c330fa0afe39d5b7e29c286aeb339b9/docker/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783 /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime master:24 - cgroup cgroup rw,memory"
849 690 0:96 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw"
850 690 0:103 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k"
851 688 8:1 /var/lib/docker/volumes/runner-xxurkrix-project-26945533-concurrent-0-cache-c33bcaa1fd2c77edfc3893b41966cea8/_data/f5/nginx/agent/product/nginx-agent /home/nginx rw,nosuid,nodev,relatime - ext4 /dev/sda1 rw,commit=30"
852 688 8:1 /var/lib/docker/volumes/57c921339b5a325e203a47e75038301773eee8ff89c0db2504958094eaf92297/_data/containers/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783/resolv.conf /etc/resolv.conf rw,nosuid,nodev,relatime - ext4 /dev/sda1 rw,commit=30"
853 688 8:1 /var/lib/docker/volumes/57c921339b5a325e203a47e75038301773eee8ff89c0db2504958094eaf92297/_data/containers/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783/hostname /etc/hostname rw,nosuid,nodev,relatime - ext4 /dev/sda1 rw,commit=30"
854 688 8:1 /var/lib/docker/volumes/57c921339b5a325e203a47e75038301773eee8ff89c0db2504958094eaf92297/_data/containers/54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783/hosts /etc/hosts rw,nosuid,nodev,relatime - ext4 /dev/sda1 rw,commit=30"
634 689 0:97 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw"
635 689 0:97 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw"
636 689 0:97 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw"
637 689 0:97 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw"
638 689 0:97 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw"
639 689 0:104 / /proc/acpi ro,relatime - tmpfs tmpfs ro"
656 689 0:99 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755"
657 689 0:99 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755"
658 689 0:99 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755"
665 689 0:105 / /proc/scsi ro,relatime - tmpfs tmpfs ro"
666 798 0:106 / /sys/firmware ro,relatime - tmpfs tmpfs ro`,
	`1630 1497 0:233 / / rw,relatime master:435 - overlay overlay rw,lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/63/fs,upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/64/fs,workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/64/work
1631 1630 0:235 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
1632 1630 0:236 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1633 1632 0:237 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
1634 1632 0:223 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
1635 1630 0:228 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
1636 1635 0:27 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw,nsdelegate,memory_recursiveprot
1637 1630 8:3 /var/lib/docker/volumes/minikube/_data/lib/kubelet/pods/a9708d2c-e69e-4fe4-a77b-f871c4cd6930/etc-hosts /etc/hosts rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1638 1632 8:3 /var/lib/docker/volumes/minikube/_data/lib/kubelet/pods/a9708d2c-e69e-4fe4-a77b-f871c4cd6930/containers/ubuntu/ed75da3e /dev/termination-log rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1639 1630 8:3 /var/lib/docker/volumes/minikube/_data/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/a07a6878dac42baaba3595e04ec6781088428d7c293ff2d6b424876b1e69044d/hostname /etc/hostname rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1640 1630 8:3 /var/lib/docker/volumes/minikube/_data/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/a07a6878dac42baaba3595e04ec6781088428d7c293ff2d6b424876b1e69044d/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1641 1632 0:220 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k,inode64
1642 1630 0:219 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw,size=4019648k,inode64
1498 1631 0:235 /asound /proc/asound ro,nosuid,nodev,noexec,relatime - proc proc rw
1499 1631 0:235 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
1503 1631 0:235 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
1518 1631 0:235 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
1519 1631 0:235 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
1520 1631 0:235 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
1521 1631 0:238 / /proc/acpi ro,relatime - tmpfs tmpfs ro,inode64
1522 1631 0:236 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1523 1631 0:236 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1524 1631 0:236 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1525 1631 0:239 / /proc/scsi ro,relatime - tmpfs tmpfs ro,inode64
1526 1635 0:240 / /sys/firmware ro,relatime - tmpfs tmpfs ro,inode64`,
	`1656 1480 0:220 / / rw,relatime master:482 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/M7KGGSN5AAQ47YCPCDG6WFLSQV:/var/lib/docker/overlay2/l/RUOVMUFQAEZDMIUFHFFU4AIRRF,upperdir=/var/lib/docker/overlay2/6061e9bb1a206bac6191c9dad4e02ed4aee8bde31149bbbf1adc10b243a76ebc/diff,workdir=/var/lib/docker/overlay2/6061e9bb1a206bac6191c9dad4e02ed4aee8bde31149bbbf1adc10b243a76ebc/work
1657 1656 0:222 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
1658 1656 0:223 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1659 1658 0:224 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
1660 1656 0:179 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
1661 1660 0:27 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw,nsdelegate,memory_recursiveprot
1662 1658 0:174 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
1663 1658 8:3 /var/lib/docker/volumes/minikube/_data/lib/kubelet/pods/a9708d2c-e69e-4fe4-a77b-f871c4cd6930/containers/ubuntu/8d3b665e /dev/termination-log rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1664 1656 8:3 /var/lib/docker/volumes/minikube/_data/lib/docker/containers/4bc33970f0c2e3ee7f14c023aff0e7a403c649e7e8b7dd64808ba62479d1a1da/resolv.conf /etc/resolv.conf rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1665 1656 8:3 /var/lib/docker/volumes/minikube/_data/lib/docker/containers/4bc33970f0c2e3ee7f14c023aff0e7a403c649e7e8b7dd64808ba62479d1a1da/hostname /etc/hostname rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1666 1656 8:3 /var/lib/docker/volumes/minikube/_data/lib/kubelet/pods/a9708d2c-e69e-4fe4-a77b-f871c4cd6930/etc-hosts /etc/hosts rw,relatime - ext4 /dev/sda3 rw,errors=remount-ro
1667 1658 0:171 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k,inode64
1668 1656 0:156 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw,size=4019648k,inode64
1488 1657 0:222 /asound /proc/asound ro,nosuid,nodev,noexec,relatime - proc proc rw
1490 1657 0:222 /bus /proc/bus ro,nosuid,nodev,noexec,relatime - proc proc rw
1492 1657 0:222 /fs /proc/fs ro,nosuid,nodev,noexec,relatime - proc proc rw
1498 1657 0:222 /irq /proc/irq ro,nosuid,nodev,noexec,relatime - proc proc rw
1499 1657 0:222 /sys /proc/sys ro,nosuid,nodev,noexec,relatime - proc proc rw
1500 1657 0:222 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
1501 1657 0:225 / /proc/acpi ro,relatime - tmpfs tmpfs ro,inode64
1502 1657 0:223 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1503 1657 0:223 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1504 1657 0:223 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755,inode64
1505 1657 0:226 / /proc/scsi ro,relatime - tmpfs tmpfs ro,inode64
1506 1660 0:227 / /sys/firmware ro,relatime - tmpfs tmpfs ro,inode64`,
}

func TestNewHostInfo(t *testing.T) {
	version := "test"
	tags := []string{"testTag"}
	configDirs := "/testDirs"
	env := EnvironmentType{}
	host := env.NewHostInfo(version, &tags, configDirs, true)
	assert.Equal(t, version, host.Agent)
	assert.GreaterOrEqual(t, host.Boot, uint64(1))
	assert.NotEmpty(t, host.Hostname)
	assert.NotEmpty(t, host.DisplayName)
	assert.NotEmpty(t, host.OsType)
	assert.GreaterOrEqual(t, len(host.Uuid), 36)
	assert.GreaterOrEqual(t, len(host.Partitons), 1)
	assert.GreaterOrEqual(t, len(host.Network.Interfaces), 1)
	assert.GreaterOrEqual(t, len(host.Processor), 1)
	assert.NotEmpty(t, host.Processor[0].Architecture)
	assert.GreaterOrEqual(t, len(strings.Split(host.Uname, " ")), 5)
	assert.NotEmpty(t, host.Release)
	assert.Equal(t, tags, host.Tags)
}

func TestReleaseInfo(t *testing.T) {
	release := releaseInfo("dummy-release") // No release file present
	assert.NotEmptyf(t, release, "release is empty %v", release)
}

func TestNetworks(t *testing.T) {
	env := EnvironmentType{}
	networks := env.networks()
	assert.GreaterOrEqual(t, len(networks.Interfaces), 1)
	assert.NotNil(t, networks.Default)
}

// MockEnvironment is a mock implementation of the Environment interface for testing purposes.
type MockEnvironment struct {
	isContainer bool
}

// IsContainer is a mock function for IsContainer() method.
func (m *MockEnvironment) IsContainer() bool {
	// Mock implementation for testing.
	return m.isContainer
}

func (m *MockEnvironment) Virtualization() (string, string) {
	realEnv := &EnvironmentType{}
	return realEnv.Virtualization()
}

func TestVirtualization(t *testing.T) {
	mockEnv := &MockEnvironment{isContainer: false}

	// Test normal VM
	virtualizationFunc = func(_ context.Context) (string, string, error) {
		return "LXC", "host", nil
	}

	virtualizationSystem, virtualizationRole := mockEnv.Virtualization()

	assert.Equal(t, "LXC", virtualizationSystem)
	assert.Equal(t, "host", virtualizationRole)

	// Test container
	virtualizationFunc = func(_ context.Context) (string, string, error) {
		return "docker", "host", nil
	}

	mockEnv.isContainer = true

	virtualizationSystem, virtualizationRole = mockEnv.Virtualization()

	assert.Equal(t, "container", virtualizationSystem)
	assert.Equal(t, "host", virtualizationRole)
}

func TestProcessors(t *testing.T) {
	env := EnvironmentType{}
	processorInfo := env.processors("arm64")
	// at least one network interface
	assert.GreaterOrEqual(t, processorInfo[0].GetCpus(), int32(1))
	// non-empty architecture
	assert.NotEmpty(t, processorInfo[0].GetArchitecture())
	assert.Equal(t, "arm64", processorInfo[0].GetArchitecture())
}

type fakeShell struct {
	output map[string]string
	errors map[string]error
}

func (f *fakeShell) Exec(cmd string, arg ...string) ([]byte, error) {
	key := strings.Join(append([]string{cmd}, arg...), " ")
	if err, ok := f.errors[key]; ok {
		return nil, err
	}
	if out, ok := f.output[key]; ok {
		return []byte(out), nil
	}
	return nil, fmt.Errorf("unexpected command %s", key)
}

func TestGetCacheInfo(t *testing.T) {
	tempShellCommander := shell
	defer func() { shell = tempShellCommander }()
	tests := []struct {
		name             string
		shell            Shell
		defaultCacheInfo map[string]string
		expect           map[string]string
	}{
		{
			name: "lscpu error",
			shell: &fakeShell{
				errors: map[string]error{
					"lscpu": errors.New("nope"),
				},
			},
			defaultCacheInfo: map[string]string{
				"L1d": "64 KiB",
				"L1i": "96 KiB",
				"L2":  "2 MiB",
				"L3":  "1 MiB",
			},
			expect: map[string]string{
				"L1d": "64 KiB",
				"L1i": "96 KiB",
				"L2":  "2 MiB",
				"L3":  "1 MiB",
			},
		},
		{
			name: "default cache info absent",
			shell: &fakeShell{
				output: map[string]string{
					"lscpu": lscpuInfo1,
				},
			},
			defaultCacheInfo: map[string]string{
				"L1d": "-1",
				"L1i": "-1",
				"L2":  "-1",
				"L3":  "-1",
			},
			expect: map[string]string{
				"L1d": "64 KiB",
				"L1i": "96 KiB",
				"L2":  "2 MiB",
				"L3":  "-1",
			},
		},
		{
			name: "os-release present with quote",
			shell: &fakeShell{
				output: map[string]string{
					"lscpu": lscpuInfo2,
				},
			},
			defaultCacheInfo: map[string]string{
				"L1d": "-1",
				"L1i": "-1",
				"L2":  "-1",
				"L3":  "-1",
			},
			expect: map[string]string{
				"L1d": "64 KiB",
				"L1i": "96 KiB",
				"L2":  "2 MiB",
				"L3":  "-1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shell = tt.shell
			actual := getCacheInfo(tt.defaultCacheInfo)
			assert.Equal(t, tt.expect, actual)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name   string
		bytes  int
		expect string
	}{
		{
			name:   "TestFormatBytesNotSupported",
			bytes:  -1,
			expect: "-1",
		},
		{
			name:   "TestFormatBytesinB",
			bytes:  512,
			expect: "512 B",
		},
		{
			name:   "TestFormatBytesinKiB",
			bytes:  131072,
			expect: "128 KiB",
		},
		{
			name:   "TestFormatBytesinMiB",
			bytes:  4194304,
			expect: "4 MiB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatBytes(tt.bytes)
			assert.Equal(t, tt.expect, actual)
		})
	}
}

func TestProcesses(t *testing.T) {
	env := EnvironmentType{}
	processesInfo := env.Processes()
	// at least 2 processes (if nginx is on the system)
	assert.GreaterOrEqual(t, len(processesInfo), 0)
}

func TestWriteFiles(t *testing.T) {
	files := []*proto.File{
		{
			Name:        "/tmp/test.conf",
			Contents:    []byte("contents"),
			Permissions: "0644",
		},
	}

	AllowedDirectoriesMap := map[string]struct{}{"/tmp": {}}

	for _, file := range files {
		assert.NoFileExists(t, file.GetName())
	}
	backup, err := sdk.NewConfigApplyWithIgnoreDirectives("", nil, []string{})
	assert.NoError(t, err)

	env := EnvironmentType{}
	assert.NoError(t, env.WriteFiles(backup, files, "/tmp", AllowedDirectoriesMap))

	for _, file := range files {
		assert.FileExists(t, file.GetName())

		contents, err := os.ReadFile(file.GetName())
		assert.NoError(t, err)
		assert.Equal(t, file.GetContents(), contents)

		os.Remove(file.GetName())
		assert.NoFileExists(t, file.GetName())
	}
}

func TestWriteFilesWhenExists(t *testing.T) {
	files := []*proto.File{
		{
			Name:        "/tmp/multiple1.conf",
			Contents:    []byte("multi"),
			Permissions: "0644",
		},
		{
			Name:        "/tmp/multiple2.conf",
			Contents:    []byte("ple"),
			Permissions: "0644",
		},
	}

	AllowedDirectoriesMap := map[string]struct{}{"/tmp": {}}

	backup, err := sdk.NewConfigApplyWithIgnoreDirectives("", nil, []string{})
	assert.NoError(t, err)
	for _, file := range files {
		assert.NoFileExists(t, file.GetName())
	}

	env := EnvironmentType{}
	assert.NoError(t, env.WriteFiles(backup, files, "/tmp", AllowedDirectoriesMap))
	assert.NoError(t, env.WriteFiles(backup, files, "/tmp", AllowedDirectoriesMap))

	for _, file := range files {
		assert.FileExists(t, file.GetName())

		contents, err := os.ReadFile(file.GetName())
		assert.NoError(t, err)
		assert.Equal(t, file.GetContents(), contents)

		os.Remove(file.GetName())
		assert.NoFileExists(t, file.GetName())
	}
}

func TestWriteFilesNotAllowed(t *testing.T) {
	files := []*proto.File{
		{
			Name:        "/tmp/multiple1.conf",
			Contents:    []byte("multi"),
			Permissions: "0644",
		},
	}
	backup, err := sdk.NewConfigApplyWithIgnoreDirectives("", nil, []string{})
	assert.NoError(t, err)

	AllowedDirectoriesMap := map[string]struct{}{"/opt": {}}

	env := EnvironmentType{}
	assert.Error(t, env.WriteFiles(backup, files, "/tmp", AllowedDirectoriesMap))

	for _, file := range files {
		assert.NoFileExists(t, file.GetName())
	}
}

func TestWriteFile(t *testing.T) {
	env := &EnvironmentType{}
	file := &proto.File{
		Name:        "/tmp/sub-1/sub-2/write.conf",
		Contents:    []byte("contents"),
		Permissions: "0777",
	}
	backup, err := sdk.NewConfigApplyWithIgnoreDirectives("", nil, []string{})
	assert.NoError(t, err)
	assert.NoError(t, env.WriteFile(backup, file, "/tmp"))
	assert.FileExists(t, file.GetName())

	contents, err := os.ReadFile(file.GetName())
	assert.NoError(t, err)
	assert.Equal(t, file.GetContents(), contents)

	os.Remove(file.GetName())
	assert.NoFileExists(t, file.GetName())
}

func TestDeleteFile(t *testing.T) {
	env := &EnvironmentType{}
	tempDir := t.TempDir()
	fileName := tempDir + "/test.txt"
	file := &proto.File{
		Name:        fileName,
		Contents:    []byte("contents"),
		Permissions: "0777",
	}
	backup, err := sdk.NewConfigApplyWithIgnoreDirectives("", nil, []string{})
	assert.NoError(t, err)
	assert.NoError(t, env.WriteFile(backup, file, "/tmp"))
	assert.FileExists(t, file.GetName())

	contents, err := os.ReadFile(file.GetName())
	assert.NoError(t, err)
	assert.Equal(t, file.GetContents(), contents)

	assert.NoError(t, env.DeleteFile(backup, fileName))
	assert.NoFileExists(t, file.GetName())

	// verify that file being deleted is backed up in case we need to rollback
	_, ok := backup.GetNotExists()[fileName]
	assert.True(t, ok)
	_, ok = backup.GetExisting()[fileName]
	assert.False(t, ok)
}

func TestParseOsReleaseFile(t *testing.T) {
	tests := []struct {
		name             string
		osReleaseContent string
		expect           map[string]string
	}{
		{
			name:             "ubuntu os-release info",
			osReleaseContent: ubuntuReleaseInfo,
			expect: map[string]string{
				"VERSION_ID":       "20.04",
				"VERSION":          "20.04.5 LTS (Focal Fossa)",
				"VERSION_CODENAME": "focal",
				"NAME":             "Ubuntu",
				"ID":               "ubuntu",
			},
		},
		{
			name:             "fedora os-release info",
			osReleaseContent: fedoraOsReleaseInfo,
			expect: map[string]string{
				"VERSION_ID": "32",
				"VERSION":    "32 (Workstation Edition)",
				"NAME":       "Fedora",
				"ID":         "fedora",
			},
		},
		{
			name:             "os-release info with no name",
			osReleaseContent: osReleaseInfoWithNoName,
			expect: map[string]string{
				"VERSION_ID":       "20.04",
				"VERSION":          "20.04.5 LTS (Focal Fossa)",
				"VERSION_CODENAME": "focal",
				"NAME":             "unix",
				"ID":               "ubuntu",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.osReleaseContent)
			osRelease, _ := parseOsReleaseFile(reader)
			for releaseInfokey := range tt.expect {
				assert.Equal(t, osRelease[releaseInfokey], tt.expect[releaseInfokey])
			}
		})
	}
}

func TestMergeHostAndOsReleaseInfo(t *testing.T) {
	tests := []struct {
		name        string
		hostRelease *proto.ReleaseInfo
		osRelease   map[string]string
		expect      *proto.ReleaseInfo
	}{
		{
			name: "os-release info present",
			hostRelease: &proto.ReleaseInfo{
				VersionId: "20.04",
				Version:   "5.15.0-1028-aws",
				Codename:  "linux",
				Name:      "debian",
				Id:        "ubuntu",
			},
			osRelease: map[string]string{
				"VERSION_ID":       "20.04",
				"VERSION":          "20.04.5 LTS (Focal Fossa)",
				"VERSION_CODENAME": "focal",
				"NAME":             "Ubuntu",
				"ID":               "ubuntu",
			},
			expect: &proto.ReleaseInfo{
				VersionId: "20.04",
				Version:   "20.04.5 LTS (Focal Fossa)",
				Codename:  "focal",
				Name:      "Ubuntu",
				Id:        "ubuntu",
			},
		},
		{
			name: "os-release info value missing",
			osRelease: map[string]string{
				"VERSION_ID":       "32",
				"VERSION":          "32 (Workstation Edition)",
				"VERSION_CODENAME": "",
				"NAME":             "Fedora",
				"ID":               "fedora",
			},
			hostRelease: &proto.ReleaseInfo{
				VersionId: "32",
				Version:   "Fedora 32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "Fedora",
			},
			expect: &proto.ReleaseInfo{
				VersionId: "32",
				Version:   "32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "fedora",
			},
		},
		{
			name: "os-release info field missing",
			osRelease: map[string]string{
				"VERSION_ID": "32",
				"VERSION":    "32 (Workstation Edition)",
				"NAME":       "Fedora",
				"ID":         "fedora",
			},
			hostRelease: &proto.ReleaseInfo{
				VersionId: "32",
				Version:   "Fedora 32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "Fedora",
			},
			expect: &proto.ReleaseInfo{
				VersionId: "32",
				Version:   "32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "fedora",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseInfo := mergeHostAndOsReleaseInfo(tt.hostRelease, tt.osRelease)
			assert.Equal(t, tt.expect, releaseInfo)
		})
	}
}

func TestGetContainerID(t *testing.T) {
	tests := []struct {
		name      string
		mountInfo string
		expect    string
		expectErr bool
	}{
		{
			name:      "unknown cgroups format",
			mountInfo: envMountInfo[0],
			expect:    "",
			expectErr: true,
		},
		{
			name:      "cgroups v1",
			mountInfo: envMountInfo[1],
			expect:    "f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4",
			expectErr: false,
		},
		{
			name:      "cgroups v2",
			mountInfo: envMountInfo[2],
			expect:    "bc22cfd94f0d32e476d3187519ab39bd8af99ca2af1e3e69c6c82e9c157551be",
			expectErr: false,
		},
		{
			name:      "k8s container",
			mountInfo: envMountInfo[3],
			expect:    "d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80",
			expectErr: false,
		},
		{
			name:      "k8s container 1",
			mountInfo: envMountInfo[4],
			expect:    "c165b8760bf6a5687d806ba33f1da4f78c81fb3f28e3e9568620da989277ee2a",
			expectErr: false,
		},
		{
			name:      "k8s container 2",
			mountInfo: envMountInfo[5],
			expect:    "2c7f9d8e2490c1a83ffdb83fc7e49159b9f670c5f34668b32f3b98adb200d2da",
			expectErr: false,
		},
		{
			name:      "cro-i container",
			mountInfo: envMountInfo[6],
			expect:    "9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c",
			expectErr: false,
		},
		{
			name:      "different var folder location",
			mountInfo: envMountInfo[7],
			expect:    "54aeeb59e870cfa120c741ad7b92381c3ff47d4602b2f18435c49a9857b3e783",
			expectErr: false,
		},
		{
			name:      "minikube containerd",
			mountInfo: envMountInfo[8],
			expect:    "a07a6878dac42baaba3595e04ec6781088428d7c293ff2d6b424876b1e69044d",
			expectErr: false,
		},
		{
			name:      "minikube docker",
			mountInfo: envMountInfo[9],
			expect:    "4bc33970f0c2e3ee7f14c023aff0e7a403c649e7e8b7dd64808ba62479d1a1da",
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountInfoFile, err := os.CreateTemp(os.TempDir(), "mountInfo-")
			if err != nil {
				t.Fatalf("Cannot create temporary file: %v", err)
			}
			if _, err := mountInfoFile.Write([]byte(tt.mountInfo)); err != nil {
				t.Fatalf("Failed to write temp file %s: %v", mountInfoFile.Name(), err)
			}
			mountInfoFile.Close()

			containerID, err := getContainerID(mountInfoFile.Name())
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expect, containerID)

			os.Remove(mountInfoFile.Name())
		})
	}
}

func TestCGroupV1Check(t *testing.T) {
	tests := []struct {
		name      string
		mountInfo string
		expect    bool
		expectErr bool
	}{
		{
			name:      "unknown cgroups format",
			mountInfo: envMountInfo[0],
			expect:    false,
			expectErr: true,
		},
		{
			name:      "cgroups v1",
			mountInfo: envMountInfo[1],
			expect:    true,
			expectErr: false,
		},
		{
			name:      "k8s container",
			mountInfo: envMountInfo[3],
			expect:    true,
			expectErr: false,
		},
		{
			name:      "k8s container 1",
			mountInfo: envMountInfo[4],
			expect:    true,
			expectErr: false,
		},
		{
			name:      "k8s container 2",
			mountInfo: envMountInfo[5],
			expect:    true,
			expectErr: false,
		},
		{
			name:      "cro-i container",
			mountInfo: envMountInfo[6],
			expect:    false,
			expectErr: true,
		},
		{
			name:      "different var folder location",
			mountInfo: envMountInfo[7],
			expect:    true,
			expectErr: false,
		},
		{
			name:      "minikube containerd",
			mountInfo: envMountInfo[8],
			expect:    true,
			expectErr: false,
		},
		{
			name:      "minikube docker",
			mountInfo: envMountInfo[9],
			expect:    true,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountInfoFile, err := os.CreateTemp(os.TempDir(), "cGroupV1Check-")
			if err != nil {
				t.Fatalf("Cannot create temporary file: %v", err)
			}
			if _, err := mountInfoFile.Write([]byte(tt.mountInfo)); err != nil {
				t.Fatalf("Failed to write temp file %s: %v", mountInfoFile.Name(), err)
			}
			mountInfoFile.Close()

			result, err := cGroupV1Check(mountInfoFile.Name())
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expect, result)

			os.Remove(mountInfoFile.Name())
		})
	}
}

func TestGetNginxProcess(t *testing.T) {
	tests := []struct {
		name   string
		pName  string
		cmd    string
		expect bool
	}{
		{
			name:   "nginx process",
			pName:  "nginx",
			cmd:    "nginx: master process /usr/sbin/nginx -c /etc/nginx/nginx.conf",
			expect: true,
		},
		{
			name:   "non nginx process",
			pName:  "nginx-asg-sync",
			cmd:    "/usr/sbin/nginx-asg-sync -log_path=/var/log/nginx-asg-sync/nginx-asg-sync.log",
			expect: false,
		},
		{
			name:   "upgrade process",
			pName:  "nginx-test",
			cmd:    "nginx: upgrade",
			expect: false,
		},
	}

	for _, tt := range tests {
		env := EnvironmentType{}
		isNginxProcess := false
		t.Run(tt.name, func(t *testing.T) {
			isNginxProcess = env.isNginxProcess(tt.pName, tt.cmd)
		})
		assert.Equal(t, tt.expect, isNginxProcess)
	}
}
