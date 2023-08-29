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

#[cfg(test)]
mod tests {
    use crate::decoder::Decoder;
    use crate::encoder::Encoder;
    use crate::kind::Kind;

    use base64::{engine::general_purpose, Engine as _};
    use serde::Deserialize;
    use serde_json::Value;
    use std::error::Error;
    use std::fs;
    use std::io::Cursor;

    #[derive(Debug, Deserialize)]
    struct RawTestData {
        name: String,
        kind: u8,
        #[serde(rename = "encodedValue")]
        encoded_value: String,
        #[serde(rename = "decodedValue")]
        decoded_value: Value,
    }

    struct TestData {
        name: String,
        kind: Kind,
        decoded_value: Value,
        encoded_value: Vec<u8>,
    }

    fn get_test_data() -> Vec<TestData> {
        let test_data = fs::read("./integration-test-data.json").unwrap();
        return serde_json::from_slice::<Vec<RawTestData>>(test_data.as_ref())
            .unwrap()
            .into_iter()
            .map(|td| {
                return TestData {
                    name: td.name,
                    kind: Kind::from(td.kind),
                    decoded_value: td.decoded_value,
                    encoded_value: general_purpose::STANDARD.decode(td.encoded_value).unwrap(),
                };
            })
            .collect::<Vec<TestData>>();
    }

    #[test]
    fn test_decode() {
        let test_data = get_test_data();

        for mut td in test_data {
            let mut decoder = Cursor::new(td.encoded_value.as_mut());

            match td.kind {
                Kind::None => {
                    let val = decoder.decode_none();

                    if td.decoded_value.is_null() {
                        assert_eq!(val, true)
                    } else {
                        assert_eq!(val, false)
                    }
                }

                Kind::Bool => {
                    let val = decoder.decode_bool().unwrap();

                    assert_eq!(val, td.decoded_value.as_bool().unwrap());
                }

                Kind::U8 => {
                    let val = decoder.decode_u8().unwrap();

                    assert_eq!(val as u64, td.decoded_value.as_u64().unwrap());
                }

                Kind::U16 => {
                    let val = decoder.decode_u16().unwrap();

                    assert_eq!(val as u64, td.decoded_value.as_u64().unwrap());
                }

                Kind::U32 => {
                    let val = decoder.decode_u32().unwrap();

                    assert_eq!(val as u64, td.decoded_value.as_u64().unwrap());
                }

                Kind::U64 => {
                    let val = decoder.decode_u64().unwrap();

                    assert_eq!(val as u64, td.decoded_value.as_u64().unwrap());
                }

                Kind::I32 => {
                    let val = decoder.decode_i32().unwrap();

                    assert_eq!(val as i64, td.decoded_value.as_i64().unwrap());
                }

                Kind::I64 => {
                    let val = decoder.decode_i64().unwrap();

                    assert_eq!(val as i64, td.decoded_value.as_i64().unwrap());
                }

                Kind::F32 => {
                    let val = decoder.decode_f32().unwrap();

                    assert!(
                        (val as f32 - td.decoded_value.as_f64().unwrap() as f32) < f32::EPSILON
                    );
                }

                Kind::F64 => {
                    let val = decoder.decode_f64().unwrap();

                    assert!(
                        (val as f64 - td.decoded_value.as_f64().unwrap() as f64) < f64::EPSILON
                    );
                }

                Kind::Array => {
                    let len = decoder.decode_array(Kind::String).unwrap();

                    let expected = td.decoded_value.as_array().unwrap();

                    assert_eq!(expected.len(), len);

                    for (i, _) in expected.into_iter().enumerate() {
                        assert_eq!(
                            expected[i].as_str().unwrap(),
                            decoder.decode_string().unwrap()
                        )
                    }
                }

                Kind::Map => {
                    let len = decoder.decode_map(Kind::String, Kind::U32).unwrap();

                    let expected = td.decoded_value.as_object().unwrap();

                    assert_eq!(expected.len(), len);

                    for (expected_key, expected_value) in expected {
                        let actual_key = decoder.decode_string().unwrap();
                        let actual_value = decoder.decode_u32().unwrap();

                        assert_eq!(expected_key.as_str(), actual_key);
                        assert_eq!(expected_value.as_u64().unwrap(), actual_value as u64);
                    }
                }

                Kind::Bytes => {
                    let val = decoder.decode_bytes().unwrap();

                    assert_eq!(
                        val,
                        general_purpose::STANDARD
                            .decode(td.decoded_value.as_str().unwrap())
                            .unwrap()
                    );
                }

                Kind::String => {
                    let val = decoder.decode_string().unwrap();

                    assert_eq!(val, td.decoded_value.as_str().unwrap());
                }

                Kind::Error => {
                    let val = decoder.decode_error().unwrap();

                    assert_eq!(val.to_string(), td.decoded_value.as_str().unwrap());
                }

                _ => panic!("Unimplemented decoder for test {}", td.name),
            }
        }
    }

    #[test]
    fn test_encode() {
        let test_data = get_test_data();

        for td in test_data {
            let mut encoder = Cursor::new(Vec::with_capacity(512));

            match td.kind {
                Kind::None => {
                    let val = encoder.encode_none().unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::Bool => {
                    let val = encoder
                        .encode_bool(td.decoded_value.as_bool().unwrap())
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::U8 => {
                    let val = encoder
                        .encode_u8(td.decoded_value.as_u64().unwrap() as u8)
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::U16 => {
                    let val = encoder
                        .encode_u16(td.decoded_value.as_u64().unwrap() as u16)
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::U32 => {
                    let val = encoder
                        .encode_u32(td.decoded_value.as_u64().unwrap() as u32)
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::U64 => {
                    let val = encoder
                        .encode_u64(td.decoded_value.as_u64().unwrap())
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::I32 => {
                    let val = encoder
                        .encode_i32(td.decoded_value.as_i64().unwrap() as i32)
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::I64 => {
                    let val = encoder
                        .encode_i64(td.decoded_value.as_i64().unwrap())
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::F32 => {
                    let val = encoder
                        .encode_f32(td.decoded_value.as_f64().unwrap() as f32)
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::F64 => {
                    let val = encoder
                        .encode_f64(td.decoded_value.as_f64().unwrap())
                        .unwrap();

                    for (i, expected) in (&td.encoded_value).into_iter().enumerate() {
                        // Ignore last byte; 64-bit float precision
                        if i < td.encoded_value.len() - 1 {
                            assert_eq!(*expected, val.get_ref()[i])
                        }
                    }
                }

                Kind::Array => {
                    let mut val = encoder
                        .encode_array(td.decoded_value.as_array().unwrap().len(), Kind::String)
                        .unwrap();

                    let expected = td.decoded_value.as_array().unwrap();

                    for el in expected.into_iter() {
                        val = val.encode_str(el.as_str().unwrap()).unwrap();
                    }

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::Map => {
                    let mut val = encoder
                        .encode_map(
                            td.decoded_value.as_object().unwrap().len(),
                            Kind::String,
                            Kind::U32,
                        )
                        .unwrap();

                    let expected = td.decoded_value.as_object().unwrap();

                    for (expected_key, expected_value) in expected {
                        val = val
                            .encode_string(&expected_key)
                            .unwrap()
                            .encode_u32(expected_value.as_u64().unwrap() as u32)
                            .unwrap();
                    }

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::Bytes => {
                    let val = encoder
                        .encode_bytes(
                            &general_purpose::STANDARD
                                .decode(td.decoded_value.as_str().unwrap())
                                .unwrap(),
                        )
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::String => {
                    let val = encoder
                        .encode_str(td.decoded_value.as_str().unwrap())
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                Kind::Error => {
                    let val = encoder
                        .encode_error(Box::<dyn Error>::from(td.decoded_value.as_str().unwrap()))
                        .unwrap();

                    assert_eq!(*val.get_ref(), td.encoded_value);
                }

                _ => panic!("Unimplemented decoder for test {}", td.name),
            }
        }
    }
}
