// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/google/google-api-go-client/blob/master/examples/main.go

package main

import (
	"context"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tckz/gapi"
	_ "github.com/tckz/gapi/firebase"
	_ "github.com/tckz/gapi/pubsub"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Flags
var (
	serviceAccountJson = flag.String("service-account", "", "/path/to/service-account.json")
	clientID           = flag.String("clientid", "", "OAuth 2.0 Client ID.  If non-empty, overrides --clientid_file")
	clientIDFile       = flag.String("clientid-file", "clientid.dat",
		"Name of a file containing just the project's OAuth 2.0 Client ID from https://developers.google.com/console.")
	secret     = flag.String("secret", "", "OAuth 2.0 Client Secret.  If non-empty, overrides --secret_file")
	secretFile = flag.String("secret-file", "clientsecret.dat",
		"Name of a file containing just the project's OAuth 2.0 Client Secret from https://developers.google.com/console.")
	cacheToken  = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	debug       = flag.Bool("debug", false, "show HTTP traffic")
	showVersion = flag.Bool("version", false, "show Version")
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
	if *debug {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
			Transport: &gapi.LogTransport{http.DefaultTransport},
		})
	}
	client := func() *http.Client {
		if *serviceAccountJson == "" {
			config := &oauth2.Config{
				ClientID:     valueOrFileContents(*clientID, *clientIDFile),
				ClientSecret: valueOrFileContents(*secret, *secretFile),
				Endpoint:     google.Endpoint,
				Scopes:       []string{command.Scope},
			}

			return newOAuthClient(ctx, config)
		} else {
			bytes, err := os.ReadFile(*serviceAccountJson)
			if err != nil {
				log.Fatalf("*** Failed to ReadFile(%s): %v", *serviceAccountJson, err)
			}

			c, err := google.JWTConfigFromJSON(bytes, command.Scope)
			if err != nil {
				log.Fatalf("*** Failed to JWTConfigFromJSON(%s): %v", *serviceAccountJson, err)
			}
			return c.Client(ctx)
		}
	}()
	ec := command.Func(client, flag.Args()[1:])
	os.Exit(ec)
}

func osUserCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "linux", "freebsd":
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}
	log.Printf("TODO: osUserCacheDir on GOOS %q", runtime.GOOS)
	return "."
}

func tokenCacheFile(config *oauth2.Config) string {
	hash := fnv.New32a()
	hash.Write([]byte(config.ClientID))
	hash.Write([]byte(config.ClientSecret))
	hash.Write([]byte(strings.Join(config.Scopes, " ")))
	fn := fmt.Sprintf("gapi-tok%v", hash.Sum32())
	return filepath.Join(osUserCacheDir(), url.QueryEscape(fn))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	if !*cacheToken {
		return nil, errors.New("--cachetoken is false")
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := new(oauth2.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}

func newOAuthClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile(config)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = tokenFromWeb(ctx, config)
		saveToken(cacheFile, token)
	} else {
		log.Printf("Using cached token %#v from %q", token, cacheFile)
	}

	return config.Client(ctx, token)
}

func tokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		log.Printf("no code")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return token
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

func valueOrFileContents(value string, filename string) string {
	if value != "" {
		return value
	}
	slurp, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	return strings.TrimSpace(string(slurp))
}
