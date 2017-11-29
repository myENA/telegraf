package cloudstack

import (
	"github.com/influxdata/telegraf"

	"net/http"
	"os"
	"regexp"
	"fmt"
	"github.com/influxdata/telegraf/plugins/inputs"
	"encoding/json"
	"net/url"
	"strconv"
	"github.com/pkg/errors"
)

//Regex pattern to match some domain fields used for tags
var pattern = regexp.MustCompile("^(haschild|id|name|parentdomainid|parentdomainname|path|state|networkdomain)$")
const (
	sampleConfig = `
  ## You can skip the client setup portion of this config if the following environment variables are set:
  ## CLOUDSTACK_API_URL
  ## CLOUDSTACK_API_KEY
  ## CLOUDSTACK_SECRET_KEY

  ## Specify the cloudstack api url. This can also be extracted from CLOUDSTACK_API_URL
  api_url = "http://localhost:8080/client/api"

  ## The api key for the cloudstack API. This can also be extracted from CLOUDSTACK_API_KEY
  api_key = ""

  ## The api secret key for the cloudstack API. This can also be extracted from CLOUDSTACK_SECRET_KEY
  secret_key = ""

  verify_ssl = false
`
)

// MockPlugin struct should be named the same as the Plugin
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
	return `This is an example plugin`
}

// SampleConfig will populate the sample configuration portion of the plugin's configuration
func (c *CloudStack) SampleConfig() string {
	return sampleConfig
}

func init() {
	cs := &CloudStack{}
	inputs.Add("cloudstack", func() telegraf.Input { return cs.newApiConnection() })
}

// Check if ENV variables are set
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

// Get all the virtualMachine data by domainId
func (c *CloudStack) listVirtualMachines(domainId string) ([]interface{}, error) {
	respJson := make(map[string]interface{})

	//Refresh the resource data
	_, err := c.newRequest("updateResourceCount", url.Values{"domainid": []string{domainId}})

	if err != nil {
		return nil, err
	}

	jsonData, err := c.newRequest("listVirtualMachines", url.Values{"domainid": []string{domainId}})

	err = json.Unmarshal(jsonData, &respJson)

	if err != nil {
		return nil, err
	}

	if domain, ok := respJson["virtualmachine"]; ok {
		if vms, ok := domain.([]interface{}); ok {
			return vms, nil
		}
	}

	return nil, nil
}


// Get the Domain data from CS
func (c *CloudStack) listDomains() ([]interface{}, error) {
	respJson := make(map[string]interface{})
	jsonData, err := c.newRequest("listDomains", url.Values{"listall": []string{"true"}})

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &respJson)

	if err != nil {
		return nil, err
	}

	if domain, ok := respJson["domain"]; ok {
		if dmns, ok := domain.([]interface{}); ok {
			return dmns, nil
		}
	}

	return nil, errors.New("Could not find domain in response")
}

// Cloudstack Domain API does not properly calculate total cpu and memory across all vm's
func (c *CloudStack) calculateTotals(vmMap []interface{}) (float64, float64) {
	var memTotal float64
	var cpuTotal float64


	memTotal = 0
	cpuTotal = 0

	for _, vm := range vmMap {
		if domain, ok := vm.(map[string]interface{}); ok {
			cpuTotal = cpuTotal + domain["cpunumber"].(float64)
			memTotal = memTotal + domain["memory"].(float64)
		}
	}

	return memTotal, cpuTotal
}


func strToInt(value string, acc telegraf.Accumulator) int {
	newInt, err := strconv.Atoi(value)
	if err != nil {
		acc.AddError(fmt.Errorf("unable to  convert string to float: %#v", value))
	}

	return newInt
}

// Ship the domain data
func (c *CloudStack) gatherDomainData(data []interface{}, acc telegraf.Accumulator) error {

	fields := make(map[string]interface{})
	tags := make(map[string]string)

	for _, d := range data {

		if message, ok := d.(map[string]interface{}); ok {

			vmData, err := c.listVirtualMachines(message["id"].(string))

			message["memorytotal"], message["cputotal"] = c.calculateTotals(vmData)

			if err != nil {
				acc.AddError(fmt.Errorf("unable to get VM data for domain %s", message["name"].(string)))
			}


			for k, v := range message {
				// Grab all the tags and add to tags map
				if pattern.MatchString(k) {
					var value string
					var err error
					switch v.(type) {
					case bool:
						value = strconv.FormatBool(v.(bool))
					default:
						value = v.(string)
					}
					if err != nil {
						acc.AddError(fmt.Errorf(""))
					} else {
						tags[k] = value
					}
				} else {

					// Handle the data based on value type
					switch v.(type) {
					case string :
						// Setting fields that are unlimited to -1 since there is no float value for unlimited
						if v.(string) == "Unlimited" {
							fields[k] = -1
						} else {
							switch k {
							case "cpuavailable":
								// If cpuavailable is not already set to int
								if _, isInt := v.(int); !isInt {
									value := strToInt(v.(string), acc)
									fields[k] = value - int(message["cputotal"].(float64))
								}
							case "memoryavailable":
								// If memoryavailable is not already set to int
								if _, isInt := v.(int); !isInt {
									value := strToInt(v.(string), acc)
									fields[k] = value - int(message["memorytotal"].(float64))
								}

							// String fields could be unlimited and parse to float fails, must be an int
							default:
								fields[k] = strToInt(v.(string), acc)
							}
						}
					case float64:
							fields[k] = v
					default:
						acc.AddError(fmt.Errorf("cloudstack plugin got unexpected value type for %v", v))
					}

				}

			}
			acc.AddFields("cs_domain", fields, tags)
		}
	}
	return nil
}

//// Gather VM data
//func (c *CloudStack) gatherVMData(domains []interface{}, acc telegraf.Accumulator) error {
//	// Range over the domains
//	for _, domain := range domains{
//		fields := make(map[string]interface{})
//		tags := make(map[string]string)
//		// Range over all the objects in the domains []interface
//		if vmDomain, ok := domain.(map[string]interface{}); ok {
//
//			// Get the vms for the passed domain id
//			vmData, err := c.listVirtualMachines(vmDomain["id"].(string))
//
//			if err != nil {
//				acc.AddError(fmt.Errorf("unable to get VM data for vm %s", vmDomain["name"].(string)))
//				return err
//			}
//
//			if vmData != nil {
//				// vmData is a []interface loop again
//				for _ , vmMap := range vmData{
//
//					// make sure we have a map[string]interface{}
//					if vm, ok := vmMap.(map[string]interface{}); ok {
//						for k, v := range vm {
//
//							// Make the magic. Luckily most of the values parse correctly by type
//							switch v.(type) {
//
//							// Strings will be the tags, except cpuused
//							case string:
//
//								// Cpu percentage is passed as a string we need to parse it and add it to fields
//								if k == "cpuused" {
//									//Get rid of the pesky %
//									stringFloat := strings.Replace(v.(string), "%", "", 1)
//									//Parse our float
//									newFloat, err := strconv.ParseFloat(stringFloat, 64)
//
//									if err != nil {
//										acc.AddError(fmt.Errorf("error parsing cpu used percetnage to float: %s", err))
//									}
//
//									fields[k] = newFloat
//								}
//
//								tags[k] = v.(string)
//							// Bools will be a tag
//							case bool:
//								tags[k] = strconv.FormatBool(v.(bool))
//							// Floats will be metrics
//							case float64:
//								fields[k] = v
//							default:
//								// nic is a different animal all together, it is a []interface{}{map[string]interface{}}
//								// we just want the network name so we have to do some work
//								if k == "nic"{
//									if nic, ok := v.([]interface{}); ok {
//										for _, nicInt := range nic {
//											if nicData, ok := nicInt.(map[string]interface{}); ok {
//												tags["networkname"] = nicData["networkname"].(string)
//											}
//
//										}
//									}
//								}
//							}
//						}
//					}
//
//				}
//			}
//
//		}
//		acc.AddFields("cs_vm", fields, tags)
//	}
//	return nil
//}



// Gather defines what data the plugin will gather.
func (c *CloudStack) Gather(acc telegraf.Accumulator) error {
	data, err := c.listDomains()

	if err != nil {
		acc.AddError(fmt.Errorf("error listing domain data: %s", err))
		return err
	}

	if err := c.gatherDomainData(data, acc); err != nil {
		acc.AddError(err)
	}

	//if err := c.gatherVMData(data, acc); err != nil {
	//	acc.AddError(err)
	//}


	return nil
}
