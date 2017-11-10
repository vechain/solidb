# SoliDB

A Distributed Content-Addressable Storage

## Features

* Immutability
* Scalability
* Fault-tolerance
* High availability

## Components

### Node	

Node runs as a service to manage slices of the whole data collection. A cluster is composed with several nodes.

### Master

A command line tool to manage cluster.

## Quickstart

### Get source code

```shell
$ git clone https://github.com/vechain/solidb.git
```

### Compile & install

Golang tool chain is required to be installed. Also, [glide](https://github.com/Masterminds/glide) is required for package management.

```shell
$ cd solidb
$ glide install
$ make install
```

You'll get *solidb* command in $GOPATH/bin.

### Start a node

```shell
$ solidb node
```

To get help for more options:

```shell
$ solidb node -h
```

### Maintain

#### Create a cluster

```shell
$ solidb new path-of-master-dir
```
  
  Master dir can be relative or absolute path. To specify number of redundant copies for the whole data collection, add option '—replicas n', where n must be >= 1, defaults to 2.
  
  
  Enter master dir to perform further configurations.
	
```shell
$ cd path-of-master-dir.solidb
```

#### Add node

```shell
$ solidb add addr-of-node
```  

#### Display list of nodes

```shell
$ solidb list
```

#### Remove node

  Remove first node in the list.
  
```shell
$ solidb remove 0
```
  Addr or ID can also be specified as argument of *remove* command.  
  
#### Alternate node
  
```shell
$ solidb alter 0 --addr new-addr --weight 2
```  

#### Propose config

  Generate cluster config according to current draft, and send the config to all nodes.

```shell
$ solidb propose
```
  *The count of nodes in cluster should be >= replicas, or an error will be printed.*

#### Sync command
  
  Tell all nodes in newest config to sync slices that are allocated.

```shell
$ solidb sync
```

#### Query status of nodes

```shell
$ solidb status
```
  This command will display status of nodes in proposed config, e.g.
  
```shell
a97c…0ecc	192.168.31.182:3001	3,3,2	103/103
abe0…0a10	192.168.31.182:3002	3,3,2	103/103
a121…d57e	192.168.31.182:3003	3,3,2	102/102
1617…b42f	192.168.31.182:3004	3,3,2	102/102
34a5…d372	192.168.31.182:3005	3,3,2	102/102
```

column 0: ID of node
column 1: address of node
column 3: config revisions, newest/synced/approved
column 4: synced slice count/total slice count

#### Approve config

Once all nodes achieve 'synced' state, the 'approve' command can be sent to make newest config take effect.

```shell
$ solidb approve
```

### Access Blobs
We call blobs for data stored in solidb. The content type of blob is not cared about.

To store a blob:

```shell
$ curl -X POST -d "hello world" http://addr-of-one-node/blobs
{"key":"256c83b297114d201b30179f3f0ef0cace9783622da5974326b436178aeef6"}
```

to retrive it back:

```shell
$ curl http://addr-of-one-node/blobs/256c83b297114d201b30179f3f0ef0cace9783622da5974326b436178aeef6
hello world
```


