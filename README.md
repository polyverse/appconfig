# DEPRECATION NOTICE

Please note that this repository has been deprecated and is no longer actively maintained by Polyverse Corporation.  It may be removed in the future, but for now remains public for the benefit of any users.

Importantly, as the repository has not been maintained, it may contain unpatched security issues and other critical issues.  Use at your own risk.

While it is not maintained, we would graciously consider any pull requests in accordance with our Individual Contributor License Agreement.  https://github.com/polyverse/contributor-license-agreement

For any other issues, please feel free to contact info@polyverse.com

---

# appconfig
--
    import "github.com/polyverse-security/appconfig"

Easily define configuration parameters for your app and this package will
collect the values in the following order, each overriding the previous if a
value is provided: (1) code-specified default value, (2) configuration file, (3)
environmental variables and (4) command-line. A more powerful alternative to
Go's flag package.

Features: - Automatic support beyond command-line arguments (Go's flag package)
to configuration files and environmental variables. - Configuration files that
contain multiple configurations or share configuration data with other apps. -
Specify whether a parameter is required - Specify a type (e.g., int, bool,
string) for your parameter - Support for unmarshalled JSON objects as parameter
### values

A full example implementation is available in example/.

## Usage

#### type Config

```go
type Config struct {
}
```

This is the object that's returned from appconfig.NewConfig(). They key methods
are:

    Get(key string) interface{} // returns value of parameter key
    PrintUsage(message string)   // prints usage with optional preceeding message

#### func  NewConfig

```go
func NewConfig(params map[string]Param) (Config, error)
```
Create a new `map[string]Param` and then add the parameters you want your
application to support.

Example:

    params := make(map[string]appconfig.Param)
    params["config"] = appconfig.Param{Type:appconfig.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
    params["proxy-addr"] = appconfig.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
    params["statsd_addr"] = appconfig.Param{Usage:"StatsD address:port."}
    config := NewConfig(params)

There are a lot of debug-level messages sent to syslog.

On MacOS, add the following to /etc/asl.conf to capture the debug messages:

    # Rules for /var/log/appconfig.log
    > appconfig.log mode=0640 format=std rotate=seq compress file_max=1M all_max=3M debug=1
    ? [= Sender appconfig] [<= Level debug] file appconfig.log

#### func (*Config) Get

```go
func (c *Config) Get(key string) interface{}
```
Pass the parameter key and the value will be returned with the proper type (if
Type is explicitly specified). Only bool and int are converted if the value from
the command-line is used since these are treated as strings. The type resulting
from JSON unmarshalling are preserved so, for example, Objects in JSON will be
returned as type map[string]interface{}.

#### func (*Config) GetKeysWithPrefix

```go
func (c *Config) GetKeysWithPrefix() map[string]string
```
This is a helper function that returns the parameter name prepended with the
proper switch prefix. The default prefix is "-" but that might be overriden that
with Param.PrefixOverride. Since the prefixes are stripped and the name used as
the key for the paramters map, this helper function allows you to reconstruct
the command-line switch.

#### func (*Config) GetParamKeysByType

```go
func (c *Config) GetParamKeysByType(paramType ParamType) []string
```
This is a helper function that returns a string array of all parameter names
where the Param.Type matches the paramType argument.

#### func (*Config) PrintUsage

```go
func (c *Config) PrintUsage(message string)
```
This method prints out "Usage:" followed by two aligned columns. The first is
the switch (including prefix) and the second is the Usage. You can optionally
provide a string that will be prepended to the output.

#### type Param

```go
type Param struct {
	Type           ParamType   // Use if you want explicit type conversion
	Default        interface{} // Default value. If ommited, initialized value is based on Type.
	Usage          string      // Description of parameter; used by `PrintUsage(message string)`
	Required       bool        // Is the parameter required? Default is false.
	PrefixOverride string      // Override the argument identifier prefix. Default is "-".
}
```

This is the struct you use to specify the properties of each parameter.
`appconfig.NewConfig(params map[string]Param)` expects you to pass an array of
this struct with the parameter name being the map index.

None of the fields are required.

#### type ParamType

```go
type ParamType int
```

ParamType is an optional property of the Param struct. If ommitted, there is no
type-checking of the parameter value.

```go
const (
	PARAM_STRING      ParamType = iota // Converts nil to ""
	PARAM_INT                          // Converts environmental variables and command-line values from string to int
	PARAM_BOOL                         // Converts environmental variables and command-line values from string to bool
	PARAM_OBJECT                       // Currently a noop
	PARAM_CONFIG_JSON                  // Value represents the JSON config file.
	PARAM_CONFIG_NODE                  // Specifies a different "root node" in the config file.
	PARAM_USAGE                        // Usage flag. Typically -h, -help or --help.
)
```
Constants for the ParamType type.
