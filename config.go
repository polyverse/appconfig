// Easily define configuration parameters for your app and this package will
// collect the values in the following order, each overriding the previous
// if a value is provided: (1) code-specified default value, (2) configuration
// file, (3) environmental variables and (4) command-line. A more powerful
// alternative to Go's flag package.
//
// Features:
// - Automatic support beyond command-line arguments (Go's flag package) to configuration files and environmental variables.
// - Configuration files that contain multiple configurations or share configuration data with other apps.
// - Specify whether a parameter is required
// - Specify a type (e.g., int, bool, string) for your parameter
// - Support for unmarshalled JSON objects as parameter values
//
// A full example implementation is available in example/.
//
package appconfig

import "fmt"
import "os"
import "strings"
import "strconv"
import "reflect"
import "encoding/json"
import "github.com/Sirupsen/logrus"

const default_prefix = "-"

// ParamType is an optional property of the Param struct. If ommitted, there
// is no type-checking of the parameter value.
type ParamType int

// Constants for the ParamType type.
const (
  PARAM_STRING ParamType = iota // Converts nil to ""
  PARAM_INT         // Converts environmental variables and command-line values from string to int
  PARAM_BOOL        // Converts environmental variables and command-line values from string to bool
  PARAM_OBJECT      // Currently a noop
  PARAM_CONFIG_JSON // Value represents the JSON config file.
  PARAM_CONFIG_NODE // Specifies a different "root node" in the config file.
  PARAM_USAGE       // Usage flag. Typically -h, -help or --help.
)

// This is the struct you use to specify the properties of each parameter.
// `appconfig.NewConfig(params map[string]Param)` expects you to pass
// an array of this struct with the parameter name being the map index.
//
// None of the fields are required.
type Param struct {
  Type ParamType        // Use if you want explicit type conversion
  Default interface{}   // Default value. If ommited, initialized value is based on Type.
  Usage string          // Description of parameter; used by `PrintUsage(message string)`
  Required bool         // Is the parameter required? Default is false.
  PrefixOverride string // Override the argument identifier prefix. Default is "-".
}

// This is the object that's returned from appconfig.NewConfig(). They key
// methods are:
//   Get(key string) interface{} // returns value of parameter key
//   PrintUsage(message string)   // prints usage with optional preceeding message
type Config struct {
  values map[string]interface{} // use Get() to retreive the values
  params map[string]Param       // NewConfig() constructor values are kept as reference for other Config methods
}

// This is a helper function that returns the parameter name prepended with
// the proper switch prefix. The default prefix is "-" but that might be
// overriden that with Param.PrefixOverride. Since the prefixes are stripped and
// the name used as the key for the paramters map, this helper function allows
// you to reconstruct the command-line switch.
func (c *Config) GetKeysWithPrefix() map[string]string {
  keys := make(map[string]string)
  for param := range c.params {
    prefix := c.params[param].PrefixOverride
    if prefix == "" {  // No PrefixOverride was specified.
      prefix = default_prefix
    }
    keys[param] = prefix + param
  }

  return keys
}

// This is a helper function that returns a string array of all parameter names
// where the Param.Type matches the paramType argument.
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
//   params := make(map[string]appconfig.Param)
//   params["config"] = appconfig.Param{Type:appconfig.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
//   params["proxy-addr"] = appconfig.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
//   params["statsd_addr"] = appconfig.Param{Usage:"StatsD address:port."}
//   config := NewConfig(params)
//
// There are a lot of debug-level messages sent to syslog.
//
// On MacOS, add the following to /etc/asl.conf to capture the debug messages:
//
//   # Rules for /var/log/appconfig.log
//   > appconfig.log mode=0640 format=std rotate=seq compress file_max=1M all_max=3M debug=1
//   ? [= Sender appconfig] [<= Level debug] file appconfig.log
//
func NewConfig(params map[string]Param) (Config, error) {
  logger := logrus.New()
  /*
  if err != nil {
    return Config{}, err
  }
  */

  config := Config{make(map[string]interface{}),params} // initialize the return value

  args := make(map[string]string) // local map to hold environmental and command-line key-value pairs

  // Check to see if environmental variables matching the parameter names exists
  logger.Debug("Checking environmental variables...")
  for param := range params {
    val := os.Getenv(param)
    if val != "" {
      args[param] = val
      logger.Debug(fmt.Sprintf("----> Found match: %s = %s", param, args[param]))
    }
  }
  logger.Debug(fmt.Sprintf("--> Done. Environmental variables: %v", args))

  // Enumerate the command-line arguments
  logger.Debug(fmt.Sprintf("Processing command-line arguments: %v", os.Args[1:]))
  // Compare each argument with list of supported paramters
  for i := 1; i <= len(os.Args[1:]); i++ {
    logger.Debug(fmt.Sprintf("--> Process argument: %s", os.Args[i]))
    match := false // flag to specify whether argument was found in list of supported paramters
    for param := range params {
      kv := strings.Split(os.Args[i], "=") // split the argument into key + value
      prefix := default_prefix
      if params[param].PrefixOverride != "" {
        prefix = params[param].PrefixOverride // prefix override was specified for this parameter. override default prefix.
      }
      arg := strings.TrimPrefix(kv[0], prefix) // strip out the prefix so we can index the map cleanly
      if param == arg {
        // set the kv pair in the args map
        match = true
        if len(kv) == 1 { // split resulted in a key but no value (e.g., "--debug")
          args[arg] = "true" // if value isn't provided, default to true
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
      logger.Errorf(err.Error()) // send to syslog
      return Config{}, err // instead of returning the current config object, let's be more deterministic and return an empty Config struct
    }
  }
  logger.Debug(fmt.Sprintf("--> Done. Environment variables + command-line arguments overrides: %v", args))

  // Usage support
  usageFlags := config.GetParamKeysByType(PARAM_USAGE)
  for i := 0; i < len(usageFlags); i++ { // there should only be 0 or 1 PARAM_USAGE params, but just in case there's more...
    if _, ok := args[usageFlags[i]]; ok { // has a value been provided for this flag
      isTrue, err := strconv.ParseBool(args[usageFlags[i]]) // Environmental variables and command-line arguments are strings. Use ParseBool to account for "true", "TRUE", "1", etc.
      if err != nil {
        logger.Errorf(err.Error()) // send to syslog
        return Config{}, err
      }
      if isTrue {
        config.values[usageFlags[i]] = true // populate the return object with just this value
        logger.Debug(fmt.Sprintf("PARAM_USAGE flag '%s' set to true.", usageFlags[i]))
        return config, nil
      }
    }
  }

  // Find out the config file (if provided)
  configJsonKey := ""
  if len(config.GetParamKeysByType(PARAM_CONFIG_JSON)) > 0 { //TODO: need a more elegant way to do this
    configJsonKey = config.GetParamKeysByType(PARAM_CONFIG_JSON)[0]
  }
  configJson := ""
  if configJsonKey != "" { // check if a parameter of type PARAM_CONFIG_JSON was specified
    if str, ok := args[configJsonKey]; ok {
      configJson = str // string value found in args[] array
    } else {
      if (params[configJsonKey].Default != nil) && (reflect.TypeOf(params[configJsonKey].Default).Kind() == reflect.String) { // nothing found in env or cmd-line; check Default value
        configJson = params[configJsonKey].Default.(string) // safe to assert
      }
    }
  }

  // Find out whether we can use the entire config file or whether we need to filter a node.
  configNodeKey := ""
  if len(config.GetParamKeysByType(PARAM_CONFIG_NODE)) > 0 { //TODO: need a more elegant way to do this
    configNodeKey = config.GetParamKeysByType(PARAM_CONFIG_NODE)[0]
  }
  configNode := ""
  if configNodeKey != "" {  // check if a parameter of type PARAM_CONFIG_NODE was specified
    if str, ok := args[configNodeKey]; ok {
      configNode = str // string value found in args[] array
    } else {
      if (params[configNodeKey].Default != nil) && (reflect.TypeOf(params[configNodeKey].Default).Kind() == reflect.String) { // nothing found in env or cmd-line; check Default value
        configNode = params[configNodeKey].Default.(string) // safe to assert
      }
    }
  }

  configVals := make(map[string]interface{}) // configJson file will be unmarshalled into this map
  if configJson != "" {
    logger.Debug(fmt.Sprintf("Reading config file: file = '%s', node = '%s'", configJson, configNode))

    if f, err := os.Open(configJson); err != nil {
      logger.Errorf(err.Error()) // send to syslog
      return Config{}, err
    } else { // opened file successfully
      jsonParser := json.NewDecoder(f)
      if err := jsonParser.Decode(&configVals); err != nil {
        logger.Errorf(err.Error()) // send to syslog
        return Config{}, err
      }
    }
    logger.Debug(fmt.Sprintf("--> Loaded JSON config file: %v", configVals))

    // If a configNode is specified, then the config file is expected to have
    // more info than needed. Set configVals to just the portion we're interested in.
    if configNode != "" {
      if (configVals[configNode] != nil) && (reflect.TypeOf(configVals[configNode]).String() == "map[string]interface {}") {
        configVals = configVals[configNode].(map[string]interface{}) // safe to assert
        logger.Debug(fmt.Sprintf("--> Filtering JSON based on PARAM_CONFIG_NODE = '%s': %v", configNode, configVals))
      } else {
        err := fmt.Errorf("Node '%s' not found in JSON file '%s'.", configNode, configJson)
        logger.Errorf(err.Error()) // send to syslog
        return Config{}, err
      }
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
    if configVals[param] != nil {
      config.values[param] = configVals[param]
      logger.Debug(fmt.Sprintf("----> Config file override: %s = %v (type: %s)", param, configVals[param], reflect.TypeOf(configVals[param])))
    }
    if args[param] != "" {
      config.values[param] = args[param]
      logger.Debug(fmt.Sprintf("----> Environmental and command-line override: %s = %v (type: %s)", param, args[param], reflect.TypeOf(args[param])))
    }

    if _, ok := config.values[param]; !ok {
      if params[param].Required {
        err := fmt.Errorf("Missing required parameter '%s'.", param)
        logger.Errorf(err.Error()) // send to syslog
        return config, err
      }
      switch params[param].Type {
        case PARAM_STRING, PARAM_CONFIG_JSON, PARAM_CONFIG_NODE: {
          config.values[param] = ""
        }
        case PARAM_INT: {
          config.values[param] = 0
        }
        case PARAM_BOOL, PARAM_USAGE: {
          config.values[param] = false
        }
      }
    }

    if _, ok := config.values[param]; ok {
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
