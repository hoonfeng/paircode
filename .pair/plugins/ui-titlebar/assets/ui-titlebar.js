var UiTitlebar = (function(exports, vue, uiState_js, api, pluginRuntime_js) {
  "use strict";
  const _export_sfc = (sfc, props) => {
    const target = sfc.__vccOpts || sfc;
    for (const [key, val] of props) {
      target[key] = val;
    }
    return target;
  };
  const _hoisted_1$2 = { class: "menubar" };
  const _hoisted_2$2 = ["onClick", "onMouseenter"];
  const _hoisted_3$1 = {
    key: 0,
    class: "menu-divider"
  };
  const _hoisted_4$1 = ["onClick"];
  const _hoisted_5 = { class: "menu-item-label" };
  const _hoisted_6 = {
    key: 0,
    class: "menu-item-shortcut"
  };
  const _sfc_main$2 = {
    __name: "MenuBar",
    setup(__props, { expose: __expose }) {
      const menus = [
        {
          label: "帮助",
          items: [
            { label: "常见问题", action: "help-faq" },
            { label: "快速开始", action: "help-getting-started" },
            { label: "文档中心", action: "help-docs" },
            { label: "功能介绍", action: "help-features" },
            { label: "API 文档", action: "help-api" },
            { label: "工具文档", action: "help-tools" },
            { label: "快捷键参考", action: "help-shortcuts" },
            { divider: true },
            { label: "关于 PairCode IDE", action: "about" }
          ]
        }
      ];
      const openMenu = vue.ref(null);
      const btnRefs = vue.ref({});
      const dropdownPos = vue.ref({ x: 0, y: 0 });
      let closeTimer = null;
      const currentItems = vue.computed(() => {
        const m = menus.find((m2) => m2.label === openMenu.value);
        return m ? m.items : [];
      });
      const dropdownStyle = vue.computed(() => ({
        left: dropdownPos.value.x + "px",
        top: dropdownPos.value.y + "px"
      }));
      function toggleMenu(event, label) {
        event.stopPropagation();
        if (openMenu.value === label) {
          openMenu.value = null;
          return;
        }
        const btn = btnRefs.value[label];
        if (btn) {
          const rect = btn.getBoundingClientRect();
          dropdownPos.value = { x: rect.left, y: rect.bottom };
        }
        openMenu.value = label;
        cancelClose();
      }
      function hoverMenu(label) {
        if (openMenu.value && openMenu.value !== label) {
          const btn = btnRefs.value[label];
          if (btn) {
            const rect = btn.getBoundingClientRect();
            dropdownPos.value = { x: rect.left, y: rect.bottom };
          }
          openMenu.value = label;
        }
      }
      const scheduleClose = () => {
        closeTimer = setTimeout(() => {
          openMenu.value = null;
        }, 200);
      };
      const cancelClose = () => {
        if (closeTimer) {
          clearTimeout(closeTimer);
          closeTimer = null;
        }
      };
      function closeMenu() {
        openMenu.value = null;
      }
      __expose({ closeMenu });
      const execItem = async (item) => {
        openMenu.value = null;
        const a = item.action;
        if (a === "view-explorer") {
          uiState_js.state.focusMode = false;
          uiState_js.state.activeActivity = "explorer";
          uiState_js.state.sidebarVisible = true;
          return;
        }
        if (a === "view-search") {
          uiState_js.state.focusMode = false;
          uiState_js.state.activeActivity = "search";
          uiState_js.state.sidebarVisible = true;
          return;
        }
        if (a === "view-git") {
          uiState_js.state.focusMode = false;
          uiState_js.state.activeActivity = "source";
          uiState_js.state.sidebarVisible = true;
          return;
        }
        if (a === "toggle-sidebar") {
          uiState_js.state.sidebarVisible = !uiState_js.state.sidebarVisible;
          return;
        }
        if (a === "toggle-terminal") {
          uiState_js.state.bottomPanelVisible = !uiState_js.state.bottomPanelVisible;
          uiState_js.state.bottomPanelTab = "terminal";
          return;
        }
        if (a === "toggle-right") {
          uiState_js.state.rightPanelVisible = !uiState_js.state.rightPanelVisible;
          return;
        }
        if (a === "focus-mode") {
          uiState_js.state.focusMode = !uiState_js.state.focusMode;
          if (uiState_js.state.focusMode) {
            uiState_js.state.bottomPanelVisible = false;
          }
          return;
        }
        if (a === "new-file") {
          const name = await window.$prompt("文件名:", "", "新建文件");
          if (!name) return;
          const dir = uiState_js.state.activeFile ? uiState_js.state.activeFile.substring(0, uiState_js.state.activeFile.lastIndexOf("\\")) : uiState_js.state.workspaceFolders[0] || uiState_js.state.workspaceRoot;
          if (!dir) return window.$toast("请先在文件浏览器中打开一个目录", "warning");
          try {
            await api.apiPost("/fs/write", { path: dir + "\\" + name, content: "" });
            window.dispatchEvent(new CustomEvent("refresh-tree"));
          } catch (err) {
            window.$toast("创建失败: " + err.message, "error");
          }
          return;
        }
        if (a === "open-file") {
          const path = await window.$prompt("输入文件路径:", "", "打开文件");
          if (!path) return;
          if (!uiState_js.state.openFiles.includes(path)) uiState_js.state.openFiles.push(path);
          uiState_js.state.activeFile = path;
          return;
        }
        if (a === "open-folder") {
          const path = await window.$prompt("输入文件夹路径:", "", "打开文件夹");
          if (!path) return;
          try {
            await api.apiPost("/workspace", { action: "add-folder", path });
            window.dispatchEvent(new CustomEvent("refresh-tree"));
          } catch (err) {
            window.$toast("添加失败: " + err.message, "error");
          }
          return;
        }
        if (a === "add-folder") {
          const path = await window.$prompt("输入要添加的文件夹路径:", "", "添加文件夹");
          if (!path) return;
          try {
            await api.apiPost("/workspace", { action: "add-folder", path });
            window.dispatchEvent(new CustomEvent("refresh-tree"));
          } catch (err) {
            window.$toast("添加失败: " + err.message, "error");
          }
          return;
        }
        if (a === "save") {
          if (!uiState_js.state.activeFile || uiState_js.state.fileContents[uiState_js.state.activeFile] === void 0) return;
          try {
            await api.apiPost("/fs/write", { path: uiState_js.state.activeFile, content: uiState_js.state.fileContents[uiState_js.state.activeFile] });
            uiState_js.state.fileDirty[uiState_js.state.activeFile] = false;
          } catch (err) {
            window.$toast("保存失败: " + err.message, "error");
          }
          return;
        }
        if (a === "save-all") {
          for (const f of uiState_js.state.openFiles) {
            if (uiState_js.state.fileDirty[f] && uiState_js.state.fileContents[f] !== void 0) {
              try {
                await api.apiPost("/fs/write", { path: f, content: uiState_js.state.fileContents[f] });
                uiState_js.state.fileDirty[f] = false;
              } catch {
              }
            }
          }
          return;
        }
        if (a === "save-workspace") {
          window.$toast("工作区已自动保存", "success");
          return;
        }
        if (a === "manage-workspace") {
          uiState_js.state.activeActivity = "explorer";
          uiState_js.state.sidebarVisible = true;
          return;
        }
        if (a === "close-project") {
          uiState_js.state.openFiles = [];
          uiState_js.state.activeFile = "";
          uiState_js.state.fileContents = {};
          return;
        }
        if (a === "close-workspace") {
          if (await window.$confirm("关闭工作区？所有未保存更改将丢失。")) {
            uiState_js.state.workspaceRoot = "";
            uiState_js.state.workspaceFolders = [];
            uiState_js.state.fileTree = [];
            uiState_js.state.openFiles = [];
            uiState_js.state.activeFile = "";
            uiState_js.state.fileContents = {};
          }
          return;
        }
        if (a === "undo") {
          window.dispatchEvent(new CustomEvent("editor-undo"));
          return;
        }
        if (a === "redo") {
          window.dispatchEvent(new CustomEvent("editor-redo"));
          return;
        }
        if (a === "cut" || a === "copy" || a === "paste") {
          document.execCommand(a);
          return;
        }
        if (a === "find-chat") {
          uiState_js.state.rightPanelVisible = true;
          return;
        }
        if (a === "global-search") {
          uiState_js.state.activeActivity = "search";
          uiState_js.state.sidebarVisible = true;
          return;
        }
        if (a === "find-file") {
          uiState_js.state.activeActivity = "search";
          uiState_js.state.sidebarVisible = true;
          return;
        }
        if (a === "new-terminal") {
          uiState_js.state.bottomPanelVisible = true;
          uiState_js.state.bottomPanelTab = "terminal";
          return;
        }
        if (a === "clear-terminal") {
          window.dispatchEvent(new CustomEvent("clear-terminal"));
          return;
        }
        if (a === "open-settings") {
          if (uiState_js.showSettings) uiState_js.showSettings.value = true;
          return;
        }
        if (a === "open-marketplace") {
          if (uiState_js.showMarketplace) uiState_js.showMarketplace.value = true;
          return;
        }
        if (a === "help-faq") {
          if (uiState_js.showHelpWrapper) {
            uiState_js.showHelpWrapper.value = "faq";
            return;
          }
          return;
        }
        if (a === "help-getting-started") {
          if (uiState_js.showHelpWrapper) {
            uiState_js.showHelpWrapper.value = "getting-started";
            return;
          }
          return;
        }
        if (a === "help-docs") {
          if (uiState_js.showHelpWrapper) uiState_js.showHelpWrapper.value = true;
          return;
        }
        if (a === "help-features") {
          if (uiState_js.showHelpWrapper) {
            uiState_js.showHelpWrapper.value = "features";
            return;
          }
          return;
        }
        if (a === "help-api") {
          if (uiState_js.showHelpWrapper) {
            uiState_js.showHelpWrapper.value = "api";
            return;
          }
          return;
        }
        if (a === "help-tools") {
          if (uiState_js.showHelpWrapper) {
            uiState_js.showHelpWrapper.value = "tools";
            return;
          }
          return;
        }
        if (a === "help-shortcuts") {
          if (uiState_js.showHelpWrapper) {
            uiState_js.showHelpWrapper.value = "shortcuts";
            return;
          }
          return;
        }
        if (a === "about") {
          if (uiState_js.showAbout) uiState_js.showAbout.value = true;
          return;
        }
      };
      const handleDocClick = (e) => {
        if (openMenu.value) {
          const path = e.composedPath ? e.composedPath() : [];
          const inMenu = path.some((el) => el.classList && el.classList.contains("menu-dropdown"));
          const inBtn = path.some((el) => el.classList && el.classList.contains("menu-btn") || el.closest && el.closest(".menubar"));
          if (!inMenu && !inBtn) openMenu.value = null;
        }
      };
      vue.onMounted(() => document.addEventListener("click", handleDocClick));
      vue.onUnmounted(() => document.removeEventListener("click", handleDocClick));
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", _hoisted_1$2, [
          (vue.openBlock(), vue.createElementBlock(
            vue.Fragment,
            null,
            vue.renderList(menus, (menu) => {
              return vue.createElementVNode("div", {
                key: menu.label,
                class: "menu-group"
              }, [
                vue.createElementVNode("button", {
                  class: "menu-btn",
                  ref_for: true,
                  ref: (el) => {
                    if (el) btnRefs.value[menu.label] = el;
                  },
                  onClick: ($event) => toggleMenu($event, menu.label),
                  onMouseenter: ($event) => hoverMenu(menu.label)
                }, vue.toDisplayString(menu.label), 41, _hoisted_2$2)
              ]);
            }),
            64
            /* STABLE_FRAGMENT */
          )),
          openMenu.value ? (vue.openBlock(), vue.createElementBlock(
            "div",
            {
              key: 0,
              class: "menu-dropdown",
              style: vue.normalizeStyle(dropdownStyle.value),
              onMouseleave: _cache[0] || (_cache[0] = ($event) => scheduleClose()),
              onMouseenter: _cache[1] || (_cache[1] = ($event) => cancelClose())
            },
            [
              (vue.openBlock(true), vue.createElementBlock(
                vue.Fragment,
                null,
                vue.renderList(currentItems.value, (item, i) => {
                  return vue.openBlock(), vue.createElementBlock(
                    vue.Fragment,
                    { key: i },
                    [
                      item.divider ? (vue.openBlock(), vue.createElementBlock("div", _hoisted_3$1)) : item.label ? (vue.openBlock(), vue.createElementBlock("div", {
                        key: 1,
                        class: vue.normalizeClass(["menu-item", { disabled: item.disabled }]),
                        onClick: ($event) => !item.disabled && execItem(item)
                      }, [
                        vue.createElementVNode(
                          "span",
                          _hoisted_5,
                          vue.toDisplayString(item.label),
                          1
                          /* TEXT */
                        ),
                        item.shortcut ? (vue.openBlock(), vue.createElementBlock(
                          "span",
                          _hoisted_6,
                          vue.toDisplayString(item.shortcut),
                          1
                          /* TEXT */
                        )) : vue.createCommentVNode("v-if", true)
                      ], 10, _hoisted_4$1)) : vue.createCommentVNode("v-if", true)
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
          )) : vue.createCommentVNode("v-if", true)
        ]);
      };
    }
  };
  const MenuBar = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-7ae0e8b5"]]);
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
  const logoUrl = "data:image/svg+xml,%3csvg%20xmlns='http://www.w3.org/2000/svg'%20width='512'%20height='512'%20viewBox='0%200%20512%20512'%3e%3cdefs%3e%3c!--%20背景渐变（深色科技风）%20--%3e%3clinearGradient%20id='bgGrad'%20x1='0'%20y1='0'%20x2='1'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%230a1628'/%3e%3cstop%20offset='100%25'%20stop-color='%230d1f2e'/%3e%3c/linearGradient%3e%3c!--%20左侧尖括号渐变（科技蓝）%20--%3e%3clinearGradient%20id='leftBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='100%25'%20stop-color='%230077b6'/%3e%3c/linearGradient%3e%3c!--%20右侧尖括号渐变（科技绿）%20--%3e%3clinearGradient%20id='rightBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300e676'/%3e%3cstop%20offset='100%25'%20stop-color='%2300c853'/%3e%3c/linearGradient%3e%3c!--%20中间连接线（蓝绿渐变）%20--%3e%3clinearGradient%20id='connector'%20x1='0'%20y1='0'%20x2='1'%20y2='0'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='50%25'%20stop-color='%2300e5ff'/%3e%3cstop%20offset='100%25'%20stop-color='%2300e676'/%3e%3c/linearGradient%3e%3c!--%20外发光%20--%3e%3cfilter%20id='glow'%3e%3cfeGaussianBlur%20stdDeviation='4'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3cfilter%20id='softGlow'%3e%3cfeGaussianBlur%20stdDeviation='8'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3c/defs%3e%3c!--%20圆角方形背景（深色科技底）%20--%3e%3crect%20x='32'%20y='32'%20width='448'%20height='448'%20rx='96'%20ry='96'%20fill='url(%23bgGrad)'%20stroke='%231a3a4a'%20stroke-width='2'/%3e%3c!--%20左侧%20%3c%20尖括号（三段式直线%20—%20科技蓝，代表代码输入/开发者）%20--%3e%3cpath%20d='M180%20150%20L96%20256%20L180%20362'%20stroke='url(%23leftBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20右侧%20%3e%20尖括号（三段式直线%20—%20科技绿，代表代码输出/AI伙伴）%20--%3e%3cpath%20d='M332%20150%20L416%20256%20L332%20362'%20stroke='url(%23rightBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20中间「=」连接线已移除（图标只留%20%3c%20%3e%20尖括号%20+%20中心%20AI%20核心光点）。%20--%3e%3c!--%20中心光点（代表%20AI%20核心%20—%20亮青色）%20--%3e%3ccircle%20cx='256'%20cy='256'%20r='18'%20fill='transparent'%20stroke='%2300e5ff'%20stroke-width='3'%20opacity='0.6'/%3e%3ccircle%20cx='256'%20cy='256'%20r='8'%20fill='%2300e5ff'%20opacity='0.9'%3e%3canimate%20attributeName='opacity'%20values='0.6;1;0.6'%20dur='2s'%20repeatCount='indefinite'/%3e%3c/circle%3e%3c/svg%3e";
  const _hoisted_1 = { class: "app-logo" };
  const _hoisted_2 = ["src"];
  const _hoisted_3 = { class: "title-center" };
  const _hoisted_4 = { class: "title-right" };
  const _sfc_main = {
    __name: "UiTitlebar",
    setup(__props) {
      const menuBarRef = vue.ref(null);
      const titlebarRightEl = vue.ref(null);
      let titlebarRightUnsub = null;
      const wsList = uiState_js.state.wsList;
      const closeAllMenus = () => {
        var _a, _b;
        if (menuBarRef.value) (_b = (_a = menuBarRef.value).closeMenu) == null ? void 0 : _b.call(_a);
      };
      vue.onMounted(() => {
        titlebarRightUnsub = pluginRuntime_js.mountListSlot(titlebarRightEl, "titlebar-right");
      });
      vue.onUnmounted(() => {
        if (titlebarRightUnsub) {
          titlebarRightUnsub();
          titlebarRightUnsub = null;
        }
      });
      return (_ctx, _cache) => {
        return vue.openBlock(), vue.createElementBlock("div", {
          class: "titlebar",
          onClick: closeAllMenus
        }, [
          vue.createElementVNode("div", _hoisted_1, [
            vue.createElementVNode("img", {
              src: vue.unref(logoUrl),
              class: "logo-img",
              alt: "PairCode"
            }, null, 8, _hoisted_2)
          ]),
          vue.createVNode(
            MenuBar,
            {
              ref_key: "menuBarRef",
              ref: menuBarRef
            },
            null,
            512
            /* NEED_PATCH */
          ),
          vue.createElementVNode(
            "div",
            _hoisted_3,
            vue.toDisplayString(vue.unref(uiState_js.state).workspaceName),
            1
            /* TEXT */
          ),
          vue.createElementVNode("div", _hoisted_4, [
            vue.unref(wsList).length > 1 ? (vue.openBlock(), vue.createElementBlock("button", {
              key: 0,
              class: "ws-quick-btn",
              onClick: _cache[0] || (_cache[0] = ($event) => uiState_js.showQuickSwitcher.value = !vue.unref(uiState_js.showQuickSwitcher)),
              title: "快速切换工作区"
            }, [
              vue.createVNode(SvgIcon, {
                name: "folder",
                size: 14
              })
            ])) : vue.createCommentVNode("v-if", true),
            vue.createCommentVNode(" ★ titlebar-right 槽位（list 型）：标题栏右侧细粒度叠加（插件加小按钮/状态） "),
            vue.createElementVNode(
              "div",
              {
                ref_key: "titlebarRightEl",
                ref: titlebarRightEl,
                class: "plugin-slot-host plugin-slot-titlebar"
              },
              null,
              512
              /* NEED_PATCH */
            )
          ])
        ]);
      };
    }
  };
  const UiTitlebar2 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-c1c31662"]]);
  function mount(el) {
    const app = vue.createApp(UiTitlebar2);
    app.mount(el);
    return () => {
      app.unmount();
    };
  }
  exports.mount = mount;
  Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
  return exports;
})({}, window.__PAIRCODE_CORE.Vue, window.__PAIRCODE_CORE.uiState, window.__PAIRCODE_CORE.api, window.__PAIRCODE_CORE.pluginRuntime);
