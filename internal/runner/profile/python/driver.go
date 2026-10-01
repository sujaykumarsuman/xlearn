package python

// CompileDriver is the compile step (`python3 -I -S -B -c CompileDriver /w/app.pyz`, cwd /src):
//
//  1. compile() every source (SyntaxError, IndentationError, a null byte → CE), one JSON
//     diagnostic line per failure on stderr;
//  2. read XL_REQUIRES from __main__.py (the class and the (method, positional arity) pairs
//     the generated harness calls) and check solution.py's top level defines them; a miss is
//     a CE positioned in __main__.py, the harness file (judge shows its fixed CE text, m3-06).
//     A class with bases, a decorated method or a name bound some other way is accepted
//     unchecked;
//  3. write a deterministic zipapp: the sources sorted, stored, fixed timestamps and modes.
//
// It runs only Python's own parser on the learner's file, never the file itself.
const CompileDriver = `import ast, json, os, sys, zipfile
NAMES = ("__main__.py", "solution.py", "xl_prelude.py")
ok = True
def diag(f, line, col, msg):
    global ok
    ok = False
    sys.stderr.write(json.dumps({"file": f, "line": line or 0, "col": col or 0, "msg": msg}) + "\n")
srcs = {}
for n in NAMES:
    try:
        with open(n, "rb") as fh:
            b = fh.read()
    except FileNotFoundError:
        continue
    try:
        tree = compile(b, n, "exec", ast.PyCF_ONLY_AST, dont_inherit=True)
        compile(tree, n, "exec", dont_inherit=True)
    except SyntaxError as e:
        diag(n, e.lineno, e.offset, type(e).__name__ + ": " + str(e.msg))
        continue
    except (ValueError, RecursionError, MemoryError, OverflowError) as e:
        diag(n, 1, 0, type(e).__name__ + ": " + str(e))
        continue
    srcs[n] = (b, tree)
def arity_ok(fn, n):
    if fn.decorator_list:
        return True
    a = fn.args
    pos = a.posonlyargs + a.args
    if not pos:
        return False
    pos = pos[1:]
    if any(d is None for d in a.kw_defaults):
        return False
    lo = len(pos) - len(a.defaults)
    if a.vararg is not None:
        return n >= lo
    return lo <= n <= len(pos)
def check(tree, cls, methods):
    found = None
    bound = False
    for node in tree.body:
        if isinstance(node, ast.ClassDef) and node.name == cls:
            found = node
        elif isinstance(node, (ast.Assign, ast.AnnAssign, ast.Import, ast.ImportFrom, ast.FunctionDef)):
            names = []
            if isinstance(node, ast.Assign):
                names = [t.id for t in node.targets if isinstance(t, ast.Name)]
            elif isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
                names = [node.target.id]
            elif isinstance(node, (ast.Import, ast.ImportFrom)):
                names = [(a.asname or a.name).split(".")[0] for a in node.names]
            elif isinstance(node, ast.FunctionDef):
                names = [node.name]
            bound = bound or cls in names
    if found is None:
        return [] if bound else ["solution.py does not define class %s" % cls]
    if found.bases or found.keywords:
        return []
    defs, other = {}, set()
    for node in found.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            defs[node.name] = node
        elif isinstance(node, ast.Assign):
            other.update(t.id for t in node.targets if isinstance(t, ast.Name))
    out = []
    for name, n in methods:
        if name in other:
            continue
        fn = defs.get(name)
        if fn is None:
            if name == "__init__" and n == 0:
                continue
            out.append("class %s has no method %s taking %d argument(s)" % (cls, name, n))
        elif not arity_ok(fn, n):
            out.append("%s.%s does not take %d positional argument(s)" % (cls, name, n))
    return out
main = srcs.get("__main__.py")
if main is not None and "solution.py" in srcs:
    for node in main[1].body:
        if isinstance(node, ast.Assign) and len(node.targets) == 1 and isinstance(node.targets[0], ast.Name) and node.targets[0].id == "XL_REQUIRES":
            cls, methods = ast.literal_eval(node.value)
            for msg in check(srcs["solution.py"][1], cls, methods):
                diag("__main__.py", node.lineno, 1, msg)
if not ok:
    sys.exit(1)
with zipfile.ZipFile(sys.argv[1], "w", zipfile.ZIP_STORED) as z:
    for n in sorted(srcs):
        zi = zipfile.ZipInfo(n, date_time=(1980, 1, 1, 0, 0, 0))
        zi.create_system = 3
        zi.external_attr = 0o100444 << 16
        z.writestr(zi, srcs[n][0])
`
