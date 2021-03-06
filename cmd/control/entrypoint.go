package control

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/rancher/os/log"
	"golang.org/x/net/context"

	"github.com/rancher/os/cmd/cloudinitexecute"
	"github.com/rancher/os/config"
	"github.com/rancher/os/docker"
	"github.com/rancher/os/util"
)

const (
	ca     = "/etc/ssl/certs/ca-certificates.crt"
	caBase = "/etc/ssl/certs/ca-certificates.crt.rancher"
)

func entrypointAction(c *cli.Context) error {
	if _, err := os.Stat("/host/dev"); err == nil {
		cmd := exec.Command("mount", "--rbind", "/host/dev", "/dev")
		if err := cmd.Run(); err != nil {
			log.Errorf("Failed to mount /dev: %v", err)
		}
	}

	if err := util.FileCopy(caBase, ca); err != nil && !os.IsNotExist(err) {
		log.Error(err)
	}

	cfg := config.LoadConfig()

	shouldWriteFiles := false
	for _, file := range cfg.WriteFiles {
		if file.Container != "" {
			shouldWriteFiles = true
		}
	}

	if shouldWriteFiles {
		writeFiles(cfg)
	}

	setupCommandSymlinks()

	if len(os.Args) < 3 {
		return nil
	}

	binary, err := exec.LookPath(os.Args[2])
	if err != nil {
		return err
	}

	return syscall.Exec(binary, os.Args[2:], os.Environ())
}

func writeFiles(cfg *config.CloudConfig) error {
	id, err := util.GetCurrentContainerID()
	if err != nil {
		return err
	}
	client, err := docker.NewSystemClient()
	if err != nil {
		return err
	}
	info, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return err
	}

	cloudinitexecute.WriteFiles(cfg, info.Name[1:])
	return nil
}

func setupCommandSymlinks() {
	for _, link := range []symlink{
		{config.RosBin, "/usr/bin/cloud-init-execute"},
		{config.RosBin, "/usr/bin/cloud-init-save"},
		{config.RosBin, "/usr/bin/dockerlaunch"},
		{config.RosBin, "/usr/bin/respawn"},
		{config.RosBin, "/usr/bin/system-docker"},
		{config.RosBin, "/usr/sbin/netconf"},
		{config.RosBin, "/usr/sbin/wait-for-docker"},
		{config.RosBin, "/usr/sbin/poweroff"},
		{config.RosBin, "/usr/sbin/reboot"},
		{config.RosBin, "/usr/sbin/halt"},
		{config.RosBin, "/usr/sbin/shutdown"},
		{config.RosBin, "/sbin/poweroff"},
		{config.RosBin, "/sbin/reboot"},
		{config.RosBin, "/sbin/halt"},
		{config.RosBin, "/sbin/shutdown"},
	} {
		os.Remove(link.newname)
		if err := os.Symlink(link.oldname, link.newname); err != nil {
			log.Error(err)
		}
	}
}
