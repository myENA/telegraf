package cloudstack

import (
	"encoding/json"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/myENA/telegraf/plugins/inputs"
	"net/http"
	"regexp"
	"strconv"
)

const sampleConfig = `
  ## You can skip the client setup portion of this config if the following environment variables are set:
  ## CLOUDSTACK_API_URL
  ## CLOUDSTACK_API_KEY
  ## CLOUDSTACK_SECRET_KEY

  ## Specify the cloudstack api url. This can also be extracted from CLOUDSTACK_API_URL
  api_url = "http://localhost:8080/client/api"

  ## The api key for the cloudstack API. This can also be extracted from CLOUDSTACK_API_KEY
  api_access_key = ""

  ## The api secret key for the cloudstack API. This can also be extracted from CLOUDSTACK_SECRET_KEY
  api_secret_key = ""
`

// Cloudstack struct Plugin
type Cloudstack struct {
	client     *http.Client
	ApiUrl     string
	APIKey     string
	SecretKey  string
	VerifySsl  bool
	DomainIds  []string
	AllDomains bool
}

// Create the CS Client
func (c *Cloudstack) newApiConnection() {
	newClient(c.ApiUrl, c.APIKey, c.SecretKey, false, c.VerifySsl)
}

// Get the Domain data from CS
func (c *Cloudstack) listDomains() (map[string]interface{}, error) {
	respJson := make(map[string]interface{})
	jsonData, err := c.newRequest("listDomains", nil)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &respJson)

	if err != nil {
		return nil, err
	}

	return respJson, nil
}

// Build all our interfaces for telegraf
func (c *Cloudstack) buildFields(message map[string]interface{}, acc telegraf.Accumulator) (map[string]interface{}, map[string]string) {

	// Use regex to filter out all the tag fields
	pattern, _ := regexp.Compile("(haschild|id|name|parentdomainid|parentdomainname|path|state|networkdomain)")

	fields := make(map[string]interface{})
	tags := make(map[string]string)

	for k, v := range message {

		// Exlude all the fields we will be using for tags
		if pattern.MatchString(k) {
			tags[k] = v.(string)
		}

		// Setting fields that are unlimited to -1 since there is no float value for unlimited
		if v == "Unlimited" {
			fields[k] = -1
		} else {
			newFloat, err := strconv.ParseFloat(v.(string), 64)

			if err != nil {
				acc.AddError(fmt.Errorf("error parsing float from %s: %s", v.(string), err))
				continue
			}
			fields[k] = newFloat
		}
	}

	return fields, tags
}

func init() {
	inputs.Add("rancher", func() telegraf.Input { return &Cloudstack{} })
}

// Description will appear directly above the plugin definition in the config file
func (c *Cloudstack) Description() string {
	return `This plugin queries the Cloudstack api listDomains command and grabs domain the data.`
}

// SampleConfig will populate the sample configuration portion of the plugin's configuration
func (c *Cloudstack) SampleConfig() string {
	return sampleConfig
}

// Gather defines what data the plugin will gather.
func (c *Cloudstack) Gather(acc telegraf.Accumulator) error {
	c.newApiConnection()

	data, err := c.listDomains()

	if err != nil {
		acc.AddError(fmt.Errorf("error listing domain data: %s", err))
		return err
	}

	fields, tags := c.buildFields(data, acc)

	acc.AddFields("cs_domains", fields, tags)

	return nil
}
