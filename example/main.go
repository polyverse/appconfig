package main

import "os"
import "fmt"
import "reflect"
import "time"
import "github.com/polyverse-security/appconfig"

func main() {
  // Set logging level to Debug so we can see all the Debug messages from polyverse-security/appconfig

  // Specify the arguments
  params := make(map[string]appconfig.Param)
  params["config"] = appconfig.Param{Type:appconfig.PARAM_CONFIG_JSON, Default:"config.json", Usage:"json config file.", Required:false}
  params["config-node"] = appconfig.Param{Type:appconfig.PARAM_CONFIG_NODE, Default:"example", Usage:"root node in the config file.", Required:false}
  params["debug"] = appconfig.Param{Type:appconfig.PARAM_BOOL, Default:false, Usage:"verbose output.", PrefixOverride:"--"}
  params["port"] = appconfig.Param{Type:appconfig.PARAM_STRING, Default:":8080", Usage:"bind-to port.", Required:true}
  params["statsd_addr"] = appconfig.Param{Type:appconfig.PARAM_STRING, Usage:"statsd endpoint."}
  params["timeout"] = appconfig.Param{Type:appconfig.PARAM_INT, Usage:"server timeout (ms).", Default:1000}
  params["help"] = appconfig.Param{Type:appconfig.PARAM_USAGE, Default:false, Usage:"print usage.", Required:false, PrefixOverride:"--"}

  fmt.Printf("\nThe following parameters have been defined:")
  var str string
  for param := range params {
    str = str + fmt.Sprintf("\n\tparam=\"%s\", Type=%v, Default=%v, Usage=\"%v\", Required=%v, PrefixOverride=\"%v\"", param, params[param].Type, params[param].Default, params[param].Usage, params[param].Required, params[param].PrefixOverride)
  }
  fmt.Printf("%s\n\n", str)

  start := time.Now()
  fmt.Printf("*** Calling appconfig.NewConfig()...\n")
  if appconfig.GetBoolFromCommandLine("debug", params) {
    appconfig.SetLogLevel(appconfig.DebugLevel)
  }
  config, err := appconfig.NewConfig(params) // values is determined in the following order: (1) Default, (2) config file, (3) environmental variables and (4)command-line, each overriding the previous if value is provided.
  if err != nil {
    fmt.Printf("Error: %s\n", err.Error())
    os.Exit(1)
  }
  fmt.Printf("*** Done. Elapsed time: %v\n", time.Since(start))

  if config.Get("help").(bool) {
    config.PrintUsage("This app is a sample implementation of the polyverse-security/appconfig package.\n\n")
    os.Exit(0)
  }

  // Output the Values
  fmt.Printf("\nResult:\n")
  for param := range params {
    fmt.Printf("\tparam = %s, value = %v, type = %s\n", param, config.Get(param), reflect.TypeOf(config.Get(param)))
  }

}
