package serializer

import "reflect"

type Serializer interface {
	Unserialize([]byte)([]reflect.Value, error)
	Serialize([]reflect.Value)([]byte, error)
}
