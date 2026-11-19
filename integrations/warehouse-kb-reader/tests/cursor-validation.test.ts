import test from "node:test";
import assert from "node:assert/strict";
import { CursorCodec } from "../src/cursor/cursor-codec.js";

const SECRET = "test-internal-cursor-secret-key-12345";
const codec = new CursorCodec(SECRET);

const DOC_ID = "01a1900e-ab0b-749c-9808-af69babace6c";
const HASH = "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e";

test("CursorCodec - encodes and decodes valid pagination cursor", () => {
  const cursor = codec.encode({
    doc_id: DOC_ID,
    expected_hash: HASH,
    line_offset: 25,
  });

  assert.equal(typeof cursor, "string");
  assert.ok(cursor.includes("."));

  const decoded = codec.decode(cursor, DOC_ID, HASH);
  assert.ok(decoded);
  assert.equal(decoded.doc_id, DOC_ID);
  assert.equal(decoded.expected_hash, HASH);
  assert.equal(decoded.line_offset, 25);
});

test("CursorCodec - rejects cursor if doc_id does not match", () => {
  const cursor = codec.encode({
    doc_id: DOC_ID,
    expected_hash: HASH,
    line_offset: 25,
  });

  const otherDocId = "01a1900e-ab0d-7aee-90cc-c66d35860036";
  const decoded = codec.decode(cursor, otherDocId, HASH);
  assert.equal(decoded, null);
});

test("CursorCodec - rejects cursor if content hash changed (source changed between pages)", () => {
  const cursor = codec.encode({
    doc_id: DOC_ID,
    expected_hash: HASH,
    line_offset: 25,
  });

  const newHash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
  const decoded = codec.decode(cursor, DOC_ID, newHash);
  assert.equal(decoded, null);
});

test("CursorCodec - rejects tampered cursor signature", () => {
  const cursor = codec.encode({
    doc_id: DOC_ID,
    expected_hash: HASH,
    line_offset: 25,
  });

  const [data, sig] = cursor.split(".");
  const tamperedSig = sig.slice(0, -2) + "xx";
  const tamperedCursor = `${data}.${tamperedSig}`;

  const decoded = codec.decode(tamperedCursor, DOC_ID, HASH);
  assert.equal(decoded, null);
});

test("CursorCodec - rejects corrupted payload data", () => {
  assert.equal(codec.decode("invalid-format", DOC_ID, HASH), null);
  assert.equal(codec.decode("", DOC_ID, HASH), null);
  assert.equal(codec.decode("a.b.c", DOC_ID, HASH), null);
});
