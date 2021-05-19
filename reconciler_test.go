// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	irc "github.com/fluffle/goirc/client"
)

func makeTestReconciler(config *Config) (*ChannelReconciler, chan bool, chan bool, *FakeTime) {

	sessionUp := make(chan bool)
	sessionDown := make(chan bool)

	client := irc.Client(makeGOIRCConfig(config))
	client.Config().Flood = true
	client.HandleFunc(irc.CONNECTED,
		func(*irc.Conn, *irc.Line) {
			sessionUp <- true
		})

	client.HandleFunc(irc.DISCONNECTED,
		func(*irc.Conn, *irc.Line) {
			sessionDown <- false
		})

	fakeDelayerMaker := &FakeDelayerMaker{}
	fakeTime := &FakeTime{
		afterChan: make(chan time.Time, 1),
	}
	reconciler := NewChannelReconciler(config, client, fakeDelayerMaker, fakeTime)

	return reconciler, sessionUp, sessionDown, fakeTime
}

func TestPreJoinChannels(t *testing.T) {
	server, port := makeTestServer(t)
	config := makeTestIRCConfig(port)
	config.IRCChannels = []IRCChannel{
		IRCChannel{Name: "#foo"},
		IRCChannel{Name: "#bar"},
		IRCChannel{Name: "#baz"},
	}
	reconciler, sessionUp, sessionDown, _ := makeTestReconciler(config)

	var testStep sync.WaitGroup

	joinedChannels := []string{}

	joinHandler := func(conn *bufio.ReadWriter, line *irc.Line) error {
		joinedChannels = append(joinedChannels, line.Args[0])
		if len(joinedChannels) == 3 {
			testStep.Done()
		}
		return hJOIN(conn, line)
	}
	server.SetHandler("JOIN", joinHandler)

	testStep.Add(1)

	reconciler.client.Connect()

	<-sessionUp
	reconciler.Start(context.Background())

	testStep.Wait()

	reconciler.client.Quit("see ya")
	<-sessionDown
	reconciler.Stop()

	server.Stop()

	expectedJoinedChannels := []string{"#bar", "#baz", "#foo"}
	sort.Strings(joinedChannels)

	if !reflect.DeepEqual(expectedJoinedChannels, joinedChannels) {
		t.Error("Did not pre-join channels")
	}
}

func TestKeepJoining(t *testing.T) {
	server, port := makeTestServer(t)
	config := makeTestIRCConfig(port)
	reconciler, sessionUp, sessionDown, fakeTime := makeTestReconciler(config)

	var testStep sync.WaitGroup

	var joinedCounter int

	// Confirm join only after a few attempts
	joinHandler := func(conn *bufio.ReadWriter, line *irc.Line) error {
		joinedCounter++

		if joinedCounter == 3 {
			testStep.Done()
			return hJOIN(conn, line)
		} else {
			fakeTime.afterChan <- time.Now()
		}
		return nil
	}
	server.SetHandler("JOIN", joinHandler)

	testStep.Add(1)

	reconciler.client.Connect()

	<-sessionUp
	reconciler.Start(context.Background())

	testStep.Wait()

	reconciler.client.Quit("see ya")
	<-sessionDown
	reconciler.Stop()

	server.Stop()

	expectedJoinedCounter := 3

	if !reflect.DeepEqual(expectedJoinedCounter, joinedCounter) {
		t.Error("Did not keep joining")
	}
}

func TestKickRejoin(t *testing.T) {
	server, port := makeTestServer(t)
	config := makeTestIRCConfig(port)
	reconciler, sessionUp, sessionDown, _ := makeTestReconciler(config)

	var testStep sync.WaitGroup

	// Wait for channel to be joined
	joinHandler := func(conn *bufio.ReadWriter, line *irc.Line) error {
		hJOIN(conn, line)
		testStep.Done()
		return nil
	}
	server.SetHandler("JOIN", joinHandler)

	testStep.Add(1)

	reconciler.client.Connect()

	<-sessionUp
	reconciler.Start(context.Background())

	testStep.Wait()

	// Kick and wait for channel to be joined again
	testStep.Add(1)
	server.SendMsg(":test!~test@example.com KICK #foo foo :Bye!\n")

	testStep.Wait()

	reconciler.client.Quit("see ya")
	<-sessionDown
	reconciler.Stop()

	server.Stop()

}
