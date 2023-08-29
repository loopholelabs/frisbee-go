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

pub enum Kind {
    None = 0x00,
    Array = 0x01,
    Map = 0x02,
    Any = 0x03,
    Bytes = 0x04,
    String = 0x05,
    Error = 0x06,
    Bool = 0x07,
    U8 = 0x08,
    U16 = 0x09,
    U32 = 0x0a,
    U64 = 0x0b,
    I32 = 0x0c,
    I64 = 0x0d,
    F32 = 0x0e,
    F64 = 0x0f,

    Unknown,
}

impl From<u8> for Kind {
    fn from(orig: u8) -> Self {
        match orig {
            0x00 => Kind::None,
            0x01 => Kind::Array,
            0x02 => Kind::Map,
            0x03 => Kind::Any,
            0x04 => Kind::Bytes,
            0x05 => Kind::String,
            0x06 => Kind::Error,
            0x07 => Kind::Bool,
            0x08 => Kind::U8,
            0x09 => Kind::U16,
            0x0a => Kind::U32,
            0x0b => Kind::U64,
            0x0c => Kind::I32,
            0x0d => Kind::I64,
            0x0e => Kind::F32,
            0x0f => Kind::F64,

            _ => Kind::Unknown,
        }
    }
}
