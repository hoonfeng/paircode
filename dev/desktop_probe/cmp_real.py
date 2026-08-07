#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""cmp_real.py — 对齐 real_tree_wb.json（wb-ui 元素树）与 ide_tree_edge.json
（Edge 参照树），按 tag+class 逐层对齐，输出几何（x/y/w/h）与样式
（display/color/bg）差异。

用法：
    python dev/desktop_probe/cmp_real.py [--max N]

输出格式：
    [depth:idx] tag.class  wb(x,y,w,h) vs edge(x,y,w,h)  diff=...
"""
import json
import sys
import os


def load_tree(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def norm_style(node):
    """统一字段名：edge 树用 cls/disp/col/bg/fs，wb 树用 class/display/color/bg/fontSz"""
    return {
        "cls": node.get("class") or node.get("cls") or "",
        "disp": node.get("display") or node.get("disp") or "",
        "col": node.get("color") or node.get("col") or "",
        "bg": node.get("bg") or "",
        "fs": node.get("fontSz") or node.get("fs") or "",
    }


def norm_geo(node):
    return (node.get("x", 0), node.get("y", 0), node.get("w", 0), node.get("h", 0))


def key_of(node):
    return (node.get("tag", ""), (node.get("class") or node.get("cls") or ""))


def walk(node, depth, idx, out):
    out.append((depth, idx, node))
    for i, c in enumerate(node.get("children", []) or []):
        walk(c, depth + 1, i, out)


def fmt_geo(g):
    return "(%d,%d %dx%d)" % (g[0], g[1], g[2], g[3])


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    wb_tree = load_tree(os.path.join(here, "real_tree_wb.json"))
    edge_tree = load_tree(os.path.join(here, "ide_tree_edge.json"))

    wb_nodes = []
    edge_nodes = []
    walk(wb_tree, 0, 0, wb_nodes)
    walk(edge_tree, 0, 0, edge_nodes)

    max_rows = 200
    if "--max" in sys.argv:
        max_rows = int(sys.argv[sys.argv.index("--max") + 1])

    # 按 (depth, tag+class) 对齐：同一深度同一标签同一 class 视为同一元素
    wb_by_key = {}
    for d, i, n in wb_nodes:
        wb_by_key.setdefault((d, key_of(n)), []).append((i, n))
    edge_by_key = {}
    for d, i, n in edge_nodes:
        edge_by_key.setdefault((d, key_of(n)), []).append((i, n))

    print("=== 元素数: wb=%d edge=%d ===" % (len(wb_nodes), len(edge_nodes)))
    printed = 0
    for (d, key), items in sorted(wb_by_key.items()):
        tag, cls = key
        wb_item = items[0]
        edge_items = edge_by_key.get((d, key), [])
        if not edge_items:
            continue
        edge_item = edge_items[0]
        _, wbn = wb_item
        _, edgen = edge_item
        wg = norm_geo(wbn)
        eg = norm_geo(edgen)
        ws = norm_style(wbn)
        es = norm_style(edgen)
        diffs = []
        if wg != eg:
            diffs.append("geo %s vs %s" % (fmt_geo(wg), fmt_geo(eg)))
        if ws["disp"] != es["disp"]:
            diffs.append("disp %r vs %r" % (ws["disp"], es["disp"]))
        if ws["bg"] != es["bg"]:
            diffs.append("bg %r vs %r" % (ws["bg"], es["bg"]))
        if ws["col"] != es["col"]:
            diffs.append("col %r vs %r" % (ws["col"], es["col"]))
        label = "%s.%s" % (tag, cls) if cls else tag
        if diffs:
            print("[%d:%d] %-40s | %s" % (d, wbn.get("idx", 0), label, " ; ".join(diffs)))
            printed += 1
            if printed >= max_rows:
                print("...(截断，共 %d 处差异)" % printed)
                return
    print("=== 差异总数: %d ===" % printed)


if __name__ == "__main__":
    main()
