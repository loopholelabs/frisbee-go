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

import { BOOLEAN_TRUE } from "./encoder";
import { Kind } from "./kind";

/* eslint-disable no-bitwise */

const CONTINUATION = 0x80;
const MAXLEN16 = 3;
const MAXLEN32 = 5;
const MAXLEN64 = 10;

export class InvalidBooleanError extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidBooleanError.prototype);
  }
}

export class InvalidUint8Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidUint8Error.prototype);
  }
}

export class InvalidUint16Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidUint16Error.prototype);
  }
}

export class InvalidUint32Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidUint32Error.prototype);
  }
}

export class InvalidUint64Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidUint64Error.prototype);
  }
}

export class InvalidInt32Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidInt32Error.prototype);
  }
}

export class InvalidInt64Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidInt64Error.prototype);
  }
}

export class InvalidFloat32Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidFloat32Error.prototype);
  }
}

export class InvalidFloat64Error extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidFloat64Error.prototype);
  }
}

export class InvalidArrayError extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidArrayError.prototype);
  }
}

export class InvalidMapError extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidMapError.prototype);
  }
}

export class InvalidUint8ArrayError extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidUint8ArrayError.prototype);
  }
}

export class InvalidStringError extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidStringError.prototype);
  }
}

export class InvalidErrorError extends Error {
  constructor() {
    super();

    Object.setPrototypeOf(this, InvalidErrorError.prototype);
  }
}

export class Decoder {
  #pos = 0;

  constructor(private buf: Uint8Array) {
    this.buf = buf;
  }

  private peek(): number {
    return this.buf[this.#pos];
  }

  private pop(): number {
    const val = this.buf[this.#pos];
    this.#pos += 1;
    return val;
  }

  get length(): number {
    return this.buf.length - this.#pos;
  }

  private varint(maxLen: number, signed = false): bigint {
    let num = 0n;
    let shift = 0n;
    for (let i = 1; i < maxLen + 1; i += 1) {
      const b = this.pop();
      if (b < BigInt(CONTINUATION)) {
        num += BigInt(b) << shift;
        if (signed) {
          // two's complement
          num = num % 2n === 0n ? num / 2n : -(num + 1n) / 2n;
        }
        return num;
      }
      num += BigInt(b & (CONTINUATION - 1)) << shift;
      shift += 7n;
    }
    return num;
  }

  null(): boolean {
    const val = (this.peek() as Kind) === Kind.Null;
    if (val) {
      this.#pos += 1;
    }
    return val;
  }

  any(): boolean {
    const val = (this.peek() as Kind) === Kind.Any;
    if (val) {
      this.#pos += 1;
    }
    return val;
  }

  boolean(): boolean {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Boolean) {
      throw new InvalidBooleanError();
    }
    return this.pop() === BOOLEAN_TRUE;
  }

  uint8(): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Uint8) {
      throw new InvalidUint8Error();
    }
    const dataView = new DataView(
      this.buf.buffer,
      this.buf.byteOffset + this.#pos,
      1
    );
    const val = dataView.getUint8(0);
    this.#pos += 1;
    return val;
  }

  uint16(): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Uint16) {
      throw new InvalidUint16Error();
    }

    let num = 0;
    let shift = 0;
    for (let i = 1; i < MAXLEN16 + 1; i += 1) {
      const b = this.pop();
      if (b < CONTINUATION) {
        return (num | (b << shift)) >>> 0;
      }
      num |= b & ((CONTINUATION - 1) << shift);
      shift += 7;
    }
    return num;
  }

  uint32(): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Uint32) {
      throw new InvalidUint32Error();
    }

    return Number(this.varint(MAXLEN32));
  }

  uint64(): bigint {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Uint64) {
      throw new InvalidUint64Error();
    }

    return this.varint(MAXLEN64);
  }

  int32(): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Int32) {
      throw new InvalidInt32Error();
    }

    return Number(this.varint(MAXLEN32, true));
  }

  int64(): bigint {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Int64) {
      throw new InvalidInt64Error();
    }

    return this.varint(MAXLEN64, true);
  }

  float32(): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Float32) {
      throw new InvalidFloat32Error();
    }
    const dataView = new DataView(
      this.buf.buffer,
      this.buf.byteOffset + this.#pos,
      4
    );
    const val = dataView.getFloat32(0);
    this.#pos += 4;
    return val;
  }

  float64(): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Float64) {
      throw new InvalidFloat64Error();
    }
    const dataView = new DataView(
      this.buf.buffer,
      this.buf.byteOffset + this.#pos,
      8
    );
    const val = dataView.getFloat64(0);
    this.#pos += 8;
    return val;
  }

  array(valueKind: Kind): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Array) {
      throw new InvalidArrayError();
    }

    const definedValueKind = this.pop() as Kind;
    if (!(definedValueKind in Kind) || valueKind !== definedValueKind) {
      throw new InvalidArrayError();
    }

    return this.uint32();
  }

  map(keyKind: Kind, valueKind: Kind): number {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Map) {
      throw new InvalidMapError();
    }

    const definedKeyKind = this.pop() as Kind;
    if (!(definedKeyKind in Kind) || keyKind !== definedKeyKind) {
      throw new InvalidMapError();
    }

    const definedValueKind = this.pop() as Kind;
    if (!(valueKind in Kind) || valueKind !== definedValueKind) {
      throw new InvalidMapError();
    }

    return this.uint32();
  }

  uint8Array(): Uint8Array {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Uint8Array) {
      throw new InvalidUint8ArrayError();
    }

    const size = this.uint32();
    const value = this.buf.slice(this.#pos, this.#pos + size);
    this.#pos += size;
    return value;
  }

  string(): string {
    const kind = this.pop() as Kind;
    if (kind !== Kind.String) {
      throw new InvalidStringError();
    }

    const size = this.uint32();
    const value = new TextDecoder().decode(
      this.buf.slice(this.#pos, this.#pos + size)
    );
    this.#pos += size;
    return value;
  }

  error(): Error {
    const kind = this.pop() as Kind;
    if (kind !== Kind.Error) {
      throw new InvalidErrorError();
    }

    const nestedType = this.pop() as Kind;
    if (nestedType !== Kind.String) {
      throw new InvalidErrorError();
    }

    const size = this.uint32();
    const value = new Error(
      new TextDecoder().decode(this.buf.slice(this.#pos, this.#pos + size))
    );
    this.#pos += size;
    return value;
  }
}
