package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pointdns "github.com/copper/go-pointdns"
)

var email string
var apiKey string
var domain string
var host string
var interval int

func init() {
	const (
		defaultEmail    = ""
		defaultAPIKey   = ""
		defaultHost     = ""
		defaultDomain   = ""
		defaultInterval = 15
	)

	flag.StringVar(&email, "email", defaultEmail, "your pointhq email address")
	flag.StringVar(&apiKey, "apiKey", defaultAPIKey, "your pointhq api key")
	flag.StringVar(&domain, "domain", defaultDomain, "the domain name")
	flag.StringVar(&host, "host", defaultHost, "the host record to update")
	flag.IntVar(&interval, "interval", defaultInterval, "how often to check for changes")
}

func main() {
	p := fmt.Println
	flag.Parse()

	if email == "" {
		p("you must provide your pointhq email address")
		return
	}

	if apiKey == "" {
		p("you must provide your pointhq api key")
		return
	}

	if domain == "" {
		p("you must provide your pointhq domain name. e.g. mydomain.com")
		return
	}

	if host == "" {
		p("you must provide the host record you want to update. e.g. home")
		return
	}

	client := pointdns.NewClient(email, apiKey)
	hostname := fmt.Sprintf("%s.%s.", host, domain)

	ipchan := make(chan string, 1)
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	p("INF: Running pointupd for", hostname)
	p("INF: Interval set to", interval)

	go func() {
		sig := <-signals
		p("INF: Signal received", sig)
		close(ipchan)
	}()

	var record pointdns.Record

	go func() {
		checkIP(record.Data, ipchan)

		for t := range ticker.C {
			p("INF: Checking for ip changed from", record.Data, t)
			checkIP(record.Data, ipchan)
		}
	}()

	for ip := range ipchan {

		if record.Id > 0 {
			record.Data = ip
			saved, err := record.Save()
			if err != nil {
				p("ERR: Unable to update record for", hostname, err)
			}
			if saved {
				p("INF: Updated record for", hostname, ip)
			} else {
				p("ERR: Record not saved", hostname, ip)
			}
		} else {
			p("INF: Looking for dns record for", hostname)
			zones, _ := client.Zones()
			for _, zone := range zones {
				if zone.Name == domain {
					records, _ := zone.Records()
					for _, r := range records {
						if r.Name == hostname {
							record = r
						}
					}

					if record.Id == 0 {
						p("INF: Unable to find existing record for", hostname)
						newRecord := pointdns.Record{
							Name:       hostname,
							Data:       ip,
							RecordType: "A",
							Ttl:        600,
							ZoneId:     zone.Id,
						}

						created, err := client.CreateRecord(&newRecord)
						if err != nil {
							p("ERR: Unable to create new record", err)
						}

						if created {
							p("INF: Created a new record for", hostname, ip)
							record = newRecord
						} else {
							p("ERR: Unable to create new record for", hostname, ip)
						}
					} else if record.Data != ip {
						ipchan <- ip
					}
				} else {
					p("INF: Skipping domain", zone.Name)
				}
			}
		}
	}

	p("INF: Exiting...")
}

func checkIP(CurrentIP string, Channel chan string) (string, error) {
	p := fmt.Println

	ip, err := getIP()
	p("INF: Found ip address", ip)
	if err != nil {
		p("ERR: Unable to get current ip address", err)
	} else if ip != CurrentIP {
		p("INF: Record needs updating, queueing...")
		Channel <- ip
	}

	return ip, err
}

func getIP() (string, error) {
	res, err := http.Get("http://allyourips.herokuapp.com/")
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)
	return fmt.Sprintf("%s", data), nil
}
