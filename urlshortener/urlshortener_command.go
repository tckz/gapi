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

// based on https://github.com/google/google-api-go-client/blob/master/examples/urlshortener.go

package urlshortener

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tckz/gapi"

	google_urlshortener "google.golang.org/api/urlshortener/v1"
)

func init() {
	gapi.RegisterCommand("urlshortener", google_urlshortener.UrlshortenerScope, urlShortenerMain)
}

type subCommandFunc func(*google_urlshortener.Service, []string) int

var subCommands = map[string]subCommandFunc{
	"get":    getUrl,
	"insert": insertUrl,
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: urlshortener {insert|get}\n")
}

func urlShortenerMain(client *http.Client, argv []string) int {
	if len(argv) == 0 {
		usage()
		return 2
	}

	svc, err := google_urlshortener.New(client)
	if err != nil {
		log.Fatalf("Unable to create UrlShortener service: %v", err)
	}

	sub := argv[0]
	if f, ok := subCommands[sub]; !ok {
		usage()
		return 2
	} else {
		return f(svc, argv[1:])
	}
}

// getUrl short -> long
func getUrl(svc *google_urlshortener.Service, argv []string) int {

	if len(argv) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: urlshortener get http://goo.gl/xxxxx     (to look up details)\n")
		return 1
	}

	urlstr := argv[0]

	// Firebase: https://xxxx.app.goo.gl/
	url, err := svc.Url.Get(urlstr).Do()
	if err != nil {
		log.Fatalf("URL Get: %v", err)
	}
	json, err := url.MarshalJSON()
	if err != nil {
		log.Fatalf("*** Failed to MarshalJSON(%s): %v", urlstr, err)
	}
	fmt.Printf("%s\n", json)
	return 0
}

// insertUrl long -> short
func insertUrl(svc *google_urlshortener.Service, argv []string) int {
	if len(argv) == 0 {
		fmt.Fprintf(os.Stderr, "Usage:urlshortener insert http://example.com/long (to shorten)\n")
		return 1
	}

	urlstr := argv[0]
	url, err := svc.Url.Insert(&google_urlshortener.Url{
		Kind:    "urlshortener#url", // Not really needed
		LongUrl: urlstr,
	}).Do()
	if err != nil {
		log.Fatalf("URL Insert: %v", err)
	}
	fmt.Printf("Shortened %s => %s\n", urlstr, url.Id)
	return 0
}
