var UiModals = (function(exports, vue, uiState_js, pluginRuntime_js, api) {
  "use strict";var __defProp = Object.defineProperty;
var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
var __publicField = (obj, key, value) => __defNormalProp(obj, typeof key !== "symbol" ? key + "" : key, value);

  var _a;
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
  const _sfc_main$8 = {
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
  const SvgIcon = /* @__PURE__ */ _export_sfc(_sfc_main$8, [["__scopeId", "data-v-faf69761"]]);
  const _hoisted_1$6 = { class: "modal-content" };
  const _hoisted_2$6 = { class: "modal-body" };
  const _hoisted_3$6 = {
    key: 0,
    class: "settings-tabs"
  };
  const _hoisted_4$5 = ["onClick"];
  const _hoisted_5$5 = { class: "settings-content" };
  const _hoisted_6$5 = { key: 0 };
  const _hoisted_7$5 = {
    key: 0,
    class: "group-title"
  };
  const _hoisted_8$5 = ["title"];
  const _hoisted_9$5 = ["title"];
  const _hoisted_10$5 = ["onUpdate:modelValue"];
  const _hoisted_11$5 = ["title"];
  const _hoisted_12$5 = { class: "field-control" };
  const _hoisted_13$4 = ["type", "onUpdate:modelValue", "placeholder"];
  const _hoisted_14$4 = ["onUpdate:modelValue", "min", "max", "step"];
  const _hoisted_15$3 = ["onUpdate:modelValue", "onChange"];
  const _hoisted_16$3 = ["value"];
  const _hoisted_17$3 = ["onUpdate:modelValue", "placeholder"];
  const _hoisted_18$3 = { class: "slider-row" };
  const _hoisted_19$3 = ["onUpdate:modelValue", "min", "max", "step"];
  const _hoisted_20$3 = { class: "slider-val" };
  const _hoisted_21$2 = { class: "color-row" };
  const _hoisted_22$2 = ["onUpdate:modelValue"];
  const _hoisted_23$1 = { class: "color-code" };
  const _hoisted_24$1 = ["value", "onInput", "placeholder"];
  const _hoisted_25$1 = ["placeholder"];
  const _hoisted_26$1 = ["onUpdate:modelValue"];
  const _hoisted_27$1 = {
    key: 0,
    class: "setting-hint"
  };
  const _hoisted_28$1 = {
    key: 0,
    class: "settings-empty"
  };
  const _sfc_main$7 = {
    __name: "SettingsModal",
    emits: ["close"],
    setup(__props, { emit: __emit }) {
      const emit = __emit;
      const activeTab = vue.ref("");
      const tabs = vue.computed(() => {
        const list = (uiState_js.state.pluginSchemas || []).map((s) => ({
          key: s.key,
          title: s.title || s.key,
          groups: groupFields(s.fields || [])
        }));
        if (list.length && !activeTab.value) activeTab.value = list[0].key;
        return list;
      });
      function groupFields(fields) {
        const groups = [];
        const map = {};
        for (const f of fields) {
          const g2 = f.group || "";
          if (!map[g2]) {
            map[g2] = [];
            groups.push({ title: g2, fields: map[g2] });
          }
          map[g2].push(f);
        }
        return groups;
      }
      const modelData = vue.ref(null);
      let lastProvider = "";
      async function loadModels() {
        try {
          modelData.value = await api.getModels();
        } catch {
          modelData.value = null;
        }
      }
      function modelsFor(provider) {
        const m2 = modelData.value && modelData.value.models || {};
        return m2[provider] || [];
      }
      function dynamicOptions(tabKey, f) {
        var _a2, _b, _c;
        if (f.optionsSource === "models") {
          const cur = (_a2 = form[tabKey]) == null ? void 0 : _a2[f.name];
          const list = modelsFor((_b = form["ai"]) == null ? void 0 : _b.provider);
          if (cur && !list.includes(cur)) return [...list, cur];
          return list;
        }
        if (f.optionsSource === "providers") {
          const list = modelData.value && modelData.value.providers || [];
          if (list.length) {
            const cur = (_c = form[tabKey]) == null ? void 0 : _c[f.name];
            if (cur && !list.includes(cur)) return [...list, cur];
            return list;
          }
          return f.options || [];
        }
        return f.options || [];
      }
      function onSelectChange(f) {
        if (!f.linkField || !form["ai"]) return;
        const ai = form["ai"];
        const urls = modelData.value && modelData.value.providerBaseURLs || {};
        const oldDefault = urls[lastProvider];
        const b2 = ai[f.linkField];
        if (b2 === void 0 || b2 === "" || oldDefault && b2 === oldDefault) {
          ai[f.linkField] = urls[ai.provider] || "";
        }
        lastProvider = ai.provider;
      }
      const form = vue.reactive({});
      const projectInst = vue.ref("");
      function zeroValue(type) {
        switch (type) {
          case "checkbox":
            return false;
          case "number":
            return 0;
          case "tags":
            return [];
          default:
            return "";
        }
      }
      function buildForm() {
        for (const key of Object.keys(form)) delete form[key];
        const top = uiState_js.state.settings || {};
        const pvals = top.pluginSettings || {};
        for (const s of uiState_js.state.pluginSchemas || []) {
          form[s.key] = {};
          for (const f of s.fields || []) {
            let v2;
            if (f.type === "project") {
              continue;
            }
            if (f.binding) {
              v2 = top[f.binding] !== void 0 ? top[f.binding] : f.default;
            } else {
              const cur = pvals[s.key] || {};
              v2 = cur[f.name] !== void 0 ? cur[f.name] : f.default;
            }
            if (v2 === void 0) v2 = zeroValue(f.type);
            if (f.type === "checkbox") v2 = !!v2;
            if (f.type === "number") v2 = typeof v2 === "number" ? v2 : Number(v2) || 0;
            if (f.type === "tags") v2 = Array.isArray(v2) ? v2 : [];
            form[s.key][f.name] = v2;
          }
        }
        const hasProject = (uiState_js.state.pluginSchemas || []).some((s) => (s.fields || []).some((f) => f.type === "project"));
        projectInst.value = "";
        if (hasProject) loadProjectInstructions();
      }
      function tagsText(tabKey, f) {
        var _a2;
        const v2 = (_a2 = form[tabKey]) == null ? void 0 : _a2[f.name];
        return Array.isArray(v2) ? v2.join(", ") : v2 || "";
      }
      function onTagsInput(tabKey, f, ev) {
        form[tabKey][f.name] = ev.target.value.split(",").map((s) => s.trim()).filter(Boolean);
      }
      async function loadProjectInstructions() {
        try {
          const proj = await api.getInstructions("project");
          projectInst.value = proj.content || "";
        } catch {
        }
      }
      function loadSettings() {
        var _a2;
        buildForm();
        if ((_a2 = uiState_js.state.settings) == null ? void 0 : _a2.theme) uiState_js.applyTheme(uiState_js.state.settings.theme);
      }
      const resetForm = () => {
        loadSettings();
      };
      const saveSettings = async () => {
        try {
          const top = { ...uiState_js.state.settings || {} };
          const pluginOut = {};
          let themeChanged = false;
          for (const s of uiState_js.state.pluginSchemas || []) {
            const vals = form[s.key] || {};
            for (const f of s.fields || []) {
              if (f.type === "project") {
                await api.saveInstructions("project", projectInst.value);
                continue;
              }
              const v2 = vals[f.name];
              if (f.binding) {
                if (f.name === "theme" && v2 !== top[f.binding]) themeChanged = true;
                top[f.binding] = v2;
              } else {
                if (!pluginOut[s.key]) pluginOut[s.key] = {};
                pluginOut[s.key][f.name] = v2;
              }
            }
          }
          await api.apiPut("/settings", { settings: top, pluginSettings: pluginOut });
          uiState_js.state.settings = top;
          if (themeChanged) uiState_js.applyTheme(top.theme);
          window.$toast("设置已保存", "success");
          emit("close");
        } catch (err) {
          window.$toast("保存失败: " + err.message, "error");
        }
      };
      vue.onMounted(() => {
        loadSettings();
        loadModels();
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", {
          class: "modal-overlay",
          onClick: _cache[2] || (_cache[2] = vue.withModifiers(($event) => _ctx.$emit("close"), ["self"]))
        }, [
          vue.createElementVNode("div", _hoisted_1$6, [
            vue.createElementVNode("h2", null, [
              vue.createVNode(SvgIcon, {
                name: "settings",
                size: 18
              }),
              _cache[3] || (_cache[3] = vue.createTextVNode(
                " 设置 ",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("button", {
                class: "modal-close",
                onClick: _cache[0] || (_cache[0] = ($event) => _ctx.$emit("close"))
              }, "×")
            ]),
            vue.createElementVNode("div", _hoisted_2$6, [
              vue.createCommentVNode(" ═══ 纯 schema 驱动：所有配置 tab 由插件 ctx.registerSettings 注册 ═══ "),
              tabs.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_3$6, [
                (vue.openBlock(true), vue.createElementBlock(
                  vue.Fragment,
                  null,
                  vue.renderList(tabs.value, (t) => {
                    return vue.openBlock(), vue.createElementBlock("button", {
                      key: t.key,
                      class: vue.normalizeClass(["settings-tab", { active: activeTab.value === t.key }]),
                      onClick: ($event) => activeTab.value = t.key
                    }, vue.toDisplayString(t.title), 11, _hoisted_4$5);
                  }),
                  128
                  /* KEYED_FRAGMENT */
                ))
              ])) : vue.createCommentVNode("v-if", true),
              vue.createElementVNode("div", _hoisted_5$5, [
                (vue.openBlock(true), vue.createElementBlock(
                  vue.Fragment,
                  null,
                  vue.renderList(tabs.value, (tab) => {
                    return vue.openBlock(), vue.createElementBlock(
                      vue.Fragment,
                      {
                        key: tab.key
                      },
                      [
                        activeTab.value === tab.key ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_6$5, [
                          (vue.openBlock(true), vue.createElementBlock(
                            vue.Fragment,
                            null,
                            vue.renderList(tab.groups, (grp) => {
                              return vue.openBlock(), vue.createElementBlock("div", {
                                key: grp.title || "__main",
                                class: "setting-group"
                              }, [
                                grp.title ? (vue.openBlock(), vue.createElementBlock(
                                  "div",
                                  _hoisted_7$5,
                                  vue.toDisplayString(grp.title),
                                  1
                                  /* TEXT */
                                )) : vue.createCommentVNode("v-if", true),
                                (vue.openBlock(true), vue.createElementBlock(
                                  vue.Fragment,
                                  null,
                                  vue.renderList(grp.fields, (f) => {
                                    return vue.openBlock(), vue.createElementBlock(
                                      "div",
                                      {
                                        key: f.name,
                                        class: vue.normalizeClass(["setting-row", { "row-toggle": f.type === "checkbox" }])
                                      },
                                      [
                                        vue.createCommentVNode(" checkbox：label 与开关同行 "),
                                        f.type === "checkbox" ? (vue.openBlock(), vue.createElementBlock(
                                          vue.Fragment,
                                          { key: 0 },
                                          [
                                            vue.createElementVNode("label", {
                                              class: "field-label",
                                              title: f.hint
                                            }, vue.toDisplayString(f.label), 9, _hoisted_8$5),
                                            vue.createElementVNode("label", {
                                              class: "pp-switch",
                                              title: f.hint
                                            }, [
                                              vue.withDirectives(vue.createElementVNode("input", {
                                                type: "checkbox",
                                                "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event
                                              }, null, 8, _hoisted_10$5), [
                                                [vue.vModelCheckbox, form[tab.key][f.name]]
                                              ]),
                                              _cache[4] || (_cache[4] = vue.createElementVNode(
                                                "span",
                                                { class: "pp-switch-track" },
                                                null,
                                                -1
                                                /* CACHED */
                                              ))
                                            ], 8, _hoisted_9$5)
                                          ],
                                          64
                                          /* STABLE_FRAGMENT */
                                        )) : (vue.openBlock(), vue.createElementBlock(
                                          vue.Fragment,
                                          { key: 1 },
                                          [
                                            vue.createCommentVNode(" 其他类型：label 在上、控件在下、说明文字在控件下方（不挤占输入区） "),
                                            vue.createElementVNode("label", {
                                              class: "field-label",
                                              title: f.hint
                                            }, vue.toDisplayString(f.label), 9, _hoisted_11$5),
                                            vue.createElementVNode("div", _hoisted_12$5, [
                                              vue.createCommentVNode(" text / password "),
                                              f.type === "text" || f.type === "password" ? vue.withDirectives((vue.openBlock(), vue.createElementBlock("input", {
                                                key: 0,
                                                type: f.type === "password" ? "password" : "text",
                                                "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event,
                                                placeholder: f.placeholder
                                              }, null, 8, _hoisted_13$4)), [
                                                [vue.vModelDynamic, form[tab.key][f.name]]
                                              ]) : f.type === "number" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 1 },
                                                [
                                                  vue.createCommentVNode(" number "),
                                                  vue.withDirectives(vue.createElementVNode("input", {
                                                    type: "number",
                                                    "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event,
                                                    min: f.min,
                                                    max: f.max,
                                                    step: f.step
                                                  }, null, 8, _hoisted_14$4), [
                                                    [
                                                      vue.vModelText,
                                                      form[tab.key][f.name],
                                                      void 0,
                                                      { number: true }
                                                    ]
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : f.type === "select" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 2 },
                                                [
                                                  vue.createCommentVNode(" select（optionsSource 驱动动态数据源：models=按服务商模型列表 / providers=服务商列表） "),
                                                  vue.withDirectives(vue.createElementVNode("select", {
                                                    "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event,
                                                    class: "field-select",
                                                    onChange: ($event) => onSelectChange(f)
                                                  }, [
                                                    (vue.openBlock(true), vue.createElementBlock(
                                                      vue.Fragment,
                                                      null,
                                                      vue.renderList(dynamicOptions(tab.key, f), (o) => {
                                                        return vue.openBlock(), vue.createElementBlock("option", {
                                                          key: o,
                                                          value: o
                                                        }, vue.toDisplayString(o), 9, _hoisted_16$3);
                                                      }),
                                                      128
                                                      /* KEYED_FRAGMENT */
                                                    ))
                                                  ], 40, _hoisted_15$3), [
                                                    [vue.vModelSelect, form[tab.key][f.name]]
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : f.type === "textarea" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 3 },
                                                [
                                                  vue.createCommentVNode(" textarea "),
                                                  vue.withDirectives(vue.createElementVNode("textarea", {
                                                    "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event,
                                                    class: "field-textarea",
                                                    rows: "4",
                                                    placeholder: f.placeholder
                                                  }, null, 8, _hoisted_17$3), [
                                                    [vue.vModelText, form[tab.key][f.name]]
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : f.type === "slider" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 4 },
                                                [
                                                  vue.createCommentVNode(" slider "),
                                                  vue.createElementVNode("div", _hoisted_18$3, [
                                                    vue.withDirectives(vue.createElementVNode("input", {
                                                      type: "range",
                                                      "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event,
                                                      min: f.min != null ? f.min : 0,
                                                      max: f.max != null ? f.max : 100,
                                                      step: f.step || 1
                                                    }, null, 8, _hoisted_19$3), [
                                                      [
                                                        vue.vModelText,
                                                        form[tab.key][f.name],
                                                        void 0,
                                                        { number: true }
                                                      ]
                                                    ]),
                                                    vue.createElementVNode(
                                                      "span",
                                                      _hoisted_20$3,
                                                      vue.toDisplayString(form[tab.key][f.name]),
                                                      1
                                                      /* TEXT */
                                                    )
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : f.type === "color" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 5 },
                                                [
                                                  vue.createCommentVNode(" color "),
                                                  vue.createElementVNode("div", _hoisted_21$2, [
                                                    vue.withDirectives(vue.createElementVNode("input", {
                                                      type: "color",
                                                      "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event
                                                    }, null, 8, _hoisted_22$2), [
                                                      [vue.vModelText, form[tab.key][f.name]]
                                                    ]),
                                                    vue.createElementVNode(
                                                      "code",
                                                      _hoisted_23$1,
                                                      vue.toDisplayString(form[tab.key][f.name]),
                                                      1
                                                      /* TEXT */
                                                    )
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : f.type === "tags" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 6 },
                                                [
                                                  vue.createCommentVNode(" tags（逗号分隔数组） "),
                                                  vue.createElementVNode("input", {
                                                    type: "text",
                                                    class: "field-tags",
                                                    value: tagsText(tab.key, f),
                                                    onInput: ($event) => onTagsInput(tab.key, f, $event),
                                                    placeholder: f.placeholder || "逗号分隔"
                                                  }, null, 40, _hoisted_24$1)
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : f.type === "project" ? (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 7 },
                                                [
                                                  vue.createCommentVNode(" project（平台特殊：项目级指令，经 /api/instructions 读写） "),
                                                  vue.withDirectives(vue.createElementVNode("textarea", {
                                                    "onUpdate:modelValue": _cache[1] || (_cache[1] = ($event) => projectInst.value = $event),
                                                    class: "field-textarea",
                                                    rows: "4",
                                                    placeholder: f.placeholder
                                                  }, null, 8, _hoisted_25$1), [
                                                    [vue.vModelText, projectInst.value]
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              )) : (vue.openBlock(), vue.createElementBlock(
                                                vue.Fragment,
                                                { key: 8 },
                                                [
                                                  vue.createCommentVNode(" 兜底 text "),
                                                  vue.withDirectives(vue.createElementVNode("input", {
                                                    type: "text",
                                                    "onUpdate:modelValue": ($event) => form[tab.key][f.name] = $event
                                                  }, null, 8, _hoisted_26$1), [
                                                    [vue.vModelText, form[tab.key][f.name]]
                                                  ])
                                                ],
                                                2112
                                                /* STABLE_FRAGMENT, DEV_ROOT_FRAGMENT */
                                              ))
                                            ]),
                                            f.hint ? (vue.openBlock(), vue.createElementBlock(
                                              "span",
                                              _hoisted_27$1,
                                              vue.toDisplayString(f.hint),
                                              1
                                              /* TEXT */
                                            )) : vue.createCommentVNode("v-if", true)
                                          ],
                                          64
                                          /* STABLE_FRAGMENT */
                                        ))
                                      ],
                                      2
                                      /* CLASS */
                                    );
                                  }),
                                  128
                                  /* KEYED_FRAGMENT */
                                ))
                              ]);
                            }),
                            128
                            /* KEYED_FRAGMENT */
                          ))
                        ])) : vue.createCommentVNode("v-if", true)
                      ],
                      64
                      /* STABLE_FRAGMENT */
                    );
                  }),
                  128
                  /* KEYED_FRAGMENT */
                )),
                !tabs.value.length ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_28$1, "暂无配置项（等待插件注册…）")) : vue.createCommentVNode("v-if", true)
              ])
            ]),
            vue.createElementVNode("div", { class: "modal-footer" }, [
              vue.createElementVNode("button", {
                class: "btn-secondary",
                onClick: resetForm
              }, "撤销"),
              vue.createElementVNode("button", {
                class: "btn-primary",
                onClick: saveSettings
              }, "保存设置")
            ])
          ])
        ]);
      };
    }
  };
  const SettingsModal = /* @__PURE__ */ _export_sfc(_sfc_main$7, [["__scopeId", "data-v-11cfe509"]]);
  const _hoisted_1$5 = { class: "modal-content sys-modal" };
  const _hoisted_2$5 = { class: "modal-header" };
  const _hoisted_3$5 = { class: "modal-body" };
  const _hoisted_4$4 = {
    key: 0,
    class: "loading"
  };
  const _hoisted_5$4 = {
    key: 1,
    class: "sys-info"
  };
  const _hoisted_6$4 = { class: "info-row" };
  const _hoisted_7$4 = { class: "info-row" };
  const _hoisted_8$4 = { class: "info-row" };
  const _hoisted_9$4 = { class: "info-row" };
  const _hoisted_10$4 = { class: "info-row" };
  const _hoisted_11$4 = { class: "info-row" };
  const _hoisted_12$4 = { class: "modal-footer" };
  const _sfc_main$6 = {
    __name: "SystemModal",
    emits: ["close"],
    setup(__props, { emit: __emit }) {
      const loading = vue.ref(true);
      const info = vue.ref({});
      vue.onMounted(async () => {
        try {
          info.value = await api.apiGet("/system/info");
        } catch {
        }
        loading.value = false;
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", {
          class: "modal-overlay",
          onClick: _cache[2] || (_cache[2] = vue.withModifiers(($event) => _ctx.$emit("close"), ["self"]))
        }, [
          vue.createElementVNode("div", _hoisted_1$5, [
            vue.createElementVNode("div", _hoisted_2$5, [
              _cache[3] || (_cache[3] = vue.createElementVNode(
                "h2",
                null,
                "ℹ 系统信息",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("button", {
                class: "modal-close",
                onClick: _cache[0] || (_cache[0] = ($event) => _ctx.$emit("close"))
              }, "×")
            ]),
            vue.createElementVNode("div", _hoisted_3$5, [
              loading.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_4$4, "加载中...")) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_5$4, [
                vue.createElementVNode("div", _hoisted_6$4, [
                  _cache[4] || (_cache[4] = vue.createElementVNode(
                    "label",
                    null,
                    "主机名",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString(info.value.hostname),
                    1
                    /* TEXT */
                  )
                ]),
                vue.createElementVNode("div", _hoisted_7$4, [
                  _cache[5] || (_cache[5] = vue.createElementVNode(
                    "label",
                    null,
                    "当前目录",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString(info.value.cwd),
                    1
                    /* TEXT */
                  )
                ]),
                vue.createElementVNode("div", _hoisted_8$4, [
                  _cache[6] || (_cache[6] = vue.createElementVNode(
                    "label",
                    null,
                    "操作系统",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString(info.value.os),
                    1
                    /* TEXT */
                  )
                ]),
                vue.createElementVNode("div", _hoisted_9$4, [
                  _cache[7] || (_cache[7] = vue.createElementVNode(
                    "label",
                    null,
                    "Go 版本",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString(info.value.goos),
                    1
                    /* TEXT */
                  )
                ]),
                vue.createElementVNode("div", _hoisted_10$4, [
                  _cache[8] || (_cache[8] = vue.createElementVNode(
                    "label",
                    null,
                    "工作区",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString(info.value.workspace),
                    1
                    /* TEXT */
                  )
                ]),
                vue.createElementVNode("div", _hoisted_11$4, [
                  _cache[9] || (_cache[9] = vue.createElementVNode(
                    "label",
                    null,
                    "文件夹",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString((info.value.folders || []).join(", ")),
                    1
                    /* TEXT */
                  )
                ])
              ]))
            ]),
            vue.createElementVNode("div", _hoisted_12$4, [
              vue.createElementVNode("button", {
                class: "btn-secondary",
                onClick: _cache[1] || (_cache[1] = ($event) => _ctx.$emit("close"))
              }, "关闭")
            ])
          ])
        ]);
      };
    }
  };
  const SystemModal = /* @__PURE__ */ _export_sfc(_sfc_main$6, [["__scopeId", "data-v-c27b6ec9"]]);
  const _hoisted_1$4 = { class: "modal-content source-modal" };
  const _hoisted_2$4 = { class: "modal-header" };
  const _hoisted_3$4 = { class: "modal-footer" };
  const _sfc_main$5 = {
    __name: "SourceModal",
    emits: ["close"],
    setup(__props) {
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", {
          class: "modal-overlay",
          onClick: _cache[2] || (_cache[2] = vue.withModifiers(($event) => _ctx.$emit("close"), ["self"]))
        }, [
          vue.createElementVNode("div", _hoisted_1$4, [
            vue.createElementVNode("div", _hoisted_2$4, [
              _cache[3] || (_cache[3] = vue.createElementVNode(
                "h2",
                null,
                "⎔ 源代码管理",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("button", {
                class: "modal-close",
                onClick: _cache[0] || (_cache[0] = ($event) => _ctx.$emit("close"))
              }, "×")
            ]),
            _cache[4] || (_cache[4] = vue.createElementVNode(
              "div",
              { class: "modal-body" },
              [
                vue.createElementVNode("p", { style: { "color": "var(--text-muted)", "text-align": "center", "margin-top": "40px" } }, [
                  vue.createTextVNode(" Git 集成开发中"),
                  vue.createElementVNode("br"),
                  vue.createElementVNode("br"),
                  vue.createTextVNode(" 功能规划："),
                  vue.createElementVNode("br"),
                  vue.createTextVNode(" · Git 状态查看"),
                  vue.createElementVNode("br"),
                  vue.createTextVNode(" · 暂存/提交/推送"),
                  vue.createElementVNode("br"),
                  vue.createTextVNode(" · 分支管理"),
                  vue.createElementVNode("br"),
                  vue.createTextVNode(" · Diff 对比 ")
                ])
              ],
              -1
              /* CACHED */
            )),
            vue.createElementVNode("div", _hoisted_3$4, [
              vue.createElementVNode("button", {
                class: "btn-secondary",
                onClick: _cache[1] || (_cache[1] = ($event) => _ctx.$emit("close"))
              }, "关闭")
            ])
          ])
        ]);
      };
    }
  };
  const SourceModal = /* @__PURE__ */ _export_sfc(_sfc_main$5, [["__scopeId", "data-v-2e060397"]]);
  const _hoisted_1$3 = { class: "modal-overlay" };
  const _hoisted_2$3 = { class: "modal-content market-modal" };
  const _hoisted_3$3 = { class: "modal-header" };
  const _hoisted_4$3 = { class: "market-tabs" };
  const _hoisted_5$3 = ["onClick"];
  const _hoisted_6$3 = { class: "modal-body" };
  const _hoisted_7$3 = {
    key: 0,
    class: "market-search"
  };
  const _hoisted_8$3 = { class: "search-icon" };
  const _hoisted_9$3 = ["disabled", "title"];
  const _hoisted_10$3 = {
    key: 1,
    class: "installed-toolbar"
  };
  const _hoisted_11$3 = {
    key: 2,
    class: "mcp-form"
  };
  const _hoisted_12$3 = { class: "mcp-form-row" };
  const _hoisted_13$3 = { class: "mcp-form-row" };
  const _hoisted_14$3 = { class: "mcp-form-row" };
  const _hoisted_15$2 = { class: "mcp-form-row" };
  const _hoisted_16$2 = { class: "mcp-form-actions" };
  const _hoisted_17$2 = ["disabled"];
  const _hoisted_18$2 = {
    key: 0,
    class: "mcp-form-error"
  };
  const _hoisted_19$2 = {
    key: 3,
    class: "mcp-form"
  };
  const _hoisted_20$2 = { class: "mcp-form-row" };
  const _hoisted_21$1 = { class: "mcp-form-row" };
  const _hoisted_22$1 = { class: "mcp-form-row" };
  const _hoisted_23 = { class: "mcp-form-row" };
  const _hoisted_24 = { class: "mcp-form-actions" };
  const _hoisted_25 = ["disabled"];
  const _hoisted_26 = {
    key: 4,
    class: "skill-viewer"
  };
  const _hoisted_27 = { class: "skill-viewer-header" };
  const _hoisted_28 = { class: "skill-viewer-content" };
  const _hoisted_29 = {
    key: 5,
    class: "market-loading"
  };
  const _hoisted_30 = {
    key: 0,
    class: "installed-group"
  };
  const _hoisted_31 = { class: "ii-icon icon-mcp" };
  const _hoisted_32 = ["title"];
  const _hoisted_33 = { class: "ii-body" };
  const _hoisted_34 = { class: "ii-name" };
  const _hoisted_35 = { class: "ii-desc" };
  const _hoisted_36 = { class: "ii-badge" };
  const _hoisted_37 = { class: "ii-actions" };
  const _hoisted_38 = ["onClick", "title"];
  const _hoisted_39 = ["onClick"];
  const _hoisted_40 = ["onClick"];
  const _hoisted_41 = {
    key: 1,
    class: "installed-group"
  };
  const _hoisted_42 = { class: "ii-icon icon-skill" };
  const _hoisted_43 = { class: "ii-body" };
  const _hoisted_44 = { class: "ii-name" };
  const _hoisted_45 = { class: "ii-desc" };
  const _hoisted_46 = { class: "ii-badge" };
  const _hoisted_47 = { class: "ii-actions" };
  const _hoisted_48 = ["value", "onChange", "title"];
  const _hoisted_49 = ["onClick"];
  const _hoisted_50 = ["onClick"];
  const _hoisted_51 = {
    key: 2,
    class: "market-empty"
  };
  const _hoisted_52 = { class: "me-icon" };
  const _hoisted_53 = { class: "mi-body" };
  const _hoisted_54 = { class: "mi-name" };
  const _hoisted_55 = { class: "mi-desc" };
  const _hoisted_56 = { class: "mi-meta" };
  const _hoisted_57 = {
    key: 0,
    class: "mi-tags"
  };
  const _hoisted_58 = {
    key: 1,
    class: "mi-installed"
  };
  const _hoisted_59 = {
    key: 0,
    class: "mi-install-area"
  };
  const _hoisted_60 = ["onClick", "disabled"];
  const _hoisted_61 = ["onUpdate:modelValue"];
  const _hoisted_62 = ["onClick", "disabled"];
  const _hoisted_63 = ["onClick"];
  const _hoisted_64 = {
    key: 0,
    class: "market-empty"
  };
  const _hoisted_65 = { class: "me-icon" };
  const _hoisted_66 = { key: 0 };
  const _hoisted_67 = { key: 1 };
  const _hoisted_68 = { class: "modal-footer" };
  const _hoisted_69 = { class: "market-count" };
  const _hoisted_70 = {
    key: 0,
    class: "market-error"
  };
  const _sfc_main$4 = {
    __name: "MarketplaceModal",
    emits: ["close"],
    setup(__props, { emit: __emit }) {
      const tab = vue.ref("all");
      const query = vue.ref("");
      const items = vue.ref([]);
      const installing = vue.ref("");
      const loading = vue.ref(false);
      const refreshing = vue.ref(false);
      const error = vue.ref("");
      const refreshTip = vue.ref("");
      const listRef = vue.ref(null);
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
          const [mcpList, skillList] = await Promise.all([
            api.getMcpList("all"),
            api.getSkillsList()
          ]);
          installedMCPs.value = mcpList || [];
          installedSkills.value = skillList || [];
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
      async function setSkillStatus(item, status) {
        var _a2, _b;
        try {
          await api.saveSkillStatus(item.name, item.level || "project", status);
          item.status = status;
          (_a2 = window.$toast) == null ? void 0 : _a2.call(window, `技能「${item.name}」状态已设为 ${status === "off" ? "关闭" : status === "max" ? "始终激活" : "按需"}`, "success");
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
        var _a2, _b;
        refreshing.value = true;
        error.value = "";
        try {
          const result = await api.apiPost("/marketplace/refresh", {});
          refreshTip.value = result.status || "已刷新";
          (_a2 = window.$toast) == null ? void 0 : _a2.call(window, result.message || "远程市场已刷新", "success");
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
        var _a2, _b;
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
          } else if (item.kind === "plugin") {
            body.scope = "project";
          }
          const result = await api.apiPost("/marketplace/install", body);
          item.installed = true;
          (_a2 = window.$toast) == null ? void 0 : _a2.call(window, result.message || "安装成功", "success");
        } catch (err) {
          error.value = "安装失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "安装失败: " + err.message, "error");
        } finally {
          installing.value = "";
        }
      }
      async function toggleMCP(item) {
        var _a2, _b;
        try {
          const result = await api.saveMcpItem({ action: "toggle", name: item.name, level: item.level });
          item.enabled = result.enabled;
          (_a2 = window.$toast) == null ? void 0 : _a2.call(window, `MCP 服务器「${item.name}」${result.enabled ? "已启用" : "已禁用"}`, "success");
        } catch (err) {
          error.value = "切换失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "切换失败: " + err.message, "error");
        }
      }
      async function uninstallItem(item) {
        var _a2, _b;
        error.value = "";
        try {
          const isNpm = (item.source || "").startsWith("npm:");
          if (isNpm) {
            await api.apiPost("/marketplace/uninstall", { id: item.id, kind: "plugin", source: item.source });
          } else if (item.kind === "mcp") {
            await api.saveMcpItem({ action: "delete", name: item.id, level: "user" });
          } else if (item.kind === "skill") {
            await api.deleteSkill(item.id);
          } else if (item.kind === "plugin") {
            await api.apiPost("/toolsets/remove", { name: item.id.replace(/^plugin-/, ""), scope: "project" });
          }
          item.installed = false;
          (_a2 = window.$toast) == null ? void 0 : _a2.call(window, "已卸载: " + item.name, "success");
        } catch (err) {
          error.value = "卸载失败: " + err.message;
          (_b = window.$toast) == null ? void 0 : _b.call(window, "卸载失败: " + err.message, "error");
        }
      }
      vue.onMounted(() => {
        loadSources();
        doSearch();
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1$3, [
          vue.createElementVNode("div", _hoisted_2$3, [
            vue.createElementVNode("div", _hoisted_3$3, [
              vue.createElementVNode("h2", null, [
                vue.createVNode(SvgIcon, {
                  name: "package",
                  size: 20
                }),
                _cache[19] || (_cache[19] = vue.createTextVNode(
                  " 市场",
                  -1
                  /* CACHED */
                ))
              ]),
              vue.createElementVNode("div", _hoisted_4$3, [
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
                    }, vue.toDisplayString(s.label), 11, _hoisted_5$3);
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
                onClick: _cache[2] || (_cache[2] = ($event) => _ctx.$emit("close"))
              }, "×")
            ]),
            vue.createElementVNode("div", _hoisted_6$3, [
              vue.createCommentVNode(" 搜索栏（非「已安装」tab 显示） "),
              tab.value !== "installed" ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_7$3, [
                vue.createElementVNode("div", _hoisted_8$3, [
                  vue.createVNode(SvgIcon, {
                    name: "search",
                    size: 14
                  })
                ]),
                vue.withDirectives(vue.createElementVNode(
                  "input",
                  {
                    "onUpdate:modelValue": _cache[3] || (_cache[3] = ($event) => query.value = $event),
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
                  onClick: _cache[4] || (_cache[4] = ($event) => {
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
                ], 8, _hoisted_9$3)
              ])) : vue.createCommentVNode("v-if", true),
              vue.createCommentVNode(" 「已安装」tab 的操作栏 "),
              tab.value === "installed" ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_10$3, [
                vue.createElementVNode("button", {
                  class: "btn-add-mcp",
                  onClick: _cache[5] || (_cache[5] = ($event) => showAddMCP.value = true)
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "plus",
                    size: 14
                  }),
                  _cache[20] || (_cache[20] = vue.createTextVNode(
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
                  _cache[21] || (_cache[21] = vue.createTextVNode(
                    " 刷新",
                    -1
                    /* CACHED */
                  ))
                ])
              ])) : vue.createCommentVNode("v-if", true),
              vue.createCommentVNode(" 添加 MCP 表单 "),
              showAddMCP.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_11$3, [
                vue.createElementVNode("div", _hoisted_12$3, [
                  _cache[22] || (_cache[22] = vue.createElementVNode(
                    "label",
                    null,
                    "名称",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[6] || (_cache[6] = ($event) => mcpForm.value.name = $event),
                      placeholder: "如 my-server"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, mcpForm.value.name]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_13$3, [
                  _cache[23] || (_cache[23] = vue.createElementVNode(
                    "label",
                    null,
                    "命令",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[7] || (_cache[7] = ($event) => mcpForm.value.command = $event),
                      placeholder: "如 npx / uvx"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, mcpForm.value.command]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_14$3, [
                  _cache[24] || (_cache[24] = vue.createElementVNode(
                    "label",
                    null,
                    "参数",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[8] || (_cache[8] = ($event) => mcpForm.value.argsText = $event),
                      placeholder: "如 -y @modelcontextprotocol/server-filesystem"
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, mcpForm.value.argsText]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_15$2, [
                  _cache[26] || (_cache[26] = vue.createElementVNode(
                    "label",
                    null,
                    "层级",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "select",
                    {
                      "onUpdate:modelValue": _cache[9] || (_cache[9] = ($event) => mcpForm.value.level = $event)
                    },
                    [..._cache[25] || (_cache[25] = [
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
                vue.createElementVNode("div", _hoisted_16$2, [
                  vue.createElementVNode("button", {
                    class: "btn-primary",
                    onClick: saveMCP,
                    disabled: !mcpForm.value.name || savingMCP.value
                  }, vue.toDisplayString(savingMCP.value ? "保存中…" : "保存"), 9, _hoisted_17$2),
                  vue.createElementVNode("button", {
                    class: "btn-secondary",
                    onClick: _cache[10] || (_cache[10] = ($event) => {
                      showAddMCP.value = false;
                      resetMCPForm();
                    })
                  }, "取消")
                ]),
                mcpError.value ? (vue.openBlock(), vue.createElementBlock(
                  "div",
                  _hoisted_18$2,
                  vue.toDisplayString(mcpError.value),
                  1
                  /* TEXT */
                )) : vue.createCommentVNode("v-if", true)
              ])) : vue.createCommentVNode("v-if", true),
              vue.createCommentVNode(" 编辑 MCP 表单 "),
              editingMCP.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_19$2, [
                vue.createElementVNode("div", _hoisted_20$2, [
                  _cache[27] || (_cache[27] = vue.createElementVNode(
                    "label",
                    null,
                    "名称",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[11] || (_cache[11] = ($event) => editMCPForm.value.name = $event)
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, editMCPForm.value.name]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_21$1, [
                  _cache[28] || (_cache[28] = vue.createElementVNode(
                    "label",
                    null,
                    "命令",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[12] || (_cache[12] = ($event) => editMCPForm.value.command = $event)
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, editMCPForm.value.command]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_22$1, [
                  _cache[29] || (_cache[29] = vue.createElementVNode(
                    "label",
                    null,
                    "参数",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      "onUpdate:modelValue": _cache[13] || (_cache[13] = ($event) => editMCPForm.value.argsText = $event)
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelText, editMCPForm.value.argsText]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_23, [
                  _cache[31] || (_cache[31] = vue.createElementVNode(
                    "label",
                    null,
                    "层级",
                    -1
                    /* CACHED */
                  )),
                  vue.withDirectives(vue.createElementVNode(
                    "select",
                    {
                      "onUpdate:modelValue": _cache[14] || (_cache[14] = ($event) => editMCPForm.value.level = $event)
                    },
                    [..._cache[30] || (_cache[30] = [
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
                vue.createElementVNode("div", _hoisted_24, [
                  vue.createElementVNode("button", {
                    class: "btn-primary",
                    onClick: updateMCP,
                    disabled: !editMCPForm.value.name
                  }, "保存", 8, _hoisted_25),
                  vue.createElementVNode("button", {
                    class: "btn-secondary",
                    onClick: _cache[15] || (_cache[15] = ($event) => editingMCP.value = false)
                  }, "取消")
                ])
              ])) : vue.createCommentVNode("v-if", true),
              vue.createCommentVNode(" 技能内容查看 "),
              viewingSkill.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_26, [
                vue.createElementVNode("div", _hoisted_27, [
                  vue.createElementVNode(
                    "strong",
                    null,
                    vue.toDisplayString(viewingSkill.value.name),
                    1
                    /* TEXT */
                  ),
                  vue.createElementVNode("button", {
                    class: "modal-close",
                    onClick: _cache[16] || (_cache[16] = ($event) => viewingSkill.value = null)
                  }, "×")
                ]),
                vue.createElementVNode(
                  "pre",
                  _hoisted_28,
                  vue.toDisplayString(viewingSkill.value.content),
                  1
                  /* TEXT */
                )
              ])) : vue.createCommentVNode("v-if", true),
              vue.createCommentVNode(" 加载状态 "),
              loading.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_29, [..._cache[32] || (_cache[32] = [
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
                      vue.createCommentVNode(" MCP 分组 "),
                      installedMCPs.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_30, [
                        _cache[33] || (_cache[33] = vue.createElementVNode(
                          "div",
                          { class: "installed-group-title" },
                          "MCP 服务器",
                          -1
                          /* CACHED */
                        )),
                        (vue.openBlock(true), vue.createElementBlock(
                          vue.Fragment,
                          null,
                          vue.renderList(installedMCPs.value, (item) => {
                            return vue.openBlock(), vue.createElementBlock("div", {
                              key: "mcp-" + item.name + "-" + item.level,
                              class: "installed-item"
                            }, [
                              vue.createElementVNode("div", _hoisted_31, [
                                vue.createVNode(SvgIcon, {
                                  name: "package",
                                  size: 18
                                })
                              ]),
                              vue.createElementVNode("div", {
                                class: vue.normalizeClass(["ii-status-dot", item.enabled === false ? "dot-disabled" : item._connected ? "dot-connected" : "dot-idle"]),
                                title: item.enabled === false ? "已禁用" : item._connected ? "已连接" : "未连接"
                              }, null, 10, _hoisted_32),
                              vue.createElementVNode("div", _hoisted_33, [
                                vue.createElementVNode(
                                  "div",
                                  _hoisted_34,
                                  vue.toDisplayString(item.name),
                                  1
                                  /* TEXT */
                                ),
                                vue.createElementVNode(
                                  "div",
                                  _hoisted_35,
                                  vue.toDisplayString(item.command) + " " + vue.toDisplayString((item.args || []).join(" ")),
                                  1
                                  /* TEXT */
                                ),
                                vue.createElementVNode(
                                  "span",
                                  _hoisted_36,
                                  "MCP · " + vue.toDisplayString(item.level === "project" ? "工作区级" : "用户级"),
                                  1
                                  /* TEXT */
                                )
                              ]),
                              vue.createElementVNode("div", _hoisted_37, [
                                vue.createElementVNode("button", {
                                  class: vue.normalizeClass(["ii-btn ii-toggle", { "is-enabled": item.enabled !== false }]),
                                  onClick: ($event) => toggleMCP(item),
                                  title: item.enabled === false ? "点击启用" : "点击禁用"
                                }, vue.toDisplayString(item.enabled === false ? "禁用" : "启用"), 11, _hoisted_38),
                                vue.createElementVNode("button", {
                                  class: "ii-btn ii-edit",
                                  onClick: ($event) => startEditMCP(item),
                                  title: "编辑"
                                }, "编辑", 8, _hoisted_39),
                                vue.createElementVNode("button", {
                                  class: "ii-btn ii-del",
                                  onClick: ($event) => delMCP(item),
                                  title: "删除"
                                }, "删除", 8, _hoisted_40)
                              ])
                            ]);
                          }),
                          128
                          /* KEYED_FRAGMENT */
                        ))
                      ])) : vue.createCommentVNode("v-if", true),
                      vue.createCommentVNode(" 技能分组 "),
                      installedSkills.value.length > 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_41, [
                        _cache[36] || (_cache[36] = vue.createElementVNode(
                          "div",
                          { class: "installed-group-title" },
                          "技能",
                          -1
                          /* CACHED */
                        )),
                        (vue.openBlock(true), vue.createElementBlock(
                          vue.Fragment,
                          null,
                          vue.renderList(installedSkills.value, (item) => {
                            return vue.openBlock(), vue.createElementBlock("div", {
                              key: "skill-" + item.name + "-" + item.level,
                              class: "installed-item"
                            }, [
                              vue.createElementVNode("div", _hoisted_42, [
                                vue.createVNode(SvgIcon, {
                                  name: "code",
                                  size: 18
                                })
                              ]),
                              vue.createElementVNode("div", _hoisted_43, [
                                vue.createElementVNode(
                                  "div",
                                  _hoisted_44,
                                  vue.toDisplayString(item.name),
                                  1
                                  /* TEXT */
                                ),
                                vue.createElementVNode(
                                  "div",
                                  _hoisted_45,
                                  vue.toDisplayString(item.description || "无描述"),
                                  1
                                  /* TEXT */
                                ),
                                vue.createElementVNode("span", _hoisted_46, [
                                  _cache[34] || (_cache[34] = vue.createTextVNode(
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
                              vue.createElementVNode("div", _hoisted_47, [
                                vue.createElementVNode("select", {
                                  class: "ss-status-select",
                                  value: item.status || "on",
                                  onChange: ($event) => setSkillStatus(item, $event.target.value),
                                  title: statusTitle(item)
                                }, [..._cache[35] || (_cache[35] = [
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
                                ])], 40, _hoisted_48),
                                vue.createElementVNode("button", {
                                  class: "ii-btn ii-view",
                                  onClick: ($event) => viewSkill(item),
                                  title: "查看内容"
                                }, "查看", 8, _hoisted_49),
                                item.level !== "system" ? (vue.openBlock(), vue.createElementBlock("button", {
                                  key: 0,
                                  class: "ii-btn ii-del",
                                  onClick: ($event) => delSkill(item),
                                  title: "删除"
                                }, "删除", 8, _hoisted_50)) : vue.createCommentVNode("v-if", true)
                              ])
                            ]);
                          }),
                          128
                          /* KEYED_FRAGMENT */
                        ))
                      ])) : vue.createCommentVNode("v-if", true),
                      installedMCPs.value.length === 0 && installedSkills.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_51, [
                        vue.createElementVNode("div", _hoisted_52, [
                          vue.createVNode(SvgIcon, {
                            name: "package",
                            size: 32
                          })
                        ]),
                        _cache[37] || (_cache[37] = vue.createElementVNode(
                          "div",
                          null,
                          "暂无已安装的 MCP 服务器或技能",
                          -1
                          /* CACHED */
                        )),
                        _cache[38] || (_cache[38] = vue.createElementVNode(
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
                            vue.createElementVNode("div", _hoisted_53, [
                              vue.createElementVNode(
                                "div",
                                _hoisted_54,
                                vue.toDisplayString(item.name),
                                1
                                /* TEXT */
                              ),
                              vue.createElementVNode(
                                "div",
                                _hoisted_55,
                                vue.toDisplayString(item.description),
                                1
                                /* TEXT */
                              ),
                              vue.createElementVNode("div", _hoisted_56, [
                                vue.createElementVNode(
                                  "span",
                                  {
                                    class: vue.normalizeClass(["mi-type", "type-" + item.kind])
                                  },
                                  vue.toDisplayString(item.kind === "mcp" ? "MCP" : item.kind === "plugin" ? "插件" : "技能"),
                                  3
                                  /* TEXT, CLASS */
                                ),
                                item.tags ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_57, [
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
                                item.installed ? (vue.openBlock(), vue.createElementBlock("span", _hoisted_58, [
                                  vue.createVNode(SvgIcon, {
                                    name: "check",
                                    size: 10
                                  }),
                                  _cache[39] || (_cache[39] = vue.createTextVNode(
                                    " 已安装",
                                    -1
                                    /* CACHED */
                                  ))
                                ])) : vue.createCommentVNode("v-if", true)
                              ])
                            ]),
                            !item.installed ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_59, [
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
                              ], 8, _hoisted_60)) : (vue.openBlock(), vue.createElementBlock(
                                vue.Fragment,
                                { key: 1 },
                                [
                                  vue.withDirectives(vue.createElementVNode("select", {
                                    "onUpdate:modelValue": ($event) => item._installScope = $event,
                                    class: "mi-scope-select",
                                    onClick: _cache[17] || (_cache[17] = vue.withModifiers(() => {
                                    }, ["stop"]))
                                  }, [..._cache[40] || (_cache[40] = [
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
                                  ])], 8, _hoisted_61), [
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
                                  ], 8, _hoisted_62)
                                ],
                                64
                                /* STABLE_FRAGMENT */
                              ))
                            ])) : (vue.openBlock(), vue.createElementBlock("button", {
                              key: 1,
                              class: "mi-uninstall-btn",
                              onClick: ($event) => uninstallItem(item)
                            }, [
                              vue.createVNode(SvgIcon, {
                                name: "trash",
                                size: 12
                              }),
                              _cache[41] || (_cache[41] = vue.createTextVNode(
                                " 卸载 ",
                                -1
                                /* CACHED */
                              ))
                            ], 8, _hoisted_63))
                          ]);
                        }),
                        128
                        /* KEYED_FRAGMENT */
                      )),
                      !loading.value && items.value.length === 0 ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_64, [
                        vue.createElementVNode("div", _hoisted_65, [
                          vue.createVNode(SvgIcon, {
                            name: "package",
                            size: 32
                          })
                        ]),
                        query.value ? (vue.openBlock(), vue.createElementBlock(
                          "div",
                          _hoisted_66,
                          '未找到匹配 "' + vue.toDisplayString(query.value) + '" 的条目',
                          1
                          /* TEXT */
                        )) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_67, "市场中暂无可用条目")),
                        _cache[42] || (_cache[42] = vue.createElementVNode(
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
            vue.createElementVNode("div", _hoisted_68, [
              vue.createElementVNode(
                "span",
                _hoisted_69,
                vue.toDisplayString(tab.value === "installed" ? installedMCPs.value.length + installedSkills.value.length : items.value.length) + " 个条目",
                1
                /* TEXT */
              ),
              error.value ? (vue.openBlock(), vue.createElementBlock(
                "span",
                _hoisted_70,
                vue.toDisplayString(error.value),
                1
                /* TEXT */
              )) : vue.createCommentVNode("v-if", true),
              _cache[43] || (_cache[43] = vue.createElementVNode(
                "span",
                { class: "market-tip" },
                "安装后下次对话生效",
                -1
                /* CACHED */
              )),
              vue.createElementVNode("button", {
                class: "btn-secondary",
                onClick: _cache[18] || (_cache[18] = ($event) => _ctx.$emit("close"))
              }, "关闭")
            ])
          ])
        ]);
      };
    }
  };
  const MarketplaceModal = /* @__PURE__ */ _export_sfc(_sfc_main$4, [["__scopeId", "data-v-383abd54"]]);
  const featuresMd = '# 功能介绍\n\nPairCode IDE 是一款纯 Web 端的 AI 辅助编程开发环境。你只需打开浏览器，在对话面板中用自然语言描述需求，AI 就能理解你的意图，直接生成代码、修改文件、执行命令、管理版本——把 IDE 从工具变为你的编程搭档。\n\n---\n\n## AI 对话编程\n\n**用自然语言驱动整个开发流程，就像跟资深开发者聊天一样跟 AI 交流。**\n\n在右侧对话面板中，你只需用自然语言描述需求，AI 就会理解你的意图并自动完成相应操作。无论是"帮我写一个 REST API"还是"把这个函数改成异步的"，AI 都能立刻执行。\n\n- **流式实时输出** — AI 的思考过程和操作结果实时显示，你始终能看清它在想什么、做什么\n- **透明可追溯** — 每一步操作都有详细上下文，不是黑盒\n- **随时干预** — 如果 AI 方向跑偏，可以随时给出反馈，AI 会立即调整\n\n---\n\n## 自主编程模式\n\n**AI 独立完成复杂的多步骤任务，你只需做最关键的决定。**\n\n开启自主模式后，AI 能自己分析项目结构、扫描代码问题、制定修复计划并逐个执行。你可以在关键节点审核确认，其他步骤 AI 自动完成。执行进度实时可见，你可以随时暂停、中止或补充指令。\n\n**Agent 核心采用 deepseek-harness 双层循环架构**：\n- **turn / step 双层边界** — 每次工具执行都有独立的 step 事件（开始/结束/摘要），每轮用户交互是 turn，进度颗粒度清晰可追溯\n- **inbox 双队列** — 任务转向（next-step）与后续追问（next-turn）分队列消费，多轮交互不粘连\n- **消息组装与落盘对齐** — agentloop 编号与消息序列严格一致，历史恢复与实时流状态吻合\n- **历史注入精简** — 去掉冗余前缀标注与时间戳，系统提示内置多轮规则，长对话上下文更干净\n\n---\n\n## 智能代码编辑器\n\n**内置浏览器端编辑器，让你在同一个窗口中完成所有编辑工作。**\n\n- **语法高亮** — 支持 Go、TypeScript、Python、Rust、Java、Vue、HTML、CSS 等主流语言\n- **代码折叠** — 折叠函数和代码块，聚焦关键逻辑\n- **多标签页** — 同时打开编辑多个文件，标签栏快捷切换\n- **括号匹配与自动缩进** — 代码结构清晰可见\n- **十六进制查看器** — 查看二进制文件的原始字节内容\n- **图片预览** — 在编辑器中直接显示图片文件\n\n---\n\n## 文件管理\n\n**完整的工作区文件管理能力，所有操作一目了然。**\n\n- **目录树浏览** — 以树形结构展示项目目录，支持展开 / 折叠\n- **文件操作** — 新建、编辑、保存、删除、重命名、移动文件\n- **多文件夹工作区** — 同时管理多个目录，组合成一个统一的工作区\n- **快速切换工作区** — 在最近使用的项目之间一键切换\n- **文件搜索** — 按文件名快速定位\n- **内容搜索** — 在整个工作区按关键词搜索代码内容\n\n---\n\n## Git 版本控制\n\n**在对话中完成所有 Git 操作，告别记忆复杂命令。**\n\n你只需用自然语言告诉 AI 你想做什么：\n- "查看当前仓库状态"\n- "暂存所有修改，提交信息为\'修复登录校验\'"\n- "创建一个名为 feature-search 的分支"\n- "从远程拉取最新代码"\n\nAI 会自动执行对应的 Git 操作并返回结果。你也可以通过 Git 面板查看文件变更的逐行对比。\n\n---\n\n## 内置终端\n\n**浏览器中的终端，无需切换窗口。**\n\n终端面板直接内嵌在 IDE 底部，打开即用。AI 也能自动使用终端执行命令、读取输出并分析结果。支持多标签页，方便同时运行不同任务。\n\n---\n\n## 帮助文档中心\n\n**结构化的帮助文档体系，快速找到你需要的信息。**\n\n帮助面板侧边栏按分类组织文档：\n\n| 分类 | 包含文档 |\n|------|---------|\n| **文档中心** | 快速开始、功能介绍、API 文档、工具文档、快捷键、常见问题 |\n| **其他** | 更新日志 |\n\n- **按分类导航** — 文档归入"文档中心"分组，找什么一目了然\n- **文档间跳转** — 关于面板与帮助面板之间可互相跳转\n- **翻页浏览** — 文档底部支持上一页/下一页顺序阅读\n- **搜索过滤** — 侧边栏搜索框可快速筛选文档\n\n---\n\n## API 二次开发支持\n\n**完整的 HTTP REST API + WebSocket 协议文档，支持第三方基于本 IDE 进行二次开发。**\n\n- **详细的请求/响应格式** — 每个 API 接口提供 JSON Schema 请求体、完整响应示例、字段说明和错误码\n- **WebSocket 协议定义** — 完整的 AI 事件流协议文档（15+ 事件类型、数据结构、典型事件序列）\n- **终端协议文档** — PTY WebSocket 的初始化流程、控制消息格式、白名单限制等\n- **API 索引速查表** — 按功能分类列出所有 60+ API 端点，方便快速查找\n\n所有 API 仅监听本地回环地址，安全可控。\n\n---\n\n## 代码知识图谱\n\n**AI 能理解你的代码结构和调用关系，不仅仅是搜索文本。**\n\nCodeGraph 将项目的代码结构构建成可查询的知识图谱，让 AI 理解函数之间的调用关系、类型的层次结构和文件的依赖网络。AI 可以准确找到某个函数的所有调用者、分析修改影响范围、查看完整的类型继承链。\n\n**多项目独立建图** — 在多项目工作区中，每个项目独立构建知识图谱（主项目用共享库、非主项目用各自存储），跨项目切换不串数据，工具通过 project 参数精确路由到目标项目。\n\n---\n\n## 对话历史管理\n\n**每次对话自动保存，随时回溯，不会丢失。**\n\n- 对话自动持久化到本地磁盘，刷新页面不会丢失\n- 左侧对话列表展示所有历史记录，支持继续之前的话题\n- 不同工作区的对话自动隔离，各项目互不干扰\n- 支持向前翻页加载更多历史消息\n\n---\n\n## BUG 自动检测与修复\n\n**AI 主动扫描代码问题并生成修复方案，反复验证直到全部通过。**\n\n- 自动运行编译检查和测试，标记所有错误位置\n- 分析错误根因，生成具体的修复方案\n- 修复后再次验证，支持多轮迭代\n- 修复前会展示改动内容，你可以审阅确认\n\n---\n\n## Skills / MCP / 工具集扩展\n\n**通过扩展增强 AI 的能力，让 IDE 更贴合你的工作流。**\n\n- **Skills（技能）** — 可复用的工作流程模板，AI 在对应场景中自动加载使用\n- **MCP（模型上下文协议）** — 标准化的工具扩展协议，可为 AI 添加自定义能力（如查询内部数据库、调用第三方 API）\n- **工具集（Toolset）** — 按项目需求组合的插件包，动态构建并固化到工作区，可导出/导入/发布市场\n- **内置市场** — 一键浏览和安装社区贡献的扩展（技能 / MCP / 插件工具集三类）\n\n---\n\n## 插件化自定义工具\n\n**通过 JS / TS / Go / Lua 插件扩展 AI 的工具集，一切皆插件。**\n\nPairCode IDE 的工具体系全部插件化——内置功能（文件/搜索/Git/Web/记忆/任务/图谱等 21 组）以插件形态装配，你也可以编写自己的插件扩展能力：\n\n- **JS / TS 插件** — 通过 `cordis_define` 定义函数形态插件，支持 `apply(ctx, config)` 注入服务、timer 定时器、跨 goroutine 执行锁；TS 插件由内置编译器（esbuild 纯 Go）直接转译加载，无需 Node.js\n- **Go 插件** — 内置插件框架，核心功能组全部以 Go 插件装配，`cordis_inspect` 可查看工具归属，`cordis_stop` 卸载整组\n- **Lua 工具** — 支持 Lua 脚本自定义工具，封装常用命令组合与自定义数据处理逻辑\n- **沙箱防护** — VM 超时防护、schema 校验，插件异常不影响主进程\n\n## 工具集生态\n\n**按项目需求动态组合工具集，固化到工作区，可导出分享。**\n\n- **动态构建** — 描述你的项目需求（如"Go 后端 + 前端调试"），AI 分析项目结构后自动组合所需工具并创建工具集插件\n- **固化与重建** — 工具集固化到 `.pair/toolsets/`，随项目走；显式调用可更新重建\n- **导出 / 导入 / 市场** — 工具集可导出为 JSON 分享，或发布到市场供他人一键安装（project/user 两种范围）\n- **LLM 意图分析** — 分析项目目的时由 LLM 参与理解（语言无关，不固化任何语言模板），跨语言项目同样适用\n\n---\n\n## 项目知识库\n\n**把项目架构、模块职责和设计决策沉淀成结构化知识库，AI 跨会话持续了解你的项目。**\n\n- **树形分支组织** — 知识按 目标 / 架构 / 实现 / 关键点 / 设计思想 分类，深挖有细节、浏览有全貌\n- **跨会话记忆** — AI 每次接手项目自动加载知识库导航，无需从零分析项目\n- **团队共享** — 知识库存入项目 `.pair/` 目录，随项目版本控制，团队协作时信息不丢失\n- **过期检测** — 自动验证知识条目引用的文件/目录是否存在，失效条目提示清理\n- **AGENTS.md 分层** — 项目说明、环境配置、开发指南分层管理，.agents 路径兼容\n\n## 记忆系统\n\n**AI 能跨会话记住你的偏好和项目决策。**\n\nAI 会记住你的编码偏好、经常使用的模式和做过的决策。下次打开 IDE 时，AI 会自动引用这些记忆，无需重复说明。记忆可搜索、可管理。\n\n---\n\n## 任务与规划管理\n\n**复杂的多步骤开发任务有条不紊地执行。**\n\nAI 会自动分解复杂任务为可追踪的子任务步骤，每步的执行状态和结果清晰可见。支持依赖关系管理，任务清单持久化，重启不会丢失。\n\n---\n\n## 主题与个性化\n\n**按照你的喜好定制 IDE 外观。**\n\n- **四套预设主题** — 暗色科技风、白色简约风、暖色温暖风、暗夜紫风格\n- **即时切换** — 切换主题立即生效，无需刷新\n- **统一字体方案** — 界面字体和代码字体分别配置\n\n---\n\n## 多模型支持\n\n**灵活选择 AI 模型后端。**\n\n支持 OpenAI、Claude 等主流 AI 服务商。可为"执行任务"和"制定规划"分别配置不同的模型。所有模型配置在设置面板中集中管理，支持自定义 API 地址。\n\n---\n\n## 安全设计\n\n**你的代码和数据始终在你的控制之下。**\n\n- **本地运行** — 所有操作在本地计算机执行，不经过第三方云端\n- **路径隔离** — 文件操作限定在工作区目录范围内\n- **审批机制** — 写文件和执行命令等敏感操作需你确认\n- **本地地址** — IDE 服务仅监听本地回环地址，默认不对外暴露\n\n---\n\n## 操作界面速览\n\n| 区域 | 说明 |\n|------|------|\n| **标题栏** | 顶部菜单栏，提供帮助文档、设置等入口 |\n| **活动栏** | 左侧图标栏，切换文件浏览、搜索、Git 等功能面板 |\n| **侧栏** | 文件树、搜索面板、Git 面板等工具区域 |\n| **主编辑区** | 代码编辑区域，支持多标签页切换 |\n| **对话面板** | 右侧 AI 对话区域，与 AI 交流的核心界面 |\n| **状态栏** | 底部状态信息，显示文件编码、行号列号 |\n| **终端面板** | 底部内置终端，执行命令和脚本 |\n\n---\n\n## 快捷键一览\n\n| 快捷键 | 功能 |\n|--------|------|\n| Ctrl+S | 保存当前文件 |\n| Ctrl+B | 切换侧栏显示 |\n| Ctrl+\\` | 切换终端面板 |\n| Ctrl+K | 专注模式（隐藏所有面板） |\n| Ctrl+Shift+E | 切换到文件浏览器 |\n| Ctrl+Shift+F | 全局搜索 |\n| Ctrl+Shift+T | 打开对话面板 |\n| Ctrl+Shift+C | 切换对话面板 |\n';
  const apiDocsMd = '# API 文档\r\n\r\nPairCode IDE 内置了一套完整的 HTTP REST API + WebSocket 实时通信协议，供 Web 前端与后端核心功能交互，也**支持第三方开发者基于本 API 进行二次开发**。所有 API 地址均以 `/api` 开头，返回 JSON 格式数据。\r\n\r\n> **安全提示**：所有 API 仅监听本地回环地址（127.0.0.1），默认不对外暴露。请勿将服务端口暴露到公网或局域网。\r\n\r\n---\r\n\r\n## 通用约定\r\n\r\n### 请求格式\r\n- 查询参数（GET）直接在 URL 中传递\r\n- POST / PUT 请求体使用 `application/json`\r\n- 无特殊说明时，Content-Type 为 `application/json`\r\n\r\n### 响应格式\r\n| 场景 | 格式 | 说明 |\r\n|------|------|------|\r\n| 成功 | JSON 对象 或 JSON 数组 | 直接返回业务数据 |\r\n| 错误 | `{"error": "错误描述信息"}` | HTTP 状态码 4xx/5xx |\r\n\r\n### 错误码惯例\r\n| HTTP 状态码 | 含义 |\r\n|-------------|------|\r\n| 200 | 成功 |\r\n| 400 | 参数错误 / 请求体错误 |\r\n| 404 | 资源不存在 |\r\n| 405 | 方法不允许（如 GET 用了 POST） |\r\n| 500 | 服务器内部错误 |\r\n\r\n---\r\n\r\n## 一、服务健康检查\r\n\r\n检查 IDE 后端服务是否正常运行。\r\n\r\n```\r\nGET /api/health\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "status": "ok",\r\n  "workspace": "F:/projects/my-app",\r\n  "folders": ["F:/projects/my-app"]\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| status | string | 固定 `"ok"` |\r\n| workspace | string | 当前工作区路径 |\r\n| folders | string[] | 工作区包含的文件夹列表 |\r\n\r\n---\r\n\r\n## 二、文件系统操作\r\n\r\n浏览、读写和管理工作区内的文件与目录。\r\n\r\n### 2.1 列出目录\r\n\r\n```\r\nGET /api/fs/list?path={目录路径}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 目录路径，省略时返回工作区根目录 |\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {"name": "src", "isDir": true, "size": 4096, "modTime": "2026-07-11T10:00:00Z"},\r\n  {"name": "main.go", "isDir": false, "size": 2048, "modTime": "2026-07-11T09:30:00Z"}\r\n]\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| name | string | 文件/目录名 |\r\n| isDir | boolean | 是否为目录 |\r\n| size | number | 文件大小（字节） |\r\n| modTime | string | 最后修改时间（ISO 8601） |\r\n\r\n---\r\n\r\n### 2.2 读取文件\r\n\r\n```\r\nGET /api/fs/read?path={文件路径}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 是 | 文件路径 |\r\n\r\n**响应：** 返回文件文本内容（字符串）。\r\n\r\n---\r\n\r\n### 2.3 写入文件\r\n\r\n```\r\nPOST /api/fs/write\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "src/main.go",\r\n  "content": "package main\\n\\nfunc main() {\\n\\tprintln(\\"hello\\")\\n}\\n"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 是 | 文件路径（相对于工作区或绝对路径） |\r\n| content | string | 是 | 文件内容（覆盖写入，自动创建目录） |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 2.4 搜索文件内容\r\n\r\n```\r\nGET /api/fs/search?q={关键词}&path={搜索路径}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| q | string | 是 | 搜索关键词 |\r\n| path | string | 否 | 搜索目录，省略时使用工作区根目录 |\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {"file": "src/main.go", "line": 15, "text": "func handleRequest(w http.ResponseWriter, r *http.Request) {"},\r\n  {"file": "src/utils.go", "line": 42, "text": "// handleRequest 处理 HTTP 请求"}\r\n]\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| file | string | 文件相对路径 |\r\n| line | number | 行号 |\r\n| text | string | 匹配行的内容 |\r\n\r\n**自动忽略的目录：** `.git`、`node_modules`、`vendor`、`.pair`、`__pycache__`、`bin` 等。**仅搜索文本文件扩展名**（`.go` `.js` `.ts` `.vue` `.html` `.css` `.json` `.md` `.py` `.rs` `.java` 等 50+ 种）。\r\n\r\n---\r\n\r\n### 2.5 重命名/移动文件\r\n\r\n```\r\nPOST /api/fs/rename\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "oldPath": "src/old.go",\r\n  "newPath": "src/new.go"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| oldPath | string | 是 | 原路径 |\r\n| newPath | string | 是 | 新路径 |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 2.6 删除文件/目录\r\n\r\n```\r\nPOST /api/fs/delete\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "src/temp.go"\r\n}\r\n```\r\n\r\n> ⚠️ 不可恢复，递归删除目录及其所有内容。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 2.7 创建目录\r\n\r\n```\r\nPOST /api/fs/mkdir\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "src/new-folder"\r\n}\r\n```\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 2.8 获取图片数据\r\n\r\n```\r\nGET /api/fs/image?path={图片路径}\r\n```\r\n\r\n**参数：** `path` — 图片文件路径（支持 PNG / JPEG）\r\n\r\n**响应：** Base64 编码的图片数据字符串（不含 `data:image/...` 前缀）。\r\n\r\n**响应头：** `Content-Type: text/plain; charset=utf-8`\r\n\r\n---\r\n\r\n### 2.9 获取文件信息\r\n\r\n```\r\nGET /api/fs/file-info?path={文件路径}\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "name": "main.go",\r\n  "path": "F:/projects/my-app/src/main.go",\r\n  "size": 2048,\r\n  "modTime": "2026-07-11T09:30:00Z",\r\n  "isDir": false\r\n}\r\n```\r\n\r\n---\r\n\r\n### 2.10 十六进制查看\r\n\r\n```\r\nGET /api/fs/hex?path={文件路径}&offset={偏移}&length={长度}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 是 | 文件路径 |\r\n| offset | number | 否 | 起始字节偏移（默认 0） |\r\n| length | number | 否 | 读取字节数（默认 512，最大 4096） |\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "hex": "4d5a90000300000004000000ffff0000b80000000000000040",\r\n  "text": "MZ.............@",\r\n  "offset": 0,\r\n  "length": 32\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| hex | string | 十六进制字符串 |\r\n| text | string | ASCII 可打印字符（不可打印的替换为 `.`） |\r\n| offset | number | 起始偏移 |\r\n| length | number | 返回的字节数 |\r\n\r\n---\r\n\r\n### 2.11 列出磁盘驱动器\r\n\r\n```\r\nGET /api/fs/drives\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n["C:\\\\", "D:\\\\", "E:\\\\"]\r\n```\r\n\r\n---\r\n\r\n## 三、工作区管理\r\n\r\n### 3.1 获取当前工作区\r\n\r\n```\r\nGET /api/workspace\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "root": "F:/projects/my-app",\r\n  "folders": ["F:/projects/my-app"],\r\n  "loaded": true\r\n}\r\n```\r\n\r\n### 3.2 切换/设置工作区\r\n\r\n```\r\nPOST /api/workspace\r\n```\r\n\r\n**请求体（切换工作区）：**\r\n```json\r\n{\r\n  "path": "F:/projects/another-project"\r\n}\r\n```\r\n\r\n**请求体（添加文件夹）：**\r\n```json\r\n{\r\n  "addFolder": "F:/projects/shared-lib"\r\n}\r\n```\r\n\r\n**请求体（创建新工作区）：**\r\n```json\r\n{\r\n  "create": "F:/projects/new-project"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 按场景 | 切换工作区到指定路径 |\r\n| addFolder | string | 按场景 | 在当前工作区添加文件夹 |\r\n| create | string | 按场景 | 创建新目录并切换为其工作区 |\r\n\r\n**响应：** 返回更新后的工作区信息（同 GET 响应格式）。\r\n\r\n---\r\n\r\n## 四、设置管理\r\n\r\n### 4.1 读取设置\r\n\r\n```\r\nGET /api/settings\r\n```\r\n\r\n**响应：** 返回完整 `AppSettings` 对象（字段较多，按需取用）：\r\n\r\n```json\r\n{\r\n  "provider": "deepseek",\r\n  "baseURL": "https://api.deepseek.com/v1",\r\n  "apiKey": "sk-xxx",\r\n  "planModel": "deepseek-v4-pro",\r\n  "executeModel": "deepseek-v4-flash",\r\n  "reviewModel": "deepseek-v4-pro",\r\n  "temperature": "0.3",\r\n  "thinkingMode": "thinking",\r\n  "maxTokens": 131072,\r\n  "contextMaxTokens": 64000,\r\n  "lastProject": "F:/projects/my-app",\r\n  "workspaceFolders": ["F:/projects/my-app"],\r\n  "recentProjects": ["F:/projects/app1"],\r\n  "reviewMode": "auto",\r\n  "reviewBlacklist": [],\r\n  "reviewWhitelist": [],\r\n  "autonomous": false,\r\n  "autoCollapse": true,\r\n  "maxIterations": 50,\r\n  "maxParallelAgents": 3,\r\n  "maxReviewRetries": 3,\r\n  "autoIterateOnRejection": true,\r\n  "requireHumanApprovalForDestructive": true,\r\n  "aiReview": false,\r\n  "autoCommit": true,\r\n  "luaTools": true,\r\n  "enableBenchmarking": true,\r\n  "systemInstructions": "",\r\n  "searxngUrl": "",\r\n  "ignoreDirs": [],\r\n  "defaultShell": "auto",\r\n  "termFontSize": 13,\r\n  "termEncoding": "auto",\r\n  "theme": "dark",\r\n  "fontFamily": "\'Cascadia Code\', Consolas, monospace",\r\n  "editorFontSize": 14,\r\n  "tabSize": 2,\r\n  "wordWrap": false,\r\n  "hideMinimap": false,\r\n  "autoConnectMCP": true,\r\n  "skillEnabledOverrides": {},\r\n  "skillStatusOverrides": {},\r\n  "mcpEnabledOverrides": {},\r\n  "customProviders": []\r\n}\r\n```\r\n\r\n### 4.2 保存设置\r\n\r\n```\r\nPUT /api/settings?convId={对话ID}\r\n```\r\n\r\n**请求体：** 与 GET 返回格式相同，只需传入要修改的字段（增量合并，未传字段保持不变）。\r\n\r\n**参数：** `convId` — 可选，当前对话 ID。当 `reviewMode` 字段变更时，实时更新该对话的 Loop 审核模式。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 五、系统工具\r\n\r\n### 5.1 系统信息\r\n\r\n```\r\nGET /api/system/info\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "hostname": "DESKTOP-ABC123",\r\n  "cwd": "F:/projects/my-app",\r\n  "os": "windows",\r\n  "goos": "windows",\r\n  "workspace": "F:/projects/my-app",\r\n  "folders": ["F:/projects/my-app"],\r\n  "version": "v1.1.2"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| hostname | string | 主机名 |\r\n| cwd | string | 当前工作目录 |\r\n| os | string | 操作系统名称 |\r\n| goos | string | Go 平台标识 |\r\n| workspace | string | IDE 工作区根路径 |\r\n| folders | string[] | 工作区文件夹列表 |\r\n| version | string | IDE 版本号（由打包器注入） |\r\n\r\n### 5.2 执行命令\r\n\r\n```\r\nPOST /api/system/exec\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "command": "go build ./cmd/app",\r\n  "cwd": "F:/projects/my-app"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| command | string | 是 | 要执行的命令 |\r\n| cwd | string | 否 | 工作目录（默认工作区根目录） |\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "stdout": "# github.com/foo/app\\nsrc/main.go:42: undefined: x\\n",\r\n  "stderr": "",\r\n  "exitCode": 2\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| stdout | string | 标准输出 |\r\n| stderr | string | 标准错误 |\r\n| exitCode | number | 退出码（0 = 成功） |\r\n\r\n> **安全限制：** 命令在工作区目录下执行；禁止交互式命令（如 `vim`）。\r\n\r\n---\r\n\r\n## 六、AI 模型\r\n\r\n### 获取可用模型列表\r\n\r\n```\r\nGET /api/models\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "providers": [\r\n    {\r\n      "name": "openai",\r\n      "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"]\r\n    },\r\n    {\r\n      "name": "claude",\r\n      "models": ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"]\r\n    }\r\n  ],\r\n  "current": {\r\n    "provider": "openai",\r\n    "model": "gpt-4"\r\n  }\r\n}\r\n```\r\n\r\n---\r\n\r\n## 七、对话管理\r\n\r\n### 7.1 对话列表\r\n\r\n```\r\nGET /api/conversations?workspace={工作区路径}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| workspace | string | 否 | 工作区路径，省略时使用当前工作区 |\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {\r\n    "id": "conv_1741680000000",\r\n    "title": "修复登录页面样式",\r\n    "createdAt": "2026-07-11T10:00:00Z",\r\n    "messageCount": 12,\r\n    "workspace": "F:/projects/my-app"\r\n  }\r\n]\r\n```\r\n\r\n### 7.2 创建对话\r\n\r\n```\r\nPOST /api/conversations\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "title": "新对话",\r\n  "workspace": "F:/projects/my-app"\r\n}\r\n```\r\n\r\n**响应：** 返回创建的对话对象（同 GET 列表中的格式）。\r\n\r\n### 7.3 获取对话详情（含消息）\r\n\r\n```\r\nGET /api/conversations/{convId}\r\n```\r\n\r\n**响应：** 返回该对话的最近 50 条消息：\r\n\r\n```json\r\n{\r\n  "messages": [\r\n    {"role": "user", "content": "帮我写一个 HTTP 服务", "createdAt": "2026-07-11T10:00:00Z"},\r\n    {"role": "assistant", "content": "好的，我来创建...", "createdAt": "2026-07-11T10:00:05Z"}\r\n  ],\r\n  "total": 42\r\n}\r\n```\r\n\r\n### 7.4 更新对话\r\n\r\n```\r\nPUT /api/conversations/{convId}\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "title": "新的标题"\r\n}\r\n```\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 7.5 删除对话\r\n\r\n```\r\nDELETE /api/conversations/{convId}\r\n```\r\n\r\n**响应：** `{"ok": true}`（同时删除该对话的所有消息）。\r\n\r\n### 7.6 获取消息列表（分页）\r\n\r\n```\r\nGET /api/conversations/{convId}/messages?limit={数量}&before={索引}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| limit | number | 否 | 返回消息条数（默认 50） |\r\n| before | number | 否 | 从消息索引 before 处开始往前加载（用于分页翻历史） |\r\n\r\n**响应：**\r\n```json\r\n{\r\n  "messages": [\r\n    {"role": "user", "content": "第一条消息", "createdAt": "..."},\r\n    {"role": "assistant", "content": "回复", "createdAt": "..."}\r\n  ],\r\n  "total": 42\r\n}\r\n```\r\n\r\n> 连续的 assistant 消息会被合并（`MergeConsecutiveAssistants`）。\r\n\r\n### 7.7 添加消息\r\n\r\n```\r\nPOST /api/conversations/{convId}/messages\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "role": "user",\r\n  "content": "继续上一个话题"\r\n}\r\n```\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 7.8 消息总数\r\n\r\n```\r\nGET /api/conversations/{convId}/messages/count\r\n```\r\n\r\n**响应：** `{"count": 42}`\r\n\r\n### 7.9 发送消息给 AI（非阻塞）\r\n\r\n```\r\nPOST /api/chat/send\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "message": "帮我创建一个 Go HTTP 服务",\r\n  "sessionId": "sess_xxx",\r\n  "convId": "conv_1741680000000",\r\n  "autonomous": false,\r\n  "workspaceRoot": "F:/projects/my-app"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| message | string | 是 | 用户消息内容（最长 50000 字符，超出截断） |\r\n| sessionId | string | 否 | 会话 ID |\r\n| convId | string | 否 | 对话 ID（留空则自动生成 `conv_{时间戳}`） |\r\n| autonomous | boolean | 否 | 是否启用自主模式（默认 false） |\r\n| workspaceRoot | string | 否 | 工作区路径（默认当前工作区） |\r\n\r\n**响应：** `{"sessionId": "sess_xxx", "convId": "conv_1741680000000"}`\r\n\r\nAI 的回复不在此响应的 Body 中返回，而是通过 **WebSocket 实时推送**事件流（见第十七章）。\r\n\r\n**前置条件：** 必须先配置 API Key 和模型。\r\n\r\n---\r\n\r\n### 7.10 停止 AI 响应\r\n\r\n```\r\nPOST /api/chat/stop?convId={对话ID}\r\n```\r\n\r\n**参数：** `convId` — 要停止的对话 ID。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 7.11 审批操作\r\n\r\n```\r\nPOST /api/chat/approve\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "convId": "conv_xxx",\r\n  "approved": true,\r\n  "reply": "请把函数名改为驼峰命名法"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| convId | string | 是 | 对话 ID |\r\n| approved | boolean | 是 | 批准（true）或拒绝（false） |\r\n| reply | string | 否 | 拒绝时的反馈/纠正建议 |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 7.12 发送运行时反馈\r\n\r\n```\r\nPOST /api/chat/feedback\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "convId": "conv_xxx",\r\n  "feedback": "请改用更简洁的实现方式"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| convId | string | 是 | 对话 ID |\r\n| feedback | string | 是 | 反馈/纠正内容 |\r\n\r\n**工作原理：** 在 AI 下次 LLM 调用前，将反馈内容作为用户消息注入本轮上下文，让 AI 在下一次回复中响应用户的补充或纠正。\r\n\r\n---\r\n\r\n### 7.13 回答 ask_user 提问\r\n\r\n```\r\nPOST /api/chat/answer\r\n```\r\n\r\n当 AI 通过 `ask_user` 工具向用户提问时，用此接口发送回答。\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "convId": "conv_xxx",\r\n  "answer": "用 POST 方法"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| convId | string | 是 | 对话 ID |\r\n| answer | string | 是 | 用户的回答 |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n### 7.14 回滚消息\r\n\r\n```\r\nPOST /api/chat/rollback\r\n```\r\n\r\n回滚到指定用户消息之前的状态：恢复该消息关联的所有文件快照，并删除该消息之后的对话历史。\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "convId": "conv_xxx",\r\n  "msgIdx": 3\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| convId | string | 是 | 对话 ID |\r\n| msgIdx | number | 是 | 用户消息索引（0 基），回滚到此消息之前 |\r\n\r\n**响应：** `{"ok": true, "msgIdx": 3}`\r\n\r\n---\r\n\r\n### 7.15 压缩上下文\r\n\r\n```\r\nPOST /api/chat/compact?convId={对话ID}\r\n```\r\n\r\n手动触发上下文压缩：将对话中间部分的老消息压缩为摘要，释放 token 预算。\r\n\r\n**参数：** `convId` — 对话 ID。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 八、指令与思想\r\n\r\n### 8.1 读取指令\r\n\r\n```\r\nGET /api/instructions?scope={作用域}\r\n```\r\n\r\n**参数：** `scope` — 指令作用域（如 `"system"`、`"user"`）。\r\n\r\n**响应：** 返回指令文本内容（字符串）。\r\n\r\n### 8.2 保存指令\r\n\r\n```\r\nPUT /api/instructions?scope={作用域}\r\n```\r\n\r\n**请求体：** 纯文本字符串（指令内容）。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 8.3 读取行为指导\r\n\r\n```\r\n```\r\n\r\n**响应：** 返回 AI 行为指导配置文本。\r\n\r\n### 8.4 保存行为指导\r\n\r\n```\r\n```\r\n\r\n**请求体：** 纯文本字符串。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 九、任务与规划\r\n\r\n> **注意：** 任务由 Agent 通过 `update_tasks` / `update_plan` 工具自主管理。以下 API 仅提供前端只读查询接口。\r\n\r\n### 9.1 获取任务列表\r\n\r\n```\r\nGET /api/tasks?convId={对话ID}\r\n```\r\n\r\n**参数：** `convId` — 可选，过滤指定对话的任务。\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "tasks": [\r\n    {\r\n      "step": "创建 HTTP 服务文件",\r\n      "status": "completed",\r\n      "taskId": "task_1",\r\n      "description": "在 src/server.go 创建 HTTP 服务",\r\n      "created_at": "2026-07-11T10:00:00Z"\r\n    }\r\n  ]\r\n}\r\n```\r\n\r\n> 任务数据持久化在工作区 `.pair/tasks/*.json`，由 Agent 的 `update_tasks` 工具写入。\r\n\r\n### 9.2 读取任务规划文档\r\n\r\n```\r\nGET /api/taskplan?name={规划名}\r\n```\r\n\r\n列出或读取 Markdown 格式的规划文档。\r\n\r\n**参数：** `name` — 可选，指定规划文档名（不含 `.md` 后缀）；省略则返回所有规划文档列表。\r\n\r\n**GET 响应（列出全部）：**\r\n```json\r\n[\r\n  {"name": "refactor-auth", "file": "F:/projects/.pair/tasks/refactor-auth.md"}\r\n]\r\n```\r\n\r\n**GET 响应（读单个）：**\r\n```json\r\n{\r\n  "name": "refactor-auth",\r\n  "content": "## 重构计划\\n1. 提取认证中间件\\n2. 添加 JWT 支持"\r\n}\r\n```\r\n\r\n### 9.3 追加/完成规划文档\r\n\r\n```\r\nPOST /api/taskplan\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "name": "refactor-auth",\r\n  "content": "- 完成 JWT 集成",\r\n  "action": "append"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| name | string | 否 | 规划名称（省略则自动生成 `plan_日期时间`） |\r\n| content | string | 是 | 要追加的内容（Markdown） |\r\n| action | string | 否 | `"append"`（追加）或 `"complete"`（追加"[已完成] 时间戳"），默认 `"append"` |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 十、Git 版本控制\r\n\r\n所有 Git API 均在**当前工作区目录**（或指定仓库路径）下执行。\r\n\r\n### 10.1 初始化仓库\r\n\r\n```\r\nPOST /api/git/init?path={目录路径}\r\n```\r\n\r\n**参数：** `path` — 目标目录（默认当前工作区）。\r\n\r\n**响应：** `{"output": "Initialized empty Git repository in ..."}`\r\n\r\n---\r\n\r\n### 10.2 仓库状态\r\n\r\n```\r\nGET /api/git/status?path={仓库路径}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 仓库路径（默认当前工作区） |\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "branch": "main",\r\n  "changes": [\r\n    {"path": "src/main.go", "status": "M", "staged": false},\r\n    {"path": "src/utils.go", "status": "M", "staged": true}\r\n  ],\r\n  "untracked": ["src/new.go"],\r\n  "ahead": 1,\r\n  "behind": 0\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| branch | string | 当前分支名 |\r\n| changes[].path | string | 变更文件路径 |\r\n| changes[].status | string | 状态码：`M`(修改) `A`(新增) `D`(删除) `R`(重命名) |\r\n| changes[].staged | boolean | 是否已暂存 |\r\n| untracked | string[] | 未跟踪文件列表 |\r\n| ahead | number | 领先远程的提交数 |\r\n| behind | number | 落后远程的提交数 |\r\n\r\n### 10.3 查看差异\r\n\r\n```\r\nGET /api/git/diff?path={仓库路径}&file={文件路径}&staged={是否暂存}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 仓库路径 |\r\n| file | string | 否 | 指定文件（省略则返回所有变更的 diff） |\r\n| staged | string | 否 | `"true"` = 只显示已暂存差异（--cached） |\r\n\r\n**响应：** 返回 diff 文本（字符串）。\r\n\r\n### 10.4 暂存文件\r\n\r\n```\r\nPOST /api/git/add\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "F:/projects/my-app",\r\n  "files": ["src/main.go", "src/utils.go"]\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 仓库路径（默认工作区） |\r\n| files | string[] | 否 | 要暂存的文件列表（省略则暂存全部 `-A`） |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.5 取消暂存\r\n\r\n```\r\nPOST /api/git/reset\r\n```\r\n\r\n**请求体：** 格式同 `git/add`。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.6 提交\r\n\r\n```\r\nPOST /api/git/commit\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "F:/projects/my-app",\r\n  "message": "feat: 添加用户认证模块"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 仓库路径 |\r\n| message | string | 是 | 提交信息 |\r\n\r\n**响应：**\r\n```json\r\n{\r\n  "ok": true,\r\n  "hash": "a1b2c3d4e5f6..."\r\n}\r\n```\r\n\r\n### 10.7 查看提交历史\r\n\r\n```\r\nGET /api/git/log?path={仓库路径}&count={数量}&file={文件路径}\r\n```\r\n\r\n> **别名：** `/api/git-log`（绕过部分浏览器广告拦截器对 `/api/git/log` 的误杀）。\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 仓库路径 |\r\n| count | number | 否 | 返回条数（默认 15） |\r\n| file | string | 否 | 限定某文件的提交历史 |\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {\r\n    "hash": "a1b2c3d",\r\n    "author": "user",\r\n    "date": "2026-07-11 10:00:00",\r\n    "message": "feat: 添加用户认证模块"\r\n  }\r\n]\r\n```\r\n\r\n### 10.8 分支管理\r\n\r\n```\r\nPOST /api/git/branch\r\n```\r\n\r\n| 操作 | 请求体 | 说明 |\r\n|------|--------|------|\r\n| 创建 | `{"path":"...","name":"feature-x","action":"create"}` | 创建新分支 |\r\n| 删除 | `{"path":"...","name":"feature-x","action":"delete"}` | 删除分支 |\r\n| 列表 | `{"path":"...","action":"list"}` | 列出所有分支 |\r\n| 切换 | `{"path":"...","name":"feature-x","action":"checkout"}` | 切换分支 |\r\n\r\n**响应：** 列表操作返回 `["main", "feature-x", ...]`，其他返回 `{"ok": true}`。\r\n\r\n### 10.9 切换分支 / 恢复文件\r\n\r\n```\r\nPOST /api/git/checkout\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "F:/projects/my-app",\r\n  "branch": "feature-x",\r\n  "file": "src/main.go"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| branch | string | 按场景 | 切换到的分支名 |\r\n| file | string | 按场景 | 恢复指定文件到 HEAD（branch 和 file 二选一） |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.10 贮藏\r\n\r\n```\r\nPOST /api/git/stash\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "F:/projects/my-app",\r\n  "action": "push",\r\n  "message": "暂存当前 WIP"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| path | string | 否 | 仓库路径 |\r\n| action | string | 否 | `"push"`(贮藏,默认) \\| `"pop"`(恢复) \\| `"apply"`(应用) \\| `"drop"`(丢弃) |\r\n| message | string | 否 | 贮藏备注 |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.11 查看贮藏列表\r\n\r\n```\r\nGET /api/git/stash-list?path={仓库路径}\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {"index": 0, "message": "暂存当前 WIP"},\r\n  {"index": 1, "message": "On feature-x: 临时保存"}\r\n]\r\n```\r\n\r\n### 10.12 管理 `.gitignore`\r\n\r\n```\r\nGET /api/git/ignore?path={仓库路径}\r\nPOST /api/git/ignore?path={仓库路径}\r\n```\r\n\r\n**GET 响应：** 返回当前 `.gitignore` 内容：\r\n```json\r\n{\r\n  "content": "*.log\\n.env\\nbuild/",\r\n  "rules": ["*.log", ".env", "build/"]\r\n}\r\n```\r\n\r\n**POST 请求体（覆盖写入）：**\r\n```json\r\n{\r\n  "content": "*.log\\n.env\\nnode_modules/"\r\n}\r\n```\r\n\r\n**POST 请求体（追加一行）：**\r\n```json\r\n{\r\n  "append": "dist/"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| content | string | 按场景 | 完整覆盖 `.gitignore` 内容 |\r\n| append | string | 按场景 | 追加一行到 `.gitignore`（content 和 append 二选一） |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.13 丢弃修改\r\n\r\n```\r\nPOST /api/git/discard\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "F:/projects/my-app",\r\n  "files": ["src/main.go"]\r\n}\r\n```\r\n\r\n> ⚠️ 不可恢复！丢弃工作区未暂存的修改。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.14 推送\r\n\r\n```\r\nPOST /api/git/push\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "path": "F:/projects/my-app",\r\n  "remote": "origin",\r\n  "branch": "main"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| remote | string | 否 | 远程名（默认 `"origin"`） |\r\n| branch | string | 否 | 分支名（默认当前分支） |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.15 拉取\r\n\r\n```\r\nPOST /api/git/pull\r\n```\r\n\r\n**请求体：** 同 `git/push`。\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 10.16 远程仓库管理\r\n\r\n```\r\nGET /api/git/remote?path={仓库路径}\r\nPOST /api/git/remote?path={仓库路径}\r\n```\r\n\r\n**GET 响应示例：**\r\n```json\r\n[\r\n  {"name": "origin", "url": "https://github.com/user/repo.git"}\r\n]\r\n```\r\n\r\n**POST 请求体：**\r\n```json\r\n{\r\n  "name": "upstream",\r\n  "url": "https://github.com/other/repo.git",\r\n  "action": "add"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| name | string | 是 | 远程名 |\r\n| url | string | 是 | 远程 URL |\r\n| action | string | 否 | `"add"`（添加）或 `"remove"`（删除），默认 `"add"` |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 十一、Skills 技能\r\n\r\n### 11.1 技能列表\r\n\r\n```\r\nGET /api/skills/list\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {\r\n    "name": "code-review",\r\n    "description": "代码审查工作流",\r\n    "mode": "auto",\r\n    "version": "1.0"\r\n  }\r\n]\r\n```\r\n\r\n### 11.2 读取技能\r\n\r\n```\r\nGET /api/skills/read?name={技能名}&level={层级}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| name | string | 是 | 技能名 |\r\n| level | string | 否 | `"system"`（全局）或 `"project"`（项目，默认） |\r\n\r\n**响应：** 返回技能的完整 Markdown 内容。\r\n\r\n### 11.3 保存/更新技能状态\r\n\r\n```\r\nPOST /api/skills/save\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "name": "code-review",\r\n  "level": "project",\r\n  "action": "set-status",\r\n  "status": "on"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| name | string | 是 | 技能名 |\r\n| level | string | 否 | `"system"` / `"project"`（默认 project） |\r\n| action | string | 是 | 固定 `"set-status"` |\r\n| status | string | 是 | `"off"` \\| `"on"` \\| `"max"` |\r\n\r\n**响应：** `{"ok": true, "action": "set-status", "name": "code-review", "status": "on"}`\r\n\r\n### 11.4 删除技能\r\n\r\n```\r\nPOST /api/skills/delete\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "name": "code-review"\r\n}\r\n```\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 十二、MCP 扩展\r\n\r\n### 12.1 MCP 列表\r\n\r\n```\r\nGET /api/mcp/list?level={层级}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| level | string | 否 | 层级过滤（`"user"`、`"project"`） |\r\n\r\n### 12.2 MCP 保存/管理\r\n\r\n```\r\nPOST /api/mcp/save\r\n```\r\n\r\n统一管理 MCP 的添加、更新、删除和启用切换。\r\n\r\n**请求体（添加/更新）：**\r\n```json\r\n{\r\n  "name": "my-db",\r\n  "command": "node",\r\n  "args": ["mcp-server-db/index.js"],\r\n  "level": "project"\r\n}\r\n```\r\n\r\n**请求体（删除）：**\r\n```json\r\n{\r\n  "action": "delete",\r\n  "name": "my-db",\r\n  "level": "project"\r\n}\r\n```\r\n\r\n**请求体（启用/禁用切换）：**\r\n```json\r\n{\r\n  "action": "toggle",\r\n  "name": "my-db",\r\n  "level": "project"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| action | string | 否 | `"delete"`（删除）\\| `"toggle"`（启用切换），省略则为新增/更新 |\r\n| name | string | 是 | MCP 名称 |\r\n| command | string | 新增时必填 | 启动命令 |\r\n| args | string[] | 否 | 命令参数 |\r\n| level | string | 否 | `"user"`（用户级）\\| `"project"`（项目级），默认 user |\r\n\r\n**响应：** `{"ok": true, "action": "...", "name": "..."}`\r\n\r\n---\r\n\r\n## 十三、Token 统计\r\n\r\n### 获取 Token 用量\r\n\r\n```\r\nGET /api/tokens/stats?workspaceRoot={工作区路径}\r\n```\r\n\r\n**参数：** `workspaceRoot` — 工作区路径（默认当前工作区）。\r\n\r\n**响应示例：**\r\n```json\r\n{\r\n  "workspaceRoot": "F:/projects/my-app",\r\n  "promptTokens": 125000,\r\n  "completionTokens": 45000,\r\n  "totalTokens": 170000,\r\n  "cost": 0.85\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 说明 |\r\n|------|------|------|\r\n| promptTokens | number | 提示词 Token 数 |\r\n| completionTokens | number | 补全 Token 数 |\r\n| totalTokens | number | 总 Token 数 |\r\n| cost | number | 估算费用（美元） |\r\n\r\n---\r\n\r\n## 十四、调试日志\r\n\r\n### 14.1 日志列表\r\n\r\n```\r\nGET /api/debug/logs\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {"id": "log_001", "time": "2026-07-11T10:00:00Z", "session": "sess_xxx", "summary": "工具调用: read_file src/main.go"}\r\n]\r\n```\r\n\r\n### 14.2 日志详情\r\n\r\n```\r\nGET /api/debug/logs/{日志ID}\r\n```\r\n\r\n**响应：** 返回指定日志的完整内容。\r\n\r\n---\r\n\r\n## 十五、技能市场\r\n\r\n### 15.1 搜索市场\r\n\r\n```\r\nGET /api/marketplace/search?q={关键词}&kind={类型}\r\n```\r\n\r\n**参数：**\r\n| 参数 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| q | string | 否 | 搜索关键词 |\r\n| kind | string | 否 | 类型（`"mcp"`、`"skill"`、`"all"`） |\r\n\r\n### 15.2 安装扩展\r\n\r\n```\r\nPOST /api/marketplace/install\r\n```\r\n\r\n**请求体：**\r\n```json\r\n{\r\n  "id": "skill-code-review",\r\n  "scope": "project"\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| id | string | 是 | 扩展 ID |\r\n| scope | string | 否 | 安装范围（`"user"`、`"project"`） |\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n### 15.3 刷新市场缓存\r\n\r\n```\r\nPOST /api/marketplace/refresh\r\n```\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 十六、记忆系统\r\n\r\n### 16.1 搜索记忆\r\n\r\n```\r\nGET /api/memory/search?q={关键词}\r\n```\r\n\r\n**响应示例：**\r\n```json\r\n[\r\n  {"name": "项目编码规范", "description": "使用驼峰命名法", "type": "project", "content": "..."}\r\n]\r\n```\r\n\r\n### 16.2 记忆列表\r\n\r\n```\r\nGET /api/memory/list\r\n```\r\n\r\n### 16.3 重建索引\r\n\r\n```\r\nPOST /api/memory/rebuild\r\n```\r\n\r\n**响应：** `{"ok": true}`\r\n\r\n---\r\n\r\n## 十七、插件与工具集管理\r\n\r\nPairCode IDE 的工具系统全部插件化（一切皆插件）。插件（plugin）是工具的最小可复用单元，工具集（toolset）是按项目需求组合的命名插件包。相关 API：\r\n\r\n### 17.1 插件管理\r\n\r\n```\r\nGET   /api/plugins            # 列出已注册插件（含工具归属）\r\nGET   /api/plugins/detail     # 插件详情\r\nPOST  /api/plugins/define     # 定义 JS/TS 插件\r\nPOST  /api/plugins/action     # 插件动作（run/stop/inspect 等）\r\nPOST  /api/plugins/event      # 插件事件\r\nGET   /api/plugins/client-state   # host/client 双半客户端状态\r\nPOST  /api/plugins/client-events  # 客户端事件\r\n```\r\n\r\n### 17.2 工具集管理\r\n\r\n```\r\nGET   /api/toolsets           # 列出工具集\r\nPOST  /api/toolsets/build     # 动态构建工具集（按项目+需求组合插件）\r\nGET   /api/toolsets/export    # 导出工具集 JSON\r\nPOST  /api/toolsets/import    # 导入工具集（project/user 范围）\r\nPOST  /api/toolsets/remove    # 移除工具集\r\n```\r\n\r\n### 17.3 工具配置\r\n\r\n```\r\nGET   /api/tools              # 工具清单（含启用/审核状态）\r\nPOST  /api/tools/save         # 保存工具配置\r\nPOST  /api/tools/review       # 审核配置\r\n```\r\n\r\n---\r\n\r\n## 十八、WebSocket 实时通信协议\r\n\r\nPairCode IDE 使用 **WebSocket** 实现双向实时通信。\r\n\r\n### 17.1 AI 事件推送\r\n\r\n```\r\nws://127.0.0.1:{port}/ws\r\n```\r\n\r\n**用途：** 接收 AI 对话的事件流（思考过程、工具调用、回复内容、错误等）。\r\n\r\n**协议：** 纯文本帧（JSON），**服务端单向推送**，客户端无需发送任何消息。\r\n\r\n#### 事件类型总表\r\n\r\n| 事件类型 | 说明 | 前端展示 |\r\n|---------|------|---------|\r\n| `thinking` | LLM 思考链增量 | 流式显示思考过程（斜体/灰色） |\r\n| `content` | LLM 正文回复增量 | 流式显示正文内容 |\r\n| `tool_call` | AI 即将执行某工具 | 显示工具调用卡片（工具名+参数） |\r\n| `tool_result` | 工具执行结果返回 | 显示结果摘要 |\r\n| `usage` | Token 用量统计 | 更新 Token 计数器 |\r\n| `approval` | 请求用户审批写类操作 | 显示审批对话框（含工具名、参数、文件路径） |\r\n| `error` | 出错或触发止损 | 显示错误信息 |\r\n| `done` | 本次 AI 回复完成 | 关闭加载状态 |\r\n| `compacted` | 上下文已压缩（旧消息被摘要替换） | 显示一条素色提示 |\r\n| `evaluation` | 自主模式任务评分 | 显示评分卡 |\r\n| `circling` | 检测到 AI 重复绕圈 | 显示"换思路"提示 |\r\n| `notice` | 后台任务通知 | 显示一条素色提示 |\r\n| `phase` | 自主模式阶段切换 | 显示阶段指示器（规划/执行/评测） |\r\n| `final` | 单轮委托完成（delegate 用） | 同 done |\r\n\r\n#### 事件 JSON 格式\r\n\r\n```json\r\n{\r\n  "type": "thinking",\r\n  "content": "我来分析一下这个需求...",\r\n  "tool": "",\r\n  "args": "",\r\n  "callId": "",\r\n  "agentName": "",\r\n  "usage": null,\r\n  "doneReason": ""\r\n}\r\n```\r\n\r\n| 字段 | 类型 | 必含 | 说明 |\r\n|------|------|------|------|\r\n| type | string | 是 | 事件类型（见上表） |\r\n| content | string | 按场景 | thinking/content/error/final 时携带文本内容 |\r\n| tool | string | 按场景 | tool_call/tool_result 时携带工具名 |\r\n| args | string | 按场景 | tool_call 时携带工具参数的 JSON 字符串 |\r\n| callId | string | 按场景 | 工具调用 ID，用于关联 tool_call → tool_result |\r\n| agentName | string | 按场景 | 事件来源 Agent 名。空串=主 Agent，非空=子 Agent |\r\n| usage | object | 按场景 | usage 时携带：`{promptTokens:N, completionTokens:N, totalTokens:N}` |\r\n| doneReason | string | 按场景 | done 时携带完成原因（`"completed"`、`"stopped"`、`"error"`） |\r\n\r\n#### 典型事件序列\r\n\r\n```\r\n→ {type:"thinking", content:"我来分析一下..."}\r\n→ {type:"tool_call", tool:"read_file", args:"{\\"path\\":\\"main.go\\"}", callId:"call_1"}\r\n→ {type:"tool_result", tool:"read_file", content:"文件内容...", callId:"call_1"}\r\n→ {type:"thinking", content:"看到文件结构了，接下来..."}\r\n→ {type:"tool_call", tool:"edit_file", args:"{\\"path\\":\\"main.go\\",\\"content\\":\\"...\\"}", callId:"call_2"}\r\n→ {type:"approval", tool:"edit_file", args:"{\\"path\\":\\"main.go\\"}", callId:"call_2"}\r\n   （等待用户审批 → 调用 POST /api/chat/approve）\r\n→ {type:"tool_result", tool:"edit_file", content:"文件已更新", callId:"call_2"}\r\n→ {type:"content", content:"已完成修改，以下是改动内容..."}\r\n→ {type:"usage", content:"", usage:{promptTokens:1200, completionTokens:350, totalTokens:1550}}\r\n→ {type:"done", doneReason:"completed"}\r\n```\r\n\r\n> **重要：** WebSocket 连接为全局单连接，推送**所有**会话的事件。事件中的 `convId` 字段（若存在）用于区分不同对话。前端需根据 `convId` 路由到对应的对话面板。\r\n\r\n---\r\n\r\n### 17.2 终端 WebSocket\r\n\r\n```\r\nws://127.0.0.1:{port}/api/terminal/ws\r\n```\r\n\r\n**用途：** 内置终端的双向输入输出通道，每连接对应一个 PTY 终端会话。\r\n\r\n#### 协议规则\r\n\r\n| 帧类型 | 方向 | 说明 |\r\n|--------|------|------|\r\n| 文本帧 (JSON) | 客户端→服务端 | 控制消息 |\r\n| 文本帧 (JSON) | 服务端→客户端 | 状态通知 |\r\n| 二进制帧 | 双向 | 原始 PTY I/O 字节流（含 VT 转义序列，由 xterm.js 渲染） |\r\n\r\n#### 控制消息格式\r\n\r\n**客户端 → 服务端（初始化）：**\r\n```json\r\n{"type": "init", "shell": "cmd", "cwd": "F:/projects/my-app"}\r\n```\r\n\r\n| 字段 | 类型 | 必填 | 说明 |\r\n|------|------|------|------|\r\n| type | string | 是 | 固定 `"init"` |\r\n| shell | string | 是 | Shell 名：`"cmd"` \\| `"powershell"` \\| `"gitbash"`（白名单限制） |\r\n| cwd | string | 是 | 工作目录（禁止穿越出工作区） |\r\n\r\n**客户端 → 服务端（调整大小）：**\r\n```json\r\n{"type": "resize", "cols": 120, "rows": 30}\r\n```\r\n\r\n**服务端 → 客户端：**\r\n```json\r\n{"type": "ready"}\r\n{"type": "error", "msg": "shell 不在白名单中"}\r\n{"type": "closed"}\r\n```\r\n\r\n#### 安全措施\r\n\r\n- Shell 白名单：仅允许 `cmd`、`powershell`、`gitbash`\r\n- `cwd` 路径校验：禁止穿越出工作区\r\n- PTY 关闭时强制终止子进程\r\n- 并发 PTY 会话数限制：最多 16 个\r\n\r\n---\r\n\r\n## 附录：API 索引速查\r\n\r\n### 基础 API\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| GET | `/api/health` | 健康检查 |\r\n| GET | `/api/system/info` | 系统信息+版本号 |\r\n| POST | `/api/system/exec` | 执行命令 |\r\n\r\n### 文件系统 (11 个)\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| GET | `/api/fs/list` | 列出目录 |\r\n| GET | `/api/fs/read` | 读取文件 |\r\n| POST | `/api/fs/write` | 写入文件 |\r\n| GET | `/api/fs/search` | 搜索内容 |\r\n| POST | `/api/fs/rename` | 重命名/移动 |\r\n| POST | `/api/fs/delete` | 删除 |\r\n| POST | `/api/fs/mkdir` | 创建目录 |\r\n| GET | `/api/fs/image` | 图片 Base64 |\r\n| GET | `/api/fs/file-info` | 文件信息 |\r\n| GET | `/api/fs/hex` | 十六进制查看 |\r\n| GET | `/api/fs/drives` | 磁盘驱动器列表 |\r\n\r\n### 工作区 & 设置\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| GET/POST | `/api/workspace` | 工作区管理 |\r\n| GET/PUT | `/api/settings` | 设置管理 |\r\n\r\n### AI 对话 (9 个)\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| POST | `/api/chat/send` | 发送消息给 AI |\r\n| POST | `/api/chat/stop` | 停止 AI 回复 |\r\n| POST | `/api/chat/approve` | 审批操作 |\r\n| POST | `/api/chat/feedback` | 发送运行时反馈 |\r\n| POST | `/api/chat/answer` | 回答 ask_user 提问 |\r\n| POST | `/api/chat/rollback` | 回滚到指定消息前 |\r\n| POST | `/api/chat/compact` | 手动压缩上下文 |\r\n| GET | `/api/models` | 可用模型列表 |\r\n\r\n### 对话管理 (8 个)\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| GET | `/api/conversations` | 对话列表 |\r\n| POST | `/api/conversations` | 创建对话 |\r\n| GET | `/api/conversations/{id}` | 对话详情（含消息） |\r\n| PUT | `/api/conversations/{id}` | 更新对话 |\r\n| DELETE | `/api/conversations/{id}` | 删除对话 |\r\n| GET | `/api/conversations/{id}/messages` | 消息列表（分页） |\r\n| POST | `/api/conversations/{id}/messages` | 添加消息 |\r\n| GET | `/api/conversations/{id}/messages/count` | 消息总数 |\r\n\r\n### Git (16 个)\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| POST | `/api/git/init` | 初始化仓库 |\r\n| GET | `/api/git/status` | 仓库状态 |\r\n| GET | `/api/git/diff` | 查看差异 |\r\n| POST | `/api/git/add` | 暂存 |\r\n| POST | `/api/git/reset` | 取消暂存 |\r\n| POST | `/api/git/commit` | 提交 |\r\n| GET | `/api/git/log` | 提交历史 |\r\n| GET | `/api/git-log` | 提交历史（别名） |\r\n| POST | `/api/git/branch` | 分支管理 |\r\n| POST | `/api/git/checkout` | 切换分支/恢复文件 |\r\n| POST | `/api/git/stash` | 贮藏 |\r\n| GET | `/api/git/stash-list` | 贮藏列表 |\r\n| GET/POST | `/api/git/ignore` | 管理 .gitignore |\r\n| POST | `/api/git/discard` | 丢弃修改 |\r\n| POST | `/api/git/push` | 推送 |\r\n| POST | `/api/git/pull` | 拉取 |\r\n| GET/POST | `/api/git/remote` | 远程仓库管理 |\r\n\r\n### 扩展 & 系统\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| GET | `/api/skills/list` | 技能列表 |\r\n| GET | `/api/skills/read` | 读取技能 |\r\n| POST | `/api/skills/save` | 保存/更新技能状态 |\r\n| POST | `/api/skills/delete` | 删除技能 |\r\n| GET | `/api/mcp/list` | MCP 列表 |\r\n| POST | `/api/mcp/save` | MCP 保存/管理 |\r\n| GET | `/api/tokens/stats` | Token 统计 |\r\n| GET | `/api/debug/logs` | 调试日志列表 |\r\n| GET | `/api/debug/logs/{id}` | 调试日志详情 |\r\n| GET | `/api/memory/search` | 搜索记忆 |\r\n| GET | `/api/memory/list` | 记忆列表 |\r\n| POST | `/api/memory/rebuild` | 重建记忆索引 |\r\n| GET | `/api/marketplace/search` | 市场搜索 |\r\n| POST | `/api/marketplace/install` | 安装扩展 |\r\n| POST | `/api/marketplace/refresh` | 刷新市场缓存 |\r\n| GET/PUT | `/api/instructions` | 指令管理 |\r\n| GET | `/api/tasks` | 任务列表（只读查询） |\r\n| GET/POST | `/api/taskplan` | 规划文档管理 |\r\n\r\n### 插件 & 工具集\r\n| 方法 | 端点 | 用途 |\r\n|------|------|------|\r\n| GET | `/api/plugins` | 插件列表（含工具归属） |\r\n| GET | `/api/plugins/detail` | 插件详情 |\r\n| POST | `/api/plugins/define` | 定义 JS/TS 插件 |\r\n| POST | `/api/plugins/action` | 插件动作（run/stop/inspect） |\r\n| POST | `/api/plugins/event` | 插件事件 |\r\n| GET | `/api/plugins/client-state` | host/client 客户端状态 |\r\n| POST | `/api/plugins/client-events` | 客户端事件 |\r\n| GET | `/api/toolsets` | 工具集列表 |\r\n| POST | `/api/toolsets/build` | 动态构建工具集 |\r\n| GET | `/api/toolsets/export` | 导出工具集 JSON |\r\n| POST | `/api/toolsets/import` | 导入工具集 |\r\n| POST | `/api/toolsets/remove` | 移除工具集 |\r\n| GET | `/api/tools` | 工具清单 |\r\n| POST | `/api/tools/save` | 保存工具配置 |\r\n| POST | `/api/tools/review` | 审核配置 |\r\n\r\n---\r\n\r\n### WebSocket 端点\r\n| 端点 | 用途 |\r\n|------|------|\r\n| `ws://host/ws` | AI 事件流推送（思考/工具/结果/完成） |\r\n| `ws://host/api/terminal/ws` | PTY 终端双向 I/O |\r\n';
  const toolsMd = '# AI 工具文档\n\nPairCode IDE 中的 AI 助手拥有丰富的内置能力，可以像你使用 IDE 一样操作文件、搜索代码、运行命令、管理版本。你只需用自然语言告诉 AI 你想做什么，AI 会自动选择合适的工具来完成任务。\n\n所有工具对 AI 完全开放，你无需记忆工具名称——只需描述需求，AI 自动判断该用什么。\n\n---\n\n## 一、代码阅读与搜索\n\n**浏览项目结构、搜索代码内容和定位符号定义，是 AI 理解你代码的基础能力。**\n\nAI 可以像你一样阅读和浏览项目代码：\n\n- 读取文件内容（可按行号范围读取部分内容）\n- 列出目录下的文件和子目录\n- 按关键词或正则表达式在文件内容中搜索\n- 按通配符模式递归查找文件\n- 搜索函数、类型、结构体等符号的定义位置\n- 查看指定文件中所有检测到的符号\n- 搜索某个符号在项目中的所有引用位置\n- 列出项目中所有导出的公开符号\n- 查看文件的导入依赖和反向依赖\n- 分析修改某个文件后可能影响的其他文件\n- 检测项目中的循环依赖\n\n---\n\n## 二、代码知识图谱 CodeGraph\n\n**AI 能理解你的代码结构和调用关系，而不仅仅是搜索文本。**\n\nCodeGraph 将项目的代码整体结构构建成可查询的知识图谱，让 AI 像理解知识一样理解你的代码：\n\n- 构建或更新项目的代码知识图谱\n- 查看知识图谱的统计信息\n- 按名称查找函数或方法的定义位置和签名\n- 获取结构体或接口的完整层次结构（字段、方法、嵌入类型）\n- 查询哪些函数调用了指定的某个函数\n- 查询某个函数内部调用了哪些其他函数\n- 分析修改某个函数或类型后可能影响的范围\n- 在知识图谱中按名称搜索代码实体\n- 查询代码实体的 Git 变更历史\n\n---\n\n## 三、文件操作\n\n**读写和编辑工作区内的文件，是 AI 帮你写代码的主要方式。**\n\nAI 可以直接在工作区中进行文件操作：\n\n- 将内容写入指定文件（覆盖模式，自动创建父目录）\n- 精确替换文件中的一段文本\n- 将文件或目录移动到新位置（也可用于重命名）\n- 删除指定文件\n- 将文件恢复到修改前的版本\n- 查看某个文件的所有修改历史版本\n\n---\n\n## 四、命令执行\n\n**在工作区中运行命令，AI 也能用命令行来完成任务。**\n\n- 执行一条 shell 命令并等待结果返回\n- 在后台启动一条长命令（如启动开发服务器）\n- 读取后台进程累积的输出内容\n- 停止正在运行的后台进程\n- 直接执行一段代码（自动探测语言，写临时文件运行 Go / Python / Node.js 并返回结果）\n\n---\n\n## 五、网络与搜索\n\n**AI 可以联网获取信息或搜索资料。**\n\n- 抓取网页内容并提取纯文本\n- 通过搜索引擎检索网络信息\n\n---\n\n## 六、网页验证与截图\n\n**AI 可以打开网页、截图并分析页面内容，用于验证前端效果。**\n\n- 在浏览器中打开网页，可输入文字、点击元素、检查控制台错误并截图\n- 获取 JavaScript 渲染后的页面文本内容（适合单页应用）\n- 截取桌面或指定窗口的屏幕\n- 截取指定 URL 的网页\n\n---\n\n## 七、图像分析\n\n**AI 可以"看"图片并理解其中的内容。**\n\n- 读取图片文件内容（供支持视觉的模型直接理解图像）\n- 分析图片中的颜色分布、色块区域和基本图形\n- 从图片中识别文字，支持中英文混合识别\n\n---\n\n## 八、二进制分析\n\n**查看和分析二进制文件的内容，用于逆向工程或文件格式分析。**\n\n- 分析二进制文件的大小、类型和十六进制预览\n- 将 Base64 编码的内容写入二进制文件\n- 从二进制文件中提取可打印的字符串\n- 在二进制文件中搜索指定的字节模式或文本\n- 在二进制文件的指定位置写入字节补丁\n- 解析可执行文件的结构（架构、入口、节区、导入导出）\n- 计算文件的 MD5、SHA1、SHA256 哈希值\n- 按块计算文件的香农熵（识别压缩或加密区域）\n\n---\n\n## 九、办公文档\n\n**读写常见的办公文档格式，包括表格、文档和 PDF。**\n\n- 读取 CSV 或 TSV 文件并以表格形式展示\n- 将数据写入 CSV 或 TSV 文件\n- 将 JSON 数组数据转为 Markdown 表格\n- 对表格数据的数值列做统计（求和、均值、最大值等）\n- 按文件扩展名分组统计代码行数\n- 读取和生成 Word 文档\n- 读取和创建 Excel 文件\n- 提取 PDF 文件的文本内容（扫描型 PDF 自动进行 OCR 识别）\n- 将 Markdown 文本转换为 HTML\n\n---\n\n## 十、Git 版本控制\n\n**在对话中完成 Git 操作，AI 可以帮你管理代码版本。**\n\n- 查看工作区的 Git 状态\n- 查看文件的变更内容\n- 查看最近的提交历史\n- 查看某次提交的详情和改动\n- 逐行查看文件的最后修改人和提交信息\n- 将文件加入暂存区\n- 提交已暂存的改动\n- 列出、创建或删除分支\n- 切换分支或恢复文件的修改\n- 将工作区的改动暂存起来，稍后恢复\n\n---\n\n## 十一、调试器\n\n**AI 可以启动调试会话，设置断点并检查程序运行状态。**\n\n- 启动 Go 程序的调试会话\n- 停止当前的调试会话\n- 在指定文件的指定行设置断点\n- 从暂停状态继续执行程序\n- 单步跳过（不进入函数内部）\n- 单步进入（进入函数调用内部）\n- 单步跳出（执行到函数返回）\n- 查看当前线程的调用栈\n- 查看当前暂停点的变量值\n- 在暂停状态下求值表达式\n- 查看当前调试会话的状态\n\n---\n\n## 十二、项目知识库\n\n**将项目架构、模块职责和设计决策记录下来，让 AI 跨会话了解你的项目。**\n\n- 写入一条项目知识（如架构说明或设计决策）\n- 读取某条项目知识的详细内容\n- 列出知识库的所有条目概览\n- 按关键词搜索知识库内容\n- 删除某条项目知识\n- 生成项目目录结构概览\n\n---\n\n## 十三、记忆系统\n\n**AI 可以记住你的偏好、历史决策和项目约束，跨对话持续积累。**\n\n- 写入一条持久记忆，AI 在后续对话中自动参考\n- 读取某条记忆的详细内容\n- 按关键词搜索已有记忆\n- 列出所有历史记忆的摘要\n- 删除一条过时的记忆\n- 查询记忆库中的总条目数\n\n---\n\n## 十四、BUG 检测与修复\n\n**AI 可以自动发现代码中的问题并给出修复方案。**\n\n- 分析构建或测试的输出，提取错误位置和上下文\n- 全量检测项目中的 BUG，自动运行编译和测试检查\n- 自动检测 BUG 并生成修复方案，支持多次迭代修复\n\n---\n\n## 十五、任务与规划\n\n**AI 可以追踪任务进度和执行计划，确保复杂的多步骤任务有条不紊。**\n\n- 创建一个新的子任务并跟踪其状态\n- 更新任务清单中各项任务的进度状态\n- 维护和更新执行计划的步骤清单\n- 任务全部完成后生成提交信息\n\n---\n\n## 十六、技能与 MCP 管理\n\n**管理和扩展 AI 的能力——技能是工作流模板，MCP 是标准化的工具扩展协议。**\n\n- 列出所有可用的技能及其激活模式\n- 加载某个技能的完整内容供 AI 使用\n- 加载技能的附加资源文件\n- 创建或更新一个技能模板\n- 删除一个项目级技能\n- 列出已配置的 MCP 服务器\n- 新增或删除 MCP 服务器扩展\n\n---\n\n## 十七、市场\n\n**浏览和安装来自公共市场的技能和 MCP 扩展。**\n\n- 在市场检索可安装的 MCP 服务器或技能\n- 从市场安装指定的扩展\n\n---\n\n## 十八、插件管理\n\n**管理 JS / TS / Go / Lua 插件——一切皆插件，自定义和扩展 AI 的工具集。**\n\n- 定义一个函数形态的 JS/TS 插件（支持 apply(ctx, config) 注入服务、timer 定时器、跨 goroutine 执行锁）\n- 查看已注册插件的详情（含工具归属：每个工具来自哪个插件，可整体卸载回收）\n- 对插件执行查询（inspect 内部状态）\n- 运行插件注册的服务或回调\n- 列出 / 停止已注册的插件服务\n- 撤销（undefine）一个已定义的插件\n- 列出所有已创建的 Lua 自定义工具\n- 创建一个新的 Lua 自定义工具\n- 更新现有 Lua 工具的代码或参数\n- 删除一个 Lua 自定义工具\n\n---\n\n## 十九、工具集管理\n\n**按项目需求动态组合工具集，固化/导出/导入，构建处理本身也插件化。**\n\n- 分析项目结构与需求，动态组合所需工具并创建工具集插件（固化到工作区 `.pair/toolsets/`）\n- 列出当前项目可用的工具集\n- 查看某个工具集的详细内容\n- 导出工具集为 JSON（可提交 Git / 发布市场）\n- 从 JSON 或文件导入工具集（project 工作区级 / user 全局级）\n- 移除不再需要的工具集\n\n---\n\n## 二十、其他工具\n\n**辅助性工具，在特定场景下帮助 AI 更好地与你协作。**\n\n- **用户提问** — 当 AI 遇到关键决策点时，向你提问以澄清需求\n- **任务委派** — 将复杂任务委托给子 AI 独立完成\n- **资产清单** — 查看和使用已保存的经验胶囊和最佳实践\n';
  const shortcutsMd = "# 快捷键参考\r\n\r\nPairCode IDE 提供了丰富的快捷键，帮助你更高效地编写代码和管理项目。以下按功能分类列出所有可用的快捷键。\r\n\r\n---\r\n\r\n## 一、通用操作\r\n\r\n**控制 IDE 界面面板的显示与隐藏，快速切换工作布局。**\r\n\r\n| 快捷键 | 功能 | 适用范围 |\r\n|--------|------|----------|\r\n| Ctrl+B | 切换侧栏（文件浏览器）显示/隐藏 | 全局 |\r\n| Ctrl+\\` | 切换终端面板显示/隐藏 | 全局 |\r\n| Ctrl+K | 专注模式：隐藏所有面板，聚焦代码编辑区 | 全局 |\r\n| Ctrl+Shift+C | 切换对话面板显示/隐藏 | 全局 |\r\n| Escape | 关闭当前模态框或菜单 | 全局 |\r\n\r\n## 二、文件编辑\r\n\r\n**编辑器中常用的编辑操作，与主流编辑器保持一致。**\r\n\r\n| 快捷键 | 功能 | 适用范围 |\r\n|--------|------|----------|\r\n| Ctrl+S | 保存当前文件 | 编辑器 |\r\n| Ctrl+Z | 撤销操作 | 编辑器 |\r\n| Ctrl+Shift+Z / Ctrl+Y | 重做操作 | 编辑器 |\r\n| Ctrl+X | 剪切选中的内容 | 编辑器 |\r\n| Ctrl+C | 复制选中的内容 | 编辑器 |\r\n| Ctrl+V | 粘贴剪贴板内容 | 编辑器 |\r\n| Ctrl+A | 全选当前文件内容 | 编辑器 |\r\n| Ctrl+F | 在当前文件中搜索 | 编辑器 |\r\n| Ctrl+H | 在当前文件中查找替换 | 编辑器 |\r\n| Ctrl+P | 按文件名快速打开文件 | 编辑器 |\r\n\r\n## 三、导航与视图\r\n\r\n**在不同功能面板之间快速切换，无需鼠标操作。**\r\n\r\n| 快捷键 | 功能 | 适用范围 |\r\n|--------|------|----------|\r\n| Ctrl+Shift+E | 切换到文件浏览器 | 全局 |\r\n| Ctrl+Shift+F | 全局搜索（在工作区中搜内容） | 全局 |\r\n| Ctrl+Shift+T | 打开对话面板 | 全局 |\r\n| F2 | 重命名选中的文件或文件夹 | 文件树 |\r\n| Ctrl+Tab | 在打开的文件标签页之间切换 | 编辑器 |\r\n| Ctrl+W | 关闭当前文件标签页 | 编辑器 |\r\n\r\n## 四、对话面板\r\n\r\n**AI 对话输入区的快捷操作。**\r\n\r\n| 快捷键 | 功能 | 适用范围 |\r\n|--------|------|----------|\r\n| Enter | 发送消息给 AI | 对话面板 |\r\n| Shift+Enter | 换行（多行输入） | 对话面板 |\r\n| Ctrl+Up | 切换到上一条对话 | 对话面板 |\r\n| Ctrl+Down | 切换到下一条对话 | 对话面板 |\r\n\r\n## 五、终端\r\n\r\n**终端面板的操作快捷键。**\r\n\r\n| 快捷键 | 功能 | 适用范围 |\r\n|--------|------|----------|\r\n| Ctrl+\\` | 打开/关闭终端面板 | 全局 |\r\n| Ctrl+Shift+\\` | 新建终端标签页 | 终端 |\r\n| Ctrl+W | 关闭当前终端标签页 | 终端 |\r\n| Ctrl+C | 中断当前正在运行的命令 | 终端 |\r\n\r\n## 六、多标签页导航\r\n\r\n| 快捷键 | 功能 | 适用范围 |\r\n|--------|------|----------|\r\n| Ctrl+Tab | 切换到下一个文件标签页 | 编辑器 |\r\n| Ctrl+Shift+Tab | 切换到上一个文件标签页 | 编辑器 |\r\n| Ctrl+PageUp | 切换到上一个文件标签页 | 编辑器 |\r\n| Ctrl+PageDown | 切换到下一个文件标签页 | 编辑器 |\r\n| Ctrl+W | 关闭当前文件标签页 | 编辑器 |\r\n";
  const faqMd = '# 常见问题\n\n## PairCode IDE 是什么？\n\nPairCode IDE 是一款 AI 原生的纯 Web 集成开发环境。与传统 IDE 不同，你只需用浏览器打开，在对话面板中用自然语言描述需求，AI 就能理解你的意图，自动完成代码编写、文件操作、命令执行等工作——让编程从手工操作转变为对话驱动。\n\n## 需要安装桌面客户端吗？\n\n不需要。PairCode IDE 是纯 Web 应用，你只需启动后台服务，然后用浏览器（推荐 Chrome、Edge、Firefox）访问即可。所有界面在浏览器中渲染，无需安装任何桌面客户端。\n\n## AI 能做什么？\n\nAI 可以读写和编辑你的代码文件、在工作区中执行命令、搜索和浏览项目结构、管理 Git 版本控制、启动调试会话、处理图片和办公文档、搜索网络信息，还能截图验证网页效果。基本上，日常开发中你能做的事情，AI 都可以帮你完成。\n\n## 如何让 AI 执行命令？\n\n你可以在对话中直接告诉 AI 需要运行什么命令，例如"运行测试"或"启动项目"。AI 会自动在终端中执行并返回结果输出。涉及文件写入和命令执行的操作会先请求你的确认。\n\n## 文件保存在哪里？\n\n所有文件都保存在你本地的工作区目录中。PairCode IDE 直接读写你本地磁盘上的文件，不经过云端存储。你可以在文件浏览器中看到完整的项目目录结构，用系统的文件管理器也能找到它们。\n\n## 如何切换 AI 模型？\n\n在设置面板的"AI 模型"选项卡中，你可以选择不同的 AI 服务商和模型。支持接入 OpenAI、Claude 等多种主流模型后端。你可以为执行任务和制定规划分别配置不同的模型。\n\n## 如何安装更多技能？\n\n在市场中可以浏览和安装社区贡献的技能模板、MCP 扩展和工具集插件。技能是可复用的工作流程模板，MCP 扩展可以给 AI 添加新的能力，工具集是按项目需求组合的插件包（可通过 `toolset_build` 动态构建并固化到工作区）。打开市场面板，搜索你需要的功能，一键即可安装使用。\n\n## 对话历史会丢失吗？\n\n不会。每次对话都会自动保存在本地磁盘上，你可以随时在对话列表中查看历史记录、继续之前的对话或开启新话题。切换工作区时，各项目的对话会自动隔离，互不干扰。\n\n## 如何保护隐私？\n\n所有操作都在你的本地计算机上执行，代码和对话内容不会发送到外部服务器（AI 模型调用除外，你可以选择使用本地模型避免数据外出）。API 服务只监听本地回环地址，默认不对外暴露。文件操作限定在工作区范围内。\n\n## 页面刷新后数据还在吗？\n\n大部分数据都会保留：\n- **对话历史** — 自动持久化到磁盘，刷新后完整恢复\n- **打开的文件** — 刷新后自动重新打开\n- **工作区状态** — 侧栏位置、面板大小等布局信息保存在浏览器中\n- **设置** — 主题、AI 模型配置等设置持久化到磁盘\n\n## 编辑器里的代码没有高亮怎么办？\n\n编辑器会根据文件扩展名自动切换语言模式。如果文件扩展名不常见，代码高亮可能无法自动识别。建议确认文件扩展名是否被支持，或使用常见的扩展名保存文件。\n\n## 什么是自主模式？和普通对话有什么区别？\n\n**普通模式**：你发一条指令，AI 执行并回复，然后等待你下一条指令。\n\n**自主模式**：你交给 AI 一个复杂任务（如"修复所有编译错误"），AI 会自动分解任务、逐个执行、迭代验证，直到全部完成。你不需要逐条发指令，只需在关键节点确认即可。\n\n## 能让 AI 访问我的私有 API 吗？\n\n可以通过 MCP（模型上下文协议）扩展来实现。在设置中添加自定义 MCP 服务器，AI 就能通过它访问你的私有 API、数据库或内部服务。\n\n## 遇到问题怎么办？\n\n你可以查看帮助菜单中的文档中心，里面有功能介绍、API 文档、工具文档和快捷键参考等详细资料。如果问题仍然无法解决，可以在对话中向 AI 描述你遇到的问题，它会尽力协助排查。\n';
  const gettingStartedMd = '# 快速开始\n\n欢迎使用 PairCode IDE！以下指南将带你快速上手，从打开工作区到用 AI 写代码，只需几分钟。\n\n---\n\n## 打开 IDE\n\nPairCode IDE 是一个纯 Web 应用，启动后台服务后，直接在浏览器中访问对应地址即可使用。所有界面在浏览器中渲染，无需安装任何桌面客户端。\n\n> 建议使用 Chrome、Edge 或 Firefox 等现代浏览器获得最佳体验。\n\n---\n\n## 设置工作区\n\n工作区是 IDE 操作的基础——所有文件操作、AI 对话和命令执行都将在这个目录范围内进行。\n\n1. 点击左侧活动栏顶部的**文件图标**打开文件浏览器\n2. 在文件浏览器顶部输入你的项目文件夹的完整路径\n3. 按回车确认，IDE 会自动加载该目录下的所有文件和子目录\n\n你也可以同时添加多个文件夹到同一个工作区中，方便跨目录浏览和管理代码。\n\n---\n\n## 与 AI 对话\n\n右侧的**对话面板**是 PairCode IDE 的核心交互界面。你只需用自然语言描述需求，AI 就能理解并执行。\n\n直接在输入框中输入你的需求，例如：\n\n- "创建一个 Go 文件，实现一个返回 JSON 的 HTTP 服务"\n- "帮我优化这个函数，加上错误处理和参数校验"\n- "搜索项目中所有调用了 Post 的地方"\n- "运行项目中的所有测试，并告诉我哪些失败了"\n- "把我的改动提交到 Git"\n\n按 Enter 发送消息，Shift+Enter 换行。AI 会实时流式展示它的思考过程、工具调用和结果输出。\n\n### 常用对话技巧\n\n| 技巧 | 说明 |\n|------|------|\n| **明确具体** | 越具体，AI 理解越准确。如"写一个函数"不如"写一个读取 JSON 配置文件的函数" |\n| **分步沟通** | 复杂任务可以分步骤告诉 AI，先分析，再重构 |\n| **提供上下文** | 在对话中粘贴错误信息或代码片段，AI 能给出更精准的修复方案 |\n| **使用反馈** | 如果 AI 输出不满意，直接指出问题，AI 会调整方案重新尝试 |\n\n---\n\n## 编辑代码\n\nAI 生成的代码会直接写入到文件中。你也可以在编辑器中手动查看和修改代码：\n\n- **多标签页** — 同时打开多个文件，在标签栏切换\n- **语法高亮** — 支持 Go、TypeScript、Python、Rust、Java、Vue 等主流语言\n- **代码折叠** — 折叠函数和代码块，聚焦关键逻辑\n- **Ctrl+S** — 保存当前文件的修改\n\n你还可以在编辑器中查看二进制文件的十六进制内容，或直接预览图片文件。\n\n---\n\n## 运行与调试\n\n### 使用内置终端\n\n按 Ctrl+\\` 打开 IDE 底部的终端面板，可以直接在工作区目录下运行命令。支持多标签页，方便在不同任务间切换。\n\n### 让 AI 帮你运行\n\n你也可以直接在对话中告诉 AI："运行项目并告诉我结果"或"执行 npm test"。AI 会自动在终端中执行命令、读取输出，并根据结果决定下一步操作。\n\n---\n\n## 版本控制\n\nGit 操作完全融入 AI 对话流程。你用自然语言就能完成所有 Git 操作：\n\n- "查看当前仓库状态"\n- "暂存所有修改并提交"\n- "创建一个新分支并切换过去"\n- "从远程拉取最新代码"\n\n你也可以通过左侧 Git 面板查看文件变更的详细对比，逐行确认每次改动的具体内容。\n\n---\n\n## 个性化设置\n\n点击活动栏的**齿轮图标**打开设置面板，你可以：\n\n- **AI 模型** — 选择不同的 AI 服务商和模型\n- **外观主题** — 切换暗色、白色、暖色和暗夜紫四套主题\n- **工作区管理** — 查看和切换最近使用的工作区\n- **系统指令** — 自定义 AI 的行为指导原则\n\n---\n\n## 探索更多\n\nPairCode IDE 还有更多强大功能等待你探索。欢迎查阅帮助文档中的其他章节：\n\n- **功能介绍** — 了解所有功能模块的详细说明\n- **工具文档** — 查看 AI 可使用的全部内置能力\n- **快捷键参考** — 常用快捷键一览\n- **API 文档** — 后端 HTTP API 接口说明\n- **常见问题** — 常见问题与解答\n';
  const changelogMd = '# 更新日志\r\n\r\n> 所有 PairCode IDE 的重要变更均记录在此文件中。\r\n\r\n---\r\n\r\n## 1.2.1 — 2026-08-15\r\n\r\n### 新增\r\n- **按 deepseek-harness 设计重写 Agent 核心** — 双层循环（turn/step 边界事件、inbox 双队列对齐 next-step/next-turn），消息组装与落盘对齐 harness（agentloop 编号 ↔ 消息序列推导），系统提示精简为 harness 模式（`WB_FULL_TOOLS=1` 恢复全量工具）\r\n- **一切皆插件** — Go 插件框架 + goja JS 动态插件，goja 运行时完全内置（双仓库去除 replace），JS 插件沙箱支持 timer 服务（ctx.timeout/interval）与跨 goroutine 执行锁\r\n- **内置 TS 编译器** — esbuild 纯 Go 转译（无 CGO/npm 依赖），TS 插件可直接加载（`cordis_define` 支持 js/ts/自动探测），多文件 TS bundle（Build stdin + mock 包）\r\n- **工具全插件化** — 21 个内置功能插件（core/fs/git/web/shell/memory/task/project-info/codegraph/debug/vision/office/lsp 等），`cordis_inspect` 可见工具归属插件，Unload 可回收整组\r\n- **多项目支持** — 工具 project 参数路由（文件类/搜索/Git 全套），codegraph 按项目独立建图与查询（非主项目用各自 JSONStore，天然隔离），memory/project-info 工具显式 project 参数化\r\n- **工具集生态** — 模板插件化动态构建（`toolset_build` 按项目+需求自动组合工具并固化到工作区）、固化/导出/导入/市场发布（plugin 类型）、LLM 项目意图分析（语言无关，不固化任何语言模板）\r\n- **插件生态 P0-P2** — 函数形态 + `apply(ctx, config)` + inject 服务 + VM 超时防护 + schema 校验 + 插件管理 UI（host/client 双半）+ client inspect provider\r\n- **项目知识库树形化** — 树分支组织（目标/架构/实现/关键点/设计思想）+ AGENTS.md 分层 + .agents 路径兼容\r\n- **历史注入对齐 harness** — 删除【历史轮次】前缀标注与 task 时间戳，系统提示补充多轮对话规则\r\n- **ask_user 选项内输入** — 支持 single / multi / single-with-input / text 四态交互，修复参数名混淆导致选项不出现的问题\r\n- **遗留五件套** — notes 写入同步 + read_image 工具 + run_code 嵌套 + prompt 注册中心 + 知识库过期检查修复\r\n\r\n### 修复\r\n- **移除未完成注入** — TOOL_OUTCOME_UNKNOWN / interrupted 机制移除，无 result 的 tool_call 以空占位维持配对契约，不再向模型注入「中断/未完成」语义\r\n- **知识库过期验证误报** — 152 条假警告清零，159 条全绿\r\n\r\n---\r\n\r\n## 1.1.8 — 2026-08-11\r\n\r\n### 新增\r\n- **OCR / 图色识别能力** — 图片文字识别（中英文混合）与颜色分布分析，工具配置持久化 + 前端工具面板（2026-08-04）\r\n- **对话历史注入膨胀三层压缩** — 固定背景 / 动态日志 / 长时压缩三层方案，控制上下文体积（2026-08-04）\r\n- **异常中断后继续未完成对话** — 中断后可直接继续，不丢上下文（2026-08-06）\r\n- **后台进程跨轮存活** — run_background 进程不再因每轮重建注册表而丢失（全局单例 bgRegistry）（2026-08-11）\r\n- **多项目工具** — Lua 工具 / 工具配置按项目加载 + project 参数路由（2026-08-11）\r\n- **背景摘要注入位置修复** — 压缩摘要固定在 task 前注入（前缀稳定），动态日志追加末尾，KV 缓存零损失优化（2026-08-08）\r\n\r\n### 修复\r\n- 关闭 run 内自动压缩，改由外层时机控制（2026-08-05）\r\n- 历史消息配对错乱 — 用户消息重复存储导致 tool 配对错乱（lastUser 锚点重组）\r\n- 历史消息分段导致多气泡 — 连续 assistant 消息合并显示\r\n- 多轮对话 user 后 tool 粘连 + OnBatchPersist 偏移 — 压缩后固定偏移失效，改 lastUser 锚点重组\r\n- 归档双 bug — ①Windows 归档静默失效（句柄未关闭 + os.Rename 不能覆盖）→ 显式 Close + 三步法原子替换；②归档摘要孤立 assistant 消息污染 LLM 上下文 → 改 role=user +【历史归档】标注\r\n- 多根路径解析 Bug — 优先匹配文件实际存在的根目录\r\n\r\n---\r\n\r\n## 1.1.6 — 2026-07-30\r\n\r\n### 修复\r\n- **修复编辑器 Ctrl+F 不生效** — CodeMirror `search()` 扩展注册的 `openSearchPanel` 与自定义搜索面板 keymap 冲突，使用 `Prec.high()` 确保自定义 handler 优先执行，Ctrl+F 正确唤出中文搜索面板\r\n- **搜索面板图标全部换为 SVG** — Unicode 字符（▲▼↔×）和文本标签（Aa ·\\* 全词）全部替换为内联 SVG 图标，与界面风格统一\r\n- **修复前端 API 路径缺少前导斜杠导致 404** — `apiURL()` 拼接时对无前导斜杠的 path 自动补全，`/apitools/review` 修正为 `/api/tools/review`\r\n- **修复 codegraph 增量构建仍全量重写 SQLite** — `SQLiteStore.Save()` 在增量模式下调用的 `RemoveFileEntities` 清理旧数据，不再 `DELETE FROM` 全表\r\n\r\n### 改进\r\n- **编辑器中文搜索面板** — 新建 `FindPanel.vue` 组件，替换 CodeMirror 默认英文搜索面板，支持查找/替换/大小写敏感/正则/全词匹配\r\n- **codegraph 增量构建测试** — 新增 `TestSQLiteStoreIncrementalPreserves` 和 `TestSQLiteStoreIncrementalBuild` 验证增量构建与并行完整性\r\n\r\n---\r\n\r\n## 1.1.5 — 2026-07-29\r\n\r\n### 新增\r\n- **run_command 后台化** — `run_command` 改用后台启动+轮询模式，不再阻塞 Agent 循环，可被上下文取消中断，超时后 LLM 可选择等待或继续\r\n- **审核配置改为工作区级** — 审核黑白名单从全局 settings.json 迁移到工作区 .pair/tools.json，不同工作区可独立配置，避免动态工具（Lua）在不同工作区间混淆\r\n- **Lua 工具补齐 Tool 结构** — `buildLuaTool` 自动设置 UsageGuide/Category/Enabled 字段，与标准工具结构一致\r\n- **工具配置弹窗合并** — 「启用开关」和「审核黑白名单」合并为同一「工具配置」弹窗，标签页切换，避免歧义\r\n- **自主模式 Follow-up 持续驱动** — Agent 自然终止后，通过 `OnNextTask` 回调自动注入 follow-up 消息，无需手动触发「继续」\r\n- **流式更新机制** — Registry 新增 `OnToolUpdate` 回调，工具执行中间结果实时推送给前端\r\n- **工具 UsageGuide 全覆盖** — 全部 ~140 个工具添加 `UsageGuide` 使用指导，明确何时用、为何优于 `run_command`、常见误区\r\n- **启动日志详细化** — 启动时输出版本号、Go 版本、平台架构、工作目录、各工作区文件夹路径\r\n\r\n### 改进\r\n- **工具体系升级**\r\n  - `Tool` 结构体新增 `UsageGuide`、`Category`、`Enabled` 字段\r\n  - `Registry` 新增 `EnabledDefinitions()` 按状态过滤工具定义\r\n  - 新增 `AllToolMeta()` API 供前端展示工具开关列表\r\n  - 工具使用指南文本动态注入系统提示，引导 LLM 优先使用专用工具\r\n- **窗口管理** — `run_command` / `run_background` 均设置 `HideWindow=true`，不再弹出 cmd 窗口\r\n- **信号监听移除** — main 函数移除信号监听，进程不会因子进程结束而自动退出\r\n\r\n### 修复\r\n- **debug_start 启动修复** — 拆解 `dlv dap` 启动流程，分别发送 Initialize 和 Launch 请求，兼容 dlv 最新版本\r\n\r\n---\r\n\r\n## 1.1.2 — 2026-07-21\r\n\r\n### 新增\r\n- **附件标签化** — 消息中的文件/代码/图片附件不再嵌入正文，改为独立药丸形标签显示在用户消息文字下方，视觉更清爽\r\n- **粘贴长文本自动转临时附件** — 输入框粘贴超过 2000 字符的文本时，自动写入 `_temp/` 目录并作为附件挂载，避免大段代码/日志撑爆输入区\r\n\r\n### 改进\r\n- `addToChat` 瘦身：文件添加到对话不再预读文件内容（40KB 截断已无意义），仅传递路径引用\r\n- 目录引用新增 `type:dir` 支持，提示 agent 使用 `list_files` 查看\r\n- 选中代码添加对话现在保留代码内容尾注供 agent 直接参考（截断 3000 字）\r\n\r\n### 修复\r\n- 文件树 Shift/Ctrl 多选逻辑修复：范围选择改为基于同级节点列表，清除后重新选中\r\n\r\n---\r\n\r\n## 1.1.1 — 2026-07-21\r\n\r\n### 新增\r\n- **审核配置界面重设计** — 从纯文本输入改为工具卡片式交互：所有工具按类别分组（文件操作、命令执行、Git、网络、截图、图像、二进制、办公文档、CodeGraph、调试器、知识库、记忆、LSP、BUG检测、任务管理、扩展市场等），每个工具显示中文名称，点击切换三态（默认 → 黑名单 → 白名单），支持搜索过滤，配置更直观高效\r\n\r\n### 修复\r\n- **修复新对话空状态提示位置偏移** — "开始新的对话，发送消息即可与 AI 助手对话"提示及图标从左下角偏移修正为居中显示\r\n\r\n### 改进\r\n- 版本号统一升级至 1.1.1（前端 package.json、后端 main.go、打包配置）\r\n\r\n---\r\n\r\n## 1.1.0 — 2026-07-20\r\n\r\n### 新增\r\n- **自主模式原生终止** — 去掉 `finish_task` 强制结束机制，Agent 自然输出后直接结束循环，交互更流畅\r\n- **Agent 性能优化（P0-P3 五轮）** — eventRing 环形缓冲器减少内存分配、进度可视化（阶段指示器+工具调用计数+耗时）、工具描述精简减少 Token 消耗、并行工具执行机制、预压缩上下文避免截断\r\n- **会话连贯性增强** — 新对话开始时自动注入 Git 变更感知、代码图谱统计、工作区结构概览，Agent 无需从零分析项目\r\n\r\n### 改进\r\n- **ChatView 重构** — 消息渲染管线全面优化，新增交互超时保护、审核驳回追踪、折叠/展开状态持久化\r\n- **审核配置 UI 优化** — 弹窗改为向上弹出（bottom:100%），防止被视口底部裁切\r\n- **编辑工具 v2 升级** — 更精确的符号级定位，减少行号偏移问题\r\n- **kill_process 增强** — 改为杀进程树，彻底清理子进程\r\n- **自主模式架构重构** — ephemeralMsgs 隔离内层消息，长时压缩精准保留推理上下文\r\n\r\n### 修复\r\n- 修复 `planExpanded` / `tasksExpanded` / `currentPhase` 重复声明导致的运行时崩溃\r\n- 修复 `currentTasks` 未声明导致前端 `undefined.length` 崩溃\r\n- 修复自然终止代码缩进丢失导致逻辑在循环外不执行\r\n\r\n---\r\n\r\n## 1.0.20 — 2026-07-18\r\n\r\n### 修复\r\n- **修复消息排序** — `_idx` 统一取 `max(existing)+1`，解决历史消息加载后序号错乱\r\n- **修复用户反馈消息合并** — 用户反馈正确合并到 agent 输出气泡中，不再产生额外用户消息气泡\r\n- **修复消息发送双占位竞态** — `switchConv` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，避免两个 assistant 气泡\r\n- **修复 WS 连接与历史加载竞态** — `processStatus` 事件正确处理连接状态转换\r\n\r\n### 改进\r\n- 审核配置弹窗改为向上弹出（`bottom:100%`），防止被视口底部裁切\r\n- 移除压缩按钮，简化 UI\r\n\r\n---\r\n\r\n## 1.0.19 — 2026-07-17\r\n\r\n### 修复\r\n- **修复 Web 端文件树不显示** — `FileExplorer.vue` 的 `<script setup>` 编译后 JS 中存在变量暂时性死区（TDZ），导致 `setup()` 抛出 `Cannot access \'d\' before initialization`，文件树组件挂载失败。重建前端并重新编译 `companion.exe` 嵌入新版 dist 后修复\r\n- **修复后端 dist 嵌入路径不一致** — `cmd/companion/main.go` 通过 `//go:embed web-ui/dist` 引用 companion 目录下的副本，但此前构建脚本将 dist 输出到 `cmd/desktop/web-ui/dist/`，两者不同步导致嵌入的仍是旧版 JS。统一构建流程后将新版 dist 正确复制到 `cmd/companion/web-ui/dist/`\r\n\r\n### 改进\r\n- 统一更新版本号至 1.0.19（后端 main.go、两个前端的 package.json）\r\n\r\n---\r\n\r\n## 1.0.8 — 2026-07-17\r\n\r\n### 新增\r\n- **多项目工作区支持** — 系统提示自动遍历所有工作区根目录，读取各自 `.pair/project.md` 环境配置注入给 AI，跨项目协作时准确感知每个项目的编译方式、CGO 开关等信息\r\n- **CodeGraph 多项目全量建图** — `codegraph_build` 支持对所有工作区项目建图并合并到同一个知识图谱（`rebuild=true`），跨项目符号搜索成为可能\r\n- **阻塞命令自动拦截** — 新增 `isBlockingCommand` 检测，自动拦截 dev server、watch 模式、`go run .`、`npm run dev` 等长期进程命令，提示改用 `run_background`，避免阻塞 AI 循环\r\n\r\n### 改进\r\n- **审核放行逻辑优化** — `run_command` 阻塞命令不再自动放行，强制走 LLM 审核；`run_background` 保持安全命令自动放行\r\n- **工具描述优化** — `run_command` 描述明确禁止长期进程并列出典型误用场景；`run_background` 强调作为长期进程首选工具\r\n- **系统提示增强** — 「错误恢复」和「防止卡死」两处加入阻塞/后台区分铁律，降低误用 `run_command` 概率\r\n\r\n---\r\n\r\n## 1.0.7 — 2026-07-17\r\n\r\n### 修复\r\n- **修复刷新页面后 ask_user 提交造成额外气泡** — 页面刷新后 `switchConv` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，不再另建新占位，避免两个 assistant 气泡\r\n\r\n### 改进\r\n- 统一更新版本号至 1.0.7（前端 package.json、后端 main.go、打包脚本）\r\n\r\n---\r\n\r\n## 1.0.6 — 2026-07-17\r\n\r\n### 修复\r\n- **修复消息持久化比较口径不一致** — `PersistNewMessages` 中 `persistedCount` 使用 `countJSONLLines`（统计文件总行数含 System），与 `histNonSystemCount`（统计非 System 消息数）口径不同，导致含 tool_call 的 assistant 消息在工具执行前被误判为"已落盘"而跳过写入。阻塞工具（如 ask_user）的前端始终无响应。改用 `readJSONL` 精确统计非 System 消息数\r\n- **修复对话/任务/执行计划 API 空实现** — `GET /api/conversations/{id}` 缺 agent 运行状态，`GET /api/tasks` 和 `GET /api/taskplan` 原返回对话列表（完全错误的 stub），改为返回真实数据\r\n\r\n---\r\n\r\n## 1.0.5 — 2026-07-17\r\n\r\n### 改进\r\n- **消息持久化重构** — `PersistNewMessages` 改为全量覆盖写 JSONL，消除 diff 计算的竞态问题；`MessageStore` 新增 `ReplaceHistory` 支持历史压缩；`MergeLastAssistantRun` 移除，各轮次独立存储以保留 reasoning 完整时序\r\n\r\n### 修复\r\n- **修复 send on closed channel panic** — 移除三处 `go func` 在无监听者时向 channel 发送导致的崩溃\r\n- **修复 PersistNewMessages 上下文压缩后新消息丢失** — 全量替换模式确保压缩后的摘要消息不被覆盖\r\n- **修复自动提交仅提交主工作区** — `doAutoCommit` 遍历所有工作区执行 git add + commit\r\n- **修复 idx 空洞导致消息跳过持久化** — `PersistNewMessages` 内部不再跳过 System/User 消息，确保序号连续\r\n\r\n---\r\n\r\n## 1.0.4 — 2026-07-17\r\n\r\n### 新增\r\n- **技能状态三级配置** — 技能可设为「关闭 / 按需加载 / 始终激活」三种模式，灵活控制 AI 行为\r\n- **市场安装范围选择** — 安装 MCP 服务器或技能时，支持选择 user（全局）或 project（项目级）范围\r\n\r\n### 改进\r\n- **对话历史持久化增强** — 页面刷新后对话完整恢复，不再因浏览器关闭丢失上下文；后端全面接管消息状态管理，前端不再依赖本地缓存\r\n- **消息展示优化** — 连续同一角色的消息自动合并显示（如多个 assistant 回复合并为一条），阅读更流畅\r\n- **停止信号可靠性提升** — Agent 异常结束或用户主动停止时，前端能可靠收到停止信号并更新 UI 状态\r\n\r\n### 修复\r\n- 修复切换对话时 loading 状态卡死的问题（switchConv 提前放行占位消息）\r\n- 修复消息历史顺序错乱和思考链（reasoning_content）丢失的严重问题\r\n- 修复 MergeConsecutiveAssistants 跳过 RoleTool 消息导致工具调用结果不完整的问题\r\n\r\n---\r\n\r\n## 1.0.3 — 2026-07-17\r\n\r\n### 改进\r\n- **子进程窗口管理** — 所有后台子进程（Git 操作、BUG 检测编译/测试、Lua 工具执行、桥接命令）统一隐藏控制台窗口，避免黑框闪烁\r\n- **会话持久化** — OnBatchPersist 回调从"每 5 轮"改为"每轮迭代"写盘，降低异常丢失风险\r\n- **代码搜索提示修复** — codegraph 搜索无结果时正确显示查询内容而非空占位符\r\n\r\n### 修复\r\n- **PersistNewMessages idx 空洞 bug** — 修复因跳过 System/User 角色消息导致消息序号不连续、后续消息无法正确持久化的严重问题（db_store.go + db_adapter.go）\r\n\r\n---\r\n\r\n## 1.0.2 — 2026-07-16\r\n\r\n### 改进\r\n- **文档同步** — features.md 同步到最新版本，移除冗余的"版本信息与更新日志"章节\r\n\r\n---\r\n\r\n## 1.0.1 — 2026-07-11\r\n\r\n### 新增\r\n- **更新日志页面** — 帮助文档中新增更新日志页面，版本历史一目了然\r\n- **WebSocket 协议文档** — API 文档补充完整 WebSocket 事件类型与负载定义\r\n- **系统版本报告** — `/api/system/info` 现在返回 `version` 字段，前端"关于"面板同步显示\r\n\r\n### 改进\r\n- **API 文档全面重写** — 每个接口增加请求体 JSON Schema、响应示例和错误码说明，便于二次开发\r\n- **帮助文档重构** — 文档归入"文档中心"分类，导航更清晰\r\n\r\n---\r\n\r\n## 1.0.0 — 2026-07-01\r\n\r\n### 新增\r\n- **AI 对话编程** — 用自然语言驱动 AI 读写文件、执行命令、管理 Git\r\n- **自主 Agent 模式** — AI 自动分析项目、制定计划并执行多步骤任务\r\n- **代码编辑器** — 内置多标签页编辑器，支持语法高亮、代码折叠、十六进制查看\r\n- **文件管理** — 工作区目录树浏览、文件搜索、批量操作\r\n- **Git 版本控制** — 对话驱动的 Git 操作（状态查看、暂存、提交、分支管理）\r\n- **内置终端** — 浏览器中的终端面板，支持 AI 自动执行命令\r\n- **对话历史管理** — 自动保存、回溯与继续历史对话\r\n- **BUG 自动检测修复** — AI 扫描编译/测试问题并自动修复\r\n- **Skills / MCP 扩展** — 可复用的工作流模板和模型上下文协议扩展\r\n- **记忆系统** — AI 跨会话记住用户偏好和历史决策\r\n- **任务与规划管理** — 复杂任务分解为可追踪的子步骤\r\n- **Lua 自定义工具** — 通过 Lua 脚本创建自定义 AI 工具\r\n- **代码知识图谱** — 函数调用关系、类型层次、影响范围分析\r\n- **多模型支持** — 灵活切换 AI 模型后端（OpenAI / Claude 等）\r\n- **主题系统** — 四套预设主题（暗色、白色、暖色、暗夜紫）\r\n- **调试器** — 支持 Go 程序的断点、单步和变量查看\r\n- **网页验证工具** — 自动打开 URL、截图、分析页面效果\r\n- **办公文档处理** — 读取 Word / Excel / PDF 文件，支持 OCR\r\n\r\n### 技术架构\r\n- 后端使用 Go 语言，前端使用 Vue 3 + CodeMirror\r\n- WebSocket 实时推送 AI 事件流\r\n- 内嵌前端资源（go:embed），单二进制分发\r\n- 纯本地运行，所有 API 仅监听本地回环地址\r\n';
  function M() {
    return { async: false, breaks: false, extensions: null, gfm: true, hooks: null, pedantic: false, renderer: null, silent: false, tokenizer: null, walkTokens: null };
  }
  var T = M();
  function N(l) {
    T = l;
  }
  var _ = { exec: () => null };
  function E(l) {
    let e = [];
    return (t) => {
      let n = Math.max(0, Math.min(3, t - 1)), s = e[n];
      return s || (s = l(n), e[n] = s), s;
    };
  }
  function d(l, e = "") {
    let t = typeof l == "string" ? l : l.source, n = { replace: (s, r) => {
      let i = typeof r == "string" ? r : r.source;
      return i = i.replace(m.caret, "$1"), t = t.replace(s, i), n;
    }, getRegex: () => new RegExp(t, e) };
    return n;
  }
  var Te = ((l = "") => {
    try {
      return !!new RegExp("(?<=1)(?<!1)" + l);
    } catch {
      return false;
    }
  })(), m = { codeRemoveIndent: /^(?: {1,4}| {0,3}\t)/gm, outputLinkReplace: /\\([\[\]])/g, indentCodeCompensation: /^(\s+)(?:```)/, beginningSpace: /^\s+/, endingHash: /#$/, startingSpaceChar: /^ /, endingSpaceChar: / $/, nonSpaceChar: /[^ ]/, newLineCharGlobal: /\n/g, tabCharGlobal: /\t/g, multipleSpaceGlobal: /\s+/g, blankLine: /^[ \t]*$/, doubleBlankLine: /\n[ \t]*\n[ \t]*$/, blockquoteStart: /^ {0,3}>/, blockquoteSetextReplace: /\n {0,3}((?:=+|-+) *)(?=\n|$)/g, blockquoteSetextReplace2: /^ {0,3}>[ \t]?/gm, listReplaceNesting: /^ {1,4}(?=( {4})*[^ ])/g, listIsTask: /^\[[ xX]\] +\S/, listReplaceTask: /^\[[ xX]\] +/, listTaskCheckbox: /\[[ xX]\]/, anyLine: /\n.*\n/, hrefBrackets: /^<(.*)>$/, tableDelimiter: /[:|]/, tableAlignChars: /^\||\| *$/g, tableRowBlankLine: /\n[ \t]*$/, tableAlignRight: /^ *-+: *$/, tableAlignCenter: /^ *:-+: *$/, tableAlignLeft: /^ *:-+ *$/, startATag: /^<a /i, endATag: /^<\/a>/i, startPreScriptTag: /^<(pre|code|kbd|script)(\s|>)/i, endPreScriptTag: /^<\/(pre|code|kbd|script)(\s|>)/i, startAngleBracket: /^</, endAngleBracket: />$/, pedanticHrefTitle: /^([^'"]*[^\s])\s+(['"])(.*)\2/, unicodeAlphaNumeric: /[\p{L}\p{N}]/u, escapeTest: /[&<>"']/, escapeReplace: /[&<>"']/g, escapeTestNoEncode: /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/, escapeReplaceNoEncode: /[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/g, caret: /(^|[^\[])\^/g, percentDecode: /%25/g, findPipe: /\|/g, splitPipe: / \|/, slashPipe: /\\\|/g, carriageReturn: /\r\n|\r/g, spaceLine: /^ +$/gm, notSpaceStart: /^\S*/, endingNewline: /\n$/, listItemRegex: (l) => new RegExp(`^( {0,3}${l})((?:[	 ][^\\n]*)?(?:\\n|$))`), nextBulletRegex: E((l) => new RegExp(`^ {0,${l}}(?:[*+-]|\\d{1,9}[.)])((?:[ 	][^\\n]*)?(?:\\n|$))`)), hrRegex: E((l) => new RegExp(`^ {0,${l}}((?:- *){3,}|(?:_ *){3,}|(?:\\* *){3,})(?:\\n+|$)`)), fencesBeginRegex: E((l) => new RegExp(`^ {0,${l}}(?:\`\`\`|~~~)`)), headingBeginRegex: E((l) => new RegExp(`^ {0,${l}}#`)), htmlBeginRegex: E((l) => new RegExp(`^ {0,${l}}<(?:[a-z].*>|!--)`, "i")), blockquoteBeginRegex: E((l) => new RegExp(`^ {0,${l}}>`)) }, Oe = /^(?:[ \t]*(?:\n|$))+/, we = /^((?: {4}| {0,3}\t)[^\n]+(?:\n(?:[ \t]*(?:\n|$))*)?)+/, ye = /^ {0,3}(`{3,}(?=[^`\n]*(?:\n|$))|~{3,})([^\n]*)(?:\n|$)(?:|([\s\S]*?)(?:\n|$))(?: {0,3}\1[~`]* *(?=\n|$)|$)/, B = /^ {0,3}((?:-[\t ]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})(?:\n+|$)/, Pe = /^ {0,3}(#{1,6})(?=\s|$)(.*)(?:\n+|$)/, j = / {0,3}(?:[*+-]|\d{1,9}[.)])/, oe = /^(?!bull |blockCode|fences|blockquote|heading|html|table)((?:.|\n(?!\s*?\n|bull |blockCode|fences|blockquote|heading|html|table))+?)\n {0,3}(=+|-+) *(?:\n+|$)/, ae = d(oe).replace(/bull/g, j).replace(/blockCode/g, /(?: {4}| {0,3}\t)/).replace(/fences/g, / {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g, / {0,3}>/).replace(/heading/g, / {0,3}#{1,6}/).replace(/html/g, / {0,3}<[^\n>]+>\n/).replace(/\|table/g, "").getRegex(), Se = d(oe).replace(/bull/g, j).replace(/blockCode/g, /(?: {4}| {0,3}\t)/).replace(/fences/g, / {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g, / {0,3}>/).replace(/heading/g, / {0,3}#{1,6}/).replace(/html/g, / {0,3}<[^\n>]+>\n/).replace(/table/g, / {0,3}\|?(?:[:\- ]*\|)+[\:\- ]*\n/).getRegex(), F = /^([^\n]+(?:\n(?!hr|heading|lheading|blockquote|fences|list|html|table| +\n)[^\n]+)*)/, $e = /^[^\n]+/, U = /(?!\s*\])(?:\\[\s\S]|[^\[\]\\])+/, Le = d(/^ {0,3}\[(label)\]: *(?:\n[ \t]*)?([^<\s][^\s]*|<.*?>)(?:(?: +(?:\n[ \t]*)?| *\n[ \t]*)(title))? *(?:\n+|$)/).replace("label", U).replace("title", /(?:"(?:\\"?|[^"\\])*"|'[^'\n]*(?:\n[^'\n]+)*\n?'|\([^()]*\))/).getRegex(), _e = d(/^(bull)([ \t][^\n]*?)?(?:\n|$)/).replace(/bull/g, j).getRegex(), H = "address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul", K = /<!--(?:-?>|[\s\S]*?(?:-->|$))/, ze = d("^ {0,3}(?:<(script|pre|style|textarea)[\\s>][\\s\\S]*?(?:</\\1>[^\\n]*\\n+|$)|comment[^\\n]*(\\n+|$)|<\\?[\\s\\S]*?(?:\\?>\\n*|$)|<![A-Z][\\s\\S]*?(?:>\\n*|$)|<!\\[CDATA\\[[\\s\\S]*?(?:\\]\\]>\\n*|$)|</?(tag)(?: +|\\n|/?>)[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|<(?!script|pre|style|textarea)([a-z][\\w-]*)(?:attribute)*? */?>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|</(?!script|pre|style|textarea)[a-z][\\w-]*\\s*>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$))", "i").replace("comment", K).replace("tag", H).replace("attribute", / +[a-zA-Z:_][\w.:-]*(?: *= *"[^"\n]*"| *= *'[^'\n]*'| *= *[^\s"'=<>`]+)?/).getRegex(), le = d(F).replace("hr", B).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("|lheading", "").replace("|table", "").replace("blockquote", " {0,3}>").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", H).getRegex(), Me = d(/^( {0,3}> ?(paragraph|[^\n]*)(?:\n|$))+/).replace("paragraph", le).getRegex(), W = { blockquote: Me, code: we, def: Le, fences: ye, heading: Pe, hr: B, html: ze, lheading: ae, list: _e, newline: Oe, paragraph: le, table: _, text: $e }, se = d("^ *([^\\n ].*)\\n {0,3}((?:\\| *)?:?-+:? *(?:\\| *:?-+:? *)*(?:\\| *)?)(?:\\n((?:(?! *\\n|hr|heading|blockquote|code|fences|list|html).*(?:\\n|$))*)\\n*|$)").replace("hr", B).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("blockquote", " {0,3}>").replace("code", "(?: {4}| {0,3}	)[^\\n]").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", H).getRegex(), Ee = { ...W, lheading: Se, table: se, paragraph: d(F).replace("hr", B).replace("heading", " {0,3}#{1,6}(?:\\s|$)").replace("|lheading", "").replace("table", se).replace("blockquote", " {0,3}>").replace("fences", " {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list", " {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html", "</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag", H).getRegex() }, Ie = { ...W, html: d(`^ *(?:comment *(?:\\n|\\s*$)|<(tag)[\\s\\S]+?</\\1> *(?:\\n{2,}|\\s*$)|<tag(?:"[^"]*"|'[^']*'|\\s[^'"/>\\s]*)*?/?> *(?:\\n{2,}|\\s*$))`).replace("comment", K).replace(/tag/g, "(?!(?:a|em|strong|small|s|cite|q|dfn|abbr|data|time|code|var|samp|kbd|sub|sup|i|b|u|mark|ruby|rt|rp|bdi|bdo|span|br|wbr|ins|del|img)\\b)\\w+(?!:|[^\\w\\s@]*@)\\b").getRegex(), def: /^ *\[([^\]]+)\]: *<?([^\s>]+)>?(?: +(["(][^\n]+[")]))? *(?:\n+|$)/, heading: /^(#{1,6})(.*)(?:\n+|$)/, fences: _, lheading: /^(.+?)\n {0,3}(=+|-+) *(?:\n+|$)/, paragraph: d(F).replace("hr", B).replace("heading", ` *#{1,6} *[^
]`).replace("lheading", ae).replace("|table", "").replace("blockquote", " {0,3}>").replace("|fences", "").replace("|list", "").replace("|html", "").replace("|tag", "").getRegex() }, Ae = /^\\([!"#$%&'()*+,\-./:;<=>?@\[\]\\^_`{|}~])/, Ce = /^(`+)([^`]|[^`][\s\S]*?[^`])\1(?!`)/, ue = /^( {2,}|\\)\n(?!\s*$)/, Be = /^(`+|[^`])(?:(?= {2,}\n)|[\s\S]*?(?:(?=[\\<!\[`*_]|\b_|$)|[^ ](?= {2,}\n)))/, I = /[\p{P}\p{S}]/u, Z = /[\s\p{P}\p{S}]/u, X = /[^\s\p{P}\p{S}]/u, De = d(/^((?![*_])punctSpace)/, "u").replace(/punctSpace/g, Z).getRegex(), pe = /(?!~)[\p{P}\p{S}]/u, qe = /(?!~)[\s\p{P}\p{S}]/u, ve = /(?:[^\s\p{P}\p{S}]|~)/u, He = d(/link|precode-code|html/, "g").replace("link", /\[(?:[^\[\]`]|(?<a>`+)[^`]+\k<a>(?!`))*?\]\((?:\\[\s\S]|[^\\\(\)]|\((?:\\[\s\S]|[^\\\(\)])*\))*\)/).replace("precode-", Te ? "(?<!`)()" : "(^^|[^`])").replace("code", /(?<b>`+)[^`]+\k<b>(?!`)/).replace("html", /<(?! )[^<>]*?>/).getRegex(), ce = /^(?:\*+(?:((?!\*)punct)|([^\s*]))?)|^_+(?:((?!_)punct)|([^\s_]))?/, Ze = d(ce, "u").replace(/punct/g, I).getRegex(), Ge = d(ce, "u").replace(/punct/g, pe).getRegex(), he = "^[^_*]*?__[^_*]*?\\*[^_*]*?(?=__)|[^*]+(?=[^*])|(?!\\*)punct(\\*+)(?=[\\s]|$)|notPunctSpace(\\*+)(?!\\*)(?=punctSpace|$)|(?!\\*)punctSpace(\\*+)(?=notPunctSpace)|[\\s](\\*+)(?!\\*)(?=punct)|(?!\\*)punct(\\*+)(?!\\*)(?=punct)|notPunctSpace(\\*+)(?=notPunctSpace)", Ne = d(he, "gu").replace(/notPunctSpace/g, X).replace(/punctSpace/g, Z).replace(/punct/g, I).getRegex(), Qe = d(he, "gu").replace(/notPunctSpace/g, ve).replace(/punctSpace/g, qe).replace(/punct/g, pe).getRegex(), je = d("^[^_*]*?\\*\\*[^_*]*?_[^_*]*?(?=\\*\\*)|[^_]+(?=[^_])|(?!_)punct(_+)(?=[\\s]|$)|notPunctSpace(_+)(?!_)(?=punctSpace|$)|(?!_)punctSpace(_+)(?=notPunctSpace)|[\\s](_+)(?!_)(?=punct)|(?!_)punct(_+)(?!_)(?=punct)", "gu").replace(/notPunctSpace/g, X).replace(/punctSpace/g, Z).replace(/punct/g, I).getRegex(), Fe = d(/^~~?(?:((?!~)punct)|[^\s~])/, "u").replace(/punct/g, I).getRegex(), Ue = "^[^~]+(?=[^~])|(?!~)punct(~~?)(?=[\\s]|$)|notPunctSpace(~~?)(?!~)(?=punctSpace|$)|(?!~)punctSpace(~~?)(?=notPunctSpace)|[\\s](~~?)(?!~)(?=punct)|(?!~)punct(~~?)(?!~)(?=punct)|notPunctSpace(~~?)(?=notPunctSpace)", Ke = d(Ue, "gu").replace(/notPunctSpace/g, X).replace(/punctSpace/g, Z).replace(/punct/g, I).getRegex(), We = d(/\\(punct)/, "gu").replace(/punct/g, I).getRegex(), Xe = d(/^<(scheme:[^\s\x00-\x1f<>]*|email)>/).replace("scheme", /[a-zA-Z][a-zA-Z0-9+.-]{1,31}/).replace("email", /[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+(@)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+(?![-_])/).getRegex(), Je = d(K).replace("(?:-->|$)", "-->").getRegex(), Ve = d("^comment|^</[a-zA-Z][\\w:-]*\\s*>|^<[a-zA-Z][\\w-]*(?:attribute)*?\\s*/?>|^<\\?[\\s\\S]*?\\?>|^<![a-zA-Z]+\\s[\\s\\S]*?>|^<!\\[CDATA\\[[\\s\\S]*?\\]\\]>").replace("comment", Je).replace("attribute", /\s+[a-zA-Z:_][\w.:-]*(?:\s*=\s*"[^"]*"|\s*=\s*'[^']*'|\s*=\s*[^\s"'=<>`]+)?/).getRegex(), v = /(?:\[(?:\\[\s\S]|[^\[\]\\])*\]|\\[\s\S]|`+(?!`)[^`]*?`+(?!`)|``+(?=\])|[^\[\]\\`])*?/, Ye = d(/^!?\[(label)\]\(\s*(href)(?:(?:[ \t]+(?:\n[ \t]*)?|\n[ \t]*)(title))?\s*\)/).replace("label", v).replace("href", /<(?:\\.|[^\n<>\\])+>|[^ \t\n\x00-\x1f]*/).replace("title", /"(?:\\"?|[^"\\])*"|'(?:\\'?|[^'\\])*'|\((?:\\\)?|[^)\\])*\)/).getRegex(), ke = d(/^!?\[(label)\]\[(ref)\]/).replace("label", v).replace("ref", U).getRegex(), de = d(/^!?\[(ref)\](?:\[\])?/).replace("ref", U).getRegex(), et = d("reflink|nolink(?!\\()", "g").replace("reflink", ke).replace("nolink", de).getRegex(), ie = /[hH][tT][tT][pP][sS]?|[fF][tT][pP]/, J = { _backpedal: _, anyPunctuation: We, autolink: Xe, blockSkip: He, br: ue, code: Ce, del: _, delLDelim: _, delRDelim: _, emStrongLDelim: Ze, emStrongRDelimAst: Ne, emStrongRDelimUnd: je, escape: Ae, link: Ye, nolink: de, punctuation: De, reflink: ke, reflinkSearch: et, tag: Ve, text: Be, url: _ }, tt = { ...J, link: d(/^!?\[(label)\]\((.*?)\)/).replace("label", v).getRegex(), reflink: d(/^!?\[(label)\]\s*\[([^\]]*)\]/).replace("label", v).getRegex() }, Q = { ...J, emStrongRDelimAst: Qe, emStrongLDelim: Ge, delLDelim: Fe, delRDelim: Ke, url: d(/^((?:protocol):\/\/|www\.)(?:[a-zA-Z0-9\-]+\.?)+[^\s<]*|^email/).replace("protocol", ie).replace("email", /[A-Za-z0-9._+-]+(@)[a-zA-Z0-9-_]+(?:\.[a-zA-Z0-9-_]*[a-zA-Z0-9])+(?![-_])/).getRegex(), _backpedal: /(?:[^?!.,:;*_'"~()&]+|\([^)]*\)|&(?![a-zA-Z0-9]+;$)|[?!.,:;*_'"~)]+(?!$))+/, del: /^(~~?)(?=[^\s~])((?:\\[\s\S]|[^\\])*?(?:\\[\s\S]|[^\s~\\]))\1(?=[^~]|$)/, text: d(/^([`~]+|[^`~])(?:(?= {2,}\n)|(?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)|[\s\S]*?(?:(?=[\\<!\[`*~_]|\b_|protocol:\/\/|www\.|$)|[^ ](?= {2,}\n)|[^a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-](?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)))/).replace("protocol", ie).getRegex() }, nt = { ...Q, br: d(ue).replace("{2,}", "*").getRegex(), text: d(Q.text).replace("\\b_", "\\b_| {2,}\\n").replace(/\{2,\}/g, "*").getRegex() }, D = { normal: W, gfm: Ee, pedantic: Ie }, A = { normal: J, gfm: Q, breaks: nt, pedantic: tt };
  var rt = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }, ge = (l) => rt[l];
  function O(l, e) {
    if (e) {
      if (m.escapeTest.test(l)) return l.replace(m.escapeReplace, ge);
    } else if (m.escapeTestNoEncode.test(l)) return l.replace(m.escapeReplaceNoEncode, ge);
    return l;
  }
  function V(l) {
    try {
      l = encodeURI(l).replace(m.percentDecode, "%");
    } catch {
      return null;
    }
    return l;
  }
  function Y(l, e) {
    var _a2;
    let t = l.replace(m.findPipe, (r, i, o) => {
      let u = false, a = i;
      for (; --a >= 0 && o[a] === "\\"; ) u = !u;
      return u ? "|" : " |";
    }), n = t.split(m.splitPipe), s = 0;
    if (n[0].trim() || n.shift(), n.length > 0 && !((_a2 = n.at(-1)) == null ? void 0 : _a2.trim()) && n.pop(), e) if (n.length > e) n.splice(e);
    else for (; n.length < e; ) n.push("");
    for (; s < n.length; s++) n[s] = n[s].trim().replace(m.slashPipe, "|");
    return n;
  }
  function $(l, e, t) {
    let n = l.length;
    if (n === 0) return "";
    let s = 0;
    for (; s < n; ) {
      let r = l.charAt(n - s - 1);
      if (r === e && true) s++;
      else break;
    }
    return l.slice(0, n - s);
  }
  function ee(l) {
    let e = l.split(`
`), t = e.length - 1;
    for (; t >= 0 && m.blankLine.test(e[t]); ) t--;
    return e.length - t <= 2 ? l : e.slice(0, t + 1).join(`
`);
  }
  function fe(l, e) {
    if (l.indexOf(e[1]) === -1) return -1;
    let t = 0;
    for (let n = 0; n < l.length; n++) if (l[n] === "\\") n++;
    else if (l[n] === e[0]) t++;
    else if (l[n] === e[1] && (t--, t < 0)) return n;
    return t > 0 ? -2 : -1;
  }
  function me(l, e = 0) {
    let t = e, n = "";
    for (let s of l) if (s === "	") {
      let r = 4 - t % 4;
      n += " ".repeat(r), t += r;
    } else n += s, t++;
    return n;
  }
  function xe(l, e, t, n, s) {
    let r = e.href, i = e.title || null, o = l[1].replace(s.other.outputLinkReplace, "$1");
    n.state.inLink = true;
    let u = { type: l[0].charAt(0) === "!" ? "image" : "link", raw: t, href: r, title: i, text: o, tokens: n.inlineTokens(o) };
    return n.state.inLink = false, u;
  }
  function st(l, e, t) {
    let n = l.match(t.other.indentCodeCompensation);
    if (n === null) return e;
    let s = n[1];
    return e.split(`
`).map((r) => {
      let i = r.match(t.other.beginningSpace);
      if (i === null) return r;
      let [o] = i;
      return o.length >= s.length ? r.slice(s.length) : r;
    }).join(`
`);
  }
  var w = class {
    constructor(e) {
      __publicField(this, "options");
      __publicField(this, "rules");
      __publicField(this, "lexer");
      this.options = e || T;
    }
    space(e) {
      let t = this.rules.block.newline.exec(e);
      if (t && t[0].length > 0) return { type: "space", raw: t[0] };
    }
    code(e) {
      let t = this.rules.block.code.exec(e);
      if (t) {
        let n = this.options.pedantic ? t[0] : ee(t[0]), s = n.replace(this.rules.other.codeRemoveIndent, "");
        return { type: "code", raw: n, codeBlockStyle: "indented", text: s };
      }
    }
    fences(e) {
      let t = this.rules.block.fences.exec(e);
      if (t) {
        let n = t[0], s = st(n, t[3] || "", this.rules);
        return { type: "code", raw: n, lang: t[2] ? t[2].trim().replace(this.rules.inline.anyPunctuation, "$1") : t[2], text: s };
      }
    }
    heading(e) {
      let t = this.rules.block.heading.exec(e);
      if (t) {
        let n = t[2].trim();
        if (this.rules.other.endingHash.test(n)) {
          let s = $(n, "#");
          (this.options.pedantic || !s || this.rules.other.endingSpaceChar.test(s)) && (n = s.trim());
        }
        return { type: "heading", raw: $(t[0], `
`), depth: t[1].length, text: n, tokens: this.lexer.inline(n) };
      }
    }
    hr(e) {
      let t = this.rules.block.hr.exec(e);
      if (t) return { type: "hr", raw: $(t[0], `
`) };
    }
    blockquote(e) {
      let t = this.rules.block.blockquote.exec(e);
      if (t) {
        let n = $(t[0], `
`).split(`
`), s = "", r = "", i = [];
        for (; n.length > 0; ) {
          let o = false, u = [], a;
          for (a = 0; a < n.length; a++) if (this.rules.other.blockquoteStart.test(n[a])) u.push(n[a]), o = true;
          else if (!o) u.push(n[a]);
          else break;
          n = n.slice(a);
          let c = u.join(`
`), p = c.replace(this.rules.other.blockquoteSetextReplace, `
    $1`).replace(this.rules.other.blockquoteSetextReplace2, "");
          s = s ? `${s}
${c}` : c, r = r ? `${r}
${p}` : p;
          let k = this.lexer.state.top;
          if (this.lexer.state.top = true, this.lexer.blockTokens(p, i, true), this.lexer.state.top = k, n.length === 0) break;
          let h = i.at(-1);
          if ((h == null ? void 0 : h.type) === "code") break;
          if ((h == null ? void 0 : h.type) === "blockquote") {
            let R = h, f = R.raw + `
` + n.join(`
`), S = this.blockquote(f);
            i[i.length - 1] = S, s = s.substring(0, s.length - R.raw.length) + S.raw, r = r.substring(0, r.length - R.text.length) + S.text;
            break;
          } else if ((h == null ? void 0 : h.type) === "list") {
            let R = h, f = R.raw + `
` + n.join(`
`), S = this.list(f);
            i[i.length - 1] = S, s = s.substring(0, s.length - h.raw.length) + S.raw, r = r.substring(0, r.length - R.raw.length) + S.raw, n = f.substring(i.at(-1).raw.length).split(`
`);
            continue;
          }
        }
        return { type: "blockquote", raw: s, tokens: i, text: r };
      }
    }
    list(e) {
      let t = this.rules.block.list.exec(e);
      if (t) {
        let n = t[1].trim(), s = n.length > 1, r = { type: "list", raw: "", ordered: s, start: s ? +n.slice(0, -1) : "", loose: false, items: [] };
        n = s ? `\\d{1,9}\\${n.slice(-1)}` : `\\${n}`, this.options.pedantic && (n = s ? n : "[*+-]");
        let i = this.rules.other.listItemRegex(n), o = false;
        for (; e; ) {
          let a = false, c = "", p = "";
          if (!(t = i.exec(e)) || this.rules.block.hr.test(e)) break;
          c = t[0], e = e.substring(c.length);
          let k = me(t[2].split(`
`, 1)[0], t[1].length), h = e.split(`
`, 1)[0], R = !k.trim(), f = 0;
          if (this.options.pedantic ? (f = 2, p = k.trimStart()) : R ? f = t[1].length + 1 : (f = k.search(this.rules.other.nonSpaceChar), f = f > 4 ? 1 : f, p = k.slice(f), f += t[1].length), R && this.rules.other.blankLine.test(h) && (c += h + `
`, e = e.substring(h.length + 1), a = true), !a) {
            let S = this.rules.other.nextBulletRegex(f), te = this.rules.other.hrRegex(f), ne = this.rules.other.fencesBeginRegex(f), re = this.rules.other.headingBeginRegex(f), be = this.rules.other.htmlBeginRegex(f), Re = this.rules.other.blockquoteBeginRegex(f);
            for (; e; ) {
              let G = e.split(`
`, 1)[0], C;
              if (h = G, this.options.pedantic ? (h = h.replace(this.rules.other.listReplaceNesting, "  "), C = h) : C = h.replace(this.rules.other.tabCharGlobal, "    "), ne.test(h) || re.test(h) || be.test(h) || Re.test(h) || S.test(h) || te.test(h)) break;
              if (C.search(this.rules.other.nonSpaceChar) >= f || !h.trim()) p += `
` + C.slice(f);
              else {
                if (R || k.replace(this.rules.other.tabCharGlobal, "    ").search(this.rules.other.nonSpaceChar) >= 4 || ne.test(k) || re.test(k) || te.test(k)) break;
                p += `
` + h;
              }
              R = !h.trim(), c += G + `
`, e = e.substring(G.length + 1), k = C.slice(f);
            }
          }
          r.loose || (o ? r.loose = true : this.rules.other.doubleBlankLine.test(c) && (o = true)), r.items.push({ type: "list_item", raw: c, task: !!this.options.gfm && this.rules.other.listIsTask.test(p), loose: false, text: p, tokens: [] }), r.raw += c;
        }
        let u = r.items.at(-1);
        if (u) u.raw = u.raw.trimEnd(), u.text = u.text.trimEnd();
        else return;
        r.raw = r.raw.trimEnd();
        for (let a of r.items) {
          this.lexer.state.top = false, a.tokens = this.lexer.blockTokens(a.text, []);
          let c = a.tokens[0];
          if (a.task && ((c == null ? void 0 : c.type) === "text" || (c == null ? void 0 : c.type) === "paragraph")) {
            a.text = a.text.replace(this.rules.other.listReplaceTask, ""), c.raw = c.raw.replace(this.rules.other.listReplaceTask, ""), c.text = c.text.replace(this.rules.other.listReplaceTask, "");
            for (let k = this.lexer.inlineQueue.length - 1; k >= 0; k--) if (this.rules.other.listIsTask.test(this.lexer.inlineQueue[k].src)) {
              this.lexer.inlineQueue[k].src = this.lexer.inlineQueue[k].src.replace(this.rules.other.listReplaceTask, "");
              break;
            }
            let p = this.rules.other.listTaskCheckbox.exec(a.raw);
            if (p) {
              let k = { type: "checkbox", raw: p[0] + " ", checked: p[0] !== "[ ]" };
              a.checked = k.checked, r.loose ? a.tokens[0] && ["paragraph", "text"].includes(a.tokens[0].type) && "tokens" in a.tokens[0] && a.tokens[0].tokens ? (a.tokens[0].raw = k.raw + a.tokens[0].raw, a.tokens[0].text = k.raw + a.tokens[0].text, a.tokens[0].tokens.unshift(k)) : a.tokens.unshift({ type: "paragraph", raw: k.raw, text: k.raw, tokens: [k] }) : a.tokens.unshift(k);
            }
          } else a.task && (a.task = false);
          if (!r.loose) {
            let p = a.tokens.filter((h) => h.type === "space"), k = p.length > 0 && p.some((h) => this.rules.other.anyLine.test(h.raw));
            r.loose = k;
          }
        }
        if (r.loose) for (let a of r.items) {
          a.loose = true;
          for (let c of a.tokens) c.type === "text" && (c.type = "paragraph");
        }
        return r;
      }
    }
    html(e) {
      let t = this.rules.block.html.exec(e);
      if (t) {
        let n = ee(t[0]);
        return { type: "html", block: true, raw: n, pre: t[1] === "pre" || t[1] === "script" || t[1] === "style", text: n };
      }
    }
    def(e) {
      let t = this.rules.block.def.exec(e);
      if (t) {
        let n = t[1].toLowerCase().replace(this.rules.other.multipleSpaceGlobal, " "), s = t[2] ? t[2].replace(this.rules.other.hrefBrackets, "$1").replace(this.rules.inline.anyPunctuation, "$1") : "", r = t[3] ? t[3].substring(1, t[3].length - 1).replace(this.rules.inline.anyPunctuation, "$1") : t[3];
        return { type: "def", tag: n, raw: $(t[0], `
`), href: s, title: r };
      }
    }
    table(e) {
      var _a2;
      let t = this.rules.block.table.exec(e);
      if (!t || !this.rules.other.tableDelimiter.test(t[2])) return;
      let n = Y(t[1]), s = t[2].replace(this.rules.other.tableAlignChars, "").split("|"), r = ((_a2 = t[3]) == null ? void 0 : _a2.trim()) ? t[3].replace(this.rules.other.tableRowBlankLine, "").split(`
`) : [], i = { type: "table", raw: $(t[0], `
`), header: [], align: [], rows: [] };
      if (n.length === s.length) {
        for (let o of s) this.rules.other.tableAlignRight.test(o) ? i.align.push("right") : this.rules.other.tableAlignCenter.test(o) ? i.align.push("center") : this.rules.other.tableAlignLeft.test(o) ? i.align.push("left") : i.align.push(null);
        for (let o = 0; o < n.length; o++) i.header.push({ text: n[o], tokens: this.lexer.inline(n[o]), header: true, align: i.align[o] });
        for (let o of r) i.rows.push(Y(o, i.header.length).map((u, a) => ({ text: u, tokens: this.lexer.inline(u), header: false, align: i.align[a] })));
        return i;
      }
    }
    lheading(e) {
      let t = this.rules.block.lheading.exec(e);
      if (t) {
        let n = t[1].trim();
        return { type: "heading", raw: $(t[0], `
`), depth: t[2].charAt(0) === "=" ? 1 : 2, text: n, tokens: this.lexer.inline(n) };
      }
    }
    paragraph(e) {
      let t = this.rules.block.paragraph.exec(e);
      if (t) {
        let n = t[1].charAt(t[1].length - 1) === `
` ? t[1].slice(0, -1) : t[1];
        return { type: "paragraph", raw: t[0], text: n, tokens: this.lexer.inline(n) };
      }
    }
    text(e) {
      let t = this.rules.block.text.exec(e);
      if (t) return { type: "text", raw: t[0], text: t[0], tokens: this.lexer.inline(t[0]) };
    }
    escape(e) {
      let t = this.rules.inline.escape.exec(e);
      if (t) return { type: "escape", raw: t[0], text: t[1] };
    }
    tag(e) {
      let t = this.rules.inline.tag.exec(e);
      if (t) return !this.lexer.state.inLink && this.rules.other.startATag.test(t[0]) ? this.lexer.state.inLink = true : this.lexer.state.inLink && this.rules.other.endATag.test(t[0]) && (this.lexer.state.inLink = false), !this.lexer.state.inRawBlock && this.rules.other.startPreScriptTag.test(t[0]) ? this.lexer.state.inRawBlock = true : this.lexer.state.inRawBlock && this.rules.other.endPreScriptTag.test(t[0]) && (this.lexer.state.inRawBlock = false), { type: "html", raw: t[0], inLink: this.lexer.state.inLink, inRawBlock: this.lexer.state.inRawBlock, block: false, text: t[0] };
    }
    link(e) {
      let t = this.rules.inline.link.exec(e);
      if (t) {
        let n = t[2].trim();
        if (!this.options.pedantic && this.rules.other.startAngleBracket.test(n)) {
          if (!this.rules.other.endAngleBracket.test(n)) return;
          let i = $(n.slice(0, -1), "\\");
          if ((n.length - i.length) % 2 === 0) return;
        } else {
          let i = fe(t[2], "()");
          if (i === -2) return;
          if (i > -1) {
            let u = (t[0].indexOf("!") === 0 ? 5 : 4) + t[1].length + i;
            t[2] = t[2].substring(0, i), t[0] = t[0].substring(0, u).trim(), t[3] = "";
          }
        }
        let s = t[2], r = "";
        if (this.options.pedantic) {
          let i = this.rules.other.pedanticHrefTitle.exec(s);
          i && (s = i[1], r = i[3]);
        } else r = t[3] ? t[3].slice(1, -1) : "";
        return s = s.trim(), this.rules.other.startAngleBracket.test(s) && (this.options.pedantic && !this.rules.other.endAngleBracket.test(n) ? s = s.slice(1) : s = s.slice(1, -1)), xe(t, { href: s && s.replace(this.rules.inline.anyPunctuation, "$1"), title: r && r.replace(this.rules.inline.anyPunctuation, "$1") }, t[0], this.lexer, this.rules);
      }
    }
    reflink(e, t) {
      let n;
      if ((n = this.rules.inline.reflink.exec(e)) || (n = this.rules.inline.nolink.exec(e))) {
        let s = (n[2] || n[1]).replace(this.rules.other.multipleSpaceGlobal, " "), r = t[s.toLowerCase()];
        if (!r) {
          let i = n[0].charAt(0);
          return { type: "text", raw: i, text: i };
        }
        return xe(n, r, n[0], this.lexer, this.rules);
      }
    }
    emStrong(e, t, n = "") {
      let s = this.rules.inline.emStrongLDelim.exec(e);
      if (!s || !s[1] && !s[2] && !s[3] && !s[4] || s[4] && n.match(this.rules.other.unicodeAlphaNumeric)) return;
      if (!(s[1] || s[3] || "") || !n || this.rules.inline.punctuation.exec(n)) {
        let i = [...s[0]].length - 1, o, u, a = i, c = 0, p = s[0][0] === "*" ? this.rules.inline.emStrongRDelimAst : this.rules.inline.emStrongRDelimUnd;
        for (p.lastIndex = 0, t = t.slice(-1 * e.length + i); (s = p.exec(t)) !== null; ) {
          if (o = s[1] || s[2] || s[3] || s[4] || s[5] || s[6], !o) continue;
          if (u = [...o].length, s[3] || s[4]) {
            a += u;
            continue;
          } else if ((s[5] || s[6]) && i % 3 && !((i + u) % 3)) {
            c += u;
            continue;
          }
          if (a -= u, a > 0) continue;
          u = Math.min(u, u + a + c);
          let k = [...s[0]][0].length, h = e.slice(0, i + s.index + k + u);
          if (Math.min(i, u) % 2) {
            let f = h.slice(1, -1);
            return { type: "em", raw: h, text: f, tokens: this.lexer.inlineTokens(f) };
          }
          let R = h.slice(2, -2);
          return { type: "strong", raw: h, text: R, tokens: this.lexer.inlineTokens(R) };
        }
      }
    }
    codespan(e) {
      let t = this.rules.inline.code.exec(e);
      if (t) {
        let n = t[2].replace(this.rules.other.newLineCharGlobal, " "), s = this.rules.other.nonSpaceChar.test(n), r = this.rules.other.startingSpaceChar.test(n) && this.rules.other.endingSpaceChar.test(n);
        return s && r && (n = n.substring(1, n.length - 1)), { type: "codespan", raw: t[0], text: n };
      }
    }
    br(e) {
      let t = this.rules.inline.br.exec(e);
      if (t) return { type: "br", raw: t[0] };
    }
    del(e, t, n = "") {
      let s = this.rules.inline.delLDelim.exec(e);
      if (!s) return;
      if (!(s[1] || "") || !n || this.rules.inline.punctuation.exec(n)) {
        let i = [...s[0]].length - 1, o, u, a = i, c = this.rules.inline.delRDelim;
        for (c.lastIndex = 0, t = t.slice(-1 * e.length + i); (s = c.exec(t)) !== null; ) {
          if (o = s[1] || s[2] || s[3] || s[4] || s[5] || s[6], !o || (u = [...o].length, u !== i)) continue;
          if (s[3] || s[4]) {
            a += u;
            continue;
          }
          if (a -= u, a > 0) continue;
          u = Math.min(u, u + a);
          let p = [...s[0]][0].length, k = e.slice(0, i + s.index + p + u), h = k.slice(i, -i);
          return { type: "del", raw: k, text: h, tokens: this.lexer.inlineTokens(h) };
        }
      }
    }
    autolink(e) {
      let t = this.rules.inline.autolink.exec(e);
      if (t) {
        let n, s;
        return t[2] === "@" ? (n = t[1], s = "mailto:" + n) : (n = t[1], s = n), { type: "link", raw: t[0], text: n, href: s, tokens: [{ type: "text", raw: n, text: n }] };
      }
    }
    url(e) {
      var _a2;
      let t;
      if (t = this.rules.inline.url.exec(e)) {
        let n, s;
        if (t[2] === "@") n = t[0], s = "mailto:" + n;
        else {
          let r;
          do
            r = t[0], t[0] = ((_a2 = this.rules.inline._backpedal.exec(t[0])) == null ? void 0 : _a2[0]) ?? "";
          while (r !== t[0]);
          n = t[0], t[1] === "www." ? s = "http://" + t[0] : s = t[0];
        }
        return { type: "link", raw: t[0], text: n, href: s, tokens: [{ type: "text", raw: n, text: n }] };
      }
    }
    inlineText(e) {
      let t = this.rules.inline.text.exec(e);
      if (t) {
        let n = this.lexer.state.inRawBlock;
        return { type: "text", raw: t[0], text: t[0], escaped: n };
      }
    }
  };
  var x = class l {
    constructor(e) {
      __publicField(this, "tokens");
      __publicField(this, "options");
      __publicField(this, "state");
      __publicField(this, "inlineQueue");
      __publicField(this, "tokenizer");
      this.tokens = [], this.tokens.links = /* @__PURE__ */ Object.create(null), this.options = e || T, this.options.tokenizer = this.options.tokenizer || new w(), this.tokenizer = this.options.tokenizer, this.tokenizer.options = this.options, this.tokenizer.lexer = this, this.inlineQueue = [], this.state = { inLink: false, inRawBlock: false, top: true };
      let t = { other: m, block: D.normal, inline: A.normal };
      this.options.pedantic ? (t.block = D.pedantic, t.inline = A.pedantic) : this.options.gfm && (t.block = D.gfm, this.options.breaks ? t.inline = A.breaks : t.inline = A.gfm), this.tokenizer.rules = t;
    }
    static get rules() {
      return { block: D, inline: A };
    }
    static lex(e, t) {
      return new l(t).lex(e);
    }
    static lexInline(e, t) {
      return new l(t).inlineTokens(e);
    }
    lex(e) {
      e = e.replace(m.carriageReturn, `
`), this.blockTokens(e, this.tokens);
      for (let t = 0; t < this.inlineQueue.length; t++) {
        let n = this.inlineQueue[t];
        this.inlineTokens(n.src, n.tokens);
      }
      return this.inlineQueue = [], this.tokens;
    }
    blockTokens(e, t = [], n = false) {
      var _a2, _b, _c;
      this.tokenizer.lexer = this, this.options.pedantic && (e = e.replace(m.tabCharGlobal, "    ").replace(m.spaceLine, ""));
      let s = 1 / 0;
      for (; e; ) {
        if (e.length < s) s = e.length;
        else {
          this.infiniteLoopError(e.charCodeAt(0));
          break;
        }
        let r;
        if ((_b = (_a2 = this.options.extensions) == null ? void 0 : _a2.block) == null ? void 0 : _b.some((o) => (r = o.call({ lexer: this }, e, t)) ? (e = e.substring(r.raw.length), t.push(r), true) : false)) continue;
        if (r = this.tokenizer.space(e)) {
          e = e.substring(r.raw.length);
          let o = t.at(-1);
          r.raw.length === 1 && o !== void 0 ? o.raw += `
` : t.push(r);
          continue;
        }
        if (r = this.tokenizer.code(e)) {
          e = e.substring(r.raw.length);
          let o = t.at(-1);
          (o == null ? void 0 : o.type) === "paragraph" || (o == null ? void 0 : o.type) === "text" ? (o.raw += (o.raw.endsWith(`
`) ? "" : `
`) + r.raw, o.text += `
` + r.text, this.inlineQueue.at(-1).src = o.text) : t.push(r);
          continue;
        }
        if (r = this.tokenizer.fences(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.heading(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.hr(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.blockquote(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.list(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.html(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.def(e)) {
          e = e.substring(r.raw.length);
          let o = t.at(-1);
          (o == null ? void 0 : o.type) === "paragraph" || (o == null ? void 0 : o.type) === "text" ? (o.raw += (o.raw.endsWith(`
`) ? "" : `
`) + r.raw, o.text += `
` + r.raw, this.inlineQueue.at(-1).src = o.text) : this.tokens.links[r.tag] || (this.tokens.links[r.tag] = { href: r.href, title: r.title }, t.push(r));
          continue;
        }
        if (r = this.tokenizer.table(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        if (r = this.tokenizer.lheading(e)) {
          e = e.substring(r.raw.length), t.push(r);
          continue;
        }
        let i = e;
        if ((_c = this.options.extensions) == null ? void 0 : _c.startBlock) {
          let o = 1 / 0, u = e.slice(1), a;
          this.options.extensions.startBlock.forEach((c) => {
            a = c.call({ lexer: this }, u), typeof a == "number" && a >= 0 && (o = Math.min(o, a));
          }), o < 1 / 0 && o >= 0 && (i = e.substring(0, o + 1));
        }
        if (this.state.top && (r = this.tokenizer.paragraph(i))) {
          let o = t.at(-1);
          n && (o == null ? void 0 : o.type) === "paragraph" ? (o.raw += (o.raw.endsWith(`
`) ? "" : `
`) + r.raw, o.text += `
` + r.text, this.inlineQueue.pop(), this.inlineQueue.at(-1).src = o.text) : t.push(r), n = i.length !== e.length, e = e.substring(r.raw.length);
          continue;
        }
        if (r = this.tokenizer.text(e)) {
          e = e.substring(r.raw.length);
          let o = t.at(-1);
          (o == null ? void 0 : o.type) === "text" ? (o.raw += (o.raw.endsWith(`
`) ? "" : `
`) + r.raw, o.text += `
` + r.text, this.inlineQueue.pop(), this.inlineQueue.at(-1).src = o.text) : t.push(r);
          continue;
        }
        if (e) {
          this.infiniteLoopError(e.charCodeAt(0));
          break;
        }
      }
      return this.state.top = true, t;
    }
    inline(e, t = []) {
      return this.inlineQueue.push({ src: e, tokens: t }), t;
    }
    inlineTokens(e, t = []) {
      var _a2, _b, _c, _d, _e2;
      this.tokenizer.lexer = this;
      let n = e, s = null;
      if (this.tokens.links) {
        let a = Object.keys(this.tokens.links);
        if (a.length > 0) for (; (s = this.tokenizer.rules.inline.reflinkSearch.exec(n)) !== null; ) a.includes(s[0].slice(s[0].lastIndexOf("[") + 1, -1)) && (n = n.slice(0, s.index) + "[" + "a".repeat(s[0].length - 2) + "]" + n.slice(this.tokenizer.rules.inline.reflinkSearch.lastIndex));
      }
      for (; (s = this.tokenizer.rules.inline.anyPunctuation.exec(n)) !== null; ) n = n.slice(0, s.index) + "++" + n.slice(this.tokenizer.rules.inline.anyPunctuation.lastIndex);
      let r;
      for (; (s = this.tokenizer.rules.inline.blockSkip.exec(n)) !== null; ) r = s[2] ? s[2].length : 0, n = n.slice(0, s.index + r) + "[" + "a".repeat(s[0].length - r - 2) + "]" + n.slice(this.tokenizer.rules.inline.blockSkip.lastIndex);
      n = ((_b = (_a2 = this.options.hooks) == null ? void 0 : _a2.emStrongMask) == null ? void 0 : _b.call({ lexer: this }, n)) ?? n;
      let i = false, o = "", u = 1 / 0;
      for (; e; ) {
        if (e.length < u) u = e.length;
        else {
          this.infiniteLoopError(e.charCodeAt(0));
          break;
        }
        i || (o = ""), i = false;
        let a;
        if ((_d = (_c = this.options.extensions) == null ? void 0 : _c.inline) == null ? void 0 : _d.some((p) => (a = p.call({ lexer: this }, e, t)) ? (e = e.substring(a.raw.length), t.push(a), true) : false)) continue;
        if (a = this.tokenizer.escape(e)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.tag(e)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.link(e)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.reflink(e, this.tokens.links)) {
          e = e.substring(a.raw.length);
          let p = t.at(-1);
          a.type === "text" && (p == null ? void 0 : p.type) === "text" ? (p.raw += a.raw, p.text += a.text) : t.push(a);
          continue;
        }
        if (a = this.tokenizer.emStrong(e, n, o)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.codespan(e)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.br(e)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.del(e, n, o)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (a = this.tokenizer.autolink(e)) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        if (!this.state.inLink && (a = this.tokenizer.url(e))) {
          e = e.substring(a.raw.length), t.push(a);
          continue;
        }
        let c = e;
        if ((_e2 = this.options.extensions) == null ? void 0 : _e2.startInline) {
          let p = 1 / 0, k = e.slice(1), h;
          this.options.extensions.startInline.forEach((R) => {
            h = R.call({ lexer: this }, k), typeof h == "number" && h >= 0 && (p = Math.min(p, h));
          }), p < 1 / 0 && p >= 0 && (c = e.substring(0, p + 1));
        }
        if (a = this.tokenizer.inlineText(c)) {
          e = e.substring(a.raw.length), a.raw.slice(-1) !== "_" && (o = a.raw.slice(-1)), i = true;
          let p = t.at(-1);
          (p == null ? void 0 : p.type) === "text" ? (p.raw += a.raw, p.text += a.text) : t.push(a);
          continue;
        }
        if (e) {
          this.infiniteLoopError(e.charCodeAt(0));
          break;
        }
      }
      return t;
    }
    infiniteLoopError(e) {
      let t = "Infinite loop on byte: " + e;
      if (this.options.silent) console.error(t);
      else throw new Error(t);
    }
  };
  var y = class {
    constructor(e) {
      __publicField(this, "options");
      __publicField(this, "parser");
      this.options = e || T;
    }
    space(e) {
      return "";
    }
    code({ text: e, lang: t, escaped: n }) {
      var _a2;
      let s = (_a2 = (t || "").match(m.notSpaceStart)) == null ? void 0 : _a2[0], r = e.replace(m.endingNewline, "") + `
`;
      return s ? '<pre><code class="language-' + O(s) + '">' + (n ? r : O(r, true)) + `</code></pre>
` : "<pre><code>" + (n ? r : O(r, true)) + `</code></pre>
`;
    }
    blockquote({ tokens: e }) {
      return `<blockquote>
${this.parser.parse(e)}</blockquote>
`;
    }
    html({ text: e }) {
      return e;
    }
    def(e) {
      return "";
    }
    heading({ tokens: e, depth: t }) {
      return `<h${t}>${this.parser.parseInline(e)}</h${t}>
`;
    }
    hr(e) {
      return `<hr>
`;
    }
    list(e) {
      let t = e.ordered, n = e.start, s = "";
      for (let o = 0; o < e.items.length; o++) {
        let u = e.items[o];
        s += this.listitem(u);
      }
      let r = t ? "ol" : "ul", i = t && n !== 1 ? ' start="' + n + '"' : "";
      return "<" + r + i + `>
` + s + "</" + r + `>
`;
    }
    listitem(e) {
      return `<li>${this.parser.parse(e.tokens)}</li>
`;
    }
    checkbox({ checked: e }) {
      return "<input " + (e ? 'checked="" ' : "") + 'disabled="" type="checkbox"> ';
    }
    paragraph({ tokens: e }) {
      return `<p>${this.parser.parseInline(e)}</p>
`;
    }
    table(e) {
      let t = "", n = "";
      for (let r = 0; r < e.header.length; r++) n += this.tablecell(e.header[r]);
      t += this.tablerow({ text: n });
      let s = "";
      for (let r = 0; r < e.rows.length; r++) {
        let i = e.rows[r];
        n = "";
        for (let o = 0; o < i.length; o++) n += this.tablecell(i[o]);
        s += this.tablerow({ text: n });
      }
      return s && (s = `<tbody>${s}</tbody>`), `<table>
<thead>
` + t + `</thead>
` + s + `</table>
`;
    }
    tablerow({ text: e }) {
      return `<tr>
${e}</tr>
`;
    }
    tablecell(e) {
      let t = this.parser.parseInline(e.tokens), n = e.header ? "th" : "td";
      return (e.align ? `<${n} align="${e.align}">` : `<${n}>`) + t + `</${n}>
`;
    }
    strong({ tokens: e }) {
      return `<strong>${this.parser.parseInline(e)}</strong>`;
    }
    em({ tokens: e }) {
      return `<em>${this.parser.parseInline(e)}</em>`;
    }
    codespan({ text: e }) {
      return `<code>${O(e, true)}</code>`;
    }
    br(e) {
      return "<br>";
    }
    del({ tokens: e }) {
      return `<del>${this.parser.parseInline(e)}</del>`;
    }
    link({ href: e, title: t, tokens: n }) {
      let s = this.parser.parseInline(n), r = V(e);
      if (r === null) return s;
      e = r;
      let i = '<a href="' + e + '"';
      return t && (i += ' title="' + O(t) + '"'), i += ">" + s + "</a>", i;
    }
    image({ href: e, title: t, text: n, tokens: s }) {
      s && (n = this.parser.parseInline(s, this.parser.textRenderer));
      let r = V(e);
      if (r === null) return O(n);
      e = r;
      let i = `<img src="${e}" alt="${O(n)}"`;
      return t && (i += ` title="${O(t)}"`), i += ">", i;
    }
    text(e) {
      return "tokens" in e && e.tokens ? this.parser.parseInline(e.tokens) : "escaped" in e && e.escaped ? e.text : O(e.text);
    }
  };
  var L = class {
    strong({ text: e }) {
      return e;
    }
    em({ text: e }) {
      return e;
    }
    codespan({ text: e }) {
      return e;
    }
    del({ text: e }) {
      return e;
    }
    html({ text: e }) {
      return e;
    }
    text({ text: e }) {
      return e;
    }
    link({ text: e }) {
      return "" + e;
    }
    image({ text: e }) {
      return "" + e;
    }
    br() {
      return "";
    }
    checkbox({ raw: e }) {
      return e;
    }
  };
  var b = class l {
    constructor(e) {
      __publicField(this, "options");
      __publicField(this, "renderer");
      __publicField(this, "textRenderer");
      this.options = e || T, this.options.renderer = this.options.renderer || new y(), this.renderer = this.options.renderer, this.renderer.options = this.options, this.renderer.parser = this, this.textRenderer = new L();
    }
    static parse(e, t) {
      return new l(t).parse(e);
    }
    static parseInline(e, t) {
      return new l(t).parseInline(e);
    }
    parse(e) {
      var _a2, _b;
      this.renderer.parser = this;
      let t = "";
      for (let n = 0; n < e.length; n++) {
        let s = e[n];
        if ((_b = (_a2 = this.options.extensions) == null ? void 0 : _a2.renderers) == null ? void 0 : _b[s.type]) {
          let i = s, o = this.options.extensions.renderers[i.type].call({ parser: this }, i);
          if (o !== false || !["space", "hr", "heading", "code", "table", "blockquote", "list", "html", "def", "paragraph", "text"].includes(i.type)) {
            t += o || "";
            continue;
          }
        }
        let r = s;
        switch (r.type) {
          case "space": {
            t += this.renderer.space(r);
            break;
          }
          case "hr": {
            t += this.renderer.hr(r);
            break;
          }
          case "heading": {
            t += this.renderer.heading(r);
            break;
          }
          case "code": {
            t += this.renderer.code(r);
            break;
          }
          case "table": {
            t += this.renderer.table(r);
            break;
          }
          case "blockquote": {
            t += this.renderer.blockquote(r);
            break;
          }
          case "list": {
            t += this.renderer.list(r);
            break;
          }
          case "checkbox": {
            t += this.renderer.checkbox(r);
            break;
          }
          case "html": {
            t += this.renderer.html(r);
            break;
          }
          case "def": {
            t += this.renderer.def(r);
            break;
          }
          case "paragraph": {
            t += this.renderer.paragraph(r);
            break;
          }
          case "text": {
            t += this.renderer.text(r);
            break;
          }
          default: {
            let i = 'Token with "' + r.type + '" type was not found.';
            if (this.options.silent) return console.error(i), "";
            throw new Error(i);
          }
        }
      }
      return t;
    }
    parseInline(e, t = this.renderer) {
      var _a2, _b;
      this.renderer.parser = this;
      let n = "";
      for (let s = 0; s < e.length; s++) {
        let r = e[s];
        if ((_b = (_a2 = this.options.extensions) == null ? void 0 : _a2.renderers) == null ? void 0 : _b[r.type]) {
          let o = this.options.extensions.renderers[r.type].call({ parser: this }, r);
          if (o !== false || !["escape", "html", "link", "image", "strong", "em", "codespan", "br", "del", "text"].includes(r.type)) {
            n += o || "";
            continue;
          }
        }
        let i = r;
        switch (i.type) {
          case "escape": {
            n += t.text(i);
            break;
          }
          case "html": {
            n += t.html(i);
            break;
          }
          case "link": {
            n += t.link(i);
            break;
          }
          case "image": {
            n += t.image(i);
            break;
          }
          case "checkbox": {
            n += t.checkbox(i);
            break;
          }
          case "strong": {
            n += t.strong(i);
            break;
          }
          case "em": {
            n += t.em(i);
            break;
          }
          case "codespan": {
            n += t.codespan(i);
            break;
          }
          case "br": {
            n += t.br(i);
            break;
          }
          case "del": {
            n += t.del(i);
            break;
          }
          case "text": {
            n += t.text(i);
            break;
          }
          default: {
            let o = 'Token with "' + i.type + '" type was not found.';
            if (this.options.silent) return console.error(o), "";
            throw new Error(o);
          }
        }
      }
      return n;
    }
  };
  var P = (_a = class {
    constructor(e) {
      __publicField(this, "options");
      __publicField(this, "block");
      this.options = e || T;
    }
    preprocess(e) {
      return e;
    }
    postprocess(e) {
      return e;
    }
    processAllTokens(e) {
      return e;
    }
    emStrongMask(e) {
      return e;
    }
    provideLexer(e = this.block) {
      return e ? x.lex : x.lexInline;
    }
    provideParser(e = this.block) {
      return e ? b.parse : b.parseInline;
    }
  }, __publicField(_a, "passThroughHooks", /* @__PURE__ */ new Set(["preprocess", "postprocess", "processAllTokens", "emStrongMask"])), __publicField(_a, "passThroughHooksRespectAsync", /* @__PURE__ */ new Set(["preprocess", "postprocess", "processAllTokens"])), _a);
  var q = class {
    constructor(...e) {
      __publicField(this, "defaults", M());
      __publicField(this, "options", this.setOptions);
      __publicField(this, "parse", this.parseMarkdown(true));
      __publicField(this, "parseInline", this.parseMarkdown(false));
      __publicField(this, "Parser", b);
      __publicField(this, "Renderer", y);
      __publicField(this, "TextRenderer", L);
      __publicField(this, "Lexer", x);
      __publicField(this, "Tokenizer", w);
      __publicField(this, "Hooks", P);
      this.use(...e);
    }
    walkTokens(e, t) {
      var _a2, _b;
      let n = [];
      for (let s of e) switch (n = n.concat(t.call(this, s)), s.type) {
        case "table": {
          let r = s;
          for (let i of r.header) n = n.concat(this.walkTokens(i.tokens, t));
          for (let i of r.rows) for (let o of i) n = n.concat(this.walkTokens(o.tokens, t));
          break;
        }
        case "list": {
          let r = s;
          n = n.concat(this.walkTokens(r.items, t));
          break;
        }
        default: {
          let r = s;
          ((_b = (_a2 = this.defaults.extensions) == null ? void 0 : _a2.childTokens) == null ? void 0 : _b[r.type]) ? this.defaults.extensions.childTokens[r.type].forEach((i) => {
            let o = r[i].flat(1 / 0);
            n = n.concat(this.walkTokens(o, t));
          }) : r.tokens && (n = n.concat(this.walkTokens(r.tokens, t)));
        }
      }
      return n;
    }
    use(...e) {
      let t = this.defaults.extensions || { renderers: {}, childTokens: {} };
      return e.forEach((n) => {
        let s = { ...n };
        if (s.async = this.defaults.async || s.async || false, n.extensions && (n.extensions.forEach((r) => {
          if (!r.name) throw new Error("extension name required");
          if ("renderer" in r) {
            let i = t.renderers[r.name];
            i ? t.renderers[r.name] = function(...o) {
              let u = r.renderer.apply(this, o);
              return u === false && (u = i.apply(this, o)), u;
            } : t.renderers[r.name] = r.renderer;
          }
          if ("tokenizer" in r) {
            if (!r.level || r.level !== "block" && r.level !== "inline") throw new Error("extension level must be 'block' or 'inline'");
            let i = t[r.level];
            i ? i.unshift(r.tokenizer) : t[r.level] = [r.tokenizer], r.start && (r.level === "block" ? t.startBlock ? t.startBlock.push(r.start) : t.startBlock = [r.start] : r.level === "inline" && (t.startInline ? t.startInline.push(r.start) : t.startInline = [r.start]));
          }
          "childTokens" in r && r.childTokens && (t.childTokens[r.name] = r.childTokens);
        }), s.extensions = t), n.renderer) {
          let r = this.defaults.renderer || new y(this.defaults);
          for (let i in n.renderer) {
            if (!(i in r)) throw new Error(`renderer '${i}' does not exist`);
            if (["options", "parser"].includes(i)) continue;
            let o = i, u = n.renderer[o], a = r[o];
            r[o] = (...c) => {
              let p = u.apply(r, c);
              return p === false && (p = a.apply(r, c)), p || "";
            };
          }
          s.renderer = r;
        }
        if (n.tokenizer) {
          let r = this.defaults.tokenizer || new w(this.defaults);
          for (let i in n.tokenizer) {
            if (!(i in r)) throw new Error(`tokenizer '${i}' does not exist`);
            if (["options", "rules", "lexer"].includes(i)) continue;
            let o = i, u = n.tokenizer[o], a = r[o];
            r[o] = (...c) => {
              let p = u.apply(r, c);
              return p === false && (p = a.apply(r, c)), p;
            };
          }
          s.tokenizer = r;
        }
        if (n.hooks) {
          let r = this.defaults.hooks || new P();
          for (let i in n.hooks) {
            if (!(i in r)) throw new Error(`hook '${i}' does not exist`);
            if (["options", "block"].includes(i)) continue;
            let o = i, u = n.hooks[o], a = r[o];
            P.passThroughHooks.has(i) ? r[o] = (c) => {
              if (this.defaults.async && P.passThroughHooksRespectAsync.has(i)) return (async () => {
                let k = await u.call(r, c);
                return a.call(r, k);
              })();
              let p = u.call(r, c);
              return a.call(r, p);
            } : r[o] = (...c) => {
              if (this.defaults.async) return (async () => {
                let k = await u.apply(r, c);
                return k === false && (k = await a.apply(r, c)), k;
              })();
              let p = u.apply(r, c);
              return p === false && (p = a.apply(r, c)), p;
            };
          }
          s.hooks = r;
        }
        if (n.walkTokens) {
          let r = this.defaults.walkTokens, i = n.walkTokens;
          s.walkTokens = function(o) {
            let u = [];
            return u.push(i.call(this, o)), r && (u = u.concat(r.call(this, o))), u;
          };
        }
        this.defaults = { ...this.defaults, ...s };
      }), this;
    }
    setOptions(e) {
      return this.defaults = { ...this.defaults, ...e }, this;
    }
    lexer(e, t) {
      return x.lex(e, t ?? this.defaults);
    }
    parser(e, t) {
      return b.parse(e, t ?? this.defaults);
    }
    parseMarkdown(e) {
      return (n, s) => {
        let r = { ...s }, i = { ...this.defaults, ...r }, o = this.onError(!!i.silent, !!i.async);
        if (this.defaults.async === true && r.async === false) return o(new Error("marked(): The async option was set to true by an extension. Remove async: false from the parse options object to return a Promise."));
        if (typeof n > "u" || n === null) return o(new Error("marked(): input parameter is undefined or null"));
        if (typeof n != "string") return o(new Error("marked(): input parameter is of type " + Object.prototype.toString.call(n) + ", string expected"));
        if (i.hooks && (i.hooks.options = i, i.hooks.block = e), i.async) return (async () => {
          let u = i.hooks ? await i.hooks.preprocess(n) : n, c = await (i.hooks ? await i.hooks.provideLexer(e) : e ? x.lex : x.lexInline)(u, i), p = i.hooks ? await i.hooks.processAllTokens(c) : c;
          i.walkTokens && await Promise.all(this.walkTokens(p, i.walkTokens));
          let h = await (i.hooks ? await i.hooks.provideParser(e) : e ? b.parse : b.parseInline)(p, i);
          return i.hooks ? await i.hooks.postprocess(h) : h;
        })().catch(o);
        try {
          i.hooks && (n = i.hooks.preprocess(n));
          let a = (i.hooks ? i.hooks.provideLexer(e) : e ? x.lex : x.lexInline)(n, i);
          i.hooks && (a = i.hooks.processAllTokens(a)), i.walkTokens && this.walkTokens(a, i.walkTokens);
          let p = (i.hooks ? i.hooks.provideParser(e) : e ? b.parse : b.parseInline)(a, i);
          return i.hooks && (p = i.hooks.postprocess(p)), p;
        } catch (u) {
          return o(u);
        }
      };
    }
    onError(e, t) {
      return (n) => {
        if (n.message += `
Please report this to https://github.com/markedjs/marked.`, e) {
          let s = "<p>An error occurred:</p><pre>" + O(n.message + "", true) + "</pre>";
          return t ? Promise.resolve(s) : s;
        }
        if (t) return Promise.reject(n);
        throw n;
      };
    }
  };
  var z = new q();
  function g(l, e) {
    return z.parse(l, e);
  }
  g.options = g.setOptions = function(l) {
    return z.setOptions(l), g.defaults = z.defaults, N(g.defaults), g;
  };
  g.getDefaults = M;
  g.defaults = T;
  g.use = function(...l) {
    return z.use(...l), g.defaults = z.defaults, N(g.defaults), g;
  };
  g.walkTokens = function(l, e) {
    return z.walkTokens(l, e);
  };
  g.parseInline = z.parseInline;
  g.Parser = b;
  g.parser = b.parse;
  g.Renderer = y;
  g.TextRenderer = L;
  g.Lexer = x;
  g.lexer = x.lex;
  g.Tokenizer = w;
  g.Hooks = P;
  g.parse = g;
  g.options;
  g.setOptions;
  g.use;
  g.walkTokens;
  g.parseInline;
  b.parse;
  x.lex;
  const _hoisted_1$2 = { class: "modal-content help-modal" };
  const _hoisted_2$2 = { class: "modal-header" };
  const _hoisted_3$2 = { class: "header-actions" };
  const _hoisted_4$2 = { class: "modal-body" };
  const _hoisted_5$2 = { class: "doc-sidebar" };
  const _hoisted_6$2 = { class: "doc-sidebar-group" };
  const _hoisted_7$2 = { class: "doc-sidebar-header" };
  const _hoisted_8$2 = ["onClick"];
  const _hoisted_9$2 = { class: "doc-sidebar-group" };
  const _hoisted_10$2 = { class: "doc-sidebar-header" };
  const _hoisted_11$2 = { class: "doc-content" };
  const _hoisted_12$2 = ["innerHTML"];
  const _hoisted_13$2 = ["innerHTML"];
  const _hoisted_14$2 = ["innerHTML"];
  const _hoisted_15$1 = ["innerHTML"];
  const _hoisted_16$1 = ["innerHTML"];
  const _hoisted_17$1 = ["innerHTML"];
  const _hoisted_18$1 = ["innerHTML"];
  const _hoisted_19$1 = { class: "doc-pagination" };
  const _hoisted_20$1 = ["disabled"];
  const _hoisted_21 = { class: "page-info" };
  const _hoisted_22 = ["disabled"];
  const _sfc_main$3 = {
    __name: "HelpModal",
    props: {
      initialDoc: { type: String, default: "getting-started" }
    },
    emits: ["close", "openAbout"],
    setup(__props, { emit: __emit }) {
      const props = __props;
      const activeDoc = vue.ref(props.initialDoc);
      const searchQuery = vue.ref("");
      const contentRef = vue.ref(null);
      const centerDocs = [
        { id: "getting-started", title: "快速开始", icon: "home" },
        { id: "features", title: "功能介绍", icon: "info" },
        { id: "api", title: "API 文档", icon: "code" },
        { id: "tools", title: "工具文档", icon: "tool" },
        { id: "shortcuts", title: "快捷键", icon: "keyboard" },
        { id: "faq", title: "常见问题", icon: "help" }
      ];
      const filteredCenterDocs = vue.ref(centerDocs);
      const flatDocs = vue.computed(() => {
        return [...filteredCenterDocs.value, { id: "changelog", title: "更新日志", icon: "activity" }];
      });
      const currentDocIndex = vue.computed(() => flatDocs.value.findIndex((d2) => d2.id === activeDoc.value));
      const hasPrev = vue.computed(() => currentDocIndex.value > 0);
      const hasNext = vue.computed(() => currentDocIndex.value < flatDocs.value.length - 1);
      function filterDocs() {
        const q2 = searchQuery.value.toLowerCase().trim();
        if (!q2) {
          filteredCenterDocs.value = centerDocs;
          return;
        }
        filteredCenterDocs.value = centerDocs.filter(
          (d2) => d2.title.toLowerCase().includes(q2) || d2.id.includes(q2)
        );
        if (filteredCenterDocs.value.length > 0 && !filteredCenterDocs.value.find((d2) => d2.id === activeDoc.value)) {
          activeDoc.value = filteredCenterDocs.value[0].id;
        }
      }
      const renderMd = (md) => {
        const html = g(md, { breaks: true, gfm: true });
        return html.replace(/<table>/g, '<table class="doc-table">');
      };
      const renderedFaq = vue.computed(() => renderMd(faqMd));
      const renderedGettingStarted = vue.computed(() => renderMd(gettingStartedMd));
      const renderedFeatures = vue.computed(() => renderMd(featuresMd));
      const renderedApi = vue.computed(() => renderMd(apiDocsMd));
      const renderedTools = vue.computed(() => renderMd(toolsMd));
      const renderedShortcuts = vue.computed(() => renderMd(shortcutsMd));
      const renderedChangelog = vue.computed(() => renderMd(changelogMd));
      function prevDoc() {
        var _a2;
        if (hasPrev.value) {
          activeDoc.value = flatDocs.value[currentDocIndex.value - 1].id;
          (_a2 = contentRef.value) == null ? void 0 : _a2.scrollTo(0, 0);
        }
      }
      function nextDoc() {
        var _a2;
        if (hasNext.value) {
          activeDoc.value = flatDocs.value[currentDocIndex.value + 1].id;
          (_a2 = contentRef.value) == null ? void 0 : _a2.scrollTo(0, 0);
        }
      }
      vue.onMounted(() => {
        filterDocs();
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", {
          class: "modal-overlay",
          onClick: _cache[3] || (_cache[3] = vue.withModifiers(($event) => _ctx.$emit("close"), ["self"]))
        }, [
          vue.createElementVNode("div", _hoisted_1$2, [
            vue.createCommentVNode(" 头部 "),
            vue.createElementVNode("div", _hoisted_2$2, [
              vue.createElementVNode("h2", null, [
                vue.createVNode(SvgIcon, {
                  name: "book-open",
                  size: 18
                }),
                _cache[4] || (_cache[4] = vue.createTextVNode(
                  " 帮助文档",
                  -1
                  /* CACHED */
                ))
              ]),
              vue.createElementVNode("div", _hoisted_3$2, [
                vue.createElementVNode("button", {
                  class: "btn-about",
                  onClick: _cache[0] || (_cache[0] = ($event) => _ctx.$emit("openAbout")),
                  title: "关于 PairCode"
                }, [
                  vue.createVNode(SvgIcon, {
                    name: "info",
                    size: 14
                  }),
                  _cache[5] || (_cache[5] = vue.createTextVNode(
                    " 关于 ",
                    -1
                    /* CACHED */
                  ))
                ]),
                vue.createElementVNode("button", {
                  class: "modal-close",
                  onClick: _cache[1] || (_cache[1] = ($event) => _ctx.$emit("close"))
                }, "×")
              ])
            ]),
            vue.createCommentVNode(" 主体 "),
            vue.createElementVNode("div", _hoisted_4$2, [
              vue.createCommentVNode(" 侧边导航 "),
              vue.createElementVNode("div", _hoisted_5$2, [
                vue.createCommentVNode(" 文档中心分组 "),
                vue.createElementVNode("div", _hoisted_6$2, [
                  vue.createElementVNode("div", _hoisted_7$2, [
                    vue.createVNode(SvgIcon, {
                      name: "book",
                      size: 14
                    }),
                    _cache[6] || (_cache[6] = vue.createElementVNode(
                      "span",
                      null,
                      "文档中心",
                      -1
                      /* CACHED */
                    ))
                  ]),
                  (vue.openBlock(true), vue.createElementBlock(
                    vue.Fragment,
                    null,
                    vue.renderList(filteredCenterDocs.value, (doc) => {
                      return vue.openBlock(), vue.createElementBlock("div", {
                        key: doc.id,
                        class: vue.normalizeClass(["doc-nav-item", { active: activeDoc.value === doc.id }]),
                        onClick: ($event) => activeDoc.value = doc.id
                      }, [
                        vue.createVNode(SvgIcon, {
                          name: doc.icon,
                          size: 16
                        }, null, 8, ["name"]),
                        vue.createElementVNode(
                          "span",
                          null,
                          vue.toDisplayString(doc.title),
                          1
                          /* TEXT */
                        )
                      ], 10, _hoisted_8$2);
                    }),
                    128
                    /* KEYED_FRAGMENT */
                  ))
                ]),
                vue.createCommentVNode(" 更新日志 "),
                vue.createElementVNode("div", _hoisted_9$2, [
                  vue.createElementVNode("div", _hoisted_10$2, [
                    vue.createVNode(SvgIcon, {
                      name: "clock",
                      size: 14
                    }),
                    _cache[7] || (_cache[7] = vue.createElementVNode(
                      "span",
                      null,
                      "其他",
                      -1
                      /* CACHED */
                    ))
                  ]),
                  vue.createElementVNode(
                    "div",
                    {
                      class: vue.normalizeClass(["doc-nav-item", { active: activeDoc.value === "changelog" }]),
                      onClick: _cache[2] || (_cache[2] = ($event) => activeDoc.value = "changelog")
                    },
                    [
                      vue.createVNode(SvgIcon, {
                        name: "activity",
                        size: 16
                      }),
                      _cache[8] || (_cache[8] = vue.createElementVNode(
                        "span",
                        null,
                        "更新日志",
                        -1
                        /* CACHED */
                      ))
                    ],
                    2
                    /* CLASS */
                  )
                ])
              ]),
              vue.createCommentVNode(" 文档内容 "),
              vue.createElementVNode("div", _hoisted_11$2, [
                vue.createElementVNode(
                  "div",
                  {
                    class: "doc-content-inner",
                    ref_key: "contentRef",
                    ref: contentRef
                  },
                  [
                    activeDoc.value === "faq" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 0,
                      class: "doc-markdown",
                      innerHTML: renderedFaq.value
                    }, null, 8, _hoisted_12$2)) : activeDoc.value === "getting-started" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 1,
                      class: "doc-markdown",
                      innerHTML: renderedGettingStarted.value
                    }, null, 8, _hoisted_13$2)) : activeDoc.value === "features" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 2,
                      class: "doc-markdown",
                      innerHTML: renderedFeatures.value
                    }, null, 8, _hoisted_14$2)) : activeDoc.value === "api" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 3,
                      class: "doc-markdown",
                      innerHTML: renderedApi.value
                    }, null, 8, _hoisted_15$1)) : activeDoc.value === "tools" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 4,
                      class: "doc-markdown",
                      innerHTML: renderedTools.value
                    }, null, 8, _hoisted_16$1)) : activeDoc.value === "shortcuts" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 5,
                      class: "doc-markdown",
                      innerHTML: renderedShortcuts.value
                    }, null, 8, _hoisted_17$1)) : activeDoc.value === "changelog" ? (vue.openBlock(), vue.createElementBlock("div", {
                      key: 6,
                      class: "doc-markdown",
                      innerHTML: renderedChangelog.value
                    }, null, 8, _hoisted_18$1)) : vue.createCommentVNode("v-if", true)
                  ],
                  512
                  /* NEED_PATCH */
                ),
                vue.createCommentVNode(" 底部翻页 "),
                vue.createElementVNode("div", _hoisted_19$1, [
                  vue.createElementVNode("button", {
                    class: "page-btn",
                    onClick: prevDoc,
                    disabled: !hasPrev.value
                  }, [
                    vue.createVNode(SvgIcon, {
                      name: "chevron-left",
                      size: 14
                    }),
                    _cache[9] || (_cache[9] = vue.createTextVNode(
                      " 上一页 ",
                      -1
                      /* CACHED */
                    ))
                  ], 8, _hoisted_20$1),
                  vue.createElementVNode(
                    "span",
                    _hoisted_21,
                    vue.toDisplayString(currentDocIndex.value + 1) + " / " + vue.toDisplayString(flatDocs.value.length),
                    1
                    /* TEXT */
                  ),
                  vue.createElementVNode("button", {
                    class: "page-btn",
                    onClick: nextDoc,
                    disabled: !hasNext.value
                  }, [
                    _cache[10] || (_cache[10] = vue.createTextVNode(
                      " 下一页 ",
                      -1
                      /* CACHED */
                    )),
                    vue.createVNode(SvgIcon, {
                      name: "chevron-right",
                      size: 14
                    })
                  ], 8, _hoisted_22)
                ])
              ])
            ])
          ])
        ]);
      };
    }
  };
  const HelpModal = /* @__PURE__ */ _export_sfc(_sfc_main$3, [["__scopeId", "data-v-667c64dc"]]);
  const logoUrl = "data:image/svg+xml,%3csvg%20xmlns='http://www.w3.org/2000/svg'%20width='512'%20height='512'%20viewBox='0%200%20512%20512'%3e%3cdefs%3e%3c!--%20背景渐变（深色科技风）%20--%3e%3clinearGradient%20id='bgGrad'%20x1='0'%20y1='0'%20x2='1'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%230a1628'/%3e%3cstop%20offset='100%25'%20stop-color='%230d1f2e'/%3e%3c/linearGradient%3e%3c!--%20左侧尖括号渐变（科技蓝）%20--%3e%3clinearGradient%20id='leftBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='100%25'%20stop-color='%230077b6'/%3e%3c/linearGradient%3e%3c!--%20右侧尖括号渐变（科技绿）%20--%3e%3clinearGradient%20id='rightBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300e676'/%3e%3cstop%20offset='100%25'%20stop-color='%2300c853'/%3e%3c/linearGradient%3e%3c!--%20中间连接线（蓝绿渐变）%20--%3e%3clinearGradient%20id='connector'%20x1='0'%20y1='0'%20x2='1'%20y2='0'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='50%25'%20stop-color='%2300e5ff'/%3e%3cstop%20offset='100%25'%20stop-color='%2300e676'/%3e%3c/linearGradient%3e%3c!--%20外发光%20--%3e%3cfilter%20id='glow'%3e%3cfeGaussianBlur%20stdDeviation='4'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3cfilter%20id='softGlow'%3e%3cfeGaussianBlur%20stdDeviation='8'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3c/defs%3e%3c!--%20圆角方形背景（深色科技底）%20--%3e%3crect%20x='32'%20y='32'%20width='448'%20height='448'%20rx='96'%20ry='96'%20fill='url(%23bgGrad)'%20stroke='%231a3a4a'%20stroke-width='2'/%3e%3c!--%20左侧%20%3c%20尖括号（三段式直线%20—%20科技蓝，代表代码输入/开发者）%20--%3e%3cpath%20d='M180%20150%20L96%20256%20L180%20362'%20stroke='url(%23leftBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20右侧%20%3e%20尖括号（三段式直线%20—%20科技绿，代表代码输出/AI伙伴）%20--%3e%3cpath%20d='M332%20150%20L416%20256%20L332%20362'%20stroke='url(%23rightBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20中间「=」连接线已移除（图标只留%20%3c%20%3e%20尖括号%20+%20中心%20AI%20核心光点）。%20--%3e%3c!--%20中心光点（代表%20AI%20核心%20—%20亮青色）%20--%3e%3ccircle%20cx='256'%20cy='256'%20r='18'%20fill='transparent'%20stroke='%2300e5ff'%20stroke-width='3'%20opacity='0.6'/%3e%3ccircle%20cx='256'%20cy='256'%20r='8'%20fill='%2300e5ff'%20opacity='0.9'%3e%3canimate%20attributeName='opacity'%20values='0.6;1;0.6'%20dur='2s'%20repeatCount='indefinite'/%3e%3c/circle%3e%3c/svg%3e";
  const _hoisted_1$1 = { class: "modal-content about-modal" };
  const _hoisted_2$1 = { class: "modal-header" };
  const _hoisted_3$1 = { class: "modal-body" };
  const _hoisted_4$1 = { class: "about-left-col" };
  const _hoisted_5$1 = { class: "about-hero" };
  const _hoisted_6$1 = { class: "about-logo" };
  const _hoisted_7$1 = ["src"];
  const _hoisted_8$1 = { class: "about-version" };
  const _hoisted_9$1 = { class: "about-right-col" };
  const _hoisted_10$1 = { class: "about-section" };
  const _hoisted_11$1 = { class: "feature-list" };
  const _hoisted_12$1 = { class: "about-section" };
  const _hoisted_13$1 = {
    key: 0,
    class: "sys-info"
  };
  const _hoisted_14$1 = { class: "info-row" };
  const _hoisted_15 = { class: "info-row" };
  const _hoisted_16 = { class: "info-row" };
  const _hoisted_17 = { class: "info-path" };
  const _hoisted_18 = { class: "info-row" };
  const _hoisted_19 = {
    key: 1,
    class: "loading-info"
  };
  const _hoisted_20 = { class: "modal-footer" };
  const _sfc_main$2 = {
    __name: "AboutModal",
    props: {
      showHelpBtn: { type: Boolean, default: true }
    },
    emits: ["close", "openHelp"],
    setup(__props, { emit: __emit }) {
      const version = vue.ref("");
      const sysInfo = vue.ref({});
      const sysLoading = vue.ref(true);
      vue.onMounted(async () => {
        try {
          const info = await api.apiGet("/system/info");
          sysInfo.value = info;
          if (info.version) version.value = info.version;
        } catch {
        }
        sysLoading.value = false;
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", {
          class: "modal-overlay",
          onClick: _cache[3] || (_cache[3] = vue.withModifiers(($event) => _ctx.$emit("close"), ["self"]))
        }, [
          vue.createElementVNode("div", _hoisted_1$1, [
            vue.createElementVNode("div", _hoisted_2$1, [
              vue.createElementVNode("h2", null, [
                vue.createVNode(SvgIcon, {
                  name: "info",
                  size: 18
                }),
                _cache[4] || (_cache[4] = vue.createTextVNode(
                  " 关于 PairCode",
                  -1
                  /* CACHED */
                ))
              ]),
              vue.createElementVNode("button", {
                class: "modal-close",
                onClick: _cache[0] || (_cache[0] = ($event) => _ctx.$emit("close"))
              }, "×")
            ]),
            vue.createElementVNode("div", _hoisted_3$1, [
              vue.createCommentVNode(" 左列：Logo + 描述 + 技术栈 "),
              vue.createElementVNode("div", _hoisted_4$1, [
                vue.createCommentVNode(" Logo + 标题 "),
                vue.createElementVNode("div", _hoisted_5$1, [
                  vue.createElementVNode("div", _hoisted_6$1, [
                    vue.createElementVNode("img", {
                      src: vue.unref(logoUrl),
                      class: "about-logo-img",
                      alt: "PairCode"
                    }, null, 8, _hoisted_7$1)
                  ]),
                  _cache[5] || (_cache[5] = vue.createElementVNode(
                    "div",
                    { class: "about-title" },
                    "PairCode IDE",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode(
                    "div",
                    _hoisted_8$1,
                    "版本 " + vue.toDisplayString(version.value),
                    1
                    /* TEXT */
                  )
                ]),
                vue.createCommentVNode(" 描述 "),
                _cache[6] || (_cache[6] = vue.createElementVNode(
                  "div",
                  { class: "about-section" },
                  [
                    vue.createElementVNode("p", { class: "about-description" }, " PairCode IDE 是一款纯 Web 端的 AI 辅助编程集成开发环境， 专为浏览器而设计。无需安装任何桌面客户端或本地 IDE 软件， 打开浏览器即可开始编程。它将 AI 对话能力深度融入编码工作流， 你只需用自然语言描述需求，AI 就能自动理解上下文、读写文件、执行命令、 管理版本控制。从代码生成到项目运维，在同一个浏览器窗口中全部完成。 ")
                  ],
                  -1
                  /* CACHED */
                )),
                vue.createCommentVNode(" 技术栈 "),
                _cache[7] || (_cache[7] = vue.createStaticVNode('<div class="about-section" data-v-cdb64a03><div class="section-title" data-v-cdb64a03>技术栈</div><div class="tech-stack" data-v-cdb64a03><span class="tech-badge" data-v-cdb64a03>Go</span><span class="tech-badge" data-v-cdb64a03>Vue 3</span><span class="tech-badge" data-v-cdb64a03>WebSocket</span><span class="tech-badge" data-v-cdb64a03>CodeMirror</span><span class="tech-badge" data-v-cdb64a03>插件化工具</span><span class="tech-badge" data-v-cdb64a03>TS 编译器</span><span class="tech-badge" data-v-cdb64a03>MCP</span><span class="tech-badge" data-v-cdb64a03>CodeGraph</span><span class="tech-badge" data-v-cdb64a03>DAP</span></div></div>', 1))
              ]),
              vue.createCommentVNode(" 右列：特性 + 系统信息 "),
              vue.createElementVNode("div", _hoisted_9$1, [
                vue.createCommentVNode(" 特性亮点 "),
                vue.createElementVNode("div", _hoisted_10$1, [
                  _cache[18] || (_cache[18] = vue.createElementVNode(
                    "div",
                    { class: "section-title" },
                    "主要特性",
                    -1
                    /* CACHED */
                  )),
                  vue.createElementVNode("ul", _hoisted_11$1, [
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "bot",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[8] || (_cache[8] = vue.createTextVNode(
                        " AI 对话编程 — 用自然语言与 AI 对话，自动生成与重构代码",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "file",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[9] || (_cache[9] = vue.createTextVNode(
                        " 智能代码编辑器 — 多语言语法高亮，浏览器中流畅编辑",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "git-branch",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[10] || (_cache[10] = vue.createTextVNode(
                        " Git 版本控制 — 在对话中完成全部 Git 操作",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "terminal",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[11] || (_cache[11] = vue.createTextVNode(
                        " 内置终端 — 无需离开浏览器即可执行命令",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "search",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[12] || (_cache[12] = vue.createTextVNode(
                        " 全局搜索 — 快速搜索文件与代码内容",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "settings",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[13] || (_cache[13] = vue.createTextVNode(
                        " 自主 Agent 模式 — AI 主动分析项目并自动执行任务",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "grid",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[14] || (_cache[14] = vue.createTextVNode(
                        " 对话历史管理 — 自动保存、回溯与继续历史对话",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "tool",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[15] || (_cache[15] = vue.createTextVNode(
                        " Skills / MCP 扩展 — 通过技能市场扩展 IDE 能力",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "code",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[16] || (_cache[16] = vue.createTextVNode(
                        " 内置调试器 — 支持 Go 程序的断点、单步和变量查看",
                        -1
                        /* CACHED */
                      ))
                    ]),
                    vue.createElementVNode("li", null, [
                      vue.createVNode(SvgIcon, {
                        name: "image",
                        size: 14,
                        color: "var(--accent)"
                      }),
                      _cache[17] || (_cache[17] = vue.createTextVNode(
                        " 网页验证 — 打开 URL、截图、分析页面效果",
                        -1
                        /* CACHED */
                      ))
                    ])
                  ])
                ]),
                vue.createCommentVNode(" 系统信息 "),
                vue.createElementVNode("div", _hoisted_12$1, [
                  _cache[23] || (_cache[23] = vue.createElementVNode(
                    "div",
                    { class: "section-title" },
                    "系统信息",
                    -1
                    /* CACHED */
                  )),
                  !sysLoading.value ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_13$1, [
                    vue.createElementVNode("div", _hoisted_14$1, [
                      _cache[19] || (_cache[19] = vue.createElementVNode(
                        "span",
                        { class: "info-label" },
                        "主机名",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode(
                        "span",
                        null,
                        vue.toDisplayString(sysInfo.value.hostname),
                        1
                        /* TEXT */
                      )
                    ]),
                    vue.createElementVNode("div", _hoisted_15, [
                      _cache[20] || (_cache[20] = vue.createElementVNode(
                        "span",
                        { class: "info-label" },
                        "操作系统",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode(
                        "span",
                        null,
                        vue.toDisplayString(sysInfo.value.os),
                        1
                        /* TEXT */
                      )
                    ]),
                    vue.createElementVNode("div", _hoisted_16, [
                      _cache[21] || (_cache[21] = vue.createElementVNode(
                        "span",
                        { class: "info-label" },
                        "工作区",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode(
                        "span",
                        _hoisted_17,
                        vue.toDisplayString(sysInfo.value.workspace),
                        1
                        /* TEXT */
                      )
                    ]),
                    vue.createElementVNode("div", _hoisted_18, [
                      _cache[22] || (_cache[22] = vue.createElementVNode(
                        "span",
                        { class: "info-label" },
                        "平台信息",
                        -1
                        /* CACHED */
                      )),
                      vue.createElementVNode(
                        "span",
                        null,
                        vue.toDisplayString(sysInfo.value.goos),
                        1
                        /* TEXT */
                      )
                    ])
                  ])) : (vue.openBlock(), vue.createElementBlock("div", _hoisted_19, "加载中..."))
                ])
              ])
            ]),
            vue.createCommentVNode(" 底部 "),
            vue.createElementVNode("div", _hoisted_20, [
              __props.showHelpBtn ? (vue.openBlock(), vue.createElementBlock("button", {
                key: 0,
                class: "btn-primary",
                onClick: _cache[1] || (_cache[1] = ($event) => _ctx.$emit("openHelp"))
              }, [
                vue.createVNode(SvgIcon, {
                  name: "book-open",
                  size: 14
                }),
                _cache[24] || (_cache[24] = vue.createTextVNode(
                  " 查看帮助文档 ",
                  -1
                  /* CACHED */
                ))
              ])) : vue.createCommentVNode("v-if", true),
              vue.createElementVNode("button", {
                class: "btn-secondary",
                onClick: _cache[2] || (_cache[2] = ($event) => _ctx.$emit("close"))
              }, "关闭")
            ])
          ])
        ]);
      };
    }
  };
  const AboutModal = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-cdb64a03"]]);
  const _hoisted_1 = { class: "toast-container" };
  const _hoisted_2 = {
    class: "dlg-box",
    style: { "max-width": "400px" }
  };
  const _hoisted_3 = { class: "dlg-title" };
  const _hoisted_4 = { class: "dlg-body" };
  const _hoisted_5 = { class: "dlg-actions" };
  const _hoisted_6 = {
    class: "dlg-box",
    style: { "max-width": "420px" }
  };
  const _hoisted_7 = { class: "dlg-title" };
  const _hoisted_8 = {
    class: "dlg-body",
    style: { "display": "flex", "flex-direction": "column", "gap": "8px" }
  };
  const _hoisted_9 = { style: { "font-size": "13px", "color": "var(--text-secondary)" } };
  const _hoisted_10 = ["placeholder"];
  const _hoisted_11 = { class: "dlg-actions" };
  const _hoisted_12 = {
    class: "dlg-box",
    style: { "max-width": "400px" }
  };
  const _hoisted_13 = { class: "dlg-title" };
  const _hoisted_14 = {
    class: "dlg-body",
    style: { "white-space": "pre-line" }
  };
  const _sfc_main$1 = {
    __name: "GlobalDialogs",
    setup(__props) {
      const promptInputRef = vue.ref(null);
      vue.watch(() => uiState_js.dialogState.show, (v2) => {
        if (v2 && uiState_js.dialogState.type === "prompt") {
          vue.nextTick(() => {
            var _a2, _b;
            (_a2 = promptInputRef.value) == null ? void 0 : _a2.focus();
            (_b = promptInputRef.value) == null ? void 0 : _b.select();
          });
        }
      });
      function handleConfirm() {
        if (uiState_js.dialogState.type === "prompt") {
          const val = uiState_js.dialogState.inputValue;
          uiState_js.dialogState.show = false;
          if (uiState_js.dialogState.resolve) uiState_js.dialogState.resolve(val);
        } else if (uiState_js.dialogState.type === "confirm" && uiState_js.dialogState.checkboxLabel) {
          const confirmed = true;
          const checked = uiState_js.dialogState.checkboxValue;
          uiState_js.dialogState.show = false;
          if (uiState_js.dialogState.resolve) uiState_js.dialogState.resolve({ confirmed, checked });
        } else {
          const val = true;
          uiState_js.dialogState.show = false;
          if (uiState_js.dialogState.resolve) uiState_js.dialogState.resolve(val);
        }
        uiState_js.dialogState.resolve = null;
      }
      function handleCancel() {
        if (uiState_js.dialogState.type === "confirm" && uiState_js.dialogState.checkboxLabel) {
          uiState_js.dialogState.show = false;
          if (uiState_js.dialogState.resolve) uiState_js.dialogState.resolve({ confirmed: false, checked: uiState_js.dialogState.checkboxValue });
        } else if (uiState_js.dialogState.type === "prompt") {
          uiState_js.dialogState.show = false;
          if (uiState_js.dialogState.resolve) uiState_js.dialogState.resolve(null);
        } else {
          uiState_js.dialogState.show = false;
          if (uiState_js.dialogState.resolve) uiState_js.dialogState.resolve(false);
        }
        uiState_js.dialogState.resolve = null;
      }
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock(
          vue.Fragment,
          null,
          [
            vue.createCommentVNode(" Toast 通知区域 "),
            vue.createElementVNode("div", _hoisted_1, [
              (vue.openBlock(true), vue.createElementBlock(
                vue.Fragment,
                null,
                vue.renderList(vue.unref(uiState_js.dialogState).toasts, (t) => {
                  return vue.openBlock(), vue.createElementBlock(
                    "div",
                    {
                      key: t.id,
                      class: vue.normalizeClass(["toast-item", "toast-" + (t.type || "info")])
                    },
                    vue.toDisplayString(t.message),
                    3
                    /* TEXT, CLASS */
                  );
                }),
                128
                /* KEYED_FRAGMENT */
              ))
            ]),
            vue.createCommentVNode(" Confirm 对话框 "),
            vue.unref(uiState_js.dialogState).show && vue.unref(uiState_js.dialogState).type === "confirm" ? (vue.openBlock(), vue.createElementBlock("div", {
              key: 0,
              class: "dlg-overlay",
              onClick: vue.withModifiers(handleCancel, ["self"])
            }, [
              vue.createElementVNode("div", _hoisted_2, [
                vue.createElementVNode(
                  "div",
                  _hoisted_3,
                  vue.toDisplayString(vue.unref(uiState_js.dialogState).title),
                  1
                  /* TEXT */
                ),
                vue.createElementVNode(
                  "div",
                  _hoisted_4,
                  vue.toDisplayString(vue.unref(uiState_js.dialogState).message),
                  1
                  /* TEXT */
                ),
                vue.unref(uiState_js.dialogState).checkboxLabel ? (vue.openBlock(), vue.createElementBlock("label", {
                  key: 0,
                  class: "dlg-checkbox",
                  onClick: _cache[1] || (_cache[1] = vue.withModifiers(() => {
                  }, ["stop"]))
                }, [
                  vue.withDirectives(vue.createElementVNode(
                    "input",
                    {
                      type: "checkbox",
                      "onUpdate:modelValue": _cache[0] || (_cache[0] = ($event) => vue.unref(uiState_js.dialogState).checkboxValue = $event)
                    },
                    null,
                    512
                    /* NEED_PATCH */
                  ), [
                    [vue.vModelCheckbox, vue.unref(uiState_js.dialogState).checkboxValue]
                  ]),
                  vue.createElementVNode(
                    "span",
                    null,
                    vue.toDisplayString(vue.unref(uiState_js.dialogState).checkboxLabel),
                    1
                    /* TEXT */
                  )
                ])) : vue.createCommentVNode("v-if", true),
                vue.createElementVNode("div", _hoisted_5, [
                  vue.createElementVNode(
                    "button",
                    {
                      class: "dlg-btn",
                      onClick: handleCancel
                    },
                    vue.toDisplayString(vue.unref(uiState_js.dialogState).cancelText),
                    1
                    /* TEXT */
                  ),
                  vue.createElementVNode(
                    "button",
                    {
                      class: "dlg-btn primary",
                      onClick: handleConfirm
                    },
                    vue.toDisplayString(vue.unref(uiState_js.dialogState).confirmText),
                    1
                    /* TEXT */
                  )
                ])
              ])
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" Prompt 对话框 "),
            vue.unref(uiState_js.dialogState).show && vue.unref(uiState_js.dialogState).type === "prompt" ? (vue.openBlock(), vue.createElementBlock("div", {
              key: 1,
              class: "dlg-overlay",
              onClick: vue.withModifiers(handleCancel, ["self"])
            }, [
              vue.createElementVNode("div", _hoisted_6, [
                vue.createElementVNode(
                  "div",
                  _hoisted_7,
                  vue.toDisplayString(vue.unref(uiState_js.dialogState).title),
                  1
                  /* TEXT */
                ),
                vue.createElementVNode("div", _hoisted_8, [
                  vue.createElementVNode(
                    "span",
                    _hoisted_9,
                    vue.toDisplayString(vue.unref(uiState_js.dialogState).message),
                    1
                    /* TEXT */
                  ),
                  vue.withDirectives(vue.createElementVNode("input", {
                    ref_key: "promptInputRef",
                    ref: promptInputRef,
                    "onUpdate:modelValue": _cache[2] || (_cache[2] = ($event) => vue.unref(uiState_js.dialogState).inputValue = $event),
                    placeholder: vue.unref(uiState_js.dialogState).inputPlaceholder,
                    class: "dlg-input",
                    onKeyup: [
                      vue.withKeys(handleConfirm, ["enter"]),
                      vue.withKeys(handleCancel, ["escape"])
                    ]
                  }, null, 40, _hoisted_10), [
                    [vue.vModelText, vue.unref(uiState_js.dialogState).inputValue]
                  ])
                ]),
                vue.createElementVNode("div", _hoisted_11, [
                  vue.createElementVNode(
                    "button",
                    {
                      class: "dlg-btn",
                      onClick: handleCancel
                    },
                    vue.toDisplayString(vue.unref(uiState_js.dialogState).cancelText),
                    1
                    /* TEXT */
                  ),
                  vue.createElementVNode(
                    "button",
                    {
                      class: "dlg-btn primary",
                      onClick: handleConfirm
                    },
                    vue.toDisplayString(vue.unref(uiState_js.dialogState).confirmText),
                    1
                    /* TEXT */
                  )
                ])
              ])
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" Alert 信息框 "),
            vue.unref(uiState_js.dialogState).show && vue.unref(uiState_js.dialogState).type === "alert" ? (vue.openBlock(), vue.createElementBlock("div", {
              key: 2,
              class: "dlg-overlay",
              onClick: vue.withModifiers(handleConfirm, ["self"])
            }, [
              vue.createElementVNode("div", _hoisted_12, [
                vue.createElementVNode(
                  "div",
                  _hoisted_13,
                  vue.toDisplayString(vue.unref(uiState_js.dialogState).title),
                  1
                  /* TEXT */
                ),
                vue.createElementVNode(
                  "div",
                  _hoisted_14,
                  vue.toDisplayString(vue.unref(uiState_js.dialogState).message),
                  1
                  /* TEXT */
                ),
                vue.createElementVNode("div", { class: "dlg-actions" }, [
                  vue.createElementVNode("button", {
                    class: "dlg-btn primary",
                    onClick: handleConfirm
                  }, "确定")
                ])
              ])
            ])) : vue.createCommentVNode("v-if", true)
          ],
          64
          /* STABLE_FRAGMENT */
        );
      };
    }
  };
  const GlobalDialogs = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-0271e4ae"]]);
  const _sfc_main = {
    __name: "UiModals",
    setup(__props) {
      const overlaySlotEl = vue.ref(null);
      let overlayUnsub = null;
      function onAboutOpenHelp() {
        uiState_js.showAbout.value = false;
        uiState_js.showHelp.value = true;
        uiState_js.helpDocTarget.value = "getting-started";
      }
      function onHelpOpenAbout() {
        uiState_js.showHelp.value = false;
        uiState_js.showAbout.value = true;
      }
      vue.onMounted(() => {
        overlayUnsub = pluginRuntime_js.mountListSlot(overlaySlotEl, "overlay", { isActive: (n) => pluginRuntime_js.isOverlayActive("overlay", n) });
      });
      vue.onUnmounted(() => {
        if (overlayUnsub) {
          overlayUnsub();
          overlayUnsub = null;
        }
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock(
          vue.Fragment,
          null,
          [
            vue.unref(uiState_js.showSettings) ? (vue.openBlock(), vue.createBlock(SettingsModal, {
              key: 0,
              onClose: _cache[0] || (_cache[0] = ($event) => uiState_js.showSettings.value = false)
            })) : vue.createCommentVNode("v-if", true),
            vue.unref(uiState_js.showSystem) ? (vue.openBlock(), vue.createBlock(SystemModal, {
              key: 1,
              onClose: _cache[1] || (_cache[1] = ($event) => uiState_js.showSystem.value = false)
            })) : vue.createCommentVNode("v-if", true),
            vue.unref(uiState_js.showSource) ? (vue.openBlock(), vue.createBlock(SourceModal, {
              key: 2,
              onClose: _cache[2] || (_cache[2] = ($event) => uiState_js.showSource.value = false)
            })) : vue.createCommentVNode("v-if", true),
            vue.unref(uiState_js.showMarketplace) ? (vue.openBlock(), vue.createBlock(MarketplaceModal, {
              key: 3,
              onClose: _cache[3] || (_cache[3] = ($event) => uiState_js.showMarketplace.value = false)
            })) : vue.createCommentVNode("v-if", true),
            vue.unref(uiState_js.showHelp) ? (vue.openBlock(), vue.createBlock(HelpModal, {
              key: 4,
              onClose: _cache[4] || (_cache[4] = ($event) => uiState_js.showHelp.value = false),
              onOpenAbout: onHelpOpenAbout,
              initialDoc: vue.unref(uiState_js.helpDocTarget)
            }, null, 8, ["initialDoc"])) : vue.createCommentVNode("v-if", true),
            vue.unref(uiState_js.showAbout) ? (vue.openBlock(), vue.createBlock(AboutModal, {
              key: 5,
              onClose: _cache[5] || (_cache[5] = ($event) => uiState_js.showAbout.value = false),
              onOpenHelp: onAboutOpenHelp
            })) : vue.createCommentVNode("v-if", true),
            vue.createVNode(GlobalDialogs),
            vue.createCommentVNode(" ★ overlay 槽位（list 型）：插件注册的浮动层条目叠加渲染（badge/toast/status pill 等） "),
            vue.createElementVNode(
              "div",
              {
                ref_key: "overlaySlotEl",
                ref: overlaySlotEl,
                class: "plugin-overlay-host"
              },
              null,
              512
              /* NEED_PATCH */
            )
          ],
          64
          /* STABLE_FRAGMENT */
        );
      };
    }
  };
  const UiModals2 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-95a2c5d7"]]);
  function mount(el) {
    const app = vue.createApp(UiModals2);
    app.mount(el);
    return () => {
      app.unmount();
    };
  }
  exports.mount = mount;
  Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
  return exports;
})({}, window.__PAIRCODE_CORE.Vue, window.__PAIRCODE_CORE.uiState, window.__PAIRCODE_CORE.pluginRuntime, window.__PAIRCODE_CORE.api);
