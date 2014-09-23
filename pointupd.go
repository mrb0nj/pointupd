package main

import (
	"flag"
	"fmt"
	pointdns "github.com/copper/go-pointdns"
	"io/ioutil"
	"net/http"
	"time"
)

var email string
var apiKey string
var zone string
var record string
var savedIp string

var Record pointdns.Record
var Zone pointdns.Zone

func init() {
	const (
		defaultEmail  = ""
		defaultApiKey = ""
		defaultRecord = ""
		defaultZone   = ""
	)

	flag.StringVar(&email, "email", defaultEmail, "your registered pointhq email or username")
	flag.StringVar(&apiKey, "apiKey", defaultApiKey, "your pointhq api key")
	flag.StringVar(&zone, "zone", defaultZone, "the zone that 'record' belongs to")
	flag.StringVar(&record, "record", defaultRecord, "the record to update")
}

func main() {
	flag.Parse()

	ipchange = make(chan string)

	if email == "" || apiKey == "" || zone == "" || record == "" {
		fmt.Println("Invalid arguments")
		return
	}

	client := pointdns.NewClient(email, apiKey)
	hostname := fmt.Sprintf("%s.%s.", record, zone)
	ticker := time.NewTicker(15 * time.Minute)

	go func() {
		for t := range ticker.C {
			currentIp, err := getIp()
			if err != nil {
				fmt.Println(err)
			} else if currentIp != savedIp {
				ipchange <- currentIp
				fmt.Println("Updating DNS to match new IP:", currentIp)
			}
		}
	}()

	for newIp := range ipchange {

		if Zone == nil || Record == nil {
			zones, err := client.Zones()
			if err != nil {
				fmt.Println(err)
				return
			}

			for _, z := range zones {
				if z.Name == zone {
					Zone = z
					records, _ := z.Records()
					for _, r := range records {
						if r.Name == hostname {
							Record = r
						}
					}
				}
			}
		}

		if Record.Id == 0 {
			newRecord := pointdns.Record{
				Name:       hostname,
				Data:       newIp,
				RecordType: "A",
				Ttl:        600,
				ZoneId:     Zone.Id,
			}
			savedRecord, err := client.CreateRecord(&newRecord)
			if err != nil {
				fmt.Println(err)
			}
			if savedRecord {
				fmt.Println("Created a new Record:", hostname)
				Record = newRecord
			}
		} else {
			Record.Data = newIp
			Record.Save()
		}

		fmt.Println("Saved DNS record for IP:", newIp)
	}
}

func getIp() (string, error) {
	res, err := http.Get("http://icanhazip.com")
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	data, _ := ioutil.ReadAll(res.Body)
	ip := fmt.Sprintf("%s", data)

	return ip, nil
}
