package ctrlnet

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

type LinkSettings struct {
	// Latency between links (in milliseconds)
	Latency uint

	// Jitter in latency values (in milliseconds)
	Jitter uint

	// Bandwidth of link (in bits per second)
	Bandwidth uint

	// Upload bandwidth
	Upload uint

	// PacketLoss percentage on the links (in whole percentage points)
	PacketLoss uint8
}

func (ls *LinkSettings) cmd(iface string, init bool) []string {
	var cmd = "change"
	if init {
		cmd = "add"
	}

	base := []string{"tc", "qdisc", cmd, "dev", iface, "root", "netem"}

	// even if latency is zero, put it on so the command never fails
	base = append(base, "delay", fmt.Sprintf("%dms", ls.Latency))
	if ls.Jitter != 0 {
		base = append(base, fmt.Sprintf("%dms", ls.Jitter), "distribution", "normal")
	}

	if ls.Bandwidth != 0 {
		base = append(base, "rate", fmt.Sprint(ls.Bandwidth))
	}

	if ls.PacketLoss != 0 {
		base = append(base, "loss", fmt.Sprintf("%d%%", ls.PacketLoss))
	}

	return base
}

func SetHTB(iface string, settings *LinkSettings) error {
	doinit, err := initLink(iface)
	if err != nil {
		return err
	}
	var cmd = "change"
	if doinit {
		cmd = "add"
	}

	configureparent := []string{"tc", "qdisc", cmd, "dev", iface, "root", "handle", "1:", "htb", "default", "20"}
	out, err := exec.Command(configureparent[0], configureparent[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error configuring parent qdisc: %s - %s", string(out), err)
	}

	setupload := []string{"tc", "class", cmd, "dev", iface, "parent", "1:", "classid", "1:1", "htb"}
	if settings.Upload != 0 {
		setupload = append(setupload, "rate", fmt.Sprint(settings.Upload), "ceil", fmt.Sprint(settings.Upload))
	}
	out, err = exec.Command(setupload[0], setupload[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting upload bandwidth: %s - %s", string(out), err)
	}

	out, err = exec.Command("tc", "filter", "add", "dev", iface, "protocol", "ip", "parent", "1:", "prio", "1", "matchall", "flowid", "1:1").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error configuring class filter: %s - %s", string(out), err)
	}

	return nil
}

func initLink(name string) (bool, error) {
	out, err := exec.Command("tc", "qdisc", "show").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("dev listing failed: %s - %s", string(out), err)
	}

	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.Contains(l, name) && strings.Contains(l, "netem") {
			return false, nil
		}
	}

	return true, nil
}

func SetLink(name string, settings *LinkSettings) error {
	doinit, err := initLink(name)
	if err != nil {
		return err
	}
	args := settings.cmd(name, doinit)
	c := exec.Command(args[0], args[1:]...)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting link: %s - %s", string(out), err)
	}

	return nil
}

func GetInterfaces(filter string) ([]string, error) {
	ifs, err := ioutil.ReadDir("/sys/devices/virtual/net")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, i := range ifs {
		if strings.Contains(i.Name(), filter) {
			out = append(out, i.Name())
		}
	}
	return out, nil
}
