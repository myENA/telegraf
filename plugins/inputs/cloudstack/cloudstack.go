package cloudstack

import (
	"encoding/json"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/myENA/telegraf/plugins/inputs"
	"net/http"
	"regexp"
	"strconv"
	"os"
)

const (
	sampleConfig = `
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
)

// Cloudstack struct Plugin
type CloudStack struct {
	client     *http.Client
	ApiUrl     string `toml:"api_url"`
	APIKey     string `toml:"api_key"`
	SecretKey  string `toml:"secret_key"`
	VerifySsl  bool
	DomainIds  []string
	AllDomains bool
}

// Description will appear directly above the plugin definition in the config file
func (c *CloudStack) Description() string {
	return `This plugin queries the CloudStack api listDomains command and grabs domain the data.`
}

// SampleConfig will populate the sample configuration portion of the plugin's configuration
func (c *CloudStack) SampleConfig() string {
	return sampleConfig
}

func init() {
	cs := &CloudStack{}
	inputs.Add("cloudstack", func() telegraf.Input { return cs.newApiConnection() })
}


func (c *CloudStack) checkEnvVariables() {
	if os.Getenv("CLOUDSTACK_API_URL") != "" {
		c.ApiUrl = os.Getenv("CLOUDSTACK_API_URL")
	}

	if os.Getenv("CLOUDSTACK_API_KEY") != "" {
		c.APIKey = os.Getenv("CLOUDSTACK_API_KEY")
	}

	if os.Getenv("CLOUDSTACK_SECRET_KEY") != "" {
		c.SecretKey = os.Getenv("CLOUDSTACK_SECRET_KEY")
	}
}

// Create the CS Client
func (c *CloudStack) newApiConnection() *CloudStack {
	c.checkEnvVariables()
	return c.newClient(c.ApiUrl, c.APIKey, c.SecretKey, false, c.VerifySsl)
}

// Get the Domain data from CS
func (c *CloudStack) listDomains() (map[string]interface{}, error) {
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
func (c *CloudStack) buildFields(message map[string]interface{}, acc telegraf.Accumulator) (map[string]interface{}, map[string]string) {

	// Use regex to filter out all the tag fields
	pattern, _ := regexp.Compile("(haschild|id|name|parentdomainid|parentdomainname|path|state|networkdomain)")

	fields := make(map[string]interface{})
	tags := make(map[string]string)

	for k, v := range message {

		// Exlude all the fields we will be using for tags
		if pattern.MatchString(k) {
			tags[k] = v.(string)
		} else {

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
	}

	return fields, tags
}


// Gather defines what data the plugin will gather.
func (c *CloudStack) Gather(acc telegraf.Accumulator) error {
	data, err := c.listDomains()

	if err != nil {
		acc.AddError(fmt.Errorf("error listing domain data: %s", err))
		return err
	}

	fields, tags := c.buildFields(data, acc)

	acc.AddFields("cs_domains", fields, tags)

	return nil
}
