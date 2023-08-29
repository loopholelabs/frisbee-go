/*
    Copyright 2023 Loophole Labs

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

use crate::kind::Kind;
use byteorder::{BigEndian, ReadBytesExt};
use std::error::Error;
use std::fmt::{Display, Formatter};
use std::io::{Cursor, Read};
use std::str;

#[derive(Debug, PartialEq)]
pub enum DecodingError {
    InvalidNone,
    InvalidArray,
    InvalidMap,
    InvalidBytes,
    InvalidString,
    InvalidError,
    InvalidBool,
    InvalidU8,
    InvalidU16,
    InvalidU32,
    InvalidU64,
    InvalidI32,
    InvalidI64,
    InvalidF32,
    InvalidF64,
    InvalidEnum,
    InvalidStruct,
}

impl Display for DecodingError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{self:?}")
    }
}

impl Error for DecodingError {}

const VARINT_LEN16: u16 = 3;
const VARINT_LEN32: u32 = 5;
const VARINT_LEN64: u64 = 10;
const CONTINUATION: u8 = 0x80;

pub trait Decoder {
    fn decode_none(&mut self) -> bool;
    fn decode_array(&mut self, val_kind: Kind) -> Result<usize, DecodingError>;
    fn decode_map(&mut self, key_kind: Kind, val_kind: Kind) -> Result<usize, DecodingError>;
    fn decode_bytes(&mut self) -> Result<Vec<u8>, DecodingError>;
    fn decode_string(&mut self) -> Result<String, DecodingError>;
    fn decode_error(&mut self) -> Result<Box<dyn Error>, DecodingError>;
    fn decode_bool(&mut self) -> Result<bool, DecodingError>;
    fn decode_u8(&mut self) -> Result<u8, DecodingError>;
    fn decode_u16(&mut self) -> Result<u16, DecodingError>;
    fn decode_u32(&mut self) -> Result<u32, DecodingError>;
    fn decode_u64(&mut self) -> Result<u64, DecodingError>;
    fn decode_i32(&mut self) -> Result<i32, DecodingError>;
    fn decode_i64(&mut self) -> Result<i64, DecodingError>;
    fn decode_f32(&mut self) -> Result<f32, DecodingError>;
    fn decode_f64(&mut self) -> Result<f64, DecodingError>;
}

impl Decoder for Cursor<&mut Vec<u8>> {
    fn decode_none(&mut self) -> bool {
        match self.read_u8() {
            Ok(kind) => {
                if kind == Kind::None as u8 {
                    return true;
                }
                self.set_position(self.position() - 1);
            }
            Err(_) => {}
        }
        false
    }

    fn decode_array(&mut self, val_kind: Kind) -> Result<usize, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidArray)?;
        let defined_val_kind = self.read_u8().ok().ok_or(DecodingError::InvalidArray)?;
        if kind == Kind::Array as u8 && val_kind as u8 == defined_val_kind {
            return match self.decode_u32() {
                Err(err) => Err(err),
                Ok(val) => Ok(val as usize),
            };
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidU32)
    }

    fn decode_map(&mut self, key_kind: Kind, val_kind: Kind) -> Result<usize, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidMap)?;
        let defined_key_kind = self.read_u8().ok().ok_or(DecodingError::InvalidMap)?;
        let defined_val_kind = self.read_u8().ok().ok_or(DecodingError::InvalidMap)?;
        if kind == Kind::Map as u8
            && key_kind as u8 == defined_key_kind
            && val_kind as u8 == defined_val_kind
        {
            return match self.decode_u32() {
                Err(err) => Err(err),
                Ok(val) => Ok(val as usize),
            };
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidMap)
    }

    fn decode_bytes(&mut self) -> Result<Vec<u8>, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidBytes)?;
        if kind == Kind::Bytes as u8 {
            let size = self.decode_u32()? as usize;
            let mut buf = vec![0u8; size];
            self.read_exact(&mut buf)
                .ok()
                .ok_or(DecodingError::InvalidBytes)?;
            return Ok(buf);
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidBytes)
    }

    fn decode_string(&mut self) -> Result<String, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidString)?;

        if kind == Kind::String as u8 {
            let size = self.decode_u32()? as usize;
            let mut str_buf = vec![0u8; size];
            self.read_exact(&mut str_buf)
                .ok()
                .ok_or(DecodingError::InvalidString)?;

            let result = str::from_utf8(&str_buf)
                .ok()
                .ok_or(DecodingError::InvalidString)?;
            return Ok(result.to_owned());
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidString)
    }

    fn decode_error(&mut self) -> Result<Box<dyn Error>, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidError)?;
        let nested_kind = self.read_u8().ok().ok_or(DecodingError::InvalidError)?;
        if kind == Kind::Error as u8 && nested_kind == Kind::String as u8 {
            let size = self.decode_u32()? as usize;
            let mut str_buf = vec![0u8; size];
            self.read_exact(&mut str_buf)
                .ok()
                .ok_or(DecodingError::InvalidError)?;

            let result = str::from_utf8(&str_buf)
                .ok()
                .ok_or(DecodingError::InvalidError)?;
            return Ok(Box::<dyn Error>::from(result.to_owned()));
        }
        self.set_position(self.position() - 2);
        Err(DecodingError::InvalidError)
    }

    fn decode_bool(&mut self) -> Result<bool, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidBool)?;
        if kind == Kind::Bool as u8 {
            let val = self.read_u8().ok().ok_or(DecodingError::InvalidBool)?;
            return Ok(val == 1);
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidBool)
    }

    fn decode_u8(&mut self) -> Result<u8, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidU8)?;
        if kind == Kind::U8 as u8 {
            return self.read_u8().ok().ok_or(DecodingError::InvalidU8);
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidU8)
    }

    fn decode_u16(&mut self) -> Result<u16, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidU16)?;
        if kind == Kind::U16 as u8 {
            let mut x: u16 = 0;
            let mut s: u32 = 0;

            for _ in 0..VARINT_LEN16 {
                let byte = self.read_u8().ok().ok_or(DecodingError::InvalidU16)?;
                if byte < CONTINUATION {
                    return Ok(x | (byte as u16) << s);
                }
                x |= (byte as u16 & ((CONTINUATION as u16) - 1)) << s;
                s += 7;
            }
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidU32)
    }

    fn decode_u32(&mut self) -> Result<u32, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidU32)?;
        if kind == Kind::U32 as u8 {
            let mut x: u32 = 0;
            let mut s: u32 = 0;

            for _ in 0..VARINT_LEN32 {
                let byte = self.read_u8().ok().ok_or(DecodingError::InvalidU32)?;
                if byte < CONTINUATION {
                    return Ok(x | (byte as u32) << s);
                }
                x |= (byte as u32 & ((CONTINUATION as u32) - 1)) << s;
                s += 7;
            }
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidU32)
    }

    fn decode_u64(&mut self) -> Result<u64, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidU64)?;
        if kind == Kind::U64 as u8 {
            let mut x: u64 = 0;
            let mut s: u32 = 0;

            for _ in 0..VARINT_LEN64 {
                let byte = self.read_u8().ok().ok_or(DecodingError::InvalidU64)?;
                if byte < CONTINUATION {
                    return Ok(x | (byte as u64) << s);
                }
                x |= (byte as u64 & ((CONTINUATION as u64) - 1)) << s;
                s += 7;
            }
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidU32)
    }

    fn decode_i32(&mut self) -> Result<i32, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidI32)?;
        if kind == Kind::I32 as u8 {
            let mut ux: u32 = 0;
            let mut s: u32 = 0;

            for _ in 0..VARINT_LEN32 {
                let byte = self.read_u8().ok().ok_or(DecodingError::InvalidI32)?;
                if byte < CONTINUATION {
                    ux |= (byte as u32) << s;
                    let mut x = (ux >> 1) as i32;
                    if ux & 1 != 0 {
                        x = x.wrapping_add(1).wrapping_neg();
                    }
                    return Ok(x);
                }
                ux |= (byte as u32 & ((CONTINUATION as u32) - 1)) << s;
                s += 7;
            }
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidI32)
    }

    fn decode_i64(&mut self) -> Result<i64, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidI64)?;
        if kind == Kind::I64 as u8 {
            let mut ux: u64 = 0;
            let mut s: u32 = 0;

            for _ in 0..VARINT_LEN64 {
                let byte = self.read_u8().ok().ok_or(DecodingError::InvalidI64)?;
                if byte < CONTINUATION {
                    ux |= (byte as u64) << s;
                    let mut x = (ux >> 1) as i64;
                    if ux & 1 != 0 {
                        x = x.wrapping_add(1).wrapping_neg();
                    }
                    return Ok(x);
                }
                ux |= (byte as u64 & ((CONTINUATION as u64) - 1)) << s;
                s += 7;
            }
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidI64)
    }

    fn decode_f32(&mut self) -> Result<f32, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidF32)?;
        if kind == Kind::F32 as u8 {
            return self
                .read_f32::<BigEndian>()
                .ok()
                .ok_or(DecodingError::InvalidF32);
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidF32)
    }

    fn decode_f64(&mut self) -> Result<f64, DecodingError> {
        let kind = self.read_u8().ok().ok_or(DecodingError::InvalidF64)?;
        if kind == Kind::F64 as u8 {
            return self
                .read_f64::<BigEndian>()
                .ok()
                .ok_or(DecodingError::InvalidF64);
        }
        self.set_position(self.position() - 1);
        Err(DecodingError::InvalidF64)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::encoder::Encoder;
    use std::collections::HashMap;

    #[test]
    fn test_decode_nil() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        encoder.encode_none().unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_none();
        assert_eq!(val, true);
        assert_eq!(decoder.get_ref().len() - decoder.position() as usize, 0);
        let next_val = decoder.decode_none();
        assert_eq!(next_val, false);
    }

    #[test]
    fn test_decode_array() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let m = ["1".to_string(), "2".to_string(), "3".to_string()];

        encoder.encode_array(m.len(), Kind::String).unwrap();
        for i in m.clone() {
            encoder.encode_string(&i).unwrap();
        }

        let mut decoder = Cursor::new(encoder.get_mut());
        let size = decoder.decode_array(Kind::String).unwrap() as usize;
        assert_eq!(size, m.len());

        let mut mv: Vec<String> = Vec::with_capacity(size);
        for i in 0..size {
            let val = decoder.decode_string().unwrap();
            mv.push(val.to_string());
            assert_eq!(mv[i], m[i]);
        }
        assert_eq!(mv, m);

        let error = decoder.decode_array(Kind::String).unwrap_err();
        assert_eq!(error, DecodingError::InvalidArray);
    }

    #[test]
    fn test_decode_map() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let mut m: HashMap<String, u32> = HashMap::new();
        m.insert(String::from("1"), 1);
        m.insert(String::from("2"), 2);
        m.insert(String::from("3"), 3);

        encoder
            .encode_map(m.len(), Kind::String, Kind::U32)
            .unwrap();
        for (k, v) in m.clone() {
            encoder.encode_string(&k).unwrap().encode_u32(v).unwrap();
        }

        let mut decoder = Cursor::new(encoder.get_mut());
        let size = decoder.decode_map(Kind::String, Kind::U32).unwrap() as usize;
        assert_eq!(size, m.len());

        let mut mv = HashMap::new();
        for _ in 0..size {
            let k = decoder.decode_string().unwrap();
            let v = decoder.decode_u32().unwrap();
            mv.insert(k, v);
        }
        assert_eq!(mv, m);

        let error = decoder.decode_map(Kind::String, Kind::U32).unwrap_err();
        assert_eq!(error, DecodingError::InvalidMap);
    }

    #[test]
    fn test_decode_bytes() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = "Test String".as_bytes();
        encoder.encode_bytes(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let bytes = decoder.decode_bytes().unwrap();
        assert_eq!(bytes, v);
    }

    #[test]
    fn test_decode_string() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = "Test String".to_string();
        encoder.encode_string(&v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_string().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_string().unwrap_err();
        assert_eq!(error, DecodingError::InvalidString);
    }

    #[test]
    fn test_decode_error() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = "Test String";
        encoder.encode_error(Box::<dyn Error>::from(v)).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_error().unwrap();
        assert_eq!(val.to_string(), v);

        let error = decoder.decode_error().unwrap_err();
        assert_eq!(error, DecodingError::InvalidError);
    }

    #[test]
    fn test_decode_bool() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        encoder.encode_bool(true).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_bool().unwrap();
        assert_eq!(val, true);

        let error = decoder.decode_bool().unwrap_err();
        assert_eq!(error, DecodingError::InvalidBool);
    }

    #[test]
    fn test_decode_u8() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = 32 as u8;
        encoder.encode_u8(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_u8().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_u8().unwrap_err();
        assert_eq!(error, DecodingError::InvalidU8);
    }

    #[test]
    fn test_decode_u16() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = 1024 as u16;
        encoder.encode_u16(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_u16().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_u16().unwrap_err();
        assert_eq!(error, DecodingError::InvalidU16);
    }

    #[test]
    fn test_decode_u32() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = 4294967290 as u32;
        encoder.encode_u32(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_u32().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_u32().unwrap_err();
        assert_eq!(error, DecodingError::InvalidU32);
    }

    #[test]
    fn test_decode_u64() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = 18446744073709551610 as u64;
        encoder.encode_u64(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_u64().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_u64().unwrap_err();
        assert_eq!(error, DecodingError::InvalidU64);
    }

    #[test]
    fn test_decode_i32() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = -2147483648;
        let vneg = -32;
        encoder.encode_i32(v).unwrap();
        encoder.encode_i32(vneg).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_i32().unwrap();
        assert_eq!(val, v);

        let val = decoder.decode_i32().unwrap();
        assert_eq!(val, vneg);

        let error = decoder.decode_i32().unwrap_err();
        assert_eq!(error, DecodingError::InvalidI32);
    }

    #[test]
    fn test_decode_i64() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = -9223372036854775808 as i64;
        encoder.encode_i64(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_i64().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_i64().unwrap_err();
        assert_eq!(error, DecodingError::InvalidI64);
    }

    #[test]
    fn test_decode_f32() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = -2147483.648 as f32;
        encoder.encode_f32(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_f32().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_f32().unwrap_err();
        assert_eq!(error, DecodingError::InvalidF32);
    }

    #[test]
    fn test_decode_f64() {
        let mut encoder = Cursor::new(Vec::with_capacity(512));
        let v = -922337203.477580 as f64;
        encoder.encode_f64(v).unwrap();

        let mut decoder = Cursor::new(encoder.get_mut());
        let val = decoder.decode_f64().unwrap();
        assert_eq!(val, v);

        let error = decoder.decode_f64().unwrap_err();
        assert_eq!(error, DecodingError::InvalidF64);
    }
}
