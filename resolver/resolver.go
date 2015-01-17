// package resolver contains functions to handle resolving .mesos
// domains
package resolver

import (
	"errors"
	"github.com/mesosphere/mesos-dns/logging"
	"github.com/mesosphere/mesos-dns/records"
	"github.com/miekg/dns"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	recurseCnt = 3
)

// resolveOut queries other nameserver
// randomly picks from the list that is not mesos
func (res *Resolver) resolveOut(r *dns.Msg, nameserver string, proto string, cnt int) (*dns.Msg, error) {

	var in *dns.Msg
	var err error

	c := new(dns.Client)
	c.Net = proto

	var t time.Duration = 5 * 1e9
	if res.Config.Timeout != 0 {
		t = time.Duration(int64(res.Config.Timeout * 1e9))
	}

	c.DialTimeout = t
	c.ReadTimeout = t
	c.WriteTimeout = t

	in, _, err = c.Exchange(r, nameserver)

	// recurse
	if in != nil && len(in.Answer) == 0 && !in.MsgHdr.Authoritative && len(in.Ns) > 0 {

		if cnt == recurseCnt {
			logging.CurLog.NonMesosRecursed += 1
		}

		if cnt > 0 {

			if soa, ok := (in.Ns[0]).(*dns.SOA); ok {
				return res.resolveOut(r, soa.Ns+":53", proto, cnt-1)
			}
		}

	}

	return in, err
}

// cleanWild strips any wildcards out thus mapping cleanly to the
// original serviceName
func cleanWild(dom string) string {
	if strings.Contains(dom, ".*") {
		return strings.Replace(dom, ".*", "", -1)
	} else {
		return dom
	}
}

// splitDomain splits dom into host and port pair
func (res *Resolver) splitDomain(dom string) (host string, port int) {
	s := strings.Split(dom, ":")
	host = s[0]

	// As won't have ports
	if len(s) == 1 {
		return host, 0
	} else {
		port, _ = strconv.Atoi(s[1])
		return host, port
	}
}

// formatSRV returns the SRV resource record for target
func (res *Resolver) formatSRV(name string, target string) (*dns.SRV, error) {
	ttl := uint32(res.Config.TTL)

	h, p := res.splitDomain(target)

	return &dns.SRV{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeSRV,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Priority: 0,
		Weight:   0,
		Port:     uint16(p),
		Target:   h + ".",
	}, nil
}

// formatA returns the A resource record for target
func (res *Resolver) formatA(dom string, target string) (*dns.A, error) {
	ttl := uint32(res.Config.TTL)

	h, _ := res.splitDomain(target)

	ip, err := net.ResolveIPAddr("ip4", h)
	if err != nil {
		return nil, err
	} else {

		a := ip.IP

		return &dns.A{
			Hdr: dns.RR_Header{
				Name:   dom,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl},
			A: a.To4(),
		}, nil
	}
}

// shuffleAnswers reorders answers for very basic load balancing
func shuffleAnswers(answers []dns.RR) []dns.RR {
	rand.Seed(time.Now().UTC().UnixNano())

	n := len(answers)
	for i := 0; i < n; i++ {
		r := i + rand.Intn(n-i)
		answers[r], answers[i] = answers[i], answers[r]
	}

	return answers
}

// HandleNonMesos makes non-mesos queries
func (res *Resolver) HandleNonMesos(w dns.ResponseWriter, r *dns.Msg) {
	var err error
	var m *dns.Msg

	proto := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		proto = "tcp"
	}

	for i := 0; i < len(res.Config.Resolvers); i++ {
		nameserver := res.Config.Resolvers[i] + ":53"
		m, err = res.resolveOut(r, nameserver, proto, recurseCnt)
		if err == nil {
			break
		}
	}

	if err != nil {
		logging.Error.Println(r.Question[0].Name)
		logging.Error.Println(err)
	}

	// resolveOut returns nil Msg sometimes cause of perf
	if m == nil {
		m = new(dns.Msg)
		m.SetReply(r)
		m.SetRcode(r, 2)
		err = errors.New("nil msg")
	}

	// tracing info
	logging.CurLog.NonMesosRequests += 1

	if err != nil {
		logging.Error.Println(err)
		logging.CurLog.NonMesosFailed += 1
	} else {

		if len(m.Answer) == 0 {
			logging.CurLog.NonMesosNXDomain += 1
		} else {
			logging.CurLog.NonMesosSuccess += 1
		}
	}

	err = w.WriteMsg(m)
	if err != nil {
		logging.Error.Println(err)
	}
}

// HandleMesos is a resolver request handler that responds to a resource
// question with resource answer(s)
// it can handle {A, SRV, ANY}
func (res *Resolver) HandleMesos(w dns.ResponseWriter, r *dns.Msg) {
	var err error

	dom := cleanWild(r.Question[0].Name)
	qType := r.Question[0].Qtype

	m := new(dns.Msg)
	m.Authoritative = true
	m.RecursionAvailable = true
	m.SetReply(r)

	if qType == dns.TypeSRV {

		for i := 0; i < len(res.rs.SRVs[dom]); i++ {
			rr, err := res.formatSRV(r.Question[0].Name, res.rs.SRVs[dom][i])
			if err != nil {
				logging.Error.Println(err)
			} else {
				m.Answer = append(m.Answer, rr)
			}
		}

	} else if qType == dns.TypeA {

		for i := 0; i < len(res.rs.As[dom]); i++ {
			rr, err := res.formatA(dom, res.rs.As[dom][i])
			if err != nil {
				logging.Error.Println(err)
			} else {
				m.Answer = append(m.Answer, rr)
			}

		}

	} else if qType == dns.TypeANY {

		// refactor me
		for i := 0; i < len(res.rs.As[dom]); i++ {
			rr, err := res.formatA(r.Question[0].Name, res.rs.As[dom][i])
			if err != nil {
				logging.Error.Println(err)
			} else {
				m.Answer = append(m.Answer, rr)
			}
		}

		for i := 0; i < len(res.rs.SRVs[dom]); i++ {
			rr, err := res.formatSRV(dom, res.rs.SRVs[dom][i])
			if err != nil {
				logging.Error.Println(err)
			} else {
				m.Answer = append(m.Answer, rr)
			}
		}

	}

	// shuffle answers
	m.Answer = shuffleAnswers(m.Answer)

	// tracing info
	logging.CurLog.MesosRequests += 1

	if err != nil {
		logging.CurLog.MesosFailed += 1
	} else {
		if len(m.Answer) == 0 {
			logging.CurLog.MesosNXDomain += 1
			logging.Error.Println("total A rrs:\t" + strconv.Itoa(len(res.rs.As)))
			logging.Error.Println("failed looking for " + r.Question[0].String())
		} else {
			logging.CurLog.MesosSuccess += 1
		}
	}

	err = w.WriteMsg(m)
	if err != nil {
		logging.Error.Println(err)
	}
}

// Serve starts a dns server for net protocol
func (res *Resolver) Serve(net string) {

	server := &dns.Server{
		Addr:       ":" + strconv.Itoa(res.Config.Port),
		Net:        net,
		TsigSecret: nil}

	err := server.ListenAndServe()
	if err != nil {
		logging.Error.Printf("Failed to setup "+net+" server: %s\n", err.Error())
	}
}

// Resolver holds configuration information and the resource records
// refactor me
type Resolver struct {
	rs     records.RecordGenerator
	Config records.Config
}

// Reload triggers a new refresh from mesos master
func (res *Resolver) Reload() {
	t := records.RecordGenerator{}
	t.ParseState(res.Config)

	res.rs = t
}
