package cloudstack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

//Borrowed from the xanzy/cloudstack library, no need to import the whole thing for one call

//  Build new client
func newClient(apiurl string, apikey string, secret string, async bool, verifyssl bool) *Cloudstack {
	cs := &Cloudstack{
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifyssl}, // If verifyssl is true, skipping the verify should be false and vice versa
			},
			Timeout: time.Duration(60 * time.Second),
		},
		ApiUrl:    apiurl,
		APIKey:    apikey,
		SecretKey: secret,
	}
	return cs
}

// Encode url values
func encodeValues(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := k + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

// Generic function to get the first raw value from a response as json.RawMessage
func getRawValue(b json.RawMessage) (json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for _, v := range m {
		return v, nil
	}
	return nil, fmt.Errorf("Unable to extract the raw value from:\n\n%s\n\n", string(b))
}

// CS requuires a signature with the request, this method (borrowed from xanzy) takes care of generating the sig
func (cs *Cloudstack) newRequest(api string, params url.Values) (json.RawMessage, error) {
	params.Set("apiKey", cs.APIKey)
	params.Set("command", api)
	params.Set("response", "json")

	// Generate signature for API call
	// * Serialize parameters, URL encoding only values and sort them by key, done by encodeValues
	// * Convert the entire argument string to lowercase
	// * Replace all instances of '+' to '%20'
	// * Calculate HMAC SHA1 of argument string with CloudStack secret
	// * URL encode the string and convert to base64
	s := encodeValues(params)
	s2 := strings.ToLower(s)
	s3 := strings.Replace(s2, "+", "%20", -1)
	mac := hmac.New(sha1.New, []byte(cs.SecretKey))
	mac.Write([]byte(s3))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	var err error
	var resp *http.Response

	// Create the final URL before we issue the request
	url := cs.ApiUrl + "?" + s + "&signature=" + url.QueryEscape(signature)

	// Make a GET call
	resp, err = cs.client.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Need to get the raw value to make the result play nice
	b, err = getRawValue(b)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {

	}
	return b, nil
}
