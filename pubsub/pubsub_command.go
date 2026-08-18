package pubsub

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/tckz/gapi"
	"google.golang.org/api/option"
	"google.golang.org/api/pubsub/v1"
)

func init() {
	gapi.RegisterCommand("pubsub", pubsub.PubsubScope, pubsubMain)
}

type subCommandFunc func(context.Context, *pubsub.Service, []string) int

var subCommands = map[string]subCommandFunc{
	"pull-message": pullMessage,
	"send-dummy":   sendDummy,
}

func usage() {
	cmds := lo.Keys(subCommands)
	sort.Strings(cmds)
	fmt.Fprintf(os.Stderr, "Usage: pubsub {%s}\n", strings.Join(cmds, "|"))
}

func pubsubMain(ctx context.Context, client *http.Client, argv []string) int {
	if len(argv) == 0 {
		usage()
		return 2
	}

	svc, err := pubsub.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Pubsub service: %v", err)
	}

	sub := argv[0]
	if f, ok := subCommands[sub]; !ok {
		usage()
		return 2
	} else {
		return f(ctx, svc, argv[1:])
	}
}

func sendDummy(ctx context.Context, svc *pubsub.Service, argv []string) int {

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
	for range *parallel {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case n, ok := <-ch:
					if !ok {
						return
					}
					req := &pubsub.PublishRequest{
						Messages: []*pubsub.PubsubMessage{
							{
								Data: base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(n))),
							},
						},
					}
					if r, err := svc.Projects.Topics.Publish(topicName, req).Context(ctx).Do(); err != nil {
						log.Fatalf("*** Failed to Publish: %v", err)
					} else {
						fmt.Fprintf(os.Stderr, "%s\n", r.MessageIds)
					}
				}
			}
		})
	}

	for i := range *count {
		select {
		case <-ctx.Done():
			break
		default:
			ch <- i
		}
	}
	close(ch)

	wg.Wait()

	return 0
}

func pullMessage(ctx context.Context, svc *pubsub.Service, argv []string) int {

	if len(argv) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: pubsub pull-message subscription\n")
		return 2
	}

	subName := argv[0]
	pullRequest := &pubsub.PullRequest{
		ReturnImmediately: false,
		MaxMessages:       1,
	}
	for {
		pullResponse, err := svc.Projects.Subscriptions.Pull(subName, pullRequest).Context(ctx).Do()
		if err != nil {
			log.Fatalf("pullMessages Pull().Do() failed: %v", err)
		}
		for _, receivedMessage := range pullResponse.ReceivedMessages {
			data, err := base64.StdEncoding.DecodeString(receivedMessage.Message.Data)
			if err != nil {
				log.Fatalf("pullMessages DecodeString() failed: %v", err)
			}
			fmt.Printf("%s\n", data)
			ackRequest := &pubsub.AcknowledgeRequest{
				AckIds: []string{receivedMessage.AckId},
			}
			if _, err = svc.Projects.Subscriptions.Acknowledge(subName, ackRequest).Do(); err != nil {
				log.Printf("pullMessages Acknowledge().Do() failed: %v", err)
			}
		}
	}

	return 0
}
