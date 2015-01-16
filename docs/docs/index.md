---
title: Building and Running Mesos-DNS
---

## Building And Running Mesos-DNS

### Building Mesos-DNS

To build Mesos-DNS, you need to install `go` on your computer using [these instructions](https://golang.org/doc/install). If you install go to a custom location, make sure that the `GOROOT` environment variable is properly set and that `$GOROOT/bin` is added to `PATH` environment variable. You must set the `GOPATH` environment variable to point to the directory where outside `go` packages will be installed.  

To build Mesos-DNS: 

```
go get github.com/miekg/dns
go get github.com/mesosphere/mesos-dns
cd $GOPATH/src/github.com/mesosphere/mesos-dns
go build -o mesos-dns main.go
``` 

`mesos-dns` is a statically-linked binary that can be installed anywhere. We have built and tested Mesos-DNS with `go` versions 1.3.3 and 1.4. Newer versions of `go` should work as well. 


### Running Mesos-DNS

To run Mesos-DNS, you first need to install the `mesos-dns` binary somewhere a selected server. The server can be the same machine as one of the Mesos masters, one of the slaves, or a dedicated machine on the same network. Next, follow [these instructions](configuration-parameters.html) to create a configuration file for your cluster. You can launch Mesos-DNS with: 

```
sudo mesos-dns -config=config.json & 
```

For fault tolerance, we ***recommend*** that you use [Marathon](https://mesosphere.github.io/marathon) to launch Mesos-DNS on one of the Mesos slaves. If Mesos-DNS fails, Marathon will re-launch it immediately, ensuring nearly uninterrupted service. You can select which slave is used for Mesos-DNS with [Marathon constraints](https://github.com/mesosphere/marathon/blob/master/docs/docs/constraints.md) on the slave hostname or any slave attribute. For example, the following json description instructs Marathon to launch Mesos-DNS on the slave with hostname `10.181.64.13`:

```
{
"cmd": "sudo  /usr/local/mesos-dns/mesos-dns -config=/usr/local/mesos-dns/config.json",
"cpus": 1.0, 
"mem": 1024,
"id": "mesos-dns",
"instances": 1,
"constraints": [["hostname", "CLUSTER", "10.181.64.13"]]
}
```
Note that the `hostname` field refers to the hostname by the slave when it registers with Mesos. It may not be an IP address or a valid hostname of any kind. 


### Slave Setup

To allow Mesos tasks to access Mesos-DNS, you must edit the file `/etc/resolv.conf` and add a new nameserver. For instance, if `mesos-dns` runs on the server with IP address `10.181.64.13`, you should add the line `nameserver 10.181.64.13` at the ***beginning*** of `/etc/resolv.conf` of every slave node. This can be achieve by running:

```
sudo sed -i 'nameserver 10.181.64.13' /etc/resolv.conf
```

If multiple instances of Mesos-DNS are launched, add a nameserver line for each one at the beginning of `/etc/resolv.conf`. The order of these entries determines the order that the slave will use to contact Mesos-DNS instances. 

All other nameserver settings in `/etc/resolv.conf` should remain unchanged. 
