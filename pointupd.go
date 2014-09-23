package main

import (
	"flag"
	"fmt"
	pointdns "github.com/copper/go-pointdns"
	"io/ioutil"
	"net/http"
)

var email string
var apiKey string
var zone string
var record string

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
	fmt.Println("Starting...")
	flag.Parse()

	if email == "" || apiKey == "" || zone == "" || record == "" {
		fmt.Println("Invalid arguments")
		return
	}

	myIp, err := getIp()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(myIp)

	client := pointdns.NewClient(email, apiKey)

	zones, err := client.Zones()
	if err != nil {
		fmt.Println(err)
		return
	}

	hostname := fmt.Sprintf("%s.%s.", record, zone)

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

	fmt.Println("Record: ", Record.Id > 0)
	if Record.Id == 0 {
		newRecord := pointdns.Record{
			Name:       hostname,
			Data:       myIp,
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
		Record.Data = myIp
		Record.Save()
	}

	fmt.Println(Record)
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
