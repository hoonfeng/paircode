import json, sys

WB  = r'F:\syproject\gou-ide\dev\desktop_probe\ide_tree_wb.json'
EDGE = r'F:\syproject\gou-ide\dev\desktop_probe\ide_tree_edge.json'

TOL = 1.0  # px 容忍

def load(p):
    return json.load(open(p, encoding='utf-8'))

def label(n):
    s = n.get('tag','')
    if n.get('id'): s += '#' + n['id']
    cls = n.get('class') or n.get('cls') or ''
    if cls:
        # 只保留第一个 class token（Vue scoped 类不同源，去掉 data-v-）
        tok = [t for t in cls.split() if not t.startswith('data-v-')]
        if tok: s += '.' + tok[0]
    return s

def sig(n):
    return n.get('tag','')

def norm_color(s):
    """把 #rrggbb / rgb() / rgba() 归一化为元组，消除格式差异。"""
    s = (s or '').strip()
    if not s:
        return ''
    if s.startswith('#'):
        h = s[1:]
        if len(h) == 3:
            h = ''.join(c * 2 for c in h)
        if len(h) == 8:
            return (int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16), round(int(h[6:8], 16) / 255, 2))
        return (int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16), 1.0)
    if s.startswith('rgba(') and s.endswith(')'):
        p = [x.strip() for x in s[5:-1].split(',')]
        return (int(p[0]), int(p[1]), int(p[2]), round(float(p[3]), 2))
    if s.startswith('rgb(') and s.endswith(')'):
        p = [x.strip() for x in s[4:-1].split(',')]
        return (int(p[0]), int(p[1]), int(p[2]), 1.0)
    return s

def walk_pairs(w, e, path, out):
    """按树路径对齐 wb-ui 树与 Edge 树（同层兄弟按 idx 对齐）。"""
    lw, le = label(w), label(e)
    if lw != le:
        out.append(('TAG', path, lw, le))
    # 几何对比（只看有实际尺寸的元素）
    ww, wh = w.get('w',0), w.get('h',0)
    ew, eh = e.get('w',0), e.get('h',0)
    if ww > TOL or wh > TOL or ew > TOL or eh > TOL:
        for f in ('x','y','w','h'):
            a, b = w.get(f,0), e.get(f,0)
            if abs(a-b) > TOL:
                out.append((f.upper(), path+'/'+lw, a, b))
                break
    # 样式对比（颜色归一化后比较）
    for f in ('bg','col','fs'):
        a, b = w.get(f,''), e.get(f,'')
        if a and b:
            if f == 'bg' or f == 'col':
                if norm_color(a) != norm_color(b):
                    out.append(('STYLE:'+f, path+'/'+lw, a, b))
            elif a != b:
                out.append(('STYLE:'+f, path+'/'+lw, a, b))
    wc = w.get('children') or []
    ec = e.get('children') or []
    for i in range(max(len(wc), len(ec))):
        if i < len(wc) and i < len(ec):
            walk_pairs(wc[i], ec[i], path + '/' + lw + f'[{i}]', out)
        elif i < len(wc):
            out.append(('ONLY_WB', path + '/' + lw + f'[{i}]', label(wc[i]), ''))
        else:
            out.append(('ONLY_EDGE', path + '/' + lw + f'[{i}]', '', label(ec[i])))

def find_body(n):
    """递归找 body 节点（tag=='body'）。"""
    if n.get('tag') == 'body':
        return n
    for c in (n.get('children') or []):
        r = find_body(c)
        if r: return r
    return None

def main():
    w = load(WB); e = load(EDGE)
    wb_body = find_body(w)
    eb_body = find_body(e)
    if not wb_body or not eb_body:
        print('未找到 body:', bool(wb_body), bool(eb_body))
        return
    out = []
    walk_pairs(wb_body, eb_body, 'body', out)
    # 汇总
    from collections import Counter
    kinds = Counter(o[0] for o in out)
    print('=== wb-ui vs Edge 元素树对比 ===')
    print('总差异:', len(out), kinds)
    print()
    # 按类型分组打印（TAG/几何优先）
    for kind in ('TAG','X','Y','W','H','STYLE:bg','STYLE:col','STYLE:fs','ONLY_WB','ONLY_EDGE'):
        items = [o for o in out if o[0]==kind]
        if not items: continue
        print(f'--- {kind} ({len(items)}) ---')
        for o in items[:25]:
            if kind == 'TAG':
                print(f'  {o[1]:70s} wb={o[2]!r:30s} edge={o[3]!r}')
            elif kind.startswith('STYLE'):
                print(f'  {o[1]:70s} wb={o[2]!r:20s} edge={o[3]!r}')
            elif kind.startswith('ONLY'):
                print(f'  {o[1]:70s} extra={o[2] or o[3]!r}')
            else:
                print(f'  {o[1]:70s} wb={o[2]} edge={o[3]}')
        print()

main()
