---
title: Mesos-DNS FAQ
---

##  Frequently Asked Questions

---

#### Mesos-DNS fails to launch

Make sure that the port used for Mesos-DNS is available and not used by another program. To use the recommended port `53`, must launch Mesos-DNS as root. 

---

#### Mesos-DNS does resolve names in the Mesos domain

Check the configuration file to make sure that Mesos-DNS is directed to right master(s) for the Mesos cluster (`masters`). 
 
---

#### Mesos-DNS does resolve names outside of the Mesos domain

Check the configuration file to make sure that Mesos-DNS is configured with the IP address of  external DNS servers (`resolvers`).
