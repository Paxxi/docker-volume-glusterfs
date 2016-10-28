package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/calavera/docker-volume-glusterfs/rest"
	"github.com/docker/go-plugins-helpers/volume"
)

type volumeName struct {
	name        string
	connections int
}

type glusterfsDriver struct {
	root       string
	restClient *rest.Client
	servers    []string
	volumes    map[string]*volumeName
	mutex      *sync.Mutex
}

func newGlusterfsDriver(root, restAddress, gfsBase string, servers []string) glusterfsDriver {
	driver := glusterfsDriver{
		root:    root,
		servers: servers,
		volumes: map[string]*volumeName{},
		mutex:   &sync.Mutex{},
	}
	if len(restAddress) > 0 {
		driver.restClient = rest.NewClient(restAddress, gfsBase)
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

	if driver.restClient != nil {
		exist, err := driver.restClient.VolumeExist(request.Name)
		if err != nil {
			return volume.Response{Err: err.Error()}
		}

		if !exist {
			if err := driver.restClient.CreateVolume(request.Name, driver.servers); err != nil {
				return volume.Response{Err: err.Error()}
			}
		}
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
			if driver.restClient != nil {
				if err := driver.restClient.StopVolume(request.Name); err != nil {
					return volume.Response{Err: err.Error()}
				}
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

	if err := driver.mountVolume(request.Name, mount); err != nil {
		return volume.Response{Err: err.Error()}
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
		if s.connections == 1 {
			if err := driver.unmountVolume(mount); err != nil {
				return volume.Response{Err: err.Error()}
			}
		}
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
	return filepath.Join(driver.root, name)
}

func (driver *glusterfsDriver) mountVolume(name, destination string) error {
	var serverNodes []string
	for _, server := range driver.servers {
		serverNodes = append(serverNodes, fmt.Sprintf("-s %s", server))
	}

	cmd := fmt.Sprintf("glusterfs --volfile-id=%s %s %s", name, strings.Join(serverNodes[:], " "), destination)
	if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

func (driver *glusterfsDriver) unmountVolume(target string) error {
	cmd := fmt.Sprintf("umount %s", target)
	if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

func (driver glusterfsDriver) Capabilities(request volume.Request) volume.Response {
	var res volume.Response
	res.Capabilities = volume.Capability{Scope: "local"}
	return res
}
