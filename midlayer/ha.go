package midlayer

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/j-keck/arping"

	consul "github.com/hashicorp/consul/api"
)

type ConsulClient struct {
	Client *consul.Client
	l      *log.Logger
}

func (cc *ConsulClient) GetSession(sessionName string) string {
	name := cc.GetAgentName()
	sessions, _, err := cc.Client.Session().List(nil)
	if err != nil {
		cc.l.Println(err)
	}
	for _, session := range sessions {
		if session.Name == sessionName && session.Node == name {
			return session.ID
		}
	}

	cc.l.Println("No leadership sessions found, creating...")

	sessionEntry := &consul.SessionEntry{Name: sessionName}
	session, _, err := cc.Client.Session().Create(sessionEntry, nil)
	if err != nil {
		cc.l.Println(err)
	}
	return session
}

func (cc *ConsulClient) GetKey(keyName string) (*consul.KVPair, error) {
	kv, _, err := cc.Client.KV().Get(keyName, nil)
	return kv, err
}

func (cc *ConsulClient) AquireSessionKey(key string, session string) (bool, error) {

	pair := &consul.KVPair{
		Key:     key,
		Value:   []byte(cc.GetAgentName()),
		Session: session,
	}

	acquired, _, err := cc.Client.KV().Acquire(pair, nil)

	return acquired, err
}

func (cc *ConsulClient) GetAgentName() string {
	agent, _ := cc.Client.Agent().Self()
	return agent["Config"]["NodeName"].(string)
}

func (cc *ConsulClient) ReleaseKey(key *consul.KVPair) (bool, error) {
	released, _, err := cc.Client.KV().Release(key, nil)
	return released, err
}

type LeaderElection struct {
	LeaderKey     string
	WatchWaitTime int
	StopElection  chan bool
	Client        *ConsulClient
	l             *log.Logger
}

func (le *LeaderElection) Shutdown(ctx context.Context) error {
	le.StopElection <- true
	return le.StepDown()
}

func (le *LeaderElection) StepDown() error {
	if le.IsLeader() {
		client := le.Client
		name := client.GetAgentName()
		session := le.GetSession(le.LeaderKey)
		key := &consul.KVPair{Key: le.LeaderKey, Value: []byte(name), Session: session}
		released, err := client.ReleaseKey(key)
		if !released || err != nil {
			return err
		} else {
			le.l.Println("Released leadership")
		}
	}
	return nil
}

func (le *LeaderElection) IsLeader() bool {
	client := le.Client
	name := client.GetAgentName()
	session := le.GetSession(le.LeaderKey)
	kv, err := client.GetKey(le.LeaderKey)
	if err != nil || kv == nil {
		if err != nil {
			le.l.Println(err)
		}
		le.l.Println("Leadership key is missing")
		return false
	}

	return name == string(kv.Value) && session == kv.Session
}

func (le *LeaderElection) GetSession(sessionName string) string {
	client := le.Client
	session := client.GetSession(sessionName)
	return session
}

func (le *LeaderElection) ElectLeader(wakeme chan bool) {
	client := le.Client
	name := client.GetAgentName()
	stop := false
	imleader := false
	cLeader := ""
	cSession := ""
	for !stop {
		select {
		case <-le.StopElection:
			stop = true
			le.l.Println("Stopping election")
		default:
			if !le.IsLeader() {
				if imleader {
					// I've lost leader ship - bail!!!
					// Send myself a SIGINT so that the clean-up handlers do their things.
					p, e := os.FindProcess(os.Getpid())
					if e == nil {
						e = p.Signal(os.Interrupt)
						if e != nil {
							le.l.Printf("NO LONGER LEADER, BUT I THINK I AM.  I FAILED TO SIGNAL MYSELF.  DIE!! %v\n", e)
							os.Exit(1)

						}
					} else {
						le.l.Printf("NO LONGER LEADER, BUT I THINK I AM.  I CAN NOT FIND MYSELF.  DIE!! %v\n", e)
						os.Exit(1)
					}
					return
				}
				session := le.GetSession(le.LeaderKey)
				acquired, err := client.AquireSessionKey(le.LeaderKey, session)
				if acquired {
					le.l.Printf("%s is now the leader\n", name)
					if !imleader {
						wakeme <- true
					}
					imleader = true
				}
				if err != nil {
					le.l.Println(err)
				}
			}

			kv, err := client.GetKey(le.LeaderKey)
			if err != nil {
				le.l.Println(err)
			} else {

				if kv != nil && kv.Session != "" {
					if cLeader != string(kv.Value) || cSession != string(kv.Session) {
						le.l.Printf("Current leader: %s\n", string(kv.Value))
						le.l.Printf("Leader Session: %s\n", string(kv.Session))
						cLeader = string(kv.Value)
						cSession = string(kv.Session)
					}
				}
			}

			time.Sleep(time.Duration(le.WatchWaitTime) * time.Second)
		}
	}
}

func BecomeLeader(l *log.Logger) *LeaderElection {
	consulclient, _ := consul.NewClient(consul.DefaultConfig())
	le := &LeaderElection{
		StopElection:  make(chan bool),                           // The channel for stopping the election
		LeaderKey:     "service/drp/leader",                      // The leadership key to create/acquire
		WatchWaitTime: 1,                                         // Time in seconds to check for leadership
		Client:        &ConsulClient{l: l, Client: consulclient}, // The injected Consul api client
		l:             l,
	}

	wakeme := make(chan bool)
	go le.ElectLeader(wakeme)
	<-wakeme
	return le
}

func runCmd(command ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return stdout.Bytes(), stderr.Bytes(), err
}

func AddIP(addr, iface string) error {
	ip, _, err := net.ParseCIDR(addr)
	if err != nil {
		return err
	}
	var cmd []string
	switch runtime.GOOS {
	case "darwin":
		cmd = []string{"ifconfig", iface, "alias", addr}
	case "linux":
		cmd = []string{"ip", "address", "add", addr, "dev", iface}
	default:
		return fmt.Errorf("Unsupported platform: %s", runtime.GOOS)
	}
	if _, _, err := runCmd(cmd...); err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		if err := arping.GratuitousArpOverIfaceByName(ip, iface); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 50)
	}
	return nil
}

func RemoveIP(addr, iface string) error {
	var cmd []string
	switch runtime.GOOS {
	case "darwin":
		cmd = []string{"ifconfig", iface, "-alias", addr}
	case "linux":
		cmd = []string{"ip", "address", "del", addr, "dev", iface}
	default:
		return fmt.Errorf("Unsupported platform: %s", runtime.GOOS)
	}
	_, _, err := runCmd(cmd...)
	return err

}
