var UiSidebar = (function(exports, vue, uiState_js, api, pluginRuntime_js) {
  "use strict";
  const _export_sfc = (sfc, props) => {
    const target = sfc.__vccOpts || sfc;
    for (const [key, val] of props) {
      target[key] = val;
    }
    return target;
  };
  const _hoisted_1$7 = ["width", "height"];
  const _hoisted_2$7 = {
    key: 0,
    d: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
  };
  const _sfc_main$7 = {
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
          __props.name === "folder" ? (vue.openBlock(), vue.createElementBlock("path", _hoisted_2$7)) : __props.name === "folder-open" ? (vue.openBlock(), vue.createElementBlock(
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
        ], 8, _hoisted_1$7);
      };
    }
  };
  const SvgIcon = /* @__PURE__ */ _export_sfc(_sfc_main$7, [["__scopeId", "data-v-faf69761"]]);
  const _hoisted_1$6 = {
    key: 0,
    class: "ctx-title"
  };
  const _hoisted_2$6 = {
    key: 0,
    class: "ctx-separator"
  };
  const _hoisted_3$6 = ["onClick"];
  const _hoisted_4$5 = { class: "ctx-label" };
  const _hoisted_5$4 = {
    key: 1,
    class: "ctx-shortcut"
  };
  const _sfc_main$6 = {
    __name: "ContextMenu",
    setup(__props, { expose: __expose }) {
      const visible = vue.ref(false);
      const position = vue.ref({ x: 0, y: 0 });
      const items = vue.ref([]);
      const title = vue.ref("");
      let resolvePromise = null;
      const menuStyle = vue.computed(() => ({
        left: position.value.x + "px",
        top: position.value.y + "px"
      }));
      function show(opts) {
        return new Promise((resolve) => {
          const { x, y, items: menuItems, title: t } = opts;
          let px = x;
          let py = y;
          const itemCount = menuItems.filter((m) => !m.separator).length;
          const sepCount = menuItems.filter((m) => m.separator).length;
          const menuHeight = itemCount * 28 + sepCount * 8 + 40;
          if (px + 200 > window.innerWidth - 10) px = Math.max(10, window.innerWidth - 210);
          if (py + menuHeight > window.innerHeight - 10) py = Math.max(10, window.innerHeight - menuHeight - 10);
          position.value = { x: px, y: py };
          items.value = menuItems;
          title.value = t || "";
          visible.value = true;
          resolvePromise = resolve;
        });
      }
      function close(result = null) {
        visible.value = false;
        if (resolvePromise) {
          resolvePromise(result);
          resolvePromise = null;
        }
      }
      function onItemClick(item) {
        if (item.disabled) return;
        close(item.action || item.label);
      }
      function handleKeydown(e) {
        if (e.key === "Escape" && visible.value) close(null);
      }
      vue.onMounted(() => document.addEventListener("keydown", handleKeydown));
      vue.onUnmounted(() => document.removeEventListener("keydown", handleKeydown));
      __expose({ show, close });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createBlock(vue.Teleport, { to: "body" }, [
          visible.value ? (vue.openBlock(), vue.createElementBlock(
            "div",
            {
              key: 0,
              class: "context-menu-overlay",
              onMousedown: vue.withModifiers(close, ["self"]),
              onContextmenu: vue.withModifiers(close, ["prevent"])
            },
            [
              vue.createElementVNode(
                "div",
                {
                  class: "context-menu",
                  style: vue.normalizeStyle(menuStyle.value),
                  onContextmenu: _cache[0] || (_cache[0] = vue.withModifiers(() => {
                  }, ["prevent"]))
                },
                [
                  title.value ? (vue.openBlock(), vue.createElementBlock(
                    "div",
                    _hoisted_1$6,
                    vue.toDisplayString(title.value),
                    1
                    /* TEXT */
                  )) : vue.createCommentVNode("v-if", true),
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(items.value, (item, i) => {
                      return vue.openBlock(), vue.createElementBlock(
                        vue.Fragment,
                        { key: i },
                        [
                          vue.createCommentVNode(" 分隔线：独立元素，不包裹在 ctx-item 中 "),
                          item.separator ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_2$6)) : (vue.openBlock(), vue.createElementBlock(
                            vue.Fragment,
                            { key: 1 },
                            [
                              vue.createCommentVNode(" 普通菜单项 "),
                              vue.createElementVNode("div", {
                                class: vue.normalizeClass(["ctx-item", { disabled: item.disabled }]),
                                onClick: ($event) => onItemClick(item)
                              }, [
                                item.icon ? (vue.openBlock(), vue.createBlock(SvgIcon, {
                                  key: 0,
                                  name: item.icon,
                                  size: 14,
                                  class: "ctx-icon"
                                }, null, 8, ["name"])) : vue.createCommentVNode("v-if", true),
                                vue.createElementVNode(
                                  "span",
                                  _hoisted_4$5,
                                  vue.toDisplayString(item.label),
                                  1
                                  /* TEXT */
                                ),
                                item.shortcut ? (vue.openBlock(), vue.createElementBlock(
                                  "span",
                                  _hoisted_5$4,
                                  vue.toDisplayString(item.shortcut),
                                  1
                                  /* TEXT */
                                )) : vue.createCommentVNode("v-if", true)
                              ], 10, _hoisted_3$6)
                            ],
                            2112
                            /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                          ))
                        ],
                        64
                        /* STABLE_FRAGMENT */
                      );
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  ))
                ],
                36
                /* STYLE, NEED_HYDRATION */
              )
            ],
            32
            /* NEED_HYDRATION */
          )) : vue.createCommentVNode("v-if", true)
        ]);
      };
    }
  };
  const ContextMenu = /* @__PURE__ */ _export_sfc(_sfc_main$6, [["__scopeId", "data-v-dde82381"]]);
  const _hoisted_1$5 = { class: "file-tree-item" };
  const _hoisted_2$5 = {
    key: 0,
    class: "chevron-wrap"
  };
  const _hoisted_3$5 = {
    key: 1,
    class: "chevron-placeholder"
  };
  const _hoisted_4$4 = { key: 0 };
  const _sfc_main$5 = {
    __name: "FileTreeItem",
    props: {
      item: { type: Object, required: true },
      parentPath: { type: String, default: "" },
      depth: { type: Number, default: 0 },
      defaultExpanded: { type: Boolean, default: false },
      siblings: { type: Array, default: () => [] },
      // 兄弟节点列表（用于 Shift 范围选择）
      siblingIndex: { type: Number, default: 0 }
      // 在 siblings 中的索引
    },
    emits: ["fileClick"],
    setup(__props, { emit: __emit }) {
      const props = __props;
      const emit = __emit;
      const childFullPath = vue.computed(() => {
        if (!props.parentPath) return props.item.path || props.item.name;
        return props.parentPath + "\\" + props.item.name;
      });
      const fullPath = childFullPath;
      const expanded = vue.ref(
        uiState_js.state.expandedDirs[fullPath.value] ?? (props.defaultExpanded && props.item.isDir)
      );
      const children = vue.ref(props.item.children || []);
      const loaded = vue.ref(props.item.loaded || false);
      const dragOver = vue.ref(false);
      const renaming = vue.ref(false);
      const renameValue = vue.ref("");
      const renameInputRef = vue.ref(null);
      const contextMenuRef = vue.ref(null);
      if (props.defaultExpanded && props.item.isDir && !props.item.children) {
        api.apiGet("/fs/list", { path: fullPath.value }).then((entries) => {
          children.value = entries || [];
          loaded.value = true;
        }).catch(() => {
        });
      }
      vue.computed(() => {
        return props.parentPath || "";
      });
      const fileIcon = vue.computed(() => {
        if (props.item.isDir) return expanded.value ? "folder-open" : "folder";
        const ext = (props.item.name || "").split(".").pop().toLowerCase();
        const iconMap = {
          js: "file-js",
          jsx: "file-js",
          ts: "file-ts",
          tsx: "file-ts",
          go: "file-go",
          py: "file-py",
          java: "file-java",
          html: "file-html",
          htm: "file-html",
          css: "file-css",
          scss: "file-css",
          less: "file-css",
          json: "file-json",
          yaml: "file-text",
          yml: "file-text",
          toml: "file-text",
          md: "file-md",
          mdx: "file-md",
          vue: "file-vue",
          svelte: "file-code",
          rs: "file-code",
          rb: "file-code",
          php: "file-code",
          c: "file-code",
          cpp: "file-code",
          h: "file-code",
          hpp: "file-code",
          swift: "file-code",
          kt: "file-code",
          dart: "file-code",
          xml: "file-code",
          svg: "file-code",
          gitignore: "file-text",
          env: "file-text",
          editorconfig: "file-text",
          mod: "file-text",
          sum: "file-text",
          png: "file",
          jpg: "file",
          jpeg: "file",
          gif: "file",
          ico: "file",
          woff: "file",
          woff2: "file",
          ttf: "file",
          eot: "file",
          zip: "file",
          tar: "file",
          gz: "file",
          rar: "file",
          pdf: "file",
          doc: "file",
          docx: "file",
          xls: "file",
          xlsx: "file"
        };
        return iconMap[ext] || "file";
      });
      const isSelected = vue.computed(() => uiState_js.state.selectedFilePaths.includes(fullPath.value));
      const handleClick = async (e) => {
        if (props.item.isDir) {
          expanded.value = !expanded.value;
          uiState_js.state.expandedDirs[fullPath.value] = expanded.value;
          if (expanded.value && !loaded.value) {
            try {
              const entries = await api.apiGet("/fs/list", { path: fullPath.value });
              children.value = entries || [];
              loaded.value = true;
            } catch {
            }
          }
          return;
        }
        if (e.ctrlKey || e.metaKey) {
          const idx = uiState_js.state.selectedFilePaths.indexOf(fullPath.value);
          if (idx >= 0) {
            uiState_js.state.selectedFilePaths.splice(idx, 1);
          } else {
            uiState_js.state.selectedFilePaths.push(fullPath.value);
          }
          uiState_js.state.lastClickedFilePath = fullPath.value;
        } else if (e.shiftKey && uiState_js.state.lastClickedFilePath) {
          const sibs = props.siblings;
          if (sibs && sibs.length > 0) {
            const sibPaths = sibs.map((s) => {
              if (typeof s === "string") return s;
              return s.path || (props.parentPath ? props.parentPath + "\\" + s.name : s.name);
            });
            const anchorIdx = sibPaths.indexOf(uiState_js.state.lastClickedFilePath);
            const curIdx = sibPaths.indexOf(fullPath.value);
            if (anchorIdx >= 0 && curIdx >= 0) {
              uiState_js.state.selectedFilePaths.length = 0;
              const start = Math.min(anchorIdx, curIdx);
              const end = Math.max(anchorIdx, curIdx);
              for (let i = start; i <= end; i++) {
                const sp = sibPaths[i];
                if (sp && sp !== fullPath.value) {
                  uiState_js.state.selectedFilePaths.push(sp);
                }
              }
              uiState_js.state.selectedFilePaths.push(fullPath.value);
            } else {
              uiState_js.state.selectedFilePaths.length = 0;
              uiState_js.state.selectedFilePaths.push(fullPath.value);
              uiState_js.state.lastClickedFilePath = fullPath.value;
            }
          } else {
            uiState_js.state.selectedFilePaths.length = 0;
            uiState_js.state.selectedFilePaths.push(fullPath.value);
            uiState_js.state.lastClickedFilePath = fullPath.value;
          }
        } else {
          uiState_js.state.selectedFilePaths.length = 0;
          uiState_js.state.selectedFilePaths.push(fullPath.value);
          uiState_js.state.lastClickedFilePath = fullPath.value;
          emit("fileClick", fullPath.value);
        }
      };
      async function showContextMenu(e) {
        await vue.nextTick();
        const isDir = props.item.isDir;
        const path = fullPath.value;
        const name = props.item.name;
        const isMultiSelected = !isDir && uiState_js.state.selectedFilePaths.length > 1 && uiState_js.state.selectedFilePaths.includes(path);
        const selCount = isMultiSelected ? uiState_js.state.selectedFilePaths.length : 0;
        let menuItems = [];
        if (isDir) {
          menuItems = [
            { label: "展开/折叠", action: "toggle-expand" },
            { separator: true },
            { label: "新建文件", icon: "file-plus", action: "new-file" },
            { label: "新建文件夹", icon: "folder-plus", action: "new-folder" },
            { separator: true },
            { label: "添加到对话", icon: "message-square", action: "add-to-chat" },
            { separator: true },
            { label: "剪切", action: "cut" },
            { label: "复制", action: "copy" },
            { label: "粘贴", action: "paste" },
            { separator: true },
            { label: "重命名", shortcut: "F2", action: "rename" },
            { label: "删除", action: "delete" },
            { separator: true },
            { label: "复制路径", icon: "copy", action: "copy-path" },
            { label: "复制相对路径", action: "copy-rel-path" },
            { separator: true },
            { label: "在终端中打开", icon: "terminal", action: "open-terminal" },
            { label: "在资源管理器中显示", action: "show-in-explorer" },
            { label: "从工作区移除", action: "remove-from-workspace" }
          ];
        } else {
          menuItems = [
            { label: "打开", action: "open" },
            { label: "打开到侧边", action: "open-side" },
            { separator: true },
            { label: "添加到对话", icon: "message-square", action: "add-to-chat" },
            { separator: true },
            { label: "剪切", action: "cut" },
            { label: "复制", action: "copy" },
            { label: "粘贴", action: "paste" },
            { separator: true },
            { label: "重命名", shortcut: "F2", action: "rename" },
            { label: isMultiSelected ? `删除选中的 ${selCount} 个文件` : "删除", action: "delete" },
            { separator: true },
            { label: "复制路径", icon: "copy", action: "copy-path" },
            { label: "复制相对路径", action: "copy-rel-path" },
            { label: "复制文件名", action: "copy-filename" },
            { separator: true },
            { label: "在终端中打开", icon: "terminal", action: "open-terminal" },
            { label: "在资源管理器中显示", action: "show-in-explorer" }
          ];
        }
        const result = await contextMenuRef.value.show({
          x: e.clientX,
          y: e.clientY,
          title: isMultiSelected ? `已选中 ${selCount} 个文件` : name,
          items: menuItems
        });
        if (!result) return;
        switch (result) {
          // 基本操作
          case "open":
            openFile(path);
            break;
          case "open-side":
            openFile(path);
            break;
          case "toggle-expand":
            handleClick();
            break;
          // 新建
          case "new-file":
            await createNewFile();
            break;
          case "new-folder":
            await createNewFolder();
            break;
          // 剪贴板
          case "cut":
            copyPath(path);
            break;
          case "copy":
            copyPath(path);
            break;
          case "paste":
            break;
          // 文件操作
          case "rename":
            startRename();
            break;
          case "delete": {
            if (isMultiSelected) {
              await deleteSelected();
            } else {
              await deleteItem();
            }
            break;
          }
          // 路径复制
          case "copy-path":
            copyPath(path);
            break;
          case "copy-rel-path":
            copyRelPath(path);
            break;
          case "copy-filename":
            navigator.clipboard.writeText(name).catch(() => {
            });
            break;
          // 终端/系统
          case "open-terminal":
            openInTerminal(path, isDir);
            break;
          case "show-in-explorer":
            showInExplorer(path);
            break;
          // 添加到对话
          case "add-to-chat":
            await addToChat(path, name, isDir);
            break;
          // 添加到工作区
          case "remove-from-workspace":
            await removeFromWorkspace(path);
            break;
        }
      }
      function openFile(path) {
        if (!uiState_js.state.openFiles.includes(path)) uiState_js.state.openFiles.push(path);
        uiState_js.state.activeFile = path;
        loadFileContent(path);
      }
      async function loadFileContent(path) {
        if (uiState_js.state.fileDirty[path]) return;
        delete uiState_js.state.fileContents[path];
        delete uiState_js.state.fileSavedContent[path];
        delete uiState_js.state.fileDirty[path];
        try {
          const data = await api.apiGet("/fs/read", { path });
          const normalized = (data.content || "").replace(/\r\n/g, "\n");
          uiState_js.state.fileSavedContent[path] = normalized;
          uiState_js.state.fileContents[path] = normalized;
          uiState_js.state.fileDirty[path] = false;
        } catch (err) {
          uiState_js.state.fileContents[path] = "// 错误: " + err.message;
        }
      }
      async function createNewFile() {
        const name = await window.$prompt("输入文件名:", "", "新建文件");
        if (!name) return;
        try {
          await api.apiPost("/fs/write", { path: fullPath.value + "\\" + name, content: "" });
          await reloadChildren();
        } catch (err) {
          window.$toast("创建文件失败: " + err.message, "error");
        }
      }
      async function createNewFolder() {
        const name = await window.$prompt("输入文件夹名:", "", "新建文件夹");
        if (!name) return;
        try {
          await api.apiPost("/fs/mkdir", { path: fullPath.value + "\\" + name });
          await reloadChildren();
        } catch (err) {
          window.$toast("创建文件夹失败: " + err.message, "error");
        }
      }
      function startRename() {
        renameValue.value = props.item.name;
        renaming.value = true;
        vue.nextTick(() => {
          if (renameInputRef.value) {
            renameInputRef.value.focus();
            renameInputRef.value.select();
          }
        });
      }
      async function confirmRename() {
        if (!renaming.value) return;
        const newName = renameValue.value.trim();
        renaming.value = false;
        if (!newName || newName === props.item.name) return;
        const from = fullPath.value;
        const to = props.parentPath + "\\" + newName;
        try {
          await api.apiPost("/fs/rename", { from, to });
          await reloadChildren();
          if (uiState_js.state.activeFile === from) {
            uiState_js.state.activeFile = to;
            const idx = uiState_js.state.openFiles.indexOf(from);
            if (idx !== -1) {
              uiState_js.state.openFiles[idx] = to;
            }
            if (uiState_js.state.fileContents[from]) {
              uiState_js.state.fileContents[to] = uiState_js.state.fileContents[from];
              delete uiState_js.state.fileContents[from];
            }
            if (uiState_js.state.fileDirty[from]) {
              uiState_js.state.fileDirty[to] = uiState_js.state.fileDirty[from];
              delete uiState_js.state.fileDirty[from];
            }
          }
        } catch (err) {
          window.$toast("重命名失败: " + err.message, "error");
        }
      }
      function cancelRename() {
        renaming.value = false;
      }
      async function deleteItem() {
        if (!await window.$confirm("确认删除 " + (props.item.isDir ? "文件夹" : "文件") + ' "' + props.item.name + '" ？')) return;
        try {
          await api.apiPost("/fs/delete", { path: fullPath.value });
          if (uiState_js.state.activeFile === fullPath.value) {
            uiState_js.state.openFiles = uiState_js.state.openFiles.filter((f) => f !== fullPath.value);
            delete uiState_js.state.fileContents[fullPath.value];
            delete uiState_js.state.fileDirty[fullPath.value];
            uiState_js.state.activeFile = uiState_js.state.openFiles[uiState_js.state.openFiles.length - 1] || "";
          }
          await reloadChildren();
        } catch (err) {
          window.$toast("删除失败: " + err.message, "error");
        }
      }
      async function deleteSelected() {
        const paths = [...uiState_js.state.selectedFilePaths];
        if (paths.length === 0) return;
        if (!await window.$confirm(`确认删除选中的 ${paths.length} 个文件？`)) return;
        let ok = 0;
        let fail = 0;
        for (const fp of paths) {
          try {
            await api.apiPost("/fs/delete", { path: fp });
            ok++;
            if (uiState_js.state.activeFile === fp) {
              uiState_js.state.openFiles = uiState_js.state.openFiles.filter((f) => f !== fp);
              delete uiState_js.state.fileContents[fp];
              delete uiState_js.state.fileDirty[fp];
            }
          } catch (err) {
            fail++;
            console.warn(`[批量删除] 删除 ${fp} 失败:`, err);
          }
        }
        uiState_js.state.activeFile = uiState_js.state.openFiles[uiState_js.state.openFiles.length - 1] || "";
        uiState_js.state.selectedFilePaths.length = 0;
        if (fail === 0) {
          window.$toast(`已删除 ${ok} 个文件`, "success");
        } else {
          window.$toast(`删除完成: ${ok} 成功, ${fail} 失败`, "error");
        }
        await reloadChildren();
      }
      function copyPath(path) {
        navigator.clipboard.writeText(path).catch(() => {
        });
      }
      function copyRelPath(path) {
        const root = uiState_js.state.workspaceRoot || "";
        if (root && path.startsWith(root)) {
          const rel = path.slice(root.length).replace(/^\\/, "");
          navigator.clipboard.writeText(rel).catch(() => {
          });
        } else {
          navigator.clipboard.writeText(path).catch(() => {
          });
        }
      }
      function openInTerminal(path, isDir) {
        const dir = isDir ? path : props.parentPath || path.substring(0, path.lastIndexOf("\\"));
        uiState_js.state.bottomPanelVisible = true;
        uiState_js.state.bottomPanelTab = "terminal";
        window.dispatchEvent(new CustomEvent("terminal-cwd", { detail: { cwd: dir } }));
      }
      function showInExplorer(path) {
        const cmd = `explorer /select,"${path}"`;
        try {
          api.apiPost("/system/exec", { command: cmd });
        } catch {
        }
      }
      async function removeFromWorkspace(path) {
        try {
          await api.apiPost("/workspace", { action: "remove-folder", path });
          window.dispatchEvent(new CustomEvent("refresh-tree"));
        } catch (err) {
          window.$toast("移除失败: " + err.message, "error");
        }
      }
      async function addToChat(path, name, isDir) {
        if (isDir) {
          window.dispatchEvent(new CustomEvent("add-to-chat", {
            detail: { type: "dir", path, filename: name }
          }));
        } else {
          window.dispatchEvent(new CustomEvent("add-to-chat", {
            detail: { type: "file", path, filename: name }
          }));
        }
        uiState_js.state.rightPanelVisible = true;
      }
      async function reloadChildren() {
        try {
          const entries = await api.apiGet("/fs/list", { path: props.parentPath });
          children.value = entries || [];
        } catch {
        }
        window.dispatchEvent(new CustomEvent("refresh-tree"));
      }
      const onDragStart = (e) => {
        let paths = [];
        if (uiState_js.state.selectedFilePaths.length > 1 && uiState_js.state.selectedFilePaths.includes(fullPath.value)) {
          paths = uiState_js.state.selectedFilePaths;
        } else {
          paths = [fullPath.value];
        }
        e.dataTransfer.setData("text/plain", JSON.stringify(paths));
        e.dataTransfer.effectAllowed = e.ctrlKey ? "copy" : "move";
        if (e.dataTransfer.setDragImage && paths.length === 1) {
          const el = e.currentTarget;
          e.dataTransfer.setDragImage(el, 10, 10);
        }
      };
      const onDragOver = (e) => {
        if (props.item.isDir) {
          dragOver.value = true;
          e.dataTransfer.dropEffect = e.ctrlKey ? "copy" : "move";
        }
      };
      const onDragLeave = () => {
        dragOver.value = false;
      };
      const onDrop = async (e) => {
        dragOver.value = false;
        if (!props.item.isDir) return;
        const raw = e.dataTransfer.getData("text/plain");
        if (!raw) return;
        let paths = [];
        try {
          paths = JSON.parse(raw);
        } catch {
          paths = [raw];
        }
        if (!Array.isArray(paths)) paths = [paths];
        if (paths.length === 0) return;
        const targetDir = fullPath.value;
        const isCopy = e.ctrlKey || e.shiftKey;
        let successCount = 0;
        let failCount = 0;
        for (const srcPath of paths) {
          if (!srcPath || srcPath === targetDir || srcPath.startsWith(targetDir + "\\")) continue;
          const srcName = srcPath.split("\\").pop();
          const destPath = targetDir + "\\" + srcName;
          try {
            if (isCopy) {
              await api.apiPost("/fs/copy", { from: srcPath, to: destPath });
              if (uiState_js.state.fileContents[srcPath]) {
                uiState_js.state.fileContents[destPath] = uiState_js.state.fileContents[srcPath];
                uiState_js.state.fileSavedContent[destPath] = uiState_js.state.fileSavedContent[srcPath];
              }
            } else {
              await api.apiPost("/fs/rename", { from: srcPath, to: destPath });
              if (uiState_js.state.activeFile === srcPath) {
                uiState_js.state.activeFile = destPath;
                const idx = uiState_js.state.openFiles.indexOf(srcPath);
                if (idx !== -1) uiState_js.state.openFiles[idx] = destPath;
              }
              if (uiState_js.state.fileContents[srcPath]) {
                uiState_js.state.fileContents[destPath] = uiState_js.state.fileContents[srcPath];
                uiState_js.state.fileSavedContent[destPath] = uiState_js.state.fileSavedContent[srcPath];
                delete uiState_js.state.fileContents[srcPath];
                delete uiState_js.state.fileSavedContent[srcPath];
                delete uiState_js.state.fileDirty[srcPath];
              }
            }
            successCount++;
          } catch (err) {
            console.warn("[拖拽] 操作失败:", srcPath, "→", destPath, err);
            failCount++;
          }
        }
        if (successCount > 0) {
          window.dispatchEvent(new CustomEvent("refresh-tree"));
          window.$toast(isCopy ? `已复制 ${successCount} 个${failCount > 0 ? "（" + failCount + " 个失败）" : ""}` : `已移动 ${successCount} 个${failCount > 0 ? "（" + failCount + " 个失败）" : ""}`, "success");
        } else if (failCount > 0) {
          window.$toast("拖拽操作失败: " + failCount + " 个错误", "error");
        }
      };
      function onRefreshTree() {
        if (expanded.value && props.item.isDir) {
          uiState_js.state.expandedDirs[fullPath.value] = true;
          loaded.value = false;
          api.apiGet("/fs/list", { path: fullPath.value }).then((entries) => {
            children.value = entries || [];
            loaded.value = true;
          }).catch(() => {
          });
        }
      }
      vue.onMounted(() => {
        window.addEventListener("refresh-tree", onRefreshTree);
      });
      vue.onUnmounted(() => {
        window.removeEventListener("refresh-tree", onRefreshTree);
      });
      return (_ctx, _cache) => {
        const _component_FileTreeItem = vue.resolveComponent("FileTreeItem", true);
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1$5, [
          vue.createElementVNode(
            "div",
            {
              class: vue.normalizeClass(["item-row", { "drag-over": dragOver.value, "selected": isSelected.value }]),
              style: vue.normalizeStyle({ paddingLeft: __props.depth * 16 + "px" }),
              draggable: "true",
              onClick: _cache[0] || (_cache[0] = ($event) => handleClick($event)),
              onContextmenu: vue.withModifiers(showContextMenu, ["prevent"]),
              onDragstart: onDragStart,
              onDragover: vue.withModifiers(onDragOver, ["prevent"]),
              onDragleave: onDragLeave,
              onDrop: vue.withModifiers(onDrop, ["prevent"])
            },
            [
              __props.item.isDir ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_2$5, [
                vue.createVNode(SvgIcon, {
                  name: "chevron-right",
                  size: 10,
                  class: vue.normalizeClass(["chevron", { expanded: expanded.value }])
                }, null, 8, ["class"])
              ])) : (vue.openBlock(), vue.createElementBlock("span", _hoisted_3$5)),
              vue.createVNode(SvgIcon, {
                name: fileIcon.value,
                size: 14
              }, null, 8, ["name"]),
              vue.createElementVNode(
                "span",
                {
                  class: vue.normalizeClass(["item-name", { active: vue.unref(uiState_js.state).activeFile === vue.unref(fullPath) }])
                },
                vue.toDisplayString(__props.item.name),
                3
                /* TEXT, CLASS */
              )
            ],
            38
            /* CLASS, STYLE, NEED_HYDRATION */
          ),
          expanded.value && __props.item.isDir && children.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_4$4, [
            (vue.openBlock(true), vue.createElementBlock(
              vue.Fragment,
              null,
              vue.renderList(children.value, (child, ci) => {
                return vue.openBlock(), vue.createBlock(_component_FileTreeItem, {
                  key: vue.unref(fullPath) + "\\" + child.name + "_" + ci,
                  item: child,
                  parentPath: vue.unref(fullPath),
                  depth: __props.depth + 1,
                  siblings: children.value,
                  siblingIndex: ci,
                  onFileClick: _cache[1] || (_cache[1] = (p) => emit("fileClick", p))
                }, null, 8, ["item", "parentPath", "depth", "siblings", "siblingIndex"]);
              }),
              128
              /* KEYED_FRAGMENT */
            ))
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 重命名输入框 "),
          renaming.value ? (vue.openBlock(), vue.createElementBlock(
            "div",
            {
              key: 1,
              class: "rename-input",
              style: vue.normalizeStyle({ paddingLeft: __props.depth * 16 + 28 + "px" })
            },
            [
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  ref_key: "renameInputRef",
                  ref: renameInputRef,
                  "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => renameValue.value = $event),
                  class: "rename-field",
                  onKeyup: [
                    vue.withKeys(confirmRename, ["enter"]),
                    vue.withKeys(cancelRename, ["escape"])
                  ],
                  onBlur: confirmRename
                },
                null,
                544
                /* NEED_HYDRATION, NEED_PATCH */
              ), [
                [vue.vModelText, renameValue.value]
              ])
            ],
            4
            /* STYLE */
          )) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 右键菜单 "),
          vue.createVNode(
            ContextMenu,
            {
              ref_key: "contextMenuRef",
              ref: contextMenuRef
            },
            null,
            512
            /* NEED_PATCH */
          )
        ]);
      };
    }
  };
  const FileTreeItem = /* @__PURE__ */ _export_sfc(_sfc_main$5, [["__scopeId", "data-v-493c29a3"]]);
  const _hoisted_1$4 = { class: "dialog-box ts-transfer-box" };
  const _hoisted_2$4 = { class: "dialog-title" };
  const _hoisted_3$4 = { class: "dialog-title-main" };
  const _hoisted_4$3 = { class: "ts-transfer-body" };
  const _hoisted_5$3 = { class: "ts-transfer-col" };
  const _hoisted_6$3 = { class: "ts-transfer-list" };
  const _hoisted_7$3 = { class: "ts-transfer-group-head" };
  const _hoisted_8$3 = { class: "ts-transfer-check" };
  const _hoisted_9$3 = ["checked", "onChange"];
  const _hoisted_10$3 = { class: "ts-transfer-group-name" };
  const _hoisted_11$3 = { class: "ts-tool-desc" };
  const _hoisted_12$3 = ["onClick"];
  const _hoisted_13$3 = ["title"];
  const _hoisted_14$3 = ["checked", "onChange"];
  const _hoisted_15$3 = {
    key: 0,
    class: "ts-empty"
  };
  const _hoisted_16$3 = { class: "ts-transfer-ops" };
  const _hoisted_17$3 = ["disabled"];
  const _hoisted_18$3 = ["disabled"];
  const _hoisted_19$3 = { class: "ts-transfer-col" };
  const _hoisted_20$3 = { class: "ts-transfer-list" };
  const _hoisted_21$3 = { class: "ts-transfer-group-head" };
  const _hoisted_22$3 = { class: "ts-transfer-check" };
  const _hoisted_23$3 = ["checked", "onChange"];
  const _hoisted_24$3 = { class: "ts-transfer-group-name" };
  const _hoisted_25$2 = { class: "ts-tool-desc" };
  const _hoisted_26$2 = ["onClick"];
  const _hoisted_27$2 = ["title"];
  const _hoisted_28$2 = ["checked", "onChange"];
  const _hoisted_29$2 = {
    key: 0,
    class: "ts-transfer-group"
  };
  const _hoisted_30$2 = { class: "ts-transfer-group-head" };
  const _hoisted_31$2 = { class: "ts-transfer-check" };
  const _hoisted_32$2 = ["checked"];
  const _hoisted_33$2 = { class: "ts-tool-desc" };
  const _hoisted_34$2 = ["checked", "onChange"];
  const _hoisted_35$2 = {
    key: 1,
    class: "ts-empty"
  };
  const _hoisted_36$2 = {
    key: 0,
    class: "ts-msg"
  };
  const _sfc_main$4 = {
    __name: "ToolsetTransfer",
    props: {
      groups: { type: Array, default: () => [] },
      // 插件分组（source=plugin，含 tools[].enabled）
      joined: { type: Array, default: () => [] },
      // 兼容保留（内置组名，不再用于插件分组判定）
      manualTools: { type: Array, default: () => [] },
      workspaceRoot: { type: String, default: "" }
      // ★ 目标工作区（工作区隔离：管理操作只影响本工作区工具集）
    },
    emits: ["close", "changed"],
    setup(__props, { emit: __emit }) {
      const props = __props;
      const emit = __emit;
      const leftGroups = vue.computed(() => {
        return props.groups.filter((g) => (g.tools || []).some((t) => !t.enabled)).map((g) => ({ ...g, tools: (g.tools || []).filter((t) => !t.enabled) }));
      });
      const joinedGroups = vue.computed(() => {
        return props.groups.filter((g) => (g.tools || []).some((t) => t.enabled)).map((g) => ({ ...g, tools: (g.tools || []).filter((t) => t.enabled) }));
      });
      const manualTools = vue.computed(() => props.manualTools);
      const leftSelected = vue.reactive({});
      const rightSelected = vue.reactive({});
      const busy = vue.ref(false);
      const msg = vue.ref("");
      const msgErr = vue.ref(false);
      const anyLeftSelected = vue.computed(() => Object.values(leftSelected).some(Boolean));
      const anyRightSelected = vue.computed(() => Object.values(rightSelected).some(Boolean));
      function groupAllChecked(g, left) {
        const sel = left ? leftSelected : rightSelected;
        return g.tools.length > 0 && g.tools.every((t) => sel[t.name || t]);
      }
      function toggleGroup(g, left) {
        const sel = left ? leftSelected : rightSelected;
        const target = !groupAllChecked(g, left);
        for (const t of g.tools) sel[t.name || t] = target;
      }
      function toggleSelect(name, left) {
        const sel = left ? leftSelected : rightSelected;
        sel[name] = !sel[name];
      }
      function selectAllLeft() {
        const target = !leftGroups.value.every((g) => groupAllChecked(g, true));
        for (const g of leftGroups.value) {
          for (const t of g.tools) leftSelected[t.name] = target;
        }
      }
      function selectAllRight() {
        const groups = [...joinedGroups.value, ...manualTools.value.length ? [{ tools: manualTools.value }] : []];
        const target = !groups.every((g) => groupAllChecked(g, false));
        for (const g of groups) {
          for (const t of g.tools) rightSelected[t.name || t] = target;
        }
      }
      let once = null;
      async function callOnce(fn) {
        if (once) return;
        once = true;
        busy.value = true;
        try {
          await fn();
          msg.value = "";
          msgErr.value = false;
        } catch (e) {
          msg.value = String(e && e.message || e);
          msgErr.value = true;
        } finally {
          busy.value = false;
          once = null;
        }
      }
      async function addSelected() {
        const byPlugin = {};
        for (const g of leftGroups.value) {
          const names = (g.tools || []).map((t) => t.name).filter((n) => leftSelected[n]);
          if (names.length) byPlugin[g.name] = { joined: !!g.joined, names, source: g.source };
        }
        try {
          for (const [pn, info] of Object.entries(byPlugin)) {
            if (info.source !== "plugin") {
              for (const tn of info.names) {
                await callOnce(() => api.builtinPlugins({ tool: tn, enabled: true }, props.workspaceRoot));
              }
            } else if (info.joined) {
              for (const tn of info.names) {
                await callOnce(() => api.toolsetEdit({ name: "default", action: "enable_tool", plugin_name: pn, tool: tn, workspaceRoot: props.workspaceRoot }));
              }
            } else {
              await callOnce(() => api.toolsetEdit({ name: "default", action: "add_plugin", plugin_name: pn, tools: info.names.join(","), workspaceRoot: props.workspaceRoot }));
            }
          }
          emit("changed");
        } catch (e) {
        }
      }
      async function removeSelected() {
        const byPlugin = {};
        for (const g of joinedGroups.value) {
          const names = (g.tools || []).map((t) => t.name).filter((n) => rightSelected[n]);
          if (names.length) byPlugin[g.name] = { source: g.source, names };
        }
        const manualNames = manualTools.value.filter((n) => rightSelected[n]);
        try {
          for (const [pn, info] of Object.entries(byPlugin)) {
            for (const tn of info.names) {
              if (info.source !== "plugin") {
                await callOnce(() => api.builtinPlugins({ tool: tn, enabled: false }, props.workspaceRoot));
              } else {
                await callOnce(() => api.toolsetEdit({ name: "default", action: "rm_tool", plugin_name: pn, tool: tn, workspaceRoot: props.workspaceRoot }));
              }
            }
          }
          for (const n of manualNames) {
            await callOnce(() => api.builtinPlugins({ tool: n, enabled: false }, props.workspaceRoot));
          }
          emit("changed");
        } catch (e) {
        }
      }
      async function addGroup(g) {
        try {
          if (g.source !== "plugin") {
            if (g.joined) {
              for (const t of g.tools) {
                if (!t.enabled) {
                  await callOnce(() => api.builtinPlugins({ tool: t.name, enabled: true }, props.workspaceRoot));
                }
              }
            } else {
              await callOnce(() => api.builtinPlugins({ group: g.name, enabled: true }, props.workspaceRoot));
            }
          } else if (g.joined) {
            for (const t of g.tools) {
              await callOnce(() => api.toolsetEdit({ name: "default", action: "enable_tool", plugin_name: g.name, tool: t.name, workspaceRoot: props.workspaceRoot }));
            }
          } else {
            await callOnce(() => api.toolsetEdit({ name: "default", action: "add_plugin", plugin_name: g.name, workspaceRoot: props.workspaceRoot }));
          }
          emit("changed");
        } catch (e) {
        }
      }
      async function removeGroup(g) {
        try {
          if (g.source !== "plugin") {
            await callOnce(() => api.builtinPlugins({ group: g.name, enabled: false }, props.workspaceRoot));
          } else {
            await callOnce(() => api.toolsetEdit({ name: "default", action: "rm_plugin", plugin_name: g.name, workspaceRoot: props.workspaceRoot }));
          }
          emit("changed");
        } catch (e) {
        }
      }
      async function removeManualTools() {
        try {
          for (const n of props.manualTools) {
            await callOnce(() => api.builtinPlugins({ tool: n, enabled: false }, props.workspaceRoot));
          }
          emit("changed");
        } catch (e) {
        }
      }
      function close() {
        emit("close");
      }
      return (_ctx, _cache) => {
        const _component_SvgIcon = vue.resolveComponent("SvgIcon");
        return vue.openBlock(), vue.createElementBlock(
          vue.Fragment,
          null,
          [
            vue.createCommentVNode(" 工作区工具集（builtin）穿梭框：左=未加入，右=已加入，勾选批量加入/移出 "),
            vue.createElementVNode("div", {
              class: "dialog-overlay",
              onClick: vue.withModifiers(close, ["self"])
            }, [
              vue.createElementVNode("div", _hoisted_1$4, [
                vue.createElementVNode("div", _hoisted_2$4, [
                  vue.createElementVNode("span", _hoisted_3$4, [
                    vue.createVNode(_component_SvgIcon, {
                      name: "package",
                      size: 14
                    }),
                    _cache[1] || (_cache[1] = vue.createTextVNode(
                      " 管理工作区工具集",
                      -1
                      /* CACHED */
                    ))
                  ]),
                  _cache[2] || (_cache[2] = vue.createElementVNode(
                    "span",
                    { class: "dialog-title-sub" },
                    "插件工具 · 勾选后加入 / 移出工作区工具集",
                    -1
                    /* CACHED */
                  ))
                ]),
                vue.createElementVNode("div", _hoisted_4$3, [
                  vue.createCommentVNode(" 左：未加入 "),
                  vue.createElementVNode("div", _hoisted_5$3, [
                    vue.createElementVNode("div", { class: "ts-transfer-col-head" }, [
                      _cache[3] || (_cache[3] = vue.createElementVNode(
                        "span",
                        { class: "ts-col-title" },
                        "未加入",
                        -1
                        /* CACHED */
                      )),
                      _cache[4] || (_cache[4] = vue.createElementVNode(
                        "span",
                        { class: "ts-col-hint" },
                        "勾选后 → 加入",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode("span", { class: "ts-col-head-actions" }, [
                        vue.createElementVNode("button", {
                          class: "ts-btn mini",
                          onClick: selectAllLeft
                        }, "全选")
                      ])
                    ]),
                    vue.createElementVNode("div", _hoisted_6$3, [
                      (vue.openBlock(true), vue.createElementBlock(
                        vue.Fragment,
                        null,
                        vue.renderList(leftGroups.value, (g) => {
                          return vue.openBlock(), vue.createElementBlock("div", {
                            key: g.name,
                            class: "ts-transfer-group"
                          }, [
                            vue.createElementVNode("div", _hoisted_7$3, [
                              vue.createElementVNode("label", _hoisted_8$3, [
                                vue.createElementVNode("input", {
                                  type: "checkbox",
                                  checked: groupAllChecked(g, true),
                                  onChange: ($event) => toggleGroup(g, true)
                                }, null, 40, _hoisted_9$3),
                                vue.createElementVNode(
                                  "span",
                                  _hoisted_10$3,
                                  vue.toDisplayString(g.name),
                                  1
                                  /* TEXT */
                                ),
                                vue.createElementVNode(
                                  "span",
                                  _hoisted_11$3,
                                  vue.toDisplayString(g.tools.length) + " 工具",
                                  1
                                  /* TEXT */
                                )
                              ]),
                              vue.createElementVNode("button", {
                                class: "ts-btn mini",
                                onClick: ($event) => addGroup(g)
                              }, "整组加入", 8, _hoisted_12$3)
                            ]),
                            (vue.openBlock(true), vue.createElementBlock(
                              vue.Fragment,
                              null,
                              vue.renderList(g.tools, (t) => {
                                return vue.openBlock(), vue.createElementBlock("label", {
                                  key: t.name,
                                  class: "ts-transfer-check ts-transfer-tool",
                                  title: t.desc
                                }, [
                                  vue.createElementVNode("input", {
                                    type: "checkbox",
                                    checked: leftSelected[t.name],
                                    onChange: ($event) => toggleSelect(t.name, true)
                                  }, null, 40, _hoisted_14$3),
                                  vue.createElementVNode(
                                    "span",
                                    null,
                                    vue.toDisplayString(t.name),
                                    1
                                    /* TEXT */
                                  )
                                ], 8, _hoisted_13$3);
                              }),
                              128
                              /* KEYED_FRAGMENT */
                            ))
                          ]);
                        }),
                        128
                        /* KEYED_FRAGMENT */
                      )),
                      !leftGroups.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_15$3, "全部已加入")) : vue.createCommentVNode("v-if", true)
                    ])
                  ]),
                  vue.createCommentVNode(" 中间操作列 "),
                  vue.createElementVNode("div", _hoisted_16$3, [
                    vue.createElementVNode("button", {
                      class: "ts-btn primary",
                      onClick: addSelected,
                      disabled: !anyLeftSelected.value,
                      title: "把选中的工具加入工作区工具集"
                    }, "加入 →", 8, _hoisted_17$3),
                    vue.createElementVNode("button", {
                      class: "ts-btn danger",
                      onClick: removeSelected,
                      disabled: !anyRightSelected.value,
                      title: "把选中的工具移出工作区工具集"
                    }, "← 移出", 8, _hoisted_18$3)
                  ]),
                  vue.createCommentVNode(" 右：已加入 "),
                  vue.createElementVNode("div", _hoisted_19$3, [
                    vue.createElementVNode("div", { class: "ts-transfer-col-head" }, [
                      _cache[5] || (_cache[5] = vue.createElementVNode(
                        "span",
                        { class: "ts-col-title" },
                        "已加入",
                        -1
                        /* CACHED */
                      )),
                      _cache[6] || (_cache[6] = vue.createElementVNode(
                        "span",
                        { class: "ts-col-hint" },
                        "勾选后 ← 移出",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode("span", { class: "ts-col-head-actions" }, [
                        vue.createElementVNode("button", {
                          class: "ts-btn mini",
                          onClick: selectAllRight
                        }, "全选")
                      ])
                    ]),
                    vue.createElementVNode("div", _hoisted_20$3, [
                      (vue.openBlock(true), vue.createElementBlock(
                        vue.Fragment,
                        null,
                        vue.renderList(joinedGroups.value, (g) => {
                          return vue.openBlock(), vue.createElementBlock("div", {
                            key: g.name,
                            class: "ts-transfer-group"
                          }, [
                            vue.createElementVNode("div", _hoisted_21$3, [
                              vue.createElementVNode("label", _hoisted_22$3, [
                                vue.createElementVNode("input", {
                                  type: "checkbox",
                                  checked: groupAllChecked(g, false),
                                  onChange: ($event) => toggleGroup(g, false)
                                }, null, 40, _hoisted_23$3),
                                vue.createElementVNode(
                                  "span",
                                  _hoisted_24$3,
                                  vue.toDisplayString(g.name),
                                  1
                                  /* TEXT */
                                ),
                                vue.createElementVNode(
                                  "span",
                                  _hoisted_25$2,
                                  vue.toDisplayString(g.tools.length) + " 工具",
                                  1
                                  /* TEXT */
                                )
                              ]),
                              vue.createElementVNode("button", {
                                class: "ts-btn mini danger",
                                onClick: ($event) => removeGroup(g)
                              }, "整组移出", 8, _hoisted_26$2)
                            ]),
                            (vue.openBlock(true), vue.createElementBlock(
                              vue.Fragment,
                              null,
                              vue.renderList(g.tools, (t) => {
                                return vue.openBlock(), vue.createElementBlock("label", {
                                  key: t.name,
                                  class: "ts-transfer-check ts-transfer-tool",
                                  title: t.desc
                                }, [
                                  vue.createElementVNode("input", {
                                    type: "checkbox",
                                    checked: rightSelected[t.name],
                                    onChange: ($event) => toggleSelect(t.name, false)
                                  }, null, 40, _hoisted_28$2),
                                  vue.createElementVNode(
                                    "span",
                                    null,
                                    vue.toDisplayString(t.name),
                                    1
                                    /* TEXT */
                                  )
                                ], 8, _hoisted_27$2);
                              }),
                              128
                              /* KEYED_FRAGMENT */
                            ))
                          ]);
                        }),
                        128
                        /* KEYED_FRAGMENT */
                      )),
                      manualTools.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_29$2, [
                        vue.createElementVNode("div", _hoisted_30$2, [
                          vue.createElementVNode("label", _hoisted_31$2, [
                            vue.createElementVNode("input", {
                              type: "checkbox",
                              checked: groupAllChecked({ tools: manualTools.value }, false),
                              onChange: _cache[0] || (_cache[0] = ($event) => toggleGroup({ tools: manualTools.value }, false))
                            }, null, 40, _hoisted_32$2),
                            _cache[7] || (_cache[7] = vue.createElementVNode(
                              "span",
                              { class: "ts-transfer-group-name" },
                              "_manual（手动）",
                              -1
                              /* CACHED */
                            )),
                            vue.createElementVNode(
                              "span",
                              _hoisted_33$2,
                              vue.toDisplayString(manualTools.value.length) + " 工具",
                              1
                              /* TEXT */
                            )
                          ]),
                          vue.createElementVNode("button", {
                            class: "ts-btn mini danger",
                            onClick: removeManualTools
                          }, "整组移出")
                        ]),
                        (vue.openBlock(true), vue.createElementBlock(
                          vue.Fragment,
                          null,
                          vue.renderList(manualTools.value, (t) => {
                            return vue.openBlock(), vue.createElementBlock("label", {
                              key: t,
                              class: "ts-transfer-check ts-transfer-tool"
                            }, [
                              vue.createElementVNode("input", {
                                type: "checkbox",
                                checked: rightSelected[t],
                                onChange: ($event) => toggleSelect(t, false)
                              }, null, 40, _hoisted_34$2),
                              vue.createElementVNode(
                                "span",
                                null,
                                vue.toDisplayString(t),
                                1
                                /* TEXT */
                              )
                            ]);
                          }),
                          128
                          /* KEYED_FRAGMENT */
                        ))
                      ])) : vue.createCommentVNode("v-if", true),
                      !joinedGroups.value.length && !manualTools.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_35$2, "暂无已加入工具")) : vue.createCommentVNode("v-if", true)
                    ])
                  ])
                ]),
                busy.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_36$2, "操作中…")) : vue.createCommentVNode("v-if", true),
                msg.value ? (vue.openBlock(), vue.createElementBlock(
                  "div",
                  {
                    key: 1,
                    class: vue.normalizeClass(["ts-msg", { err: msgErr.value }])
                  },
                  vue.toDisplayString(msg.value),
                  3
                  /* TEXT, CLASS */
                )) : vue.createCommentVNode("v-if", true),
                vue.createElementVNode("div", { class: "dialog-footer" }, [
                  vue.createElementVNode("button", {
                    class: "ts-btn ghost",
                    onClick: close
                  }, "关闭")
                ])
              ])
            ])
          ],
          2112
          /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
        );
      };
    }
  };
  const ToolsetTransfer = /* @__PURE__ */ _export_sfc(_sfc_main$4, [["__scopeId", "data-v-0c20a7bc"]]);
  const _hoisted_1$3 = { class: "file-explorer" };
  const _hoisted_2$3 = { class: "explorer-toolbar" };
  const _hoisted_3$3 = { class: "ws-section" };
  const _hoisted_4$2 = ["onClick", "onContextmenu"];
  const _hoisted_5$2 = { class: "ws-left" };
  const _hoisted_6$2 = { class: "ws-name" };
  const _hoisted_7$2 = { class: "ws-right" };
  const _hoisted_8$2 = {
    key: 0,
    class: "ws-notify",
    title: "有待处理"
  };
  const _hoisted_9$2 = {
    key: 1,
    class: "ws-badge"
  };
  const _hoisted_10$2 = {
    key: 0,
    class: "ws-empty"
  };
  const _hoisted_11$2 = { class: "project-section" };
  const _hoisted_12$2 = {
    key: 0,
    class: "proj-empty"
  };
  const _hoisted_13$2 = {
    key: 2,
    class: "proj-empty"
  };
  const _hoisted_14$2 = { class: "ts-divider" };
  const _hoisted_15$2 = {
    key: 0,
    class: "ts-body"
  };
  const _hoisted_16$2 = { class: "ts-build" };
  const _hoisted_17$2 = { class: "ts-add-list" };
  const _hoisted_18$2 = { class: "ts-add-group-title" };
  const _hoisted_19$2 = ["title"];
  const _hoisted_20$2 = { class: "ts-add-tool-name" };
  const _hoisted_21$2 = ["onClick"];
  const _hoisted_22$2 = {
    key: 0,
    class: "ts-add-group"
  };
  const _hoisted_23$2 = ["title"];
  const _hoisted_24$2 = { class: "ts-add-tool-name" };
  const _hoisted_25$1 = ["onClick"];
  const _hoisted_26$1 = {
    key: 1,
    class: "ts-empty"
  };
  const _hoisted_27$1 = {
    class: "dialog-box",
    style: { "max-width": "420px" }
  };
  const _hoisted_28$1 = { class: "dialog-body" };
  const _hoisted_29$1 = { class: "input-row" };
  const _hoisted_30$1 = { class: "dialog-footer" };
  const _hoisted_31$1 = {
    key: 0,
    class: "dlg-error"
  };
  const _hoisted_32$1 = ["disabled"];
  const _hoisted_33$1 = { class: "dialog-box dir-browser-box" };
  const _hoisted_34$1 = { class: "dialog-title" };
  const _hoisted_35$1 = { class: "dir-browser" };
  const _hoisted_36$1 = { class: "dir-breadcrumb" };
  const _hoisted_37$1 = ["disabled"];
  const _hoisted_38$1 = { class: "bc-path" };
  const _hoisted_39$1 = {
    key: 0,
    class: "dir-list"
  };
  const _hoisted_40$1 = ["onDblclick"];
  const _hoisted_41$1 = { class: "dir-name" };
  const _hoisted_42$1 = {
    key: 1,
    class: "dir-list"
  };
  const _hoisted_43$1 = ["onClick"];
  const _hoisted_44$1 = { class: "dir-name" };
  const _hoisted_45$1 = {
    key: 0,
    class: "dir-empty"
  };
  const _hoisted_46$1 = { class: "dialog-footer" };
  const _hoisted_47$1 = {
    key: 0,
    class: "dlg-error"
  };
  const _hoisted_48$1 = ["disabled"];
  const _sfc_main$3 = {
    __name: "FileExplorer",
    setup(__props) {
      const wsItems = vue.computed(() => uiState_js.state.wsList);
      const currentFolders = vue.computed(() => {
        if (!uiState_js.state.workspaceRoot) return [];
        const cur = uiState_js.state.wsList.find((w) => w.path === uiState_js.state.workspaceRoot);
        if (cur && cur.folders && cur.folders.length > 0) {
          const merged = [...cur.folders, ...uiState_js.state.workspaceFolders].filter(Boolean);
          return [...new Set(merged)];
        }
        if (uiState_js.state.workspaceFolders.length > 0) return [...new Set(uiState_js.state.workspaceFolders.filter(Boolean))];
        return [uiState_js.state.workspaceRoot];
      });
      function folderParent(folderPath) {
        const idx = folderPath.lastIndexOf("\\");
        return idx > 0 ? folderPath.substring(0, idx) : "";
      }
      async function switchToWorkspace(ws) {
        if (ws.path === uiState_js.state.workspaceRoot) return;
        try {
          const folders = ws.folders || [];
          const res = await api.apiPost("/workspace", {
            action: "switch",
            root: ws.path,
            folders: folders.filter((f) => f !== ws.path)
          });
          uiState_js.state.workspaceRoot = ws.path;
          uiState_js.state.workspaceFolders = folders.length > 0 ? [...folders] : [ws.path];
          uiState_js.state.settings.workspaceFolders = [...uiState_js.state.workspaceFolders];
          uiState_js.state.workspaceName = ws.name || ws.path.split("\\").filter(Boolean).pop() || ws.path;
          document.title = "PairCode IDE - " + uiState_js.state.workspaceName;
          uiState_js.state.openFiles = [];
          uiState_js.state.activeFile = "";
          uiState_js.state.fileContents = {};
          uiState_js.state.fileSavedContent = {};
          uiState_js.state.fileDirty = {};
          await loadConversationsForWorkspace(ws.path);
          ws.notify = false;
          uiState_js.state.notificationCount = 0;
        } catch (err) {
          console.error("切换工作区失败:", err);
        }
      }
      async function loadConversationsForWorkspace(path) {
        uiState_js.state.conversations = [];
        uiState_js.state.currentConvId = "";
        uiState_js.state.messages = [];
        if (typeof path !== "string" || !path) return;
        try {
          const list = await api.apiGet("/conversations", { workspace: path });
          uiState_js.state.conversations = list || [];
        } catch (e) {
          console.warn("从后端加载对话消息失败:", e);
        }
      }
      const showWorkspaceDialog = vue.ref(false);
      const newWsName = vue.ref("");
      const newWsPath = vue.ref("");
      const wsError = vue.ref("");
      const wsContextMenuRef = vue.ref(null);
      async function showWsContextMenu(e, ws) {
        await vue.nextTick();
        const result = await wsContextMenuRef.value.show({
          x: e.clientX,
          y: e.clientY,
          title: ws.name,
          items: [
            { label: "添加项目", icon: "plus", action: "add-project" },
            { label: "新建项目", icon: "folder-plus", action: "new-project" },
            { separator: true },
            { label: "重命名工作区", action: "rename" },
            { label: "删除工作区", action: "delete" },
            { separator: true },
            { label: "在终端中打开", icon: "terminal", action: "open-terminal" },
            { label: "复制路径", icon: "copy", action: "copy-path" }
          ]
        });
        if (!result) return;
        switch (result) {
          case "add-project":
            openBrowseDialog("add");
            break;
          case "new-project":
            uiState_js.state.workspaceRoot = ws.path;
            uiState_js.state.workspaceName = ws.name;
            openBrowseDialog("new");
            break;
          case "rename": {
            const name = await window.$prompt("新名称:", ws.name, "重命名工作区");
            if (name && name.trim()) {
              ws.name = name.trim();
              await saveWsList();
            }
            break;
          }
          case "delete":
            (async () => {
              var _a, _b;
              const result2 = await window.$confirmWithCheckbox(
                `确认删除工作区 "${ws.name}"？`,
                "删除工作区",
                "同时删除该工作区的对话历史、快照等文件 (.pair目录)"
              );
              if (!result2 || !result2.confirmed) return;
              try {
                await api.apiPost("/workspace", { action: "delete", root: ws.path, deleteFiles: result2.checked });
              } catch (e2) {
                console.warn("删除工作区后端失败:", e2);
              }
              uiState_js.state.wsList = uiState_js.state.wsList.filter((w) => w.path !== ws.path);
              await saveWsList();
              if (uiState_js.state.workspaceRoot === ws.path) {
                uiState_js.state.workspaceRoot = ((_a = uiState_js.state.wsList[0]) == null ? void 0 : _a.path) || "";
                uiState_js.state.workspaceName = ((_b = uiState_js.state.wsList[0]) == null ? void 0 : _b.name) || "";
                if (uiState_js.state.workspaceRoot) await switchToWorkspace(uiState_js.state.wsList[0]);
              }
            })();
            break;
          case "open-terminal": {
            uiState_js.state.bottomPanelVisible = true;
            uiState_js.state.bottomPanelTab = "terminal";
            window.dispatchEvent(new CustomEvent("terminal-cwd", { detail: { cwd: ws.path } }));
            break;
          }
          case "copy-path":
            navigator.clipboard.writeText(ws.path).catch(() => {
            });
            break;
        }
      }
      async function createWorkspace() {
        const name = newWsName.value.trim();
        if (!name) return;
        wsError.value = "";
        try {
          const res = await api.apiPost("/workspace", { action: "create", name, root: newWsPath.value.trim() || "" });
          if (res.ok || res.root) {
            const newPath = res.root || "";
            const ws = { path: newPath, name, folders: [newPath], notify: false };
            uiState_js.state.wsList.push(ws);
            showWorkspaceDialog.value = false;
            newWsName.value = "";
            newWsPath.value = "";
            await switchToWorkspace(ws);
            await saveWsList();
          }
        } catch (err) {
          wsError.value = err.message;
        }
      }
      async function saveWsList() {
        try {
          const resp = await api.apiGet("/settings");
          const settings = resp.settings || resp;
          settings.recentProjects = (uiState_js.state.wsList || []).slice(0, 20).map((w) => w.path).filter(Boolean);
          await api.apiPut("/settings", settings);
        } catch (e) {
        }
      }
      const browseVisible = vue.ref(false);
      const browseMode = vue.ref("add");
      const browsePath = vue.ref("");
      const browseDrives = vue.ref([]);
      const browseEntries = vue.ref([]);
      const browseSelected = vue.ref("");
      const browseError = vue.ref("");
      const newProjectName = vue.ref("");
      const browseTitle = vue.computed(() => ({
        add: "选择项目目录",
        new: "选择父目录创建新项目",
        "ws-path": "选择工作区保存路径"
      })[browseMode.value] || "浏览目录");
      const browseConfirmDisabled = vue.computed(() => {
        if (browseMode.value === "new") return !newProjectName.value.trim();
        return !browseSelected.value && !browsePath.value;
      });
      function openBrowseDialog(mode) {
        browseMode.value = mode;
        browseVisible.value = true;
        browseError.value = "";
        newProjectName.value = "";
        browsePath.value = "";
        browseSelected.value = "";
        browseEntries.value = [];
        api.apiGet("/fs/drives").then((d) => {
          browseDrives.value = d || [];
        }).catch(() => {
        });
      }
      function closeBrowse() {
        browseVisible.value = false;
        browseError.value = "";
      }
      function browseSelect(entry) {
        if (!entry.isDir) return;
        const full = browsePath.value + "\\" + entry.name;
        browseSelected.value = full;
        browsePath.value = full;
        loadBrowseDir(full);
      }
      async function browseEnter(path) {
        browsePath.value = path;
        browseSelected.value = "";
        loadBrowseDir(path);
      }
      async function browseGoUp() {
        if (!browsePath.value) return;
        const parts = browsePath.value.replace(/\\$/, "").split("\\");
        if (parts.length <= 1) {
          browsePath.value = "";
          browseEntries.value = [];
          browseSelected.value = "";
          return;
        }
        parts.pop();
        browsePath.value = parts.join("\\") + "\\";
        loadBrowseDir(browsePath.value);
      }
      async function loadBrowseDir(path) {
        try {
          browseEntries.value = await api.apiGet("/fs/list", { path });
        } catch (err) {
          browseError.value = err.message;
        }
      }
      async function browseConfirm() {
        browseError.value = "";
        if (browseMode.value === "new") {
          const name = newProjectName.value.trim();
          if (!name) return;
          const dir = browsePath.value;
          if (!dir) {
            browseError.value = "请先选择父目录";
            return;
          }
          try {
            await api.apiPost("/workspace", { action: "new-project", name, parentDir: dir });
            await refreshCurrentWs();
            closeBrowse();
          } catch (err) {
            browseError.value = err.message;
          }
        } else if (browseMode.value === "ws-path") {
          const p = browseSelected.value || browsePath.value;
          if (!p) {
            browseError.value = "请先选择目录";
            return;
          }
          newWsPath.value = p;
          closeBrowse();
        } else {
          const p = browseSelected.value || browsePath.value;
          if (!p) {
            browseError.value = "请先选择目录";
            return;
          }
          try {
            await api.apiPost("/workspace", { action: "add-folder", path: p });
            await refreshCurrentWs();
            closeBrowse();
          } catch (err) {
            browseError.value = err.message;
          }
        }
      }
      async function refreshCurrentWs() {
        try {
          const health = await api.apiGet("/health");
          uiState_js.state.workspaceFolders = health.folders || [];
          const cur = uiState_js.state.wsList.find((w) => w.path === uiState_js.state.workspaceRoot);
          if (cur) cur.folders = [...uiState_js.state.workspaceFolders];
          uiState_js.state.settings.workspaceFolders = [...uiState_js.state.workspaceFolders];
          await saveWsList();
        } catch (e) {
          console.warn("刷新工作区失败:", e);
        }
      }
      let _refreshingTree = false;
      let _savedTreeScrollTop = 0;
      async function refreshAll() {
        if (_refreshingTree) return;
        _refreshingTree = true;
        const container = document.querySelector(".project-section");
        if (container) _savedTreeScrollTop = container.scrollTop;
        try {
          const health = await api.apiGet("/health");
          uiState_js.state.workspaceFolders = health.folders || [];
          uiState_js.state.workspaceRoot = health.workspace || uiState_js.state.workspaceRoot;
          for (const ws of uiState_js.state.wsList) {
            if (ws.path === uiState_js.state.workspaceRoot) {
              ws.folders = [...uiState_js.state.workspaceFolders];
            }
          }
          uiState_js.state.settings.workspaceFolders = [...uiState_js.state.workspaceFolders];
          await saveWsList();
          window.dispatchEvent(new CustomEvent("refresh-tree"));
        } catch (e) {
          console.warn("刷新全部失败:", e);
        } finally {
          _refreshingTree = false;
          if (_savedTreeScrollTop > 0) {
            vue.nextTick(() => {
              const c = document.querySelector(".project-section");
              if (c) c.scrollTop = _savedTreeScrollTop;
            });
          }
        }
      }
      function openFile(path) {
        if (!uiState_js.state.openFiles.includes(path)) uiState_js.state.openFiles.push(path);
        uiState_js.state.activeFile = path;
        loadFileContent(path);
      }
      async function loadFileContent(path) {
        if (uiState_js.state.fileDirty[path]) return;
        delete uiState_js.state.fileContents[path];
        delete uiState_js.state.fileSavedContent[path];
        delete uiState_js.state.fileDirty[path];
        const ext = (path.split(".").pop() || "").toLowerCase();
        const imgExts = ["png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "ico"];
        const knownBinExts = [
          "exe",
          "dll",
          "so",
          "dylib",
          "zip",
          "tar",
          "gz",
          "rar",
          "7z",
          "pdf",
          "ttf",
          "otf",
          "woff",
          "woff2",
          "eot",
          "ico",
          "icns"
        ];
        if (imgExts.includes(ext) || knownBinExts.includes(ext)) {
          uiState_js.state.fileContents[path] = "";
          return;
        }
        try {
          const data = await api.apiGet("/fs/read", { path });
          const normalized = (data.content || "").replace(/\r\n/g, "\n");
          uiState_js.state.fileSavedContent[path] = normalized;
          uiState_js.state.fileContents[path] = normalized;
          uiState_js.state.fileDirty[path] = false;
        } catch (err) {
          uiState_js.state.fileContents[path] = `// 错误: ${err.message}`;
        }
      }
      const tsAddSearch = vue.ref("");
      const tsMsg = vue.ref("");
      const tsMsgErr = vue.ref(false);
      const builtinInfo = vue.ref(null);
      const tsOpen = vue.ref(false);
      try {
        const saved = localStorage.getItem("paircode-ts-open");
        if (saved !== null) tsOpen.value = saved === "1";
      } catch {
      }
      vue.computed(() => {
        var _a, _b, _c;
        let n = 0;
        const joined = new Set(((_a = builtinInfo.value) == null ? void 0 : _a.joined) || []);
        for (const g of ((_b = builtinInfo.value) == null ? void 0 : _b.groups) || []) {
          if (g.source === "builtin" && joined.has(g.name)) n += (g.tools || []).length;
        }
        return n + (((_c = builtinInfo.value) == null ? void 0 : _c.manualTools) || []).length;
      });
      const joinedGroups = vue.computed(() => {
        var _a, _b, _c;
        const joined = new Set(((_a = builtinInfo.value) == null ? void 0 : _a.joined) || []);
        const bg = (((_b = builtinInfo.value) == null ? void 0 : _b.groups) || []).filter((g) => g.source === "builtin" && joined.has(g.name));
        const pg = (((_c = builtinInfo.value) == null ? void 0 : _c.plugins) || []).filter((g) => g.source === "plugin").map((g) => ({ ...g, tools: (g.tools || []).filter((t) => t.enabled) })).filter((g) => (g.tools || []).length > 0);
        return [...bg, ...pg];
      });
      const manualToolNames = vue.computed(() => {
        var _a;
        return ((_a = builtinInfo.value) == null ? void 0 : _a.manualTools) || [];
      });
      const manualToolObjs = vue.computed(() => manualToolNames.value.map((n) => ({ name: n, desc: "手动加入的工具" })));
      const tsTransferOpen = vue.ref(false);
      function openTransfer() {
        tsTransferOpen.value = true;
      }
      function onTransferChanged() {
        loadBuiltin();
      }
      vue.watch(() => uiState_js.state.workspaceRoot, () => {
        loadBuiltin();
      });
      const joinedTools = vue.computed(() => {
        var _a, _b, _c, _d;
        const set = {};
        const joined = new Set(((_a = builtinInfo.value) == null ? void 0 : _a.joined) || []);
        for (const g of ((_b = builtinInfo.value) == null ? void 0 : _b.groups) || []) {
          if (g.source === "builtin" && joined.has(g.name)) for (const t of g.tools) set[t.name] = true;
        }
        for (const g of ((_c = builtinInfo.value) == null ? void 0 : _c.plugins) || []) {
          for (const t of g.tools || []) if (t.enabled) set[t.name] = true;
        }
        for (const tn of ((_d = builtinInfo.value) == null ? void 0 : _d.manualTools) || []) set[tn] = true;
        return set;
      });
      vue.computed(() => {
        var _a;
        return ((_a = builtinInfo.value) == null ? void 0 : _a.groups) || [];
      });
      vue.computed(() => {
        var _a;
        return ((_a = builtinInfo.value) == null ? void 0 : _a.plugins) || [];
      });
      function filterTools(tools) {
        const q = tsAddSearch.value.trim().toLowerCase();
        if (!q) return tools;
        return tools.filter((t) => t.name.toLowerCase().includes(q));
      }
      async function loadBuiltin() {
        try {
          builtinInfo.value = await api.builtinPlugins(void 0, uiState_js.state.workspaceRoot);
        } catch (e) {
          console.warn("[toolset] 内置工具包加载失败", e);
        }
      }
      async function toggleToolsetTool(t, g) {
        try {
          if (g && g.source === "plugin") {
            if (!joinedTools.value[t.name]) {
              const res = await api.toolsetEdit({ name: "default", action: "add_plugin", plugin_name: g.name, tools: t.name, workspaceRoot: uiState_js.state.workspaceRoot });
              tsMsg.value = (res == null ? void 0 : res.message) || "已加入 " + t.name;
            } else {
              const res = await api.toolsetEdit({ name: "default", action: "rm_tool", plugin_name: g.name, tool: t.name, workspaceRoot: uiState_js.state.workspaceRoot });
              tsMsg.value = (res == null ? void 0 : res.message) || "已移出 " + t.name;
            }
          } else {
            const enabled = !joinedTools.value[t.name];
            const res = await api.builtinPlugins({ tool: t.name, enabled }, uiState_js.state.workspaceRoot);
            tsMsg.value = (res == null ? void 0 : res.message) || (enabled ? "已添加" : "已移除") + " " + t.name;
          }
          tsMsgErr.value = false;
          await loadBuiltin();
        } catch (err) {
          tsMsgErr.value = true;
          tsMsg.value = "操作失败: " + (err.message || err);
        }
      }
      function toggleTs() {
        tsOpen.value = !tsOpen.value;
        try {
          localStorage.setItem("paircode-ts-open", tsOpen.value ? "1" : "0");
        } catch {
        }
      }
      vue.onMounted(() => {
        window.addEventListener("refresh-tree", refreshAll);
        window.addEventListener("refresh-workspace", refreshCurrentWs);
        loadBuiltin();
      });
      vue.onUnmounted(() => {
        window.removeEventListener("refresh-tree", refreshAll);
        window.removeEventListener("refresh-workspace", refreshCurrentWs);
      });
      return (_ctx, _cache) => {
        var _a, _b, _c;
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1$3, [
          vue.createCommentVNode(" 顶部工具栏 "),
          vue.createElementVNode("div", _hoisted_2$3, [
            _cache[10] || (_cache[10] = vue.createElementVNode(
              "span",
              { class: "tb-title" },
              "工作区",
              -1
              /* CACHED */
            )),
            _cache[11] || (_cache[11] = vue.createElementVNode(
              "span",
              { class: "tb-spacer" },
              null,
              -1
              /* CACHED */
            )),
            vue.createElementVNode("button", {
              class: "tb-btn",
              onClick: _cache[0] || (_cache[0] = ($event) => showWorkspaceDialog.value = true),
              title: "新建工作区"
            }, [
              vue.createVNode(SvgIcon, {
                name: "plus",
                size: 14
              })
            ]),
            vue.createElementVNode("button", {
              class: "tb-btn",
              onClick: refreshAll,
              title: "刷新"
            }, [
              vue.createVNode(SvgIcon, {
                name: "refresh",
                size: 14
              })
            ])
          ]),
          vue.createCommentVNode(" ── 上方：工作区列表 ── "),
          vue.createElementVNode("div", _hoisted_3$3, [
            (vue.openBlock(true), vue.createElementBlock(
              vue.Fragment,
              null,
              vue.renderList(wsItems.value, (ws) => {
                return vue.openBlock(), vue.createElementBlock("div", {
                  key: ws.path,
                  class: vue.normalizeClass(["ws-item", { "ws-active": ws.path === vue.unref(uiState_js.state).workspaceRoot }]),
                  onClick: ($event) => switchToWorkspace(ws),
                  onContextmenu: vue.withModifiers(($event) => showWsContextMenu($event, ws), ["prevent"])
                }, [
                  vue.createElementVNode("div", _hoisted_5$2, [
                    vue.createVNode(SvgIcon, {
                      name: "folder",
                      size: 14,
                      class: "ws-icon"
                    }),
                    vue.createElementVNode(
                      "span",
                      _hoisted_6$2,
                      vue.toDisplayString(ws.name),
                      1
                      /* TEXT */
                    )
                  ]),
                  vue.createElementVNode("div", _hoisted_7$2, [
                    ws.notify ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_8$2, "●")) : vue.createCommentVNode("v-if", true),
                    ws.path === vue.unref(uiState_js.state).workspaceRoot ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_9$2, "当前")) : vue.createCommentVNode("v-if", true)
                  ])
                ], 42, _hoisted_4$2);
              }),
              128
              /* KEYED_FRAGMENT */
            )),
            wsItems.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_10$2, [
              _cache[12] || (_cache[12] = vue.createElementVNode(
                "span",
                null,
                "暂无工作区",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("button", {
                class: "ws-create-btn",
                onClick: _cache[1] || (_cache[1] = ($event) => showWorkspaceDialog.value = true)
              }, "创建")
            ])) : vue.createCommentVNode("v-if", true)
          ]),
          vue.createCommentVNode(" ── 分隔线 ── "),
          _cache[21] || (_cache[21] = vue.createElementVNode(
            "div",
            { class: "ws-divider" },
            [
              vue.createElementVNode("span", { class: "divider-label" }, "项目")
            ],
            -1
            /* CACHED */
          )),
          vue.createCommentVNode(" ── 下方：当前工作区的项目列表 ── "),
          vue.createElementVNode("div", _hoisted_11$2, [
            !vue.unref(uiState_js.state).workspaceRoot ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_12$2, "请先选择工作区")) : currentFolders.value.length > 0 ? (vue.openBlock(true), vue.createElementBlock(
              vue.Fragment,
              { key: 1 },
              vue.renderList(currentFolders.value, (folder, fi) => {
                return vue.openBlock(), vue.createBlock(FileTreeItem, {
                  key: folder,
                  item: { name: folder.split("\\").pop(), isDir: true, path: folder },
                  parentPath: folderParent(folder),
                  depth: 0,
                  defaultExpanded: true,
                  siblings: currentFolders.value,
                  siblingIndex: fi,
                  onFileClick: openFile
                }, null, 8, ["item", "parentPath", "siblings", "siblingIndex"]);
              }),
              128
              /* KEYED_FRAGMENT */
            )) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_13$2, [..._cache[13] || (_cache[13] = [
              vue.createElementVNode(
                "span",
                null,
                "暂无项目，在工作区上右键添加",
                -1
                /* CACHED */
              )
            ])]))
          ]),
          vue.createCommentVNode(" ── 工具集（卷帘：与文件树同区，工作区 .pair/toolsets/） ── "),
          vue.createElementVNode("div", _hoisted_14$2, [
            vue.createElementVNode(
              "div",
              {
                class: vue.normalizeClass(["ts-header", { open: tsOpen.value }]),
                onClick: toggleTs,
                title: "工具集（工作区内，可折叠）"
              },
              [
                vue.createVNode(SvgIcon, {
                  name: "package",
                  size: 12,
                  class: "ts-header-icon"
                }),
                _cache[14] || (_cache[14] = vue.createElementVNode(
                  "span",
                  { class: "divider-label ts-label" },
                  "工具集",
                  -1
                  /* CACHED */
                )),
                _cache[15] || (_cache[15] = vue.createElementVNode(
                  "span",
                  { class: "ts-spacer" },
                  null,
                  -1
                  /* CACHED */
                )),
                vue.createVNode(SvgIcon, {
                  name: "chevron-right",
                  size: 11,
                  class: vue.normalizeClass(["ts-chevron", { open: tsOpen.value }])
                }, null, 8, ["class"])
              ],
              2
              /* CLASS */
            ),
            tsOpen.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_15$2, [
              vue.createCommentVNode(" 工作区工具集（builtin 已加入内容：可移出） "),
              vue.createElementVNode("div", _hoisted_16$2, [
                vue.createElementVNode("div", { class: "ts-build-head" }, [
                  _cache[16] || (_cache[16] = vue.createElementVNode(
                    "span",
                    { class: "ts-build-title" },
                    "工作区工具集",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode("button", {
                    class: "ts-btn mini",
                    onClick: openTransfer,
                    title: "穿梭框批量管理：未加入 ↔ 已加入"
                  }, "管理")
                ]),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => tsAddSearch.value = $event),
                    placeholder: "搜索工具名…",
                    class: "ts-input"
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelText, tsAddSearch.value]
                ]),
                vue.createElementVNode("div", _hoisted_17$2, [
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(joinedGroups.value, (g) => {
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: g.name,
                        class: "ts-add-group"
                      }, [
                        vue.createElementVNode("div", _hoisted_18$2, [
                          vue.createElementVNode(
                            "span",
                            null,
                            vue.toDisplayString(g.name),
                            1
                            /* TEXT */
                          )
                        ]),
                        (vue.openBlock(true), vue.createElementBlock(
                          vue.Fragment,
                          null,
                          vue.renderList(filterTools(g.tools), (t) => {
                            return vue.openBlock(), vue.createElementBlock("div", {
                              key: t.name,
                              class: "ts-add-tool",
                              title: t.desc
                            }, [
                              vue.createElementVNode(
                                "span",
                                _hoisted_20$2,
                                vue.toDisplayString(t.name),
                                1
                                /* TEXT */
                              ),
                              vue.createElementVNode("button", {
                                class: "ts-btn mini danger",
                                onClick: ($event) => toggleToolsetTool(t, g),
                                title: "移出工作区工具集（该工具对 agent 不可见）"
                              }, "移出", 8, _hoisted_21$2)
                            ], 8, _hoisted_19$2);
                          }),
                          128
                          /* KEYED_FRAGMENT */
                        ))
                      ]);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  )),
                  manualToolNames.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_22$2, [
                    _cache[17] || (_cache[17] = vue.createElementVNode(
                      "div",
                      { class: "ts-add-group-title" },
                      [
                        vue.createElementVNode("span", null, "_manual（手动）")
                      ],
                      -1
                      /* CACHED */
                    )),
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(filterTools(manualToolObjs.value), (t) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: t.name,
                          class: "ts-add-tool",
                          title: t.desc
                        }, [
                          vue.createElementVNode(
                            "span",
                            _hoisted_24$2,
                            vue.toDisplayString(t.name),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode("button", {
                            class: "ts-btn mini danger",
                            onClick: ($event) => toggleToolsetTool(t, _ctx.g),
                            title: "移出工作区工具集（该工具对 agent 不可见）"
                          }, "移出", 8, _hoisted_25$1)
                        ], 8, _hoisted_23$2);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])) : vue.createCommentVNode("v-if", true),
                  !joinedGroups.value.length && !manualToolNames.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_26$1, "未加入任何工具。点「管理」在穿梭框中加入。")) : vue.createCommentVNode("v-if", true)
                ]),
                tsMsg.value ? (vue.openBlock(), vue.createElementBlock(
                  "div",
                  {
                    key: 0,
                    class: vue.normalizeClass(["ts-msg", { err: tsMsgErr.value }])
                  },
                  vue.toDisplayString(tsMsg.value),
                  3
                  /* TEXT, CLASS */
                )) : vue.createCommentVNode("v-if", true)
              ])
            ])) : vue.createCommentVNode("v-if", true)
          ]),
          vue.createCommentVNode(" ===== 新建工作区对话框 ===== "),
          showWorkspaceDialog.value ? (vue.openBlock(), vue.createElementBlock("div", {
            key: 0,
            class: "dialog-overlay",
            onClick: _cache[7] || (_cache[7] = vue.withModifiers(($event) => showWorkspaceDialog.value = false, ["self"]))
          }, [
            vue.createElementVNode("div", _hoisted_27$1, [
              _cache[20] || (_cache[20] = vue.createElementVNode(
                "div",
                { class: "dialog-title" },
                "新建工作区",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("div", _hoisted_28$1, [
                _cache[18] || (_cache[18] = vue.createElementVNode(
                  "label",
                  null,
                  "工作区名称",
                  -1
                  /* CACHED */
                )),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => newWsName.value = $event),
                    class: "dlg-input",
                    placeholder: "例如: my-project",
                    onKeyup: vue.withKeys(createWorkspace, ["enter"])
                  },
                  null,
                  544
                  /* NEED_HYDRATION, NEED_PATCH */
                ), [
                  [vue.vModelText, newWsName.value]
                ]),
                _cache[19] || (_cache[19] = vue.createElementVNode(
                  "label",
                  null,
                  "保存路径（可选）",
                  -1
                  /* CACHED */
                )),
                vue.createElementVNode("div", _hoisted_29$1, [
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[4] || (_cache[4] = ($event) => newWsPath.value = $event),
                      class: "dlg-input flex-1",
                      placeholder: "留空则用默认目录"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, newWsPath.value]
                  ]),
                  vue.createElementVNode("button", {
                    class: "dlg-btn-sm",
                    onClick: _cache[5] || (_cache[5] = ($event) => openBrowseDialog("ws-path"))
                  }, "浏览")
                ])
              ]),
              vue.createElementVNode("div", _hoisted_30$1, [
                wsError.value ? (vue.openBlock(), vue.createElementBlock(
                  "span",
                  _hoisted_31$1,
                  vue.toDisplayString(wsError.value),
                  1
                  /* TEXT */
                )) : vue.createCommentVNode("v-if", true),
                vue.createElementVNode("button", {
                  class: "dlg-btn",
                  onClick: _cache[6] || (_cache[6] = ($event) => showWorkspaceDialog.value = false)
                }, "取消"),
                vue.createElementVNode("button", {
                  class: "dlg-btn primary",
                  onClick: createWorkspace,
                  disabled: !newWsName.value.trim()
                }, "创建", 8, _hoisted_32$1)
              ])
            ])
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" ===== 目录浏览对话框 ===== "),
          browseVisible.value ? (vue.openBlock(), vue.createElementBlock("div", {
            key: 1,
            class: "dialog-overlay",
            onClick: vue.withModifiers(closeBrowse, ["self"])
          }, [
            vue.createElementVNode("div", _hoisted_33$1, [
              vue.createElementVNode(
                "div",
                _hoisted_34$1,
                vue.toDisplayString(browseTitle.value),
                1
                /* TEXT */
              ),
              vue.createElementVNode("div", _hoisted_35$1, [
                vue.createElementVNode("div", _hoisted_36$1, [
                  vue.createElementVNode("button", {
                    class: "bc-btn",
                    onClick: browseGoUp,
                    disabled: browsePath.value === ""
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "chevron-right",
                      size: 14,
                      style: { "transform": "rotate(180deg)" }
                    })
                  ], 8, _hoisted_37$1),
                  vue.createElementVNode(
                    "span",
                    _hoisted_38$1,
                    vue.toDisplayString(browsePath.value || "选择驱动器..."),
                    1
                    /* TEXT */
                  )
                ]),
                browsePath.value === "" ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_39$1, [
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(browseDrives.value, (drive) => {
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: drive,
                        class: "dir-item dir-drive",
                        onDblclick: ($event) => browseEnter(drive)
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: "drive",
                          size: 14
                        }),
                        vue.createElementVNode(
                          "span",
                          _hoisted_41$1,
                          vue.toDisplayString(drive),
                          1
                          /* TEXT */
                        )
                      ], 40, _hoisted_40$1);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  ))
                ])) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_42$1, [
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(browseEntries.value, (entry) => {
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: entry.name,
                        class: vue.normalizeClass(["dir-item", { "dir-selected": browseSelected.value === browsePath.value + "\\" + entry.name }]),
                        onClick: ($event) => browseSelect(entry)
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: entry.isDir ? "folder" : "file",
                          size: 14
                        }, null, 8, ["name"]),
                        vue.createElementVNode(
                          "span",
                          _hoisted_44$1,
                          vue.toDisplayString(entry.name),
                          1
                          /* TEXT */
                        )
                      ], 10, _hoisted_43$1);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  )),
                  browseEntries.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_45$1, "空目录")) : vue.createCommentVNode("v-if", true)
                ]))
              ]),
              vue.createElementVNode("div", _hoisted_46$1, [
                browseError.value ? (vue.openBlock(), vue.createElementBlock(
                  "span",
                  _hoisted_47$1,
                  vue.toDisplayString(browseError.value),
                  1
                  /* TEXT */
                )) : vue.createCommentVNode("v-if", true),
                browseMode.value === "new" ? vue.withDirectives((vue.openBlock(), vue.createElementBlock(
                  "input",
                  {
                    key: 1,
                    "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => newProjectName.value = $event),
                    class: "dlg-input",
                    style: { "flex": "1", "margin-right": "8px" },
                    placeholder: "项目名称",
                    onKeyup: vue.withKeys(browseConfirm, ["enter"])
                  },
                  null,
                  544
                  /* NEED_HYDRATION, NEED_PATCH */
                )), [
                  [vue.vModelText, newProjectName.value]
                ]) : vue.createCommentVNode("v-if", true),
                vue.createElementVNode("button", {
                  class: "dlg-btn",
                  onClick: closeBrowse
                }, "取消"),
                vue.createElementVNode("button", {
                  class: "dlg-btn primary",
                  onClick: browseConfirm,
                  disabled: browseConfirmDisabled.value
                }, "确认", 8, _hoisted_48$1)
              ])
            ])
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 右键菜单 "),
          vue.createVNode(
            ContextMenu,
            {
              ref_key: "wsContextMenuRef",
              ref: wsContextMenuRef
            },
            null,
            512
            /* NEED_PATCH */
          ),
          vue.createCommentVNode(" 工作区工具集穿梭框（未加入 ↔ 已加入 批量管理） "),
          tsTransferOpen.value ? (vue.openBlock(), vue.createBlock(ToolsetTransfer, {
            key: 2,
            groups: ((_a = builtinInfo.value) == null ? void 0 : _a.plugins) || [],
            joined: ((_b = builtinInfo.value) == null ? void 0 : _b.joined) || [],
            "manual-tools": ((_c = builtinInfo.value) == null ? void 0 : _c.manualTools) || [],
            "workspace-root": vue.unref(uiState_js.state).workspaceRoot,
            onClose: _cache[9] || (_cache[9] = ($event) => tsTransferOpen.value = false),
            onChanged: onTransferChanged
          }, null, 8, ["groups", "joined", "manual-tools", "workspace-root"])) : vue.createCommentVNode("v-if", true)
        ]);
      };
    }
  };
  const FileExplorer = /* @__PURE__ */ _export_sfc(_sfc_main$3, [["__scopeId", "data-v-8ed578ee"]]);
  const _hoisted_1$2 = { class: "search-panel" };
  const _hoisted_2$2 = { class: "sp-mode-bar" };
  const _hoisted_3$2 = { class: "sp-field" };
  const _hoisted_4$1 = { class: "sp-input-wrap" };
  const _hoisted_5$1 = {
    key: 0,
    class: "sp-field"
  };
  const _hoisted_6$1 = { class: "sp-input-wrap" };
  const _hoisted_7$1 = ["disabled"];
  const _hoisted_8$1 = { class: "sp-path-row" };
  const _hoisted_9$1 = { class: "sp-options" };
  const _hoisted_10$1 = {
    class: "sp-opt",
    title: "区分大小写"
  };
  const _hoisted_11$1 = {
    class: "sp-opt",
    title: "全词匹配"
  };
  const _hoisted_12$1 = {
    class: "sp-opt",
    title: "使用正则表达式"
  };
  const _hoisted_13$1 = {
    key: 1,
    class: "sp-results"
  };
  const _hoisted_14$1 = { class: "sp-result-header" };
  const _hoisted_15$1 = { class: "sp-result-count" };
  const _hoisted_16$1 = ["onClick"];
  const _hoisted_17$1 = { class: "sp-file-path" };
  const _hoisted_18$1 = { class: "sp-file-count" };
  const _hoisted_19$1 = {
    key: 0,
    class: "sp-file-items"
  };
  const _hoisted_20$1 = ["onClick", "title"];
  const _hoisted_21$1 = { class: "sp-result-line" };
  const _hoisted_22$1 = ["innerHTML"];
  const _hoisted_23$1 = {
    key: 2,
    class: "sp-no-results"
  };
  const _hoisted_24$1 = {
    key: 3,
    class: "sp-hint"
  };
  const _sfc_main$2 = {
    __name: "SearchPanel",
    setup(__props) {
      const query = vue.ref("");
      const replaceText = vue.ref("");
      const searchPath = vue.ref("");
      const searched = vue.ref(false);
      const caseSensitive = vue.ref(false);
      const wholeWord = vue.ref(false);
      const useRegex = vue.ref(false);
      const mode = vue.ref("search");
      const replaceStatus = vue.ref(null);
      const groupExpanded = vue.ref({});
      const groupedResults = vue.computed(() => {
        const map = {};
        for (const r of uiState_js.state.searchResults) {
          if (!map[r.file]) {
            map[r.file] = { file: r.file, items: [], expanded: groupExpanded.value[r.file] !== false };
          }
          map[r.file].items.push(r);
        }
        const list = Object.values(map);
        for (const g of list) {
          if (groupExpanded.value[g.file] === void 0) groupExpanded.value[g.file] = true;
        }
        return list;
      });
      function toggleGroup(file) {
        groupExpanded.value[file] = !groupExpanded.value[file];
      }
      function highlightMatch(text, q) {
        if (!text || !q) return text;
        try {
          const escaped = q.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
          const re = new RegExp("(" + escaped + ")", caseSensitive.value ? "g" : "gi");
          return text.replace(re, '<mark class="sp-match">$1</mark>');
        } catch {
          return text;
        }
      }
      const doSearch = async () => {
        if (!query.value.trim()) return;
        searched.value = true;
        uiState_js.state.searchResults = [];
        replaceStatus.value = null;
        try {
          const params = { q: query.value, path: searchPath.value || uiState_js.state.workspaceRoot };
          if (caseSensitive.value) params.case_sensitive = "1";
          if (wholeWord.value) params.whole_word = "1";
          if (useRegex.value) params.regex = "1";
          const results = await api.apiGet("/fs/search", params);
          uiState_js.state.searchResults = results || [];
        } catch (err) {
          uiState_js.state.searchResults = [];
          console.error("搜索失败:", err);
        }
      };
      const openResult = (result) => {
        const path = result.file;
        if (!uiState_js.state.openFiles.includes(path)) uiState_js.state.openFiles.push(path);
        uiState_js.state.activeFile = path;
        if (!uiState_js.state.fileContents[path]) {
          api.apiGet("/fs/read", { path }).then((d) => {
            uiState_js.state.fileContents[path] = d.content || "";
            uiState_js.state.fileSavedContent[path] = uiState_js.state.fileContents[path];
            uiState_js.state.fileDirty[path] = false;
          }).catch((e) => console.warn("[搜索] 读取文件失败:", path, e));
        }
      };
      const replaceAll = async () => {
        if (!query.value.trim() || uiState_js.state.searchResults.length === 0) return;
        replaceStatus.value = { type: "info", message: "正在替换..." };
        const files = {};
        for (const r of uiState_js.state.searchResults) {
          if (!files[r.file]) files[r.file] = /* @__PURE__ */ new Set();
          files[r.file].add(r.line);
        }
        let successCount = 0;
        let failCount = 0;
        const fileList = Object.keys(files);
        for (const filePath of fileList) {
          try {
            const data = await api.apiGet("/fs/read", { path: filePath });
            let content = data.content || "";
            let pattern = query.value;
            if (!useRegex.value) pattern = pattern.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
            if (wholeWord.value) pattern = "\\b" + pattern + "\\b";
            const flags = caseSensitive.value ? "g" : "gi";
            const re = new RegExp(pattern, flags);
            const newContent = content.replace(re, replaceText.value);
            if (newContent === content) continue;
            await api.apiPost("/fs/write", { path: filePath, content: newContent });
            const normalized = newContent.replace(/\r\n/g, "\n");
            if (uiState_js.state.openFiles.includes(filePath)) {
              uiState_js.state.fileContents[filePath] = normalized;
              uiState_js.state.fileSavedContent[filePath] = normalized;
              uiState_js.state.fileDirty[filePath] = false;
            }
            successCount++;
          } catch (err) {
            console.warn("[替换] 文件替换失败:", filePath, err);
            failCount++;
          }
        }
        await doSearch();
        replaceStatus.value = {
          type: failCount > 0 ? "warn" : "success",
          message: failCount > 0 ? `完成：${successCount} 个文件已替换，${failCount} 个失败` : `✅ 全部完成：${successCount} 个文件已替换`
        };
        window.dispatchEvent(new CustomEvent("refresh-tree"));
        setTimeout(() => {
          replaceStatus.value = null;
        }, 5e3);
      };
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1$2, [
          vue.createCommentVNode(" 模式切换：搜索 / 替换 "),
          vue.createElementVNode("div", _hoisted_2$2, [
            vue.createElementVNode(
              "button",
              {
                class: vue.normalizeClass(["sp-mode-btn", { active: mode.value === "search" }]),
                onClick: _cache[0] || (_cache[0] = ($event) => mode.value = "search")
              },
              "查找",
              2
              /* CLASS */
            ),
            vue.createElementVNode(
              "button",
              {
                class: vue.normalizeClass(["sp-mode-btn", { active: mode.value === "replace" }]),
                onClick: _cache[1] || (_cache[1] = ($event) => mode.value = "replace")
              },
              "替换",
              2
              /* CLASS */
            )
          ]),
          vue.createCommentVNode(" 搜索输入 "),
          vue.createElementVNode("div", _hoisted_3$2, [
            vue.createElementVNode("div", _hoisted_4$1, [
              vue.createVNode(SvgIcon, {
                name: "search",
                size: 13,
                class: "sp-input-icon"
              }),
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  type: "text",
                  "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => query.value = $event),
                  placeholder: "搜索...",
                  onKeydown: vue.withKeys(doSearch, ["enter"]),
                  class: "sp-input"
                },
                null,
                544
                /* NEED_HYDRATION, NEED_PATCH */
              ), [
                [vue.vModelText, query.value]
              ])
            ]),
            vue.createElementVNode("button", {
              onClick: doSearch,
              class: "sp-go-btn"
            }, "查找")
          ]),
          vue.createCommentVNode(" 替换输入（替换模式下显示）"),
          mode.value === "replace" ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_5$1, [
            vue.createElementVNode("div", _hoisted_6$1, [
              vue.createVNode(SvgIcon, {
                name: "edit",
                size: 13,
                class: "sp-input-icon"
              }),
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  type: "text",
                  "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => replaceText.value = $event),
                  placeholder: "替换为...",
                  onKeydown: vue.withKeys(replaceAll, ["enter"]),
                  class: "sp-input"
                },
                null,
                544
                /* NEED_HYDRATION, NEED_PATCH */
              ), [
                [vue.vModelText, replaceText.value]
              ])
            ]),
            vue.createElementVNode("button", {
              onClick: replaceAll,
              class: "sp-replace-btn",
              disabled: !vue.unref(uiState_js.state).searchResults.length || !query.value.trim()
            }, "全部替换", 8, _hoisted_7$1)
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 搜索路径 "),
          vue.createElementVNode("div", _hoisted_8$1, [
            vue.withDirectives(vue.createElementVNode(
              "input",
              {
                type: "text",
                "onUpdate:modelValue": _cache[4] || (_cache[4] = ($event) => searchPath.value = $event),
                placeholder: "搜索路径（默认工作区）",
                class: "sp-path-input"
              },
              null,
              512
              /* NEED_PATCH */
            ), [
              [vue.vModelText, searchPath.value]
            ]),
            searchPath.value ? (vue.openBlock(), vue.createElementBlock("button", {
              key: 0,
              class: "sp-clear-btn",
              onClick: _cache[5] || (_cache[5] = ($event) => searchPath.value = ""),
              title: "清除路径"
            }, "×")) : vue.createCommentVNode("v-if", true)
          ]),
          vue.createCommentVNode(" 选项行 "),
          vue.createElementVNode("div", _hoisted_9$1, [
            vue.createElementVNode("label", _hoisted_10$1, [
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  type: "checkbox",
                  "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => caseSensitive.value = $event)
                },
                null,
                512
                /* NEED_PATCH */
              ), [
                [vue.vModelCheckbox, caseSensitive.value]
              ]),
              _cache[9] || (_cache[9] = vue.createElementVNode(
                "span",
                null,
                "Aa",
                -1
                /* CACHED */
              ))
            ]),
            vue.createElementVNode("label", _hoisted_11$1, [
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  type: "checkbox",
                  "onUpdate:modelValue": _cache[7] || (_cache[7] = ($event) => wholeWord.value = $event)
                },
                null,
                512
                /* NEED_PATCH */
              ), [
                [vue.vModelCheckbox, wholeWord.value]
              ]),
              _cache[10] || (_cache[10] = vue.createElementVNode(
                "span",
                null,
                "全词",
                -1
                /* CACHED */
              ))
            ]),
            vue.createElementVNode("label", _hoisted_12$1, [
              vue.withDirectives(vue.createElementVNode(
                "input",
                {
                  type: "checkbox",
                  "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => useRegex.value = $event)
                },
                null,
                512
                /* NEED_PATCH */
              ), [
                [vue.vModelCheckbox, useRegex.value]
              ]),
              _cache[11] || (_cache[11] = vue.createElementVNode(
                "span",
                null,
                "正则",
                -1
                /* CACHED */
              ))
            ])
          ]),
          vue.createCommentVNode(" 结果区域 "),
          vue.unref(uiState_js.state).searchResults.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_13$1, [
            vue.createElementVNode("div", _hoisted_14$1, [
              vue.createElementVNode(
                "span",
                _hoisted_15$1,
                vue.toDisplayString(vue.unref(uiState_js.state).searchResults.length) + " 个文件匹配",
                1
                /* TEXT */
              ),
              mode.value === "replace" ? (vue.openBlock(), vue.createElementBlock("button", {
                key: 0,
                class: "sp-replace-all-sm",
                onClick: replaceAll
              }, "全部替换")) : vue.createCommentVNode("v-if", true)
            ]),
            vue.createCommentVNode(" 按文件分组 "),
            (vue.openBlock(true), vue.createElementBlock(
              vue.Fragment,
              null,
              vue.renderList(groupedResults.value, (group, gi) => {
                return vue.openBlock(), vue.createElementBlock("div", {
                  key: gi,
                  class: "sp-file-group"
                }, [
                  vue.createElementVNode("div", {
                    class: "sp-file-title",
                    onClick: ($event) => toggleGroup(gi)
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: group.expanded ? "chevron-down" : "chevron-right",
                      size: 10
                    }, null, 8, ["name"]),
                    vue.createElementVNode(
                      "span",
                      _hoisted_17$1,
                      vue.toDisplayString(group.file),
                      1
                      /* TEXT */
                    ),
                    vue.createElementVNode(
                      "span",
                      _hoisted_18$1,
                      vue.toDisplayString(group.items.length) + " 处",
                      1
                      /* TEXT */
                    )
                  ], 8, _hoisted_16$1),
                  group.expanded ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_19$1, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(group.items, (r, ri) => {
                        return vue.openBlock(), vue.createElementBlock("div", {
                          key: ri,
                          class: "sp-result-row",
                          onClick: ($event) => openResult(r),
                          title: r.text
                        }, [
                          vue.createElementVNode(
                            "span",
                            _hoisted_21$1,
                            vue.toDisplayString(r.line),
                            1
                            /* TEXT */
                          ),
                          vue.createElementVNode("span", {
                            class: "sp-result-text",
                            innerHTML: highlightMatch(r.text, query.value)
                          }, null, 8, _hoisted_22$1)
                        ], 8, _hoisted_20$1);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])) : vue.createCommentVNode("v-if", true)
                ]);
              }),
              128
              /* KEYED_FRAGMENT */
            ))
          ])) : searched.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_23$1, [
            vue.createVNode(SvgIcon, {
              name: "search-off",
              size: 20,
              class: "sp-no-icon"
            }),
            _cache[12] || (_cache[12] = vue.createElementVNode(
              "span",
              null,
              "未找到匹配内容",
              -1
              /* CACHED */
            ))
          ])) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_24$1, [..._cache[13] || (_cache[13] = [
            vue.createElementVNode(
              "span",
              null,
              "输入关键词后按 Enter 搜索",
              -1
              /* CACHED */
            )
          ])])),
          vue.createCommentVNode(" 替换状态提示 "),
          replaceStatus.value ? (vue.openBlock(), vue.createElementBlock(
            "div",
            {
              key: 4,
              class: vue.normalizeClass(["sp-status", replaceStatus.value.type])
            },
            vue.toDisplayString(replaceStatus.value.message),
            3
            /* TEXT, CLASS */
          )) : vue.createCommentVNode("v-if", true)
        ]);
      };
    }
  };
  const SearchPanel = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-2304b715"]]);
  const _hoisted_1$1 = { class: "plugin-panel" };
  const _hoisted_2$1 = { class: "pp-header" };
  const _hoisted_3$1 = { class: "pp-title" };
  const _hoisted_4 = { class: "pp-actions" };
  const _hoisted_5 = {
    key: 0,
    class: "pp-toolset"
  };
  const _hoisted_6 = { class: "pp-ts-head" };
  const _hoisted_7 = { label: "工作区工具集（本项目）" };
  const _hoisted_8 = ["value"];
  const _hoisted_9 = ["value"];
  const _hoisted_10 = {
    key: 0,
    class: "pp-ts-body"
  };
  const _hoisted_11 = { class: "pp-ts-title" };
  const _hoisted_12 = { class: "pp-ts-scope" };
  const _hoisted_13 = { class: "pp-ts-prow" };
  const _hoisted_14 = { class: "pp-ts-pname" };
  const _hoisted_15 = ["onClick"];
  const _hoisted_16 = {
    key: 1,
    class: "pp-ts-muted"
  };
  const _hoisted_17 = {
    key: 0,
    class: "pp-ts-purpose"
  };
  const _hoisted_18 = {
    key: 1,
    class: "pp-ts-tools"
  };
  const _hoisted_19 = ["title"];
  const _hoisted_20 = ["checked", "onChange"];
  const _hoisted_21 = {
    key: 2,
    class: "pp-ts-muted"
  };
  const _hoisted_22 = { class: "pp-ts-add" };
  const _hoisted_23 = ["value"];
  const _hoisted_24 = ["disabled"];
  const _hoisted_25 = {
    key: 1,
    class: "pp-ts-empty"
  };
  const _hoisted_26 = {
    key: 1,
    class: "pp-new"
  };
  const _hoisted_27 = { class: "pp-new-foot" };
  const _hoisted_28 = { class: "pp-check" };
  const _hoisted_29 = ["disabled"];
  const _hoisted_30 = {
    key: 2,
    class: "pp-client"
  };
  const _hoisted_31 = { class: "pp-client-tabs" };
  const _hoisted_32 = ["onClick"];
  const _hoisted_33 = { class: "pp-client-tab-title" };
  const _hoisted_34 = {
    key: 3,
    class: "pp-slots"
  };
  const _hoisted_35 = ["title"];
  const _hoisted_36 = { class: "pp-slots-title" };
  const _hoisted_37 = { class: "pp-slot-info" };
  const _hoisted_38 = { class: "pp-slot-title-row" };
  const _hoisted_39 = { class: "pp-slot-id" };
  const _hoisted_40 = ["value", "onChange", "title"];
  const _hoisted_41 = { value: "" };
  const _hoisted_42 = ["value"];
  const _hoisted_43 = {
    key: 1,
    class: "pp-slot-list"
  };
  const _hoisted_44 = ["checked", "onChange"];
  const _hoisted_45 = {
    key: 0,
    class: "pp-slot-empty"
  };
  const _hoisted_46 = { class: "pp-list" };
  const _hoisted_47 = {
    key: 0,
    class: "pp-builtin"
  };
  const _hoisted_48 = ["title"];
  const _hoisted_49 = { class: "pp-builtin-title" };
  const _hoisted_50 = { class: "pp-builtin-grow" };
  const _hoisted_51 = { class: "pp-builtin-gname" };
  const _hoisted_52 = {
    key: 0,
    class: "pp-builtin-gdesc"
  };
  const _hoisted_53 = { class: "pp-builtin-gcount" };
  const _hoisted_54 = ["onClick"];
  const _hoisted_55 = ["onClick"];
  const _hoisted_56 = { class: "pp-builtin-tools" };
  const _hoisted_57 = ["title"];
  const _hoisted_58 = { class: "pp-d-tname" };
  const _hoisted_59 = ["title"];
  const _hoisted_60 = ["checked", "onChange"];
  const _hoisted_61 = {
    key: 1,
    class: "pp-loading"
  };
  const _hoisted_62 = {
    key: 2,
    class: "pp-empty"
  };
  const _hoisted_63 = {
    key: 3,
    class: "pp-empty"
  };
  const _hoisted_64 = ["onClick"];
  const _hoisted_65 = ["title"];
  const _hoisted_66 = {
    key: 0,
    class: "pp-badge",
    title: "全局插件：跨工作区生效（UI 类），不属于任何工具集"
  };
  const _hoisted_67 = {
    key: 1,
    class: "pp-badge",
    title: "含 client 半（浏览器 UI，运行中自动装载）"
  };
  const _hoisted_68 = ["title"];
  const _hoisted_69 = ["title"];
  const _hoisted_70 = ["checked", "onChange"];
  const _hoisted_71 = {
    key: 0,
    class: "pp-detail"
  };
  const _hoisted_72 = {
    key: 0,
    class: "pp-d-purpose"
  };
  const _hoisted_73 = {
    key: 1,
    class: "pp-d-line"
  };
  const _hoisted_74 = { key: 0 };
  const _hoisted_75 = {
    key: 2,
    class: "pp-d-line"
  };
  const _hoisted_76 = {
    key: 3,
    class: "pp-d-line"
  };
  const _hoisted_77 = {
    key: 4,
    class: "pp-d-tools"
  };
  const _hoisted_78 = { class: "pp-d-tools-title" };
  const _hoisted_79 = ["title"];
  const _hoisted_80 = ["title"];
  const _hoisted_81 = ["checked", "onChange"];
  const _hoisted_82 = {
    key: 5,
    class: "pp-d-code"
  };
  const _hoisted_83 = { class: "pp-d-code-head" };
  const _hoisted_84 = ["onClick"];
  const _hoisted_85 = { class: "pp-d-actions" };
  const _hoisted_86 = ["onClick"];
  const _hoisted_87 = ["onClick"];
  const _hoisted_88 = ["onClick"];
  const _sfc_main$1 = {
    __name: "PluginPanel",
    setup(__props) {
      const plugins = vue.ref([]);
      const loading = vue.ref(false);
      const refreshing = vue.ref(false);
      const loadError = vue.ref(false);
      const expanded = vue.reactive({});
      const showNew = vue.ref(false);
      const defining = vue.ref(false);
      const slotsOpen = vue.ref(false);
      const builtinOpen = vue.ref(false);
      const newMsg = vue.ref("");
      const newMsgErr = vue.ref(false);
      const activePanelId = vue.ref("");
      const clientPanelEl = vue.ref(null);
      const showToolset = vue.ref(false);
      const toolsetMetas = vue.ref([]);
      const tsName = vue.ref("");
      const tsDetail = vue.ref(null);
      const addPluginName = vue.ref("");
      const newForm = vue.reactive({ purpose: "", code: "", client: "", language: "", run: true });
      async function loadToolsets() {
        try {
          const list = await api.getToolsets() || [];
          toolsetMetas.value = list.filter((t) => t.scope !== "global");
        } catch (e) {
          toolsetMetas.value = [];
        }
        if (tsName.value && !toolsetMetas.value.some((t) => t.name === tsName.value)) tsName.value = "";
        if (tsName.value) await loadToolsetDetail();
        else tsDetail.value = null;
      }
      async function loadToolsetDetail() {
        if (!tsName.value) {
          tsDetail.value = null;
          return;
        }
        try {
          tsDetail.value = await api.getToolsets(tsName.value);
        } catch (e) {
          window.$toast && window.$toast("加载工具集失败: " + (e.message || e), "error");
        }
      }
      function pluginToolsOf(pname) {
        const p = plugins.value.find((x) => x.name === pname);
        return p && p.tools || [];
      }
      function isToolDisabled(pl, t) {
        return (pl.disabledTools || []).includes(t);
      }
      async function edit(data) {
        try {
          const res = await api.toolsetEdit({ name: tsName.value, ...data });
          window.$toast && window.$toast(res && res.message || "操作成功", "info");
          await loadToolsetDetail();
          await refresh();
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      function toggleTool(pl, t) {
        edit({ action: isToolDisabled(pl, t) ? "enable_tool" : "rm_tool", plugin_name: pl.name, tool: t });
      }
      async function doAddPlugin() {
        if (!addPluginName.value) return;
        await edit({ action: "add_plugin", plugin_name: addPluginName.value });
        addPluginName.value = "";
      }
      const addablePlugins = vue.computed(() => {
        const inTs = new Set((tsDetail.value && tsDetail.value.plugins || []).map((p) => p.name));
        return plugins.value.filter((p) => !inTs.has(p.name));
      });
      vue.watch(showToolset, (v) => {
        if (v) loadToolsets();
      });
      function fetchPluginsJSON() {
        return new Promise((resolve, reject) => {
          const x = new XMLHttpRequest();
          x.open("GET", "/api/plugins", true);
          x.timeout = 8e3;
          x.onload = () => {
            if (x.status >= 200 && x.status < 300) {
              try {
                resolve(JSON.parse(x.responseText));
              } catch (e) {
                reject(e);
              }
            } else reject(new Error("HTTP " + x.status));
          };
          x.onerror = () => reject(new Error("network error"));
          x.ontimeout = () => reject(new Error("timeout"));
          x.send();
        });
      }
      async function refresh() {
        refreshing.value = true;
        loading.value = true;
        loadError.value = false;
        try {
          let list = [];
          for (let attempt = 0; attempt < 2; attempt++) {
            try {
              const data = await fetchPluginsJSON();
              if (Array.isArray(data)) {
                list = data;
                break;
              }
            } catch (e) {
              list = [];
            }
          }
          plugins.value = list;
          if (!list.length) loadError.value = true;
          const detailTargets = plugins.value.filter((p) => p.hasClient && !p.clientCode);
          await Promise.allSettled(detailTargets.map(async (p) => {
            try {
              const d = await api.getPluginDetail(p.name);
              if (d && d.clientCode) p.clientCode = d.clientCode;
            } catch (e) {
            }
          }));
          await pluginRuntime_js.syncClientHalves(plugins.value);
        } catch (e) {
          console.warn("[plugin] 加载失败", e);
          loadError.value = true;
        } finally {
          loading.value = false;
          refreshing.value = false;
        }
      }
      async function toggleDetail(p) {
        expanded[p.name] = !expanded[p.name];
        if (expanded[p.name] && p.hasClient && !p.clientCode) {
          try {
            const d = await api.getPluginDetail(p.name);
            if (d) Object.assign(p, d);
          } catch (e) {
          }
        }
      }
      const wsToolsetMap = vue.ref(null);
      async function loadWsToolsetMap() {
        try {
          const info = await api.builtinPlugins(null, uiState_js.state.workspaceRoot);
          const map = {};
          for (const g of info && info.plugins || []) {
            map[g.name] = { joined: !!g.joined, tools: {} };
            for (const t of g.tools || []) map[g.name].tools[t.name] = !!t.enabled;
          }
          wsToolsetMap.value = map;
        } catch (e) {
          wsToolsetMap.value = null;
        }
      }
      const builtinGroups = vue.ref([]);
      async function loadBuiltinGroups() {
        try {
          const info = await api.builtinPlugins(null, uiState_js.state.workspaceRoot);
          const groups = (info && info.groups || []).filter((g) => g.source !== "plugin" && (g.tools || []).length).map((g) => ({
            ...g,
            anyOn: (g.tools || []).some((x) => x.enabled),
            allOn: (g.tools || []).length > 0 && (g.tools || []).every((x) => x.enabled)
          }));
          builtinGroups.value = groups;
        } catch (e) {
          builtinGroups.value = [];
        }
      }
      async function toggleBuiltinTool(t, checked) {
        try {
          const res = await api.builtinPlugins({ tool: t.name, enabled: checked }, uiState_js.state.workspaceRoot);
          window.$toast && window.$toast((res == null ? void 0 : res.message) || (checked ? "已加入工作区工具集" : "已移出工作区工具集") + " " + t.name, "info");
          await loadBuiltinGroups();
          await loadWsToolsetMap();
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      async function enableBuiltinGroup(g) {
        try {
          const res = await api.builtinPlugins({ group: g.name, enabled: true }, uiState_js.state.workspaceRoot);
          window.$toast && window.$toast((res == null ? void 0 : res.message) || "已整组加入工作区工具集", "info");
          await loadBuiltinGroups();
          await loadWsToolsetMap();
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      async function disableBuiltinGroup(g) {
        try {
          const res = await api.builtinPlugins({ group: g.name, enabled: false }, uiState_js.state.workspaceRoot);
          window.$toast && window.$toast((res == null ? void 0 : res.message) || "已整组移出工作区工具集", "info");
          await loadBuiltinGroups();
          await loadWsToolsetMap();
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      function pluginToolOn(p, t) {
        var _a;
        return ((_a = p.toolCordisVisible) == null ? void 0 : _a[t]) !== false;
      }
      function uiSlotsOf(pname) {
        return pluginRuntime_js.clientSlots.filter((s) => s.pluginName === pname);
      }
      function uiPluginActive(pname) {
        const slots = uiSlotsOf(pname);
        if (!slots.length) return false;
        return slots.some((s) => {
          if (s.kind === "list") return pluginRuntime_js.isOverlayActive(s.slotId, s.pluginName);
          return pluginRuntime_js.isPluginUIEnabled(s.pluginName);
        });
      }
      function toggleUiPlugin(p, on) {
        const slots = uiSlotsOf(p.name);
        for (const s of slots) {
          if (s.kind === "list") {
            pluginRuntime_js.setOverlayActive(s.slotId, s.pluginName, on);
          } else {
            pluginRuntime_js.setPluginUIEnabled(s.pluginName, on);
            if (on) pluginRuntime_js.setSlotOwner(s.slotId, s.pluginName);
          }
        }
        const recover = p.name === "ui-sidebar" && !on ? "（已停用；恢复入口：右下角壳级按钮）" : "";
        window.$toast && window.$toast(on ? "已启用 " + p.name + " 的 UI（" + slots.map((s) => s.slotId).join(", ") + "）" : "已停用 " + p.name + " 的 UI（区域恢复空态）" + recover, on ? "info" : "warn");
        pluginRuntime_js.emitSlotChanged();
      }
      async function doAction(p, action) {
        try {
          await api.pluginAction(p.name, action);
          if (action === "undefine") {
            pluginRuntime_js.unloadClientHalf(p.name);
            delete expanded[p.name];
            plugins.value = plugins.value.filter((x) => x.name !== p.name);
          } else {
            await refresh();
          }
          window.$toast && window.$toast(`${action === "start" ? "已启动" : action === "stop" ? "已停止" : "已删除"} ${p.name}`, "info");
        } catch (e) {
          window.$toast && window.$toast(e.message || "操作失败", "error");
        }
      }
      async function doDefine() {
        defining.value = true;
        newMsg.value = "";
        newMsgErr.value = false;
        try {
          const res = await api.definePlugin({
            purpose: newForm.purpose,
            code: newForm.code,
            client: newForm.client || void 0,
            language: newForm.language || void 0,
            run: newForm.run
          });
          newMsg.value = `已定义 ${res.id}（${res.state}）`;
          showNew.value = false;
          await refresh();
        } catch (e) {
          newMsgErr.value = true;
          newMsg.value = e.message || "定义失败";
        } finally {
          defining.value = false;
        }
      }
      function copyText(t) {
        try {
          navigator.clipboard.writeText(t);
          window.$toast && window.$toast("已复制", "info");
        } catch (e) {
        }
      }
      function selectPanel(id) {
        activePanelId.value = id;
        renderActivePanel();
      }
      async function renderActivePanel() {
        await vue.nextTick();
        const el = clientPanelEl.value;
        if (!el) return;
        el.innerHTML = "";
        const panel = pluginRuntime_js.clientPanels.find((p) => p.id === activePanelId.value);
        if (panel && panel.render) {
          try {
            panel.render(el, pluginRuntime_js.getUIFor(panel.pluginName));
          } catch (e) {
            console.warn("[plugin] 面板渲染错误", panel.id, e);
            el.innerHTML = '<div style="color:var(--text-muted);padding:8px;font-size:12px">面板渲染失败</div>';
          }
        }
      }
      function onPanelsChanged(panels) {
        if (!activePanelId.value || !panels.some((p) => p.id === activePanelId.value)) {
          activePanelId.value = panels.length ? panels[0].id : "";
        }
        renderActivePanel();
      }
      const slotGroups = vue.ref([]);
      let slotUnsub = null;
      function refreshSlots() {
        const keys = [...new Set(pluginRuntime_js.clientSlots.map((s) => s.slotId + "::" + s.kind))];
        slotGroups.value = keys.map((k) => {
          const [slotId, kind] = k.split("::");
          const candidates = pluginRuntime_js.getSlotCandidates(slotId).filter((c) => c.kind === kind);
          return { slotId, kind, owner: pluginRuntime_js.getSlotOwner(slotId), candidates, builtin: null };
        });
      }
      function overlayActive(slotId, pluginName) {
        return pluginRuntime_js.isOverlayActive(slotId, pluginName);
      }
      function toggleOverlay(slotId, pluginName, on) {
        pluginRuntime_js.setOverlayActive(slotId, pluginName, on);
      }
      function switchSlot(slotId, pluginName) {
        pluginRuntime_js.setSlotOwner(slotId, pluginName || "");
        refreshSlots();
      }
      vue.onMounted(() => {
        pluginRuntime_js.setPanelMount(onPanelsChanged);
        slotUnsub = pluginRuntime_js.setSlotMount(refreshSlots);
        pluginRuntime_js.startPolling();
        refresh();
        loadWsToolsetMap();
        loadBuiltinGroups();
      });
      vue.onUnmounted(() => {
        pluginRuntime_js.stopPolling();
        pluginRuntime_js.setPanelMount(null);
        if (slotUnsub) {
          slotUnsub();
          slotUnsub = null;
        }
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1$1, [
          vue.createCommentVNode(" 头部 "),
          vue.createElementVNode("div", _hoisted_2$1, [
            vue.createElementVNode("span", _hoisted_3$1, [
              vue.createVNode(SvgIcon, {
                name: "puzzle",
                size: 14
              }),
              _cache[12] || (_cache[12] = vue.createTextVNode(
                " 插件",
                -1
                /* CACHED */
              ))
            ]),
            vue.createElementVNode("div", _hoisted_4, [
              vue.createElementVNode("button", {
                class: "pp-icon-btn",
                onClick: refresh,
                title: "刷新"
              }, [
                vue.createVNode(SvgIcon, {
                  name: "refresh",
                  size: 13,
                  class: vue.normalizeClass({ spinning: refreshing.value })
                }, null, 8, ["class"])
              ]),
              vue.createElementVNode(
                "button",
                {
                  class: vue.normalizeClass(["pp-icon-btn", { active: showToolset.value }]),
                  onClick: _cache[0] || (_cache[0] = ($event) => showToolset.value = !showToolset.value),
                  title: "工具集管理（插件化：加插件/删插件/摘工具）"
                },
                [
                  vue.createVNode(SvgIcon, {
                    name: "layers",
                    size: 13
                  })
                ],
                2
                /* CLASS */
              ),
              vue.createElementVNode("button", {
                class: "pp-icon-btn",
                onClick: _cache[1] || (_cache[1] = ($event) => showNew.value = !showNew.value),
                title: "新建插件"
              }, [
                vue.createVNode(SvgIcon, {
                  name: "plus",
                  size: 14
                })
              ])
            ])
          ]),
          vue.createCommentVNode(" 工具集管理（插件化思路：add_plugin / rm_plugin / rm_tool / enable_tool） "),
          showToolset.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_5, [
            vue.createElementVNode("div", _hoisted_6, [
              vue.withDirectives(vue.createElementVNode(
                "select",
                {
                  "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => tsName.value = $event),
                  class: "pp-input pp-lang",
                  onChange: loadToolsetDetail
                },
                [
                  _cache[13] || (_cache[13] = vue.createElementVNode(
                    "option",
                    { value: "" },
                    "选择工具集…",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode("optgroup", _hoisted_7, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(toolsetMetas.value.filter((x) => x.scope !== "builtin"), (t) => {
                        return vue.openBlock(), vue.createElementBlock("option", {
                          key: t.name,
                          value: t.name
                        }, vue.toDisplayString(t.name) + "（" + vue.toDisplayString(t.pluginCount) + " 插件）", 9, _hoisted_8);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    )),
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(toolsetMetas.value.filter((x) => x.scope === "builtin"), (t) => {
                        return vue.openBlock(), vue.createElementBlock("option", {
                          key: t.name,
                          value: t.name
                        }, vue.toDisplayString(t.name) + "（" + vue.toDisplayString(t.pluginCount) + " 插件·内置默认）", 9, _hoisted_9);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ])
                ],
                544
                /* NEED_HYDRATION, NEED_PATCH */
              ), [
                [vue.vModelSelect, tsName.value]
              ]),
              vue.createElementVNode("button", {
                class: "pp-btn",
                onClick: loadToolsets
              }, "刷新")
            ]),
            tsDetail.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_10, [
              vue.createElementVNode("div", _hoisted_11, [
                vue.createTextVNode(
                  vue.toDisplayString(tsDetail.value.name) + " ",
                  1
                  /* TEXT */
                ),
                vue.createElementVNode(
                  "span",
                  _hoisted_12,
                  vue.toDisplayString(tsDetail.value.project ? tsDetail.value.project + "·" : "") + vue.toDisplayString(tsDetail.value.description || "工具集"),
                  1
                  /* TEXT */
                )
              ]),
              (vue.openBlock(true), vue.createElementBlock(
                vue.Fragment,
                null,
                vue.renderList(tsDetail.value.plugins, (pl) => {
                  return vue.openBlock(), vue.createElementBlock("div", {
                    key: pl.name,
                    class: "pp-ts-plugin"
                  }, [
                    vue.createElementVNode("div", _hoisted_13, [
                      vue.createElementVNode(
                        "span",
                        _hoisted_14,
                        vue.toDisplayString(pl.name),
                        1
                        /* TEXT */
                      ),
                      tsDetail.value.scope !== "builtin" ? (vue.openBlock(), vue.createElementBlock("button", {
                        key: 0,
                        class: "pp-btn danger",
                        onClick: ($event) => edit({ action: "rm_plugin", plugin_name: pl.name })
                      }, "移出工具集", 8, _hoisted_15)) : (vue.openBlock(), vue.createElementBlock("span", _hoisted_16, "内置"))
                    ]),
                    pl.purpose ? (vue.openBlock(), vue.createElementBlock(
                      "div",
                      _hoisted_17,
                      vue.toDisplayString(pl.purpose),
                      1
                      /* TEXT */
                    )) : vue.createCommentVNode("v-if", true),
                    pluginToolsOf(pl.name).length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_18, [
                      (vue.openBlock(true), vue.createElementBlock(
                        vue.Fragment,
                        null,
                        vue.renderList(pluginToolsOf(pl.name), (t) => {
                          return vue.openBlock(), vue.createElementBlock("label", {
                            key: t,
                            class: "pp-ts-tool",
                            title: isToolDisabled(pl, t) ? "已摘除（对 agent 不可见），点击恢复" : "点击摘除（插件保留、工具不可见）"
                          }, [
                            vue.createElementVNode("input", {
                              type: "checkbox",
                              checked: !isToolDisabled(pl, t),
                              onChange: ($event) => toggleTool(pl, t)
                            }, null, 40, _hoisted_20),
                            vue.createElementVNode(
                              "span",
                              {
                                class: vue.normalizeClass({ off: isToolDisabled(pl, t) })
                              },
                              vue.toDisplayString(t),
                              3
                              /* TEXT, CLASS */
                            )
                          ], 8, _hoisted_19);
                        }),
                        128
                        /* KEYED_FRAGMENT */
                      ))
                    ])) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_21, "（插件未运行或无工具）"))
                  ]);
                }),
                128
                /* KEYED_FRAGMENT */
              )),
              vue.createElementVNode("div", _hoisted_22, [
                vue.withDirectives(vue.createElementVNode(
                  "select",
                  {
                    "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => addPluginName.value = $event),
                    class: "pp-input pp-lang"
                  },
                  [
                    _cache[14] || (_cache[14] = vue.createElementVNode(
                      "option",
                      { value: "" },
                      "把宿主插件加入工具集…",
                      -1
                      /* CACHED */
                    )),
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(addablePlugins.value, (p) => {
                        return vue.openBlock(), vue.createElementBlock("option", {
                          key: p.name,
                          value: p.name
                        }, [
                          vue.createTextVNode(
                            vue.toDisplayString(p.name),
                            1
                            /* TEXT */
                          ),
                          p.tools && p.tools.length ? (vue.openBlock(), vue.createElementBlock(
                            vue.Fragment,
                            { key: 0 },
                            [
                              vue.createTextVNode(
                                "（" + vue.toDisplayString(p.tools.length) + " 工具）",
                                1
                                /* TEXT */
                              )
                            ],
                            64
                            /* STABLE_FRAGMENT */
                          )) : vue.createCommentVNode("v-if", true)
                        ], 8, _hoisted_23);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ],
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelSelect, addPluginName.value]
                ]),
                vue.createElementVNode("button", {
                  class: "pp-btn primary",
                  disabled: !addPluginName.value,
                  onClick: doAddPlugin
                }, "加入", 8, _hoisted_24)
              ])
            ])) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_25, "选择上方工具集查看/编辑其插件与工具"))
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 新建插件表单 "),
          showNew.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_26, [
            _cache[17] || (_cache[17] = vue.createElementVNode(
              "div",
              { class: "pp-new-title" },
              "新建 JS 动态插件",
              -1
              /* CACHED */
            )),
            vue.withDirectives(vue.createElementVNode(
              "input",
              {
                "onUpdate:modelValue": _cache[4] || (_cache[4] = ($event) => newForm.purpose = $event),
                placeholder: "用途说明（必填）",
                class: "pp-input"
              },
              null,
              512
              /* NEED_PATCH */
            ), [
              [vue.vModelText, newForm.purpose]
            ]),
            vue.withDirectives(vue.createElementVNode(
              "textarea",
              {
                "onUpdate:modelValue": _cache[5] || (_cache[5] = ($event) => newForm.code = $event),
                placeholder: "host 半代码（必填）：(async () => { return { name, apply(ctx, config) } })()",
                class: "pp-textarea code",
                rows: "6"
              },
              null,
              512
              /* NEED_PATCH */
            ), [
              [vue.vModelText, newForm.code]
            ]),
            vue.withDirectives(vue.createElementVNode(
              "textarea",
              {
                "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => newForm.client = $event),
                placeholder: "client 半代码（可选）：(ui) => { ui.registerPanel({...}); ui.on('ui:xxx', fn) }",
                class: "pp-textarea code",
                rows: "4"
              },
              null,
              512
              /* NEED_PATCH */
            ), [
              [vue.vModelText, newForm.client]
            ]),
            vue.createElementVNode("div", _hoisted_27, [
              vue.withDirectives(vue.createElementVNode(
                "select",
                {
                  "onUpdate:modelValue": _cache[7] || (_cache[7] = ($event) => newForm.language = $event),
                  class: "pp-input pp-lang"
                },
                [..._cache[15] || (_cache[15] = [
                  vue.createElementVNode(
                    "option",
                    { value: "" },
                    "语言(自动)",
                    -1
                    /* CACHED */
                  ),
                  vue.createElementVNode(
                    "option",
                    { value: "js" },
                    "js",
                    -1
                    /* CACHED */
                  ),
                  vue.createElementVNode(
                    "option",
                    { value: "ts" },
                    "ts",
                    -1
                    /* CACHED */
                  )
                ])],
                512
                /* NEED_PATCH */
              ), [
                [vue.vModelSelect, newForm.language]
              ]),
              vue.createElementVNode("label", _hoisted_28, [
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    type: "checkbox",
                    "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => newForm.run = $event)
                  },
                  null,
                  512
                  /* NEED_PATCH */
                ), [
                  [vue.vModelCheckbox, newForm.run]
                ]),
                _cache[16] || (_cache[16] = vue.createTextVNode(
                  " 定义后立即装载",
                  -1
                  /* CACHED */
                ))
              ]),
              vue.createElementVNode("button", {
                class: "pp-btn primary",
                disabled: defining.value || !newForm.purpose || !newForm.code,
                onClick: doDefine
              }, vue.toDisplayString(defining.value ? "定义中…" : "定义"), 9, _hoisted_29)
            ]),
            newMsg.value ? (vue.openBlock(), vue.createElementBlock(
              "div",
              {
                key: 0,
                class: vue.normalizeClass(["pp-new-msg", { err: newMsgErr.value }])
              },
              vue.toDisplayString(newMsg.value),
              3
              /* TEXT, CLASS */
            )) : vue.createCommentVNode("v-if", true)
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" 客户端面板区（client 半注册的自定义面板） "),
          vue.unref(pluginRuntime_js.clientPanels).length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_30, [
            vue.createElementVNode("div", _hoisted_31, [
              (vue.openBlock(true), vue.createElementBlock(
                vue.Fragment,
                null,
                vue.renderList(vue.unref(pluginRuntime_js.clientPanels), (p) => {
                  return vue.openBlock(), vue.createElementBlock("div", {
                    key: p.id,
                    class: vue.normalizeClass(["pp-client-tab", { active: activePanelId.value === p.id }]),
                    onClick: ($event) => selectPanel(p.id)
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: p.icon || "sparkles",
                      size: 12
                    }, null, 8, ["name"]),
                    vue.createElementVNode(
                      "span",
                      _hoisted_33,
                      vue.toDisplayString(p.title),
                      1
                      /* TEXT */
                    )
                  ], 10, _hoisted_32);
                }),
                128
                /* KEYED_FRAGMENT */
              ))
            ]),
            vue.createElementVNode(
              "div",
              {
                ref_key: "clientPanelEl",
                ref: clientPanelEl,
                class: "pp-client-body"
              },
              null,
              512
              /* NEED_PATCH */
            )
          ])) : vue.createCommentVNode("v-if", true),
          vue.createCommentVNode(" UI 槽位区（Slot 系统：client 半注册的可替换界面区域，如底部状态栏） "),
          slotGroups.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_34, [
            vue.createElementVNode("div", {
              class: "pp-slots-head",
              onClick: _cache[9] || (_cache[9] = ($event) => slotsOpen.value = !slotsOpen.value),
              title: slotsOpen.value ? "点击收起 UI 槽位列表" : "点击展开 UI 槽位列表",
              style: { "cursor": "pointer" }
            }, [
              vue.createElementVNode("span", _hoisted_36, [
                vue.createVNode(SvgIcon, {
                  name: "layers",
                  size: 13
                }),
                _cache[18] || (_cache[18] = vue.createTextVNode(
                  " UI 槽位",
                  -1
                  /* CACHED */
                ))
              ]),
              _cache[19] || (_cache[19] = vue.createElementVNode(
                "span",
                { class: "pp-slots-sub" },
                "插件可替换的界面区域",
                -1
                /* CACHED */
              )),
              vue.createVNode(SvgIcon, {
                name: "chevron-right",
                size: 11,
                class: vue.normalizeClass(["pp-chevron", { open: slotsOpen.value }])
              }, null, 8, ["class"])
            ], 8, _hoisted_35),
            slotsOpen.value ? (vue.openBlock(true), vue.createElementBlock(
              vue.Fragment,
              { key: 0 },
              vue.renderList(slotGroups.value, (g) => {
                return vue.openBlock(), vue.createElementBlock("div", {
                  key: g.slotId + "::" + g.kind,
                  class: "pp-slot-row"
                }, [
                  vue.createElementVNode("div", _hoisted_37, [
                    vue.createElementVNode("div", _hoisted_38, [
                      vue.createElementVNode(
                        "span",
                        _hoisted_39,
                        vue.toDisplayString(g.slotId),
                        1
                        /* TEXT */
                      ),
                      vue.createElementVNode(
                        "span",
                        {
                          class: vue.normalizeClass(["pp-slot-kind", g.kind === "list" ? "kind-list" : "kind-single"])
                        },
                        vue.toDisplayString(g.kind === "list" ? "叠加" : "替换"),
                        3
                        /* TEXT, CLASS */
                      )
                    ]),
                    vue.createElementVNode(
                      "span",
                      {
                        class: vue.normalizeClass(["pp-slot-owner", { builtin: !g.owner && g.kind !== "list" }])
                      },
                      vue.toDisplayString(g.kind === "list" ? g.candidates.length ? g.candidates.length + " 个叠加条目" : "（无叠加条目）" : g.owner ? g.owner : g.builtin ? "内置组件" : "（无宿主）"),
                      3
                      /* TEXT, CLASS */
                    )
                  ]),
                  vue.createCommentVNode(" single 槽位：下拉切换占用者（内置默认 / 插件占用者） "),
                  g.kind !== "list" ? (vue.openBlock(), vue.createElementBlock("select", {
                    key: 0,
                    class: "pp-input pp-slot-select",
                    value: g.owner,
                    onChange: ($event) => switchSlot(g.slotId, $event.target.value),
                    title: "切换 " + g.slotId + " 区域的渲染者"
                  }, [
                    vue.createElementVNode(
                      "option",
                      _hoisted_41,
                      vue.toDisplayString(g.builtin ? "内置组件（默认）" : "（未占用）"),
                      1
                      /* TEXT */
                    ),
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(g.candidates, (c) => {
                        return vue.openBlock(), vue.createElementBlock("option", {
                          key: c.pluginName,
                          value: c.pluginName
                        }, vue.toDisplayString(c.pluginName) + " · " + vue.toDisplayString(c.title), 9, _hoisted_42);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    ))
                  ], 40, _hoisted_40)) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_43, [
                    (vue.openBlock(true), vue.createElementBlock(
                      vue.Fragment,
                      null,
                      vue.renderList(g.candidates, (c) => {
                        return vue.openBlock(), vue.createElementBlock("label", {
                          key: c.pluginName,
                          class: "pp-slot-list-item"
                        }, [
                          vue.createElementVNode("input", {
                            type: "checkbox",
                            checked: overlayActive(g.slotId, c.pluginName),
                            onChange: ($event) => toggleOverlay(g.slotId, c.pluginName, $event.target.checked)
                          }, null, 40, _hoisted_44),
                          vue.createElementVNode(
                            "span",
                            null,
                            vue.toDisplayString(c.pluginName) + " · " + vue.toDisplayString(c.title),
                            1
                            /* TEXT */
                          )
                        ]);
                      }),
                      128
                      /* KEYED_FRAGMENT */
                    )),
                    !g.candidates.length ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_45, "（无叠加条目）")) : vue.createCommentVNode("v-if", true)
                  ]))
                ]);
              }),
              128
              /* KEYED_FRAGMENT */
            )) : vue.createCommentVNode("v-if", true)
          ])) : vue.createCommentVNode("v-if", true),
          vue.createElementVNode("div", _hoisted_46, [
            vue.createCommentVNode(" 内置工具区（框架自带工具：默认在工作区工具集，可勾选加入/移出）\n      ★ 2026-08-20：默认收缩（折叠），点击标题展开 "),
            builtinGroups.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_47, [
              vue.createElementVNode("div", {
                class: "pp-builtin-head",
                onClick: _cache[10] || (_cache[10] = ($event) => builtinOpen.value = !builtinOpen.value),
                title: builtinOpen.value ? "点击收起内置工具区" : "点击展开内置工具区",
                style: { "cursor": "pointer" }
              }, [
                vue.createElementVNode("span", _hoisted_49, [
                  vue.createVNode(SvgIcon, {
                    name: "package",
                    size: 12
                  }),
                  _cache[20] || (_cache[20] = vue.createTextVNode(
                    " 内置工具（框架自带）",
                    -1
                    /* CACHED */
                  ))
                ]),
                _cache[21] || (_cache[21] = vue.createElementVNode(
                  "span",
                  { class: "pp-builtin-sub" },
                  "勾选=在工作区工具集中（agent 可用）；取消=移出",
                  -1
                  /* CACHED */
                )),
                vue.createVNode(SvgIcon, {
                  name: "chevron-right",
                  size: 11,
                  class: vue.normalizeClass(["pp-chevron", { open: builtinOpen.value }])
                }, null, 8, ["class"])
              ], 8, _hoisted_48),
              builtinOpen.value ? (vue.openBlock(true), vue.createElementBlock(
                vue.Fragment,
                { key: 0 },
                vue.renderList(builtinGroups.value, (g) => {
                  return vue.openBlock(), vue.createElementBlock("div", {
                    key: g.name,
                    class: "pp-builtin-group"
                  }, [
                    vue.createElementVNode("div", _hoisted_50, [
                      vue.createElementVNode(
                        "span",
                        _hoisted_51,
                        vue.toDisplayString(g.title || g.name),
                        1
                        /* TEXT */
                      ),
                      g.desc ? (vue.openBlock(), vue.createElementBlock(
                        "span",
                        _hoisted_52,
                        vue.toDisplayString(g.desc),
                        1
                        /* TEXT */
                      )) : vue.createCommentVNode("v-if", true),
                      vue.createElementVNode(
                        "span",
                        _hoisted_53,
                        vue.toDisplayString(g.tools.length) + " 工具",
                        1
                        /* TEXT */
                      ),
                      !g.allOn ? (vue.openBlock(), vue.createElementBlock("button", {
                        key: 1,
                        class: "pp-btn mini",
                        onClick: ($event) => enableBuiltinGroup(g),
                        title: "整组加入工作区工具集"
                      }, "整组启用", 8, _hoisted_54)) : vue.createCommentVNode("v-if", true),
                      g.anyOn ? (vue.openBlock(), vue.createElementBlock("button", {
                        key: 2,
                        class: "pp-btn mini danger",
                        onClick: ($event) => disableBuiltinGroup(g),
                        title: "整组移出工作区工具集"
                      }, "整组移出", 8, _hoisted_55)) : vue.createCommentVNode("v-if", true)
                    ]),
                    vue.createElementVNode("div", _hoisted_56, [
                      (vue.openBlock(true), vue.createElementBlock(
                        vue.Fragment,
                        null,
                        vue.renderList(g.tools, (t) => {
                          return vue.openBlock(), vue.createElementBlock("div", {
                            key: t.name,
                            class: "pp-d-tool",
                            title: t.desc
                          }, [
                            vue.createElementVNode(
                              "span",
                              _hoisted_58,
                              vue.toDisplayString(t.name),
                              1
                              /* TEXT */
                            ),
                            vue.createElementVNode("label", {
                              class: "pp-switch",
                              title: t.enabled ? "在工作区工具集中（agent 可用）；点击移出" : "未加入工作区工具集；点击加入"
                            }, [
                              vue.createElementVNode("input", {
                                type: "checkbox",
                                checked: t.enabled,
                                onChange: ($event) => toggleBuiltinTool(t, $event.target.checked)
                              }, null, 40, _hoisted_60),
                              _cache[22] || (_cache[22] = vue.createElementVNode(
                                "span",
                                { class: "pp-switch-track" },
                                null,
                                -1
                                /* CACHED */
                              ))
                            ], 8, _hoisted_59)
                          ], 8, _hoisted_57);
                        }),
                        128
                        /* KEYED_FRAGMENT */
                      ))
                    ])
                  ]);
                }),
                128
                /* KEYED_FRAGMENT */
              )) : vue.createCommentVNode("v-if", true)
            ])) : vue.createCommentVNode("v-if", true),
            loading.value && plugins.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_61, [
              vue.createVNode(SvgIcon, {
                name: "refresh",
                size: 16,
                class: "spinner"
              }),
              _cache[23] || (_cache[23] = vue.createElementVNode(
                "span",
                null,
                "加载插件…",
                -1
                /* CACHED */
              ))
            ])) : loadError.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_62, [
              vue.createVNode(SvgIcon, {
                name: "puzzle",
                size: 22,
                color: "var(--text-muted)"
              }),
              _cache[24] || (_cache[24] = vue.createElementVNode(
                "span",
                null,
                "插件列表加载失败",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("button", {
                class: "pp-btn primary",
                onClick: refresh
              }, "重试")
            ])) : plugins.value.length === 0 && !loading.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_63, [
              vue.createVNode(SvgIcon, {
                name: "puzzle",
                size: 22,
                color: "var(--text-muted)"
              }),
              _cache[25] || (_cache[25] = vue.createElementVNode(
                "span",
                null,
                "暂无插件",
                -1
                /* CACHED */
              )),
              _cache[26] || (_cache[26] = vue.createElementVNode(
                "span",
                { class: "pp-empty-sub" },
                "点击上方 + 新建 JS 动态插件，或用对话 cordis_define 定义",
                -1
                /* CACHED */
              ))
            ])) : vue.createCommentVNode("v-if", true),
            (vue.openBlock(true), vue.createElementBlock(
              vue.Fragment,
              null,
              vue.renderList(plugins.value, (p) => {
                return vue.openBlock(), vue.createElementBlock("div", {
                  key: p.name,
                  class: "pp-item"
                }, [
                  vue.createElementVNode("div", {
                    class: "pp-item-row",
                    onClick: ($event) => toggleDetail(p)
                  }, [
                    vue.createElementVNode(
                      "span",
                      {
                        class: vue.normalizeClass(["pp-state", p.state === "running" ? "on" : "off"])
                      },
                      null,
                      2
                      /* CLASS */
                    ),
                    vue.createElementVNode("span", {
                      class: "pp-name",
                      title: p.purpose || p.name
                    }, vue.toDisplayString(p.name), 9, _hoisted_65),
                    vue.createElementVNode(
                      "span",
                      {
                        class: vue.normalizeClass(["pp-src", p.source])
                      },
                      vue.toDisplayString(p.source),
                      3
                      /* TEXT, CLASS */
                    ),
                    p.scope === "global" ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_66, "全局")) : vue.createCommentVNode("v-if", true),
                    p.hasClient ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_67, "UI")) : vue.createCommentVNode("v-if", true),
                    p.tools && p.tools.length ? (vue.openBlock(), vue.createElementBlock("span", {
                      key: 2,
                      class: "pp-count",
                      title: p.tools.join(", ")
                    }, vue.toDisplayString(p.tools.length) + " 工具", 9, _hoisted_68)) : vue.createCommentVNode("v-if", true),
                    p.hasClient && uiSlotsOf(p.name).length ? (vue.openBlock(), vue.createElementBlock(
                      vue.Fragment,
                      { key: 3 },
                      [
                        vue.createElementVNode(
                          "span",
                          {
                            class: vue.normalizeClass(["pp-ui-label", { on: uiPluginActive(p.name) }])
                          },
                          vue.toDisplayString(uiPluginActive(p.name) ? "UI 已启用" : "UI 未启用"),
                          3
                          /* TEXT, CLASS */
                        ),
                        vue.createElementVNode("label", {
                          class: "pp-switch",
                          title: uiPluginActive(p.name) ? "停用该插件的 UI（恢复内置界面）" : "启用该插件的 UI（替换对应界面区域）"
                        }, [
                          vue.createElementVNode("input", {
                            type: "checkbox",
                            checked: uiPluginActive(p.name),
                            onChange: ($event) => toggleUiPlugin(p, $event.target.checked),
                            onClick: _cache[11] || (_cache[11] = vue.withModifiers(() => {
                            }, ["stop"]))
                          }, null, 40, _hoisted_70),
                          _cache[27] || (_cache[27] = vue.createElementVNode(
                            "span",
                            { class: "pp-switch-track" },
                            null,
                            -1
                            /* CACHED */
                          ))
                        ], 8, _hoisted_69)
                      ],
                      64
                      /* STABLE_FRAGMENT */
                    )) : vue.createCommentVNode("v-if", true),
                    vue.createVNode(SvgIcon, {
                      name: "chevron-right",
                      size: 12,
                      class: vue.normalizeClass(["pp-chevron", { open: expanded[p.name] }])
                    }, null, 8, ["class"])
                  ], 8, _hoisted_64),
                  expanded[p.name] ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_71, [
                    p.purpose ? (vue.openBlock(), vue.createElementBlock(
                      "div",
                      _hoisted_72,
                      vue.toDisplayString(p.purpose),
                      1
                      /* TEXT */
                    )) : vue.createCommentVNode("v-if", true),
                    p.defId ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_73, [
                      vue.createTextVNode(
                        "定义: " + vue.toDisplayString(p.defId),
                        1
                        /* TEXT */
                      ),
                      p.version ? (vue.openBlock(), vue.createElementBlock(
                        "span",
                        _hoisted_74,
                        " · " + vue.toDisplayString(p.version),
                        1
                        /* TEXT */
                      )) : vue.createCommentVNode("v-if", true)
                    ])) : vue.createCommentVNode("v-if", true),
                    p.provides && p.provides.length ? (vue.openBlock(), vue.createElementBlock(
                      "div",
                      _hoisted_75,
                      "服务: " + vue.toDisplayString(p.provides.join(", ")),
                      1
                      /* TEXT */
                    )) : vue.createCommentVNode("v-if", true),
                    p.sections && p.sections.length ? (vue.openBlock(), vue.createElementBlock(
                      "div",
                      _hoisted_76,
                      "提示片段: " + vue.toDisplayString(p.sections.join(", ")),
                      1
                      /* TEXT */
                    )) : vue.createCommentVNode("v-if", true),
                    p.tools && p.tools.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_77, [
                      vue.createElementVNode(
                        "div",
                        _hoisted_78,
                        "工具（" + vue.toDisplayString(p.tools.length) + "）· 勾选=对 cordis（JS 插件运行时）可见；取消=隐藏。agent 可用性由工作区工具集决定",
                        1
                        /* TEXT */
                      ),
                      (vue.openBlock(true), vue.createElementBlock(
                        vue.Fragment,
                        null,
                        vue.renderList(p.tools, (t) => {
                          return vue.openBlock(), vue.createElementBlock("div", {
                            key: t,
                            class: "pp-d-tool"
                          }, [
                            vue.createElementVNode("span", {
                              class: "pp-d-tname",
                              title: t
                            }, vue.toDisplayString(t), 9, _hoisted_79),
                            vue.createElementVNode("label", {
                              class: "pp-switch",
                              title: pluginToolOn(p, t) ? "对 cordis 可见；点击隐藏（JS 插件 ctx.tools 看不到它）" : "对 cordis 隐藏；点击恢复可见"
                            }, [
                              vue.createElementVNode("input", {
                                type: "checkbox",
                                checked: pluginToolOn(p, t),
                                onChange: ($event) => _ctx.togglePluginTool(p, t)
                              }, null, 40, _hoisted_81),
                              _cache[28] || (_cache[28] = vue.createElementVNode(
                                "span",
                                { class: "pp-switch-track" },
                                null,
                                -1
                                /* CACHED */
                              ))
                            ], 8, _hoisted_80)
                          ]);
                        }),
                        128
                        /* KEYED_FRAGMENT */
                      ))
                    ])) : vue.createCommentVNode("v-if", true),
                    p.clientCode ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_82, [
                      vue.createElementVNode("div", _hoisted_83, [
                        _cache[29] || (_cache[29] = vue.createElementVNode(
                          "span",
                          null,
                          "client 半源码",
                          -1
                          /* CACHED */
                        )),
                        vue.createElementVNode("button", {
                          class: "pp-icon-btn",
                          onClick: ($event) => copyText(p.clientCode),
                          title: "复制"
                        }, [
                          vue.createVNode(SvgIcon, {
                            name: "copy",
                            size: 11
                          })
                        ], 8, _hoisted_84)
                      ]),
                      vue.createElementVNode(
                        "pre",
                        null,
                        vue.toDisplayString(p.clientCode),
                        1
                        /* TEXT */
                      )
                    ])) : vue.createCommentVNode("v-if", true),
                    vue.createElementVNode("div", _hoisted_85, [
                      vue.createCommentVNode(" UI 类插件（client 半已装载且有槽位）不再显示「停止插件」：\n                 UI 可见性已由勾选/UI 开关控制（取消勾选=隐藏，勾选=恢复），\n                 stop 会卸载 client 半并清空槽位条目 → 勾选框消失无法再启用。\n                 stopped 状态仍保留「启动插件」按钮作为恢复路径。 "),
                      p.state === "running" ? (vue.openBlock(), vue.createElementBlock(
                        vue.Fragment,
                        { key: 0 },
                        [
                          !(p.hasClient && uiSlotsOf(p.name).length) ? (vue.openBlock(), vue.createElementBlock("button", {
                            key: 0,
                            class: "pp-btn",
                            title: "停止整个插件（其全部工具对 agent 不可见）；单工具请用上方工具开关",
                            onClick: ($event) => doAction(p, "stop")
                          }, "停止插件", 8, _hoisted_86)) : vue.createCommentVNode("v-if", true)
                        ],
                        64
                        /* STABLE_FRAGMENT */
                      )) : (vue.openBlock(), vue.createElementBlock("button", {
                        key: 1,
                        class: "pp-btn primary",
                        onClick: ($event) => doAction(p, "start")
                      }, "启动插件", 8, _hoisted_87)),
                      p.source === "js" ? (vue.openBlock(), vue.createElementBlock("button", {
                        key: 2,
                        class: "pp-btn danger",
                        onClick: ($event) => doAction(p, "undefine")
                      }, "删除定义", 8, _hoisted_88)) : vue.createCommentVNode("v-if", true)
                    ])
                  ])) : vue.createCommentVNode("v-if", true)
                ]);
              }),
              128
              /* KEYED_FRAGMENT */
            ))
          ])
        ]);
      };
    }
  };
  const PluginPanel = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-0dafce5a"]]);
  const _hoisted_1 = { class: "sidebar-header" };
  const _hoisted_2 = { class: "sidebar-content" };
  const _hoisted_3 = {
    key: 4,
    class: "sidebar-placeholder"
  };
  const _sfc_main = {
    __name: "Sidebar",
    setup(__props) {
      const headerTitle = vue.computed(() => {
        const titles = { explorer: "文件浏览器", search: "搜索", source: "源代码管理", plugins: "插件" };
        return titles[uiState_js.state.activeActivity] || "";
      });
      const gitHost = vue.ref(null);
      let gitUnmount = null;
      let gitRetryTimer = null;
      function mountGitPanel() {
        const el = gitHost.value;
        if (!el) return;
        el.innerHTML = "";
        const mod = window.GitPanel;
        if (mod && typeof mod.mount === "function") {
          try {
            gitUnmount = mod.mount(el);
            return;
          } catch (e) {
            console.warn("[sidebar] Git 面板挂载失败", e);
            el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + "</div>";
            return;
          }
        }
        if (gitRetryTimer) return;
        let tries = 0;
        gitRetryTimer = setInterval(() => {
          tries++;
          if (window.GitPanel) {
            clearInterval(gitRetryTimer);
            gitRetryTimer = null;
            mountGitPanel();
            return;
          }
          if (tries >= 8) {
            clearInterval(gitRetryTimer);
            gitRetryTimer = null;
            el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">Git 面板未就绪（git-api 插件未启用）</div>';
          }
        }, 800);
        el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">Git 面板加载中...</div>';
      }
      function unmountGitPanel() {
        if (gitRetryTimer) {
          clearInterval(gitRetryTimer);
          gitRetryTimer = null;
        }
        if (gitUnmount) {
          try {
            gitUnmount();
          } catch (e) {
          }
          gitUnmount = null;
        }
      }
      vue.watch(() => uiState_js.state.activeActivity, (a) => {
        if (a === "source") {
          vue.nextTick(mountGitPanel);
        } else {
          unmountGitPanel();
        }
      });
      vue.onUnmounted(unmountGitPanel);
      let dragging = false;
      let startX = 0;
      let startW = 0;
      function startResize(e) {
        dragging = true;
        startX = e.clientX;
        startW = uiState_js.sidebarWidth.value;
        document.addEventListener("mousemove", onMove);
        document.addEventListener("mouseup", stopResize);
        document.body.style.cursor = "ew-resize";
        document.body.style.userSelect = "none";
      }
      function onMove(e) {
        if (!dragging) return;
        uiState_js.sidebarWidth.value = Math.max(120, Math.min(800, startW + (e.clientX - startX)));
      }
      function stopResize() {
        dragging = false;
        document.removeEventListener("mousemove", onMove);
        document.removeEventListener("mouseup", stopResize);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        try {
          localStorage.setItem("paircode-sidebar-width", String(uiState_js.sidebarWidth.value));
        } catch {
        }
      }
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock(
          "div",
          {
            class: "sidebar",
            style: vue.normalizeStyle({ width: vue.unref(uiState_js.sidebarWidth) + "px" })
          },
          [
            vue.createElementVNode("div", _hoisted_1, [
              vue.createElementVNode(
                "span",
                null,
                vue.toDisplayString(headerTitle.value),
                1
                /* TEXT */
              )
            ]),
            vue.createElementVNode("div", _hoisted_2, [
              vue.unref(uiState_js.state).activeActivity === "explorer" ? (vue.openBlock(), vue.createBlock(FileExplorer, { key: 0 })) : vue.unref(uiState_js.state).activeActivity === "search" ? (vue.openBlock(), vue.createBlock(SearchPanel, { key: 1 })) : vue.unref(uiState_js.state).activeActivity === "source" ? (vue.openBlock(), vue.createElementBlock(
                vue.Fragment,
                { key: 2 },
                [
                  vue.createCommentVNode(" Git 源代码管理面板：由 git-api 插件加载 bundle 到 window.GitPanel，\n           本组件动态挂载（跨 bundle，不能静态 import） "),
                  vue.createElementVNode(
                    "div",
                    {
                      ref_key: "gitHost",
                      ref: gitHost,
                      class: "git-host"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  )
                ],
                2112
                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
              )) : vue.unref(uiState_js.state).activeActivity === "plugins" ? (vue.openBlock(), vue.createBlock(PluginPanel, { key: 3 })) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_3, [..._cache[0] || (_cache[0] = [
                vue.createElementVNode(
                  "span",
                  null,
                  "面板加载中...",
                  -1
                  /* CACHED */
                )
              ])]))
            ]),
            vue.createCommentVNode(" 拖拽分隔条（放在 Sidebar 内，绝对定位在右侧边缘） "),
            vue.createElementVNode(
              "div",
              {
                class: "sidebar-resizer",
                onMousedown: vue.withModifiers(startResize, ["prevent"])
              },
              null,
              32
              /* NEED_HYDRATION */
            )
          ],
          4
          /* STYLE */
        );
      };
    }
  };
  const Sidebar = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-a38248a1"]]);
  function mount(el) {
    const app = vue.createApp(Sidebar);
    app.mount(el);
    return () => {
      app.unmount();
    };
  }
  exports.mount = mount;
  Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
  return exports;
})({}, window.__PAIRCODE_CORE.Vue, window.__PAIRCODE_CORE.uiState, window.__PAIRCODE_CORE.api, window.__PAIRCODE_CORE.pluginRuntime);
