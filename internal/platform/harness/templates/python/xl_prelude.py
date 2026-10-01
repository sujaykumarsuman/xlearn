# xl_prelude.py: the Python harness runtime of xlearn internal/platform/harness (func-json@1,
# class-ops@1). Public and versioned with the harness; DO NOT EDIT a copy. The generated
# __main__.py installs the node classes its signature uses into builtins (so solution.py
# sees ListNode, TreeNode or Node without importing them), then drives one case. It is one
# static file for every signature. Names starting with xl or _ are reserved.
#
# Wire format (README.md): one u32 big-endian framed canonical JSON frame in on fd 3, one out
# on fd 4: {"ok":<value>}, {"panic":"<class>"} or {"error":"<code>"}. The bytes match the Go
# harness exactly (the cross-language golden test is the guard).

import builtins
import json
import os
import sys

MAX_NODES = 1000000
MAX_FRAME = 64 << 20
_I32 = (-(1 << 31), (1 << 31) - 1)
_I64 = (-(1 << 63), (1 << 63) - 1)
_INF = float("inf")


class ListNode:
    def __init__(self, val=0, next=None):
        self.val = val
        self.next = next


class TreeNode:
    def __init__(self, val=0, left=None, right=None):
        self.val = val
        self.left = left
        self.right = right


class Node:
    def __init__(self, val=0, neighbors=None):
        self.val = val
        self.neighbors = neighbors if neighbors is not None else []


_NODES = {"ListNode": ListNode, "TreeNode": TreeNode, "Node": Node}


def install(names):
    """Make the signature's node classes visible to solution.py (through builtins)."""
    for n in names:
        setattr(builtins, n, _NODES[n])


class Fail(Exception):
    """A harness error: {"error":"<code>"}."""

    def __init__(self, code):
        Exception.__init__(self, code)
        self.code = code


def fail(code):
    raise Fail(code)


# ---- fd 3 / fd 4 ----


def _read_n(fd, n):
    parts = []
    while n > 0:
        try:
            b = os.read(fd, min(n, 1 << 20))
        except OSError:
            fail("bad_input")
        if not b:
            fail("bad_input")
        parts.append(b)
        n -= len(b)
    return b"".join(parts)


def _bad_constant(_):
    fail("bad_input")


def read_input():
    n = int.from_bytes(_read_n(3, 4), "big")
    if n > MAX_FRAME:
        fail("bad_input")
    b = _read_n(3, n)
    try:
        return json.loads(b.decode("utf-8"), parse_constant=_bad_constant)
    except ValueError:
        fail("bad_input")


def _write_frame(payload):
    out = len(payload).to_bytes(4, "big") + payload
    while out:
        try:
            k = os.write(4, out)
        except InterruptedError:
            continue
        except OSError:
            os._exit(98)
        if k <= 0:
            os._exit(98)
        out = out[k:]


# ---- inputs (the JSON-decoded case, checked against the registry types) ----


def args_of(p, n):
    """func-json@1: {"args":[…]} with exactly n arguments."""
    if type(p) is not dict or list(p) != ["args"]:
        fail("bad_input")
    a = p["args"]
    if type(a) is not list or len(a) != n:
        fail("bad_input")
    return a


def ops_of(p, cls):
    """class-ops@1: {"args":[…],"ops":[…]}, ops[0] the class; returns (ops, args)."""
    if type(p) is not dict or sorted(p) != ["args", "ops"]:
        fail("bad_input")
    ops, args = p["ops"], p["args"]
    if type(ops) is not list or type(args) is not list or not ops or len(ops) != len(args):
        fail("bad_input")
    for o in ops:
        if type(o) is not str:
            fail("bad_input")
    if ops[0] != cls:
        fail("bad_input")
    return ops, args


def arity(a, n):
    if type(a) is not list or len(a) != n:
        fail("bad_input")
    return a


def _ints(v, lo, hi):
    if type(v) is not list:
        fail("bad_input")
    if v and (set(map(type, v)) != {int} or min(v) < lo or max(v) > hi):
        fail("bad_input")
    return v


def dec_int(v):
    if type(v) is not int or v < _I32[0] or v > _I32[1]:
        fail("bad_input")
    return v


def dec_int64(v):
    if type(v) is not int or v < _I64[0] or v > _I64[1]:
        fail("bad_input")
    return v


def dec_float64(v):
    if type(v) is int:
        try:
            v = float(v)
        except OverflowError:
            fail("bad_input")
    elif type(v) is not float:
        fail("bad_input")
    if v != v or v == _INF or v == -_INF:
        fail("bad_input")
    return v


def dec_bool(v):
    if type(v) is not bool:
        fail("bad_input")
    return v


def dec_string(v):
    if type(v) is not str:
        fail("bad_input")
    return v


def _arr(v, f):
    if type(v) is not list:
        fail("bad_input")
    return [f(x) for x in v]


def dec_int_1(v):
    return _ints(v, _I32[0], _I32[1])


def dec_int64_1(v):
    return _ints(v, _I64[0], _I64[1])


def dec_float64_1(v):
    return _arr(v, dec_float64)


def dec_bool_1(v):
    return _arr(v, dec_bool)


def dec_string_1(v):
    return _arr(v, dec_string)


def dec_int_2(v):
    return _arr(v, dec_int_1)


def dec_int64_2(v):
    return _arr(v, dec_int64_1)


def dec_float64_2(v):
    return _arr(v, dec_float64_1)


def dec_bool_2(v):
    return _arr(v, dec_bool_1)


def dec_string_2(v):
    return _arr(v, dec_string_1)


def dec_ListNode(v):
    head = None
    for x in reversed(dec_int_1(v)):
        head = ListNode(x, head)
    return head


def dec_ListNode_1(v):
    return _arr(v, dec_ListNode)


def dec_TreeNode(v):
    if type(v) is not list:
        fail("bad_input")
    vals = [None if x is None else dec_int(x) for x in v]
    if not vals or vals[0] is None:
        return None
    root = TreeNode(vals[0])
    q = [root]
    qi = 0
    i = 1
    while qi < len(q) and i < len(vals):
        n = q[qi]
        qi += 1
        if vals[i] is not None:
            n.left = TreeNode(vals[i])
            q.append(n.left)
        i += 1
        if i < len(vals) and vals[i] is not None:
            n.right = TreeNode(vals[i])
            q.append(n.right)
        i += 1
    return root


def dec_TreeNode_1(v):
    return _arr(v, dec_TreeNode)


def dec_GraphNode(v):
    adj = _arr(v, dec_int_1)
    if not adj:
        return None
    nodes = [Node(i + 1) for i in range(len(adj))]
    for i, row in enumerate(adj):
        for k in row:
            if k < 1 or k > len(adj):
                fail("bad_input")
            nodes[i].neighbors.append(nodes[k - 1])
    return nodes[0]


# ---- outputs (canonical JSON; a value of the wrong type is the learner's TypeError) ----


class Writer:
    __slots__ = ("parts", "nodes")

    def __init__(self):
        self.parts = []
        self.nodes = 0

    def raw(self, s):
        self.parts.append(s)

    def node(self):
        self.nodes += 1
        if self.nodes > MAX_NODES:
            fail("too_many_nodes")


def _type_error(want, v):
    raise TypeError("the harness wants %s, got %s" % (want, type(v).__name__))


def _int_text(v):
    if type(v) is not int:
        _type_error("int", v)
    return str(v)


def gofloat(v):
    """Go's strconv.FormatFloat(v, 'g', -1, 64) for a finite float: the shortest round-trip
    digits, %e when the exponent is < -4 or >= 6 (two exponent digits at least), else %f."""
    if v == 0:
        return "0"
    r = repr(v)
    neg = r[0] == "-"
    if neg:
        r = r[1:]
    m, _, e = r.partition("e")
    ex = int(e) if e else 0
    ip, _, fp = m.partition(".")
    full = ip + fp
    digs = full.lstrip("0")
    dp = len(ip) - (len(full) - len(digs)) + ex
    digs = digs.rstrip("0") or "0"
    nd = len(digs)
    exp = dp - 1
    sign = "-" if neg else ""
    if exp < -4 or exp >= 6:
        mant = digs[0] + ("." + digs[1:] if nd > 1 else "")
        a = -exp if exp < 0 else exp
        return "%s%se%s%02d" % (sign, mant, "-" if exp < 0 else "+", a)
    if dp > 0:
        ipart = digs[:dp] + "0" * (dp - nd) if dp > nd else digs[:dp]
    else:
        ipart = "0"
    prec = max(nd - dp, 0)
    if prec == 0:
        return sign + ipart
    frac = "".join(digs[j] if 0 <= j < nd else "0" for j in range(dp, dp + prec))
    return sign + ipart + "." + frac


def _float_text(v):
    if type(v) is int and type(v) is not bool:
        try:
            v = float(v)
        except OverflowError:
            fail("non_finite")
    elif type(v) is not float:
        _type_error("float", v)
    if v != v or v == _INF or v == -_INF:
        fail("non_finite")
    return gofloat(v)


def _bool_text(v):
    if type(v) is not bool:
        _type_error("bool", v)
    return "true" if v else "false"


_ESC = {0x22: '\\"', 0x5C: "\\\\"}
for _c in range(0x20):
    _ESC[_c] = "\\u%04x" % _c
del _c


def _string_text(v):
    if type(v) is not str:
        _type_error("str", v)
    if not v.isascii():
        try:
            v.encode("utf-8")
        except UnicodeEncodeError:
            fail("invalid_utf8")
    return '"' + v.translate(_ESC) + '"'


def _seq(v):
    if v is None:
        return ()
    if type(v) is not list and type(v) is not tuple:
        _type_error("list", v)
    return v


def enc_int(w, v):
    w.raw(_int_text(v))


def enc_int64(w, v):
    w.raw(_int_text(v))


def enc_float64(w, v):
    w.raw(_float_text(v))


def enc_bool(w, v):
    w.raw(_bool_text(v))


def enc_string(w, v):
    w.raw(_string_text(v))


def _join(v, f):
    return "[" + ",".join(map(f, _seq(v))) + "]"


def _int_list_text(v):
    v = _seq(v)
    if v and set(map(type, v)) != {int}:
        for x in v:
            _int_text(x)
    return "[" + ",".join(map(str, v)) + "]"


def enc_int_1(w, v):
    w.raw(_int_list_text(v))


def enc_int64_1(w, v):
    w.raw(_int_list_text(v))


def enc_float64_1(w, v):
    w.raw(_join(v, _float_text))


def enc_bool_1(w, v):
    w.raw(_join(v, _bool_text))


def enc_string_1(w, v):
    w.raw(_join(v, _string_text))


def enc_int_2(w, v):
    w.raw(_join(v, _int_list_text))


def enc_int64_2(w, v):
    w.raw(_join(v, _int_list_text))


def enc_float64_2(w, v):
    w.raw(_join(v, lambda r: _join(r, _float_text)))


def enc_bool_2(w, v):
    w.raw(_join(v, lambda r: _join(r, _bool_text)))


def enc_string_2(w, v):
    w.raw(_join(v, lambda r: _join(r, _string_text)))


def enc_ListNode(w, head):
    seen = set()
    out = []
    n = head
    while n is not None:
        i = id(n)
        if i in seen:
            fail("cycle")
        seen.add(i)
        w.node()
        out.append(_int_text(n.val))
        n = n.next
    w.raw("[" + ",".join(out) + "]")


def enc_TreeNode(w, root):
    if root is None:
        w.raw("[]")
        return
    seen = set()
    q = [root]
    i = 0
    while i < len(q):
        n = q[i]
        i += 1
        if n is None:
            continue
        k = id(n)
        if k in seen:
            fail("cycle")
        seen.add(k)
        w.node()
        q.append(n.left)
        q.append(n.right)
    end = len(q)
    while end > 0 and q[end - 1] is None:
        end -= 1
    w.raw("[" + ",".join("null" if n is None else _int_text(n.val) for n in q[:end]) + "]")


def enc_GraphNode(w, start):
    if start is None:
        w.raw("[]")
        return
    by_val = {}
    seen = {id(start)}
    q = [start]
    i = 0
    while i < len(q):
        n = q[i]
        i += 1
        w.node()
        v = n.val
        if type(v) is not int:
            _type_error("int", v)
        if v in by_val:
            fail("invalid_graph")
        by_val[v] = n
        for m in _seq(n.neighbors):
            if m is None:
                fail("invalid_graph")
            if id(m) not in seen:
                seen.add(id(m))
                q.append(m)
    rows = []
    for v in range(1, len(q) + 1):
        n = by_val.get(v)
        if n is None:
            fail("invalid_graph")
        rows.append("[" + ",".join(_int_text(m.val) for m in _seq(n.neighbors)) + "]")
    w.raw("[" + ",".join(rows) + "]")


def _nodes_list(f):
    def enc(w, v):
        w.raw("[")
        for k, e in enumerate(_seq(v)):
            if k:
                w.raw(",")
            f(w, e)
        w.raw("]")

    return enc


enc_ListNode_1 = _nodes_list(enc_ListNode)
enc_TreeNode_1 = _nodes_list(enc_TreeNode)


# ---- the case driver ----

_PANICS = (
    ("RecursionError", RecursionError),
    ("ZeroDivisionError", ZeroDivisionError),
    ("IndexError", IndexError),
    ("KeyError", KeyError),
    ("ValueError", ValueError),
    ("TypeError", TypeError),
    ("AttributeError", AttributeError),
    ("AssertionError", AssertionError),
)


def panic_class(e):
    for name, cls in _PANICS:
        if isinstance(e, cls):
            return name
    return "other"


def run(case):
    """Run one case and write exactly one frame on fd 4: case(input) returns a Writer holding
    the {"ok"} frame; a harness Fail is {"error"}, any other exception {"panic"}. A learner
    exit or an uncatchable failure leaves fd 4 empty."""
    sys.setrecursionlimit(1000000)
    try:
        out = "".join(case(read_input()).parts)
    except Fail as f:
        out = '{"error":"' + f.code + '"}'
    except Exception as e:  # noqa: BLE001 - every learner exception is a panic class
        out = '{"panic":"' + panic_class(e) + '"}'
    _write_frame(out.encode("utf-8"))
    for s in (sys.stdout, sys.stderr):
        try:
            s.flush()
        except Exception:  # noqa: BLE001
            pass
    os._exit(0)
