package main

import (
	dnsLib "utils/ddnsupdate/lib"
)

func main() {
	profile := dnsLib.New()

	// Listen for IP changes
	go profile.StartListener()

	for {
		// Block until we have a new IP
		profile.WaitForChanges()

		// Handle the change someway
		go profile.UpdateRecord()
	}
}
