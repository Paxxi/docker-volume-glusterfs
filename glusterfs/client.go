package glusterfs

import (
	"log"
	"os/exec"
	"strings"
)

// GlusterVolume represents the available gluster volumes.
type GlusterVolume struct {
	Name string
}

// GlusterClient is the http client that sends requests to the gluster API.
type GlusterClient struct {
}

// NewClient initializes a new client.
func NewClient() GlusterClient {
	return GlusterClient{}
}

// VolumeExist returns whether a volume exist in the cluster with a given name or not.
func (client GlusterClient) VolumeExist(name string) (bool, error) {
	vols, err := client.Volumes()
	if err != nil {
		return false, err
	}

	for _, v := range vols {
		if v.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// Volumes returns a list of available gluster volumes
func (client GlusterClient) Volumes() ([]GlusterVolume, error) {
	command := exec.Command("gluster", "volume", "list")
	output, err := command.Output()
	if err != nil {
		return nil, err
	}

	outString := string(output)
	lines := strings.Split(outString, "\n")
	volumes := make([]GlusterVolume, len(lines))
	for _, v := range lines {
		volumes = append(volumes, GlusterVolume{v})
	}
	return volumes, nil
}

// Mount takes the server, volume and mountPath and mounts the gluster volume at the specified path
func (client GlusterClient) Mount(servers []string, volume string, mountPath string) error {
	args := make([]string, 2*len(servers)+3)

	args = append(args, "--volfile-id", volume)
	for _, server := range servers {
		args = append(args, "-s", server)
	}

	// mount path needs to be last
	args = append(args, mountPath)
	command := exec.Command("/usr/sbin/glusterfs", args...)
	err := command.Run()
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

// Unmount unmounts the volume
func (client GlusterClient) Unmount(mountPath string) error {
	command := exec.Command("umount", mountPath)
	err := command.Run()
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

// CreateVolume creates a new volume with the given name in the cluster.
// func (client GlusterClient) CreateVolume(name string, peers []string) error {
// 	u := fmt.Sprintf("%s%s", client.addr, fmt.Sprintf(volumeCreatePath, name))
// 	fmt.Println(u)

// 	bricks := make([]string, len(peers))
// 	for i, p := range peers {
// 		bricks[i] = fmt.Sprintf("%s:%s", p, filepath.Join(client.base, name))
// 	}

// 	params := url.Values{
// 		"bricks":    {strings.Join(bricks, ",")},
// 		"replica":   {strconv.Itoa(len(peers))},
// 		"transport": {"tcp"},
// 		"start":     {"true"},
// 		"force":     {"true"},
// 	}

// 	resp, err := http.PostForm(u, params)
// 	if err != nil {
// 		return err
// 	}

// 	return responseCheck(resp)
// }

// // StopVolume stops the volume with the given name in the cluster.
// func (client GlusterClient) StopVolume(name string) error {
// 	u := fmt.Sprintf("%s%s", client.addr, fmt.Sprintf(volumeStopPath, name))

// 	req, err := http.NewRequest("PUT", u, nil)
// 	if err != nil {
// 		return err
// 	}

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return err
// 	}

// 	return responseCheck(resp)
// }
