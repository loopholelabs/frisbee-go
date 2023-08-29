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

import { TextDecoder, TextEncoder } from "util";
import {
  Decoder,
  InvalidArrayError,
  InvalidBooleanError,
  InvalidErrorError,
  InvalidFloat32Error,
  InvalidFloat64Error,
  InvalidInt32Error,
  InvalidInt64Error,
  InvalidMapError,
  InvalidStringError,
  InvalidUint16Error,
  InvalidUint32Error,
  InvalidUint64Error,
  InvalidUint8ArrayError,
  InvalidUint8Error,
} from "./decoder";
import { Encoder } from "./encoder";
import { Kind } from "./kind";

window.TextEncoder = TextEncoder;
window.TextDecoder = TextDecoder as typeof window["TextDecoder"];

describe("Decoder", () => {
  it("Can decode Null", () => {
    const encoded = new Encoder().null().bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.null();

    expect(value).toBe(true);
    expect(decoder.length).toBe(0);

    const decodedNext = decoder.null();
    expect(decodedNext).toBe(false);
  });

  it("Can decode Any", () => {
    const encoded = new Encoder().any().bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.any();

    expect(value).toBe(true);
    expect(decoder.length).toBe(0);

    const decodedNext = decoder.any();
    expect(decodedNext).toBe(false);
  });

  it("Can decode true Boolean", () => {
    const expected = true;

    const encoded = new Encoder().boolean(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.boolean();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.boolean()).toThrowError(InvalidBooleanError);
  });

  it("Can decode false Boolean", () => {
    const expected = false;

    const encoded = new Encoder().boolean(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.boolean();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.boolean()).toThrowError(InvalidBooleanError);
  });

  it("Can decode Uint8", () => {
    const expected = 32;

    const encoded = new Encoder().uint8(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.uint8();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.uint8()).toThrowError(InvalidUint8Error);
  });

  it("Can decode Uint16", () => {
    const expected = 1024;

    const encoded = new Encoder().uint16(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.uint16();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.uint16()).toThrowError(InvalidUint16Error);
  });

  it("Can decode Uint32", () => {
    const expected = 4294967290;

    const encoded = new Encoder().uint32(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.uint32();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);
    expect(() => decoder.uint32()).toThrowError(InvalidUint32Error);
  });

  it("Can decode Uint64", () => {
    const expected = 18446744073709551610n;

    const encoded = new Encoder().uint64(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.uint64();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.uint64()).toThrowError(InvalidUint64Error);
  });

  it("Can decode Int32", () => {
    const expected = 2147483647;
    const expectedNegative = -2147483647;

    const encoded = new Encoder().int32(expected).int32(expectedNegative).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.int32();
    const valueNegative = decoder.int32();

    expect(value).toBe(expected);
    expect(valueNegative).toBe(expectedNegative);
    expect(decoder.length).toBe(0);

    expect(() => decoder.int32()).toThrowError(InvalidInt32Error);
  });

  it("Can decode Int64", () => {
    const expected = 9223372036854775807n;
    const expectedNegative = -9223372036854775807n;

    const encoded = new Encoder().int64(expected).int64(expectedNegative).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.int64();
    const valueNegative = decoder.int64();

    expect(value).toBe(expected);
    expect(valueNegative).toBe(expectedNegative);
    expect(decoder.length).toBe(0);

    expect(() => decoder.int64()).toThrowError(InvalidInt64Error);
  });

  it("Can decode Float32", () => {
    const expected = -214648.34432;

    const encoded = new Encoder().float32(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.float32();

    expect(value).toBeCloseTo(expected, 2);
    expect(decoder.length).toBe(0);

    expect(() => decoder.float32()).toThrowError(InvalidFloat32Error);
  });

  it("Can decode Float64", () => {
    const expected = -922337203685.2345;

    const encoded = new Encoder().float64(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.float64();

    expect(value).toBeCloseTo(expected, 4);
    expect(decoder.length).toBe(0);

    expect(() => decoder.float64()).toThrowError(InvalidFloat64Error);
  });

  it("Can decode Array", () => {
    const expected = ["1", "2", "3"];

    const encoder = new Encoder().array(expected.length, Kind.String);
    expected.forEach((el) => {
      encoder.string(el);
    });
    const decoder = new Decoder(encoder.bytes);

    const size = decoder.array(Kind.String);

    expect(size).toBe(expected.length);

    for (let i = 0; i < size; i += 1) {
      const value = decoder.string();

      expect(value).toBe(expected[i]);
    }

    expect(decoder.length).toBe(0);

    expect(() => decoder.array(Kind.String)).toThrowError(InvalidArrayError);
  });

  it("Can decode Map", () => {
    const expected = new Map<string, number>();
    expected.set("1", 1);
    expected.set("2", 2);
    expected.set("3", 3);

    const encoder = new Encoder();
    encoder.map(expected.size, Kind.String, Kind.Uint32);

    expected.forEach((value, key) => {
      encoder.string(key).uint32(value);
    });

    const decoder = new Decoder(encoder.bytes);
    const size = decoder.map(Kind.String, Kind.Uint32);

    expect(size).toBe(expected.size);

    for (let i = 0; i < size; i += 1) {
      const key = decoder.string();
      const value = decoder.uint32();

      expect(expected.get(key.toString())).toBe(value);
    }

    expect(decoder.length).toBe(0);

    expect(() => decoder.map(Kind.String, Kind.Uint32)).toThrowError(
      InvalidMapError
    );
  });

  it("Can decode Uint8Array", () => {
    const expected = new TextEncoder().encode("Test String");

    const encoded = new Encoder().uint8Array(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.uint8Array();

    expect(value.buffer).toEqual(expected.buffer);
    expect(decoder.length).toBe(0);

    expect(() => decoder.uint8Array()).toThrowError(InvalidUint8ArrayError);
  });

  it("Can decode String", () => {
    const expected = "Test String";

    const encoded = new Encoder().string(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.string();

    expect(value).toBe(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.string()).toThrowError(InvalidStringError);
  });

  it("Can decode Error", () => {
    const expected = new Error("Test String");

    const encoded = new Encoder().error(expected).bytes;
    const decoder = new Decoder(encoded);

    const value = decoder.error();

    expect(value).toEqual(expected);
    expect(decoder.length).toBe(0);

    expect(() => decoder.error()).toThrowError(InvalidErrorError);
    expect(() => {
      const encodedWithMissingStringKind = new Encoder().error(expected).bytes;
      const decoderMissingStringKind = new Decoder(
        encodedWithMissingStringKind
      );

      encodedWithMissingStringKind[1] = 999999;

      decoderMissingStringKind.error();
    }).toThrowError(InvalidErrorError);
  });
});
