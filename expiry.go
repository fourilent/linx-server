package main

import (
	"time"

	"github.com/dustin/go-humanize"
)

var defaultExpiryList = []uint64{
	60,
	300,
	3600,
	7200,
	14400,
	28800,
	43200,
	86400,
	604800,
	2419200,
	31536000,
}

type ExpirationTime struct {
	Seconds uint64
	Human   string
}

// Return a list of expiration times and their humanized versions
func listExpirationTimes() []ExpirationTime {
	epoch := time.Now()
	actualExpiryInList := false
	var expiryList []ExpirationTime

	for _, expiryEntry := range defaultExpiryList {
		if Config.maxExpiry == 0 || expiryEntry <= Config.maxExpiry {
			if expiryEntry == Config.maxExpiry {
				actualExpiryInList = true
			}

			duration := time.Duration(expiryEntry) * time.Second
			expiryList = append(expiryList, ExpirationTime{
				Seconds: expiryEntry,
				Human:   humanize.RelTime(epoch, epoch.Add(duration), "", ""),
			})
		}
	}

	if Config.maxExpiry == 0 {
		expiryList = append(expiryList, ExpirationTime{
			0,
			"never",
		})
	} else if !actualExpiryInList {
		duration := time.Duration(Config.maxExpiry) * time.Second
		expiryList = append(expiryList, ExpirationTime{
			Config.maxExpiry,
			humanize.RelTime(epoch, epoch.Add(duration), "", ""),
		})
	}

	return expiryList
}
