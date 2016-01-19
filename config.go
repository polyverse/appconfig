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
import (
	log "github.com/Sirupsen/logrus"
)

//var log = logrus.New() // create a global instance of logger

const default_prefix = "-"

// ParamType is an optional property of the Param struct. If ommitted, there
// is no type-checking of the parameter value.
type ParamType int

// Constants for the ParamType type.
const (
	PARAM_STRING            ParamType = iota // Converts nil to ""
	PARAM_INT                                // Converts environmental variables and command-line values from string to int
	PARAM_BOOL                               // Converts environmental variables and command-line values from string to bool
	PARAM_OBJECT                             // Currently a noop
	PARAM_CONFIG_READ_ENV                    //Value represents whether environment variables should be read and used (allows explicit control)
	PARAM_CONFIG_JSON_FILE                   // Value represents the JSON config file.
	PARAM_CONFIG_JSON_STDIN                  // Value represents the JSON input from stdin (standard input)
	PARAM_CONFIG_NODE                        // Specifies a different "root node" in the config file (shared by both json-inputs).
	PARAM_USAGE                              // Usage flag. Typically -h, -help or --help.
)

// This is the struct you use to specify the properties of each parameter.
// `appconfig.NewConfig(params map[string]Param)` expects you to pass
// an array of this struct with the parameter name being the map index.
//
// None of the fields are required.
type Param struct {
	Type           ParamType              // Use if you want explicit type conversion
	Default        interface{}            // Default value. If ommited, initialized value is based on Type.
	Usage          string                 // Description of parameter; used by `PrintUsage(message string)`
	Required       bool                   // Is the parameter required? Default is false.
	PrefixOverride string                 // Override the argument identifier prefix. Default is "-".
	Validate       func(interface{}) bool //Set a function that can validate the parameter upon parsing.
}

// This is the object that's returned from appconfig.NewConfig(). They key
// methods are:
//   Get(key string) interface{} // returns value of parameter key
//   PrintUsage(message string)   // prints usage with optional preceeding message
type Config struct {
	values map[string]interface{} // use Get() to retreive the values
	params map[string]Param       // NewConfig() constructor values are kept as reference for other Config methods
}

// Level type
type Level uint8

const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

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
	config := Config{make(map[string]interface{}), params} // initialize the return value

	// Enumerate the command-line arguments
	args, err := processCommandLine(params)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Errorf("Error processing command-line.")
		os.Exit(1)
	}

	// Before proceeding, let's check for the PARAM_USAGE types and return early if it's set to true
	b, err := isCommandLineUsageTypeTrue(args, &config)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Errorf("Error determining whether usage flag is set.")
		os.Exit(1)
	}
	if b {
		return config, nil // usage flag .value[param]true is set from isCommandLineUsageTypeTrue()
	}

	envs := make(map[string]string)
	if ok, _ := strconv.ParseBool(getPreliminaryConfigValue(config, args, params, PARAM_CONFIG_READ_ENV)); ok {
		var err error
		// Check to see if environmental variables matching the parameter names exists
		envs, err = getValsFromEnvVars(params)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Errorf("Error processing command-line.")
			os.Exit(1)
		}
	}

	configJson := getPreliminaryConfigValue(config, args, params, PARAM_CONFIG_JSON_FILE)
	configNode := getPreliminaryConfigValue(config, args, params, PARAM_CONFIG_NODE)

	configFileVals := make(map[string]interface{}) // configJson file will be unmarshalled into this map
	if configJson != "" {
		log.Debugf("Reading config file: file = '%s', node = '%s'", configJson, configNode)

		if f, err := os.Open(configJson); err != nil {
			log.Errorf(err.Error()) // send to syslog
			os.Exit(1)
		} else { // opened file successfully
			configFileVals = parseJsonFromFile(f, configJson, configNode)
		}
	} else {
		log.Debugf("No configuration file specified.")
	}

	configStdinVals := make(map[string]interface{})                   //ConfigJson from stdin will be unmarshalled into this map
	if ok, _ := strconv.ParseBool(getPreliminaryConfigValue(config, args, params, PARAM_CONFIG_JSON_STDIN)); ok {
		configStdinVals = parseJsonFromFile(os.Stdin, "stdin (standard input)", configNode)
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
		if configFileVals[param] != nil {
			config.values[param] = configFileVals[param]
			log.Debugf("----> Config file override: %s = %v (type: %s)", param, configFileVals[param], reflect.TypeOf(configFileVals[param]))
		}
		if configStdinVals[param] != nil {
			config.values[param] = configStdinVals[param]
			log.Debugf("----> Config stdin (standard input) override: %s = %v (type: %s)", param, configStdinVals[param], reflect.TypeOf(configStdinVals[param]))
		}
		if envs[param] != "" {
			config.values[param] = envs[param]
			log.Debugf("----> Environmental variable override: %s = %v (type: %s)", param, args[param], reflect.TypeOf(args[param]))
		}
		if args[param] != "" {
			config.values[param] = args[param]
			log.Debugf("----> Command-line override: %s = %v (type: %s)", param, args[param], reflect.TypeOf(args[param]))
		}

		if _, ok := config.values[param]; !ok {
			if params[param].Required {
				err := fmt.Errorf("Missing required parameter '%s'.", param)
				log.Errorf(err.Error())
				return config, err
			}
			switch params[param].Type {
			case PARAM_STRING, PARAM_CONFIG_JSON_FILE, PARAM_CONFIG_NODE:
				{
					config.values[param] = ""
				}
			case PARAM_INT:
				{
					config.values[param] = 0
				}
			case PARAM_BOOL, PARAM_USAGE, PARAM_CONFIG_JSON_STDIN, PARAM_CONFIG_READ_ENV:
				{
					config.values[param] = false
				}
			}
		}

		if _, ok := config.values[param]; ok {
			switch params[param].Type {
			case PARAM_BOOL:
				{
					if reflect.TypeOf(config.values[param]).Name() == "string" {
						config.values[param], _ = strconv.ParseBool(config.values[param].(string))
						log.Debugf("----> Type mismatch. converted string to bool: %s = %v (type: %s)", param, config.values[param], reflect.TypeOf(config.values[param]))
					}
				}
			case PARAM_INT:
				{
					if reflect.TypeOf(config.values[param]).Name() == "string" {
						config.values[param], _ = strconv.Atoi(config.values[param].(string))
						log.Debugf("----> Type mismatch. converted string to int: %s = %v (type: %s)", param, config.values[param], reflect.TypeOf(config.values[param]))
					}
				}
			}
		}

		log.Debugf("Validating configuration values against validator functions...")
		if validate := params[param].Validate; validate != nil {
			log.Debugf("----> Validator found for param %s", param)
			if value, ok := config.values[param]; ok {
				log.Debug("----> Validating param %s value %v", param, value)
				if !validate(value) {
					err := fmt.Errorf("Validation failed for param %s with value %v", param, value)
					log.Errorf(err.Error())
					return Config{}, err
				}
			}
		}
	}

	log.Debugf("Done. Final config values: %v", config.values)
	return config, nil
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
		if prefix == "" { // No PrefixOverride was specified.
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

// Pass the parameter key and the value will be returned with the proper type
// (if Type is explicitly specified). Only bool and int are converted if the
// value from the command-line is used since these are treated as strings.
// The type resulting from JSON unmarshalling are preserved so, for example,
// Objects in JSON will be returned as type map[string]interface{}.
func (c *Config) Get(key string) interface{} {
	return c.values[key]
}

func (c *Config) GetInt(key string) int {
	if reflect.TypeOf(c.values[key]).String() == "int" {
		return c.values[key].(int)
	} else {
		return 0
	}
}

func (c *Config) GetBool(key string) bool {
	if reflect.TypeOf(c.values[key]).String() == "bool" {
		return c.values[key].(bool)
	} else {
		return false
	}
}

func (c *Config) GetString(key string) string {
	if reflect.TypeOf(c.values[key]).String() == "string" {
		return c.values[key].(string)
	} else {
		return ""
	}
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

// SetLevel sets the standard logger level.
func SetLogLevel(level Level) {
	log.SetLevel(log.Level(level))
	log.Debugf("SetLogLevel(): %s", log.GetLevel().String())
}

func processCommandLine(params map[string]Param) (map[string]string, error) {
	args := make(map[string]string) // local map to hold environmental and command-line key-value pairs

	log.Debugf("Processing command-line arguments: %v", os.Args[1:])
	// Compare each argument with list of supported paramters
	for i := 1; i <= len(os.Args[1:]); i++ {
		log.Debugf("--> Process argument: %s", os.Args[i])
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
				log.Debugf("----> Found match: %s = %s", param, args[arg])
				break
			}
		}
		if !match {
			log.Debugf("----> No match.")
			err := fmt.Errorf("'%s' is not a supported flag.", os.Args[i])
			log.Errorf(err.Error()) // send to syslog
			return nil, err         // instead of returning the current config object, let's be more deterministic and return an empty Config struct
		}
	}

	log.Debugf("--> Done. Command-line arguments overrides: %v", args)

	return args, nil
}

func getValsFromEnvVars(params map[string]Param) (map[string]string, error) {
	envs := make(map[string]string)

	log.Debugf("Checking environmental variables...")

	for param := range params {
		val := os.Getenv(param)
		if val != "" {
			envs[param] = val
			log.Debugf("----> Found match: %s = %s", param, envs[param])
		}
	}

	log.Debugf("--> Done. Environmental variables: %v", envs)

	return envs, nil
}

func isCommandLineUsageTypeTrue(args map[string]string, config *Config) (bool, error) {
	log.Debugf("Checking command-line for usage switch...")
	// Usage support
	usageFlags := config.GetParamKeysByType(PARAM_USAGE)
	for i := 0; i < len(usageFlags); i++ { // there should only be 0 or 1 PARAM_USAGE params, but just in case there's more...
		if _, ok := args[usageFlags[i]]; ok { // has a value been provided for this flag
			isTrue, err := strconv.ParseBool(args[usageFlags[i]]) // Environmental variables and command-line arguments are strings. Use ParseBool to account for "true", "TRUE", "1", etc.
			if err != nil {
				return false, err
			}
			if isTrue {
				log.Debugf("--> Usage flag '%s' set to true.", usageFlags[i])
				config.values[usageFlags[i]] = true // set config.value[]
				return true, nil
			}
		}
	}
	log.Debugf("--> Usage flag is not set to true.")
	return false, nil
}

func GetBoolFromCommandLine(param string, params map[string]Param) bool {
	args, err := processCommandLine(params)
	if err != nil {
		return false
	}
	if val, ok := args[param]; ok {
		if val != "" {
			b, _ := strconv.ParseBool(val)
			return b
		}
	}

	return false
}

func parseJsonFromFile(f *os.File, configFileName string, configNode string) map[string]interface{} {
	if f == nil {
		log.Errorf("Json input from file/stdin was specified, but file descriptor was nil.")
		os.Exit(1)
	}

	config := make(map[string]interface{})

	jsonParser := json.NewDecoder(f)
	if err := jsonParser.Decode(&config); err != nil {
		log.Errorf(err.Error()) // send to syslog
		os.Exit(1)
	}
	log.Debugf("--> Loaded JSON config file: %v", configFileName)

	// If a configNode is specified, then the config file is expected to have
	// more info than needed. Set configVals to just the portion we're interested in.
	if configNode != "" {
		if (config[configNode] != nil) && (reflect.TypeOf(config[configNode]).String() == "map[string]interface {}") {
			config = config[configNode].(map[string]interface{}) // safe to assert
			log.Debugf("--> Filtering JSON based on PARAM_CONFIG_NODE = '%s': %v", configNode, config)
		} else {
			err := fmt.Errorf("Node '%s' not found in JSON file '%s'.", configNode, configFileName)
			log.Errorf(err.Error())
			os.Exit(1)
		}
	}

	return config

}

func getPreliminaryConfigValue(config Config, args map[string]string, params map[string]Param, configKeyType ParamType) string {
	// Reset the root node in the config file to a child node, if necessary
	configKey := ""
	if len(config.GetParamKeysByType(configKeyType)) > 0 { //TODO: need a more elegant way to do this
		configKey = config.GetParamKeysByType(configKeyType)[0]
	}
	configValue := ""
	if configKey != "" { // check if a parameter of type PARAM_CONFIG_NODE was specified
		if str, ok := args[configKey]; ok {
			configValue = str // string value found in args[] array
		} else {
			if (params[configKey].Default != nil) && (reflect.TypeOf(params[configKey].Default).Kind() == reflect.String) { // nothing found in env or cmd-line; check Default value
				configValue = params[configKey].Default.(string) // safe to assert
			}
		}
	}
	return configValue
}
