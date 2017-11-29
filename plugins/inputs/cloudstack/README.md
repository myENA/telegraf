# Example Input Plugin

This plugin pulls domain data from Cloudstack. Due to the complexities of how cloudstack pulls VM information it
does not pull VM metric data from listResourceData. A lot of the initial work for this has been done, but
each management server stores different pieces of data about vm performance and the plugin would need to hit all the management servers then reconcile the
data before shipping

### Configuration:

This section contains the default TOML to configure the plugin.  You can
generate it using `telegraf --usage <plugin-name>`.

```toml
# Description
[[inputs.cloudstack]]
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
```

### Metrics:
All metrics passed are listed here with a description:

https://cloudstack.apache.org/api/apidocs-4.10/apis/listDomains.html

### Example Output:

```
cs_domain,haschild=false,id=a4a1fc77-9d02-4b60-b799-c09730c22bb2,path=ROOT/k12/2201,name=2201,
state=Active,host=dhendel-hp,parentdomainid=f10185f2-5d6a-40b5-b278-edf494993e9b,parentdomainname=k12 
vpctotal=1,vmtotal=0,snapshottotal=0,primarystoragetotal=0,snapshotlimit=128i,projectavailable=-1i,
secondarystorageavailable=-1i,volumelimit=-1i,cputotal=0,volumetotal=0,memorylimit=40000i,
primarystoragelimit=1536i,iptotal=0,templateavailable=-1i,projecttotal=0,vmavailable=-1i,
primarystorageavailable=1536i,level=2,memorytotal=0,secondarystoragelimit=-1i,memoryavailable=40000i,
volumeavailable=-1i,templatelimit=-1i,snapshotavailable=128i,cpuavailable=16i,vmlimit=-1i,
secondarystoragetotal=0,networkavailable=-1i,networklimit=-1i,templatetotal=0,cpulimit=16i,
vpcavailable=-1i,ipavailable=4i,projectlimit=-1i,networktotal=1,vpclimit=-1i,iplimit=4i 1511984405000000000
```
