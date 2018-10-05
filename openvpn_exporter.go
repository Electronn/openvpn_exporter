package main

import (
	"flag"
	"net/http"
	"os"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"time"
	"regexp"
	"io/ioutil"
	"strings"
	"encoding/json"
	"strconv"
)

var (
	ListenAddr = flag.String("listenaddr", ":9509", "ovpnserver_exporter listen address")
	MetricsPath = flag.String("metricspath", "/metrics", "URL path for surfacing collected metrics")
	ovpnlog = flag.String("ovpn.log", "/var/log/status.log", "Absolute path for OpenVPN server log")
)

var (
	ovpnclientscount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ovpn_clients_count",
		Help: "Current OpenVPN logged in users",
	})
	ovpnmaxbcastmcastqueue = prometheus.NewGauge(prometheus.GaugeOpts{
                Name: "ovpn_maxmacatbcastqueue",
                Help: "Current Max Broadcast/Multicast queue",
        })
	ovpnremote = prometheus.NewGaugeVec(prometheus.GaugeOpts{
                Name: "ovpn_remote",
                Help: "OpenVPN users statistics",
        },
	[]string{"client","ip"},
	)
	ovpnbytesr = prometheus.NewGaugeVec(prometheus.GaugeOpts{
                Name: "ovpn_bytesr",
                Help: "OpenVPN user Bytes Received",
        },
        []string{"client", "number"},
        )
        ovpnbytess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
                Name: "ovpn_bytess",
                Help: "OpenVPN user Bytes Sent",
        },
        []string{"client", "number"},
        )
        ovpnrouting = prometheus.NewGaugeVec(prometheus.GaugeOpts{
                Name: "ovpn_routing",
                Help: "OpenVPN Routing Table",
        },
        []string{"record_number", "client", "local_ip", "remote_ip"},
        )



)

const (
	time_layout = "Mon Jan _2 15:04:05 2006"
)

func init() {
	prometheus.MustRegister(ovpnclientscount)
        prometheus.MustRegister(ovpnmaxbcastmcastqueue)
	prometheus.MustRegister(ovpnremote)
	prometheus.MustRegister(ovpnbytesr)
        prometheus.MustRegister(ovpnbytess)
        prometheus.MustRegister(ovpnrouting)
}

type Body struct {
	Updated string `json:"updated"`
	Clients []struct {
		Client        string `json:"client"`
		Remote        string `json:"remote"`
		BytesReceived string `json:"bytes_received"`
		BytesSent     string `json:"bytes_sent"`
	} `json:"clients"`
	Routing []struct {
		LocalIP string `json:"local_ip"`
		Client  string `json:"client"`
		RealIP  string `json:"real_ip"`
	} `json:"routing"`
	MaxBcastMcastQueue string `json:"max_bcast_mcast_queue"`
}

func main() {
	flag.Parse()
        if *ovpnlog == "" { log.Fatal("OpenVPN status log absolute path must be set with '-ovpn.log' flag") }
        if _, err := os.Stat(*ovpnlog); os.IsNotExist(err) { log.Fatal("File: ",*ovpnlog," does not exists")}
	var landingPage = []byte(`<html><head><title>OpenVPN exporter exporter</title></head><body><h1>OpenVPN server stats exporter</h1><p><a href='` + *MetricsPath + `'>Metrics</a></p></body></html>`)
	go func() {
		for {
			byt := []byte(convert_ovpn_status(*ovpnlog))
			b := &Body{}
			json.Unmarshal(byt, b)
			strmcast, _ := strconv.ParseFloat(b.MaxBcastMcastQueue, 64)
			ovpnclientscount.Set(float64(len(b.Clients)))
			ovpnmaxbcastmcastqueue.Set(strmcast)
			for i, _ := range b.Clients {
				bytesr, _ := strconv.Atoi(b.Clients[i].BytesReceived)
				bytess, _ := strconv.Atoi(b.Clients[i].BytesSent)
				ovpnremote.WithLabelValues(b.Clients[i].Client, b.Clients[i].Remote).Set(float64(i+1))
				ovpnbytesr.WithLabelValues(b.Clients[i].Client, strconv.Itoa(i+1)).Set(float64(bytesr))
                                ovpnbytess.WithLabelValues(b.Clients[i].Client, strconv.Itoa(i+1)).Set(float64(bytess))
			}
                        for i, _ := range b.Routing {
                                ovpnrouting.WithLabelValues(strconv.Itoa(i+1), b.Routing[i].Client, b.Routing[i].LocalIP, b.Routing[i].RealIP)
			}
 
			time.Sleep(time.Duration(1000 * time.Millisecond))
		}
	}()
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write(landingPage) } )
        http.Handle(*MetricsPath, promhttp.Handler())
        log.Info("Listening on: ", *ListenAddr)
        log.Fatal(http.ListenAndServe(*ListenAddr, nil))

}

func time_convert(times string) time.Time {
        t2, _ := time.Parse(time_layout, times)
        return t2
}

func convert_ovpn_status(logfile string) string {
        b, _:= ioutil.ReadFile(logfile)
	var resultstring,resclient,resroute,resline string
        lines := strings.Split(string(b), "\n")
        for _, line := range lines { resline = resline + " " + line }
        reg := regexp.MustCompile("OpenVPN CLIENT LIST Updated,(.*?) Common Name,Real Address,Bytes Received,Bytes Sent,Connected Since (.*?) ROUTING TABLE Virtual Address,Common Name,Real Address,Last Ref (.*?) GLOBAL STATS Max bcast/mcast queue length,(.*?) END")
        regclients := regexp.MustCompile("(.*),(.*),(.*),(.*),")
        regroutes := regexp.MustCompile("(.*),(.*),(.*),")
        regipport := regexp.MustCompile("(.*):(.*)")
        s := regexp.MustCompile("((Mon|Tue|Thu|Wed|Fri|Sat|Sun) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \\d{1,2} \\d{2}:\\d{2}:\\d{2} \\d{4})").Split(reg.FindStringSubmatch(resline)[2], -1)
        s2 := regexp.MustCompile("((Mon|Tue|Thu|Wed|Fri|Sat|Sun) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \\d{1,2} \\d{2}:\\d{2}:\\d{2} \\d{4})").Split(reg.FindStringSubmatch(resline)[3], -1)
        resultstring = resultstring + "{" + "\"updated\": " + "\"" + reg.FindStringSubmatch(resline)[1] + "\"" + "," + "\"clients\": ["
        for i := 0; i < len(s) - 1; i++ {
                resclient = resclient + "{" + "\"client\": " + "\"" + regclients.FindStringSubmatch(s[i])[1] + "\"," + "\"remote\": " + "\"" + regipport.FindStringSubmatch(regclients.FindStringSubmatch(s[i])[2])[1] + "\"," + "\"bytes_received\": " + "\"" + regclients.FindStringSubmatch(s[i])[3] + "\"," + "\"bytes_sent\": " + "\"" + regclients.FindStringSubmatch(s[i])[4] + "\"}"
                if i == len(s) - 2 { resclient = resclient + "]," } else { resclient = resclient + "," }
        }
        resultstring = resultstring + resclient
        resultstring = resultstring + "\"routing\": ["
        for i := 0; i < len(s2) - 1; i++ {
                resroute = resroute + "{" + "\"local_ip\": " + "\"" + regroutes.FindStringSubmatch(s2[i])[1] + "\"," + "\"client\": " + "\"" + regroutes.FindStringSubmatch(s2[i])[2] + "\"," + "\"real_ip\" :" + "\"" + regipport.FindStringSubmatch(regroutes.FindStringSubmatch(s2[i])[3])[1] + "\"}"
                if i == len(s2) - 2 { resroute = resroute + "]," } else { resroute = resroute + "," }
        }
        resultstring = resultstring + resroute
        resultstring = resultstring + "\"max_bcast_mcast_queue\": " + "\"" + reg.FindStringSubmatch(resline)[4] + "\"}"
        return resultstring
}
