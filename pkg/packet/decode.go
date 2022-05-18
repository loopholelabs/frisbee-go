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

package packet

import (
	"math"

	"github.com/pkg/errors"
)

const (
	emptyString = ""
)

var (
	InvalidSlice   = errors.New("invalid slice encoding")
	InvalidMap     = errors.New("invalid map encoding")
	InvalidBytes   = errors.New("invalid bytes encoding")
	InvalidString  = errors.New("invalid string encoding")
	InvalidError   = errors.New("invalid error encoding")
	InvalidBool    = errors.New("invalid bool encoding")
	InvalidUint8   = errors.New("invalid uint8 encoding")
	InvalidUint16  = errors.New("invalid uint16 encoding")
	InvalidUint32  = errors.New("invalid uint32 encoding")
	InvalidUint64  = errors.New("invalid uint64 encoding")
	InvalidInt32   = errors.New("invalid int32 encoding")
	InvalidInt64   = errors.New("invalid int64 encoding")
	InvalidFloat32 = errors.New("invalid float32 encoding")
	InvalidFloat64 = errors.New("invalid float64 encoding")
)

func decodeNil(b []byte) ([]byte, bool) {
	if len(b) > 0 {
		if b[0] == NilKind[0] {
			return b[1:], true
		}
	}
	return b, false
}

func decodeMap(b []byte, keyKind, valueKind Kind) ([]byte, uint32, error) {
	if len(b) > 2 {
		if b[0] == MapKind[0] && b[1] == keyKind[0] && b[2] == valueKind[0] {
			var size uint32
			var err error
			b, size, err = decodeUint32(b[3:])
			if err != nil {
				return b, 0, InvalidMap
			}
			return b, size, nil
		}
	}
	return b, 0, InvalidMap
}

func decodeSlice(b []byte, kind Kind) ([]byte, uint32, error) {
	if len(b) > 1 {
		if b[0] == SliceKind[0] && b[1] == kind[0] {
			var size uint32
			var err error
			b, size, err = decodeUint32(b[2:])
			if err != nil {
				return b, 0, InvalidSlice
			}
			return b, size, nil
		}
	}
	return b, 0, InvalidSlice
}

func decodeBytes(b, ret []byte) ([]byte, []byte, error) {
	if len(b) > 0 {
		if b[0] == BytesKind[0] {
			var size uint32
			var err error
			b, size, err = decodeUint32(b[1:])
			if err != nil {
				return b, nil, InvalidBytes
			}
			if len(b) > int(size)-1 {
				if len(ret) < int(size) {
					if ret == nil {
						ret = make([]byte, size)
						copy(ret, b[:size])
					} else {
						ret = append(ret[:0], b[:size]...)
					}
				} else {
					copy(ret[0:], b[:size])
				}
				return b[size:], ret, nil
			}
		}
	}
	return b, nil, InvalidBytes
}

func decodeString(b []byte) ([]byte, string, error) {
	if len(b) > 0 {
		if b[0] == StringKind[0] {
			var size uint32
			var err error
			b, size, err = decodeUint32(b[1:])
			if err != nil {
				return b, emptyString, InvalidString
			}
			if len(b) > int(size)-1 {
				return b[size:], string(b[:size]), nil
			}
		}
	}
	return b, emptyString, InvalidString
}

func decodeError(b []byte) ([]byte, error, error) {
	if len(b) > 0 {
		if b[0] == ErrorKind[0] {
			var val string
			var err error
			b, val, err = decodeString(b[1:])
			if err != nil {
				return b, nil, InvalidError
			}
			return b, Error(val), nil
		}
	}
	return b, nil, InvalidError
}

func decodeBool(b []byte) ([]byte, bool, error) {
	if len(b) > 1 {
		if b[0] == BoolKind[0] {
			if b[1] == trueBool {
				return b[2:], true, nil
			} else {
				return b[2:], false, nil
			}
		}
	}
	return b, false, InvalidBool
}

func decodeUint8(b []byte) ([]byte, uint8, error) {
	if len(b) > 1 {
		if b[0] == Uint8Kind[0] {
			return b[2:], b[1], nil
		}
	}
	return b, 0, InvalidUint8
}

func decodeUint16(b []byte) ([]byte, uint16, error) {
	if len(b) > 2 {
		if b[0] == Uint16Kind[0] {
			return b[3:], uint16(b[2]) | uint16(b[1])<<8, nil
		}
	}
	return b, 0, InvalidUint16
}

func decodeUint32(b []byte) ([]byte, uint32, error) {
	if len(b) > 4 {
		if b[0] == Uint32Kind[0] {
			return b[5:], uint32(b[4]) | uint32(b[3])<<8 | uint32(b[2])<<16 | uint32(b[1])<<24, nil
		}
	}
	return b, 0, InvalidUint32
}

func decodeUint64(b []byte) ([]byte, uint64, error) {
	if len(b) > 8 {
		if b[0] == Uint64Kind[0] {
			return b[9:], uint64(b[8]) | uint64(b[7])<<8 | uint64(b[6])<<16 | uint64(b[5])<<24 |
				uint64(b[4])<<32 | uint64(b[3])<<40 | uint64(b[2])<<48 | uint64(b[1])<<56, nil
		}
	}
	return b, 0, InvalidUint64
}

func decodeInt32(b []byte) ([]byte, int32, error) {
	if len(b) > 4 {
		if b[0] == Int32Kind[0] {
			return b[5:], int32(uint32(b[4]) | uint32(b[3])<<8 | uint32(b[2])<<16 | uint32(b[1])<<24), nil
		}
	}
	return b, 0, InvalidInt32
}

func decodeInt64(b []byte) ([]byte, int64, error) {
	if len(b) > 8 {
		if b[0] == Int64Kind[0] {
			return b[9:], int64(uint64(b[8]) | uint64(b[7])<<8 | uint64(b[6])<<16 | uint64(b[5])<<24 |
				uint64(b[4])<<32 | uint64(b[3])<<40 | uint64(b[2])<<48 | uint64(b[1])<<56), nil
		}
	}
	return b, 0, InvalidInt64
}

func decodeFloat32(b []byte) ([]byte, float32, error) {
	if len(b) > 4 {
		if b[0] == Float32Kind[0] {
			return b[5:], math.Float32frombits(uint32(b[4]) | uint32(b[3])<<8 | uint32(b[2])<<16 | uint32(b[1])<<24), nil
		}
	}
	return b, 0, InvalidFloat32
}

func decodeFloat64(b []byte) ([]byte, float64, error) {
	if len(b) > 8 {
		if b[0] == Float64Kind[0] {
			return b[9:], math.Float64frombits(uint64(b[8]) | uint64(b[7])<<8 | uint64(b[6])<<16 | uint64(b[5])<<24 |
				uint64(b[4])<<32 | uint64(b[3])<<40 | uint64(b[2])<<48 | uint64(b[1])<<56), nil
		}
	}
	return b, 0, InvalidFloat64
}
