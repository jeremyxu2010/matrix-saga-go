package serializer

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"errors"
	"fmt"
)

type GobSerializer struct {
}

func NewGobSerializer()Serializer{
	return &GobSerializer{}
}

func (s *GobSerializer)Unserialize(payloads []byte)([]reflect.Value, error){
	b := bytes.NewReader(payloads)
	dec := gob.NewDecoder(b)
	args := make([]interface{}, 0)
	err := dec.Decode(&args)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("decode values failed, %v", err))
	}
	values := make([]reflect.Value, 0)
	for _, arg := range args {
		values = append(values, reflect.ValueOf(arg))
	}
	return values, nil
}

func (s *GobSerializer)Serialize(values []reflect.Value)([]byte, error){
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	args := make([]interface{}, 0)
	for _, value := range values {
		args = append(args, value.Interface())
	}
	err := enc.Encode(args)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("encode args failed, %v", err))
	}
	return b.Bytes(), nil
}
