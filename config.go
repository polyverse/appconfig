// This package enables you to easily define configuration parameters for your
// application and then handles the collection of parameter values automatically
// in the following order, each overriding the previous (if a value is
// specified): (1) default value (in code), (2) configuration file, (3)
// environmental variable and (4) command-line. You can also set properties for
// each parameter (e.g., default value, is required, value type).
//
// Additionally, you can specify the root node in the configuration file which
// will allow the same configuration file to contain configuration data for
// multiple applications and/or configurations.
//
// A full example implementation is available in example/.
package appconfig

import "fmt"
import "os"
import "strings"
import "strconv"
import "reflect"
import "encoding/json"
import "log/syslog"

const default_prefix = "-"

// ParamType is an optional property of the Param struct. If ommitted, there
// is no type-checking of the parameter value.
type ParamType int

// Constants for the ParamType type.
const (
  PARAM_STRING ParamType = iota // Currently a noop
  PARAM_INT         // Converts environmental variables and command-line values from string to int
  PARAM_BOOL        // Converts environmental variables and command-line values from string to bool
  PARAM_OBJECT      // Currently a noop
  PARAM_CONFIG_JSON // Value represents the JSON config file.
  PARAM_CONFIG_NODE // Specifies a different "root node" in the config file.
)

// This is the struct you use to specify the properties of each parameter.
// The `NewConfig(params map[string]Param) (Config, err)` expects you to pass
// an array of this struct with the parameter name being the map index.
//
// None of the fields are required.
type Param struct {
  Type ParamType        // Use if you want explicit type conversion
  Default interface{}   // Default value. If ommited, default value is nil.
  Usage string          // Description of param. Used by `PrintUsage(message string)`
  Required bool         // Is the parameter required? Default is false.
  PrefixOverride string // Override the argument identifying prefix. Default is "-".
}

// This is the object that's returned from NewConfig() and provides methods
// such as `Get(key string) interface{}` (retrieve values) and
// `PrintUsage(message string)`.
type Config struct {
  values map[string]interface{}
  params map[string]Param
}

// This is a helper function that returns the parameter name prepended with
// the proper switch prefix. The default prefix is "-" but you can override that
// with Param.PrefixOverride. Since the prefixes are stripped and the name used
// as the key for the paramters map, this helper function allows you to
// reconstruct the command-line switch.
func (c *Config) GetKeysWithPrefix() map[string]string {
  keys := make(map[string]string)
  for param := range c.params {
    prefix := c.params[param].PrefixOverride
    if prefix == "" {
      prefix = default_prefix
    }
    keys[param] = prefix + param
    //fmt.Printf("keys[param] = %s\n", prefix + param)
  }

  return keys
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
// Create a new `map[string]Param` and then add the parameters you want your
// application to support.
//
// Example:
//
//   params := make(map[string]cnf.Param)
//   params["config"] = simpleconfig.Param{Type:cnf.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
//   params["proxy-addr"] = cnf.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
//   params["statsd_addr"] = cnf.Param{Usage:"StatsD address:port."}
//   config := NewConfig(params)
//
// There are a lot of debug-level messages sent to syslog.
//
// On MacOS, add the following to /etc/asl.conf to capture the debug messages:
//
// ```
// # Rules for /var/log/appconfig.log
// > appconfig.log mode=0640 format=std rotate=seq compress file_max=1M all_max=3M debug=1
// ? [= Sender appconfig] [<= Level debug] file appconfig.log
// ```
func NewConfig(params map[string]Param) (Config, error) {
  logger, err := syslog.New(syslog.LOG_INFO, "appconfig")
  if err != nil {
    return Config{}, err
  }
  config := Config{make(map[string]interface{}),params}

  args := make(map[string]string)

  // Make a map of the environmental variables
  if err := logger.Debug("Checking environmental variables..."); err != nil {
    fmt.Printf("err: %s", err)
  }
  for param := range params {
    val := os.Getenv(param)
    if val != "" {
      args[param] = os.Getenv(param)
      logger.Debug(fmt.Sprintf("----> Found match: %s = %s", param, args[param]))
    }
  }
  logger.Debug(fmt.Sprintf("--> Done. Environmental variables: %v", args))

  // Make a map of the command-line arguments
  logger.Debug(fmt.Sprintf("Processing command-line arguments: %v", os.Args[1:]))

  // enumerate the command-line arguments
  for i := 1; i <= len(os.Args[1:]); i++ {
    // compare each argument with list of supported paramters
    logger.Debug(fmt.Sprintf("--> Process argument: %s", os.Args[i]))
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
          args[arg] = "true" // if value isn't provided, default to "true"
        } else {
          args[arg] = kv[1]
        }
        logger.Debug(fmt.Sprintf("----> Found match: %s = %s", param, args[arg]))
        break
      }
    }
    if !match {
      logger.Debug("----> No match.")
      err := fmt.Errorf("'%s' is not a supported flag.", os.Args[i])
      logger.Err(err.Error()) // send to syslog
      return config, err
    }
  }
  logger.Debug(fmt.Sprintf("--> Done. Environment variables + command-line arguments overrides: %v", args))

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
    logger.Debug(fmt.Sprintf("Reading config file: file = %s, node = %s", configJson, configNode))

    if f, err := os.Open(configJson); err != nil {
      logger.Err(err.Error()) // send to syslog
      return config, err
    } else { // opened file successfully
      jsonParser := json.NewDecoder(f)
      if err := jsonParser.Decode(&jsonvals); err != nil {
        logger.Err(err.Error()) // send to syslog
        return config, err
      }
    }
    logger.Debug(fmt.Sprintf("-->Loaded json config file: %v", jsonvals))

    // If a configNode is specified, then the config file is expected to have
    // more info than needed. Set jsonvals to just the portion we're interested in.
    if configNode != "" {
      jsonvals = jsonvals[configNode].(map[string]interface{})
      logger.Debug(fmt.Sprintf("-->Trimming JSON based on PARAM_CONFIG_NODE = '%s': %v", configNode, jsonvals))
    }
  } else {
    logger.Debug("No configuration file specified.")
  }

  logger.Debug("Finalizing configuration values...")
  for param := range params {
    logger.Debug(fmt.Sprintf("--> Processing param: %s", param))
    if params[param].Default != nil {
      config.values[param] = params[param].Default
      logger.Debug(fmt.Sprintf("----> Setting default: %s = %v (type: %s)", param, params[param].Default, reflect.TypeOf(params[param].Default)))
    } else {
      logger.Debug("----> No default value provided.")
    }
    if jsonvals[param] != nil {
      config.values[param] = jsonvals[param]
      logger.Debug(fmt.Sprintf("----> Config file override: %s = %v (type: %s)", param, jsonvals[param], reflect.TypeOf(jsonvals[param])))
    }
    if args[param] != "" {
      config.values[param] = args[param]
      logger.Debug(fmt.Sprintf("----> Environmental and command-line override: %s = %v (type: %s)", param, args[param], reflect.TypeOf(args[param])))
    }

    if _, ok := config.values[param]; !ok {
      if params[param].Required {
        err := fmt.Errorf("Missing required parameter '%s'.", param)
        logger.Err(err.Error()) // send to syslog
        return config, err
      }
    }

    switch params[param].Type {
      case PARAM_BOOL: {
        if reflect.TypeOf(config.values[param]).Name() == "string" {
          config.values[param], _ = strconv.ParseBool(config.values[param].(string))
          logger.Debug(fmt.Sprintf("----> Type mismatch. converted string to bool: %s = %v (type: %s)", param, config.values[param], reflect.TypeOf(config.values[param])))
        }
      }
      case PARAM_INT: {
        if reflect.TypeOf(config.values[param]).Name() == "string" {
          config.values[param], _ = strconv.Atoi(config.values[param].(string))
          logger.Debug(fmt.Sprintf("----> Type mismatch. converted string to int: %s = %v (type: %s)", param, config.values[param], reflect.TypeOf(config.values[param])))
        }
      }
    }
  }

  logger.Debug(fmt.Sprintf("Done. Final config values: %v", config.values))
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
  fmt.Printf("%sUsage: %s [options]\n\noptions:\n", message, os.Args[0])

  maxlen := 0
  keys := c.GetKeysWithPrefix()
  for key := range keys {
    if len(key) > maxlen {
      maxlen = len(key)
      //fmt.Printf("maxlen is now %v\n", len(key))
    }
  }

  for param := range c.params {
    padded := keys[param]
    for i := len(padded); i <= maxlen; i++ {
      padded = padded + " "
    }
    def := c.params[param].Default
    if def != nil {
      def = fmt.Sprintf("(default: %v)", def)
    } else {
      def = ""
    }
    fmt.Printf("  %s   %s %s\n", padded, c.params[param].Usage, def)
  }
}
