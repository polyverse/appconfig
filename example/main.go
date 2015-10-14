package main

import "os"
import "fmt"
import "reflect"
import "encoding/json"
import cnf "github.com/polyverse-security/appconfig"
import log "github.com/Sirupsen/logrus"

func main() {
  log.SetLevel(log.DebugLevel)

  // Specify the arguments
  params := make(map[string]cnf.Param)
  params["config"] = cnf.Param{Type:cnf.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
  params["config-node"] = cnf.Param{Type:cnf.PARAM_CONFIG_NODE, Default:"crypto-proxy", Usage:"Node within the configuration file.", Required:false}
  params["debug"] = cnf.Param{Type:cnf.PARAM_BOOL, Default:false, Usage:"Debug mode.", PrefixOverride:"--"}
  params["proxy-addr"] = cnf.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
  params["remote_addr"] = cnf.Param{Usage:"Remote address[:port].", Required:true}
  params["statsd_addr"] = cnf.Param{Usage:"StatsD address:port."}
  params["ProxyRules"] = cnf.Param{Usage:"Maps routes to javascript handler functions", Required:true}
  params["buffer_size"] = cnf.Param{Type:cnf.PARAM_INT, Default:1024}
  params["help"] = cnf.Param{Type:cnf.PARAM_BOOL, Default:false, Usage:"Prints usage.", Required:false, PrefixOverride:"--"}

  fmt.Printf("\nThe following parameters have been defined:\n")
  b, err := json.Marshal(params)
  if err != nil {
    log.WithFields(log.Fields{"err":err}).Errorf("Error marshalling params.")
    os.Exit(1)
  }
  fmt.Printf("%s\n", string(b))

  config, err := cnf.NewConfig(params) // values is determined in the following order: (1) Default, (2) config file, (3) environmental variables and (4)command-line, each overriding the previous if value is provided.

  if config.Get("help").(bool) {
    config.PrintUsage("This app is a sample implementation of the polyverse-security/appconfig package.\n\n")
    os.Exit(0)
  }

  // Output the Values
  fmt.Printf("\nResult:\n")
  for param := range params {
    fmt.Printf("param = %s, value = %v, type = %s\n", param, config.Get(param), reflect.TypeOf(config.Get(param)))
  }

}
