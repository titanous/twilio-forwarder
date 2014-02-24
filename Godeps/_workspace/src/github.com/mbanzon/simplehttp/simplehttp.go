// Package simplehttp provides some simple methods and types to do
// HTTP queries with form values and parameters easily - especially
// if the returned result is expected to be JSON or XML.
//
// Author: Michael Banzon
package simplehttp

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Holds all information used to make a HTTP request.
type SimpleHTTPRequest struct {
	Method            string
	URL               string
	Parameters        map[string][]string
	FormValues        map[string][]string
	Headers           map[string]string
	BasicAuthUser     string
	BasicAuthPassword string

	LastResponseCode int
	LastRawResponse  []byte
}

// Creates a new SimpleHTTPRequest instance.
func NewSimpleHTTPRequest(method, url string) *SimpleHTTPRequest {
	return &SimpleHTTPRequest{Method: method, URL: url}
}

// Creates a new instance of SimpleHTTPRequest to make a GET request.
func NewGetRequest(url string) *SimpleHTTPRequest {
	return NewSimpleHTTPRequest("GET", url)
}

// Creates a new instance of SimpleHTTPRequest to make a POST request.
func NewPostRequest(url string) *SimpleHTTPRequest {
	return NewSimpleHTTPRequest("POST", url)
}

// Creates a new instance of SimpleHTTPRequest to make a DELETE request.
func NewDeleteRequest(url string) *SimpleHTTPRequest {
	return NewSimpleHTTPRequest("DELETE", url)
}

// Adds a parameter to the generated query string.
func (r *SimpleHTTPRequest) AddParameter(name, value string) {
	if r.Parameters == nil {
		r.Parameters = make(map[string][]string)
	}
	r.Parameters[name] = append(r.Parameters[name], value)
}

// Adds a form value to the request.
func (r *SimpleHTTPRequest) AddFormValue(name, value string) {
	if r.FormValues == nil {
		r.FormValues = make(map[string][]string)
	}
	r.FormValues[name] = append(r.FormValues[name], value)
}

// Adds a header that will be sent with the HTTP request.
func (r *SimpleHTTPRequest) AddHeader(name, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
}

// Sets username and password for basic authentication.
func (r *SimpleHTTPRequest) SetBasicAuth(user, password string) {
	r.BasicAuthUser = user
	r.BasicAuthPassword = password
}

// Clears the last received response.
func (r *SimpleHTTPRequest) ClearLastResponse() {
	r.LastResponseCode = -1
	r.LastRawResponse = nil
}

// Makes the prepared request and tries to unmarshal the result as JSON to the supplied interface.
func (r *SimpleHTTPRequest) MakeJSONRequest(v interface{}) error {
	responseBody, err := r.MakeRequest()
	if err != nil {
		return err
	}
	return json.Unmarshal(responseBody, v)
}

// Makes the prepared request and tries to unmarshal the result as XML to the supplied interface.
func (r *SimpleHTTPRequest) MakeXMLRequest(v interface{}) error {
	responseBody, err := r.MakeRequest()
	if err != nil {
		return err
	}
	return xml.Unmarshal(responseBody, v)
}

// Makes the prepared request and returns the result as a byte-slice.
func (r *SimpleHTTPRequest) MakeRequest() ([]byte, error) {
	url, err := r.generateUrlWithParameters()
	if err != nil {
		return make([]byte, 0), err
	}
	bodyData, hasBody := r.makeBodyData()

	var body io.Reader
	if hasBody {
		body = bytes.NewBufferString(bodyData.Encode())
	} else {
		body = nil
	}

	req, err := http.NewRequest(r.Method, url, body)
	if err != nil {
		return make([]byte, 0), err
	}

	if r.BasicAuthUser != "" && r.BasicAuthPassword != "" {
		req.SetBasicAuth(r.BasicAuthUser, r.BasicAuthPassword)
	}

	for header, value := range r.Headers {
		req.Header.Add(header, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	r.LastRawResponse = nil
	if resp != nil {
		r.LastResponseCode = resp.StatusCode
	}
	if err != nil {
		return make([]byte, 0), err
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return make([]byte, 0), err
	}

	r.LastRawResponse = responseBody
	return responseBody, nil
}

func (r *SimpleHTTPRequest) makeBodyData() (data url.Values, hasBody bool) {
	data = url.Values{}
	if r.FormValues != nil && len(r.FormValues) > 0 {
		hasBody = true
		r.AddHeader("Content-Type", "application/x-www-form-urlencoded")
		for name, values := range r.FormValues {
			for _, value := range values {
				data.Add(name, value)
			}
		}
	} else {
		hasBody = false
	}

	return
}

func (r *SimpleHTTPRequest) generateUrlWithParameters() (string, error) {
	url, err := url.Parse(r.URL)
	if err != nil {
		return "", err
	}
	q := url.Query()
	if r.Parameters != nil && len(r.Parameters) > 0 {
		for name, values := range r.Parameters {
			for _, value := range values {
				q.Add(name, value)
			}
		}
	}
	url.RawQuery = q.Encode()

	return url.String(), nil
}
