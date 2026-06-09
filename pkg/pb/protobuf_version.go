package pb

// ProtobufSourceRepository is the protocol definition repository used to generate this package.
const ProtobufSourceRepository = "gaiasec-protobuf"

// ProtobufSourceCommit is the gaiasec-protobuf commit used to generate this package.
const ProtobufSourceCommit = "fcf1ac7ca723-dirty"

func ProtobufVersionString() string {
	return ProtobufSourceRepository + "@" + ProtobufSourceCommit
}
