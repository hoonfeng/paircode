var UiModals=(function(K,e,b,Be,A){"use strict";var Va=Object.defineProperty;var Na=(K,e,b)=>e in K?Va(K,e,{enumerable:!0,configurable:!0,writable:!0,value:b}):K[e]=b;var D=(K,e,b)=>Na(K,typeof e!="symbol"?e+"":e,b);var de;const z=(r,t)=>{const n=r.__vccOpts||r;for(const[o,l]of t)n[o]=l;return n},qe=["width","height"],Je={key:0,d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"},R=z({__name:"SvgIcon",props:{name:{type:String,required:!0},size:{type:Number,default:16}},setup(r){return(t,n)=>(e.openBlock(),e.createElementBlock("svg",{class:"svg-icon",width:r.size,height:r.size,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor","stroke-width":"2","stroke-linecap":"round","stroke-linejoin":"round"},[e.createCommentVNode(" Folder "),r.name==="folder"?(e.openBlock(),e.createElementBlock("path",Je)):r.name==="folder-open"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" Folder Open "),n[0]||(n[0]=e.createElementVNode("path",{d:"M6 17l-3-9h18l-3 9H6z"},null,-1)),n[1]||(n[1]=e.createElementVNode("path",{d:"M4 8V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v3"},null,-1))],64)):r.name==="file"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" File "),n[2]||(n[2]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[3]||(n[3]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1))],64)):r.name==="file-code"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createCommentVNode(" File Code "),n[4]||(n[4]=e.createStaticVNode('<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" data-v-4492abde></path><polyline points="14 2 14 8 20 8" data-v-4492abde></polyline><line x1="10" y1="12" x2="8" y2="14" data-v-4492abde></line><line x1="10" y1="16" x2="8" y2="18" data-v-4492abde></line><line x1="14" y1="12" x2="16" y2="14" data-v-4492abde></line><line x1="14" y1="16" x2="16" y2="18" data-v-4492abde></line>',6))],64)):r.name==="file-text"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:4},[e.createCommentVNode(" File Text / Document "),n[5]||(n[5]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[6]||(n[6]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[7]||(n[7]=e.createElementVNode("line",{x1:"9",y1:"13",x2:"15",y2:"13"},null,-1)),n[8]||(n[8]=e.createElementVNode("line",{x1:"9",y1:"17",x2:"15",y2:"17"},null,-1))],64)):r.name==="search"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:5},[e.createCommentVNode(" Search "),n[9]||(n[9]=e.createElementVNode("circle",{cx:"11",cy:"11",r:"8"},null,-1)),n[10]||(n[10]=e.createElementVNode("line",{x1:"21",y1:"21",x2:"16.65",y2:"16.65"},null,-1))],64)):r.name==="terminal"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:6},[e.createCommentVNode(" Terminal / Console "),n[11]||(n[11]=e.createElementVNode("polyline",{points:"4 17 10 11 4 5"},null,-1)),n[12]||(n[12]=e.createElementVNode("line",{x1:"12",y1:"19",x2:"20",y2:"19"},null,-1))],64)):r.name==="chat"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:7},[e.createCommentVNode(" Chat / Message "),n[13]||(n[13]=e.createElementVNode("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"},null,-1))],64)):r.name==="settings"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:8},[e.createCommentVNode(" Gear / Settings "),n[14]||(n[14]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"3"},null,-1)),n[15]||(n[15]=e.createElementVNode("path",{d:"M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"},null,-1))],64)):r.name==="home"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:9},[e.createCommentVNode(" Home "),n[16]||(n[16]=e.createElementVNode("path",{d:"M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"},null,-1)),n[17]||(n[17]=e.createElementVNode("polyline",{points:"9 22 9 12 15 12 15 22"},null,-1))],64)):r.name==="chevron-right"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:10},[e.createCommentVNode(" Chevron Right "),n[18]||(n[18]=e.createElementVNode("polyline",{points:"9 6 15 12 9 18"},null,-1))],64)):r.name==="chevron-down"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:11},[e.createCommentVNode(" Chevron Down (Rotated chevron-right) "),n[19]||(n[19]=e.createElementVNode("polyline",{points:"6 9 12 15 18 9"},null,-1))],64)):r.name==="plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:12},[e.createCommentVNode(" Plus / Add "),n[20]||(n[20]=e.createElementVNode("line",{x1:"12",y1:"5",x2:"12",y2:"19"},null,-1)),n[21]||(n[21]=e.createElementVNode("line",{x1:"5",y1:"12",x2:"19",y2:"12"},null,-1))],64)):r.name==="close"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:13},[e.createCommentVNode(" Close / X "),n[22]||(n[22]=e.createElementVNode("line",{x1:"18",y1:"6",x2:"6",y2:"18"},null,-1)),n[23]||(n[23]=e.createElementVNode("line",{x1:"6",y1:"6",x2:"18",y2:"18"},null,-1))],64)):r.name==="refresh"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:14},[e.createCommentVNode(" Refresh "),n[24]||(n[24]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[25]||(n[25]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 1 1-2.12-9.36L23 10"},null,-1))],64)):r.name==="drive"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:15},[e.createCommentVNode(" Hard Drive / Disk "),n[26]||(n[26]=e.createElementVNode("line",{x1:"22",y1:"12",x2:"2",y2:"12"},null,-1)),n[27]||(n[27]=e.createElementVNode("path",{d:"M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"},null,-1)),n[28]||(n[28]=e.createElementVNode("line",{x1:"6",y1:"16",x2:"6.01",y2:"16"},null,-1)),n[29]||(n[29]=e.createElementVNode("line",{x1:"10",y1:"16",x2:"10.01",y2:"16"},null,-1))],64)):r.name==="source-control"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:16},[e.createCommentVNode(" Source Control / Git Branch "),n[30]||(n[30]=e.createElementVNode("line",{x1:"6",y1:"3",x2:"6",y2:"15"},null,-1)),n[31]||(n[31]=e.createElementVNode("circle",{cx:"18",cy:"6",r:"3"},null,-1)),n[32]||(n[32]=e.createElementVNode("circle",{cx:"6",cy:"18",r:"3"},null,-1)),n[33]||(n[33]=e.createElementVNode("path",{d:"M18 9a9 9 0 0 1-9 9"},null,-1))],64)):r.name==="git-branch"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:17},[e.createCommentVNode(" Git Branch "),n[34]||(n[34]=e.createElementVNode("line",{x1:"6",y1:"3",x2:"6",y2:"15"},null,-1)),n[35]||(n[35]=e.createElementVNode("circle",{cx:"18",cy:"6",r:"3"},null,-1)),n[36]||(n[36]=e.createElementVNode("circle",{cx:"6",cy:"18",r:"3"},null,-1)),n[37]||(n[37]=e.createElementVNode("path",{d:"M18 9a9 9 0 0 1-9 9"},null,-1))],64)):r.name==="git-pull"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:18},[e.createCommentVNode(" Git Pull "),n[38]||(n[38]=e.createStaticVNode('<circle cx="18" cy="18" r="3" data-v-4492abde></circle><circle cx="6" cy="6" r="3" data-v-4492abde></circle><path d="M13 6h3a2 2 0 0 1 2 2v7" data-v-4492abde></path><line x1="6" y1="18" x2="6" y2="9" data-v-4492abde></line><polyline points="9 9 6 6 3 9" data-v-4492abde></polyline>',5))],64)):r.name==="git-push"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:19},[e.createCommentVNode(" Git Push "),n[39]||(n[39]=e.createStaticVNode('<circle cx="18" cy="6" r="3" data-v-4492abde></circle><circle cx="6" cy="18" r="3" data-v-4492abde></circle><path d="M13 18h-2a2 2 0 0 1-2-2V9" data-v-4492abde></path><line x1="6" y1="6" x2="6" y2="15" data-v-4492abde></line><polyline points="9 15 6 18 3 15" data-v-4492abde></polyline>',5))],64)):r.name==="output"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:20},[e.createCommentVNode(" Output / Window "),n[40]||(n[40]=e.createElementVNode("rect",{x:"2",y:"3",width:"20",height:"14",rx:"2",ry:"2"},null,-1)),n[41]||(n[41]=e.createElementVNode("line",{x1:"8",y1:"21",x2:"16",y2:"21"},null,-1)),n[42]||(n[42]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12",y2:"21"},null,-1))],64)):r.name==="warning"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:21},[e.createCommentVNode(" Warning / Alert "),n[43]||(n[43]=e.createElementVNode("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"},null,-1)),n[44]||(n[44]=e.createElementVNode("line",{x1:"12",y1:"9",x2:"12",y2:"13"},null,-1)),n[45]||(n[45]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"},null,-1))],64)):r.name==="undo"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:22},[e.createCommentVNode(" Undo "),n[46]||(n[46]=e.createElementVNode("polyline",{points:"1 4 1 10 7 10"},null,-1)),n[47]||(n[47]=e.createElementVNode("path",{d:"M3.51 15a9 9 0 1 0 2.13-9.36L1 10"},null,-1))],64)):r.name==="redo"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:23},[e.createCommentVNode(" Redo "),n[48]||(n[48]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[49]||(n[49]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 1 1-2.12-9.36L23 10"},null,-1))],64)):r.name==="package"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:24},[e.createCommentVNode(" Package / Box / Store "),n[50]||(n[50]=e.createElementVNode("line",{x1:"16.5",y1:"9.4",x2:"7.5",y2:"4.21"},null,-1)),n[51]||(n[51]=e.createElementVNode("path",{d:"M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"},null,-1)),n[52]||(n[52]=e.createElementVNode("polyline",{points:"3.27 6.96 12 12.01 20.73 6.96"},null,-1)),n[53]||(n[53]=e.createElementVNode("line",{x1:"12",y1:"22.08",x2:"12",y2:"12"},null,-1))],64)):r.name==="globe"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:25},[e.createCommentVNode(" Globe / External "),n[54]||(n[54]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[55]||(n[55]=e.createElementVNode("line",{x1:"2",y1:"12",x2:"22",y2:"12"},null,-1)),n[56]||(n[56]=e.createElementVNode("path",{d:"M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"},null,-1))],64)):r.name==="download"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:26},[e.createCommentVNode(" Download (更新/安装) "),n[57]||(n[57]=e.createElementVNode("path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"},null,-1)),n[58]||(n[58]=e.createElementVNode("polyline",{points:"7 10 12 15 17 10"},null,-1)),n[59]||(n[59]=e.createElementVNode("line",{x1:"12",y1:"15",x2:"12",y2:"3"},null,-1))],64)):r.name==="cycle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:27},[e.createCommentVNode(" Refresh / Cycle (for agent) "),n[60]||(n[60]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[61]||(n[61]=e.createElementVNode("polyline",{points:"1 20 1 14 7 14"},null,-1)),n[62]||(n[62]=e.createElementVNode("path",{d:"M3.51 9a9 9 0 0 1 14.85-3.36L23 10"},null,-1)),n[63]||(n[63]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 0 1-14.85 3.36L1 14"},null,-1))],64)):r.name==="send"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:28},[e.createCommentVNode(" Send (arrow up) "),n[64]||(n[64]=e.createElementVNode("line",{x1:"12",y1:"19",x2:"12",y2:"5"},null,-1)),n[65]||(n[65]=e.createElementVNode("polyline",{points:"5 12 12 5 19 12"},null,-1))],64)):r.name==="send-plane"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:29},[e.createCommentVNode(" Send Plane (paper airplane) "),n[66]||(n[66]=e.createElementVNode("line",{x1:"22",y1:"2",x2:"11",y2:"13"},null,-1)),n[67]||(n[67]=e.createElementVNode("polygon",{points:"22 2 15 22 11 13 2 9 22 2"},null,-1))],64)):r.name==="stop-dot"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:30},[e.createCommentVNode(" Stop Dot (pulsing circle) "),n[68]||(n[68]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"6",class:"stop-pulse"},null,-1)),n[69]||(n[69]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10",class:"stop-pulse-ring"},null,-1))],64)):r.name==="wrench"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:31},[e.createCommentVNode(" Wrench / Tool "),n[70]||(n[70]=e.createElementVNode("path",{d:"M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"},null,-1))],64)):r.name==="database"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:32},[e.createCommentVNode(" Database "),n[71]||(n[71]=e.createElementVNode("ellipse",{cx:"12",cy:"5",rx:"9",ry:"3"},null,-1)),n[72]||(n[72]=e.createElementVNode("path",{d:"M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"},null,-1)),n[73]||(n[73]=e.createElementVNode("path",{d:"M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"},null,-1))],64)):r.name==="user"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:33},[e.createCommentVNode(" User / Person "),n[74]||(n[74]=e.createElementVNode("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"},null,-1)),n[75]||(n[75]=e.createElementVNode("circle",{cx:"12",cy:"7",r:"4"},null,-1))],64)):r.name==="info"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:34},[e.createCommentVNode(" Info "),n[76]||(n[76]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[77]||(n[77]=e.createElementVNode("line",{x1:"12",y1:"16",x2:"12",y2:"12"},null,-1)),n[78]||(n[78]=e.createElementVNode("line",{x1:"12",y1:"8",x2:"12.01",y2:"8"},null,-1))],64)):r.name==="lightbulb"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:35},[e.createCommentVNode(" Lightbulb / Suggestion "),n[79]||(n[79]=e.createElementVNode("path",{d:"M9 18h6"},null,-1)),n[80]||(n[80]=e.createElementVNode("path",{d:"M10 22h4"},null,-1)),n[81]||(n[81]=e.createElementVNode("path",{d:"M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14"},null,-1))],64)):r.name==="sparkles"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:36},[e.createCommentVNode(" Sparkles / Auto "),n[82]||(n[82]=e.createStaticVNode('<path d="M13.5 4L15 8l4 .5L15 12l1.5 4-4-2-4 2L10 12l-4-3.5L10 8z" data-v-4492abde></path><line x1="3" y1="18" x2="3" y2="21" data-v-4492abde></line><line x1="21" y1="18" x2="21" y2="21" data-v-4492abde></line><line x1="7" y1="20" x2="11" y2="20" data-v-4492abde></line><line x1="17" y1="20" x2="19" y2="20" data-v-4492abde></line>',5))],64)):r.name==="bot"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:37},[e.createCommentVNode(" Bot / AI "),n[83]||(n[83]=e.createStaticVNode('<rect x="3" y="11" width="18" height="10" rx="2" data-v-4492abde></rect><circle cx="12" cy="5" r="2" data-v-4492abde></circle><path d="M12 7v4" data-v-4492abde></path><line x1="8" y1="16" x2="8" y2="16" data-v-4492abde></line><line x1="16" y1="16" x2="16" y2="16" data-v-4492abde></line>',5))],64)):r.name==="file-js"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:38},[e.createCommentVNode(" File Type Icons "),n[84]||(n[84]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[85]||(n[85]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[86]||(n[86]=e.createElementVNode("text",{x:"8",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"JS",-1))],64)):r.name==="file-ts"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:39},[n[87]||(n[87]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[88]||(n[88]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[89]||(n[89]=e.createElementVNode("text",{x:"8",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"TS",-1))],64)):r.name==="file-go"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:40},[n[90]||(n[90]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[91]||(n[91]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[92]||(n[92]=e.createElementVNode("text",{x:"9",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Go",-1))],64)):r.name==="file-py"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:41},[n[93]||(n[93]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[94]||(n[94]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[95]||(n[95]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Py",-1))],64)):r.name==="file-java"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:42},[n[96]||(n[96]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[97]||(n[97]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[98]||(n[98]=e.createElementVNode("text",{x:"6",y:"17","font-size":"8",fill:"currentColor","font-weight":"bold",stroke:"none"},"Java",-1))],64)):r.name==="file-html"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:43},[n[99]||(n[99]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[100]||(n[100]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[101]||(n[101]=e.createElementVNode("text",{x:"6",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"HTML",-1))],64)):r.name==="file-css"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:44},[n[102]||(n[102]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[103]||(n[103]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[104]||(n[104]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"CSS",-1))],64)):r.name==="file-json"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:45},[n[105]||(n[105]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[106]||(n[106]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[107]||(n[107]=e.createElementVNode("text",{x:"5",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"{ }",-1))],64)):r.name==="file-md"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:46},[n[108]||(n[108]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[109]||(n[109]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[110]||(n[110]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"MD",-1))],64)):r.name==="file-vue"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:47},[n[111]||(n[111]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[112]||(n[112]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[113]||(n[113]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Vue",-1))],64)):r.name==="copy"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:48},[e.createCommentVNode(" Copy "),n[114]||(n[114]=e.createElementVNode("rect",{x:"9",y:"9",width:"13",height:"13",rx:"2",ry:"2"},null,-1)),n[115]||(n[115]=e.createElementVNode("path",{d:"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"},null,-1))],64)):r.name==="minus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:49},[e.createCommentVNode(" Minus "),n[116]||(n[116]=e.createElementVNode("line",{x1:"5",y1:"12",x2:"19",y2:"12"},null,-1))],64)):r.name==="edit"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:50},[e.createCommentVNode(" Edit / Rename "),n[117]||(n[117]=e.createElementVNode("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"},null,-1)),n[118]||(n[118]=e.createElementVNode("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"},null,-1))],64)):r.name==="trash"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:51},[e.createCommentVNode(" Trash / Delete "),n[119]||(n[119]=e.createElementVNode("polyline",{points:"3 6 5 6 21 6"},null,-1)),n[120]||(n[120]=e.createElementVNode("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"},null,-1))],64)):r.name==="file-plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:52},[e.createCommentVNode(" File Plus / New File "),n[121]||(n[121]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[122]||(n[122]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[123]||(n[123]=e.createElementVNode("line",{x1:"12",y1:"18",x2:"12",y2:"12"},null,-1)),n[124]||(n[124]=e.createElementVNode("line",{x1:"9",y1:"15",x2:"15",y2:"15"},null,-1))],64)):r.name==="message-square"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:53},[e.createCommentVNode(" Folder Plus / New Folder "),n[125]||(n[125]=e.createElementVNode("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"},null,-1))],64)):r.name==="folder-plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:54},[n[126]||(n[126]=e.createElementVNode("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v3"},null,-1)),n[127]||(n[127]=e.createElementVNode("line",{x1:"12",y1:"11",x2:"12",y2:"17"},null,-1)),n[128]||(n[128]=e.createElementVNode("line",{x1:"9",y1:"14",x2:"15",y2:"14"},null,-1))],64)):r.name==="brain"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:55},[e.createCommentVNode(" Brain / Thinking "),n[129]||(n[129]=e.createElementVNode("path",{d:"M12 2a4 4 0 0 0-4 4v1a5 5 0 0 0-5 5v1a4 4 0 0 0 3 3.87V17a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-.13A4 4 0 0 0 21 13v-1a5 5 0 0 0-5-5V6a4 4 0 0 0-4-4z"},null,-1)),n[130]||(n[130]=e.createElementVNode("path",{d:"M9 12v2"},null,-1)),n[131]||(n[131]=e.createElementVNode("path",{d:"M15 12v2"},null,-1)),n[132]||(n[132]=e.createElementVNode("path",{d:"M12 9v5"},null,-1))],64)):r.name==="check"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:56},[e.createCommentVNode(" Check / Success "),n[133]||(n[133]=e.createElementVNode("polyline",{points:"20 6 9 17 4 12"},null,-1))],64)):r.name==="clock"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:57},[e.createCommentVNode(" Clock / Pending "),n[134]||(n[134]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[135]||(n[135]=e.createElementVNode("polyline",{points:"12 6 12 12 16 14"},null,-1))],64)):r.name==="help"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:58},[e.createCommentVNode(" Help / Question "),n[136]||(n[136]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[137]||(n[137]=e.createElementVNode("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"},null,-1)),n[138]||(n[138]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"},null,-1))],64)):r.name==="shield"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:59},[e.createCommentVNode(" Shield / Approval "),n[139]||(n[139]=e.createElementVNode("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"},null,-1))],64)):r.name==="shield-off"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:60},[e.createCommentVNode(" Shield Off / No Review "),n[140]||(n[140]=e.createElementVNode("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"},null,-1)),n[141]||(n[141]=e.createElementVNode("line",{x1:"4",y1:"4",x2:"20",y2:"20",stroke:"currentColor","stroke-width":"2","stroke-linecap":"round"},null,-1))],64)):r.name==="code"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:61},[e.createCommentVNode(" Code / Brackets "),n[142]||(n[142]=e.createElementVNode("polyline",{points:"16 18 22 12 16 6"},null,-1)),n[143]||(n[143]=e.createElementVNode("polyline",{points:"8 6 2 12 8 18"},null,-1))],64)):r.name==="list"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:62},[e.createCommentVNode(" List / Menu "),n[144]||(n[144]=e.createStaticVNode('<line x1="8" y1="6" x2="21" y2="6" data-v-4492abde></line><line x1="8" y1="12" x2="21" y2="12" data-v-4492abde></line><line x1="8" y1="18" x2="21" y2="18" data-v-4492abde></line><line x1="3" y1="6" x2="3.01" y2="6" data-v-4492abde></line><line x1="3" y1="12" x2="3.01" y2="12" data-v-4492abde></line><line x1="3" y1="18" x2="3.01" y2="18" data-v-4492abde></line>',6))],64)):r.name==="layers"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:63},[e.createCommentVNode(" Layers / Stack / Context "),n[145]||(n[145]=e.createElementVNode("polygon",{points:"12 2 2 7 12 12 22 7 12 2"},null,-1)),n[146]||(n[146]=e.createElementVNode("polyline",{points:"2 17 12 22 22 17"},null,-1)),n[147]||(n[147]=e.createElementVNode("polyline",{points:"2 12 12 17 22 12"},null,-1))],64)):r.name==="eye"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:64},[e.createCommentVNode(" Eye / Show "),n[148]||(n[148]=e.createElementVNode("path",{d:"M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"},null,-1)),n[149]||(n[149]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"3"},null,-1))],64)):r.name==="eye-off"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:65},[e.createCommentVNode(" Eye Off / Hide "),n[150]||(n[150]=e.createElementVNode("path",{d:"M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"},null,-1)),n[151]||(n[151]=e.createElementVNode("path",{d:"M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"},null,-1)),n[152]||(n[152]=e.createElementVNode("line",{x1:"1",y1:"1",x2:"23",y2:"23"},null,-1))],64)):r.name==="bug"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:66},[e.createCommentVNode(" Bug "),n[153]||(n[153]=e.createStaticVNode('<rect x="8" y="2" width="8" height="4" rx="1" ry="1" data-v-4492abde></rect><path d="M20 12h-3a5 5 0 0 1-5 5 5 5 0 0 1-5-5H4" data-v-4492abde></path><path d="M4 8h16" data-v-4492abde></path><path d="M12 2v7" data-v-4492abde></path><path d="M9 17l-3 4" data-v-4492abde></path><path d="M15 17l3 4" data-v-4492abde></path>',6))],64)):r.name==="check-circle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:67},[e.createCommentVNode(" Check Circle "),n[154]||(n[154]=e.createElementVNode("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"},null,-1)),n[155]||(n[155]=e.createElementVNode("polyline",{points:"22 4 12 14.01 9 11.01"},null,-1))],64)):r.name==="book-open"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:68},[e.createCommentVNode(" Book Open / Documentation "),n[156]||(n[156]=e.createElementVNode("path",{d:"M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"},null,-1)),n[157]||(n[157]=e.createElementVNode("path",{d:"M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"},null,-1))],64)):r.name==="tool"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:69},[e.createCommentVNode(" Tool / Wrench alternate "),n[158]||(n[158]=e.createElementVNode("path",{d:"M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"},null,-1))],64)):r.name==="keyboard"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:70},[e.createCommentVNode(" Keyboard "),n[159]||(n[159]=e.createStaticVNode('<rect x="2" y="4" width="20" height="16" rx="2" ry="2" data-v-4492abde></rect><line x1="6" y1="8" x2="6.01" y2="8" data-v-4492abde></line><line x1="10" y1="8" x2="10.01" y2="8" data-v-4492abde></line><line x1="14" y1="8" x2="14.01" y2="8" data-v-4492abde></line><line x1="18" y1="8" x2="18.01" y2="8" data-v-4492abde></line><line x1="6" y1="12" x2="6.01" y2="12" data-v-4492abde></line><line x1="10" y1="12" x2="10.01" y2="12" data-v-4492abde></line><line x1="14" y1="12" x2="14.01" y2="12" data-v-4492abde></line><line x1="18" y1="12" x2="18.01" y2="12" data-v-4492abde></line><line x1="6" y1="16" x2="18" y2="16" data-v-4492abde></line>',10))],64)):r.name==="chevron-left"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:71},[e.createCommentVNode(" Chevron Left "),n[160]||(n[160]=e.createElementVNode("polyline",{points:"15 6 9 12 15 18"},null,-1))],64)):r.name==="grid"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:72},[e.createCommentVNode(" Grid / App Grid "),n[161]||(n[161]=e.createElementVNode("rect",{x:"3",y:"3",width:"7",height:"7"},null,-1)),n[162]||(n[162]=e.createElementVNode("rect",{x:"14",y:"3",width:"7",height:"7"},null,-1)),n[163]||(n[163]=e.createElementVNode("rect",{x:"14",y:"14",width:"7",height:"7"},null,-1)),n[164]||(n[164]=e.createElementVNode("rect",{x:"3",y:"14",width:"7",height:"7"},null,-1))],64)):r.name==="puzzle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:73},[e.createCommentVNode(" Puzzle / 插件 "),n[165]||(n[165]=e.createElementVNode("path",{d:"M4 7h3a2 2 0 0 1 4 0h9v9h-3a2 2 0 0 0-4 0H4z"},null,-1)),n[166]||(n[166]=e.createElementVNode("path",{d:"M11 7v9"},null,-1))],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:74},[n[167]||(n[167]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[168]||(n[168]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[169]||(n[169]=e.createElementVNode("line",{x1:"9",y1:"13",x2:"15",y2:"13"},null,-1)),n[170]||(n[170]=e.createElementVNode("line",{x1:"9",y1:"17",x2:"15",y2:"17"},null,-1))],64))],8,qe))}},[["__scopeId","data-v-4492abde"]]),Ke={class:"me-field"},Ze={class:"me-label"},_e={class:"me-editor"},Qe={class:"me-input-row"},Xe=["placeholder","onKeydown"],Ye={class:"me-tags"},ve={key:0,class:"me-empty"},en=["onClick"],Te=z({__name:"ModelEditor",props:{models:{type:Array,default:()=>[]},label:{type:String,default:"可用模型（回车或逗号分隔添加；支持整段粘贴）"},placeholder:{type:String,default:"输入模型名，回车添加…"}},emits:["change"],setup(r,{emit:t}){const n=r,o=t,l=e.ref(""),s=e.ref([...n.models]);e.watch(()=>n.models,u=>{s.value=[...u]});function a(){const u=l.value.split(/[\n,，]/).map(w=>w.trim()).filter(Boolean);let h=!1;for(const w of u)s.value.includes(w)||(s.value.push(w),h=!0);h&&o("change",[...s.value]),l.value=""}function m(u){const h=(u.clipboardData||window.clipboardData).getData("text");if(/[,\n，]/.test(h)){u.preventDefault();const w=h.split(/[\n,，]/).map(T=>T.trim()).filter(Boolean);let V=!1;for(const T of w)s.value.includes(T)||(s.value.push(T),V=!0);V&&o("change",[...s.value]),l.value=""}}function c(u){s.value.splice(u,1),o("change",[...s.value])}return(u,h)=>(e.openBlock(),e.createElementBlock("div",Ke,[e.createElementVNode("span",Ze,e.toDisplayString(r.label),1),e.createElementVNode("div",_e,[e.createElementVNode("div",Qe,[e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":h[0]||(h[0]=w=>l.value=w),class:"me-input",placeholder:r.placeholder,onKeydown:e.withKeys(e.withModifiers(a,["prevent"]),["enter"]),onPaste:m},null,40,Xe),[[e.vModelText,l.value]]),e.createElementVNode("button",{class:"me-btn",onClick:a},"添加")]),e.createElementVNode("div",Ye,[s.value.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("span",ve,"暂无模型——添加后 AI tab 的模型下拉会按服务商显示")),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(s.value,(w,V)=>(e.openBlock(),e.createElementBlock("span",{key:w+V,class:"me-tag"},[e.createTextVNode(e.toDisplayString(w)+" ",1),e.createElementVNode("button",{class:"me-x",title:"移除",onClick:T=>c(V)},"×",8,en)]))),128))])])]))}},[["__scopeId","data-v-6ff4531a"]]),nn={class:"provider-manager"},tn={class:"pm-toolbar"},rn={class:"pm-count"},on={key:0,class:"pm-edit"},ln={class:"pm-field"},an={class:"pm-field"},sn={class:"pm-field"},cn={class:"pm-field-label"},dn=["title"],pn=["value"],mn={key:0,class:"pm-protocol-hint"},gn={class:"pm-field"},kn={class:"pm-field"},hn={class:"pm-field"},fn=["value"],un={class:"pm-field"},yn={class:"pm-params"},bn={key:0,class:"pm-param-rows"},En=["title"],xn=["title"],Vn=["onUpdate:modelValue"],Nn=["onUpdate:modelValue","title"],wn=["value"],Bn=["onUpdate:modelValue","min","step","placeholder","title"],Tn=["onUpdate:modelValue","placeholder","title"],Sn={key:1,class:"pm-params-empty"},Cn={class:"pm-edit-actions"},In=["disabled"],Pn={key:1,class:"pm-cards"},An={key:0,class:"pm-edit"},Mn={class:"pm-edit-title"},$n={class:"pm-field"},Dn={class:"pm-field"},Rn={class:"pm-field"},Ln={class:"pm-field-label"},Fn=["title"],Gn=["value"],On={key:0,class:"pm-protocol-hint"},Un={class:"pm-field"},zn={class:"pm-field"},jn={class:"pm-field"},Hn=["value"],Wn={class:"pm-field"},qn={class:"pm-params"},Jn={key:0,class:"pm-param-rows"},Kn=["title"],Zn=["title"],_n=["onUpdate:modelValue"],Qn=["onUpdate:modelValue","title"],Xn=["value"],Yn=["onUpdate:modelValue","min","step","placeholder","title"],vn=["onUpdate:modelValue","placeholder","title"],et={key:1,class:"pm-params-empty"},nt={class:"pm-edit-actions"},tt=["disabled"],rt={key:1,class:"pm-card"},ot={class:"pm-card-head"},lt=["title"],at={class:"pm-ops"},st=["onClick"],it=["onClick"],ct=["title"],dt=["title"],pt={class:"pm-ctx"},mt={class:"pm-models"},gt={key:0,class:"pm-none"},kt={key:1,class:"pm-params-summary"},ht={key:2,class:"pm-empty"},ft={key:3,class:"pm-error"},ut=z({__name:"ProviderManager",props:{modelParamFields:{type:Array,default:()=>[]},modelEditor:{type:Object,default:()=>({})},protocolLabel:{type:String,default:"LLM 协议"},protocolOptions:{type:Array,default:()=>[]},protocolHint:{type:String,default:""}},emits:["saved"],setup(r,{emit:t}){const n=t,o=r,l=e.ref([]),s=e.ref(""),a=e.ref({name:"",baseURL:"",contextMaxTokens:0,temperature:"",thinkingMode:"",maxTokens:0,protocol:""}),m=e.ref([]),c=e.ref({}),u=e.ref(""),h=e.ref(!1),w=["","none","minimal","low","medium","high","xhigh","max"],V=e.computed(()=>{const N=o.modelParamFields.find(f=>f.name==="thinkingMode");return N&&Array.isArray(N.options)&&N.options.length?N.options:w});function T(){const N={};for(const f of o.modelParamFields)f.type==="checkbox"?N[f.name]=!1:f.type==="number"?N[f.name]=0:N[f.name]="";return N}function y(N){const f=l.value.find(E=>E.name===N);return JSON.parse(JSON.stringify(f&&f.modelParams||{}))}async function B(){try{const N=await A.getModels(),f=N.providerModelParams||{},E=N.providerTemperatures||{},i=N.providerMaxTokens||{},d=N.providerThinkingModes||{};l.value=(N.providers||[]).map(g=>({name:g,baseURL:(N.providerBaseURLs||{})[g]||"",contextMaxTokens:(N.providerContexts||{})[g]||0,protocol:(N.providerProtocols||{})[g]||"",models:(N.models||{})[g]||[],temperature:E[g]||"",thinkingMode:d[g]||"",maxTokens:i[g]||0,modelParams:f[g]||{}})),u.value=""}catch(N){u.value="加载服务商失败: "+(N.message||N)}}e.onMounted(B);function I(){s.value="__new__",a.value={name:"",baseURL:"",contextMaxTokens:0,temperature:"",thinkingMode:"",maxTokens:0,protocol:""},m.value=[],c.value={},u.value=""}function L(N){s.value=N.name,a.value={name:N.name,baseURL:N.baseURL,contextMaxTokens:N.contextMaxTokens||0,temperature:N.temperature||"",thinkingMode:N.thinkingMode||"",maxTokens:N.maxTokens||0,protocol:N.protocol||""},m.value=[...N.models||[]];const f=y(N.name),E=T();for(const i of m.value)f[i]||(f[i]={...E});c.value=f,u.value=""}function F(N){const f={...c.value},E=T();for(const i of N)f[i]||(f[i]={...E});for(const i of Object.keys(f))N.includes(i)||delete f[i];c.value=f,m.value=N}function U(){s.value="",u.value=""}function H(){const N={};for(const f of l.value)N[f.name]={baseURL:f.baseURL,models:f.models,contextMaxTokens:f.contextMaxTokens||0,protocol:f.protocol||"",temperature:f.temperature||"",thinkingMode:f.thinkingMode||"",maxTokens:f.maxTokens||0,modelParams:f.modelParams||{}};return N}async function j(){const N=s.value,f=N!=="__new__"&&N!=="",E=a.value.name.trim()||(f?N:"");if(!E){u.value="服务商名称不能为空";return}if(E!==N&&l.value.some(i=>i.name===E)){u.value=`服务商「${E}」已存在`;return}h.value=!0;try{let i=null;f&&E!==N&&(i=await A.renameProvider(N,E));const d=H();if(f&&E!==N&&delete d[N],d[E]={baseURL:a.value.baseURL.trim(),models:m.value,contextMaxTokens:Math.max(0,Number(a.value.contextMaxTokens)||0),protocol:(a.value.protocol||"").trim(),temperature:String(a.value.temperature??"").trim(),thinkingMode:String(a.value.thinkingMode??"").trim(),maxTokens:Math.max(0,Number(a.value.maxTokens)||0),modelParams:G()},await A.saveModels(d),s.value="",await B(),n("saved"),i&&i.renamed){const g=(i.updatedPresets||[]).length;window.$toast(`已改名为「${E}」`+(g?`，同步更新 ${g} 条 AI 配置`:""),"success")}}catch(i){u.value=(f&&E!==N?"改名失败: ":"保存失败: ")+(i.message||i)}finally{h.value=!1}}function G(){const N={};for(const[f,E]of Object.entries(c.value)){const i=E||{},d={};for(const g of o.modelParamFields){const k=i[g.name];g.type==="checkbox"?k===!0&&(d[g.name]=!0):g.type==="number"?Number(k)>0&&(d[g.name]=Number(k)):k!==""&&k!==void 0&&k!==null&&(d[g.name]=k)}Object.keys(d).length&&(N[f]=d)}return N}async function P(N){let f=[];try{const d=await A.getAiPresets(),g=d&&d.presets||{};f=Object.keys(g).filter(k=>(g[k]||{}).provider===N.name).sort()}catch{}let E=`删除服务商「${N.name}」？
（AI tab 将不再可选该服务商）`;if(f.length&&(E+=`

以下 ${f.length} 条 AI 配置仍引用它：
· ${f.join(`
· `)}

删除后它们仍可继续聊天（配置是完整快照），但在对话面板里会失去模型分组；建议先在「AI 配置」里把它们改选到其他服务商。`),!window.confirm(E))return;const i=H();delete i[N.name];try{await A.saveModels(i),await B(),n("saved")}catch(d){u.value="删除失败: "+(d.message||d)}}function S(N){const f=l.value.find(d=>d.name===N);if(!f)return"";const E=Object.keys(f.modelParams||{}).length,i=[];return f.temperature&&i.push("温度 "+f.temperature),f.thinkingMode&&i.push("思考 "+f.thinkingMode),f.maxTokens>0&&i.push("输出上限 "+f.maxTokens),E&&i.push("模型参数 "+E+" 个"),i.join(" · ")}return(N,f)=>(e.openBlock(),e.createElementBlock("div",nn,[e.createCommentVNode(" 工具栏 "),e.createElementVNode("div",tn,[e.createElementVNode("span",rn,e.toDisplayString(l.value.length)+" 个服务商",1),e.createElementVNode("button",{class:"pm-btn pm-primary",onClick:I},"+ 新增服务商")]),e.createCommentVNode(" 新增表单（工具栏下方展开，紧邻按钮不跳动） "),s.value==="__new__"?(e.openBlock(),e.createElementBlock("div",on,[f[21]||(f[21]=e.createElementVNode("div",{class:"pm-edit-title"},"新增服务商",-1)),e.createElementVNode("div",ln,[f[14]||(f[14]=e.createElementVNode("span",{class:"pm-field-label"},"服务商名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[0]||(f[0]=E=>a.value.name=E),placeholder:"如 deepseek"},null,512),[[e.vModelText,a.value.name]])]),e.createElementVNode("div",an,[f[15]||(f[15]=e.createElementVNode("span",{class:"pm-field-label"},"API URL（基础地址或完整端点）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[1]||(f[1]=E=>a.value.baseURL=E),placeholder:"https://api.deepseek.com/v1（基础地址；旧完整端点亦兼容）"},null,512),[[e.vModelText,a.value.baseURL]])]),e.createElementVNode("div",sn,[e.createElementVNode("span",cn,e.toDisplayString(r.protocolLabel),1),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":f[2]||(f[2]=E=>a.value.protocol=E),title:r.protocolHint},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.protocolOptions,E=>(e.openBlock(),e.createElementBlock("option",{key:"p"+E,value:E},e.toDisplayString(E||"默认"),9,pn))),128))],8,dn),[[e.vModelSelect,a.value.protocol]]),r.protocolHint?(e.openBlock(),e.createElementBlock("span",mn,e.toDisplayString(r.protocolHint),1)):e.createCommentVNode("v-if",!0)]),e.createElementVNode("div",gn,[f[16]||(f[16]=e.createElementVNode("span",{class:"pm-field-label"},"上下文大小（Token）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[3]||(f[3]=E=>a.value.contextMaxTokens=E),type:"number",min:"0",step:"1000",placeholder:"0=不限制（模型级未配置时的默认窗口）"},null,512),[[e.vModelText,a.value.contextMaxTokens]])]),e.createElementVNode("div",kn,[f[17]||(f[17]=e.createElementVNode("span",{class:"pm-field-label"},"默认温度",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[4]||(f[4]=E=>a.value.temperature=E),placeholder:"如 0.3（空=不设；模型级可覆盖）"},null,512),[[e.vModelText,a.value.temperature]])]),e.createCommentVNode(` ★ 2026-09-25 服务商级生成参数：生成参数唯一来源 = 本页（模型级可逐模型覆盖），
           设置面板的「生成参数」页已移除（选项来自插件 schema modelParamFields）。 `),e.createElementVNode("div",hn,[f[18]||(f[18]=e.createElementVNode("span",{class:"pm-field-label"},"默认思考档位",-1)),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":f[5]||(f[5]=E=>a.value.thinkingMode=E),title:"思考档位（OpenAI reasoning.effort 口径）：空=不下发；模型级可覆盖"},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(V.value,E=>(e.openBlock(),e.createElementBlock("option",{key:"t"+E,value:E},e.toDisplayString(E||"默认（不设）"),9,fn))),128))],512),[[e.vModelSelect,a.value.thinkingMode]])]),e.createElementVNode("div",un,[f[19]||(f[19]=e.createElementVNode("span",{class:"pm-field-label"},"默认输出 Token（最大输出）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[6]||(f[6]=E=>a.value.maxTokens=E),type:"number",min:"0",step:"1024",placeholder:"0=不设（模型级可覆盖）"},null,512),[[e.vModelText,a.value.maxTokens]])]),e.createVNode(Te,{models:m.value,label:r.modelEditor.label||"可用模型（回车或逗号分隔添加；支持整段粘贴）",placeholder:r.modelEditor.placeholder||"输入模型名，回车添加…",onChange:F},null,8,["models","label","placeholder"]),e.createElementVNode("div",yn,[f[20]||(f[20]=e.createElementVNode("div",{class:"pm-params-title"},"模型参数（每模型独立配置；对话里也可临时切换思考档位）",-1)),m.value.length?(e.openBlock(),e.createElementBlock("div",bn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(m.value,E=>(e.openBlock(),e.createElementBlock("div",{key:E,class:"pm-param-row"},[e.createElementVNode("span",{class:"pm-param-model",title:E},e.toDisplayString(E),9,En),e.createCommentVNode(" ★ 2026-08-21 schema 驱动：按 modelParamFields 动态渲染（checkbox/select/number/text） "),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.modelParamFields,i=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:i.name},[i.type==="checkbox"?(e.openBlock(),e.createElementBlock("label",{key:0,class:"pm-param-check",title:i.hint||i.label},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":d=>c.value[E][i.name]=d},null,8,Vn),[[e.vModelCheckbox,c.value[E][i.name]]]),e.createTextVNode(" "+e.toDisplayString(i.label),1)],8,xn)):i.type==="select"?e.withDirectives((e.openBlock(),e.createElementBlock("select",{key:1,"onUpdate:modelValue":d=>c.value[E][i.name]=d,title:i.hint||i.label},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(i.options||[],d=>(e.openBlock(),e.createElementBlock("option",{key:"o"+d,value:d},e.toDisplayString(d===""?i.label+"默认":d),9,wn))),128))],8,Nn)),[[e.vModelSelect,c.value[E][i.name]]]):i.type==="number"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:2,"onUpdate:modelValue":d=>c.value[E][i.name]=d,type:"number",min:i.min??0,step:i.step??1,placeholder:i.label,title:i.hint||i.label},null,8,Bn)),[[e.vModelText,c.value[E][i.name],void 0,{number:!0}]]):e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:3,"onUpdate:modelValue":d=>c.value[E][i.name]=d,type:"text",placeholder:i.label,title:i.hint||i.label},null,8,Tn)),[[e.vModelText,c.value[E][i.name]]])],64))),128))]))),128))])):(e.openBlock(),e.createElementBlock("div",Sn,"添加模型后，可逐模型配置参数（温度/思考/输出/上下文/多模态…）"))]),e.createElementVNode("div",Cn,[e.createElementVNode("button",{class:"pm-btn pm-primary",disabled:h.value,onClick:j},e.toDisplayString(h.value?"保存中…":"保存服务商"),9,In),e.createElementVNode("button",{class:"pm-btn",onClick:U},"取消")])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 服务商卡片列表（编辑时在卡片位置就地展开表单，不跳顶） "),l.value.length?(e.openBlock(),e.createElementBlock("div",Pn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(l.value,E=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:E.name},[s.value===E.name?(e.openBlock(),e.createElementBlock("div",An,[e.createElementVNode("div",Mn,"编辑服务商："+e.toDisplayString(E.name),1),e.createElementVNode("div",$n,[f[22]||(f[22]=e.createElementVNode("span",{class:"pm-field-label"},"服务商名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[7]||(f[7]=i=>a.value.name=i),placeholder:"如 deepseek"},null,512),[[e.vModelText,a.value.name]]),f[23]||(f[23]=e.createElementVNode("span",{class:"pm-hint"},"改名会一并同步引用该服务商的 AI 配置（连接信息不丢）；空=保持原名",-1))]),e.createElementVNode("div",Dn,[f[24]||(f[24]=e.createElementVNode("span",{class:"pm-field-label"},"API URL（基础地址或完整端点）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[8]||(f[8]=i=>a.value.baseURL=i),placeholder:"https://api.deepseek.com/v1（基础地址；旧完整端点亦兼容）"},null,512),[[e.vModelText,a.value.baseURL]])]),e.createElementVNode("div",Rn,[e.createElementVNode("span",Ln,e.toDisplayString(r.protocolLabel),1),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":f[9]||(f[9]=i=>a.value.protocol=i),title:r.protocolHint},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.protocolOptions,i=>(e.openBlock(),e.createElementBlock("option",{key:"p"+i,value:i},e.toDisplayString(i||"默认"),9,Gn))),128))],8,Fn),[[e.vModelSelect,a.value.protocol]]),r.protocolHint?(e.openBlock(),e.createElementBlock("span",On,e.toDisplayString(r.protocolHint),1)):e.createCommentVNode("v-if",!0)]),e.createElementVNode("div",Un,[f[25]||(f[25]=e.createElementVNode("span",{class:"pm-field-label"},"上下文大小（Token）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[10]||(f[10]=i=>a.value.contextMaxTokens=i),type:"number",min:"0",step:"1000",placeholder:"0=不限制（模型级未配置时的默认窗口）"},null,512),[[e.vModelText,a.value.contextMaxTokens]])]),e.createElementVNode("div",zn,[f[26]||(f[26]=e.createElementVNode("span",{class:"pm-field-label"},"默认温度",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[11]||(f[11]=i=>a.value.temperature=i),placeholder:"如 0.3（空=不设；模型级可覆盖）"},null,512),[[e.vModelText,a.value.temperature]])]),e.createElementVNode("div",jn,[f[27]||(f[27]=e.createElementVNode("span",{class:"pm-field-label"},"默认思考档位",-1)),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":f[12]||(f[12]=i=>a.value.thinkingMode=i),title:"思考档位（OpenAI reasoning.effort 口径）：空=不下发；模型级可覆盖"},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(V.value,i=>(e.openBlock(),e.createElementBlock("option",{key:"t"+i,value:i},e.toDisplayString(i||"默认（不设）"),9,Hn))),128))],512),[[e.vModelSelect,a.value.thinkingMode]])]),e.createElementVNode("div",Wn,[f[28]||(f[28]=e.createElementVNode("span",{class:"pm-field-label"},"默认输出 Token（最大输出）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[13]||(f[13]=i=>a.value.maxTokens=i),type:"number",min:"0",step:"1024",placeholder:"0=不设（模型级可覆盖）"},null,512),[[e.vModelText,a.value.maxTokens]])]),e.createVNode(Te,{models:m.value,label:r.modelEditor.label||"可用模型（回车或逗号分隔添加；支持整段粘贴）",placeholder:r.modelEditor.placeholder||"输入模型名，回车添加…",onChange:F},null,8,["models","label","placeholder"]),e.createElementVNode("div",qn,[f[29]||(f[29]=e.createElementVNode("div",{class:"pm-params-title"},"模型参数（每模型独立配置；对话里也可临时切换思考档位）",-1)),m.value.length?(e.openBlock(),e.createElementBlock("div",Jn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(m.value,i=>(e.openBlock(),e.createElementBlock("div",{key:i,class:"pm-param-row"},[e.createElementVNode("span",{class:"pm-param-model",title:i},e.toDisplayString(i),9,Kn),e.createCommentVNode(" ★ 2026-08-21 schema 驱动：按 modelParamFields 动态渲染（checkbox/select/number/text） "),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.modelParamFields,d=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:d.name},[d.type==="checkbox"?(e.openBlock(),e.createElementBlock("label",{key:0,class:"pm-param-check",title:d.hint||d.label},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":g=>c.value[i][d.name]=g},null,8,_n),[[e.vModelCheckbox,c.value[i][d.name]]]),e.createTextVNode(" "+e.toDisplayString(d.label),1)],8,Zn)):d.type==="select"?e.withDirectives((e.openBlock(),e.createElementBlock("select",{key:1,"onUpdate:modelValue":g=>c.value[i][d.name]=g,title:d.hint||d.label},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(d.options||[],g=>(e.openBlock(),e.createElementBlock("option",{key:"o"+g,value:g},e.toDisplayString(g===""?d.label+"默认":g),9,Xn))),128))],8,Qn)),[[e.vModelSelect,c.value[i][d.name]]]):d.type==="number"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:2,"onUpdate:modelValue":g=>c.value[i][d.name]=g,type:"number",min:d.min??0,step:d.step??1,placeholder:d.label,title:d.hint||d.label},null,8,Yn)),[[e.vModelText,c.value[i][d.name],void 0,{number:!0}]]):e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:3,"onUpdate:modelValue":g=>c.value[i][d.name]=g,type:"text",placeholder:d.label,title:d.hint||d.label},null,8,vn)),[[e.vModelText,c.value[i][d.name]]])],64))),128))]))),128))])):(e.openBlock(),e.createElementBlock("div",et,"添加模型后，可逐模型配置参数（温度/思考/输出/上下文/多模态…）"))]),e.createElementVNode("div",nt,[e.createElementVNode("button",{class:"pm-btn pm-primary",disabled:h.value,onClick:j},e.toDisplayString(h.value?"保存中…":"保存服务商"),9,tt),e.createElementVNode("button",{class:"pm-btn",onClick:U},"取消")])])):(e.openBlock(),e.createElementBlock("div",rt,[e.createElementVNode("div",ot,[e.createElementVNode("span",{class:"pm-name",title:E.name},e.toDisplayString(E.name),9,lt),e.createElementVNode("div",at,[e.createElementVNode("button",{class:"pm-btn pm-small",onClick:i=>L(E)},"编辑",8,st),e.createElementVNode("button",{class:"pm-btn pm-small pm-danger",onClick:i=>P(E)},"删除",8,it)])]),e.createElementVNode("div",{class:"pm-url",title:E.baseURL},e.toDisplayString(E.baseURL||"未配置 API URL"),9,ct),E.protocol?(e.openBlock(),e.createElementBlock("div",{key:0,class:"pm-protocol",title:r.protocolHint},"协议 "+e.toDisplayString(E.protocol),9,dt)):e.createCommentVNode("v-if",!0),e.createElementVNode("div",pt,e.toDisplayString(E.contextMaxTokens>0?"上下文 "+(E.contextMaxTokens/1e3).toFixed(0)+"K Token":"上下文 未限制"),1),e.createElementVNode("div",mt,[E.models.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("span",gt,"（未配置模型）")),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(E.models,i=>(e.openBlock(),e.createElementBlock("span",{key:i,class:"pm-tag"},e.toDisplayString(i),1))),128))]),S(E.name)?(e.openBlock(),e.createElementBlock("div",kt,e.toDisplayString(S(E.name)),1)):e.createCommentVNode("v-if",!0)]))],64))),128))])):s.value!=="__new__"?(e.openBlock(),e.createElementBlock("div",ht,"暂无服务商，点「+ 新增服务商」添加")):e.createCommentVNode("v-if",!0),u.value?(e.openBlock(),e.createElementBlock("div",ft,e.toDisplayString(u.value),1)):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-2eaed0aa"]]),yt={class:"pm-manager"},bt={class:"mgm-toolbar"},Et={class:"mgm-count"},xt={key:0,class:"mgm-edit"},Vt={class:"mgm-edit-title"},Nt={class:"mgm-field"},wt={class:"mgm-field-label"},Bt={key:0,class:"mgm-required"},Tt=["onUpdate:modelValue","onChange"],St=["value"],Ct=["onUpdate:modelValue","placeholder"],It=["onUpdate:modelValue","placeholder"],Pt={key:3,class:"mgm-field-hint"},At={class:"mgm-edit-actions"},Mt=["disabled"],$t={key:1,class:"mgm-cards"},Dt={class:"mgm-card-head"},Rt=["title"],Lt={key:0,class:"pm-active-badge"},Ft={class:"mgm-ops"},Gt=["disabled","onClick"],Ot=["onClick"],Ut=["onClick"],zt={class:"pm-preview"},jt={class:"pm-snap-row"},Ht={class:"pm-snap-row"},Wt={key:2,class:"mgm-empty"},qt={key:3,class:"mgm-error"},Jt=z({__name:"PresetManager",props:{presetFields:{type:Array,default:()=>[]}},emits:["saved"],setup(r,{expose:t,emit:n}){const o=r,l=n,s=e.ref({}),a=e.computed(()=>Object.keys(s.value||{})),m=e.ref(""),c=e.ref(!1),u=e.ref(""),h=e.ref(!1),w=e.ref(""),V=e.ref(""),T=e.ref(null),y=e.computed(()=>T.value&&T.value.providers||[]),B=e.ref({name:"",provider:"",baseURL:"",apiKey:""});function I(i={}){const d={name:"",provider:"",baseURL:"",apiKey:""};for(const g of o.presetFields)g.name in d||(d[g.name]="");return Object.assign(d,i)}function L(i){V.value=i,setTimeout(()=>{V.value===i&&(V.value="")},4e3)}async function F(){try{const[i,d,g]=await Promise.all([A.getAiPresets().catch(()=>({presets:{}})),A.apiGet("/settings").catch(()=>({settings:{}})),A.getModels().catch(()=>null)]);s.value=i&&i.presets||{},m.value=d&&d.settings&&d.settings.preset||"",T.value=g}catch(i){L("加载失败: "+(i.message||i))}}function U(i){const d=T.value||{};return{baseURL:d.providerBaseURLs&&d.providerBaseURLs[i]||"",models:d.models&&d.models[i]||[]}}function H(){const i=window&&window.__PAIRCODE_CORE&&window.__PAIRCODE_CORE.uiState&&window.__PAIRCODE_CORE.uiState.state&&window.__PAIRCODE_CORE.uiState.state.settings||{};let d={};i.preset&&s.value&&s.value[i.preset]&&(d=s.value[i.preset]);const g=d.provider||y.value[0]||"",k=U(g);B.value=I({provider:g,baseURL:d.baseURL||k.baseURL||"",apiKey:d.apiKey||""}),u.value="",c.value=!0}function j(i){const d=s.value&&s.value[i]||{};B.value=I({name:i,provider:d.provider||"",baseURL:d.baseURL||"",apiKey:d.apiKey||""}),u.value=i,c.value=!0}function G(){c.value=!1,u.value=""}function P(i){return i.source==="providers"?y.value||[]:i.options||[]}function S(i){if(i.name==="provider"&&B.value.provider){const d=U(B.value.provider);B.value.baseURL=d.baseURL||""}}e.watch(()=>B.value.provider,(i,d)=>{!c.value||d===""||i!==d&&S({name:"provider"})});async function N(){const i=B.value.name.trim();if(!i){L("请输入配置名称");return}const d=B.value.provider||"";if(!d){L("请选择服务商");return}for(const k of o.presetFields)if(k.required&&!B.value[k.name]){L("请填写"+k.label);return}const g=U(d);B.value.baseURL||(B.value.baseURL=g.baseURL||""),h.value=!0,V.value="";try{const k={provider:d,baseURL:B.value.baseURL,apiKey:B.value.apiKey};for(const p of o.presetFields)p.name!=="provider"&&p.name!=="apiKey"&&(k[p.name]=B.value[p.name]);if(u.value&&u.value!==i){const p={...s.value||{}};p[i]=k,delete p[u.value];const x=await A.saveAiPresets(p);if(!(x&&x.ok)){L(x&&x.error||"保存失败");return}s.value=p,m.value===u.value&&(m.value=i,await A.apiPut("/settings",{settings:{preset:i},pluginSettings:{}}).catch(()=>{}))}else{const p=await A.saveAiPreset("save",i,k);if(!(p&&p.ok)){L(p&&p.error||"保存失败");return}s.value=p.presets||s.value}G(),l("saved")}catch(k){L("保存失败: "+(k.message||k))}finally{h.value=!1}}async function f(i){w.value=i,V.value="";try{const d=await A.saveAiPreset("apply",i);d&&d.ok?(m.value=i,l("saved")):L(d&&d.error||"应用失败")}catch(d){L("应用失败: "+(d.message||d))}finally{w.value=""}}async function E(i){if(confirm("删除配置「"+i+"」？")){V.value="";try{const d=await A.saveAiPreset("delete",i);d&&d.ok?(s.value=d.presets||s.value,m.value===i&&(m.value=""),l("saved")):L(d&&d.error||"删除失败")}catch(d){L("删除失败: "+(d.message||d))}}}return e.onMounted(F),t({load:F}),(i,d)=>(e.openBlock(),e.createElementBlock("div",yt,[e.createCommentVNode(" 工具栏 "),e.createElementVNode("div",bt,[e.createElementVNode("span",Et,e.toDisplayString(a.value.length)+" 个配置",1),e.createElementVNode("button",{class:"mgm-btn mgm-primary",onClick:H,title:"添加一条 AI 配置（服务商 + API Key）"},"＋ 添加新配置")]),e.createCommentVNode(" 添加 / 编辑表单（点击添加/编辑才弹出） "),c.value?(e.openBlock(),e.createElementBlock("div",xt,[e.createElementVNode("div",Vt,e.toDisplayString(u.value?"编辑配置："+u.value:"添加新配置"),1),e.createElementVNode("div",Nt,[d[1]||(d[1]=e.createElementVNode("span",{class:"mgm-field-label"},"配置名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":d[0]||(d[0]=g=>B.value.name=g),type:"text",placeholder:"如：主力 / 写作备用…",onKeydown:e.withKeys(N,["enter"])},null,544),[[e.vModelText,B.value.name]])]),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.presetFields,g=>(e.openBlock(),e.createElementBlock("div",{key:g.name,class:"mgm-field"},[e.createElementVNode("span",wt,[e.createTextVNode(e.toDisplayString(g.label),1),g.required?(e.openBlock(),e.createElementBlock("span",Bt,"*")):e.createCommentVNode("v-if",!0)]),e.createCommentVNode(" select 类型（服务商选择） "),g.type==="select"?e.withDirectives((e.openBlock(),e.createElementBlock("select",{key:0,"onUpdate:modelValue":k=>B.value[g.name]=k,class:"mgm-select",onChange:k=>S(g)},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(P(g),k=>(e.openBlock(),e.createElementBlock("option",{key:k,value:k},e.toDisplayString(k),9,St))),128))],40,Tt)),[[e.vModelSelect,B.value[g.name]]]):g.type==="password"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" password 类型（API Key） "),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":k=>B.value[g.name]=k,type:"password",placeholder:g.placeholder||""},null,8,Ct),[[e.vModelText,B.value[g.name]]])],2112)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" text 兜底 "),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":k=>B.value[g.name]=k,type:"text",placeholder:g.placeholder||""},null,8,It),[[e.vModelText,B.value[g.name]]])],2112)),g.hint?(e.openBlock(),e.createElementBlock("span",Pt,e.toDisplayString(g.hint),1)):e.createCommentVNode("v-if",!0)]))),128)),e.createElementVNode("div",At,[e.createElementVNode("button",{class:"mgm-btn mgm-primary",disabled:h.value,onClick:N},e.toDisplayString(h.value?"保存中…":"保存配置"),9,Mt),e.createElementVNode("button",{class:"mgm-btn",onClick:G},"取消")])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 配置卡片列表（主视图） "),a.value.length?(e.openBlock(),e.createElementBlock("div",$t,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,g=>(e.openBlock(),e.createElementBlock("div",{key:g,class:e.normalizeClass(["mgm-card",{"pm-active":g===m.value}])},[e.createElementVNode("div",Dt,[e.createElementVNode("span",{class:"mgm-name",title:g},[e.createTextVNode(e.toDisplayString(g),1),g===m.value?(e.openBlock(),e.createElementBlock("span",Lt,"使用中")):e.createCommentVNode("v-if",!0)],8,Rt),e.createElementVNode("div",Ft,[e.createElementVNode("button",{class:"mgm-btn mgm-small",disabled:w.value===g,onClick:k=>f(g)},e.toDisplayString(w.value===g?"应用中…":"应用"),9,Gt),e.createElementVNode("button",{class:"mgm-btn mgm-small",onClick:k=>j(g)},"编辑",8,Ot),e.createElementVNode("button",{class:"mgm-btn mgm-small mgm-danger",onClick:k=>E(g)},"删除",8,Ut)])]),e.createElementVNode("div",zt,[e.createElementVNode("div",jt,[d[2]||(d[2]=e.createElementVNode("span",null,"服务商",-1)),e.createElementVNode("b",null,e.toDisplayString((s.value[g]||{}).provider||"—"),1)]),e.createElementVNode("div",Ht,[d[3]||(d[3]=e.createElementVNode("span",null,"API Key",-1)),e.createElementVNode("b",null,e.toDisplayString((s.value[g]||{}).apiKey?"已配置":"未配置"),1)])])],2))),128))])):c.value?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("div",Wt,[e.createElementVNode("div",{class:"mgm-empty-box"},[d[4]||(d[4]=e.createElementVNode("div",{class:"mgm-empty-title"},"还没有 AI 配置",-1)),d[5]||(d[5]=e.createElementVNode("div",{class:"mgm-empty-sub"},"添加一条服务商 + API Key，保存后即可在对话面板中选模型。",-1)),e.createElementVNode("button",{class:"mgm-btn mgm-primary",onClick:H},"＋ 添加新配置")])])),V.value?(e.openBlock(),e.createElementBlock("div",qt,e.toDisplayString(V.value),1)):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-3e18e523"]]),Kt={class:"modal-content"},Zt={class:"modal-head"},_t={class:"mh-text"},Qt={class:"mh-title"},Xt={class:"mh-sub"},Yt={class:"modal-body"},vt={key:0,class:"settings-tabs"},er={key:0,class:"settings-tabs-filter-wrap"},nr=["onClick"],tr={class:"st-label"},rr={key:1,class:"settings-tabs-none"},or={class:"settings-content"},lr={key:0},ar={key:0,class:"group-title"},sr=["title"],ir=["title"],cr=["onUpdate:modelValue"],dr=["title"],pr={class:"field-control"},mr=["type","onUpdate:modelValue","placeholder"],gr=["onUpdate:modelValue","min","max","step"],kr=["onUpdate:modelValue","onChange"],hr=["value"],fr={class:"theme-gallery"},ur=["title","onClick"],yr={class:"tg-preview"},br={class:"tg-text"},Er={class:"tg-label"},xr={key:0,class:"tg-desc"},Vr={class:"color-dots"},Nr=["title","onClick"],wr=["onUpdate:modelValue","placeholder"],Br={class:"slider-row"},Tr=["onUpdate:modelValue","min","max","step"],Sr={class:"slider-val"},Cr={class:"color-row"},Ir=["onUpdate:modelValue"],Pr={class:"color-code"},Ar=["value","onInput","placeholder"],Mr=["placeholder"],$r=["onUpdate:modelValue"],Dr={key:0,class:"setting-hint"},Rr={key:0,class:"settings-empty"},Lr=z({__name:"SettingsModal",emits:["close"],setup(r,{emit:t}){const n=t,o=e.ref(""),l=e.ref(""),s=e.computed(()=>{const i=(b.state.pluginSchemas||[]).map(d=>({key:d.key,title:d.title||d.key,groups:m(d.fields||[])}));return i.length&&!o.value&&(o.value=i[0].key),i}),a=e.computed(()=>{const i=l.value.trim().toLowerCase();if(!i)return s.value;const d=s.value.find(k=>k.key===o.value),g=s.value.filter(k=>(k.title||k.key).toLowerCase().includes(i));return d&&!g.includes(d)&&g.unshift(d),g});function m(i){const d=[],g={};for(const k of i){const p=k.group||"";g[p]||(g[p]=[],d.push({title:p,fields:g[p]})),g[p].push(k)}return d}const c=e.ref(null);let u="";async function h(){try{c.value=await A.getModels()}catch{c.value=null}}function w(i){return i?(c.value&&c.value.models||{})[i]||[]:[]}function V(i,d){var g,k,p;if(d.optionsSource==="models"){const x=(g=y[i])==null?void 0:g[d.name],$=w((k=y.ai)==null?void 0:k.provider);return x&&!$.includes(x)?[...$,x]:$}if(d.optionsSource==="providers"){const x=c.value&&c.value.providers||[];if(x.length){const $=(p=y[i])==null?void 0:p[d.name];return $&&!x.includes($)?[...x,$]:x}return d.options||[]}return d.options||[]}function T(i){if(!y.ai)return;const d=y.ai,g=i.linkFields||(i.linkField?[i.linkField]:[]);if(!g.length)return;const k=c.value||{},p=k.providerBaseURLs||{},x=k.providerKeys||{},$=d.provider,_=p[u];for(const pe of g)if(pe==="apiKey")d[pe]=x[$]||"";else{const Ve=d[pe];(Ve===void 0||Ve===""||_&&Ve===_)&&(d[pe]=p[$]||"")}u=$}const y=e.reactive({}),B=e.ref("");function I(i){return Array.isArray(i.swatches)&&i.swatches.length?i.swatches:(i.options||[]).map(d=>({value:d,label:d,colors:[]}))}function L(i){switch(i){case"checkbox":return!1;case"number":return 0;case"tags":return[];default:return""}}function F(){for(const k of Object.keys(y))delete y[k];const i=b.state.settings||{};u=i.provider||"";const d=i.pluginSettings||{};for(const k of b.state.pluginSchemas||[]){y[k.key]={};for(const p of k.fields||[]){let x;if(!(p.type==="project"||p.type==="provider-manager"||p.type==="model-params-manager"||p.type==="preset-manager")){if(p.binding)x=i[p.binding]!==void 0?i[p.binding]:p.default;else{const $=d[k.key]||{};x=$[p.name]!==void 0?$[p.name]:p.default}x===void 0&&(x=L(p.type)),p.type==="checkbox"&&(x=!!x),p.type==="number"&&(x=typeof x=="number"?x:Number(x)||0),p.type==="tags"&&(x=Array.isArray(x)?x:[]),y[k.key][p.name]=x}}}const g=(b.state.pluginSchemas||[]).some(k=>(k.fields||[]).some(p=>p.type==="project"));B.value="",g&&j()}function U(i,d){var k;const g=(k=y[i])==null?void 0:k[d.name];return Array.isArray(g)?g.join(", "):g||""}function H(i,d,g){y[i][d.name]=g.target.value.split(",").map(k=>k.trim()).filter(Boolean)}async function j(){try{const i=await A.getInstructions("project");B.value=i.content||""}catch{}}function G(){var i;F(),(i=b.state.settings)!=null&&i.theme&&b.applyTheme(b.state.settings.theme)}const P=()=>{G()},S=e.ref(0);async function N(){await h(),S.value++}async function f(){try{const i=await A.apiGet("/settings");i&&i.settings&&(b.state.settings=i.settings,await h(),G())}catch{}}const E=async()=>{try{let i={};try{const p=await A.apiGet("/settings");i=p&&p.settings||{}}catch{}const d={...i},g={...i.pluginSettings||{}};let k=!1;for(const p of b.state.pluginSchemas||[]){const x=y[p.key]||{};for(const $ of p.fields||[]){if($.type==="project"){await A.saveInstructions("project",B.value);continue}if($.type==="provider-manager"||$.type==="model-params-manager"||$.type==="preset-manager")continue;const _=x[$.name];$.binding?($.name==="theme"&&_!==d[$.binding]&&(k=!0),d[$.binding]=_):(g[p.key]||(g[p.key]={}),g[p.key][$.name]=_)}}await A.apiPut("/settings",{settings:d,pluginSettings:g}),b.state.settings=d,k&&b.applyTheme(d.theme),window.$toast("设置已保存","success"),n("close")}catch(i){window.$toast("保存失败: "+i.message,"error")}};return F(),e.onMounted(async()=>{await h(),G()}),(i,d)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:d[3]||(d[3]=e.withModifiers(g=>i.$emit("close"),["self"]))},[e.createElementVNode("div",Kt,[e.createCommentVNode(` ★ 2026-09-24 对齐设计稿 appearance-theme th782（头部 = 标题 + 副标题两行）：
           副标题显示**当前分类名**（真实数据，非装饰文案）。 `),e.createElementVNode("div",Zt,[e.createElementVNode("div",_t,[e.createElementVNode("div",Qt,[e.createVNode(R,{name:"settings",size:16}),d[4]||(d[4]=e.createTextVNode(" 设置",-1))]),e.createElementVNode("div",Xt,e.toDisplayString((s.value.find(g=>g.key===o.value)||{}).title||"所有分类"),1)]),e.createElementVNode("button",{class:"modal-close",onClick:d[0]||(d[0]=g=>i.$emit("close"))},"×")]),e.createElementVNode("div",Yt,[e.createCommentVNode(" ═══ 纯 schema 驱动：所有配置 tab 由插件 ctx.registerSettings 注册 ═══ "),s.value.length?(e.openBlock(),e.createElementBlock("div",vt,[e.createCommentVNode(" 设计稿 th783：分组标题「设置分类」 "),d[5]||(d[5]=e.createElementVNode("div",{class:"settings-tabs-title"},"设置分类",-1)),s.value.length>6?(e.openBlock(),e.createElementBlock("div",er,[e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":d[1]||(d[1]=g=>l.value=g),class:"settings-tabs-filter",type:"text",placeholder:"筛选设置…"},null,512),[[e.vModelText,l.value]])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 设计稿 th755-th773：项 32 高 / radius 8 / 前置图标（选中态=check） "),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,g=>(e.openBlock(),e.createElementBlock("button",{key:g.key,class:e.normalizeClass(["settings-tab",{active:o.value===g.key}]),onClick:k=>o.value=g.key},[e.createVNode(R,{name:o.value===g.key?"check":"chevron-right",size:14},null,8,["name"]),e.createElementVNode("span",tr,e.toDisplayString(g.title),1)],10,nr))),128)),a.value.length===0?(e.openBlock(),e.createElementBlock("div",rr,"无匹配设置")):e.createCommentVNode("v-if",!0)])):e.createCommentVNode("v-if",!0),e.createElementVNode("div",or,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(s.value,g=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:g.key},[o.value===g.key?(e.openBlock(),e.createElementBlock("div",lr,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(g.groups,k=>(e.openBlock(),e.createElementBlock("div",{key:k.title||"__main",class:"setting-group"},[k.title?(e.openBlock(),e.createElementBlock("div",ar,e.toDisplayString(k.title),1)):e.createCommentVNode("v-if",!0),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(k.fields,p=>(e.openBlock(),e.createElementBlock("div",{key:p.name,class:e.normalizeClass(["setting-row",{"row-toggle":p.type==="checkbox"}])},[e.createCommentVNode(" checkbox：label 与开关同行 "),p.type==="checkbox"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:0},[e.createElementVNode("label",{class:"field-label",title:p.hint},e.toDisplayString(p.label),9,sr),e.createElementVNode("label",{class:"pp-switch",title:p.hint},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":x=>y[g.key][p.name]=x},null,8,cr),[[e.vModelCheckbox,y[g.key][p.name]]]),d[6]||(d[6]=e.createElementVNode("span",{class:"pp-switch-track"},null,-1))],8,ir)],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" 其他类型：label 在上、控件在下、说明文字在控件下方（不挤占输入区） "),e.createElementVNode("label",{class:"field-label",title:p.hint},e.toDisplayString(p.label),9,dr),e.createElementVNode("div",pr,[e.createCommentVNode(" text / password "),p.type==="text"||p.type==="password"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:0,class:"field-input",type:p.type==="password"?"password":"text","onUpdate:modelValue":x=>y[g.key][p.name]=x,placeholder:p.placeholder},null,8,mr)),[[e.vModelDynamic,y[g.key][p.name]]]):p.type==="number"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" number "),e.withDirectives(e.createElementVNode("input",{class:"field-input",type:"number","onUpdate:modelValue":x=>y[g.key][p.name]=x,min:p.min,max:p.max,step:p.step},null,8,gr),[[e.vModelText,y[g.key][p.name],void 0,{number:!0}]])],2112)):p.type==="select"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" select（optionsSource 驱动动态数据源：models=按服务商模型列表 / providers=服务商列表） "),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":x=>y[g.key][p.name]=x,class:"field-select",onChange:x=>T(p)},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(V(g.key,p),x=>(e.openBlock(),e.createElementBlock("option",{key:x,value:x},e.toDisplayString(x),9,hr))),128))],40,kr),[[e.vModelSelect,y[g.key][p.name]]])],2112)):p.type==="theme-gallery"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createCommentVNode(` theme-gallery（主题画廊：缩略色卡选择）
                           ★ 值仍是 string 主题 id，binding 直连 AppSettings.theme —— 与 select 同数据模型，
                             仅换一种更直观的呈现（色卡 + 明暗标识）。无联动字段故不调 onSelectChange。 `),e.createElementVNode("div",fr,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(I(p),x=>(e.openBlock(),e.createElementBlock("button",{key:x.value,type:"button",class:e.normalizeClass(["tg-item",{active:y[g.key][p.name]===x.value}]),title:x.label,onClick:$=>y[g.key][p.name]=x.value},[e.createCommentVNode(" 设计稿 th584：预览区 48 高（主题代表色横向平铺） "),e.createElementVNode("span",yr,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(x.colors,($,_)=>(e.openBlock(),e.createElementBlock("i",{key:_,style:e.normalizeStyle({background:$})},null,4))),128))]),e.createElementVNode("span",br,[e.createElementVNode("span",Er,e.toDisplayString(x.label),1),e.createCommentVNode(` ★ 2026-09-25：定位文案（设计稿每张卡 = 预览 + 名称 + 一句定位，
                                 如「默认暗色，久看不累」）。由插件 swatches[].desc 提供。 `),x.desc?(e.openBlock(),e.createElementBlock("span",xr,e.toDisplayString(x.desc),1)):e.createCommentVNode("v-if",!0)])],10,ur))),128))])],2112)):p.type==="color-dots"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:4},[e.createCommentVNode(` color-dots（色点选择：强调色覆盖等 —— 圆点 + 选中描边，悬停看名称）
                           ★ 值仍是 string，与 select / theme-gallery 同数据模型，仅换呈现形态。 `),e.createElementVNode("div",Vr,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(I(p),x=>(e.openBlock(),e.createElementBlock("button",{key:x.value,type:"button",class:e.normalizeClass(["cd-item",{active:String(y[g.key][p.name]||"")===x.value}]),title:x.label,onClick:$=>y[g.key][p.name]=x.value},[e.createElementVNode("i",{class:"cd-dot",style:e.normalizeStyle({background:x.colors&&x.colors[0]||"transparent"})},null,4)],10,Nr))),128))])],2112)):p.type==="textarea"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:5},[e.createCommentVNode(" textarea "),e.withDirectives(e.createElementVNode("textarea",{"onUpdate:modelValue":x=>y[g.key][p.name]=x,class:"field-textarea",rows:"4",placeholder:p.placeholder},null,8,wr),[[e.vModelText,y[g.key][p.name]]])],2112)):p.type==="slider"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:6},[e.createCommentVNode(" slider "),e.createElementVNode("div",Br,[e.withDirectives(e.createElementVNode("input",{type:"range","onUpdate:modelValue":x=>y[g.key][p.name]=x,min:p.min!=null?p.min:0,max:p.max!=null?p.max:100,step:p.step||1},null,8,Tr),[[e.vModelText,y[g.key][p.name],void 0,{number:!0}]]),e.createElementVNode("span",Sr,e.toDisplayString(y[g.key][p.name]),1)])],2112)):p.type==="color"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:7},[e.createCommentVNode(" color "),e.createElementVNode("div",Cr,[e.withDirectives(e.createElementVNode("input",{type:"color","onUpdate:modelValue":x=>y[g.key][p.name]=x},null,8,Ir),[[e.vModelText,y[g.key][p.name]]]),e.createElementVNode("code",Pr,e.toDisplayString(y[g.key][p.name]),1)])],2112)):p.type==="tags"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:8},[e.createCommentVNode(" tags（逗号分隔数组） "),e.createElementVNode("input",{type:"text",class:"field-input",value:U(g.key,p),onInput:x=>H(g.key,p,x),placeholder:p.placeholder||"逗号分隔"},null,40,Ar)],2112)):p.type==="project"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:9},[e.createCommentVNode(" project（平台特殊：项目级指令，经 /api/instructions 读写） "),e.withDirectives(e.createElementVNode("textarea",{"onUpdate:modelValue":d[2]||(d[2]=x=>B.value=x),class:"field-textarea",rows:"4",placeholder:p.placeholder},null,8,Mr),[[e.vModelText,B.value]])],2112)):p.type==="provider-manager"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:10},[e.createCommentVNode(" provider-manager（服务商维护面板：CRUD /api/models，独立保存，不参与普通表单） "),e.createVNode(ut,{"model-param-fields":p.modelParamFields||[],"model-editor":p.modelEditor||{},"protocol-label":p.protocolLabel||"LLM 协议","protocol-options":p.protocolOptions||[],"protocol-hint":p.protocolHint||"",onSaved:N},null,8,["model-param-fields","model-editor","protocol-label","protocol-options","protocol-hint"])],2112)):p.type==="preset-manager"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:11},[e.createCommentVNode(" preset-manager（AI 配置预设面板：CRUD /api/ai-presets，独立保存，不参与普通表单） "),(e.openBlock(),e.createBlock(Jt,{key:S.value,"preset-fields":p.presetFields||[],onSaved:f},null,8,["preset-fields"]))],2112)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:12},[e.createCommentVNode(" 兜底 text "),e.withDirectives(e.createElementVNode("input",{class:"field-input",type:"text","onUpdate:modelValue":x=>y[g.key][p.name]=x},null,8,$r),[[e.vModelText,y[g.key][p.name]]])],2112))]),p.hint?(e.openBlock(),e.createElementBlock("span",Dr,e.toDisplayString(p.hint),1)):e.createCommentVNode("v-if",!0)],64))],2))),128))]))),128))])):e.createCommentVNode("v-if",!0)],64))),128)),s.value.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("div",Rr,"暂无配置项（等待插件注册…）"))])]),e.createCommentVNode(` ★ 2026-09-24 对齐设计稿 th790（底 48：说明 + 信息）：左侧补说明文本，
           与「撤销 / 保存设置」按钮同行（设计稿未画按钮，但 schema 驱动需显式保存）。 `),e.createElementVNode("div",{class:"modal-footer"},[d[7]||(d[7]=e.createElementVNode("span",{class:"mf-note"},"配置项由插件注册，修改后点「保存设置」生效",-1)),e.createElementVNode("button",{class:"btn-secondary",onClick:P},"撤销"),e.createElementVNode("button",{class:"btn-primary",onClick:E},"保存设置")])])]))}},[["__scopeId","data-v-062893bb"]]),Fr={class:"modal-content sys-modal"},Gr={class:"modal-header"},Or={class:"modal-body"},Ur={key:0,class:"loading"},zr={key:1,class:"sys-info"},jr={class:"info-row"},Hr={class:"info-row"},Wr={class:"info-row"},qr={class:"info-row"},Jr={class:"info-row"},Kr={class:"info-row"},Zr={class:"modal-footer"},_r=z({__name:"SystemModal",emits:["close"],setup(r,{emit:t}){const n=e.ref(!0),o=e.ref({});return e.onMounted(async()=>{try{o.value=await A.apiGet("/system/info")}catch{}n.value=!1}),(l,s)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:s[2]||(s[2]=e.withModifiers(a=>l.$emit("close"),["self"]))},[e.createElementVNode("div",Fr,[e.createElementVNode("div",Gr,[s[3]||(s[3]=e.createElementVNode("h2",null,"ℹ 系统信息",-1)),e.createElementVNode("button",{class:"modal-close",onClick:s[0]||(s[0]=a=>l.$emit("close"))},"×")]),e.createElementVNode("div",Or,[n.value?(e.openBlock(),e.createElementBlock("div",Ur,"加载中...")):(e.openBlock(),e.createElementBlock("div",zr,[e.createElementVNode("div",jr,[s[4]||(s[4]=e.createElementVNode("label",null,"主机名",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.hostname),1)]),e.createElementVNode("div",Hr,[s[5]||(s[5]=e.createElementVNode("label",null,"当前目录",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.cwd),1)]),e.createElementVNode("div",Wr,[s[6]||(s[6]=e.createElementVNode("label",null,"操作系统",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.os),1)]),e.createElementVNode("div",qr,[s[7]||(s[7]=e.createElementVNode("label",null,"Go 版本",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.goos),1)]),e.createElementVNode("div",Jr,[s[8]||(s[8]=e.createElementVNode("label",null,"工作区",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.workspace),1)]),e.createElementVNode("div",Kr,[s[9]||(s[9]=e.createElementVNode("label",null,"文件夹",-1)),e.createElementVNode("span",null,e.toDisplayString((o.value.folders||[]).join(", ")),1)])]))]),e.createElementVNode("div",Zr,[e.createElementVNode("button",{class:"btn-secondary",onClick:s[1]||(s[1]=a=>l.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-bb4d0117"]]),Qr={class:"modal-content source-modal"},Xr={class:"modal-header"},Yr={class:"modal-footer"},vr=z({__name:"SourceModal",emits:["close"],setup(r){return(t,n)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:n[2]||(n[2]=e.withModifiers(o=>t.$emit("close"),["self"]))},[e.createElementVNode("div",Qr,[e.createElementVNode("div",Xr,[n[3]||(n[3]=e.createElementVNode("h2",null,"⎔ 源代码管理",-1)),e.createElementVNode("button",{class:"modal-close",onClick:n[0]||(n[0]=o=>t.$emit("close"))},"×")]),n[4]||(n[4]=e.createElementVNode("div",{class:"modal-body"},[e.createElementVNode("p",{style:{color:"var(--text-muted)","text-align":"center","margin-top":"40px"}},[e.createTextVNode(" Git 集成开发中"),e.createElementVNode("br"),e.createElementVNode("br"),e.createTextVNode(" 功能规划："),e.createElementVNode("br"),e.createTextVNode(" · Git 状态查看"),e.createElementVNode("br"),e.createTextVNode(" · 暂存/提交/推送"),e.createElementVNode("br"),e.createTextVNode(" · 分支管理"),e.createElementVNode("br"),e.createTextVNode(" · Diff 对比 ")])],-1)),e.createElementVNode("div",Yr,[e.createElementVNode("button",{class:"btn-secondary",onClick:n[1]||(n[1]=o=>t.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-bc520229"]]),eo=`# 功能介绍

PairCode IDE 是一款纯 Web 端的 AI 辅助编程开发环境。你只需打开浏览器，在对话面板中用自然语言描述需求，AI 就能理解你的意图，直接生成代码、修改文件、执行命令、管理版本——把 IDE 从工具变为你的编程搭档。

---

## AI 对话编程

**用自然语言驱动整个开发流程，就像跟资深开发者聊天一样跟 AI 交流。**

在右侧对话面板中，你只需用自然语言描述需求，AI 就会理解你的意图并自动完成相应操作。无论是"帮我写一个 REST API"还是"把这个函数改成异步的"，AI 都能立刻执行。

- **流式实时输出** — AI 的思考过程和操作结果实时显示，你始终能看清它在想什么、做什么
- **透明可追溯** — 每一步操作都有详细上下文，不是黑盒
- **随时干预** — 如果 AI 方向跑偏，可以随时给出反馈，AI 会立即调整

---

## 自主编程模式

**AI 独立完成复杂的多步骤任务，你只需做最关键的决定。**

开启自主模式后，AI 能自己分析项目结构、扫描代码问题、制定修复计划并逐个执行。你可以在关键节点审核确认，其他步骤 AI 自动完成。执行进度实时可见，你可以随时暂停、中止或补充指令。

**Agent 核心采用双层循环（turn/step）架构**：
- **turn / step 双层边界** — 每次工具执行都有独立的 step 事件（开始/结束/摘要），每轮用户交互是 turn，进度颗粒度清晰可追溯
- **inbox 双队列** — 任务转向（next-step）与后续追问（next-turn）分队列消费，多轮交互不粘连
- **消息组装与落盘对齐** — agentloop 编号与消息序列严格一致，历史恢复与实时流状态吻合
- **历史注入精简** — 去掉冗余前缀标注与时间戳，系统提示内置多轮规则，长对话上下文更干净
- **段预算双闸门 + 自动续跑** — 单个执行段同时受「步数预算」与「工具调用预算」约束（默认各 120），任一达上限即结束本段并自动开启下一段继续干活，长任务不会半途停下；续跑段数上限可配（默认 20 段）。三项都在「设置 → Agent」里实时调整，改完即生效
- **会话交接** — 续跑或重新提问时若历史已很长，自动整理成一份「会话交接·提交消息」（相关性判断 + 目标 / 已完成 / 当前状态 / 关键决策 / 下一步），上下文不再无限膨胀；历史不长时则原样注入，不干扰缓存

---

## 智能代码编辑器

**内置浏览器端编辑器，让你在同一个窗口中完成所有编辑工作。**

- **语法高亮** — 支持 Go、TypeScript、Python、Rust、Java、Vue、HTML、CSS 等主流语言
- **代码折叠** — 折叠函数和代码块，聚焦关键逻辑
- **多标签页** — 同时打开编辑多个文件，标签栏快捷切换
- **括号匹配与自动缩进** — 代码结构清晰可见
- **十六进制查看器** — 查看二进制文件的原始字节内容
- **图片预览** — 在编辑器中直接显示图片文件

---

## 文件管理

**完整的工作区文件管理能力，所有操作一目了然。**

- **目录树浏览** — 以树形结构展示项目目录，支持展开 / 折叠
- **文件操作** — 新建、编辑、保存、删除、重命名、移动文件
- **多文件夹工作区** — 同时管理多个目录，组合成一个统一的工作区
- **快速切换工作区** — 在最近使用的项目之间一键切换
- **文件搜索** — 按文件名快速定位
- **内容搜索** — 在整个工作区按关键词搜索代码内容

---

## Git 版本控制

**在对话中完成所有 Git 操作，告别记忆复杂命令。**

你只需用自然语言告诉 AI 你想做什么：
- "查看当前仓库状态"
- "暂存所有修改，提交信息为'修复登录校验'"
- "创建一个名为 feature-search 的分支"
- "从远程拉取最新代码"

AI 会自动执行对应的 Git 操作并返回结果。你也可以通过 Git 面板查看文件变更的逐行对比。

---

## 内置终端

**浏览器中的终端，无需切换窗口。**

终端面板直接内嵌在 IDE 底部，打开即用。AI 也能自动使用终端执行命令、读取输出并分析结果。支持多标签页，方便同时运行不同任务。

---

## 帮助文档中心

**结构化的帮助文档体系，快速找到你需要的信息。**

帮助面板侧边栏按分类组织文档：

| 分类 | 包含文档 |
|------|---------|
| **文档中心** | 快速开始、功能介绍、API 文档、工具文档、快捷键、常见问题 |
| **其他** | 更新日志 |

- **按分类导航** — 文档归入"文档中心"分组，找什么一目了然
- **文档间跳转** — 关于面板与帮助面板之间可互相跳转
- **翻页浏览** — 文档底部支持上一页/下一页顺序阅读
- **搜索过滤** — 侧边栏搜索框可快速筛选文档

---

## API 二次开发支持

**完整的 HTTP REST API + WebSocket 协议文档，支持第三方基于本 IDE 进行二次开发。**

- **详细的请求/响应格式** — 每个 API 接口提供 JSON Schema 请求体、完整响应示例、字段说明和错误码
- **WebSocket 协议定义** — 完整的 AI 事件流协议文档（15+ 事件类型、数据结构、典型事件序列）
- **终端协议文档** — PTY WebSocket 的初始化流程、控制消息格式、白名单限制等
- **API 索引速查表** — 按功能分类列出所有 60+ API 端点，方便快速查找

所有 API 监听所有网络接口（0.0.0.0），本机与局域网内设备均可访问；请勿将端口暴露到公网。

---

## 代码知识图谱

**AI 能理解你的代码结构和调用关系，不仅仅是搜索文本。**

CodeGraph 将项目的代码结构构建成可查询的知识图谱，让 AI 理解函数之间的调用关系、类型的层次结构和文件的依赖网络。AI 可以准确找到某个函数的所有调用者、分析修改影响范围、查看完整的类型继承链。

**多项目独立建图** — 在多项目工作区中，每个项目独立构建知识图谱（主项目用共享库、非主项目用各自存储），跨项目切换不串数据，工具通过 project 参数精确路由到目标项目。

---

## 对话历史管理

**每次对话自动保存，随时回溯，不会丢失。**

- 对话自动持久化到本地磁盘，刷新页面不会丢失
- 左侧对话列表展示所有历史记录，支持继续之前的话题
- 不同工作区的对话自动隔离，各项目互不干扰
- 支持向前翻页加载更多历史消息
- **长对话自动交接** — 历史过长时按需生成交接摘要（见「自主编程模式」），既保留任务连续性，又避免上下文膨胀拖慢响应
- **运行统计条** — 对话面板显示本轮耗时 / 步数 / 工具调用次数 / 输出速度，运行中实时刷新、结束后定格；断开连接会自动重连并补齐该会话的任务与消息

---

## BUG 自动检测与修复

**AI 主动扫描代码问题并生成修复方案，反复验证直到全部通过。**

- 自动运行编译检查和测试，标记所有错误位置
- 分析错误根因，生成具体的修复方案
- 修复后再次验证，支持多轮迭代
- 修复前会展示改动内容，你可以审阅确认

---

## Skills / MCP / 工具集扩展

**通过扩展增强 AI 的能力，让 IDE 更贴合你的工作流。**

- **Skills（技能）** — 可复用的工作流程模板，AI 在对应场景中自动加载使用
- **MCP（模型上下文协议）** — 标准化的工具扩展协议，可为 AI 添加自定义能力（如查询内部数据库、调用第三方 API）
- **工具集（Toolset）** — 按项目需求组合的插件包，动态构建并固化到工作区，可导出/导入/发布市场
- **场景工具集（创造模式）** — 直接说 \`/创造 需求描述\`，AI 会先盘点现有能力、再组合出一个「场景」工具集（本质是工具面白名单），固化后可在对话面板一键切换，让 AI 只看到当前任务真正需要的工具
- **内置市场** — 一键浏览和安装社区贡献的扩展（技能 / MCP / 插件工具集三类）

---

## 插件化自定义工具

**通过 JS / TS 插件扩展 AI 的工具集，一切皆插件。**

PairCode IDE 的工具体系全部插件化——内置功能（文件 / 搜索 / 命令 / 网页 / 记忆 / 任务 / 代码图谱 / 办公文档 / 二进制分析等）以插件形态装配，你也可以编写自己的插件扩展能力：

- **JS / TS 插件** — 通过 \`cordis(op=define)\` 定义函数形态插件，支持 \`apply(ctx, config)\` 注入服务、timer 定时器、跨 goroutine 执行锁；TS 插件由内置编译器（esbuild 纯 Go）直接转译加载，无需 Node.js
- **插件管理单入口** — \`cordis\` 按 \`op\` 分派：\`define\` 定义 / \`inspect\` 查看工具归属与运行时诊断 / \`run\`·\`stop\` 装卸 / \`services\`·\`query\` 查宿主服务与协议签名 / \`undefine\` 撤销
- **沙箱防护** — VM 超时防护、schema 校验，插件异常不影响主进程

## 工具集生态

**按项目需求动态组合工具集，固化到工作区，可导出分享。**

- **动态构建** — 描述你的项目需求（如"Go 后端 + 前端调试"），AI 分析项目结构后自动组合所需工具并创建工具集插件
- **固化与重建** — 工具集固化到 \`.pair/toolsets/\`，随项目走；显式调用可更新重建
- **导出 / 导入 / 市场** — 工具集可导出为 JSON 分享，或发布到市场供他人一键安装（project/user 两种范围）
- **LLM 意图分析** — 分析项目目的时由 LLM 参与理解（语言无关，不固化任何语言模板），跨语言项目同样适用

---

## 项目知识库

**把项目架构、模块职责和设计决策沉淀成结构化知识库，AI 跨会话持续了解你的项目。**

- **树形分支组织** — 知识按 目标 / 架构 / 实现 / 关键点 / 设计思想 分类，深挖有细节、浏览有全貌
- **跨会话记忆** — AI 每次接手项目自动加载知识库导航，无需从零分析项目
- **团队共享** — 知识库存入项目 \`.pair/\` 目录，随项目版本控制，团队协作时信息不丢失
- **过期检测** — 自动验证知识条目引用的文件/目录是否存在，失效条目提示清理
- **AGENTS.md 分层** — 项目说明、环境配置、开发指南分层管理，.agents 路径兼容

## 记忆系统

**AI 能跨会话记住你的偏好和项目决策。**

AI 会记住你的编码偏好、经常使用的模式和做过的决策。下次打开 IDE 时，AI 会自动引用这些记忆，无需重复说明。记忆可搜索、可管理。

---

## 任务与规划管理

**复杂的多步骤开发任务有条不紊地执行。**

AI 会自动分解复杂任务为可追踪的子任务步骤，每步的执行状态和结果清晰可见。支持依赖关系管理，任务清单持久化，重启不会丢失。

---

## 主题与个性化

**按照你的喜好定制 IDE 外观。**

- **四套预设主题** — 暗色科技风、白色简约风、暖色温暖风、暗夜紫风格
- **即时切换** — 切换主题立即生效，无需刷新
- **统一字体方案** — 界面字体和代码字体分别配置

---

## 多模型支持

**灵活选择 AI 模型后端。**

支持 OpenAI、Claude 等主流 AI 服务商。可为"执行任务"和"制定规划"分别配置不同的模型。所有模型配置在设置面板中集中管理，支持自定义 API 地址。

---

## 安全设计

**你的代码和数据始终在你的控制之下。**

- **本地运行** — 所有操作在本地计算机执行，不经过第三方云端
- **路径隔离** — 文件操作限定在工作区目录范围内
- **审批机制** — 写文件和执行命令等敏感操作需你确认
- **监听地址** — IDE 服务监听所有接口（0.0.0.0），本机用 localhost 访问，局域网设备用本机 IP 访问

---

## 操作界面速览

| 区域 | 说明 |
|------|------|
| **标题栏** | 顶部菜单栏，提供帮助文档、设置等入口 |
| **活动栏** | 左侧图标栏，切换文件浏览、搜索、Git 等功能面板 |
| **侧栏** | 文件树、搜索面板、Git 面板等工具区域 |
| **主编辑区** | 代码编辑区域，支持多标签页切换 |
| **对话面板** | 右侧 AI 对话区域，与 AI 交流的核心界面 |
| **状态栏** | 底部状态信息，显示文件编码、行号列号 |
| **终端面板** | 底部内置终端，执行命令和脚本 |

---

## 快捷键一览

| 快捷键 | 功能 |
|--------|------|
| Ctrl+S | 保存当前文件 |
| Ctrl+B | 切换侧栏显示 |
| Ctrl+\\\` | 切换终端面板 |
| Ctrl+K | 专注模式（隐藏所有面板） |
| Ctrl+Shift+E | 切换到文件浏览器 |
| Ctrl+Shift+F | 全局搜索 |
| Ctrl+Shift+T | 打开对话面板 |
| Ctrl+Shift+C | 切换对话面板 |
`,no=`# API 文档\r
\r
PairCode IDE 内置了一套完整的 HTTP REST API + WebSocket 实时通信协议，供 Web 前端与后端核心功能交互，也**支持第三方开发者基于本 API 进行二次开发**。所有 API 地址均以 \`/api\` 开头，返回 JSON 格式数据。\r
\r
> **安全提示**：Web 服务监听所有接口（0.0.0.0），本机可用 \`http://localhost:{port}\` 访问，局域网内其他设备可用本机 IP 访问。请勿将服务端口暴露到公网，并注意防火墙与访问控制。\r
\r
---\r
\r
## 通用约定\r
\r
### 请求格式\r
- 查询参数（GET）直接在 URL 中传递\r
- POST / PUT 请求体使用 \`application/json\`\r
- 无特殊说明时，Content-Type 为 \`application/json\`\r
\r
### 响应格式\r
| 场景 | 格式 | 说明 |\r
|------|------|------|\r
| 成功 | JSON 对象 或 JSON 数组 | 直接返回业务数据 |\r
| 错误 | \`{"error": "错误描述信息"}\` | HTTP 状态码 4xx/5xx |\r
\r
### 错误码惯例\r
| HTTP 状态码 | 含义 |\r
|-------------|------|\r
| 200 | 成功 |\r
| 400 | 参数错误 / 请求体错误 |\r
| 404 | 资源不存在 |\r
| 405 | 方法不允许（如 GET 用了 POST） |\r
| 500 | 服务器内部错误 |\r
\r
---\r
\r
## 一、服务健康检查\r
\r
检查 IDE 后端服务是否正常运行。\r
\r
\`\`\`\r
GET /api/health\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "status": "ok",\r
  "workspace": "F:/projects/my-app",\r
  "folders": ["F:/projects/my-app"]\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| status | string | 固定 \`"ok"\` |\r
| workspace | string | 当前工作区路径 |\r
| folders | string[] | 工作区包含的文件夹列表 |\r
\r
---\r
\r
## 二、文件系统操作\r
\r
浏览、读写和管理工作区内的文件与目录。\r
\r
> **路径语义（多项目工作区）**\r
> - **相对路径一律相对「主项目根」解析**（主项目 = 工作区第一个文件夹，不是进程 cwd）；返回与记录的文件路径均为主项目根相对路径（跨项目时形如 \`../<项目名>/…\`）。\r
> - 访问**工作区其他文件夹（其他项目）**必须传**绝对路径**——相对路径不会自动跳到其他项目，越界会直接报「路径不在当前项目内」，请勿反复重试同一相对路径。\r
> - Agent 侧工具（\`read\`/\`write\`/\`glob\`/\`grep\`/\`exec_command\`）另有 \`project\` 参数（项目目录名 / 相对主项目的路径 / 绝对路径）可直接把解析根切到目标项目；HTTP 接口无此参数，用绝对路径等价。\r
\r
### 2.1 列出目录\r
\r
\`\`\`\r
GET /api/fs/list?path={目录路径}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 目录路径（相对主项目根解析；跨项目请传绝对路径），省略时返回主项目根目录 |\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {"name": "src", "isDir": true, "size": 4096, "modTime": "2026-07-11T10:00:00Z"},\r
  {"name": "main.go", "isDir": false, "size": 2048, "modTime": "2026-07-11T09:30:00Z"}\r
]\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| name | string | 文件/目录名 |\r
| isDir | boolean | 是否为目录 |\r
| size | number | 文件大小（字节） |\r
| modTime | string | 最后修改时间（ISO 8601） |\r
\r
---\r
\r
### 2.2 读取文件\r
\r
\`\`\`\r
GET /api/fs/read?path={文件路径}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 是 | 文件路径（相对主项目根解析；跨项目请传绝对路径） |\r
\r
**响应：** 返回文件文本内容（字符串）。\r
\r
---\r
\r
### 2.3 写入文件\r
\r
\`\`\`\r
POST /api/fs/write\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "src/main.go",\r
  "content": "package main\\n\\nfunc main() {\\n\\tprintln(\\"hello\\")\\n}\\n"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 是 | 文件路径（相对主项目根解析；跨项目请传绝对路径） |\r
| content | string | 是 | 文件内容（覆盖写入，自动创建目录） |\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 2.4 搜索文件内容\r
\r
\`\`\`\r
GET /api/fs/search?q={关键词}&path={搜索路径}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| q | string | 是 | 搜索关键词 |\r
| path | string | 否 | 搜索目录（相对主项目根解析；跨项目请传绝对路径），省略时使用主项目根 |\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {"file": "src/main.go", "line": 15, "text": "func handleRequest(w http.ResponseWriter, r *http.Request) {"},\r
  {"file": "src/utils.go", "line": 42, "text": "// handleRequest 处理 HTTP 请求"}\r
]\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| file | string | 文件相对路径 |\r
| line | number | 行号 |\r
| text | string | 匹配行的内容 |\r
\r
**自动忽略的目录**（与内核搜索忽略集 \`internal/agent/search.go\` 同源）：\r
- 依赖库/模块库：\`node_modules\` \`bower_components\` \`jspm_packages\` \`vendor\` \`pods\` \`.pnpm-store\` \`.yarn\` \`.dart_tool\` \`.bundle\` \`venv\` \`.venv\` \`__pycache__\` \`.pytest_cache\` \`.mypy_cache\` \`.ruff_cache\` \`.tox\`\r
- 构建产物/缓存：\`dist\` \`build\` \`out\` \`target\` \`.next\` \`.nuxt\` \`.svelte-kit\` \`.output\` \`.angular\` \`.gradle\` \`.cache\` \`.turbo\` \`.parcel-cache\` \`.eslintcache\` \`coverage\` \`.nyc_output\` \`.terraform\`\r
- VCS/IDE：\`.git\` \`.svn\` \`.hg\` \`.idea\` \`.vscode\` \`.vs\`\r
- IDE 运行数据：\`.pair\` \`_temp\` \`tmp\` \`logs\` \`bin\` \`release\` \`obj\` \`screenshots\` \`gocache\` \`.agent-teams\` \`.verify-tmp\` \`.chrome-test\` \`源码备份\` 等\r
\r
> 本接口按**目录名任意深度**剪枝；Agent 工具（\`grep\`/\`glob\`，内核实现）对 IDE 运行数据目录更精细——\`_temp\`/\`bin\`/\`release\`/\`logs\`/\`screenshots\` 等仅当位于项目根第一层时才剪枝。\r
> 跳过只作用于**递归下降**：把 \`path\` 直接指进被忽略目录，仍可搜索其内容。可用设置项 \`ignoreDirs\` 追加自定义忽略目录名（实时生效，无需重启）。\r
\r
**仅搜索文本文件扩展名**（\`.go\` \`.js\` \`.ts\` \`.vue\` \`.html\` \`.css\` \`.json\` \`.md\` \`.py\` \`.rs\` \`.java\` 等 50+ 种）。\r
\r
---\r
\r
### 2.5 重命名/移动文件\r
\r
\`\`\`\r
POST /api/fs/rename\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "oldPath": "src/old.go",\r
  "newPath": "src/new.go"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| oldPath | string | 是 | 原路径 |\r
| newPath | string | 是 | 新路径 |\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 2.6 删除文件/目录\r
\r
\`\`\`\r
POST /api/fs/delete\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "src/temp.go"\r
}\r
\`\`\`\r
\r
> ⚠️ 不可恢复，递归删除目录及其所有内容。\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 2.7 创建目录\r
\r
\`\`\`\r
POST /api/fs/mkdir\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "src/new-folder"\r
}\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 2.8 获取图片数据\r
\r
\`\`\`\r
GET /api/fs/image?path={图片路径}\r
\`\`\`\r
\r
**参数：** \`path\` — 图片文件路径（支持 PNG / JPEG）\r
\r
**响应：** Base64 编码的图片数据字符串（不含 \`data:image/...\` 前缀）。\r
\r
**响应头：** \`Content-Type: text/plain; charset=utf-8\`\r
\r
---\r
\r
### 2.9 获取文件信息\r
\r
\`\`\`\r
GET /api/fs/file-info?path={文件路径}\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "name": "main.go",\r
  "path": "F:/projects/my-app/src/main.go",\r
  "size": 2048,\r
  "modTime": "2026-07-11T09:30:00Z",\r
  "isDir": false\r
}\r
\`\`\`\r
\r
---\r
\r
### 2.10 十六进制查看\r
\r
\`\`\`\r
GET /api/fs/hex?path={文件路径}&offset={偏移}&length={长度}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 是 | 文件路径 |\r
| offset | number | 否 | 起始字节偏移（默认 0） |\r
| length | number | 否 | 读取字节数（默认 512，最大 4096） |\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "hex": "4d5a90000300000004000000ffff0000b80000000000000040",\r
  "text": "MZ.............@",\r
  "offset": 0,\r
  "length": 32\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| hex | string | 十六进制字符串 |\r
| text | string | ASCII 可打印字符（不可打印的替换为 \`.\`） |\r
| offset | number | 起始偏移 |\r
| length | number | 返回的字节数 |\r
\r
---\r
\r
### 2.11 列出磁盘驱动器\r
\r
\`\`\`\r
GET /api/fs/drives\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
["C:\\\\", "D:\\\\", "E:\\\\"]\r
\`\`\`\r
\r
---\r
\r
## 三、工作区管理\r
\r
### 3.1 获取当前工作区\r
\r
\`\`\`\r
GET /api/workspace\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "root": "F:/projects/my-app",\r
  "folders": ["F:/projects/my-app"],\r
  "loaded": true\r
}\r
\`\`\`\r
\r
### 3.2 切换/设置工作区\r
\r
\`\`\`\r
POST /api/workspace\r
\`\`\`\r
\r
**请求体（切换工作区）：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/another-project"\r
}\r
\`\`\`\r
\r
**请求体（添加文件夹）：**\r
\`\`\`json\r
{\r
  "addFolder": "F:/projects/shared-lib"\r
}\r
\`\`\`\r
\r
**请求体（创建新工作区）：**\r
\`\`\`json\r
{\r
  "create": "F:/projects/new-project"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 按场景 | 切换工作区到指定路径 |\r
| addFolder | string | 按场景 | 在当前工作区添加文件夹 |\r
| create | string | 按场景 | 创建新目录并切换为其工作区 |\r
\r
**响应：** 返回更新后的工作区信息（同 GET 响应格式）。\r
\r
---\r
\r
## 四、设置管理\r
\r
### 4.1 读取设置\r
\r
\`\`\`\r
GET /api/settings\r
\`\`\`\r
\r
**响应：** 返回全局设置、插件配置描述与加载状态三部分：\r
\r
- \`settings\` — 全局设置对象（落盘 \`config/settings.json\`），只含**当前版本仍支持**的字段\r
- \`schemas\` — 各插件通过 \`ctx.registerSettings\` 注册的配置项描述（\`key\` / \`title\` / \`fields[]\`），\r
  设置面板按此动态渲染；插件配置的取值在 \`settings.pluginSettings.<插件key>\` 下\r
- \`loaded\` — 配置文件是否成功加载\r
\r
\`\`\`json\r
{\r
  "loaded": true,\r
  "settings": {\r
    "provider": "deepseek",\r
    "baseURL": "https://api.deepseek.com/v1/chat/completions",\r
    "apiKey": "sk-xxx",\r
    "model": "deepseek-v4-flash",\r
    "planModel": "deepseek-v4-pro",\r
    "executeModel": "deepseek-v4-flash",\r
    "reviewModel": "deepseek-v4-pro",\r
    "preset": "默认",\r
    "modelParams": {},\r
    "temperature": "0.3",\r
    "thinkingMode": "thinking",\r
    "maxTokens": 131072,\r
    "contextMaxTokens": 64000,\r
    "lastProject": "F:/projects/my-app",\r
    "workspaceFolders": ["F:/projects/my-app"],\r
    "workspaceFolderLists": {"F:/projects/my-app": ["F:/projects/my-app"]},\r
    "recentProjects": ["F:/projects/app1"],\r
    "reviewMode": "auto",\r
    "reviewBlacklist": [],\r
    "reviewWhitelist": [],\r
    "autonomous": false,\r
    "autoIterateOnRejection": true,\r
    "systemInstructions": "",\r
    "ignoreDirs": [],\r
    "theme": "dark",\r
    "fontSize": 14,\r
    "tabSize": 2,\r
    "skillEnabledOverrides": {},\r
    "skillStatusOverrides": {},\r
    "pluginSettings": {\r
      "agentloop": {\r
        "stepBudget": 120,\r
        "toolCallBudget": 120,\r
        "maxToolBudgetSegments": 20\r
      }\r
    }\r
  },\r
  "schemas": [\r
    {\r
      "key": "agentloop",\r
      "title": "Agent 循环（agentloop）",\r
      "fields": [\r
        {"name": "stepBudget", "label": "单段步数预算", "type": "number", "default": 120},\r
        {"name": "toolCallBudget", "label": "单段工具调用预算", "type": "number", "default": 120},\r
        {"name": "maxToolBudgetSegments", "label": "最大自动续跑段数", "type": "number", "default": 20}\r
      ]\r
    }\r
  ]\r
}\r
\`\`\`\r
\r
**段预算（双闸门）字段说明：**\r
\r
| 字段（\`pluginSettings.agentloop\`） | 默认 | 语义 |\r
|------|------|------|\r
| stepBudget | 120 | 单段步数预算（一步 = 一次模型调用；一次回复并列多个工具调用仍算 1 步）。0 或留空 = 默认，负数 = 不限 |\r
| toolCallBudget | 120 | 单段工具调用预算（仅统计实际执行，审批驳回 / 截断不计数）。0 或留空 = 默认，负数 = 不限 |\r
| maxToolBudgetSegments | 20 | 自动续跑段数上限（不允许「不限」，防失控）。0 / 留空 / 负数 = 默认 20 |\r
\r
> 任一闸门达上限即结束当前段并自动开启续跑段（同会话历史保留）；修改后立即生效，无需重启。\r
\r
### 4.2 保存设置\r
\r
\`\`\`\r
PUT /api/settings?convId={对话ID}\r
\`\`\`\r
\r
**请求体：** 与 GET 返回格式相同，只需传入要修改的字段（增量合并，未传字段保持不变）。\r
\r
**参数：** \`convId\` — 可选，当前对话 ID。当 \`reviewMode\` 字段变更时，实时更新该对话的 Loop 审核模式。\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 五、系统工具\r
\r
### 5.1 系统信息\r
\r
\`\`\`\r
GET /api/system/info\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "hostname": "DESKTOP-ABC123",\r
  "cwd": "F:/projects/my-app",\r
  "os": "windows",\r
  "goos": "windows",\r
  "workspace": "F:/projects/my-app",\r
  "folders": ["F:/projects/my-app"],\r
  "version": "v1.6.8"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| hostname | string | 主机名 |\r
| cwd | string | 当前工作目录 |\r
| os | string | 操作系统名称 |\r
| goos | string | Go 平台标识 |\r
| workspace | string | IDE 工作区根路径 |\r
| folders | string[] | 工作区文件夹列表 |\r
| version | string | IDE 版本号（由打包器注入） |\r
\r
### 5.2 执行命令\r
\r
\`\`\`\r
POST /api/system/exec\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "command": "go build ./cmd/app",\r
  "cwd": "F:/projects/my-app"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| command | string | 是 | 要执行的命令 |\r
| cwd | string | 否 | 工作目录（默认工作区根目录） |\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "stdout": "# github.com/foo/app\\nsrc/main.go:42: undefined: x\\n",\r
  "stderr": "",\r
  "exitCode": 2\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| stdout | string | 标准输出 |\r
| stderr | string | 标准错误 |\r
| exitCode | number | 退出码（0 = 成功） |\r
\r
> **安全限制：** 命令在工作区目录下执行；禁止交互式命令（如 \`vim\`）。\r
\r
---\r
\r
## 六、AI 模型\r
\r
### 获取可用模型列表\r
\r
\`\`\`\r
GET /api/models\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "providers": [\r
    {\r
      "name": "openai",\r
      "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"]\r
    },\r
    {\r
      "name": "claude",\r
      "models": ["claude-3-opus", "claude-3-sonnet", "claude-3-haiku"]\r
    }\r
  ],\r
  "current": {\r
    "provider": "openai",\r
    "model": "gpt-4"\r
  }\r
}\r
\`\`\`\r
\r
---\r
\r
## 七、对话管理\r
\r
### 7.1 对话列表\r
\r
\`\`\`\r
GET /api/conversations?workspace={工作区路径}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| workspace | string | 否 | 工作区路径，省略时使用当前工作区 |\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {\r
    "id": "conv_1741680000000",\r
    "title": "修复登录页面样式",\r
    "createdAt": "2026-07-11T10:00:00Z",\r
    "messageCount": 12,\r
    "workspace": "F:/projects/my-app"\r
  }\r
]\r
\`\`\`\r
\r
### 7.2 创建对话\r
\r
\`\`\`\r
POST /api/conversations\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "title": "新对话",\r
  "workspace": "F:/projects/my-app"\r
}\r
\`\`\`\r
\r
**响应：** 返回创建的对话对象（同 GET 列表中的格式）。\r
\r
### 7.3 获取对话详情（含消息）\r
\r
\`\`\`\r
GET /api/conversations/{convId}\r
\`\`\`\r
\r
**响应：** 返回该对话的最近 50 条消息：\r
\r
\`\`\`json\r
{\r
  "messages": [\r
    {"role": "user", "content": "帮我写一个 HTTP 服务", "createdAt": "2026-07-11T10:00:00Z"},\r
    {"role": "assistant", "content": "好的，我来创建...", "createdAt": "2026-07-11T10:00:05Z"}\r
  ],\r
  "total": 42\r
}\r
\`\`\`\r
\r
### 7.4 更新对话\r
\r
\`\`\`\r
PUT /api/conversations/{convId}\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "title": "新的标题"\r
}\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 7.5 删除对话\r
\r
\`\`\`\r
DELETE /api/conversations/{convId}\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`（同时删除该对话的所有消息）。\r
\r
### 7.6 获取消息列表（分页）\r
\r
\`\`\`\r
GET /api/conversations/{convId}/messages?limit={数量}&before={索引}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| limit | number | 否 | 返回消息条数（默认 50） |\r
| before | number | 否 | 从消息索引 before 处开始往前加载（用于分页翻历史） |\r
\r
**响应：**\r
\`\`\`json\r
{\r
  "messages": [\r
    {"role": "user", "content": "第一条消息", "createdAt": "..."},\r
    {"role": "assistant", "content": "回复", "createdAt": "..."}\r
  ],\r
  "total": 42\r
}\r
\`\`\`\r
\r
> 连续的 assistant 消息会被合并（\`MergeConsecutiveAssistants\`）。\r
\r
### 7.7 添加消息\r
\r
\`\`\`\r
POST /api/conversations/{convId}/messages\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "role": "user",\r
  "content": "继续上一个话题"\r
}\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 7.8 消息总数\r
\r
\`\`\`\r
GET /api/conversations/{convId}/messages/count\r
\`\`\`\r
\r
**响应：** \`{"count": 42}\`\r
\r
### 7.9 发送消息给 AI（非阻塞）\r
\r
\`\`\`\r
POST /api/chat/send\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "message": "帮我创建一个 Go HTTP 服务",\r
  "sessionId": "sess_xxx",\r
  "convId": "conv_1741680000000",\r
  "autonomous": false,\r
  "workspaceRoot": "F:/projects/my-app"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| message | string | 是 | 用户消息内容（最长 50000 字符，超出截断） |\r
| sessionId | string | 否 | 会话 ID |\r
| convId | string | 否 | 对话 ID（留空则自动生成 \`conv_{时间戳}\`） |\r
| autonomous | boolean | 否 | 是否启用自主模式（默认 false） |\r
| workspaceRoot | string | 否 | 工作区路径（默认当前工作区） |\r
\r
**响应：** \`{"sessionId": "sess_xxx", "convId": "conv_1741680000000"}\`\r
\r
AI 的回复不在此响应的 Body 中返回，而是通过 **WebSocket 实时推送**事件流（见第十七章）。\r
\r
**前置条件：** 必须先配置 API Key 和模型。\r
\r
---\r
\r
### 7.10 停止 AI 响应\r
\r
\`\`\`\r
POST /api/chat/stop?convId={对话ID}\r
\`\`\`\r
\r
**参数：** \`convId\` — 要停止的对话 ID。\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 7.11 审批操作\r
\r
\`\`\`\r
POST /api/chat/approve\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "convId": "conv_xxx",\r
  "approved": true,\r
  "reply": "请把函数名改为驼峰命名法"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| convId | string | 是 | 对话 ID |\r
| approved | boolean | 是 | 批准（true）或拒绝（false） |\r
| reply | string | 否 | 拒绝时的反馈/纠正建议 |\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 7.12 发送运行时反馈\r
\r
\`\`\`\r
POST /api/chat/feedback\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "convId": "conv_xxx",\r
  "feedback": "请改用更简洁的实现方式"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| convId | string | 是 | 对话 ID |\r
| feedback | string | 是 | 反馈/纠正内容 |\r
\r
**工作原理：** 在 AI 下次 LLM 调用前，将反馈内容作为用户消息注入本轮上下文，让 AI 在下一次回复中响应用户的补充或纠正。\r
\r
---\r
\r
### 7.13 回答 ask_user 提问\r
\r
\`\`\`\r
POST /api/chat/answer\r
\`\`\`\r
\r
当 AI 通过 \`ask_user\` 工具向用户提问时，用此接口发送回答。\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "convId": "conv_xxx",\r
  "answer": "用 POST 方法"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| convId | string | 是 | 对话 ID |\r
| answer | string | 是 | 用户的回答 |\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
### 7.14 压缩上下文\r
\r
\`\`\`\r
POST /api/chat/compact?convId={对话ID}\r
\`\`\`\r
\r
手动触发上下文压缩：将对话中间部分的老消息压缩为摘要，释放 token 预算。\r
\r
**参数：** \`convId\` — 对话 ID。\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 八、指令与思想\r
\r
### 8.1 读取指令\r
\r
\`\`\`\r
GET /api/instructions?scope={作用域}\r
\`\`\`\r
\r
**参数：** \`scope\` — 指令作用域（如 \`"system"\`、\`"user"\`）。\r
\r
**响应：** 返回指令文本内容（字符串）。\r
\r
### 8.2 保存指令\r
\r
\`\`\`\r
PUT /api/instructions?scope={作用域}\r
\`\`\`\r
\r
**请求体：** 纯文本字符串（指令内容）。\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 8.3 读取行为指导\r
\r
\`\`\`\r
\`\`\`\r
\r
**响应：** 返回 AI 行为指导配置文本。\r
\r
### 8.4 保存行为指导\r
\r
\`\`\`\r
\`\`\`\r
\r
**请求体：** 纯文本字符串。\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 九、任务与规划\r
\r
> **注意：** 任务由 Agent 通过 \`update_tasks\` / \`update_plan\` 工具自主管理。以下 API 仅提供前端只读查询接口。\r
\r
### 9.1 获取任务列表\r
\r
\`\`\`\r
GET /api/tasks?convId={对话ID}\r
\`\`\`\r
\r
**参数：** \`convId\` — 可选，过滤指定对话的任务。\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "tasks": [\r
    {\r
      "step": "创建 HTTP 服务文件",\r
      "status": "completed",\r
      "taskId": "task_1",\r
      "description": "在 src/server.go 创建 HTTP 服务",\r
      "created_at": "2026-07-11T10:00:00Z"\r
    }\r
  ]\r
}\r
\`\`\`\r
\r
> 任务数据持久化在工作区 \`.pair/tasks/*.json\`，由 Agent 的 \`update_tasks\` 工具写入。\r
\r
### 9.2 读取任务规划文档\r
\r
\`\`\`\r
GET /api/taskplan?name={规划名}\r
\`\`\`\r
\r
列出或读取 Markdown 格式的规划文档。\r
\r
**参数：** \`name\` — 可选，指定规划文档名（不含 \`.md\` 后缀）；省略则返回所有规划文档列表。\r
\r
**GET 响应（列出全部）：**\r
\`\`\`json\r
[\r
  {"name": "refactor-auth", "file": "F:/projects/.pair/tasks/refactor-auth.md"}\r
]\r
\`\`\`\r
\r
**GET 响应（读单个）：**\r
\`\`\`json\r
{\r
  "name": "refactor-auth",\r
  "content": "## 重构计划\\n1. 提取认证中间件\\n2. 添加 JWT 支持"\r
}\r
\`\`\`\r
\r
### 9.3 追加/完成规划文档\r
\r
\`\`\`\r
POST /api/taskplan\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "name": "refactor-auth",\r
  "content": "- 完成 JWT 集成",\r
  "action": "append"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| name | string | 否 | 规划名称（省略则自动生成 \`plan_日期时间\`） |\r
| content | string | 是 | 要追加的内容（Markdown） |\r
| action | string | 否 | \`"append"\`（追加）或 \`"complete"\`（追加"[已完成] 时间戳"），默认 \`"append"\` |\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 十、Git 版本控制\r
\r
所有 Git API 均在**当前工作区目录**（或指定仓库路径）下执行。\r
\r
### 10.1 初始化仓库\r
\r
\`\`\`\r
POST /api/git/init?path={目录路径}\r
\`\`\`\r
\r
**参数：** \`path\` — 目标目录（默认当前工作区）。\r
\r
**响应：** \`{"output": "Initialized empty Git repository in ..."}\`\r
\r
---\r
\r
### 10.2 仓库状态\r
\r
\`\`\`\r
GET /api/git/status?path={仓库路径}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 仓库路径（默认当前工作区） |\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "branch": "main",\r
  "changes": [\r
    {"path": "src/main.go", "status": "M", "staged": false},\r
    {"path": "src/utils.go", "status": "M", "staged": true}\r
  ],\r
  "untracked": ["src/new.go"],\r
  "ahead": 1,\r
  "behind": 0\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| branch | string | 当前分支名 |\r
| changes[].path | string | 变更文件路径 |\r
| changes[].status | string | 状态码：\`M\`(修改) \`A\`(新增) \`D\`(删除) \`R\`(重命名) |\r
| changes[].staged | boolean | 是否已暂存 |\r
| untracked | string[] | 未跟踪文件列表 |\r
| ahead | number | 领先远程的提交数 |\r
| behind | number | 落后远程的提交数 |\r
\r
### 10.3 查看差异\r
\r
\`\`\`\r
GET /api/git/diff?path={仓库路径}&file={文件路径}&staged={是否暂存}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 仓库路径 |\r
| file | string | 否 | 指定文件（省略则返回所有变更的 diff） |\r
| staged | string | 否 | \`"true"\` = 只显示已暂存差异（--cached） |\r
\r
**响应：** 返回 diff 文本（字符串）。\r
\r
### 10.4 暂存文件\r
\r
\`\`\`\r
POST /api/git/add\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/my-app",\r
  "files": ["src/main.go", "src/utils.go"]\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 仓库路径（默认工作区） |\r
| files | string[] | 否 | 要暂存的文件列表（省略则暂存全部 \`-A\`） |\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.5 取消暂存\r
\r
\`\`\`\r
POST /api/git/reset\r
\`\`\`\r
\r
**请求体：** 格式同 \`git/add\`。\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.6 提交\r
\r
\`\`\`\r
POST /api/git/commit\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/my-app",\r
  "message": "feat: 添加用户认证模块"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 仓库路径 |\r
| message | string | 是 | 提交信息 |\r
\r
**响应：**\r
\`\`\`json\r
{\r
  "ok": true,\r
  "hash": "a1b2c3d4e5f6..."\r
}\r
\`\`\`\r
\r
### 10.7 查看提交历史\r
\r
\`\`\`\r
GET /api/git/log?path={仓库路径}&count={数量}&file={文件路径}\r
\`\`\`\r
\r
> **别名：** \`/api/git-log\`（绕过部分浏览器广告拦截器对 \`/api/git/log\` 的误杀）。\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 仓库路径 |\r
| count | number | 否 | 返回条数（默认 15） |\r
| file | string | 否 | 限定某文件的提交历史 |\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {\r
    "hash": "a1b2c3d",\r
    "author": "user",\r
    "date": "2026-07-11 10:00:00",\r
    "message": "feat: 添加用户认证模块"\r
  }\r
]\r
\`\`\`\r
\r
### 10.8 分支管理\r
\r
\`\`\`\r
POST /api/git/branch\r
\`\`\`\r
\r
| 操作 | 请求体 | 说明 |\r
|------|--------|------|\r
| 创建 | \`{"path":"...","name":"feature-x","action":"create"}\` | 创建新分支 |\r
| 删除 | \`{"path":"...","name":"feature-x","action":"delete"}\` | 删除分支 |\r
| 列表 | \`{"path":"...","action":"list"}\` | 列出所有分支 |\r
| 切换 | \`{"path":"...","name":"feature-x","action":"checkout"}\` | 切换分支 |\r
\r
**响应：** 列表操作返回 \`["main", "feature-x", ...]\`，其他返回 \`{"ok": true}\`。\r
\r
### 10.9 切换分支 / 恢复文件\r
\r
\`\`\`\r
POST /api/git/checkout\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/my-app",\r
  "branch": "feature-x",\r
  "file": "src/main.go"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| branch | string | 按场景 | 切换到的分支名 |\r
| file | string | 按场景 | 恢复指定文件到 HEAD（branch 和 file 二选一） |\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.10 贮藏\r
\r
\`\`\`\r
POST /api/git/stash\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/my-app",\r
  "action": "push",\r
  "message": "暂存当前 WIP"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 仓库路径 |\r
| action | string | 否 | \`"push"\`(贮藏,默认) \\| \`"pop"\`(恢复) \\| \`"apply"\`(应用) \\| \`"drop"\`(丢弃) |\r
| message | string | 否 | 贮藏备注 |\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.11 查看贮藏列表\r
\r
\`\`\`\r
GET /api/git/stash-list?path={仓库路径}\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {"index": 0, "message": "暂存当前 WIP"},\r
  {"index": 1, "message": "On feature-x: 临时保存"}\r
]\r
\`\`\`\r
\r
### 10.12 管理 \`.gitignore\`\r
\r
\`\`\`\r
GET /api/git/ignore?path={仓库路径}\r
POST /api/git/ignore?path={仓库路径}\r
\`\`\`\r
\r
**GET 响应：** 返回当前 \`.gitignore\` 内容：\r
\`\`\`json\r
{\r
  "content": "*.log\\n.env\\nbuild/",\r
  "rules": ["*.log", ".env", "build/"]\r
}\r
\`\`\`\r
\r
**POST 请求体（覆盖写入）：**\r
\`\`\`json\r
{\r
  "content": "*.log\\n.env\\nnode_modules/"\r
}\r
\`\`\`\r
\r
**POST 请求体（追加一行）：**\r
\`\`\`json\r
{\r
  "append": "dist/"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| content | string | 按场景 | 完整覆盖 \`.gitignore\` 内容 |\r
| append | string | 按场景 | 追加一行到 \`.gitignore\`（content 和 append 二选一） |\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.13 丢弃修改\r
\r
\`\`\`\r
POST /api/git/discard\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/my-app",\r
  "files": ["src/main.go"]\r
}\r
\`\`\`\r
\r
> ⚠️ 不可恢复！丢弃工作区未暂存的修改。\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.14 推送\r
\r
\`\`\`\r
POST /api/git/push\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "path": "F:/projects/my-app",\r
  "remote": "origin",\r
  "branch": "main"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| remote | string | 否 | 远程名（默认 \`"origin"\`） |\r
| branch | string | 否 | 分支名（默认当前分支） |\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.15 拉取\r
\r
\`\`\`\r
POST /api/git/pull\r
\`\`\`\r
\r
**请求体：** 同 \`git/push\`。\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 10.16 远程仓库管理\r
\r
\`\`\`\r
GET /api/git/remote?path={仓库路径}\r
POST /api/git/remote?path={仓库路径}\r
\`\`\`\r
\r
**GET 响应示例：**\r
\`\`\`json\r
[\r
  {"name": "origin", "url": "https://github.com/user/repo.git"}\r
]\r
\`\`\`\r
\r
**POST 请求体：**\r
\`\`\`json\r
{\r
  "name": "upstream",\r
  "url": "https://github.com/other/repo.git",\r
  "action": "add"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| name | string | 是 | 远程名 |\r
| url | string | 是 | 远程 URL |\r
| action | string | 否 | \`"add"\`（添加）或 \`"remove"\`（删除），默认 \`"add"\` |\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 十一、Skills 技能\r
\r
### 11.1 技能列表\r
\r
\`\`\`\r
GET /api/skills/list\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {\r
    "name": "code-review",\r
    "description": "代码审查工作流",\r
    "mode": "auto",\r
    "version": "1.0"\r
  }\r
]\r
\`\`\`\r
\r
### 11.2 读取技能\r
\r
\`\`\`\r
GET /api/skills/read?name={技能名}&level={层级}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| name | string | 是 | 技能名 |\r
| level | string | 否 | \`"system"\`（全局）或 \`"project"\`（项目，默认） |\r
\r
**响应：** 返回技能的完整 Markdown 内容。\r
\r
### 11.3 保存/更新技能状态\r
\r
\`\`\`\r
POST /api/skills/save\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "name": "code-review",\r
  "level": "project",\r
  "action": "set-status",\r
  "status": "on"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| name | string | 是 | 技能名 |\r
| level | string | 否 | \`"system"\` / \`"project"\`（默认 project） |\r
| action | string | 是 | 固定 \`"set-status"\` |\r
| status | string | 是 | \`"off"\` \\| \`"on"\` \\| \`"max"\` |\r
\r
**响应：** \`{"ok": true, "action": "set-status", "name": "code-review", "status": "on"}\`\r
\r
### 11.4 删除技能\r
\r
\`\`\`\r
POST /api/skills/delete\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "name": "code-review"\r
}\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 十二、MCP 扩展\r
\r
### 12.1 MCP 列表\r
\r
\`\`\`\r
GET /api/mcp/list?level={层级}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| level | string | 否 | 层级过滤（\`"user"\`、\`"project"\`） |\r
\r
### 12.2 MCP 保存/管理\r
\r
\`\`\`\r
POST /api/mcp/save\r
\`\`\`\r
\r
统一管理 MCP 的添加、更新、删除和启用切换。\r
\r
**请求体（添加/更新）：**\r
\`\`\`json\r
{\r
  "name": "my-db",\r
  "command": "node",\r
  "args": ["mcp-server-db/index.js"],\r
  "level": "project"\r
}\r
\`\`\`\r
\r
**请求体（删除）：**\r
\`\`\`json\r
{\r
  "action": "delete",\r
  "name": "my-db",\r
  "level": "project"\r
}\r
\`\`\`\r
\r
**请求体（启用/禁用切换）：**\r
\`\`\`json\r
{\r
  "action": "toggle",\r
  "name": "my-db",\r
  "level": "project"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| action | string | 否 | \`"delete"\`（删除）\\| \`"toggle"\`（启用切换），省略则为新增/更新 |\r
| name | string | 是 | MCP 名称 |\r
| command | string | 新增时必填 | 启动命令 |\r
| args | string[] | 否 | 命令参数 |\r
| level | string | 否 | \`"user"\`（用户级）\\| \`"project"\`（项目级），默认 user |\r
\r
**响应：** \`{"ok": true, "action": "...", "name": "..."}\`\r
\r
---\r
\r
## 十三、Token 统计\r
\r
### 获取 Token 用量\r
\r
\`\`\`\r
GET /api/tokens/stats?workspaceRoot={工作区路径}\r
\`\`\`\r
\r
**参数：** \`workspaceRoot\` — 工作区路径（默认当前工作区）。\r
\r
**响应示例：**\r
\`\`\`json\r
{\r
  "workspaceRoot": "F:/projects/my-app",\r
  "promptTokens": 125000,\r
  "completionTokens": 45000,\r
  "totalTokens": 170000,\r
  "cost": 0.85\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 说明 |\r
|------|------|------|\r
| promptTokens | number | 提示词 Token 数 |\r
| completionTokens | number | 补全 Token 数 |\r
| totalTokens | number | 总 Token 数 |\r
| cost | number | 估算费用（美元） |\r
\r
---\r
\r
## 十四、调试日志\r
\r
### 14.1 日志列表\r
\r
\`\`\`\r
GET /api/debug/logs\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {"id": "log_001", "time": "2026-07-11T10:00:00Z", "session": "sess_xxx", "summary": "工具调用: read_file src/main.go"}\r
]\r
\`\`\`\r
\r
### 14.2 日志详情\r
\r
\`\`\`\r
GET /api/debug/logs/{日志ID}\r
\`\`\`\r
\r
**响应：** 返回指定日志的完整内容。\r
\r
---\r
\r
## 十五、技能市场\r
\r
### 15.1 搜索市场\r
\r
\`\`\`\r
GET /api/marketplace/search?q={关键词}&kind={类型}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| q | string | 否 | 搜索关键词 |\r
| kind | string | 否 | 类型（\`"mcp"\`、\`"skill"\`、\`"all"\`） |\r
\r
### 15.2 安装扩展\r
\r
\`\`\`\r
POST /api/marketplace/install\r
\`\`\`\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "id": "skill-code-review",\r
  "scope": "project"\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| id | string | 是 | 扩展 ID |\r
| scope | string | 否 | 安装范围（\`"user"\`、\`"project"\`） |\r
\r
**响应：** \`{"ok": true}\`\r
\r
### 15.3 刷新市场缓存\r
\r
\`\`\`\r
POST /api/marketplace/refresh\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 十六、记忆系统\r
\r
### 16.1 搜索记忆\r
\r
\`\`\`\r
GET /api/memory/search?q={关键词}\r
\`\`\`\r
\r
**响应示例：**\r
\`\`\`json\r
[\r
  {"name": "项目编码规范", "description": "使用驼峰命名法", "type": "project", "content": "..."}\r
]\r
\`\`\`\r
\r
### 16.2 记忆列表\r
\r
\`\`\`\r
GET /api/memory/list\r
\`\`\`\r
\r
### 16.3 重建索引\r
\r
\`\`\`\r
POST /api/memory/rebuild\r
\`\`\`\r
\r
**响应：** \`{"ok": true}\`\r
\r
---\r
\r
## 十七、插件与工具集管理\r
\r
PairCode IDE 的工具系统全部插件化（一切皆插件）。插件（plugin）是工具的最小可复用单元，工具集（toolset）是按项目需求组合的命名插件包。相关 API：\r
\r
### 17.1 插件管理\r
\r
\`\`\`\r
GET   /api/plugins            # 列出已注册插件（含工具归属）\r
GET   /api/plugins/detail     # 插件详情\r
POST  /api/plugins/define     # 定义 JS/TS 插件\r
POST  /api/plugins/action     # 插件动作（run/stop/inspect 等）\r
POST  /api/plugins/event      # 插件事件\r
GET   /api/plugins/client-state   # host/client 双半客户端状态\r
POST  /api/plugins/client-events  # 客户端事件\r
\`\`\`\r
\r
### 17.2 工具集管理\r
\r
\`\`\`\r
GET   /api/toolsets           # 列出工具集\r
POST  /api/toolsets/build     # 动态构建工具集（按项目+需求组合插件）\r
GET   /api/toolsets/export    # 导出工具集 JSON\r
POST  /api/toolsets/import    # 导入工具集（project/user 范围）\r
POST  /api/toolsets/remove    # 移除工具集\r
\`\`\`\r
\r
### 17.3 工具配置\r
\r
\`\`\`\r
GET   /api/tools              # 工具清单（含启用/审核状态）\r
POST  /api/tools/save         # 保存工具配置\r
POST  /api/tools/review       # 审核配置\r
\`\`\`\r
\r
---\r
\r
## 十八、WebSocket 实时通信协议\r
\r
PairCode IDE 使用 **WebSocket** 实现双向实时通信。\r
\r
### 17.1 AI 事件推送\r
\r
\`\`\`\r
ws://127.0.0.1:{port}/ws\r
\`\`\`\r
\r
**用途：** 接收 AI 对话的事件流（思考过程、工具调用、回复内容、错误等）。\r
\r
**协议：** 纯文本帧（JSON），**服务端单向推送**，客户端无需发送任何消息。\r
\r
#### 事件类型总表\r
\r
| 事件类型 | 说明 | 前端展示 |\r
|---------|------|---------|\r
| \`thinking\` | LLM 思考链增量 | 流式显示思考过程（斜体/灰色） |\r
| \`content\` | LLM 正文回复增量 | 流式显示正文内容 |\r
| \`tool_call\` | AI 即将执行某工具 | 显示工具调用卡片（工具名+参数） |\r
| \`tool_result\` | 工具执行结果返回 | 显示结果摘要 |\r
| \`usage\` | Token 用量统计 | 更新 Token 计数器 |\r
| \`approval\` | 请求用户审批写类操作 | 显示审批对话框（含工具名、参数、文件路径） |\r
| \`error\` | 出错或触发止损 | 显示错误信息 |\r
| \`done\` | 本次 AI 回复完成 | 关闭加载状态 |\r
| \`compacted\` | 上下文已压缩（旧消息被摘要替换） | 显示一条素色提示 |\r
| \`evaluation\` | 自主模式任务评分 | 显示评分卡 |\r
| \`circling\` | 检测到 AI 重复绕圈 | 显示"换思路"提示 |\r
| \`notice\` | 后台任务通知 | 显示一条素色提示 |\r
| \`phase\` | 自主模式阶段切换 | 显示阶段指示器（规划/执行/评测） |\r
| \`final\` | 单轮委托完成（delegate 用） | 同 done |\r
\r
#### 事件 JSON 格式\r
\r
\`\`\`json\r
{\r
  "type": "thinking",\r
  "content": "我来分析一下这个需求...",\r
  "tool": "",\r
  "args": "",\r
  "callId": "",\r
  "agentName": "",\r
  "usage": null,\r
  "doneReason": ""\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必含 | 说明 |\r
|------|------|------|------|\r
| type | string | 是 | 事件类型（见上表） |\r
| content | string | 按场景 | thinking/content/error/final 时携带文本内容 |\r
| tool | string | 按场景 | tool_call/tool_result 时携带工具名 |\r
| args | string | 按场景 | tool_call 时携带工具参数的 JSON 字符串 |\r
| callId | string | 按场景 | 工具调用 ID，用于关联 tool_call → tool_result |\r
| agentName | string | 按场景 | 事件来源 Agent 名。空串=主 Agent，非空=子 Agent |\r
| usage | object | 按场景 | usage 时携带：\`{promptTokens:N, completionTokens:N, totalTokens:N}\` |\r
| doneReason | string | 按场景 | done 时携带完成原因（\`"completed"\`、\`"stopped"\`、\`"error"\`） |\r
\r
#### 典型事件序列\r
\r
\`\`\`\r
→ {type:"thinking", content:"我来分析一下..."}\r
→ {type:"tool_call", tool:"read_file", args:"{\\"path\\":\\"main.go\\"}", callId:"call_1"}\r
→ {type:"tool_result", tool:"read_file", content:"文件内容...", callId:"call_1"}\r
→ {type:"thinking", content:"看到文件结构了，接下来..."}\r
→ {type:"tool_call", tool:"edit_file", args:"{\\"path\\":\\"main.go\\",\\"content\\":\\"...\\"}", callId:"call_2"}\r
→ {type:"approval", tool:"edit_file", args:"{\\"path\\":\\"main.go\\"}", callId:"call_2"}\r
   （等待用户审批 → 调用 POST /api/chat/approve）\r
→ {type:"tool_result", tool:"edit_file", content:"文件已更新", callId:"call_2"}\r
→ {type:"content", content:"已完成修改，以下是改动内容..."}\r
→ {type:"usage", content:"", usage:{promptTokens:1200, completionTokens:350, totalTokens:1550}}\r
→ {type:"done", doneReason:"completed"}\r
\`\`\`\r
\r
> **重要：** WebSocket 连接为全局单连接，推送**所有**会话的事件。事件中的 \`convId\` 字段（若存在）用于区分不同对话。前端需根据 \`convId\` 路由到对应的对话面板。\r
\r
---\r
\r
### 17.2 终端 WebSocket\r
\r
\`\`\`\r
ws://127.0.0.1:{port}/api/terminal/ws\r
\`\`\`\r
\r
**用途：** 内置终端的双向输入输出通道，每连接对应一个 PTY 终端会话。\r
\r
#### 协议规则\r
\r
| 帧类型 | 方向 | 说明 |\r
|--------|------|------|\r
| 文本帧 (JSON) | 客户端→服务端 | 控制消息 |\r
| 文本帧 (JSON) | 服务端→客户端 | 状态通知 |\r
| 二进制帧 | 双向 | 原始 PTY I/O 字节流（含 VT 转义序列，由 xterm.js 渲染） |\r
\r
#### 控制消息格式\r
\r
**客户端 → 服务端（初始化）：**\r
\`\`\`json\r
{"type": "init", "shell": "cmd", "cwd": "F:/projects/my-app"}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| type | string | 是 | 固定 \`"init"\` |\r
| shell | string | 是 | Shell 名：\`"cmd"\` \\| \`"powershell"\` \\| \`"gitbash"\`（白名单限制） |\r
| cwd | string | 是 | 工作目录（禁止穿越出工作区） |\r
\r
**客户端 → 服务端（调整大小）：**\r
\`\`\`json\r
{"type": "resize", "cols": 120, "rows": 30}\r
\`\`\`\r
\r
**服务端 → 客户端：**\r
\`\`\`json\r
{"type": "ready"}\r
{"type": "error", "msg": "shell 不在白名单中"}\r
{"type": "closed"}\r
\`\`\`\r
\r
#### 安全措施\r
\r
- Shell 白名单：仅允许 \`cmd\`、\`powershell\`、\`gitbash\`\r
- \`cwd\` 路径校验：禁止穿越出工作区\r
- PTY 关闭时强制终止子进程\r
- 并发 PTY 会话数限制：最多 16 个\r
\r
---\r
\r
## 附录：API 索引速查\r
\r
### 基础 API\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| GET | \`/api/health\` | 健康检查 |\r
| GET | \`/api/system/info\` | 系统信息+版本号 |\r
| POST | \`/api/system/exec\` | 执行命令 |\r
\r
### 文件系统 (11 个)\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| GET | \`/api/fs/list\` | 列出目录 |\r
| GET | \`/api/fs/read\` | 读取文件 |\r
| POST | \`/api/fs/write\` | 写入文件 |\r
| GET | \`/api/fs/search\` | 搜索内容 |\r
| POST | \`/api/fs/rename\` | 重命名/移动 |\r
| POST | \`/api/fs/delete\` | 删除 |\r
| POST | \`/api/fs/mkdir\` | 创建目录 |\r
| GET | \`/api/fs/image\` | 图片 Base64 |\r
| GET | \`/api/fs/file-info\` | 文件信息 |\r
| GET | \`/api/fs/hex\` | 十六进制查看 |\r
| GET | \`/api/fs/drives\` | 磁盘驱动器列表 |\r
\r
### 工作区 & 设置\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| GET/POST | \`/api/workspace\` | 工作区管理 |\r
| GET/PUT | \`/api/settings\` | 设置管理 |\r
\r
### AI 对话 (9 个)\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| POST | \`/api/chat/send\` | 发送消息给 AI |\r
| POST | \`/api/chat/stop\` | 停止 AI 回复 |\r
| POST | \`/api/chat/approve\` | 审批操作 |\r
| POST | \`/api/chat/feedback\` | 发送运行时反馈 |\r
| POST | \`/api/chat/answer\` | 回答 ask_user 提问 |\r
| POST | \`/api/chat/compact\` | 手动压缩上下文 |\r
| GET | \`/api/models\` | 可用模型列表 |\r
\r
### 对话管理 (8 个)\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| GET | \`/api/conversations\` | 对话列表 |\r
| POST | \`/api/conversations\` | 创建对话 |\r
| GET | \`/api/conversations/{id}\` | 对话详情（含消息） |\r
| PUT | \`/api/conversations/{id}\` | 更新对话 |\r
| DELETE | \`/api/conversations/{id}\` | 删除对话 |\r
| GET | \`/api/conversations/{id}/messages\` | 消息列表（分页） |\r
| POST | \`/api/conversations/{id}/messages\` | 添加消息 |\r
| GET | \`/api/conversations/{id}/messages/count\` | 消息总数 |\r
\r
### Git (16 个)\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| POST | \`/api/git/init\` | 初始化仓库 |\r
| GET | \`/api/git/status\` | 仓库状态 |\r
| GET | \`/api/git/diff\` | 查看差异 |\r
| POST | \`/api/git/add\` | 暂存 |\r
| POST | \`/api/git/reset\` | 取消暂存 |\r
| POST | \`/api/git/commit\` | 提交 |\r
| GET | \`/api/git/log\` | 提交历史 |\r
| GET | \`/api/git-log\` | 提交历史（别名） |\r
| POST | \`/api/git/branch\` | 分支管理 |\r
| POST | \`/api/git/checkout\` | 切换分支/恢复文件 |\r
| POST | \`/api/git/stash\` | 贮藏 |\r
| GET | \`/api/git/stash-list\` | 贮藏列表 |\r
| GET/POST | \`/api/git/ignore\` | 管理 .gitignore |\r
| POST | \`/api/git/discard\` | 丢弃修改 |\r
| POST | \`/api/git/push\` | 推送 |\r
| POST | \`/api/git/pull\` | 拉取 |\r
| GET/POST | \`/api/git/remote\` | 远程仓库管理 |\r
\r
### 扩展 & 系统\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| GET | \`/api/skills/list\` | 技能列表 |\r
| GET | \`/api/skills/read\` | 读取技能 |\r
| POST | \`/api/skills/save\` | 保存/更新技能状态 |\r
| POST | \`/api/skills/delete\` | 删除技能 |\r
| GET | \`/api/mcp/list\` | MCP 列表 |\r
| POST | \`/api/mcp/save\` | MCP 保存/管理 |\r
| GET | \`/api/tokens/stats\` | Token 统计 |\r
| GET | \`/api/debug/logs\` | 调试日志列表 |\r
| GET | \`/api/debug/logs/{id}\` | 调试日志详情 |\r
| GET | \`/api/memory/search\` | 搜索记忆 |\r
| GET | \`/api/memory/list\` | 记忆列表 |\r
| POST | \`/api/memory/rebuild\` | 重建记忆索引 |\r
| GET | \`/api/marketplace/search\` | 市场搜索 |\r
| POST | \`/api/marketplace/install\` | 安装扩展 |\r
| POST | \`/api/marketplace/refresh\` | 刷新市场缓存 |\r
| GET/PUT | \`/api/instructions\` | 指令管理 |\r
| GET | \`/api/tasks\` | 任务列表（只读查询） |\r
| GET/POST | \`/api/taskplan\` | 规划文档管理 |\r
\r
### 插件 & 工具集\r
| 方法 | 端点 | 用途 |\r
|------|------|------|\r
| GET | \`/api/plugins\` | 插件列表（含工具归属） |\r
| GET | \`/api/plugins/detail\` | 插件详情 |\r
| POST | \`/api/plugins/define\` | 定义 JS/TS 插件 |\r
| POST | \`/api/plugins/action\` | 插件动作（run/stop/inspect） |\r
| POST | \`/api/plugins/event\` | 插件事件 |\r
| GET | \`/api/plugins/client-state\` | host/client 客户端状态 |\r
| POST | \`/api/plugins/client-events\` | 客户端事件 |\r
| GET | \`/api/toolsets\` | 工具集列表 |\r
| POST | \`/api/toolsets/build\` | 动态构建工具集 |\r
| GET | \`/api/toolsets/export\` | 导出工具集 JSON |\r
| POST | \`/api/toolsets/import\` | 导入工具集 |\r
| POST | \`/api/toolsets/remove\` | 移除工具集 |\r
| GET | \`/api/tools\` | 工具清单 |\r
| POST | \`/api/tools/save\` | 保存工具配置 |\r
| POST | \`/api/tools/review\` | 审核配置 |\r
\r
---\r
\r
### WebSocket 端点\r
| 端点 | 用途 |\r
|------|------|\r
| \`ws://host/ws\` | AI 事件流推送（思考/工具/结果/完成） |\r
| \`ws://host/api/terminal/ws\` | PTY 终端双向 I/O |\r
`,to=`# AI 工具文档\r
\r
PairCode IDE 中的 AI 助手拥有丰富的内置能力，可以像你使用 IDE 一样操作文件、搜索代码、运行命令、管理版本。你只需用自然语言告诉 AI 你想做什么，AI 会自动选择合适的工具来完成任务。\r
\r
所有工具对 AI 完全开放，你无需记忆工具名称——只需描述需求，AI 自动判断该用什么。\r
\r
---\r
\r
## 一、代码阅读与搜索\r
\r
**浏览项目结构、搜索代码内容和定位符号定义，是 AI 理解你代码的基础能力。**\r
\r
AI 可以像你一样阅读和浏览项目代码：\r
\r
- 读取文件内容（可按行号范围读取部分内容）\r
- 列出目录下的文件和子目录\r
- 按关键词或正则表达式在文件内容中搜索\r
- 按通配符模式递归查找文件\r
- 搜索函数、类型、结构体等符号的定义位置\r
- 查看指定文件中所有检测到的符号\r
- 搜索某个符号在项目中的所有引用位置\r
- 列出项目中所有导出的公开符号\r
- 查看文件的导入依赖和反向依赖\r
- 分析修改某个文件后可能影响的其他文件\r
- 检测项目中的循环依赖\r
\r
---\r
\r
## 二、代码知识图谱 CodeGraph\r
\r
**AI 能理解你的代码结构和调用关系，而不仅仅是搜索文本。**\r
\r
CodeGraph 将项目的代码整体结构构建成可查询的知识图谱，让 AI 像理解知识一样理解你的代码：\r
\r
- 构建或更新项目的代码知识图谱\r
- 查看知识图谱的统计信息\r
- 按名称查找函数或方法的定义位置和签名\r
- 获取结构体或接口的完整层次结构（字段、方法、嵌入类型）\r
- 查询哪些函数调用了指定的某个函数\r
- 查询某个函数内部调用了哪些其他函数\r
- 分析修改某个函数或类型后可能影响的范围\r
- 在知识图谱中按名称搜索代码实体\r
- 查询代码实体的 Git 变更历史\r
\r
---\r
\r
## 三、文件操作\r
\r
**读写和编辑工作区内的文件，是 AI 帮你写代码的主要方式。**\r
\r
AI 可以直接在工作区中进行文件操作：\r
\r
- 将内容写入指定文件（覆盖模式，自动创建父目录）\r
- **对文件打补丁**（\`apply_patch\`）：一次调用可含多个文件与四类操作（新增 / 更新 / 删除 / 移动），更新用上下文行定位（免行号、免转义）\r
- 将文件或目录移动到新位置（也可用于重命名）\r
- 删除指定文件\r
\r
> AI 修改文件时优先「打补丁」——只写变更部分，不做整文件覆盖，误改风险更低。\r
\r
---\r
\r
## 四、命令执行\r
\r
**在工作区中运行命令，AI 也能用命令行来完成任务。**\r
\r
- 执行一条 shell 命令并等待结果返回\r
- 在后台启动一条长命令（如启动开发服务器）\r
- 读取后台进程累积的输出内容\r
- 停止正在运行的后台进程\r
- 直接执行一段代码（自动探测语言，写临时文件运行 Go / Python / Node.js 并返回结果）\r
\r
---\r
\r
## 五、网络与搜索\r
\r
**AI 可以联网获取信息或搜索资料。**\r
\r
- 抓取网页内容并提取纯文本\r
- 通过搜索引擎检索网络信息\r
\r
---\r
\r
## 六、网页验证与截图\r
\r
**AI 可以打开网页、截图并分析页面内容，用于验证前端效果。**\r
\r
- 在浏览器中打开网页，可输入文字、点击元素、检查控制台错误并截图\r
- 获取 JavaScript 渲染后的页面文本内容（适合单页应用）\r
- 截屏并保存（单工具三形态：整个桌面 / 按窗口标题 / 按坐标区域）\r
\r
---\r
\r
## 七、图像分析\r
\r
**AI 可以「看」图片并理解其中的内容。**\r
\r
- 读取图片文件并把图片本身交给模型看（识别截图文字、分析界面布局、验证渲染效果、描述图表）\r
- 路径可以没有扩展名（按文件内容识别格式）；过大的图片由运行时自动降采样\r
- 需要当前模型支持图片输入\r
\r
> 扫描型 PDF 的文字识别（OCR）由 PDF 读取工具自动处理，无需单独操作。\r
\r
---\r
\r
## 八、二进制分析\r
\r
**查看和分析二进制文件的内容，用于逆向工程或文件格式分析。**\r
\r
- 分析二进制文件的大小、类型和十六进制预览\r
- 将 Base64 编码的内容写入二进制文件\r
- 从二进制文件中提取可打印的字符串\r
- 在二进制文件中搜索指定的字节模式或文本\r
- 在二进制文件的指定位置写入字节补丁\r
- 解析可执行文件的结构（架构、入口、节区、导入导出）\r
- 计算文件的 MD5、SHA1、SHA256 哈希值\r
- 按块计算文件的香农熵（识别压缩或加密区域）\r
\r
---\r
\r
## 九、办公文档\r
\r
**读写常见的办公文档格式，包括表格、文档和 PDF。**\r
\r
- 读取 CSV 或 TSV 文件并以表格形式展示\r
- 将数据写入 CSV 或 TSV 文件\r
- 将 JSON 数组数据转为 Markdown 表格\r
- 对表格数据的数值列做统计（求和、均值、最大值等）\r
- 按文件扩展名分组统计代码行数\r
- 读取和生成 Word 文档\r
- 读取和创建 Excel 文件\r
- 提取 PDF 文件的文本内容（扫描型 PDF 自动进行 OCR 识别）\r
- 将 Markdown 文本转换为 HTML\r
\r
---\r
\r
## 十、Git 版本控制\r
\r
**在对话中完成 Git 操作，AI 通过命令执行直接使用 git。**\r
\r
- 查看工作区状态、变更内容、提交历史与某次提交的详情\r
- 逐行查看文件的最后修改人与提交信息\r
- 暂存、提交、分支的列出创建与删除、切换分支或恢复文件修改\r
- 将改动暂存起来稍后恢复（stash）、拉取与推送\r
\r
> 代码知识图谱另提供「查询代码实体的 Git 变更历史」，适合追溯某个符号的演化。\r
\r
---\r
\r
## 十一、项目知识库\r
\r
**将项目架构、模块职责和设计决策记录下来，让 AI 跨会话了解你的项目。**\r
\r
- 写入一条项目知识（如架构说明或设计决策）\r
- 读取某条项目知识的详细内容\r
- 列出知识库的所有条目概览\r
- 按关键词搜索知识库内容\r
- 删除某条项目知识\r
- 生成项目目录结构概览\r
\r
---\r
\r
## 十二、记忆系统\r
\r
**AI 可以记住你的偏好、历史决策和项目约束，跨对话持续积累。**\r
\r
- 写入一条持久记忆，AI 在后续对话中自动参考\r
- 读取某条记忆的详细内容\r
- 按关键词搜索已有记忆\r
- 列出所有历史记忆的摘要\r
- 删除一条过时的记忆\r
- 查询记忆库中的总条目数\r
\r
---\r
\r
## 十三、BUG 检测与修复\r
\r
**AI 可以自动发现代码中的问题并给出修复方案。**\r
\r
- 分析构建或测试的输出，提取错误位置和上下文\r
- 全量检测项目中的 BUG，自动运行编译和测试检查\r
- 自动检测 BUG 并生成修复方案，支持多次迭代修复\r
\r
---\r
\r
## 十四、任务与规划\r
\r
**AI 可以追踪任务进度和执行计划，确保复杂的多步骤任务有条不紊。**\r
\r
- 维护任务清单（全量替换，自动持久化到磁盘）：新增 / 更新进度状态 / 标记完成或取消\r
- 任务可声明依赖关系，形成可追踪的任务 DAG\r
- **团队编排**：把任务 DAG 升级为带质量门禁的团队计划——依赖门禁、质量契约（目标 + 验收标准）、\r
  两阶段批准（先出计划、你批准后再执行）、修复与复审自动派生，由 AI 自己按依赖顺序逐步执行\r
- 随时检查任务完成进度，并获取「下一步做什么」的建议\r
\r
---\r
\r
## 十五、技能与 MCP 管理\r
\r
**管理和扩展 AI 的能力——技能是工作流模板，MCP 是标准化的工具扩展协议。**\r
\r
- 列出所有可用的技能及其激活模式\r
- 加载某个技能的完整内容供 AI 使用\r
- 加载技能的附加资源文件\r
- 创建或更新一个技能模板\r
- 删除一个项目级技能\r
- 列出已配置的 MCP 服务器\r
- 新增或删除 MCP 服务器扩展\r
\r
---\r
\r
## 十六、市场\r
\r
**浏览和安装来自公共市场的技能和 MCP 扩展。**\r
\r
- 在市场检索可安装的 MCP 服务器或技能\r
- 从市场安装指定的扩展\r
\r
---\r
\r
## 十七、插件管理\r
\r
**管理 JS / TS 动态插件——一切皆插件，自定义和扩展 AI 的工具集。**\r
\r
插件管理是单一入口（\`cordis\`），按 \`op\` 分派：\r
\r
- \`define\` 定义一个函数形态的 JS / TS 插件（支持 \`apply(ctx, config)\` 注入服务、timer 定时器、跨 goroutine 执行锁；TS 由内置编译器转译，无需 Node.js）\r
- \`inspect\` 查看插件运行时诊断（工具归属、版本链、源码诊断）与插件自检摘要\r
- \`run\` / \`stop\` 装载或停止插件（goja 沙箱求值并 apply，可传配置）\r
- \`services\` 列出宿主服务与方法签名；\`query\` 按协议精确查询（平台 / provider / method）\r
- \`undefine\` 撤销一个已定义的插件\r
\r
> 写插件前先用 \`services\` / \`query\` 查精确签名，不要凭记忆臆测 API。\r
\r
---\r
\r
## 十八、工具集管理\r
\r
**按项目需求动态组合工具集，固化/导出/导入，构建处理本身也插件化。**\r
\r
- 分析项目结构与需求，动态组合所需工具并创建工具集插件（固化到工作区 \`.pair/toolsets/\`）\r
- 列出当前项目可用的工具集\r
- 查看某个工具集的详细内容\r
- 导出工具集为 JSON（可提交 Git / 发布市场）\r
- 从 JSON 或文件导入工具集（project 工作区级 / user 全局级）\r
- 移除不再需要的工具集\r
- **创造模式**：用一句需求（\`/创造 需求描述\`）自主创建「场景」工具集——先盘点当前可用能力\r
  （插件与内置工具组），再组合成清单并固化到安装目录 \`.pair/toolsets/<场景名>.json\`；\r
  场景 = 工具面白名单，在对话面板选择后该场景内的工具才对 AI 可见（支持中文场景名）\r
\r
---\r
\r
## 十九、其他工具\r
\r
**辅助性工具，在特定场景下帮助 AI 更好地与你协作。**\r
\r
- **用户提问** — 遇到关键决策点或需求歧义时向你提问（支持纯文本 / 单选 / 多选 / 选项内自定义输入）\r
- **智能资源** — 统一检索经验胶囊、技能基因、记忆、项目知识库、技能与 MCP 服务器；可保存修复经验胶囊供后续复用\r
- **运行统计** — 查看各工具的调用次数、成功率与最近调用记录，识别高频失败工具\r
- **历史检索** — 按关键词搜索已完成对话（标题 / 摘要 / 标签 / 关键点）\r
- **代码统计** — 按扩展名或目录分组统计代码行数（总行 / 代码行 / 注释行 / 空行），自动跳过依赖与构建产物\r
`,ro=`# 快捷键参考\r
\r
PairCode IDE 提供了丰富的快捷键，帮助你更高效地编写代码和管理项目。以下按功能分类列出所有可用的快捷键。\r
\r
---\r
\r
## 一、通用操作\r
\r
**控制 IDE 界面面板的显示与隐藏，快速切换工作布局。**\r
\r
| 快捷键 | 功能 | 适用范围 |\r
|--------|------|----------|\r
| Ctrl+B | 切换侧栏（文件浏览器）显示/隐藏 | 全局 |\r
| Ctrl+\\\` | 切换终端面板显示/隐藏 | 全局 |\r
| Ctrl+K | 专注模式：隐藏所有面板，聚焦代码编辑区 | 全局 |\r
| Ctrl+Shift+C | 切换对话面板显示/隐藏 | 全局 |\r
| Escape | 关闭当前模态框或菜单 | 全局 |\r
\r
## 二、文件编辑\r
\r
**编辑器中常用的编辑操作，与主流编辑器保持一致。**\r
\r
| 快捷键 | 功能 | 适用范围 |\r
|--------|------|----------|\r
| Ctrl+S | 保存当前文件 | 编辑器 |\r
| Ctrl+Z | 撤销操作 | 编辑器 |\r
| Ctrl+Shift+Z / Ctrl+Y | 重做操作 | 编辑器 |\r
| Ctrl+X | 剪切选中的内容 | 编辑器 |\r
| Ctrl+C | 复制选中的内容 | 编辑器 |\r
| Ctrl+V | 粘贴剪贴板内容 | 编辑器 |\r
| Ctrl+A | 全选当前文件内容 | 编辑器 |\r
| Ctrl+F | 在当前文件中搜索 | 编辑器 |\r
| Ctrl+H | 在当前文件中查找替换 | 编辑器 |\r
| Ctrl+P | 按文件名快速打开文件 | 编辑器 |\r
\r
## 三、导航与视图\r
\r
**在不同功能面板之间快速切换，无需鼠标操作。**\r
\r
| 快捷键 | 功能 | 适用范围 |\r
|--------|------|----------|\r
| Ctrl+Shift+E | 切换到文件浏览器 | 全局 |\r
| Ctrl+Shift+F | 全局搜索（在工作区中搜内容） | 全局 |\r
| Ctrl+Shift+T | 打开对话面板 | 全局 |\r
| F2 | 重命名选中的文件或文件夹 | 文件树 |\r
| Ctrl+Tab | 在打开的文件标签页之间切换 | 编辑器 |\r
| Ctrl+W | 关闭当前文件标签页 | 编辑器 |\r
\r
## 四、对话面板\r
\r
**AI 对话输入区的快捷操作。**\r
\r
| 快捷键 | 功能 | 适用范围 |\r
|--------|------|----------|\r
| Enter | 发送消息给 AI | 对话面板 |\r
| Shift+Enter | 换行（多行输入） | 对话面板 |\r
| Ctrl+Up | 切换到上一条对话 | 对话面板 |\r
| Ctrl+Down | 切换到下一条对话 | 对话面板 |\r
\r
## 五、终端\r
\r
**终端面板的操作快捷键。**\r
\r
| 快捷键 | 功能 | 适用范围 |\r
|--------|------|----------|\r
| Ctrl+\\\` | 打开/关闭终端面板 | 全局 |\r
| Ctrl+Shift+\\\` | 新建终端标签页 | 终端 |\r
| Ctrl+W | 关闭当前终端标签页 | 终端 |\r
| Ctrl+C | 中断当前正在运行的命令 | 终端 |\r
\r
## 六、多标签页导航\r
\r
| 快捷键 | 功能 | 适用范围 |\r
|--------|------|----------|\r
| Ctrl+Tab | 切换到下一个文件标签页 | 编辑器 |\r
| Ctrl+Shift+Tab | 切换到上一个文件标签页 | 编辑器 |\r
| Ctrl+PageUp | 切换到上一个文件标签页 | 编辑器 |\r
| Ctrl+PageDown | 切换到下一个文件标签页 | 编辑器 |\r
| Ctrl+W | 关闭当前文件标签页 | 编辑器 |\r
`,oo=`# 常见问题\r
\r
## PairCode IDE 是什么？\r
\r
PairCode IDE 是一款 AI 原生的纯 Web 集成开发环境。与传统 IDE 不同，你只需用浏览器打开，在对话面板中用自然语言描述需求，AI 就能理解你的意图，自动完成代码编写、文件操作、命令执行等工作——让编程从手工操作转变为对话驱动。\r
\r
## 需要安装桌面客户端吗？\r
\r
不需要。PairCode IDE 是纯 Web 应用，你只需启动后台服务，然后用浏览器（推荐 Chrome、Edge、Firefox）访问即可。所有界面在浏览器中渲染，无需安装任何桌面客户端。\r
\r
## AI 能做什么？\r
\r
AI 可以读写和编辑你的代码文件、在工作区中执行命令、搜索和浏览项目结构、用 Git 管理版本、处理图片和办公文档、查阅网络资料，还能截图并验证网页效果。基本上，日常开发中你能做的事情，AI 都可以帮你完成。\r
\r
## 如何让 AI 执行命令？\r
\r
你可以在对话中直接告诉 AI 需要运行什么命令，例如"运行测试"或"启动项目"。AI 会自动在终端中执行并返回结果输出。涉及文件写入和命令执行的操作会先请求你的确认。\r
\r
## 文件保存在哪里？\r
\r
所有文件都保存在你本地的工作区目录中。PairCode IDE 直接读写你本地磁盘上的文件，不经过云端存储。你可以在文件浏览器中看到完整的项目目录结构，用系统的文件管理器也能找到它们。\r
\r
## 如何切换 AI 模型？\r
\r
在设置面板的"AI 模型"选项卡中，你可以选择不同的 AI 服务商和模型。支持接入 OpenAI、Claude 等多种主流模型后端。你可以为执行任务和制定规划分别配置不同的模型。\r
\r
## 如何安装更多技能？\r
\r
在市场中可以浏览和安装社区贡献的技能模板、MCP 扩展和工具集插件。技能是可复用的工作流程模板，MCP 扩展可以给 AI 添加新的能力，工具集是按项目需求组合的插件包（可通过 \`toolset_build\` 动态构建并固化到工作区）。打开市场面板，搜索你需要的功能，一键即可安装使用。\r
\r
## 对话历史会丢失吗？\r
\r
不会。每次对话都会自动保存在本地磁盘上，你可以随时在对话列表中查看历史记录、继续之前的对话或开启新话题。切换工作区时，各项目的对话会自动隔离，互不干扰。\r
\r
## 如何保护隐私？\r
\r
所有操作都在你的本地计算机上执行，代码和对话内容不会发送到外部服务器（AI 模型调用除外，你可以选择使用本地模型避免数据外出）。Web 服务监听所有接口（0.0.0.0），局域网内其他设备可通过本机 IP 访问；请勿将端口暴露到公网。文件操作限定在工作区范围内。\r
\r
## 页面刷新后数据还在吗？\r
\r
大部分数据都会保留：\r
- **对话历史** — 自动持久化到磁盘，刷新后完整恢复\r
- **打开的文件** — 刷新后自动重新打开\r
- **工作区状态** — 侧栏位置、面板大小等布局信息保存在浏览器中\r
- **设置** — 主题、AI 模型配置等设置持久化到磁盘\r
\r
## 编辑器里的代码没有高亮怎么办？\r
\r
编辑器会根据文件扩展名自动切换语言模式。如果文件扩展名不常见，代码高亮可能无法自动识别。建议确认文件扩展名是否被支持，或使用常见的扩展名保存文件。\r
\r
## 什么是自主模式？和普通对话有什么区别？\r
\r
**普通模式**：你发一条指令，AI 执行并回复，然后等待你下一条指令。\r
\r
**自主模式**：你交给 AI 一个复杂任务（如"修复所有编译错误"），AI 会自动分解任务、逐个执行、迭代验证，直到全部完成。你不需要逐条发指令，只需在关键节点确认即可。\r
\r
单个执行段有「步数预算」和「工具调用预算」（默认各 120，可在设置 → Agent 调整）：任一达到上限，AI 会自动结束这一段并接着开下一段继续做（对话上下文保留），长任务不会因为预算用尽而停下，也不需要你手动催它继续。\r
\r
## 能让 AI 访问我的私有 API 吗？\r
\r
可以通过 MCP（模型上下文协议）扩展来实现。在设置中添加自定义 MCP 服务器，AI 就能通过它访问你的私有 API、数据库或内部服务。\r
\r
## 遇到问题怎么办？\r
\r
你可以查看帮助菜单中的文档中心，里面有功能介绍、API 文档、工具文档和快捷键参考等详细资料。如果问题仍然无法解决，可以在对话中向 AI 描述你遇到的问题，它会尽力协助排查。\r
`,lo=`# 快速开始\r
\r
欢迎使用 PairCode IDE！以下指南将带你快速上手，从打开工作区到用 AI 写代码，只需几分钟。\r
\r
---\r
\r
## 打开 IDE\r
\r
PairCode IDE 是一个纯 Web 应用，启动后台服务后，直接在浏览器中访问对应地址即可使用。所有界面在浏览器中渲染，无需安装任何桌面客户端。\r
\r
> 建议使用 Chrome、Edge 或 Firefox 等现代浏览器获得最佳体验。\r
\r
---\r
\r
## 设置工作区\r
\r
工作区是 IDE 操作的基础——所有文件操作、AI 对话和命令执行都将在这个目录范围内进行。\r
\r
1. 点击左侧活动栏顶部的**文件图标**打开文件浏览器\r
2. 在文件浏览器顶部输入你的项目文件夹的完整路径\r
3. 按回车确认，IDE 会自动加载该目录下的所有文件和子目录\r
\r
你也可以同时添加多个文件夹到同一个工作区中，方便跨目录浏览和管理代码。\r
\r
---\r
\r
## 与 AI 对话\r
\r
右侧的**对话面板**是 PairCode IDE 的核心交互界面。你只需用自然语言描述需求，AI 就能理解并执行。\r
\r
直接在输入框中输入你的需求，例如：\r
\r
- "创建一个 Go 文件，实现一个返回 JSON 的 HTTP 服务"\r
- "帮我优化这个函数，加上错误处理和参数校验"\r
- "搜索项目中所有调用了 Post 的地方"\r
- "运行项目中的所有测试，并告诉我哪些失败了"\r
- "把我的改动提交到 Git"\r
\r
按 Enter 发送消息，Shift+Enter 换行。AI 会实时流式展示它的思考过程、工具调用和结果输出。\r
\r
### 常用对话技巧\r
\r
| 技巧 | 说明 |\r
|------|------|\r
| **明确具体** | 越具体，AI 理解越准确。如"写一个函数"不如"写一个读取 JSON 配置文件的函数" |\r
| **分步沟通** | 复杂任务可以分步骤告诉 AI，先分析，再重构 |\r
| **提供上下文** | 在对话中粘贴错误信息或代码片段，AI 能给出更精准的修复方案 |\r
| **使用反馈** | 如果 AI 输出不满意，直接指出问题，AI 会调整方案重新尝试 |\r
\r
---\r
\r
## 编辑代码\r
\r
AI 生成的代码会直接写入到文件中。你也可以在编辑器中手动查看和修改代码：\r
\r
- **多标签页** — 同时打开多个文件，在标签栏切换\r
- **语法高亮** — 支持 Go、TypeScript、Python、Rust、Java、Vue 等主流语言\r
- **代码折叠** — 折叠函数和代码块，聚焦关键逻辑\r
- **Ctrl+S** — 保存当前文件的修改\r
\r
你还可以在编辑器中查看二进制文件的十六进制内容，或直接预览图片文件。\r
\r
---\r
\r
## 运行与调试\r
\r
### 使用内置终端\r
\r
按 Ctrl+\\\` 打开 IDE 底部的终端面板，可以直接在工作区目录下运行命令。支持多标签页，方便在不同任务间切换。\r
\r
### 让 AI 帮你运行\r
\r
你也可以直接在对话中告诉 AI："运行项目并告诉我结果"或"执行 npm test"。AI 会自动在终端中执行命令、读取输出，并根据结果决定下一步操作。\r
\r
---\r
\r
## 版本控制\r
\r
Git 操作完全融入 AI 对话流程。你用自然语言就能完成所有 Git 操作：\r
\r
- "查看当前仓库状态"\r
- "暂存所有修改并提交"\r
- "创建一个新分支并切换过去"\r
- "从远程拉取最新代码"\r
\r
你也可以通过左侧 Git 面板查看文件变更的详细对比，逐行确认每次改动的具体内容。\r
\r
---\r
\r
## 个性化设置\r
\r
点击活动栏的**齿轮图标**打开设置面板，你可以：\r
\r
- **AI 模型** — 选择不同的 AI 服务商和模型\r
- **外观主题** — 切换暗色、白色、暖色和暗夜紫四套主题\r
- **工作区管理** — 查看和切换最近使用的工作区\r
- **系统指令** — 自定义 AI 的行为指导原则\r
- **Agent 预算** — 调整单段步数 / 工具调用预算与自动续跑段数上限，控制长任务的执行节奏\r
\r
---\r
\r
## 探索更多\r
\r
PairCode IDE 还有更多强大功能等待你探索。欢迎查阅帮助文档中的其他章节：\r
\r
- **功能介绍** — 了解所有功能模块的详细说明\r
- **工具文档** — 查看 AI 可使用的全部内置能力\r
- **快捷键参考** — 常用快捷键一览\r
- **API 文档** — 后端 HTTP API 接口说明\r
- **常见问题** — 常见问题与解答\r
`,ao='# 更新日志\n\n> 所有 PairCode IDE 的重要变更均记录在此文件中。\n\n---\n\n## 1.6.8 — 2026-09-25\n\n> 本版是 **v1.6.7 发布之后的修复与清理批次**（含后端「生成参数单源化」）：泛化死代码扫描\n> （`chat-utils.js` 整文件删除、14 处死绑定、6 个零调用 API）、编辑器右键选中链路修复与\n> 监听器累积、任务进度实时同步、聊天工具行排版、「上次运行 / 执行中…」冗余状态条删除。\n> 全部条目均由**真实 UI 实测**驱动，并新增右键链路回归守卫脚本。\n\n### 变更 / 改进\n\n- **生成参数单源化：设置面板「生成参数」页移除** —— 生成参数（温度 / 思考档位 / 最大输出 /\n  上下文窗口）此前有两处入口：设置面板的「生成参数」页（全局默认，存 `pluginSettings.generation`）\n  与「服务商」页（服务商级 + 模型级）。前者处于装配链**最低优先级**（服务商级一配即被覆盖），\n  在设置面板改了温度/输出/窗口却「改了不生效」；同时服务商级当时**没有思考档位字段**，\n  反倒让它成了思考档位的唯一全局入口。现收敛为**唯一来源 = 服务商配置**：\n  · 「服务商」页服务商级新增**默认思考档位**（`models.json` 服务商条目 `thinkingMode`）；\n    装配器「服务商配置为准」段一并纳入思考档位（模型级 `modelParams[模型].thinkingMode`\n    > 服务商级 `thinkingMode`）——此前服务商面板里设的思考档位不生效；\n  · 「生成参数」设置页删除；旧值（`pluginSettings.generation` + settings 顶层旧字段）由\n    `core.MigrateGenerationToProvider()` 一次性迁入**激活配置对应服务商**的服务商级字段\n    （只补空、不覆盖服务商面板里已设的值；激活配置无可用服务商时保留旧值待下次启动重试）；\n  · 装配器兜底改为 `GEN_DEFAULTS` **机制常量**（不是配置面、无 UI 入口，仅在服务商级/模型级/\n    AI 配置都未配置时使用）；「保存 AI 配置」快照的生成参数改从**激活配置**取（此前取全局段），\n    源头移除后仍保持「整套配置快照」语义。\n\n### 修复\n\n- **聊天里的工具调用行跑到最右侧、状态做成胶囊 tag** —— 用户反馈工具行「整体像右侧标签」：\n  `.tr-pill` 用 `margin-left:auto` 把状态推出行尾、工具名固定 `width:240px`，中间留大片空白。\n  现改为**消息流内左对齐成组**：工具名自适应（`max-width:220px`）+ 运行中 spinner +\n  结果摘要紧随其后（`.tr-status`，**无背景无圆角**、`max-width:62%` 可省略，仅以文字色区分\n  成功/错误/运行中）+ chevron 紧随内容；行本体沿用设计稿 th83/th90 的行式（h32 / bg=surface-2 /\n  r8 / border），整行可点与展开区（参数 / 结果 / 命令 / 输出）全部保留。\n- **右栏「任务进度」不实时同步** —— 此前只在挂载 / 切会话时 `GET /api/tasks`，agent 运行中调用\n  `update_tasks` 改了清单 UI 不刷新。现由 `agent-events.js` 在**任务类工具执行完成\n  （tool_result）**与**回合结束（done）**时广播 `paircode:tasks-changed`，`StatsRail` 据此重拉\n  当前会话任务（150ms 防抖；事件带 convId 时只刷当前会话）。★ 不在 `tool_call` 触发 —— 那是\n  「即将执行」，任务尚未落盘会读到旧清单；done 再广播一次作兜底（工具异常未发 tool_result 时\n  仍能对齐）。全程零轮询。\n- **聊天面板顶部的「上次运行 / 执行中…」状态条已删除** —— 用户反馈「运行统计本身已有状态，\n  这条是否已不需要」。核实后确认**冗余并整条删除**（`.phase-bar`，消息区最顶部）：\n  ① 它与右栏 StatsRail「运行统计」卡**同源**（同一份 `state.runStatsByConv` ←\n  `GET /api/conversations/{id}/run-stats`），而右栏摘要已给「运行中 / 已完成 · 耗时 · 输出速度」\n  + 步数 / 工具调用 / LLM 调用明细；② 它的「阶段」分支无生产方 —— Go 侧 `EventPhase` 常量\n  **只有定义、零发送点**，JS / 插件侧也无任何 `phase` 事件发送点 → `state.phaseByConv` 恒空；\n  ③ 进度条按「已耗时 ÷ 60min、封顶 95%」**估算**，非真实进度，易被误读为「快完成 / 卡住」；\n  ④ 设计稿 `shell-midnight` th130 子树本无该节点。删除后运行态可见性**不受影响**：消息流\n  「思考中...」banner + 输入区停止按钮（发送 ↔ 停止切换）+ 工具行 spinner + 右栏运行统计。\n- **清理只服务该状态条的死代码** —— `RightPanel.vue` 移除 `phaseText` / `runBarTitle` /\n  `phaseProgress` / `phaseIcon()` / `currentPhase` / `agentRunningConv` / `runStatsVisible` /\n  `hasRunStats` / `toggleRunStats` / `runTick` 计时器与 `phaseTimer`、`onPhaseChange` /\n  `onPhaseEnd` 钩子，以及 CSS `.phase-bar*` / `.phase-stats*` / `.phs-*` / `.pst-text`（模板侧\n  已无引用）；另移除构建期告警的未使用导入 `rightPanelWidth`。★ 保留 `fetchRunStats` 调用\n  （会话切换 / 续跑时拉取）—— 右栏运行统计卡依赖它写入状态，删 UI 不等于删数据链路。\n  同步订正两处指回已删 UI 的注释（`ui-state.js` 的 `runStatsCollapsed` 标注「已无 UI 消费者，\n  字段仅为兼容旧持久化偏好」；`StatsRail.vue` 运行统计卡头注改为「数据源唯一，本卡」），并给\n  三个仍断言 `.phase-bar` 的旧 CDP 脚本（`cdp-verify-conv-tasks` / `cdp-verify-runstats-backend`\n  / `cdp-verify-ws-gate-fallback`）加失效标注 —— 避免后人把它们当回归基线而误判失败。\n\n- **编辑器右键菜单丢选中片段（功能降级）+ 监听器累积** —— `CodeEditor.vue` 声明的事件名是\n  `contextmenu-selection` 而实际发的是 `emit(\'contextmenu\')`，父组件 `EditorArea.vue` 的\n  `@contextmenu` 因此被 Vue 当作**原生 DOM 事件**透传到根元素：handler 收到裸 `MouseEvent`\n  （无 `hasSelection` / `text` / `lineStart` / `lineEnd`）→ 有选中文本时菜单**恒走「无选中」\n  分支**，「AI: 添加到对话」按**整文件**加入（实测载荷 `{type:\'file\'}`，期望\n  `{type:\'selection\'}` + 行号 + 内容）；同时 `createEditor()` 内 `addEventListener(\'contextmenu\')`\n  用匿名函数且从不移除，而该方法会因切换文件 / 改字号被反复调用（wrapper 元素不重建）——\n  实测改 4 次字号后监听器 **1 → 5** 累积，一次右键被处理多次且 `ContextMenu.show` 的\n  `resolvePromise` 被覆盖（先到的 Promise 永久 pending）。现统一为 **emit 通道**（声明名与\n  发送名一致 → 不再 fallthrough）+ 句柄引用化**幂等注册**（另在 `onBeforeUnmount` 清理）。\n  实测：监听器恒为 1、菜单走「有选中」分支、载荷 `type=selection` 且带行号与内容。\n- **清理泛化扫描确认的死代码** —— `chat-utils.js` **整文件删除**（159 行；头注声称「供\n  RightPanel.vue 使用」但全仓零引用，两个导出 `useMessageCombos` / `isSystemMsg` 亦零引用）；\n  `RightPanel.vue` 摘除 **15 个死绑定**（`toggleRight` / `toggleFocus` / `toolsetLabel` /\n  `convListWidth` / `convList` / `reviewBtnLabel` / `showNudge` / `pendingAskCallId` /\n  `dismissNextSteps` / `segMode` / `wsTokenStats` / `convCtxStats` / `deleteConv` /\n  `handleTaskTool` / `currentNudge`——模板段命中数逐项为 0、无 `defineExpose` 暴露、跨文件零引用）\n  并移除随之无用的 `setFocusMode` 导入；`api.js` 删除 6 个零调用方法（`isWebSocketOpen` /\n  `answerChat` / `approveChat` / `chatCompact` / `getMessagesCount` / `getUIBoot`，同步从\n  `export default` 表移除——其中 `approveChat` 封装缺 `reply` 字段，真实调用点一直直发\n  `/chat/approve` 且带 `reply`）；`model-parsers.js` 删 `partsBbox` / `partsTriCount`，\n  `ui-state.js` 删 `showQuickSwitcher`。\n- **事件分发链补兜底（不再静默丢弃）** —— `agent-events.js` 的 `processAgentEvent` 分发链原本\n  没有 `else` 分支：有生产方但前端未接的事件无声消失（典型是 Go 侧 `OnToolUpdate` →\n  `EventToolUpdate` 携带的工具执行中间结果）。现加兜底分支累计类型计数 + 首次 `console.debug`\n  提示，并导出 `getUnconsumedEventTypes()` 供排查。★ 该通道「接通（需设计流式工具输出展示位）\n  还是下线 Go 侧 emit」属产品决策，代码中已就地标注。\n\n### 文档\n\n- 另给 3 个断言已失效选择器的旧 CDP 脚本（`cdp-verify-toolset-tab` / `-toolset-panel` /\n  `-chat-input`：`.tset-item` / `.toolset-card` / `.ts-header` / `.ts-divider` / `.tp-grabber` /\n  `sp-trigger` / `sp-pop` 均已随 UI 改版消失）加 ⚠️ 失效标注，避免后人误当回归基线。\n\n### 验证\n\n- **编辑器右键链路回归守卫**（`scripts/cdp-verify-ctxmenu-chain.cjs`，独立实例 9099 +\n  headless Chrome）：修复前 3 项 FAIL（监听器 1→5 累积 / 菜单呈无选中版 / 载荷 `type=file`），\n  修复后 **6/6 PASS**，控制台 0 error、0 warning。\n- **生成参数单源化实测**（独立实例 `WEB_PORT=9098` + 临时 install-dir，9090 未动）：以真实旧配置\n  启动（`pluginSettings.generation` = temperature 0.3 / thinkingMode high / maxTokens 131072 /\n  contextMaxTokens 1000000；激活配置 `ds-vision` → 服务商 deepseek）——启动日志\n  「已把生成参数旧值迁入服务商 "deepseek" 的服务商级配置」，`models.json` 的 deepseek 条目获得\n  四项值、`settings.json` 的 `generation` 注册段与顶层旧字段清空。数据面：`GET /api/settings`\n  的 `schemas` **不含** `generation`；`GET /api/models` 的 `providerThinkingModes.deepseek = "high"`。\n  装配链（假 Key 触发，看 `[provider] global 装配结果`）：模型级 `thinkingMode=max` 覆盖服务商级\n  `high`；清掉模型级后回落为**服务商级 `high`**（本次新增能力）。CDP 交互实测 **14/14 PASS**：\n  设置分类列表无「生成参数」、服务商编辑表单含「默认思考档位」且当前值 = 迁入的 `high`、\n  页面 0 异常。\n- **工具调用行展示 + 任务进度实时同步实测**（独立实例 `WEB_PORT=9098` + 临时 install-dir，\n  9090 未动；CDP 9223）：几何探针 `.tr-pill` **不存在**，工具行 rect left=328 / width=793，\n  工具名 24px（357→381），结果摘要紧随名称**左对齐同行**；截图视觉确认左对齐成组、无右侧胶囊、\n  成功 / 错误 / 运行中三色可辨。右栏实时性走**真实 UI 发送路径**（`.chat-input`\n  contenteditable 填文本 + `.send-btn` 点击，非 API 直发 —— 直发会绕过前端运行态就观察不到\n  「运行中」窗口）驱动真实 agent 调用 `update_tasks`：任务卡于 **+1585ms** 出现，而 agent\n  **+5114ms** 才结束（此刻 `chatLoading` / 运行态均为 true）→ 证明是**运行中实时同步**而非\n  结束后补刷；最终徽标 3/3、控制台 0 错误，**10/10 PASS**（另有静态链路含「无事件不刷新 /\n  有事件即刷新」对照 **12/12**）。因实例内既有 key 均为无效假 key（401），真实链路改用本地\n  mock LLM（内置延迟）驱动两轮 `tool_calls`；脚本自动清理临时会话与任务文件。\n- 任务相关回归：`go test ./internal/agent/ -run \'TestUpdateTasksBindsConvID*|TestUpdateTasksNoConv*|\n  TestUpdateTasksRuntimeRoot|TestUseTaskManagerPerRoot|TestCheckFinalReadiness_NoTodos\'`\n  **6/6 PASS**。产物一致性：`ui-right-panel.js` / `.css` 与壳 `index-G_Ay0AZC.js` + `index.html`\n  的**真源 vs `bin/.pair` 镜像 md5 全等**，旧产物 `index-C6dXkUaW.js` 已无残留。\n- **「上次运行 / 执行中…」状态条删除实测**（独立实例 `WEB_PORT=9099` + 临时 install-dir，\n  9090 未动；CDP 9223）：真实 UI 发送路径（`.chat-input` contenteditable + `.send-btn`）驱动\n  真实 agent —— 运行中 `.phase-bar` **不存在**，同时 `.msg-loading-banner`「思考中...」与\n  `.stop-btn` 均在位；回合结束后该会话已有运行统计（`steps=2 / toolCalls=1 / durationMs=4846`，\n  即旧实现在此**必然**显示「上次运行」），而 `.phase-bar` 仍**不存在**、右栏摘要为\n  「已完成 · 4s · 5.3 t/s」；`.chat-area` 首个可见子元素 = 任务横幅、与 `.chat-messages`\n  间隙 **0px**（无残留空条），控制台 0 错误 —— **13/13 PASS**（空闲 / 运行 / 结束三态 + 几何\n  探针）。截图视觉复核：消息区顶部无橙色条、「思考中...」与红色停止按钮正常、布局无错位。\n\n---\n\n## 1.6.7 — 2026-09-25\n\n> 本版以**长会话加载性能**为主线（四轮专项，全部由真实会话实测数据驱动）：会话切换时\n> 50 条消息的响应体从 **19.57MB 降到 3.58MB（−81.7%）**、首屏白屏从 **15.7s 降到 2.7s**、\n> git 状态查询从 **1.2s 降到 1~4ms** —— 而首屏渲染节点数与改造前**逐项一致**、控制台零\n> 错误，因为削掉的全部是「折叠态不消费」的字段（展开时按需取回全文），可见交互未变。\n> 同期落地**主题 v2（8 套主题插件化 + 设置面板主题画廊）**，并修复场景胶囊、选择器\n> 当前值、滚动条、监督者面板遮挡等一批实测缺陷。\n\n### 变更 / 改进\n\n- **首屏渲染：折叠优先 + 按需挂载（性能第 1 轮）** —— 会话切换白屏 13 秒的真凶是**折叠标记\n  注入时机晚于首次渲染**：`switchConv` 先把消息交给 Vue 渲染（此时全展开），之后才 `await`\n  任务与监督者数据，最后才折叠 → 实测 t=4.4s 已建 **47025 个 DOM 节点**（1547 个 Markdown\n  渲染器），t=15.7s 才降到 1674（白渲染 13 秒再整批丢弃）。现改为 ① `apiLoadAndBuildConv`\n  返回前就在**数据源头**注入折叠标记（幂等）；② `loadAutopilotRounds` 由 `await` 改后台执行\n  （不再推迟首屏滚底）；③ `AutopilotPanel` 折叠区 `v-if` 懒挂载 + 轨迹尾部 200 条；\n  ④ `MarkdownRenderer` 的 mermaid **改运行时按需注入** —— 此前 `import mermaid` 让\n  mermaid.min.js（3.6MB）被 rollup 打进**每一个**引用它的区域包（`ui-right-panel.js` 3.47MB /\n  `ui-editor.js` 4.39MB，首屏 JS 合计 7.86MB），而区域包是 iife 单包构建、动态 import 会被\n  内联，无法靠 code-splitting 拆分。**收益**：首屏 DOM 47025 → **1412**；达到最终态\n  15.7s → **2.7s**；`.chat-messages` 节点 46238 → 879；Markdown 渲染器 1547 → 26；\n  CPU program 11082ms → 1865ms、GC 3456ms → 599ms。\n- **接口瘦身与重复请求消除（性能第 2 轮）** —— `/api/conversations/<id>/messages` 省略前端\n  不消费的重字段（`message.reasoning_content` 与 `segments.thinking` 同内容、\n  `message.tool_calls` 与 `segments.tool_call` 重复），响应 **−38.3%**；\n  `/api/autopilot/rounds` 的 trace 全文按折叠态消费口径裁剪，**−66.8%**（条数不变）；\n  前端 `apiGet` 层新增 **GET 并发合并**（同 URL 在途请求共享，订阅窗口内的重复拉取合并为\n  一次），实测 messages ×2→×1、rounds ×5→×2。\n- **折叠段惰性加载（性能第 3 轮）** —— 实测证明瓶颈是**单条消息内的超长段**而非条数\n  （前 30 条已占满全部体量，缩小页长收益为 0）：`thinking.content` 与 `tool_call.argsRaw`\n  在折叠态**完全不消费**，故只回前 400 字符预览 + `_trunc`（原始字符数）标记，展开时经\n  新增接口 `GET /api/conversations/<id>/messages/segment?idx=&seg=` 按需取回全文\n  （毫秒级、在途合并、失败静默降级保留预览）。消息摘要只用正文前 60 字符与工具计数，\n  完全不受影响。\n- **`tool_call.result` 段级裁断 + 错误标志预计算（性能第 4 轮）** —— result 是剩余体积的\n  最大头（可裁 1069 段 / 2.79M 字符）。折叠行只用它做三件事：胶囊文案（前 120 字符）、\n  未知工具摘要（前 80 字符）、错误正则判色（**跑全文**）—— 前两者 400 字符预览足够，\n  后者改为**后端在裁断前按全文预计算 `_err`**，前端 `isToolErr()` 优先取该标志、未裁断时\n  照旧跑正则。★ 这不是过度设计：实测该会话有 **156 段**的错误关键词只落在 400 字符之后，\n  只留预览会让这些胶囊的错误色变成成功色。`finish_task` 的 result **不裁**（它会被转成\n  正文 content 段）。**收益**：消息响应体 6.93MB → **3.58MB**（再降 45.8%）。\n- **`/api/git/status` 短 TTL 缓存** —— 该接口每次要跑 4~5 个 git 子进程（实测 1.2~1.4s），\n  而状态栏每 15s 轮询一次。现加 **5s TTL**（按工作区目录分键）并支持 `?refresh=1` 强制实时；\n  `GitPanel` 的每个写操作后刷新一律带 `?refresh=1`（否则用户暂存/提交后会看到**操作前**的\n  文件列表）。实测命中 **1~4ms（约 290×）**，TTL 过期后恢复真实执行，命中与实时返回内容\n  逐字节一致。\n- **主题 v2：8 套主题插件化 + 主题画廊** —— 主题 CSS 原本硬编码在壳 `index.html`（8 套主题块，\n  678 行），换主题必须改壳。现迁入 `ui-appearance` 插件（`theme-<id>.css`，每份 54 个令牌），\n  由 client 半用 `MutationObserver` 监听 `<html>` 的 class 注入 `<link>`；壳仅保留一份 `:root`\n  基准令牌兜底（插件未装配时回落到 Midnight，界面不裸奔）。设置面板的 `theme` 由下拉框\n  改为 **8 项缩略色卡画廊**（值仍是 string、绑定未变，只换呈现）。旧 id（dark/night/light/warm）\n  经别名表**行为不变**。\n- **插件设置 schema 支持色板/画廊字段** —— 注册字段新增 `swatches`（每项\n  `{value,label,scheme,colors}`），解析侧校验必需键（`value` 为空即丢弃，防止数组里混入\n  非选项对象而渲染出「坏卡」）。\n- **应用背景能力** —— 新增背景来源 / 图片 / 不透明度 / 模糊四个设置项（`ui-appearance`）。\n- **标题栏「帮助」菜单移至左侧**（按用户指令调整 `ui-titlebar` 结构）。\n- **任务进度卡展示完整列表** —— 右栏「任务进度」卡改为**卡内滚动**展示全部任务，不再\n  截断行数或用「还有 N 项…」摘要（按用户明确指令）。\n\n### 修复\n\n- **应用内「API 文档 / 更新日志」与主壳是两条产物链** —— 二者经 `?raw` 打进 **UI 区域包**\n  （ui-modals），只重建主壳不会生效；本版发版时**全量重建 15 个区域包**。\n- **会话「场景」胶囊** —— 此前是写死的假数据，现改为真实**会话级工具集**，并修复切换对话\n  瞬间显示「上一个对话的场景」的窗口期。\n- **下拉选择器不显示当前值** —— 原生 `<select>` 的 `v-model` 值不在选项集合时\n  `selectedIndex` 回落为 -1，浏览器会显示成第一个 disabled 占位项；模型/场景选择器现按值\n  归一化后匹配。\n- **滚动条样式不生效** —— `scrollbar-color` 是继承属性且会**废掉** `::-webkit-scrollbar`\n  伪元素（实测写了 4px 仍显示平台原生 17px）；另有两处遗漏容器一并修正。\n- **监督者面板展开时被输入框遮挡** —— 根因是高度几何算术（头部 32 + body 写死 300 + 边距 4\n  = 336px）而非 z-index；改为按内容自适应 + 关键信息优先（评判 / 下一步 / 证据），并修正\n  滚动锚点。\n- **终端配色不跟随主题** —— `@xterm/xterm@6` 已无 `setOption()`，原 `watch(state.theme)`\n  里的调用静默失败，改用 `options` 赋值。\n- **主题画廊无任何卡片高亮** —— 默认值与存量配置是旧 id 而画廊按新 id 判定选中态；新增\n  `ThemeIDAliases` 与 `NormalizeThemeID()`，加载时归一化（无配置文件时也生效），幂等。\n- **外观页背景设置项消失** —— 插件 schema 里 4 个背景字段被误插进 `swatches` 数组内部\n  （语法合法、预检不报错），移回 `fields` 顶层；解析侧同时加必需键校验。\n\n### 文档\n\n- 应用内「更新日志 / API 文档」同步本版内容；`/api/system/info` 版本示例更新为 `v1.6.7`。\n\n### 验证\n\n- `go build ./cmd/companion ./internal/agent` 与同范围 `go vet` 通过（`go build ./...` 因\n  `embedding_onnx.go` 引用的 onnxruntime_go 需 build tag 而失败，属固有环境问题）。\n- **性能（同口径实测）**：消息响应体 19.57MB → **3.58MB（−81.7%）**；`segments` 类型分布\n  与完整响应**完全一致**（thinking 1766 / content 1625 / tool_call 1851 / ask_user 1）；\n  逐字段校验 5019 段**非预期差异 0**；`_trunc` 按「等于任一被裁字段原文长度」校验\n  2838/2838 吻合；非裁剪路径仍可取回**完整数据**（19.56MB）。\n- **CDP 端到端**：全量展开 24 条消息 → 页面 **1776** 个工具行与接口 `tool_call` 段\n  **一一对应、逐行错误色判定不一致 0**（其中 1089 段依 `_err` 预计算）；展开被裁断的段\n  取回全文长度与原文**完全一致**（3290 / 8225 字符）；首屏 DOM 1696、控制台 **0 错误**。\n- 回归基线：首屏 `dom / chat / ap / folded / md` = 1690 / 879 / 268 / 24 / 26，与优化前\n  逐项一致。\n\n---\n\n## 1.6.6 — 2026-09-22\n\n### 变更 / 改进\n\n- **移除「文件快照」能力（编辑文件不再产生快照）** — 此前 `write` / `apply_patch` 在改文件前会把原文件复制到 `.pair/snapshots/<相对路径>/<时间戳>`，超量时按文件保留最近 20 份，并注册 `restore_snapshot` / `list_snapshots` 两个工具供查询与恢复（`.pair/rollback/msg-snapshots.json` 记录快照与用户消息的关联）。该能力整体下线：内核 `internal/agent/snapshot.go` / `rollback.go` 删除，写前快照调用点（`write` / `apply_patch` 内核 / harness 辅助）清除，插件 `tool-harness` 中两个工具声明同步摘除；启动时不再创建 `.pair/snapshots/` 目录，编辑文件不再写入任何快照文件。\n- **移除「回退到消息」** — 该功能依赖文件快照（恢复该消息关联的文件 + 截断其后对话历史），随快照下线：消息气泡悬停出现的「回退」按钮、会话回滚接口（HTTP 端点与内核 API 注册）一并移除。需要回溯文件改动请使用 git 历史；已有 `.pair/snapshots/` 与 `.pair/rollback/` 残留数据不再被读取，可自行删除。\n\n### 修复\n\n- **应用内「API 文档」残留已移除接口** — `HelpModal` 经 `api-docs.md?raw` 打进 **UI 区域包**（`.pair/plugins/ui-modals/assets/ui-modals.js`），与 vite 主壳是**两条独立产物链**；上一轮只重建了主壳，导致「帮助 → API 文档」仍显示 7.14 回滚接口。现 `node scripts/build-ui.mjs` 全量重建 15 个区域包（并 `--region modals` 复建），包内检索该接口路径与回退提示文案均为 0。\n\n### 文档\n\n- 应用内「更新日志 / API 文档」同步本版内容；`/api/system/info` 版本示例更新为 `v1.6.6`。\n\n### 验证\n\n- `go build ./...` / `go vet` 通过；`go test ./internal/agent/ ./internal/server/handler` 全绿。\n- UI 区域包重建后检索：`ui-modals.js` / `ui-right-panel.js` 回滚残留 = 0，`ui-modals.js` 含 `### 7.14 压缩上下文`；同步 `bin/.pair/plugins` 镜像后复验同为 0。\n- 遗留数据清理：`.pair/snapshots`（56MB）与 `.pair/rollback`（404KB）已删除，`ls` 确认不存在（本机 9090 仍为旧二进制，安装新版前该目录可能被旧逻辑重建，属预期）。\n- 发布包冒烟：解压 `release/PairCode-1.6.6.zip` 以独立端口启动，`/api/system/info` 返回 `1.6.6`，首屏控制台 0 错误。\n\n---\n\n## 1.6.5 — 2026-09-21\n\n> 本版把「自主模式」重做为**监督者（「人」）驱动**：工作 agent 每次自然结束，由一个独立的\n> 监督者回合审核产出、评判质量、决定下一步——监督者自己能调工具核查证据（读文件 / 搜索 /\n> git / 跑命令），裁决与完整轨迹实时进看板、刷新后仍可回放。插件面同时开放\n> `registerAutopilot` 与 `ctx.subagent.run`（策略在插件、能力在宿主）。另把**生成参数与连接\n> 信息从内核剥离**（改由 AI 配置 / 服务商配置 / 插件注册段提供），并修复工作区级技能被内置\n> 技能压制的问题。\n\n### 新增\n\n- **自主模式：监督者（autopilot）插件** — 新增 `.pair/plugins/autopilot`（策略全在插件：角色提示词 / 任务书 / 裁决语义 / 记录落盘 / 看板数据），宿主提供能力 `ctx.loopFactory.registerAutopilot({id, decide})`：工作 agent 每次自然结束（无 tool_call + 有正文）时，宿主在会话续轮处调用 `decide(req)`——任务书含用户目标 / 工作 agent 本轮汇报 / 运行统计 / 最近工作记录；返回 `continue + task` 则把指令作为新任务唤醒工作 agent，`done` 则整轮收尾。监督者是**具备全部工具的独立回合**（自行核查证据后经 `submit_result` 提交裁决：评判 + 下一步指令 + 证据），同一插件名重复注册即替换策略。\n- **子 agent 能力 `ctx.subagent.run(spec)`** — 插件可发起「子 agent 回合」：独立系统提示 / 任务书 / 模型 / 工具白名单与黑名单 / 独立历史 / 超时与轮次上限；事件带来源标注（`agentName`）供前端分区渲染，返回轨迹分段、工具调用、用量、耗时与结束方式。同步阻塞式调用（异步插件同样可用），未注册 provider 或能力不可用时给出明确错误。\n- **自主模式看板** — 右侧面板新增「监督者」折叠面板（对话区下方）：头部显示监督者轮次与末次裁决，每轮卡片含序号 / 裁决徽标 / 耗时·步数·工具数·tokens / 评判 / 下一步指令 / 证据·轨迹·工作 agent 侧记录（可折叠）。数据双通道：`GET /api/autopilot/rounds?convId=…`（切会话、刷新、WS 重连、会话结束补拉）+ `ui:autopilot:round` 实时事件；监督者回合的实时事件经 WS 下发并带 `agentName=supervisor`，前端按来源分区。记录落盘 `.pair/autopilot/<convId>.jsonl`（单会话滚动保留 200 轮），可溯源回放。\n- **服务商改名与删除引用提示** — 设置面板「服务商」tab 名称框解除禁用；保存时先调 `POST /api/models/rename`（同步迁移 `models.json` 键**并**更新 `ai-presets.json` 里引用旧名的 AI 配置，避免连接信息丢失），失败即中止；删除服务商时确认框列出仍引用它的 AI 配置名，并说明「仍可继续聊天（配置是完整快照），但对话面板会失去模型分组」。\n- **主界面可消费插件事件** — 插件运行时在分发宿主 `ui:` 事件时于 window 广播 `pair-plugin-event`：主界面组件（不是插件实例，原本收不到）也能消费插件事件，无监听者时零副作用。\n\n### 变更 / 改进\n\n- **循环装配器改为「链」语义** — `ctx.loopFactory.register` 从「单槽位后注册整体覆盖」改为**装配器链**：多插件按注册顺序依次叠加，仅「同一插件名重复注册」替换该项（卸载自动摘除）。修复此前后装载插件会把先装载插件（agentloop 的系统提示追加 / 分段预算 / 审核模式）装配参数整体吃掉的问题。\n- **生成参数来源唯一化** — 优先级：模型级（`models.json` 的 `modelParams[模型]`）> 服务商级（`models.json` 服务商条目）> AI 配置（`ai-presets.json`）> 全局默认（插件注册段 `generation`，存 `pluginSettings.generation`）。Go 内核零直读（原 `core.Temperature()` / `Settings.{MaxTokens,ThinkingMode,ContextMaxTokens}` 直读取消），设置面板新增由 agentloop 插件注册的「生成参数」页；插件侧一律 `ctx.getSettings(\'generation\')` 实时读取（改设置即时生效）。\n- **连接信息退出内核** — AI 连接字段（provider / baseURL / apiKey / model / executeModel / planModel / reviewModel）此前在 `settings.json` 顶层与 `ai-presets.json` 双份存储；现唯一来源 = **AI 配置**，`settings` 只保留激活配置名 `preset`。旧值由 `core.MigrateLegacyConnectionToPreset()` 一次性迁移（只补空字段、不覆盖已有配置），`core.MainModel()` / `core.Configured()` 删除（启动日志改打印激活配置名）。\n- **旧自主控制器退役** — 删除 `internal/agent/autonomous_controller.go`（「任务队列驱动下一阶段」实现）；自主模式唯一入口 = 监督者裁决，会话管理 / 任务管理 / 循环随之收敛。\n\n### 修复\n\n- **工作区级技能不再被内置技能压制** — 同名技能同时存在于系统 / 工作区 / 全局时，此前 `load_skill` 永远返回**内置旧版**（`skill_list` 还会出现两条同名条目、提示词重复注入）。现按 **工作区 > 全局 > 内置** 归并去重后返回。\n- **磁盘 Node 桥轨插件的可见性** — 修复 `tool-voice` 等桥轨插件在插件列表 / 工具集中不可见（磁盘插件正确交接给 Node 桥）。\n- **前端事件处理残留** — `agent-events.js` 的 usage 分支删除本地累加残留（运行统计唯一真源在后端），消除每次 usage 事件的 `ReferenceError`；自主模式收尾后前端「运行中」状态残留一并修正。\n\n### 文档\n\n- `docs/plugin-development.md`（774 → 934 行）：新增 `ctx.commands`（§4.7）、自主模式与 `ctx.subagent.run`（§4.8，含决策器返回契约与最小骨架）；inject 服务清单 9 → **25 个**；循环装配器链语义与坑表补充 5 条。\n- 技能 `cordis-plugin-development`（工作区版 / 内置版同步为 283 行）：补铁律、自主模式骨架、UI 插件实战要点、坑表。\n- 应用内「更新日志 / API 文档」同步本版内容；`/api/system/info` 版本示例更新为 `v1.6.5`。\n\n### 验证\n\n- `go test ./internal/agent/` 全绿（含自主模式 / 装配器链 / 会话监督 / 技能层级新增用例与整包回归）。\n- 技能层级探针（真实目录：工作区 `.pair/skills` + 安装目录 `config/skills`）：同名归并后选中**工作区版**。\n- 前端：`npm run build` + `node scripts/build-ui.mjs` 重建主壳与区域包，看板在真实会话下实时追加、刷新后经接口回放，控制台 0 错误。\n- 发布包冒烟：解压 `release/PairCode-1.6.5.zip` 以独立端口启动，`/api/system/info` 返回 `1.6.5`，插件与工具集装载正常、首屏控制台 0 错误。\n\n---\n\n## 1.6.4 — 2026-09-19\n\n> 本版新增**应用内在线更新**：直接对接 GitHub Releases，一键完成「检查 → 下载 → 校验 → 替换 →\n> 重启」，用户不再需要手动下载整包。更新源为 `releases/latest`（stable）或 prerelease 频道，\n> 用 release 元数据自带的 `digest` 做 SHA-256 校验，替换时**用户数据与配置永不被覆盖**。\n\n### 新增\n\n- **在线更新（GitHub Releases 直连）** — 新增引擎 `internal/update`（可脱离宿主单测）：清单解析支持三条通道（GitHub `/releases/latest` stable、`/releases` 列表 prerelease、自定义 feed 的 http/file 路径），按平台匹配资产（`PairCode-<版本>.zip` / `-linux-` / `-darwin-`）；校验优先用 release 元数据 `assets[].digest`（`sha256:…`），无需额外校验资产。下载走**镜像 → `api.github.com` 资产端点 → 直链**三级降级（`github.com` 直连常超时：实测 HEAD 21s 无响应而 API 资产端点 206 正常），支持 `.part` 断点续传、流式 sha256 与停滞看门狗；解压带 zip-slip / zip bomb 防护，并做包结构冒烟（主程序必须存在）。替换按**保护名单**过滤（`config/**`、`.pair` 用户数据、`logs` / `screenshots` / `_temp` / `release` 一律不覆盖）；Windows 上先用同卷 `rename` 在线替换运行中的 exe（失败降级为写 `.new`，退出后由脚本 `move`），保留 `.old` 备份并落盘 `last-apply.json`；重启脚本（`restart-<ts>.bat` / `.sh`）以脱离进程方式等待退出 → 清备份 → 启动新程序 → 自删。\n- **更新接口与设置项** — 宿主新增 `cmd/companion/update_api.go`（引擎单例 + 配置装配 + 6 个 handler）与内核路由 `update.check` / `update.download` / `update.apply` / `update.status` / `update.cancel` / `update.config`；设置段插件 `.pair/plugins/app-update` 提供更新源（github/custom）、仓库、频道（stable/prerelease）、自定义清单地址、镜像前缀、自动检查与间隔、强制校验、保留备份等开关。\n- **「关于」弹窗更新卡片** — 前端 `UpdateCard.vue` 接入关于弹窗：显示当前/最新版本与检查按钮，下载阶段展示进度（速率/总量/阶段），就绪态展示包内文件数并支持「安装并重启」；可**预览替换清单**（将写 N 个文件 / 保护名单跳过 M 个）。就绪态复用已下载缓存，不重复下载。\n\n### 文档\n\n- `docs/online-update-design.md`（新增）：完整设计（分发端点、清单与校验、下载降级链、解压与替换安全、重启机制、API 契约、配置项、UI 形态、失败路径），含 §10 验证方案与 §11 **真实 GitHub 源端到端验证记录**。\n- 应用内「更新日志 / API 文档」同步本版内容；`/api/system/info` 版本示例更新为 `v1.6.4`。\n\n### 验证\n\n- 单测 `go test ./internal/update`：11 项（版本比较 / 资产匹配 / digest 比对 / 保护名单 / zip-slip / 解压冒烟 / 端到端下载校验 / 断点续传 / 篡改拒绝 / 替换与备份 / 取消）。\n- **真实 GitHub 源端到端**（临时 `v9.9.9` prerelease + 真实资产，测毕已删除并确认 `releases/latest` 回到 v1.6.3）：检查 0.83s 发现新版本 → 下载 86 MB（峰值 10.3 MB/s）且 sha256 落盘/远端/本地**三方一致** → 解压 243 文件 → 预演 240 写 / 3 跳过 / 0 失败 → 真实替换成功且进程存活 → `restart=true` 自重启接管（新进程约 2s 起来，接口随二进制切换）→ 保护名单强证明（篡改包内同名的受保护文件后重跑 apply，用户内容原样保留）。\n- CDP 浏览器端到端：关于弹窗 → 更新卡片 → 预览替换清单，控制台 0 错误（`screenshots/update-*.png`）。\n\n---\n\n## 1.6.3 — 2026-09-19\n\n> 本版为创作域插件体系落地：六大创作域（画板 / UI 设计 / 3D 建模 / 音乐 / 2D 角色 / 人声）以\n> 「一域一包」形态上线，新增独立发布插件渠道 `plugins-dist/`，建模内核补齐带孔挤出 / 倒角 /\n> 圆角 / 扭转 / 扫掠与多边形 BSP 布尔路径（严格水密）；插件工作区根解析统一，市场恢复版本号与\n> 更新提示，已安装页支持按类型筛选。\n\n### 新增\n\n- **六大创作域插件（画板 / UI 设计 / 3D 模型 / 音乐 / 2D 角色 / 人声）** — 六个创作领域插件落地，每个领域一个插件包，含工具面（host 半：注册模型可直接调用的工具）+ 客户端面板（client 半）+ 前端资产。UI 与工具**同包**分发：面板不再作为独立插件包发布，`ui-art` / `ui-design` / `ui-model` / `ui-music` / `ui-rig` / `ui-voice` 的 UI 半已并入对应的 `@paircode/tool-{art,design,model,music,rig,voice}`（版本 `0.2.0`）；npm 上的旧 `@paircode/ui-*` 仍可安装，但已标记**废弃**（安装时提示「已并入 `@paircode/tool-<x>`」），请改用 `tool-*` 包。\n- **独立发布插件渠道 `plugins-dist/`** — 创作域插件采用「随市场独立分发、不随 IDE 发版」的节奏，此前与 IDE 基线插件同处 `.pair/plugins` 无法区分。新增 `plugins-dist/` 作为独立发布插件真源（与 `.pair/plugins` 同权：区域发现、发布扫描、UI 构建都会扫描它，但**不进 IDE 发布包**），本地开发与验收以 junction 挂载（挂载态天然不进包）；新增护栏脚本 `scripts/verify-dist-isolation.mjs` 校验「独立发布包 ↔ 打包排除项」逐项一致。\n- **主内容区视图 `registerView`（插件在主内容区开 tab）** — 插件可在主内容区 tab 栏开一个与对话 / 编辑器 / 市场 / 工具集同级的视图 tab（`ui.registerView({id,title,icon,order,open,href,render})`），与既有 `registerPanel`（藏在插件面板里的客户端面板）分工见 `docs/plugin-development.md` 新增对照表。挂载策略为**懒挂载 + 保持**：首次激活才渲染，切 tab 不卸载（保住 3D 视角 / 滚动位置等状态），关闭 tab 或卸载插件才清理，开启状态持久化；同时新增「并排对话」布局（主区分左右两栏：对话 + 当前视图，可换边）。\n- **建模工具（tool-model）能力补齐**：\n  - **2D 带孔轮廓挤出** — `extrude` 新增 `holes`（与 `profile` 同口径的孔轮廓数组），一步生成带孔板，不再需要 `subtract` 布尔（同尺寸挖 2 孔：276 面 / 4 ms，对比布尔 5763 面 / 218 ms）；洞方向统一、越界与相交明确报错，不静默降级；`center` 偏移对 circle / star / polygon 一致生效（偏心孔定位）。\n  - **扭转挤出（`twist` + `segments`）与扫掠（`type:"sweep"` + `along`/`closed`/`twist`/`scale`/`up`）** — 按层刚体旋转，洞随外形同步转（带孔即内螺旋槽），每层扭转角 < 180° 校验防自交；扫掠框架用平行传输（Rodrigues，等价 RMF），规避 Frenet 在直线段无定义、拐点翻转导致的自交。\n  - **倒角（chamfer）与圆角（fillet）** — 网格级边重建；凸体走 H-rep 精确构造（面平面 ∩ 斜面 / 球面半空间 → 三平面交点枚举 → 面内角度排序 → 扇形三角化 → 统一外向化），完全绕开布尔。同场景对照：H-rep 60 面 / 开边界 0 / χ=2，走 BSP intersect 则 1274 面 / 1274 条开边界 / χ=−126。\n  - **圆角顶点混合（rolling-ball）** — 关键认识是换表示而非补面：圆角的几何本质是**形态学开运算**（先按球磨小再滚回），凸体可解析构造，于是立方体 12 棱全圆角不再依赖「逐边补面 + 顶点补片」（该路线实测开边界 206~554，顶点相交是死结）。\n  - **多边形 BSP 布尔路径（I.6-2）并接线为默认** — BSP 全程保留凸多边形、导出前才三角化，共面分组天然免费（节点平面即其平面）。默认路径切换后**面数 −71%、M4 真缺口 −52.5%**；三角路径保留为回退 / 对照（`setMeshBooleanLegacy(true)` 或插件 `config.legacyMeshBoolean=true`），`statsOut.path` 回填实际所走路径。\n  - **文件注册式实时预览** — 工具写出模型文件即登记并广播，面板列出工作区识别到的文件、点击当场预览（STL bin/ascii、OBJ、glTF、GLB），外部改动自动重绘（事件 + 定时复核）。\n- **五域批量方法与「建工程即带内容」（`*_add`）** — 新增 `art_add(shapes)` / `design_add(nodes)` / `music_add(notes)` / `rig_add(parts)` / `voice_add(edits)`，统一形态：单件写法兼容（同层参数）+ 数组批量 + **整批校验通过才落盘**（复用各域既有 op 引擎，与 `*_edit` 同源）；建工程可一次带内容（`art_project` shapes、`design_project` screens[].root、`music_project` tracks[].notes、`rig_model` parts、`voice_import` paths/ids/edits）。`art` / `design` / `music` 的 `*_edit` 改为返回结构化 JSON（`ok` / `applied` / `log` / `summary`[/`tokensChanged`]），失败定位到具体条目（「第 N 条 op（名）失败，整批未写入任何改动」）。验证：Node 层每域 6~7 项（批量 / 事务性 / 单件兼容 / 失败定位）+ goja 宿主探针 + `voice` Node 侧 7 项 + `/api/tools` 可见 `*_add`。\n- **建模面板「登记并预览」（登记表驱动）** — 面板不再扫描工作区（旧口径会把工作区里任意 `*.json` 也列进来）：host 半删除 `scanWorkspaceArtifacts` 与 `SCAN_SKIP_DIRS`，`listArtifacts` 只列**登记表**（工具产出 + 面板手动登记，最新在前），返回 `scanned:false` / `scanRemoved:true`；顶栏路径框改为「登记并预览」（走 `claimArtifact`），移除「扫描工作区：开/关」切换与扫描来源合并逻辑，空态与帮助文案同步。\n- **市场显示版本号与更新提示、已安装页类型筛选** — 市场列表条目在类型标签旁显示 `v{latest}`，已安装插件条目补一行「有更新：vX → vY」或「已装 vX · 已是最新」，可更新时右侧出现「更新到 vY」（复用 `updatePlugin`）；已安装页插件条目显示**本地版本**（不再依赖 `config.npm` 是否存在）、可更新时给「有新版 vY」徽标 + 更新按钮；已安装页新增一排类型筛选 tag（**全部 / 插件 / MCP / 技能** + 计数，计数为 0 也可点击 → 显示空态提示，故不禁用）。版本对照由 `ensureUpdates()` 静默拉取 `/marketplace/check-update` 并 60s 复用，市场与已安装页共用同一份；宿主 `searchNpmMCP` 的 MCP 条目补 `version`（市场里 MCP 也显示版本号）。验证：CDP 端到端两脚本 PASS（市场 57 条中 37 条带版本号；搜 `tool-bug` → `v1.0.3` / 「有更新：v1.0.2 → v1.0.3」；已安装 tag 全部 52 / 插件 35 / MCP 0 / 技能 17），控制台 0 错误。\n\n### 修复\n\n- **布尔输出达成严格水密（M4 真缺口归零）** — 根因是各次布尔用各自的容差（随输入精度递增），同一条长边在相邻两侧的分割点跨次错开约 1~1.4 eps，任何焊接容差都合并不掉，严格半边上永远找不到配对。现按四项消解 T 缝：落在未配对边内部且贴近端点的顶点**吸附到端点**、真分割点**拓扑分割**插入未配对边、焊接比 0.5×→**1.5×eps**、**不劣化保护**（消解后若指标变差即回退）。结果：严格未配对 0 / 真缺口 0 / 缝总长 0 mm（此前 208 条真缺口），`meshRepair` 与 `meshWatertightReport` 同容差。踩坑已写入注释：T 缝插入必须**从该边的对顶点出发逐边分割并递归**（对整条边界链扇形分割会把「插入点 + 边两端点」三点共线拼成零面积三角形 → 非流形），插入点排序必须稳定（否则同向重复边）。\n- **带孔挤出 cap 出现反向三角形** — 自研「earcut 风格」耳切把 earcut 的两处兜底换成了「对角线可见性硬拦」，拦过狠时一轮找不到耳即退化为「取最大凸角强切」，强切切穿零宽桥接通道（3 洞场景 70 个 cap 三角形中 19 个负面积）。现完整移植 mapbox/earcut（filterPoints → cureLocalIntersections → splitEarcut 三级递进兜底），并修正连带回归：earcut 的 filterPoints 会删共线点，而轮廓上的共线中间点正是侧壁顶点，删掉会使 cap 边界比侧壁少边 — 改为只删重复点，并新增边界一致性校验（面积守恒抓不住「重叠 + 缺失互相抵消」）。\n- **布尔真缺口根因修复（BSP 平面容差与量化坐标）** — 共面判定阈值 `EPS_PLANE` 固定为 1e-6，比本场景空间分辨率（meshEpsilon ≈ 3.5e-4）小 350 倍，把「近共面」误判为「跨平面」，同一几何平面被拆进多个 BSP 节点、在远处切出 1e-2 量级坐标偏差（反向三角形 11 → 0）。同时量化顶点表原按平面节点独立建立，板面与孔壁节点对「同一个几何顶点」各取各的代表坐标（边界错开约 1 格），改为跨节点共享；环边跨节点对齐继续收尾（三项修复后真缺口 893 → 726，零回退）。\n- **Node 桥轨插件的面板在装载后被静默清掉** — 前端 `plugin-runtime.js` 的 `syncClientHalves` 在清理孤儿 client 半时会连带卸载由 boot 图（`dsh.ui` 区域包）装载的实例；声明了运行期 npm 依赖的 Node 桥轨插件（如 `tool-voice`）不出现在 `/api/plugins` 清单里，其 client 半只经 `/api/ui-boot` 下发，于是「人声」面板 / 视图在 boot 之后被静默移除。现按来源区分卸载对象。\n- **多工作区 / 新会话下截图等产物落错盘** — 内嵌工具注册表原为「首次 root 永久缓存」的单例，第二个工作区或新会话仍复用第一个 root，导致 `screenshot_stage` / `web_debug` 等落盘工具的产物写进**旧工作区**（新工作区里找不到文件，被误判为「截图不落盘」）。现改为按 root 键控缓存（互斥保护），截图目录兜底绝对化。\n\n- **更新检查对「手动放置的插件包」恒为空（市场永远不提示更新）** — 根因：更新检查只认磁盘插件包 `package.json` 里的 `config.npm`，而**只有市场安装链路会写该字段**，本仓库 36 个插件包全部是手动放置 / 同步的，于是 `/api/marketplace/check-update` **恒返回 `[]`**，已安装面板永远显示「无 npm 来源插件」、市场也永远不提示更新。现按官方约定**推断** `@paircode/<磁盘插件名>`（`manifest.name` 含 `/` 时直接作包名），registry 校验存在才判为 npm 来源、`current` 取包内 `version`，校验失败静默跳过（只回本地版本，不误报更新）；新增 `fetchNPMInfoChecked` 区分「包不存在（404）」与瞬时网络错误，**只对确定的 404 做 10 分钟负缓存**（否则每次「检查更新」都要对几十个非官方包打无用往返），成功结果不缓存以保证 latest 实时；`checkUpdates` 改为返回**全部磁盘插件包**（含非 npm 来源，供前端显示本地版本）+ 6 路并发 + 按名排序，`metaByPkg` 补推断使「更新」动作对这类包同样生效（否则 `/marketplace/update` 报「非 npm 来源」）。实测：`check-update` 由 `[]` → 36 条（36/36 识别为 npm 来源）；`tool-bug` 本地版本临时改 1.0.2 → 立即 `updateable:true`（latest 1.0.3），改回 → 可更新数归 0。测试：`internal/agent/npm_plugin_update_test.go`（3 例，httptest 桩 registry）。\n- **动态插件 `ctx.fs` 写错工作区（切工作区后仍写旧根）** — 根因：动态插件在 define 阶段把宿主根固化为闭包 / 上下文快照，装载期又拿不到触发者会话根，于是 `define` 探针写进了上一个工作区目录。新增 `PluginHost.SetWorkspaceRoot` / `WorkspaceRoot`（同步 `h.root`、根上下文、`workspaceRoot` 服务值与各已注册插件上下文），主工作区变更时由 `OnSyncWorkspace` 调用（主工作区被移除 → 同步空串，插件内解析**显式报错**而非静默写回旧根）；`ctxServiceRoot` 收敛为单一真相源五档（工具调用会话根 > UI invoke 根 > 装载期会话根 > 插件上下文根（实时）> 全局主根，全空报错），`buildFSService` 删除手写根解析副本与闭包快照，并顺带修复 `fs.roots` 把根写回闭包污染后续调用的问题。回归测试 `internal/agent/wsroot_probe_test.go` 3 项 PASS；端到端（独立实例 9098）：wsA 生成探针文件 → 切 wsB 再 define → wsB 落盘且 wsA 不再被写。\n- **`plugin-publisher` 扫不到 junction 插件** — `Dirent.isDirectory()` 对 Windows junction 返回 `false`（既非目录也非 symlink），改用 `statSync` 跟随重解析点复核；现象是开发态把 `.pair/plugins/tool-<x>` 以 junction 挂到 `plugins-dist/tool-<x>` 后，`--list`、交互式菜单与 Web UI 的插件列表里都看不到六个创作域插件（发布工具漏扫，不是插件本身问题）。与 Go 端 `isPluginDirEntry` 同口径。\n- **`tool-voice` 发布包缺实现模块（装上即报错）** — `tool-voice@0.3.2` 的 tarball 里没有 `lib/`，而 `index.js` 有 8 处 `require(\'./lib/...\')`；双保险都漏了 `lib`：`PUBLISH_FILES`（拷贝白名单）与 `buildPackage` 覆写的 `pkg.files`（`npm publish <dir>` 仍按 `files` 字段过滤）。修复后重发 0.3.3（含 `lib`，15 个文件）已进 registry。教训：判定发布结果不能只看 `npm publish` 退出码，须等 registry 版本端点 200 / dist-tags 落实（本次因读取端传播延迟 + 409 `previously staged version` 语义误判，另 bump 出内容相同的 0.3.4）。\n\n### 变更 / 改进\n\n- **插件版本与分发收口** — 六个创作域包统一 `0.2.0`；IDE 发布包的插件排除项由 11 项收敛为 6（独立发布包 = `tool-{art,design,model,music,rig,voice}`）；`tool-voice` 包名规范化为 `tool-voice`（原名带 scope 会让 `/api/ui-boot` 的入口 URL 带 `@`，导致 `/plugins-assets` 静态路由 404）。\n- **旧 `ui-*` 引用清理** — 清除代码注释、工具提示、CI 示例、文档中残留的 `ui-*` 包引用（面板名改以 `client.js` 注册标题为准，如「ui-art 面板」→「**画板**」面板）；发布脚本的孤儿包废弃文案（`ORPHAN_HINTS`）补上六域映射，`--deprecate-orphans` 会给出「已并入 `@paircode/tool-<x>`（UI 与工具同包）」的准确说明。\n- **建模面数口径更正** — 上一轮报告的面数对比取自不同焊接 eps 的诊断网格，两者不可比；同口径实测为 4416 → 4536 面（**+2.7%**，表面积变化 −4.7e-6%）。并给出结论：对已水密的网格再做共面合并是**以水密换面数**（4536 → 1187 面，但非流形 1 / 缝长 7.86，再修也回不来），两者同时追求必然振荡，故默认布尔路径不做共面合并（共面合并只保留在 legacy 对照分支）。\n- **六域移除插件面板注册，统一面板外壳** — 六域 `client.js` 移除 `ui.registerPanel`，只保留 `ui.registerView`（主内容区 tab；`assets` bundle 与 `dsh.ui` / `client` 字段保留——`registerView` 共用同一 bundle、`/api/ui-boot` 靠它算 rev、npm 安装需保整包）；面板统一外壳新增 `PanelShell.vue`，六个 `Panel.vue` 改用它并重建 `assets/*-panel.{js,css}`；六个 `index.js` 共 49 处工具参数描述补「相对主项目根解析」基准；新增审计脚本 `scripts/audit-plugin-ui-register.cjs`（客户端注册面，6/6 通过）与 `scripts/audit-plugin-tool-desc.cjs`（工具参数描述基准）。线上 tarball 核验 `registerPanel` 调用为 0。\n- **六创作域包版本对齐 registry** — `plugins-dist` 六包（`@paircode/tool-{art,design,model,music,rig,voice}`）发布时由 `plugin-publisher` 按内容指纹自动升 patch，本版把提升后的版本号回写仓库：统一 `0.3.2`（`tool-voice` 另含 `lib`）。\n\n### 文档\n- `docs/creative-domain-landing-plan.md`：落地形态改为「一域一包」，补五域推广与两条踩坑（scope 包名导致静态路由 404、boot 图与孤儿卸载冲突）；`.pair/project.md` 同步创作域插件目录关系。\n- 应用内「更新日志 / API 文档」同步本版内容；`/api/system/info` 版本示例更新为 `v1.6.3`。\n- `docs/plugin-development.md` 新增 §5.1「相对路径的根从哪来」五档优先级表；`config/skills/cordis-plugin-development/SKILL.md` 新增 §0 铁律（仅用户明确要求才 `define`、优先复用磁盘插件、临时插件用完清理）与 §2.5 工作区根解析纪律，并修正过时描述「插件只存在于内存不落盘」（`define` 实际会固化到安装目录）。\n\n---\n\n## 1.6.2 — 2026-09-13\n\n> 本版修复会话历史展示与上下文注入两类问题（`v1.6.2` 已于 2026-09-13 打 tag 并推送，更新日志条目随本版一并补齐）。\n\n### 修复\n\n- **会话历史现在展示真实落盘数据（归档原文可见）** — 长会话的早期消息此前会被自动归档：原文逐字移入 `{会话}.jsonl.archived.jsonl`，主文件只留一行统计（「**历史归档**（共 N 条消息：用户 x/助手 y/工具 z）」），**前端展示线只读主文件**，于是归档前的真实历史再也翻不到，只剩那块没有内容的统计提示。现改为：归档时生成**真压缩摘要**（逐轮「用户任务 + 该轮结论 + 工具调用次数」，说明原文保留位置），供模型侧恢复上下文；而**展示线改为「归档原文 + 主文件」合并读取**，前端可一路向上滚动回翻到对话最早的真实消息（归档区消息用负序号标识，主文件消息保持原序号，回滚等语义不变）。归档区消息不支持回滚（会明确提示，避免误清空历史）。\n- **移除「背景上下文」折叠条（语义有歧义）** — 消息流里由系统注入的上下文消息（早期历史摘要、会话交接视图、历史遗留的背景快照）此前被统一渲染成「**背景上下文**（非当前任务）」折叠条，与用户真实消息混在一起、含义不清。现该类消息统一由后端识别并从展示历史中排除（模型侧仍可见，作为上下文的一部分），前端折叠条与其样式/识别逻辑整体移除；对话窗口只呈现用户真实消息与助手回复。\n\n---\n\n## 1.6.1 — 2026-09-13\n\n> 本版为 1.6.0 之后的修复与开发工具入库（含社区贡献 PR #1）。\n\n### 修复\n\n- **专注模式 / 左栏 / 会话列表显隐统一入口（社区贡献）** — 侧栏、会话列表、编辑器显隐此前各自直接改状态：专注模式（Ctrl+K）只隐藏编辑器，组件内的 `watch(focusMode)` 还会覆盖「显式唤出侧栏」的用户意图；会话列表面板没有独立开关。现收归统一入口：`setFocusMode()` 进入专注时收起左栏与会话列表、退出时还原进入前的选择（专注态内手动切换会同步还原目标，退出时不被旧值覆盖）；新增会话列表面板独立开关 **Ctrl+Shift+L**（选择持久化；专注态内切换属临时调整，不写入偏好）；视图菜单与编辑器右键命令统一为「退出专注 → 显式显示侧栏」，靠语句顺序保证用户意图不被还原覆盖。\n- **插件装载较慢时历史消息不出现（社区贡献）** — 历史加载判定由「是否已存在非占位消息」改为「该会话是否加载过历史」：此前兜底定时器先写入快照/事件，切换会话即被判定为「已有内容」而跳过加载，历史永不出现。兜底提示文案同步澄清为「未在 N 毫秒内开始（插件装载较慢时的预期兜底）——先应用实时事件，历史随后加载」。\n- **网页验证（web_debug）白屏误报（社区贡献）** — 白屏 / DOM 探测脚本末尾写成了箭头函数立即调用形式，在表达式位置属语法错误，探测整体失败后文本长度恒为 0，**凡探不到文字的页面（Vue/React 根应用、Canvas/SVG 应用、文本位于不可见节点）一律误报「页面白屏」**。现改为纯函数表达式，并补 `textContent` 兜底、元素计数与探测失败可见化；白屏判定改按「探测失败 → 不判定；有文字 → 非白屏；无文字但 DOM 元素 ≥ 20 → 非白屏（仅提示文本不可提取）；否则白屏」给出，「未提取到可见文本」与真正白屏分开表述。\n- **网页验证「可见 0 个」误报（社区贡献）** — 页面元素超过 5000 时探测会主动跳过可见性统计（性能保护），计数保持初值 0，报告据此输出误导性的「可见 0 个」，易被误读为页面异常。现严格区分「可见性未统计（元素过多）」与「真的是 0 个可见」。\n- **控制台日志占位符未替换（社区贡献）** — `console.warn("x %d y %s", 1, "a")` 此前被拼成 `x %d y %s 1 a`；现按浏览器 console 语义替换 `%d/%i/%f/%s/%o/%O/%j`（`%c` 丢弃对应样式参数，`%%` 还原为 `%`，参数不足保留原占位符，多余参数追加末尾）。\n- **服务监听地址说明与实际不符** — 服务已显式监听 `0.0.0.0`（支持局域网访问），应用内文档仍在提示「仅监听本地回环地址（127.0.0.1）」；现统一更新为「监听所有接口（0.0.0.0）：本机用 `localhost` 访问、局域网内其他设备用本机 IP 访问；请勿将端口暴露到公网，并注意防火墙与访问控制」。\n\n### 改进\n\n- **开发 / 诊断脚本入库（社区贡献）** — 把此前散落在临时目录、有长期价值的开发与诊断脚本与文档归档进 `scripts/`（`pair-switch/` 实例切换、`github-sync/` 五阶段仓库对齐、`d-sync/` 差异报告、web 验证夹具等），新增 `scripts/README-dev-tools.md` 说明各脚本用途；同批入库 `scripts/deploy-pair-full.ps1`（全量部署：清单校验 + 自动回滚）。\n\n### 文档\n\n- 应用内「更新日志 / API 文档 / 功能介绍 / 常见问题」同步本版修复与监听地址说明；`/api/system/info` 版本示例更新为 `v1.6.1`。\n\n---\n\n## 1.6.0 — 2026-09-12\n\n> 本版汇总 v1.4.15 之后至 v1.6.0 的全部工作（含 1.5.x 期间未单独记录的变更）。\n\n### 新增\n\n- **段预算双闸门 + 自动分段续跑** — 单段（一次 Run）同时受两道预算约束：**步数预算**（`stepBudget`，一步 = 一次 LLM 调用，一次回复并列多个工具调用仍算 1 步）与**工具调用预算**（`toolCallBudget`），**任一达上限即结束本段并自动开启续跑段**——同会话历史保留、压缩照常、目标续轮不受影响，长任务不再因单段预算耗尽而中断。续跑段数上限 `maxToolBudgetSegments`（默认 20；不允许「不限」，防失控）。三项均可在 **设置 → Agent** 实时配置（默认 120 / 120 / 20；0 或留空 = 默认，工具/步数预算负数 = 不限，超上限自动钳制），配置由插件注册、装配器透传，改完即生效无需重启。\n- **会话交接（提交消息）** — 续跑与新提交对话两个时机自动整理上下文：历史达阈值（≥ max(30% 窗口, 24K tokens) 或 ≥100 条）时生成「会话交接·提交消息」（相关性判断 高/部分/无关 + 目标 / 已完成 / 当前状态 / 关键决策 / 下一步），替代全量历史注入；未达阈值**逐字节原样注入**（缓存前缀零影响）。复用期内增量小则复用、超阈值刷新，并定期做极轻量语义复检（独立小请求，实测 ≈350 tokens），相关性变化时驱动保留深度（16 / 8 / 4 条）。落盘与展示**从不改写原文**（视图不入盘）；`PAIR_HANDOFF=0` 可整体关闭。\n- **创造模式 `/创造 <需求>`** — 用一句需求自主创建「场景」工具集：能力盘点 → 组合（优先复用现有插件）→ 固化到 `.pair/toolsets/<场景名>.json`，对话面板「工具集」选择器即时可切换（支持中文场景名）。\n- **LLM 请求/响应追踪（llm-trace 插件）** — 逐请求落盘前缀命中分析（JSONL），可精确定位缓存前缀断裂点，用于长对话成本优化与问题定位。\n- **运行统计条** — 对话面板显示本轮耗时 / 步数 / 工具次数 / 输出 token / token 速度（运行中每秒刷新、结束后定格）。\n- **对话面板可靠性** — 切换对话自动拉取该会话任务（含会话切换竞态保护）；WebSocket 断线自动重连并补偿拉取任务与消息。\n- **团队编排无成员化** — 任务 DAG（依赖门禁）、质量门禁（契约字段 / verdict / findings，修复 + 复审自动派生）、两阶段批准（staged → approve → running）、就绪任务提示（claim/update/approve/status 均返回「下一步」），由队长单 Agent 多步执行完成，不再派生成员会话。\n- **绕圈 / 死循环重复检测回归** — 识别模型重复相同调用或重复内容，避免无效空转。\n\n### 变更 / 改进\n\n- **工具面收敛（单工具形态）** — project_info / memory / goal / load_skill / cordis / codegraph 关系查询等多工具组合并为「单工具 + `op` / `mode` 参数分派」，LLM 可见工具面显著缩小、误选率下降。\n- **`apply_patch` 统一编辑入口** — 补丁式编辑替代原 edit / multi_edit 系列（旧工具全链移除）。\n- **插件面精简** — 36 → 29：移除 tool-git（git 走 `exec_command`）/ tool-entryconfig / host-capability-probe / web-api；tool-vision → tool-web、tool-snapshot → tool-harness、tool-asset → tool-resource 合并。\n- **按需工具（deferred）机制移除** — 低频工具直接注册进 LLM 工具面（不再隐藏不可用），可见范围由场景白名单 + `/命令` 激活控制；`/命令` 激活的工具跨对话保持、每轮重注册。\n- **子 Agent 默认关闭** — 默认不注册 subagent 系列工具，需要时以 `PAIR_ALLOW_SUBAGENTS=1` 显式开启。\n- **「最大迭代数」配置项移除** — 段内迭代安全上限改为由预算派生（生效预算 + 5），段的收束交给双闸门 + 自动续跑，无需手工调参。\n- **配置项注册纪律** — 插件配置一律由插件自身 `ctx.registerSettings` 注册（存 `pluginSettings.<插件>`），不再新增全局设置顶层字段，避免设置面膨胀。\n- **任务按会话隔离** — 任务清单写入 `ConvID`，多会话共享任务文件不再互相覆盖。\n- **多项目寻址统一** — 工具相对路径一律相对主项目根解析，跨项目传 `project` 参数或绝对路径；搜索忽略目录统一真相源（依赖库 / 构建产物 / VCS / 缓存 + 项目根运行数据 + 用户自定义）。\n\n### 修复\n\n- **写通道审核 P1：同工具不同参数被连带误驳回** — 驳回指纹由「仅工具名」改为「工具名 + 参数指纹（FNV-1a）」，某次驳回不再影响该工具其他参数的合法调用。\n- **写通道审核 P2/P3：写类工具送审字段为空导致必驳回** — Git 写类等工具参数为 files / message / target，后端审核只读 `path` / `content` 双空；现按工具形态归一化提取审核字段，AI 审核模式不再 100% 误驳回。\n- **长对话缓存前缀断裂** — 两处根因修复：① 兜底 user 占位符错位（改为末尾追加，反馈 / 压缩 / 新 Run 翻转场景不再整段 miss）；② JS 路径压缩冷却不递减（补 `tick()` 推进与压缩事件补发，前端可见压缩提示）。\n- **多段续跑落盘丢消息** — 持久化基准由「固定原始历史」改为「可变底账」，第二段不再全量覆盖丢掉第一段新增消息。\n- **会话级工具集白名单污染全局** — 会话注册表改存值拷贝，多会话构建不再互踩（`/api/tools` 状态不再被最近一次会话改写）。\n- **`/命令` 二次执行不唤醒 agent** — 已激活的按需命令重复执行同样唤醒对话（此前只有结果卡片、对话不动）。\n\n### 文档\n\n- 应用内文档（快速开始 / 功能介绍 / API 文档 / 工具文档 / 快捷键 / 常见问题 / 更新日志）与仓库 README 同步本版变化（双闸门配置、会话交接、创造模式、团队编排与工具面收敛）。\n\n---\n\n## 1.4.15 — 2026-09-02\n\n### 修复\n- **模型切换报「对话不存在 / 会话不存在」** — 根因：打包管线缺失 vite 壳构建步骤（pipeline 只跑 build-ui 区域插件 + robocopy 同步旧 dist），打包复用过期前端产物，致 `setConvModel` 调用参数错位（5 参新调用打在 4 参旧签名上，workspaceRoot 被传成配置名「硅基flash」）→ 后端按错位 workspaceRoot 路由隔离 store 查无会话报 400\n- **packager.json pipeline 新增 `build-ui-frontend` 步骤** — 打包前显式构建 vite 壳（plugins-src/ui-app → .pair/assets/runtime/web），保证发布包前端产物始终来自新构建\n- **后端跨 store 兜底防误报** — SessionManager 新增 `FindConversation`（指定 workspaceRoot store 查不到时遍历已打开 store 找回），GET/PUT 会话接口在参数缺失/错位时不再误报「不存在」，会话级模型切换落盘到会话真实所属工作区\n\n### 改进\n- 版本号整体提升至 v1.4.15（main.go 缺省 / packager.json / 前端 package.json）\n\n---\n\n## 1.2.1 — 2026-08-15\n\n### 新增\n- **按双层循环范式重写 Agent 核心** — 双层循环（turn/step 边界事件、inbox 双队列对齐 next-step/next-turn），消息组装与落盘对齐事件模型（agentloop 编号 ↔ 消息序列推导），系统提示精简为基础工具集模式（`WB_FULL_TOOLS=1` 恢复全量工具）\n- **一切皆插件** — Go 插件框架 + goja JS 动态插件，goja 运行时完全内置（双仓库去除 replace），JS 插件沙箱支持 timer 服务（ctx.timeout/interval）与跨 goroutine 执行锁\n- **内置 TS 编译器** — esbuild 纯 Go 转译（无 CGO/npm 依赖），TS 插件可直接加载（`cordis_define` 支持 js/ts/自动探测），多文件 TS bundle（Build stdin + mock 包）\n- **工具全插件化** — 21 个内置功能插件（core/fs/git/web/shell/memory/task/project-info/codegraph/debug/vision/office/lsp 等），`cordis_inspect` 可见工具归属插件，Unload 可回收整组\n- **多项目支持** — 工具 project 参数路由（文件类/搜索/Git 全套），codegraph 按项目独立建图与查询（非主项目用各自 JSONStore，天然隔离），memory/project-info 工具显式 project 参数化\n- **工具集生态** — 模板插件化动态构建（`toolset_build` 按项目+需求自动组合工具并固化到工作区）、固化/导出/导入/市场发布（plugin 类型）、LLM 项目意图分析（语言无关，不固化任何语言模板）\n- **插件生态 P0-P2** — 函数形态 + `apply(ctx, config)` + inject 服务 + VM 超时防护 + schema 校验 + 插件管理 UI（host/client 双半）+ client inspect provider\n- **项目知识库树形化** — 树分支组织（目标/架构/实现/关键点/设计思想）+ AGENTS.md 分层 + .agents 路径兼容\n- **历史注入对齐 harness** — 删除【历史轮次】前缀标注与 task 时间戳，系统提示补充多轮对话规则\n- **ask_user 选项内输入** — 支持 single / multi / single-with-input / text 四态交互，修复参数名混淆导致选项不出现的问题\n- **遗留五件套** — notes 写入同步 + read_image 工具 + run_code 嵌套 + prompt 注册中心 + 知识库过期检查修复\n\n### 修复\n- **移除未完成注入** — TOOL_OUTCOME_UNKNOWN / interrupted 机制移除，无 result 的 tool_call 以空占位维持配对契约，不再向模型注入「中断/未完成」语义\n- **知识库过期验证误报** — 152 条假警告清零，159 条全绿\n\n---\n\n## 1.1.8 — 2026-08-11\n\n### 新增\n- **OCR / 图色识别能力** — 图片文字识别（中英文混合）与颜色分布分析，工具配置持久化 + 前端工具面板（2026-08-04）\n- **对话历史注入膨胀三层压缩** — 固定背景 / 动态日志 / 长时压缩三层方案，控制上下文体积（2026-08-04）\n- **异常中断后继续未完成对话** — 中断后可直接继续，不丢上下文（2026-08-06）\n- **后台进程跨轮存活** — run_background 进程不再因每轮重建注册表而丢失（全局单例 bgRegistry）（2026-08-11）\n- **多项目工具** — Lua 工具 / 工具配置按项目加载 + project 参数路由（2026-08-11）\n- **背景摘要注入位置修复** — 压缩摘要固定在 task 前注入（前缀稳定），动态日志追加末尾，KV 缓存零损失优化（2026-08-08）\n\n### 修复\n- 关闭 run 内自动压缩，改由外层时机控制（2026-08-05）\n- 历史消息配对错乱 — 用户消息重复存储导致 tool 配对错乱（lastUser 锚点重组）\n- 历史消息分段导致多气泡 — 连续 assistant 消息合并显示\n- 多轮对话 user 后 tool 粘连 + OnBatchPersist 偏移 — 压缩后固定偏移失效，改 lastUser 锚点重组\n- 归档双 bug — ①Windows 归档静默失效（句柄未关闭 + os.Rename 不能覆盖）→ 显式 Close + 三步法原子替换；②归档摘要孤立 assistant 消息污染 LLM 上下文 → 改 role=user +【历史归档】标注\n- 多根路径解析 Bug — 优先匹配文件实际存在的根目录\n\n---\n\n## 1.1.6 — 2026-07-30\n\n### 修复\n- **修复编辑器 Ctrl+F 不生效** — CodeMirror `search()` 扩展注册的 `openSearchPanel` 与自定义搜索面板 keymap 冲突，使用 `Prec.high()` 确保自定义 handler 优先执行，Ctrl+F 正确唤出中文搜索面板\n- **搜索面板图标全部换为 SVG** — Unicode 字符（▲▼↔×）和文本标签（Aa ·\\* 全词）全部替换为内联 SVG 图标，与界面风格统一\n- **修复前端 API 路径缺少前导斜杠导致 404** — `apiURL()` 拼接时对无前导斜杠的 path 自动补全，`/apitools/review` 修正为 `/api/tools/review`\n- **修复 codegraph 增量构建仍全量重写 SQLite** — `SQLiteStore.Save()` 在增量模式下调用的 `RemoveFileEntities` 清理旧数据，不再 `DELETE FROM` 全表\n\n### 改进\n- **编辑器中文搜索面板** — 新建 `FindPanel.vue` 组件，替换 CodeMirror 默认英文搜索面板，支持查找/替换/大小写敏感/正则/全词匹配\n- **codegraph 增量构建测试** — 新增 `TestSQLiteStoreIncrementalPreserves` 和 `TestSQLiteStoreIncrementalBuild` 验证增量构建与并行完整性\n\n---\n\n## 1.1.5 — 2026-07-29\n\n### 新增\n- **run_command 后台化** — `run_command` 改用后台启动+轮询模式，不再阻塞 Agent 循环，可被上下文取消中断，超时后 LLM 可选择等待或继续\n- **审核配置改为工作区级** — 审核黑白名单从全局 settings.json 迁移到工作区 .pair/tools.json，不同工作区可独立配置，避免动态工具（Lua）在不同工作区间混淆\n- **Lua 工具补齐 Tool 结构** — `buildLuaTool` 自动设置 UsageGuide/Category/Enabled 字段，与标准工具结构一致\n- **工具配置弹窗合并** — 「启用开关」和「审核黑白名单」合并为同一「工具配置」弹窗，标签页切换，避免歧义\n- **自主模式 Follow-up 持续驱动** — Agent 自然终止后，通过 `OnNextTask` 回调自动注入 follow-up 消息，无需手动触发「继续」\n- **流式更新机制** — Registry 新增 `OnToolUpdate` 回调，工具执行中间结果实时推送给前端\n- **工具 UsageGuide 全覆盖** — 全部 ~140 个工具添加 `UsageGuide` 使用指导，明确何时用、为何优于 `run_command`、常见误区\n- **启动日志详细化** — 启动时输出版本号、Go 版本、平台架构、工作目录、各工作区文件夹路径\n\n### 改进\n- **工具体系升级**\n  - `Tool` 结构体新增 `UsageGuide`、`Category`、`Enabled` 字段\n  - `Registry` 新增 `EnabledDefinitions()` 按状态过滤工具定义\n  - 新增 `AllToolMeta()` API 供前端展示工具开关列表\n  - 工具使用指南文本动态注入系统提示，引导 LLM 优先使用专用工具\n- **窗口管理** — `run_command` / `run_background` 均设置 `HideWindow=true`，不再弹出 cmd 窗口\n- **信号监听移除** — main 函数移除信号监听，进程不会因子进程结束而自动退出\n\n### 修复\n- **debug_start 启动修复** — 拆解 `dlv dap` 启动流程，分别发送 Initialize 和 Launch 请求，兼容 dlv 最新版本\n\n---\n\n## 1.1.2 — 2026-07-21\n\n### 新增\n- **附件标签化** — 消息中的文件/代码/图片附件不再嵌入正文，改为独立药丸形标签显示在用户消息文字下方，视觉更清爽\n- **粘贴长文本自动转临时附件** — 输入框粘贴超过 2000 字符的文本时，自动写入 `_temp/` 目录并作为附件挂载，避免大段代码/日志撑爆输入区\n\n### 改进\n- `addToChat` 瘦身：文件添加到对话不再预读文件内容（40KB 截断已无意义），仅传递路径引用\n- 目录引用新增 `type:dir` 支持，提示 agent 使用 `list_files` 查看\n- 选中代码添加对话现在保留代码内容尾注供 agent 直接参考（截断 3000 字）\n\n### 修复\n- 文件树 Shift/Ctrl 多选逻辑修复：范围选择改为基于同级节点列表，清除后重新选中\n\n---\n\n## 1.1.1 — 2026-07-21\n\n### 新增\n- **审核配置界面重设计** — 从纯文本输入改为工具卡片式交互：所有工具按类别分组（文件操作、命令执行、Git、网络、截图、图像、二进制、办公文档、CodeGraph、调试器、知识库、记忆、LSP、BUG检测、任务管理、扩展市场等），每个工具显示中文名称，点击切换三态（默认 → 黑名单 → 白名单），支持搜索过滤，配置更直观高效\n\n### 修复\n- **修复新对话空状态提示位置偏移** — "开始新的对话，发送消息即可与 AI 助手对话"提示及图标从左下角偏移修正为居中显示\n\n### 改进\n- 版本号统一升级至 1.1.1（前端 package.json、后端 main.go、打包配置）\n\n---\n\n## 1.1.0 — 2026-07-20\n\n### 新增\n- **自主模式原生终止** — 去掉 `finish_task` 强制结束机制，Agent 自然输出后直接结束循环，交互更流畅\n- **Agent 性能优化（P0-P3 五轮）** — eventRing 环形缓冲器减少内存分配、进度可视化（阶段指示器+工具调用计数+耗时）、工具描述精简减少 Token 消耗、并行工具执行机制、预压缩上下文避免截断\n- **会话连贯性增强** — 新对话开始时自动注入 Git 变更感知、代码图谱统计、工作区结构概览，Agent 无需从零分析项目\n\n### 改进\n- **ChatView 重构** — 消息渲染管线全面优化，新增交互超时保护、审核驳回追踪、折叠/展开状态持久化\n- **审核配置 UI 优化** — 弹窗改为向上弹出（bottom:100%），防止被视口底部裁切\n- **编辑工具 v2 升级** — 更精确的符号级定位，减少行号偏移问题\n- **kill_process 增强** — 改为杀进程树，彻底清理子进程\n- **自主模式架构重构** — ephemeralMsgs 隔离内层消息，长时压缩精准保留推理上下文\n\n### 修复\n- 修复 `planExpanded` / `tasksExpanded` / `currentPhase` 重复声明导致的运行时崩溃\n- 修复 `currentTasks` 未声明导致前端 `undefined.length` 崩溃\n- 修复自然终止代码缩进丢失导致逻辑在循环外不执行\n\n---\n\n## 1.0.20 — 2026-07-18\n\n### 修复\n- **修复消息排序** — `_idx` 统一取 `max(existing)+1`，解决历史消息加载后序号错乱\n- **修复用户反馈消息合并** — 用户反馈正确合并到 agent 输出气泡中，不再产生额外用户消息气泡\n- **修复消息发送双占位竞态** — `switchConv` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，避免两个 assistant 气泡\n- **修复 WS 连接与历史加载竞态** — `processStatus` 事件正确处理连接状态转换\n\n### 改进\n- 审核配置弹窗改为向上弹出（`bottom:100%`），防止被视口底部裁切\n- 移除压缩按钮，简化 UI\n\n---\n\n## 1.0.19 — 2026-07-17\n\n### 修复\n- **修复 Web 端文件树不显示** — `FileExplorer.vue` 的 `<script setup>` 编译后 JS 中存在变量暂时性死区（TDZ），导致 `setup()` 抛出 `Cannot access \'d\' before initialization`，文件树组件挂载失败。重建前端并重新编译 `companion.exe` 嵌入新版 dist 后修复\n- **修复后端 dist 嵌入路径不一致** — `cmd/companion/main.go` 通过 `//go:embed web-ui/dist` 引用 companion 目录下的副本，但此前构建脚本将 dist 输出到 `cmd/desktop/web-ui/dist/`，两者不同步导致嵌入的仍是旧版 JS。统一构建流程后将新版 dist 正确复制到 `cmd/companion/web-ui/dist/`\n\n### 改进\n- 统一更新版本号至 1.0.19（后端 main.go、两个前端的 package.json）\n\n---\n\n## 1.0.8 — 2026-07-17\n\n### 新增\n- **多项目工作区支持** — 系统提示自动遍历所有工作区根目录，读取各自 `.pair/project.md` 环境配置注入给 AI，跨项目协作时准确感知每个项目的编译方式、CGO 开关等信息\n- **CodeGraph 多项目全量建图** — `codegraph_build` 支持对所有工作区项目建图并合并到同一个知识图谱（`rebuild=true`），跨项目符号搜索成为可能\n- **阻塞命令自动拦截** — 新增 `isBlockingCommand` 检测，自动拦截 dev server、watch 模式、`go run .`、`npm run dev` 等长期进程命令，提示改用 `run_background`，避免阻塞 AI 循环\n\n### 改进\n- **审核放行逻辑优化** — `run_command` 阻塞命令不再自动放行，强制走 LLM 审核；`run_background` 保持安全命令自动放行\n- **工具描述优化** — `run_command` 描述明确禁止长期进程并列出典型误用场景；`run_background` 强调作为长期进程首选工具\n- **系统提示增强** — 「错误恢复」和「防止卡死」两处加入阻塞/后台区分铁律，降低误用 `run_command` 概率\n\n---\n\n## 1.0.7 — 2026-07-17\n\n### 修复\n- **修复刷新页面后 ask_user 提交造成额外气泡** — 页面刷新后 `switchConv` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，不再另建新占位，避免两个 assistant 气泡\n\n### 改进\n- 统一更新版本号至 1.0.7（前端 package.json、后端 main.go、打包脚本）\n\n---\n\n## 1.0.6 — 2026-07-17\n\n### 修复\n- **修复消息持久化比较口径不一致** — `PersistNewMessages` 中 `persistedCount` 使用 `countJSONLLines`（统计文件总行数含 System），与 `histNonSystemCount`（统计非 System 消息数）口径不同，导致含 tool_call 的 assistant 消息在工具执行前被误判为"已落盘"而跳过写入。阻塞工具（如 ask_user）的前端始终无响应。改用 `readJSONL` 精确统计非 System 消息数\n- **修复对话/任务/执行计划 API 空实现** — `GET /api/conversations/{id}` 缺 agent 运行状态，`GET /api/tasks` 和 `GET /api/taskplan` 原返回对话列表（完全错误的 stub），改为返回真实数据\n\n---\n\n## 1.0.5 — 2026-07-17\n\n### 改进\n- **消息持久化重构** — `PersistNewMessages` 改为全量覆盖写 JSONL，消除 diff 计算的竞态问题；`MessageStore` 新增 `ReplaceHistory` 支持历史压缩；`MergeLastAssistantRun` 移除，各轮次独立存储以保留 reasoning 完整时序\n\n### 修复\n- **修复 send on closed channel panic** — 移除三处 `go func` 在无监听者时向 channel 发送导致的崩溃\n- **修复 PersistNewMessages 上下文压缩后新消息丢失** — 全量替换模式确保压缩后的摘要消息不被覆盖\n- **修复自动提交仅提交主工作区** — `doAutoCommit` 遍历所有工作区执行 git add + commit\n- **修复 idx 空洞导致消息跳过持久化** — `PersistNewMessages` 内部不再跳过 System/User 消息，确保序号连续\n\n---\n\n## 1.0.4 — 2026-07-17\n\n### 新增\n- **技能状态三级配置** — 技能可设为「关闭 / 按需加载 / 始终激活」三种模式，灵活控制 AI 行为\n- **市场安装范围选择** — 安装 MCP 服务器或技能时，支持选择 user（全局）或 project（项目级）范围\n\n### 改进\n- **对话历史持久化增强** — 页面刷新后对话完整恢复，不再因浏览器关闭丢失上下文；后端全面接管消息状态管理，前端不再依赖本地缓存\n- **消息展示优化** — 连续同一角色的消息自动合并显示（如多个 assistant 回复合并为一条），阅读更流畅\n- **停止信号可靠性提升** — Agent 异常结束或用户主动停止时，前端能可靠收到停止信号并更新 UI 状态\n\n### 修复\n- 修复切换对话时 loading 状态卡死的问题（switchConv 提前放行占位消息）\n- 修复消息历史顺序错乱和思考链（reasoning_content）丢失的严重问题\n- 修复 MergeConsecutiveAssistants 跳过 RoleTool 消息导致工具调用结果不完整的问题\n\n---\n\n## 1.0.3 — 2026-07-17\n\n### 改进\n- **子进程窗口管理** — 所有后台子进程（Git 操作、BUG 检测编译/测试、Lua 工具执行、桥接命令）统一隐藏控制台窗口，避免黑框闪烁\n- **会话持久化** — OnBatchPersist 回调从"每 5 轮"改为"每轮迭代"写盘，降低异常丢失风险\n- **代码搜索提示修复** — codegraph 搜索无结果时正确显示查询内容而非空占位符\n\n### 修复\n- **PersistNewMessages idx 空洞 bug** — 修复因跳过 System/User 角色消息导致消息序号不连续、后续消息无法正确持久化的严重问题（db_store.go + db_adapter.go）\n\n---\n\n## 1.0.2 — 2026-07-16\n\n### 改进\n- **文档同步** — features.md 同步到最新版本，移除冗余的"版本信息与更新日志"章节\n\n---\n\n## 1.0.1 — 2026-07-11\n\n### 新增\n- **更新日志页面** — 帮助文档中新增更新日志页面，版本历史一目了然\n- **WebSocket 协议文档** — API 文档补充完整 WebSocket 事件类型与负载定义\n- **系统版本报告** — `/api/system/info` 现在返回 `version` 字段，前端"关于"面板同步显示\n\n### 改进\n- **API 文档全面重写** — 每个接口增加请求体 JSON Schema、响应示例和错误码说明，便于二次开发\n- **帮助文档重构** — 文档归入"文档中心"分类，导航更清晰\n\n---\n\n## 1.0.0 — 2026-07-01\n\n### 新增\n- **AI 对话编程** — 用自然语言驱动 AI 读写文件、执行命令、管理 Git\n- **自主 Agent 模式** — AI 自动分析项目、制定计划并执行多步骤任务\n- **代码编辑器** — 内置多标签页编辑器，支持语法高亮、代码折叠、十六进制查看\n- **文件管理** — 工作区目录树浏览、文件搜索、批量操作\n- **Git 版本控制** — 对话驱动的 Git 操作（状态查看、暂存、提交、分支管理）\n- **内置终端** — 浏览器中的终端面板，支持 AI 自动执行命令\n- **对话历史管理** — 自动保存、回溯与继续历史对话\n- **BUG 自动检测修复** — AI 扫描编译/测试问题并自动修复\n- **Skills / MCP 扩展** — 可复用的工作流模板和模型上下文协议扩展\n- **记忆系统** — AI 跨会话记住用户偏好和历史决策\n- **任务与规划管理** — 复杂任务分解为可追踪的子步骤\n- **Lua 自定义工具** — 通过 Lua 脚本创建自定义 AI 工具\n- **代码知识图谱** — 函数调用关系、类型层次、影响范围分析\n- **多模型支持** — 灵活切换 AI 模型后端（OpenAI / Claude 等）\n- **主题系统** — 四套预设主题（暗色、白色、暖色、暗夜紫）\n- **调试器** — 支持 Go 程序的断点、单步和变量查看\n- **网页验证工具** — 自动打开 URL、截图、分析页面效果\n- **办公文档处理** — 读取 Word / Excel / PDF 文件，支持 OCR\n\n### 技术架构\n- 后端使用 Go 语言，前端使用 Vue 3 + CodeMirror\n- WebSocket 实时推送 AI 事件流\n- 内嵌前端资源（go:embed），单二进制分发\n- 纯本地运行，所有 API 仅监听本地回环地址\n';function me(){return{async:!1,breaks:!1,extensions:null,gfm:!0,hooks:null,pedantic:!1,renderer:null,silent:!1,tokenizer:null,walkTokens:null}}var Q=me();function Se(r){Q=r}var X={exec:()=>null};function v(r){let t=[];return n=>{let o=Math.max(0,Math.min(3,n-1)),l=t[o];return l||(l=r(o),t[o]=l),l}}function C(r,t=""){let n=typeof r=="string"?r:r.source,o={replace:(l,s)=>{let a=typeof s=="string"?s:s.source;return a=a.replace(O.caret,"$1"),n=n.replace(l,a),o},getRegex:()=>new RegExp(n,t)};return o}var so=((r="")=>{try{return!!new RegExp("(?<=1)(?<!1)"+r)}catch{return!1}})(),O={codeRemoveIndent:/^(?: {1,4}| {0,3}\t)/gm,outputLinkReplace:/\\([\[\]])/g,indentCodeCompensation:/^(\s+)(?:```)/,beginningSpace:/^\s+/,endingHash:/#$/,startingSpaceChar:/^ /,endingSpaceChar:/ $/,nonSpaceChar:/[^ ]/,newLineCharGlobal:/\n/g,tabCharGlobal:/\t/g,multipleSpaceGlobal:/\s+/g,blankLine:/^[ \t]*$/,doubleBlankLine:/\n[ \t]*\n[ \t]*$/,blockquoteStart:/^ {0,3}>/,blockquoteSetextReplace:/\n {0,3}((?:=+|-+) *)(?=\n|$)/g,blockquoteSetextReplace2:/^ {0,3}>[ \t]?/gm,listReplaceNesting:/^ {1,4}(?=( {4})*[^ ])/g,listIsTask:/^\[[ xX]\] +\S/,listReplaceTask:/^\[[ xX]\] +/,listTaskCheckbox:/\[[ xX]\]/,anyLine:/\n.*\n/,hrefBrackets:/^<(.*)>$/,tableDelimiter:/[:|]/,tableAlignChars:/^\||\| *$/g,tableRowBlankLine:/\n[ \t]*$/,tableAlignRight:/^ *-+: *$/,tableAlignCenter:/^ *:-+: *$/,tableAlignLeft:/^ *:-+ *$/,startATag:/^<a /i,endATag:/^<\/a>/i,startPreScriptTag:/^<(pre|code|kbd|script)(\s|>)/i,endPreScriptTag:/^<\/(pre|code|kbd|script)(\s|>)/i,startAngleBracket:/^</,endAngleBracket:/>$/,pedanticHrefTitle:/^([^'"]*[^\s])\s+(['"])(.*)\2/,unicodeAlphaNumeric:/[\p{L}\p{N}]/u,escapeTest:/[&<>"']/,escapeReplace:/[&<>"']/g,escapeTestNoEncode:/[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/,escapeReplaceNoEncode:/[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/g,caret:/(^|[^\[])\^/g,percentDecode:/%25/g,findPipe:/\|/g,splitPipe:/ \|/,slashPipe:/\\\|/g,carriageReturn:/\r\n|\r/g,spaceLine:/^ +$/gm,notSpaceStart:/^\S*/,endingNewline:/\n$/,listItemRegex:r=>new RegExp(`^( {0,3}${r})((?:[	 ][^\\n]*)?(?:\\n|$))`),nextBulletRegex:v(r=>new RegExp(`^ {0,${r}}(?:[*+-]|\\d{1,9}[.)])((?:[ 	][^\\n]*)?(?:\\n|$))`)),hrRegex:v(r=>new RegExp(`^ {0,${r}}((?:- *){3,}|(?:_ *){3,}|(?:\\* *){3,})(?:\\n+|$)`)),fencesBeginRegex:v(r=>new RegExp(`^ {0,${r}}(?:\`\`\`|~~~)`)),headingBeginRegex:v(r=>new RegExp(`^ {0,${r}}#`)),htmlBeginRegex:v(r=>new RegExp(`^ {0,${r}}<(?:[a-z].*>|!--)`,"i")),blockquoteBeginRegex:v(r=>new RegExp(`^ {0,${r}}>`))},io=/^(?:[ \t]*(?:\n|$))+/,co=/^((?: {4}| {0,3}\t)[^\n]+(?:\n(?:[ \t]*(?:\n|$))*)?)+/,po=/^ {0,3}(`{3,}(?=[^`\n]*(?:\n|$))|~{3,})([^\n]*)(?:\n|$)(?:|([\s\S]*?)(?:\n|$))(?: {0,3}\1[~`]* *(?=\n|$)|$)/,ne=/^ {0,3}((?:-[\t ]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})(?:\n+|$)/,mo=/^ {0,3}(#{1,6})(?=\s|$)(.*)(?:\n+|$)/,ge=/ {0,3}(?:[*+-]|\d{1,9}[.)])/,Ce=/^(?!bull |blockCode|fences|blockquote|heading|html|table)((?:.|\n(?!\s*?\n|bull |blockCode|fences|blockquote|heading|html|table))+?)\n {0,3}(=+|-+) *(?:\n+|$)/,Ie=C(Ce).replace(/bull/g,ge).replace(/blockCode/g,/(?: {4}| {0,3}\t)/).replace(/fences/g,/ {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g,/ {0,3}>/).replace(/heading/g,/ {0,3}#{1,6}/).replace(/html/g,/ {0,3}<[^\n>]+>\n/).replace(/\|table/g,"").getRegex(),go=C(Ce).replace(/bull/g,ge).replace(/blockCode/g,/(?: {4}| {0,3}\t)/).replace(/fences/g,/ {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g,/ {0,3}>/).replace(/heading/g,/ {0,3}#{1,6}/).replace(/html/g,/ {0,3}<[^\n>]+>\n/).replace(/table/g,/ {0,3}\|?(?:[:\- ]*\|)+[\:\- ]*\n/).getRegex(),ke=/^([^\n]+(?:\n(?!hr|heading|lheading|blockquote|fences|list|html|table| +\n)[^\n]+)*)/,ko=/^[^\n]+/,he=/(?!\s*\])(?:\\[\s\S]|[^\[\]\\])+/,ho=C(/^ {0,3}\[(label)\]: *(?:\n[ \t]*)?([^<\s][^\s]*|<.*?>)(?:(?: +(?:\n[ \t]*)?| *\n[ \t]*)(title))? *(?:\n+|$)/).replace("label",he).replace("title",/(?:"(?:\\"?|[^"\\])*"|'[^'\n]*(?:\n[^'\n]+)*\n?'|\([^()]*\))/).getRegex(),fo=C(/^(bull)([ \t][^\n]*?)?(?:\n|$)/).replace(/bull/g,ge).getRegex(),oe="address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul",fe=/<!--(?:-?>|[\s\S]*?(?:-->|$))/,uo=C("^ {0,3}(?:<(script|pre|style|textarea)[\\s>][\\s\\S]*?(?:</\\1>[^\\n]*\\n+|$)|comment[^\\n]*(\\n+|$)|<\\?[\\s\\S]*?(?:\\?>\\n*|$)|<![A-Z][\\s\\S]*?(?:>\\n*|$)|<!\\[CDATA\\[[\\s\\S]*?(?:\\]\\]>\\n*|$)|</?(tag)(?: +|\\n|/?>)[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|<(?!script|pre|style|textarea)([a-z][\\w-]*)(?:attribute)*? */?>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|</(?!script|pre|style|textarea)[a-z][\\w-]*\\s*>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$))","i").replace("comment",fe).replace("tag",oe).replace("attribute",/ +[a-zA-Z:_][\w.:-]*(?: *= *"[^"\n]*"| *= *'[^'\n]*'| *= *[^\s"'=<>`]+)?/).getRegex(),Pe=C(ke).replace("hr",ne).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("|lheading","").replace("|table","").replace("blockquote"," {0,3}>").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",oe).getRegex(),yo=C(/^( {0,3}> ?(paragraph|[^\n]*)(?:\n|$))+/).replace("paragraph",Pe).getRegex(),ue={blockquote:yo,code:co,def:ho,fences:po,heading:mo,hr:ne,html:uo,lheading:Ie,list:fo,newline:io,paragraph:Pe,table:X,text:ko},Ae=C("^ *([^\\n ].*)\\n {0,3}((?:\\| *)?:?-+:? *(?:\\| *:?-+:? *)*(?:\\| *)?)(?:\\n((?:(?! *\\n|hr|heading|blockquote|code|fences|list|html).*(?:\\n|$))*)\\n*|$)").replace("hr",ne).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("blockquote"," {0,3}>").replace("code","(?: {4}| {0,3}	)[^\\n]").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",oe).getRegex(),bo={...ue,lheading:go,table:Ae,paragraph:C(ke).replace("hr",ne).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("|lheading","").replace("table",Ae).replace("blockquote"," {0,3}>").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",oe).getRegex()},Eo={...ue,html:C(`^ *(?:comment *(?:\\n|\\s*$)|<(tag)[\\s\\S]+?</\\1> *(?:\\n{2,}|\\s*$)|<tag(?:"[^"]*"|'[^']*'|\\s[^'"/>\\s]*)*?/?> *(?:\\n{2,}|\\s*$))`).replace("comment",fe).replace(/tag/g,"(?!(?:a|em|strong|small|s|cite|q|dfn|abbr|data|time|code|var|samp|kbd|sub|sup|i|b|u|mark|ruby|rt|rp|bdi|bdo|span|br|wbr|ins|del|img)\\b)\\w+(?!:|[^\\w\\s@]*@)\\b").getRegex(),def:/^ *\[([^\]]+)\]: *<?([^\s>]+)>?(?: +(["(][^\n]+[")]))? *(?:\n+|$)/,heading:/^(#{1,6})(.*)(?:\n+|$)/,fences:X,lheading:/^(.+?)\n {0,3}(=+|-+) *(?:\n+|$)/,paragraph:C(ke).replace("hr",ne).replace("heading",` *#{1,6} *[^
]`).replace("lheading",Ie).replace("|table","").replace("blockquote"," {0,3}>").replace("|fences","").replace("|list","").replace("|html","").replace("|tag","").getRegex()},xo=/^\\([!"#$%&'()*+,\-./:;<=>?@\[\]\\^_`{|}~])/,Vo=/^(`+)([^`]|[^`][\s\S]*?[^`])\1(?!`)/,Me=/^( {2,}|\\)\n(?!\s*$)/,No=/^(`+|[^`])(?:(?= {2,}\n)|[\s\S]*?(?:(?=[\\<!\[`*_]|\b_|$)|[^ ](?= {2,}\n)))/,ee=/[\p{P}\p{S}]/u,le=/[\s\p{P}\p{S}]/u,ye=/[^\s\p{P}\p{S}]/u,wo=C(/^((?![*_])punctSpace)/,"u").replace(/punctSpace/g,le).getRegex(),$e=/(?!~)[\p{P}\p{S}]/u,Bo=/(?!~)[\s\p{P}\p{S}]/u,To=/(?:[^\s\p{P}\p{S}]|~)/u,So=C(/link|precode-code|html/,"g").replace("link",/\[(?:[^\[\]`]|(?<a>`+)[^`]+\k<a>(?!`))*?\]\((?:\\[\s\S]|[^\\\(\)]|\((?:\\[\s\S]|[^\\\(\)])*\))*\)/).replace("precode-",so?"(?<!`)()":"(^^|[^`])").replace("code",/(?<b>`+)[^`]+\k<b>(?!`)/).replace("html",/<(?! )[^<>]*?>/).getRegex(),De=/^(?:\*+(?:((?!\*)punct)|([^\s*]))?)|^_+(?:((?!_)punct)|([^\s_]))?/,Co=C(De,"u").replace(/punct/g,ee).getRegex(),Io=C(De,"u").replace(/punct/g,$e).getRegex(),Re="^[^_*]*?__[^_*]*?\\*[^_*]*?(?=__)|[^*]+(?=[^*])|(?!\\*)punct(\\*+)(?=[\\s]|$)|notPunctSpace(\\*+)(?!\\*)(?=punctSpace|$)|(?!\\*)punctSpace(\\*+)(?=notPunctSpace)|[\\s](\\*+)(?!\\*)(?=punct)|(?!\\*)punct(\\*+)(?!\\*)(?=punct)|notPunctSpace(\\*+)(?=notPunctSpace)",Po=C(Re,"gu").replace(/notPunctSpace/g,ye).replace(/punctSpace/g,le).replace(/punct/g,ee).getRegex(),Ao=C(Re,"gu").replace(/notPunctSpace/g,To).replace(/punctSpace/g,Bo).replace(/punct/g,$e).getRegex(),Mo=C("^[^_*]*?\\*\\*[^_*]*?_[^_*]*?(?=\\*\\*)|[^_]+(?=[^_])|(?!_)punct(_+)(?=[\\s]|$)|notPunctSpace(_+)(?!_)(?=punctSpace|$)|(?!_)punctSpace(_+)(?=notPunctSpace)|[\\s](_+)(?!_)(?=punct)|(?!_)punct(_+)(?!_)(?=punct)","gu").replace(/notPunctSpace/g,ye).replace(/punctSpace/g,le).replace(/punct/g,ee).getRegex(),$o=C(/^~~?(?:((?!~)punct)|[^\s~])/,"u").replace(/punct/g,ee).getRegex(),Do="^[^~]+(?=[^~])|(?!~)punct(~~?)(?=[\\s]|$)|notPunctSpace(~~?)(?!~)(?=punctSpace|$)|(?!~)punctSpace(~~?)(?=notPunctSpace)|[\\s](~~?)(?!~)(?=punct)|(?!~)punct(~~?)(?!~)(?=punct)|notPunctSpace(~~?)(?=notPunctSpace)",Ro=C(Do,"gu").replace(/notPunctSpace/g,ye).replace(/punctSpace/g,le).replace(/punct/g,ee).getRegex(),Lo=C(/\\(punct)/,"gu").replace(/punct/g,ee).getRegex(),Fo=C(/^<(scheme:[^\s\x00-\x1f<>]*|email)>/).replace("scheme",/[a-zA-Z][a-zA-Z0-9+.-]{1,31}/).replace("email",/[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+(@)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+(?![-_])/).getRegex(),Go=C(fe).replace("(?:-->|$)","-->").getRegex(),Oo=C("^comment|^</[a-zA-Z][\\w:-]*\\s*>|^<[a-zA-Z][\\w-]*(?:attribute)*?\\s*/?>|^<\\?[\\s\\S]*?\\?>|^<![a-zA-Z]+\\s[\\s\\S]*?>|^<!\\[CDATA\\[[\\s\\S]*?\\]\\]>").replace("comment",Go).replace("attribute",/\s+[a-zA-Z:_][\w.:-]*(?:\s*=\s*"[^"]*"|\s*=\s*'[^']*'|\s*=\s*[^\s"'=<>`]+)?/).getRegex(),ae=/(?:\[(?:\\[\s\S]|[^\[\]\\])*\]|\\[\s\S]|`+(?!`)[^`]*?`+(?!`)|``+(?=\])|[^\[\]\\`])*?/,Uo=C(/^!?\[(label)\]\(\s*(href)(?:(?:[ \t]+(?:\n[ \t]*)?|\n[ \t]*)(title))?\s*\)/).replace("label",ae).replace("href",/<(?:\\.|[^\n<>\\])+>|[^ \t\n\x00-\x1f]*/).replace("title",/"(?:\\"?|[^"\\])*"|'(?:\\'?|[^'\\])*'|\((?:\\\)?|[^)\\])*\)/).getRegex(),Le=C(/^!?\[(label)\]\[(ref)\]/).replace("label",ae).replace("ref",he).getRegex(),Fe=C(/^!?\[(ref)\](?:\[\])?/).replace("ref",he).getRegex(),zo=C("reflink|nolink(?!\\()","g").replace("reflink",Le).replace("nolink",Fe).getRegex(),Ge=/[hH][tT][tT][pP][sS]?|[fF][tT][pP]/,be={_backpedal:X,anyPunctuation:Lo,autolink:Fo,blockSkip:So,br:Me,code:Vo,del:X,delLDelim:X,delRDelim:X,emStrongLDelim:Co,emStrongRDelimAst:Po,emStrongRDelimUnd:Mo,escape:xo,link:Uo,nolink:Fe,punctuation:wo,reflink:Le,reflinkSearch:zo,tag:Oo,text:No,url:X},jo={...be,link:C(/^!?\[(label)\]\((.*?)\)/).replace("label",ae).getRegex(),reflink:C(/^!?\[(label)\]\s*\[([^\]]*)\]/).replace("label",ae).getRegex()},Ee={...be,emStrongRDelimAst:Ao,emStrongLDelim:Io,delLDelim:$o,delRDelim:Ro,url:C(/^((?:protocol):\/\/|www\.)(?:[a-zA-Z0-9\-]+\.?)+[^\s<]*|^email/).replace("protocol",Ge).replace("email",/[A-Za-z0-9._+-]+(@)[a-zA-Z0-9-_]+(?:\.[a-zA-Z0-9-_]*[a-zA-Z0-9])+(?![-_])/).getRegex(),_backpedal:/(?:[^?!.,:;*_'"~()&]+|\([^)]*\)|&(?![a-zA-Z0-9]+;$)|[?!.,:;*_'"~)]+(?!$))+/,del:/^(~~?)(?=[^\s~])((?:\\[\s\S]|[^\\])*?(?:\\[\s\S]|[^\s~\\]))\1(?=[^~]|$)/,text:C(/^([`~]+|[^`~])(?:(?= {2,}\n)|(?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)|[\s\S]*?(?:(?=[\\<!\[`*~_]|\b_|protocol:\/\/|www\.|$)|[^ ](?= {2,}\n)|[^a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-](?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)))/).replace("protocol",Ge).getRegex()},Ho={...Ee,br:C(Me).replace("{2,}","*").getRegex(),text:C(Ee.text).replace("\\b_","\\b_| {2,}\\n").replace(/\{2,\}/g,"*").getRegex()},se={normal:ue,gfm:bo,pedantic:Eo},te={normal:be,gfm:Ee,breaks:Ho,pedantic:jo},Wo={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"},Oe=r=>Wo[r];function J(r,t){if(t){if(O.escapeTest.test(r))return r.replace(O.escapeReplace,Oe)}else if(O.escapeTestNoEncode.test(r))return r.replace(O.escapeReplaceNoEncode,Oe);return r}function Ue(r){try{r=encodeURI(r).replace(O.percentDecode,"%")}catch{return null}return r}function ze(r,t){var s;let n=r.replace(O.findPipe,(a,m,c)=>{let u=!1,h=m;for(;--h>=0&&c[h]==="\\";)u=!u;return u?"|":" |"}),o=n.split(O.splitPipe),l=0;if(o[0].trim()||o.shift(),o.length>0&&!((s=o.at(-1))!=null&&s.trim())&&o.pop(),t)if(o.length>t)o.splice(t);else for(;o.length<t;)o.push("");for(;l<o.length;l++)o[l]=o[l].trim().replace(O.slashPipe,"|");return o}function Z(r,t,n){let o=r.length;if(o===0)return"";let l=0;for(;l<o&&r.charAt(o-l-1)===t;)l++;return r.slice(0,o-l)}function je(r){let t=r.split(`
`),n=t.length-1;for(;n>=0&&O.blankLine.test(t[n]);)n--;return t.length-n<=2?r:t.slice(0,n+1).join(`
`)}function qo(r,t){if(r.indexOf(t[1])===-1)return-1;let n=0;for(let o=0;o<r.length;o++)if(r[o]==="\\")o++;else if(r[o]===t[0])n++;else if(r[o]===t[1]&&(n--,n<0))return o;return n>0?-2:-1}function Jo(r,t=0){let n=t,o="";for(let l of r)if(l==="	"){let s=4-n%4;o+=" ".repeat(s),n+=s}else o+=l,n++;return o}function He(r,t,n,o,l){let s=t.href,a=t.title||null,m=r[1].replace(l.other.outputLinkReplace,"$1");o.state.inLink=!0;let c={type:r[0].charAt(0)==="!"?"image":"link",raw:n,href:s,title:a,text:m,tokens:o.inlineTokens(m)};return o.state.inLink=!1,c}function Ko(r,t,n){let o=r.match(n.other.indentCodeCompensation);if(o===null)return t;let l=o[1];return t.split(`
`).map(s=>{let a=s.match(n.other.beginningSpace);if(a===null)return s;let[m]=a;return m.length>=l.length?s.slice(l.length):s}).join(`
`)}var ie=class{constructor(r){D(this,"options");D(this,"rules");D(this,"lexer");this.options=r||Q}space(r){let t=this.rules.block.newline.exec(r);if(t&&t[0].length>0)return{type:"space",raw:t[0]}}code(r){let t=this.rules.block.code.exec(r);if(t){let n=this.options.pedantic?t[0]:je(t[0]),o=n.replace(this.rules.other.codeRemoveIndent,"");return{type:"code",raw:n,codeBlockStyle:"indented",text:o}}}fences(r){let t=this.rules.block.fences.exec(r);if(t){let n=t[0],o=Ko(n,t[3]||"",this.rules);return{type:"code",raw:n,lang:t[2]?t[2].trim().replace(this.rules.inline.anyPunctuation,"$1"):t[2],text:o}}}heading(r){let t=this.rules.block.heading.exec(r);if(t){let n=t[2].trim();if(this.rules.other.endingHash.test(n)){let o=Z(n,"#");(this.options.pedantic||!o||this.rules.other.endingSpaceChar.test(o))&&(n=o.trim())}return{type:"heading",raw:Z(t[0],`
`),depth:t[1].length,text:n,tokens:this.lexer.inline(n)}}}hr(r){let t=this.rules.block.hr.exec(r);if(t)return{type:"hr",raw:Z(t[0],`
`)}}blockquote(r){let t=this.rules.block.blockquote.exec(r);if(t){let n=Z(t[0],`
`).split(`
`),o="",l="",s=[];for(;n.length>0;){let a=!1,m=[],c;for(c=0;c<n.length;c++)if(this.rules.other.blockquoteStart.test(n[c]))m.push(n[c]),a=!0;else if(!a)m.push(n[c]);else break;n=n.slice(c);let u=m.join(`
`),h=u.replace(this.rules.other.blockquoteSetextReplace,`
    $1`).replace(this.rules.other.blockquoteSetextReplace2,"");o=o?`${o}
${u}`:u,l=l?`${l}
${h}`:h;let w=this.lexer.state.top;if(this.lexer.state.top=!0,this.lexer.blockTokens(h,s,!0),this.lexer.state.top=w,n.length===0)break;let V=s.at(-1);if((V==null?void 0:V.type)==="code")break;if((V==null?void 0:V.type)==="blockquote"){let T=V,y=T.raw+`
`+n.join(`
`),B=this.blockquote(y);s[s.length-1]=B,o=o.substring(0,o.length-T.raw.length)+B.raw,l=l.substring(0,l.length-T.text.length)+B.text;break}else if((V==null?void 0:V.type)==="list"){let T=V,y=T.raw+`
`+n.join(`
`),B=this.list(y);s[s.length-1]=B,o=o.substring(0,o.length-V.raw.length)+B.raw,l=l.substring(0,l.length-T.raw.length)+B.raw,n=y.substring(s.at(-1).raw.length).split(`
`);continue}}return{type:"blockquote",raw:o,tokens:s,text:l}}}list(r){let t=this.rules.block.list.exec(r);if(t){let n=t[1].trim(),o=n.length>1,l={type:"list",raw:"",ordered:o,start:o?+n.slice(0,-1):"",loose:!1,items:[]};n=o?`\\d{1,9}\\${n.slice(-1)}`:`\\${n}`,this.options.pedantic&&(n=o?n:"[*+-]");let s=this.rules.other.listItemRegex(n),a=!1;for(;r;){let c=!1,u="",h="";if(!(t=s.exec(r))||this.rules.block.hr.test(r))break;u=t[0],r=r.substring(u.length);let w=Jo(t[2].split(`
`,1)[0],t[1].length),V=r.split(`
`,1)[0],T=!w.trim(),y=0;if(this.options.pedantic?(y=2,h=w.trimStart()):T?y=t[1].length+1:(y=w.search(this.rules.other.nonSpaceChar),y=y>4?1:y,h=w.slice(y),y+=t[1].length),T&&this.rules.other.blankLine.test(V)&&(u+=V+`
`,r=r.substring(V.length+1),c=!0),!c){let B=this.rules.other.nextBulletRegex(y),I=this.rules.other.hrRegex(y),L=this.rules.other.fencesBeginRegex(y),F=this.rules.other.headingBeginRegex(y),U=this.rules.other.htmlBeginRegex(y),H=this.rules.other.blockquoteBeginRegex(y);for(;r;){let j=r.split(`
`,1)[0],G;if(V=j,this.options.pedantic?(V=V.replace(this.rules.other.listReplaceNesting,"  "),G=V):G=V.replace(this.rules.other.tabCharGlobal,"    "),L.test(V)||F.test(V)||U.test(V)||H.test(V)||B.test(V)||I.test(V))break;if(G.search(this.rules.other.nonSpaceChar)>=y||!V.trim())h+=`
`+G.slice(y);else{if(T||w.replace(this.rules.other.tabCharGlobal,"    ").search(this.rules.other.nonSpaceChar)>=4||L.test(w)||F.test(w)||I.test(w))break;h+=`
`+V}T=!V.trim(),u+=j+`
`,r=r.substring(j.length+1),w=G.slice(y)}}l.loose||(a?l.loose=!0:this.rules.other.doubleBlankLine.test(u)&&(a=!0)),l.items.push({type:"list_item",raw:u,task:!!this.options.gfm&&this.rules.other.listIsTask.test(h),loose:!1,text:h,tokens:[]}),l.raw+=u}let m=l.items.at(-1);if(m)m.raw=m.raw.trimEnd(),m.text=m.text.trimEnd();else return;l.raw=l.raw.trimEnd();for(let c of l.items){this.lexer.state.top=!1,c.tokens=this.lexer.blockTokens(c.text,[]);let u=c.tokens[0];if(c.task&&((u==null?void 0:u.type)==="text"||(u==null?void 0:u.type)==="paragraph")){c.text=c.text.replace(this.rules.other.listReplaceTask,""),u.raw=u.raw.replace(this.rules.other.listReplaceTask,""),u.text=u.text.replace(this.rules.other.listReplaceTask,"");for(let w=this.lexer.inlineQueue.length-1;w>=0;w--)if(this.rules.other.listIsTask.test(this.lexer.inlineQueue[w].src)){this.lexer.inlineQueue[w].src=this.lexer.inlineQueue[w].src.replace(this.rules.other.listReplaceTask,"");break}let h=this.rules.other.listTaskCheckbox.exec(c.raw);if(h){let w={type:"checkbox",raw:h[0]+" ",checked:h[0]!=="[ ]"};c.checked=w.checked,l.loose?c.tokens[0]&&["paragraph","text"].includes(c.tokens[0].type)&&"tokens"in c.tokens[0]&&c.tokens[0].tokens?(c.tokens[0].raw=w.raw+c.tokens[0].raw,c.tokens[0].text=w.raw+c.tokens[0].text,c.tokens[0].tokens.unshift(w)):c.tokens.unshift({type:"paragraph",raw:w.raw,text:w.raw,tokens:[w]}):c.tokens.unshift(w)}}else c.task&&(c.task=!1);if(!l.loose){let h=c.tokens.filter(V=>V.type==="space"),w=h.length>0&&h.some(V=>this.rules.other.anyLine.test(V.raw));l.loose=w}}if(l.loose)for(let c of l.items){c.loose=!0;for(let u of c.tokens)u.type==="text"&&(u.type="paragraph")}return l}}html(r){let t=this.rules.block.html.exec(r);if(t){let n=je(t[0]);return{type:"html",block:!0,raw:n,pre:t[1]==="pre"||t[1]==="script"||t[1]==="style",text:n}}}def(r){let t=this.rules.block.def.exec(r);if(t){let n=t[1].toLowerCase().replace(this.rules.other.multipleSpaceGlobal," "),o=t[2]?t[2].replace(this.rules.other.hrefBrackets,"$1").replace(this.rules.inline.anyPunctuation,"$1"):"",l=t[3]?t[3].substring(1,t[3].length-1).replace(this.rules.inline.anyPunctuation,"$1"):t[3];return{type:"def",tag:n,raw:Z(t[0],`
`),href:o,title:l}}}table(r){var a;let t=this.rules.block.table.exec(r);if(!t||!this.rules.other.tableDelimiter.test(t[2]))return;let n=ze(t[1]),o=t[2].replace(this.rules.other.tableAlignChars,"").split("|"),l=(a=t[3])!=null&&a.trim()?t[3].replace(this.rules.other.tableRowBlankLine,"").split(`
`):[],s={type:"table",raw:Z(t[0],`
`),header:[],align:[],rows:[]};if(n.length===o.length){for(let m of o)this.rules.other.tableAlignRight.test(m)?s.align.push("right"):this.rules.other.tableAlignCenter.test(m)?s.align.push("center"):this.rules.other.tableAlignLeft.test(m)?s.align.push("left"):s.align.push(null);for(let m=0;m<n.length;m++)s.header.push({text:n[m],tokens:this.lexer.inline(n[m]),header:!0,align:s.align[m]});for(let m of l)s.rows.push(ze(m,s.header.length).map((c,u)=>({text:c,tokens:this.lexer.inline(c),header:!1,align:s.align[u]})));return s}}lheading(r){let t=this.rules.block.lheading.exec(r);if(t){let n=t[1].trim();return{type:"heading",raw:Z(t[0],`
`),depth:t[2].charAt(0)==="="?1:2,text:n,tokens:this.lexer.inline(n)}}}paragraph(r){let t=this.rules.block.paragraph.exec(r);if(t){let n=t[1].charAt(t[1].length-1)===`
`?t[1].slice(0,-1):t[1];return{type:"paragraph",raw:t[0],text:n,tokens:this.lexer.inline(n)}}}text(r){let t=this.rules.block.text.exec(r);if(t)return{type:"text",raw:t[0],text:t[0],tokens:this.lexer.inline(t[0])}}escape(r){let t=this.rules.inline.escape.exec(r);if(t)return{type:"escape",raw:t[0],text:t[1]}}tag(r){let t=this.rules.inline.tag.exec(r);if(t)return!this.lexer.state.inLink&&this.rules.other.startATag.test(t[0])?this.lexer.state.inLink=!0:this.lexer.state.inLink&&this.rules.other.endATag.test(t[0])&&(this.lexer.state.inLink=!1),!this.lexer.state.inRawBlock&&this.rules.other.startPreScriptTag.test(t[0])?this.lexer.state.inRawBlock=!0:this.lexer.state.inRawBlock&&this.rules.other.endPreScriptTag.test(t[0])&&(this.lexer.state.inRawBlock=!1),{type:"html",raw:t[0],inLink:this.lexer.state.inLink,inRawBlock:this.lexer.state.inRawBlock,block:!1,text:t[0]}}link(r){let t=this.rules.inline.link.exec(r);if(t){let n=t[2].trim();if(!this.options.pedantic&&this.rules.other.startAngleBracket.test(n)){if(!this.rules.other.endAngleBracket.test(n))return;let s=Z(n.slice(0,-1),"\\");if((n.length-s.length)%2===0)return}else{let s=qo(t[2],"()");if(s===-2)return;if(s>-1){let a=(t[0].indexOf("!")===0?5:4)+t[1].length+s;t[2]=t[2].substring(0,s),t[0]=t[0].substring(0,a).trim(),t[3]=""}}let o=t[2],l="";if(this.options.pedantic){let s=this.rules.other.pedanticHrefTitle.exec(o);s&&(o=s[1],l=s[3])}else l=t[3]?t[3].slice(1,-1):"";return o=o.trim(),this.rules.other.startAngleBracket.test(o)&&(this.options.pedantic&&!this.rules.other.endAngleBracket.test(n)?o=o.slice(1):o=o.slice(1,-1)),He(t,{href:o&&o.replace(this.rules.inline.anyPunctuation,"$1"),title:l&&l.replace(this.rules.inline.anyPunctuation,"$1")},t[0],this.lexer,this.rules)}}reflink(r,t){let n;if((n=this.rules.inline.reflink.exec(r))||(n=this.rules.inline.nolink.exec(r))){let o=(n[2]||n[1]).replace(this.rules.other.multipleSpaceGlobal," "),l=t[o.toLowerCase()];if(!l){let s=n[0].charAt(0);return{type:"text",raw:s,text:s}}return He(n,l,n[0],this.lexer,this.rules)}}emStrong(r,t,n=""){let o=this.rules.inline.emStrongLDelim.exec(r);if(!(!o||!o[1]&&!o[2]&&!o[3]&&!o[4]||o[4]&&n.match(this.rules.other.unicodeAlphaNumeric))&&(!(o[1]||o[3])||!n||this.rules.inline.punctuation.exec(n))){let l=[...o[0]].length-1,s,a,m=l,c=0,u=o[0][0]==="*"?this.rules.inline.emStrongRDelimAst:this.rules.inline.emStrongRDelimUnd;for(u.lastIndex=0,t=t.slice(-1*r.length+l);(o=u.exec(t))!==null;){if(s=o[1]||o[2]||o[3]||o[4]||o[5]||o[6],!s)continue;if(a=[...s].length,o[3]||o[4]){m+=a;continue}else if((o[5]||o[6])&&l%3&&!((l+a)%3)){c+=a;continue}if(m-=a,m>0)continue;a=Math.min(a,a+m+c);let h=[...o[0]][0].length,w=r.slice(0,l+o.index+h+a);if(Math.min(l,a)%2){let T=w.slice(1,-1);return{type:"em",raw:w,text:T,tokens:this.lexer.inlineTokens(T)}}let V=w.slice(2,-2);return{type:"strong",raw:w,text:V,tokens:this.lexer.inlineTokens(V)}}}}codespan(r){let t=this.rules.inline.code.exec(r);if(t){let n=t[2].replace(this.rules.other.newLineCharGlobal," "),o=this.rules.other.nonSpaceChar.test(n),l=this.rules.other.startingSpaceChar.test(n)&&this.rules.other.endingSpaceChar.test(n);return o&&l&&(n=n.substring(1,n.length-1)),{type:"codespan",raw:t[0],text:n}}}br(r){let t=this.rules.inline.br.exec(r);if(t)return{type:"br",raw:t[0]}}del(r,t,n=""){let o=this.rules.inline.delLDelim.exec(r);if(o&&(!o[1]||!n||this.rules.inline.punctuation.exec(n))){let l=[...o[0]].length-1,s,a,m=l,c=this.rules.inline.delRDelim;for(c.lastIndex=0,t=t.slice(-1*r.length+l);(o=c.exec(t))!==null;){if(s=o[1]||o[2]||o[3]||o[4]||o[5]||o[6],!s||(a=[...s].length,a!==l))continue;if(o[3]||o[4]){m+=a;continue}if(m-=a,m>0)continue;a=Math.min(a,a+m);let u=[...o[0]][0].length,h=r.slice(0,l+o.index+u+a),w=h.slice(l,-l);return{type:"del",raw:h,text:w,tokens:this.lexer.inlineTokens(w)}}}}autolink(r){let t=this.rules.inline.autolink.exec(r);if(t){let n,o;return t[2]==="@"?(n=t[1],o="mailto:"+n):(n=t[1],o=n),{type:"link",raw:t[0],text:n,href:o,tokens:[{type:"text",raw:n,text:n}]}}}url(r){var n;let t;if(t=this.rules.inline.url.exec(r)){let o,l;if(t[2]==="@")o=t[0],l="mailto:"+o;else{let s;do s=t[0],t[0]=((n=this.rules.inline._backpedal.exec(t[0]))==null?void 0:n[0])??"";while(s!==t[0]);o=t[0],t[1]==="www."?l="http://"+t[0]:l=t[0]}return{type:"link",raw:t[0],text:o,href:l,tokens:[{type:"text",raw:o,text:o}]}}}inlineText(r){let t=this.rules.inline.text.exec(r);if(t){let n=this.lexer.state.inRawBlock;return{type:"text",raw:t[0],text:t[0],escaped:n}}}},W=class Ne{constructor(t){D(this,"tokens");D(this,"options");D(this,"state");D(this,"inlineQueue");D(this,"tokenizer");this.tokens=[],this.tokens.links=Object.create(null),this.options=t||Q,this.options.tokenizer=this.options.tokenizer||new ie,this.tokenizer=this.options.tokenizer,this.tokenizer.options=this.options,this.tokenizer.lexer=this,this.inlineQueue=[],this.state={inLink:!1,inRawBlock:!1,top:!0};let n={other:O,block:se.normal,inline:te.normal};this.options.pedantic?(n.block=se.pedantic,n.inline=te.pedantic):this.options.gfm&&(n.block=se.gfm,this.options.breaks?n.inline=te.breaks:n.inline=te.gfm),this.tokenizer.rules=n}static get rules(){return{block:se,inline:te}}static lex(t,n){return new Ne(n).lex(t)}static lexInline(t,n){return new Ne(n).inlineTokens(t)}lex(t){t=t.replace(O.carriageReturn,`
`),this.blockTokens(t,this.tokens);for(let n=0;n<this.inlineQueue.length;n++){let o=this.inlineQueue[n];this.inlineTokens(o.src,o.tokens)}return this.inlineQueue=[],this.tokens}blockTokens(t,n=[],o=!1){var s,a,m;this.tokenizer.lexer=this,this.options.pedantic&&(t=t.replace(O.tabCharGlobal,"    ").replace(O.spaceLine,""));let l=1/0;for(;t;){if(t.length<l)l=t.length;else{this.infiniteLoopError(t.charCodeAt(0));break}let c;if((a=(s=this.options.extensions)==null?void 0:s.block)!=null&&a.some(h=>(c=h.call({lexer:this},t,n))?(t=t.substring(c.raw.length),n.push(c),!0):!1))continue;if(c=this.tokenizer.space(t)){t=t.substring(c.raw.length);let h=n.at(-1);c.raw.length===1&&h!==void 0?h.raw+=`
`:n.push(c);continue}if(c=this.tokenizer.code(t)){t=t.substring(c.raw.length);let h=n.at(-1);(h==null?void 0:h.type)==="paragraph"||(h==null?void 0:h.type)==="text"?(h.raw+=(h.raw.endsWith(`
`)?"":`
`)+c.raw,h.text+=`
`+c.text,this.inlineQueue.at(-1).src=h.text):n.push(c);continue}if(c=this.tokenizer.fences(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.heading(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.hr(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.blockquote(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.list(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.html(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.def(t)){t=t.substring(c.raw.length);let h=n.at(-1);(h==null?void 0:h.type)==="paragraph"||(h==null?void 0:h.type)==="text"?(h.raw+=(h.raw.endsWith(`
`)?"":`
`)+c.raw,h.text+=`
`+c.raw,this.inlineQueue.at(-1).src=h.text):this.tokens.links[c.tag]||(this.tokens.links[c.tag]={href:c.href,title:c.title},n.push(c));continue}if(c=this.tokenizer.table(t)){t=t.substring(c.raw.length),n.push(c);continue}if(c=this.tokenizer.lheading(t)){t=t.substring(c.raw.length),n.push(c);continue}let u=t;if((m=this.options.extensions)!=null&&m.startBlock){let h=1/0,w=t.slice(1),V;this.options.extensions.startBlock.forEach(T=>{V=T.call({lexer:this},w),typeof V=="number"&&V>=0&&(h=Math.min(h,V))}),h<1/0&&h>=0&&(u=t.substring(0,h+1))}if(this.state.top&&(c=this.tokenizer.paragraph(u))){let h=n.at(-1);o&&(h==null?void 0:h.type)==="paragraph"?(h.raw+=(h.raw.endsWith(`
`)?"":`
`)+c.raw,h.text+=`
`+c.text,this.inlineQueue.pop(),this.inlineQueue.at(-1).src=h.text):n.push(c),o=u.length!==t.length,t=t.substring(c.raw.length);continue}if(c=this.tokenizer.text(t)){t=t.substring(c.raw.length);let h=n.at(-1);(h==null?void 0:h.type)==="text"?(h.raw+=(h.raw.endsWith(`
`)?"":`
`)+c.raw,h.text+=`
`+c.text,this.inlineQueue.pop(),this.inlineQueue.at(-1).src=h.text):n.push(c);continue}if(t){this.infiniteLoopError(t.charCodeAt(0));break}}return this.state.top=!0,n}inline(t,n=[]){return this.inlineQueue.push({src:t,tokens:n}),n}inlineTokens(t,n=[]){var u,h,w,V,T;this.tokenizer.lexer=this;let o=t,l=null;if(this.tokens.links){let y=Object.keys(this.tokens.links);if(y.length>0)for(;(l=this.tokenizer.rules.inline.reflinkSearch.exec(o))!==null;)y.includes(l[0].slice(l[0].lastIndexOf("[")+1,-1))&&(o=o.slice(0,l.index)+"["+"a".repeat(l[0].length-2)+"]"+o.slice(this.tokenizer.rules.inline.reflinkSearch.lastIndex))}for(;(l=this.tokenizer.rules.inline.anyPunctuation.exec(o))!==null;)o=o.slice(0,l.index)+"++"+o.slice(this.tokenizer.rules.inline.anyPunctuation.lastIndex);let s;for(;(l=this.tokenizer.rules.inline.blockSkip.exec(o))!==null;)s=l[2]?l[2].length:0,o=o.slice(0,l.index+s)+"["+"a".repeat(l[0].length-s-2)+"]"+o.slice(this.tokenizer.rules.inline.blockSkip.lastIndex);o=((h=(u=this.options.hooks)==null?void 0:u.emStrongMask)==null?void 0:h.call({lexer:this},o))??o;let a=!1,m="",c=1/0;for(;t;){if(t.length<c)c=t.length;else{this.infiniteLoopError(t.charCodeAt(0));break}a||(m=""),a=!1;let y;if((V=(w=this.options.extensions)==null?void 0:w.inline)!=null&&V.some(I=>(y=I.call({lexer:this},t,n))?(t=t.substring(y.raw.length),n.push(y),!0):!1))continue;if(y=this.tokenizer.escape(t)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.tag(t)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.link(t)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.reflink(t,this.tokens.links)){t=t.substring(y.raw.length);let I=n.at(-1);y.type==="text"&&(I==null?void 0:I.type)==="text"?(I.raw+=y.raw,I.text+=y.text):n.push(y);continue}if(y=this.tokenizer.emStrong(t,o,m)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.codespan(t)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.br(t)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.del(t,o,m)){t=t.substring(y.raw.length),n.push(y);continue}if(y=this.tokenizer.autolink(t)){t=t.substring(y.raw.length),n.push(y);continue}if(!this.state.inLink&&(y=this.tokenizer.url(t))){t=t.substring(y.raw.length),n.push(y);continue}let B=t;if((T=this.options.extensions)!=null&&T.startInline){let I=1/0,L=t.slice(1),F;this.options.extensions.startInline.forEach(U=>{F=U.call({lexer:this},L),typeof F=="number"&&F>=0&&(I=Math.min(I,F))}),I<1/0&&I>=0&&(B=t.substring(0,I+1))}if(y=this.tokenizer.inlineText(B)){t=t.substring(y.raw.length),y.raw.slice(-1)!=="_"&&(m=y.raw.slice(-1)),a=!0;let I=n.at(-1);(I==null?void 0:I.type)==="text"?(I.raw+=y.raw,I.text+=y.text):n.push(y);continue}if(t){this.infiniteLoopError(t.charCodeAt(0));break}}return n}infiniteLoopError(t){let n="Infinite loop on byte: "+t;if(this.options.silent)console.error(n);else throw new Error(n)}},ce=class{constructor(r){D(this,"options");D(this,"parser");this.options=r||Q}space(r){return""}code({text:r,lang:t,escaped:n}){var s;let o=(s=(t||"").match(O.notSpaceStart))==null?void 0:s[0],l=r.replace(O.endingNewline,"")+`
`;return o?'<pre><code class="language-'+J(o)+'">'+(n?l:J(l,!0))+`</code></pre>
`:"<pre><code>"+(n?l:J(l,!0))+`</code></pre>
`}blockquote({tokens:r}){return`<blockquote>
${this.parser.parse(r)}</blockquote>
`}html({text:r}){return r}def(r){return""}heading({tokens:r,depth:t}){return`<h${t}>${this.parser.parseInline(r)}</h${t}>
`}hr(r){return`<hr>
`}list(r){let t=r.ordered,n=r.start,o="";for(let a=0;a<r.items.length;a++){let m=r.items[a];o+=this.listitem(m)}let l=t?"ol":"ul",s=t&&n!==1?' start="'+n+'"':"";return"<"+l+s+`>
`+o+"</"+l+`>
`}listitem(r){return`<li>${this.parser.parse(r.tokens)}</li>
`}checkbox({checked:r}){return"<input "+(r?'checked="" ':"")+'disabled="" type="checkbox"> '}paragraph({tokens:r}){return`<p>${this.parser.parseInline(r)}</p>
`}table(r){let t="",n="";for(let l=0;l<r.header.length;l++)n+=this.tablecell(r.header[l]);t+=this.tablerow({text:n});let o="";for(let l=0;l<r.rows.length;l++){let s=r.rows[l];n="";for(let a=0;a<s.length;a++)n+=this.tablecell(s[a]);o+=this.tablerow({text:n})}return o&&(o=`<tbody>${o}</tbody>`),`<table>
<thead>
`+t+`</thead>
`+o+`</table>
`}tablerow({text:r}){return`<tr>
${r}</tr>
`}tablecell(r){let t=this.parser.parseInline(r.tokens),n=r.header?"th":"td";return(r.align?`<${n} align="${r.align}">`:`<${n}>`)+t+`</${n}>
`}strong({tokens:r}){return`<strong>${this.parser.parseInline(r)}</strong>`}em({tokens:r}){return`<em>${this.parser.parseInline(r)}</em>`}codespan({text:r}){return`<code>${J(r,!0)}</code>`}br(r){return"<br>"}del({tokens:r}){return`<del>${this.parser.parseInline(r)}</del>`}link({href:r,title:t,tokens:n}){let o=this.parser.parseInline(n),l=Ue(r);if(l===null)return o;r=l;let s='<a href="'+r+'"';return t&&(s+=' title="'+J(t)+'"'),s+=">"+o+"</a>",s}image({href:r,title:t,text:n,tokens:o}){o&&(n=this.parser.parseInline(o,this.parser.textRenderer));let l=Ue(r);if(l===null)return J(n);r=l;let s=`<img src="${r}" alt="${J(n)}"`;return t&&(s+=` title="${J(t)}"`),s+=">",s}text(r){return"tokens"in r&&r.tokens?this.parser.parseInline(r.tokens):"escaped"in r&&r.escaped?r.text:J(r.text)}},xe=class{strong({text:r}){return r}em({text:r}){return r}codespan({text:r}){return r}del({text:r}){return r}html({text:r}){return r}text({text:r}){return r}link({text:r}){return""+r}image({text:r}){return""+r}br(){return""}checkbox({raw:r}){return r}},q=class we{constructor(t){D(this,"options");D(this,"renderer");D(this,"textRenderer");this.options=t||Q,this.options.renderer=this.options.renderer||new ce,this.renderer=this.options.renderer,this.renderer.options=this.options,this.renderer.parser=this,this.textRenderer=new xe}static parse(t,n){return new we(n).parse(t)}static parseInline(t,n){return new we(n).parseInline(t)}parse(t){var o,l;this.renderer.parser=this;let n="";for(let s=0;s<t.length;s++){let a=t[s];if((l=(o=this.options.extensions)==null?void 0:o.renderers)!=null&&l[a.type]){let c=a,u=this.options.extensions.renderers[c.type].call({parser:this},c);if(u!==!1||!["space","hr","heading","code","table","blockquote","list","html","def","paragraph","text"].includes(c.type)){n+=u||"";continue}}let m=a;switch(m.type){case"space":{n+=this.renderer.space(m);break}case"hr":{n+=this.renderer.hr(m);break}case"heading":{n+=this.renderer.heading(m);break}case"code":{n+=this.renderer.code(m);break}case"table":{n+=this.renderer.table(m);break}case"blockquote":{n+=this.renderer.blockquote(m);break}case"list":{n+=this.renderer.list(m);break}case"checkbox":{n+=this.renderer.checkbox(m);break}case"html":{n+=this.renderer.html(m);break}case"def":{n+=this.renderer.def(m);break}case"paragraph":{n+=this.renderer.paragraph(m);break}case"text":{n+=this.renderer.text(m);break}default:{let c='Token with "'+m.type+'" type was not found.';if(this.options.silent)return console.error(c),"";throw new Error(c)}}}return n}parseInline(t,n=this.renderer){var l,s;this.renderer.parser=this;let o="";for(let a=0;a<t.length;a++){let m=t[a];if((s=(l=this.options.extensions)==null?void 0:l.renderers)!=null&&s[m.type]){let u=this.options.extensions.renderers[m.type].call({parser:this},m);if(u!==!1||!["escape","html","link","image","strong","em","codespan","br","del","text"].includes(m.type)){o+=u||"";continue}}let c=m;switch(c.type){case"escape":{o+=n.text(c);break}case"html":{o+=n.html(c);break}case"link":{o+=n.link(c);break}case"image":{o+=n.image(c);break}case"checkbox":{o+=n.checkbox(c);break}case"strong":{o+=n.strong(c);break}case"em":{o+=n.em(c);break}case"codespan":{o+=n.codespan(c);break}case"br":{o+=n.br(c);break}case"del":{o+=n.del(c);break}case"text":{o+=n.text(c);break}default:{let u='Token with "'+c.type+'" type was not found.';if(this.options.silent)return console.error(u),"";throw new Error(u)}}}return o}},re=(de=class{constructor(r){D(this,"options");D(this,"block");this.options=r||Q}preprocess(r){return r}postprocess(r){return r}processAllTokens(r){return r}emStrongMask(r){return r}provideLexer(r=this.block){return r?W.lex:W.lexInline}provideParser(r=this.block){return r?q.parse:q.parseInline}},D(de,"passThroughHooks",new Set(["preprocess","postprocess","processAllTokens","emStrongMask"])),D(de,"passThroughHooksRespectAsync",new Set(["preprocess","postprocess","processAllTokens"])),de),Zo=class{constructor(...r){D(this,"defaults",me());D(this,"options",this.setOptions);D(this,"parse",this.parseMarkdown(!0));D(this,"parseInline",this.parseMarkdown(!1));D(this,"Parser",q);D(this,"Renderer",ce);D(this,"TextRenderer",xe);D(this,"Lexer",W);D(this,"Tokenizer",ie);D(this,"Hooks",re);this.use(...r)}walkTokens(r,t){var o,l;let n=[];for(let s of r)switch(n=n.concat(t.call(this,s)),s.type){case"table":{let a=s;for(let m of a.header)n=n.concat(this.walkTokens(m.tokens,t));for(let m of a.rows)for(let c of m)n=n.concat(this.walkTokens(c.tokens,t));break}case"list":{let a=s;n=n.concat(this.walkTokens(a.items,t));break}default:{let a=s;(l=(o=this.defaults.extensions)==null?void 0:o.childTokens)!=null&&l[a.type]?this.defaults.extensions.childTokens[a.type].forEach(m=>{let c=a[m].flat(1/0);n=n.concat(this.walkTokens(c,t))}):a.tokens&&(n=n.concat(this.walkTokens(a.tokens,t)))}}return n}use(...r){let t=this.defaults.extensions||{renderers:{},childTokens:{}};return r.forEach(n=>{let o={...n};if(o.async=this.defaults.async||o.async||!1,n.extensions&&(n.extensions.forEach(l=>{if(!l.name)throw new Error("extension name required");if("renderer"in l){let s=t.renderers[l.name];s?t.renderers[l.name]=function(...a){let m=l.renderer.apply(this,a);return m===!1&&(m=s.apply(this,a)),m}:t.renderers[l.name]=l.renderer}if("tokenizer"in l){if(!l.level||l.level!=="block"&&l.level!=="inline")throw new Error("extension level must be 'block' or 'inline'");let s=t[l.level];s?s.unshift(l.tokenizer):t[l.level]=[l.tokenizer],l.start&&(l.level==="block"?t.startBlock?t.startBlock.push(l.start):t.startBlock=[l.start]:l.level==="inline"&&(t.startInline?t.startInline.push(l.start):t.startInline=[l.start]))}"childTokens"in l&&l.childTokens&&(t.childTokens[l.name]=l.childTokens)}),o.extensions=t),n.renderer){let l=this.defaults.renderer||new ce(this.defaults);for(let s in n.renderer){if(!(s in l))throw new Error(`renderer '${s}' does not exist`);if(["options","parser"].includes(s))continue;let a=s,m=n.renderer[a],c=l[a];l[a]=(...u)=>{let h=m.apply(l,u);return h===!1&&(h=c.apply(l,u)),h||""}}o.renderer=l}if(n.tokenizer){let l=this.defaults.tokenizer||new ie(this.defaults);for(let s in n.tokenizer){if(!(s in l))throw new Error(`tokenizer '${s}' does not exist`);if(["options","rules","lexer"].includes(s))continue;let a=s,m=n.tokenizer[a],c=l[a];l[a]=(...u)=>{let h=m.apply(l,u);return h===!1&&(h=c.apply(l,u)),h}}o.tokenizer=l}if(n.hooks){let l=this.defaults.hooks||new re;for(let s in n.hooks){if(!(s in l))throw new Error(`hook '${s}' does not exist`);if(["options","block"].includes(s))continue;let a=s,m=n.hooks[a],c=l[a];re.passThroughHooks.has(s)?l[a]=u=>{if(this.defaults.async&&re.passThroughHooksRespectAsync.has(s))return(async()=>{let w=await m.call(l,u);return c.call(l,w)})();let h=m.call(l,u);return c.call(l,h)}:l[a]=(...u)=>{if(this.defaults.async)return(async()=>{let w=await m.apply(l,u);return w===!1&&(w=await c.apply(l,u)),w})();let h=m.apply(l,u);return h===!1&&(h=c.apply(l,u)),h}}o.hooks=l}if(n.walkTokens){let l=this.defaults.walkTokens,s=n.walkTokens;o.walkTokens=function(a){let m=[];return m.push(s.call(this,a)),l&&(m=m.concat(l.call(this,a))),m}}this.defaults={...this.defaults,...o}}),this}setOptions(r){return this.defaults={...this.defaults,...r},this}lexer(r,t){return W.lex(r,t??this.defaults)}parser(r,t){return q.parse(r,t??this.defaults)}parseMarkdown(r){return(t,n)=>{let o={...n},l={...this.defaults,...o},s=this.onError(!!l.silent,!!l.async);if(this.defaults.async===!0&&o.async===!1)return s(new Error("marked(): The async option was set to true by an extension. Remove async: false from the parse options object to return a Promise."));if(typeof t>"u"||t===null)return s(new Error("marked(): input parameter is undefined or null"));if(typeof t!="string")return s(new Error("marked(): input parameter is of type "+Object.prototype.toString.call(t)+", string expected"));if(l.hooks&&(l.hooks.options=l,l.hooks.block=r),l.async)return(async()=>{let a=l.hooks?await l.hooks.preprocess(t):t,m=await(l.hooks?await l.hooks.provideLexer(r):r?W.lex:W.lexInline)(a,l),c=l.hooks?await l.hooks.processAllTokens(m):m;l.walkTokens&&await Promise.all(this.walkTokens(c,l.walkTokens));let u=await(l.hooks?await l.hooks.provideParser(r):r?q.parse:q.parseInline)(c,l);return l.hooks?await l.hooks.postprocess(u):u})().catch(s);try{l.hooks&&(t=l.hooks.preprocess(t));let a=(l.hooks?l.hooks.provideLexer(r):r?W.lex:W.lexInline)(t,l);l.hooks&&(a=l.hooks.processAllTokens(a)),l.walkTokens&&this.walkTokens(a,l.walkTokens);let m=(l.hooks?l.hooks.provideParser(r):r?q.parse:q.parseInline)(a,l);return l.hooks&&(m=l.hooks.postprocess(m)),m}catch(a){return s(a)}}}onError(r,t){return n=>{if(n.message+=`
Please report this to https://github.com/markedjs/marked.`,r){let o="<p>An error occurred:</p><pre>"+J(n.message+"",!0)+"</pre>";return t?Promise.resolve(o):o}if(t)return Promise.reject(n);throw n}}},Y=new Zo;function M(r,t){return Y.parse(r,t)}M.options=M.setOptions=function(r){return Y.setOptions(r),M.defaults=Y.defaults,Se(M.defaults),M},M.getDefaults=me,M.defaults=Q,M.use=function(...r){return Y.use(...r),M.defaults=Y.defaults,Se(M.defaults),M},M.walkTokens=function(r,t){return Y.walkTokens(r,t)},M.parseInline=Y.parseInline,M.Parser=q,M.parser=q.parse,M.Renderer=ce,M.TextRenderer=xe,M.Lexer=W,M.lexer=W.lex,M.Tokenizer=ie,M.Hooks=re,M.parse=M,M.options,M.setOptions,M.use,M.walkTokens,M.parseInline,q.parse,W.lex;const _o={class:"modal-content help-modal"},Qo={class:"modal-header"},Xo={class:"header-actions"},Yo={class:"modal-body"},vo={class:"doc-sidebar"},el={class:"doc-sidebar-group"},nl={class:"doc-sidebar-header"},tl=["onClick"],rl={class:"doc-sidebar-group"},ol={class:"doc-sidebar-header"},ll={class:"doc-content"},al=["innerHTML"],sl=["innerHTML"],il=["innerHTML"],cl=["innerHTML"],dl=["innerHTML"],pl=["innerHTML"],ml=["innerHTML"],gl={class:"doc-pagination"},kl=["disabled"],hl={class:"page-info"},fl=["disabled"],ul=z({__name:"HelpModal",props:{initialDoc:{type:String,default:"getting-started"}},emits:["close","openAbout"],setup(r,{emit:t}){const n=r,o=e.ref(n.initialDoc),l=e.ref(""),s=e.ref(null),a=[{id:"getting-started",title:"快速开始",icon:"home"},{id:"features",title:"功能介绍",icon:"info"},{id:"api",title:"API 文档",icon:"code"},{id:"tools",title:"工具文档",icon:"tool"},{id:"shortcuts",title:"快捷键",icon:"keyboard"},{id:"faq",title:"常见问题",icon:"help"}],m=e.ref(a),c=e.computed(()=>[...m.value,{id:"changelog",title:"更新日志",icon:"activity"}]),u=e.computed(()=>c.value.findIndex(P=>P.id===o.value)),h=e.computed(()=>u.value>0),w=e.computed(()=>u.value<c.value.length-1);function V(){const P=l.value.toLowerCase().trim();if(!P){m.value=a;return}m.value=a.filter(S=>S.title.toLowerCase().includes(P)||S.id.includes(P)),m.value.length>0&&!m.value.find(S=>S.id===o.value)&&(o.value=m.value[0].id)}const T=P=>M(P,{breaks:!0,gfm:!0}).replace(/<table>/g,'<table class="doc-table">'),y=e.computed(()=>T(oo)),B=e.computed(()=>T(lo)),I=e.computed(()=>T(eo)),L=e.computed(()=>T(no)),F=e.computed(()=>T(to)),U=e.computed(()=>T(ro)),H=e.computed(()=>T(ao));function j(){var P;h.value&&(o.value=c.value[u.value-1].id,(P=s.value)==null||P.scrollTo(0,0))}function G(){var P;w.value&&(o.value=c.value[u.value+1].id,(P=s.value)==null||P.scrollTo(0,0))}return e.onMounted(()=>{V()}),(P,S)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:S[3]||(S[3]=e.withModifiers(N=>P.$emit("close"),["self"]))},[e.createElementVNode("div",_o,[e.createCommentVNode(" 头部 "),e.createElementVNode("div",Qo,[e.createElementVNode("h2",null,[e.createVNode(R,{name:"book-open",size:18}),S[4]||(S[4]=e.createTextVNode(" 帮助文档",-1))]),e.createElementVNode("div",Xo,[e.createElementVNode("button",{class:"btn-about",onClick:S[0]||(S[0]=N=>P.$emit("openAbout")),title:"关于 PairCode"},[e.createVNode(R,{name:"info",size:14}),S[5]||(S[5]=e.createTextVNode(" 关于 ",-1))]),e.createElementVNode("button",{class:"modal-close",onClick:S[1]||(S[1]=N=>P.$emit("close"))},"×")])]),e.createCommentVNode(" 主体 "),e.createElementVNode("div",Yo,[e.createCommentVNode(" 侧边导航 "),e.createElementVNode("div",vo,[e.createCommentVNode(" 文档中心分组 "),e.createElementVNode("div",el,[e.createElementVNode("div",nl,[e.createVNode(R,{name:"book",size:14}),S[6]||(S[6]=e.createElementVNode("span",null,"文档中心",-1))]),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(m.value,N=>(e.openBlock(),e.createElementBlock("div",{key:N.id,class:e.normalizeClass(["doc-nav-item",{active:o.value===N.id}]),onClick:f=>o.value=N.id},[e.createVNode(R,{name:N.icon,size:16},null,8,["name"]),e.createElementVNode("span",null,e.toDisplayString(N.title),1)],10,tl))),128))]),e.createCommentVNode(" 更新日志 "),e.createElementVNode("div",rl,[e.createElementVNode("div",ol,[e.createVNode(R,{name:"clock",size:14}),S[7]||(S[7]=e.createElementVNode("span",null,"其他",-1))]),e.createElementVNode("div",{class:e.normalizeClass(["doc-nav-item",{active:o.value==="changelog"}]),onClick:S[2]||(S[2]=N=>o.value="changelog")},[e.createVNode(R,{name:"activity",size:16}),S[8]||(S[8]=e.createElementVNode("span",null,"更新日志",-1))],2)])]),e.createCommentVNode(" 文档内容 "),e.createElementVNode("div",ll,[e.createElementVNode("div",{class:"doc-content-inner",ref_key:"contentRef",ref:s},[o.value==="faq"?(e.openBlock(),e.createElementBlock("div",{key:0,class:"doc-markdown",innerHTML:y.value},null,8,al)):o.value==="getting-started"?(e.openBlock(),e.createElementBlock("div",{key:1,class:"doc-markdown",innerHTML:B.value},null,8,sl)):o.value==="features"?(e.openBlock(),e.createElementBlock("div",{key:2,class:"doc-markdown",innerHTML:I.value},null,8,il)):o.value==="api"?(e.openBlock(),e.createElementBlock("div",{key:3,class:"doc-markdown",innerHTML:L.value},null,8,cl)):o.value==="tools"?(e.openBlock(),e.createElementBlock("div",{key:4,class:"doc-markdown",innerHTML:F.value},null,8,dl)):o.value==="shortcuts"?(e.openBlock(),e.createElementBlock("div",{key:5,class:"doc-markdown",innerHTML:U.value},null,8,pl)):o.value==="changelog"?(e.openBlock(),e.createElementBlock("div",{key:6,class:"doc-markdown",innerHTML:H.value},null,8,ml)):e.createCommentVNode("v-if",!0)],512),e.createCommentVNode(" 底部翻页 "),e.createElementVNode("div",gl,[e.createElementVNode("button",{class:"page-btn",onClick:j,disabled:!h.value},[e.createVNode(R,{name:"chevron-left",size:14}),S[9]||(S[9]=e.createTextVNode(" 上一页 ",-1))],8,kl),e.createElementVNode("span",hl,e.toDisplayString(u.value+1)+" / "+e.toDisplayString(c.value.length),1),e.createElementVNode("button",{class:"page-btn",onClick:G,disabled:!w.value},[S[10]||(S[10]=e.createTextVNode(" 下一页 ",-1)),e.createVNode(R,{name:"chevron-right",size:14})],8,fl)])])])])]))}},[["__scopeId","data-v-0f9c0147"]]),yl={class:"uc"},bl={class:"uc-line"},El={class:"uc-cur"},xl={key:0,class:"uc-new"},Vl={class:"uc-warn"},Nl=["disabled"],wl={key:0,class:"uc-prog"},Bl={class:"uc-bar"},Tl={class:"uc-pct"},Sl={key:1,class:"uc-plan"},Cl={key:2,class:"uc-notes"},Il={class:"uc-pre"},We=z({__name:"UpdateCard",props:{currentVersion:{type:String,default:""}},setup(r){const t=r,n=e.ref("idle"),o=e.ref({}),l=e.ref(""),s=e.ref(!1),a=e.ref(""),m=e.ref(""),c=e.ref({}),u=e.ref(0),h=e.ref(!1),w=e.ref(!1),V=e.ref("");let T=null;const y=["checking","downloading","verifying","extracting","applying","restarting"],B=e.computed(()=>y.includes(n.value)),I=e.computed(()=>{const k=String(t.currentVersion||"").trim();return k?k[0]==="v"||k[0]==="V"?k:"v"+k:"—"}),L=e.computed(()=>n.value==="checking"?"检查中…":s.value?"重新检查":"检查更新"),F=e.computed(()=>m.value||o.value.message||"尚未检查更新"),U=e.computed(()=>{switch(n.value){case"downloading":case"verifying":case"extracting":case"applying":return"package";case"ready":return"check-circle";case"error":return"shield-off";case"available":return"package";default:return"refresh"}}),H=e.computed(()=>n.value==="error"?"var(--danger)":"var(--accent)"),j=e.computed(()=>(((c.value||{}).speedBps||0)/1048576).toFixed(2));function G(k){const p=Number(k||0);return p>=1073741824?(p/1073741824).toFixed(2)+" GB":(p/1048576).toFixed(1)+" MB"}function P(k){k&&(o.value=k,n.value=k.stage||"idle",l.value=k.latest||"",s.value=!!k.hasUpdate,a.value=k.notes||"",m.value=k.error||"",c.value=k.progress||{},u.value=Number(c.value.percent||0),B.value?S():N())}function S(){T||(T=setInterval(async()=>{try{const k=await A.apiGet("/update/status");P(k.state)}catch{}},900))}function N(){T&&(clearInterval(T),T=null)}async function f(){h.value=!0,V.value="",n.value="checking",m.value="";try{const k=await A.apiGet("/update/check");P(k.state),k.error&&(m.value=k.error)}catch(k){m.value="检查更新失败："+((k==null?void 0:k.message)||k),n.value="error"}finally{h.value=!1}}async function E(){V.value="";try{const k=await A.apiPost("/update/download");P(k.state),k.error&&(m.value=k.error),S()}catch(k){m.value="下载失败："+((k==null?void 0:k.message)||k)}}async function i(){try{const k=await A.apiPost("/update/cancel");P(k.state)}catch{}}async function d(){V.value="正在生成替换计划…";try{const k=await A.apiPost("/update/apply",{dryRun:!0,restart:!1}),p=(k.result||{}).plan||{};V.value=`替换预览：将写 ${(p.write||[]).length} 个文件（${G(p.bytes)}），保护名单跳过 ${(p.skip||[]).length} 个（config/ 与 .pair 用户数据），主程序 ${p.mainProgram}`,k.error&&(V.value="预览失败："+k.error)}catch(k){V.value="预览失败："+((k==null?void 0:k.message)||k)}}async function g(k){w.value=!1,V.value=k?"正在安装并重启，页面将在数秒后断开…":"正在安装（不重启）…";try{const p=await A.apiPost("/update/apply",{dryRun:!1,restart:k});if(P(p.state),p.error){V.value="安装失败："+p.error;return}const x=(p.result||{}).plan||{};V.value=k?`已安装 v${l.value}，正在重启（备份：${x.backupPath||"-"}）`:`已安装 v${l.value}（${(x.write||[]).length} 个文件），重启后生效；旧版本备份：${x.backupPath||"-"}`}catch(p){V.value="安装失败："+((p==null?void 0:p.message)||p)}}return e.onMounted(async()=>{try{const k=await A.apiGet("/update/status");P(k.state)}catch{}}),e.onUnmounted(N),(k,p)=>(e.openBlock(),e.createElementBlock("div",yl,[e.createCommentVNode(" 主行：状态 + 操作 "),e.createElementVNode("div",bl,[e.createVNode(R,{name:U.value,size:14,color:H.value},null,8,["name","color"]),p[4]||(p[4]=e.createElementVNode("span",{class:"uc-title"},"软件更新",-1)),e.createElementVNode("span",El,"当前 "+e.toDisplayString(I.value),1),s.value?(e.openBlock(),e.createElementBlock("span",xl,"→ 新版本 v"+e.toDisplayString(l.value),1)):e.createCommentVNode("v-if",!0),e.createElementVNode("span",{class:e.normalizeClass(["uc-state",{"uc-err":!!m.value}])},e.toDisplayString(F.value),3),p[5]||(p[5]=e.createElementVNode("span",{class:"uc-gap"},null,-1)),w.value?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createElementVNode("span",Vl,"确认安装 v"+e.toDisplayString(l.value)+" 并重启？",1),e.createElementVNode("button",{class:"uc-btn uc-primary",onClick:p[0]||(p[0]=x=>g(!0))},"确认安装并重启"),e.createElementVNode("button",{class:"uc-btn",onClick:p[1]||(p[1]=x=>w.value=!1)},"取消")],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[!B.value&&n.value!=="ready"?(e.openBlock(),e.createElementBlock("button",{key:0,class:"uc-btn",disabled:h.value,onClick:f},[e.createVNode(R,{name:"refresh",size:12}),e.createTextVNode(" "+e.toDisplayString(L.value),1)],8,Nl)):e.createCommentVNode("v-if",!0),n.value==="available"?(e.openBlock(),e.createElementBlock("button",{key:1,class:"uc-btn uc-primary",onClick:E},"下载并安装")):e.createCommentVNode("v-if",!0),B.value?(e.openBlock(),e.createElementBlock("button",{key:2,class:"uc-btn",onClick:i},"取消")):e.createCommentVNode("v-if",!0),n.value==="ready"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createElementVNode("button",{class:"uc-btn",onClick:d},"预览替换清单"),e.createElementVNode("button",{class:"uc-btn uc-primary",onClick:p[2]||(p[2]=x=>w.value=!0)},"立即安装并重启"),e.createElementVNode("button",{class:"uc-btn",onClick:p[3]||(p[3]=x=>g(!1))},"仅安装，稍后重启")],64)):e.createCommentVNode("v-if",!0)],64))]),e.createCommentVNode(" 进度条 "),B.value?(e.openBlock(),e.createElementBlock("div",wl,[e.createElementVNode("div",Bl,[e.createElementVNode("div",{class:"uc-fill",style:e.normalizeStyle({width:u.value.toFixed(1)+"%"})},null,4)]),e.createElementVNode("span",Tl,e.toDisplayString(u.value.toFixed(1))+"% ｜ "+e.toDisplayString(G(c.value.downloaded))+" / "+e.toDisplayString(G(c.value.total))+" ｜ "+e.toDisplayString(j.value)+" MB/s",1)])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 替换计划（预览） "),V.value?(e.openBlock(),e.createElementBlock("div",Sl,e.toDisplayString(V.value),1)):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 更新说明（折叠） "),s.value&&a.value?(e.openBlock(),e.createElementBlock("details",Cl,[e.createElementVNode("summary",null,"更新说明（v"+e.toDisplayString(l.value)+"）",1),e.createElementVNode("pre",Il,e.toDisplayString(a.value),1)])):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-14a60d34"]]),Pl="data:image/svg+xml,%3csvg%20xmlns='http://www.w3.org/2000/svg'%20width='512'%20height='512'%20viewBox='0%200%20512%20512'%3e%3cdefs%3e%3c!--%20背景渐变（深色科技风）%20--%3e%3clinearGradient%20id='bgGrad'%20x1='0'%20y1='0'%20x2='1'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%230a1628'/%3e%3cstop%20offset='100%25'%20stop-color='%230d1f2e'/%3e%3c/linearGradient%3e%3c!--%20左侧尖括号渐变（科技蓝）%20--%3e%3clinearGradient%20id='leftBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='100%25'%20stop-color='%230077b6'/%3e%3c/linearGradient%3e%3c!--%20右侧尖括号渐变（科技绿）%20--%3e%3clinearGradient%20id='rightBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300e676'/%3e%3cstop%20offset='100%25'%20stop-color='%2300c853'/%3e%3c/linearGradient%3e%3c!--%20中间连接线（蓝绿渐变）%20--%3e%3clinearGradient%20id='connector'%20x1='0'%20y1='0'%20x2='1'%20y2='0'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='50%25'%20stop-color='%2300e5ff'/%3e%3cstop%20offset='100%25'%20stop-color='%2300e676'/%3e%3c/linearGradient%3e%3c!--%20外发光%20--%3e%3cfilter%20id='glow'%3e%3cfeGaussianBlur%20stdDeviation='4'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3cfilter%20id='softGlow'%3e%3cfeGaussianBlur%20stdDeviation='8'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3c/defs%3e%3c!--%20圆角方形背景（深色科技底）%20--%3e%3crect%20x='32'%20y='32'%20width='448'%20height='448'%20rx='96'%20ry='96'%20fill='url(%23bgGrad)'%20stroke='%231a3a4a'%20stroke-width='2'/%3e%3c!--%20左侧%20%3c%20尖括号（三段式直线%20—%20科技蓝，代表代码输入/开发者）%20--%3e%3cpath%20d='M180%20150%20L96%20256%20L180%20362'%20stroke='url(%23leftBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20右侧%20%3e%20尖括号（三段式直线%20—%20科技绿，代表代码输出/AI伙伴）%20--%3e%3cpath%20d='M332%20150%20L416%20256%20L332%20362'%20stroke='url(%23rightBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20中间「=」连接线已移除（图标只留%20%3c%20%3e%20尖括号%20+%20中心%20AI%20核心光点）。%20--%3e%3c!--%20中心光点（代表%20AI%20核心%20—%20亮青色）%20--%3e%3ccircle%20cx='256'%20cy='256'%20r='18'%20fill='transparent'%20stroke='%2300e5ff'%20stroke-width='3'%20opacity='0.6'/%3e%3ccircle%20cx='256'%20cy='256'%20r='8'%20fill='%2300e5ff'%20opacity='0.9'%3e%3canimate%20attributeName='opacity'%20values='0.6;1;0.6'%20dur='2s'%20repeatCount='indefinite'/%3e%3c/circle%3e%3c/svg%3e",Al={class:"modal-content about-modal"},Ml={class:"modal-header"},$l={class:"modal-body"},Dl={class:"about-left-col"},Rl={class:"about-hero"},Ll={class:"about-logo"},Fl=["src"],Gl={class:"about-version"},Ol={class:"about-right-col"},Ul={class:"about-section"},zl={class:"feature-list"},jl={class:"about-section"},Hl={key:0,class:"sys-info"},Wl={class:"info-row"},ql={class:"info-row"},Jl={class:"info-row"},Kl={class:"info-path"},Zl={class:"info-row"},_l={key:1,class:"loading-info"},Ql={class:"about-update"},Xl={class:"modal-footer"},Yl=z({__name:"AboutModal",props:{showHelpBtn:{type:Boolean,default:!0}},emits:["close","openHelp"],setup(r,{emit:t}){const n=e.ref(""),o=e.ref({}),l=e.ref(!0);return e.onMounted(async()=>{try{const s=await A.apiGet("/system/info");o.value=s,s.version&&(n.value=s.version)}catch{}l.value=!1}),(s,a)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:a[3]||(a[3]=e.withModifiers(m=>s.$emit("close"),["self"]))},[e.createElementVNode("div",Al,[e.createElementVNode("div",Ml,[e.createElementVNode("h2",null,[e.createVNode(R,{name:"info",size:18}),a[4]||(a[4]=e.createTextVNode(" 关于 PairCode",-1))]),e.createElementVNode("button",{class:"modal-close",onClick:a[0]||(a[0]=m=>s.$emit("close"))},"×")]),e.createElementVNode("div",$l,[e.createCommentVNode(" 左列：Logo + 描述 + 技术栈 "),e.createElementVNode("div",Dl,[e.createCommentVNode(" Logo + 标题 "),e.createElementVNode("div",Rl,[e.createElementVNode("div",Ll,[e.createElementVNode("img",{src:e.unref(Pl),class:"about-logo-img",alt:"PairCode"},null,8,Fl)]),a[5]||(a[5]=e.createElementVNode("div",{class:"about-title"},"PairCode IDE",-1)),e.createElementVNode("div",Gl,"版本 "+e.toDisplayString(n.value),1)]),e.createCommentVNode(" 描述 "),a[6]||(a[6]=e.createElementVNode("div",{class:"about-section"},[e.createElementVNode("p",{class:"about-description"}," PairCode IDE 是一款纯 Web 端的 AI 辅助编程集成开发环境， 专为浏览器而设计。无需安装任何桌面客户端或本地 IDE 软件， 打开浏览器即可开始编程。它将 AI 对话能力深度融入编码工作流， 你只需用自然语言描述需求，AI 就能自动理解上下文、读写文件、执行命令、 管理版本控制。从代码生成到项目运维，在同一个浏览器窗口中全部完成。 ")],-1)),e.createCommentVNode(" 技术栈 "),a[7]||(a[7]=e.createStaticVNode('<div class="about-section" data-v-226f7b6a><div class="section-title" data-v-226f7b6a>技术栈</div><div class="tech-stack" data-v-226f7b6a><span class="tech-badge" data-v-226f7b6a>Go</span><span class="tech-badge" data-v-226f7b6a>Vue 3</span><span class="tech-badge" data-v-226f7b6a>WebSocket</span><span class="tech-badge" data-v-226f7b6a>CodeMirror</span><span class="tech-badge" data-v-226f7b6a>插件化工具</span><span class="tech-badge" data-v-226f7b6a>TS 编译器</span><span class="tech-badge" data-v-226f7b6a>MCP</span><span class="tech-badge" data-v-226f7b6a>CodeGraph</span><span class="tech-badge" data-v-226f7b6a>DAP</span></div></div>',1))]),e.createCommentVNode(" 右列：特性 + 系统信息 "),e.createElementVNode("div",Ol,[e.createCommentVNode(" 特性亮点 "),e.createElementVNode("div",Ul,[a[18]||(a[18]=e.createElementVNode("div",{class:"section-title"},"主要特性",-1)),e.createElementVNode("ul",zl,[e.createElementVNode("li",null,[e.createVNode(R,{name:"bot",size:14,color:"var(--accent)"}),a[8]||(a[8]=e.createTextVNode(" AI 对话编程 — 用自然语言与 AI 对话，自动生成与重构代码",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"file",size:14,color:"var(--accent)"}),a[9]||(a[9]=e.createTextVNode(" 智能代码编辑器 — 多语言语法高亮，浏览器中流畅编辑",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"git-branch",size:14,color:"var(--accent)"}),a[10]||(a[10]=e.createTextVNode(" Git 版本控制 — 在对话中完成全部 Git 操作",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"terminal",size:14,color:"var(--accent)"}),a[11]||(a[11]=e.createTextVNode(" 内置终端 — 无需离开浏览器即可执行命令",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"search",size:14,color:"var(--accent)"}),a[12]||(a[12]=e.createTextVNode(" 全局搜索 — 快速搜索文件与代码内容",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"settings",size:14,color:"var(--accent)"}),a[13]||(a[13]=e.createTextVNode(" 自主 Agent 模式 — AI 主动分析项目并自动执行任务",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"grid",size:14,color:"var(--accent)"}),a[14]||(a[14]=e.createTextVNode(" 对话历史管理 — 自动保存、回溯与继续历史对话",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"tool",size:14,color:"var(--accent)"}),a[15]||(a[15]=e.createTextVNode(" Skills / MCP 扩展 — 通过技能市场扩展 IDE 能力",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"code",size:14,color:"var(--accent)"}),a[16]||(a[16]=e.createTextVNode(" 内置调试器 — 支持 Go 程序的断点、单步和变量查看",-1))]),e.createElementVNode("li",null,[e.createVNode(R,{name:"image",size:14,color:"var(--accent)"}),a[17]||(a[17]=e.createTextVNode(" 网页验证 — 打开 URL、截图、分析页面效果",-1))])])]),e.createCommentVNode(" 系统信息 "),e.createElementVNode("div",jl,[a[23]||(a[23]=e.createElementVNode("div",{class:"section-title"},"系统信息",-1)),l.value?(e.openBlock(),e.createElementBlock("div",_l,"加载中...")):(e.openBlock(),e.createElementBlock("div",Hl,[e.createElementVNode("div",Wl,[a[19]||(a[19]=e.createElementVNode("span",{class:"info-label"},"主机名",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.hostname),1)]),e.createElementVNode("div",ql,[a[20]||(a[20]=e.createElementVNode("span",{class:"info-label"},"操作系统",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.os),1)]),e.createElementVNode("div",Jl,[a[21]||(a[21]=e.createElementVNode("span",{class:"info-label"},"工作区",-1)),e.createElementVNode("span",Kl,e.toDisplayString(o.value.workspace),1)]),e.createElementVNode("div",Zl,[a[22]||(a[22]=e.createElementVNode("span",{class:"info-label"},"平台信息",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.goos),1)])]))])])]),e.createCommentVNode(" ★ 软件更新（在线更新：检查 → 下载 → 校验 → 安装重启；数据源 /api/update/*）"),e.createElementVNode("div",Ql,[e.createVNode(We,{"current-version":n.value},null,8,["current-version"])]),e.createCommentVNode(" 底部 "),e.createElementVNode("div",Xl,[r.showHelpBtn?(e.openBlock(),e.createElementBlock("button",{key:0,class:"btn-primary",onClick:a[1]||(a[1]=m=>s.$emit("openHelp"))},[e.createVNode(R,{name:"book-open",size:14}),a[24]||(a[24]=e.createTextVNode(" 查看帮助文档 ",-1))])):e.createCommentVNode("v-if",!0),e.createElementVNode("button",{class:"btn-secondary",onClick:a[2]||(a[2]=m=>s.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-226f7b6a"]]),vl={class:"modal-header"},ea={class:"modal-title"},na={class:"modal-body"},ta=z({__name:"Modal",props:{maxWidth:{type:String,default:"480px"}},emits:["close"],setup(r,{emit:t}){const n=t,o=e.ref(!0);function l(s){s.key==="Escape"&&n("close")}return e.onMounted(()=>document.addEventListener("keydown",l)),e.onUnmounted(()=>document.removeEventListener("keydown",l)),(s,a)=>(e.openBlock(),e.createBlock(e.Teleport,{to:"body"},[o.value?(e.openBlock(),e.createElementBlock("div",{key:0,class:"modal-overlay",onClick:a[1]||(a[1]=e.withModifiers(m=>s.$emit("close"),["self"]))},[e.createElementVNode("div",{class:"modal-container",style:e.normalizeStyle({maxWidth:r.maxWidth})},[e.createElementVNode("div",vl,[e.createElementVNode("span",ea,[e.renderSlot(s.$slots,"title",{},()=>[a[2]||(a[2]=e.createTextVNode("提示",-1))],!0)]),e.createElementVNode("button",{class:"modal-close",onClick:a[0]||(a[0]=m=>s.$emit("close"))},[e.createVNode(R,{name:"close",size:14})])]),e.createElementVNode("div",na,[e.renderSlot(s.$slots,"default",{},void 0,!0)])],4)])):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-1f96761a"]]),ra={class:"um"},oa=z({__name:"UpdateModal",emits:["close"],setup(r){const t=e.ref("");return e.onMounted(async()=>{try{const n=await A.apiGet("/system/info");n&&n.version&&(t.value=n.version)}catch{}}),(n,o)=>(e.openBlock(),e.createBlock(ta,{"max-width":"640px",onClose:o[0]||(o[0]=l=>n.$emit("close"))},{title:e.withCtx(()=>[e.createVNode(R,{name:"download",size:14}),o[1]||(o[1]=e.createTextVNode(" 软件更新 ",-1))]),default:e.withCtx(()=>[e.createElementVNode("div",ra,[e.createVNode(We,{"current-version":t.value},null,8,["current-version"]),o[2]||(o[2]=e.createElementVNode("div",{class:"um-tips"},[e.createElementVNode("p",null,[e.createTextVNode("· 安装只替换程序与内置资源，"),e.createElementVNode("b",null,"config/ 与 .pair/ 下的用户数据、配置、技能不会被覆盖"),e.createTextVNode("。")]),e.createElementVNode("p",null,"· 「立即安装并重启」会在替换完成后自动重启服务；「仅安装，稍后重启」等你下次重启生效。"),e.createElementVNode("p",null,"· 更新源 / 发布频道 / 自动检查 / 校验开关：设置面板 → 软件更新。")],-1))])]),_:1}))}},[["__scopeId","data-v-6c94c04d"]]),la={class:"toast-container"},aa={class:"dlg-box",style:{"max-width":"400px"}},sa={class:"dlg-title"},ia={class:"dlg-body"},ca={class:"dlg-actions"},da={class:"dlg-box",style:{"max-width":"420px"}},pa={class:"dlg-title"},ma={class:"dlg-body",style:{display:"flex","flex-direction":"column",gap:"8px"}},ga={style:{"font-size":"13px",color:"var(--text-secondary)"}},ka=["placeholder"],ha={class:"dlg-actions"},fa={class:"dlg-box",style:{"max-width":"400px"}},ua={class:"dlg-title"},ya={class:"dlg-body",style:{"white-space":"pre-line"}},ba=z({__name:"GlobalDialogs",setup(r){const t=e.ref(null);e.watch(()=>b.dialogState.show,l=>{l&&b.dialogState.type==="prompt"&&e.nextTick(()=>{var s,a;(s=t.value)==null||s.focus(),(a=t.value)==null||a.select()})});function n(){if(b.dialogState.type==="prompt"){const l=b.dialogState.inputValue;b.dialogState.show=!1,b.dialogState.resolve&&b.dialogState.resolve(l)}else if(b.dialogState.type==="confirm"&&b.dialogState.checkboxLabel){const s=b.dialogState.checkboxValue;b.dialogState.show=!1,b.dialogState.resolve&&b.dialogState.resolve({confirmed:!0,checked:s})}else b.dialogState.show=!1,b.dialogState.resolve&&b.dialogState.resolve(!0);b.dialogState.resolve=null}function o(){b.dialogState.type==="confirm"&&b.dialogState.checkboxLabel?(b.dialogState.show=!1,b.dialogState.resolve&&b.dialogState.resolve({confirmed:!1,checked:b.dialogState.checkboxValue})):b.dialogState.type==="prompt"?(b.dialogState.show=!1,b.dialogState.resolve&&b.dialogState.resolve(null)):(b.dialogState.show=!1,b.dialogState.resolve&&b.dialogState.resolve(!1)),b.dialogState.resolve=null}return(l,s)=>(e.openBlock(),e.createElementBlock(e.Fragment,null,[e.createCommentVNode(" Toast 通知区域 "),e.createElementVNode("div",la,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(e.unref(b.dialogState).toasts,a=>(e.openBlock(),e.createElementBlock("div",{key:a.id,class:e.normalizeClass(["toast-item","toast-"+(a.type||"info")])},e.toDisplayString(a.message),3))),128))]),e.createCommentVNode(" Confirm 对话框 "),e.unref(b.dialogState).show&&e.unref(b.dialogState).type==="confirm"?(e.openBlock(),e.createElementBlock("div",{key:0,class:"dlg-overlay",onClick:e.withModifiers(o,["self"])},[e.createElementVNode("div",aa,[e.createElementVNode("div",sa,e.toDisplayString(e.unref(b.dialogState).title),1),e.createElementVNode("div",ia,e.toDisplayString(e.unref(b.dialogState).message),1),e.unref(b.dialogState).checkboxLabel?(e.openBlock(),e.createElementBlock("label",{key:0,class:"dlg-checkbox",onClick:s[1]||(s[1]=e.withModifiers(()=>{},["stop"]))},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":s[0]||(s[0]=a=>e.unref(b.dialogState).checkboxValue=a)},null,512),[[e.vModelCheckbox,e.unref(b.dialogState).checkboxValue]]),e.createElementVNode("span",null,e.toDisplayString(e.unref(b.dialogState).checkboxLabel),1)])):e.createCommentVNode("v-if",!0),e.createElementVNode("div",ca,[e.createElementVNode("button",{class:"dlg-btn",onClick:o},e.toDisplayString(e.unref(b.dialogState).cancelText),1),e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},e.toDisplayString(e.unref(b.dialogState).confirmText),1)])])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" Prompt 对话框 "),e.unref(b.dialogState).show&&e.unref(b.dialogState).type==="prompt"?(e.openBlock(),e.createElementBlock("div",{key:1,class:"dlg-overlay",onClick:e.withModifiers(o,["self"])},[e.createElementVNode("div",da,[e.createElementVNode("div",pa,e.toDisplayString(e.unref(b.dialogState).title),1),e.createElementVNode("div",ma,[e.createElementVNode("span",ga,e.toDisplayString(e.unref(b.dialogState).message),1),e.withDirectives(e.createElementVNode("input",{ref_key:"promptInputRef",ref:t,"onUpdate:modelValue":s[2]||(s[2]=a=>e.unref(b.dialogState).inputValue=a),placeholder:e.unref(b.dialogState).inputPlaceholder,class:"dlg-input",onKeyup:[e.withKeys(n,["enter"]),e.withKeys(o,["escape"])]},null,40,ka),[[e.vModelText,e.unref(b.dialogState).inputValue]])]),e.createElementVNode("div",ha,[e.createElementVNode("button",{class:"dlg-btn",onClick:o},e.toDisplayString(e.unref(b.dialogState).cancelText),1),e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},e.toDisplayString(e.unref(b.dialogState).confirmText),1)])])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" Alert 信息框 "),e.unref(b.dialogState).show&&e.unref(b.dialogState).type==="alert"?(e.openBlock(),e.createElementBlock("div",{key:2,class:"dlg-overlay",onClick:e.withModifiers(n,["self"])},[e.createElementVNode("div",fa,[e.createElementVNode("div",ua,e.toDisplayString(e.unref(b.dialogState).title),1),e.createElementVNode("div",ya,e.toDisplayString(e.unref(b.dialogState).message),1),e.createElementVNode("div",{class:"dlg-actions"},[e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},"确定")])])])):e.createCommentVNode("v-if",!0)],64))}},[["__scopeId","data-v-b4faa9fc"]]),Ea=z({__name:"UiModals",setup(r){const t=e.ref(null);let n=null;function o(){b.showAbout.value=!1,b.showHelp.value=!0,b.helpDocTarget.value="getting-started"}function l(){b.showHelp.value=!1,b.showAbout.value=!0}return e.onMounted(()=>{n=Be.mountListSlot(t,"overlay",{isActive:s=>Be.isOverlayActive("overlay",s)})}),e.onUnmounted(()=>{n&&(n(),n=null)}),(s,a)=>(e.openBlock(),e.createElementBlock(e.Fragment,null,[e.unref(b.showSettings)?(e.openBlock(),e.createBlock(Lr,{key:0,onClose:a[0]||(a[0]=m=>b.showSettings.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(b.showSystem)?(e.openBlock(),e.createBlock(_r,{key:1,onClose:a[1]||(a[1]=m=>b.showSystem.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(b.showSource)?(e.openBlock(),e.createBlock(vr,{key:2,onClose:a[2]||(a[2]=m=>b.showSource.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(b.showHelp)?(e.openBlock(),e.createBlock(ul,{key:3,onClose:a[3]||(a[3]=m=>b.showHelp.value=!1),onOpenAbout:l,initialDoc:e.unref(b.helpDocTarget)},null,8,["initialDoc"])):e.createCommentVNode("v-if",!0),e.unref(b.showAbout)?(e.openBlock(),e.createBlock(Yl,{key:4,onClose:a[4]||(a[4]=m=>b.showAbout.value=!1),onOpenHelp:o})):e.createCommentVNode("v-if",!0),e.unref(b.showUpdate)?(e.openBlock(),e.createBlock(oa,{key:5,onClose:a[5]||(a[5]=m=>b.showUpdate.value=!1)})):e.createCommentVNode("v-if",!0),e.createVNode(ba),e.createCommentVNode(" ★ overlay 槽位（list 型）：插件注册的浮动层条目叠加渲染（badge/toast/status pill 等） "),e.createElementVNode("div",{ref_key:"overlaySlotEl",ref:t,class:"plugin-overlay-host"},null,512)],64))}},[["__scopeId","data-v-a045ab80"]]);function xa(r){const t=e.createApp(Ea);return t.mount(r),()=>{t.unmount()}}return K.mount=xa,Object.defineProperty(K,Symbol.toStringTag,{value:"Module"}),K})({},window.__PAIRCODE_CORE.Vue,window.__PAIRCODE_CORE.uiState,window.__PAIRCODE_CORE.pluginRuntime,window.__PAIRCODE_CORE.api);
