package args

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/google/uuid"
)

type NodeConfig struct {
	Name       string            `json:"name"`
	Cpus       int               `json:"cpus"`
	Mem        resource.Quantity `json:"mem"`
	Uuid       uuid.UUID         `json:"uuid"`
	RootfsUuid uuid.UUID         `json:"rootfs_uuid"`
	Image      string            `json:"image"`
	NetMacAddr string            `json:"net_mac_addr"`
	NetIfName  string            `json:"net_if_name"`
	Cmdline    []string          `json:"cmdline"`
	Disk       []string          `json:"disk"`
	Directory  []string          `json:"directory"`
}

func (i *NodeConfig) MachineId() string { return strings.ReplaceAll(i.Uuid.String(), "-", "") }
func (i *NodeConfig) ImageBasePath(imageRoot string, rest ...string) string {
	args := []string{imageRoot, i.Image}
	args = append(args, rest...)
	return filepath.Join(args...)
}
func (i *NodeConfig) NodeBasePath(nodeRoot string, rest ...string) string {
	args := []string{nodeRoot, i.Name}
	args = append(args, rest...)
	return filepath.Join(args...)
}
func (i *NodeConfig) KernelCommandline(partUuid string, machineId string) string {
	args := append([]string(nil), i.Cmdline...)
	args = append(args, fmt.Sprintf("base=UUID=%s", partUuid))
	args = append(args, fmt.Sprintf("systemd.machine_id=%s", machineId))
	return strings.Join(args, " ")
}
func (i *NodeConfig) PlatformArg(name string, machineId string, nodeUUID string) string {
	args := append([]string(nil), i.Cmdline...)
	args = append(args, fmt.Sprintf("oem_strings=amuzes-%s", name))
	args = append(args, fmt.Sprintf("serial_number=%s", machineId))
	args = append(args, fmt.Sprintf("uuid=%s", nodeUUID))
	return strings.Join(args, ",")
}

func (i *NodeConfig) VirtiofsArgs(directory string, sockPath string) []string {
	var args []string
	args = append(args, "--allow-direct-io")
	args = append(args, "--announce-submounts")
	args = append(args, "--writeback")
	args = append(args, "--xattr")
	args = append(args, "--posix-acl")
	args = append(args, "--thread-pool-size", strconv.Itoa(i.Cpus))
	args = append(args, "--cache", "auto")
	args = append(args, "--inode-file-handles=prefer")
	args = append(args, "--shared-dir", directory)
	args = append(args, "--socket-path", sockPath)
	return args
}

func (i *NodeConfig) CommandArgs(imageRoot string, nodeRoot string) []string {
	machineId := i.MachineId()
	nodeUUID := i.Uuid.String()

	var args []string
	args = append(args, "--platform", i.PlatformArg(i.Name, machineId, nodeUUID))
	args = append(args, "--kernel", i.ImageBasePath(imageRoot, "vmlinuz"))
	args = append(args, "--initramfs", i.ImageBasePath(imageRoot, "initramfs.img"))
	args = append(args, "--cmdline", i.KernelCommandline(i.RootfsUuid.String(), machineId))
	args = append(args, "--cpus", fmt.Sprintf("boot=%d", i.Cpus))
	args = append(args, "--memory", fmt.Sprintf("size=%s,shared=on,mergeable=on,thp=on", strings.TrimSuffix(i.Mem.String(), "i")))
	args = append(args, "--console", "pty")
	args = append(args, "--serial", "off")

	args = append(args, "--serial", "off")

	args = append(args, "--api-socket", fmt.Sprintf("path=%s", i.NodeBasePath(nodeRoot, "run", "monitor.sock")))
	args = append(args, "--net", fmt.Sprintf("mac=%s,host_mac=,tap=%s,ip=,mask=,num_queues=2,queue_size=128", i.NetMacAddr, i.NetIfName))
	for _, filename := range i.Directory {
		args = append(args, "--fs", fmt.Sprintf("tag=%s,socket=%s,num_queues=1,queue_size=1024", filename, i.NodeBasePath(nodeRoot, "run", fmt.Sprintf("virtiofsd_%s.sock", filename))))
	}
	args = append(args, "--disk", fmt.Sprintf("path=%s,direct=on,readonly=on,num_queues=2,queue_size=128", i.ImageBasePath(imageRoot, "root.img")))
	for _, filename := range i.Disk {
		args = append(args, "--disk", fmt.Sprintf("path=%s,direct=on,readonly=on,num_queues=2,queue_size=128", i.NodeBasePath(nodeRoot, filename)))
	}
	args = append(args, "--watchdog")
	args = append(args, "--pvpanic")

	return args
}
