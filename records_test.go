package main

import (
	"log"
	"net"
	"os"
	"path"
	"testing"
)

var data = []byte(`{"entries":[{"name":"boop","ip":"127.0.0.1"}]}`)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(path.Join(dir, "records.json"), data, 0600)
	if err != nil {
		t.Fatal(err)
	}

	records := &Records{}
	err = records.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	ip := records.RetrieveIP("boop")
	expected := net.IPv4(127, 0, 0, 1)
	if !ip.Equal(expected) {
		t.Fatalf("expected %q got %q\n", expected, ip)
	}
}

func TesDelete(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(path.Join(dir, "records.json"), data, 0600)
	if err != nil {
		t.Fatal(err)
	}

	records := &Records{}
	err = records.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(records.Entries) != 1 {
		log.Fatalf("expected 1 entry got: %d\n", len(records.Entries))
	}
	records.Delete("boop")
	if len(records.Entries) != 0 {
		t.Fatalf("expected 0 entries have: %dn", len(records.Entries))
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	records := &Records{}

	records.Entries = append(records.Entries, Record{
		IP:   net.IPv4(127, 0, 0, 1),
		Name: "boop",
	})

	err := records.Save(dir)
	if err != nil {
		t.Fatal(err)
	}
}
