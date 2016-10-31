# Docker volume plugin for GlusterFS

## This is quite fresh so use at your own risk

This plugin uses GlusterFS as distributed data storage for containers.

[![TravisCI](https://travis-ci.org/Paxxi/docker-volume-glusterfs.svg)](https://travis-ci.org/Paxxi/docker-volume-glusterfs)

## Installation

Using go (until we get proper binaries):

```
$ go get github.com/Paxxi/docker-volume-glusterfs
```

## Usage

This plugin doesn't create volumes in your GlusterFS cluster and it's currently restricted to a single volume
as that's the current need I have

1 - Start the plugin using this command:

```
$ sudo docker-volume-glusterfs -gfs-base /mnt/gfs -root your-volume-name -server server1:server2:server3
```

We use the flag `-servers` to specify where to find the GlusterFS servers. The server names are separated by colon.

2 - Start your docker containers with the option `--volume-driver=glusterfs` and use the first part of `--volume` to specify the remote volume that you want to connect to:

```
$ sudo docker run --volume-driver glusterfs --volume datastore:/data alpine touch /data/helo
```

## LICENSE

MIT
