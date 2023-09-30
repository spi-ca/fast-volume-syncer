package args

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"

	"strings"
	"testing"

	"github.com/ghodss/yaml"
)

func TestUnit(t *testing.T) {

	memorySize := strings.TrimSuffix(resource.NewQuantity(5*1024*1024*1024, resource.BinarySI).String(), "i")
	fmt.Printf("memorySize = %v\n", memorySize)
	q, err := resource.ParseQuantity(fmt.Sprintf("%si", "5G"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("memorySize = %v\n", q)

}

func TestCfg(t *testing.T) {
	config := []byte(`name: kube-master-01
cpus: 2
mem: 4Gi
uuid: 00000000-0000-4000-8000-000000000000
rootfs_uuid: 00000000-0000-4000-8000-000000000000
image: kube-master
net_mac_addr: 2e:f4:5f:11:1b:56
net_if_name: vmtap-km01
cmdline:
- console=hvc0
- cpuidle.governor=haltpoll
- clocksource=kvm-clock
- net.ifnames=0
- quiet
- loglevel=3
disk:
- data.img
directory:
- configuration`)
	cfg := &NodeConfig{}

	j, err := yaml.YAMLToJSON(config)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(j, cfg)

	if err != nil {
		panic(err)
	}

	cfg_path := cfg.NodeBasePath("/srv/vmm/nodes", "configuration")
	cfg_sock := cfg.NodeBasePath("/srv/vmm/nodes", "run", "virtiofsd_configuration.sock")

	fmt.Printf("v = %s\n",
		strings.Join(cfg.VirtiofsArgs(cfg_path, cfg_sock), " \\\n\t"),
	)
	fmt.Printf("v = %v\n", cfg)
	fmt.Printf("v = %s\n",
		strings.Join(cfg.CommandArgs("/srv/vmm/images", "/srv/vmm/nodes"), " \\\n\t"),
	)
	//
	//newFd, err := syscall.Open(".", syscall.O_TMPFILE|os.O_RDWR|os.O_CREATE|os.O_APPEND|syscall.O_CLOEXEC, 0o644)
	//if err != nil {
	//	_ = os.Rename(rotateFilename, filename)
	//	return "", fmt.Errorf("failed to rename a file(%s): %w", filename, err)
	//}

}
