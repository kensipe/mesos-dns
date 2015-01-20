mesos-dns
=========

DNS for service discovery with Mesos. 
Refer to the [initial design document](https://docs.google.com/a/mesosphere.io/document/d/1h-ptANif4RZNWKTAJXsG0s4ZjfpY7GLrY2zRwmrIBAc/edit?usp=sharing) for details. 

__Install go__
  ```shell
  sudo apt-get install git-core
  wget https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz
  tar xzf go*
  sudo mv go /usr/local/.
  # puts this into ~/.profile
  export PATH=$PATH:/usr/local/go/bin
  export GOROOT=/usr/local/go
  export PATH=$PATH:$GOROOT/bin
  export GOPATH=$HOME/go
  ```
 
__Build mesos-dns__

  ```shell
  go get github.com/miekg/dns
  go get github.com/mesosphere/mesos-dns
  # git clone git@github.com:mesosphere/mesos-dns.git
  cd $GOPATH/src/github.com/mesosphere/mesos-dns
  go build -o mesos-dns main.go
  ```

__Configure mesos-dns server__
  ```shell
  cp config.json.sample config.json 
  # edit config.json
  # "masters" --> list of IP addresses for mesos masters
  # "port" --> REST API port for mesos masters
  # "resolver" --> port for mesos-dns (53 is strongly recommended)
  # "domain" --> the DNS domain for the mesos cluster
  ```
__Configure mesos slaves__
  ```
  # edit /etc/resolv.conf
  # add "nameserver <ip address of mesos-dns>" at the very beginning
  ```

  You can use the exact same resolv.conf for both the dns server and the
  mesos slave:
  ```
  domain c.buoyant-keel-783.internal.
  search c.buoyant-keel-783.internal. 163602273704.google.internal. google.internal.

  nameserver 10.128.208.110
  nameserver 169.254.169.254
  nameserver 10.0.0.1
  ```

__Run__
  ```shell
  // root only needed if you are using port 53 (recommended)
  sudo ./mesos-dns
  ```

__Warnings__
* make sure that mesos-dns uses a port that is not blocked 
