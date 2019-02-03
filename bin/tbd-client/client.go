package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/childoftheuniverse/tbd-client"
	"github.com/childoftheuniverse/tlsconfig"
)

func main() {
	var ctx context.Context
	var tbc *tbdClient.TokenBucketClient
	var tlsConfig *tls.Config
	var timeout time.Duration
	var cancel context.CancelFunc
	var serverName string
	var rootCaPath string
	var certPath string
	var keyPath string
	var thisHost string
	var flags []string
	var err error
	var ok bool

	var family string
	var bucket string
	var amount int64

	if thisHost, err = os.Hostname(); err != nil {
		log.Printf("Warning: cannot determine host name: %s", err)
	}

	flag.StringVar(&serverName, "remote", fmt.Sprintf("%s:9008", thisHost),
		"Address (gRPC compatible) to contact for the token bucket service")
	flag.StringVar(&rootCaPath, "root-ca", "",
		"Path to the root certificate to verify the server against")
	flag.StringVar(&certPath, "cert", "",
		"Path to the client certificate. TLS disabled if unset.")
	flag.StringVar(&keyPath, "key", "",
		"Path to the client private key. TLS disabled if unset.")
	flag.DurationVar(&timeout, "timeout", 5*time.Second,
		"Amount of time before failing the server request")
	flag.Parse()

	flags = flag.Args()
	if len(flags) < 3 {
		log.Fatal("Usage: tbd-client family bucket amount")
	}
	family = flags[0]
	bucket = flags[1]
	if amount, err = strconv.ParseInt(flags[2], 10, 63); err != nil {
		log.Fatalf("Cannot parse amount %s as number: %s", flags[2], err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if rootCaPath != "" && certPath != "" && keyPath != "" {
		if tlsConfig, err = tlsconfig.TLSConfigWithRootCAAndCert(
			rootCaPath, certPath, keyPath); err != nil {
			log.Fatal("Error reading TLS credentials: ", err)
		}
	}

	tbc, err = tbdClient.NewTokenBucketClient(serverName, tlsConfig)
	if err != nil {
		log.Fatal("Error connecting to token bucket service: ", err)
	}

	if ok, err = tbc.TokenRequest(ctx, family, bucket, amount); err != nil {
		log.Printf("Error requesting tokens for (%s,%s): %s", family, bucket, err)
	}

	if ok {
		log.Print("Success")
	} else {
		log.Print("Failure")
	}
}
