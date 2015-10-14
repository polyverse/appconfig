# simpleconfig
--
    import "github.com/polyverse-security/gouava/simpleconfig"

This package makes it easy to define paramaters that your application accepts
and supports default values and usage like Go's flag package but also allows you
to specify type and whether the parameter is required.

Once you define the paramters, this package will take the default value you
specified and then override that value in the following order: 1. JSON
configuration file 2. environmental variables 3. command-line

Additionally, you can specify the root node of the configuration file which will
allow the same configuration file to contain configuration data for multiple
applications.

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
Create a new map[string]Param and then populate it with the parameters you want
your application to accept.

Example:

    ```
    params := make(map[string]cnf.Param)
    params["config"] = simpleconfig.Param{Type:cnf.PARAM_CONFIG_JSON, Default:"polyverse.json", Usage:"JSON configuration file.", Required:false}
    params["proxy-addr"] = cnf.Param{Default:":8080", Usage:"List to [address]:port.", Required:true}
    params["statsd_addr"] = cnf.Param{Usage:"StatsD address:port."}
    config := NewConfig(params)
    ```

There are a lot of `logrus.Debugf()` statements that will be output if loglevel
is set to `log.DebugLevel`. The debug output will tell you exactly where the
values are taken from.

#### func (*Config) Get

```go
func (c *Config) Get(key string) interface{}
```
Pass the parameter key and the value will be returned with the proper type (if
Type is explicitly specified). Only bool and int are converted if the value from
the command-line is used since these are treated as strings. The type resulting
from JSON unmarshalling are preserved so, for example, Objects in JSON will be
returned as type map[string]interface{}.

#### func (*Config) GetKeyWithPrefix

```go
func (c *Config) GetKeyWithPrefix(key string) string
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
	Type           ParamType   // if ommited, default is 0 an no type-checking will be performed.
	Default        interface{} // if ommited, the default value is nil.
	Usage          string      // if ommitted, no usage will be provided for this parameter when calling `PrintUsage(message string)`
	Required       bool        // if ommitted, default is false
	PrefixOverride string      // if ommitted, default is "-"
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
	PARAM_STRING ParamType = iota
	PARAM_INT
	PARAM_BOOL
	PARAM_OBJECT
	PARAM_CONFIG_JSON
	PARAM_CONFIG_NODE
)
```
Constants for the ParamType type.
