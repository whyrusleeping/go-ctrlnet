package ctrlnet

import (
	"fmt"
	"os/exec"
	"strings"
)

type LinkSettings struct {
	Latency    int
	Jitter     int
	Bandwidth  int
	PacketLoss int
}

func (ls *LinkSettings) cmd(iface string) []string {
	base := []string{"tc", "qdisc", "change", "dev", iface, "root", "netem"}

	// even if latency is zero, put it on so the command never fails
	base = append(base, "delay", fmt.Sprintf("%dms", ls.Latency))
	if ls.Jitter > 0 {
		base = append(base, fmt.Sprintf("%dms", ls.Jitter), "distribution", "normal")
	}

	if ls.Bandwidth > 0 {
		base = append(base, "rate", fmt.Sprint(ls.Bandwidth))
	}

	if ls.PacketLoss > 0 {
		base = append(base, "loss", fmt.Sprintf("%d%%", ls.PacketLoss))
	}

	return base
}

func initLink(name string) error {
	out, err := exec.Command("tc", "qdisc", "show").CombinedOutput()
	if err != nil {
		return fmt.Errorf("dev listing failed: %s - %s", string(out), err)
	}

	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.Contains(l, name) {
			return nil
		}
	}

	return SetLink(name, new(LinkSettings))
}

func SetLink(name string, settings *LinkSettings) error {
	err := initLink(name)
	if err != nil {
		return err
	}
	args := settings.cmd(name)
	c := exec.Command(args[0], args[1:]...)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting link: %s - %s", string(out), err)
	}

	return nil
}
