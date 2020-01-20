# Proton

Proton is a library to read and write [protocol buffers](https://developers.google.com/protocol-buffers/) messages. It is still in development and the api is not stable.
If you look for a library to use it in production, have a look at the alternatives in the section "Comparison" below.

# Comparison

Go already has at least two excellent libraries - the official reference implementation [golang/protobuf](https://github.com/golang/protobuf) and the compatible [gogo/protobuf](https://github.com/gogo/protobuf).
Both are actively developed and maintained, but they work in a similar way: eager marshalling and unmarshalling of data based on structs. Both also require supporting runtime libraries for generated code.

Proton strives to enable a low level access to the messages. These are the early days, but it should enable lazy reading or selective scanning for tagged fields, great support for field masks, leaving messages as `[]byte` and other shenanigans like baked in support for well-known types, binary descriptors and the protoc plugin system.

Additionally, it is an experiment in how simple it is to do so.

And it may very well fail.

If it does not, this text will be rewritten and become less whimsy.

# Background

Protobuf supports only 4 low level ways to encode data - all little endian:
* `varint`
* `uint64`
* `uint32`
* `varint` encoded length followed by `[]byte` (in Go parlance)

These are used to encode 18 different types. Most of those are encoded as varints.

`string`, `bytes` and all messages are encoded as `varint` length encoded bytes.
Skipping contained messages that are not needed is entirely possible and reduces parsing effort.
Proton will support that.
