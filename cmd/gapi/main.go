package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
	"github.com/samber/lo"
	"github.com/tckz/gapi"
	_ "github.com/tckz/gapi/pubsub"
	"golang.org/x/oauth2/google"
)

var (
	serviceAccountJSON = flag.String("service-account", "", "/path/to/service-account.json")
	showVersion        = flag.Bool("version", false, "show Version")
)
var (
	version = "to be replaced"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> <sub-command> [sub command args]\n\nPossible Commands:\n\n", path.Base(os.Args[0]))
	for _, n := range gapi.ListCommandNames() {
		fmt.Fprintf(os.Stderr, "  * %s\n", n)
	}
	os.Exit(2)
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s\n", version)
		return
	}

	if flag.NArg() == 0 {
		usage()
	}

	name := flag.Arg(0)
	command, err := gapi.GetCommand(name)
	if err != nil {
		usage()
	}

	ctx := context.Background()
	client := func() *http.Client {
		if *serviceAccountJSON != "" {
			b, err := os.ReadFile(*serviceAccountJSON)
			if err != nil {
				log.Fatalf("*** Failed to ReadFile(%s): %v", *serviceAccountJSON, err)
			}

			c, err := google.JWTConfigFromJSON(b, command.Scope)
			if err != nil {
				log.Fatalf("*** Failed to JWTConfigFromJSON(%s): %v", *serviceAccountJSON, err)
			}
			return c.Client(ctx)
		}

		// use ADC
		creds := lo.Must(credentials.DetectDefault(&credentials.DetectOptions{
			Scopes: []string{command.Scope},
		}))
		cl, err := httptransport.NewClient(&httptransport.Options{Credentials: creds})
		if err != nil {
			log.Fatalf("*** httptransport.NewClient: %v", err)
		}
		return cl
	}()
	if ec := command.Func(ctx, client, flag.Args()[1:]); ec != 0 {
		os.Exit(ec)
	}
}
