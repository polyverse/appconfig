### Output
This sample output shows running `$ ./example` and `$ ./example --debug`. The corresponding source code is `main.go`. Performance of this example without the verbose `--debug` output is 94.259µs.

```
$ pwd
/Users/chanaoka/code/go/src/github.com/polyverse-security/appconfig/example
$ go build
$ ./example

The following parameters have been defined:
	param="help", Type=2, Default=false, Usage="print usage.", Required=false, PrefixOverride="--"
	param="config", Type=4, Default=config.json, Usage="json config file.", Required=false, PrefixOverride=""
	param="config-node", Type=5, Default=example, Usage="root node in the config file.", Required=false, PrefixOverride=""
	param="debug", Type=2, Default=false, Usage="verbose output.", Required=false, PrefixOverride="--"
	param="port", Type=0, Default=:8080, Usage="bind-to port.", Required=true, PrefixOverride=""
	param="statsd_addr", Type=0, Default=<nil>, Usage="statsd endpoint.", Required=false, PrefixOverride=""
	param="timeout", Type=1, Default=1000, Usage="server timeout (ms).", Required=false, PrefixOverride=""

*** Calling appconfig.NewConfig()...
*** Done. Elapsed time: 94.259µs

Result:
	param = config, value = config.json, type = string
	param = config-node, value = example, type = string
	param = debug, value = false, type = bool
	param = port, value = :8080, type = string
	param = statsd_addr, value = localhost:8125, type = string
	param = timeout, value = 5000, type = int
	param = help, value = false, type = bool

$ ./example --debug

The following parameters have been defined:
	param="help", Type=2, Default=false, Usage="print usage.", Required=false, PrefixOverride="--"
	param="config", Type=4, Default=config.json, Usage="json config file.", Required=false, PrefixOverride=""
	param="config-node", Type=5, Default=example, Usage="root node in the config file.", Required=false, PrefixOverride=""
	param="debug", Type=2, Default=false, Usage="verbose output.", Required=false, PrefixOverride="--"
	param="port", Type=0, Default=:8080, Usage="bind-to port.", Required=true, PrefixOverride=""
	param="statsd_addr", Type=0, Default=<nil>, Usage="statsd endpoint.", Required=false, PrefixOverride=""
	param="timeout", Type=1, Default=1000, Usage="server timeout (ms).", Required=false, PrefixOverride=""

*** Calling appconfig.NewConfig()...
DEBU[0000] SetLogLevel(): debug
DEBU[0000] Checking environmental variables...
DEBU[0000] ----> Found match: timeout = 5000
DEBU[0000] --> Done. Environmental variables: map[timeout:5000]
DEBU[0000] Processing command-line arguments: [--debug]
DEBU[0000] --> Process argument: --debug
DEBU[0000] ----> Found match: debug = true
DEBU[0000] --> Done. Command-line arguments overrides: map[debug:true]
DEBU[0000] Reading config file: file = 'config.json', node = 'example'
DEBU[0000] --> Loaded JSON config file: map[proxy:map[ProxyRules:map[/(.*?):map[ScriptFile:js/proxy.js RequestHandler:RequestHandler(httpRequest) ResponseHandler:ResponseHandler(httpResponse)]] debug:false proxy-addr::80 remote_addr:http://localhost:8080 statsd_addr:localhost:8125] example:map[debug:false statsd_addr:localhost:8125]]
DEBU[0000] --> Filtering JSON based on PARAM_CONFIG_NODE = 'example': map[debug:false statsd_addr:localhost:8125]
DEBU[0000] Finalizing configuration values...
DEBU[0000] --> Processing param: config
DEBU[0000] ----> Setting default: config = config.json (type: string)
DEBU[0000] --> Processing param: config-node
DEBU[0000] ----> Setting default: config-node = example (type: string)
DEBU[0000] --> Processing param: debug
DEBU[0000] ----> Setting default: debug = false (type: bool)
DEBU[0000] ----> Config file override: debug = false (type: bool)
DEBU[0000] ----> Command-line override: debug = true (type: string)
DEBU[0000] ----> Type mismatch. converted string to bool: debug = true (type: bool)
DEBU[0000] --> Processing param: port
DEBU[0000] ----> Setting default: port = :8080 (type: string)
DEBU[0000] --> Processing param: statsd_addr
DEBU[0000] ----> No default value provided.
DEBU[0000] ----> Config file override: statsd_addr = localhost:8125 (type: string)
DEBU[0000] --> Processing param: timeout
DEBU[0000] ----> Setting default: timeout = 1000 (type: int)
DEBU[0000] ----> Environmental variable override: timeout =  (type: string)
DEBU[0000] ----> Type mismatch. converted string to int: timeout = 5000 (type: int)
DEBU[0000] --> Processing param: help
DEBU[0000] ----> Setting default: help = false (type: bool)
DEBU[0000] Done. Final config values: map[statsd_addr:localhost:8125 timeout:5000 help:false config:config.json config-node:example debug:true port::8080]
*** Done. Elapsed time: 438.929µs

Result:
	param = config, value = config.json, type = string
	param = config-node, value = example, type = string
	param = debug, value = true, type = bool
	param = port, value = :8080, type = string
	param = statsd_addr, value = localhost:8125, type = string
	param = timeout, value = 5000, type = int
	param = help, value = false, type = bool
```
