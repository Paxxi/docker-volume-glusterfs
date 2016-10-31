package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Paxxi/docker-volume-glusterfs/glusterfs"
	"github.com/docker/go-plugins-helpers/volume"
)

type volumeName struct {
	name        string
	connections int
}

type glusterfsDriver struct {
	volumeID  string
	mountPath string
	client    glusterfs.GlusterClient
	servers   []string
	mutex     *sync.Mutex
}

func newGlusterfsDriver(volumeID, mountPath string, servers []string) glusterfsDriver {
	driver := glusterfsDriver{
		volumeID:  volumeID,
		mountPath: mountPath,
		client:    glusterfs.NewClient(),
		servers:   servers,
		mutex:     &sync.Mutex{},
	}
	return driver
}

func (driver glusterfsDriver) Create(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	log.Printf("Creating volume %s at %s\n", request.Name, mount)

	_, err := os.Lstat(mount)

	if os.IsNotExist(err) {
		return volume.Response{}
	}

	return volume.Response{Err: err.Error()}
}

func (driver glusterfsDriver) Remove(request volume.Request) volume.Response {
	log.Printf("Removing volume %s\n", request.Name)
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)

	err := os.RemoveAll(mount)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (driver glusterfsDriver) Path(request volume.Request) volume.Response {
	return volume.Response{Mountpoint: driver.mountpoint(request.Name)}
}

func (driver glusterfsDriver) Mount(request volume.MountRequest) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	log.Printf("Mounting volume %s on %s\n", request.Name, mount)

	err := os.MkdirAll(mount, 0755)
	if err == nil {
		return volume.Response{Mountpoint: mount}
	}

	fi, err := os.Lstat(mount)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	if fi.IsDir() {
		return volume.Response{Mountpoint: mount}
	}

	return volume.Response{Err: fmt.Sprintf("%v already exist and it's not a directory", mount)}
}

func (driver glusterfsDriver) Unmount(request volume.UnmountRequest) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	log.Printf("Unmounting volume %s from %s\n", request.Name, mount)

	return volume.Response{}
}

func (driver glusterfsDriver) Get(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	fi, err := os.Lstat(mount)
	if err != nil || !fi.IsDir() {
		return volume.Response{Err: fmt.Sprintf("Unable to find volume mounted on %s", mount)}
	}
	return volume.Response{Mountpoint: mount}
}

func (driver glusterfsDriver) List(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	fileList, err := ioutil.ReadDir(driver.mountPath)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	vols := make([]*volume.Volume, len(fileList))

	for _, fi := range fileList {
		if fi.IsDir() {
			vols = append(vols, &volume.Volume{Name: fi.Name(), Mountpoint: driver.mountpoint(fi.Name())})
		}
	}
	return volume.Response{Volumes: vols}
}

func (driver *glusterfsDriver) mountpoint(name string) string {
	return filepath.Join(driver.mountPath, name)
}

func (driver *glusterfsDriver) mountVolume() error {
	err := driver.client.Mount(driver.servers, driver.volumeID, driver.mountPath)
	if err != nil {
		log.Println(fmt.Sprintf("Failed to mount volume %s at %s from servers %s", driver.volumeID, driver.mountPath, strings.Join(driver.servers, ", ")))
		return err
	}

	return nil
}

func (driver *glusterfsDriver) unmountVolume() error {
	err := driver.client.Unmount(driver.mountPath)
	if err != nil {
		log.Println("Failed to unmount volume: ", driver.mountPath)
		return err
	}

	return nil
}

func (driver glusterfsDriver) Capabilities(request volume.Request) volume.Response {
	var res volume.Response
	res.Capabilities = volume.Capability{Scope: "global"}
	return res
}
