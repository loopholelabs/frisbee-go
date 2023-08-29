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

import { Kind } from "./kind";

/* eslint-disable no-bitwise */

export const BOOLEAN_FALSE = 0x00;
export const BOOLEAN_TRUE = 0x01;
const CONTINUATION = 0x80;
const PRE_CONTINUATION = 0x7f;
const REST_BYTES = 7;

export class Encoder {
  #pos = 0;

  #buf: Uint8Array;

  constructor(buf?: Uint8Array) {
    if (buf === undefined) {
      this.#buf = new Uint8Array(512);
    } else {
      this.#buf = buf;
    }
  }

  private resize(minLen: number) {
    if (this.#buf.length - this.#pos < minLen) {
      const oldBuf = this.#buf;

      this.#buf = new Uint8Array(this.#buf.buffer.byteLength * 2);
      this.#buf.set(oldBuf);
    }
  }

  get bytes() {
    return new Uint8Array(this.#buf.buffer, 0, this.#pos);
  }

  private varint(value: number, kind: Kind, maxBytes: number, signed = false) {
    let val = value;
    this.resize(maxBytes);
    this.#buf[this.#pos] = kind;
    this.#pos += 1;

    if (signed) {
      // two's complement
      val = value >= 0 ? value * 2 : value * -2 - 1;
    }
    while (val >= CONTINUATION) {
      this.#buf[this.#pos++] = val | CONTINUATION;
      val >>>= REST_BYTES;
    }
    this.#buf[this.#pos++] = val;
    return this;
  }

  private varintBig(
    value: bigint,
    kind: Kind,
    maxBytes: number,
    signed = false
  ) {
    let val = BigInt(value);
    this.resize(maxBytes);
    this.#buf[this.#pos] = kind;
    this.#pos += 1;

    if (signed) {
      // two's complement
      val = val >= 0 ? val * 2n : val * -2n - 1n;
    }
    while (val >= CONTINUATION) {
      this.#buf[this.#pos++] =
        Number(val & BigInt(PRE_CONTINUATION)) | CONTINUATION;
      val >>= BigInt(REST_BYTES);
    }
    this.#buf[this.#pos++] = Number(val);
    return this;
  }

  null() {
    this.resize(1);
    this.#buf[this.#pos] = Kind.Null;
    this.#pos += 1;
    return this;
  }

  any() {
    this.resize(1);
    this.#buf[this.#pos] = Kind.Any;
    this.#pos += 1;
    return this;
  }

  boolean(value: boolean) {
    this.resize(2);
    this.#buf[this.#pos] = Kind.Boolean;
    this.#buf[this.#pos + 1] = value ? BOOLEAN_TRUE : BOOLEAN_FALSE;
    this.#pos += 2;
    return this;
  }

  uint8(value: number) {
    this.resize(2);
    this.#buf[this.#pos] = Kind.Uint8;

    const dataView = new DataView(Uint8Array.from([0]).buffer);
    dataView.setUint8(0, value);
    const bytes = new Uint8Array(dataView.buffer);

    this.#buf.set(bytes, this.#pos + 1);
    this.#pos += bytes.length + 1;
    return this;
  }

  uint16(value: number) {
    return this.varint(value, Kind.Uint16, 3);
  }

  uint32(value: number) {
    return this.varint(value, Kind.Uint32, 5);
  }

  uint64(value: bigint) {
    return this.varintBig(value, Kind.Uint64, 9);
  }

  int32(value: number) {
    return this.varint(value, Kind.Int32, 5, true);
  }

  int64(value: bigint) {
    return this.varintBig(value, Kind.Int64, 9, true);
  }

  float32(value: number) {
    this.resize(5);
    this.#buf[this.#pos] = Kind.Float32;

    const dataView = new DataView(Float32Array.from([0]).buffer);
    dataView.setFloat32(0, value);
    const bytes = new Uint8Array(dataView.buffer);

    this.#buf.set(bytes, this.#pos + 1);
    this.#pos += bytes.length + 1;
    return this;
  }

  float64(value: number) {
    this.resize(9);
    this.#buf[this.#pos] = Kind.Float64;

    const dataView = new DataView(Float64Array.from([0]).buffer);
    dataView.setFloat64(0, value);
    const bytes = new Uint8Array(dataView.buffer);

    this.#buf.set(bytes, this.#pos + 1);
    this.#pos += bytes.length + 1;
    return this;
  }

  array(size: number, valueKind: Kind) {
    this.resize(2);
    this.#buf[this.#pos] = Kind.Array;
    this.#buf[this.#pos + 1] = valueKind;
    this.#pos += 2;
    this.uint32(size);
    return this;
  }

  map(size: number, keyKind: Kind, valueKind: Kind) {
    this.resize(3);
    this.#buf[this.#pos] = Kind.Map;
    this.#buf[this.#pos + 1] = keyKind;
    this.#buf[this.#pos + 2] = valueKind;
    this.#pos += 3;
    this.uint32(size);
    return this;
  }

  uint8Array(value: Uint8Array) {
    this.resize(5 + value.length);
    this.#buf[this.#pos] = Kind.Uint8Array;
    this.#pos += 1;
    this.uint32(value.length);
    this.#buf.set(value, this.#pos);
    this.#pos += value.length;
    return this;
  }

  string(value: string) {
    const v = new TextEncoder().encode(value);

    this.resize(5 + v.length);
    this.#buf[this.#pos] = Kind.String;
    this.#pos += 1;
    this.uint32(v.length);
    this.#buf.set(v, this.#pos);
    this.#pos += v.length;
    return this;
  }

  error(value: Error) {
    const v = new TextEncoder().encode(value.message);

    this.resize(6 + v.length);
    this.#buf[this.#pos] = Kind.Error;
    this.#buf[this.#pos + 1] = Kind.String;
    this.#pos += 2;
    this.uint32(v.length);
    this.#buf.set(v, this.#pos);
    this.#pos += v.length;
    return this;
  }
}
