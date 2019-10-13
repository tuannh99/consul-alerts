package main

import (
	log "github.com/AcalephStorage/consul-alerts/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	consulapi "github.com/AcalephStorage/consul-alerts/Godeps/_workspace/src/github.com/hashicorp/consul/api"
	"time"
	"sync"
)

const LockKey = "consul-alerts/leader"

type LeaderElection struct {
	lock           *consulapi.Lock
	cleanupChannel chan struct{}
	stopChannel    chan struct{}
	leader         bool
	mutexConfig    *sync.RWMutex
}

func (l *LeaderElection) start() {
	clean := false
	for !clean {
		select {
		case <-l.cleanupChannel:
			clean = true
		default:
			log.Infoln("Running for leader election...")
			intChan, _ := l.lock.Lock(l.stopChannel)
			if intChan != nil {
				log.Infoln("Now acting as leader.")
				l.mutexConfig.Lock()
				l.leader = true
				l.mutexConfig.Unlock()
				<-intChan
				l.mutexConfig.Lock()
				l.leader = false
				l.mutexConfig.Unlock()
				log.Infoln("Lost leadership.")
				l.lock.Unlock()
				l.lock.Destroy()
			} else {
				time.Sleep(10000 * time.Millisecond)
			}
		}
	}
}

func (l *LeaderElection) stop() {
	log.Infoln("cleaning up")
	l.cleanupChannel <- struct{}{}
	l.stopChannel <- struct{}{}
	l.lock.Unlock()
	l.lock.Destroy()
	l.mutexConfig.Lock()
	l.leader = false
	l.mutexConfig.Unlock()
	log.Infoln("cleanup done")
}

func startLeaderElection(addr, dc, acl string, mutexConfig *sync.RWMutex) *LeaderElection {
	config := consulapi.DefaultConfig()
	config.Address = addr
	config.Datacenter = dc
	config.Token = acl
	client, _ := consulapi.NewClient(config)
	lock, _ := client.LockKey(LockKey)

	leader := &LeaderElection{
		lock:           lock,
		cleanupChannel: make(chan struct{}, 1),
		stopChannel:    make(chan struct{}, 1),
		mutexConfig:    mutexConfig,
	}

	go leader.start()

	return leader
}

func hasLeader() bool {
	return consulClient.CheckKeyExists(LockKey)
}
