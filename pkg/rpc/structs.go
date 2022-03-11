/*
	Copyright 2022 Loophole Labs

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		   http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package rpc

import (
	"errors"
	"github.com/loopholelabs/frisbee/internal/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strings"
)

const (
	mapOpen  = "map["
	mapClose = "]"
)

var (
	unknownKind        = errors.New("unknown or unsupported protoreflect.Kind")
	unknownCardinality = errors.New("unknown or unsupported protoreflect.Cardinality")
)

var (
	typeLUT = map[protoreflect.Kind]string{
		protoreflect.BoolKind:     "bool",
		protoreflect.Int32Kind:    "int32",
		protoreflect.Sint32Kind:   "int32",
		protoreflect.Uint32Kind:   "uint32",
		protoreflect.Int64Kind:    "int64",
		protoreflect.Sint64Kind:   "int64",
		protoreflect.Uint64Kind:   "uint64",
		protoreflect.Sfixed32Kind: "int32",
		protoreflect.Sfixed64Kind: "int64",
		protoreflect.Fixed32Kind:  "uint32",
		protoreflect.Fixed64Kind:  "uint64",
		protoreflect.FloatKind:    "float32",
		protoreflect.DoubleKind:   "float64",
		protoreflect.StringKind:   "string",
		protoreflect.BytesKind:    "[]byte",
	}

	encodeLUT = map[protoreflect.Kind]string{
		protoreflect.BoolKind:     ".Bool",
		protoreflect.Int32Kind:    ".Int32",
		protoreflect.Sint32Kind:   ".Int32",
		protoreflect.Uint32Kind:   ".Uint32",
		protoreflect.Int64Kind:    ".Int64",
		protoreflect.Sint64Kind:   ".Int64",
		protoreflect.Uint64Kind:   ".Uint64",
		protoreflect.Sfixed32Kind: ".Int32",
		protoreflect.Sfixed64Kind: ".Int64",
		protoreflect.Fixed32Kind:  ".Uint32",
		protoreflect.Fixed64Kind:  ".Uint64",
		protoreflect.StringKind:   ".String",
		protoreflect.FloatKind:    ".Float32",
		protoreflect.DoubleKind:   ".Float64",
		protoreflect.BytesKind:    ".Bytes",
		protoreflect.EnumKind:     ".Uint32",
	}

	decodeLUT = map[protoreflect.Kind]string{
		protoreflect.BoolKind:     ".Bool",
		protoreflect.Int32Kind:    ".Int32",
		protoreflect.Sint32Kind:   ".Int32",
		protoreflect.Uint32Kind:   ".Uint32",
		protoreflect.Int64Kind:    ".Int64",
		protoreflect.Sint64Kind:   ".Int64",
		protoreflect.Uint64Kind:   ".Uint64",
		protoreflect.Sfixed32Kind: ".Int32",
		protoreflect.Sfixed64Kind: ".Int64",
		protoreflect.Fixed32Kind:  ".Uint32",
		protoreflect.Fixed64Kind:  ".Uint64",
		protoreflect.StringKind:   ".String",
		protoreflect.FloatKind:    ".Float32",
		protoreflect.DoubleKind:   ".Float64",
		protoreflect.BytesKind:    ".Bytes",
		protoreflect.EnumKind:     ".Uint32",
	}

	kindLUT = map[protoreflect.Kind]string{
		protoreflect.BoolKind:     "packer.BoolKind",
		protoreflect.Int32Kind:    "packer.Int32Kind",
		protoreflect.Sint32Kind:   "packer.Int32Kind",
		protoreflect.Uint32Kind:   "packer.Uint32Kind",
		protoreflect.Int64Kind:    "packer.Int64Kind",
		protoreflect.Sint64Kind:   "packer.Int64Kind",
		protoreflect.Uint64Kind:   "packer.Uint64Kind",
		protoreflect.Sfixed32Kind: "packer.Int32Kind",
		protoreflect.Sfixed64Kind: "packer.Int64Kind",
		protoreflect.Fixed32Kind:  "packer.Uint32Kind",
		protoreflect.Fixed64Kind:  "packer.Uint64Kind",
		protoreflect.StringKind:   "packer.StringKind",
		protoreflect.FloatKind:    "packer.Float32Kind",
		protoreflect.DoubleKind:   "packer.Float64Kind",
		protoreflect.BytesKind:    "packer.BytesKind",
		protoreflect.EnumKind:     "packer.Uint32Kind",
	}
)

func findValue(field protoreflect.FieldDescriptor) string {
	if kind, ok := typeLUT[field.Kind()]; !ok {
		switch field.Kind() {
		case protoreflect.EnumKind:
			switch field.Cardinality() {
			case protoreflect.Optional, protoreflect.Required:
				return utils.CamelCase(string(field.Enum().FullName()))
			case protoreflect.Repeated:
				return utils.CamelCase(utils.AppendString(slice, string(field.Enum().FullName())))
			default:
				panic(unknownCardinality)
			}
		case protoreflect.MessageKind:
			if field.IsMap() {
				return utils.CamelCase(utils.AppendString(string(field.FullName()), mapSuffix))
			} else {
				switch field.Cardinality() {
				case protoreflect.Optional, protoreflect.Required:
					return utils.AppendString(pointer, utils.CamelCase(string(field.Message().FullName())))
				case protoreflect.Repeated:
					return utils.AppendString(slice, pointer, utils.CamelCase(string(field.Message().FullName())))
				default:
					panic(unknownCardinality)
				}
			}
		default:
			panic(unknownKind)
		}
	} else {
		if field.Cardinality() == protoreflect.Repeated {
			kind = slice + kind
		}
		return kind
	}
}

func writeGetFunc(f File, name string, fields protoreflect.FieldDescriptors) {
	f.P("func New", utils.CamelCase(name), "() *", utils.CamelCase(name), " {")
	f.P(tab, "return &", utils.CamelCase(name), "{")
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.Kind() == protoreflect.MessageKind && field.Cardinality() != protoreflect.Repeated {
			f.P(tab, tab, utils.CamelCase(string(field.Name())), ": New", utils.CamelCase(string(field.Message().FullName())), "(),")
		}
	}
	f.P(tab, "}")
	f.P("}")
	f.P()
}

func writeError(f File, name string) {
	f.P("func (x *", utils.CamelCase(name), ") Error(p *packet.Packet, err error) {")
	f.P(tab, "packer.Encoder(p).Error(err)")
	f.P("}")
	f.P()
}

func writeEncode(f File, name string, fields protoreflect.FieldDescriptors) {
	f.P("func (x *", utils.CamelCase(name), ") Encode(p *packet.Packet) {")
	f.P(tab, "if x == nil {")
	f.P(tab, tab, "packer.Encoder(p).Nil()")
	f.P(tab, "} else if x.error != nil {")
	f.P(tab, tab, "packer.Encoder(p).Error(x.error)")
	f.P(tab, "} else {")
	var messageFields []protoreflect.FieldDescriptor
	var sliceFields []protoreflect.FieldDescriptor
	builder := new(strings.Builder)
	builder.WriteString(".Bool(x.ignore)")
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.Cardinality() == protoreflect.Repeated && !field.IsMap() {
			sliceFields = append(sliceFields, field)
		} else {
			if encoder, ok := encodeLUT[field.Kind()]; !ok {
				switch field.Kind() {
				case protoreflect.MessageKind:
					messageFields = append(messageFields, field)
				default:
					panic(unknownKind)
				}
			} else {
				builder.WriteString(encoder)
				if field.Kind() == protoreflect.EnumKind {
					builder.WriteString("(uint32")
				}
				builder.WriteString("(x.")
				builder.WriteString(utils.CamelCase(string(field.Name())))
				builder.WriteString(")")
				if field.Kind() == protoreflect.EnumKind {
					builder.WriteString(")")
				}
			}
		}
	}
	f.P(tab, tab, "packer.Encoder(p)", builder.String())
	writeEncodeSlices(f, sliceFields)
	writeEncodeMessages(f, messageFields)
	f.P(tab, "}")
	f.P("}")
	f.P()
}

func writeEncodeSlices(f File, sliceFields []protoreflect.FieldDescriptor) {
	for _, field := range sliceFields {
		if encoder, ok := encodeLUT[field.Kind()]; !ok {
			switch field.Kind() {
			case protoreflect.MessageKind:
				f.P(tab, tab, "packer.Encoder(p).Slice(uint32(len(x.", utils.CamelCase(string(field.Name())), ")), ", packerAnyKind, ")")
				f.P(tab, tab, "for _, v := range x.", utils.CamelCase(string(field.Name())), " {")
				f.P(tab, tab, tab, "v.Encode(p)")
			default:
				panic(unknownKind)
			}
		} else {
			f.P(tab, tab, "packer.Encoder(p).Slice(uint32(len(x.", utils.CamelCase(string(field.Name())), ")),", kindLUT[field.Kind()], ")")
			f.P(tab, tab, "for _, v := range x.", utils.CamelCase(string(field.Name())), " {")
			f.P(tab, tab, tab, "packer.Encoder(p)", encoder, "(v)")
		}
		f.P(tab, tab, "}")
	}
}

func writeEncodeMessages(f File, messageFields []protoreflect.FieldDescriptor) {
	for _, field := range messageFields {
		f.P(tab, tab, "x.", utils.CamelCase(string(field.Name())), ".Encode(p)")
	}
}

func writeEncodeMap(f File, field protoreflect.FieldDescriptor) {
	f.P("func (x ", utils.CamelCase(string(field.FullName())), mapSuffix, ") Encode(p *packet.Packet) {")
	f.P(tab, "if x == nil {")
	f.P(tab, tab, "packer.Encoder(p).Nil()")
	f.P(tab, "} else {")
	var keyKind string
	var valKind string
	var ok bool
	if keyKind, ok = kindLUT[field.MapKey().Kind()]; !ok {
		switch field.MapKey().Kind() {
		case protoreflect.MessageKind:
			keyKind = packerAnyKind
		default:
			panic(unknownKind)
		}
	}

	if valKind, ok = kindLUT[field.MapValue().Kind()]; !ok {
		switch field.MapValue().Kind() {
		case protoreflect.MessageKind:
			valKind = packerAnyKind
		default:
			panic(unknownKind)
		}
	}

	f.P(tab, tab, "packer.Encoder(p).Map(uint32(len(x)), ", keyKind, ", ", valKind, ")")
	f.P(tab, tab, "for k, v := range x {")
	if encoder, ok := encodeLUT[field.MapKey().Kind()]; !ok {
		switch field.MapKey().Kind() {
		case protoreflect.MessageKind:
			f.P(tab, tab, tab, "k.Encode(p)")
		default:
			panic(unknownKind)
		}
	} else {
		if field.MapKey().Kind() == protoreflect.EnumKind {
			f.P(tab, tab, tab, "packer.Encoder(p)", encoder, "(uint32(k))")
		} else {
			f.P(tab, tab, tab, "packer.Encoder(p)", encoder, "(k)")
		}
	}

	if encoder, ok := encodeLUT[field.MapValue().Kind()]; !ok {
		switch field.MapValue().Kind() {
		case protoreflect.MessageKind:
			f.P(tab, tab, tab, "v.Encode(p)")
		default:
			panic(unknownKind)
		}
	} else {
		if field.MapValue().Kind() == protoreflect.EnumKind {
			f.P(tab, tab, tab, "packer.Encoder(p)", encoder, "(uint32(v))")
		} else {
			f.P(tab, tab, tab, "packer.Encoder(p)", encoder, "(v)")
		}
	}
	f.P(tab, tab, "}")
	f.P(tab, tab, tab, "")
	f.P(tab, "}")
	f.P("}")
	f.P()
}

func writeDecodeMap(f File, field protoreflect.FieldDescriptor) {
	f.P("func (x ", utils.CamelCase(string(field.FullName())), mapSuffix, ") decode(d *packer.Decoder, size uint32) error {")
	f.P(tab, "if size == 0 {")
	f.P(tab, tab, "return nil")
	f.P(tab, "}")
	f.P(tab, "var k ", findValue(field.MapKey()))
	if field.MapKey().Kind() == protoreflect.EnumKind {
		f.P(tab, tab, "var ", utils.CamelCase(string(field.MapKey().Name())), "Temp uint32")
	}
	f.P(tab, "var v ", findValue(field.MapValue()))
	if field.MapValue().Kind() == protoreflect.EnumKind {
		f.P(tab, tab, "var ", utils.CamelCase(string(field.MapValue().Name())), "Temp uint32")
	}
	f.P(tab, "var err error")
	f.P(tab, "for i := uint32(0); i < size; i++ {")

	if decoder, ok := decodeLUT[field.MapKey().Kind()]; !ok {
		switch field.MapKey().Kind() {
		case protoreflect.MessageKind:
			f.P(tab, tab, "k = New", utils.CamelCase(string(field.MapKey().Message().FullName())), "()")
			f.P(tab, tab, "err = k.decode(d)")
		default:
			panic(unknownKind)
		}
	} else {
		if field.MapKey().Kind() == protoreflect.EnumKind {
			f.P(tab, tab, utils.CamelCase(string(field.MapKey().Name())), "Temp, err = d", decoder, "()")
			f.P(tab, tab, "k = ", findValue(field.MapKey()), "(", utils.CamelCase(string(field.MapKey().Name())), "Temp)")
		} else {
			f.P(tab, tab, "k, err = d", decoder, "()")
		}
	}

	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return err")
	f.P(tab, tab, "}")

	if decoder, ok := decodeLUT[field.MapValue().Kind()]; !ok {
		switch field.MapValue().Kind() {
		case protoreflect.MessageKind:
			f.P(tab, tab, "v = New", utils.CamelCase(string(field.MapValue().Message().FullName())), "()")
			f.P(tab, tab, "err = v.decode(d)")
		default:
			panic(unknownKind)
		}
	} else {
		if field.MapValue().Kind() == protoreflect.EnumKind {
			f.P(tab, tab, utils.CamelCase(string(field.MapValue().Name())), "Temp, err = d", decoder, "()")
			f.P(tab, tab, "v = ", findValue(field.MapValue()), "(", utils.CamelCase(string(field.MapValue().Name())), "Temp)")
		} else {
			f.P(tab, tab, "v, err = d", decoder, "()")
		}
	}

	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return err")
	f.P(tab, tab, "}")

	f.P(tab, tab, "x[k] = v")
	f.P(tab, "}")
	f.P(tab, "return nil")
	f.P("}")
	f.P()
}

func writeDecode(f File, name string) {
	f.P("func (x *", utils.CamelCase(name), ") Decode(p *packet.Packet) error {")
	f.P(tab, "if x == nil {")
	f.P(tab, tab, "return NilDecode")
	f.P(tab, "}")
	f.P(tab, "d := packer.GetDecoder(p)")
	f.P(tab, "return x.decode(d)")
	f.P("}")
	f.P()
}

func writeInternalDecode(f File, name string, fields protoreflect.FieldDescriptors) {
	f.P("func (x *", utils.CamelCase(name), ") decode(d *packer.Decoder) error {")
	f.P(tab, "if d.Nil() {")
	f.P(tab, tab, "return nil")
	f.P(tab, "}")
	f.P(tab, "var err error")
	f.P(tab, "x.error, err = d.Error()")
	f.P(tab, "if err != nil {")
	f.P(tab, tab, "x.ignore, err = d.Bool()")
	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return err")
	f.P(tab, tab, "}")
	var messageFields []protoreflect.FieldDescriptor
	var sliceFields []protoreflect.FieldDescriptor
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.Cardinality() == protoreflect.Repeated && !field.IsMap() {
			sliceFields = append(sliceFields, field)
		} else {
			if decoder, ok := decodeLUT[field.Kind()]; !ok {
				switch field.Kind() {
				case protoreflect.MessageKind:
					messageFields = append(messageFields, field)
				default:
					panic(unknownKind)
				}
			} else {
				if field.Kind() == protoreflect.BytesKind {
					decoder += "(nil)"
				} else {
					decoder += "()"
				}
				if field.Kind() == protoreflect.EnumKind {
					f.P(tab, tab, "var ", utils.CamelCase(string(field.Name())), "Temp uint32")
					f.P(tab, tab, utils.CamelCase(string(field.Name())), "Temp, err = d", decoder)
					f.P(tab, tab, "x.", utils.CamelCase(string(field.Name())), " = ", findValue(field), "(", utils.CamelCase(string(field.Name())), "Temp)")
				} else {
					f.P(tab, tab, "x.", utils.CamelCase(string(field.Name())), ", err = d", decoder)
				}

				f.P(tab, tab, "if err != nil {")
				f.P(tab, tab, tab, "return err")
				f.P(tab, tab, "}")
			}
		}
	}
	if len(sliceFields) > 0 {
		f.P(tab, tab, "var sliceSize uint32")
	}
	for _, field := range sliceFields {
		var kind string
		var ok bool
		if kind, ok = kindLUT[field.Kind()]; !ok {
			switch field.Kind() {
			case protoreflect.MessageKind:
				kind = packerAnyKind
			default:
				panic(unknownKind)
			}
		}
		f.P(tab, tab, "sliceSize, err = d.Slice(", kind, ")")
		f.P(tab, tab, "if err != nil {")
		f.P(tab, tab, tab, "return err")
		f.P(tab, tab, "}")
		f.P(tab, tab, "if uint32(len(x.", utils.CamelCase(string(field.Name())), ")) != sliceSize {")
		f.P(tab, tab, tab, "x.", utils.CamelCase(string(field.Name())), " = make(", findValue(field), ", sliceSize)")
		f.P(tab, tab, "}")
		f.P(tab, tab, "for i := uint32(0); i < sliceSize; i++ {")
		if decoder, ok := decodeLUT[field.Kind()]; !ok {
			switch field.Kind() {
			case protoreflect.MessageKind:
				f.P(tab, tab, tab, "err = x.", utils.CamelCase(string(field.Name())), "[i].decode(d)")
			default:
				panic(unknownKind)
			}
		} else {
			f.P(tab, tab, tab, "x.", utils.CamelCase(string(field.Name())), "[i], err = d", decoder, "()")
		}
		f.P(tab, tab, tab, "if err != nil {")
		f.P(tab, tab, tab, tab, "return err")
		f.P(tab, tab, tab, "}")
		f.P(tab, tab, "}")
	}
	for _, field := range messageFields {
		if field.IsMap() {
			f.P(tab, tab, "if !d.Nil() {")
			var keyKind string
			var valKind string
			var ok bool
			if keyKind, ok = kindLUT[field.MapKey().Kind()]; !ok {
				switch field.Kind() {
				case protoreflect.MessageKind:
					keyKind = packerAnyKind
				default:
					panic(unknownKind)
				}
			}

			if valKind, ok = kindLUT[field.MapValue().Kind()]; !ok {
				switch field.Kind() {
				case protoreflect.MessageKind:
					valKind = packerAnyKind
				default:
					panic(unknownKind)
				}
			}
			f.P(tab, tab, utils.CamelCase(string(field.Name())), "Size, err := d.Map(", keyKind, ", ", valKind, ")")
			f.P(tab, tab, "if err != nil {")
			f.P(tab, tab, tab, "return err")
			f.P(tab, tab, "}")
			f.P(tab, tab, "x.", utils.CamelCase(string(field.Name())), " = New", utils.CamelCase(string(field.FullName())), mapSuffix, "(", utils.CamelCase(string(field.Name())), "Size)")
			f.P(tab, tab, "err = x.", utils.CamelCase(string(field.Name())), ".decode(d, ", utils.CamelCase(string(field.Name())), "Size)")
			f.P(tab, tab, "if err != nil {")
			f.P(tab, tab, tab, "return err")
			f.P(tab, tab, "}")
			f.P(tab, "}")
		} else {
			f.P(tab, tab, "if x.", utils.CamelCase(string(field.Name())), " == nil {")
			f.P(tab, tab, tab, "x.", utils.CamelCase(string(field.Name())), " = New", utils.CamelCase(string(field.Message().FullName())), "()")
			f.P(tab, tab, "}")
			f.P(tab, tab, "err = x.", utils.CamelCase(string(field.Name())), ".decode(d)")
			f.P(tab, tab, "if err != nil {")
			f.P(tab, tab, tab, "return err")
			f.P(tab, tab, "}")
		}
	}
	f.P(tab, "}")
	f.P(tab, "d.Return()")
	f.P(tab, "return nil")
	f.P("}")
	f.P()
}

func writeStructs(f File, name string, fields protoreflect.FieldDescriptors, messages protoreflect.MessageDescriptors) {
	for i := 0; i < messages.Len(); i++ {
		message := messages.Get(i)
		if !message.IsMapEntry() {
			writeStructs(f, string(message.FullName()), message.Fields(), message.Messages())
		}
	}
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.IsMap() {
			mapKeyValue := findValue(field.MapKey())
			mapValueValue := findValue(field.MapValue())
			f.P(typeOpen, utils.CamelCase(string(field.FullName())), mapSuffix, space, mapOpen, mapKeyValue, mapClose, mapValueValue)
			f.P()
			f.P("func New", utils.CamelCase(string(field.FullName())), mapSuffix, "(size uint32) ", mapOpen, mapKeyValue, mapClose, mapValueValue, " {")
			f.P(tab, "return make(", mapOpen, mapKeyValue, mapClose, mapValueValue, ", size)")
			f.P("}")
			f.P()
			writeEncodeMap(f, field)
			writeDecodeMap(f, field)
		}
	}
	f.P(typeOpen, utils.CamelCase(name), structOpen)
	f.P(tab, "error error")
	f.P(tab, "ignore bool")
	f.P()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		value := findValue(field)
		f.P(tab, utils.CamelCase(string(field.Name())), space, value)
	}
	f.P(typeClose)
	f.P()

	writeGetFunc(f, name, fields)
	writeError(f, name)
	writeEncode(f, name, fields)
	writeDecode(f, name)
	writeInternalDecode(f, name, fields)
}
