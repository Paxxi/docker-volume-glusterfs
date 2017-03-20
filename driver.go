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
	uid       *int
	gid       *int
	mutex     *sync.Mutex
}

func newGlusterfsDriver(volumeID, mountPath string, servers []string, uid *int, gid *int) glusterfsDriver {
	driver := glusterfsDriver{
		volumeID:  volumeID,
		mountPath: mountPath,
		client:    glusterfs.NewClient(),
		servers:   servers,
		mutex:     &sync.Mutex{},
		uid:       uid,
		gid:       gid,
	}
	return driver
}

func (driver glusterfsDriver) Create(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	log.Printf("Driver::Create Creating volume %s at %s\n", request.Name, mount)

	fi, err := os.Lstat(mount)
	if err == nil && fi.IsDir() {
		log.Printf("Driver::Create volume exists and is a directory")
		return volume.Response{}
	} else if err == nil {
		log.Print("Driver::Create volume exists and is not a directory")
		return volume.Response{Err: fmt.Sprintf("path with name %s already exists and is not a directory", request.Name)}
	}

	if os.IsNotExist(err) {
		log.Print("Driver::Create no such volume exists, will be created")
		return volume.Response{}
	}

	log.Print("Driver::Create something bad happened, volume exists but is not a directory or file")
	return volume.Response{Err: fmt.Sprintf("Volume with name %s already exists", request.Name)}
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
	log.Printf("Driver::Mount Mounting volume %s on %s\n", request.Name, mount)

	err := os.MkdirAll(mount, 0777)
	if err == nil {
		log.Printf("Driver::Mount created volume successfuly")
		if err = driver.chown(mount, driver.uid, driver.gid); err != nil {
			log.Printf("Driver::Mount Failed to chown mount, continuing")
		}
		return volume.Response{Mountpoint: mount}
	}

	fi, err := os.Lstat(mount)
	if err != nil {
		log.Printf("Driver::Mount lstat failed %s", err.Error())
		return volume.Response{Err: err.Error()}
	}

	if fi.IsDir() {
		log.Printf("Driver::Mount volume is a directory, returning success")
		if err = driver.chown(mount, driver.uid, driver.gid); err != nil {
			log.Printf("Driver::Mount Failed to chown mount, continuing")
		}
		return volume.Response{Mountpoint: mount}
	}

	log.Printf("Driver::Mount %v already exist and it's not a directory", mount)
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
	return volume.Response{Volume: &volume.Volume{Name: request.Name, Mountpoint: mount}}
}

func (driver glusterfsDriver) List(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	fileList, err := ioutil.ReadDir(driver.mountPath)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	vols := make([]*volume.Volume, 0)

	for _, fi := range fileList {
		if fi.IsDir() {
			mount := driver.mountpoint(fi.Name())
			log.Printf("Found dir %s with mountpoint %s\n", fi.Name(), mount)
			vols = append(vols, &volume.Volume{Name: fi.Name(), Mountpoint: mount})
		}
	}
	return volume.Response{Volumes: vols}
}

func (driver *glusterfsDriver) mountpoint(name string) string {
	return filepath.Join(driver.mountPath, name)
}

func (driver *glusterfsDriver) chown(name string, uid *int, gid *int) error {
	if uid == nil && gid == nil {
		return nil
	}

	return os.Chown(name, *uid, *gid)
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
