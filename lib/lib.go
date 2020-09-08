package lib

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const GET_IP_FROM_URL string = "https://api.ipify.org"

type dDNS_UpdateInfo struct {
	host     string
	domain   string
	password string
}

type dDNS_Profile struct {
	addr     string
	C        chan string
	interval time.Duration

	updateInfo dDNS_UpdateInfo
}

type DNSProfile interface {
	StartListener()
	UpdateRecord()
	WaitForChanges()
}

func parseFlags() (interval time.Duration, host, domain, password string) {
	var pwdFilepath string

	flag.DurationVar(&interval, "interval", time.Duration(time.Second*30), "Time to wait after each IP check. Must be positive.")
	flag.StringVar(&host, "host", "", "(required) The host of the domain to update.")
	flag.StringVar(&domain, "domain", "", "(required) The domain associate with the DNS account.")
	flag.StringVar(&pwdFilepath, "pwdFile", "", "(required) A file containing the DNS password to update the host record.")
	flag.Parse()

	if host == "" || domain == "" || pwdFilepath == "" {
		fmt.Fprintf(os.Stderr, "Please provide all required flags. Run with --help to get program usage.\n")
		os.Exit(1)
	}

	fstat, err := os.Stat(pwdFilepath)
	if os.IsNotExist(err) || !fstat.Mode().IsRegular() {
		fmt.Fprintf(os.Stderr, "Password file path is not correct, file does not exist or of incorrect type.\n")
		os.Exit(1)
	}

	pwd, err := ioutil.ReadFile(pwdFilepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password file: %s\n", err)
		os.Exit(1)
	}

	password = strings.TrimSuffix(string(pwd), "\n")
	return interval, host, domain, password
}

func New() DNSProfile {
	interval, host, domain, password := parseFlags()

	ddnsprofile := dDNS_Profile{}

	ddnsprofile.addr = ""
	ddnsprofile.C = make(chan string, 1)
	ddnsprofile.interval = interval

	ddnsprofile.updateInfo = dDNS_UpdateInfo{
		host:     host,
		domain:   domain,
		password: password,
	}

	return &ddnsprofile
}

func (profile *dDNS_Profile) StartListener() {
	for {
		ipaddr, err := fetchExternalIP()

		if err != nil {
			profile.addr = ""
			continue
		} else {
			if profile.addr != ipaddr {
				profile.addr = ipaddr

				profile.C <- profile.addr
			}
		}

		time.Sleep(profile.interval)
	}
}

func fetchExternalIP() (string, error) {
	var client = &http.Client{
		Timeout: time.Duration(time.Second * 1),
	}

	response, err := client.Get(GET_IP_FROM_URL)
	if err != nil {
		return "", errors.New("Could not get IP: No internet?")
	}

	bodyBuffer, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBuffer), nil
}

/**
 * Update the IP entry on the DynamicDNS
 *
 * @param  dDNS_Profile
 * @return void
 */
func (profile *dDNS_Profile) UpdateRecord() {
	updateUrl := fmt.Sprintf("https://dynamicdns.park-your-domain.com/update?host=%s&domain=%s&password=%s&ip=%s", profile.updateInfo.host, profile.updateInfo.domain, profile.updateInfo.password, profile.addr)

	response, err := http.Get(updateUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: ", err)
	}

	if response.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Could not update remote record. Response status: %s", response.Status)
	}

	fmt.Printf("Updated new IP: %s\n", profile.addr)
}


func (profile *dDNS_Profile) WaitForChanges() {
	<-profile.C
}