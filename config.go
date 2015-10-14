// This package makes it easy to define paramaters that your application accepts
// and supports default values and usage like Go's flag package but also allows
// you to specify type and whether the parameter is required.
//
// Once you define the paramters, this package will take the default value
// you specified and then override that value in the following order:
// 1. JSON configuration file
// 2. environmental variables
// 3. command-line
//
// Additionally, you can specify the root node of the configuration file which
// will allow the same configuration file to contain configuration data for
// multiple applications.
//
// A full example implementation is available in example/.
package simpleconfig

import "fmt"
import "os"
import "strings"
import "strconv"
import "reflect"
import "encoding/json"
import log "github.com/Sirupsen/logrus"

// ParamType is an optional property of the Param struct. If ommitted, there
// is no type-checking of the parameter value.
type ParamType int

// Constants for the ParamType type.
const (
  PARAM_STRING ParamType = iota
  PARAM_INT
  PARAM_BOOL
  PARAM_OBJECT
  PARAM_CONFIG_JSON
  PARAM_CONFIG_NODE
)

const default_prefix = "-"

// This is the struct you use to specify the properties of each parameter.
// The `NewConfig(params map[string]Param) (Config, err)` expects you to pass
// an array of this struct with the parameter name being the map index.
//
// None of the fields are required.
type Param struct {
  Type ParamType // if ommited, default is 0 an no type-checking will be performed.
  Default interface{} // if ommited, the default value is nil.
  Usage string // if ommitted, no usage will be provided for this parameter when calling `PrintUsage(message string)`
  Required bool // if ommitted, default is false
  PrefixOverride string // if ommitted, default is "-"
}

// This is the object that's returned from NewConfig() and provides methods
// such as `Get(key string) interface{}` (retrieve values) and `PrintUsage(message string)`.
type Config struct {
  values map[string]interface{}
  params map[string]Param
}

// This is a helper function that returns the parameter name prepended with
// the proper switch prefix. The default prefix is "-" but you can override that
// with Param.PrefixOverride. Since the prefixes are stripped and the name used
// as the key for the paramters map, this helper function allows you to reconstruct
// the command-line switch.
func (c *Config) GetKeyWithPrefix(key string) string {
  prefix := c.params[key].PrefixOverride
  if prefix == "" {
    prefix = default_prefix
  }
  return prefix + key
}

// This is a helper function that returns a string array of all parameter names
// where the Param.Type matches what you pass in.
func (c *Config) GetParamKeysByType(paramType ParamType) []string {
  var returnParams []string

  for param := range c.params {
    if c.params[param].Type == paramType {
      returnParams = append(returnParams, param)
    }
  }
  return returnParams
}
// Create a new map[string]Param and then populate it with the parameters you
// want your application to accept.
//
// Example:
//
//   ```
//   params := make(map[string]cnf.Param)
//   params["config"] = simpleconfig.Param{Type:cnf.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
//   params["proxy-addr"] = cnf.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
//   params["statsd_addr"] = cnf.Param{Usage:"StatsD address:port."}
//   config := NewConfig(params)
//   ```
//
// There are a lot of `logrus.Debugf()` statements that will be output if
// loglevel is set to `log.DebugLevel`. The debug output will tell you exactly
// where the values are taken from.
func NewConfig(params map[string]Param) (Config, error) {
  config := Config{make(map[string]interface{}),params}

  args := make(map[string]string)

  // Make a map of the environmental variables
  log.Debug("Checking environmental variables...")
  for param := range params {
    val := os.Getenv(param)
    if val != "" {
      args[param] = os.Getenv(param)
      log.Debugf("----> Found match: %s = %s", param, args[param])
    }
  }
  log.Debugf("--> Done. Environmental variables: %v", args)

  // Make a map of the command-line arguments
  log.Debugf("Processing command-line arguments: %v", os.Args[1:])

  // enumerate the command-line arguments
  for i := 1; i <= len(os.Args[1:]); i++ {
    // compare each argument with list of supported paramters
    log.Debugf("--> Process argument: %s", os.Args[i])
    match := false // flag to specify whether argument was found in list of supported paramters
    for param := range params {
      kv := strings.Split(os.Args[i], "=") // split the argument into key + value
      prefix := default_prefix
      if params[param].PrefixOverride != "" {
        prefix = params[param].PrefixOverride
      }
      arg := strings.TrimPrefix(kv[0], prefix)
      if param == arg {
        // set the kv pair in the args map
        match = true
        if len(kv) == 1 {
          args[arg] = "true" // if value isn't provided, default to true
        } else {
          args[arg] = kv[1]
        }
        log.Debugf("----> Found match: %s = %s", param, args[arg])
        break
      }
    }
    if !match {
      log.Debugf("----> No match.")
      err := fmt.Errorf("'%s' is not a supported flag.", os.Args[i])
      return config, err
    }
  }
  log.Debugf("--> Done. Environment variables + command-line arguments overrides: %v", args)

  configJson := args[config.GetParamKeysByType(PARAM_CONFIG_JSON)[0]]
  if configJson == "" {
    configJson = params[config.GetParamKeysByType(PARAM_CONFIG_JSON)[0]].Default.(string)
  }
  configNode := args[config.GetParamKeysByType(PARAM_CONFIG_NODE)[0]]
  if configNode == "" {
    configNode = params[config.GetParamKeysByType(PARAM_CONFIG_NODE)[0]].Default.(string)
  }

  jsonvals := make(map[string]interface{})
  if configJson != "" {
    log.Debugf("Reading config file: file = %s, node = %s", configJson, configNode)

    if f, err := os.Open(configJson); err != nil {
      log.WithFields(log.Fields{"configJson":configJson}).Errorf("Error opening config file.")
      return config, err
    } else { // opened file successfully
      jsonParser := json.NewDecoder(f)
      if err := jsonParser.Decode(&jsonvals); err != nil {
        log.WithFields(log.Fields{"configJson":configJson, "Error": err}).Errorf("Error unmarshalling JSON script.")
        return config, err
      }
    }
    log.WithFields(log.Fields{"JSON":jsonvals}).Debugf("-->Loaded json config file.")

    // If a configNode is specified, then the config file is expected to have
    // more info than needed. Set jsonvals to just the portion we're interested in.
    if configNode != "" {
      jsonvals = jsonvals[configNode].(map[string]interface{})
      log.WithFields(log.Fields{"JSON":jsonvals}).Debugf("-->Trimming JSON based on PARAM_CONFIG_NODE = '%s'", configNode)
    }
  } else {
    log.Debugf("No configuration file specified.")
  }

  log.Debugf("Finalizing configuration values...")
  for param := range params {
    log.Debugf("--> Processing param: %s", param)
    if params[param].Default != nil {
      config.values[param] = params[param].Default
      log.Debugf("----> Setting default: %s = %v (type: %s)", param, params[param].Default, reflect.TypeOf(params[param].Default))
    } else {
      log.Debugf("----> No default value provided.")
    }
    if jsonvals[param] != nil {
      config.values[param] = jsonvals[param]
      log.Debugf("----> Config file override: %s = %v (type: %s)", param, jsonvals[param], reflect.TypeOf(jsonvals[param]))
    }
    if args[param] != "" {
      config.values[param] = args[param]
      log.Debugf("----> Environmental and command-line override: %s = %v (type: %s)", param, args[param], reflect.TypeOf(args[param]))
    }

    if _, ok := config.values[param]; !ok {
      if params[param].Required {
        err := fmt.Errorf("Missing required parameter '%s'.", param)
        return config, err
      }
    }

    switch params[param].Type {
      case PARAM_BOOL: {
        if reflect.TypeOf(config.values[param]).Name() == "string" {
          config.values[param], _ = strconv.ParseBool(config.values[param].(string))
          log.Debugf("----> Type mismatch. converted string to bool: %s = %v (type: %s)", param, config.values[param], reflect.TypeOf(config.values[param]))
        }
      }
    }
  }

  log.Debugf("Done. Final config values: %v", config.values)
  return config, nil
}

// Pass the parameter key and the value will be returned with the proper type
// (if Type is explicitly specified). Only bool and int are converted if the
// value from the command-line is used since these are treated as strings.
// The type resulting from JSON unmarshalling are preserved so, for example,
// Objects in JSON will be returned as type map[string]interface{}.
func (c *Config) Get(key string) interface{} {
  return c.values[key]
}

// This method prints out "Usage:" followed by two aligned columns. The first
// is the switch (including prefix) and the second is the Usage.
// You can optionally provide a string that will be prepended to the output.
func (c *Config) PrintUsage(message string) {
  fmt.Printf("%sUsage: %s [options]\n", os.Args[0], message)

  maxlen := 0
  for param := range c.params {
    if len(param) > maxlen {
      maxlen = len(param)
    }
  }

  for param := range c.params {
    padded := c.GetKeyWithPrefix(param)
    for i := len(padded); i <= maxlen; i++ {
      padded = padded + " "
    }
    def := c.params[param].Default
    if def != nil {
      def = fmt.Sprintf("(default: %v)", def)
    } else {
      def = ""
    }
    fmt.Printf("  %s     %s %s\n", padded, c.params[param].Usage, def)
  }
}
