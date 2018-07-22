# c2cli

CLI to send commands to C2 over gRPC.

Commands, formatted as `command arg1 arg2`, where `key` is a 512-bit value passed in hex format, `id` is an alias as a string (hashed to get 256-bit id), and `topic` is a string:

* newClient: `nc id key`
* removeClient: `rc id`
* newTopicClient: `ntc id topic`
* removeTopicClient: `rtc id topic`
* resetClient: `rsc id` 
* newTopic: `nt topic`
* removeTopic: `rt topic`
* newClientKey: `nck id key`

See `help` for interactive shell commands.
