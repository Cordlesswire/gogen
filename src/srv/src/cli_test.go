package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

var emptyParamCases = []string{
	"--logLevel=",
	"--logLevel=INVALID",
	"--serverAddress=",
	"--statsNetwork=INVALID",
	"--statsFlushPeriod=-1",
}

func TestCliEmptyParamError(t *testing.T) {
	for _, param := range emptyParamCases {
		os.Args = []string{"~#PROJECT#~", param}
		cmd, err := cli()
		if err != nil {
			t.Error(fmt.Errorf("An error wasn't expected: %v", err))
			return
		}
		if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
			t.Error(fmt.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype))
			return
		}

		old := os.Stderr // keep backup of the real stdout
		defer func() { os.Stderr = old }()
		os.Stderr = nil

		// execute the main function
		if err := cmd.Execute(); err == nil {
			t.Error(fmt.Errorf("An error was expected"))
		}
	}
}

func TestCli(t *testing.T) {
	os.Args = []string{
		"~#PROJECT#~",
		"--serverAddress=:8765",
		"--statsPrefix=~#PROJECT#~test",
		"--statsNetwork=udp",
		"--statsAddress=:8125",
		"--statsFlushPeriod=100",
	}
	cmd, err := cli()
	if err != nil {
		t.Error(fmt.Errorf("An error wasn't expected: %v", err))
		return
	}
	if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
		t.Error(fmt.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype))
		return
	}

	old := os.Stderr // keep backup of the real stdout
	defer func() { os.Stderr = old }()
	os.Stderr = nil

	// use two separate channels for server and client testing
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// start server
		if err := cmd.Execute(); err != nil {
			t.Error(fmt.Errorf("An error was not expected: %v", err))
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		// wait for the http server connection to start
		time.Sleep(1000 * time.Millisecond)

		// test index
		testEndPoint(t, "GET", "/", "", 200)
		// test 404
		testEndPoint(t, "GET", "/INVALID", "", 404)
		// test 405
		testEndPoint(t, "DELETE", "/", "", 405)
		// test valid endpoints
		testEndPoint(t, "GET", "/status", "", 200)

		wg.Done()
	}()
	wg.Wait()
}

// return true if the input is a JSON
func isJSON(s []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(s, &js) == nil
}

func testEndPoint(t *testing.T, method string, path string, data string, code int) {
	var payload = []byte(data)
	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:8765%s", path), bytes.NewBuffer(payload))
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != code {
		t.Error(fmt.Errorf("The espected status code is %d, found %d", code, resp.StatusCode))
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
		return
	}
	if !isJSON(body) {
		t.Error(fmt.Errorf("The body is not a JSON"))
	}
}