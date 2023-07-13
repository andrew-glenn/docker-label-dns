package dockerSocket

import (
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/andrew-glenn/docker-compose-dns/types"
	dockerApi "github.com/fsouza/go-dockerclient"
)

func Start(dockerHost string, dnsEntries map[string]types.Records) {
	docker, err := dockerApi.NewClient(dockerHost)

	if err != nil {
		log.Fatal(err)
	}

	spy := &Spy{
		docker: docker,
		dns:    dnsEntries,
	}

	spy.Watch()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
}

type Spy struct {
	docker *dockerApi.Client
	dns    map[string]types.Records
}

func (s *Spy) Watch() {

	s.registerRunningContainers()

	events := make(chan *dockerApi.APIEvents)
	s.docker.AddEventListener(events)

	go s.readEventStream(events)
}

func (s *Spy) registerRunningContainers() {
	containers, err := s.docker.ListContainers(dockerApi.ListContainersOptions{})
	if err != nil {
		log.Fatalf("Unable to register running containers: %v", err)
	}
	for _, c := range containers {
		log.Printf("Inspecting container %s", c.ID)
		container, err := s.docker.InspectContainer(c.ID)
		if err != nil {
			log.Fatalf("Unable to register running containers: %v", err)
		}
		s.registerRecordsForContainer(container)
	}
}

func (s *Spy) registerRecordsForContainer(container *dockerApi.Container) {
	for _, fqdn := range s.relevantContainerLabels(container) {
		log.Printf("Registering: %s with IP: %s", fqdn, container.NetworkSettings.IPAddress)
		ip, _, err := net.ParseCIDR(container.NetworkSettings.IPAddress + "/32")
		if err != nil {
			log.Fatalf("unable to parse IP address: %v", err)
		}
		s.dns[fqdn+"."] = types.Records{
			A: []net.IP{ip},
		}
	}
}

func (s *Spy) relevantContainerLabels(container *dockerApi.Container) []string {
	var l []string
	log.Print(container.Config.Labels)
	for lk, lv := range container.Config.Labels {
		if lk == "dns-entries" {
			records := strings.Split(lv, ",")
			for _, r := range records {
				l = append(l, r)
			}
		}
	}
	return l
}

func (s *Spy) deregisterContainerRecords(container *dockerApi.Container) {
	for _, fqdn := range s.relevantContainerLabels(container) {
		log.Printf("Removing FQDN: %s", fqdn)
		delete(s.dns, fqdn+".")
	}
}

func (s *Spy) readEventStream(events chan *dockerApi.APIEvents) {
	for msg := range events {
		s.mutateContainerInCache(msg.ID, msg.Status)
	}
}

func (s *Spy) mutateContainerInCache(id string, status string) {

	container, err := s.docker.InspectContainer(id)
	if err != nil {
		log.Printf("Unable to inspect container %s, skipping", id)
		return
	}

	var running = regexp.MustCompile("start|^Up.*$")
	var stopping = regexp.MustCompile("die")

	switch {
	case running.MatchString(status):
		log.Printf("Registered container starting: %s", id)
		s.registerRecordsForContainer(container)
	case stopping.MatchString(status):
		log.Printf("Registered container dying: %s", id)
		s.deregisterContainerRecords(container)
	}
}
