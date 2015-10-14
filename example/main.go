package main

import "os"
import "fmt"
import "time"
import "reflect"
import cnf "github.com/polyverse-security/gouava/simpleconfig"
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
  params["help"] = cnf.Param{Type:cnf.PARAM_BOOL, Default:false, Usage:"Prints usage.", Required:false, PrefixOverride:"--"}

  fmt.Printf("\nThe following parameters have been defined:\n")
  for param := range params {
    fmt.Printf("param = %s, Default = %v, Type = %v, Required = %v, Usage = %s\n", param, params[param].Default, params[param].Type, params[param].Required, params[param].Usage)
  }
  fmt.Printf("\n")

  start := time.Now()
  config, err := cnf.NewConfig(params) // parses values from config file, then environmental variables, and finally the command-line
  log.Infof("simpleconfig.NewConfig() took: %v", time.Since(start))
  if err != nil {
    log.Errorf("Error: %v\n", err)
    config.PrintUsage("")
    os.Exit(1)
  }

  if config.Get("help").(bool) {
    config.PrintUsage("This is an example implementation of the polyverse-security/gouava/simpleconfig package.")
    os.Exit(0)
  }

  // Output the Values
  fmt.Printf("\nResult:\n")
  for param := range params {
    fmt.Printf("param = %s, value = %v, type = %s\n", param, config.Get(param), reflect.TypeOf(config.Get(param)))
  }

}
