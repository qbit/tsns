package main

import (
	"encoding/json"
	"html/template"
	"net"
	"os"
	"path"
	"slices"
	"sync"

	"github.com/miekg/dns"
)

type Record struct {
	Name string `json:"name"`
	IP   net.IP `json:"ip"`
}

type Records struct {
	Entries []Record `json:"entries"`
	mu      sync.RWMutex
	templ   *template.Template
}

func (r *Records) Delete(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	newEntries := slices.DeleteFunc(r.Entries, func(n Record) bool {
		return n.Name == name
	})
	r.Entries = newEntries
}

func (r *Records) RetrieveIP(name string) *net.IP {
	for _, entry := range r.Entries {
		if dns.Fqdn(entry.Name) == dns.Fqdn(name) {
			return &entry.IP
		}
	}
	return nil
}

func (r *Records) Load(dir string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, err := os.ReadFile(path.Join(dir, "records.json"))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, r)
}

func (r *Records) Save(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(dir, "records.json"), b, 0600)
}
