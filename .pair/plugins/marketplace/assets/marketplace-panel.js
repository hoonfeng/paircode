var MarketplacePanel = (function(exports, vue, api, uiState_js) {
  "use strict";
  const _export_sfc = (sfc, props) => {
    const target = sfc.__vccOpts || sfc;
    for (const [key, val] of props) {
      target[key] = val;
    }
    return target;
  };
  const _hoisted_1$1 = ["width", "height"];
  const _hoisted_2$1 = {
    key: 0,
    d: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
  };
  const _sfc_main$1 = {
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
          __props.name === "folder" ? (vue.openBlock(), vue.createElementBlock("path", _hoisted_2$1)) : __props.name === "folder-open" ? (vue.openBlock(), vue.createElementBlock(
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
        ], 8, _hoisted_1$1);
      };
    }
  };
  const SvgIcon = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-faf69761"]]);
  const _hoisted_1 = { class: "market-panel" };
  const _hoisted_2 = { class: "market-header" };
  const _hoisted_3 = { class: "market-tabs" };
  const _hoisted_4 = ["onClick"];
  const _hoisted_5 = { class: "market-body" };
  const _hoisted_6 = {
    key: 0,
    class: "market-search"
  };
  const _hoisted_7 = { class: "search-icon" };
  const _hoisted_8 = ["disabled", "title"];
  const _hoisted_9 = {
    key: 1,
    class: "installed-toolbar"
  };
  const _hoisted_10 = {
    key: 2,
    class: "mcp-form"
  };
  const _hoisted_11 = { class: "mcp-form-row" };
  const _hoisted_12 = { class: "mcp-form-row" };
  const _hoisted_13 = { class: "mcp-form-row" };
  const _hoisted_14 = { class: "mcp-form-row" };
  const _hoisted_15 = { class: "mcp-form-actions" };
  const _hoisted_16 = ["disabled"];
  const _hoisted_17 = {
    key: 0,
    class: "mcp-form-error"
  };
  const _hoisted_18 = {
    key: 3,
    class: "mcp-form"
  };
  const _hoisted_19 = { class: "mcp-form-row" };
  const _hoisted_20 = { class: "mcp-form-row" };
  const _hoisted_21 = { class: "mcp-form-row" };
  const _hoisted_22 = { class: "mcp-form-row" };
  const _hoisted_23 = { class: "mcp-form-actions" };
  const _hoisted_24 = ["disabled"];
  const _hoisted_25 = {
    key: 4,
    class: "skill-viewer"
  };
  const _hoisted_26 = { class: "skill-viewer-header" };
  const _hoisted_27 = { class: "skill-viewer-content" };
  const _hoisted_28 = {
    key: 5,
    class: "market-loading"
  };
  const _hoisted_29 = {
    key: 0,
    class: "installed-group"
  };
  const _hoisted_30 = ["title"];
  const _hoisted_31 = { class: "ig-count" };
  const _hoisted_32 = { class: "ii-icon icon-plugin" };
  const _hoisted_33 = ["title"];
  const _hoisted_34 = { class: "ii-body" };
  const _hoisted_35 = { class: "ii-name" };
  const _hoisted_36 = { class: "ii-desc" };
  const _hoisted_37 = { class: "ii-badge" };
  const _hoisted_38 = {
    key: 0,
    class: "badge-system"
  };
  const _hoisted_39 = { class: "ii-actions" };
  const _hoisted_40 = ["onClick"];
  const _hoisted_41 = ["onClick"];
  const _hoisted_42 = ["onClick"];
  const _hoisted_43 = {
    key: 1,
    class: "installed-group"
  };
  const _hoisted_44 = ["title"];
  const _hoisted_45 = { class: "ig-count" };
  const _hoisted_46 = { class: "ii-icon icon-mcp" };
  const _hoisted_47 = ["title"];
  const _hoisted_48 = { class: "ii-body" };
  const _hoisted_49 = { class: "ii-name" };
  const _hoisted_50 = { class: "ii-desc" };
  const _hoisted_51 = { class: "ii-badge" };
  const _hoisted_52 = { class: "ii-actions" };
  const _hoisted_53 = ["onClick", "title"];
  const _hoisted_54 = ["onClick"];
  const _hoisted_55 = ["onClick"];
  const _hoisted_56 = {
    key: 2,
    class: "installed-group"
  };
  const _hoisted_57 = ["title"];
  const _hoisted_58 = { class: "ig-count" };
  const _hoisted_59 = { class: "ii-icon icon-skill" };
  const _hoisted_60 = { class: "ii-body" };
  const _hoisted_61 = { class: "ii-name" };
  const _hoisted_62 = { class: "ii-desc" };
  const _hoisted_63 = { class: "ii-badge" };
  const _hoisted_64 = { class: "ii-actions" };
  const _hoisted_65 = ["value", "onChange", "title"];
  const _hoisted_66 = ["onClick"];
  const _hoisted_67 = ["onClick"];
  const _hoisted_68 = {
    key: 3,
    class: "market-empty"
  };
  const _hoisted_69 = { class: "me-icon" };
  const _hoisted_70 = { class: "mi-body" };
  const _hoisted_71 = { class: "mi-name" };
  const _hoisted_72 = { class: "mi-desc" };
  const _hoisted_73 = { class: "mi-meta" };
  const _hoisted_74 = {
    key: 0,
    class: "mi-tags"
  };
  const _hoisted_75 = {
    key: 1,
    class: "mi-installed"
  };
  const _hoisted_76 = {
    key: 0,
    class: "mi-install-area"
  };
  const _hoisted_77 = ["onClick", "disabled"];
  const _hoisted_78 = ["onUpdate:modelValue"];
  const _hoisted_79 = ["onClick", "disabled"];
  const _hoisted_80 = {
    key: 1,
    class: "mi-install-area"
  };
  const _hoisted_81 = ["onClick"];
  const _hoisted_82 = {
    key: 0,
    class: "market-empty"
  };
  const _hoisted_83 = { class: "me-icon" };
  const _hoisted_84 = { key: 0 };
  const _hoisted_85 = { key: 1 };
  const _hoisted_86 = { class: "market-footer" };
  const _hoisted_87 = { class: "market-count" };
  const _hoisted_88 = {
    key: 0,
    class: "market-error"
  };
  const _sfc_main = {
    __name: "MarketplacePanel",
    setup(__props) {
      function closePanel() {
        uiState_js.state.activeActivity = "explorer";
      }
      const tab = vue.ref("all");
      const query = vue.ref("");
      const items = vue.ref([]);
      const installing = vue.ref("");
      const loading = vue.ref(false);
      const refreshing = vue.ref(false);
      const error = vue.ref("");
      const refreshTip = vue.ref("");
      const listRef = vue.ref(null);
      const collapsedGroups = vue.ref(/* @__PURE__ */ new Set());
      function toggleGroup(g) {
        const s = new Set(collapsedGroups.value);
        s.has(g) ? s.delete(g) : s.add(g);
        collapsedGroups.value = s;
      }
      const sources = vue.ref([]);
      const marketTabs = vue.computed(() => {
        const labelMap = { skill: "技能", mcp: "MCP", plugin: "插件/工具集" };
        return (sources.value || []).map((s) => ({ kind: s.kind, label: labelMap[s.kind] || s.name || s.kind }));
      });
      async function loadSources() {
        try {
          const srcs = await api.apiGet("/marketplace/sources");
          sources.value = srcs || [];
        } catch (e) {
        }
      }
      const installedMCPs = vue.ref([]);
      const installedSkills = vue.ref([]);
      const installedPlugins = vue.ref([]);
      const showAddMCP = vue.ref(false);
      const savingMCP = vue.ref(false);
      const mcpError = vue.ref("");
      const editingMCP = vue.ref(false);
      const viewingSkill = vue.ref(null);
      const mcpForm = vue.ref({ name: "", command: "", argsText: "", level: "user" });
      const editMCPForm = vue.ref({ name: "", command: "", argsText: "", level: "user", origName: "" });
      function resetMCPForm() {
        mcpForm.value = { name: "", command: "", argsText: "", level: "user" };
        mcpError.value = "";
      }
      async function loadInstalled() {
        loading.value = true;
        error.value = "";
        try {
          const [mcpList, skillList, pluginList] = await Promise.all([
            api.getMcpList("all"),
            api.getSkillsList(),
            api.listPlugins()
          ]);
          installedMCPs.value = mcpList || [];
          installedSkills.value = skillList || [];
          installedPlugins.value = pluginList || [];
        } catch (err) {
          error.value = "加载失败: " + err.message;
        } finally {
          loading.value = false;
        }
      }
      async function saveMCP() {
        const f = mcpForm.value;
        if (!f.name) return;
        savingMCP.value = true;
        mcpError.value = "";
        try {
          const args = f.argsText ? f.argsText.split(" ").filter(Boolean) : [];
          await api.saveMcpItem({ action: "save", name: f.name, command: f.command || "npx", args, level: f.level });
          showAddMCP.value = false;
          resetMCPForm();
          await loadInstalled();
        } catch (err) {
          mcpError.value = err.message;
        } finally {
          savingMCP.value = false;
        }
      }
      function startEditMCP(item) {
        editMCPForm.value = {
          origName: item.name,
          name: item.name,
          command: item.command,
          argsText: (item.args || []).join(" "),
          level: item.level
        };
        editingMCP.value = true;
      }
      async function updateMCP() {
        const f = editMCPForm.value;
        if (!f.name) return;
        try {
          if (f.origName !== f.name) {
            await api.saveMcpItem({ action: "delete", name: f.origName, level: f.level });
          }
          const args = f.argsText ? f.argsText.split(" ").filter(Boolean) : [];
          await api.saveMcpItem({ action: "save", name: f.name, command: f.command || "npx", args, level: f.level });
          editingMCP.value = false;
          await loadInstalled();
        } catch (err) {
          error.value = "保存失败: " + err.message;
        }
      }
      async function delMCP(item) {
        if (!confirm(`确认删除 MCP 服务器「${item.name}」？`)) return;
        try {
          await api.saveMcpItem({ action: "delete", name: item.name, level: item.level });
          await loadInstalled();
        } catch (err) {
          error.value = "删除失败: " + err.message;
        }
      }
      async function viewSkill(item) {
        try {
          const data = await api.readSkill(item.name, item.level);
          viewingSkill.value = data || { name: item.name, content: "（内容读取失败）" };
        } catch (err) {
          viewingSkill.value = { name: item.name, content: "读取失败: " + err.message };
        }
      }
      async function delSkill(item) {
        if (!confirm(`确认删除技能「${item.name}」？`)) return;
        try {
          await api.deleteSkill(item.name);
          await loadInstalled();
        } catch (err) {
          error.value = "删除失败: " + err.message;
        }
      }
      function statusTitle(item) {
        if (item.status === "off") return "已关闭：技能完全禁用，不加载";
        if (item.status === "max") return "始终激活：技能常驻 system prompt";
        return "按需：根据关键词/文件匹配自动激活";
      }
      const CORE_PLUGINS = ["agentloop", "core-api", "marketplace", "web-api", "fs-api", "git-api"];
      function isCorePlugin(name) {
        return CORE_PLUGINS.includes(name);
      }
      async function togglePlugin(item) {
        var _a, _b;
        const action = item.state === "running" ? "stop" : "start";
        try {
          await api.pluginAction(item.name, action);
          item.state = action === "stop" ? "stopped" : "running";
          (_a = window.$toast) == null ? void 0 : _a.call(window, `插件「${item.name}」已${action === "stop" ? "停止" : "启动"}`, "success");
        } catch (err) {
          error.value = "操作失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "操作失败: " + err.message, "error");
        }
      }
      async function uninstallPlugin(item) {
        var _a, _b;
        if (!confirm(`确认卸载插件「${item.name}」？
将删除 .pair/plugins/${item.name} 目录，如需恢复可从市场重新安装。`)) return;
        try {
          await api.pluginAction(item.name, "undefine");
          installedPlugins.value = installedPlugins.value.filter((p) => p.name !== item.name);
          (_a = window.$toast) == null ? void 0 : _a.call(window, `插件「${item.name}」已卸载`, "success");
        } catch (err) {
          error.value = "卸载失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "卸载失败: " + err.message, "error");
        }
      }
      async function setSkillStatus(item, status) {
        var _a, _b;
        try {
          await api.saveSkillStatus(item.name, item.level || "project", status);
          item.status = status;
          (_a = window.$toast) == null ? void 0 : _a.call(window, `技能「${item.name}」状态已设为 ${status === "off" ? "关闭" : status === "max" ? "始终激活" : "按需"}`, "success");
        } catch (err) {
          error.value = "设置失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "设置失败: " + err.message, "error");
        }
      }
      let debounceTimer = null;
      function debounceSearch() {
        clearTimeout(debounceTimer);
        loading.value = true;
        debounceTimer = setTimeout(doSearch, 250);
      }
      async function doSearch() {
        loading.value = true;
        error.value = "";
        try {
          const kind = tab.value === "all" ? "" : tab.value;
          const results = await api.apiGet("/marketplace/search", {
            q: query.value,
            kind
          });
          items.value = results || [];
        } catch (err) {
          error.value = "搜索失败: " + err.message;
          items.value = [];
        } finally {
          loading.value = false;
        }
      }
      async function refreshRemote() {
        var _a, _b;
        refreshing.value = true;
        error.value = "";
        try {
          const result = await api.apiPost("/marketplace/refresh", {});
          refreshTip.value = result.status || "已刷新";
          (_a = window.$toast) == null ? void 0 : _a.call(window, result.message || "远程市场已刷新", "success");
          await doSearch();
        } catch (err) {
          error.value = "刷新失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "刷新远程市场失败: " + err.message, "error");
        } finally {
          refreshing.value = false;
          setTimeout(() => {
            refreshTip.value = "";
          }, 5e3);
        }
      }
      async function installItem(item, scope) {
        var _a, _b;
        installing.value = item.id;
        error.value = "";
        try {
          const body = {
            id: item.id,
            kind: item.kind || "",
            command: item.command || "",
            args: item.args || [],
            source: item.source || "",
            description: item.description || ""
          };
          if (item.kind === "mcp") {
            body.scope = scope || "user";
            body.name = item.name || String(item.id).replace(/^npm-/, "");
          } else if (item.kind === "plugin") {
            body.scope = "project";
          }
          const result = await api.apiPost("/marketplace/install", body);
          item.installed = true;
          (_a = window.$toast) == null ? void 0 : _a.call(window, result.message || "安装成功", "success");
        } catch (err) {
          error.value = "安装失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "安装失败: " + err.message, "error");
        } finally {
          installing.value = "";
        }
      }
      async function toggleMCP(item) {
        var _a, _b;
        try {
          const result = await api.saveMcpItem({ action: "toggle", name: item.name, level: item.level });
          item.enabled = result.enabled;
          (_a = window.$toast) == null ? void 0 : _a.call(window, `MCP 服务器「${item.name}」${result.enabled ? "已启用" : "已禁用"}`, "success");
        } catch (err) {
          error.value = "切换失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "切换失败: " + err.message, "error");
        }
      }
      async function uninstallItem(item) {
        var _a, _b;
        error.value = "";
        try {
          const result = await api.apiPost("/marketplace/uninstall", {
            id: item.id,
            kind: item.kind,
            source: item.source || ""
          });
          item.installed = false;
          (_a = window.$toast) == null ? void 0 : _a.call(window, result.message || "已卸载: " + item.name, "success");
          await doSearch();
        } catch (err) {
          error.value = "卸载失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "卸载失败: " + err.message, "error");
        }
      }
      vue.onMounted(() => {
        loadSources();
        if (!query.value) query.value = "paircode";
        doSearch();
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1, [
          vue.createElementVNode("div", _hoisted_2, [
            vue.createElementVNode("h2", null, [
              vue.createVNode(SvgIcon, {
                name: "package",
                size: 20
              }),
              _cache[20] || (_cache[20] = vue.createTextVNode(
                " 市场",
                -1
                /* CACHED */
              ))
            ]),
            vue.createElementVNode("div", _hoisted_3, [
              vue.createElementVNode(
                "button",
                {
                  class: vue.normalizeClass({ active: tab.value === "all" }),
                  onClick: _cache[0] || (_cache[0] = ($event) => {
                    tab.value = "all";
                    doSearch();
                  })
                },
                "全部",
                2
                /* CLASS */
              ),
              (vue.openBlock(true), vue.createElementBlock(
                vue.Fragment,
                null,
                vue.renderList(marketTabs.value, (s) => {
                  return vue.openBlock(), vue.createElementBlock("button", {
                    key: s.kind,
                    class: vue.normalizeClass({ active: tab.value === s.kind }),
                    onClick: ($event) => {
                      tab.value = s.kind;
                      doSearch();
                    }
                  }, vue.toDisplayString(s.label), 11, _hoisted_4);
                }),
                128
                /* KEYED_FRAGMENT */
              )),
              vue.createElementVNode(
                "button",
                {
                  class: vue.normalizeClass({ active: tab.value === "installed" }),
                  onClick: _cache[1] || (_cache[1] = ($event) => {
                    tab.value = "installed";
                    loadInstalled();
                  })
                },
                "已安装",
                2
                /* CLASS */
              )
            ]),
            vue.createElementVNode("button", {
              class: "modal-close",
              onClick: closePanel
            }, "×")
          ]),
          vue.createElementVNode("div", _hoisted_5, [
            vue.createCommentVNode(" 搜索栏（非「已安装」tab 显示） "),
            tab.value !== "installed" ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_6, [
              vue.createElementVNode("div", _hoisted_7, [
                vue.createVNode(SvgIcon, {
                  name: "search",
                  size: 14
                })
              ]),
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => query.value = $event),
                  placeholder: "搜索 MCP / 技能 / npm 插件…",
                  onInput: debounceSearch,
                  class: "search-input"
                },
                null,
                544
                /* NEED_HYDRATION, NEED_PATCH */
              ), [
                [vue.vModelText, query.value]
              ]),
              query.value ? (vue.openBlock(), vue.createElementBlock("button", {
                key: 0,
                class: "search-clear",
                onClick: _cache[3] || (_cache[3] = ($event) => {
                  query.value = "";
                  doSearch();
                })
              }, "×")) : vue.createCommentVNode("v-if", true),
              vue.createElementVNode("button", {
                class: "market-refresh-btn",
                onClick: refreshRemote,
                disabled: refreshing.value,
                title: refreshTip.value
              }, [
                vue.createVNode(SvgIcon, {
                  name: refreshing.value ? "cycle" : "refresh",
                  size: 14
                }, null, 8, ["name"]),
                vue.createTextVNode(
                  " " + vue.toDisplayString(refreshing.value ? "刷新中…" : "刷新"),
                  1
                  /* TEXT */
                )
              ], 8, _hoisted_8)
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" 「已安装」tab 的操作栏 "),
            tab.value === "installed" ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_9, [
              vue.createElementVNode("button", {
                class: "btn-add-mcp",
                onClick: _cache[4] || (_cache[4] = ($event) => showAddMCP.value = true)
              }, [
                vue.createVNode(SvgIcon, {
                  name: "plus",
                  size: 14
                }),
                _cache[21] || (_cache[21] = vue.createTextVNode(
                  " 添加 MCP 服务器",
                  -1
                  /* CACHED */
                ))
              ]),
              vue.createElementVNode("button", {
                class: "btn-refresh",
                onClick: loadInstalled
              }, [
                vue.createVNode(SvgIcon, {
                  name: "refresh",
                  size: 14
                }),
                _cache[22] || (_cache[22] = vue.createTextVNode(
                  " 刷新",
                  -1
                  /* CACHED */
                ))
              ])
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" 添加 MCP 表单 "),
            showAddMCP.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_10, [
              vue.createElementVNode("div", _hoisted_11, [
                _cache[23] || (_cache[23] = vue.createElementVNode(
                  "label",
                  null,
                  "名称",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[5] || (_cache[5] = ($event) => mcpForm.value.name = $event),
                    placeholder: "如 my-server"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, mcpForm.value.name]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_12, [
                _cache[24] || (_cache[24] = vue.createElementVNode(
                  "label",
                  null,
                  "命令",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => mcpForm.value.command = $event),
                    placeholder: "如 npx / uvx"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, mcpForm.value.command]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_13, [
                _cache[25] || (_cache[25] = vue.createElementVNode(
                  "label",
                  null,
                  "参数",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[7] || (_cache[7] = ($event) => mcpForm.value.argsText = $event),
                    placeholder: "如 -y @modelcontextprotocol/server-filesystem"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, mcpForm.value.argsText]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_14, [
                _cache[27] || (_cache[27] = vue.createElementVNode(
                  "label",
                  null,
                  "层级",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "select",
                  {
                    "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => mcpForm.value.level = $event)
                  },
                  [..._cache[26] || (_cache[26] = [
                    vue.createElementVNode(
                      "option",
                      { value: "user" },
                      "用户级（全局）",
                      -1
                      /* CACHED */
                    ),
                    vue.createElementVNode(
                      "option",
                      { value: "project" },
                      "工作区级",
                      -1
                      /* CACHED */
                    )
                  ])],
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelSelect, mcpForm.value.level]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_15, [
                vue.createElementVNode("button", {
                  class: "btn-primary",
                  onClick: saveMCP,
                  disabled: !mcpForm.value.name || savingMCP.value
                }, vue.toDisplayString(savingMCP.value ? "保存中…" : "保存"), 9, _hoisted_16),
                vue.createElementVNode("button", {
                  class: "btn-secondary",
                  onClick: _cache[9] || (_cache[9] = ($event) => {
                    showAddMCP.value = false;
                    resetMCPForm();
                  })
                }, "取消")
              ]),
              mcpError.value ? (vue.openBlock(), vue.createElementBlock(
                "div",
                _hoisted_17,
                vue.toDisplayString(mcpError.value),
                1
                /* TEXT */
              )) : vue.createCommentVNode("v-if", true)
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" 编辑 MCP 表单 "),
            editingMCP.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_18, [
              vue.createElementVNode("div", _hoisted_19, [
                _cache[28] || (_cache[28] = vue.createElementVNode(
                  "label",
                  null,
                  "名称",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[10] || (_cache[10] = ($event) => editMCPForm.value.name = $event)
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, editMCPForm.value.name]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_20, [
                _cache[29] || (_cache[29] = vue.createElementVNode(
                  "label",
                  null,
                  "命令",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[11] || (_cache[11] = ($event) => editMCPForm.value.command = $event)
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, editMCPForm.value.command]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_21, [
                _cache[30] || (_cache[30] = vue.createElementVNode(
                  "label",
                  null,
                  "参数",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[12] || (_cache[12] = ($event) => editMCPForm.value.argsText = $event)
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, editMCPForm.value.argsText]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_22, [
                _cache[32] || (_cache[32] = vue.createElementVNode(
                  "label",
                  null,
                  "层级",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "select",
                  {
                    "onUpdate:modelValue": _cache[13] || (_cache[13] = ($event) => editMCPForm.value.level = $event)
                  },
                  [..._cache[31] || (_cache[31] = [
                    vue.createElementVNode(
                      "option",
                      { value: "user" },
                      "用户级（全局）",
                      -1
                      /* CACHED */
                    ),
                    vue.createElementVNode(
                      "option",
                      { value: "project" },
                      "工作区级",
                      -1
                      /* CACHED */
                    )
                  ])],
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelSelect, editMCPForm.value.level]
                ])
              ]),
              vue.createElementVNode("div", _hoisted_23, [
                vue.createElementVNode("button", {
                  class: "btn-primary",
                  onClick: updateMCP,
                  disabled: !editMCPForm.value.name
                }, "保存", 8, _hoisted_24),
                vue.createElementVNode("button", {
                  class: "btn-secondary",
                  onClick: _cache[14] || (_cache[14] = ($event) => editingMCP.value = false)
                }, "取消")
              ])
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" 技能内容查看 "),
            viewingSkill.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_25, [
              vue.createElementVNode("div", _hoisted_26, [
                vue.createElementVNode(
                  "strong",
                  null,
                  vue.toDisplayString(viewingSkill.value.name),
                  1
                  /* TEXT */
                ),
                vue.createElementVNode("button", {
                  class: "modal-close",
                  onClick: _cache[15] || (_cache[15] = ($event) => viewingSkill.value = null)
                }, "×")
              ]),
              vue.createElementVNode(
                "pre",
                _hoisted_27,
                vue.toDisplayString(viewingSkill.value.content),
                1
                /* TEXT */
              )
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" 加载状态 "),
            loading.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_28, [..._cache[33] || (_cache[33] = [
              vue.createElementVNode(
                "span",
                { class: "dot-pulse" },
                null,
                -1
                /* CACHED */
              ),
              vue.createElementVNode(
                "span",
                null,
                "搜索中...",
                -1
                /* CACHED */
              )
            ])])) : tab.value === "installed" ? (vue.openBlock(), vue.createElementBlock(
              vue.Fragment,
              { key: 6 },
              [
                vue.createCommentVNode(" 已安装列表 "),
                vue.createElementVNode(
                  "div",
                  {
                    class: "market-list",
                    ref_key: "listRef",
                    ref: listRef
                  },
                  [
                    vue.createCommentVNode(" 插件分组（★ 2026-08-20 新增：磁盘插件清单） "),
                    installedPlugins.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_29, [
                      vue.createElementVNode("div", {
                        class: "installed-group-title",
                        onClick: _cache[16] || (_cache[16] = ($event) => toggleGroup("plugin")),
                        title: collapsedGroups.value.has("plugin") ? "展开插件列表" : "收起插件列表"
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: collapsedGroups.value.has("plugin") ? "chevron-right" : "chevron-down",
                          size: 12,
                          class: "ig-arrow"
                        }, null, 8, ["name"]),
                        _cache[34] || (_cache[34] = vue.createElementVNode(
                          "span",
                          null,
                          "插件",
                          -1
                          /* CACHED */
                        )),
                        vue.createElementVNode(
                          "span",
                          _hoisted_31,
                          vue.toDisplayString(installedPlugins.value.length),
                          1
                          /* TEXT */
                        )
                      ], 8, _hoisted_30),
                      vue.withDirectives(vue.createElementVNode(
                        "div",
                        null,
                        [
                          (vue.openBlock(true), vue.createElementBlock(
                            vue.Fragment,
                            null,
                            vue.renderList(installedPlugins.value, (item) => {
                              return vue.openBlock(), vue.createElementBlock("div", {
                                key: "plugin-" + item.name,
                                class: "installed-item"
                              }, [
                                vue.createElementVNode("div", _hoisted_32, [
                                  vue.createVNode(SvgIcon, {
                                    name: "puzzle",
                                    size: 18
                                  })
                                ]),
                                vue.createElementVNode("div", {
                                  class: vue.normalizeClass(["ii-status-dot", item.state === "running" ? "dot-connected" : "dot-disabled"]),
                                  title: item.state === "running" ? "运行中" : "已停止"
                                }, null, 10, _hoisted_33),
                                vue.createElementVNode("div", _hoisted_34, [
                                  vue.createElementVNode(
                                    "div",
                                    _hoisted_35,
                                    vue.toDisplayString(item.name),
                                    1
                                    /* TEXT */
                                  ),
                                  vue.createElementVNode(
                                    "div",
                                    _hoisted_36,
                                    vue.toDisplayString(item.purpose || "（无描述）"),
                                    1
                                    /* TEXT */
                                  ),
                                  vue.createElementVNode("span", _hoisted_37, [
                                    isCorePlugin(item.name) ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_38, "系统")) : vue.createCommentVNode("v-if", true),
                                    _cache[35] || (_cache[35] = vue.createTextVNode(
                                      " 插件 · ",
                                      -1
                                      /* CACHED */
                                    )),
                                    vue.createElementVNode(
                                      "span",
                                      {
                                        class: vue.normalizeClass("status-" + (item.state === "running" ? "on" : "off"))
                                      },
                                      vue.toDisplayString(item.state === "running" ? "运行中" : "已停止"),
                                      3
                                      /* TEXT, CLASS */
                                    ),
                                    vue.createTextVNode(
                                      " · " + vue.toDisplayString(item.scope === "global" ? "全局" : "工作区"),
                                      1
                                      /* TEXT */
                                    )
                                  ])
                                ]),
                                vue.createElementVNode("div", _hoisted_39, [
                                  item.state === "running" ? (vue.openBlock(), vue.createElementBlock("button", {
                                    key: 0,
                                    class: "ii-btn ii-toggle",
                                    onClick: ($event) => togglePlugin(item),
                                    title: "停止：插件及其工具/UI 不再生效"
                                  }, "停止", 8, _hoisted_40)) : (vue.openBlock(), vue.createElementBlock("button", {
                                    key: 1,
                                    class: "ii-btn ii-toggle is-enabled",
                                    onClick: ($event) => togglePlugin(item),
                                    title: "启动插件"
                                  }, "启动", 8, _hoisted_41)),
                                  !isCorePlugin(item.name) ? (vue.openBlock(), vue.createElementBlock("button", {
                                    key: 2,
                                    class: "ii-btn ii-del",
                                    onClick: ($event) => uninstallPlugin(item),
                                    title: "卸载：删除插件包目录，可重新从市场安装"
                                  }, "卸载", 8, _hoisted_42)) : vue.createCommentVNode("v-if", true)
                                ])
                              ]);
                            }),
                            128
                            /* KEYED_FRAGMENT */
                          ))
                        ],
                        512
                        /* NEED_PATCH */
                      ), [
                        [vue.vShow, !collapsedGroups.value.has("plugin")]
                      ])
                    ])) : vue.createCommentVNode("v-if", true),
                    vue.createCommentVNode(" MCP 分组 "),
                    installedMCPs.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_43, [
                      vue.createElementVNode("div", {
                        class: "installed-group-title",
                        onClick: _cache[17] || (_cache[17] = ($event) => toggleGroup("mcp")),
                        title: collapsedGroups.value.has("mcp") ? "展开 MCP 列表" : "收起 MCP 列表"
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: collapsedGroups.value.has("mcp") ? "chevron-right" : "chevron-down",
                          size: 12,
                          class: "ig-arrow"
                        }, null, 8, ["name"]),
                        _cache[36] || (_cache[36] = vue.createElementVNode(
                          "span",
                          null,
                          "MCP 服务器",
                          -1
                          /* CACHED */
                        )),
                        vue.createElementVNode(
                          "span",
                          _hoisted_45,
                          vue.toDisplayString(installedMCPs.value.length),
                          1
                          /* TEXT */
                        )
                      ], 8, _hoisted_44),
                      vue.withDirectives(vue.createElementVNode(
                        "div",
                        null,
                        [
                          (vue.openBlock(true), vue.createElementBlock(
                            vue.Fragment,
                            null,
                            vue.renderList(installedMCPs.value, (item) => {
                              return vue.openBlock(), vue.createElementBlock("div", {
                                key: "mcp-" + item.name + "-" + item.level,
                                class: "installed-item"
                              }, [
                                vue.createElementVNode("div", _hoisted_46, [
                                  vue.createVNode(SvgIcon, {
                                    name: "package",
                                    size: 18
                                  })
                                ]),
                                vue.createElementVNode("div", {
                                  class: vue.normalizeClass(["ii-status-dot", item.enabled === false ? "dot-disabled" : item._connected ? "dot-connected" : "dot-idle"]),
                                  title: item.enabled === false ? "已禁用" : item._connected ? "已连接" : "未连接"
                                }, null, 10, _hoisted_47),
                                vue.createElementVNode("div", _hoisted_48, [
                                  vue.createElementVNode(
                                    "div",
                                    _hoisted_49,
                                    vue.toDisplayString(item.name),
                                    1
                                    /* TEXT */
                                  ),
                                  vue.createElementVNode(
                                    "div",
                                    _hoisted_50,
                                    vue.toDisplayString(item.command) + " " + vue.toDisplayString((item.args || []).join(" ")),
                                    1
                                    /* TEXT */
                                  ),
                                  vue.createElementVNode(
                                    "span",
                                    _hoisted_51,
                                    "MCP · " + vue.toDisplayString(item.level === "project" ? "工作区级" : "用户级"),
                                    1
                                    /* TEXT */
                                  )
                                ]),
                                vue.createElementVNode("div", _hoisted_52, [
                                  vue.createElementVNode("button", {
                                    class: vue.normalizeClass(["ii-btn ii-toggle", { "is-enabled": item.enabled !== false }]),
                                    onClick: ($event) => toggleMCP(item),
                                    title: item.enabled === false ? "点击启用" : "点击禁用"
                                  }, vue.toDisplayString(item.enabled === false ? "禁用" : "启用"), 11, _hoisted_53),
                                  vue.createElementVNode("button", {
                                    class: "ii-btn ii-edit",
                                    onClick: ($event) => startEditMCP(item),
                                    title: "编辑"
                                  }, "编辑", 8, _hoisted_54),
                                  vue.createElementVNode("button", {
                                    class: "ii-btn ii-del",
                                    onClick: ($event) => delMCP(item),
                                    title: "删除"
                                  }, "删除", 8, _hoisted_55)
                                ])
                              ]);
                            }),
                            128
                            /* KEYED_FRAGMENT */
                          ))
                        ],
                        512
                        /* NEED_PATCH */
                      ), [
                        [vue.vShow, !collapsedGroups.value.has("mcp")]
                      ])
                    ])) : vue.createCommentVNode("v-if", true),
                    vue.createCommentVNode(" 技能分组 "),
                    installedSkills.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_56, [
                      vue.createElementVNode("div", {
                        class: "installed-group-title",
                        onClick: _cache[18] || (_cache[18] = ($event) => toggleGroup("skill")),
                        title: collapsedGroups.value.has("skill") ? "展开技能列表" : "收起技能列表"
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: collapsedGroups.value.has("skill") ? "chevron-right" : "chevron-down",
                          size: 12,
                          class: "ig-arrow"
                        }, null, 8, ["name"]),
                        _cache[37] || (_cache[37] = vue.createElementVNode(
                          "span",
                          null,
                          "技能",
                          -1
                          /* CACHED */
                        )),
                        vue.createElementVNode(
                          "span",
                          _hoisted_58,
                          vue.toDisplayString(installedSkills.value.length),
                          1
                          /* TEXT */
                        )
                      ], 8, _hoisted_57),
                      vue.withDirectives(vue.createElementVNode(
                        "div",
                        null,
                        [
                          (vue.openBlock(true), vue.createElementBlock(
                            vue.Fragment,
                            null,
                            vue.renderList(installedSkills.value, (item) => {
                              return vue.openBlock(), vue.createElementBlock("div", {
                                key: "skill-" + item.name + "-" + item.level,
                                class: "installed-item"
                              }, [
                                vue.createElementVNode("div", _hoisted_59, [
                                  vue.createVNode(SvgIcon, {
                                    name: "code",
                                    size: 18
                                  })
                                ]),
                                vue.createElementVNode("div", _hoisted_60, [
                                  vue.createElementVNode(
                                    "div",
                                    _hoisted_61,
                                    vue.toDisplayString(item.name),
                                    1
                                    /* TEXT */
                                  ),
                                  vue.createElementVNode(
                                    "div",
                                    _hoisted_62,
                                    vue.toDisplayString(item.description || "无描述"),
                                    1
                                    /* TEXT */
                                  ),
                                  vue.createElementVNode("span", _hoisted_63, [
                                    _cache[38] || (_cache[38] = vue.createTextVNode(
                                      " 技能 · ",
                                      -1
                                      /* CACHED */
                                    )),
                                    vue.createElementVNode(
                                      "span",
                                      {
                                        class: vue.normalizeClass("status-" + (item.status || "on"))
                                      },
                                      vue.toDisplayString(item.status === "off" ? "关闭" : item.status === "max" ? "始终激活" : "按需"),
                                      3
                                      /* TEXT, CLASS */
                                    ),
                                    vue.createTextVNode(
                                      " · " + vue.toDisplayString(item.level === "system" ? "用户级" : "工作区级"),
                                      1
                                      /* TEXT */
                                    )
                                  ])
                                ]),
                                vue.createElementVNode("div", _hoisted_64, [
                                  vue.createElementVNode("select", {
                                    class: "ss-status-select",
                                    value: item.status || "on",
                                    onChange: ($event) => setSkillStatus(item, $event.target.value),
                                    title: statusTitle(item)
                                  }, [..._cache[39] || (_cache[39] = [
                                    vue.createElementVNode(
                                      "option",
                                      { value: "off" },
                                      "关闭",
                                      -1
                                      /* CACHED */
                                    ),
                                    vue.createElementVNode(
                                      "option",
                                      { value: "on" },
                                      "按需",
                                      -1
                                      /* CACHED */
                                    ),
                                    vue.createElementVNode(
                                      "option",
                                      { value: "max" },
                                      "始终激活",
                                      -1
                                      /* CACHED */
                                    )
                                  ])], 40, _hoisted_65),
                                  vue.createElementVNode("button", {
                                    class: "ii-btn ii-view",
                                    onClick: ($event) => viewSkill(item),
                                    title: "查看内容"
                                  }, "查看", 8, _hoisted_66),
                                  item.level !== "system" ? (vue.openBlock(), vue.createElementBlock("button", {
                                    key: 0,
                                    class: "ii-btn ii-del",
                                    onClick: ($event) => delSkill(item),
                                    title: "删除"
                                  }, "删除", 8, _hoisted_67)) : vue.createCommentVNode("v-if", true)
                                ])
                              ]);
                            }),
                            128
                            /* KEYED_FRAGMENT */
                          ))
                        ],
                        512
                        /* NEED_PATCH */
                      ), [
                        [vue.vShow, !collapsedGroups.value.has("skill")]
                      ])
                    ])) : vue.createCommentVNode("v-if", true),
                    installedMCPs.value.length === 0 && installedSkills.value.length === 0 && installedPlugins.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_68, [
                      vue.createElementVNode("div", _hoisted_69, [
                        vue.createVNode(SvgIcon, {
                          name: "package",
                          size: 32
                        })
                      ]),
                      _cache[40] || (_cache[40] = vue.createElementVNode(
                        "div",
                        null,
                        "暂无已安装内容",
                        -1
                        /* CACHED */
                      )),
                      _cache[41] || (_cache[41] = vue.createElementVNode(
                        "div",
                        { class: "me-hint" },
                        "切换到「全部」tab 搜索安装，或点击上方「添加 MCP 服务器」",
                        -1
                        /* CACHED */
                      ))
                    ])) : vue.createCommentVNode("v-if", true)
                  ],
                  512
                  /* NEED_PATCH */
                )
              ],
              2112
              /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
            )) : (vue.openBlock(), vue.createElementBlock(
              vue.Fragment,
              { key: 7 },
              [
                vue.createCommentVNode(" 搜索/浏览列表 "),
                vue.createElementVNode(
                  "div",
                  {
                    class: "market-list",
                    ref_key: "listRef",
                    ref: listRef
                  },
                  [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(items.value, (item) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: item.id,
                          class: "market-item"
                        }, [
                          vue.createElementVNode(
                            "div",
                            {
                              class: vue.normalizeClass(["mi-icon", "icon-" + item.kind])
                            },
                            [
                              vue.createVNode(SvgIcon, {
                                name: item.kind === "plugin" ? "puzzle" : item.kind === "skill" ? "code" : "package",
                                size: 20
                              }, null, 8, ["name"])
                            ],
                            2
                            /* CLASS */
                          ),
                          vue.createElementVNode("div", _hoisted_70, [
                            vue.createElementVNode(
                              "div",
                              _hoisted_71,
                              vue.toDisplayString(item.name),
                              1
                              /* TEXT */
                            ),
                            vue.createElementVNode(
                              "div",
                              _hoisted_72,
                              vue.toDisplayString(item.description),
                              1
                              /* TEXT */
                            ),
                            vue.createElementVNode("div", _hoisted_73, [
                              vue.createElementVNode(
                                "span",
                                {
                                  class: vue.normalizeClass(["mi-type", "type-" + item.kind])
                                },
                                vue.toDisplayString(item.kind === "mcp" ? "MCP" : item.kind === "plugin" ? "插件" : "技能"),
                                3
                                /* TEXT, CLASS */
                              ),
                              item.tags ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_74, [
                                (vue.openBlock(true), vue.createElementBlock(
                                  vue.Fragment,
                                  null,
                                  vue.renderList(item.tags, (tag) => {
                                    return vue.openBlock(), vue.createElementBlock(
                                      "span",
                                      {
                                        key: tag,
                                        class: "mi-tag"
                                      },
                                      vue.toDisplayString(tag),
                                      1
                                      /* TEXT */
                                    );
                                  }),
                                  128
                                  /* KEYED_FRAGMENT */
                                ))
                              ])) : vue.createCommentVNode("v-if", true),
                              item.installed ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_75, [
                                vue.createVNode(SvgIcon, {
                                  name: "check",
                                  size: 10
                                }),
                                _cache[42] || (_cache[42] = vue.createTextVNode(
                                  " 已安装",
                                  -1
                                  /* CACHED */
                                ))
                              ])) : vue.createCommentVNode("v-if", true)
                            ])
                          ]),
                          !item.installed ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_76, [
                            item.kind !== "mcp" ? (vue.openBlock(), vue.createElementBlock("button", {
                              key: 0,
                              class: "mi-install-btn",
                              onClick: ($event) => installItem(item, "user"),
                              disabled: installing.value === item.id
                            }, [
                              installing.value === item.id ? (vue.openBlock(), vue.createBlock(SvgIcon, {
                                key: 0,
                                name: "cycle",
                                size: 12
                              })) : vue.createCommentVNode("v-if", true),
                              vue.createTextVNode(
                                " " + vue.toDisplayString(installing.value === item.id ? "安装中…" : "安装"),
                                1
                                /* TEXT */
                              )
                            ], 8, _hoisted_77)) : (vue.openBlock(), vue.createElementBlock(
                              vue.Fragment,
                              { key: 1 },
                              [
                                vue.withDirectives(vue.createElementVNode("select", {
                                  "onUpdate:modelValue": ($event) => item._installScope = $event,
                                  class: "mi-scope-select",
                                  onClick: _cache[19] || (_cache[19] = vue.withModifiers(() => {
                                  }, ["stop"]))
                                }, [..._cache[43] || (_cache[43] = [
                                  vue.createElementVNode(
                                    "option",
                                    { value: "user" },
                                    "用户级",
                                    -1
                                    /* CACHED */
                                  ),
                                  vue.createElementVNode(
                                    "option",
                                    { value: "project" },
                                    "工作区级",
                                    -1
                                    /* CACHED */
                                  )
                                ])], 8, _hoisted_78), [
                                  [vue.vModelSelect, item._installScope]
                                ]),
                                vue.createElementVNode("button", {
                                  class: "mi-install-btn",
                                  onClick: ($event) => installItem(item, item._installScope || "user"),
                                  disabled: installing.value === item.id
                                }, [
                                  installing.value === item.id ? (vue.openBlock(), vue.createBlock(SvgIcon, {
                                    key: 0,
                                    name: "cycle",
                                    size: 12
                                  })) : vue.createCommentVNode("v-if", true),
                                  vue.createTextVNode(
                                    " " + vue.toDisplayString(installing.value === item.id ? "安装中…" : "安装"),
                                    1
                                    /* TEXT */
                                  )
                                ], 8, _hoisted_79)
                              ],
                              64
                              /* STABLE_FRAGMENT */
                            ))
                          ])) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_80, [
                            vue.createElementVNode("button", {
                              class: "mi-uninstall-btn",
                              onClick: ($event) => uninstallItem(item),
                              title: "卸载：从配置中移除"
                            }, [
                              vue.createVNode(SvgIcon, {
                                name: "trash",
                                size: 12
                              }),
                              _cache[44] || (_cache[44] = vue.createTextVNode(
                                " 卸载 ",
                                -1
                                /* CACHED */
                              ))
                            ], 8, _hoisted_81)
                          ]))
                        ]);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    )),
                    !loading.value && items.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_82, [
                      vue.createElementVNode("div", _hoisted_83, [
                        vue.createVNode(SvgIcon, {
                          name: "package",
                          size: 32
                        })
                      ]),
                      query.value ? (vue.openBlock(), vue.createElementBlock(
                        "div",
                        _hoisted_84,
                        '未找到匹配 "' + vue.toDisplayString(query.value) + '" 的条目',
                        1
                        /* TEXT */
                      )) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_85, "市场中暂无可用条目")),
                      _cache[45] || (_cache[45] = vue.createElementVNode(
                        "div",
                        { class: "me-hint" },
                        "试试其他关键词或分类",
                        -1
                        /* CACHED */
                      ))
                    ])) : vue.createCommentVNode("v-if", true)
                  ],
                  512
                  /* NEED_PATCH */
                )
              ],
              2112
              /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
            ))
          ]),
          vue.createElementVNode("div", _hoisted_86, [
            vue.createElementVNode(
              "span",
              _hoisted_87,
              vue.toDisplayString(tab.value === "installed" ? installedMCPs.value.length + installedSkills.value.length + installedPlugins.value.length : items.value.length) + " 个条目",
              1
              /* TEXT */
            ),
            error.value ? (vue.openBlock(), vue.createElementBlock(
              "span",
              _hoisted_88,
              vue.toDisplayString(error.value),
              1
              /* TEXT */
            )) : vue.createCommentVNode("v-if", true),
            _cache[46] || (_cache[46] = vue.createElementVNode(
              "span",
              { class: "market-tip" },
              "安装后下次对话生效",
              -1
              /* CACHED */
            )),
            vue.createElementVNode("button", {
              class: "btn-secondary",
              onClick: closePanel
            }, "关闭")
          ])
        ]);
      };
    }
  };
  const MarketplacePanel2 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-67aec65d"]]);
  function mount(el) {
    const app = vue.createApp(MarketplacePanel2);
    app.mount(el);
    return () => {
      app.unmount();
    };
  }
  exports.mount = mount;
  Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
  return exports;
})({}, window.__PAIRCODE_CORE.Vue, window.__PAIRCODE_CORE.api, window.__PAIRCODE_CORE.uiState);
