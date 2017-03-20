package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"log"

	"github.com/docker/go-plugins-helpers/volume"
)

const glusterfsID = "_glusterfs"

var (
	defaultDir  = filepath.Join(volume.DefaultDockerRootDirectory, glusterfsID)
	serversList = flag.String("servers", "", "List of glusterfs servers")
	gfsBase     = flag.String("gfs-base", "/mnt/gfs", "Base directory where volumes are created in the cluster")
	root        = flag.String("root", defaultDir, "GlusterFS volumes root directory")
	uid         = flag.Int("uid", 0, "UID that should own the newly created mount dir")
	gid         = flag.Int("gid", 0, "GID that should own the newly created mount dir")
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

	driver := newGlusterfsDriver(*root, *gfsBase, servers, uid, gid)
	handler := volume.NewHandler(driver)

	// Try to unmount if there's anything mounted at our path
	// We don't care about an error as any issues will fail when mounting
	err := driver.unmountVolume()
	if err != nil {
		log.Println(err.Error())
	}

	err = driver.mountVolume()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(handler.ServeUnix("root", "glusterfs"))

	err = driver.unmountVolume()
	if err != nil {
		os.Exit(1)
	}
}
