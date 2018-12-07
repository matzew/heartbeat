/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/knative/pkg/cloudevents"
	"github.com/pkg/errors"
)

type Heartbeat struct {
	Sequence int    `json:"id"`
	Label    string `json:"label"`
}

var (
	sink      string
	label     string
	periodStr string

	httpClient = &http.Client{}
)

func init() {
	flag.StringVar(&sink, "sink", "", "the host url to heartbeat to")
	flag.StringVar(&label, "label", "", "a special label")
	flag.StringVar(&periodStr, "period", "100", "the number of milliseconds between heartbeats")
}

func main() {
	flag.Parse()
	var period time.Duration
	if p, err := strconv.Atoi(periodStr); err != nil {
		period = time.Duration(100) * time.Millisecond
	} else {
		period = time.Duration(p) * time.Millisecond
	}

	hb := &Heartbeat{
		Sequence: 0,
		Label:    label,
	}
	ticker := time.NewTicker(period)
	for {
		hb.Sequence++

		message := []byte(hb.Label)
		go func() {
			if err := postMessage(message); err != nil {
				log.Printf("sending event to channel failed: %v", err)
			}
		}()

		<-ticker.C
	}
}

func postMessage(message []byte) error {
	ctx := cloudevents.EventContext{
		CloudEventsVersion: cloudevents.CloudEventsVersion,
		EventType:          "dev.knative.source.heartbeats",
		EventID:            uuid.New().String(),
		Source:             "heartbeats-demo",
		EventTime:          time.Now(),
	}

	req, err := cloudevents.Binary.NewRequest(sink, message, ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create http request")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: code: %d, body: %s", resp.StatusCode, string(body))
	}

	io.Copy(ioutil.Discard, resp.Body)
	return nil
}
