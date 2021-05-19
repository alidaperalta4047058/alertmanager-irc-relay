// Copyright 2018 Google LLC
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
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestNoConfig(t *testing.T) {
	noConfigFile := ""

	config, err := LoadConfig(noConfigFile)
	if config == nil {
		t.Errorf("Expected a default config, got: %s", err)
	}

}

func TestLoadGoodConfig(t *testing.T) {
	expectedConfig := &Config{
		HTTPHost:        "test.web",
		HTTPPort:        8888,
		IRCNick:         "foo",
		IRCHost:         "irc.example.com",
		IRCPort:         1234,
		IRCHostPass:     "hostsecret",
		IRCUseSSL:       true,
		IRCChannels:     []IRCChannel{IRCChannel{Name: "#foobar"}},
		MsgTemplate:     defaultMsgTemplate,
		MsgOnce:         false,
		UsePrivmsg:      false,
		AlertBufferSize: 666,
	}
	expectedData, err := yaml.Marshal(expectedConfig)
	if err != nil {
		t.Errorf("Could not serialize test data: %s", err)
	}

	tmpfile, err := ioutil.TempFile("", "airtestconfig")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(expectedData); err != nil {
		t.Errorf("Could not write test data in tmpfile: %s", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Errorf("Could not close tmpfile: %s", err)
	}

	config, err := LoadConfig(tmpfile.Name())
	if config == nil {
		t.Errorf("Expected a config, got: %s", err)
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		t.Errorf("Could not serialize loaded config")
	}

	if string(expectedData) != string(configData) {
		t.Errorf("Loaded config does not match expected config: %s", configData)
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	expectedNickPass := "mynickpass"

	os.Setenv("NICKSERV_PASSWORD", expectedNickPass)
	defer os.Clearenv()

	tmpfile, err := ioutil.TempFile("", "airtestenvvarconfig")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	msgOnceConfigData := []byte("irc_nickname_password: $NICKSERV_PASSWORD")
	if _, err := tmpfile.Write(msgOnceConfigData); err != nil {
		t.Errorf("Could not write test data in tmpfile: %s", err)
	}
	tmpfile.Close()

	config, err := LoadConfig(tmpfile.Name())
	if config == nil {
		t.Errorf("Expected a config, got: %s", err)
	}

	if config.IRCNickPass != expectedNickPass {
		t.Errorf("Loaded unexpected value: %s (expected: %s)",
			config.IRCNickPass, expectedNickPass)
	}
}

func TestLoadBadFile(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "airtestbadfile")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	config, err := LoadConfig(tmpfile.Name())
	if err == nil || config != nil {
		t.Errorf("Expected no config upon non-existent file. err: %s", err)
	}
}

func TestLoadBadConfig(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "airtestbadconfig")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	badConfigData := []byte("footest\nbarbaz\n")
	if _, err := tmpfile.Write(badConfigData); err != nil {
		t.Errorf("Could not write test data in tmpfile: %s", err)
	}
	tmpfile.Close()

	config, err := LoadConfig(tmpfile.Name())
	if err == nil || config != nil {
		t.Errorf("Expected no config upon bad config. err: %s", err)
	}
}

func TestMsgOnceDefaultTemplate(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "airtesttemmplateonceconfig")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	msgOnceConfigData := []byte("msg_once_per_alert_group: yes")
	if _, err := tmpfile.Write(msgOnceConfigData); err != nil {
		t.Errorf("Could not write test data in tmpfile: %s", err)
	}
	tmpfile.Close()

	config, err := LoadConfig(tmpfile.Name())
	if config == nil {
		t.Errorf("Expected a config, got: %s", err)
	}

	if config.MsgTemplate != defaultMsgOnceTemplate {
		t.Errorf("Expecting defaultMsgOnceTemplate when MsgOnce is true")
	}
}

func TestMsgDefaultTemplate(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "airtesttemmplateconfig")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte("")); err != nil {
		t.Errorf("Could not write test data in tmpfile: %s", err)
	}
	tmpfile.Close()

	config, err := LoadConfig(tmpfile.Name())
	if config == nil {
		t.Errorf("Expected a config, got: %s", err)
	}

	if config.MsgTemplate != defaultMsgTemplate {
		t.Errorf("Expecting defaultMsgTemplate when MsgOnce is false")
	}
}

func TestGivenTemplateNotOverwritten(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "airtestexpectedtemmplate")
	if err != nil {
		t.Errorf("Could not create tmpfile for testing: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	expectedTemplate := "Alert {{ .Status }}: {{ .Annotations.SUMMARY }}"
	configData := []byte(fmt.Sprintf("msg_template: \"%s\"", expectedTemplate))
	if _, err := tmpfile.Write(configData); err != nil {
		t.Errorf("Could not write test data in tmpfile: %s", err)
	}
	tmpfile.Close()

	config, err := LoadConfig(tmpfile.Name())
	if config == nil {
		t.Errorf("Expected a config, got: %s", err)
	}

	if config.MsgTemplate != expectedTemplate {
		t.Errorf("Template does not match configuration")
	}
}
