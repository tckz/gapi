package firebase

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/tckz/gapi"

	firebasedynamiclinks "google.golang.org/api/firebasedynamiclinks/v1"
)

var commandName = "firebase-dynamiclinks"

func init() {
	gapi.RegisterCommand(commandName, firebasedynamiclinks.FirebaseScope, dynamicLinksMain)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s {stat} [sub-command args]\n", commandName)
}

var subCommands = map[string]func(*http.Client, []string) int{
	"stat": getLinkStats,
}

func dynamicLinksMain(client *http.Client, argv []string) int {
	if len(argv) == 0 {
		usage()
		return 2
	}

	sub := argv[0]
	if f, ok := subCommands[sub]; !ok {
		usage()
		return 2
	} else {
		return f(client, argv[1:])
	}
}

type statResult struct {
	url   string
	stats *firebasedynamiclinks.DynamicLinkStats
}

func getLinkStats(client *http.Client, argv []string) int {

	f := flag.NewFlagSet("stat", flag.ExitOnError)
	duration := f.Int("duration", 10, "Duration days")
	file := f.String("file", "", "/path/to/urls.txt")
	parallelism := f.Int("parallelism", 5, "Number of simultaneous requests")
	f.Parse(argv)

	if f.NArg() == 0 && *file == "" {
		fmt.Fprintf(os.Stderr,
			"Usage: %s https://xxx.app.goo.gl/xxxxx\n", "stat")
		return 1
	}

	svc, err := firebasedynamiclinks.New(client)
	if err != nil {
		log.Fatalf("*** Unable to create firebasedynamiclinks service: %v", err)
	}

	// My network can't reach ipv6 dest.
	svc.BasePath = "https://firebasedynamiclinks.googleapis.com/"

	wg := &sync.WaitGroup{}
	chReq := make(chan string, *parallelism)
	chResult := make(chan *statResult, *parallelism)
	for i := 0; i < *parallelism; i++ {
		wg.Add(1)
		go func() {
			for url := range chReq {
				stats, err := svc.V1.GetLinkStats(url).
					DurationDays(int64(*duration)).
					Do()
				if err != nil {
					log.Fatalf("*** Failed to GetLinkStats: %v", err)
				}

				chResult <- &statResult{url: url, stats: stats}
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(chResult)
	}()

	if *file == "" {
		for _, url := range f.Args() {
			chReq <- url
		}
	} else {
		func() {
			fp, err := os.Open(*file)
			if err != nil {
				log.Fatalf("*** Failed to Open(%s): %v", *file, err)
			}
			defer fp.Close()
			scanner := bufio.NewScanner(fp)
			for scanner.Scan() {
				url := strings.TrimSpace(scanner.Text())
				if url != "" {
					chReq <- url
				}
			}
			if err := scanner.Err(); err != nil {
				log.Fatalf("*** Failed to Scan(%s): %v", *file, err)
			}
		}()
	}
	close(chReq)

	for statResult := range chResult {
		json, err := json.Marshal(map[string]interface{}{
			"url":    statResult.url,
			"events": statResult.stats.LinkEventStats,
		})
		if err != nil {
			log.Fatalf("*** Failed to Marshal: %v", err)
		}
		fmt.Printf("%s", json)
	}

	return 0
}
