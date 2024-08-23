package main

import (
	"crypto/tls"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/miekg/dns"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
	"tailscale.com/types/nettype"
)

//go:embed templates
var templateFS embed.FS

func httpLog(r *http.Request) {
	n := time.Now()
	fmt.Printf("%s (%s) [%s] \"%s %s\" %03d\n",
		r.RemoteAddr,
		n.Format(time.RFC822Z),
		r.Method,
		r.URL.Path,
		r.Proto,
		r.ContentLength,
	)
}

func main() {
	sName := flag.String("name", "", "server name")
	dataDir := flag.String("data", "/var/lib/tsns", "path to store the records")
	flag.Parse()

	var err error
	records := &Records{}
	records.templ, err = template.New("prod").ParseFS(templateFS, "templates/*")

	_, err = os.Stat(*dataDir)
	if os.IsNotExist(err) {
		log.Fatalf("%s does not exist", *dataDir)
	} else {
		err = records.Load(*dataDir)
		// If it's just a missing file, continue along
		if !os.IsNotExist(err) {
			log.Fatalln(err)
		}
	}

	tsServer := &tsnet.Server{
		Hostname: *sName,
	}
	tsLocalClient := &tailscale.LocalClient{}
	tsLocalClient, err = tsServer.LocalClient()
	if err != nil {
		log.Fatal("can't get ts local client: ", err)
	}

	httpListen, err := tsServer.Listen("tcp", ":443")
	if err != nil {
		log.Fatal("can't listen: ", err)
	}

	dnsListen, err := tsServer.Listen("udp", ":53")
	if err != nil {
		log.Fatal("can't listen: ", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		httpLog(r)
		resp := &Response{}
		resp.Entries = records.Entries
		if err := records.templ.ExecuteTemplate(w, "index", resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("POST /records", func(w http.ResponseWriter, r *http.Request) {
		httpLog(r)
		resp := &Response{}
		ipStr := r.FormValue("ip")
		ip := net.ParseIP(ipStr)
		if ip != nil {
			rec := Record{
				Name: r.FormValue("name"),
				IP:   ip,
			}

			records.Entries = append(records.Entries, rec)
			records.Save(*dataDir)
		} else {
			resp.Error = fmt.Errorf("invalid IP: %q", ipStr)
		}

		resp.Entries = records.Entries

		if err := records.templ.ExecuteTemplate(w, "table", resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("DELETE /records/{name}", func(w http.ResponseWriter, r *http.Request) {
		httpLog(r)
		resp := Response{}
		records.Delete(r.PathValue("name"))
		records.Save(*dataDir)

		resp.Entries = records.Entries
		if err := records.templ.ExecuteTemplate(w, "table", resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	httpServer := &http.Server{
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: tsLocalClient.GetCertificate,
		},
	}

	go httpServer.ServeTLS(httpListen, "", "")

	// Gross: since we can't do a net.PacketConn on our tailnet.. we do this hack..
	// https://github.com/tailscale/tailscale/issues/5871
	for {
		conn, err := dnsListen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		// Gross Gross
		server := &dns.Server{
			PacketConn: conn.(nettype.ConnPacketConn),
			Net:        "udp",
		}

		// Groooosss
		dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
			msg := &dns.Msg{}
			msg.SetReply(r)
			for _, q := range r.Question {
				ip := records.RetrieveIP(q.Name)
				if ip != nil {
					msg.Authoritative = true
					msg.Answer = append(msg.Answer, &dns.A{
						A: *ip,
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    60,
						},
					})
				}
			}

			w.WriteMsg(msg)
		})

		/// GRRRRR OOOOO SSEEEEEEaaaaaaaa
		go func() {
			defer server.Shutdown()

			err = server.ActivateAndServe()
			if err != nil {
				log.Fatal(err)
			}
		}()

	}
}
