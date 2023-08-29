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

import { TextEncoder } from "util";
import { Encoder } from "./encoder";
import { Kind } from "./kind";

window.TextEncoder = TextEncoder;

describe("Encoder", () => {
  it("Can encode Null", () => {
    const encoded = new Encoder().null().bytes;

    expect(encoded.length).toBe(1);
    expect(encoded[0]).toBe(Kind.Null);
  });

  it("Can encode Any", () => {
    const encoded = new Encoder().any().bytes;

    expect(encoded.length).toBe(1);
    expect(encoded[0]).toBe(Kind.Any);
  });

  it("Can encode true Boolean", () => {
    const encoded = new Encoder().boolean(true).bytes;

    expect(encoded.length).toBe(2);
    expect(encoded[0]).toBe(Kind.Boolean);
    expect(encoded[1]).toBe(0x01);
  });

  it("Can encode false Boolean", () => {
    const encoded = new Encoder().boolean(false).bytes;

    expect(encoded.length).toBe(2);
    expect(encoded[0]).toBe(Kind.Boolean);
    expect(encoded[1]).toBe(0x00);
  });

  it("Can encode Uint8", () => {
    const encoded = new Encoder().uint8(32).bytes;

    expect(encoded.length).toBe(2);
    expect(encoded[0]).toBe(Kind.Uint8);
    expect(encoded[1]).toBe(32);
  });

  it("Can encode Uint16", () => {
    const encoded = new Encoder().uint16(1024).bytes;

    expect(encoded.length).toBe(3);
    expect(encoded[0]).toBe(Kind.Uint16);
    expect(encoded[1]).toBe(128);
    expect(encoded[2]).toBe(8);
  });

  it("Can encode Uint32", () => {
    const encoded = new Encoder().uint32(4294967290).bytes;

    expect(encoded.length).toBe(6);
    expect(encoded[0]).toBe(Kind.Uint32);
    expect(encoded[1]).toBe(250);
    expect(encoded[2]).toBe(255);
    expect(encoded[3]).toBe(255);
    expect(encoded[4]).toBe(255);
    expect(encoded[5]).toBe(15);
  });

  it("Can encode Uint64", () => {
    const encoded = new Encoder().uint64(18446744073709551610n).bytes;

    expect(encoded.length).toBe(11);
    expect(encoded[0]).toBe(Kind.Uint64);
    expect(encoded[1]).toBe(250);
    expect(encoded[2]).toBe(255);
    expect(encoded[3]).toBe(255);
    expect(encoded[4]).toBe(255);
    expect(encoded[5]).toBe(255);
    expect(encoded[6]).toBe(255);
    expect(encoded[7]).toBe(255);
    expect(encoded[8]).toBe(255);
    expect(encoded[9]).toBe(255);
    expect(encoded[10]).toBe(1);
  });

  it("Can encode Int32", () => {
    const encoded = new Encoder().int32(-2147483648).bytes;

    expect(encoded.length).toBe(6);
    expect(encoded[0]).toBe(Kind.Int32);
    expect(encoded[1]).toBe(255);
    expect(encoded[2]).toBe(255);
    expect(encoded[3]).toBe(255);
    expect(encoded[4]).toBe(255);
    expect(encoded[5]).toBe(15);
  });

  it("Can encode Int64", () => {
    const encoded = new Encoder().int64(-9223372036854775808n).bytes;

    expect(encoded.length).toBe(11);
    expect(encoded[0]).toBe(Kind.Int64);
    expect(encoded[1]).toBe(255);
    expect(encoded[2]).toBe(255);
    expect(encoded[3]).toBe(255);
    expect(encoded[4]).toBe(255);
    expect(encoded[5]).toBe(255);
    expect(encoded[6]).toBe(255);
    expect(encoded[7]).toBe(255);
    expect(encoded[8]).toBe(255);
    expect(encoded[9]).toBe(255);
    expect(encoded[10]).toBe(1);
  });

  it("Can encode Float32", () => {
    const encoded = new Encoder().float32(-214648.34432).bytes;

    expect(encoded.length).toBe(5);
    expect(encoded[0]).toBe(Kind.Float32);
    expect(encoded[1]).toBe(0xc8);
    expect(encoded[2]).toBe(0x51);
    expect(encoded[3]).toBe(0x9e);
    expect(encoded[4]).toBe(0x16);
  });

  it("Can encode Float64", () => {
    const encoded = new Encoder().float64(-922337203685.2345).bytes;

    expect(encoded.length).toBe(9);
    expect(encoded[0]).toBe(Kind.Float64);
    expect(encoded[1]).toBe(0xc2);
    expect(encoded[2]).toBe(0x6a);
    expect(encoded[3]).toBe(0xd7);
    expect(encoded[4]).toBe(0xf2);
    expect(encoded[5]).toBe(0x9a);
    expect(encoded[6]).toBe(0xbc);
    expect(encoded[7]).toBe(0xa7);
    expect(encoded[8]).toBe(0x81);
  });

  it("Can encode Array", () => {
    const encoded = new Encoder().array(32, Kind.String).bytes;

    expect(encoded.length).toBe(1 + 1 + 1 + 1);
    expect(encoded[0]).toBe(Kind.Array);
    expect(encoded[1]).toBe(Kind.String);
    expect(encoded[2]).toBe(Kind.Uint32);
  });

  it("Can encode Map", () => {
    const encoded = new Encoder().map(32, Kind.String, Kind.Uint32).bytes;

    expect(encoded.length).toBe(1 + 1 + 1 + 1 + 1);
    expect(encoded[0]).toBe(Kind.Map);
    expect(encoded[1]).toBe(Kind.String);
    expect(encoded[2]).toBe(Kind.Uint32);
    expect(encoded[3]).toBe(Kind.Uint32);
  });

  it("Can encode Uint8Array", () => {
    const expected = new TextEncoder().encode("Test String");

    const encoded = new Encoder().uint8Array(expected).bytes;

    expect(encoded.length).toBe(1 + 1 + 1 + expected.length);
    expect(encoded[0]).toBe(Kind.Uint8Array);
    expect(encoded.slice(6).buffer).toEqual(expected.buffer);
  });

  it("Can encode String", () => {
    const expected = "Test String";

    const encoded = new Encoder().string(expected).bytes;

    expect(encoded.length).toBe(1 + 1 + 1 + expected.length);
    expect(encoded[0]).toBe(Kind.String);
    expect(encoded.slice(6).buffer).toEqual(
      new TextEncoder().encode(expected).buffer
    );
  });

  it("Can encode Error", () => {
    const expected = new Error("Test String");

    const encoded = new Encoder().error(expected).bytes;

    expect(encoded.length).toBe(1 + 1 + 1 + 1 + expected.message.length);
    expect(encoded[0]).toBe(Kind.Error);
    expect(encoded.slice(6).buffer).toEqual(
      new TextEncoder().encode(expected.message).buffer
    );
  });
});
