# appconfig
--
    import "github.com/polyverse-security/appconfig"

Specify configuration parameters for your application and this package will grab
the values in the following order, each overriding the previous: (1) default
value (specified in your code), (2) configuration file, (3) environmental
variable and (4) command-line. You can also specify additional properties for
each parameter (e.g., default value, required, type).

Additionally, you can specify the root node of the configuration file which will
allow the same configuration file to contain configuration data for multiple
applications and/or multiple configurations.

A full example implementation is available in example/.

## Usage

#### type Config

```go
type Config struct {
}
```

This is the object that's returned from NewConfig() and provides methods such as
`Get(key string) interface{}` (retrieve values) and `PrintUsage(message
string)`.

#### func  NewConfig

```go
func NewConfig(params map[string]Param) (Config, error)
```
Create a new `map[string]Param` and then add the parameters you want your
application to support.

Example:

    params := make(map[string]cnf.Param)
    params["config"] = simpleconfig.Param{Type:cnf.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
    params["proxy-addr"] = cnf.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
    params["statsd_addr"] = cnf.Param{Usage:"StatsD address:port."}
    config := NewConfig(params)

There are a lot of debug-level messages sent to syslog.

On MacOS, add the following to /etc/asl.conf to capture the debug messages:

``` # Rules for /var/log/appconfig.log > appconfig.log mode=0640 format=std
rotate=seq compress file_max=1M all_max=3M debug=1 ? [= Sender appconfig] [<=
Level debug] file appconfig.log ```

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
proper switch prefix. The default prefix is "-" but you can override that with
Param.PrefixOverride. Since the prefixes are stripped and the name used as the
key for the paramters map, this helper function allows you to reconstruct the
command-line switch.

#### func (*Config) GetParamKeysByType

```go
func (c *Config) GetParamKeysByType(paramType ParamType) []string
```
This is a helper function that returns a string array of all parameter names
where the Param.Type matches what you pass in.

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
	Default        interface{} // Default value. If ommited, default value is nil.
	Usage          string      // Description of param. Used by `PrintUsage(message string)`
	Required       bool        // Is the parameter required? Default is false.
	PrefixOverride string      // Override the argument identifying prefix. Default is "-".
}
```

This is the struct you use to specify the properties of each parameter. The
`NewConfig(params map[string]Param) (Config, err)` expects you to pass an array
of this struct with the parameter name being the map index.

None of the fields are required.

#### type ParamType

```go
type ParamType int
```

ParamType is an optional property of the Param struct. If ommitted, there is no
type-checking of the parameter value.

```go
const (
	PARAM_STRING      ParamType = iota // Currently a noop
	PARAM_INT                          // Converts environmental variables and command-line values from string to int
	PARAM_BOOL                         // Converts environmental variables and command-line values from string to bool
	PARAM_OBJECT                       // Currently a noop
	PARAM_CONFIG_JSON                  // Value represents the JSON config file.
	PARAM_CONFIG_NODE                  // Specifies a different "root node" in the config file.
)
```
Constants for the ParamType type.
