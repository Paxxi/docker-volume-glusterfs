package main

import (
	"fmt"
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
	volumeId    string
	mountPath   string
	client      glusterfs.GlusterClient
	servers     []string
	volumes     map[string]*volumeName
	connections int
	mutex       *sync.Mutex
}

func newGlusterfsDriver(volumeID, mountPath string, servers []string) glusterfsDriver {
	driver := glusterfsDriver{
		volumeId:    volumeID,
		mountPath:   mountPath,
		client:      glusterfs.NewClient(),
		servers:     servers,
		volumes:     map[string]*volumeName{},
		connections: 0,
		mutex:       &sync.Mutex{},
	}
	return driver
}

func (driver glusterfsDriver) Create(request volume.Request) volume.Response {
	log.Printf("Creating volume %s\n", request.Name)
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)

	if _, ok := driver.volumes[mount]; ok {
		return volume.Response{}
	}

	exist, err := driver.client.VolumeExist(request.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	if !exist {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (driver glusterfsDriver) Remove(request volume.Request) volume.Response {
	log.Printf("Removing volume %s\n", request.Name)
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)

	if s, ok := driver.volumes[mount]; ok {
		if s.connections <= 1 {
			if err := driver.client.Unmount(mount); err != nil {
				return volume.Response{Err: err.Error()}
			}
			delete(driver.volumes, mount)
		}
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

	s, ok := driver.volumes[mount]
	if ok && s.connections > 0 {
		s.connections++
		return volume.Response{Mountpoint: mount}
	}

	fi, err := os.Lstat(mount)

	if os.IsNotExist(err) {
		if err := os.MkdirAll(mount, 0755); err != nil {
			return volume.Response{Err: err.Error()}
		}
	} else if err != nil {
		return volume.Response{Err: err.Error()}
	}

	if fi != nil && !fi.IsDir() {
		return volume.Response{Err: fmt.Sprintf("%v already exist and it's not a directory", mount)}
	}

	driver.volumes[mount] = &volumeName{name: request.Name, connections: 1}

	return volume.Response{Mountpoint: mount}
}

func (driver glusterfsDriver) Unmount(request volume.UnmountRequest) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	log.Printf("Unmounting volume %s from %s\n", request.Name, mount)

	if s, ok := driver.volumes[mount]; ok {
		s.connections--
	} else {
		return volume.Response{Err: fmt.Sprintf("Unable to find volume mounted on %s", mount)}
	}

	return volume.Response{}
}

func (driver glusterfsDriver) Get(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	mount := driver.mountpoint(request.Name)
	if s, ok := driver.volumes[mount]; ok {
		return volume.Response{Volume: &volume.Volume{Name: s.name, Mountpoint: driver.mountpoint(s.name)}}
	}

	return volume.Response{Err: fmt.Sprintf("Unable to find volume mounted on %s", mount)}
}

func (driver glusterfsDriver) List(request volume.Request) volume.Response {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	var vols []*volume.Volume
	for _, v := range driver.volumes {
		vols = append(vols, &volume.Volume{Name: v.name, Mountpoint: driver.mountpoint(v.name)})
	}
	return volume.Response{Volumes: vols}
}

func (driver *glusterfsDriver) mountpoint(name string) string {
	return filepath.Join(driver.volumeId, name)
}

func (driver *glusterfsDriver) mountVolume() error {
	err := driver.client.Mount(driver.servers, driver.volumeId, driver.mountPath)
	if err != nil {
		log.Println(fmt.Sprintf("Failed to mount volume %s at %s from servers %s", driver.volumeId, driver.mountPath, strings.Join(driver.servers, ", ")))
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
