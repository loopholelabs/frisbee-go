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

import * as fs from "fs/promises";
import * as JSONBigint from "json-bigint";
import { TextDecoder, TextEncoder } from "util";
import { Decoder } from "./decoder";
import { Encoder } from "./encoder";
import { Kind } from "./kind";

window.TextEncoder = TextEncoder;
window.TextDecoder = TextDecoder as typeof window["TextDecoder"];

interface ITestData {
  name: string;
  kind: Kind;
  decodedValue: any;
  encodedValue: Uint8Array;
}

const base64ToUint8Array = (base64: string) => {
  const buf = Buffer.from(base64, "base64");

  return new Uint8Array(
    buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength)
  );
};

describe("Integration test", () => {
  let testData: ITestData[] = [];

  beforeAll(async () => {
    const rawTestData = JSONBigint.parse(
      await fs.readFile("./integration-test-data.json", "utf8")
    );

    testData = (rawTestData as any[]).map((el: any) => ({
      ...el,
      encodedValue: base64ToUint8Array(el.encodedValue),
    }));
  });

  it("Can run the decode tests from the test data", () => {
    testData.forEach((v) => {
      switch (v.kind) {
        case Kind.Null: {
          const decoded = new Decoder(v.encodedValue).null();

          if (v.decodedValue === null) {
            expect(decoded).toBe(true);
          } else {
            expect(decoded).toBe(false);
          }

          return;
        }

        case Kind.Boolean: {
          const decoded = new Decoder(v.encodedValue).boolean();

          expect(decoded).toBe(v.decodedValue);

          return;
        }

        case Kind.Uint8: {
          const decoded = new Decoder(v.encodedValue).uint8();

          expect(decoded).toBe(v.decodedValue);

          return;
        }

        case Kind.Uint16: {
          const decoded = new Decoder(v.encodedValue).uint16();

          expect(decoded).toBe(v.decodedValue);

          return;
        }

        case Kind.Uint32: {
          const decoded = new Decoder(v.encodedValue).uint32();

          expect(decoded).toBe(v.decodedValue);

          return;
        }

        case Kind.Uint64: {
          const decoded = new Decoder(v.encodedValue).uint64();

          expect(decoded).toBe(BigInt(v.decodedValue));

          return;
        }

        case Kind.Int32: {
          const decoded = new Decoder(v.encodedValue).int32();

          expect(decoded).toBe(v.decodedValue);

          return;
        }

        case Kind.Int64: {
          const decoded = new Decoder(v.encodedValue).int64();

          expect(decoded).toBe(BigInt(v.decodedValue));

          return;
        }

        case Kind.Float32: {
          const decoded = new Decoder(v.encodedValue).float32();

          expect(decoded).toBeCloseTo(v.decodedValue, 2);

          return;
        }

        case Kind.Float64: {
          const decoded = new Decoder(v.encodedValue).float64();

          expect(decoded).toBe(parseFloat(v.decodedValue));

          return;
        }

        case Kind.Array: {
          const decoder = new Decoder(v.encodedValue);
          const size = decoder.array(Kind.String);

          expect(size).toBe(v.decodedValue.length);

          for (let i = 0; i < size; i += 1) {
            const value = decoder.string();

            expect(value).toBe(v.decodedValue[i]);
          }

          return;
        }

        case Kind.Map: {
          const decoder = new Decoder(v.encodedValue);
          const size = decoder.map(Kind.String, Kind.Uint32);

          expect(size).toBe(Object.keys(v.decodedValue).length);

          for (let i = 0; i < size; i += 1) {
            const key = decoder.string();
            const value = decoder.uint32();

            expect(v.decodedValue[key.toString()]).toBe(value);
          }

          return;
        }

        case Kind.Uint8Array: {
          const decoded = new Decoder(v.encodedValue).uint8Array();

          expect(decoded).toEqual(base64ToUint8Array(v.decodedValue));

          return;
        }

        case Kind.String: {
          const decoded = new Decoder(v.encodedValue).string();

          expect(decoded).toBe(v.decodedValue);

          return;
        }

        case Kind.Error: {
          const decoded = new Decoder(v.encodedValue).error();

          expect(decoded).toEqual(new Error(v.decodedValue));

          return;
        }

        default:
          throw new Error(
            `Unimplemented decoder for kind ${v.kind} and test ${v.name}`
          );
      }
    });
  });

  it("Can run the encode tests from the test data", () => {
    testData.forEach((v) => {
      switch (v.kind) {
        case Kind.Null: {
          const encoded = new Encoder().null();

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Boolean: {
          const encoded = new Encoder().boolean(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Uint8: {
          const encoded = new Encoder().uint8(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Uint16: {
          const encoded = new Encoder().uint16(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Uint32: {
          const encoded = new Encoder().uint32(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Uint64: {
          const encoded = new Encoder().uint64(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Int32: {
          const encoded = new Encoder().int32(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Int64: {
          const encoded = new Encoder().int64(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Float32: {
          const encoded = new Encoder().float32(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Float64: {
          const encoded = new Encoder().float64(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Array: {
          const encoded = new Encoder().array(
            v.decodedValue.length,
            Kind.String
          );
          v.decodedValue.forEach((el: string) => {
            encoded.string(el);
          });

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Map: {
          const encoded = new Encoder().map(
            Object.keys(v.decodedValue).length,
            Kind.String,
            Kind.Uint32
          );
          Object.entries(v.decodedValue)
            .sort(([prevKey], [currKey]) => prevKey.localeCompare(currKey))
            .forEach(([key, value]: any[]) => {
              encoded.string(key);
              encoded.uint32(value);
            });

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Uint8Array: {
          const encoded = new Encoder().uint8Array(
            base64ToUint8Array(v.decodedValue)
          );

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.String: {
          const encoded = new Encoder().string(v.decodedValue);

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        case Kind.Error: {
          const encoded = new Encoder().error(new Error(v.decodedValue));

          expect(encoded.bytes).toEqual(v.encodedValue);

          return;
        }

        default:
          throw new Error(
            `Unimplemented encoder for kind ${v.kind} and test ${v.name}`
          );
      }
    });
  });
});
