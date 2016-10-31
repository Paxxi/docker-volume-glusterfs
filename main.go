package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
)

const glusterfsID = "_glusterfs"

var (
	defaultDir  = filepath.Join(volume.DefaultDockerRootDirectory, glusterfsID)
	serversList = flag.String("servers", "", "List of glusterfs servers")
	gfsBase     = flag.String("gfs-base", "/mnt/gfs", "Base directory where volumes are created in the cluster")
	root        = flag.String("root", defaultDir, "GlusterFS volumes root directory")
)

func main() {
	var Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if len(*serversList) == 0 {
		Usage()
		os.Exit(1)
	}

	servers := strings.Split(*serversList, ":")

	driver := newGlusterfsDriver(*root, *gfsBase, servers)
	handler := volume.NewHandler(driver)
	err := driver.mountVolume()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(handler.ServeUnix("root", "glusterfs"))

	err = driver.unmountVolume()
	if err != nil {
		os.Exit(1)
	}
}
