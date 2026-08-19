var GitPanel = (function(exports, vue, uiState_js, api) {
  "use strict";
  const _export_sfc = (sfc, props) => {
    const target = sfc.__vccOpts || sfc;
    for (const [key, val] of props) {
      target[key] = val;
    }
    return target;
  };
  const _hoisted_1$2 = ["width", "height"];
  const _hoisted_2$2 = {
    key: 0,
    d: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
  };
  const _sfc_main$2 = {
    __name: "SvgIcon",
    props: {
      name: { type: String, required: true },
      size: { type: Number, default: 16 }
    },
    setup(__props) {
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("svg", {
          class: "svg-icon",
          width: __props.size,
          height: __props.size,
          viewBox: "0 0 24 24",
          fill: "none",
          stroke: "currentColor",
          "stroke-width": "2",
          "stroke-linecap": "round",
          "stroke-linejoin": "round"
        }, [
          vue.createCommentVNode(" Folder "),
          __props.name === "folder" ? (vue.openBlock(), vue.createElementBlock("path", _hoisted_2$2)) : __props.name === "folder-open" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 1 },
            [
              vue.createCommentVNode(" Folder Open "),
              _cache[0] || (_cache[0] = vue.createElementVNode(
                "path",
                { d: "M6 17l-3-9h18l-3 9H6z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[1] || (_cache[1] = vue.createElementVNode(
                "path",
                { d: "M4 8V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v3" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 2 },
            [
              vue.createCommentVNode(" File "),
              _cache[2] || (_cache[2] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[3] || (_cache[3] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-code" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 3 },
            [
              vue.createCommentVNode(" File Code "),
              _cache[4] || (_cache[4] = vue.createStaticVNode('<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" data-v-faf69761></path><polyline points="14 2 14 8 20 8" data-v-faf69761></polyline><line x1="10" y1="12" x2="8" y2="14" data-v-faf69761></line><line x1="10" y1="16" x2="8" y2="18" data-v-faf69761></line><line x1="14" y1="12" x2="16" y2="14" data-v-faf69761></line><line x1="14" y1="16" x2="16" y2="18" data-v-faf69761></line>', 6))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-text" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 4 },
            [
              vue.createCommentVNode(" File Text / Document "),
              _cache[5] || (_cache[5] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[6] || (_cache[6] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[7] || (_cache[7] = vue.createElementVNode(
                "line",
                {
                  x1: "9",
                  y1: "13",
                  x2: "15",
                  y2: "13"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[8] || (_cache[8] = vue.createElementVNode(
                "line",
                {
                  x1: "9",
                  y1: "17",
                  x2: "15",
                  y2: "17"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "search" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 5 },
            [
              vue.createCommentVNode(" Search "),
              _cache[9] || (_cache[9] = vue.createElementVNode(
                "circle",
                {
                  cx: "11",
                  cy: "11",
                  r: "8"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[10] || (_cache[10] = vue.createElementVNode(
                "line",
                {
                  x1: "21",
                  y1: "21",
                  x2: "16.65",
                  y2: "16.65"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "terminal" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 6 },
            [
              vue.createCommentVNode(" Terminal / Console "),
              _cache[11] || (_cache[11] = vue.createElementVNode(
                "polyline",
                { points: "4 17 10 11 4 5" },
                null,
                -1
                /* CACHED */
              )),
              _cache[12] || (_cache[12] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "19",
                  x2: "20",
                  y2: "19"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "chat" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 7 },
            [
              vue.createCommentVNode(" Chat / Message "),
              _cache[13] || (_cache[13] = vue.createElementVNode(
                "path",
                { d: "M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "settings" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 8 },
            [
              vue.createCommentVNode(" Gear / Settings "),
              _cache[14] || (_cache[14] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "3"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[15] || (_cache[15] = vue.createElementVNode(
                "path",
                { d: "M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "home" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 9 },
            [
              vue.createCommentVNode(" Home "),
              _cache[16] || (_cache[16] = vue.createElementVNode(
                "path",
                { d: "M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[17] || (_cache[17] = vue.createElementVNode(
                "polyline",
                { points: "9 22 9 12 15 12 15 22" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "chevron-right" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 10 },
            [
              vue.createCommentVNode(" Chevron Right "),
              _cache[18] || (_cache[18] = vue.createElementVNode(
                "polyline",
                { points: "9 6 15 12 9 18" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "chevron-down" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 11 },
            [
              vue.createCommentVNode(" Chevron Down (Rotated chevron-right) "),
              _cache[19] || (_cache[19] = vue.createElementVNode(
                "polyline",
                { points: "6 9 12 15 18 9" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "plus" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 12 },
            [
              vue.createCommentVNode(" Plus / Add "),
              _cache[20] || (_cache[20] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "5",
                  x2: "12",
                  y2: "19"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[21] || (_cache[21] = vue.createElementVNode(
                "line",
                {
                  x1: "5",
                  y1: "12",
                  x2: "19",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "close" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 13 },
            [
              vue.createCommentVNode(" Close / X "),
              _cache[22] || (_cache[22] = vue.createElementVNode(
                "line",
                {
                  x1: "18",
                  y1: "6",
                  x2: "6",
                  y2: "18"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[23] || (_cache[23] = vue.createElementVNode(
                "line",
                {
                  x1: "6",
                  y1: "6",
                  x2: "18",
                  y2: "18"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "refresh" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 14 },
            [
              vue.createCommentVNode(" Refresh "),
              _cache[24] || (_cache[24] = vue.createElementVNode(
                "polyline",
                { points: "23 4 23 10 17 10" },
                null,
                -1
                /* CACHED */
              )),
              _cache[25] || (_cache[25] = vue.createElementVNode(
                "path",
                { d: "M20.49 15a9 9 0 1 1-2.12-9.36L23 10" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "drive" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 15 },
            [
              vue.createCommentVNode(" Hard Drive / Disk "),
              _cache[26] || (_cache[26] = vue.createElementVNode(
                "line",
                {
                  x1: "22",
                  y1: "12",
                  x2: "2",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[27] || (_cache[27] = vue.createElementVNode(
                "path",
                { d: "M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[28] || (_cache[28] = vue.createElementVNode(
                "line",
                {
                  x1: "6",
                  y1: "16",
                  x2: "6.01",
                  y2: "16"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[29] || (_cache[29] = vue.createElementVNode(
                "line",
                {
                  x1: "10",
                  y1: "16",
                  x2: "10.01",
                  y2: "16"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "source-control" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 16 },
            [
              vue.createCommentVNode(" Source Control / Git Branch "),
              _cache[30] || (_cache[30] = vue.createElementVNode(
                "line",
                {
                  x1: "6",
                  y1: "3",
                  x2: "6",
                  y2: "15"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[31] || (_cache[31] = vue.createElementVNode(
                "circle",
                {
                  cx: "18",
                  cy: "6",
                  r: "3"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[32] || (_cache[32] = vue.createElementVNode(
                "circle",
                {
                  cx: "6",
                  cy: "18",
                  r: "3"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[33] || (_cache[33] = vue.createElementVNode(
                "path",
                { d: "M18 9a9 9 0 0 1-9 9" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "git-branch" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 17 },
            [
              vue.createCommentVNode(" Git Branch "),
              _cache[34] || (_cache[34] = vue.createElementVNode(
                "line",
                {
                  x1: "6",
                  y1: "3",
                  x2: "6",
                  y2: "15"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[35] || (_cache[35] = vue.createElementVNode(
                "circle",
                {
                  cx: "18",
                  cy: "6",
                  r: "3"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[36] || (_cache[36] = vue.createElementVNode(
                "circle",
                {
                  cx: "6",
                  cy: "18",
                  r: "3"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[37] || (_cache[37] = vue.createElementVNode(
                "path",
                { d: "M18 9a9 9 0 0 1-9 9" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "git-pull" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 18 },
            [
              vue.createCommentVNode(" Git Pull "),
              _cache[38] || (_cache[38] = vue.createStaticVNode('<circle cx="18" cy="18" r="3" data-v-faf69761></circle><circle cx="6" cy="6" r="3" data-v-faf69761></circle><path d="M13 6h3a2 2 0 0 1 2 2v7" data-v-faf69761></path><line x1="6" y1="18" x2="6" y2="9" data-v-faf69761></line><polyline points="9 9 6 6 3 9" data-v-faf69761></polyline>', 5))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "git-push" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 19 },
            [
              vue.createCommentVNode(" Git Push "),
              _cache[39] || (_cache[39] = vue.createStaticVNode('<circle cx="18" cy="6" r="3" data-v-faf69761></circle><circle cx="6" cy="18" r="3" data-v-faf69761></circle><path d="M13 18h-2a2 2 0 0 1-2-2V9" data-v-faf69761></path><line x1="6" y1="6" x2="6" y2="15" data-v-faf69761></line><polyline points="9 15 6 18 3 15" data-v-faf69761></polyline>', 5))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "output" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 20 },
            [
              vue.createCommentVNode(" Output / Window "),
              _cache[40] || (_cache[40] = vue.createElementVNode(
                "rect",
                {
                  x: "2",
                  y: "3",
                  width: "20",
                  height: "14",
                  rx: "2",
                  ry: "2"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[41] || (_cache[41] = vue.createElementVNode(
                "line",
                {
                  x1: "8",
                  y1: "21",
                  x2: "16",
                  y2: "21"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[42] || (_cache[42] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "17",
                  x2: "12",
                  y2: "21"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "warning" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 21 },
            [
              vue.createCommentVNode(" Warning / Alert "),
              _cache[43] || (_cache[43] = vue.createElementVNode(
                "path",
                { d: "M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[44] || (_cache[44] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "9",
                  x2: "12",
                  y2: "13"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[45] || (_cache[45] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "17",
                  x2: "12.01",
                  y2: "17"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "undo" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 22 },
            [
              vue.createCommentVNode(" Undo "),
              _cache[46] || (_cache[46] = vue.createElementVNode(
                "polyline",
                { points: "1 4 1 10 7 10" },
                null,
                -1
                /* CACHED */
              )),
              _cache[47] || (_cache[47] = vue.createElementVNode(
                "path",
                { d: "M3.51 15a9 9 0 1 0 2.13-9.36L1 10" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "redo" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 23 },
            [
              vue.createCommentVNode(" Redo "),
              _cache[48] || (_cache[48] = vue.createElementVNode(
                "polyline",
                { points: "23 4 23 10 17 10" },
                null,
                -1
                /* CACHED */
              )),
              _cache[49] || (_cache[49] = vue.createElementVNode(
                "path",
                { d: "M20.49 15a9 9 0 1 1-2.12-9.36L23 10" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "package" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 24 },
            [
              vue.createCommentVNode(" Package / Box / Store "),
              _cache[50] || (_cache[50] = vue.createElementVNode(
                "line",
                {
                  x1: "16.5",
                  y1: "9.4",
                  x2: "7.5",
                  y2: "4.21"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[51] || (_cache[51] = vue.createElementVNode(
                "path",
                { d: "M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[52] || (_cache[52] = vue.createElementVNode(
                "polyline",
                { points: "3.27 6.96 12 12.01 20.73 6.96" },
                null,
                -1
                /* CACHED */
              )),
              _cache[53] || (_cache[53] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "22.08",
                  x2: "12",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "globe" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 25 },
            [
              vue.createCommentVNode(" Globe / External "),
              _cache[54] || (_cache[54] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "10"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[55] || (_cache[55] = vue.createElementVNode(
                "line",
                {
                  x1: "2",
                  y1: "12",
                  x2: "22",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[56] || (_cache[56] = vue.createElementVNode(
                "path",
                { d: "M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "cycle" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 26 },
            [
              vue.createCommentVNode(" Refresh / Cycle (for agent) "),
              _cache[57] || (_cache[57] = vue.createElementVNode(
                "polyline",
                { points: "23 4 23 10 17 10" },
                null,
                -1
                /* CACHED */
              )),
              _cache[58] || (_cache[58] = vue.createElementVNode(
                "polyline",
                { points: "1 20 1 14 7 14" },
                null,
                -1
                /* CACHED */
              )),
              _cache[59] || (_cache[59] = vue.createElementVNode(
                "path",
                { d: "M3.51 9a9 9 0 0 1 14.85-3.36L23 10" },
                null,
                -1
                /* CACHED */
              )),
              _cache[60] || (_cache[60] = vue.createElementVNode(
                "path",
                { d: "M20.49 15a9 9 0 0 1-14.85 3.36L1 14" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "send" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 27 },
            [
              vue.createCommentVNode(" Send (arrow up) "),
              _cache[61] || (_cache[61] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "19",
                  x2: "12",
                  y2: "5"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[62] || (_cache[62] = vue.createElementVNode(
                "polyline",
                { points: "5 12 12 5 19 12" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "send-plane" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 28 },
            [
              vue.createCommentVNode(" Send Plane (paper airplane) "),
              _cache[63] || (_cache[63] = vue.createElementVNode(
                "line",
                {
                  x1: "22",
                  y1: "2",
                  x2: "11",
                  y2: "13"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[64] || (_cache[64] = vue.createElementVNode(
                "polygon",
                { points: "22 2 15 22 11 13 2 9 22 2" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "stop-dot" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 29 },
            [
              vue.createCommentVNode(" Stop Dot (pulsing circle) "),
              _cache[65] || (_cache[65] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "6",
                  class: "stop-pulse"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[66] || (_cache[66] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "10",
                  class: "stop-pulse-ring"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "wrench" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 30 },
            [
              vue.createCommentVNode(" Wrench / Tool "),
              _cache[67] || (_cache[67] = vue.createElementVNode(
                "path",
                { d: "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "database" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 31 },
            [
              vue.createCommentVNode(" Database "),
              _cache[68] || (_cache[68] = vue.createElementVNode(
                "ellipse",
                {
                  cx: "12",
                  cy: "5",
                  rx: "9",
                  ry: "3"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[69] || (_cache[69] = vue.createElementVNode(
                "path",
                { d: "M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" },
                null,
                -1
                /* CACHED */
              )),
              _cache[70] || (_cache[70] = vue.createElementVNode(
                "path",
                { d: "M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "user" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 32 },
            [
              vue.createCommentVNode(" User / Person "),
              _cache[71] || (_cache[71] = vue.createElementVNode(
                "path",
                { d: "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" },
                null,
                -1
                /* CACHED */
              )),
              _cache[72] || (_cache[72] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "7",
                  r: "4"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "info" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 33 },
            [
              vue.createCommentVNode(" Info "),
              _cache[73] || (_cache[73] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "10"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[74] || (_cache[74] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "16",
                  x2: "12",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[75] || (_cache[75] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "8",
                  x2: "12.01",
                  y2: "8"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "lightbulb" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 34 },
            [
              vue.createCommentVNode(" Lightbulb / Suggestion "),
              _cache[76] || (_cache[76] = vue.createElementVNode(
                "path",
                { d: "M9 18h6" },
                null,
                -1
                /* CACHED */
              )),
              _cache[77] || (_cache[77] = vue.createElementVNode(
                "path",
                { d: "M10 22h4" },
                null,
                -1
                /* CACHED */
              )),
              _cache[78] || (_cache[78] = vue.createElementVNode(
                "path",
                { d: "M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "sparkles" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 35 },
            [
              vue.createCommentVNode(" Sparkles / Auto "),
              _cache[79] || (_cache[79] = vue.createStaticVNode('<path d="M13.5 4L15 8l4 .5L15 12l1.5 4-4-2-4 2L10 12l-4-3.5L10 8z" data-v-faf69761></path><line x1="3" y1="18" x2="3" y2="21" data-v-faf69761></line><line x1="21" y1="18" x2="21" y2="21" data-v-faf69761></line><line x1="7" y1="20" x2="11" y2="20" data-v-faf69761></line><line x1="17" y1="20" x2="19" y2="20" data-v-faf69761></line>', 5))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "bot" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 36 },
            [
              vue.createCommentVNode(" Bot / AI "),
              _cache[80] || (_cache[80] = vue.createStaticVNode('<rect x="3" y="11" width="18" height="10" rx="2" data-v-faf69761></rect><circle cx="12" cy="5" r="2" data-v-faf69761></circle><path d="M12 7v4" data-v-faf69761></path><line x1="8" y1="16" x2="8" y2="16" data-v-faf69761></line><line x1="16" y1="16" x2="16" y2="16" data-v-faf69761></line>', 5))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-js" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 37 },
            [
              vue.createCommentVNode(" File Type Icons "),
              _cache[81] || (_cache[81] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[82] || (_cache[82] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[83] || (_cache[83] = vue.createElementVNode(
                "text",
                {
                  x: "8",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "JS",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-ts" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 38 },
            [
              _cache[84] || (_cache[84] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[85] || (_cache[85] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[86] || (_cache[86] = vue.createElementVNode(
                "text",
                {
                  x: "8",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "TS",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-go" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 39 },
            [
              _cache[87] || (_cache[87] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[88] || (_cache[88] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[89] || (_cache[89] = vue.createElementVNode(
                "text",
                {
                  x: "9",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "Go",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-py" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 40 },
            [
              _cache[90] || (_cache[90] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[91] || (_cache[91] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[92] || (_cache[92] = vue.createElementVNode(
                "text",
                {
                  x: "7",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "Py",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-java" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 41 },
            [
              _cache[93] || (_cache[93] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[94] || (_cache[94] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[95] || (_cache[95] = vue.createElementVNode(
                "text",
                {
                  x: "6",
                  y: "17",
                  "font-size": "8",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "Java",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-html" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 42 },
            [
              _cache[96] || (_cache[96] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[97] || (_cache[97] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[98] || (_cache[98] = vue.createElementVNode(
                "text",
                {
                  x: "6",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "HTML",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-css" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 43 },
            [
              _cache[99] || (_cache[99] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[100] || (_cache[100] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[101] || (_cache[101] = vue.createElementVNode(
                "text",
                {
                  x: "7",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "CSS",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-json" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 44 },
            [
              _cache[102] || (_cache[102] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[103] || (_cache[103] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[104] || (_cache[104] = vue.createElementVNode(
                "text",
                {
                  x: "5",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "{ }",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-md" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 45 },
            [
              _cache[105] || (_cache[105] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[106] || (_cache[106] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[107] || (_cache[107] = vue.createElementVNode(
                "text",
                {
                  x: "7",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "MD",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-vue" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 46 },
            [
              _cache[108] || (_cache[108] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[109] || (_cache[109] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[110] || (_cache[110] = vue.createElementVNode(
                "text",
                {
                  x: "7",
                  y: "17",
                  "font-size": "9",
                  fill: "currentColor",
                  "font-weight": "bold",
                  stroke: "none"
                },
                "Vue",
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "copy" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 47 },
            [
              vue.createCommentVNode(" Copy "),
              _cache[111] || (_cache[111] = vue.createElementVNode(
                "rect",
                {
                  x: "9",
                  y: "9",
                  width: "13",
                  height: "13",
                  rx: "2",
                  ry: "2"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[112] || (_cache[112] = vue.createElementVNode(
                "path",
                { d: "M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "minus" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 48 },
            [
              vue.createCommentVNode(" Minus "),
              _cache[113] || (_cache[113] = vue.createElementVNode(
                "line",
                {
                  x1: "5",
                  y1: "12",
                  x2: "19",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "edit" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 49 },
            [
              vue.createCommentVNode(" Edit / Rename "),
              _cache[114] || (_cache[114] = vue.createElementVNode(
                "path",
                { d: "M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" },
                null,
                -1
                /* CACHED */
              )),
              _cache[115] || (_cache[115] = vue.createElementVNode(
                "path",
                { d: "M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "trash" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 50 },
            [
              vue.createCommentVNode(" Trash / Delete "),
              _cache[116] || (_cache[116] = vue.createElementVNode(
                "polyline",
                { points: "3 6 5 6 21 6" },
                null,
                -1
                /* CACHED */
              )),
              _cache[117] || (_cache[117] = vue.createElementVNode(
                "path",
                { d: "M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "file-plus" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 51 },
            [
              vue.createCommentVNode(" File Plus / New File "),
              _cache[118] || (_cache[118] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[119] || (_cache[119] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[120] || (_cache[120] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "18",
                  x2: "12",
                  y2: "12"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[121] || (_cache[121] = vue.createElementVNode(
                "line",
                {
                  x1: "9",
                  y1: "15",
                  x2: "15",
                  y2: "15"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "message-square" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 52 },
            [
              vue.createCommentVNode(" Folder Plus / New Folder "),
              _cache[122] || (_cache[122] = vue.createElementVNode(
                "path",
                { d: "M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "folder-plus" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 53 },
            [
              _cache[123] || (_cache[123] = vue.createElementVNode(
                "path",
                { d: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v3" },
                null,
                -1
                /* CACHED */
              )),
              _cache[124] || (_cache[124] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "11",
                  x2: "12",
                  y2: "17"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[125] || (_cache[125] = vue.createElementVNode(
                "line",
                {
                  x1: "9",
                  y1: "14",
                  x2: "15",
                  y2: "14"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "brain" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 54 },
            [
              vue.createCommentVNode(" Brain / Thinking "),
              _cache[126] || (_cache[126] = vue.createElementVNode(
                "path",
                { d: "M12 2a4 4 0 0 0-4 4v1a5 5 0 0 0-5 5v1a4 4 0 0 0 3 3.87V17a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-.13A4 4 0 0 0 21 13v-1a5 5 0 0 0-5-5V6a4 4 0 0 0-4-4z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[127] || (_cache[127] = vue.createElementVNode(
                "path",
                { d: "M9 12v2" },
                null,
                -1
                /* CACHED */
              )),
              _cache[128] || (_cache[128] = vue.createElementVNode(
                "path",
                { d: "M15 12v2" },
                null,
                -1
                /* CACHED */
              )),
              _cache[129] || (_cache[129] = vue.createElementVNode(
                "path",
                { d: "M12 9v5" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "check" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 55 },
            [
              vue.createCommentVNode(" Check / Success "),
              _cache[130] || (_cache[130] = vue.createElementVNode(
                "polyline",
                { points: "20 6 9 17 4 12" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "clock" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 56 },
            [
              vue.createCommentVNode(" Clock / Pending "),
              _cache[131] || (_cache[131] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "10"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[132] || (_cache[132] = vue.createElementVNode(
                "polyline",
                { points: "12 6 12 12 16 14" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "help" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 57 },
            [
              vue.createCommentVNode(" Help / Question "),
              _cache[133] || (_cache[133] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "10"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[134] || (_cache[134] = vue.createElementVNode(
                "path",
                { d: "M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" },
                null,
                -1
                /* CACHED */
              )),
              _cache[135] || (_cache[135] = vue.createElementVNode(
                "line",
                {
                  x1: "12",
                  y1: "17",
                  x2: "12.01",
                  y2: "17"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "shield" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 58 },
            [
              vue.createCommentVNode(" Shield / Approval "),
              _cache[136] || (_cache[136] = vue.createElementVNode(
                "path",
                { d: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "shield-off" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 59 },
            [
              vue.createCommentVNode(" Shield Off / No Review "),
              _cache[137] || (_cache[137] = vue.createElementVNode(
                "path",
                { d: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[138] || (_cache[138] = vue.createElementVNode(
                "line",
                {
                  x1: "4",
                  y1: "4",
                  x2: "20",
                  y2: "20",
                  stroke: "currentColor",
                  "stroke-width": "2",
                  "stroke-linecap": "round"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "code" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 60 },
            [
              vue.createCommentVNode(" Code / Brackets "),
              _cache[139] || (_cache[139] = vue.createElementVNode(
                "polyline",
                { points: "16 18 22 12 16 6" },
                null,
                -1
                /* CACHED */
              )),
              _cache[140] || (_cache[140] = vue.createElementVNode(
                "polyline",
                { points: "8 6 2 12 8 18" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "list" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 61 },
            [
              vue.createCommentVNode(" List / Menu "),
              _cache[141] || (_cache[141] = vue.createStaticVNode('<line x1="8" y1="6" x2="21" y2="6" data-v-faf69761></line><line x1="8" y1="12" x2="21" y2="12" data-v-faf69761></line><line x1="8" y1="18" x2="21" y2="18" data-v-faf69761></line><line x1="3" y1="6" x2="3.01" y2="6" data-v-faf69761></line><line x1="3" y1="12" x2="3.01" y2="12" data-v-faf69761></line><line x1="3" y1="18" x2="3.01" y2="18" data-v-faf69761></line>', 6))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "layers" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 62 },
            [
              vue.createCommentVNode(" Layers / Stack / Context "),
              _cache[142] || (_cache[142] = vue.createElementVNode(
                "polygon",
                { points: "12 2 2 7 12 12 22 7 12 2" },
                null,
                -1
                /* CACHED */
              )),
              _cache[143] || (_cache[143] = vue.createElementVNode(
                "polyline",
                { points: "2 17 12 22 22 17" },
                null,
                -1
                /* CACHED */
              )),
              _cache[144] || (_cache[144] = vue.createElementVNode(
                "polyline",
                { points: "2 12 12 17 22 12" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "eye" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 63 },
            [
              vue.createCommentVNode(" Eye / Show "),
              _cache[145] || (_cache[145] = vue.createElementVNode(
                "path",
                { d: "M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[146] || (_cache[146] = vue.createElementVNode(
                "circle",
                {
                  cx: "12",
                  cy: "12",
                  r: "3"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "eye-off" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 64 },
            [
              vue.createCommentVNode(" Eye Off / Hide "),
              _cache[147] || (_cache[147] = vue.createElementVNode(
                "path",
                { d: "M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" },
                null,
                -1
                /* CACHED */
              )),
              _cache[148] || (_cache[148] = vue.createElementVNode(
                "path",
                { d: "M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" },
                null,
                -1
                /* CACHED */
              )),
              _cache[149] || (_cache[149] = vue.createElementVNode(
                "line",
                {
                  x1: "1",
                  y1: "1",
                  x2: "23",
                  y2: "23"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "bug" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 65 },
            [
              vue.createCommentVNode(" Bug "),
              _cache[150] || (_cache[150] = vue.createStaticVNode('<rect x="8" y="2" width="8" height="4" rx="1" ry="1" data-v-faf69761></rect><path d="M20 12h-3a5 5 0 0 1-5 5 5 5 0 0 1-5-5H4" data-v-faf69761></path><path d="M4 8h16" data-v-faf69761></path><path d="M12 2v7" data-v-faf69761></path><path d="M9 17l-3 4" data-v-faf69761></path><path d="M15 17l3 4" data-v-faf69761></path>', 6))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "check-circle" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 66 },
            [
              vue.createCommentVNode(" Check Circle "),
              _cache[151] || (_cache[151] = vue.createElementVNode(
                "path",
                { d: "M22 11.08V12a10 10 0 1 1-5.93-9.14" },
                null,
                -1
                /* CACHED */
              )),
              _cache[152] || (_cache[152] = vue.createElementVNode(
                "polyline",
                { points: "22 4 12 14.01 9 11.01" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "book-open" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 67 },
            [
              vue.createCommentVNode(" Book Open / Documentation "),
              _cache[153] || (_cache[153] = vue.createElementVNode(
                "path",
                { d: "M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[154] || (_cache[154] = vue.createElementVNode(
                "path",
                { d: "M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "tool" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 68 },
            [
              vue.createCommentVNode(" Tool / Wrench alternate "),
              _cache[155] || (_cache[155] = vue.createElementVNode(
                "path",
                { d: "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "keyboard" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 69 },
            [
              vue.createCommentVNode(" Keyboard "),
              _cache[156] || (_cache[156] = vue.createStaticVNode('<rect x="2" y="4" width="20" height="16" rx="2" ry="2" data-v-faf69761></rect><line x1="6" y1="8" x2="6.01" y2="8" data-v-faf69761></line><line x1="10" y1="8" x2="10.01" y2="8" data-v-faf69761></line><line x1="14" y1="8" x2="14.01" y2="8" data-v-faf69761></line><line x1="18" y1="8" x2="18.01" y2="8" data-v-faf69761></line><line x1="6" y1="12" x2="6.01" y2="12" data-v-faf69761></line><line x1="10" y1="12" x2="10.01" y2="12" data-v-faf69761></line><line x1="14" y1="12" x2="14.01" y2="12" data-v-faf69761></line><line x1="18" y1="12" x2="18.01" y2="12" data-v-faf69761></line><line x1="6" y1="16" x2="18" y2="16" data-v-faf69761></line>', 10))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "chevron-left" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 70 },
            [
              vue.createCommentVNode(" Chevron Left "),
              _cache[157] || (_cache[157] = vue.createElementVNode(
                "polyline",
                { points: "15 6 9 12 15 18" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "grid" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 71 },
            [
              vue.createCommentVNode(" Grid / App Grid "),
              _cache[158] || (_cache[158] = vue.createElementVNode(
                "rect",
                {
                  x: "3",
                  y: "3",
                  width: "7",
                  height: "7"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[159] || (_cache[159] = vue.createElementVNode(
                "rect",
                {
                  x: "14",
                  y: "3",
                  width: "7",
                  height: "7"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[160] || (_cache[160] = vue.createElementVNode(
                "rect",
                {
                  x: "14",
                  y: "14",
                  width: "7",
                  height: "7"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[161] || (_cache[161] = vue.createElementVNode(
                "rect",
                {
                  x: "3",
                  y: "14",
                  width: "7",
                  height: "7"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : __props.name === "puzzle" ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 72 },
            [
              vue.createCommentVNode(" Puzzle / 插件 "),
              _cache[162] || (_cache[162] = vue.createElementVNode(
                "path",
                { d: "M4 7h3a2 2 0 0 1 4 0h9v9h-3a2 2 0 0 0-4 0H4z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[163] || (_cache[163] = vue.createElementVNode(
                "path",
                { d: "M11 7v9" },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          )) : (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 73 },
            [
              _cache[164] || (_cache[164] = vue.createElementVNode(
                "path",
                { d: "M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" },
                null,
                -1
                /* CACHED */
              )),
              _cache[165] || (_cache[165] = vue.createElementVNode(
                "polyline",
                { points: "14 2 14 8 20 8" },
                null,
                -1
                /* CACHED */
              )),
              _cache[166] || (_cache[166] = vue.createElementVNode(
                "line",
                {
                  x1: "9",
                  y1: "13",
                  x2: "15",
                  y2: "13"
                },
                null,
                -1
                /* CACHED */
              )),
              _cache[167] || (_cache[167] = vue.createElementVNode(
                "line",
                {
                  x1: "9",
                  y1: "17",
                  x2: "15",
                  y2: "17"
                },
                null,
                -1
                /* CACHED */
              ))
            ],
            64
            /* STABLE_FRAGMENT */
          ))
        ], 8, _hoisted_1$2);
      };
    }
  };
  const SvgIcon = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-faf69761"]]);
  const _hoisted_1$1 = { class: "modal-header" };
  const _hoisted_2$1 = { class: "modal-title" };
  const _hoisted_3$1 = { class: "modal-body" };
  const _sfc_main$1 = {
    __name: "Modal",
    props: {
      maxWidth: { type: String, default: "480px" }
    },
    emits: ["close"],
    setup(__props, { emit: __emit }) {
      const emit = __emit;
      const visible = vue.ref(true);
      function handleKeydown(e) {
        if (e.key === "Escape") emit("close");
      }
      vue.onMounted(() => document.addEventListener("keydown", handleKeydown));
      vue.onUnmounted(() => document.removeEventListener("keydown", handleKeydown));
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createBlock(vue.Teleport, { to: "body" }, [
          visible.value ? (vue.openBlock(), vue.createElementBlock("div", {
            key: 0,
            class: "modal-overlay",
            onClick: _cache[1] || (_cache[1] = vue.withModifiers(($event) => _ctx.$emit("close"), ["self"]))
          }, [
            vue.createElementVNode(
              "div",
              {
                class: "modal-container",
                style: vue.normalizeStyle({ maxWidth: __props.maxWidth })
              },
              [
                vue.createElementVNode("div", _hoisted_1$1, [
                  vue.createElementVNode("span", _hoisted_2$1, [
                    vue.renderSlot(_ctx.$slots, "title", {}, () => [
                      _cache[2] || (_cache[2] = vue.createTextVNode(
                        "提示",
                        -1
                        /* CACHED */
                      ))
                    ], true)
                  ]),
                  vue.createElementVNode("button", {
                    class: "modal-close",
                    onClick: _cache[0] || (_cache[0] = ($event) => _ctx.$emit("close"))
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "close",
                      size: 14
                    })
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_3$1, [
                  vue.renderSlot(_ctx.$slots, "default", {}, void 0, true)
                ])
              ],
              4
              /* STYLE */
            )
          ])) : vue.createCommentVNode("v-if", true)
        ]);
      };
    }
  };
  const Modal = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-fce3d7ef"]]);
  const _hoisted_1 = { class: "git-panel" };
  const _hoisted_2 = {
    key: 0,
    class: "git-loading"
  };
  const _hoisted_3 = { class: "git-empty" };
  const _hoisted_4 = { class: "git-repo-bar" };
  const _hoisted_5 = ["value"];
  const _hoisted_6 = { class: "branch-name" };
  const _hoisted_7 = {
    key: 1,
    class: "ahead-badge",
    title: "领先上游"
  };
  const _hoisted_8 = {
    key: 2,
    class: "behind-badge",
    title: "落后上游"
  };
  const _hoisted_9 = { class: "repo-actions" };
  const _hoisted_10 = { class: "branch-menu-header" };
  const _hoisted_11 = { class: "branch-list" };
  const _hoisted_12 = ["onClick"];
  const _hoisted_13 = { class: "branch-item-name" };
  const _hoisted_14 = ["onClick"];
  const _hoisted_15 = { class: "branch-menu-footer" };
  const _hoisted_16 = { class: "git-action-bar" };
  const _hoisted_17 = ["disabled"];
  const _hoisted_18 = ["disabled"];
  const _hoisted_19 = { class: "git-sections" };
  const _hoisted_20 = { class: "section-block" };
  const _hoisted_21 = {
    key: 0,
    class: "section-items"
  };
  const _hoisted_22 = ["onClick"];
  const _hoisted_23 = {
    class: /* @__PURE__ */ vue.normalizeClass("file-status staged")
  };
  const _hoisted_24 = { class: "file-path" };
  const _hoisted_25 = { class: "file-actions" };
  const _hoisted_26 = ["onClick"];
  const _hoisted_27 = { class: "section-block" };
  const _hoisted_28 = {
    key: 0,
    class: "section-items"
  };
  const _hoisted_29 = ["onClick"];
  const _hoisted_30 = { class: "file-path conflict-text" };
  const _hoisted_31 = { class: "section-block" };
  const _hoisted_32 = {
    key: 0,
    class: "section-items"
  };
  const _hoisted_33 = ["onClick"];
  const _hoisted_34 = {
    class: /* @__PURE__ */ vue.normalizeClass("file-status modified-st")
  };
  const _hoisted_35 = { class: "file-path" };
  const _hoisted_36 = { class: "file-actions" };
  const _hoisted_37 = ["onClick"];
  const _hoisted_38 = ["onClick"];
  const _hoisted_39 = { class: "section-block" };
  const _hoisted_40 = {
    key: 0,
    class: "section-items"
  };
  const _hoisted_41 = ["onClick"];
  const _hoisted_42 = { class: "file-path untracked-text" };
  const _hoisted_43 = { class: "file-actions" };
  const _hoisted_44 = ["onClick"];
  const _hoisted_45 = {
    key: 0,
    class: "clean-hint"
  };
  const _hoisted_46 = { class: "git-history" };
  const _hoisted_47 = {
    key: 0,
    class: "history-list"
  };
  const _hoisted_48 = ["onDblclick"];
  const _hoisted_49 = { class: "commit-hash" };
  const _hoisted_50 = { class: "commit-msg" };
  const _hoisted_51 = { class: "commit-date" };
  const _hoisted_52 = { class: "form-layout" };
  const _hoisted_53 = { class: "form-hint" };
  const _hoisted_54 = { class: "form-actions" };
  const _hoisted_55 = ["disabled"];
  const _hoisted_56 = { class: "form-layout" };
  const _hoisted_57 = ["placeholder"];
  const _hoisted_58 = { class: "form-actions" };
  const _hoisted_59 = { class: "form-layout" };
  const _hoisted_60 = { class: "form-checkbox" };
  const _hoisted_61 = { class: "form-actions" };
  const _hoisted_62 = ["disabled"];
  const _hoisted_63 = { class: "overlay-panel stash-panel" };
  const _hoisted_64 = { class: "overlay-header" };
  const _hoisted_65 = { class: "stash-form" };
  const _hoisted_66 = { class: "stash-list" };
  const _hoisted_67 = { class: "stash-ref" };
  const _hoisted_68 = { class: "stash-msg" };
  const _hoisted_69 = { class: "stash-actions" };
  const _hoisted_70 = ["onClick"];
  const _hoisted_71 = ["onClick"];
  const _hoisted_72 = {
    key: 0,
    class: "stash-empty"
  };
  const _hoisted_73 = { class: "overlay-panel ignore-panel" };
  const _hoisted_74 = { class: "overlay-header" };
  const _hoisted_75 = { class: "ignore-actions" };
  const _hoisted_76 = { class: "detail-content" };
  const _hoisted_77 = { class: "detail-meta" };
  const _hoisted_78 = { class: "detail-diff" };
  const _hoisted_79 = {
    class: "form-actions",
    style: { "padding": "8px 16px" }
  };
  const _hoisted_80 = { class: "diff-title" };
  const _hoisted_81 = { class: "diff-content" };
  const _hoisted_82 = { class: "diff-text" };
  const _hoisted_83 = {
    class: "form-actions",
    style: { "padding": "8px 16px" }
  };
  const _sfc_main = {
    __name: "GitPanel",
    setup(__props) {
      const loading = vue.ref(false);
      const refreshing = vue.ref(false);
      const hasData = vue.ref(false);
      const isRepo = vue.ref(false);
      const currentBranch = vue.ref("");
      const ahead = vue.ref(0);
      const behind = vue.ref(0);
      const staged = vue.ref([]);
      const conflict = vue.ref([]);
      const modified = vue.ref([]);
      const untracked = vue.ref([]);
      const branches = vue.ref([]);
      const commits = vue.ref([]);
      const error = vue.ref("");
      const collapsed = vue.reactive({
        staged: false,
        conflict: false,
        modified: false,
        untracked: false,
        history: false
      });
      const showBranchMenu = vue.ref(false);
      const branchFilter = vue.ref("");
      const showCommitDialog = vue.ref(false);
      const showPushDialog = vue.ref(false);
      const showCreateBranch = vue.ref(false);
      const showStashPanel = vue.ref(false);
      const showIgnoreEditor = vue.ref(false);
      const showCommitDetailModal = vue.ref(false);
      const detailCommit = vue.ref(null);
      const commitDiff = vue.ref("");
      const showDiffDialog = vue.ref(false);
      const diffFilePath = vue.ref("");
      const diffContent = vue.ref("");
      const commitMsg = vue.ref("");
      const commitDesc = vue.ref("");
      const pushRemote = vue.ref("origin");
      const pushBranch = vue.ref("");
      const newBranchName = vue.ref("");
      const switchAfterCreate = vue.ref(true);
      const stashMsg = vue.ref("");
      const stashes = vue.ref([]);
      const ignoreContent = vue.ref("");
      const workspaceProjects = vue.computed(() => [...new Set((uiState_js.state.workspaceFolders || []).filter(Boolean))]);
      const gitProject = vue.ref("");
      function gitParams(extra = {}) {
        if (gitProject.value) extra.path = gitProject.value;
        return extra;
      }
      let refreshTimer = null;
      const hasModified = vue.computed(() => modified.value.length + untracked.value.length > 0);
      const stagedCount = vue.computed(() => staged.value.length);
      const totalChanges = vue.computed(() => staged.value.length + conflict.value.length + modified.value.length + untracked.value.length);
      const filteredBranches = vue.computed(() => {
        if (!branchFilter.value) return branches.value;
        const f = branchFilter.value.toLowerCase();
        return branches.value.filter((b) => b.toLowerCase().includes(f));
      });
      vue.onMounted(() => {
        if (workspaceProjects.value.length > 0) {
          gitProject.value = workspaceProjects.value[0];
        }
        loadStatus();
        refreshTimer = setInterval(loadStatus, 3e4);
        document.addEventListener("click", handleOutsideClick);
      });
      vue.onUnmounted(() => {
        if (refreshTimer) clearInterval(refreshTimer);
        document.removeEventListener("click", handleOutsideClick);
      });
      function handleOutsideClick() {
        if (showBranchMenu.value) showBranchMenu.value = false;
      }
      async function loadStatus() {
        if (loading.value) return;
        loading.value = true;
        try {
          const res = await api.apiGet("/git/status", gitParams());
          hasData.value = true;
          isRepo.value = res.isRepo || false;
          if (isRepo.value) {
            currentBranch.value = res.branch || "";
            ahead.value = res.ahead || 0;
            behind.value = res.behind || 0;
            staged.value = res.staged || [];
            conflict.value = res.conflict || [];
            modified.value = res.modified || [];
            untracked.value = res.untracked || [];
            branches.value = res.branches || [];
            error.value = res.error || "";
          } else {
            error.value = res.error || "非 Git 仓库";
          }
          if (isRepo.value) {
            try {
              const log = await api.apiGet("/git-log", gitParams({ count: 50 }));
              commits.value = log || [];
            } catch (err) {
              console.warn("[GitPanel] 加载提交历史失败:", err);
            }
          }
        } catch (err) {
          hasData.value = true;
          error.value = err.message;
          isRepo.value = false;
        } finally {
          loading.value = false;
          refreshing.value = false;
        }
      }
      async function refresh() {
        refreshing.value = true;
        await loadStatus();
        if (showStashPanel.value) await loadStashes();
      }
      async function refreshCommits() {
        try {
          commits.value = await api.apiGet("/git-log", gitParams({ count: 50 })) || [];
        } catch (err) {
          console.warn("[GitPanel] 刷新提交历史失败:", err);
        }
      }
      async function loadStashes() {
        try {
          stashes.value = await api.apiGet("/git/stash-list", gitParams()) || [];
        } catch (err) {
          console.warn("[GitPanel] 加载暂存列表失败:", err);
          stashes.value = [];
        }
      }
      async function stageAll() {
        var _a, _b;
        try {
          await api.apiPost("/git/add", { files: [] }, gitParams());
          await loadStatus();
          (_a = window.$toast) == null ? void 0 : _a.call(window, "已全部暂存", "success");
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "暂存失败: " + err.message, "error");
        }
      }
      async function stageFile(path) {
        var _a;
        try {
          await api.apiPost("/git/add", { files: [path] }, gitParams());
          await loadStatus();
        } catch (err) {
          (_a = window.$toast) == null ? void 0 : _a.call(window, "暂存失败: " + err.message, "error");
        }
      }
      async function unstageFile(path) {
        var _a;
        try {
          await api.apiPost("/git/reset", { files: [path] }, gitParams());
          await loadStatus();
        } catch (err) {
          (_a = window.$toast) == null ? void 0 : _a.call(window, "取消暂存失败: " + err.message, "error");
        }
      }
      async function discardFile(path) {
        var _a, _b;
        const ok = await ((_a = window.$confirm) == null ? void 0 : _a.call(window, `确定丢弃「${path}」的工作区更改？不可撤销。`, "丢弃更改", "确定丢弃", "取消"));
        if (!ok) return;
        try {
          await api.apiPost("/git/discard", { files: [path] }, gitParams());
          await loadStatus();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "丢弃失败: " + err.message, "error");
        }
      }
      async function doCommit() {
        var _a, _b;
        if (!commitMsg.value.trim()) return;
        try {
          await api.apiPost("/git/commit", { message: commitMsg.value, all: false }, gitParams());
          commitMsg.value = "";
          commitDesc.value = "";
          showCommitDialog.value = false;
          (_a = window.$toast) == null ? void 0 : _a.call(window, "提交成功", "success");
          await loadStatus();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "提交失败: " + err.message, "error");
        }
      }
      async function switchBranch(name) {
        var _a;
        if (name === currentBranch.value) return;
        try {
          await api.apiPost("/git/branch", { action: "switch", name }, gitParams());
          showBranchMenu.value = false;
          await loadStatus();
        } catch (err) {
          (_a = window.$toast) == null ? void 0 : _a.call(window, "切换分支失败: " + err.message, "error");
        }
      }
      async function deleteBranch(name) {
        var _a, _b;
        const ok = await ((_a = window.$confirm) == null ? void 0 : _a.call(window, `确定删除分支「${name}」？`, "删除分支", "确定删除", "取消"));
        if (!ok) return;
        try {
          await api.apiPost("/git/branch", { action: "delete", name }, gitParams());
          await loadStatus();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "删除分支失败: " + err.message, "error");
        }
      }
      async function createBranch() {
        var _a;
        if (!newBranchName.value.trim()) return;
        try {
          if (switchAfterCreate.value) {
            await api.apiPost("/git/branch", { action: "create-switch", name: newBranchName.value }, gitParams());
          } else {
            await api.apiPost("/git/branch", { action: "create", name: newBranchName.value }, gitParams());
          }
          newBranchName.value = "";
          showCreateBranch.value = false;
          await loadStatus();
        } catch (err) {
          (_a = window.$toast) == null ? void 0 : _a.call(window, "创建分支失败: " + err.message, "error");
        }
      }
      async function doPush() {
        var _a, _b;
        try {
          const body = {};
          if (pushRemote.value && pushRemote.value !== "origin") body.remote = pushRemote.value;
          if (pushBranch.value) body.branch = pushBranch.value;
          await api.apiPost("/git/push", body, gitParams());
          showPushDialog.value = false;
          (_a = window.$toast) == null ? void 0 : _a.call(window, "推送成功", "success");
          await loadStatus();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "推送失败: " + err.message, "error");
        }
      }
      async function pull() {
        var _a, _b;
        try {
          await api.apiPost("/git/pull", {}, gitParams());
          (_a = window.$toast) == null ? void 0 : _a.call(window, "拉取成功", "success");
          await loadStatus();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "拉取失败: " + err.message, "error");
        }
      }
      async function initRepo() {
        var _a, _b, _c;
        try {
          const res = await api.apiPost("/git/init", {}, gitParams());
          if (res && !res.error) {
            (_a = window.$toast) == null ? void 0 : _a.call(window, "Git 仓库已初始化", "success");
            await loadStatus();
          } else {
            (_b = window.$toast) == null ? void 0 : _b.call(window, "初始化失败: " + ((res == null ? void 0 : res.error) || "未知错误"), "error");
          }
        } catch (err) {
          (_c = window.$toast) == null ? void 0 : _c.call(window, "初始化失败: " + err.message, "error");
        }
      }
      async function stashPush() {
        var _a, _b;
        try {
          await api.apiPost("/git/stash", { action: "push", message: stashMsg.value }, gitParams());
          stashMsg.value = "";
          (_a = window.$toast) == null ? void 0 : _a.call(window, "已暂存", "success");
          await loadStatus();
          await loadStashes();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "暂存失败: " + err.message, "error");
        }
      }
      async function stashPop(index) {
        var _a, _b;
        try {
          await api.apiPost("/git/stash", { action: "pop", index }, gitParams());
          (_a = window.$toast) == null ? void 0 : _a.call(window, "已弹出暂存", "success");
          await loadStatus();
          await loadStashes();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "弹出失败: " + err.message, "error");
        }
      }
      async function stashDrop(index) {
        var _a, _b;
        const ok = await ((_a = window.$confirm) == null ? void 0 : _a.call(window, `确定删除暂存 ${index}？`, "删除暂存", "确定", "取消"));
        if (!ok) return;
        try {
          await api.apiPost("/git/stash", { action: "drop", index }, gitParams());
          await loadStashes();
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "删除失败: " + err.message, "error");
        }
      }
      async function saveIgnore() {
        var _a, _b;
        try {
          await api.apiPost("/git/ignore", { content: ignoreContent.value }, gitParams());
          showIgnoreEditor.value = false;
          (_a = window.$toast) == null ? void 0 : _a.call(window, ".gitignore 已保存", "success");
        } catch (err) {
          (_b = window.$toast) == null ? void 0 : _b.call(window, "保存失败: " + err.message, "error");
        }
      }
      async function loadIgnore() {
        try {
          const res = await api.apiGet("/git/ignore", gitParams());
          ignoreContent.value = res.content || "";
        } catch (err) {
          console.warn("[GitPanel] 加载 .gitignore 失败:", err);
          ignoreContent.value = "";
        }
      }
      async function showFileDiff(path, staged2) {
        diffFilePath.value = path;
        diffContent.value = "加载中...";
        showDiffDialog.value = true;
        try {
          const res = await api.apiGet("/git/diff", gitParams({ file: path, staged: staged2 ? "true" : "false" }));
          diffContent.value = res.diff || "（无差异）";
        } catch (err) {
          diffContent.value = "无法加载差异: " + err.message;
        }
      }
      async function showCommitDetail(c) {
        detailCommit.value = c;
        commitDiff.value = c.msg || "（无内容）";
        showCommitDetailModal.value = true;
      }
      async function copyHash(hash) {
        var _a;
        if (!hash) return;
        try {
          await navigator.clipboard.writeText(hash);
          (_a = window.$toast) == null ? void 0 : _a.call(window, "已复制", "success");
        } catch (err) {
          console.warn("[GitPanel] 复制哈希失败:", err);
        }
      }
      function toggleCollapse(key) {
        collapsed[key] = !collapsed[key];
      }
      function formatDate(d, full) {
        if (!d) return "";
        try {
          const dt = new Date(d);
          if (!isNaN(dt.getTime())) {
            if (full) {
              return dt.toLocaleString("zh-CN", {
                year: "numeric",
                month: "2-digit",
                day: "2-digit",
                hour: "2-digit",
                minute: "2-digit"
              });
            }
            return dt.toLocaleDateString("zh-CN", { month: "2-digit", day: "2-digit" });
          }
          return d.substring(0, 10);
        } catch {
          return d ? d.substring(0, 10) : "";
        }
      }
      function statusIcon(s) {
        if (s === "M" || s === "m") return "~";
        if (s === "A" || s === "a") return "+";
        if (s === "D" || s === "d") return "-";
        if (s === "R" || s === "r") return "→";
        if (s === "?" || s === "!") return s;
        return "~";
      }
      vue.watch(showStashPanel, (v) => {
        if (v) loadStashes();
      });
      vue.watch(showIgnoreEditor, (v) => {
        if (v) loadIgnore();
      });
      vue.watch(() => uiState_js.state.workspaceRoot, () => {
        if (workspaceProjects.value.length > 0) gitProject.value = workspaceProjects.value[0];
        loadStatus();
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1, [
          vue.createCommentVNode(" 加载中 "),
          loading.value && !hasData.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_2, [
            vue.createVNode(SvgIcon, {
              name: "refresh",
              size: 20,
              class: "spinner"
            }),
            _cache[39] || (_cache[39] = vue.createElementVNode(
              "span",
              null,
              "加载 Git 状态...",
              -1
              /* CACHED */
            ))
          ])) : !isRepo.value && hasData.value ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 1 },
            [
              vue.createCommentVNode(" 非 Git 仓库 "),
              vue.createElementVNode("div", _hoisted_3, [
                vue.createVNode(SvgIcon, {
                  name: "source-control",
                  size: 24,
                  color: "var(--text-muted)"
                }),
                _cache[40] || (_cache[40] = vue.createElementVNode(
                  "span",
                  null,
                  "非 Git 仓库",
                  -1
                  /* CACHED */
                )),
                _cache[41] || (_cache[41] = vue.createElementVNode(
                  "span",
                  { class: "subtitle" },
                  "此目录未初始化 Git",
                  -1
                  /* CACHED */
                )),
                vue.createElementVNode("button", {
                  class: "git-btn init-btn",
                  onClick: initRepo
                }, "初始化仓库")
              ])
            ],
            2112
            /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
          )) : isRepo.value ? (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            { key: 2 },
            [
              vue.createCommentVNode(" Git 面板主体 "),
              vue.createCommentVNode(" 仓库顶栏 "),
              vue.createElementVNode("div", _hoisted_4, [
                vue.createVNode(SvgIcon, {
                  name: "source-control",
                  size: 14,
                  color: "var(--accent)"
                }),
                vue.createCommentVNode(" 项目选择器（多根工作区时显示） "),
                workspaceProjects.value.length > 1 ? vue.withDirectives((vue.openBlock(), vue.createElementBlock(
                  "select",
                  {
                    key: 0,
                    "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => gitProject.value = $event),
                    class: "git-project-select",
                    onChange: loadStatus
                  },
                  [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(workspaceProjects.value, (p) => {
                        return vue.openBlock(), vue.createElementBlock("option", {
                          key: p,
                          value: p
                        }, vue.toDisplayString(p.split("\\").pop() || p), 9, _hoisted_5);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ],
                  544
                  /* NEED_HYDRATION, NEED_PATCH */
                )), [
                  [vue.vModelSelect, gitProject.value]
                ]) : vue.createCommentVNode("v-if", true),
                vue.createElementVNode("div", {
                  class: "git-branch-select",
                  onClick: _cache[1] || (_cache[1] = vue.withModifiers(($event) => showBranchMenu.value = !showBranchMenu.value, ["stop"]))
                }, [
                  _cache[42] || (_cache[42] = vue.createTextVNode(
                    '"> ',
                    -1
                    /* CACHED */
                  )),
                  vue.createVNode(SvgIcon, {
                    name: "git-branch",
                    size: 12
                  }),
                  vue.createElementVNode(
                    "span",
                    _hoisted_6,
                    vue.toDisplayString(currentBranch.value || "（无分支）"),
                    1
                    /* TEXT */
                  ),
                  vue.createVNode(SvgIcon, {
                    name: "chevron-down",
                    size: 10
                  })
                ]),
                ahead.value > 0 ? (vue.openBlock(), vue.createElementBlock(
                  "span",
                  _hoisted_7,
                  "↑" + vue.toDisplayString(ahead.value),
                  1
                  /* TEXT */
                )) : vue.createCommentVNode("v-if", true),
                behind.value > 0 ? (vue.openBlock(), vue.createElementBlock(
                  "span",
                  _hoisted_8,
                  "↓" + vue.toDisplayString(behind.value),
                  1
                  /* TEXT */
                )) : vue.createCommentVNode("v-if", true),
                vue.createElementVNode("div", _hoisted_9, [
                  vue.createElementVNode("button", {
                    class: "icon-btn",
                    onClick: refresh,
                    title: "刷新"
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "refresh",
                      size: 13,
                      class: vue.normalizeClass({ spinning: refreshing.value })
                    }, null, 8, ["class"])
                  ])
                ])
              ]),
              vue.createCommentVNode(" 分支管理菜单 "),
              showBranchMenu.value ? (vue.openBlock(), vue.createElementBlock("div", {
                key: 0,
                class: "branch-menu",
                onClick: _cache[5] || (_cache[5] = vue.withModifiers(() => {
                }, ["stop"]))
              }, [
                vue.createElementVNode("div", _hoisted_10, [
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => branchFilter.value = $event),
                      placeholder: "过滤分支...",
                      class: "branch-filter-input"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, branchFilter.value]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_11, [
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(filteredBranches.value, (b) => {
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: b,
                        class: vue.normalizeClass(["branch-item", { active: b === currentBranch.value }]),
                        onClick: ($event) => switchBranch(b)
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: b === currentBranch.value ? "check" : "git-branch",
                          size: 12
                        }, null, 8, ["name"]),
                        vue.createElementVNode(
                          "span",
                          _hoisted_13,
                          vue.toDisplayString(b),
                          1
                          /* TEXT */
                        ),
                        b !== currentBranch.value ? (vue.openBlock(), vue.createElementBlock("button", {
                          key: 0,
                          class: "branch-del-btn",
                          onClick: vue.withModifiers(($event) => deleteBranch(b), ["stop"]),
                          title: "删除分支"
                        }, [
                          vue.createVNode(SvgIcon, {
                            name: "close",
                            size: 10
                          })
                        ], 8, _hoisted_14)) : vue.createCommentVNode("v-if", true)
                      ], 10, _hoisted_12);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  ))
                ]),
                vue.createElementVNode("div", _hoisted_15, [
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[3] || (_cache[3] = ($event) => showCreateBranch.value = true)
                  }, "新建分支"),
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[4] || (_cache[4] = ($event) => showBranchMenu.value = false)
                  }, "关闭")
                ])
              ])) : vue.createCommentVNode("v-if", true),
              vue.createCommentVNode(" 操作栏 "),
              vue.createElementVNode("div", _hoisted_16, [
                vue.createElementVNode("button", {
                  class: "git-btn action-btn",
                  disabled: !hasModified.value,
                  onClick: stageAll
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "plus",
                    size: 12
                  }),
                  _cache[43] || (_cache[43] = vue.createTextVNode(
                    " 全部暂存 ",
                    -1
                    /* CACHED */
                  ))
                ], 8, _hoisted_17),
                vue.createElementVNode("button", {
                  class: "git-btn action-btn commit-btn",
                  disabled: stagedCount.value === 0,
                  onClick: _cache[6] || (_cache[6] = ($event) => showCommitDialog.value = true)
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "check",
                    size: 12
                  }),
                  _cache[44] || (_cache[44] = vue.createTextVNode(
                    " 提交 ",
                    -1
                    /* CACHED */
                  ))
                ], 8, _hoisted_18),
                _cache[45] || (_cache[45] = vue.createElementVNode(
                  "div",
                  { class: "action-spacer" },
                  null,
                  -1
                  /* CACHED */
                )),
                vue.createElementVNode("button", {
                  class: "icon-btn",
                  onClick: pull,
                  title: "拉取"
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "git-pull",
                    size: 13
                  })
                ]),
                vue.createElementVNode("button", {
                  class: "icon-btn",
                  onClick: _cache[7] || (_cache[7] = ($event) => showPushDialog.value = true),
                  title: "推送"
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "git-push",
                    size: 13
                  })
                ]),
                vue.createElementVNode("button", {
                  class: "icon-btn",
                  onClick: _cache[8] || (_cache[8] = ($event) => showStashPanel.value = !showStashPanel.value),
                  title: "暂存管理"
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "package",
                    size: 13
                  })
                ]),
                vue.createElementVNode("button", {
                  class: "icon-btn",
                  onClick: _cache[9] || (_cache[9] = ($event) => showIgnoreEditor.value = !showIgnoreEditor.value),
                  title: ".gitignore"
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "file-text",
                    size: 13
                  })
                ])
              ]),
              vue.createCommentVNode(" 变更区块 "),
              vue.createElementVNode("div", _hoisted_19, [
                vue.createCommentVNode(" 已暂存 "),
                vue.createElementVNode("div", _hoisted_20, [
                  vue.createElementVNode("div", {
                    class: "section-header",
                    onClick: _cache[10] || (_cache[10] = ($event) => toggleCollapse("staged"))
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: collapsed.staged ? "chevron-right" : "chevron-down",
                      size: 12,
                      color: "var(--accent)"
                    }, null, 8, ["name"]),
                    vue.createElementVNode(
                      "span",
                      null,
                      "已暂存 (" + vue.toDisplayString(staged.value.length) + ")",
                      1
                      /* TEXT */
                    )
                  ]),
                  !collapsed.staged ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_21, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(staged.value, (item) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: item.path,
                          class: "file-row",
                          onClick: ($event) => showFileDiff(item.path, true)
                        }, [
                          vue.createElementVNode(
                            "span",
                            _hoisted_23,
                            vue.toDisplayString(statusIcon(item.x)),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode(
                            "span",
                            _hoisted_24,
                            vue.toDisplayString(item.path),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode("div", _hoisted_25, [
                            vue.createElementVNode("button", {
                              class: "row-btn",
                              onClick: vue.withModifiers(($event) => unstageFile(item.path), ["stop"]),
                              title: "取消暂存"
                            }, [
                              vue.createVNode(SvgIcon, {
                                name: "minus",
                                size: 12
                              })
                            ], 8, _hoisted_26)
                          ])
                        ], 8, _hoisted_22);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])) : vue.createCommentVNode("v-if", true)
                ]),
                vue.createCommentVNode(" 冲突 "),
                vue.createElementVNode("div", _hoisted_27, [
                  vue.createElementVNode("div", {
                    class: "section-header conflict",
                    onClick: _cache[11] || (_cache[11] = ($event) => toggleCollapse("conflict"))
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: collapsed.conflict ? "chevron-right" : "chevron-down",
                      size: 12,
                      color: "#f48771"
                    }, null, 8, ["name"]),
                    vue.createElementVNode(
                      "span",
                      null,
                      "冲突 (" + vue.toDisplayString(conflict.value.length) + ")",
                      1
                      /* TEXT */
                    )
                  ]),
                  !collapsed.conflict ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_28, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(conflict.value, (item) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: item.path,
                          class: "file-row",
                          onClick: ($event) => showFileDiff(item.path, false)
                        }, [
                          _cache[46] || (_cache[46] = vue.createElementVNode(
                            "span",
                            { class: "file-status conflict-st" },
                            "!",
                            -1
                            /* CACHED */
                          )),
                          vue.createElementVNode(
                            "span",
                            _hoisted_30,
                            vue.toDisplayString(item.path),
                            1
                            /* TEXT */
                          )
                        ], 8, _hoisted_29);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])) : vue.createCommentVNode("v-if", true)
                ]),
                vue.createCommentVNode(" 已修改 "),
                vue.createElementVNode("div", _hoisted_31, [
                  vue.createElementVNode("div", {
                    class: "section-header modified",
                    onClick: _cache[12] || (_cache[12] = ($event) => toggleCollapse("modified"))
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: collapsed.modified ? "chevron-right" : "chevron-down",
                      size: 12,
                      color: "#dcdcaa"
                    }, null, 8, ["name"]),
                    vue.createElementVNode(
                      "span",
                      null,
                      "已修改 (" + vue.toDisplayString(modified.value.length) + ")",
                      1
                      /* TEXT */
                    )
                  ]),
                  !collapsed.modified ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_32, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(modified.value, (item) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: item.path,
                          class: "file-row",
                          onClick: ($event) => showFileDiff(item.path, false)
                        }, [
                          vue.createElementVNode(
                            "span",
                            _hoisted_34,
                            vue.toDisplayString(statusIcon(item.y)),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode(
                            "span",
                            _hoisted_35,
                            vue.toDisplayString(item.path),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode("div", _hoisted_36, [
                            vue.createElementVNode("button", {
                              class: "row-btn",
                              onClick: vue.withModifiers(($event) => stageFile(item.path), ["stop"]),
                              title: "暂存"
                            }, [
                              vue.createVNode(SvgIcon, {
                                name: "plus",
                                size: 12
                              })
                            ], 8, _hoisted_37),
                            vue.createElementVNode("button", {
                              class: "row-btn danger",
                              onClick: vue.withModifiers(($event) => discardFile(item.path), ["stop"]),
                              title: "丢弃"
                            }, [
                              vue.createVNode(SvgIcon, {
                                name: "trash",
                                size: 12
                              })
                            ], 8, _hoisted_38)
                          ])
                        ], 8, _hoisted_33);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])) : vue.createCommentVNode("v-if", true)
                ]),
                vue.createCommentVNode(" 未跟踪 "),
                vue.createElementVNode("div", _hoisted_39, [
                  vue.createElementVNode("div", {
                    class: "section-header untracked",
                    onClick: _cache[13] || (_cache[13] = ($event) => toggleCollapse("untracked"))
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: collapsed.untracked ? "chevron-right" : "chevron-down",
                      size: 12,
                      color: "var(--text-muted)"
                    }, null, 8, ["name"]),
                    vue.createElementVNode(
                      "span",
                      null,
                      "未跟踪 (" + vue.toDisplayString(untracked.value.length) + ")",
                      1
                      /* TEXT */
                    )
                  ]),
                  !collapsed.untracked ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_40, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(untracked.value, (item) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: item.path,
                          class: "file-row",
                          onClick: ($event) => showFileDiff(item.path, false)
                        }, [
                          _cache[47] || (_cache[47] = vue.createElementVNode(
                            "span",
                            { class: "file-status untracked-st" },
                            "?",
                            -1
                            /* CACHED */
                          )),
                          vue.createElementVNode(
                            "span",
                            _hoisted_42,
                            vue.toDisplayString(item.path),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode("div", _hoisted_43, [
                            vue.createElementVNode("button", {
                              class: "row-btn",
                              onClick: vue.withModifiers(($event) => stageFile(item.path), ["stop"]),
                              title: "暂存"
                            }, [
                              vue.createVNode(SvgIcon, {
                                name: "plus",
                                size: 12
                              })
                            ], 8, _hoisted_44)
                          ])
                        ], 8, _hoisted_41);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])) : vue.createCommentVNode("v-if", true)
                ]),
                vue.createCommentVNode(" 工作区干净 "),
                totalChanges.value === 0 && commits.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_45, [
                  vue.createVNode(SvgIcon, {
                    name: "check",
                    size: 14,
                    color: "var(--accent)"
                  }),
                  _cache[48] || (_cache[48] = vue.createElementVNode(
                    "span",
                    null,
                    "工作区干净",
                    -1
                    /* CACHED */
                  ))
                ])) : vue.createCommentVNode("v-if", true)
              ]),
              vue.createCommentVNode(" 提交历史 "),
              vue.createElementVNode("div", _hoisted_46, [
                vue.createElementVNode("div", {
                  class: "history-header",
                  onClick: _cache[14] || (_cache[14] = ($event) => toggleCollapse("history"))
                }, [
                  vue.createVNode(SvgIcon, {
                    name: collapsed.history ? "chevron-right" : "chevron-down",
                    size: 12,
                    color: "var(--accent)"
                  }, null, 8, ["name"]),
                  vue.createElementVNode(
                    "span",
                    null,
                    "提交历史 (" + vue.toDisplayString(commits.value.length) + ")",
                    1
                    /* TEXT */
                  ),
                  vue.createElementVNode("button", {
                    class: "icon-btn",
                    onClick: vue.withModifiers(refreshCommits, ["stop"]),
                    title: "刷新提交历史"
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "refresh",
                      size: 11
                    })
                  ])
                ]),
                !collapsed.history ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_47, [
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(commits.value, (c) => {
                      var _a;
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: c.hash,
                        class: "commit-row",
                        onDblclick: ($event) => showCommitDetail(c)
                      }, [
                        vue.createElementVNode(
                          "span",
                          _hoisted_49,
                          vue.toDisplayString(c.short),
                          1
                          /* TEXT */
                        ),
                        vue.createElementVNode(
                          "span",
                          _hoisted_50,
                          vue.toDisplayString((_a = c.msg) == null ? void 0 : _a.split("\n")[0]),
                          1
                          /* TEXT */
                        ),
                        vue.createElementVNode(
                          "span",
                          _hoisted_51,
                          vue.toDisplayString(formatDate(c.date)),
                          1
                          /* TEXT */
                        )
                      ], 40, _hoisted_48);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  ))
                ])) : vue.createCommentVNode("v-if", true)
              ])
            ],
            64
            /* STABLE_FRAGMENT */
          )) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" ≡≡≡ 对话框区域 ≡≡≡ "),
          vue.createCommentVNode(" 提交对话框 "),
          showCommitDialog.value ? (vue.openBlock(), vue.createBlock(Modal, {
            key: 3,
            onClose: _cache[18] || (_cache[18] = ($event) => showCommitDialog.value = false),
            maxWidth: "420px"
          }, {
            title: vue.withCtx(() => [..._cache[49] || (_cache[49] = [
              vue.createTextVNode(
                "提交变更",
                -1
                /* CACHED */
              )
            ])]),
            default: vue.withCtx(() => [
              vue.createElementVNode("div", _hoisted_52, [
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[15] || (_cache[15] = ($event) => commitMsg.value = $event),
                    placeholder: "提交信息（必填）",
                    class: "form-input",
                    onKeyup: vue.withKeys(doCommit, ["enter"])
                  },
                  null,
                  544
                  /* NEED_HYDRATION, NEED_PATCH */
                ), [
                  [vue.vModelText, commitMsg.value]
                ]),
                vue.withDirectives(vue.createElementVNode(
                  "textarea",
                  {
                    "onUpdate:modelValue": _cache[16] || (_cache[16] = ($event) => commitDesc.value = $event),
                    placeholder: "详细描述（可选）",
                    class: "form-textarea",
                    rows: "3"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, commitDesc.value]
                ]),
                vue.createElementVNode(
                  "div",
                  _hoisted_53,
                  vue.toDisplayString(stagedCount.value) + " 项已暂存",
                  1
                  /* TEXT */
                ),
                vue.createElementVNode("div", _hoisted_54, [
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[17] || (_cache[17] = ($event) => showCommitDialog.value = false)
                  }, "取消"),
                  vue.createElementVNode("button", {
                    class: "git-btn btn-primary",
                    disabled: !commitMsg.value.trim(),
                    onClick: doCommit
                  }, "提交", 8, _hoisted_55)
                ])
              ])
            ]),
            _: 1
            /* STABLE */
          })) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 推送对话框 "),
          showPushDialog.value ? (vue.openBlock(), vue.createBlock(Modal, {
            key: 4,
            onClose: _cache[22] || (_cache[22] = ($event) => showPushDialog.value = false),
            maxWidth: "380px"
          }, {
            title: vue.withCtx(() => [..._cache[50] || (_cache[50] = [
              vue.createTextVNode(
                "推送",
                -1
                /* CACHED */
              )
            ])]),
            default: vue.withCtx(() => [
              vue.createElementVNode("div", _hoisted_56, [
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[19] || (_cache[19] = ($event) => pushRemote.value = $event),
                    placeholder: "远程仓库（默认 origin）",
                    class: "form-input"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, pushRemote.value]
                ]),
                vue.withDirectives(vue.createElementVNode("input", {
                  "onUpdate:modelValue": _cache[20] || (_cache[20] = ($event) => pushBranch.value = $event),
                  placeholder: "分支（默认 " + currentBranch.value + "）",
                  class: "form-input"
                }, null, 8, _hoisted_57), [
                  [vue.vModelText, pushBranch.value]
                ]),
                vue.createElementVNode("div", _hoisted_58, [
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[21] || (_cache[21] = ($event) => showPushDialog.value = false)
                  }, "取消"),
                  vue.createElementVNode("button", {
                    class: "git-btn btn-primary",
                    onClick: doPush
                  }, "推送")
                ])
              ])
            ]),
            _: 1
            /* STABLE */
          })) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 创建分支对话框 "),
          showCreateBranch.value ? (vue.openBlock(), vue.createBlock(Modal, {
            key: 5,
            onClose: _cache[26] || (_cache[26] = ($event) => showCreateBranch.value = false),
            maxWidth: "360px"
          }, {
            title: vue.withCtx(() => [..._cache[51] || (_cache[51] = [
              vue.createTextVNode(
                "新建分支",
                -1
                /* CACHED */
              )
            ])]),
            default: vue.withCtx(() => [
              vue.createElementVNode("div", _hoisted_59, [
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[23] || (_cache[23] = ($event) => newBranchName.value = $event),
                    placeholder: "分支名",
                    class: "form-input",
                    onKeyup: vue.withKeys(createBranch, ["enter"])
                  },
                  null,
                  544
                  /* NEED_HYDRATION, NEED_PATCH */
                ), [
                  [vue.vModelText, newBranchName.value]
                ]),
                vue.createElementVNode("label", _hoisted_60, [
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[24] || (_cache[24] = ($event) => switchAfterCreate.value = $event),
                      type: "checkbox"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelCheckbox, switchAfterCreate.value]
                  ]),
                  _cache[52] || (_cache[52] = vue.createTextVNode(
                    " 创建后切换 ",
                    -1
                    /* CACHED */
                  ))
                ]),
                vue.createElementVNode("div", _hoisted_61, [
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[25] || (_cache[25] = ($event) => showCreateBranch.value = false)
                  }, "取消"),
                  vue.createElementVNode("button", {
                    class: "git-btn btn-primary",
                    disabled: !newBranchName.value.trim(),
                    onClick: createBranch
                  }, "创建", 8, _hoisted_62)
                ])
              ])
            ]),
            _: 1
            /* STABLE */
          })) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 暂存管理面板 "),
          (vue.openBlock(), vue.createBlock(vue.Teleport, { to: "body" }, [
            showStashPanel.value ? (vue.openBlock(), vue.createElementBlock("div", {
              key: 0,
              class: "overlay",
              onClick: _cache[29] || (_cache[29] = vue.withModifiers(($event) => showStashPanel.value = false, ["self"]))
            }, [
              vue.createElementVNode("div", _hoisted_63, [
                vue.createElementVNode("div", _hoisted_64, [
                  _cache[53] || (_cache[53] = vue.createElementVNode(
                    "span",
                    null,
                    "暂存管理",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode("button", {
                    class: "icon-btn",
                    onClick: _cache[27] || (_cache[27] = ($event) => showStashPanel.value = false)
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "close",
                      size: 14
                    })
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_65, [
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[28] || (_cache[28] = ($event) => stashMsg.value = $event),
                      placeholder: "暂存备注（可选）",
                      class: "form-input"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, stashMsg.value]
                  ]),
                  vue.createElementVNode("button", {
                    class: "git-btn btn-primary",
                    onClick: stashPush
                  }, "暂存")
                ]),
                vue.createElementVNode("div", _hoisted_66, [
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(stashes.value, (s) => {
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: s.index,
                        class: "stash-item"
                      }, [
                        vue.createElementVNode(
                          "span",
                          _hoisted_67,
                          vue.toDisplayString(s.index),
                          1
                          /* TEXT */
                        ),
                        vue.createElementVNode(
                          "span",
                          _hoisted_68,
                          vue.toDisplayString(s.msg),
                          1
                          /* TEXT */
                        ),
                        vue.createElementVNode("div", _hoisted_69, [
                          vue.createElementVNode("button", {
                            class: "icon-btn",
                            onClick: ($event) => stashPop(s.index),
                            title: "弹出"
                          }, [
                            vue.createVNode(SvgIcon, {
                              name: "undo",
                              size: 12
                            })
                          ], 8, _hoisted_70),
                          vue.createElementVNode("button", {
                            class: "icon-btn",
                            onClick: ($event) => stashDrop(s.index),
                            title: "删除"
                          }, [
                            vue.createVNode(SvgIcon, {
                              name: "trash",
                              size: 12
                            })
                          ], 8, _hoisted_71)
                        ])
                      ]);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  )),
                  stashes.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_72, "没有暂存的更改")) : vue.createCommentVNode("v-if", true)
                ])
              ])
            ])) : vue.createCommentVNode("v-if", true)
          ])),
          vue.createCommentVNode(" .gitignore 编辑器 "),
          (vue.openBlock(), vue.createBlock(vue.Teleport, { to: "body" }, [
            showIgnoreEditor.value ? (vue.openBlock(), vue.createElementBlock("div", {
              key: 0,
              class: "overlay",
              onClick: _cache[33] || (_cache[33] = vue.withModifiers(($event) => showIgnoreEditor.value = false, ["self"]))
            }, [
              vue.createElementVNode("div", _hoisted_73, [
                vue.createElementVNode("div", _hoisted_74, [
                  _cache[54] || (_cache[54] = vue.createElementVNode(
                    "span",
                    null,
                    ".gitignore",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode("button", {
                    class: "icon-btn",
                    onClick: _cache[30] || (_cache[30] = ($event) => showIgnoreEditor.value = false)
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "close",
                      size: 14
                    })
                  ])
                ]),
                vue.withDirectives(vue.createElementVNode(
                  "textarea",
                  {
                    "onUpdate:modelValue": _cache[31] || (_cache[31] = ($event) => ignoreContent.value = $event),
                    class: "ignore-textarea",
                    rows: "12",
                    spellcheck: "false"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, ignoreContent.value]
                ]),
                vue.createElementVNode("div", _hoisted_75, [
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[32] || (_cache[32] = ($event) => showIgnoreEditor.value = false)
                  }, "取消"),
                  vue.createElementVNode("button", {
                    class: "git-btn btn-primary",
                    onClick: saveIgnore
                  }, "保存")
                ])
              ])
            ])) : vue.createCommentVNode("v-if", true)
          ])),
          vue.createCommentVNode(" 提交详情 "),
          showCommitDetailModal.value ? (vue.openBlock(), vue.createBlock(Modal, {
            key: 6,
            onClose: _cache[36] || (_cache[36] = ($event) => showCommitDetailModal.value = false),
            maxWidth: "600px"
          }, {
            title: vue.withCtx(() => {
              var _a, _b, _c, _d;
              return [
                vue.createTextVNode(
                  vue.toDisplayString((_a = detailCommit.value) == null ? void 0 : _a.short) + " — " + vue.toDisplayString((_d = (_c = (_b = detailCommit.value) == null ? void 0 : _b.msg) == null ? void 0 : _c.split("\n")[0]) == null ? void 0 : _d.substring(0, 40)),
                  1
                  /* TEXT */
                )
              ];
            }),
            default: vue.withCtx(() => {
              var _a, _b, _c;
              return [
                vue.createElementVNode("div", _hoisted_76, [
                  vue.createElementVNode("div", _hoisted_77, [
                    vue.createElementVNode("div", null, [
                      _cache[55] || (_cache[55] = vue.createElementVNode(
                        "strong",
                        null,
                        "作者：",
                        -1
                        /* CACHED */
                      )),
                      vue.createTextVNode(
                        vue.toDisplayString((_a = detailCommit.value) == null ? void 0 : _a.author),
                        1
                        /* TEXT */
                      )
                    ]),
                    vue.createElementVNode("div", null, [
                      _cache[56] || (_cache[56] = vue.createElementVNode(
                        "strong",
                        null,
                        "日期：",
                        -1
                        /* CACHED */
                      )),
                      vue.createTextVNode(
                        vue.toDisplayString(formatDate((_b = detailCommit.value) == null ? void 0 : _b.date, true)),
                        1
                        /* TEXT */
                      )
                    ]),
                    vue.createElementVNode("div", null, [
                      _cache[57] || (_cache[57] = vue.createElementVNode(
                        "strong",
                        null,
                        "哈希：",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode(
                        "code",
                        null,
                        vue.toDisplayString((_c = detailCommit.value) == null ? void 0 : _c.hash),
                        1
                        /* TEXT */
                      )
                    ])
                  ]),
                  vue.createElementVNode("div", _hoisted_78, [
                    vue.createElementVNode(
                      "pre",
                      null,
                      vue.toDisplayString(commitDiff.value || "加载中..."),
                      1
                      /* TEXT */
                    )
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_79, [
                  vue.createElementVNode("button", {
                    class: "git-btn",
                    onClick: _cache[34] || (_cache[34] = ($event) => showCommitDetailModal.value = false)
                  }, "关闭"),
                  vue.createElementVNode("button", {
                    class: "git-btn btn-primary",
                    onClick: _cache[35] || (_cache[35] = ($event) => {
                      var _a2;
                      return copyHash((_a2 = detailCommit.value) == null ? void 0 : _a2.hash);
                    })
                  }, "复制哈希")
                ])
              ];
            }),
            _: 1
            /* STABLE */
          })) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 文件差异对话框 "),
          showDiffDialog.value ? (vue.openBlock(), vue.createBlock(Modal, {
            key: 7,
            onClose: _cache[38] || (_cache[38] = ($event) => showDiffDialog.value = false),
            maxWidth: "700px"
          }, {
            title: vue.withCtx(() => [
              vue.createElementVNode(
                "span",
                _hoisted_80,
                vue.toDisplayString(diffFilePath.value),
                1
                /* TEXT */
              )
            ]),
            default: vue.withCtx(() => [
              vue.createElementVNode("div", _hoisted_81, [
                vue.createElementVNode(
                  "pre",
                  _hoisted_82,
                  vue.toDisplayString(diffContent.value || "（无差异）"),
                  1
                  /* TEXT */
                )
              ]),
              vue.createElementVNode("div", _hoisted_83, [
                vue.createElementVNode("button", {
                  class: "git-btn",
                  onClick: _cache[37] || (_cache[37] = ($event) => showDiffDialog.value = false)
                }, "关闭")
              ])
            ]),
            _: 1
            /* STABLE */
          })) : vue.createCommentVNode("v-if", true)
        ]);
      };
    }
  };
  const GitPanel2 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-ed956158"]]);
  function mount(el) {
    const app = vue.createApp(GitPanel2);
    app.mount(el);
    return () => {
      app.unmount();
    };
  }
  exports.mount = mount;
  Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
  return exports;
})({}, window.__PAIRCODE_CORE.Vue, window.__PAIRCODE_CORE.uiState, window.__PAIRCODE_CORE.api);
