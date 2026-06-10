package pb

// ProtobufSourceRepository is the protocol definition repository used to generate this package.
const ProtobufSourceRepository = "gaiasec-protobuf"

// ProtobufSourceCommit is the gaiasec-protobuf commit used to generate this package.
const ProtobufSourceCommit = "4eaacf945b00"

func ProtobufVersionString() string {
	return ProtobufSourceRepository + "@" + ProtobufSourceCommit
}
