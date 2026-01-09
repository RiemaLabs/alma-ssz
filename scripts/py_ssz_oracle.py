#!/usr/bin/env python3

import binascii
import json
import os
import sys

from ssz.exceptions import DeserializationError
from eth_utils.toolz import partition
from ssz.constants import CHUNK_SIZE
from ssz.sedes import (
    Bitlist,
    Bitvector,
    Boolean,
    ByteList,
    List,
    Serializable,
    Vector,
    boolean,
    bytes32,
    uint8,
    uint64,
)
from ssz.sedes.basic import BasicSedes
from ssz.sedes.bitlist import get_bitlist_len
from ssz.utils import merkleize, mix_in_length, pack, pack_bits


BUG_ID = os.getenv("ALMA_PSSZ_BUG", "").strip()
SCHEMA_VALIDATION_CASES = {"PSSZ-111", "PSSZ-112", "PSSZ-116"}


class BuggyBoolean(Boolean):
    def deserialize(self, data: bytes) -> bool:
        if len(data) != 1:
            raise DeserializationError(f"Invalid serialized boolean length: {len(data)}")
        return data != b"\x00"


class StrictBitvector(Bitvector):
    def deserialize(self, data: bytes):
        byte_len = (self.bit_count + 7) // 8
        if len(data) != byte_len:
            raise DeserializationError(
                f"Invalid serialized bitvector length: {len(data)}"
            )
        if self.bit_count % 8 != 0:
            pad_mask = 0xFF << (self.bit_count % 8)
            if data[-1] & pad_mask:
                raise DeserializationError("Dirty padding bits in bitvector")
        return super().deserialize(data)


class BuggyBitlist(Bitlist):
    def deserialize(self, data: bytes):
        if len(data) == 0:
            return tuple()
        as_integer = int.from_bytes(data, "little")
        len_value = get_bitlist_len(as_integer)
        if len_value < 0:
            len_value = 0
        if len_value > self.max_bit_count:
            raise DeserializationError(
                f"Cannot deserialize length {len_value} bytes data as "
                f"Bitlist[{self.max_bit_count}]"
            )
        bits = []
        for bit_index in range(len_value):
            bits.append(bool((data[bit_index // 8] >> bit_index % 8) % 2))
        return tuple(bits)


class BuggyListNoMixIn(List):
    def get_hash_tree_root(self, value) -> bytes:
        if isinstance(self.element_sedes, BasicSedes):
            serialized_items = tuple(
                self.element_sedes.serialize(element) for element in value
            )
            merkle_leaves = pack(serialized_items)
        else:
            merkle_leaves = tuple(
                self.element_sedes.get_hash_tree_root(element) for element in value
            )
        return merkleize(merkle_leaves, limit=self.chunk_count)


class BuggyBitlistNoMixIn(Bitlist):
    def get_hash_tree_root(self, value) -> bytes:
        return merkleize(pack_bits(value), limit=self.chunk_count)


class BuggyBitlistNoLimit(Bitlist):
    def get_hash_tree_root(self, value) -> bytes:
        return mix_in_length(merkleize(pack_bits(value)), len(value))


class BuggyListNoLimit(List):
    def get_hash_tree_root(self, value) -> bytes:
        if isinstance(self.element_sedes, BasicSedes):
            serialized_items = tuple(
                self.element_sedes.serialize(element) for element in value
            )
            merkle_leaves = pack(serialized_items)
            element_size = self.element_sedes.get_fixed_size()
            limit = (len(value) * element_size + CHUNK_SIZE - 1) // CHUNK_SIZE
        else:
            merkle_leaves = tuple(
                self.element_sedes.get_hash_tree_root(element) for element in value
            )
            limit = len(value)
        return mix_in_length(merkleize(merkle_leaves, limit=limit), len(value))


class BuggyListPartition(List):
    def _deserialize_stream_to_tuple(self, stream):
        if self.element_sedes.is_fixed_sized:
            element_size = self.element_sedes.get_fixed_size()
            data = stream.read()
            if len(data) % element_size != 0:
                raise DeserializationError(
                    f"Invalid max_length. List is comprised of a fixed size sedes "
                    f"but total serialized data is not an even multiple of the "
                    f"element size. data length: {len(data)}  element size: "
                    f"{element_size}"
                )
            for segment in partition(element_size, data):
                yield self.element_sedes.deserialize(segment)
        else:
            yield from super()._deserialize_stream_to_tuple(stream)


def _pack_bytes_no_pad(data: bytes) -> tuple[bytes, ...]:
    if len(data) == 0:
        return tuple()
    return tuple(data[i : i + CHUNK_SIZE] for i in range(0, len(data), CHUNK_SIZE))


class BuggyByteListNoPad(ByteList):
    def get_hash_tree_root(self, value: bytes) -> bytes:
        serialized_value = self.serialize(value)
        merkle_leaves = _pack_bytes_no_pad(serialized_value)
        merkleized = merkleize(merkle_leaves, limit=self.chunk_count)
        return mix_in_length(merkleized, len(value))


def _bool_sedes() -> Boolean:
    if BUG_ID == "PSSZ-BOOL-DIRTY":
        return BuggyBoolean()
    return boolean


def _bitvector4_sedes() -> Bitvector:
    if BUG_ID == "PSSZ-BV-DIRTY":
        return Bitvector(4)
    return StrictBitvector(4)


def _bitlist2048_sedes() -> Bitlist:
    if BUG_ID == "PSSZ-109":
        return BuggyBitlist(2048)
    if BUG_ID == "PSSZ-82":
        return BuggyBitlistNoLimit(2048)
    if BUG_ID == "PSSZ-HTR-BITLIST-NOMIX":
        return BuggyBitlistNoMixIn(2048)
    return Bitlist(2048)


def _balances_list_sedes() -> List:
    if BUG_ID == "PSSZ-83":
        return BuggyListNoLimit(uint64, 128)
    if BUG_ID == "PSSZ-HTR-LIST-NOMIX":
        return BuggyListNoMixIn(uint64, 128)
    return List(uint64, 128)


def _byte_list_sedes() -> ByteList:
    if BUG_ID == "PSSZ-35":
        return BuggyByteListNoPad(31)
    return ByteList(31)


def _header_list_sedes(header_type) -> List:
    if BUG_ID == "PSSZ-74":
        return BuggyListPartition(header_type, 4)
    return List(header_type, 4)


def build_schema(name: str) -> type[Serializable]:
    class BeaconBlockHeader(Serializable):
        fields = (
            ("slot", uint64),
            ("proposer_index", uint64),
            ("parent_root", bytes32),
            ("state_root", bytes32),
            ("body_root", bytes32),
        )

    if name == "PSSZBoolBench":
        class PSSZBoolBench(Serializable):
            fields = (("slashed", _bool_sedes()), ("epoch", uint64), ("root", bytes32))

        return PSSZBoolBench
    if name == "PSSZBitvectorBench":
        class PSSZBitvectorBench(Serializable):
            fields = (
                ("slot", uint64),
                ("root", bytes32),
                ("justification_bits", _bitvector4_sedes()),
            )

        return PSSZBitvectorBench
    if name == "PSSZBitlistBench":
        class PSSZBitlistBench(Serializable):
            fields = (("aggregation_bits", _bitlist2048_sedes()), ("slot", uint64))

        return PSSZBitlistBench
    if name == "PSSZByteListBench":
        class PSSZByteListBench(Serializable):
            fields = (("data", _byte_list_sedes()),)

        return PSSZByteListBench
    if name == "PSSZTailBench":
        class PSSZTailBench(Serializable):
            fields = (("slot", uint64),)

        return PSSZTailBench
    if name == "PSSZHTRListBench":
        class PSSZHTRListBench(Serializable):
            fields = (("balances", _balances_list_sedes()),)

        return PSSZHTRListBench
    if name == "PSSZHeaderListBench":
        class PSSZHeaderListBench(Serializable):
            fields = (("headers", _header_list_sedes(BeaconBlockHeader)),)

        return PSSZHeaderListBench
    raise KeyError(f"unknown schema: {name}")


def _schema_validation_result(schema_name: str, length: int) -> dict:
    if schema_name == "PSSZ-111":
        if length == 0:
            if BUG_ID == "PSSZ-111":
                return {"ok": True}
            return {"ok": False, "error": "invalid vector length 0"}
        try:
            Vector(uint8, length)
        except Exception as exc:
            return {"ok": False, "error": str(exc)}
        return {"ok": True}
    if schema_name == "PSSZ-112":
        if length == 0:
            if BUG_ID == "PSSZ-112":
                return {"ok": True}
            return {"ok": False, "error": "invalid bitvector length 0"}
        try:
            Bitvector(length)
        except Exception as exc:
            return {"ok": False, "error": str(exc)}
        return {"ok": True}
    if schema_name == "PSSZ-116":
        if length == 0:
            if BUG_ID == "PSSZ-116":
                return {"ok": False, "error": "buggy list length 0"}
            return {"ok": True}
        try:
            List(uint8, length)
        except Exception as exc:
            return {"ok": False, "error": str(exc)}
        return {"ok": True}
    return {"ok": False, "error": f"unknown schema validation case: {schema_name}"}


def handle_request(req: dict) -> dict:
    op = req.get("op")
    schema_name = req.get("schema")
    if not schema_name:
        return {"ok": False, "error": "missing schema"}

    if op == "ping":
        if schema_name in SCHEMA_VALIDATION_CASES:
            return {"ok": True}
        try:
            build_schema(schema_name)
        except Exception as exc:
            return {"ok": False, "error": str(exc)}
        return {"ok": True}
    if op == "schema":
        if schema_name not in SCHEMA_VALIDATION_CASES:
            return {"ok": False, "error": f"unknown schema validation case: {schema_name}"}
        data_hex = req.get("data", "")
        try:
            data = binascii.unhexlify(data_hex)
        except (binascii.Error, ValueError) as exc:
            return {"ok": False, "error": f"invalid hex: {exc}"}
        if len(data) != 8:
            return {"ok": False, "error": f"invalid length bytes: {len(data)}"}
        length = int.from_bytes(data, "little")
        return _schema_validation_result(schema_name, length)
    if op != "decode":
        return {"ok": False, "error": f"unknown op: {op}"}

    try:
        schema = build_schema(schema_name)
    except Exception as exc:
        return {"ok": False, "error": str(exc)}

    data_hex = req.get("data", "")
    try:
        data = binascii.unhexlify(data_hex)
    except (binascii.Error, ValueError) as exc:
        return {"ok": False, "error": f"invalid hex: {exc}"}

    try:
        obj = schema.deserialize(data)
        canon = schema.serialize(obj)
        root = schema.get_hash_tree_root(obj, cache=False)
    except Exception as exc:
        return {"ok": False, "error": str(exc)}

    return {
        "ok": True,
        "canon": binascii.hexlify(canon).decode("ascii"),
        "root": binascii.hexlify(root).decode("ascii"),
    }


def main() -> int:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
        except json.JSONDecodeError as exc:
            resp = {"ok": False, "error": f"invalid json: {exc}"}
        else:
            resp = handle_request(req)
        sys.stdout.write(json.dumps(resp) + "\n")
        sys.stdout.flush()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
