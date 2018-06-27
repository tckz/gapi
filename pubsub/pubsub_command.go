// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// based on https://github.com/google/google-api-go-client/blob/master/examples/pubsub.go

package pubsub

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tckz/gapi"

	"strconv"

	"sync"

	"flag"

	gps "google.golang.org/api/pubsub/v1beta2"
)

func init() {
	gapi.RegisterCommand("pubsub", gps.PubsubScope, pubsubMain)
}

type subCommandFunc func(*gps.Service, []string) int

var subCommands = map[string]subCommandFunc{
	"pull-message": pullMessage,
	"send-dummy":   sendDummy,
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: pubsub {pull-message}\n")
}

func pubsubMain(client *http.Client, argv []string) int {
	if len(argv) == 0 {
		usage()
		return 2
	}

	svc, err := gps.New(client)
	if err != nil {
		log.Fatalf("Unable to create Pubsub service: %v", err)
	}

	sub := argv[0]
	if f, ok := subCommands[sub]; !ok {
		usage()
		return 2
	} else {
		return f(svc, argv[1:])
	}
}

func sendDummy(svc *gps.Service, argv []string) int {

	fs := flag.NewFlagSet("send-dummy", flag.ExitOnError)
	parallel := fs.Int("parallel", 1, "Parallelism of publishing process")
	count := fs.Int("count", 100, "Number of publishing")

	if err := fs.Parse(argv); err != nil {
		return 2
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Usage: pubsub send-dummy topic\n")
		return 2
	}
	topicName := fs.Arg(0)

	ch := make(chan int, *parallel)
	wg := &sync.WaitGroup{}
	for i := 0; i < *parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range ch {
				req := &gps.PublishRequest{
					Messages: []*gps.PubsubMessage{
						{
							Data: base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(n))),
						},
					},
				}
				if r, err := svc.Projects.Topics.Publish(topicName, req).Do(); err != nil {
					log.Fatalf("*** Failed to Publish: %v", err)
				} else {
					fmt.Fprintf(os.Stderr, "%s\n", r.MessageIds)
				}
			}
		}()
	}

	for i := 0; i < *count; i++ {
		ch <- i
	}
	close(ch)

	wg.Wait()

	return 0
}

func pullMessage(svc *gps.Service, argv []string) int {

	if len(argv) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: pubsub pull-message subscripiton\n")
		return 2
	}

	subName := argv[0]
	pullRequest := &gps.PullRequest{
		ReturnImmediately: false,
		MaxMessages:       1,
	}
	for {
		pullResponse, err := svc.Projects.Subscriptions.Pull(subName, pullRequest).Do()
		if err != nil {
			log.Fatalf("pullMessages Pull().Do() failed: %v", err)
		}
		for _, receivedMessage := range pullResponse.ReceivedMessages {
			data, err := base64.StdEncoding.DecodeString(receivedMessage.Message.Data)
			if err != nil {
				log.Fatalf("pullMessages DecodeString() failed: %v", err)
			}
			fmt.Printf("%s\n", data)
			ackRequest := &gps.AcknowledgeRequest{
				AckIds: []string{receivedMessage.AckId},
			}
			if _, err = svc.Projects.Subscriptions.Acknowledge(subName, ackRequest).Do(); err != nil {
				log.Printf("pullMessages Acknowledge().Do() failed: %v", err)
			}
		}
	}

	return 0
}
