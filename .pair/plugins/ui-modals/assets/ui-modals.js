var UiModals=(function(W,e,E,Ve,A){"use strict";var Vo=Object.defineProperty;var No=(W,e,E)=>e in W?Vo(W,e,{enumerable:!0,configurable:!0,writable:!0,value:E}):W[e]=E;var I=(W,e,E)=>No(W,typeof e!="symbol"?e+"":e,E);var ie;const U=(r,t)=>{const n=r.__vccOpts||r;for(const[l,o]of t)n[l]=o;return n},Ue=["width","height"],He={key:0,d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"},$=U({__name:"SvgIcon",props:{name:{type:String,required:!0},size:{type:Number,default:16}},setup(r){return(t,n)=>(e.openBlock(),e.createElementBlock("svg",{class:"svg-icon",width:r.size,height:r.size,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor","stroke-width":"2","stroke-linecap":"round","stroke-linejoin":"round"},[e.createCommentVNode(" Folder "),r.name==="folder"?(e.openBlock(),e.createElementBlock("path",He)):r.name==="folder-open"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" Folder Open "),n[0]||(n[0]=e.createElementVNode("path",{d:"M6 17l-3-9h18l-3 9H6z"},null,-1)),n[1]||(n[1]=e.createElementVNode("path",{d:"M4 8V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v3"},null,-1))],64)):r.name==="file"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" File "),n[2]||(n[2]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[3]||(n[3]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1))],64)):r.name==="file-code"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createCommentVNode(" File Code "),n[4]||(n[4]=e.createStaticVNode('<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" data-v-faf69761></path><polyline points="14 2 14 8 20 8" data-v-faf69761></polyline><line x1="10" y1="12" x2="8" y2="14" data-v-faf69761></line><line x1="10" y1="16" x2="8" y2="18" data-v-faf69761></line><line x1="14" y1="12" x2="16" y2="14" data-v-faf69761></line><line x1="14" y1="16" x2="16" y2="18" data-v-faf69761></line>',6))],64)):r.name==="file-text"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:4},[e.createCommentVNode(" File Text / Document "),n[5]||(n[5]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[6]||(n[6]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[7]||(n[7]=e.createElementVNode("line",{x1:"9",y1:"13",x2:"15",y2:"13"},null,-1)),n[8]||(n[8]=e.createElementVNode("line",{x1:"9",y1:"17",x2:"15",y2:"17"},null,-1))],64)):r.name==="search"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:5},[e.createCommentVNode(" Search "),n[9]||(n[9]=e.createElementVNode("circle",{cx:"11",cy:"11",r:"8"},null,-1)),n[10]||(n[10]=e.createElementVNode("line",{x1:"21",y1:"21",x2:"16.65",y2:"16.65"},null,-1))],64)):r.name==="terminal"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:6},[e.createCommentVNode(" Terminal / Console "),n[11]||(n[11]=e.createElementVNode("polyline",{points:"4 17 10 11 4 5"},null,-1)),n[12]||(n[12]=e.createElementVNode("line",{x1:"12",y1:"19",x2:"20",y2:"19"},null,-1))],64)):r.name==="chat"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:7},[e.createCommentVNode(" Chat / Message "),n[13]||(n[13]=e.createElementVNode("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"},null,-1))],64)):r.name==="settings"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:8},[e.createCommentVNode(" Gear / Settings "),n[14]||(n[14]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"3"},null,-1)),n[15]||(n[15]=e.createElementVNode("path",{d:"M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"},null,-1))],64)):r.name==="home"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:9},[e.createCommentVNode(" Home "),n[16]||(n[16]=e.createElementVNode("path",{d:"M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"},null,-1)),n[17]||(n[17]=e.createElementVNode("polyline",{points:"9 22 9 12 15 12 15 22"},null,-1))],64)):r.name==="chevron-right"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:10},[e.createCommentVNode(" Chevron Right "),n[18]||(n[18]=e.createElementVNode("polyline",{points:"9 6 15 12 9 18"},null,-1))],64)):r.name==="chevron-down"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:11},[e.createCommentVNode(" Chevron Down (Rotated chevron-right) "),n[19]||(n[19]=e.createElementVNode("polyline",{points:"6 9 12 15 18 9"},null,-1))],64)):r.name==="plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:12},[e.createCommentVNode(" Plus / Add "),n[20]||(n[20]=e.createElementVNode("line",{x1:"12",y1:"5",x2:"12",y2:"19"},null,-1)),n[21]||(n[21]=e.createElementVNode("line",{x1:"5",y1:"12",x2:"19",y2:"12"},null,-1))],64)):r.name==="close"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:13},[e.createCommentVNode(" Close / X "),n[22]||(n[22]=e.createElementVNode("line",{x1:"18",y1:"6",x2:"6",y2:"18"},null,-1)),n[23]||(n[23]=e.createElementVNode("line",{x1:"6",y1:"6",x2:"18",y2:"18"},null,-1))],64)):r.name==="refresh"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:14},[e.createCommentVNode(" Refresh "),n[24]||(n[24]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[25]||(n[25]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 1 1-2.12-9.36L23 10"},null,-1))],64)):r.name==="drive"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:15},[e.createCommentVNode(" Hard Drive / Disk "),n[26]||(n[26]=e.createElementVNode("line",{x1:"22",y1:"12",x2:"2",y2:"12"},null,-1)),n[27]||(n[27]=e.createElementVNode("path",{d:"M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"},null,-1)),n[28]||(n[28]=e.createElementVNode("line",{x1:"6",y1:"16",x2:"6.01",y2:"16"},null,-1)),n[29]||(n[29]=e.createElementVNode("line",{x1:"10",y1:"16",x2:"10.01",y2:"16"},null,-1))],64)):r.name==="source-control"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:16},[e.createCommentVNode(" Source Control / Git Branch "),n[30]||(n[30]=e.createElementVNode("line",{x1:"6",y1:"3",x2:"6",y2:"15"},null,-1)),n[31]||(n[31]=e.createElementVNode("circle",{cx:"18",cy:"6",r:"3"},null,-1)),n[32]||(n[32]=e.createElementVNode("circle",{cx:"6",cy:"18",r:"3"},null,-1)),n[33]||(n[33]=e.createElementVNode("path",{d:"M18 9a9 9 0 0 1-9 9"},null,-1))],64)):r.name==="git-branch"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:17},[e.createCommentVNode(" Git Branch "),n[34]||(n[34]=e.createElementVNode("line",{x1:"6",y1:"3",x2:"6",y2:"15"},null,-1)),n[35]||(n[35]=e.createElementVNode("circle",{cx:"18",cy:"6",r:"3"},null,-1)),n[36]||(n[36]=e.createElementVNode("circle",{cx:"6",cy:"18",r:"3"},null,-1)),n[37]||(n[37]=e.createElementVNode("path",{d:"M18 9a9 9 0 0 1-9 9"},null,-1))],64)):r.name==="git-pull"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:18},[e.createCommentVNode(" Git Pull "),n[38]||(n[38]=e.createStaticVNode('<circle cx="18" cy="18" r="3" data-v-faf69761></circle><circle cx="6" cy="6" r="3" data-v-faf69761></circle><path d="M13 6h3a2 2 0 0 1 2 2v7" data-v-faf69761></path><line x1="6" y1="18" x2="6" y2="9" data-v-faf69761></line><polyline points="9 9 6 6 3 9" data-v-faf69761></polyline>',5))],64)):r.name==="git-push"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:19},[e.createCommentVNode(" Git Push "),n[39]||(n[39]=e.createStaticVNode('<circle cx="18" cy="6" r="3" data-v-faf69761></circle><circle cx="6" cy="18" r="3" data-v-faf69761></circle><path d="M13 18h-2a2 2 0 0 1-2-2V9" data-v-faf69761></path><line x1="6" y1="6" x2="6" y2="15" data-v-faf69761></line><polyline points="9 15 6 18 3 15" data-v-faf69761></polyline>',5))],64)):r.name==="output"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:20},[e.createCommentVNode(" Output / Window "),n[40]||(n[40]=e.createElementVNode("rect",{x:"2",y:"3",width:"20",height:"14",rx:"2",ry:"2"},null,-1)),n[41]||(n[41]=e.createElementVNode("line",{x1:"8",y1:"21",x2:"16",y2:"21"},null,-1)),n[42]||(n[42]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12",y2:"21"},null,-1))],64)):r.name==="warning"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:21},[e.createCommentVNode(" Warning / Alert "),n[43]||(n[43]=e.createElementVNode("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"},null,-1)),n[44]||(n[44]=e.createElementVNode("line",{x1:"12",y1:"9",x2:"12",y2:"13"},null,-1)),n[45]||(n[45]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"},null,-1))],64)):r.name==="undo"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:22},[e.createCommentVNode(" Undo "),n[46]||(n[46]=e.createElementVNode("polyline",{points:"1 4 1 10 7 10"},null,-1)),n[47]||(n[47]=e.createElementVNode("path",{d:"M3.51 15a9 9 0 1 0 2.13-9.36L1 10"},null,-1))],64)):r.name==="redo"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:23},[e.createCommentVNode(" Redo "),n[48]||(n[48]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[49]||(n[49]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 1 1-2.12-9.36L23 10"},null,-1))],64)):r.name==="package"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:24},[e.createCommentVNode(" Package / Box / Store "),n[50]||(n[50]=e.createElementVNode("line",{x1:"16.5",y1:"9.4",x2:"7.5",y2:"4.21"},null,-1)),n[51]||(n[51]=e.createElementVNode("path",{d:"M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"},null,-1)),n[52]||(n[52]=e.createElementVNode("polyline",{points:"3.27 6.96 12 12.01 20.73 6.96"},null,-1)),n[53]||(n[53]=e.createElementVNode("line",{x1:"12",y1:"22.08",x2:"12",y2:"12"},null,-1))],64)):r.name==="globe"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:25},[e.createCommentVNode(" Globe / External "),n[54]||(n[54]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[55]||(n[55]=e.createElementVNode("line",{x1:"2",y1:"12",x2:"22",y2:"12"},null,-1)),n[56]||(n[56]=e.createElementVNode("path",{d:"M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"},null,-1))],64)):r.name==="cycle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:26},[e.createCommentVNode(" Refresh / Cycle (for agent) "),n[57]||(n[57]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[58]||(n[58]=e.createElementVNode("polyline",{points:"1 20 1 14 7 14"},null,-1)),n[59]||(n[59]=e.createElementVNode("path",{d:"M3.51 9a9 9 0 0 1 14.85-3.36L23 10"},null,-1)),n[60]||(n[60]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 0 1-14.85 3.36L1 14"},null,-1))],64)):r.name==="send"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:27},[e.createCommentVNode(" Send (arrow up) "),n[61]||(n[61]=e.createElementVNode("line",{x1:"12",y1:"19",x2:"12",y2:"5"},null,-1)),n[62]||(n[62]=e.createElementVNode("polyline",{points:"5 12 12 5 19 12"},null,-1))],64)):r.name==="send-plane"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:28},[e.createCommentVNode(" Send Plane (paper airplane) "),n[63]||(n[63]=e.createElementVNode("line",{x1:"22",y1:"2",x2:"11",y2:"13"},null,-1)),n[64]||(n[64]=e.createElementVNode("polygon",{points:"22 2 15 22 11 13 2 9 22 2"},null,-1))],64)):r.name==="stop-dot"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:29},[e.createCommentVNode(" Stop Dot (pulsing circle) "),n[65]||(n[65]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"6",class:"stop-pulse"},null,-1)),n[66]||(n[66]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10",class:"stop-pulse-ring"},null,-1))],64)):r.name==="wrench"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:30},[e.createCommentVNode(" Wrench / Tool "),n[67]||(n[67]=e.createElementVNode("path",{d:"M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"},null,-1))],64)):r.name==="database"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:31},[e.createCommentVNode(" Database "),n[68]||(n[68]=e.createElementVNode("ellipse",{cx:"12",cy:"5",rx:"9",ry:"3"},null,-1)),n[69]||(n[69]=e.createElementVNode("path",{d:"M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"},null,-1)),n[70]||(n[70]=e.createElementVNode("path",{d:"M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"},null,-1))],64)):r.name==="user"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:32},[e.createCommentVNode(" User / Person "),n[71]||(n[71]=e.createElementVNode("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"},null,-1)),n[72]||(n[72]=e.createElementVNode("circle",{cx:"12",cy:"7",r:"4"},null,-1))],64)):r.name==="info"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:33},[e.createCommentVNode(" Info "),n[73]||(n[73]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[74]||(n[74]=e.createElementVNode("line",{x1:"12",y1:"16",x2:"12",y2:"12"},null,-1)),n[75]||(n[75]=e.createElementVNode("line",{x1:"12",y1:"8",x2:"12.01",y2:"8"},null,-1))],64)):r.name==="lightbulb"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:34},[e.createCommentVNode(" Lightbulb / Suggestion "),n[76]||(n[76]=e.createElementVNode("path",{d:"M9 18h6"},null,-1)),n[77]||(n[77]=e.createElementVNode("path",{d:"M10 22h4"},null,-1)),n[78]||(n[78]=e.createElementVNode("path",{d:"M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14"},null,-1))],64)):r.name==="sparkles"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:35},[e.createCommentVNode(" Sparkles / Auto "),n[79]||(n[79]=e.createStaticVNode('<path d="M13.5 4L15 8l4 .5L15 12l1.5 4-4-2-4 2L10 12l-4-3.5L10 8z" data-v-faf69761></path><line x1="3" y1="18" x2="3" y2="21" data-v-faf69761></line><line x1="21" y1="18" x2="21" y2="21" data-v-faf69761></line><line x1="7" y1="20" x2="11" y2="20" data-v-faf69761></line><line x1="17" y1="20" x2="19" y2="20" data-v-faf69761></line>',5))],64)):r.name==="bot"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:36},[e.createCommentVNode(" Bot / AI "),n[80]||(n[80]=e.createStaticVNode('<rect x="3" y="11" width="18" height="10" rx="2" data-v-faf69761></rect><circle cx="12" cy="5" r="2" data-v-faf69761></circle><path d="M12 7v4" data-v-faf69761></path><line x1="8" y1="16" x2="8" y2="16" data-v-faf69761></line><line x1="16" y1="16" x2="16" y2="16" data-v-faf69761></line>',5))],64)):r.name==="file-js"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:37},[e.createCommentVNode(" File Type Icons "),n[81]||(n[81]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[82]||(n[82]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[83]||(n[83]=e.createElementVNode("text",{x:"8",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"JS",-1))],64)):r.name==="file-ts"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:38},[n[84]||(n[84]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[85]||(n[85]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[86]||(n[86]=e.createElementVNode("text",{x:"8",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"TS",-1))],64)):r.name==="file-go"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:39},[n[87]||(n[87]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[88]||(n[88]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[89]||(n[89]=e.createElementVNode("text",{x:"9",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Go",-1))],64)):r.name==="file-py"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:40},[n[90]||(n[90]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[91]||(n[91]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[92]||(n[92]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Py",-1))],64)):r.name==="file-java"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:41},[n[93]||(n[93]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[94]||(n[94]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[95]||(n[95]=e.createElementVNode("text",{x:"6",y:"17","font-size":"8",fill:"currentColor","font-weight":"bold",stroke:"none"},"Java",-1))],64)):r.name==="file-html"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:42},[n[96]||(n[96]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[97]||(n[97]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[98]||(n[98]=e.createElementVNode("text",{x:"6",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"HTML",-1))],64)):r.name==="file-css"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:43},[n[99]||(n[99]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[100]||(n[100]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[101]||(n[101]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"CSS",-1))],64)):r.name==="file-json"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:44},[n[102]||(n[102]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[103]||(n[103]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[104]||(n[104]=e.createElementVNode("text",{x:"5",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"{ }",-1))],64)):r.name==="file-md"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:45},[n[105]||(n[105]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[106]||(n[106]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[107]||(n[107]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"MD",-1))],64)):r.name==="file-vue"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:46},[n[108]||(n[108]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[109]||(n[109]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[110]||(n[110]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Vue",-1))],64)):r.name==="copy"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:47},[e.createCommentVNode(" Copy "),n[111]||(n[111]=e.createElementVNode("rect",{x:"9",y:"9",width:"13",height:"13",rx:"2",ry:"2"},null,-1)),n[112]||(n[112]=e.createElementVNode("path",{d:"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"},null,-1))],64)):r.name==="minus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:48},[e.createCommentVNode(" Minus "),n[113]||(n[113]=e.createElementVNode("line",{x1:"5",y1:"12",x2:"19",y2:"12"},null,-1))],64)):r.name==="edit"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:49},[e.createCommentVNode(" Edit / Rename "),n[114]||(n[114]=e.createElementVNode("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"},null,-1)),n[115]||(n[115]=e.createElementVNode("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"},null,-1))],64)):r.name==="trash"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:50},[e.createCommentVNode(" Trash / Delete "),n[116]||(n[116]=e.createElementVNode("polyline",{points:"3 6 5 6 21 6"},null,-1)),n[117]||(n[117]=e.createElementVNode("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"},null,-1))],64)):r.name==="file-plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:51},[e.createCommentVNode(" File Plus / New File "),n[118]||(n[118]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[119]||(n[119]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[120]||(n[120]=e.createElementVNode("line",{x1:"12",y1:"18",x2:"12",y2:"12"},null,-1)),n[121]||(n[121]=e.createElementVNode("line",{x1:"9",y1:"15",x2:"15",y2:"15"},null,-1))],64)):r.name==="message-square"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:52},[e.createCommentVNode(" Folder Plus / New Folder "),n[122]||(n[122]=e.createElementVNode("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"},null,-1))],64)):r.name==="folder-plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:53},[n[123]||(n[123]=e.createElementVNode("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v3"},null,-1)),n[124]||(n[124]=e.createElementVNode("line",{x1:"12",y1:"11",x2:"12",y2:"17"},null,-1)),n[125]||(n[125]=e.createElementVNode("line",{x1:"9",y1:"14",x2:"15",y2:"14"},null,-1))],64)):r.name==="brain"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:54},[e.createCommentVNode(" Brain / Thinking "),n[126]||(n[126]=e.createElementVNode("path",{d:"M12 2a4 4 0 0 0-4 4v1a5 5 0 0 0-5 5v1a4 4 0 0 0 3 3.87V17a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-.13A4 4 0 0 0 21 13v-1a5 5 0 0 0-5-5V6a4 4 0 0 0-4-4z"},null,-1)),n[127]||(n[127]=e.createElementVNode("path",{d:"M9 12v2"},null,-1)),n[128]||(n[128]=e.createElementVNode("path",{d:"M15 12v2"},null,-1)),n[129]||(n[129]=e.createElementVNode("path",{d:"M12 9v5"},null,-1))],64)):r.name==="check"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:55},[e.createCommentVNode(" Check / Success "),n[130]||(n[130]=e.createElementVNode("polyline",{points:"20 6 9 17 4 12"},null,-1))],64)):r.name==="clock"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:56},[e.createCommentVNode(" Clock / Pending "),n[131]||(n[131]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[132]||(n[132]=e.createElementVNode("polyline",{points:"12 6 12 12 16 14"},null,-1))],64)):r.name==="help"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:57},[e.createCommentVNode(" Help / Question "),n[133]||(n[133]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[134]||(n[134]=e.createElementVNode("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"},null,-1)),n[135]||(n[135]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"},null,-1))],64)):r.name==="shield"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:58},[e.createCommentVNode(" Shield / Approval "),n[136]||(n[136]=e.createElementVNode("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"},null,-1))],64)):r.name==="shield-off"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:59},[e.createCommentVNode(" Shield Off / No Review "),n[137]||(n[137]=e.createElementVNode("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"},null,-1)),n[138]||(n[138]=e.createElementVNode("line",{x1:"4",y1:"4",x2:"20",y2:"20",stroke:"currentColor","stroke-width":"2","stroke-linecap":"round"},null,-1))],64)):r.name==="code"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:60},[e.createCommentVNode(" Code / Brackets "),n[139]||(n[139]=e.createElementVNode("polyline",{points:"16 18 22 12 16 6"},null,-1)),n[140]||(n[140]=e.createElementVNode("polyline",{points:"8 6 2 12 8 18"},null,-1))],64)):r.name==="list"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:61},[e.createCommentVNode(" List / Menu "),n[141]||(n[141]=e.createStaticVNode('<line x1="8" y1="6" x2="21" y2="6" data-v-faf69761></line><line x1="8" y1="12" x2="21" y2="12" data-v-faf69761></line><line x1="8" y1="18" x2="21" y2="18" data-v-faf69761></line><line x1="3" y1="6" x2="3.01" y2="6" data-v-faf69761></line><line x1="3" y1="12" x2="3.01" y2="12" data-v-faf69761></line><line x1="3" y1="18" x2="3.01" y2="18" data-v-faf69761></line>',6))],64)):r.name==="layers"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:62},[e.createCommentVNode(" Layers / Stack / Context "),n[142]||(n[142]=e.createElementVNode("polygon",{points:"12 2 2 7 12 12 22 7 12 2"},null,-1)),n[143]||(n[143]=e.createElementVNode("polyline",{points:"2 17 12 22 22 17"},null,-1)),n[144]||(n[144]=e.createElementVNode("polyline",{points:"2 12 12 17 22 12"},null,-1))],64)):r.name==="eye"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:63},[e.createCommentVNode(" Eye / Show "),n[145]||(n[145]=e.createElementVNode("path",{d:"M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"},null,-1)),n[146]||(n[146]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"3"},null,-1))],64)):r.name==="eye-off"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:64},[e.createCommentVNode(" Eye Off / Hide "),n[147]||(n[147]=e.createElementVNode("path",{d:"M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"},null,-1)),n[148]||(n[148]=e.createElementVNode("path",{d:"M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"},null,-1)),n[149]||(n[149]=e.createElementVNode("line",{x1:"1",y1:"1",x2:"23",y2:"23"},null,-1))],64)):r.name==="bug"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:65},[e.createCommentVNode(" Bug "),n[150]||(n[150]=e.createStaticVNode('<rect x="8" y="2" width="8" height="4" rx="1" ry="1" data-v-faf69761></rect><path d="M20 12h-3a5 5 0 0 1-5 5 5 5 0 0 1-5-5H4" data-v-faf69761></path><path d="M4 8h16" data-v-faf69761></path><path d="M12 2v7" data-v-faf69761></path><path d="M9 17l-3 4" data-v-faf69761></path><path d="M15 17l3 4" data-v-faf69761></path>',6))],64)):r.name==="check-circle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:66},[e.createCommentVNode(" Check Circle "),n[151]||(n[151]=e.createElementVNode("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"},null,-1)),n[152]||(n[152]=e.createElementVNode("polyline",{points:"22 4 12 14.01 9 11.01"},null,-1))],64)):r.name==="book-open"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:67},[e.createCommentVNode(" Book Open / Documentation "),n[153]||(n[153]=e.createElementVNode("path",{d:"M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"},null,-1)),n[154]||(n[154]=e.createElementVNode("path",{d:"M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"},null,-1))],64)):r.name==="tool"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:68},[e.createCommentVNode(" Tool / Wrench alternate "),n[155]||(n[155]=e.createElementVNode("path",{d:"M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"},null,-1))],64)):r.name==="keyboard"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:69},[e.createCommentVNode(" Keyboard "),n[156]||(n[156]=e.createStaticVNode('<rect x="2" y="4" width="20" height="16" rx="2" ry="2" data-v-faf69761></rect><line x1="6" y1="8" x2="6.01" y2="8" data-v-faf69761></line><line x1="10" y1="8" x2="10.01" y2="8" data-v-faf69761></line><line x1="14" y1="8" x2="14.01" y2="8" data-v-faf69761></line><line x1="18" y1="8" x2="18.01" y2="8" data-v-faf69761></line><line x1="6" y1="12" x2="6.01" y2="12" data-v-faf69761></line><line x1="10" y1="12" x2="10.01" y2="12" data-v-faf69761></line><line x1="14" y1="12" x2="14.01" y2="12" data-v-faf69761></line><line x1="18" y1="12" x2="18.01" y2="12" data-v-faf69761></line><line x1="6" y1="16" x2="18" y2="16" data-v-faf69761></line>',10))],64)):r.name==="chevron-left"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:70},[e.createCommentVNode(" Chevron Left "),n[157]||(n[157]=e.createElementVNode("polyline",{points:"15 6 9 12 15 18"},null,-1))],64)):r.name==="grid"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:71},[e.createCommentVNode(" Grid / App Grid "),n[158]||(n[158]=e.createElementVNode("rect",{x:"3",y:"3",width:"7",height:"7"},null,-1)),n[159]||(n[159]=e.createElementVNode("rect",{x:"14",y:"3",width:"7",height:"7"},null,-1)),n[160]||(n[160]=e.createElementVNode("rect",{x:"14",y:"14",width:"7",height:"7"},null,-1)),n[161]||(n[161]=e.createElementVNode("rect",{x:"3",y:"14",width:"7",height:"7"},null,-1))],64)):r.name==="puzzle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:72},[e.createCommentVNode(" Puzzle / 插件 "),n[162]||(n[162]=e.createElementVNode("path",{d:"M4 7h3a2 2 0 0 1 4 0h9v9h-3a2 2 0 0 0-4 0H4z"},null,-1)),n[163]||(n[163]=e.createElementVNode("path",{d:"M11 7v9"},null,-1))],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:73},[n[164]||(n[164]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[165]||(n[165]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[166]||(n[166]=e.createElementVNode("line",{x1:"9",y1:"13",x2:"15",y2:"13"},null,-1)),n[167]||(n[167]=e.createElementVNode("line",{x1:"9",y1:"17",x2:"15",y2:"17"},null,-1))],64))],8,Ue))}},[["__scopeId","data-v-faf69761"]]),je={class:"me-field"},qe={class:"me-label"},We={class:"me-editor"},Je={class:"me-input-row"},Ze=["placeholder","onKeydown"],Ke={class:"me-tags"},Qe={key:0,class:"me-empty"},Xe=["onClick"],Ne=U({__name:"ModelEditor",props:{models:{type:Array,default:()=>[]},label:{type:String,default:"可用模型（回车或逗号分隔添加；支持整段粘贴）"},placeholder:{type:String,default:"输入模型名，回车添加…"}},emits:["change"],setup(r,{emit:t}){const n=r,l=t,o=e.ref(""),a=e.ref([...n.models]);e.watch(()=>n.models,y=>{a.value=[...y]});function s(){const y=o.value.split(/[\n,，]/).map(x=>x.trim()).filter(Boolean);let k=!1;for(const x of y)a.value.includes(x)||(a.value.push(x),k=!0);k&&l("change",[...a.value]),o.value=""}function p(y){const k=(y.clipboardData||window.clipboardData).getData("text");if(/[,\n，]/.test(k)){y.preventDefault();const x=k.split(/[\n,，]/).map(T=>T.trim()).filter(Boolean);let N=!1;for(const T of x)a.value.includes(T)||(a.value.push(T),N=!0);N&&l("change",[...a.value]),o.value=""}}function i(y){a.value.splice(y,1),l("change",[...a.value])}return(y,k)=>(e.openBlock(),e.createElementBlock("div",je,[e.createElementVNode("span",qe,e.toDisplayString(r.label),1),e.createElementVNode("div",We,[e.createElementVNode("div",Je,[e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":k[0]||(k[0]=x=>o.value=x),class:"me-input",placeholder:r.placeholder,onKeydown:e.withKeys(e.withModifiers(s,["prevent"]),["enter"]),onPaste:p},null,40,Ze),[[e.vModelText,o.value]]),e.createElementVNode("button",{class:"me-btn",onClick:s},"添加")]),e.createElementVNode("div",Ke,[a.value.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("span",Qe,"暂无模型——添加后 AI tab 的模型下拉会按服务商显示")),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,(x,N)=>(e.openBlock(),e.createElementBlock("span",{key:x+N,class:"me-tag"},[e.createTextVNode(e.toDisplayString(x)+" ",1),e.createElementVNode("button",{class:"me-x",title:"移除",onClick:T=>i(N)},"×",8,Xe)]))),128))])])]))}},[["__scopeId","data-v-a5e576a7"]]),Ye={class:"provider-manager"},_e={class:"pm-toolbar"},ve={class:"pm-count"},en={key:0,class:"pm-edit"},nn={class:"pm-field"},tn={class:"pm-field"},rn={class:"pm-field"},ln={class:"pm-params"},on={key:0,class:"pm-param-rows"},an=["title"],sn=["title"],cn=["onUpdate:modelValue"],dn=["onUpdate:modelValue","title"],pn=["value"],mn=["onUpdate:modelValue","min","step","placeholder","title"],gn=["onUpdate:modelValue","placeholder","title"],kn={key:1,class:"pm-params-empty"},hn={class:"pm-edit-actions"},fn=["disabled"],yn={key:1,class:"pm-cards"},un={key:0,class:"pm-edit"},En={class:"pm-edit-title"},bn={class:"pm-field"},xn=["value"],Vn={class:"pm-field"},Nn={class:"pm-field"},wn={class:"pm-params"},Tn={key:0,class:"pm-param-rows"},Bn=["title"],Sn=["title"],Cn=["onUpdate:modelValue"],In=["onUpdate:modelValue","title"],Pn=["value"],An=["onUpdate:modelValue","min","step","placeholder","title"],$n=["onUpdate:modelValue","placeholder","title"],Mn={key:1,class:"pm-params-empty"},Dn={class:"pm-edit-actions"},Rn=["disabled"],Gn={key:1,class:"pm-card"},Ln={class:"pm-card-head"},Fn=["title"],On={class:"pm-ops"},zn=["onClick"],Un=["onClick"],Hn=["title"],jn={class:"pm-ctx"},qn={class:"pm-models"},Wn={key:0,class:"pm-none"},Jn={key:0,class:"pm-params-summary"},Zn={key:2,class:"pm-empty"},Kn={key:3,class:"pm-error"},Qn=U({__name:"ProviderManager",props:{modelParamFields:{type:Array,default:()=>[]},modelEditor:{type:Object,default:()=>({})}},emits:["saved"],setup(r,{emit:t}){const n=t,l=r,o=e.ref([]),a=e.ref(""),s=e.ref({name:"",baseURL:"",contextMaxTokens:0}),p=e.ref([]),i=e.ref({}),y=e.ref(""),k=e.ref(!1);function x(){const V={};for(const u of l.modelParamFields)u.type==="checkbox"?V[u.name]=!1:u.type==="number"?V[u.name]=0:V[u.name]="";return V}function N(V){const u=E.state.settings&&E.state.settings.modelParams||{};return JSON.parse(JSON.stringify(u[V]||{}))}async function T(){try{const V=await A.getModels();o.value=(V.providers||[]).map(u=>({name:u,baseURL:(V.providerBaseURLs||{})[u]||"",contextMaxTokens:(V.providerContexts||{})[u]||0,models:(V.models||{})[u]||[]})),y.value=""}catch(V){y.value="加载服务商失败: "+(V.message||V)}}e.onMounted(T);function m(){a.value="__new__",s.value={name:"",baseURL:"",contextMaxTokens:0},p.value=[],i.value={},y.value=""}function P(V){a.value=V.name,s.value={name:V.name,baseURL:V.baseURL,contextMaxTokens:V.contextMaxTokens||0},p.value=[...V.models||[]];const u=N(V.name),h=x();for(const g of p.value)u[g]||(u[g]={...h});i.value=u,y.value=""}function B(V){const u={...i.value},h=x();for(const g of V)u[g]||(u[g]={...h});for(const g of Object.keys(u))V.includes(g)||delete u[g];i.value=u,p.value=V}function G(){a.value="",y.value=""}function M(){const V={};for(const u of o.value)V[u.name]={baseURL:u.baseURL,models:u.models,contextMaxTokens:u.contextMaxTokens||0};return V}async function O(){const V=s.value.name.trim()||(a.value!=="__new__"?a.value:"");if(!V){y.value="服务商名称不能为空";return}const u=M();if(a.value==="__new__"&&u[V]){y.value=`服务商「${V}」已存在`;return}u[V]={baseURL:s.value.baseURL.trim(),models:p.value,contextMaxTokens:Math.max(0,Number(s.value.contextMaxTokens)||0)},k.value=!0;try{await A.saveModels(u),await J(V),a.value="",await T(),n("saved")}catch(h){y.value="保存失败: "+(h.message||h)}finally{k.value=!1}}async function J(V){let u={};try{const f=await A.apiGet("/settings");u=f&&f.settings||{}}catch{}const h=JSON.parse(JSON.stringify(u.modelParams||{})),g={};for(const[f,d]of Object.entries(i.value)){const b=d||{},w={};for(const D of l.modelParamFields){const z=b[D.name];D.type==="checkbox"?z===!0&&(w[D.name]=!0):D.type==="number"?Number(z)>0&&(w[D.name]=Number(z)):z!==""&&z!==void 0&&z!==null&&(w[D.name]=z)}Object.keys(w).length&&(g[f]=w)}Object.keys(g).length?h[V]=g:delete h[V];const c={...u,modelParams:h};await A.apiPut("/settings",{settings:c,pluginSettings:u.pluginSettings||{}}),E.state.settings=c}async function L(V){if(!window.confirm(`删除服务商「${V.name}」？
（AI tab 将不再可选该服务商）`))return;const u=M();delete u[V.name];try{await A.saveModels(u);let h={};try{const c=await A.apiGet("/settings");h=c&&c.settings||{}}catch{}const g=JSON.parse(JSON.stringify(h.modelParams||{}));if(g[V.name]){delete g[V.name];const c={...h,modelParams:g};await A.apiPut("/settings",{settings:c,pluginSettings:h.pluginSettings||{}}),E.state.settings=c}await T(),n("saved")}catch(h){y.value="删除失败: "+(h.message||h)}}function F(V){const h=(E.state.settings&&E.state.settings.modelParams||{})[V]||{},g=Object.keys(h).length;return g?"模型参数已配置 "+g+" 个":""}return(V,u)=>(e.openBlock(),e.createElementBlock("div",Ye,[e.createCommentVNode(" 工具栏 "),e.createElementVNode("div",_e,[e.createElementVNode("span",ve,e.toDisplayString(o.value.length)+" 个服务商",1),e.createElementVNode("button",{class:"pm-btn pm-primary",onClick:m},"+ 新增服务商")]),e.createCommentVNode(" 新增表单（工具栏下方展开，紧邻按钮不跳动） "),a.value==="__new__"?(e.openBlock(),e.createElementBlock("div",en,[u[9]||(u[9]=e.createElementVNode("div",{class:"pm-edit-title"},"新增服务商",-1)),e.createElementVNode("div",nn,[u[5]||(u[5]=e.createElementVNode("span",{class:"pm-field-label"},"服务商名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":u[0]||(u[0]=h=>s.value.name=h),placeholder:"如 deepseek"},null,512),[[e.vModelText,s.value.name]])]),e.createElementVNode("div",tn,[u[6]||(u[6]=e.createElementVNode("span",{class:"pm-field-label"},"API URL（完整端点）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":u[1]||(u[1]=h=>s.value.baseURL=h),placeholder:"https://api.deepseek.com/v1/chat/completions"},null,512),[[e.vModelText,s.value.baseURL]])]),e.createElementVNode("div",rn,[u[7]||(u[7]=e.createElementVNode("span",{class:"pm-field-label"},"上下文大小（Token）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":u[2]||(u[2]=h=>s.value.contextMaxTokens=h),type:"number",min:"0",step:"1000",placeholder:"0=不限制（模型级未配置时的默认窗口）"},null,512),[[e.vModelText,s.value.contextMaxTokens]])]),e.createVNode(Ne,{models:p.value,label:r.modelEditor.label||"可用模型（回车或逗号分隔添加；支持整段粘贴）",placeholder:r.modelEditor.placeholder||"输入模型名，回车添加…",onChange:B},null,8,["models","label","placeholder"]),e.createElementVNode("div",ln,[u[8]||(u[8]=e.createElementVNode("div",{class:"pm-params-title"},"模型参数（每模型独立配置；对话里也可临时切换思考档位）",-1)),p.value.length?(e.openBlock(),e.createElementBlock("div",on,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(p.value,h=>(e.openBlock(),e.createElementBlock("div",{key:h,class:"pm-param-row"},[e.createElementVNode("span",{class:"pm-param-model",title:h},e.toDisplayString(h),9,an),e.createCommentVNode(" ★ 2026-08-21 schema 驱动：按 modelParamFields 动态渲染（checkbox/select/number/text） "),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.modelParamFields,g=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:g.name},[g.type==="checkbox"?(e.openBlock(),e.createElementBlock("label",{key:0,class:"pm-param-check",title:g.hint||g.label},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":c=>i.value[h][g.name]=c},null,8,cn),[[e.vModelCheckbox,i.value[h][g.name]]]),e.createTextVNode(" "+e.toDisplayString(g.label),1)],8,sn)):g.type==="select"?e.withDirectives((e.openBlock(),e.createElementBlock("select",{key:1,"onUpdate:modelValue":c=>i.value[h][g.name]=c,title:g.hint||g.label},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(g.options||[],c=>(e.openBlock(),e.createElementBlock("option",{key:"o"+c,value:c},e.toDisplayString(c===""?g.label+"默认":c),9,pn))),128))],8,dn)),[[e.vModelSelect,i.value[h][g.name]]]):g.type==="number"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:2,"onUpdate:modelValue":c=>i.value[h][g.name]=c,type:"number",min:g.min??0,step:g.step??1,placeholder:g.label,title:g.hint||g.label},null,8,mn)),[[e.vModelText,i.value[h][g.name],void 0,{number:!0}]]):e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:3,"onUpdate:modelValue":c=>i.value[h][g.name]=c,type:"text",placeholder:g.label,title:g.hint||g.label},null,8,gn)),[[e.vModelText,i.value[h][g.name]]])],64))),128))]))),128))])):(e.openBlock(),e.createElementBlock("div",kn,"添加模型后，可逐模型配置参数（温度/思考/输出/上下文/多模态…）"))]),e.createElementVNode("div",hn,[e.createElementVNode("button",{class:"pm-btn pm-primary",disabled:k.value,onClick:O},e.toDisplayString(k.value?"保存中…":"保存服务商"),9,fn),e.createElementVNode("button",{class:"pm-btn",onClick:G},"取消")])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 服务商卡片列表（编辑时在卡片位置就地展开表单，不跳顶） "),o.value.length?(e.openBlock(),e.createElementBlock("div",yn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(o.value,h=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:h.name},[a.value===h.name?(e.openBlock(),e.createElementBlock("div",un,[e.createElementVNode("div",En,"编辑服务商："+e.toDisplayString(h.name),1),e.createElementVNode("div",bn,[u[10]||(u[10]=e.createElementVNode("span",{class:"pm-field-label"},"服务商名称",-1)),e.createElementVNode("input",{value:h.name,disabled:""},null,8,xn)]),e.createElementVNode("div",Vn,[u[11]||(u[11]=e.createElementVNode("span",{class:"pm-field-label"},"API URL（完整端点）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":u[3]||(u[3]=g=>s.value.baseURL=g),placeholder:"https://api.deepseek.com/v1/chat/completions"},null,512),[[e.vModelText,s.value.baseURL]])]),e.createElementVNode("div",Nn,[u[12]||(u[12]=e.createElementVNode("span",{class:"pm-field-label"},"上下文大小（Token）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":u[4]||(u[4]=g=>s.value.contextMaxTokens=g),type:"number",min:"0",step:"1000",placeholder:"0=不限制（模型级未配置时的默认窗口）"},null,512),[[e.vModelText,s.value.contextMaxTokens]])]),e.createVNode(Ne,{models:p.value,label:r.modelEditor.label||"可用模型（回车或逗号分隔添加；支持整段粘贴）",placeholder:r.modelEditor.placeholder||"输入模型名，回车添加…",onChange:B},null,8,["models","label","placeholder"]),e.createElementVNode("div",wn,[u[13]||(u[13]=e.createElementVNode("div",{class:"pm-params-title"},"模型参数（每模型独立配置；对话里也可临时切换思考档位）",-1)),p.value.length?(e.openBlock(),e.createElementBlock("div",Tn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(p.value,g=>(e.openBlock(),e.createElementBlock("div",{key:g,class:"pm-param-row"},[e.createElementVNode("span",{class:"pm-param-model",title:g},e.toDisplayString(g),9,Bn),e.createCommentVNode(" ★ 2026-08-21 schema 驱动：按 modelParamFields 动态渲染（checkbox/select/number/text） "),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.modelParamFields,c=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:c.name},[c.type==="checkbox"?(e.openBlock(),e.createElementBlock("label",{key:0,class:"pm-param-check",title:c.hint||c.label},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":f=>i.value[g][c.name]=f},null,8,Cn),[[e.vModelCheckbox,i.value[g][c.name]]]),e.createTextVNode(" "+e.toDisplayString(c.label),1)],8,Sn)):c.type==="select"?e.withDirectives((e.openBlock(),e.createElementBlock("select",{key:1,"onUpdate:modelValue":f=>i.value[g][c.name]=f,title:c.hint||c.label},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(c.options||[],f=>(e.openBlock(),e.createElementBlock("option",{key:"o"+f,value:f},e.toDisplayString(f===""?c.label+"默认":f),9,Pn))),128))],8,In)),[[e.vModelSelect,i.value[g][c.name]]]):c.type==="number"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:2,"onUpdate:modelValue":f=>i.value[g][c.name]=f,type:"number",min:c.min??0,step:c.step??1,placeholder:c.label,title:c.hint||c.label},null,8,An)),[[e.vModelText,i.value[g][c.name],void 0,{number:!0}]]):e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:3,"onUpdate:modelValue":f=>i.value[g][c.name]=f,type:"text",placeholder:c.label,title:c.hint||c.label},null,8,$n)),[[e.vModelText,i.value[g][c.name]]])],64))),128))]))),128))])):(e.openBlock(),e.createElementBlock("div",Mn,"添加模型后，可逐模型配置参数（温度/思考/输出/上下文/多模态…）"))]),e.createElementVNode("div",Dn,[e.createElementVNode("button",{class:"pm-btn pm-primary",disabled:k.value,onClick:O},e.toDisplayString(k.value?"保存中…":"保存服务商"),9,Rn),e.createElementVNode("button",{class:"pm-btn",onClick:G},"取消")])])):(e.openBlock(),e.createElementBlock("div",Gn,[e.createElementVNode("div",Ln,[e.createElementVNode("span",{class:"pm-name",title:h.name},e.toDisplayString(h.name),9,Fn),e.createElementVNode("div",On,[e.createElementVNode("button",{class:"pm-btn pm-small",onClick:g=>P(h)},"编辑",8,zn),e.createElementVNode("button",{class:"pm-btn pm-small pm-danger",onClick:g=>L(h)},"删除",8,Un)])]),e.createElementVNode("div",{class:"pm-url",title:h.baseURL},e.toDisplayString(h.baseURL||"未配置 API URL"),9,Hn),e.createElementVNode("div",jn,e.toDisplayString(h.contextMaxTokens>0?"上下文 "+(h.contextMaxTokens/1e3).toFixed(0)+"K Token":"上下文 未限制"),1),e.createElementVNode("div",qn,[h.models.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("span",Wn,"（未配置模型）")),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(h.models,g=>(e.openBlock(),e.createElementBlock("span",{key:g,class:"pm-tag"},e.toDisplayString(g),1))),128))]),F(h.name)?(e.openBlock(),e.createElementBlock("div",Jn,e.toDisplayString(F(h.name)),1)):e.createCommentVNode("v-if",!0)]))],64))),128))])):a.value!=="__new__"?(e.openBlock(),e.createElementBlock("div",Zn,"暂无服务商，点「+ 新增服务商」添加")):e.createCommentVNode("v-if",!0),y.value?(e.openBlock(),e.createElementBlock("div",Kn,e.toDisplayString(y.value),1)):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-9d901dab"]]),Xn={class:"pm-manager"},Yn={class:"mgm-toolbar"},_n={class:"mgm-count"},vn={key:0,class:"mgm-edit"},et={class:"mgm-edit-title"},nt={class:"mgm-field"},tt={class:"mgm-field-label"},rt={key:0,class:"mgm-required"},lt=["onUpdate:modelValue","onChange"],ot=["value"],at=["onUpdate:modelValue","placeholder"],st=["onUpdate:modelValue","placeholder"],it={key:3,class:"mgm-field-hint"},ct={class:"mgm-edit-actions"},dt=["disabled"],pt={key:1,class:"mgm-cards"},mt={class:"mgm-card-head"},gt=["title"],kt={key:0,class:"pm-active-badge"},ht={class:"mgm-ops"},ft=["disabled","onClick"],yt=["onClick"],ut=["onClick"],Et={class:"pm-preview"},bt={class:"pm-snap-row"},xt={class:"pm-snap-row"},Vt={key:2,class:"mgm-empty"},Nt={key:3,class:"mgm-error"},wt=U({__name:"PresetManager",props:{presetFields:{type:Array,default:()=>[]}},emits:["saved"],setup(r,{expose:t,emit:n}){const l=r,o=n,a=e.computed(()=>Object.keys(presets.value||{})),s=e.ref(""),p=e.ref(!1),i=e.ref(""),y=e.ref(!1),k=e.ref(""),x=e.ref(""),N=e.ref(null),T=e.computed(()=>N.value&&N.value.providers||[]),m=e.ref({name:"",provider:"",baseURL:"",apiKey:""});function P(c={}){const f={name:"",provider:"",baseURL:"",apiKey:""};for(const d of l.presetFields)d.name in f||(f[d.name]="");return Object.assign(f,c)}function B(c){x.value=c,setTimeout(()=>{x.value===c&&(x.value="")},4e3)}async function G(){try{const[c,f,d]=await Promise.all([A.getAiPresets().catch(()=>({presets:{}})),A.apiGet("/settings").catch(()=>({settings:{}})),A.getModels().catch(()=>null)]);presets.value=c&&c.presets||{},s.value=f&&f.settings&&f.settings.preset||"",N.value=d}catch(c){B("加载失败: "+(c.message||c))}}function M(c){const f=N.value||{};return{baseURL:f.providerBaseURLs&&f.providerBaseURLs[c]||"",models:f.models&&f.models[c]||[]}}function O(){const c=window&&window.__PAIRCODE_CORE&&window.__PAIRCODE_CORE.uiState&&window.__PAIRCODE_CORE.uiState.state&&window.__PAIRCODE_CORE.uiState.state.settings||{};let f={};c.preset&&presets.value&&presets.value[c.preset]&&(f=presets.value[c.preset]);const d=f.provider||c.provider||T.value[0]||"",b=M(d);m.value=P({provider:d,baseURL:f.baseURL||c.baseURL||b.baseURL||"",apiKey:f.apiKey||""}),i.value="",p.value=!0}function J(c){const f=presets.value&&presets.value[c]||{};m.value=P({name:c,provider:f.provider||"",baseURL:f.baseURL||"",apiKey:f.apiKey||""}),i.value=c,p.value=!0}function L(){p.value=!1,i.value=""}function F(c){return c.source==="providers"?T.value||[]:c.options||[]}function V(c){if(c.name==="provider"&&m.value.provider){const f=M(m.value.provider);m.value.baseURL=f.baseURL||""}}e.watch(()=>m.value.provider,(c,f)=>{!p.value||f===""||c!==f&&V({name:"provider"})});async function u(){const c=m.value.name.trim();if(!c){B("请输入配置名称");return}const f=m.value.provider||"";if(!f){B("请选择服务商");return}for(const b of l.presetFields)if(b.required&&!m.value[b.name]){B("请填写"+b.label);return}const d=M(f);m.value.baseURL||(m.value.baseURL=d.baseURL||""),y.value=!0,x.value="";try{const b={provider:f,baseURL:m.value.baseURL,apiKey:m.value.apiKey};for(const w of l.presetFields)w.name!=="provider"&&w.name!=="apiKey"&&(b[w.name]=m.value[w.name]);if(i.value&&i.value!==c){const w={...presets.value||{}};w[c]=b,delete w[i.value];const D=await A.saveAiPresets(w);if(!(D&&D.ok)){B(D&&D.error||"保存失败");return}presets.value=w,s.value===i.value&&(s.value=c,await A.apiPut("/settings",{settings:{preset:c},pluginSettings:{}}).catch(()=>{}))}else{const w=await A.saveAiPreset("save",c,b);if(!(w&&w.ok)){B(w&&w.error||"保存失败");return}presets.value=w.presets||presets.value}L(),o("saved")}catch(b){B("保存失败: "+(b.message||b))}finally{y.value=!1}}async function h(c){k.value=c,x.value="";try{const f=await A.saveAiPreset("apply",c);f&&f.ok?(s.value=c,o("saved")):B(f&&f.error||"应用失败")}catch(f){B("应用失败: "+(f.message||f))}finally{k.value=""}}async function g(c){if(confirm("删除配置「"+c+"」？")){x.value="";try{const f=await A.saveAiPreset("delete",c);f&&f.ok?(presets.value=f.presets||presets.value,s.value===c&&(s.value=""),o("saved")):B(f&&f.error||"删除失败")}catch(f){B("删除失败: "+(f.message||f))}}}return e.onMounted(G),t({load:G}),(c,f)=>(e.openBlock(),e.createElementBlock("div",Xn,[e.createCommentVNode(" 工具栏 "),e.createElementVNode("div",Yn,[e.createElementVNode("span",_n,e.toDisplayString(a.value.length)+" 个配置",1),e.createElementVNode("button",{class:"mgm-btn mgm-primary",onClick:O,title:"添加一条 AI 配置（服务商 + API Key）"},"＋ 添加新配置")]),e.createCommentVNode(" 添加 / 编辑表单（点击添加/编辑才弹出） "),p.value?(e.openBlock(),e.createElementBlock("div",vn,[e.createElementVNode("div",et,e.toDisplayString(i.value?"编辑配置："+i.value:"添加新配置"),1),e.createElementVNode("div",nt,[f[1]||(f[1]=e.createElementVNode("span",{class:"mgm-field-label"},"配置名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":f[0]||(f[0]=d=>m.value.name=d),type:"text",placeholder:"如：主力 / 写作备用…",onKeydown:e.withKeys(u,["enter"])},null,544),[[e.vModelText,m.value.name]])]),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(r.presetFields,d=>(e.openBlock(),e.createElementBlock("div",{key:d.name,class:"mgm-field"},[e.createElementVNode("span",tt,[e.createTextVNode(e.toDisplayString(d.label),1),d.required?(e.openBlock(),e.createElementBlock("span",rt,"*")):e.createCommentVNode("v-if",!0)]),e.createCommentVNode(" select 类型（服务商选择） "),d.type==="select"?e.withDirectives((e.openBlock(),e.createElementBlock("select",{key:0,"onUpdate:modelValue":b=>m.value[d.name]=b,class:"mgm-select",onChange:b=>V(d)},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(F(d),b=>(e.openBlock(),e.createElementBlock("option",{key:b,value:b},e.toDisplayString(b),9,ot))),128))],40,lt)),[[e.vModelSelect,m.value[d.name]]]):d.type==="password"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" password 类型（API Key） "),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":b=>m.value[d.name]=b,type:"password",placeholder:d.placeholder||""},null,8,at),[[e.vModelText,m.value[d.name]]])],2112)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" text 兜底 "),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":b=>m.value[d.name]=b,type:"text",placeholder:d.placeholder||""},null,8,st),[[e.vModelText,m.value[d.name]]])],2112)),d.hint?(e.openBlock(),e.createElementBlock("span",it,e.toDisplayString(d.hint),1)):e.createCommentVNode("v-if",!0)]))),128)),e.createElementVNode("div",ct,[e.createElementVNode("button",{class:"mgm-btn mgm-primary",disabled:y.value,onClick:u},e.toDisplayString(y.value?"保存中…":"保存配置"),9,dt),e.createElementVNode("button",{class:"mgm-btn",onClick:L},"取消")])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 配置卡片列表（主视图） "),a.value.length?(e.openBlock(),e.createElementBlock("div",pt,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,d=>(e.openBlock(),e.createElementBlock("div",{key:d,class:e.normalizeClass(["mgm-card",{"pm-active":d===s.value}])},[e.createElementVNode("div",mt,[e.createElementVNode("span",{class:"mgm-name",title:d},[e.createTextVNode(e.toDisplayString(d),1),d===s.value?(e.openBlock(),e.createElementBlock("span",kt,"使用中")):e.createCommentVNode("v-if",!0)],8,gt),e.createElementVNode("div",ht,[e.createElementVNode("button",{class:"mgm-btn mgm-small",disabled:k.value===d,onClick:b=>h(d)},e.toDisplayString(k.value===d?"应用中…":"应用"),9,ft),e.createElementVNode("button",{class:"mgm-btn mgm-small",onClick:b=>J(d)},"编辑",8,yt),e.createElementVNode("button",{class:"mgm-btn mgm-small mgm-danger",onClick:b=>g(d)},"删除",8,ut)])]),e.createElementVNode("div",Et,[e.createElementVNode("div",bt,[f[2]||(f[2]=e.createElementVNode("span",null,"服务商",-1)),e.createElementVNode("b",null,e.toDisplayString((c.presets[d]||{}).provider||"—"),1)]),e.createElementVNode("div",xt,[f[3]||(f[3]=e.createElementVNode("span",null,"API Key",-1)),e.createElementVNode("b",null,e.toDisplayString((c.presets[d]||{}).apiKey?"已配置":"未配置"),1)])])],2))),128))])):p.value?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("div",Vt,[e.createElementVNode("div",{class:"mgm-empty-box"},[f[4]||(f[4]=e.createElementVNode("div",{class:"mgm-empty-title"},"还没有 AI 配置",-1)),f[5]||(f[5]=e.createElementVNode("div",{class:"mgm-empty-sub"},"添加一条服务商 + API Key，保存后即可在对话面板中选模型。",-1)),e.createElementVNode("button",{class:"mgm-btn mgm-primary",onClick:O},"＋ 添加新配置")])])),x.value?(e.openBlock(),e.createElementBlock("div",Nt,e.toDisplayString(x.value),1)):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-b1ec221c"]]),Tt={class:"modal-content"},Bt={class:"modal-body"},St={key:0,class:"settings-tabs"},Ct={key:0,class:"settings-tabs-filter-wrap"},It=["onClick"],Pt={key:1,class:"settings-tabs-none"},At={class:"settings-content"},$t={key:0},Mt={key:0,class:"group-title"},Dt=["title"],Rt=["title"],Gt=["onUpdate:modelValue"],Lt=["title"],Ft={class:"field-control"},Ot=["type","onUpdate:modelValue","placeholder"],zt=["onUpdate:modelValue","min","max","step"],Ut=["onUpdate:modelValue","onChange"],Ht=["value"],jt=["onUpdate:modelValue","placeholder"],qt={class:"slider-row"},Wt=["onUpdate:modelValue","min","max","step"],Jt={class:"slider-val"},Zt={class:"color-row"},Kt=["onUpdate:modelValue"],Qt={class:"color-code"},Xt=["value","onInput","placeholder"],Yt=["placeholder"],_t=["onUpdate:modelValue"],vt={key:0,class:"setting-hint"},er={key:0,class:"settings-empty"},nr=U({__name:"SettingsModal",emits:["close"],setup(r,{emit:t}){const n=t,l=e.ref(""),o=e.ref(""),a=e.computed(()=>{const h=(E.state.pluginSchemas||[]).map(g=>({key:g.key,title:g.title||g.key,groups:p(g.fields||[])}));return h.length&&!l.value&&(l.value=h[0].key),h}),s=e.computed(()=>{const h=o.value.trim().toLowerCase();if(!h)return a.value;const g=a.value.find(f=>f.key===l.value),c=a.value.filter(f=>(f.title||f.key).toLowerCase().includes(h));return g&&!c.includes(g)&&c.unshift(g),c});function p(h){const g=[],c={};for(const f of h){const d=f.group||"";c[d]||(c[d]=[],g.push({title:d,fields:c[d]})),c[d].push(f)}return g}const i=e.ref(null);let y="";async function k(){try{i.value=await A.getModels()}catch{i.value=null}}function x(h){return h?(i.value&&i.value.models||{})[h]||[]:[]}function N(h,g){var c,f,d;if(g.optionsSource==="models"){const b=(c=m[h])==null?void 0:c[g.name],w=x((f=m.ai)==null?void 0:f.provider);return b&&!w.includes(b)?[...w,b]:w}if(g.optionsSource==="providers"){const b=i.value&&i.value.providers||[];if(b.length){const w=(d=m[h])==null?void 0:d[g.name];return w&&!b.includes(w)?[...b,w]:b}return g.options||[]}return g.options||[]}function T(h){if(!m.ai)return;const g=m.ai,c=h.linkFields||(h.linkField?[h.linkField]:[]);if(!c.length)return;const f=i.value||{},d=f.providerBaseURLs||{},b=f.providerKeys||{},w=g.provider,D=d[y];for(const z of c)if(z==="apiKey")g[z]=b[w]||"";else{const Ee=g[z];(Ee===void 0||Ee===""||D&&Ee===D)&&(g[z]=d[w]||"")}y=w}const m=e.reactive({}),P=e.ref("");function B(h){switch(h){case"checkbox":return!1;case"number":return 0;case"tags":return[];default:return""}}function G(){for(const f of Object.keys(m))delete m[f];const h=E.state.settings||{};y=h.provider||"";const g=h.pluginSettings||{};for(const f of E.state.pluginSchemas||[]){m[f.key]={};for(const d of f.fields||[]){let b;if(!(d.type==="project"||d.type==="provider-manager"||d.type==="model-params-manager"||d.type==="preset-manager")){if(d.binding)b=h[d.binding]!==void 0?h[d.binding]:d.default;else{const w=g[f.key]||{};b=w[d.name]!==void 0?w[d.name]:d.default}b===void 0&&(b=B(d.type)),d.type==="checkbox"&&(b=!!b),d.type==="number"&&(b=typeof b=="number"?b:Number(b)||0),d.type==="tags"&&(b=Array.isArray(b)?b:[]),m[f.key][d.name]=b}}}const c=(E.state.pluginSchemas||[]).some(f=>(f.fields||[]).some(d=>d.type==="project"));P.value="",c&&J()}function M(h,g){var f;const c=(f=m[h])==null?void 0:f[g.name];return Array.isArray(c)?c.join(", "):c||""}function O(h,g,c){m[h][g.name]=c.target.value.split(",").map(f=>f.trim()).filter(Boolean)}async function J(){try{const h=await A.getInstructions("project");P.value=h.content||""}catch{}}function L(){var h;G(),(h=E.state.settings)!=null&&h.theme&&E.applyTheme(E.state.settings.theme)}const F=()=>{L()};async function V(){try{const h=await A.apiGet("/settings");h&&h.settings&&(E.state.settings=h.settings,await k(),L())}catch{}}const u=async()=>{try{let h={};try{const d=await A.apiGet("/settings");h=d&&d.settings||{}}catch{}const g={...h},c={...h.pluginSettings||{}};let f=!1;for(const d of E.state.pluginSchemas||[]){const b=m[d.key]||{};for(const w of d.fields||[]){if(w.type==="project"){await A.saveInstructions("project",P.value);continue}if(w.type==="provider-manager"||w.type==="model-params-manager"||w.type==="preset-manager")continue;const D=b[w.name];w.binding?(w.name==="theme"&&D!==g[w.binding]&&(f=!0),g[w.binding]=D):(c[d.key]||(c[d.key]={}),c[d.key][w.name]=D)}}await A.apiPut("/settings",{settings:g,pluginSettings:c}),E.state.settings=g,f&&E.applyTheme(g.theme),window.$toast("设置已保存","success"),n("close")}catch(h){window.$toast("保存失败: "+h.message,"error")}};return e.onMounted(async()=>{await k(),L()}),(h,g)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:g[3]||(g[3]=e.withModifiers(c=>h.$emit("close"),["self"]))},[e.createElementVNode("div",Tt,[e.createElementVNode("h2",null,[e.createVNode($,{name:"settings",size:18}),g[4]||(g[4]=e.createTextVNode(" 设置 ",-1)),e.createElementVNode("button",{class:"modal-close",onClick:g[0]||(g[0]=c=>h.$emit("close"))},"×")]),e.createElementVNode("div",Bt,[e.createCommentVNode(" ═══ 纯 schema 驱动：所有配置 tab 由插件 ctx.registerSettings 注册 ═══ "),a.value.length?(e.openBlock(),e.createElementBlock("div",St,[a.value.length>6?(e.openBlock(),e.createElementBlock("div",Ct,[e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":g[1]||(g[1]=c=>o.value=c),class:"settings-tabs-filter",type:"text",placeholder:"筛选设置…"},null,512),[[e.vModelText,o.value]])])):e.createCommentVNode("v-if",!0),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(s.value,c=>(e.openBlock(),e.createElementBlock("button",{key:c.key,class:e.normalizeClass(["settings-tab",{active:l.value===c.key}]),onClick:f=>l.value=c.key},e.toDisplayString(c.title),11,It))),128)),s.value.length===0?(e.openBlock(),e.createElementBlock("div",Pt,"无匹配设置")):e.createCommentVNode("v-if",!0)])):e.createCommentVNode("v-if",!0),e.createElementVNode("div",At,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,c=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:c.key},[l.value===c.key?(e.openBlock(),e.createElementBlock("div",$t,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(c.groups,f=>(e.openBlock(),e.createElementBlock("div",{key:f.title||"__main",class:"setting-group"},[f.title?(e.openBlock(),e.createElementBlock("div",Mt,e.toDisplayString(f.title),1)):e.createCommentVNode("v-if",!0),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(f.fields,d=>(e.openBlock(),e.createElementBlock("div",{key:d.name,class:e.normalizeClass(["setting-row",{"row-toggle":d.type==="checkbox"}])},[e.createCommentVNode(" checkbox：label 与开关同行 "),d.type==="checkbox"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:0},[e.createElementVNode("label",{class:"field-label",title:d.hint},e.toDisplayString(d.label),9,Dt),e.createElementVNode("label",{class:"pp-switch",title:d.hint},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":b=>m[c.key][d.name]=b},null,8,Gt),[[e.vModelCheckbox,m[c.key][d.name]]]),g[5]||(g[5]=e.createElementVNode("span",{class:"pp-switch-track"},null,-1))],8,Rt)],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" 其他类型：label 在上、控件在下、说明文字在控件下方（不挤占输入区） "),e.createElementVNode("label",{class:"field-label",title:d.hint},e.toDisplayString(d.label),9,Lt),e.createElementVNode("div",Ft,[e.createCommentVNode(" text / password "),d.type==="text"||d.type==="password"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:0,class:"field-input",type:d.type==="password"?"password":"text","onUpdate:modelValue":b=>m[c.key][d.name]=b,placeholder:d.placeholder},null,8,Ot)),[[e.vModelDynamic,m[c.key][d.name]]]):d.type==="number"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" number "),e.withDirectives(e.createElementVNode("input",{class:"field-input",type:"number","onUpdate:modelValue":b=>m[c.key][d.name]=b,min:d.min,max:d.max,step:d.step},null,8,zt),[[e.vModelText,m[c.key][d.name],void 0,{number:!0}]])],2112)):d.type==="select"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" select（optionsSource 驱动动态数据源：models=按服务商模型列表 / providers=服务商列表） "),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":b=>m[c.key][d.name]=b,class:"field-select",onChange:b=>T(d)},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(N(c.key,d),b=>(e.openBlock(),e.createElementBlock("option",{key:b,value:b},e.toDisplayString(b),9,Ht))),128))],40,Ut),[[e.vModelSelect,m[c.key][d.name]]])],2112)):d.type==="textarea"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createCommentVNode(" textarea "),e.withDirectives(e.createElementVNode("textarea",{"onUpdate:modelValue":b=>m[c.key][d.name]=b,class:"field-textarea",rows:"4",placeholder:d.placeholder},null,8,jt),[[e.vModelText,m[c.key][d.name]]])],2112)):d.type==="slider"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:4},[e.createCommentVNode(" slider "),e.createElementVNode("div",qt,[e.withDirectives(e.createElementVNode("input",{type:"range","onUpdate:modelValue":b=>m[c.key][d.name]=b,min:d.min!=null?d.min:0,max:d.max!=null?d.max:100,step:d.step||1},null,8,Wt),[[e.vModelText,m[c.key][d.name],void 0,{number:!0}]]),e.createElementVNode("span",Jt,e.toDisplayString(m[c.key][d.name]),1)])],2112)):d.type==="color"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:5},[e.createCommentVNode(" color "),e.createElementVNode("div",Zt,[e.withDirectives(e.createElementVNode("input",{type:"color","onUpdate:modelValue":b=>m[c.key][d.name]=b},null,8,Kt),[[e.vModelText,m[c.key][d.name]]]),e.createElementVNode("code",Qt,e.toDisplayString(m[c.key][d.name]),1)])],2112)):d.type==="tags"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:6},[e.createCommentVNode(" tags（逗号分隔数组） "),e.createElementVNode("input",{type:"text",class:"field-input",value:M(c.key,d),onInput:b=>O(c.key,d,b),placeholder:d.placeholder||"逗号分隔"},null,40,Xt)],2112)):d.type==="project"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:7},[e.createCommentVNode(" project（平台特殊：项目级指令，经 /api/instructions 读写） "),e.withDirectives(e.createElementVNode("textarea",{"onUpdate:modelValue":g[2]||(g[2]=b=>P.value=b),class:"field-textarea",rows:"4",placeholder:d.placeholder},null,8,Yt),[[e.vModelText,P.value]])],2112)):d.type==="provider-manager"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:8},[e.createCommentVNode(" provider-manager（服务商维护面板：CRUD /api/models，独立保存，不参与普通表单） "),e.createVNode(Qn,{"model-param-fields":d.modelParamFields||[],"model-editor":d.modelEditor||{},onSaved:k},null,8,["model-param-fields","model-editor"])],2112)):d.type==="preset-manager"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:9},[e.createCommentVNode(" preset-manager（AI 配置预设面板：CRUD /api/ai-presets，独立保存，不参与普通表单） "),e.createVNode(wt,{"preset-fields":d.presetFields||[],onSaved:V},null,8,["preset-fields"])],2112)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:10},[e.createCommentVNode(" 兜底 text "),e.withDirectives(e.createElementVNode("input",{class:"field-input",type:"text","onUpdate:modelValue":b=>m[c.key][d.name]=b},null,8,_t),[[e.vModelText,m[c.key][d.name]]])],2112))]),d.hint?(e.openBlock(),e.createElementBlock("span",vt,e.toDisplayString(d.hint),1)):e.createCommentVNode("v-if",!0)],64))],2))),128))]))),128))])):e.createCommentVNode("v-if",!0)],64))),128)),a.value.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("div",er,"暂无配置项（等待插件注册…）"))])]),e.createElementVNode("div",{class:"modal-footer"},[e.createElementVNode("button",{class:"btn-secondary",onClick:F},"撤销"),e.createElementVNode("button",{class:"btn-primary",onClick:u},"保存设置")])])]))}},[["__scopeId","data-v-1c734f53"]]),tr={class:"modal-content sys-modal"},rr={class:"modal-header"},lr={class:"modal-body"},or={key:0,class:"loading"},ar={key:1,class:"sys-info"},sr={class:"info-row"},ir={class:"info-row"},cr={class:"info-row"},dr={class:"info-row"},pr={class:"info-row"},mr={class:"info-row"},gr={class:"modal-footer"},kr=U({__name:"SystemModal",emits:["close"],setup(r,{emit:t}){const n=e.ref(!0),l=e.ref({});return e.onMounted(async()=>{try{l.value=await A.apiGet("/system/info")}catch{}n.value=!1}),(o,a)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:a[2]||(a[2]=e.withModifiers(s=>o.$emit("close"),["self"]))},[e.createElementVNode("div",tr,[e.createElementVNode("div",rr,[a[3]||(a[3]=e.createElementVNode("h2",null,"ℹ 系统信息",-1)),e.createElementVNode("button",{class:"modal-close",onClick:a[0]||(a[0]=s=>o.$emit("close"))},"×")]),e.createElementVNode("div",lr,[n.value?(e.openBlock(),e.createElementBlock("div",or,"加载中...")):(e.openBlock(),e.createElementBlock("div",ar,[e.createElementVNode("div",sr,[a[4]||(a[4]=e.createElementVNode("label",null,"主机名",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.hostname),1)]),e.createElementVNode("div",ir,[a[5]||(a[5]=e.createElementVNode("label",null,"当前目录",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.cwd),1)]),e.createElementVNode("div",cr,[a[6]||(a[6]=e.createElementVNode("label",null,"操作系统",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.os),1)]),e.createElementVNode("div",dr,[a[7]||(a[7]=e.createElementVNode("label",null,"Go 版本",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.goos),1)]),e.createElementVNode("div",pr,[a[8]||(a[8]=e.createElementVNode("label",null,"工作区",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.workspace),1)]),e.createElementVNode("div",mr,[a[9]||(a[9]=e.createElementVNode("label",null,"文件夹",-1)),e.createElementVNode("span",null,e.toDisplayString((l.value.folders||[]).join(", ")),1)])]))]),e.createElementVNode("div",gr,[e.createElementVNode("button",{class:"btn-secondary",onClick:a[1]||(a[1]=s=>o.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-c27b6ec9"]]),hr={class:"modal-content source-modal"},fr={class:"modal-header"},yr={class:"modal-footer"},ur=U({__name:"SourceModal",emits:["close"],setup(r){return(t,n)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:n[2]||(n[2]=e.withModifiers(l=>t.$emit("close"),["self"]))},[e.createElementVNode("div",hr,[e.createElementVNode("div",fr,[n[3]||(n[3]=e.createElementVNode("h2",null,"⎔ 源代码管理",-1)),e.createElementVNode("button",{class:"modal-close",onClick:n[0]||(n[0]=l=>t.$emit("close"))},"×")]),n[4]||(n[4]=e.createElementVNode("div",{class:"modal-body"},[e.createElementVNode("p",{style:{color:"var(--text-muted)","text-align":"center","margin-top":"40px"}},[e.createTextVNode(" Git 集成开发中"),e.createElementVNode("br"),e.createElementVNode("br"),e.createTextVNode(" 功能规划："),e.createElementVNode("br"),e.createTextVNode(" · Git 状态查看"),e.createElementVNode("br"),e.createTextVNode(" · 暂存/提交/推送"),e.createElementVNode("br"),e.createTextVNode(" · 分支管理"),e.createElementVNode("br"),e.createTextVNode(" · Diff 对比 ")])],-1)),e.createElementVNode("div",yr,[e.createElementVNode("button",{class:"btn-secondary",onClick:n[1]||(n[1]=l=>t.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-2e060397"]]),Er=`# 功能介绍\r
\r
PairCode IDE 是一款纯 Web 端的 AI 辅助编程开发环境。你只需打开浏览器，在对话面板中用自然语言描述需求，AI 就能理解你的意图，直接生成代码、修改文件、执行命令、管理版本——把 IDE 从工具变为你的编程搭档。\r
\r
---\r
\r
## AI 对话编程\r
\r
**用自然语言驱动整个开发流程，就像跟资深开发者聊天一样跟 AI 交流。**\r
\r
在右侧对话面板中，你只需用自然语言描述需求，AI 就会理解你的意图并自动完成相应操作。无论是"帮我写一个 REST API"还是"把这个函数改成异步的"，AI 都能立刻执行。\r
\r
- **流式实时输出** — AI 的思考过程和操作结果实时显示，你始终能看清它在想什么、做什么\r
- **透明可追溯** — 每一步操作都有详细上下文，不是黑盒\r
- **随时干预** — 如果 AI 方向跑偏，可以随时给出反馈，AI 会立即调整\r
\r
---\r
\r
## 自主编程模式\r
\r
**AI 独立完成复杂的多步骤任务，你只需做最关键的决定。**\r
\r
开启自主模式后，AI 能自己分析项目结构、扫描代码问题、制定修复计划并逐个执行。你可以在关键节点审核确认，其他步骤 AI 自动完成。执行进度实时可见，你可以随时暂停、中止或补充指令。\r
\r
**Agent 核心采用双层循环（turn/step）架构**：\r
- **turn / step 双层边界** — 每次工具执行都有独立的 step 事件（开始/结束/摘要），每轮用户交互是 turn，进度颗粒度清晰可追溯\r
- **inbox 双队列** — 任务转向（next-step）与后续追问（next-turn）分队列消费，多轮交互不粘连\r
- **消息组装与落盘对齐** — agentloop 编号与消息序列严格一致，历史恢复与实时流状态吻合\r
- **历史注入精简** — 去掉冗余前缀标注与时间戳，系统提示内置多轮规则，长对话上下文更干净\r
\r
---\r
\r
## 智能代码编辑器\r
\r
**内置浏览器端编辑器，让你在同一个窗口中完成所有编辑工作。**\r
\r
- **语法高亮** — 支持 Go、TypeScript、Python、Rust、Java、Vue、HTML、CSS 等主流语言\r
- **代码折叠** — 折叠函数和代码块，聚焦关键逻辑\r
- **多标签页** — 同时打开编辑多个文件，标签栏快捷切换\r
- **括号匹配与自动缩进** — 代码结构清晰可见\r
- **十六进制查看器** — 查看二进制文件的原始字节内容\r
- **图片预览** — 在编辑器中直接显示图片文件\r
\r
---\r
\r
## 文件管理\r
\r
**完整的工作区文件管理能力，所有操作一目了然。**\r
\r
- **目录树浏览** — 以树形结构展示项目目录，支持展开 / 折叠\r
- **文件操作** — 新建、编辑、保存、删除、重命名、移动文件\r
- **多文件夹工作区** — 同时管理多个目录，组合成一个统一的工作区\r
- **快速切换工作区** — 在最近使用的项目之间一键切换\r
- **文件搜索** — 按文件名快速定位\r
- **内容搜索** — 在整个工作区按关键词搜索代码内容\r
\r
---\r
\r
## Git 版本控制\r
\r
**在对话中完成所有 Git 操作，告别记忆复杂命令。**\r
\r
你只需用自然语言告诉 AI 你想做什么：\r
- "查看当前仓库状态"\r
- "暂存所有修改，提交信息为'修复登录校验'"\r
- "创建一个名为 feature-search 的分支"\r
- "从远程拉取最新代码"\r
\r
AI 会自动执行对应的 Git 操作并返回结果。你也可以通过 Git 面板查看文件变更的逐行对比。\r
\r
---\r
\r
## 内置终端\r
\r
**浏览器中的终端，无需切换窗口。**\r
\r
终端面板直接内嵌在 IDE 底部，打开即用。AI 也能自动使用终端执行命令、读取输出并分析结果。支持多标签页，方便同时运行不同任务。\r
\r
---\r
\r
## 帮助文档中心\r
\r
**结构化的帮助文档体系，快速找到你需要的信息。**\r
\r
帮助面板侧边栏按分类组织文档：\r
\r
| 分类 | 包含文档 |\r
|------|---------|\r
| **文档中心** | 快速开始、功能介绍、API 文档、工具文档、快捷键、常见问题 |\r
| **其他** | 更新日志 |\r
\r
- **按分类导航** — 文档归入"文档中心"分组，找什么一目了然\r
- **文档间跳转** — 关于面板与帮助面板之间可互相跳转\r
- **翻页浏览** — 文档底部支持上一页/下一页顺序阅读\r
- **搜索过滤** — 侧边栏搜索框可快速筛选文档\r
\r
---\r
\r
## API 二次开发支持\r
\r
**完整的 HTTP REST API + WebSocket 协议文档，支持第三方基于本 IDE 进行二次开发。**\r
\r
- **详细的请求/响应格式** — 每个 API 接口提供 JSON Schema 请求体、完整响应示例、字段说明和错误码\r
- **WebSocket 协议定义** — 完整的 AI 事件流协议文档（15+ 事件类型、数据结构、典型事件序列）\r
- **终端协议文档** — PTY WebSocket 的初始化流程、控制消息格式、白名单限制等\r
- **API 索引速查表** — 按功能分类列出所有 60+ API 端点，方便快速查找\r
\r
所有 API 仅监听本地回环地址，安全可控。\r
\r
---\r
\r
## 代码知识图谱\r
\r
**AI 能理解你的代码结构和调用关系，不仅仅是搜索文本。**\r
\r
CodeGraph 将项目的代码结构构建成可查询的知识图谱，让 AI 理解函数之间的调用关系、类型的层次结构和文件的依赖网络。AI 可以准确找到某个函数的所有调用者、分析修改影响范围、查看完整的类型继承链。\r
\r
**多项目独立建图** — 在多项目工作区中，每个项目独立构建知识图谱（主项目用共享库、非主项目用各自存储），跨项目切换不串数据，工具通过 project 参数精确路由到目标项目。\r
\r
---\r
\r
## 对话历史管理\r
\r
**每次对话自动保存，随时回溯，不会丢失。**\r
\r
- 对话自动持久化到本地磁盘，刷新页面不会丢失\r
- 左侧对话列表展示所有历史记录，支持继续之前的话题\r
- 不同工作区的对话自动隔离，各项目互不干扰\r
- 支持向前翻页加载更多历史消息\r
\r
---\r
\r
## BUG 自动检测与修复\r
\r
**AI 主动扫描代码问题并生成修复方案，反复验证直到全部通过。**\r
\r
- 自动运行编译检查和测试，标记所有错误位置\r
- 分析错误根因，生成具体的修复方案\r
- 修复后再次验证，支持多轮迭代\r
- 修复前会展示改动内容，你可以审阅确认\r
\r
---\r
\r
## Skills / MCP / 工具集扩展\r
\r
**通过扩展增强 AI 的能力，让 IDE 更贴合你的工作流。**\r
\r
- **Skills（技能）** — 可复用的工作流程模板，AI 在对应场景中自动加载使用\r
- **MCP（模型上下文协议）** — 标准化的工具扩展协议，可为 AI 添加自定义能力（如查询内部数据库、调用第三方 API）\r
- **工具集（Toolset）** — 按项目需求组合的插件包，动态构建并固化到工作区，可导出/导入/发布市场\r
- **内置市场** — 一键浏览和安装社区贡献的扩展（技能 / MCP / 插件工具集三类）\r
\r
---\r
\r
## 插件化自定义工具\r
\r
**通过 JS / TS / Go / Lua 插件扩展 AI 的工具集，一切皆插件。**\r
\r
PairCode IDE 的工具体系全部插件化——内置功能（文件/搜索/Git/Web/记忆/任务/图谱等 21 组）以插件形态装配，你也可以编写自己的插件扩展能力：\r
\r
- **JS / TS 插件** — 通过 \`cordis_define\` 定义函数形态插件，支持 \`apply(ctx, config)\` 注入服务、timer 定时器、跨 goroutine 执行锁；TS 插件由内置编译器（esbuild 纯 Go）直接转译加载，无需 Node.js\r
- **Go 插件** — 内置插件框架，核心功能组全部以 Go 插件装配，\`cordis_inspect\` 可查看工具归属，\`cordis_stop\` 卸载整组\r
- **Lua 工具** — 支持 Lua 脚本自定义工具，封装常用命令组合与自定义数据处理逻辑\r
- **沙箱防护** — VM 超时防护、schema 校验，插件异常不影响主进程\r
\r
## 工具集生态\r
\r
**按项目需求动态组合工具集，固化到工作区，可导出分享。**\r
\r
- **动态构建** — 描述你的项目需求（如"Go 后端 + 前端调试"），AI 分析项目结构后自动组合所需工具并创建工具集插件\r
- **固化与重建** — 工具集固化到 \`.pair/toolsets/\`，随项目走；显式调用可更新重建\r
- **导出 / 导入 / 市场** — 工具集可导出为 JSON 分享，或发布到市场供他人一键安装（project/user 两种范围）\r
- **LLM 意图分析** — 分析项目目的时由 LLM 参与理解（语言无关，不固化任何语言模板），跨语言项目同样适用\r
\r
---\r
\r
## 项目知识库\r
\r
**把项目架构、模块职责和设计决策沉淀成结构化知识库，AI 跨会话持续了解你的项目。**\r
\r
- **树形分支组织** — 知识按 目标 / 架构 / 实现 / 关键点 / 设计思想 分类，深挖有细节、浏览有全貌\r
- **跨会话记忆** — AI 每次接手项目自动加载知识库导航，无需从零分析项目\r
- **团队共享** — 知识库存入项目 \`.pair/\` 目录，随项目版本控制，团队协作时信息不丢失\r
- **过期检测** — 自动验证知识条目引用的文件/目录是否存在，失效条目提示清理\r
- **AGENTS.md 分层** — 项目说明、环境配置、开发指南分层管理，.agents 路径兼容\r
\r
## 记忆系统\r
\r
**AI 能跨会话记住你的偏好和项目决策。**\r
\r
AI 会记住你的编码偏好、经常使用的模式和做过的决策。下次打开 IDE 时，AI 会自动引用这些记忆，无需重复说明。记忆可搜索、可管理。\r
\r
---\r
\r
## 任务与规划管理\r
\r
**复杂的多步骤开发任务有条不紊地执行。**\r
\r
AI 会自动分解复杂任务为可追踪的子任务步骤，每步的执行状态和结果清晰可见。支持依赖关系管理，任务清单持久化，重启不会丢失。\r
\r
---\r
\r
## 主题与个性化\r
\r
**按照你的喜好定制 IDE 外观。**\r
\r
- **四套预设主题** — 暗色科技风、白色简约风、暖色温暖风、暗夜紫风格\r
- **即时切换** — 切换主题立即生效，无需刷新\r
- **统一字体方案** — 界面字体和代码字体分别配置\r
\r
---\r
\r
## 多模型支持\r
\r
**灵活选择 AI 模型后端。**\r
\r
支持 OpenAI、Claude 等主流 AI 服务商。可为"执行任务"和"制定规划"分别配置不同的模型。所有模型配置在设置面板中集中管理，支持自定义 API 地址。\r
\r
---\r
\r
## 安全设计\r
\r
**你的代码和数据始终在你的控制之下。**\r
\r
- **本地运行** — 所有操作在本地计算机执行，不经过第三方云端\r
- **路径隔离** — 文件操作限定在工作区目录范围内\r
- **审批机制** — 写文件和执行命令等敏感操作需你确认\r
- **本地地址** — IDE 服务仅监听本地回环地址，默认不对外暴露\r
\r
---\r
\r
## 操作界面速览\r
\r
| 区域 | 说明 |\r
|------|------|\r
| **标题栏** | 顶部菜单栏，提供帮助文档、设置等入口 |\r
| **活动栏** | 左侧图标栏，切换文件浏览、搜索、Git 等功能面板 |\r
| **侧栏** | 文件树、搜索面板、Git 面板等工具区域 |\r
| **主编辑区** | 代码编辑区域，支持多标签页切换 |\r
| **对话面板** | 右侧 AI 对话区域，与 AI 交流的核心界面 |\r
| **状态栏** | 底部状态信息，显示文件编码、行号列号 |\r
| **终端面板** | 底部内置终端，执行命令和脚本 |\r
\r
---\r
\r
## 快捷键一览\r
\r
| 快捷键 | 功能 |\r
|--------|------|\r
| Ctrl+S | 保存当前文件 |\r
| Ctrl+B | 切换侧栏显示 |\r
| Ctrl+\\\` | 切换终端面板 |\r
| Ctrl+K | 专注模式（隐藏所有面板） |\r
| Ctrl+Shift+E | 切换到文件浏览器 |\r
| Ctrl+Shift+F | 全局搜索 |\r
| Ctrl+Shift+T | 打开对话面板 |\r
| Ctrl+Shift+C | 切换对话面板 |\r
`,br=`# API 文档\r
\r
PairCode IDE 内置了一套完整的 HTTP REST API + WebSocket 实时通信协议，供 Web 前端与后端核心功能交互，也**支持第三方开发者基于本 API 进行二次开发**。所有 API 地址均以 \`/api\` 开头，返回 JSON 格式数据。\r
\r
> **安全提示**：所有 API 仅监听本地回环地址（127.0.0.1），默认不对外暴露。请勿将服务端口暴露到公网或局域网。\r
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
### 2.1 列出目录\r
\r
\`\`\`\r
GET /api/fs/list?path={目录路径}\r
\`\`\`\r
\r
**参数：**\r
| 参数 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| path | string | 否 | 目录路径，省略时返回工作区根目录 |\r
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
| path | string | 是 | 文件路径 |\r
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
| path | string | 是 | 文件路径（相对于工作区或绝对路径） |\r
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
| path | string | 否 | 搜索目录，省略时使用工作区根目录 |\r
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
**自动忽略的目录：** \`.git\`、\`node_modules\`、\`vendor\`、\`.pair\`、\`__pycache__\`、\`bin\` 等。**仅搜索文本文件扩展名**（\`.go\` \`.js\` \`.ts\` \`.vue\` \`.html\` \`.css\` \`.json\` \`.md\` \`.py\` \`.rs\` \`.java\` 等 50+ 种）。\r
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
**响应：** 返回完整 \`AppSettings\` 对象（字段较多，按需取用）：\r
\r
\`\`\`json\r
{\r
  "provider": "deepseek",\r
  "baseURL": "https://api.deepseek.com/v1/chat/completions",\r
  "apiKey": "sk-xxx",\r
  "planModel": "deepseek-v4-pro",\r
  "executeModel": "deepseek-v4-flash",\r
  "reviewModel": "deepseek-v4-pro",\r
  "temperature": "0.3",\r
  "thinkingMode": "thinking",\r
  "maxTokens": 131072,\r
  "contextMaxTokens": 64000,\r
  "lastProject": "F:/projects/my-app",\r
  "workspaceFolders": ["F:/projects/my-app"],\r
  "recentProjects": ["F:/projects/app1"],\r
  "reviewMode": "auto",\r
  "reviewBlacklist": [],\r
  "reviewWhitelist": [],\r
  "autonomous": false,\r
  "autoCollapse": true,\r
  "maxIterations": 50,\r
  "maxParallelAgents": 3,\r
  "maxReviewRetries": 3,\r
  "autoIterateOnRejection": true,\r
  "requireHumanApprovalForDestructive": true,\r
  "aiReview": false,\r
  "autoCommit": true,\r
  "luaTools": true,\r
  "enableBenchmarking": true,\r
  "systemInstructions": "",\r
  "searxngUrl": "",\r
  "ignoreDirs": [],\r
  "defaultShell": "auto",\r
  "termFontSize": 13,\r
  "termEncoding": "auto",\r
  "theme": "dark",\r
  "fontFamily": "'Cascadia Code', Consolas, monospace",\r
  "editorFontSize": 14,\r
  "tabSize": 2,\r
  "wordWrap": false,\r
  "hideMinimap": false,\r
  "autoConnectMCP": true,\r
  "skillEnabledOverrides": {},\r
  "skillStatusOverrides": {},\r
  "mcpEnabledOverrides": {},\r
  "customProviders": []\r
}\r
\`\`\`\r
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
  "version": "v1.1.2"\r
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
### 7.14 回滚消息\r
\r
\`\`\`\r
POST /api/chat/rollback\r
\`\`\`\r
\r
回滚到指定用户消息之前的状态：恢复该消息关联的所有文件快照，并删除该消息之后的对话历史。\r
\r
**请求体：**\r
\`\`\`json\r
{\r
  "convId": "conv_xxx",\r
  "msgIdx": 3\r
}\r
\`\`\`\r
\r
| 字段 | 类型 | 必填 | 说明 |\r
|------|------|------|------|\r
| convId | string | 是 | 对话 ID |\r
| msgIdx | number | 是 | 用户消息索引（0 基），回滚到此消息之前 |\r
\r
**响应：** \`{"ok": true, "msgIdx": 3}\`\r
\r
---\r
\r
### 7.15 压缩上下文\r
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
| POST | \`/api/chat/rollback\` | 回滚到指定消息前 |\r
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
`,xr=`# AI 工具文档\r
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
- 精确替换文件中的一段文本\r
- 将文件或目录移动到新位置（也可用于重命名）\r
- 删除指定文件\r
- 将文件恢复到修改前的版本\r
- 查看某个文件的所有修改历史版本\r
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
- 截取桌面或指定窗口的屏幕\r
- 截取指定 URL 的网页\r
\r
---\r
\r
## 七、图像分析\r
\r
**AI 可以"看"图片并理解其中的内容。**\r
\r
- 读取图片文件内容（供支持视觉的模型直接理解图像）\r
- 分析图片中的颜色分布、色块区域和基本图形\r
- 从图片中识别文字，支持中英文混合识别\r
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
**在对话中完成 Git 操作，AI 可以帮你管理代码版本。**\r
\r
- 查看工作区的 Git 状态\r
- 查看文件的变更内容\r
- 查看最近的提交历史\r
- 查看某次提交的详情和改动\r
- 逐行查看文件的最后修改人和提交信息\r
- 将文件加入暂存区\r
- 提交已暂存的改动\r
- 列出、创建或删除分支\r
- 切换分支或恢复文件的修改\r
- 将工作区的改动暂存起来，稍后恢复\r
\r
---\r
\r
## 十一、调试器\r
\r
**AI 可以启动调试会话，设置断点并检查程序运行状态。**\r
\r
- 启动 Go 程序的调试会话\r
- 停止当前的调试会话\r
- 在指定文件的指定行设置断点\r
- 从暂停状态继续执行程序\r
- 单步跳过（不进入函数内部）\r
- 单步进入（进入函数调用内部）\r
- 单步跳出（执行到函数返回）\r
- 查看当前线程的调用栈\r
- 查看当前暂停点的变量值\r
- 在暂停状态下求值表达式\r
- 查看当前调试会话的状态\r
\r
---\r
\r
## 十二、项目知识库\r
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
## 十三、记忆系统\r
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
## 十四、BUG 检测与修复\r
\r
**AI 可以自动发现代码中的问题并给出修复方案。**\r
\r
- 分析构建或测试的输出，提取错误位置和上下文\r
- 全量检测项目中的 BUG，自动运行编译和测试检查\r
- 自动检测 BUG 并生成修复方案，支持多次迭代修复\r
\r
---\r
\r
## 十五、任务与规划\r
\r
**AI 可以追踪任务进度和执行计划，确保复杂的多步骤任务有条不紊。**\r
\r
- 创建一个新的子任务并跟踪其状态\r
- 更新任务清单中各项任务的进度状态\r
- 维护和更新执行计划的步骤清单\r
- 任务全部完成后生成提交信息\r
\r
---\r
\r
## 十六、技能与 MCP 管理\r
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
## 十七、市场\r
\r
**浏览和安装来自公共市场的技能和 MCP 扩展。**\r
\r
- 在市场检索可安装的 MCP 服务器或技能\r
- 从市场安装指定的扩展\r
\r
---\r
\r
## 十八、插件管理\r
\r
**管理 JS / TS / Go / Lua 插件——一切皆插件，自定义和扩展 AI 的工具集。**\r
\r
- 定义一个函数形态的 JS/TS 插件（支持 apply(ctx, config) 注入服务、timer 定时器、跨 goroutine 执行锁）\r
- 查看已注册插件的详情（含工具归属：每个工具来自哪个插件，可整体卸载回收）\r
- 对插件执行查询（inspect 内部状态）\r
- 运行插件注册的服务或回调\r
- 列出 / 停止已注册的插件服务\r
- 撤销（undefine）一个已定义的插件\r
- 列出所有已创建的 Lua 自定义工具\r
- 创建一个新的 Lua 自定义工具\r
- 更新现有 Lua 工具的代码或参数\r
- 删除一个 Lua 自定义工具\r
\r
---\r
\r
## 十九、工具集管理\r
\r
**按项目需求动态组合工具集，固化/导出/导入，构建处理本身也插件化。**\r
\r
- 分析项目结构与需求，动态组合所需工具并创建工具集插件（固化到工作区 \`.pair/toolsets/\`）\r
- 列出当前项目可用的工具集\r
- 查看某个工具集的详细内容\r
- 导出工具集为 JSON（可提交 Git / 发布市场）\r
- 从 JSON 或文件导入工具集（project 工作区级 / user 全局级）\r
- 移除不再需要的工具集\r
\r
---\r
\r
## 二十、其他工具\r
\r
**辅助性工具，在特定场景下帮助 AI 更好地与你协作。**\r
\r
- **用户提问** — 当 AI 遇到关键决策点时，向你提问以澄清需求\r
- **任务委派** — 将复杂任务委托给子 AI 独立完成\r
- **资产清单** — 查看和使用已保存的经验胶囊和最佳实践\r
`,Vr=`# 快捷键参考\r
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
`,Nr=`# 常见问题\r
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
AI 可以读写和编辑你的代码文件、在工作区中执行命令、搜索和浏览项目结构、管理 Git 版本控制、启动调试会话、处理图片和办公文档、搜索网络信息，还能截图验证网页效果。基本上，日常开发中你能做的事情，AI 都可以帮你完成。\r
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
所有操作都在你的本地计算机上执行，代码和对话内容不会发送到外部服务器（AI 模型调用除外，你可以选择使用本地模型避免数据外出）。API 服务只监听本地回环地址，默认不对外暴露。文件操作限定在工作区范围内。\r
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
## 能让 AI 访问我的私有 API 吗？\r
\r
可以通过 MCP（模型上下文协议）扩展来实现。在设置中添加自定义 MCP 服务器，AI 就能通过它访问你的私有 API、数据库或内部服务。\r
\r
## 遇到问题怎么办？\r
\r
你可以查看帮助菜单中的文档中心，里面有功能介绍、API 文档、工具文档和快捷键参考等详细资料。如果问题仍然无法解决，可以在对话中向 AI 描述你遇到的问题，它会尽力协助排查。\r
`,wr=`# 快速开始\r
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
`,Tr=`# 更新日志\r
\r
> 所有 PairCode IDE 的重要变更均记录在此文件中。\r
\r
---\r
\r
## 1.2.1 — 2026-08-15\r
\r
### 新增\r
- **按双层循环范式重写 Agent 核心** — 双层循环（turn/step 边界事件、inbox 双队列对齐 next-step/next-turn），消息组装与落盘对齐事件模型（agentloop 编号 ↔ 消息序列推导），系统提示精简为基础工具集模式（\`WB_FULL_TOOLS=1\` 恢复全量工具）\r
- **一切皆插件** — Go 插件框架 + goja JS 动态插件，goja 运行时完全内置（双仓库去除 replace），JS 插件沙箱支持 timer 服务（ctx.timeout/interval）与跨 goroutine 执行锁\r
- **内置 TS 编译器** — esbuild 纯 Go 转译（无 CGO/npm 依赖），TS 插件可直接加载（\`cordis_define\` 支持 js/ts/自动探测），多文件 TS bundle（Build stdin + mock 包）\r
- **工具全插件化** — 21 个内置功能插件（core/fs/git/web/shell/memory/task/project-info/codegraph/debug/vision/office/lsp 等），\`cordis_inspect\` 可见工具归属插件，Unload 可回收整组\r
- **多项目支持** — 工具 project 参数路由（文件类/搜索/Git 全套），codegraph 按项目独立建图与查询（非主项目用各自 JSONStore，天然隔离），memory/project-info 工具显式 project 参数化\r
- **工具集生态** — 模板插件化动态构建（\`toolset_build\` 按项目+需求自动组合工具并固化到工作区）、固化/导出/导入/市场发布（plugin 类型）、LLM 项目意图分析（语言无关，不固化任何语言模板）\r
- **插件生态 P0-P2** — 函数形态 + \`apply(ctx, config)\` + inject 服务 + VM 超时防护 + schema 校验 + 插件管理 UI（host/client 双半）+ client inspect provider\r
- **项目知识库树形化** — 树分支组织（目标/架构/实现/关键点/设计思想）+ AGENTS.md 分层 + .agents 路径兼容\r
- **历史注入对齐 harness** — 删除【历史轮次】前缀标注与 task 时间戳，系统提示补充多轮对话规则\r
- **ask_user 选项内输入** — 支持 single / multi / single-with-input / text 四态交互，修复参数名混淆导致选项不出现的问题\r
- **遗留五件套** — notes 写入同步 + read_image 工具 + run_code 嵌套 + prompt 注册中心 + 知识库过期检查修复\r
\r
### 修复\r
- **移除未完成注入** — TOOL_OUTCOME_UNKNOWN / interrupted 机制移除，无 result 的 tool_call 以空占位维持配对契约，不再向模型注入「中断/未完成」语义\r
- **知识库过期验证误报** — 152 条假警告清零，159 条全绿\r
\r
---\r
\r
## 1.1.8 — 2026-08-11\r
\r
### 新增\r
- **OCR / 图色识别能力** — 图片文字识别（中英文混合）与颜色分布分析，工具配置持久化 + 前端工具面板（2026-08-04）\r
- **对话历史注入膨胀三层压缩** — 固定背景 / 动态日志 / 长时压缩三层方案，控制上下文体积（2026-08-04）\r
- **异常中断后继续未完成对话** — 中断后可直接继续，不丢上下文（2026-08-06）\r
- **后台进程跨轮存活** — run_background 进程不再因每轮重建注册表而丢失（全局单例 bgRegistry）（2026-08-11）\r
- **多项目工具** — Lua 工具 / 工具配置按项目加载 + project 参数路由（2026-08-11）\r
- **背景摘要注入位置修复** — 压缩摘要固定在 task 前注入（前缀稳定），动态日志追加末尾，KV 缓存零损失优化（2026-08-08）\r
\r
### 修复\r
- 关闭 run 内自动压缩，改由外层时机控制（2026-08-05）\r
- 历史消息配对错乱 — 用户消息重复存储导致 tool 配对错乱（lastUser 锚点重组）\r
- 历史消息分段导致多气泡 — 连续 assistant 消息合并显示\r
- 多轮对话 user 后 tool 粘连 + OnBatchPersist 偏移 — 压缩后固定偏移失效，改 lastUser 锚点重组\r
- 归档双 bug — ①Windows 归档静默失效（句柄未关闭 + os.Rename 不能覆盖）→ 显式 Close + 三步法原子替换；②归档摘要孤立 assistant 消息污染 LLM 上下文 → 改 role=user +【历史归档】标注\r
- 多根路径解析 Bug — 优先匹配文件实际存在的根目录\r
\r
---\r
\r
## 1.1.6 — 2026-07-30\r
\r
### 修复\r
- **修复编辑器 Ctrl+F 不生效** — CodeMirror \`search()\` 扩展注册的 \`openSearchPanel\` 与自定义搜索面板 keymap 冲突，使用 \`Prec.high()\` 确保自定义 handler 优先执行，Ctrl+F 正确唤出中文搜索面板\r
- **搜索面板图标全部换为 SVG** — Unicode 字符（▲▼↔×）和文本标签（Aa ·\\* 全词）全部替换为内联 SVG 图标，与界面风格统一\r
- **修复前端 API 路径缺少前导斜杠导致 404** — \`apiURL()\` 拼接时对无前导斜杠的 path 自动补全，\`/apitools/review\` 修正为 \`/api/tools/review\`\r
- **修复 codegraph 增量构建仍全量重写 SQLite** — \`SQLiteStore.Save()\` 在增量模式下调用的 \`RemoveFileEntities\` 清理旧数据，不再 \`DELETE FROM\` 全表\r
\r
### 改进\r
- **编辑器中文搜索面板** — 新建 \`FindPanel.vue\` 组件，替换 CodeMirror 默认英文搜索面板，支持查找/替换/大小写敏感/正则/全词匹配\r
- **codegraph 增量构建测试** — 新增 \`TestSQLiteStoreIncrementalPreserves\` 和 \`TestSQLiteStoreIncrementalBuild\` 验证增量构建与并行完整性\r
\r
---\r
\r
## 1.1.5 — 2026-07-29\r
\r
### 新增\r
- **run_command 后台化** — \`run_command\` 改用后台启动+轮询模式，不再阻塞 Agent 循环，可被上下文取消中断，超时后 LLM 可选择等待或继续\r
- **审核配置改为工作区级** — 审核黑白名单从全局 settings.json 迁移到工作区 .pair/tools.json，不同工作区可独立配置，避免动态工具（Lua）在不同工作区间混淆\r
- **Lua 工具补齐 Tool 结构** — \`buildLuaTool\` 自动设置 UsageGuide/Category/Enabled 字段，与标准工具结构一致\r
- **工具配置弹窗合并** — 「启用开关」和「审核黑白名单」合并为同一「工具配置」弹窗，标签页切换，避免歧义\r
- **自主模式 Follow-up 持续驱动** — Agent 自然终止后，通过 \`OnNextTask\` 回调自动注入 follow-up 消息，无需手动触发「继续」\r
- **流式更新机制** — Registry 新增 \`OnToolUpdate\` 回调，工具执行中间结果实时推送给前端\r
- **工具 UsageGuide 全覆盖** — 全部 ~140 个工具添加 \`UsageGuide\` 使用指导，明确何时用、为何优于 \`run_command\`、常见误区\r
- **启动日志详细化** — 启动时输出版本号、Go 版本、平台架构、工作目录、各工作区文件夹路径\r
\r
### 改进\r
- **工具体系升级**\r
  - \`Tool\` 结构体新增 \`UsageGuide\`、\`Category\`、\`Enabled\` 字段\r
  - \`Registry\` 新增 \`EnabledDefinitions()\` 按状态过滤工具定义\r
  - 新增 \`AllToolMeta()\` API 供前端展示工具开关列表\r
  - 工具使用指南文本动态注入系统提示，引导 LLM 优先使用专用工具\r
- **窗口管理** — \`run_command\` / \`run_background\` 均设置 \`HideWindow=true\`，不再弹出 cmd 窗口\r
- **信号监听移除** — main 函数移除信号监听，进程不会因子进程结束而自动退出\r
\r
### 修复\r
- **debug_start 启动修复** — 拆解 \`dlv dap\` 启动流程，分别发送 Initialize 和 Launch 请求，兼容 dlv 最新版本\r
\r
---\r
\r
## 1.1.2 — 2026-07-21\r
\r
### 新增\r
- **附件标签化** — 消息中的文件/代码/图片附件不再嵌入正文，改为独立药丸形标签显示在用户消息文字下方，视觉更清爽\r
- **粘贴长文本自动转临时附件** — 输入框粘贴超过 2000 字符的文本时，自动写入 \`_temp/\` 目录并作为附件挂载，避免大段代码/日志撑爆输入区\r
\r
### 改进\r
- \`addToChat\` 瘦身：文件添加到对话不再预读文件内容（40KB 截断已无意义），仅传递路径引用\r
- 目录引用新增 \`type:dir\` 支持，提示 agent 使用 \`list_files\` 查看\r
- 选中代码添加对话现在保留代码内容尾注供 agent 直接参考（截断 3000 字）\r
\r
### 修复\r
- 文件树 Shift/Ctrl 多选逻辑修复：范围选择改为基于同级节点列表，清除后重新选中\r
\r
---\r
\r
## 1.1.1 — 2026-07-21\r
\r
### 新增\r
- **审核配置界面重设计** — 从纯文本输入改为工具卡片式交互：所有工具按类别分组（文件操作、命令执行、Git、网络、截图、图像、二进制、办公文档、CodeGraph、调试器、知识库、记忆、LSP、BUG检测、任务管理、扩展市场等），每个工具显示中文名称，点击切换三态（默认 → 黑名单 → 白名单），支持搜索过滤，配置更直观高效\r
\r
### 修复\r
- **修复新对话空状态提示位置偏移** — "开始新的对话，发送消息即可与 AI 助手对话"提示及图标从左下角偏移修正为居中显示\r
\r
### 改进\r
- 版本号统一升级至 1.1.1（前端 package.json、后端 main.go、打包配置）\r
\r
---\r
\r
## 1.1.0 — 2026-07-20\r
\r
### 新增\r
- **自主模式原生终止** — 去掉 \`finish_task\` 强制结束机制，Agent 自然输出后直接结束循环，交互更流畅\r
- **Agent 性能优化（P0-P3 五轮）** — eventRing 环形缓冲器减少内存分配、进度可视化（阶段指示器+工具调用计数+耗时）、工具描述精简减少 Token 消耗、并行工具执行机制、预压缩上下文避免截断\r
- **会话连贯性增强** — 新对话开始时自动注入 Git 变更感知、代码图谱统计、工作区结构概览，Agent 无需从零分析项目\r
\r
### 改进\r
- **ChatView 重构** — 消息渲染管线全面优化，新增交互超时保护、审核驳回追踪、折叠/展开状态持久化\r
- **审核配置 UI 优化** — 弹窗改为向上弹出（bottom:100%），防止被视口底部裁切\r
- **编辑工具 v2 升级** — 更精确的符号级定位，减少行号偏移问题\r
- **kill_process 增强** — 改为杀进程树，彻底清理子进程\r
- **自主模式架构重构** — ephemeralMsgs 隔离内层消息，长时压缩精准保留推理上下文\r
\r
### 修复\r
- 修复 \`planExpanded\` / \`tasksExpanded\` / \`currentPhase\` 重复声明导致的运行时崩溃\r
- 修复 \`currentTasks\` 未声明导致前端 \`undefined.length\` 崩溃\r
- 修复自然终止代码缩进丢失导致逻辑在循环外不执行\r
\r
---\r
\r
## 1.0.20 — 2026-07-18\r
\r
### 修复\r
- **修复消息排序** — \`_idx\` 统一取 \`max(existing)+1\`，解决历史消息加载后序号错乱\r
- **修复用户反馈消息合并** — 用户反馈正确合并到 agent 输出气泡中，不再产生额外用户消息气泡\r
- **修复消息发送双占位竞态** — \`switchConv\` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，避免两个 assistant 气泡\r
- **修复 WS 连接与历史加载竞态** — \`processStatus\` 事件正确处理连接状态转换\r
\r
### 改进\r
- 审核配置弹窗改为向上弹出（\`bottom:100%\`），防止被视口底部裁切\r
- 移除压缩按钮，简化 UI\r
\r
---\r
\r
## 1.0.19 — 2026-07-17\r
\r
### 修复\r
- **修复 Web 端文件树不显示** — \`FileExplorer.vue\` 的 \`<script setup>\` 编译后 JS 中存在变量暂时性死区（TDZ），导致 \`setup()\` 抛出 \`Cannot access 'd' before initialization\`，文件树组件挂载失败。重建前端并重新编译 \`companion.exe\` 嵌入新版 dist 后修复\r
- **修复后端 dist 嵌入路径不一致** — \`cmd/companion/main.go\` 通过 \`//go:embed web-ui/dist\` 引用 companion 目录下的副本，但此前构建脚本将 dist 输出到 \`cmd/desktop/web-ui/dist/\`，两者不同步导致嵌入的仍是旧版 JS。统一构建流程后将新版 dist 正确复制到 \`cmd/companion/web-ui/dist/\`\r
\r
### 改进\r
- 统一更新版本号至 1.0.19（后端 main.go、两个前端的 package.json）\r
\r
---\r
\r
## 1.0.8 — 2026-07-17\r
\r
### 新增\r
- **多项目工作区支持** — 系统提示自动遍历所有工作区根目录，读取各自 \`.pair/project.md\` 环境配置注入给 AI，跨项目协作时准确感知每个项目的编译方式、CGO 开关等信息\r
- **CodeGraph 多项目全量建图** — \`codegraph_build\` 支持对所有工作区项目建图并合并到同一个知识图谱（\`rebuild=true\`），跨项目符号搜索成为可能\r
- **阻塞命令自动拦截** — 新增 \`isBlockingCommand\` 检测，自动拦截 dev server、watch 模式、\`go run .\`、\`npm run dev\` 等长期进程命令，提示改用 \`run_background\`，避免阻塞 AI 循环\r
\r
### 改进\r
- **审核放行逻辑优化** — \`run_command\` 阻塞命令不再自动放行，强制走 LLM 审核；\`run_background\` 保持安全命令自动放行\r
- **工具描述优化** — \`run_command\` 描述明确禁止长期进程并列出典型误用场景；\`run_background\` 强调作为长期进程首选工具\r
- **系统提示增强** — 「错误恢复」和「防止卡死」两处加入阻塞/后台区分铁律，降低误用 \`run_command\` 概率\r
\r
---\r
\r
## 1.0.7 — 2026-07-17\r
\r
### 修复\r
- **修复刷新页面后 ask_user 提交造成额外气泡** — 页面刷新后 \`switchConv\` 复用历史消息中最后一条 assistant 消息接收后续 WS 事件，不再另建新占位，避免两个 assistant 气泡\r
\r
### 改进\r
- 统一更新版本号至 1.0.7（前端 package.json、后端 main.go、打包脚本）\r
\r
---\r
\r
## 1.0.6 — 2026-07-17\r
\r
### 修复\r
- **修复消息持久化比较口径不一致** — \`PersistNewMessages\` 中 \`persistedCount\` 使用 \`countJSONLLines\`（统计文件总行数含 System），与 \`histNonSystemCount\`（统计非 System 消息数）口径不同，导致含 tool_call 的 assistant 消息在工具执行前被误判为"已落盘"而跳过写入。阻塞工具（如 ask_user）的前端始终无响应。改用 \`readJSONL\` 精确统计非 System 消息数\r
- **修复对话/任务/执行计划 API 空实现** — \`GET /api/conversations/{id}\` 缺 agent 运行状态，\`GET /api/tasks\` 和 \`GET /api/taskplan\` 原返回对话列表（完全错误的 stub），改为返回真实数据\r
\r
---\r
\r
## 1.0.5 — 2026-07-17\r
\r
### 改进\r
- **消息持久化重构** — \`PersistNewMessages\` 改为全量覆盖写 JSONL，消除 diff 计算的竞态问题；\`MessageStore\` 新增 \`ReplaceHistory\` 支持历史压缩；\`MergeLastAssistantRun\` 移除，各轮次独立存储以保留 reasoning 完整时序\r
\r
### 修复\r
- **修复 send on closed channel panic** — 移除三处 \`go func\` 在无监听者时向 channel 发送导致的崩溃\r
- **修复 PersistNewMessages 上下文压缩后新消息丢失** — 全量替换模式确保压缩后的摘要消息不被覆盖\r
- **修复自动提交仅提交主工作区** — \`doAutoCommit\` 遍历所有工作区执行 git add + commit\r
- **修复 idx 空洞导致消息跳过持久化** — \`PersistNewMessages\` 内部不再跳过 System/User 消息，确保序号连续\r
\r
---\r
\r
## 1.0.4 — 2026-07-17\r
\r
### 新增\r
- **技能状态三级配置** — 技能可设为「关闭 / 按需加载 / 始终激活」三种模式，灵活控制 AI 行为\r
- **市场安装范围选择** — 安装 MCP 服务器或技能时，支持选择 user（全局）或 project（项目级）范围\r
\r
### 改进\r
- **对话历史持久化增强** — 页面刷新后对话完整恢复，不再因浏览器关闭丢失上下文；后端全面接管消息状态管理，前端不再依赖本地缓存\r
- **消息展示优化** — 连续同一角色的消息自动合并显示（如多个 assistant 回复合并为一条），阅读更流畅\r
- **停止信号可靠性提升** — Agent 异常结束或用户主动停止时，前端能可靠收到停止信号并更新 UI 状态\r
\r
### 修复\r
- 修复切换对话时 loading 状态卡死的问题（switchConv 提前放行占位消息）\r
- 修复消息历史顺序错乱和思考链（reasoning_content）丢失的严重问题\r
- 修复 MergeConsecutiveAssistants 跳过 RoleTool 消息导致工具调用结果不完整的问题\r
\r
---\r
\r
## 1.0.3 — 2026-07-17\r
\r
### 改进\r
- **子进程窗口管理** — 所有后台子进程（Git 操作、BUG 检测编译/测试、Lua 工具执行、桥接命令）统一隐藏控制台窗口，避免黑框闪烁\r
- **会话持久化** — OnBatchPersist 回调从"每 5 轮"改为"每轮迭代"写盘，降低异常丢失风险\r
- **代码搜索提示修复** — codegraph 搜索无结果时正确显示查询内容而非空占位符\r
\r
### 修复\r
- **PersistNewMessages idx 空洞 bug** — 修复因跳过 System/User 角色消息导致消息序号不连续、后续消息无法正确持久化的严重问题（db_store.go + db_adapter.go）\r
\r
---\r
\r
## 1.0.2 — 2026-07-16\r
\r
### 改进\r
- **文档同步** — features.md 同步到最新版本，移除冗余的"版本信息与更新日志"章节\r
\r
---\r
\r
## 1.0.1 — 2026-07-11\r
\r
### 新增\r
- **更新日志页面** — 帮助文档中新增更新日志页面，版本历史一目了然\r
- **WebSocket 协议文档** — API 文档补充完整 WebSocket 事件类型与负载定义\r
- **系统版本报告** — \`/api/system/info\` 现在返回 \`version\` 字段，前端"关于"面板同步显示\r
\r
### 改进\r
- **API 文档全面重写** — 每个接口增加请求体 JSON Schema、响应示例和错误码说明，便于二次开发\r
- **帮助文档重构** — 文档归入"文档中心"分类，导航更清晰\r
\r
---\r
\r
## 1.0.0 — 2026-07-01\r
\r
### 新增\r
- **AI 对话编程** — 用自然语言驱动 AI 读写文件、执行命令、管理 Git\r
- **自主 Agent 模式** — AI 自动分析项目、制定计划并执行多步骤任务\r
- **代码编辑器** — 内置多标签页编辑器，支持语法高亮、代码折叠、十六进制查看\r
- **文件管理** — 工作区目录树浏览、文件搜索、批量操作\r
- **Git 版本控制** — 对话驱动的 Git 操作（状态查看、暂存、提交、分支管理）\r
- **内置终端** — 浏览器中的终端面板，支持 AI 自动执行命令\r
- **对话历史管理** — 自动保存、回溯与继续历史对话\r
- **BUG 自动检测修复** — AI 扫描编译/测试问题并自动修复\r
- **Skills / MCP 扩展** — 可复用的工作流模板和模型上下文协议扩展\r
- **记忆系统** — AI 跨会话记住用户偏好和历史决策\r
- **任务与规划管理** — 复杂任务分解为可追踪的子步骤\r
- **Lua 自定义工具** — 通过 Lua 脚本创建自定义 AI 工具\r
- **代码知识图谱** — 函数调用关系、类型层次、影响范围分析\r
- **多模型支持** — 灵活切换 AI 模型后端（OpenAI / Claude 等）\r
- **主题系统** — 四套预设主题（暗色、白色、暖色、暗夜紫）\r
- **调试器** — 支持 Go 程序的断点、单步和变量查看\r
- **网页验证工具** — 自动打开 URL、截图、分析页面效果\r
- **办公文档处理** — 读取 Word / Excel / PDF 文件，支持 OCR\r
\r
### 技术架构\r
- 后端使用 Go 语言，前端使用 Vue 3 + CodeMirror\r
- WebSocket 实时推送 AI 事件流\r
- 内嵌前端资源（go:embed），单二进制分发\r
- 纯本地运行，所有 API 仅监听本地回环地址\r
`;function ce(){return{async:!1,breaks:!1,extensions:null,gfm:!0,hooks:null,pedantic:!1,renderer:null,silent:!1,tokenizer:null,walkTokens:null}}var K=ce();function we(r){K=r}var Q={exec:()=>null};function Y(r){let t=[];return n=>{let l=Math.max(0,Math.min(3,n-1)),o=t[l];return o||(o=r(l),t[l]=o),o}}function S(r,t=""){let n=typeof r=="string"?r:r.source,l={replace:(o,a)=>{let s=typeof a=="string"?a:a.source;return s=s.replace(R.caret,"$1"),n=n.replace(o,s),l},getRegex:()=>new RegExp(n,t)};return l}var Br=((r="")=>{try{return!!new RegExp("(?<=1)(?<!1)"+r)}catch{return!1}})(),R={codeRemoveIndent:/^(?: {1,4}| {0,3}\t)/gm,outputLinkReplace:/\\([\[\]])/g,indentCodeCompensation:/^(\s+)(?:```)/,beginningSpace:/^\s+/,endingHash:/#$/,startingSpaceChar:/^ /,endingSpaceChar:/ $/,nonSpaceChar:/[^ ]/,newLineCharGlobal:/\n/g,tabCharGlobal:/\t/g,multipleSpaceGlobal:/\s+/g,blankLine:/^[ \t]*$/,doubleBlankLine:/\n[ \t]*\n[ \t]*$/,blockquoteStart:/^ {0,3}>/,blockquoteSetextReplace:/\n {0,3}((?:=+|-+) *)(?=\n|$)/g,blockquoteSetextReplace2:/^ {0,3}>[ \t]?/gm,listReplaceNesting:/^ {1,4}(?=( {4})*[^ ])/g,listIsTask:/^\[[ xX]\] +\S/,listReplaceTask:/^\[[ xX]\] +/,listTaskCheckbox:/\[[ xX]\]/,anyLine:/\n.*\n/,hrefBrackets:/^<(.*)>$/,tableDelimiter:/[:|]/,tableAlignChars:/^\||\| *$/g,tableRowBlankLine:/\n[ \t]*$/,tableAlignRight:/^ *-+: *$/,tableAlignCenter:/^ *:-+: *$/,tableAlignLeft:/^ *:-+ *$/,startATag:/^<a /i,endATag:/^<\/a>/i,startPreScriptTag:/^<(pre|code|kbd|script)(\s|>)/i,endPreScriptTag:/^<\/(pre|code|kbd|script)(\s|>)/i,startAngleBracket:/^</,endAngleBracket:/>$/,pedanticHrefTitle:/^([^'"]*[^\s])\s+(['"])(.*)\2/,unicodeAlphaNumeric:/[\p{L}\p{N}]/u,escapeTest:/[&<>"']/,escapeReplace:/[&<>"']/g,escapeTestNoEncode:/[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/,escapeReplaceNoEncode:/[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/g,caret:/(^|[^\[])\^/g,percentDecode:/%25/g,findPipe:/\|/g,splitPipe:/ \|/,slashPipe:/\\\|/g,carriageReturn:/\r\n|\r/g,spaceLine:/^ +$/gm,notSpaceStart:/^\S*/,endingNewline:/\n$/,listItemRegex:r=>new RegExp(`^( {0,3}${r})((?:[	 ][^\\n]*)?(?:\\n|$))`),nextBulletRegex:Y(r=>new RegExp(`^ {0,${r}}(?:[*+-]|\\d{1,9}[.)])((?:[ 	][^\\n]*)?(?:\\n|$))`)),hrRegex:Y(r=>new RegExp(`^ {0,${r}}((?:- *){3,}|(?:_ *){3,}|(?:\\* *){3,})(?:\\n+|$)`)),fencesBeginRegex:Y(r=>new RegExp(`^ {0,${r}}(?:\`\`\`|~~~)`)),headingBeginRegex:Y(r=>new RegExp(`^ {0,${r}}#`)),htmlBeginRegex:Y(r=>new RegExp(`^ {0,${r}}<(?:[a-z].*>|!--)`,"i")),blockquoteBeginRegex:Y(r=>new RegExp(`^ {0,${r}}>`))},Sr=/^(?:[ \t]*(?:\n|$))+/,Cr=/^((?: {4}| {0,3}\t)[^\n]+(?:\n(?:[ \t]*(?:\n|$))*)?)+/,Ir=/^ {0,3}(`{3,}(?=[^`\n]*(?:\n|$))|~{3,})([^\n]*)(?:\n|$)(?:|([\s\S]*?)(?:\n|$))(?: {0,3}\1[~`]* *(?=\n|$)|$)/,v=/^ {0,3}((?:-[\t ]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})(?:\n+|$)/,Pr=/^ {0,3}(#{1,6})(?=\s|$)(.*)(?:\n+|$)/,de=/ {0,3}(?:[*+-]|\d{1,9}[.)])/,Te=/^(?!bull |blockCode|fences|blockquote|heading|html|table)((?:.|\n(?!\s*?\n|bull |blockCode|fences|blockquote|heading|html|table))+?)\n {0,3}(=+|-+) *(?:\n+|$)/,Be=S(Te).replace(/bull/g,de).replace(/blockCode/g,/(?: {4}| {0,3}\t)/).replace(/fences/g,/ {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g,/ {0,3}>/).replace(/heading/g,/ {0,3}#{1,6}/).replace(/html/g,/ {0,3}<[^\n>]+>\n/).replace(/\|table/g,"").getRegex(),Ar=S(Te).replace(/bull/g,de).replace(/blockCode/g,/(?: {4}| {0,3}\t)/).replace(/fences/g,/ {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g,/ {0,3}>/).replace(/heading/g,/ {0,3}#{1,6}/).replace(/html/g,/ {0,3}<[^\n>]+>\n/).replace(/table/g,/ {0,3}\|?(?:[:\- ]*\|)+[\:\- ]*\n/).getRegex(),pe=/^([^\n]+(?:\n(?!hr|heading|lheading|blockquote|fences|list|html|table| +\n)[^\n]+)*)/,$r=/^[^\n]+/,me=/(?!\s*\])(?:\\[\s\S]|[^\[\]\\])+/,Mr=S(/^ {0,3}\[(label)\]: *(?:\n[ \t]*)?([^<\s][^\s]*|<.*?>)(?:(?: +(?:\n[ \t]*)?| *\n[ \t]*)(title))? *(?:\n+|$)/).replace("label",me).replace("title",/(?:"(?:\\"?|[^"\\])*"|'[^'\n]*(?:\n[^'\n]+)*\n?'|\([^()]*\))/).getRegex(),Dr=S(/^(bull)([ \t][^\n]*?)?(?:\n|$)/).replace(/bull/g,de).getRegex(),te="address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul",ge=/<!--(?:-?>|[\s\S]*?(?:-->|$))/,Rr=S("^ {0,3}(?:<(script|pre|style|textarea)[\\s>][\\s\\S]*?(?:</\\1>[^\\n]*\\n+|$)|comment[^\\n]*(\\n+|$)|<\\?[\\s\\S]*?(?:\\?>\\n*|$)|<![A-Z][\\s\\S]*?(?:>\\n*|$)|<!\\[CDATA\\[[\\s\\S]*?(?:\\]\\]>\\n*|$)|</?(tag)(?: +|\\n|/?>)[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|<(?!script|pre|style|textarea)([a-z][\\w-]*)(?:attribute)*? */?>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|</(?!script|pre|style|textarea)[a-z][\\w-]*\\s*>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$))","i").replace("comment",ge).replace("tag",te).replace("attribute",/ +[a-zA-Z:_][\w.:-]*(?: *= *"[^"\n]*"| *= *'[^'\n]*'| *= *[^\s"'=<>`]+)?/).getRegex(),Se=S(pe).replace("hr",v).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("|lheading","").replace("|table","").replace("blockquote"," {0,3}>").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",te).getRegex(),Gr=S(/^( {0,3}> ?(paragraph|[^\n]*)(?:\n|$))+/).replace("paragraph",Se).getRegex(),ke={blockquote:Gr,code:Cr,def:Mr,fences:Ir,heading:Pr,hr:v,html:Rr,lheading:Be,list:Dr,newline:Sr,paragraph:Se,table:Q,text:$r},Ce=S("^ *([^\\n ].*)\\n {0,3}((?:\\| *)?:?-+:? *(?:\\| *:?-+:? *)*(?:\\| *)?)(?:\\n((?:(?! *\\n|hr|heading|blockquote|code|fences|list|html).*(?:\\n|$))*)\\n*|$)").replace("hr",v).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("blockquote"," {0,3}>").replace("code","(?: {4}| {0,3}	)[^\\n]").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",te).getRegex(),Lr={...ke,lheading:Ar,table:Ce,paragraph:S(pe).replace("hr",v).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("|lheading","").replace("table",Ce).replace("blockquote"," {0,3}>").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",te).getRegex()},Fr={...ke,html:S(`^ *(?:comment *(?:\\n|\\s*$)|<(tag)[\\s\\S]+?</\\1> *(?:\\n{2,}|\\s*$)|<tag(?:"[^"]*"|'[^']*'|\\s[^'"/>\\s]*)*?/?> *(?:\\n{2,}|\\s*$))`).replace("comment",ge).replace(/tag/g,"(?!(?:a|em|strong|small|s|cite|q|dfn|abbr|data|time|code|var|samp|kbd|sub|sup|i|b|u|mark|ruby|rt|rp|bdi|bdo|span|br|wbr|ins|del|img)\\b)\\w+(?!:|[^\\w\\s@]*@)\\b").getRegex(),def:/^ *\[([^\]]+)\]: *<?([^\s>]+)>?(?: +(["(][^\n]+[")]))? *(?:\n+|$)/,heading:/^(#{1,6})(.*)(?:\n+|$)/,fences:Q,lheading:/^(.+?)\n {0,3}(=+|-+) *(?:\n+|$)/,paragraph:S(pe).replace("hr",v).replace("heading",` *#{1,6} *[^
]`).replace("lheading",Be).replace("|table","").replace("blockquote"," {0,3}>").replace("|fences","").replace("|list","").replace("|html","").replace("|tag","").getRegex()},Or=/^\\([!"#$%&'()*+,\-./:;<=>?@\[\]\\^_`{|}~])/,zr=/^(`+)([^`]|[^`][\s\S]*?[^`])\1(?!`)/,Ie=/^( {2,}|\\)\n(?!\s*$)/,Ur=/^(`+|[^`])(?:(?= {2,}\n)|[\s\S]*?(?:(?=[\\<!\[`*_]|\b_|$)|[^ ](?= {2,}\n)))/,_=/[\p{P}\p{S}]/u,re=/[\s\p{P}\p{S}]/u,he=/[^\s\p{P}\p{S}]/u,Hr=S(/^((?![*_])punctSpace)/,"u").replace(/punctSpace/g,re).getRegex(),Pe=/(?!~)[\p{P}\p{S}]/u,jr=/(?!~)[\s\p{P}\p{S}]/u,qr=/(?:[^\s\p{P}\p{S}]|~)/u,Wr=S(/link|precode-code|html/,"g").replace("link",/\[(?:[^\[\]`]|(?<a>`+)[^`]+\k<a>(?!`))*?\]\((?:\\[\s\S]|[^\\\(\)]|\((?:\\[\s\S]|[^\\\(\)])*\))*\)/).replace("precode-",Br?"(?<!`)()":"(^^|[^`])").replace("code",/(?<b>`+)[^`]+\k<b>(?!`)/).replace("html",/<(?! )[^<>]*?>/).getRegex(),Ae=/^(?:\*+(?:((?!\*)punct)|([^\s*]))?)|^_+(?:((?!_)punct)|([^\s_]))?/,Jr=S(Ae,"u").replace(/punct/g,_).getRegex(),Zr=S(Ae,"u").replace(/punct/g,Pe).getRegex(),$e="^[^_*]*?__[^_*]*?\\*[^_*]*?(?=__)|[^*]+(?=[^*])|(?!\\*)punct(\\*+)(?=[\\s]|$)|notPunctSpace(\\*+)(?!\\*)(?=punctSpace|$)|(?!\\*)punctSpace(\\*+)(?=notPunctSpace)|[\\s](\\*+)(?!\\*)(?=punct)|(?!\\*)punct(\\*+)(?!\\*)(?=punct)|notPunctSpace(\\*+)(?=notPunctSpace)",Kr=S($e,"gu").replace(/notPunctSpace/g,he).replace(/punctSpace/g,re).replace(/punct/g,_).getRegex(),Qr=S($e,"gu").replace(/notPunctSpace/g,qr).replace(/punctSpace/g,jr).replace(/punct/g,Pe).getRegex(),Xr=S("^[^_*]*?\\*\\*[^_*]*?_[^_*]*?(?=\\*\\*)|[^_]+(?=[^_])|(?!_)punct(_+)(?=[\\s]|$)|notPunctSpace(_+)(?!_)(?=punctSpace|$)|(?!_)punctSpace(_+)(?=notPunctSpace)|[\\s](_+)(?!_)(?=punct)|(?!_)punct(_+)(?!_)(?=punct)","gu").replace(/notPunctSpace/g,he).replace(/punctSpace/g,re).replace(/punct/g,_).getRegex(),Yr=S(/^~~?(?:((?!~)punct)|[^\s~])/,"u").replace(/punct/g,_).getRegex(),_r="^[^~]+(?=[^~])|(?!~)punct(~~?)(?=[\\s]|$)|notPunctSpace(~~?)(?!~)(?=punctSpace|$)|(?!~)punctSpace(~~?)(?=notPunctSpace)|[\\s](~~?)(?!~)(?=punct)|(?!~)punct(~~?)(?!~)(?=punct)|notPunctSpace(~~?)(?=notPunctSpace)",vr=S(_r,"gu").replace(/notPunctSpace/g,he).replace(/punctSpace/g,re).replace(/punct/g,_).getRegex(),el=S(/\\(punct)/,"gu").replace(/punct/g,_).getRegex(),nl=S(/^<(scheme:[^\s\x00-\x1f<>]*|email)>/).replace("scheme",/[a-zA-Z][a-zA-Z0-9+.-]{1,31}/).replace("email",/[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+(@)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+(?![-_])/).getRegex(),tl=S(ge).replace("(?:-->|$)","-->").getRegex(),rl=S("^comment|^</[a-zA-Z][\\w:-]*\\s*>|^<[a-zA-Z][\\w-]*(?:attribute)*?\\s*/?>|^<\\?[\\s\\S]*?\\?>|^<![a-zA-Z]+\\s[\\s\\S]*?>|^<!\\[CDATA\\[[\\s\\S]*?\\]\\]>").replace("comment",tl).replace("attribute",/\s+[a-zA-Z:_][\w.:-]*(?:\s*=\s*"[^"]*"|\s*=\s*'[^']*'|\s*=\s*[^\s"'=<>`]+)?/).getRegex(),le=/(?:\[(?:\\[\s\S]|[^\[\]\\])*\]|\\[\s\S]|`+(?!`)[^`]*?`+(?!`)|``+(?=\])|[^\[\]\\`])*?/,ll=S(/^!?\[(label)\]\(\s*(href)(?:(?:[ \t]+(?:\n[ \t]*)?|\n[ \t]*)(title))?\s*\)/).replace("label",le).replace("href",/<(?:\\.|[^\n<>\\])+>|[^ \t\n\x00-\x1f]*/).replace("title",/"(?:\\"?|[^"\\])*"|'(?:\\'?|[^'\\])*'|\((?:\\\)?|[^)\\])*\)/).getRegex(),Me=S(/^!?\[(label)\]\[(ref)\]/).replace("label",le).replace("ref",me).getRegex(),De=S(/^!?\[(ref)\](?:\[\])?/).replace("ref",me).getRegex(),ol=S("reflink|nolink(?!\\()","g").replace("reflink",Me).replace("nolink",De).getRegex(),Re=/[hH][tT][tT][pP][sS]?|[fF][tT][pP]/,fe={_backpedal:Q,anyPunctuation:el,autolink:nl,blockSkip:Wr,br:Ie,code:zr,del:Q,delLDelim:Q,delRDelim:Q,emStrongLDelim:Jr,emStrongRDelimAst:Kr,emStrongRDelimUnd:Xr,escape:Or,link:ll,nolink:De,punctuation:Hr,reflink:Me,reflinkSearch:ol,tag:rl,text:Ur,url:Q},al={...fe,link:S(/^!?\[(label)\]\((.*?)\)/).replace("label",le).getRegex(),reflink:S(/^!?\[(label)\]\s*\[([^\]]*)\]/).replace("label",le).getRegex()},ye={...fe,emStrongRDelimAst:Qr,emStrongLDelim:Zr,delLDelim:Yr,delRDelim:vr,url:S(/^((?:protocol):\/\/|www\.)(?:[a-zA-Z0-9\-]+\.?)+[^\s<]*|^email/).replace("protocol",Re).replace("email",/[A-Za-z0-9._+-]+(@)[a-zA-Z0-9-_]+(?:\.[a-zA-Z0-9-_]*[a-zA-Z0-9])+(?![-_])/).getRegex(),_backpedal:/(?:[^?!.,:;*_'"~()&]+|\([^)]*\)|&(?![a-zA-Z0-9]+;$)|[?!.,:;*_'"~)]+(?!$))+/,del:/^(~~?)(?=[^\s~])((?:\\[\s\S]|[^\\])*?(?:\\[\s\S]|[^\s~\\]))\1(?=[^~]|$)/,text:S(/^([`~]+|[^`~])(?:(?= {2,}\n)|(?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)|[\s\S]*?(?:(?=[\\<!\[`*~_]|\b_|protocol:\/\/|www\.|$)|[^ ](?= {2,}\n)|[^a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-](?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)))/).replace("protocol",Re).getRegex()},sl={...ye,br:S(Ie).replace("{2,}","*").getRegex(),text:S(ye.text).replace("\\b_","\\b_| {2,}\\n").replace(/\{2,\}/g,"*").getRegex()},oe={normal:ke,gfm:Lr,pedantic:Fr},ee={normal:fe,gfm:ye,breaks:sl,pedantic:al},il={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"},Ge=r=>il[r];function q(r,t){if(t){if(R.escapeTest.test(r))return r.replace(R.escapeReplace,Ge)}else if(R.escapeTestNoEncode.test(r))return r.replace(R.escapeReplaceNoEncode,Ge);return r}function Le(r){try{r=encodeURI(r).replace(R.percentDecode,"%")}catch{return null}return r}function Fe(r,t){var a;let n=r.replace(R.findPipe,(s,p,i)=>{let y=!1,k=p;for(;--k>=0&&i[k]==="\\";)y=!y;return y?"|":" |"}),l=n.split(R.splitPipe),o=0;if(l[0].trim()||l.shift(),l.length>0&&!((a=l.at(-1))!=null&&a.trim())&&l.pop(),t)if(l.length>t)l.splice(t);else for(;l.length<t;)l.push("");for(;o<l.length;o++)l[o]=l[o].trim().replace(R.slashPipe,"|");return l}function Z(r,t,n){let l=r.length;if(l===0)return"";let o=0;for(;o<l&&r.charAt(l-o-1)===t;)o++;return r.slice(0,l-o)}function Oe(r){let t=r.split(`
`),n=t.length-1;for(;n>=0&&R.blankLine.test(t[n]);)n--;return t.length-n<=2?r:t.slice(0,n+1).join(`
`)}function cl(r,t){if(r.indexOf(t[1])===-1)return-1;let n=0;for(let l=0;l<r.length;l++)if(r[l]==="\\")l++;else if(r[l]===t[0])n++;else if(r[l]===t[1]&&(n--,n<0))return l;return n>0?-2:-1}function dl(r,t=0){let n=t,l="";for(let o of r)if(o==="	"){let a=4-n%4;l+=" ".repeat(a),n+=a}else l+=o,n++;return l}function ze(r,t,n,l,o){let a=t.href,s=t.title||null,p=r[1].replace(o.other.outputLinkReplace,"$1");l.state.inLink=!0;let i={type:r[0].charAt(0)==="!"?"image":"link",raw:n,href:a,title:s,text:p,tokens:l.inlineTokens(p)};return l.state.inLink=!1,i}function pl(r,t,n){let l=r.match(n.other.indentCodeCompensation);if(l===null)return t;let o=l[1];return t.split(`
`).map(a=>{let s=a.match(n.other.beginningSpace);if(s===null)return a;let[p]=s;return p.length>=o.length?a.slice(o.length):a}).join(`
`)}var ae=class{constructor(r){I(this,"options");I(this,"rules");I(this,"lexer");this.options=r||K}space(r){let t=this.rules.block.newline.exec(r);if(t&&t[0].length>0)return{type:"space",raw:t[0]}}code(r){let t=this.rules.block.code.exec(r);if(t){let n=this.options.pedantic?t[0]:Oe(t[0]),l=n.replace(this.rules.other.codeRemoveIndent,"");return{type:"code",raw:n,codeBlockStyle:"indented",text:l}}}fences(r){let t=this.rules.block.fences.exec(r);if(t){let n=t[0],l=pl(n,t[3]||"",this.rules);return{type:"code",raw:n,lang:t[2]?t[2].trim().replace(this.rules.inline.anyPunctuation,"$1"):t[2],text:l}}}heading(r){let t=this.rules.block.heading.exec(r);if(t){let n=t[2].trim();if(this.rules.other.endingHash.test(n)){let l=Z(n,"#");(this.options.pedantic||!l||this.rules.other.endingSpaceChar.test(l))&&(n=l.trim())}return{type:"heading",raw:Z(t[0],`
`),depth:t[1].length,text:n,tokens:this.lexer.inline(n)}}}hr(r){let t=this.rules.block.hr.exec(r);if(t)return{type:"hr",raw:Z(t[0],`
`)}}blockquote(r){let t=this.rules.block.blockquote.exec(r);if(t){let n=Z(t[0],`
`).split(`
`),l="",o="",a=[];for(;n.length>0;){let s=!1,p=[],i;for(i=0;i<n.length;i++)if(this.rules.other.blockquoteStart.test(n[i]))p.push(n[i]),s=!0;else if(!s)p.push(n[i]);else break;n=n.slice(i);let y=p.join(`
`),k=y.replace(this.rules.other.blockquoteSetextReplace,`
    $1`).replace(this.rules.other.blockquoteSetextReplace2,"");l=l?`${l}
${y}`:y,o=o?`${o}
${k}`:k;let x=this.lexer.state.top;if(this.lexer.state.top=!0,this.lexer.blockTokens(k,a,!0),this.lexer.state.top=x,n.length===0)break;let N=a.at(-1);if((N==null?void 0:N.type)==="code")break;if((N==null?void 0:N.type)==="blockquote"){let T=N,m=T.raw+`
`+n.join(`
`),P=this.blockquote(m);a[a.length-1]=P,l=l.substring(0,l.length-T.raw.length)+P.raw,o=o.substring(0,o.length-T.text.length)+P.text;break}else if((N==null?void 0:N.type)==="list"){let T=N,m=T.raw+`
`+n.join(`
`),P=this.list(m);a[a.length-1]=P,l=l.substring(0,l.length-N.raw.length)+P.raw,o=o.substring(0,o.length-T.raw.length)+P.raw,n=m.substring(a.at(-1).raw.length).split(`
`);continue}}return{type:"blockquote",raw:l,tokens:a,text:o}}}list(r){let t=this.rules.block.list.exec(r);if(t){let n=t[1].trim(),l=n.length>1,o={type:"list",raw:"",ordered:l,start:l?+n.slice(0,-1):"",loose:!1,items:[]};n=l?`\\d{1,9}\\${n.slice(-1)}`:`\\${n}`,this.options.pedantic&&(n=l?n:"[*+-]");let a=this.rules.other.listItemRegex(n),s=!1;for(;r;){let i=!1,y="",k="";if(!(t=a.exec(r))||this.rules.block.hr.test(r))break;y=t[0],r=r.substring(y.length);let x=dl(t[2].split(`
`,1)[0],t[1].length),N=r.split(`
`,1)[0],T=!x.trim(),m=0;if(this.options.pedantic?(m=2,k=x.trimStart()):T?m=t[1].length+1:(m=x.search(this.rules.other.nonSpaceChar),m=m>4?1:m,k=x.slice(m),m+=t[1].length),T&&this.rules.other.blankLine.test(N)&&(y+=N+`
`,r=r.substring(N.length+1),i=!0),!i){let P=this.rules.other.nextBulletRegex(m),B=this.rules.other.hrRegex(m),G=this.rules.other.fencesBeginRegex(m),M=this.rules.other.headingBeginRegex(m),O=this.rules.other.htmlBeginRegex(m),J=this.rules.other.blockquoteBeginRegex(m);for(;r;){let L=r.split(`
`,1)[0],F;if(N=L,this.options.pedantic?(N=N.replace(this.rules.other.listReplaceNesting,"  "),F=N):F=N.replace(this.rules.other.tabCharGlobal,"    "),G.test(N)||M.test(N)||O.test(N)||J.test(N)||P.test(N)||B.test(N))break;if(F.search(this.rules.other.nonSpaceChar)>=m||!N.trim())k+=`
`+F.slice(m);else{if(T||x.replace(this.rules.other.tabCharGlobal,"    ").search(this.rules.other.nonSpaceChar)>=4||G.test(x)||M.test(x)||B.test(x))break;k+=`
`+N}T=!N.trim(),y+=L+`
`,r=r.substring(L.length+1),x=F.slice(m)}}o.loose||(s?o.loose=!0:this.rules.other.doubleBlankLine.test(y)&&(s=!0)),o.items.push({type:"list_item",raw:y,task:!!this.options.gfm&&this.rules.other.listIsTask.test(k),loose:!1,text:k,tokens:[]}),o.raw+=y}let p=o.items.at(-1);if(p)p.raw=p.raw.trimEnd(),p.text=p.text.trimEnd();else return;o.raw=o.raw.trimEnd();for(let i of o.items){this.lexer.state.top=!1,i.tokens=this.lexer.blockTokens(i.text,[]);let y=i.tokens[0];if(i.task&&((y==null?void 0:y.type)==="text"||(y==null?void 0:y.type)==="paragraph")){i.text=i.text.replace(this.rules.other.listReplaceTask,""),y.raw=y.raw.replace(this.rules.other.listReplaceTask,""),y.text=y.text.replace(this.rules.other.listReplaceTask,"");for(let x=this.lexer.inlineQueue.length-1;x>=0;x--)if(this.rules.other.listIsTask.test(this.lexer.inlineQueue[x].src)){this.lexer.inlineQueue[x].src=this.lexer.inlineQueue[x].src.replace(this.rules.other.listReplaceTask,"");break}let k=this.rules.other.listTaskCheckbox.exec(i.raw);if(k){let x={type:"checkbox",raw:k[0]+" ",checked:k[0]!=="[ ]"};i.checked=x.checked,o.loose?i.tokens[0]&&["paragraph","text"].includes(i.tokens[0].type)&&"tokens"in i.tokens[0]&&i.tokens[0].tokens?(i.tokens[0].raw=x.raw+i.tokens[0].raw,i.tokens[0].text=x.raw+i.tokens[0].text,i.tokens[0].tokens.unshift(x)):i.tokens.unshift({type:"paragraph",raw:x.raw,text:x.raw,tokens:[x]}):i.tokens.unshift(x)}}else i.task&&(i.task=!1);if(!o.loose){let k=i.tokens.filter(N=>N.type==="space"),x=k.length>0&&k.some(N=>this.rules.other.anyLine.test(N.raw));o.loose=x}}if(o.loose)for(let i of o.items){i.loose=!0;for(let y of i.tokens)y.type==="text"&&(y.type="paragraph")}return o}}html(r){let t=this.rules.block.html.exec(r);if(t){let n=Oe(t[0]);return{type:"html",block:!0,raw:n,pre:t[1]==="pre"||t[1]==="script"||t[1]==="style",text:n}}}def(r){let t=this.rules.block.def.exec(r);if(t){let n=t[1].toLowerCase().replace(this.rules.other.multipleSpaceGlobal," "),l=t[2]?t[2].replace(this.rules.other.hrefBrackets,"$1").replace(this.rules.inline.anyPunctuation,"$1"):"",o=t[3]?t[3].substring(1,t[3].length-1).replace(this.rules.inline.anyPunctuation,"$1"):t[3];return{type:"def",tag:n,raw:Z(t[0],`
`),href:l,title:o}}}table(r){var s;let t=this.rules.block.table.exec(r);if(!t||!this.rules.other.tableDelimiter.test(t[2]))return;let n=Fe(t[1]),l=t[2].replace(this.rules.other.tableAlignChars,"").split("|"),o=(s=t[3])!=null&&s.trim()?t[3].replace(this.rules.other.tableRowBlankLine,"").split(`
`):[],a={type:"table",raw:Z(t[0],`
`),header:[],align:[],rows:[]};if(n.length===l.length){for(let p of l)this.rules.other.tableAlignRight.test(p)?a.align.push("right"):this.rules.other.tableAlignCenter.test(p)?a.align.push("center"):this.rules.other.tableAlignLeft.test(p)?a.align.push("left"):a.align.push(null);for(let p=0;p<n.length;p++)a.header.push({text:n[p],tokens:this.lexer.inline(n[p]),header:!0,align:a.align[p]});for(let p of o)a.rows.push(Fe(p,a.header.length).map((i,y)=>({text:i,tokens:this.lexer.inline(i),header:!1,align:a.align[y]})));return a}}lheading(r){let t=this.rules.block.lheading.exec(r);if(t){let n=t[1].trim();return{type:"heading",raw:Z(t[0],`
`),depth:t[2].charAt(0)==="="?1:2,text:n,tokens:this.lexer.inline(n)}}}paragraph(r){let t=this.rules.block.paragraph.exec(r);if(t){let n=t[1].charAt(t[1].length-1)===`
`?t[1].slice(0,-1):t[1];return{type:"paragraph",raw:t[0],text:n,tokens:this.lexer.inline(n)}}}text(r){let t=this.rules.block.text.exec(r);if(t)return{type:"text",raw:t[0],text:t[0],tokens:this.lexer.inline(t[0])}}escape(r){let t=this.rules.inline.escape.exec(r);if(t)return{type:"escape",raw:t[0],text:t[1]}}tag(r){let t=this.rules.inline.tag.exec(r);if(t)return!this.lexer.state.inLink&&this.rules.other.startATag.test(t[0])?this.lexer.state.inLink=!0:this.lexer.state.inLink&&this.rules.other.endATag.test(t[0])&&(this.lexer.state.inLink=!1),!this.lexer.state.inRawBlock&&this.rules.other.startPreScriptTag.test(t[0])?this.lexer.state.inRawBlock=!0:this.lexer.state.inRawBlock&&this.rules.other.endPreScriptTag.test(t[0])&&(this.lexer.state.inRawBlock=!1),{type:"html",raw:t[0],inLink:this.lexer.state.inLink,inRawBlock:this.lexer.state.inRawBlock,block:!1,text:t[0]}}link(r){let t=this.rules.inline.link.exec(r);if(t){let n=t[2].trim();if(!this.options.pedantic&&this.rules.other.startAngleBracket.test(n)){if(!this.rules.other.endAngleBracket.test(n))return;let a=Z(n.slice(0,-1),"\\");if((n.length-a.length)%2===0)return}else{let a=cl(t[2],"()");if(a===-2)return;if(a>-1){let s=(t[0].indexOf("!")===0?5:4)+t[1].length+a;t[2]=t[2].substring(0,a),t[0]=t[0].substring(0,s).trim(),t[3]=""}}let l=t[2],o="";if(this.options.pedantic){let a=this.rules.other.pedanticHrefTitle.exec(l);a&&(l=a[1],o=a[3])}else o=t[3]?t[3].slice(1,-1):"";return l=l.trim(),this.rules.other.startAngleBracket.test(l)&&(this.options.pedantic&&!this.rules.other.endAngleBracket.test(n)?l=l.slice(1):l=l.slice(1,-1)),ze(t,{href:l&&l.replace(this.rules.inline.anyPunctuation,"$1"),title:o&&o.replace(this.rules.inline.anyPunctuation,"$1")},t[0],this.lexer,this.rules)}}reflink(r,t){let n;if((n=this.rules.inline.reflink.exec(r))||(n=this.rules.inline.nolink.exec(r))){let l=(n[2]||n[1]).replace(this.rules.other.multipleSpaceGlobal," "),o=t[l.toLowerCase()];if(!o){let a=n[0].charAt(0);return{type:"text",raw:a,text:a}}return ze(n,o,n[0],this.lexer,this.rules)}}emStrong(r,t,n=""){let l=this.rules.inline.emStrongLDelim.exec(r);if(!(!l||!l[1]&&!l[2]&&!l[3]&&!l[4]||l[4]&&n.match(this.rules.other.unicodeAlphaNumeric))&&(!(l[1]||l[3])||!n||this.rules.inline.punctuation.exec(n))){let o=[...l[0]].length-1,a,s,p=o,i=0,y=l[0][0]==="*"?this.rules.inline.emStrongRDelimAst:this.rules.inline.emStrongRDelimUnd;for(y.lastIndex=0,t=t.slice(-1*r.length+o);(l=y.exec(t))!==null;){if(a=l[1]||l[2]||l[3]||l[4]||l[5]||l[6],!a)continue;if(s=[...a].length,l[3]||l[4]){p+=s;continue}else if((l[5]||l[6])&&o%3&&!((o+s)%3)){i+=s;continue}if(p-=s,p>0)continue;s=Math.min(s,s+p+i);let k=[...l[0]][0].length,x=r.slice(0,o+l.index+k+s);if(Math.min(o,s)%2){let T=x.slice(1,-1);return{type:"em",raw:x,text:T,tokens:this.lexer.inlineTokens(T)}}let N=x.slice(2,-2);return{type:"strong",raw:x,text:N,tokens:this.lexer.inlineTokens(N)}}}}codespan(r){let t=this.rules.inline.code.exec(r);if(t){let n=t[2].replace(this.rules.other.newLineCharGlobal," "),l=this.rules.other.nonSpaceChar.test(n),o=this.rules.other.startingSpaceChar.test(n)&&this.rules.other.endingSpaceChar.test(n);return l&&o&&(n=n.substring(1,n.length-1)),{type:"codespan",raw:t[0],text:n}}}br(r){let t=this.rules.inline.br.exec(r);if(t)return{type:"br",raw:t[0]}}del(r,t,n=""){let l=this.rules.inline.delLDelim.exec(r);if(l&&(!l[1]||!n||this.rules.inline.punctuation.exec(n))){let o=[...l[0]].length-1,a,s,p=o,i=this.rules.inline.delRDelim;for(i.lastIndex=0,t=t.slice(-1*r.length+o);(l=i.exec(t))!==null;){if(a=l[1]||l[2]||l[3]||l[4]||l[5]||l[6],!a||(s=[...a].length,s!==o))continue;if(l[3]||l[4]){p+=s;continue}if(p-=s,p>0)continue;s=Math.min(s,s+p);let y=[...l[0]][0].length,k=r.slice(0,o+l.index+y+s),x=k.slice(o,-o);return{type:"del",raw:k,text:x,tokens:this.lexer.inlineTokens(x)}}}}autolink(r){let t=this.rules.inline.autolink.exec(r);if(t){let n,l;return t[2]==="@"?(n=t[1],l="mailto:"+n):(n=t[1],l=n),{type:"link",raw:t[0],text:n,href:l,tokens:[{type:"text",raw:n,text:n}]}}}url(r){var n;let t;if(t=this.rules.inline.url.exec(r)){let l,o;if(t[2]==="@")l=t[0],o="mailto:"+l;else{let a;do a=t[0],t[0]=((n=this.rules.inline._backpedal.exec(t[0]))==null?void 0:n[0])??"";while(a!==t[0]);l=t[0],t[1]==="www."?o="http://"+t[0]:o=t[0]}return{type:"link",raw:t[0],text:l,href:o,tokens:[{type:"text",raw:l,text:l}]}}}inlineText(r){let t=this.rules.inline.text.exec(r);if(t){let n=this.lexer.state.inRawBlock;return{type:"text",raw:t[0],text:t[0],escaped:n}}}},H=class be{constructor(t){I(this,"tokens");I(this,"options");I(this,"state");I(this,"inlineQueue");I(this,"tokenizer");this.tokens=[],this.tokens.links=Object.create(null),this.options=t||K,this.options.tokenizer=this.options.tokenizer||new ae,this.tokenizer=this.options.tokenizer,this.tokenizer.options=this.options,this.tokenizer.lexer=this,this.inlineQueue=[],this.state={inLink:!1,inRawBlock:!1,top:!0};let n={other:R,block:oe.normal,inline:ee.normal};this.options.pedantic?(n.block=oe.pedantic,n.inline=ee.pedantic):this.options.gfm&&(n.block=oe.gfm,this.options.breaks?n.inline=ee.breaks:n.inline=ee.gfm),this.tokenizer.rules=n}static get rules(){return{block:oe,inline:ee}}static lex(t,n){return new be(n).lex(t)}static lexInline(t,n){return new be(n).inlineTokens(t)}lex(t){t=t.replace(R.carriageReturn,`
`),this.blockTokens(t,this.tokens);for(let n=0;n<this.inlineQueue.length;n++){let l=this.inlineQueue[n];this.inlineTokens(l.src,l.tokens)}return this.inlineQueue=[],this.tokens}blockTokens(t,n=[],l=!1){var a,s,p;this.tokenizer.lexer=this,this.options.pedantic&&(t=t.replace(R.tabCharGlobal,"    ").replace(R.spaceLine,""));let o=1/0;for(;t;){if(t.length<o)o=t.length;else{this.infiniteLoopError(t.charCodeAt(0));break}let i;if((s=(a=this.options.extensions)==null?void 0:a.block)!=null&&s.some(k=>(i=k.call({lexer:this},t,n))?(t=t.substring(i.raw.length),n.push(i),!0):!1))continue;if(i=this.tokenizer.space(t)){t=t.substring(i.raw.length);let k=n.at(-1);i.raw.length===1&&k!==void 0?k.raw+=`
`:n.push(i);continue}if(i=this.tokenizer.code(t)){t=t.substring(i.raw.length);let k=n.at(-1);(k==null?void 0:k.type)==="paragraph"||(k==null?void 0:k.type)==="text"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.text,this.inlineQueue.at(-1).src=k.text):n.push(i);continue}if(i=this.tokenizer.fences(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.heading(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.hr(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.blockquote(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.list(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.html(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.def(t)){t=t.substring(i.raw.length);let k=n.at(-1);(k==null?void 0:k.type)==="paragraph"||(k==null?void 0:k.type)==="text"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.raw,this.inlineQueue.at(-1).src=k.text):this.tokens.links[i.tag]||(this.tokens.links[i.tag]={href:i.href,title:i.title},n.push(i));continue}if(i=this.tokenizer.table(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.lheading(t)){t=t.substring(i.raw.length),n.push(i);continue}let y=t;if((p=this.options.extensions)!=null&&p.startBlock){let k=1/0,x=t.slice(1),N;this.options.extensions.startBlock.forEach(T=>{N=T.call({lexer:this},x),typeof N=="number"&&N>=0&&(k=Math.min(k,N))}),k<1/0&&k>=0&&(y=t.substring(0,k+1))}if(this.state.top&&(i=this.tokenizer.paragraph(y))){let k=n.at(-1);l&&(k==null?void 0:k.type)==="paragraph"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.text,this.inlineQueue.pop(),this.inlineQueue.at(-1).src=k.text):n.push(i),l=y.length!==t.length,t=t.substring(i.raw.length);continue}if(i=this.tokenizer.text(t)){t=t.substring(i.raw.length);let k=n.at(-1);(k==null?void 0:k.type)==="text"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.text,this.inlineQueue.pop(),this.inlineQueue.at(-1).src=k.text):n.push(i);continue}if(t){this.infiniteLoopError(t.charCodeAt(0));break}}return this.state.top=!0,n}inline(t,n=[]){return this.inlineQueue.push({src:t,tokens:n}),n}inlineTokens(t,n=[]){var y,k,x,N,T;this.tokenizer.lexer=this;let l=t,o=null;if(this.tokens.links){let m=Object.keys(this.tokens.links);if(m.length>0)for(;(o=this.tokenizer.rules.inline.reflinkSearch.exec(l))!==null;)m.includes(o[0].slice(o[0].lastIndexOf("[")+1,-1))&&(l=l.slice(0,o.index)+"["+"a".repeat(o[0].length-2)+"]"+l.slice(this.tokenizer.rules.inline.reflinkSearch.lastIndex))}for(;(o=this.tokenizer.rules.inline.anyPunctuation.exec(l))!==null;)l=l.slice(0,o.index)+"++"+l.slice(this.tokenizer.rules.inline.anyPunctuation.lastIndex);let a;for(;(o=this.tokenizer.rules.inline.blockSkip.exec(l))!==null;)a=o[2]?o[2].length:0,l=l.slice(0,o.index+a)+"["+"a".repeat(o[0].length-a-2)+"]"+l.slice(this.tokenizer.rules.inline.blockSkip.lastIndex);l=((k=(y=this.options.hooks)==null?void 0:y.emStrongMask)==null?void 0:k.call({lexer:this},l))??l;let s=!1,p="",i=1/0;for(;t;){if(t.length<i)i=t.length;else{this.infiniteLoopError(t.charCodeAt(0));break}s||(p=""),s=!1;let m;if((N=(x=this.options.extensions)==null?void 0:x.inline)!=null&&N.some(B=>(m=B.call({lexer:this},t,n))?(t=t.substring(m.raw.length),n.push(m),!0):!1))continue;if(m=this.tokenizer.escape(t)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.tag(t)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.link(t)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.reflink(t,this.tokens.links)){t=t.substring(m.raw.length);let B=n.at(-1);m.type==="text"&&(B==null?void 0:B.type)==="text"?(B.raw+=m.raw,B.text+=m.text):n.push(m);continue}if(m=this.tokenizer.emStrong(t,l,p)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.codespan(t)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.br(t)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.del(t,l,p)){t=t.substring(m.raw.length),n.push(m);continue}if(m=this.tokenizer.autolink(t)){t=t.substring(m.raw.length),n.push(m);continue}if(!this.state.inLink&&(m=this.tokenizer.url(t))){t=t.substring(m.raw.length),n.push(m);continue}let P=t;if((T=this.options.extensions)!=null&&T.startInline){let B=1/0,G=t.slice(1),M;this.options.extensions.startInline.forEach(O=>{M=O.call({lexer:this},G),typeof M=="number"&&M>=0&&(B=Math.min(B,M))}),B<1/0&&B>=0&&(P=t.substring(0,B+1))}if(m=this.tokenizer.inlineText(P)){t=t.substring(m.raw.length),m.raw.slice(-1)!=="_"&&(p=m.raw.slice(-1)),s=!0;let B=n.at(-1);(B==null?void 0:B.type)==="text"?(B.raw+=m.raw,B.text+=m.text):n.push(m);continue}if(t){this.infiniteLoopError(t.charCodeAt(0));break}}return n}infiniteLoopError(t){let n="Infinite loop on byte: "+t;if(this.options.silent)console.error(n);else throw new Error(n)}},se=class{constructor(r){I(this,"options");I(this,"parser");this.options=r||K}space(r){return""}code({text:r,lang:t,escaped:n}){var a;let l=(a=(t||"").match(R.notSpaceStart))==null?void 0:a[0],o=r.replace(R.endingNewline,"")+`
`;return l?'<pre><code class="language-'+q(l)+'">'+(n?o:q(o,!0))+`</code></pre>
`:"<pre><code>"+(n?o:q(o,!0))+`</code></pre>
`}blockquote({tokens:r}){return`<blockquote>
${this.parser.parse(r)}</blockquote>
`}html({text:r}){return r}def(r){return""}heading({tokens:r,depth:t}){return`<h${t}>${this.parser.parseInline(r)}</h${t}>
`}hr(r){return`<hr>
`}list(r){let t=r.ordered,n=r.start,l="";for(let s=0;s<r.items.length;s++){let p=r.items[s];l+=this.listitem(p)}let o=t?"ol":"ul",a=t&&n!==1?' start="'+n+'"':"";return"<"+o+a+`>
`+l+"</"+o+`>
`}listitem(r){return`<li>${this.parser.parse(r.tokens)}</li>
`}checkbox({checked:r}){return"<input "+(r?'checked="" ':"")+'disabled="" type="checkbox"> '}paragraph({tokens:r}){return`<p>${this.parser.parseInline(r)}</p>
`}table(r){let t="",n="";for(let o=0;o<r.header.length;o++)n+=this.tablecell(r.header[o]);t+=this.tablerow({text:n});let l="";for(let o=0;o<r.rows.length;o++){let a=r.rows[o];n="";for(let s=0;s<a.length;s++)n+=this.tablecell(a[s]);l+=this.tablerow({text:n})}return l&&(l=`<tbody>${l}</tbody>`),`<table>
<thead>
`+t+`</thead>
`+l+`</table>
`}tablerow({text:r}){return`<tr>
${r}</tr>
`}tablecell(r){let t=this.parser.parseInline(r.tokens),n=r.header?"th":"td";return(r.align?`<${n} align="${r.align}">`:`<${n}>`)+t+`</${n}>
`}strong({tokens:r}){return`<strong>${this.parser.parseInline(r)}</strong>`}em({tokens:r}){return`<em>${this.parser.parseInline(r)}</em>`}codespan({text:r}){return`<code>${q(r,!0)}</code>`}br(r){return"<br>"}del({tokens:r}){return`<del>${this.parser.parseInline(r)}</del>`}link({href:r,title:t,tokens:n}){let l=this.parser.parseInline(n),o=Le(r);if(o===null)return l;r=o;let a='<a href="'+r+'"';return t&&(a+=' title="'+q(t)+'"'),a+=">"+l+"</a>",a}image({href:r,title:t,text:n,tokens:l}){l&&(n=this.parser.parseInline(l,this.parser.textRenderer));let o=Le(r);if(o===null)return q(n);r=o;let a=`<img src="${r}" alt="${q(n)}"`;return t&&(a+=` title="${q(t)}"`),a+=">",a}text(r){return"tokens"in r&&r.tokens?this.parser.parseInline(r.tokens):"escaped"in r&&r.escaped?r.text:q(r.text)}},ue=class{strong({text:r}){return r}em({text:r}){return r}codespan({text:r}){return r}del({text:r}){return r}html({text:r}){return r}text({text:r}){return r}link({text:r}){return""+r}image({text:r}){return""+r}br(){return""}checkbox({raw:r}){return r}},j=class xe{constructor(t){I(this,"options");I(this,"renderer");I(this,"textRenderer");this.options=t||K,this.options.renderer=this.options.renderer||new se,this.renderer=this.options.renderer,this.renderer.options=this.options,this.renderer.parser=this,this.textRenderer=new ue}static parse(t,n){return new xe(n).parse(t)}static parseInline(t,n){return new xe(n).parseInline(t)}parse(t){var l,o;this.renderer.parser=this;let n="";for(let a=0;a<t.length;a++){let s=t[a];if((o=(l=this.options.extensions)==null?void 0:l.renderers)!=null&&o[s.type]){let i=s,y=this.options.extensions.renderers[i.type].call({parser:this},i);if(y!==!1||!["space","hr","heading","code","table","blockquote","list","html","def","paragraph","text"].includes(i.type)){n+=y||"";continue}}let p=s;switch(p.type){case"space":{n+=this.renderer.space(p);break}case"hr":{n+=this.renderer.hr(p);break}case"heading":{n+=this.renderer.heading(p);break}case"code":{n+=this.renderer.code(p);break}case"table":{n+=this.renderer.table(p);break}case"blockquote":{n+=this.renderer.blockquote(p);break}case"list":{n+=this.renderer.list(p);break}case"checkbox":{n+=this.renderer.checkbox(p);break}case"html":{n+=this.renderer.html(p);break}case"def":{n+=this.renderer.def(p);break}case"paragraph":{n+=this.renderer.paragraph(p);break}case"text":{n+=this.renderer.text(p);break}default:{let i='Token with "'+p.type+'" type was not found.';if(this.options.silent)return console.error(i),"";throw new Error(i)}}}return n}parseInline(t,n=this.renderer){var o,a;this.renderer.parser=this;let l="";for(let s=0;s<t.length;s++){let p=t[s];if((a=(o=this.options.extensions)==null?void 0:o.renderers)!=null&&a[p.type]){let y=this.options.extensions.renderers[p.type].call({parser:this},p);if(y!==!1||!["escape","html","link","image","strong","em","codespan","br","del","text"].includes(p.type)){l+=y||"";continue}}let i=p;switch(i.type){case"escape":{l+=n.text(i);break}case"html":{l+=n.html(i);break}case"link":{l+=n.link(i);break}case"image":{l+=n.image(i);break}case"checkbox":{l+=n.checkbox(i);break}case"strong":{l+=n.strong(i);break}case"em":{l+=n.em(i);break}case"codespan":{l+=n.codespan(i);break}case"br":{l+=n.br(i);break}case"del":{l+=n.del(i);break}case"text":{l+=n.text(i);break}default:{let y='Token with "'+i.type+'" type was not found.';if(this.options.silent)return console.error(y),"";throw new Error(y)}}}return l}},ne=(ie=class{constructor(r){I(this,"options");I(this,"block");this.options=r||K}preprocess(r){return r}postprocess(r){return r}processAllTokens(r){return r}emStrongMask(r){return r}provideLexer(r=this.block){return r?H.lex:H.lexInline}provideParser(r=this.block){return r?j.parse:j.parseInline}},I(ie,"passThroughHooks",new Set(["preprocess","postprocess","processAllTokens","emStrongMask"])),I(ie,"passThroughHooksRespectAsync",new Set(["preprocess","postprocess","processAllTokens"])),ie),ml=class{constructor(...r){I(this,"defaults",ce());I(this,"options",this.setOptions);I(this,"parse",this.parseMarkdown(!0));I(this,"parseInline",this.parseMarkdown(!1));I(this,"Parser",j);I(this,"Renderer",se);I(this,"TextRenderer",ue);I(this,"Lexer",H);I(this,"Tokenizer",ae);I(this,"Hooks",ne);this.use(...r)}walkTokens(r,t){var l,o;let n=[];for(let a of r)switch(n=n.concat(t.call(this,a)),a.type){case"table":{let s=a;for(let p of s.header)n=n.concat(this.walkTokens(p.tokens,t));for(let p of s.rows)for(let i of p)n=n.concat(this.walkTokens(i.tokens,t));break}case"list":{let s=a;n=n.concat(this.walkTokens(s.items,t));break}default:{let s=a;(o=(l=this.defaults.extensions)==null?void 0:l.childTokens)!=null&&o[s.type]?this.defaults.extensions.childTokens[s.type].forEach(p=>{let i=s[p].flat(1/0);n=n.concat(this.walkTokens(i,t))}):s.tokens&&(n=n.concat(this.walkTokens(s.tokens,t)))}}return n}use(...r){let t=this.defaults.extensions||{renderers:{},childTokens:{}};return r.forEach(n=>{let l={...n};if(l.async=this.defaults.async||l.async||!1,n.extensions&&(n.extensions.forEach(o=>{if(!o.name)throw new Error("extension name required");if("renderer"in o){let a=t.renderers[o.name];a?t.renderers[o.name]=function(...s){let p=o.renderer.apply(this,s);return p===!1&&(p=a.apply(this,s)),p}:t.renderers[o.name]=o.renderer}if("tokenizer"in o){if(!o.level||o.level!=="block"&&o.level!=="inline")throw new Error("extension level must be 'block' or 'inline'");let a=t[o.level];a?a.unshift(o.tokenizer):t[o.level]=[o.tokenizer],o.start&&(o.level==="block"?t.startBlock?t.startBlock.push(o.start):t.startBlock=[o.start]:o.level==="inline"&&(t.startInline?t.startInline.push(o.start):t.startInline=[o.start]))}"childTokens"in o&&o.childTokens&&(t.childTokens[o.name]=o.childTokens)}),l.extensions=t),n.renderer){let o=this.defaults.renderer||new se(this.defaults);for(let a in n.renderer){if(!(a in o))throw new Error(`renderer '${a}' does not exist`);if(["options","parser"].includes(a))continue;let s=a,p=n.renderer[s],i=o[s];o[s]=(...y)=>{let k=p.apply(o,y);return k===!1&&(k=i.apply(o,y)),k||""}}l.renderer=o}if(n.tokenizer){let o=this.defaults.tokenizer||new ae(this.defaults);for(let a in n.tokenizer){if(!(a in o))throw new Error(`tokenizer '${a}' does not exist`);if(["options","rules","lexer"].includes(a))continue;let s=a,p=n.tokenizer[s],i=o[s];o[s]=(...y)=>{let k=p.apply(o,y);return k===!1&&(k=i.apply(o,y)),k}}l.tokenizer=o}if(n.hooks){let o=this.defaults.hooks||new ne;for(let a in n.hooks){if(!(a in o))throw new Error(`hook '${a}' does not exist`);if(["options","block"].includes(a))continue;let s=a,p=n.hooks[s],i=o[s];ne.passThroughHooks.has(a)?o[s]=y=>{if(this.defaults.async&&ne.passThroughHooksRespectAsync.has(a))return(async()=>{let x=await p.call(o,y);return i.call(o,x)})();let k=p.call(o,y);return i.call(o,k)}:o[s]=(...y)=>{if(this.defaults.async)return(async()=>{let x=await p.apply(o,y);return x===!1&&(x=await i.apply(o,y)),x})();let k=p.apply(o,y);return k===!1&&(k=i.apply(o,y)),k}}l.hooks=o}if(n.walkTokens){let o=this.defaults.walkTokens,a=n.walkTokens;l.walkTokens=function(s){let p=[];return p.push(a.call(this,s)),o&&(p=p.concat(o.call(this,s))),p}}this.defaults={...this.defaults,...l}}),this}setOptions(r){return this.defaults={...this.defaults,...r},this}lexer(r,t){return H.lex(r,t??this.defaults)}parser(r,t){return j.parse(r,t??this.defaults)}parseMarkdown(r){return(t,n)=>{let l={...n},o={...this.defaults,...l},a=this.onError(!!o.silent,!!o.async);if(this.defaults.async===!0&&l.async===!1)return a(new Error("marked(): The async option was set to true by an extension. Remove async: false from the parse options object to return a Promise."));if(typeof t>"u"||t===null)return a(new Error("marked(): input parameter is undefined or null"));if(typeof t!="string")return a(new Error("marked(): input parameter is of type "+Object.prototype.toString.call(t)+", string expected"));if(o.hooks&&(o.hooks.options=o,o.hooks.block=r),o.async)return(async()=>{let s=o.hooks?await o.hooks.preprocess(t):t,p=await(o.hooks?await o.hooks.provideLexer(r):r?H.lex:H.lexInline)(s,o),i=o.hooks?await o.hooks.processAllTokens(p):p;o.walkTokens&&await Promise.all(this.walkTokens(i,o.walkTokens));let y=await(o.hooks?await o.hooks.provideParser(r):r?j.parse:j.parseInline)(i,o);return o.hooks?await o.hooks.postprocess(y):y})().catch(a);try{o.hooks&&(t=o.hooks.preprocess(t));let s=(o.hooks?o.hooks.provideLexer(r):r?H.lex:H.lexInline)(t,o);o.hooks&&(s=o.hooks.processAllTokens(s)),o.walkTokens&&this.walkTokens(s,o.walkTokens);let p=(o.hooks?o.hooks.provideParser(r):r?j.parse:j.parseInline)(s,o);return o.hooks&&(p=o.hooks.postprocess(p)),p}catch(s){return a(s)}}}onError(r,t){return n=>{if(n.message+=`
Please report this to https://github.com/markedjs/marked.`,r){let l="<p>An error occurred:</p><pre>"+q(n.message+"",!0)+"</pre>";return t?Promise.resolve(l):l}if(t)return Promise.reject(n);throw n}}},X=new ml;function C(r,t){return X.parse(r,t)}C.options=C.setOptions=function(r){return X.setOptions(r),C.defaults=X.defaults,we(C.defaults),C},C.getDefaults=ce,C.defaults=K,C.use=function(...r){return X.use(...r),C.defaults=X.defaults,we(C.defaults),C},C.walkTokens=function(r,t){return X.walkTokens(r,t)},C.parseInline=X.parseInline,C.Parser=j,C.parser=j.parse,C.Renderer=se,C.TextRenderer=ue,C.Lexer=H,C.lexer=H.lex,C.Tokenizer=ae,C.Hooks=ne,C.parse=C,C.options,C.setOptions,C.use,C.walkTokens,C.parseInline,j.parse,H.lex;const gl={class:"modal-content help-modal"},kl={class:"modal-header"},hl={class:"header-actions"},fl={class:"modal-body"},yl={class:"doc-sidebar"},ul={class:"doc-sidebar-group"},El={class:"doc-sidebar-header"},bl=["onClick"],xl={class:"doc-sidebar-group"},Vl={class:"doc-sidebar-header"},Nl={class:"doc-content"},wl=["innerHTML"],Tl=["innerHTML"],Bl=["innerHTML"],Sl=["innerHTML"],Cl=["innerHTML"],Il=["innerHTML"],Pl=["innerHTML"],Al={class:"doc-pagination"},$l=["disabled"],Ml={class:"page-info"},Dl=["disabled"],Rl=U({__name:"HelpModal",props:{initialDoc:{type:String,default:"getting-started"}},emits:["close","openAbout"],setup(r,{emit:t}){const n=r,l=e.ref(n.initialDoc),o=e.ref(""),a=e.ref(null),s=[{id:"getting-started",title:"快速开始",icon:"home"},{id:"features",title:"功能介绍",icon:"info"},{id:"api",title:"API 文档",icon:"code"},{id:"tools",title:"工具文档",icon:"tool"},{id:"shortcuts",title:"快捷键",icon:"keyboard"},{id:"faq",title:"常见问题",icon:"help"}],p=e.ref(s),i=e.computed(()=>[...p.value,{id:"changelog",title:"更新日志",icon:"activity"}]),y=e.computed(()=>i.value.findIndex(V=>V.id===l.value)),k=e.computed(()=>y.value>0),x=e.computed(()=>y.value<i.value.length-1);function N(){const V=o.value.toLowerCase().trim();if(!V){p.value=s;return}p.value=s.filter(u=>u.title.toLowerCase().includes(V)||u.id.includes(V)),p.value.length>0&&!p.value.find(u=>u.id===l.value)&&(l.value=p.value[0].id)}const T=V=>C(V,{breaks:!0,gfm:!0}).replace(/<table>/g,'<table class="doc-table">'),m=e.computed(()=>T(Nr)),P=e.computed(()=>T(wr)),B=e.computed(()=>T(Er)),G=e.computed(()=>T(br)),M=e.computed(()=>T(xr)),O=e.computed(()=>T(Vr)),J=e.computed(()=>T(Tr));function L(){var V;k.value&&(l.value=i.value[y.value-1].id,(V=a.value)==null||V.scrollTo(0,0))}function F(){var V;x.value&&(l.value=i.value[y.value+1].id,(V=a.value)==null||V.scrollTo(0,0))}return e.onMounted(()=>{N()}),(V,u)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:u[3]||(u[3]=e.withModifiers(h=>V.$emit("close"),["self"]))},[e.createElementVNode("div",gl,[e.createCommentVNode(" 头部 "),e.createElementVNode("div",kl,[e.createElementVNode("h2",null,[e.createVNode($,{name:"book-open",size:18}),u[4]||(u[4]=e.createTextVNode(" 帮助文档",-1))]),e.createElementVNode("div",hl,[e.createElementVNode("button",{class:"btn-about",onClick:u[0]||(u[0]=h=>V.$emit("openAbout")),title:"关于 PairCode"},[e.createVNode($,{name:"info",size:14}),u[5]||(u[5]=e.createTextVNode(" 关于 ",-1))]),e.createElementVNode("button",{class:"modal-close",onClick:u[1]||(u[1]=h=>V.$emit("close"))},"×")])]),e.createCommentVNode(" 主体 "),e.createElementVNode("div",fl,[e.createCommentVNode(" 侧边导航 "),e.createElementVNode("div",yl,[e.createCommentVNode(" 文档中心分组 "),e.createElementVNode("div",ul,[e.createElementVNode("div",El,[e.createVNode($,{name:"book",size:14}),u[6]||(u[6]=e.createElementVNode("span",null,"文档中心",-1))]),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(p.value,h=>(e.openBlock(),e.createElementBlock("div",{key:h.id,class:e.normalizeClass(["doc-nav-item",{active:l.value===h.id}]),onClick:g=>l.value=h.id},[e.createVNode($,{name:h.icon,size:16},null,8,["name"]),e.createElementVNode("span",null,e.toDisplayString(h.title),1)],10,bl))),128))]),e.createCommentVNode(" 更新日志 "),e.createElementVNode("div",xl,[e.createElementVNode("div",Vl,[e.createVNode($,{name:"clock",size:14}),u[7]||(u[7]=e.createElementVNode("span",null,"其他",-1))]),e.createElementVNode("div",{class:e.normalizeClass(["doc-nav-item",{active:l.value==="changelog"}]),onClick:u[2]||(u[2]=h=>l.value="changelog")},[e.createVNode($,{name:"activity",size:16}),u[8]||(u[8]=e.createElementVNode("span",null,"更新日志",-1))],2)])]),e.createCommentVNode(" 文档内容 "),e.createElementVNode("div",Nl,[e.createElementVNode("div",{class:"doc-content-inner",ref_key:"contentRef",ref:a},[l.value==="faq"?(e.openBlock(),e.createElementBlock("div",{key:0,class:"doc-markdown",innerHTML:m.value},null,8,wl)):l.value==="getting-started"?(e.openBlock(),e.createElementBlock("div",{key:1,class:"doc-markdown",innerHTML:P.value},null,8,Tl)):l.value==="features"?(e.openBlock(),e.createElementBlock("div",{key:2,class:"doc-markdown",innerHTML:B.value},null,8,Bl)):l.value==="api"?(e.openBlock(),e.createElementBlock("div",{key:3,class:"doc-markdown",innerHTML:G.value},null,8,Sl)):l.value==="tools"?(e.openBlock(),e.createElementBlock("div",{key:4,class:"doc-markdown",innerHTML:M.value},null,8,Cl)):l.value==="shortcuts"?(e.openBlock(),e.createElementBlock("div",{key:5,class:"doc-markdown",innerHTML:O.value},null,8,Il)):l.value==="changelog"?(e.openBlock(),e.createElementBlock("div",{key:6,class:"doc-markdown",innerHTML:J.value},null,8,Pl)):e.createCommentVNode("v-if",!0)],512),e.createCommentVNode(" 底部翻页 "),e.createElementVNode("div",Al,[e.createElementVNode("button",{class:"page-btn",onClick:L,disabled:!k.value},[e.createVNode($,{name:"chevron-left",size:14}),u[9]||(u[9]=e.createTextVNode(" 上一页 ",-1))],8,$l),e.createElementVNode("span",Ml,e.toDisplayString(y.value+1)+" / "+e.toDisplayString(i.value.length),1),e.createElementVNode("button",{class:"page-btn",onClick:F,disabled:!x.value},[u[10]||(u[10]=e.createTextVNode(" 下一页 ",-1)),e.createVNode($,{name:"chevron-right",size:14})],8,Dl)])])])])]))}},[["__scopeId","data-v-203b7be1"]]),Gl="data:image/svg+xml,%3csvg%20xmlns='http://www.w3.org/2000/svg'%20width='512'%20height='512'%20viewBox='0%200%20512%20512'%3e%3cdefs%3e%3c!--%20背景渐变（深色科技风）%20--%3e%3clinearGradient%20id='bgGrad'%20x1='0'%20y1='0'%20x2='1'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%230a1628'/%3e%3cstop%20offset='100%25'%20stop-color='%230d1f2e'/%3e%3c/linearGradient%3e%3c!--%20左侧尖括号渐变（科技蓝）%20--%3e%3clinearGradient%20id='leftBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='100%25'%20stop-color='%230077b6'/%3e%3c/linearGradient%3e%3c!--%20右侧尖括号渐变（科技绿）%20--%3e%3clinearGradient%20id='rightBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300e676'/%3e%3cstop%20offset='100%25'%20stop-color='%2300c853'/%3e%3c/linearGradient%3e%3c!--%20中间连接线（蓝绿渐变）%20--%3e%3clinearGradient%20id='connector'%20x1='0'%20y1='0'%20x2='1'%20y2='0'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='50%25'%20stop-color='%2300e5ff'/%3e%3cstop%20offset='100%25'%20stop-color='%2300e676'/%3e%3c/linearGradient%3e%3c!--%20外发光%20--%3e%3cfilter%20id='glow'%3e%3cfeGaussianBlur%20stdDeviation='4'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3cfilter%20id='softGlow'%3e%3cfeGaussianBlur%20stdDeviation='8'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3c/defs%3e%3c!--%20圆角方形背景（深色科技底）%20--%3e%3crect%20x='32'%20y='32'%20width='448'%20height='448'%20rx='96'%20ry='96'%20fill='url(%23bgGrad)'%20stroke='%231a3a4a'%20stroke-width='2'/%3e%3c!--%20左侧%20%3c%20尖括号（三段式直线%20—%20科技蓝，代表代码输入/开发者）%20--%3e%3cpath%20d='M180%20150%20L96%20256%20L180%20362'%20stroke='url(%23leftBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20右侧%20%3e%20尖括号（三段式直线%20—%20科技绿，代表代码输出/AI伙伴）%20--%3e%3cpath%20d='M332%20150%20L416%20256%20L332%20362'%20stroke='url(%23rightBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20中间「=」连接线已移除（图标只留%20%3c%20%3e%20尖括号%20+%20中心%20AI%20核心光点）。%20--%3e%3c!--%20中心光点（代表%20AI%20核心%20—%20亮青色）%20--%3e%3ccircle%20cx='256'%20cy='256'%20r='18'%20fill='transparent'%20stroke='%2300e5ff'%20stroke-width='3'%20opacity='0.6'/%3e%3ccircle%20cx='256'%20cy='256'%20r='8'%20fill='%2300e5ff'%20opacity='0.9'%3e%3canimate%20attributeName='opacity'%20values='0.6;1;0.6'%20dur='2s'%20repeatCount='indefinite'/%3e%3c/circle%3e%3c/svg%3e",Ll={class:"modal-content about-modal"},Fl={class:"modal-header"},Ol={class:"modal-body"},zl={class:"about-left-col"},Ul={class:"about-hero"},Hl={class:"about-logo"},jl=["src"],ql={class:"about-version"},Wl={class:"about-right-col"},Jl={class:"about-section"},Zl={class:"feature-list"},Kl={class:"about-section"},Ql={key:0,class:"sys-info"},Xl={class:"info-row"},Yl={class:"info-row"},_l={class:"info-row"},vl={class:"info-path"},eo={class:"info-row"},no={key:1,class:"loading-info"},to={class:"modal-footer"},ro=U({__name:"AboutModal",props:{showHelpBtn:{type:Boolean,default:!0}},emits:["close","openHelp"],setup(r,{emit:t}){const n=e.ref(""),l=e.ref({}),o=e.ref(!0);return e.onMounted(async()=>{try{const a=await A.apiGet("/system/info");l.value=a,a.version&&(n.value=a.version)}catch{}o.value=!1}),(a,s)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:s[3]||(s[3]=e.withModifiers(p=>a.$emit("close"),["self"]))},[e.createElementVNode("div",Ll,[e.createElementVNode("div",Fl,[e.createElementVNode("h2",null,[e.createVNode($,{name:"info",size:18}),s[4]||(s[4]=e.createTextVNode(" 关于 PairCode",-1))]),e.createElementVNode("button",{class:"modal-close",onClick:s[0]||(s[0]=p=>a.$emit("close"))},"×")]),e.createElementVNode("div",Ol,[e.createCommentVNode(" 左列：Logo + 描述 + 技术栈 "),e.createElementVNode("div",zl,[e.createCommentVNode(" Logo + 标题 "),e.createElementVNode("div",Ul,[e.createElementVNode("div",Hl,[e.createElementVNode("img",{src:e.unref(Gl),class:"about-logo-img",alt:"PairCode"},null,8,jl)]),s[5]||(s[5]=e.createElementVNode("div",{class:"about-title"},"PairCode IDE",-1)),e.createElementVNode("div",ql,"版本 "+e.toDisplayString(n.value),1)]),e.createCommentVNode(" 描述 "),s[6]||(s[6]=e.createElementVNode("div",{class:"about-section"},[e.createElementVNode("p",{class:"about-description"}," PairCode IDE 是一款纯 Web 端的 AI 辅助编程集成开发环境， 专为浏览器而设计。无需安装任何桌面客户端或本地 IDE 软件， 打开浏览器即可开始编程。它将 AI 对话能力深度融入编码工作流， 你只需用自然语言描述需求，AI 就能自动理解上下文、读写文件、执行命令、 管理版本控制。从代码生成到项目运维，在同一个浏览器窗口中全部完成。 ")],-1)),e.createCommentVNode(" 技术栈 "),s[7]||(s[7]=e.createStaticVNode('<div class="about-section" data-v-e61d5ebd><div class="section-title" data-v-e61d5ebd>技术栈</div><div class="tech-stack" data-v-e61d5ebd><span class="tech-badge" data-v-e61d5ebd>Go</span><span class="tech-badge" data-v-e61d5ebd>Vue 3</span><span class="tech-badge" data-v-e61d5ebd>WebSocket</span><span class="tech-badge" data-v-e61d5ebd>CodeMirror</span><span class="tech-badge" data-v-e61d5ebd>插件化工具</span><span class="tech-badge" data-v-e61d5ebd>TS 编译器</span><span class="tech-badge" data-v-e61d5ebd>MCP</span><span class="tech-badge" data-v-e61d5ebd>CodeGraph</span><span class="tech-badge" data-v-e61d5ebd>DAP</span></div></div>',1))]),e.createCommentVNode(" 右列：特性 + 系统信息 "),e.createElementVNode("div",Wl,[e.createCommentVNode(" 特性亮点 "),e.createElementVNode("div",Jl,[s[18]||(s[18]=e.createElementVNode("div",{class:"section-title"},"主要特性",-1)),e.createElementVNode("ul",Zl,[e.createElementVNode("li",null,[e.createVNode($,{name:"bot",size:14,color:"var(--accent)"}),s[8]||(s[8]=e.createTextVNode(" AI 对话编程 — 用自然语言与 AI 对话，自动生成与重构代码",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"file",size:14,color:"var(--accent)"}),s[9]||(s[9]=e.createTextVNode(" 智能代码编辑器 — 多语言语法高亮，浏览器中流畅编辑",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"git-branch",size:14,color:"var(--accent)"}),s[10]||(s[10]=e.createTextVNode(" Git 版本控制 — 在对话中完成全部 Git 操作",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"terminal",size:14,color:"var(--accent)"}),s[11]||(s[11]=e.createTextVNode(" 内置终端 — 无需离开浏览器即可执行命令",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"search",size:14,color:"var(--accent)"}),s[12]||(s[12]=e.createTextVNode(" 全局搜索 — 快速搜索文件与代码内容",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"settings",size:14,color:"var(--accent)"}),s[13]||(s[13]=e.createTextVNode(" 自主 Agent 模式 — AI 主动分析项目并自动执行任务",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"grid",size:14,color:"var(--accent)"}),s[14]||(s[14]=e.createTextVNode(" 对话历史管理 — 自动保存、回溯与继续历史对话",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"tool",size:14,color:"var(--accent)"}),s[15]||(s[15]=e.createTextVNode(" Skills / MCP 扩展 — 通过技能市场扩展 IDE 能力",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"code",size:14,color:"var(--accent)"}),s[16]||(s[16]=e.createTextVNode(" 内置调试器 — 支持 Go 程序的断点、单步和变量查看",-1))]),e.createElementVNode("li",null,[e.createVNode($,{name:"image",size:14,color:"var(--accent)"}),s[17]||(s[17]=e.createTextVNode(" 网页验证 — 打开 URL、截图、分析页面效果",-1))])])]),e.createCommentVNode(" 系统信息 "),e.createElementVNode("div",Kl,[s[23]||(s[23]=e.createElementVNode("div",{class:"section-title"},"系统信息",-1)),o.value?(e.openBlock(),e.createElementBlock("div",no,"加载中...")):(e.openBlock(),e.createElementBlock("div",Ql,[e.createElementVNode("div",Xl,[s[19]||(s[19]=e.createElementVNode("span",{class:"info-label"},"主机名",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.hostname),1)]),e.createElementVNode("div",Yl,[s[20]||(s[20]=e.createElementVNode("span",{class:"info-label"},"操作系统",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.os),1)]),e.createElementVNode("div",_l,[s[21]||(s[21]=e.createElementVNode("span",{class:"info-label"},"工作区",-1)),e.createElementVNode("span",vl,e.toDisplayString(l.value.workspace),1)]),e.createElementVNode("div",eo,[s[22]||(s[22]=e.createElementVNode("span",{class:"info-label"},"平台信息",-1)),e.createElementVNode("span",null,e.toDisplayString(l.value.goos),1)])]))])])]),e.createCommentVNode(" 底部 "),e.createElementVNode("div",to,[r.showHelpBtn?(e.openBlock(),e.createElementBlock("button",{key:0,class:"btn-primary",onClick:s[1]||(s[1]=p=>a.$emit("openHelp"))},[e.createVNode($,{name:"book-open",size:14}),s[24]||(s[24]=e.createTextVNode(" 查看帮助文档 ",-1))])):e.createCommentVNode("v-if",!0),e.createElementVNode("button",{class:"btn-secondary",onClick:s[2]||(s[2]=p=>a.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-e61d5ebd"]]),lo={class:"toast-container"},oo={class:"dlg-box",style:{"max-width":"400px"}},ao={class:"dlg-title"},so={class:"dlg-body"},io={class:"dlg-actions"},co={class:"dlg-box",style:{"max-width":"420px"}},po={class:"dlg-title"},mo={class:"dlg-body",style:{display:"flex","flex-direction":"column",gap:"8px"}},go={style:{"font-size":"13px",color:"var(--text-secondary)"}},ko=["placeholder"],ho={class:"dlg-actions"},fo={class:"dlg-box",style:{"max-width":"400px"}},yo={class:"dlg-title"},uo={class:"dlg-body",style:{"white-space":"pre-line"}},Eo=U({__name:"GlobalDialogs",setup(r){const t=e.ref(null);e.watch(()=>E.dialogState.show,o=>{o&&E.dialogState.type==="prompt"&&e.nextTick(()=>{var a,s;(a=t.value)==null||a.focus(),(s=t.value)==null||s.select()})});function n(){if(E.dialogState.type==="prompt"){const o=E.dialogState.inputValue;E.dialogState.show=!1,E.dialogState.resolve&&E.dialogState.resolve(o)}else if(E.dialogState.type==="confirm"&&E.dialogState.checkboxLabel){const a=E.dialogState.checkboxValue;E.dialogState.show=!1,E.dialogState.resolve&&E.dialogState.resolve({confirmed:!0,checked:a})}else E.dialogState.show=!1,E.dialogState.resolve&&E.dialogState.resolve(!0);E.dialogState.resolve=null}function l(){E.dialogState.type==="confirm"&&E.dialogState.checkboxLabel?(E.dialogState.show=!1,E.dialogState.resolve&&E.dialogState.resolve({confirmed:!1,checked:E.dialogState.checkboxValue})):E.dialogState.type==="prompt"?(E.dialogState.show=!1,E.dialogState.resolve&&E.dialogState.resolve(null)):(E.dialogState.show=!1,E.dialogState.resolve&&E.dialogState.resolve(!1)),E.dialogState.resolve=null}return(o,a)=>(e.openBlock(),e.createElementBlock(e.Fragment,null,[e.createCommentVNode(" Toast 通知区域 "),e.createElementVNode("div",lo,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(e.unref(E.dialogState).toasts,s=>(e.openBlock(),e.createElementBlock("div",{key:s.id,class:e.normalizeClass(["toast-item","toast-"+(s.type||"info")])},e.toDisplayString(s.message),3))),128))]),e.createCommentVNode(" Confirm 对话框 "),e.unref(E.dialogState).show&&e.unref(E.dialogState).type==="confirm"?(e.openBlock(),e.createElementBlock("div",{key:0,class:"dlg-overlay",onClick:e.withModifiers(l,["self"])},[e.createElementVNode("div",oo,[e.createElementVNode("div",ao,e.toDisplayString(e.unref(E.dialogState).title),1),e.createElementVNode("div",so,e.toDisplayString(e.unref(E.dialogState).message),1),e.unref(E.dialogState).checkboxLabel?(e.openBlock(),e.createElementBlock("label",{key:0,class:"dlg-checkbox",onClick:a[1]||(a[1]=e.withModifiers(()=>{},["stop"]))},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":a[0]||(a[0]=s=>e.unref(E.dialogState).checkboxValue=s)},null,512),[[e.vModelCheckbox,e.unref(E.dialogState).checkboxValue]]),e.createElementVNode("span",null,e.toDisplayString(e.unref(E.dialogState).checkboxLabel),1)])):e.createCommentVNode("v-if",!0),e.createElementVNode("div",io,[e.createElementVNode("button",{class:"dlg-btn",onClick:l},e.toDisplayString(e.unref(E.dialogState).cancelText),1),e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},e.toDisplayString(e.unref(E.dialogState).confirmText),1)])])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" Prompt 对话框 "),e.unref(E.dialogState).show&&e.unref(E.dialogState).type==="prompt"?(e.openBlock(),e.createElementBlock("div",{key:1,class:"dlg-overlay",onClick:e.withModifiers(l,["self"])},[e.createElementVNode("div",co,[e.createElementVNode("div",po,e.toDisplayString(e.unref(E.dialogState).title),1),e.createElementVNode("div",mo,[e.createElementVNode("span",go,e.toDisplayString(e.unref(E.dialogState).message),1),e.withDirectives(e.createElementVNode("input",{ref_key:"promptInputRef",ref:t,"onUpdate:modelValue":a[2]||(a[2]=s=>e.unref(E.dialogState).inputValue=s),placeholder:e.unref(E.dialogState).inputPlaceholder,class:"dlg-input",onKeyup:[e.withKeys(n,["enter"]),e.withKeys(l,["escape"])]},null,40,ko),[[e.vModelText,e.unref(E.dialogState).inputValue]])]),e.createElementVNode("div",ho,[e.createElementVNode("button",{class:"dlg-btn",onClick:l},e.toDisplayString(e.unref(E.dialogState).cancelText),1),e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},e.toDisplayString(e.unref(E.dialogState).confirmText),1)])])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" Alert 信息框 "),e.unref(E.dialogState).show&&e.unref(E.dialogState).type==="alert"?(e.openBlock(),e.createElementBlock("div",{key:2,class:"dlg-overlay",onClick:e.withModifiers(n,["self"])},[e.createElementVNode("div",fo,[e.createElementVNode("div",yo,e.toDisplayString(e.unref(E.dialogState).title),1),e.createElementVNode("div",uo,e.toDisplayString(e.unref(E.dialogState).message),1),e.createElementVNode("div",{class:"dlg-actions"},[e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},"确定")])])])):e.createCommentVNode("v-if",!0)],64))}},[["__scopeId","data-v-0271e4ae"]]),bo=U({__name:"UiModals",setup(r){const t=e.ref(null);let n=null;function l(){E.showAbout.value=!1,E.showHelp.value=!0,E.helpDocTarget.value="getting-started"}function o(){E.showHelp.value=!1,E.showAbout.value=!0}return e.onMounted(()=>{n=Ve.mountListSlot(t,"overlay",{isActive:a=>Ve.isOverlayActive("overlay",a)})}),e.onUnmounted(()=>{n&&(n(),n=null)}),(a,s)=>(e.openBlock(),e.createElementBlock(e.Fragment,null,[e.unref(E.showSettings)?(e.openBlock(),e.createBlock(nr,{key:0,onClose:s[0]||(s[0]=p=>E.showSettings.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(E.showSystem)?(e.openBlock(),e.createBlock(kr,{key:1,onClose:s[1]||(s[1]=p=>E.showSystem.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(E.showSource)?(e.openBlock(),e.createBlock(ur,{key:2,onClose:s[2]||(s[2]=p=>E.showSource.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(E.showHelp)?(e.openBlock(),e.createBlock(Rl,{key:3,onClose:s[3]||(s[3]=p=>E.showHelp.value=!1),onOpenAbout:o,initialDoc:e.unref(E.helpDocTarget)},null,8,["initialDoc"])):e.createCommentVNode("v-if",!0),e.unref(E.showAbout)?(e.openBlock(),e.createBlock(ro,{key:4,onClose:s[4]||(s[4]=p=>E.showAbout.value=!1),onOpenHelp:l})):e.createCommentVNode("v-if",!0),e.createVNode(Eo),e.createCommentVNode(" ★ overlay 槽位（list 型）：插件注册的浮动层条目叠加渲染（badge/toast/status pill 等） "),e.createElementVNode("div",{ref_key:"overlaySlotEl",ref:t,class:"plugin-overlay-host"},null,512)],64))}},[["__scopeId","data-v-519b8494"]]);function xo(r){const t=e.createApp(bo);return t.mount(r),()=>{t.unmount()}}return W.mount=xo,Object.defineProperty(W,Symbol.toStringTag,{value:"Module"}),W})({},window.__PAIRCODE_CORE.Vue,window.__PAIRCODE_CORE.uiState,window.__PAIRCODE_CORE.pluginRuntime,window.__PAIRCODE_CORE.api);
