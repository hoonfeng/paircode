var UiModals=(function(j,e,y,xe,A){"use strict";var xo=Object.defineProperty;var bo=(j,e,y)=>e in j?xo(j,e,{enumerable:!0,configurable:!0,writable:!0,value:y}):j[e]=y;var I=(j,e,y)=>bo(j,typeof e!="symbol"?e+"":e,y);var se;const O=(r,t)=>{const n=r.__vccOpts||r;for(const[o,l]of t)n[o]=l;return n},Fe=["width","height"],ze={key:0,d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"},P=O({__name:"SvgIcon",props:{name:{type:String,required:!0},size:{type:Number,default:16}},setup(r){return(t,n)=>(e.openBlock(),e.createElementBlock("svg",{class:"svg-icon",width:r.size,height:r.size,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor","stroke-width":"2","stroke-linecap":"round","stroke-linejoin":"round"},[e.createCommentVNode(" Folder "),r.name==="folder"?(e.openBlock(),e.createElementBlock("path",ze)):r.name==="folder-open"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" Folder Open "),n[0]||(n[0]=e.createElementVNode("path",{d:"M6 17l-3-9h18l-3 9H6z"},null,-1)),n[1]||(n[1]=e.createElementVNode("path",{d:"M4 8V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v3"},null,-1))],64)):r.name==="file"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" File "),n[2]||(n[2]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[3]||(n[3]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1))],64)):r.name==="file-code"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createCommentVNode(" File Code "),n[4]||(n[4]=e.createStaticVNode('<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" data-v-faf69761></path><polyline points="14 2 14 8 20 8" data-v-faf69761></polyline><line x1="10" y1="12" x2="8" y2="14" data-v-faf69761></line><line x1="10" y1="16" x2="8" y2="18" data-v-faf69761></line><line x1="14" y1="12" x2="16" y2="14" data-v-faf69761></line><line x1="14" y1="16" x2="16" y2="18" data-v-faf69761></line>',6))],64)):r.name==="file-text"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:4},[e.createCommentVNode(" File Text / Document "),n[5]||(n[5]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[6]||(n[6]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[7]||(n[7]=e.createElementVNode("line",{x1:"9",y1:"13",x2:"15",y2:"13"},null,-1)),n[8]||(n[8]=e.createElementVNode("line",{x1:"9",y1:"17",x2:"15",y2:"17"},null,-1))],64)):r.name==="search"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:5},[e.createCommentVNode(" Search "),n[9]||(n[9]=e.createElementVNode("circle",{cx:"11",cy:"11",r:"8"},null,-1)),n[10]||(n[10]=e.createElementVNode("line",{x1:"21",y1:"21",x2:"16.65",y2:"16.65"},null,-1))],64)):r.name==="terminal"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:6},[e.createCommentVNode(" Terminal / Console "),n[11]||(n[11]=e.createElementVNode("polyline",{points:"4 17 10 11 4 5"},null,-1)),n[12]||(n[12]=e.createElementVNode("line",{x1:"12",y1:"19",x2:"20",y2:"19"},null,-1))],64)):r.name==="chat"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:7},[e.createCommentVNode(" Chat / Message "),n[13]||(n[13]=e.createElementVNode("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"},null,-1))],64)):r.name==="settings"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:8},[e.createCommentVNode(" Gear / Settings "),n[14]||(n[14]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"3"},null,-1)),n[15]||(n[15]=e.createElementVNode("path",{d:"M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"},null,-1))],64)):r.name==="home"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:9},[e.createCommentVNode(" Home "),n[16]||(n[16]=e.createElementVNode("path",{d:"M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"},null,-1)),n[17]||(n[17]=e.createElementVNode("polyline",{points:"9 22 9 12 15 12 15 22"},null,-1))],64)):r.name==="chevron-right"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:10},[e.createCommentVNode(" Chevron Right "),n[18]||(n[18]=e.createElementVNode("polyline",{points:"9 6 15 12 9 18"},null,-1))],64)):r.name==="chevron-down"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:11},[e.createCommentVNode(" Chevron Down (Rotated chevron-right) "),n[19]||(n[19]=e.createElementVNode("polyline",{points:"6 9 12 15 18 9"},null,-1))],64)):r.name==="plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:12},[e.createCommentVNode(" Plus / Add "),n[20]||(n[20]=e.createElementVNode("line",{x1:"12",y1:"5",x2:"12",y2:"19"},null,-1)),n[21]||(n[21]=e.createElementVNode("line",{x1:"5",y1:"12",x2:"19",y2:"12"},null,-1))],64)):r.name==="close"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:13},[e.createCommentVNode(" Close / X "),n[22]||(n[22]=e.createElementVNode("line",{x1:"18",y1:"6",x2:"6",y2:"18"},null,-1)),n[23]||(n[23]=e.createElementVNode("line",{x1:"6",y1:"6",x2:"18",y2:"18"},null,-1))],64)):r.name==="refresh"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:14},[e.createCommentVNode(" Refresh "),n[24]||(n[24]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[25]||(n[25]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 1 1-2.12-9.36L23 10"},null,-1))],64)):r.name==="drive"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:15},[e.createCommentVNode(" Hard Drive / Disk "),n[26]||(n[26]=e.createElementVNode("line",{x1:"22",y1:"12",x2:"2",y2:"12"},null,-1)),n[27]||(n[27]=e.createElementVNode("path",{d:"M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"},null,-1)),n[28]||(n[28]=e.createElementVNode("line",{x1:"6",y1:"16",x2:"6.01",y2:"16"},null,-1)),n[29]||(n[29]=e.createElementVNode("line",{x1:"10",y1:"16",x2:"10.01",y2:"16"},null,-1))],64)):r.name==="source-control"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:16},[e.createCommentVNode(" Source Control / Git Branch "),n[30]||(n[30]=e.createElementVNode("line",{x1:"6",y1:"3",x2:"6",y2:"15"},null,-1)),n[31]||(n[31]=e.createElementVNode("circle",{cx:"18",cy:"6",r:"3"},null,-1)),n[32]||(n[32]=e.createElementVNode("circle",{cx:"6",cy:"18",r:"3"},null,-1)),n[33]||(n[33]=e.createElementVNode("path",{d:"M18 9a9 9 0 0 1-9 9"},null,-1))],64)):r.name==="git-branch"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:17},[e.createCommentVNode(" Git Branch "),n[34]||(n[34]=e.createElementVNode("line",{x1:"6",y1:"3",x2:"6",y2:"15"},null,-1)),n[35]||(n[35]=e.createElementVNode("circle",{cx:"18",cy:"6",r:"3"},null,-1)),n[36]||(n[36]=e.createElementVNode("circle",{cx:"6",cy:"18",r:"3"},null,-1)),n[37]||(n[37]=e.createElementVNode("path",{d:"M18 9a9 9 0 0 1-9 9"},null,-1))],64)):r.name==="git-pull"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:18},[e.createCommentVNode(" Git Pull "),n[38]||(n[38]=e.createStaticVNode('<circle cx="18" cy="18" r="3" data-v-faf69761></circle><circle cx="6" cy="6" r="3" data-v-faf69761></circle><path d="M13 6h3a2 2 0 0 1 2 2v7" data-v-faf69761></path><line x1="6" y1="18" x2="6" y2="9" data-v-faf69761></line><polyline points="9 9 6 6 3 9" data-v-faf69761></polyline>',5))],64)):r.name==="git-push"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:19},[e.createCommentVNode(" Git Push "),n[39]||(n[39]=e.createStaticVNode('<circle cx="18" cy="6" r="3" data-v-faf69761></circle><circle cx="6" cy="18" r="3" data-v-faf69761></circle><path d="M13 18h-2a2 2 0 0 1-2-2V9" data-v-faf69761></path><line x1="6" y1="6" x2="6" y2="15" data-v-faf69761></line><polyline points="9 15 6 18 3 15" data-v-faf69761></polyline>',5))],64)):r.name==="output"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:20},[e.createCommentVNode(" Output / Window "),n[40]||(n[40]=e.createElementVNode("rect",{x:"2",y:"3",width:"20",height:"14",rx:"2",ry:"2"},null,-1)),n[41]||(n[41]=e.createElementVNode("line",{x1:"8",y1:"21",x2:"16",y2:"21"},null,-1)),n[42]||(n[42]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12",y2:"21"},null,-1))],64)):r.name==="warning"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:21},[e.createCommentVNode(" Warning / Alert "),n[43]||(n[43]=e.createElementVNode("path",{d:"M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"},null,-1)),n[44]||(n[44]=e.createElementVNode("line",{x1:"12",y1:"9",x2:"12",y2:"13"},null,-1)),n[45]||(n[45]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"},null,-1))],64)):r.name==="undo"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:22},[e.createCommentVNode(" Undo "),n[46]||(n[46]=e.createElementVNode("polyline",{points:"1 4 1 10 7 10"},null,-1)),n[47]||(n[47]=e.createElementVNode("path",{d:"M3.51 15a9 9 0 1 0 2.13-9.36L1 10"},null,-1))],64)):r.name==="redo"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:23},[e.createCommentVNode(" Redo "),n[48]||(n[48]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[49]||(n[49]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 1 1-2.12-9.36L23 10"},null,-1))],64)):r.name==="package"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:24},[e.createCommentVNode(" Package / Box / Store "),n[50]||(n[50]=e.createElementVNode("line",{x1:"16.5",y1:"9.4",x2:"7.5",y2:"4.21"},null,-1)),n[51]||(n[51]=e.createElementVNode("path",{d:"M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"},null,-1)),n[52]||(n[52]=e.createElementVNode("polyline",{points:"3.27 6.96 12 12.01 20.73 6.96"},null,-1)),n[53]||(n[53]=e.createElementVNode("line",{x1:"12",y1:"22.08",x2:"12",y2:"12"},null,-1))],64)):r.name==="globe"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:25},[e.createCommentVNode(" Globe / External "),n[54]||(n[54]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[55]||(n[55]=e.createElementVNode("line",{x1:"2",y1:"12",x2:"22",y2:"12"},null,-1)),n[56]||(n[56]=e.createElementVNode("path",{d:"M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"},null,-1))],64)):r.name==="cycle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:26},[e.createCommentVNode(" Refresh / Cycle (for agent) "),n[57]||(n[57]=e.createElementVNode("polyline",{points:"23 4 23 10 17 10"},null,-1)),n[58]||(n[58]=e.createElementVNode("polyline",{points:"1 20 1 14 7 14"},null,-1)),n[59]||(n[59]=e.createElementVNode("path",{d:"M3.51 9a9 9 0 0 1 14.85-3.36L23 10"},null,-1)),n[60]||(n[60]=e.createElementVNode("path",{d:"M20.49 15a9 9 0 0 1-14.85 3.36L1 14"},null,-1))],64)):r.name==="send"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:27},[e.createCommentVNode(" Send (arrow up) "),n[61]||(n[61]=e.createElementVNode("line",{x1:"12",y1:"19",x2:"12",y2:"5"},null,-1)),n[62]||(n[62]=e.createElementVNode("polyline",{points:"5 12 12 5 19 12"},null,-1))],64)):r.name==="send-plane"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:28},[e.createCommentVNode(" Send Plane (paper airplane) "),n[63]||(n[63]=e.createElementVNode("line",{x1:"22",y1:"2",x2:"11",y2:"13"},null,-1)),n[64]||(n[64]=e.createElementVNode("polygon",{points:"22 2 15 22 11 13 2 9 22 2"},null,-1))],64)):r.name==="stop-dot"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:29},[e.createCommentVNode(" Stop Dot (pulsing circle) "),n[65]||(n[65]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"6",class:"stop-pulse"},null,-1)),n[66]||(n[66]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10",class:"stop-pulse-ring"},null,-1))],64)):r.name==="wrench"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:30},[e.createCommentVNode(" Wrench / Tool "),n[67]||(n[67]=e.createElementVNode("path",{d:"M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"},null,-1))],64)):r.name==="database"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:31},[e.createCommentVNode(" Database "),n[68]||(n[68]=e.createElementVNode("ellipse",{cx:"12",cy:"5",rx:"9",ry:"3"},null,-1)),n[69]||(n[69]=e.createElementVNode("path",{d:"M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"},null,-1)),n[70]||(n[70]=e.createElementVNode("path",{d:"M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"},null,-1))],64)):r.name==="user"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:32},[e.createCommentVNode(" User / Person "),n[71]||(n[71]=e.createElementVNode("path",{d:"M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"},null,-1)),n[72]||(n[72]=e.createElementVNode("circle",{cx:"12",cy:"7",r:"4"},null,-1))],64)):r.name==="info"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:33},[e.createCommentVNode(" Info "),n[73]||(n[73]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[74]||(n[74]=e.createElementVNode("line",{x1:"12",y1:"16",x2:"12",y2:"12"},null,-1)),n[75]||(n[75]=e.createElementVNode("line",{x1:"12",y1:"8",x2:"12.01",y2:"8"},null,-1))],64)):r.name==="lightbulb"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:34},[e.createCommentVNode(" Lightbulb / Suggestion "),n[76]||(n[76]=e.createElementVNode("path",{d:"M9 18h6"},null,-1)),n[77]||(n[77]=e.createElementVNode("path",{d:"M10 22h4"},null,-1)),n[78]||(n[78]=e.createElementVNode("path",{d:"M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14"},null,-1))],64)):r.name==="sparkles"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:35},[e.createCommentVNode(" Sparkles / Auto "),n[79]||(n[79]=e.createStaticVNode('<path d="M13.5 4L15 8l4 .5L15 12l1.5 4-4-2-4 2L10 12l-4-3.5L10 8z" data-v-faf69761></path><line x1="3" y1="18" x2="3" y2="21" data-v-faf69761></line><line x1="21" y1="18" x2="21" y2="21" data-v-faf69761></line><line x1="7" y1="20" x2="11" y2="20" data-v-faf69761></line><line x1="17" y1="20" x2="19" y2="20" data-v-faf69761></line>',5))],64)):r.name==="bot"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:36},[e.createCommentVNode(" Bot / AI "),n[80]||(n[80]=e.createStaticVNode('<rect x="3" y="11" width="18" height="10" rx="2" data-v-faf69761></rect><circle cx="12" cy="5" r="2" data-v-faf69761></circle><path d="M12 7v4" data-v-faf69761></path><line x1="8" y1="16" x2="8" y2="16" data-v-faf69761></line><line x1="16" y1="16" x2="16" y2="16" data-v-faf69761></line>',5))],64)):r.name==="file-js"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:37},[e.createCommentVNode(" File Type Icons "),n[81]||(n[81]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[82]||(n[82]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[83]||(n[83]=e.createElementVNode("text",{x:"8",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"JS",-1))],64)):r.name==="file-ts"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:38},[n[84]||(n[84]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[85]||(n[85]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[86]||(n[86]=e.createElementVNode("text",{x:"8",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"TS",-1))],64)):r.name==="file-go"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:39},[n[87]||(n[87]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[88]||(n[88]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[89]||(n[89]=e.createElementVNode("text",{x:"9",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Go",-1))],64)):r.name==="file-py"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:40},[n[90]||(n[90]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[91]||(n[91]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[92]||(n[92]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Py",-1))],64)):r.name==="file-java"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:41},[n[93]||(n[93]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[94]||(n[94]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[95]||(n[95]=e.createElementVNode("text",{x:"6",y:"17","font-size":"8",fill:"currentColor","font-weight":"bold",stroke:"none"},"Java",-1))],64)):r.name==="file-html"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:42},[n[96]||(n[96]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[97]||(n[97]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[98]||(n[98]=e.createElementVNode("text",{x:"6",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"HTML",-1))],64)):r.name==="file-css"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:43},[n[99]||(n[99]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[100]||(n[100]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[101]||(n[101]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"CSS",-1))],64)):r.name==="file-json"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:44},[n[102]||(n[102]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[103]||(n[103]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[104]||(n[104]=e.createElementVNode("text",{x:"5",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"{ }",-1))],64)):r.name==="file-md"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:45},[n[105]||(n[105]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[106]||(n[106]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[107]||(n[107]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"MD",-1))],64)):r.name==="file-vue"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:46},[n[108]||(n[108]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[109]||(n[109]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[110]||(n[110]=e.createElementVNode("text",{x:"7",y:"17","font-size":"9",fill:"currentColor","font-weight":"bold",stroke:"none"},"Vue",-1))],64)):r.name==="copy"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:47},[e.createCommentVNode(" Copy "),n[111]||(n[111]=e.createElementVNode("rect",{x:"9",y:"9",width:"13",height:"13",rx:"2",ry:"2"},null,-1)),n[112]||(n[112]=e.createElementVNode("path",{d:"M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"},null,-1))],64)):r.name==="minus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:48},[e.createCommentVNode(" Minus "),n[113]||(n[113]=e.createElementVNode("line",{x1:"5",y1:"12",x2:"19",y2:"12"},null,-1))],64)):r.name==="edit"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:49},[e.createCommentVNode(" Edit / Rename "),n[114]||(n[114]=e.createElementVNode("path",{d:"M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"},null,-1)),n[115]||(n[115]=e.createElementVNode("path",{d:"M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"},null,-1))],64)):r.name==="trash"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:50},[e.createCommentVNode(" Trash / Delete "),n[116]||(n[116]=e.createElementVNode("polyline",{points:"3 6 5 6 21 6"},null,-1)),n[117]||(n[117]=e.createElementVNode("path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"},null,-1))],64)):r.name==="file-plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:51},[e.createCommentVNode(" File Plus / New File "),n[118]||(n[118]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[119]||(n[119]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[120]||(n[120]=e.createElementVNode("line",{x1:"12",y1:"18",x2:"12",y2:"12"},null,-1)),n[121]||(n[121]=e.createElementVNode("line",{x1:"9",y1:"15",x2:"15",y2:"15"},null,-1))],64)):r.name==="message-square"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:52},[e.createCommentVNode(" Folder Plus / New Folder "),n[122]||(n[122]=e.createElementVNode("path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"},null,-1))],64)):r.name==="folder-plus"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:53},[n[123]||(n[123]=e.createElementVNode("path",{d:"M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v3"},null,-1)),n[124]||(n[124]=e.createElementVNode("line",{x1:"12",y1:"11",x2:"12",y2:"17"},null,-1)),n[125]||(n[125]=e.createElementVNode("line",{x1:"9",y1:"14",x2:"15",y2:"14"},null,-1))],64)):r.name==="brain"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:54},[e.createCommentVNode(" Brain / Thinking "),n[126]||(n[126]=e.createElementVNode("path",{d:"M12 2a4 4 0 0 0-4 4v1a5 5 0 0 0-5 5v1a4 4 0 0 0 3 3.87V17a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-.13A4 4 0 0 0 21 13v-1a5 5 0 0 0-5-5V6a4 4 0 0 0-4-4z"},null,-1)),n[127]||(n[127]=e.createElementVNode("path",{d:"M9 12v2"},null,-1)),n[128]||(n[128]=e.createElementVNode("path",{d:"M15 12v2"},null,-1)),n[129]||(n[129]=e.createElementVNode("path",{d:"M12 9v5"},null,-1))],64)):r.name==="check"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:55},[e.createCommentVNode(" Check / Success "),n[130]||(n[130]=e.createElementVNode("polyline",{points:"20 6 9 17 4 12"},null,-1))],64)):r.name==="clock"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:56},[e.createCommentVNode(" Clock / Pending "),n[131]||(n[131]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[132]||(n[132]=e.createElementVNode("polyline",{points:"12 6 12 12 16 14"},null,-1))],64)):r.name==="help"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:57},[e.createCommentVNode(" Help / Question "),n[133]||(n[133]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"10"},null,-1)),n[134]||(n[134]=e.createElementVNode("path",{d:"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"},null,-1)),n[135]||(n[135]=e.createElementVNode("line",{x1:"12",y1:"17",x2:"12.01",y2:"17"},null,-1))],64)):r.name==="shield"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:58},[e.createCommentVNode(" Shield / Approval "),n[136]||(n[136]=e.createElementVNode("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"},null,-1))],64)):r.name==="shield-off"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:59},[e.createCommentVNode(" Shield Off / No Review "),n[137]||(n[137]=e.createElementVNode("path",{d:"M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"},null,-1)),n[138]||(n[138]=e.createElementVNode("line",{x1:"4",y1:"4",x2:"20",y2:"20",stroke:"currentColor","stroke-width":"2","stroke-linecap":"round"},null,-1))],64)):r.name==="code"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:60},[e.createCommentVNode(" Code / Brackets "),n[139]||(n[139]=e.createElementVNode("polyline",{points:"16 18 22 12 16 6"},null,-1)),n[140]||(n[140]=e.createElementVNode("polyline",{points:"8 6 2 12 8 18"},null,-1))],64)):r.name==="list"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:61},[e.createCommentVNode(" List / Menu "),n[141]||(n[141]=e.createStaticVNode('<line x1="8" y1="6" x2="21" y2="6" data-v-faf69761></line><line x1="8" y1="12" x2="21" y2="12" data-v-faf69761></line><line x1="8" y1="18" x2="21" y2="18" data-v-faf69761></line><line x1="3" y1="6" x2="3.01" y2="6" data-v-faf69761></line><line x1="3" y1="12" x2="3.01" y2="12" data-v-faf69761></line><line x1="3" y1="18" x2="3.01" y2="18" data-v-faf69761></line>',6))],64)):r.name==="layers"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:62},[e.createCommentVNode(" Layers / Stack / Context "),n[142]||(n[142]=e.createElementVNode("polygon",{points:"12 2 2 7 12 12 22 7 12 2"},null,-1)),n[143]||(n[143]=e.createElementVNode("polyline",{points:"2 17 12 22 22 17"},null,-1)),n[144]||(n[144]=e.createElementVNode("polyline",{points:"2 12 12 17 22 12"},null,-1))],64)):r.name==="eye"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:63},[e.createCommentVNode(" Eye / Show "),n[145]||(n[145]=e.createElementVNode("path",{d:"M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"},null,-1)),n[146]||(n[146]=e.createElementVNode("circle",{cx:"12",cy:"12",r:"3"},null,-1))],64)):r.name==="eye-off"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:64},[e.createCommentVNode(" Eye Off / Hide "),n[147]||(n[147]=e.createElementVNode("path",{d:"M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"},null,-1)),n[148]||(n[148]=e.createElementVNode("path",{d:"M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"},null,-1)),n[149]||(n[149]=e.createElementVNode("line",{x1:"1",y1:"1",x2:"23",y2:"23"},null,-1))],64)):r.name==="bug"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:65},[e.createCommentVNode(" Bug "),n[150]||(n[150]=e.createStaticVNode('<rect x="8" y="2" width="8" height="4" rx="1" ry="1" data-v-faf69761></rect><path d="M20 12h-3a5 5 0 0 1-5 5 5 5 0 0 1-5-5H4" data-v-faf69761></path><path d="M4 8h16" data-v-faf69761></path><path d="M12 2v7" data-v-faf69761></path><path d="M9 17l-3 4" data-v-faf69761></path><path d="M15 17l3 4" data-v-faf69761></path>',6))],64)):r.name==="check-circle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:66},[e.createCommentVNode(" Check Circle "),n[151]||(n[151]=e.createElementVNode("path",{d:"M22 11.08V12a10 10 0 1 1-5.93-9.14"},null,-1)),n[152]||(n[152]=e.createElementVNode("polyline",{points:"22 4 12 14.01 9 11.01"},null,-1))],64)):r.name==="book-open"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:67},[e.createCommentVNode(" Book Open / Documentation "),n[153]||(n[153]=e.createElementVNode("path",{d:"M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"},null,-1)),n[154]||(n[154]=e.createElementVNode("path",{d:"M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"},null,-1))],64)):r.name==="tool"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:68},[e.createCommentVNode(" Tool / Wrench alternate "),n[155]||(n[155]=e.createElementVNode("path",{d:"M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"},null,-1))],64)):r.name==="keyboard"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:69},[e.createCommentVNode(" Keyboard "),n[156]||(n[156]=e.createStaticVNode('<rect x="2" y="4" width="20" height="16" rx="2" ry="2" data-v-faf69761></rect><line x1="6" y1="8" x2="6.01" y2="8" data-v-faf69761></line><line x1="10" y1="8" x2="10.01" y2="8" data-v-faf69761></line><line x1="14" y1="8" x2="14.01" y2="8" data-v-faf69761></line><line x1="18" y1="8" x2="18.01" y2="8" data-v-faf69761></line><line x1="6" y1="12" x2="6.01" y2="12" data-v-faf69761></line><line x1="10" y1="12" x2="10.01" y2="12" data-v-faf69761></line><line x1="14" y1="12" x2="14.01" y2="12" data-v-faf69761></line><line x1="18" y1="12" x2="18.01" y2="12" data-v-faf69761></line><line x1="6" y1="16" x2="18" y2="16" data-v-faf69761></line>',10))],64)):r.name==="chevron-left"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:70},[e.createCommentVNode(" Chevron Left "),n[157]||(n[157]=e.createElementVNode("polyline",{points:"15 6 9 12 15 18"},null,-1))],64)):r.name==="grid"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:71},[e.createCommentVNode(" Grid / App Grid "),n[158]||(n[158]=e.createElementVNode("rect",{x:"3",y:"3",width:"7",height:"7"},null,-1)),n[159]||(n[159]=e.createElementVNode("rect",{x:"14",y:"3",width:"7",height:"7"},null,-1)),n[160]||(n[160]=e.createElementVNode("rect",{x:"14",y:"14",width:"7",height:"7"},null,-1)),n[161]||(n[161]=e.createElementVNode("rect",{x:"3",y:"14",width:"7",height:"7"},null,-1))],64)):r.name==="puzzle"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:72},[e.createCommentVNode(" Puzzle / 插件 "),n[162]||(n[162]=e.createElementVNode("path",{d:"M4 7h3a2 2 0 0 1 4 0h9v9h-3a2 2 0 0 0-4 0H4z"},null,-1)),n[163]||(n[163]=e.createElementVNode("path",{d:"M11 7v9"},null,-1))],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:73},[n[164]||(n[164]=e.createElementVNode("path",{d:"M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"},null,-1)),n[165]||(n[165]=e.createElementVNode("polyline",{points:"14 2 14 8 20 8"},null,-1)),n[166]||(n[166]=e.createElementVNode("line",{x1:"9",y1:"13",x2:"15",y2:"13"},null,-1)),n[167]||(n[167]=e.createElementVNode("line",{x1:"9",y1:"17",x2:"15",y2:"17"},null,-1))],64))],8,Fe))}},[["__scopeId","data-v-faf69761"]]),Ue={class:"me-field"},He={class:"me-editor"},je={class:"me-input-row"},Ke=["onKeydown"],qe={class:"me-tags"},We={key:0,class:"me-empty"},Je=["onClick"],be=O({__name:"ModelEditor",props:{models:{type:Array,default:()=>[]}},emits:["change"],setup(r,{emit:t}){const n=r,o=t,l=e.ref(""),a=e.ref([...n.models]);e.watch(()=>n.models,f=>{a.value=[...f]});function s(){const f=l.value.split(/[\n,，]/).map(b=>b.trim()).filter(Boolean);let k=!1;for(const b of f)a.value.includes(b)||(a.value.push(b),k=!0);k&&o("change",[...a.value]),l.value=""}function d(f){const k=(f.clipboardData||window.clipboardData).getData("text");if(/[,\n，]/.test(k)){f.preventDefault();const b=k.split(/[\n,，]/).map(w=>w.trim()).filter(Boolean);let u=!1;for(const w of b)a.value.includes(w)||(a.value.push(w),u=!0);u&&o("change",[...a.value]),l.value=""}}function i(f){a.value.splice(f,1),o("change",[...a.value])}return(f,k)=>(e.openBlock(),e.createElementBlock("div",Ue,[k[1]||(k[1]=e.createElementVNode("span",{class:"me-label"},"可用模型（回车或逗号分隔添加；支持整段粘贴）",-1)),e.createElementVNode("div",He,[e.createElementVNode("div",je,[e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":k[0]||(k[0]=b=>l.value=b),class:"me-input",placeholder:"输入模型名，回车添加…",onKeydown:e.withKeys(e.withModifiers(s,["prevent"]),["enter"]),onPaste:d},null,40,Ke),[[e.vModelText,l.value]]),e.createElementVNode("button",{class:"me-btn",onClick:s},"添加")]),e.createElementVNode("div",qe,[a.value.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("span",We,"暂无模型——添加后 AI tab 的模型下拉会按服务商显示")),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,(b,u)=>(e.openBlock(),e.createElementBlock("span",{key:b+u,class:"me-tag"},[e.createTextVNode(e.toDisplayString(b)+" ",1),e.createElementVNode("button",{class:"me-x",title:"移除",onClick:w=>i(u)},"×",8,Je)]))),128))])])]))}},[["__scopeId","data-v-e3700bd7"]]),Ze={class:"provider-manager"},Qe={class:"pm-toolbar"},Xe={class:"pm-count"},Ye={key:0,class:"pm-edit"},_e={class:"pm-field"},ve={class:"pm-field"},en={class:"pm-field"},nn={class:"pm-field"},tn={class:"pm-params"},rn={key:0,class:"pm-param-rows"},ln=["title"],on=["onUpdate:modelValue"],an=["value"],sn=["onUpdate:modelValue"],cn=["value"],dn=["onUpdate:modelValue"],pn=["onUpdate:modelValue"],mn={key:1,class:"pm-params-empty"},gn={class:"pm-edit-actions"},kn=["disabled"],hn={key:1,class:"pm-cards"},fn={key:0,class:"pm-edit"},yn={class:"pm-edit-title"},un={class:"pm-field"},En=["value"],xn={class:"pm-field"},bn={class:"pm-field"},Vn={class:"pm-field"},Nn={class:"pm-params"},wn={key:0,class:"pm-param-rows"},Tn=["title"],Sn=["onUpdate:modelValue"],Cn=["value"],Bn=["onUpdate:modelValue"],In=["value"],An=["onUpdate:modelValue"],Pn=["onUpdate:modelValue"],Mn={key:1,class:"pm-params-empty"},$n={class:"pm-edit-actions"},Dn=["disabled"],Rn={key:1,class:"pm-card"},Gn={class:"pm-card-head"},Ln=["title"],On={class:"pm-ops"},Fn=["onClick"],zn=["onClick"],Un=["title"],Hn=["title"],jn={class:"pm-ctx"},Kn={class:"pm-models"},qn={key:0,class:"pm-none"},Wn={key:0,class:"pm-params-summary"},Jn={key:2,class:"pm-empty"},Zn={key:3,class:"pm-error"},Qn=O({__name:"ProviderManager",emits:["saved"],setup(r,{emit:t}){const n=t,o=e.ref([]),l=e.ref(""),a=e.ref({name:"",baseURL:"",apiKey:"",contextMaxTokens:0}),s=e.ref([]),d=e.ref({}),i=e.ref(""),f=e.ref(!1),k=[{v:"",label:"默认"},{v:"none",label:"none（关闭）"},{v:"minimal",label:"minimal（极简）"},{v:"low",label:"low（低）"},{v:"medium",label:"medium（中）"},{v:"high",label:"high（高）"},{v:"xhigh",label:"xhigh（超高）"},{v:"max",label:"max（最大化）"}],b=["","0","0.1","0.2","0.3","0.4","0.5","0.6","0.7","0.8","0.9","1.0","1.2","1.5","2.0"];function u(h){const p=y.state.settings&&y.state.settings.modelParams||{};return JSON.parse(JSON.stringify(p[h]||{}))}async function w(){try{const h=await A.getModels();o.value=(h.providers||[]).map(p=>({name:p,baseURL:(h.providerBaseURLs||{})[p]||"",apiKey:(h.providerKeys||{})[p]||"",contextMaxTokens:(h.providerContexts||{})[p]||0,models:(h.models||{})[p]||[]})),i.value=""}catch(h){i.value="加载服务商失败: "+(h.message||h)}}e.onMounted(w);function x(){l.value="__new__",a.value={name:"",baseURL:"",apiKey:"",contextMaxTokens:0},s.value=[],d.value={},i.value=""}function N(h){l.value=h.name,a.value={name:h.name,baseURL:h.baseURL,apiKey:h.apiKey||"",contextMaxTokens:h.contextMaxTokens||0},s.value=[...h.models||[]];const p=u(h.name);for(const m of s.value)p[m]||(p[m]={temperature:"",thinkingMode:"",maxTokens:0,contextMaxTokens:0});d.value=p,i.value=""}function T(h){const p={...d.value};for(const m of h)p[m]||(p[m]={temperature:"",thinkingMode:"",maxTokens:0,contextMaxTokens:0});for(const m of Object.keys(p))h.includes(m)||delete p[m];d.value=p,s.value=h}function L(){l.value="",i.value=""}function M(){const h={};for(const p of o.value)h[p.name]={baseURL:p.baseURL,models:p.models,apiKey:p.apiKey||"",contextMaxTokens:p.contextMaxTokens||0};return h}async function D(){const h=a.value.name.trim()||(l.value!=="__new__"?l.value:"");if(!h){i.value="服务商名称不能为空";return}const p=M();if(l.value==="__new__"&&p[h]){i.value=`服务商「${h}」已存在`;return}p[h]={baseURL:a.value.baseURL.trim(),models:s.value,apiKey:(a.value.apiKey||"").trim(),contextMaxTokens:Math.max(0,Number(a.value.contextMaxTokens)||0)},f.value=!0;try{await A.saveModels(p),await K(h),l.value="",await w(),n("saved")}catch(m){i.value="保存失败: "+(m.message||m)}finally{f.value=!1}}async function K(h){let p={};try{const g=await A.apiGet("/settings");p=g&&g.settings||{}}catch{}const m=JSON.parse(JSON.stringify(p.modelParams||{})),E={};for(const[g,V]of Object.entries(d.value)){const B=V||{},G={};B.temperature!==""&&B.temperature!==void 0&&B.temperature!==null&&(G.temperature=B.temperature),B.thinkingMode&&(G.thinkingMode=B.thinkingMode),Number(B.maxTokens)>0&&(G.maxTokens=Number(B.maxTokens)),Number(B.contextMaxTokens)>0&&(G.contextMaxTokens=Number(B.contextMaxTokens)),Object.keys(G).length&&(E[g]=G)}Object.keys(E).length?m[h]=E:delete m[h];const c={...p,modelParams:m};await A.apiPut("/settings",{settings:c,pluginSettings:p.pluginSettings||{}}),y.state.settings=c}async function U(h){if(!window.confirm(`删除服务商「${h.name}」？
（AI tab 将不再可选该服务商）`))return;const p=M();delete p[h.name];try{await A.saveModels(p);let m={};try{const c=await A.apiGet("/settings");m=c&&c.settings||{}}catch{}const E=JSON.parse(JSON.stringify(m.modelParams||{}));if(E[h.name]){delete E[h.name];const c={...m,modelParams:E};await A.apiPut("/settings",{settings:c,pluginSettings:m.pluginSettings||{}}),y.state.settings=c}await w(),n("saved")}catch(m){i.value="删除失败: "+(m.message||m)}}function R(h){const m=(y.state.settings&&y.state.settings.modelParams||{})[h]||{},E=Object.keys(m).length;return E?"模型参数已配置 "+E+" 个":""}return(h,p)=>(e.openBlock(),e.createElementBlock("div",Ze,[e.createCommentVNode(" 工具栏 "),e.createElementVNode("div",Qe,[e.createElementVNode("span",Xe,e.toDisplayString(o.value.length)+" 个服务商",1),e.createElementVNode("button",{class:"pm-btn pm-primary",onClick:x},"+ 新增服务商")]),e.createCommentVNode(" 新增表单（工具栏下方展开，紧邻按钮不跳动） "),l.value==="__new__"?(e.openBlock(),e.createElementBlock("div",Ye,[p[12]||(p[12]=e.createElementVNode("div",{class:"pm-edit-title"},"新增服务商",-1)),e.createElementVNode("div",_e,[p[7]||(p[7]=e.createElementVNode("span",{class:"pm-field-label"},"服务商名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[0]||(p[0]=m=>a.value.name=m),placeholder:"如 deepseek"},null,512),[[e.vModelText,a.value.name]])]),e.createElementVNode("div",ve,[p[8]||(p[8]=e.createElementVNode("span",{class:"pm-field-label"},"Base URL",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[1]||(p[1]=m=>a.value.baseURL=m),placeholder:"https://api.deepseek.com/v1"},null,512),[[e.vModelText,a.value.baseURL]])]),e.createElementVNode("div",en,[p[9]||(p[9]=e.createElementVNode("span",{class:"pm-field-label"},"API Key（该服务商独立保存）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[2]||(p[2]=m=>a.value.apiKey=m),type:"password",placeholder:"sk-…"},null,512),[[e.vModelText,a.value.apiKey]])]),e.createElementVNode("div",nn,[p[10]||(p[10]=e.createElementVNode("span",{class:"pm-field-label"},"上下文大小（Token）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[3]||(p[3]=m=>a.value.contextMaxTokens=m),type:"number",min:"0",step:"1000",placeholder:"0=不限制（模型级未配置时的默认窗口）"},null,512),[[e.vModelText,a.value.contextMaxTokens]])]),e.createVNode(be,{models:s.value,onChange:T},null,8,["models"]),e.createElementVNode("div",tn,[p[11]||(p[11]=e.createElementVNode("div",{class:"pm-params-title"},"模型参数（每模型独立配置；对话里也可临时切换思考档位）",-1)),s.value.length?(e.openBlock(),e.createElementBlock("div",rn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(s.value,m=>(e.openBlock(),e.createElementBlock("div",{key:m,class:"pm-param-row"},[e.createElementVNode("span",{class:"pm-param-model",title:m},e.toDisplayString(m),9,ln),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":E=>d.value[m].temperature=E,title:"温度（随机性）"},[(e.openBlock(),e.createElementBlock(e.Fragment,null,e.renderList(b,E=>e.createElementVNode("option",{key:"t"+E,value:E},e.toDisplayString(E===""?"温度默认":E),9,an)),64))],8,on),[[e.vModelSelect,d.value[m].temperature]]),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":E=>d.value[m].thinkingMode=E,title:"思考档位（OpenAI 定义）"},[(e.openBlock(),e.createElementBlock(e.Fragment,null,e.renderList(k,E=>e.createElementVNode("option",{key:"k"+E.v,value:E.v},e.toDisplayString(E.label),9,cn)),64))],8,sn),[[e.vModelSelect,d.value[m].thinkingMode]]),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":E=>d.value[m].maxTokens=E,type:"number",min:"0",step:"1024",placeholder:"输出 Token",title:"最大输出 Token（0=默认）"},null,8,dn),[[e.vModelText,d.value[m].maxTokens,void 0,{number:!0}]]),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":E=>d.value[m].contextMaxTokens=E,type:"number",min:"0",step:"4096",placeholder:"上下文",title:"上下文窗口（0=默认）"},null,8,pn),[[e.vModelText,d.value[m].contextMaxTokens,void 0,{number:!0}]])]))),128))])):(e.openBlock(),e.createElementBlock("div",mn,"添加模型后，可逐模型配置温度/思考档位/输出上限/上下文窗口"))]),e.createElementVNode("div",gn,[e.createElementVNode("button",{class:"pm-btn pm-primary",disabled:f.value,onClick:D},e.toDisplayString(f.value?"保存中…":"保存服务商"),9,kn),e.createElementVNode("button",{class:"pm-btn",onClick:L},"取消")])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 服务商卡片列表（编辑时在卡片位置就地展开表单，不跳顶） "),o.value.length?(e.openBlock(),e.createElementBlock("div",hn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(o.value,m=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:m.name},[l.value===m.name?(e.openBlock(),e.createElementBlock("div",fn,[e.createElementVNode("div",yn,"编辑服务商："+e.toDisplayString(m.name),1),e.createElementVNode("div",un,[p[13]||(p[13]=e.createElementVNode("span",{class:"pm-field-label"},"服务商名称",-1)),e.createElementVNode("input",{value:m.name,disabled:""},null,8,En)]),e.createElementVNode("div",xn,[p[14]||(p[14]=e.createElementVNode("span",{class:"pm-field-label"},"Base URL",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[4]||(p[4]=E=>a.value.baseURL=E),placeholder:"https://api.deepseek.com/v1"},null,512),[[e.vModelText,a.value.baseURL]])]),e.createElementVNode("div",bn,[p[15]||(p[15]=e.createElementVNode("span",{class:"pm-field-label"},"API Key（该服务商独立保存）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[5]||(p[5]=E=>a.value.apiKey=E),type:"password",placeholder:"sk-…"},null,512),[[e.vModelText,a.value.apiKey]])]),e.createElementVNode("div",Vn,[p[16]||(p[16]=e.createElementVNode("span",{class:"pm-field-label"},"上下文大小（Token）",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":p[6]||(p[6]=E=>a.value.contextMaxTokens=E),type:"number",min:"0",step:"1000",placeholder:"0=不限制（模型级未配置时的默认窗口）"},null,512),[[e.vModelText,a.value.contextMaxTokens]])]),e.createVNode(be,{models:s.value,onChange:T},null,8,["models"]),e.createElementVNode("div",Nn,[p[17]||(p[17]=e.createElementVNode("div",{class:"pm-params-title"},"模型参数（每模型独立配置；对话里也可临时切换思考档位）",-1)),s.value.length?(e.openBlock(),e.createElementBlock("div",wn,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(s.value,E=>(e.openBlock(),e.createElementBlock("div",{key:E,class:"pm-param-row"},[e.createElementVNode("span",{class:"pm-param-model",title:E},e.toDisplayString(E),9,Tn),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":c=>d.value[E].temperature=c,title:"温度（随机性）"},[(e.openBlock(),e.createElementBlock(e.Fragment,null,e.renderList(b,c=>e.createElementVNode("option",{key:"t"+c,value:c},e.toDisplayString(c===""?"温度默认":c),9,Cn)),64))],8,Sn),[[e.vModelSelect,d.value[E].temperature]]),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":c=>d.value[E].thinkingMode=c,title:"思考档位（OpenAI 定义）"},[(e.openBlock(),e.createElementBlock(e.Fragment,null,e.renderList(k,c=>e.createElementVNode("option",{key:"k"+c.v,value:c.v},e.toDisplayString(c.label),9,In)),64))],8,Bn),[[e.vModelSelect,d.value[E].thinkingMode]]),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":c=>d.value[E].maxTokens=c,type:"number",min:"0",step:"1024",placeholder:"输出 Token",title:"最大输出 Token（0=默认）"},null,8,An),[[e.vModelText,d.value[E].maxTokens,void 0,{number:!0}]]),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":c=>d.value[E].contextMaxTokens=c,type:"number",min:"0",step:"4096",placeholder:"上下文",title:"上下文窗口（0=默认）"},null,8,Pn),[[e.vModelText,d.value[E].contextMaxTokens,void 0,{number:!0}]])]))),128))])):(e.openBlock(),e.createElementBlock("div",Mn,"添加模型后，可逐模型配置温度/思考档位/输出上限/上下文窗口"))]),e.createElementVNode("div",$n,[e.createElementVNode("button",{class:"pm-btn pm-primary",disabled:f.value,onClick:D},e.toDisplayString(f.value?"保存中…":"保存服务商"),9,Dn),e.createElementVNode("button",{class:"pm-btn",onClick:L},"取消")])])):(e.openBlock(),e.createElementBlock("div",Rn,[e.createElementVNode("div",Gn,[e.createElementVNode("span",{class:"pm-name",title:m.name},e.toDisplayString(m.name),9,Ln),e.createElementVNode("div",On,[e.createElementVNode("button",{class:"pm-btn pm-small",onClick:E=>N(m)},"编辑",8,Fn),e.createElementVNode("button",{class:"pm-btn pm-small pm-danger",onClick:E=>U(m)},"删除",8,zn)])]),e.createElementVNode("div",{class:"pm-url",title:m.baseURL},e.toDisplayString(m.baseURL||"未配置 Base URL"),9,Un),e.createElementVNode("div",{class:e.normalizeClass(["pm-key",{"pm-key-ok":m.apiKey}]),title:m.apiKey?"已配置 API Key":"未配置 API Key"},e.toDisplayString(m.apiKey?"API Key 已配置":"未配置 API Key"),11,Hn),e.createElementVNode("div",jn,e.toDisplayString(m.contextMaxTokens>0?"上下文 "+(m.contextMaxTokens/1e3).toFixed(0)+"K Token":"上下文 未限制"),1),e.createElementVNode("div",Kn,[m.models.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("span",qn,"（未配置模型）")),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(m.models,E=>(e.openBlock(),e.createElementBlock("span",{key:E,class:"pm-tag"},e.toDisplayString(E),1))),128))]),R(m.name)?(e.openBlock(),e.createElementBlock("div",Wn,e.toDisplayString(R(m.name)),1)):e.createCommentVNode("v-if",!0)]))],64))),128))])):l.value!=="__new__"?(e.openBlock(),e.createElementBlock("div",Jn,"暂无服务商，点「+ 新增服务商」添加")):e.createCommentVNode("v-if",!0),i.value?(e.openBlock(),e.createElementBlock("div",Zn,e.toDisplayString(i.value),1)):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-ce1b9f02"]]),Xn={class:"pm-manager"},Yn={class:"mgm-toolbar"},_n={class:"mgm-count"},vn={key:0,class:"mgm-edit"},et={class:"mgm-edit-title"},nt={class:"mgm-field"},tt={class:"mgm-field"},rt=["value"],lt={class:"mgm-field"},ot={class:"mgm-field"},at={class:"mgm-field"},st=["value"],it={class:"mgm-edit-actions"},ct=["disabled"],dt={key:1,class:"mgm-cards"},pt={class:"mgm-card-head"},mt=["title"],gt={key:0,class:"pm-active-badge"},kt={class:"mgm-ops"},ht=["disabled","onClick"],ft=["onClick"],yt=["onClick"],ut={class:"pm-preview"},Et={class:"pm-snap-row"},xt={class:"pm-snap-row"},bt={class:"pm-snap-row"},Vt={key:2,class:"mgm-empty"},Nt={key:3,class:"mgm-error"},wt=O({__name:"PresetManager",emits:["saved"],setup(r,{expose:t,emit:n}){const o=n,l=e.ref({}),a=e.computed(()=>Object.keys(l.value||{})),s=e.ref(""),d=e.ref(!1),i=e.ref(""),f=e.ref(!1),k=e.ref(""),b=e.ref(""),u=e.ref(null),w=e.computed(()=>u.value&&u.value.providers||[]),x=e.computed(()=>(u.value&&u.value.models||{})[N.value.provider]||[]),N=e.ref({name:"",provider:"",baseURL:"",apiKey:"",executeModel:""});function T(c){b.value=c,setTimeout(()=>{b.value===c&&(b.value="")},4e3)}function L(c){return c?c.slice(0,10)+"…":"—"}async function M(){try{const[c,g,V]=await Promise.all([A.getAiPresets().catch(()=>({presets:{}})),A.apiGet("/settings").catch(()=>({settings:{}})),A.getModels().catch(()=>null)]);l.value=c&&c.presets||{},s.value=g&&g.settings&&g.settings.preset||"",u.value=V}catch(c){T("加载失败: "+(c.message||c))}}function D(c){const g=u.value||{};return{baseURL:g.providerBaseURLs&&g.providerBaseURLs[c]||"",apiKey:g.providerKeys&&g.providerKeys[c]||"",models:g.models&&g.models[c]||[]}}function K(){const c=window&&window.__PAIRCODE_CORE&&window.__PAIRCODE_CORE.uiState&&window.__PAIRCODE_CORE.uiState.state&&window.__PAIRCODE_CORE.uiState.state.settings||{};let g={};c.preset&&l.value&&l.value[c.preset]&&(g=l.value[c.preset]);const V=g.provider||c.provider||w.value[0]||"",B=D(V),G=B.models,Y=g.executeModel||c.executeModel||"";N.value={name:"",provider:V,baseURL:g.baseURL||c.baseURL||B.baseURL||"",apiKey:g.apiKey||c.apiKey||B.apiKey||"",executeModel:G.includes(Y)?Y:G[0]||""},i.value="",d.value=!0}function U(c){const g=l.value&&l.value[c]||{};N.value={name:c,provider:g.provider||"",baseURL:g.baseURL||"",apiKey:g.apiKey||"",executeModel:g.executeModel||""},i.value=c,d.value=!0}function R(){d.value=!1,i.value=""}function h(){if(!N.value.provider)return;const c=D(N.value.provider);N.value.baseURL=c.baseURL||"",N.value.apiKey=c.apiKey||"",N.value.executeModel=c.models.includes(N.value.executeModel)?N.value.executeModel:c.models[0]||""}e.watch(()=>N.value.provider,(c,g)=>{!d.value||g===""||c!==g&&h()});async function p(){const c=N.value.name.trim();if(!c){T("请输入配置名称");return}if(!N.value.provider){T("请选择服务商");return}if(!N.value.apiKey){T("请填写 API Key");return}f.value=!0,b.value="";try{const g={provider:N.value.provider,baseURL:N.value.baseURL,apiKey:N.value.apiKey,executeModel:N.value.executeModel};if(i.value&&i.value!==c){const V={...l.value||{}};V[c]=g,delete V[i.value];const B=await A.saveAiPresets(V);if(!(B&&B.ok)){T(B&&B.error||"保存失败");return}l.value=V,s.value===i.value&&(s.value=c,await A.apiPut("/settings",{settings:{preset:c},pluginSettings:{}}).catch(()=>{}))}else{const V=await A.saveAiPreset("save",c,g);if(!(V&&V.ok)){T(V&&V.error||"保存失败");return}l.value=V.presets||l.value}R(),o("saved")}catch(g){T("保存失败: "+(g.message||g))}finally{f.value=!1}}async function m(c){k.value=c,b.value="";try{const g=await A.saveAiPreset("apply",c);g&&g.ok?(s.value=c,o("saved")):T(g&&g.error||"应用失败")}catch(g){T("应用失败: "+(g.message||g))}finally{k.value=""}}async function E(c){if(confirm("删除配置「"+c+"」？")){b.value="";try{const g=await A.saveAiPreset("delete",c);g&&g.ok?(l.value=g.presets||l.value,s.value===c&&(s.value=""),o("saved")):T(g&&g.error||"删除失败")}catch(g){T("删除失败: "+(g.message||g))}}}return e.onMounted(M),t({load:M}),(c,g)=>(e.openBlock(),e.createElementBlock("div",Xn,[e.createCommentVNode(" 工具栏 "),e.createElementVNode("div",Yn,[e.createElementVNode("span",_n,e.toDisplayString(a.value.length)+" 个配置",1),e.createElementVNode("button",{class:"mgm-btn mgm-primary",onClick:K},"＋ 添加新配置")]),e.createCommentVNode(" 添加 / 编辑表单（点击添加/编辑才弹出） "),d.value?(e.openBlock(),e.createElementBlock("div",vn,[e.createElementVNode("div",et,e.toDisplayString(i.value?"编辑配置："+i.value:"添加新配置"),1),e.createElementVNode("div",nt,[g[5]||(g[5]=e.createElementVNode("span",{class:"mgm-field-label"},"配置名称",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":g[0]||(g[0]=V=>N.value.name=V),type:"text",placeholder:"如：主力 / 写作备用…",onKeydown:e.withKeys(p,["enter"])},null,544),[[e.vModelText,N.value.name]])]),e.createElementVNode("div",tt,[g[6]||(g[6]=e.createElementVNode("span",{class:"mgm-field-label"},"服务商",-1)),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":g[1]||(g[1]=V=>N.value.provider=V),class:"mgm-select",onChange:h},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(w.value,V=>(e.openBlock(),e.createElementBlock("option",{key:V,value:V},e.toDisplayString(V),9,rt))),128))],544),[[e.vModelSelect,N.value.provider]])]),e.createElementVNode("div",lt,[g[7]||(g[7]=e.createElementVNode("span",{class:"mgm-field-label"},"Base URL",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":g[2]||(g[2]=V=>N.value.baseURL=V),type:"text",placeholder:"https://api.deepseek.com/v1"},null,512),[[e.vModelText,N.value.baseURL]])]),e.createElementVNode("div",ot,[g[8]||(g[8]=e.createElementVNode("span",{class:"mgm-field-label"},"API Key",-1)),e.withDirectives(e.createElementVNode("input",{"onUpdate:modelValue":g[3]||(g[3]=V=>N.value.apiKey=V),type:"password",placeholder:"sk-…"},null,512),[[e.vModelText,N.value.apiKey]])]),e.createElementVNode("div",at,[g[9]||(g[9]=e.createElementVNode("span",{class:"mgm-field-label"},"模型",-1)),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":g[4]||(g[4]=V=>N.value.executeModel=V),class:"mgm-select"},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(x.value,V=>(e.openBlock(),e.createElementBlock("option",{key:V,value:V},e.toDisplayString(V),9,st))),128))],512),[[e.vModelSelect,N.value.executeModel]])]),e.createElementVNode("div",it,[e.createElementVNode("button",{class:"mgm-btn mgm-primary",disabled:f.value,onClick:p},e.toDisplayString(f.value?"保存中…":"保存配置"),9,ct),e.createElementVNode("button",{class:"mgm-btn",onClick:R},"取消")])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" 配置卡片列表（主视图） "),a.value.length?(e.openBlock(),e.createElementBlock("div",dt,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(a.value,V=>(e.openBlock(),e.createElementBlock("div",{key:V,class:e.normalizeClass(["mgm-card",{"pm-active":V===s.value}])},[e.createElementVNode("div",pt,[e.createElementVNode("span",{class:"mgm-name",title:V},[e.createTextVNode(e.toDisplayString(V),1),V===s.value?(e.openBlock(),e.createElementBlock("span",gt,"使用中")):e.createCommentVNode("v-if",!0)],8,mt),e.createElementVNode("div",kt,[e.createElementVNode("button",{class:"mgm-btn mgm-small",disabled:k.value===V,onClick:B=>m(V)},e.toDisplayString(k.value===V?"应用中…":"应用"),9,ht),e.createElementVNode("button",{class:"mgm-btn mgm-small",onClick:B=>U(V)},"编辑",8,ft),e.createElementVNode("button",{class:"mgm-btn mgm-small mgm-danger",onClick:B=>E(V)},"删除",8,yt)])]),e.createElementVNode("div",ut,[e.createElementVNode("div",Et,[g[10]||(g[10]=e.createElementVNode("span",null,"服务商",-1)),e.createElementVNode("b",null,e.toDisplayString((l.value[V]||{}).provider||"—"),1)]),e.createElementVNode("div",xt,[g[11]||(g[11]=e.createElementVNode("span",null,"模型",-1)),e.createElementVNode("b",null,e.toDisplayString((l.value[V]||{}).executeModel||"—"),1)]),e.createElementVNode("div",bt,[g[12]||(g[12]=e.createElementVNode("span",null,"API Key",-1)),e.createElementVNode("b",null,e.toDisplayString(L((l.value[V]||{}).apiKey)),1)])])],2))),128))])):d.value?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("div",Vt,"还没有 AI 配置。点「＋ 添加新配置」去设置模型和 Key，保存后即可应用。")),b.value?(e.openBlock(),e.createElementBlock("div",Nt,e.toDisplayString(b.value),1)):e.createCommentVNode("v-if",!0)]))}},[["__scopeId","data-v-2cccd6a1"]]),Tt={class:"modal-content"},St={class:"modal-body"},Ct={key:0,class:"settings-tabs"},Bt=["onClick"],It={class:"settings-content"},At={key:0},Pt={key:0,class:"group-title"},Mt=["title"],$t=["title"],Dt=["onUpdate:modelValue"],Rt=["title"],Gt={class:"field-control"},Lt=["type","onUpdate:modelValue","placeholder"],Ot=["onUpdate:modelValue","min","max","step"],Ft=["onUpdate:modelValue","onChange"],zt=["value"],Ut=["onUpdate:modelValue","placeholder"],Ht={class:"slider-row"},jt=["onUpdate:modelValue","min","max","step"],Kt={class:"slider-val"},qt={class:"color-row"},Wt=["onUpdate:modelValue"],Jt={class:"color-code"},Zt=["value","onInput","placeholder"],Qt=["placeholder"],Xt=["onUpdate:modelValue"],Yt={key:0,class:"setting-hint"},_t={key:0,class:"settings-empty"},vt=O({__name:"SettingsModal",emits:["close"],setup(r,{emit:t}){const n=t,o=e.ref(""),l=e.computed(()=>{const h=(y.state.pluginSchemas||[]).map(p=>({key:p.key,title:p.title||p.key,groups:a(p.fields||[])}));return h.length&&!o.value&&(o.value=h[0].key),h});function a(h){const p=[],m={};for(const E of h){const c=E.group||"";m[c]||(m[c]=[],p.push({title:c,fields:m[c]})),m[c].push(E)}return p}const s=e.ref(null);let d="";async function i(){try{s.value=await A.getModels()}catch{s.value=null}}function f(h){return h?(s.value&&s.value.models||{})[h]||[]:[]}function k(h,p){var m,E,c;if(p.optionsSource==="models"){const g=(m=u[h])==null?void 0:m[p.name],V=f((E=u.ai)==null?void 0:E.provider);return g&&!V.includes(g)?[...V,g]:V}if(p.optionsSource==="providers"){const g=s.value&&s.value.providers||[];if(g.length){const V=(c=u[h])==null?void 0:c[p.name];return V&&!g.includes(V)?[...g,V]:g}return p.options||[]}return p.options||[]}function b(h){if(!u.ai)return;const p=u.ai,m=h.linkFields||(h.linkField?[h.linkField]:[]);if(!m.length)return;const E=s.value||{},c=E.providerBaseURLs||{},g=E.providerKeys||{},V=p.provider,B=c[d];for(const G of m)if(G==="apiKey")p[G]=g[V]||"";else{const Y=p[G];(Y===void 0||Y===""||B&&Y===B)&&(p[G]=c[V]||"")}d=V}const u=e.reactive({}),w=e.ref("");function x(h){switch(h){case"checkbox":return!1;case"number":return 0;case"tags":return[];default:return""}}function N(){for(const E of Object.keys(u))delete u[E];const h=y.state.settings||{};d=h.provider||"";const p=h.pluginSettings||{};for(const E of y.state.pluginSchemas||[]){u[E.key]={};for(const c of E.fields||[]){let g;if(!(c.type==="project"||c.type==="provider-manager"||c.type==="model-params-manager"||c.type==="preset-manager")){if(c.binding)g=h[c.binding]!==void 0?h[c.binding]:c.default;else{const V=p[E.key]||{};g=V[c.name]!==void 0?V[c.name]:c.default}g===void 0&&(g=x(c.type)),c.type==="checkbox"&&(g=!!g),c.type==="number"&&(g=typeof g=="number"?g:Number(g)||0),c.type==="tags"&&(g=Array.isArray(g)?g:[]),u[E.key][c.name]=g}}}const m=(y.state.pluginSchemas||[]).some(E=>(E.fields||[]).some(c=>c.type==="project"));w.value="",m&&M()}function T(h,p){var E;const m=(E=u[h])==null?void 0:E[p.name];return Array.isArray(m)?m.join(", "):m||""}function L(h,p,m){u[h][p.name]=m.target.value.split(",").map(E=>E.trim()).filter(Boolean)}async function M(){try{const h=await A.getInstructions("project");w.value=h.content||""}catch{}}function D(){var h;N(),(h=y.state.settings)!=null&&h.theme&&y.applyTheme(y.state.settings.theme)}const K=()=>{D()};async function U(){try{const h=await A.apiGet("/settings");h&&h.settings&&(y.state.settings=h.settings,await i(),D())}catch{}}const R=async()=>{try{let h={};try{const c=await A.apiGet("/settings");h=c&&c.settings||{}}catch{}const p={...h},m={...h.pluginSettings||{}};let E=!1;for(const c of y.state.pluginSchemas||[]){const g=u[c.key]||{};for(const V of c.fields||[]){if(V.type==="project"){await A.saveInstructions("project",w.value);continue}if(V.type==="provider-manager"||V.type==="model-params-manager"||V.type==="preset-manager")continue;const B=g[V.name];V.binding?(V.name==="theme"&&B!==p[V.binding]&&(E=!0),p[V.binding]=B):(m[c.key]||(m[c.key]={}),m[c.key][V.name]=B)}}await A.apiPut("/settings",{settings:p,pluginSettings:m}),y.state.settings=p,E&&y.applyTheme(p.theme),window.$toast("设置已保存","success"),n("close")}catch(h){window.$toast("保存失败: "+h.message,"error")}};return e.onMounted(async()=>{await i(),D()}),(h,p)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:p[2]||(p[2]=e.withModifiers(m=>h.$emit("close"),["self"]))},[e.createElementVNode("div",Tt,[e.createElementVNode("h2",null,[e.createVNode(P,{name:"settings",size:18}),p[3]||(p[3]=e.createTextVNode(" 设置 ",-1)),e.createElementVNode("button",{class:"modal-close",onClick:p[0]||(p[0]=m=>h.$emit("close"))},"×")]),e.createElementVNode("div",St,[e.createCommentVNode(" ═══ 纯 schema 驱动：所有配置 tab 由插件 ctx.registerSettings 注册 ═══ "),l.value.length?(e.openBlock(),e.createElementBlock("div",Ct,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(l.value,m=>(e.openBlock(),e.createElementBlock("button",{key:m.key,class:e.normalizeClass(["settings-tab",{active:o.value===m.key}]),onClick:E=>o.value=m.key},e.toDisplayString(m.title),11,Bt))),128))])):e.createCommentVNode("v-if",!0),e.createElementVNode("div",It,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(l.value,m=>(e.openBlock(),e.createElementBlock(e.Fragment,{key:m.key},[o.value===m.key?(e.openBlock(),e.createElementBlock("div",At,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(m.groups,E=>(e.openBlock(),e.createElementBlock("div",{key:E.title||"__main",class:"setting-group"},[E.title?(e.openBlock(),e.createElementBlock("div",Pt,e.toDisplayString(E.title),1)):e.createCommentVNode("v-if",!0),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(E.fields,c=>(e.openBlock(),e.createElementBlock("div",{key:c.name,class:e.normalizeClass(["setting-row",{"row-toggle":c.type==="checkbox"}])},[e.createCommentVNode(" checkbox：label 与开关同行 "),c.type==="checkbox"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:0},[e.createElementVNode("label",{class:"field-label",title:c.hint},e.toDisplayString(c.label),9,Mt),e.createElementVNode("label",{class:"pp-switch",title:c.hint},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":g=>u[m.key][c.name]=g},null,8,Dt),[[e.vModelCheckbox,u[m.key][c.name]]]),p[4]||(p[4]=e.createElementVNode("span",{class:"pp-switch-track"},null,-1))],8,$t)],64)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" 其他类型：label 在上、控件在下、说明文字在控件下方（不挤占输入区） "),e.createElementVNode("label",{class:"field-label",title:c.hint},e.toDisplayString(c.label),9,Rt),e.createElementVNode("div",Gt,[e.createCommentVNode(" text / password "),c.type==="text"||c.type==="password"?e.withDirectives((e.openBlock(),e.createElementBlock("input",{key:0,class:"field-input",type:c.type==="password"?"password":"text","onUpdate:modelValue":g=>u[m.key][c.name]=g,placeholder:c.placeholder},null,8,Lt)),[[e.vModelDynamic,u[m.key][c.name]]]):c.type==="number"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:1},[e.createCommentVNode(" number "),e.withDirectives(e.createElementVNode("input",{class:"field-input",type:"number","onUpdate:modelValue":g=>u[m.key][c.name]=g,min:c.min,max:c.max,step:c.step},null,8,Ot),[[e.vModelText,u[m.key][c.name],void 0,{number:!0}]])],2112)):c.type==="select"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:2},[e.createCommentVNode(" select（optionsSource 驱动动态数据源：models=按服务商模型列表 / providers=服务商列表） "),e.withDirectives(e.createElementVNode("select",{"onUpdate:modelValue":g=>u[m.key][c.name]=g,class:"field-select",onChange:g=>b(c)},[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(k(m.key,c),g=>(e.openBlock(),e.createElementBlock("option",{key:g,value:g},e.toDisplayString(g),9,zt))),128))],40,Ft),[[e.vModelSelect,u[m.key][c.name]]])],2112)):c.type==="textarea"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:3},[e.createCommentVNode(" textarea "),e.withDirectives(e.createElementVNode("textarea",{"onUpdate:modelValue":g=>u[m.key][c.name]=g,class:"field-textarea",rows:"4",placeholder:c.placeholder},null,8,Ut),[[e.vModelText,u[m.key][c.name]]])],2112)):c.type==="slider"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:4},[e.createCommentVNode(" slider "),e.createElementVNode("div",Ht,[e.withDirectives(e.createElementVNode("input",{type:"range","onUpdate:modelValue":g=>u[m.key][c.name]=g,min:c.min!=null?c.min:0,max:c.max!=null?c.max:100,step:c.step||1},null,8,jt),[[e.vModelText,u[m.key][c.name],void 0,{number:!0}]]),e.createElementVNode("span",Kt,e.toDisplayString(u[m.key][c.name]),1)])],2112)):c.type==="color"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:5},[e.createCommentVNode(" color "),e.createElementVNode("div",qt,[e.withDirectives(e.createElementVNode("input",{type:"color","onUpdate:modelValue":g=>u[m.key][c.name]=g},null,8,Wt),[[e.vModelText,u[m.key][c.name]]]),e.createElementVNode("code",Jt,e.toDisplayString(u[m.key][c.name]),1)])],2112)):c.type==="tags"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:6},[e.createCommentVNode(" tags（逗号分隔数组） "),e.createElementVNode("input",{type:"text",class:"field-input",value:T(m.key,c),onInput:g=>L(m.key,c,g),placeholder:c.placeholder||"逗号分隔"},null,40,Zt)],2112)):c.type==="project"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:7},[e.createCommentVNode(" project（平台特殊：项目级指令，经 /api/instructions 读写） "),e.withDirectives(e.createElementVNode("textarea",{"onUpdate:modelValue":p[1]||(p[1]=g=>w.value=g),class:"field-textarea",rows:"4",placeholder:c.placeholder},null,8,Qt),[[e.vModelText,w.value]])],2112)):c.type==="provider-manager"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:8},[e.createCommentVNode(" provider-manager（服务商维护面板：CRUD /api/models，独立保存，不参与普通表单） "),e.createVNode(Qn,{onSaved:i})],2112)):c.type==="preset-manager"?(e.openBlock(),e.createElementBlock(e.Fragment,{key:9},[e.createCommentVNode(" preset-manager（AI 配置预设面板：CRUD /api/ai-presets，独立保存，不参与普通表单） "),e.createVNode(wt,{onSaved:U})],2112)):(e.openBlock(),e.createElementBlock(e.Fragment,{key:10},[e.createCommentVNode(" 兜底 text "),e.withDirectives(e.createElementVNode("input",{class:"field-input",type:"text","onUpdate:modelValue":g=>u[m.key][c.name]=g},null,8,Xt),[[e.vModelText,u[m.key][c.name]]])],2112))]),c.hint?(e.openBlock(),e.createElementBlock("span",Yt,e.toDisplayString(c.hint),1)):e.createCommentVNode("v-if",!0)],64))],2))),128))]))),128))])):e.createCommentVNode("v-if",!0)],64))),128)),l.value.length?e.createCommentVNode("v-if",!0):(e.openBlock(),e.createElementBlock("div",_t,"暂无配置项（等待插件注册…）"))])]),e.createElementVNode("div",{class:"modal-footer"},[e.createElementVNode("button",{class:"btn-secondary",onClick:K},"撤销"),e.createElementVNode("button",{class:"btn-primary",onClick:R},"保存设置")])])]))}},[["__scopeId","data-v-41699918"]]),er={class:"modal-content sys-modal"},nr={class:"modal-header"},tr={class:"modal-body"},rr={key:0,class:"loading"},lr={key:1,class:"sys-info"},or={class:"info-row"},ar={class:"info-row"},sr={class:"info-row"},ir={class:"info-row"},cr={class:"info-row"},dr={class:"info-row"},pr={class:"modal-footer"},mr=O({__name:"SystemModal",emits:["close"],setup(r,{emit:t}){const n=e.ref(!0),o=e.ref({});return e.onMounted(async()=>{try{o.value=await A.apiGet("/system/info")}catch{}n.value=!1}),(l,a)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:a[2]||(a[2]=e.withModifiers(s=>l.$emit("close"),["self"]))},[e.createElementVNode("div",er,[e.createElementVNode("div",nr,[a[3]||(a[3]=e.createElementVNode("h2",null,"ℹ 系统信息",-1)),e.createElementVNode("button",{class:"modal-close",onClick:a[0]||(a[0]=s=>l.$emit("close"))},"×")]),e.createElementVNode("div",tr,[n.value?(e.openBlock(),e.createElementBlock("div",rr,"加载中...")):(e.openBlock(),e.createElementBlock("div",lr,[e.createElementVNode("div",or,[a[4]||(a[4]=e.createElementVNode("label",null,"主机名",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.hostname),1)]),e.createElementVNode("div",ar,[a[5]||(a[5]=e.createElementVNode("label",null,"当前目录",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.cwd),1)]),e.createElementVNode("div",sr,[a[6]||(a[6]=e.createElementVNode("label",null,"操作系统",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.os),1)]),e.createElementVNode("div",ir,[a[7]||(a[7]=e.createElementVNode("label",null,"Go 版本",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.goos),1)]),e.createElementVNode("div",cr,[a[8]||(a[8]=e.createElementVNode("label",null,"工作区",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.workspace),1)]),e.createElementVNode("div",dr,[a[9]||(a[9]=e.createElementVNode("label",null,"文件夹",-1)),e.createElementVNode("span",null,e.toDisplayString((o.value.folders||[]).join(", ")),1)])]))]),e.createElementVNode("div",pr,[e.createElementVNode("button",{class:"btn-secondary",onClick:a[1]||(a[1]=s=>l.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-c27b6ec9"]]),gr={class:"modal-content source-modal"},kr={class:"modal-header"},hr={class:"modal-footer"},fr=O({__name:"SourceModal",emits:["close"],setup(r){return(t,n)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:n[2]||(n[2]=e.withModifiers(o=>t.$emit("close"),["self"]))},[e.createElementVNode("div",gr,[e.createElementVNode("div",kr,[n[3]||(n[3]=e.createElementVNode("h2",null,"⎔ 源代码管理",-1)),e.createElementVNode("button",{class:"modal-close",onClick:n[0]||(n[0]=o=>t.$emit("close"))},"×")]),n[4]||(n[4]=e.createElementVNode("div",{class:"modal-body"},[e.createElementVNode("p",{style:{color:"var(--text-muted)","text-align":"center","margin-top":"40px"}},[e.createTextVNode(" Git 集成开发中"),e.createElementVNode("br"),e.createElementVNode("br"),e.createTextVNode(" 功能规划："),e.createElementVNode("br"),e.createTextVNode(" · Git 状态查看"),e.createElementVNode("br"),e.createTextVNode(" · 暂存/提交/推送"),e.createElementVNode("br"),e.createTextVNode(" · 分支管理"),e.createElementVNode("br"),e.createTextVNode(" · Diff 对比 ")])],-1)),e.createElementVNode("div",hr,[e.createElementVNode("button",{class:"btn-secondary",onClick:n[1]||(n[1]=o=>t.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-2e060397"]]),yr=`# 功能介绍

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

**Agent 核心采用 deepseek-harness 双层循环架构**：
- **turn / step 双层边界** — 每次工具执行都有独立的 step 事件（开始/结束/摘要），每轮用户交互是 turn，进度颗粒度清晰可追溯
- **inbox 双队列** — 任务转向（next-step）与后续追问（next-turn）分队列消费，多轮交互不粘连
- **消息组装与落盘对齐** — agentloop 编号与消息序列严格一致，历史恢复与实时流状态吻合
- **历史注入精简** — 去掉冗余前缀标注与时间戳，系统提示内置多轮规则，长对话上下文更干净

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

所有 API 仅监听本地回环地址，安全可控。

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
- **内置市场** — 一键浏览和安装社区贡献的扩展（技能 / MCP / 插件工具集三类）

---

## 插件化自定义工具

**通过 JS / TS / Go / Lua 插件扩展 AI 的工具集，一切皆插件。**

PairCode IDE 的工具体系全部插件化——内置功能（文件/搜索/Git/Web/记忆/任务/图谱等 21 组）以插件形态装配，你也可以编写自己的插件扩展能力：

- **JS / TS 插件** — 通过 \`cordis_define\` 定义函数形态插件，支持 \`apply(ctx, config)\` 注入服务、timer 定时器、跨 goroutine 执行锁；TS 插件由内置编译器（esbuild 纯 Go）直接转译加载，无需 Node.js
- **Go 插件** — 内置插件框架，核心功能组全部以 Go 插件装配，\`cordis_inspect\` 可查看工具归属，\`cordis_stop\` 卸载整组
- **Lua 工具** — 支持 Lua 脚本自定义工具，封装常用命令组合与自定义数据处理逻辑
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
- **本地地址** — IDE 服务仅监听本地回环地址，默认不对外暴露

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
`,ur=`# API 文档\r
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
  "baseURL": "https://api.deepseek.com/v1",\r
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
`,Er=`# AI 工具文档

PairCode IDE 中的 AI 助手拥有丰富的内置能力，可以像你使用 IDE 一样操作文件、搜索代码、运行命令、管理版本。你只需用自然语言告诉 AI 你想做什么，AI 会自动选择合适的工具来完成任务。

所有工具对 AI 完全开放，你无需记忆工具名称——只需描述需求，AI 自动判断该用什么。

---

## 一、代码阅读与搜索

**浏览项目结构、搜索代码内容和定位符号定义，是 AI 理解你代码的基础能力。**

AI 可以像你一样阅读和浏览项目代码：

- 读取文件内容（可按行号范围读取部分内容）
- 列出目录下的文件和子目录
- 按关键词或正则表达式在文件内容中搜索
- 按通配符模式递归查找文件
- 搜索函数、类型、结构体等符号的定义位置
- 查看指定文件中所有检测到的符号
- 搜索某个符号在项目中的所有引用位置
- 列出项目中所有导出的公开符号
- 查看文件的导入依赖和反向依赖
- 分析修改某个文件后可能影响的其他文件
- 检测项目中的循环依赖

---

## 二、代码知识图谱 CodeGraph

**AI 能理解你的代码结构和调用关系，而不仅仅是搜索文本。**

CodeGraph 将项目的代码整体结构构建成可查询的知识图谱，让 AI 像理解知识一样理解你的代码：

- 构建或更新项目的代码知识图谱
- 查看知识图谱的统计信息
- 按名称查找函数或方法的定义位置和签名
- 获取结构体或接口的完整层次结构（字段、方法、嵌入类型）
- 查询哪些函数调用了指定的某个函数
- 查询某个函数内部调用了哪些其他函数
- 分析修改某个函数或类型后可能影响的范围
- 在知识图谱中按名称搜索代码实体
- 查询代码实体的 Git 变更历史

---

## 三、文件操作

**读写和编辑工作区内的文件，是 AI 帮你写代码的主要方式。**

AI 可以直接在工作区中进行文件操作：

- 将内容写入指定文件（覆盖模式，自动创建父目录）
- 精确替换文件中的一段文本
- 将文件或目录移动到新位置（也可用于重命名）
- 删除指定文件
- 将文件恢复到修改前的版本
- 查看某个文件的所有修改历史版本

---

## 四、命令执行

**在工作区中运行命令，AI 也能用命令行来完成任务。**

- 执行一条 shell 命令并等待结果返回
- 在后台启动一条长命令（如启动开发服务器）
- 读取后台进程累积的输出内容
- 停止正在运行的后台进程
- 直接执行一段代码（自动探测语言，写临时文件运行 Go / Python / Node.js 并返回结果）

---

## 五、网络与搜索

**AI 可以联网获取信息或搜索资料。**

- 抓取网页内容并提取纯文本
- 通过搜索引擎检索网络信息

---

## 六、网页验证与截图

**AI 可以打开网页、截图并分析页面内容，用于验证前端效果。**

- 在浏览器中打开网页，可输入文字、点击元素、检查控制台错误并截图
- 获取 JavaScript 渲染后的页面文本内容（适合单页应用）
- 截取桌面或指定窗口的屏幕
- 截取指定 URL 的网页

---

## 七、图像分析

**AI 可以"看"图片并理解其中的内容。**

- 读取图片文件内容（供支持视觉的模型直接理解图像）
- 分析图片中的颜色分布、色块区域和基本图形
- 从图片中识别文字，支持中英文混合识别

---

## 八、二进制分析

**查看和分析二进制文件的内容，用于逆向工程或文件格式分析。**

- 分析二进制文件的大小、类型和十六进制预览
- 将 Base64 编码的内容写入二进制文件
- 从二进制文件中提取可打印的字符串
- 在二进制文件中搜索指定的字节模式或文本
- 在二进制文件的指定位置写入字节补丁
- 解析可执行文件的结构（架构、入口、节区、导入导出）
- 计算文件的 MD5、SHA1、SHA256 哈希值
- 按块计算文件的香农熵（识别压缩或加密区域）

---

## 九、办公文档

**读写常见的办公文档格式，包括表格、文档和 PDF。**

- 读取 CSV 或 TSV 文件并以表格形式展示
- 将数据写入 CSV 或 TSV 文件
- 将 JSON 数组数据转为 Markdown 表格
- 对表格数据的数值列做统计（求和、均值、最大值等）
- 按文件扩展名分组统计代码行数
- 读取和生成 Word 文档
- 读取和创建 Excel 文件
- 提取 PDF 文件的文本内容（扫描型 PDF 自动进行 OCR 识别）
- 将 Markdown 文本转换为 HTML

---

## 十、Git 版本控制

**在对话中完成 Git 操作，AI 可以帮你管理代码版本。**

- 查看工作区的 Git 状态
- 查看文件的变更内容
- 查看最近的提交历史
- 查看某次提交的详情和改动
- 逐行查看文件的最后修改人和提交信息
- 将文件加入暂存区
- 提交已暂存的改动
- 列出、创建或删除分支
- 切换分支或恢复文件的修改
- 将工作区的改动暂存起来，稍后恢复

---

## 十一、调试器

**AI 可以启动调试会话，设置断点并检查程序运行状态。**

- 启动 Go 程序的调试会话
- 停止当前的调试会话
- 在指定文件的指定行设置断点
- 从暂停状态继续执行程序
- 单步跳过（不进入函数内部）
- 单步进入（进入函数调用内部）
- 单步跳出（执行到函数返回）
- 查看当前线程的调用栈
- 查看当前暂停点的变量值
- 在暂停状态下求值表达式
- 查看当前调试会话的状态

---

## 十二、项目知识库

**将项目架构、模块职责和设计决策记录下来，让 AI 跨会话了解你的项目。**

- 写入一条项目知识（如架构说明或设计决策）
- 读取某条项目知识的详细内容
- 列出知识库的所有条目概览
- 按关键词搜索知识库内容
- 删除某条项目知识
- 生成项目目录结构概览

---

## 十三、记忆系统

**AI 可以记住你的偏好、历史决策和项目约束，跨对话持续积累。**

- 写入一条持久记忆，AI 在后续对话中自动参考
- 读取某条记忆的详细内容
- 按关键词搜索已有记忆
- 列出所有历史记忆的摘要
- 删除一条过时的记忆
- 查询记忆库中的总条目数

---

## 十四、BUG 检测与修复

**AI 可以自动发现代码中的问题并给出修复方案。**

- 分析构建或测试的输出，提取错误位置和上下文
- 全量检测项目中的 BUG，自动运行编译和测试检查
- 自动检测 BUG 并生成修复方案，支持多次迭代修复

---

## 十五、任务与规划

**AI 可以追踪任务进度和执行计划，确保复杂的多步骤任务有条不紊。**

- 创建一个新的子任务并跟踪其状态
- 更新任务清单中各项任务的进度状态
- 维护和更新执行计划的步骤清单
- 任务全部完成后生成提交信息

---

## 十六、技能与 MCP 管理

**管理和扩展 AI 的能力——技能是工作流模板，MCP 是标准化的工具扩展协议。**

- 列出所有可用的技能及其激活模式
- 加载某个技能的完整内容供 AI 使用
- 加载技能的附加资源文件
- 创建或更新一个技能模板
- 删除一个项目级技能
- 列出已配置的 MCP 服务器
- 新增或删除 MCP 服务器扩展

---

## 十七、市场

**浏览和安装来自公共市场的技能和 MCP 扩展。**

- 在市场检索可安装的 MCP 服务器或技能
- 从市场安装指定的扩展

---

## 十八、插件管理

**管理 JS / TS / Go / Lua 插件——一切皆插件，自定义和扩展 AI 的工具集。**

- 定义一个函数形态的 JS/TS 插件（支持 apply(ctx, config) 注入服务、timer 定时器、跨 goroutine 执行锁）
- 查看已注册插件的详情（含工具归属：每个工具来自哪个插件，可整体卸载回收）
- 对插件执行查询（inspect 内部状态）
- 运行插件注册的服务或回调
- 列出 / 停止已注册的插件服务
- 撤销（undefine）一个已定义的插件
- 列出所有已创建的 Lua 自定义工具
- 创建一个新的 Lua 自定义工具
- 更新现有 Lua 工具的代码或参数
- 删除一个 Lua 自定义工具

---

## 十九、工具集管理

**按项目需求动态组合工具集，固化/导出/导入，构建处理本身也插件化。**

- 分析项目结构与需求，动态组合所需工具并创建工具集插件（固化到工作区 \`.pair/toolsets/\`）
- 列出当前项目可用的工具集
- 查看某个工具集的详细内容
- 导出工具集为 JSON（可提交 Git / 发布市场）
- 从 JSON 或文件导入工具集（project 工作区级 / user 全局级）
- 移除不再需要的工具集

---

## 二十、其他工具

**辅助性工具，在特定场景下帮助 AI 更好地与你协作。**

- **用户提问** — 当 AI 遇到关键决策点时，向你提问以澄清需求
- **任务委派** — 将复杂任务委托给子 AI 独立完成
- **资产清单** — 查看和使用已保存的经验胶囊和最佳实践
`,xr=`# 快捷键参考\r
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
`,br=`# 常见问题

## PairCode IDE 是什么？

PairCode IDE 是一款 AI 原生的纯 Web 集成开发环境。与传统 IDE 不同，你只需用浏览器打开，在对话面板中用自然语言描述需求，AI 就能理解你的意图，自动完成代码编写、文件操作、命令执行等工作——让编程从手工操作转变为对话驱动。

## 需要安装桌面客户端吗？

不需要。PairCode IDE 是纯 Web 应用，你只需启动后台服务，然后用浏览器（推荐 Chrome、Edge、Firefox）访问即可。所有界面在浏览器中渲染，无需安装任何桌面客户端。

## AI 能做什么？

AI 可以读写和编辑你的代码文件、在工作区中执行命令、搜索和浏览项目结构、管理 Git 版本控制、启动调试会话、处理图片和办公文档、搜索网络信息，还能截图验证网页效果。基本上，日常开发中你能做的事情，AI 都可以帮你完成。

## 如何让 AI 执行命令？

你可以在对话中直接告诉 AI 需要运行什么命令，例如"运行测试"或"启动项目"。AI 会自动在终端中执行并返回结果输出。涉及文件写入和命令执行的操作会先请求你的确认。

## 文件保存在哪里？

所有文件都保存在你本地的工作区目录中。PairCode IDE 直接读写你本地磁盘上的文件，不经过云端存储。你可以在文件浏览器中看到完整的项目目录结构，用系统的文件管理器也能找到它们。

## 如何切换 AI 模型？

在设置面板的"AI 模型"选项卡中，你可以选择不同的 AI 服务商和模型。支持接入 OpenAI、Claude 等多种主流模型后端。你可以为执行任务和制定规划分别配置不同的模型。

## 如何安装更多技能？

在市场中可以浏览和安装社区贡献的技能模板、MCP 扩展和工具集插件。技能是可复用的工作流程模板，MCP 扩展可以给 AI 添加新的能力，工具集是按项目需求组合的插件包（可通过 \`toolset_build\` 动态构建并固化到工作区）。打开市场面板，搜索你需要的功能，一键即可安装使用。

## 对话历史会丢失吗？

不会。每次对话都会自动保存在本地磁盘上，你可以随时在对话列表中查看历史记录、继续之前的对话或开启新话题。切换工作区时，各项目的对话会自动隔离，互不干扰。

## 如何保护隐私？

所有操作都在你的本地计算机上执行，代码和对话内容不会发送到外部服务器（AI 模型调用除外，你可以选择使用本地模型避免数据外出）。API 服务只监听本地回环地址，默认不对外暴露。文件操作限定在工作区范围内。

## 页面刷新后数据还在吗？

大部分数据都会保留：
- **对话历史** — 自动持久化到磁盘，刷新后完整恢复
- **打开的文件** — 刷新后自动重新打开
- **工作区状态** — 侧栏位置、面板大小等布局信息保存在浏览器中
- **设置** — 主题、AI 模型配置等设置持久化到磁盘

## 编辑器里的代码没有高亮怎么办？

编辑器会根据文件扩展名自动切换语言模式。如果文件扩展名不常见，代码高亮可能无法自动识别。建议确认文件扩展名是否被支持，或使用常见的扩展名保存文件。

## 什么是自主模式？和普通对话有什么区别？

**普通模式**：你发一条指令，AI 执行并回复，然后等待你下一条指令。

**自主模式**：你交给 AI 一个复杂任务（如"修复所有编译错误"），AI 会自动分解任务、逐个执行、迭代验证，直到全部完成。你不需要逐条发指令，只需在关键节点确认即可。

## 能让 AI 访问我的私有 API 吗？

可以通过 MCP（模型上下文协议）扩展来实现。在设置中添加自定义 MCP 服务器，AI 就能通过它访问你的私有 API、数据库或内部服务。

## 遇到问题怎么办？

你可以查看帮助菜单中的文档中心，里面有功能介绍、API 文档、工具文档和快捷键参考等详细资料。如果问题仍然无法解决，可以在对话中向 AI 描述你遇到的问题，它会尽力协助排查。
`,Vr=`# 快速开始

欢迎使用 PairCode IDE！以下指南将带你快速上手，从打开工作区到用 AI 写代码，只需几分钟。

---

## 打开 IDE

PairCode IDE 是一个纯 Web 应用，启动后台服务后，直接在浏览器中访问对应地址即可使用。所有界面在浏览器中渲染，无需安装任何桌面客户端。

> 建议使用 Chrome、Edge 或 Firefox 等现代浏览器获得最佳体验。

---

## 设置工作区

工作区是 IDE 操作的基础——所有文件操作、AI 对话和命令执行都将在这个目录范围内进行。

1. 点击左侧活动栏顶部的**文件图标**打开文件浏览器
2. 在文件浏览器顶部输入你的项目文件夹的完整路径
3. 按回车确认，IDE 会自动加载该目录下的所有文件和子目录

你也可以同时添加多个文件夹到同一个工作区中，方便跨目录浏览和管理代码。

---

## 与 AI 对话

右侧的**对话面板**是 PairCode IDE 的核心交互界面。你只需用自然语言描述需求，AI 就能理解并执行。

直接在输入框中输入你的需求，例如：

- "创建一个 Go 文件，实现一个返回 JSON 的 HTTP 服务"
- "帮我优化这个函数，加上错误处理和参数校验"
- "搜索项目中所有调用了 Post 的地方"
- "运行项目中的所有测试，并告诉我哪些失败了"
- "把我的改动提交到 Git"

按 Enter 发送消息，Shift+Enter 换行。AI 会实时流式展示它的思考过程、工具调用和结果输出。

### 常用对话技巧

| 技巧 | 说明 |
|------|------|
| **明确具体** | 越具体，AI 理解越准确。如"写一个函数"不如"写一个读取 JSON 配置文件的函数" |
| **分步沟通** | 复杂任务可以分步骤告诉 AI，先分析，再重构 |
| **提供上下文** | 在对话中粘贴错误信息或代码片段，AI 能给出更精准的修复方案 |
| **使用反馈** | 如果 AI 输出不满意，直接指出问题，AI 会调整方案重新尝试 |

---

## 编辑代码

AI 生成的代码会直接写入到文件中。你也可以在编辑器中手动查看和修改代码：

- **多标签页** — 同时打开多个文件，在标签栏切换
- **语法高亮** — 支持 Go、TypeScript、Python、Rust、Java、Vue 等主流语言
- **代码折叠** — 折叠函数和代码块，聚焦关键逻辑
- **Ctrl+S** — 保存当前文件的修改

你还可以在编辑器中查看二进制文件的十六进制内容，或直接预览图片文件。

---

## 运行与调试

### 使用内置终端

按 Ctrl+\\\` 打开 IDE 底部的终端面板，可以直接在工作区目录下运行命令。支持多标签页，方便在不同任务间切换。

### 让 AI 帮你运行

你也可以直接在对话中告诉 AI："运行项目并告诉我结果"或"执行 npm test"。AI 会自动在终端中执行命令、读取输出，并根据结果决定下一步操作。

---

## 版本控制

Git 操作完全融入 AI 对话流程。你用自然语言就能完成所有 Git 操作：

- "查看当前仓库状态"
- "暂存所有修改并提交"
- "创建一个新分支并切换过去"
- "从远程拉取最新代码"

你也可以通过左侧 Git 面板查看文件变更的详细对比，逐行确认每次改动的具体内容。

---

## 个性化设置

点击活动栏的**齿轮图标**打开设置面板，你可以：

- **AI 模型** — 选择不同的 AI 服务商和模型
- **外观主题** — 切换暗色、白色、暖色和暗夜紫四套主题
- **工作区管理** — 查看和切换最近使用的工作区
- **系统指令** — 自定义 AI 的行为指导原则

---

## 探索更多

PairCode IDE 还有更多强大功能等待你探索。欢迎查阅帮助文档中的其他章节：

- **功能介绍** — 了解所有功能模块的详细说明
- **工具文档** — 查看 AI 可使用的全部内置能力
- **快捷键参考** — 常用快捷键一览
- **API 文档** — 后端 HTTP API 接口说明
- **常见问题** — 常见问题与解答
`,Nr=`# 更新日志\r
\r
> 所有 PairCode IDE 的重要变更均记录在此文件中。\r
\r
---\r
\r
## 1.2.1 — 2026-08-15\r
\r
### 新增\r
- **按 deepseek-harness 设计重写 Agent 核心** — 双层循环（turn/step 边界事件、inbox 双队列对齐 next-step/next-turn），消息组装与落盘对齐 harness（agentloop 编号 ↔ 消息序列推导），系统提示精简为 harness 模式（\`WB_FULL_TOOLS=1\` 恢复全量工具）\r
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
`;function ie(){return{async:!1,breaks:!1,extensions:null,gfm:!0,hooks:null,pedantic:!1,renderer:null,silent:!1,tokenizer:null,walkTokens:null}}var W=ie();function Ve(r){W=r}var J={exec:()=>null};function Q(r){let t=[];return n=>{let o=Math.max(0,Math.min(3,n-1)),l=t[o];return l||(l=r(o),t[o]=l),l}}function S(r,t=""){let n=typeof r=="string"?r:r.source,o={replace:(l,a)=>{let s=typeof a=="string"?a:a.source;return s=s.replace($.caret,"$1"),n=n.replace(l,s),o},getRegex:()=>new RegExp(n,t)};return o}var wr=((r="")=>{try{return!!new RegExp("(?<=1)(?<!1)"+r)}catch{return!1}})(),$={codeRemoveIndent:/^(?: {1,4}| {0,3}\t)/gm,outputLinkReplace:/\\([\[\]])/g,indentCodeCompensation:/^(\s+)(?:```)/,beginningSpace:/^\s+/,endingHash:/#$/,startingSpaceChar:/^ /,endingSpaceChar:/ $/,nonSpaceChar:/[^ ]/,newLineCharGlobal:/\n/g,tabCharGlobal:/\t/g,multipleSpaceGlobal:/\s+/g,blankLine:/^[ \t]*$/,doubleBlankLine:/\n[ \t]*\n[ \t]*$/,blockquoteStart:/^ {0,3}>/,blockquoteSetextReplace:/\n {0,3}((?:=+|-+) *)(?=\n|$)/g,blockquoteSetextReplace2:/^ {0,3}>[ \t]?/gm,listReplaceNesting:/^ {1,4}(?=( {4})*[^ ])/g,listIsTask:/^\[[ xX]\] +\S/,listReplaceTask:/^\[[ xX]\] +/,listTaskCheckbox:/\[[ xX]\]/,anyLine:/\n.*\n/,hrefBrackets:/^<(.*)>$/,tableDelimiter:/[:|]/,tableAlignChars:/^\||\| *$/g,tableRowBlankLine:/\n[ \t]*$/,tableAlignRight:/^ *-+: *$/,tableAlignCenter:/^ *:-+: *$/,tableAlignLeft:/^ *:-+ *$/,startATag:/^<a /i,endATag:/^<\/a>/i,startPreScriptTag:/^<(pre|code|kbd|script)(\s|>)/i,endPreScriptTag:/^<\/(pre|code|kbd|script)(\s|>)/i,startAngleBracket:/^</,endAngleBracket:/>$/,pedanticHrefTitle:/^([^'"]*[^\s])\s+(['"])(.*)\2/,unicodeAlphaNumeric:/[\p{L}\p{N}]/u,escapeTest:/[&<>"']/,escapeReplace:/[&<>"']/g,escapeTestNoEncode:/[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/,escapeReplaceNoEncode:/[<>"']|&(?!(#\d{1,7}|#[Xx][a-fA-F0-9]{1,6}|\w+);)/g,caret:/(^|[^\[])\^/g,percentDecode:/%25/g,findPipe:/\|/g,splitPipe:/ \|/,slashPipe:/\\\|/g,carriageReturn:/\r\n|\r/g,spaceLine:/^ +$/gm,notSpaceStart:/^\S*/,endingNewline:/\n$/,listItemRegex:r=>new RegExp(`^( {0,3}${r})((?:[	 ][^\\n]*)?(?:\\n|$))`),nextBulletRegex:Q(r=>new RegExp(`^ {0,${r}}(?:[*+-]|\\d{1,9}[.)])((?:[ 	][^\\n]*)?(?:\\n|$))`)),hrRegex:Q(r=>new RegExp(`^ {0,${r}}((?:- *){3,}|(?:_ *){3,}|(?:\\* *){3,})(?:\\n+|$)`)),fencesBeginRegex:Q(r=>new RegExp(`^ {0,${r}}(?:\`\`\`|~~~)`)),headingBeginRegex:Q(r=>new RegExp(`^ {0,${r}}#`)),htmlBeginRegex:Q(r=>new RegExp(`^ {0,${r}}<(?:[a-z].*>|!--)`,"i")),blockquoteBeginRegex:Q(r=>new RegExp(`^ {0,${r}}>`))},Tr=/^(?:[ \t]*(?:\n|$))+/,Sr=/^((?: {4}| {0,3}\t)[^\n]+(?:\n(?:[ \t]*(?:\n|$))*)?)+/,Cr=/^ {0,3}(`{3,}(?=[^`\n]*(?:\n|$))|~{3,})([^\n]*)(?:\n|$)(?:|([\s\S]*?)(?:\n|$))(?: {0,3}\1[~`]* *(?=\n|$)|$)/,_=/^ {0,3}((?:-[\t ]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})(?:\n+|$)/,Br=/^ {0,3}(#{1,6})(?=\s|$)(.*)(?:\n+|$)/,ce=/ {0,3}(?:[*+-]|\d{1,9}[.)])/,Ne=/^(?!bull |blockCode|fences|blockquote|heading|html|table)((?:.|\n(?!\s*?\n|bull |blockCode|fences|blockquote|heading|html|table))+?)\n {0,3}(=+|-+) *(?:\n+|$)/,we=S(Ne).replace(/bull/g,ce).replace(/blockCode/g,/(?: {4}| {0,3}\t)/).replace(/fences/g,/ {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g,/ {0,3}>/).replace(/heading/g,/ {0,3}#{1,6}/).replace(/html/g,/ {0,3}<[^\n>]+>\n/).replace(/\|table/g,"").getRegex(),Ir=S(Ne).replace(/bull/g,ce).replace(/blockCode/g,/(?: {4}| {0,3}\t)/).replace(/fences/g,/ {0,3}(?:`{3,}|~{3,})/).replace(/blockquote/g,/ {0,3}>/).replace(/heading/g,/ {0,3}#{1,6}/).replace(/html/g,/ {0,3}<[^\n>]+>\n/).replace(/table/g,/ {0,3}\|?(?:[:\- ]*\|)+[\:\- ]*\n/).getRegex(),de=/^([^\n]+(?:\n(?!hr|heading|lheading|blockquote|fences|list|html|table| +\n)[^\n]+)*)/,Ar=/^[^\n]+/,pe=/(?!\s*\])(?:\\[\s\S]|[^\[\]\\])+/,Pr=S(/^ {0,3}\[(label)\]: *(?:\n[ \t]*)?([^<\s][^\s]*|<.*?>)(?:(?: +(?:\n[ \t]*)?| *\n[ \t]*)(title))? *(?:\n+|$)/).replace("label",pe).replace("title",/(?:"(?:\\"?|[^"\\])*"|'[^'\n]*(?:\n[^'\n]+)*\n?'|\([^()]*\))/).getRegex(),Mr=S(/^(bull)([ \t][^\n]*?)?(?:\n|$)/).replace(/bull/g,ce).getRegex(),ne="address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul",me=/<!--(?:-?>|[\s\S]*?(?:-->|$))/,$r=S("^ {0,3}(?:<(script|pre|style|textarea)[\\s>][\\s\\S]*?(?:</\\1>[^\\n]*\\n+|$)|comment[^\\n]*(\\n+|$)|<\\?[\\s\\S]*?(?:\\?>\\n*|$)|<![A-Z][\\s\\S]*?(?:>\\n*|$)|<!\\[CDATA\\[[\\s\\S]*?(?:\\]\\]>\\n*|$)|</?(tag)(?: +|\\n|/?>)[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|<(?!script|pre|style|textarea)([a-z][\\w-]*)(?:attribute)*? */?>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$)|</(?!script|pre|style|textarea)[a-z][\\w-]*\\s*>(?=[ \\t]*(?:\\n|$))[\\s\\S]*?(?:(?:\\n[ 	]*)+\\n|$))","i").replace("comment",me).replace("tag",ne).replace("attribute",/ +[a-zA-Z:_][\w.:-]*(?: *= *"[^"\n]*"| *= *'[^'\n]*'| *= *[^\s"'=<>`]+)?/).getRegex(),Te=S(de).replace("hr",_).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("|lheading","").replace("|table","").replace("blockquote"," {0,3}>").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",ne).getRegex(),Dr=S(/^( {0,3}> ?(paragraph|[^\n]*)(?:\n|$))+/).replace("paragraph",Te).getRegex(),ge={blockquote:Dr,code:Sr,def:Pr,fences:Cr,heading:Br,hr:_,html:$r,lheading:we,list:Mr,newline:Tr,paragraph:Te,table:J,text:Ar},Se=S("^ *([^\\n ].*)\\n {0,3}((?:\\| *)?:?-+:? *(?:\\| *:?-+:? *)*(?:\\| *)?)(?:\\n((?:(?! *\\n|hr|heading|blockquote|code|fences|list|html).*(?:\\n|$))*)\\n*|$)").replace("hr",_).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("blockquote"," {0,3}>").replace("code","(?: {4}| {0,3}	)[^\\n]").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",ne).getRegex(),Rr={...ge,lheading:Ir,table:Se,paragraph:S(de).replace("hr",_).replace("heading"," {0,3}#{1,6}(?:\\s|$)").replace("|lheading","").replace("table",Se).replace("blockquote"," {0,3}>").replace("fences"," {0,3}(?:`{3,}(?=[^`\\n]*\\n)|~{3,})[^\\n]*\\n").replace("list"," {0,3}(?:[*+-]|1[.)])[ \\t]+[^ \\t\\n]").replace("html","</?(?:tag)(?: +|\\n|/?>)|<(?:script|pre|style|textarea|!--)").replace("tag",ne).getRegex()},Gr={...ge,html:S(`^ *(?:comment *(?:\\n|\\s*$)|<(tag)[\\s\\S]+?</\\1> *(?:\\n{2,}|\\s*$)|<tag(?:"[^"]*"|'[^']*'|\\s[^'"/>\\s]*)*?/?> *(?:\\n{2,}|\\s*$))`).replace("comment",me).replace(/tag/g,"(?!(?:a|em|strong|small|s|cite|q|dfn|abbr|data|time|code|var|samp|kbd|sub|sup|i|b|u|mark|ruby|rt|rp|bdi|bdo|span|br|wbr|ins|del|img)\\b)\\w+(?!:|[^\\w\\s@]*@)\\b").getRegex(),def:/^ *\[([^\]]+)\]: *<?([^\s>]+)>?(?: +(["(][^\n]+[")]))? *(?:\n+|$)/,heading:/^(#{1,6})(.*)(?:\n+|$)/,fences:J,lheading:/^(.+?)\n {0,3}(=+|-+) *(?:\n+|$)/,paragraph:S(de).replace("hr",_).replace("heading",` *#{1,6} *[^
]`).replace("lheading",we).replace("|table","").replace("blockquote"," {0,3}>").replace("|fences","").replace("|list","").replace("|html","").replace("|tag","").getRegex()},Lr=/^\\([!"#$%&'()*+,\-./:;<=>?@\[\]\\^_`{|}~])/,Or=/^(`+)([^`]|[^`][\s\S]*?[^`])\1(?!`)/,Ce=/^( {2,}|\\)\n(?!\s*$)/,Fr=/^(`+|[^`])(?:(?= {2,}\n)|[\s\S]*?(?:(?=[\\<!\[`*_]|\b_|$)|[^ ](?= {2,}\n)))/,X=/[\p{P}\p{S}]/u,te=/[\s\p{P}\p{S}]/u,ke=/[^\s\p{P}\p{S}]/u,zr=S(/^((?![*_])punctSpace)/,"u").replace(/punctSpace/g,te).getRegex(),Be=/(?!~)[\p{P}\p{S}]/u,Ur=/(?!~)[\s\p{P}\p{S}]/u,Hr=/(?:[^\s\p{P}\p{S}]|~)/u,jr=S(/link|precode-code|html/,"g").replace("link",/\[(?:[^\[\]`]|(?<a>`+)[^`]+\k<a>(?!`))*?\]\((?:\\[\s\S]|[^\\\(\)]|\((?:\\[\s\S]|[^\\\(\)])*\))*\)/).replace("precode-",wr?"(?<!`)()":"(^^|[^`])").replace("code",/(?<b>`+)[^`]+\k<b>(?!`)/).replace("html",/<(?! )[^<>]*?>/).getRegex(),Ie=/^(?:\*+(?:((?!\*)punct)|([^\s*]))?)|^_+(?:((?!_)punct)|([^\s_]))?/,Kr=S(Ie,"u").replace(/punct/g,X).getRegex(),qr=S(Ie,"u").replace(/punct/g,Be).getRegex(),Ae="^[^_*]*?__[^_*]*?\\*[^_*]*?(?=__)|[^*]+(?=[^*])|(?!\\*)punct(\\*+)(?=[\\s]|$)|notPunctSpace(\\*+)(?!\\*)(?=punctSpace|$)|(?!\\*)punctSpace(\\*+)(?=notPunctSpace)|[\\s](\\*+)(?!\\*)(?=punct)|(?!\\*)punct(\\*+)(?!\\*)(?=punct)|notPunctSpace(\\*+)(?=notPunctSpace)",Wr=S(Ae,"gu").replace(/notPunctSpace/g,ke).replace(/punctSpace/g,te).replace(/punct/g,X).getRegex(),Jr=S(Ae,"gu").replace(/notPunctSpace/g,Hr).replace(/punctSpace/g,Ur).replace(/punct/g,Be).getRegex(),Zr=S("^[^_*]*?\\*\\*[^_*]*?_[^_*]*?(?=\\*\\*)|[^_]+(?=[^_])|(?!_)punct(_+)(?=[\\s]|$)|notPunctSpace(_+)(?!_)(?=punctSpace|$)|(?!_)punctSpace(_+)(?=notPunctSpace)|[\\s](_+)(?!_)(?=punct)|(?!_)punct(_+)(?!_)(?=punct)","gu").replace(/notPunctSpace/g,ke).replace(/punctSpace/g,te).replace(/punct/g,X).getRegex(),Qr=S(/^~~?(?:((?!~)punct)|[^\s~])/,"u").replace(/punct/g,X).getRegex(),Xr="^[^~]+(?=[^~])|(?!~)punct(~~?)(?=[\\s]|$)|notPunctSpace(~~?)(?!~)(?=punctSpace|$)|(?!~)punctSpace(~~?)(?=notPunctSpace)|[\\s](~~?)(?!~)(?=punct)|(?!~)punct(~~?)(?!~)(?=punct)|notPunctSpace(~~?)(?=notPunctSpace)",Yr=S(Xr,"gu").replace(/notPunctSpace/g,ke).replace(/punctSpace/g,te).replace(/punct/g,X).getRegex(),_r=S(/\\(punct)/,"gu").replace(/punct/g,X).getRegex(),vr=S(/^<(scheme:[^\s\x00-\x1f<>]*|email)>/).replace("scheme",/[a-zA-Z][a-zA-Z0-9+.-]{1,31}/).replace("email",/[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+(@)[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+(?![-_])/).getRegex(),el=S(me).replace("(?:-->|$)","-->").getRegex(),nl=S("^comment|^</[a-zA-Z][\\w:-]*\\s*>|^<[a-zA-Z][\\w-]*(?:attribute)*?\\s*/?>|^<\\?[\\s\\S]*?\\?>|^<![a-zA-Z]+\\s[\\s\\S]*?>|^<!\\[CDATA\\[[\\s\\S]*?\\]\\]>").replace("comment",el).replace("attribute",/\s+[a-zA-Z:_][\w.:-]*(?:\s*=\s*"[^"]*"|\s*=\s*'[^']*'|\s*=\s*[^\s"'=<>`]+)?/).getRegex(),re=/(?:\[(?:\\[\s\S]|[^\[\]\\])*\]|\\[\s\S]|`+(?!`)[^`]*?`+(?!`)|``+(?=\])|[^\[\]\\`])*?/,tl=S(/^!?\[(label)\]\(\s*(href)(?:(?:[ \t]+(?:\n[ \t]*)?|\n[ \t]*)(title))?\s*\)/).replace("label",re).replace("href",/<(?:\\.|[^\n<>\\])+>|[^ \t\n\x00-\x1f]*/).replace("title",/"(?:\\"?|[^"\\])*"|'(?:\\'?|[^'\\])*'|\((?:\\\)?|[^)\\])*\)/).getRegex(),Pe=S(/^!?\[(label)\]\[(ref)\]/).replace("label",re).replace("ref",pe).getRegex(),Me=S(/^!?\[(ref)\](?:\[\])?/).replace("ref",pe).getRegex(),rl=S("reflink|nolink(?!\\()","g").replace("reflink",Pe).replace("nolink",Me).getRegex(),$e=/[hH][tT][tT][pP][sS]?|[fF][tT][pP]/,he={_backpedal:J,anyPunctuation:_r,autolink:vr,blockSkip:jr,br:Ce,code:Or,del:J,delLDelim:J,delRDelim:J,emStrongLDelim:Kr,emStrongRDelimAst:Wr,emStrongRDelimUnd:Zr,escape:Lr,link:tl,nolink:Me,punctuation:zr,reflink:Pe,reflinkSearch:rl,tag:nl,text:Fr,url:J},ll={...he,link:S(/^!?\[(label)\]\((.*?)\)/).replace("label",re).getRegex(),reflink:S(/^!?\[(label)\]\s*\[([^\]]*)\]/).replace("label",re).getRegex()},fe={...he,emStrongRDelimAst:Jr,emStrongLDelim:qr,delLDelim:Qr,delRDelim:Yr,url:S(/^((?:protocol):\/\/|www\.)(?:[a-zA-Z0-9\-]+\.?)+[^\s<]*|^email/).replace("protocol",$e).replace("email",/[A-Za-z0-9._+-]+(@)[a-zA-Z0-9-_]+(?:\.[a-zA-Z0-9-_]*[a-zA-Z0-9])+(?![-_])/).getRegex(),_backpedal:/(?:[^?!.,:;*_'"~()&]+|\([^)]*\)|&(?![a-zA-Z0-9]+;$)|[?!.,:;*_'"~)]+(?!$))+/,del:/^(~~?)(?=[^\s~])((?:\\[\s\S]|[^\\])*?(?:\\[\s\S]|[^\s~\\]))\1(?=[^~]|$)/,text:S(/^([`~]+|[^`~])(?:(?= {2,}\n)|(?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)|[\s\S]*?(?:(?=[\\<!\[`*~_]|\b_|protocol:\/\/|www\.|$)|[^ ](?= {2,}\n)|[^a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-](?=[a-zA-Z0-9.!#$%&'*+\/=?_`{\|}~-]+@)))/).replace("protocol",$e).getRegex()},ol={...fe,br:S(Ce).replace("{2,}","*").getRegex(),text:S(fe.text).replace("\\b_","\\b_| {2,}\\n").replace(/\{2,\}/g,"*").getRegex()},le={normal:ge,gfm:Rr,pedantic:Gr},v={normal:he,gfm:fe,breaks:ol,pedantic:ll},al={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"},De=r=>al[r];function H(r,t){if(t){if($.escapeTest.test(r))return r.replace($.escapeReplace,De)}else if($.escapeTestNoEncode.test(r))return r.replace($.escapeReplaceNoEncode,De);return r}function Re(r){try{r=encodeURI(r).replace($.percentDecode,"%")}catch{return null}return r}function Ge(r,t){var a;let n=r.replace($.findPipe,(s,d,i)=>{let f=!1,k=d;for(;--k>=0&&i[k]==="\\";)f=!f;return f?"|":" |"}),o=n.split($.splitPipe),l=0;if(o[0].trim()||o.shift(),o.length>0&&!((a=o.at(-1))!=null&&a.trim())&&o.pop(),t)if(o.length>t)o.splice(t);else for(;o.length<t;)o.push("");for(;l<o.length;l++)o[l]=o[l].trim().replace($.slashPipe,"|");return o}function q(r,t,n){let o=r.length;if(o===0)return"";let l=0;for(;l<o&&r.charAt(o-l-1)===t;)l++;return r.slice(0,o-l)}function Le(r){let t=r.split(`
`),n=t.length-1;for(;n>=0&&$.blankLine.test(t[n]);)n--;return t.length-n<=2?r:t.slice(0,n+1).join(`
`)}function sl(r,t){if(r.indexOf(t[1])===-1)return-1;let n=0;for(let o=0;o<r.length;o++)if(r[o]==="\\")o++;else if(r[o]===t[0])n++;else if(r[o]===t[1]&&(n--,n<0))return o;return n>0?-2:-1}function il(r,t=0){let n=t,o="";for(let l of r)if(l==="	"){let a=4-n%4;o+=" ".repeat(a),n+=a}else o+=l,n++;return o}function Oe(r,t,n,o,l){let a=t.href,s=t.title||null,d=r[1].replace(l.other.outputLinkReplace,"$1");o.state.inLink=!0;let i={type:r[0].charAt(0)==="!"?"image":"link",raw:n,href:a,title:s,text:d,tokens:o.inlineTokens(d)};return o.state.inLink=!1,i}function cl(r,t,n){let o=r.match(n.other.indentCodeCompensation);if(o===null)return t;let l=o[1];return t.split(`
`).map(a=>{let s=a.match(n.other.beginningSpace);if(s===null)return a;let[d]=s;return d.length>=l.length?a.slice(l.length):a}).join(`
`)}var oe=class{constructor(r){I(this,"options");I(this,"rules");I(this,"lexer");this.options=r||W}space(r){let t=this.rules.block.newline.exec(r);if(t&&t[0].length>0)return{type:"space",raw:t[0]}}code(r){let t=this.rules.block.code.exec(r);if(t){let n=this.options.pedantic?t[0]:Le(t[0]),o=n.replace(this.rules.other.codeRemoveIndent,"");return{type:"code",raw:n,codeBlockStyle:"indented",text:o}}}fences(r){let t=this.rules.block.fences.exec(r);if(t){let n=t[0],o=cl(n,t[3]||"",this.rules);return{type:"code",raw:n,lang:t[2]?t[2].trim().replace(this.rules.inline.anyPunctuation,"$1"):t[2],text:o}}}heading(r){let t=this.rules.block.heading.exec(r);if(t){let n=t[2].trim();if(this.rules.other.endingHash.test(n)){let o=q(n,"#");(this.options.pedantic||!o||this.rules.other.endingSpaceChar.test(o))&&(n=o.trim())}return{type:"heading",raw:q(t[0],`
`),depth:t[1].length,text:n,tokens:this.lexer.inline(n)}}}hr(r){let t=this.rules.block.hr.exec(r);if(t)return{type:"hr",raw:q(t[0],`
`)}}blockquote(r){let t=this.rules.block.blockquote.exec(r);if(t){let n=q(t[0],`
`).split(`
`),o="",l="",a=[];for(;n.length>0;){let s=!1,d=[],i;for(i=0;i<n.length;i++)if(this.rules.other.blockquoteStart.test(n[i]))d.push(n[i]),s=!0;else if(!s)d.push(n[i]);else break;n=n.slice(i);let f=d.join(`
`),k=f.replace(this.rules.other.blockquoteSetextReplace,`
    $1`).replace(this.rules.other.blockquoteSetextReplace2,"");o=o?`${o}
${f}`:f,l=l?`${l}
${k}`:k;let b=this.lexer.state.top;if(this.lexer.state.top=!0,this.lexer.blockTokens(k,a,!0),this.lexer.state.top=b,n.length===0)break;let u=a.at(-1);if((u==null?void 0:u.type)==="code")break;if((u==null?void 0:u.type)==="blockquote"){let w=u,x=w.raw+`
`+n.join(`
`),N=this.blockquote(x);a[a.length-1]=N,o=o.substring(0,o.length-w.raw.length)+N.raw,l=l.substring(0,l.length-w.text.length)+N.text;break}else if((u==null?void 0:u.type)==="list"){let w=u,x=w.raw+`
`+n.join(`
`),N=this.list(x);a[a.length-1]=N,o=o.substring(0,o.length-u.raw.length)+N.raw,l=l.substring(0,l.length-w.raw.length)+N.raw,n=x.substring(a.at(-1).raw.length).split(`
`);continue}}return{type:"blockquote",raw:o,tokens:a,text:l}}}list(r){let t=this.rules.block.list.exec(r);if(t){let n=t[1].trim(),o=n.length>1,l={type:"list",raw:"",ordered:o,start:o?+n.slice(0,-1):"",loose:!1,items:[]};n=o?`\\d{1,9}\\${n.slice(-1)}`:`\\${n}`,this.options.pedantic&&(n=o?n:"[*+-]");let a=this.rules.other.listItemRegex(n),s=!1;for(;r;){let i=!1,f="",k="";if(!(t=a.exec(r))||this.rules.block.hr.test(r))break;f=t[0],r=r.substring(f.length);let b=il(t[2].split(`
`,1)[0],t[1].length),u=r.split(`
`,1)[0],w=!b.trim(),x=0;if(this.options.pedantic?(x=2,k=b.trimStart()):w?x=t[1].length+1:(x=b.search(this.rules.other.nonSpaceChar),x=x>4?1:x,k=b.slice(x),x+=t[1].length),w&&this.rules.other.blankLine.test(u)&&(f+=u+`
`,r=r.substring(u.length+1),i=!0),!i){let N=this.rules.other.nextBulletRegex(x),T=this.rules.other.hrRegex(x),L=this.rules.other.fencesBeginRegex(x),M=this.rules.other.headingBeginRegex(x),D=this.rules.other.htmlBeginRegex(x),K=this.rules.other.blockquoteBeginRegex(x);for(;r;){let U=r.split(`
`,1)[0],R;if(u=U,this.options.pedantic?(u=u.replace(this.rules.other.listReplaceNesting,"  "),R=u):R=u.replace(this.rules.other.tabCharGlobal,"    "),L.test(u)||M.test(u)||D.test(u)||K.test(u)||N.test(u)||T.test(u))break;if(R.search(this.rules.other.nonSpaceChar)>=x||!u.trim())k+=`
`+R.slice(x);else{if(w||b.replace(this.rules.other.tabCharGlobal,"    ").search(this.rules.other.nonSpaceChar)>=4||L.test(b)||M.test(b)||T.test(b))break;k+=`
`+u}w=!u.trim(),f+=U+`
`,r=r.substring(U.length+1),b=R.slice(x)}}l.loose||(s?l.loose=!0:this.rules.other.doubleBlankLine.test(f)&&(s=!0)),l.items.push({type:"list_item",raw:f,task:!!this.options.gfm&&this.rules.other.listIsTask.test(k),loose:!1,text:k,tokens:[]}),l.raw+=f}let d=l.items.at(-1);if(d)d.raw=d.raw.trimEnd(),d.text=d.text.trimEnd();else return;l.raw=l.raw.trimEnd();for(let i of l.items){this.lexer.state.top=!1,i.tokens=this.lexer.blockTokens(i.text,[]);let f=i.tokens[0];if(i.task&&((f==null?void 0:f.type)==="text"||(f==null?void 0:f.type)==="paragraph")){i.text=i.text.replace(this.rules.other.listReplaceTask,""),f.raw=f.raw.replace(this.rules.other.listReplaceTask,""),f.text=f.text.replace(this.rules.other.listReplaceTask,"");for(let b=this.lexer.inlineQueue.length-1;b>=0;b--)if(this.rules.other.listIsTask.test(this.lexer.inlineQueue[b].src)){this.lexer.inlineQueue[b].src=this.lexer.inlineQueue[b].src.replace(this.rules.other.listReplaceTask,"");break}let k=this.rules.other.listTaskCheckbox.exec(i.raw);if(k){let b={type:"checkbox",raw:k[0]+" ",checked:k[0]!=="[ ]"};i.checked=b.checked,l.loose?i.tokens[0]&&["paragraph","text"].includes(i.tokens[0].type)&&"tokens"in i.tokens[0]&&i.tokens[0].tokens?(i.tokens[0].raw=b.raw+i.tokens[0].raw,i.tokens[0].text=b.raw+i.tokens[0].text,i.tokens[0].tokens.unshift(b)):i.tokens.unshift({type:"paragraph",raw:b.raw,text:b.raw,tokens:[b]}):i.tokens.unshift(b)}}else i.task&&(i.task=!1);if(!l.loose){let k=i.tokens.filter(u=>u.type==="space"),b=k.length>0&&k.some(u=>this.rules.other.anyLine.test(u.raw));l.loose=b}}if(l.loose)for(let i of l.items){i.loose=!0;for(let f of i.tokens)f.type==="text"&&(f.type="paragraph")}return l}}html(r){let t=this.rules.block.html.exec(r);if(t){let n=Le(t[0]);return{type:"html",block:!0,raw:n,pre:t[1]==="pre"||t[1]==="script"||t[1]==="style",text:n}}}def(r){let t=this.rules.block.def.exec(r);if(t){let n=t[1].toLowerCase().replace(this.rules.other.multipleSpaceGlobal," "),o=t[2]?t[2].replace(this.rules.other.hrefBrackets,"$1").replace(this.rules.inline.anyPunctuation,"$1"):"",l=t[3]?t[3].substring(1,t[3].length-1).replace(this.rules.inline.anyPunctuation,"$1"):t[3];return{type:"def",tag:n,raw:q(t[0],`
`),href:o,title:l}}}table(r){var s;let t=this.rules.block.table.exec(r);if(!t||!this.rules.other.tableDelimiter.test(t[2]))return;let n=Ge(t[1]),o=t[2].replace(this.rules.other.tableAlignChars,"").split("|"),l=(s=t[3])!=null&&s.trim()?t[3].replace(this.rules.other.tableRowBlankLine,"").split(`
`):[],a={type:"table",raw:q(t[0],`
`),header:[],align:[],rows:[]};if(n.length===o.length){for(let d of o)this.rules.other.tableAlignRight.test(d)?a.align.push("right"):this.rules.other.tableAlignCenter.test(d)?a.align.push("center"):this.rules.other.tableAlignLeft.test(d)?a.align.push("left"):a.align.push(null);for(let d=0;d<n.length;d++)a.header.push({text:n[d],tokens:this.lexer.inline(n[d]),header:!0,align:a.align[d]});for(let d of l)a.rows.push(Ge(d,a.header.length).map((i,f)=>({text:i,tokens:this.lexer.inline(i),header:!1,align:a.align[f]})));return a}}lheading(r){let t=this.rules.block.lheading.exec(r);if(t){let n=t[1].trim();return{type:"heading",raw:q(t[0],`
`),depth:t[2].charAt(0)==="="?1:2,text:n,tokens:this.lexer.inline(n)}}}paragraph(r){let t=this.rules.block.paragraph.exec(r);if(t){let n=t[1].charAt(t[1].length-1)===`
`?t[1].slice(0,-1):t[1];return{type:"paragraph",raw:t[0],text:n,tokens:this.lexer.inline(n)}}}text(r){let t=this.rules.block.text.exec(r);if(t)return{type:"text",raw:t[0],text:t[0],tokens:this.lexer.inline(t[0])}}escape(r){let t=this.rules.inline.escape.exec(r);if(t)return{type:"escape",raw:t[0],text:t[1]}}tag(r){let t=this.rules.inline.tag.exec(r);if(t)return!this.lexer.state.inLink&&this.rules.other.startATag.test(t[0])?this.lexer.state.inLink=!0:this.lexer.state.inLink&&this.rules.other.endATag.test(t[0])&&(this.lexer.state.inLink=!1),!this.lexer.state.inRawBlock&&this.rules.other.startPreScriptTag.test(t[0])?this.lexer.state.inRawBlock=!0:this.lexer.state.inRawBlock&&this.rules.other.endPreScriptTag.test(t[0])&&(this.lexer.state.inRawBlock=!1),{type:"html",raw:t[0],inLink:this.lexer.state.inLink,inRawBlock:this.lexer.state.inRawBlock,block:!1,text:t[0]}}link(r){let t=this.rules.inline.link.exec(r);if(t){let n=t[2].trim();if(!this.options.pedantic&&this.rules.other.startAngleBracket.test(n)){if(!this.rules.other.endAngleBracket.test(n))return;let a=q(n.slice(0,-1),"\\");if((n.length-a.length)%2===0)return}else{let a=sl(t[2],"()");if(a===-2)return;if(a>-1){let s=(t[0].indexOf("!")===0?5:4)+t[1].length+a;t[2]=t[2].substring(0,a),t[0]=t[0].substring(0,s).trim(),t[3]=""}}let o=t[2],l="";if(this.options.pedantic){let a=this.rules.other.pedanticHrefTitle.exec(o);a&&(o=a[1],l=a[3])}else l=t[3]?t[3].slice(1,-1):"";return o=o.trim(),this.rules.other.startAngleBracket.test(o)&&(this.options.pedantic&&!this.rules.other.endAngleBracket.test(n)?o=o.slice(1):o=o.slice(1,-1)),Oe(t,{href:o&&o.replace(this.rules.inline.anyPunctuation,"$1"),title:l&&l.replace(this.rules.inline.anyPunctuation,"$1")},t[0],this.lexer,this.rules)}}reflink(r,t){let n;if((n=this.rules.inline.reflink.exec(r))||(n=this.rules.inline.nolink.exec(r))){let o=(n[2]||n[1]).replace(this.rules.other.multipleSpaceGlobal," "),l=t[o.toLowerCase()];if(!l){let a=n[0].charAt(0);return{type:"text",raw:a,text:a}}return Oe(n,l,n[0],this.lexer,this.rules)}}emStrong(r,t,n=""){let o=this.rules.inline.emStrongLDelim.exec(r);if(!(!o||!o[1]&&!o[2]&&!o[3]&&!o[4]||o[4]&&n.match(this.rules.other.unicodeAlphaNumeric))&&(!(o[1]||o[3])||!n||this.rules.inline.punctuation.exec(n))){let l=[...o[0]].length-1,a,s,d=l,i=0,f=o[0][0]==="*"?this.rules.inline.emStrongRDelimAst:this.rules.inline.emStrongRDelimUnd;for(f.lastIndex=0,t=t.slice(-1*r.length+l);(o=f.exec(t))!==null;){if(a=o[1]||o[2]||o[3]||o[4]||o[5]||o[6],!a)continue;if(s=[...a].length,o[3]||o[4]){d+=s;continue}else if((o[5]||o[6])&&l%3&&!((l+s)%3)){i+=s;continue}if(d-=s,d>0)continue;s=Math.min(s,s+d+i);let k=[...o[0]][0].length,b=r.slice(0,l+o.index+k+s);if(Math.min(l,s)%2){let w=b.slice(1,-1);return{type:"em",raw:b,text:w,tokens:this.lexer.inlineTokens(w)}}let u=b.slice(2,-2);return{type:"strong",raw:b,text:u,tokens:this.lexer.inlineTokens(u)}}}}codespan(r){let t=this.rules.inline.code.exec(r);if(t){let n=t[2].replace(this.rules.other.newLineCharGlobal," "),o=this.rules.other.nonSpaceChar.test(n),l=this.rules.other.startingSpaceChar.test(n)&&this.rules.other.endingSpaceChar.test(n);return o&&l&&(n=n.substring(1,n.length-1)),{type:"codespan",raw:t[0],text:n}}}br(r){let t=this.rules.inline.br.exec(r);if(t)return{type:"br",raw:t[0]}}del(r,t,n=""){let o=this.rules.inline.delLDelim.exec(r);if(o&&(!o[1]||!n||this.rules.inline.punctuation.exec(n))){let l=[...o[0]].length-1,a,s,d=l,i=this.rules.inline.delRDelim;for(i.lastIndex=0,t=t.slice(-1*r.length+l);(o=i.exec(t))!==null;){if(a=o[1]||o[2]||o[3]||o[4]||o[5]||o[6],!a||(s=[...a].length,s!==l))continue;if(o[3]||o[4]){d+=s;continue}if(d-=s,d>0)continue;s=Math.min(s,s+d);let f=[...o[0]][0].length,k=r.slice(0,l+o.index+f+s),b=k.slice(l,-l);return{type:"del",raw:k,text:b,tokens:this.lexer.inlineTokens(b)}}}}autolink(r){let t=this.rules.inline.autolink.exec(r);if(t){let n,o;return t[2]==="@"?(n=t[1],o="mailto:"+n):(n=t[1],o=n),{type:"link",raw:t[0],text:n,href:o,tokens:[{type:"text",raw:n,text:n}]}}}url(r){var n;let t;if(t=this.rules.inline.url.exec(r)){let o,l;if(t[2]==="@")o=t[0],l="mailto:"+o;else{let a;do a=t[0],t[0]=((n=this.rules.inline._backpedal.exec(t[0]))==null?void 0:n[0])??"";while(a!==t[0]);o=t[0],t[1]==="www."?l="http://"+t[0]:l=t[0]}return{type:"link",raw:t[0],text:o,href:l,tokens:[{type:"text",raw:o,text:o}]}}}inlineText(r){let t=this.rules.inline.text.exec(r);if(t){let n=this.lexer.state.inRawBlock;return{type:"text",raw:t[0],text:t[0],escaped:n}}}},F=class ue{constructor(t){I(this,"tokens");I(this,"options");I(this,"state");I(this,"inlineQueue");I(this,"tokenizer");this.tokens=[],this.tokens.links=Object.create(null),this.options=t||W,this.options.tokenizer=this.options.tokenizer||new oe,this.tokenizer=this.options.tokenizer,this.tokenizer.options=this.options,this.tokenizer.lexer=this,this.inlineQueue=[],this.state={inLink:!1,inRawBlock:!1,top:!0};let n={other:$,block:le.normal,inline:v.normal};this.options.pedantic?(n.block=le.pedantic,n.inline=v.pedantic):this.options.gfm&&(n.block=le.gfm,this.options.breaks?n.inline=v.breaks:n.inline=v.gfm),this.tokenizer.rules=n}static get rules(){return{block:le,inline:v}}static lex(t,n){return new ue(n).lex(t)}static lexInline(t,n){return new ue(n).inlineTokens(t)}lex(t){t=t.replace($.carriageReturn,`
`),this.blockTokens(t,this.tokens);for(let n=0;n<this.inlineQueue.length;n++){let o=this.inlineQueue[n];this.inlineTokens(o.src,o.tokens)}return this.inlineQueue=[],this.tokens}blockTokens(t,n=[],o=!1){var a,s,d;this.tokenizer.lexer=this,this.options.pedantic&&(t=t.replace($.tabCharGlobal,"    ").replace($.spaceLine,""));let l=1/0;for(;t;){if(t.length<l)l=t.length;else{this.infiniteLoopError(t.charCodeAt(0));break}let i;if((s=(a=this.options.extensions)==null?void 0:a.block)!=null&&s.some(k=>(i=k.call({lexer:this},t,n))?(t=t.substring(i.raw.length),n.push(i),!0):!1))continue;if(i=this.tokenizer.space(t)){t=t.substring(i.raw.length);let k=n.at(-1);i.raw.length===1&&k!==void 0?k.raw+=`
`:n.push(i);continue}if(i=this.tokenizer.code(t)){t=t.substring(i.raw.length);let k=n.at(-1);(k==null?void 0:k.type)==="paragraph"||(k==null?void 0:k.type)==="text"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.text,this.inlineQueue.at(-1).src=k.text):n.push(i);continue}if(i=this.tokenizer.fences(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.heading(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.hr(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.blockquote(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.list(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.html(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.def(t)){t=t.substring(i.raw.length);let k=n.at(-1);(k==null?void 0:k.type)==="paragraph"||(k==null?void 0:k.type)==="text"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.raw,this.inlineQueue.at(-1).src=k.text):this.tokens.links[i.tag]||(this.tokens.links[i.tag]={href:i.href,title:i.title},n.push(i));continue}if(i=this.tokenizer.table(t)){t=t.substring(i.raw.length),n.push(i);continue}if(i=this.tokenizer.lheading(t)){t=t.substring(i.raw.length),n.push(i);continue}let f=t;if((d=this.options.extensions)!=null&&d.startBlock){let k=1/0,b=t.slice(1),u;this.options.extensions.startBlock.forEach(w=>{u=w.call({lexer:this},b),typeof u=="number"&&u>=0&&(k=Math.min(k,u))}),k<1/0&&k>=0&&(f=t.substring(0,k+1))}if(this.state.top&&(i=this.tokenizer.paragraph(f))){let k=n.at(-1);o&&(k==null?void 0:k.type)==="paragraph"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.text,this.inlineQueue.pop(),this.inlineQueue.at(-1).src=k.text):n.push(i),o=f.length!==t.length,t=t.substring(i.raw.length);continue}if(i=this.tokenizer.text(t)){t=t.substring(i.raw.length);let k=n.at(-1);(k==null?void 0:k.type)==="text"?(k.raw+=(k.raw.endsWith(`
`)?"":`
`)+i.raw,k.text+=`
`+i.text,this.inlineQueue.pop(),this.inlineQueue.at(-1).src=k.text):n.push(i);continue}if(t){this.infiniteLoopError(t.charCodeAt(0));break}}return this.state.top=!0,n}inline(t,n=[]){return this.inlineQueue.push({src:t,tokens:n}),n}inlineTokens(t,n=[]){var f,k,b,u,w;this.tokenizer.lexer=this;let o=t,l=null;if(this.tokens.links){let x=Object.keys(this.tokens.links);if(x.length>0)for(;(l=this.tokenizer.rules.inline.reflinkSearch.exec(o))!==null;)x.includes(l[0].slice(l[0].lastIndexOf("[")+1,-1))&&(o=o.slice(0,l.index)+"["+"a".repeat(l[0].length-2)+"]"+o.slice(this.tokenizer.rules.inline.reflinkSearch.lastIndex))}for(;(l=this.tokenizer.rules.inline.anyPunctuation.exec(o))!==null;)o=o.slice(0,l.index)+"++"+o.slice(this.tokenizer.rules.inline.anyPunctuation.lastIndex);let a;for(;(l=this.tokenizer.rules.inline.blockSkip.exec(o))!==null;)a=l[2]?l[2].length:0,o=o.slice(0,l.index+a)+"["+"a".repeat(l[0].length-a-2)+"]"+o.slice(this.tokenizer.rules.inline.blockSkip.lastIndex);o=((k=(f=this.options.hooks)==null?void 0:f.emStrongMask)==null?void 0:k.call({lexer:this},o))??o;let s=!1,d="",i=1/0;for(;t;){if(t.length<i)i=t.length;else{this.infiniteLoopError(t.charCodeAt(0));break}s||(d=""),s=!1;let x;if((u=(b=this.options.extensions)==null?void 0:b.inline)!=null&&u.some(T=>(x=T.call({lexer:this},t,n))?(t=t.substring(x.raw.length),n.push(x),!0):!1))continue;if(x=this.tokenizer.escape(t)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.tag(t)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.link(t)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.reflink(t,this.tokens.links)){t=t.substring(x.raw.length);let T=n.at(-1);x.type==="text"&&(T==null?void 0:T.type)==="text"?(T.raw+=x.raw,T.text+=x.text):n.push(x);continue}if(x=this.tokenizer.emStrong(t,o,d)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.codespan(t)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.br(t)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.del(t,o,d)){t=t.substring(x.raw.length),n.push(x);continue}if(x=this.tokenizer.autolink(t)){t=t.substring(x.raw.length),n.push(x);continue}if(!this.state.inLink&&(x=this.tokenizer.url(t))){t=t.substring(x.raw.length),n.push(x);continue}let N=t;if((w=this.options.extensions)!=null&&w.startInline){let T=1/0,L=t.slice(1),M;this.options.extensions.startInline.forEach(D=>{M=D.call({lexer:this},L),typeof M=="number"&&M>=0&&(T=Math.min(T,M))}),T<1/0&&T>=0&&(N=t.substring(0,T+1))}if(x=this.tokenizer.inlineText(N)){t=t.substring(x.raw.length),x.raw.slice(-1)!=="_"&&(d=x.raw.slice(-1)),s=!0;let T=n.at(-1);(T==null?void 0:T.type)==="text"?(T.raw+=x.raw,T.text+=x.text):n.push(x);continue}if(t){this.infiniteLoopError(t.charCodeAt(0));break}}return n}infiniteLoopError(t){let n="Infinite loop on byte: "+t;if(this.options.silent)console.error(n);else throw new Error(n)}},ae=class{constructor(r){I(this,"options");I(this,"parser");this.options=r||W}space(r){return""}code({text:r,lang:t,escaped:n}){var a;let o=(a=(t||"").match($.notSpaceStart))==null?void 0:a[0],l=r.replace($.endingNewline,"")+`
`;return o?'<pre><code class="language-'+H(o)+'">'+(n?l:H(l,!0))+`</code></pre>
`:"<pre><code>"+(n?l:H(l,!0))+`</code></pre>
`}blockquote({tokens:r}){return`<blockquote>
${this.parser.parse(r)}</blockquote>
`}html({text:r}){return r}def(r){return""}heading({tokens:r,depth:t}){return`<h${t}>${this.parser.parseInline(r)}</h${t}>
`}hr(r){return`<hr>
`}list(r){let t=r.ordered,n=r.start,o="";for(let s=0;s<r.items.length;s++){let d=r.items[s];o+=this.listitem(d)}let l=t?"ol":"ul",a=t&&n!==1?' start="'+n+'"':"";return"<"+l+a+`>
`+o+"</"+l+`>
`}listitem(r){return`<li>${this.parser.parse(r.tokens)}</li>
`}checkbox({checked:r}){return"<input "+(r?'checked="" ':"")+'disabled="" type="checkbox"> '}paragraph({tokens:r}){return`<p>${this.parser.parseInline(r)}</p>
`}table(r){let t="",n="";for(let l=0;l<r.header.length;l++)n+=this.tablecell(r.header[l]);t+=this.tablerow({text:n});let o="";for(let l=0;l<r.rows.length;l++){let a=r.rows[l];n="";for(let s=0;s<a.length;s++)n+=this.tablecell(a[s]);o+=this.tablerow({text:n})}return o&&(o=`<tbody>${o}</tbody>`),`<table>
<thead>
`+t+`</thead>
`+o+`</table>
`}tablerow({text:r}){return`<tr>
${r}</tr>
`}tablecell(r){let t=this.parser.parseInline(r.tokens),n=r.header?"th":"td";return(r.align?`<${n} align="${r.align}">`:`<${n}>`)+t+`</${n}>
`}strong({tokens:r}){return`<strong>${this.parser.parseInline(r)}</strong>`}em({tokens:r}){return`<em>${this.parser.parseInline(r)}</em>`}codespan({text:r}){return`<code>${H(r,!0)}</code>`}br(r){return"<br>"}del({tokens:r}){return`<del>${this.parser.parseInline(r)}</del>`}link({href:r,title:t,tokens:n}){let o=this.parser.parseInline(n),l=Re(r);if(l===null)return o;r=l;let a='<a href="'+r+'"';return t&&(a+=' title="'+H(t)+'"'),a+=">"+o+"</a>",a}image({href:r,title:t,text:n,tokens:o}){o&&(n=this.parser.parseInline(o,this.parser.textRenderer));let l=Re(r);if(l===null)return H(n);r=l;let a=`<img src="${r}" alt="${H(n)}"`;return t&&(a+=` title="${H(t)}"`),a+=">",a}text(r){return"tokens"in r&&r.tokens?this.parser.parseInline(r.tokens):"escaped"in r&&r.escaped?r.text:H(r.text)}},ye=class{strong({text:r}){return r}em({text:r}){return r}codespan({text:r}){return r}del({text:r}){return r}html({text:r}){return r}text({text:r}){return r}link({text:r}){return""+r}image({text:r}){return""+r}br(){return""}checkbox({raw:r}){return r}},z=class Ee{constructor(t){I(this,"options");I(this,"renderer");I(this,"textRenderer");this.options=t||W,this.options.renderer=this.options.renderer||new ae,this.renderer=this.options.renderer,this.renderer.options=this.options,this.renderer.parser=this,this.textRenderer=new ye}static parse(t,n){return new Ee(n).parse(t)}static parseInline(t,n){return new Ee(n).parseInline(t)}parse(t){var o,l;this.renderer.parser=this;let n="";for(let a=0;a<t.length;a++){let s=t[a];if((l=(o=this.options.extensions)==null?void 0:o.renderers)!=null&&l[s.type]){let i=s,f=this.options.extensions.renderers[i.type].call({parser:this},i);if(f!==!1||!["space","hr","heading","code","table","blockquote","list","html","def","paragraph","text"].includes(i.type)){n+=f||"";continue}}let d=s;switch(d.type){case"space":{n+=this.renderer.space(d);break}case"hr":{n+=this.renderer.hr(d);break}case"heading":{n+=this.renderer.heading(d);break}case"code":{n+=this.renderer.code(d);break}case"table":{n+=this.renderer.table(d);break}case"blockquote":{n+=this.renderer.blockquote(d);break}case"list":{n+=this.renderer.list(d);break}case"checkbox":{n+=this.renderer.checkbox(d);break}case"html":{n+=this.renderer.html(d);break}case"def":{n+=this.renderer.def(d);break}case"paragraph":{n+=this.renderer.paragraph(d);break}case"text":{n+=this.renderer.text(d);break}default:{let i='Token with "'+d.type+'" type was not found.';if(this.options.silent)return console.error(i),"";throw new Error(i)}}}return n}parseInline(t,n=this.renderer){var l,a;this.renderer.parser=this;let o="";for(let s=0;s<t.length;s++){let d=t[s];if((a=(l=this.options.extensions)==null?void 0:l.renderers)!=null&&a[d.type]){let f=this.options.extensions.renderers[d.type].call({parser:this},d);if(f!==!1||!["escape","html","link","image","strong","em","codespan","br","del","text"].includes(d.type)){o+=f||"";continue}}let i=d;switch(i.type){case"escape":{o+=n.text(i);break}case"html":{o+=n.html(i);break}case"link":{o+=n.link(i);break}case"image":{o+=n.image(i);break}case"checkbox":{o+=n.checkbox(i);break}case"strong":{o+=n.strong(i);break}case"em":{o+=n.em(i);break}case"codespan":{o+=n.codespan(i);break}case"br":{o+=n.br(i);break}case"del":{o+=n.del(i);break}case"text":{o+=n.text(i);break}default:{let f='Token with "'+i.type+'" type was not found.';if(this.options.silent)return console.error(f),"";throw new Error(f)}}}return o}},ee=(se=class{constructor(r){I(this,"options");I(this,"block");this.options=r||W}preprocess(r){return r}postprocess(r){return r}processAllTokens(r){return r}emStrongMask(r){return r}provideLexer(r=this.block){return r?F.lex:F.lexInline}provideParser(r=this.block){return r?z.parse:z.parseInline}},I(se,"passThroughHooks",new Set(["preprocess","postprocess","processAllTokens","emStrongMask"])),I(se,"passThroughHooksRespectAsync",new Set(["preprocess","postprocess","processAllTokens"])),se),dl=class{constructor(...r){I(this,"defaults",ie());I(this,"options",this.setOptions);I(this,"parse",this.parseMarkdown(!0));I(this,"parseInline",this.parseMarkdown(!1));I(this,"Parser",z);I(this,"Renderer",ae);I(this,"TextRenderer",ye);I(this,"Lexer",F);I(this,"Tokenizer",oe);I(this,"Hooks",ee);this.use(...r)}walkTokens(r,t){var o,l;let n=[];for(let a of r)switch(n=n.concat(t.call(this,a)),a.type){case"table":{let s=a;for(let d of s.header)n=n.concat(this.walkTokens(d.tokens,t));for(let d of s.rows)for(let i of d)n=n.concat(this.walkTokens(i.tokens,t));break}case"list":{let s=a;n=n.concat(this.walkTokens(s.items,t));break}default:{let s=a;(l=(o=this.defaults.extensions)==null?void 0:o.childTokens)!=null&&l[s.type]?this.defaults.extensions.childTokens[s.type].forEach(d=>{let i=s[d].flat(1/0);n=n.concat(this.walkTokens(i,t))}):s.tokens&&(n=n.concat(this.walkTokens(s.tokens,t)))}}return n}use(...r){let t=this.defaults.extensions||{renderers:{},childTokens:{}};return r.forEach(n=>{let o={...n};if(o.async=this.defaults.async||o.async||!1,n.extensions&&(n.extensions.forEach(l=>{if(!l.name)throw new Error("extension name required");if("renderer"in l){let a=t.renderers[l.name];a?t.renderers[l.name]=function(...s){let d=l.renderer.apply(this,s);return d===!1&&(d=a.apply(this,s)),d}:t.renderers[l.name]=l.renderer}if("tokenizer"in l){if(!l.level||l.level!=="block"&&l.level!=="inline")throw new Error("extension level must be 'block' or 'inline'");let a=t[l.level];a?a.unshift(l.tokenizer):t[l.level]=[l.tokenizer],l.start&&(l.level==="block"?t.startBlock?t.startBlock.push(l.start):t.startBlock=[l.start]:l.level==="inline"&&(t.startInline?t.startInline.push(l.start):t.startInline=[l.start]))}"childTokens"in l&&l.childTokens&&(t.childTokens[l.name]=l.childTokens)}),o.extensions=t),n.renderer){let l=this.defaults.renderer||new ae(this.defaults);for(let a in n.renderer){if(!(a in l))throw new Error(`renderer '${a}' does not exist`);if(["options","parser"].includes(a))continue;let s=a,d=n.renderer[s],i=l[s];l[s]=(...f)=>{let k=d.apply(l,f);return k===!1&&(k=i.apply(l,f)),k||""}}o.renderer=l}if(n.tokenizer){let l=this.defaults.tokenizer||new oe(this.defaults);for(let a in n.tokenizer){if(!(a in l))throw new Error(`tokenizer '${a}' does not exist`);if(["options","rules","lexer"].includes(a))continue;let s=a,d=n.tokenizer[s],i=l[s];l[s]=(...f)=>{let k=d.apply(l,f);return k===!1&&(k=i.apply(l,f)),k}}o.tokenizer=l}if(n.hooks){let l=this.defaults.hooks||new ee;for(let a in n.hooks){if(!(a in l))throw new Error(`hook '${a}' does not exist`);if(["options","block"].includes(a))continue;let s=a,d=n.hooks[s],i=l[s];ee.passThroughHooks.has(a)?l[s]=f=>{if(this.defaults.async&&ee.passThroughHooksRespectAsync.has(a))return(async()=>{let b=await d.call(l,f);return i.call(l,b)})();let k=d.call(l,f);return i.call(l,k)}:l[s]=(...f)=>{if(this.defaults.async)return(async()=>{let b=await d.apply(l,f);return b===!1&&(b=await i.apply(l,f)),b})();let k=d.apply(l,f);return k===!1&&(k=i.apply(l,f)),k}}o.hooks=l}if(n.walkTokens){let l=this.defaults.walkTokens,a=n.walkTokens;o.walkTokens=function(s){let d=[];return d.push(a.call(this,s)),l&&(d=d.concat(l.call(this,s))),d}}this.defaults={...this.defaults,...o}}),this}setOptions(r){return this.defaults={...this.defaults,...r},this}lexer(r,t){return F.lex(r,t??this.defaults)}parser(r,t){return z.parse(r,t??this.defaults)}parseMarkdown(r){return(t,n)=>{let o={...n},l={...this.defaults,...o},a=this.onError(!!l.silent,!!l.async);if(this.defaults.async===!0&&o.async===!1)return a(new Error("marked(): The async option was set to true by an extension. Remove async: false from the parse options object to return a Promise."));if(typeof t>"u"||t===null)return a(new Error("marked(): input parameter is undefined or null"));if(typeof t!="string")return a(new Error("marked(): input parameter is of type "+Object.prototype.toString.call(t)+", string expected"));if(l.hooks&&(l.hooks.options=l,l.hooks.block=r),l.async)return(async()=>{let s=l.hooks?await l.hooks.preprocess(t):t,d=await(l.hooks?await l.hooks.provideLexer(r):r?F.lex:F.lexInline)(s,l),i=l.hooks?await l.hooks.processAllTokens(d):d;l.walkTokens&&await Promise.all(this.walkTokens(i,l.walkTokens));let f=await(l.hooks?await l.hooks.provideParser(r):r?z.parse:z.parseInline)(i,l);return l.hooks?await l.hooks.postprocess(f):f})().catch(a);try{l.hooks&&(t=l.hooks.preprocess(t));let s=(l.hooks?l.hooks.provideLexer(r):r?F.lex:F.lexInline)(t,l);l.hooks&&(s=l.hooks.processAllTokens(s)),l.walkTokens&&this.walkTokens(s,l.walkTokens);let d=(l.hooks?l.hooks.provideParser(r):r?z.parse:z.parseInline)(s,l);return l.hooks&&(d=l.hooks.postprocess(d)),d}catch(s){return a(s)}}}onError(r,t){return n=>{if(n.message+=`
Please report this to https://github.com/markedjs/marked.`,r){let o="<p>An error occurred:</p><pre>"+H(n.message+"",!0)+"</pre>";return t?Promise.resolve(o):o}if(t)return Promise.reject(n);throw n}}},Z=new dl;function C(r,t){return Z.parse(r,t)}C.options=C.setOptions=function(r){return Z.setOptions(r),C.defaults=Z.defaults,Ve(C.defaults),C},C.getDefaults=ie,C.defaults=W,C.use=function(...r){return Z.use(...r),C.defaults=Z.defaults,Ve(C.defaults),C},C.walkTokens=function(r,t){return Z.walkTokens(r,t)},C.parseInline=Z.parseInline,C.Parser=z,C.parser=z.parse,C.Renderer=ae,C.TextRenderer=ye,C.Lexer=F,C.lexer=F.lex,C.Tokenizer=oe,C.Hooks=ee,C.parse=C,C.options,C.setOptions,C.use,C.walkTokens,C.parseInline,z.parse,F.lex;const pl={class:"modal-content help-modal"},ml={class:"modal-header"},gl={class:"header-actions"},kl={class:"modal-body"},hl={class:"doc-sidebar"},fl={class:"doc-sidebar-group"},yl={class:"doc-sidebar-header"},ul=["onClick"],El={class:"doc-sidebar-group"},xl={class:"doc-sidebar-header"},bl={class:"doc-content"},Vl=["innerHTML"],Nl=["innerHTML"],wl=["innerHTML"],Tl=["innerHTML"],Sl=["innerHTML"],Cl=["innerHTML"],Bl=["innerHTML"],Il={class:"doc-pagination"},Al=["disabled"],Pl={class:"page-info"},Ml=["disabled"],$l=O({__name:"HelpModal",props:{initialDoc:{type:String,default:"getting-started"}},emits:["close","openAbout"],setup(r,{emit:t}){const n=r,o=e.ref(n.initialDoc),l=e.ref(""),a=e.ref(null),s=[{id:"getting-started",title:"快速开始",icon:"home"},{id:"features",title:"功能介绍",icon:"info"},{id:"api",title:"API 文档",icon:"code"},{id:"tools",title:"工具文档",icon:"tool"},{id:"shortcuts",title:"快捷键",icon:"keyboard"},{id:"faq",title:"常见问题",icon:"help"}],d=e.ref(s),i=e.computed(()=>[...d.value,{id:"changelog",title:"更新日志",icon:"activity"}]),f=e.computed(()=>i.value.findIndex(h=>h.id===o.value)),k=e.computed(()=>f.value>0),b=e.computed(()=>f.value<i.value.length-1);function u(){const h=l.value.toLowerCase().trim();if(!h){d.value=s;return}d.value=s.filter(p=>p.title.toLowerCase().includes(h)||p.id.includes(h)),d.value.length>0&&!d.value.find(p=>p.id===o.value)&&(o.value=d.value[0].id)}const w=h=>C(h,{breaks:!0,gfm:!0}).replace(/<table>/g,'<table class="doc-table">'),x=e.computed(()=>w(br)),N=e.computed(()=>w(Vr)),T=e.computed(()=>w(yr)),L=e.computed(()=>w(ur)),M=e.computed(()=>w(Er)),D=e.computed(()=>w(xr)),K=e.computed(()=>w(Nr));function U(){var h;k.value&&(o.value=i.value[f.value-1].id,(h=a.value)==null||h.scrollTo(0,0))}function R(){var h;b.value&&(o.value=i.value[f.value+1].id,(h=a.value)==null||h.scrollTo(0,0))}return e.onMounted(()=>{u()}),(h,p)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:p[3]||(p[3]=e.withModifiers(m=>h.$emit("close"),["self"]))},[e.createElementVNode("div",pl,[e.createCommentVNode(" 头部 "),e.createElementVNode("div",ml,[e.createElementVNode("h2",null,[e.createVNode(P,{name:"book-open",size:18}),p[4]||(p[4]=e.createTextVNode(" 帮助文档",-1))]),e.createElementVNode("div",gl,[e.createElementVNode("button",{class:"btn-about",onClick:p[0]||(p[0]=m=>h.$emit("openAbout")),title:"关于 PairCode"},[e.createVNode(P,{name:"info",size:14}),p[5]||(p[5]=e.createTextVNode(" 关于 ",-1))]),e.createElementVNode("button",{class:"modal-close",onClick:p[1]||(p[1]=m=>h.$emit("close"))},"×")])]),e.createCommentVNode(" 主体 "),e.createElementVNode("div",kl,[e.createCommentVNode(" 侧边导航 "),e.createElementVNode("div",hl,[e.createCommentVNode(" 文档中心分组 "),e.createElementVNode("div",fl,[e.createElementVNode("div",yl,[e.createVNode(P,{name:"book",size:14}),p[6]||(p[6]=e.createElementVNode("span",null,"文档中心",-1))]),(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(d.value,m=>(e.openBlock(),e.createElementBlock("div",{key:m.id,class:e.normalizeClass(["doc-nav-item",{active:o.value===m.id}]),onClick:E=>o.value=m.id},[e.createVNode(P,{name:m.icon,size:16},null,8,["name"]),e.createElementVNode("span",null,e.toDisplayString(m.title),1)],10,ul))),128))]),e.createCommentVNode(" 更新日志 "),e.createElementVNode("div",El,[e.createElementVNode("div",xl,[e.createVNode(P,{name:"clock",size:14}),p[7]||(p[7]=e.createElementVNode("span",null,"其他",-1))]),e.createElementVNode("div",{class:e.normalizeClass(["doc-nav-item",{active:o.value==="changelog"}]),onClick:p[2]||(p[2]=m=>o.value="changelog")},[e.createVNode(P,{name:"activity",size:16}),p[8]||(p[8]=e.createElementVNode("span",null,"更新日志",-1))],2)])]),e.createCommentVNode(" 文档内容 "),e.createElementVNode("div",bl,[e.createElementVNode("div",{class:"doc-content-inner",ref_key:"contentRef",ref:a},[o.value==="faq"?(e.openBlock(),e.createElementBlock("div",{key:0,class:"doc-markdown",innerHTML:x.value},null,8,Vl)):o.value==="getting-started"?(e.openBlock(),e.createElementBlock("div",{key:1,class:"doc-markdown",innerHTML:N.value},null,8,Nl)):o.value==="features"?(e.openBlock(),e.createElementBlock("div",{key:2,class:"doc-markdown",innerHTML:T.value},null,8,wl)):o.value==="api"?(e.openBlock(),e.createElementBlock("div",{key:3,class:"doc-markdown",innerHTML:L.value},null,8,Tl)):o.value==="tools"?(e.openBlock(),e.createElementBlock("div",{key:4,class:"doc-markdown",innerHTML:M.value},null,8,Sl)):o.value==="shortcuts"?(e.openBlock(),e.createElementBlock("div",{key:5,class:"doc-markdown",innerHTML:D.value},null,8,Cl)):o.value==="changelog"?(e.openBlock(),e.createElementBlock("div",{key:6,class:"doc-markdown",innerHTML:K.value},null,8,Bl)):e.createCommentVNode("v-if",!0)],512),e.createCommentVNode(" 底部翻页 "),e.createElementVNode("div",Il,[e.createElementVNode("button",{class:"page-btn",onClick:U,disabled:!k.value},[e.createVNode(P,{name:"chevron-left",size:14}),p[9]||(p[9]=e.createTextVNode(" 上一页 ",-1))],8,Al),e.createElementVNode("span",Pl,e.toDisplayString(f.value+1)+" / "+e.toDisplayString(i.value.length),1),e.createElementVNode("button",{class:"page-btn",onClick:R,disabled:!b.value},[p[10]||(p[10]=e.createTextVNode(" 下一页 ",-1)),e.createVNode(P,{name:"chevron-right",size:14})],8,Ml)])])])])]))}},[["__scopeId","data-v-667c64dc"]]),Dl="data:image/svg+xml,%3csvg%20xmlns='http://www.w3.org/2000/svg'%20width='512'%20height='512'%20viewBox='0%200%20512%20512'%3e%3cdefs%3e%3c!--%20背景渐变（深色科技风）%20--%3e%3clinearGradient%20id='bgGrad'%20x1='0'%20y1='0'%20x2='1'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%230a1628'/%3e%3cstop%20offset='100%25'%20stop-color='%230d1f2e'/%3e%3c/linearGradient%3e%3c!--%20左侧尖括号渐变（科技蓝）%20--%3e%3clinearGradient%20id='leftBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='100%25'%20stop-color='%230077b6'/%3e%3c/linearGradient%3e%3c!--%20右侧尖括号渐变（科技绿）%20--%3e%3clinearGradient%20id='rightBracket'%20x1='0'%20y1='0'%20x2='0'%20y2='1'%3e%3cstop%20offset='0%25'%20stop-color='%2300e676'/%3e%3cstop%20offset='100%25'%20stop-color='%2300c853'/%3e%3c/linearGradient%3e%3c!--%20中间连接线（蓝绿渐变）%20--%3e%3clinearGradient%20id='connector'%20x1='0'%20y1='0'%20x2='1'%20y2='0'%3e%3cstop%20offset='0%25'%20stop-color='%2300d4ff'/%3e%3cstop%20offset='50%25'%20stop-color='%2300e5ff'/%3e%3cstop%20offset='100%25'%20stop-color='%2300e676'/%3e%3c/linearGradient%3e%3c!--%20外发光%20--%3e%3cfilter%20id='glow'%3e%3cfeGaussianBlur%20stdDeviation='4'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3cfilter%20id='softGlow'%3e%3cfeGaussianBlur%20stdDeviation='8'%20result='blur'/%3e%3cfeMerge%3e%3cfeMergeNode%20in='blur'/%3e%3cfeMergeNode%20in='SourceGraphic'/%3e%3c/feMerge%3e%3c/filter%3e%3c/defs%3e%3c!--%20圆角方形背景（深色科技底）%20--%3e%3crect%20x='32'%20y='32'%20width='448'%20height='448'%20rx='96'%20ry='96'%20fill='url(%23bgGrad)'%20stroke='%231a3a4a'%20stroke-width='2'/%3e%3c!--%20左侧%20%3c%20尖括号（三段式直线%20—%20科技蓝，代表代码输入/开发者）%20--%3e%3cpath%20d='M180%20150%20L96%20256%20L180%20362'%20stroke='url(%23leftBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20右侧%20%3e%20尖括号（三段式直线%20—%20科技绿，代表代码输出/AI伙伴）%20--%3e%3cpath%20d='M332%20150%20L416%20256%20L332%20362'%20stroke='url(%23rightBracket)'%20stroke-width='40'%20stroke-linejoin='round'%20fill='none'%20filter='url(%23glow)'/%3e%3c!--%20中间「=」连接线已移除（图标只留%20%3c%20%3e%20尖括号%20+%20中心%20AI%20核心光点）。%20--%3e%3c!--%20中心光点（代表%20AI%20核心%20—%20亮青色）%20--%3e%3ccircle%20cx='256'%20cy='256'%20r='18'%20fill='transparent'%20stroke='%2300e5ff'%20stroke-width='3'%20opacity='0.6'/%3e%3ccircle%20cx='256'%20cy='256'%20r='8'%20fill='%2300e5ff'%20opacity='0.9'%3e%3canimate%20attributeName='opacity'%20values='0.6;1;0.6'%20dur='2s'%20repeatCount='indefinite'/%3e%3c/circle%3e%3c/svg%3e",Rl={class:"modal-content about-modal"},Gl={class:"modal-header"},Ll={class:"modal-body"},Ol={class:"about-left-col"},Fl={class:"about-hero"},zl={class:"about-logo"},Ul=["src"],Hl={class:"about-version"},jl={class:"about-right-col"},Kl={class:"about-section"},ql={class:"feature-list"},Wl={class:"about-section"},Jl={key:0,class:"sys-info"},Zl={class:"info-row"},Ql={class:"info-row"},Xl={class:"info-row"},Yl={class:"info-path"},_l={class:"info-row"},vl={key:1,class:"loading-info"},eo={class:"modal-footer"},no=O({__name:"AboutModal",props:{showHelpBtn:{type:Boolean,default:!0}},emits:["close","openHelp"],setup(r,{emit:t}){const n=e.ref(""),o=e.ref({}),l=e.ref(!0);return e.onMounted(async()=>{try{const a=await A.apiGet("/system/info");o.value=a,a.version&&(n.value=a.version)}catch{}l.value=!1}),(a,s)=>(e.openBlock(),e.createElementBlock("div",{class:"modal-overlay",onClick:s[3]||(s[3]=e.withModifiers(d=>a.$emit("close"),["self"]))},[e.createElementVNode("div",Rl,[e.createElementVNode("div",Gl,[e.createElementVNode("h2",null,[e.createVNode(P,{name:"info",size:18}),s[4]||(s[4]=e.createTextVNode(" 关于 PairCode",-1))]),e.createElementVNode("button",{class:"modal-close",onClick:s[0]||(s[0]=d=>a.$emit("close"))},"×")]),e.createElementVNode("div",Ll,[e.createCommentVNode(" 左列：Logo + 描述 + 技术栈 "),e.createElementVNode("div",Ol,[e.createCommentVNode(" Logo + 标题 "),e.createElementVNode("div",Fl,[e.createElementVNode("div",zl,[e.createElementVNode("img",{src:e.unref(Dl),class:"about-logo-img",alt:"PairCode"},null,8,Ul)]),s[5]||(s[5]=e.createElementVNode("div",{class:"about-title"},"PairCode IDE",-1)),e.createElementVNode("div",Hl,"版本 "+e.toDisplayString(n.value),1)]),e.createCommentVNode(" 描述 "),s[6]||(s[6]=e.createElementVNode("div",{class:"about-section"},[e.createElementVNode("p",{class:"about-description"}," PairCode IDE 是一款纯 Web 端的 AI 辅助编程集成开发环境， 专为浏览器而设计。无需安装任何桌面客户端或本地 IDE 软件， 打开浏览器即可开始编程。它将 AI 对话能力深度融入编码工作流， 你只需用自然语言描述需求，AI 就能自动理解上下文、读写文件、执行命令、 管理版本控制。从代码生成到项目运维，在同一个浏览器窗口中全部完成。 ")],-1)),e.createCommentVNode(" 技术栈 "),s[7]||(s[7]=e.createStaticVNode('<div class="about-section" data-v-cdb64a03><div class="section-title" data-v-cdb64a03>技术栈</div><div class="tech-stack" data-v-cdb64a03><span class="tech-badge" data-v-cdb64a03>Go</span><span class="tech-badge" data-v-cdb64a03>Vue 3</span><span class="tech-badge" data-v-cdb64a03>WebSocket</span><span class="tech-badge" data-v-cdb64a03>CodeMirror</span><span class="tech-badge" data-v-cdb64a03>插件化工具</span><span class="tech-badge" data-v-cdb64a03>TS 编译器</span><span class="tech-badge" data-v-cdb64a03>MCP</span><span class="tech-badge" data-v-cdb64a03>CodeGraph</span><span class="tech-badge" data-v-cdb64a03>DAP</span></div></div>',1))]),e.createCommentVNode(" 右列：特性 + 系统信息 "),e.createElementVNode("div",jl,[e.createCommentVNode(" 特性亮点 "),e.createElementVNode("div",Kl,[s[18]||(s[18]=e.createElementVNode("div",{class:"section-title"},"主要特性",-1)),e.createElementVNode("ul",ql,[e.createElementVNode("li",null,[e.createVNode(P,{name:"bot",size:14,color:"var(--accent)"}),s[8]||(s[8]=e.createTextVNode(" AI 对话编程 — 用自然语言与 AI 对话，自动生成与重构代码",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"file",size:14,color:"var(--accent)"}),s[9]||(s[9]=e.createTextVNode(" 智能代码编辑器 — 多语言语法高亮，浏览器中流畅编辑",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"git-branch",size:14,color:"var(--accent)"}),s[10]||(s[10]=e.createTextVNode(" Git 版本控制 — 在对话中完成全部 Git 操作",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"terminal",size:14,color:"var(--accent)"}),s[11]||(s[11]=e.createTextVNode(" 内置终端 — 无需离开浏览器即可执行命令",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"search",size:14,color:"var(--accent)"}),s[12]||(s[12]=e.createTextVNode(" 全局搜索 — 快速搜索文件与代码内容",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"settings",size:14,color:"var(--accent)"}),s[13]||(s[13]=e.createTextVNode(" 自主 Agent 模式 — AI 主动分析项目并自动执行任务",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"grid",size:14,color:"var(--accent)"}),s[14]||(s[14]=e.createTextVNode(" 对话历史管理 — 自动保存、回溯与继续历史对话",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"tool",size:14,color:"var(--accent)"}),s[15]||(s[15]=e.createTextVNode(" Skills / MCP 扩展 — 通过技能市场扩展 IDE 能力",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"code",size:14,color:"var(--accent)"}),s[16]||(s[16]=e.createTextVNode(" 内置调试器 — 支持 Go 程序的断点、单步和变量查看",-1))]),e.createElementVNode("li",null,[e.createVNode(P,{name:"image",size:14,color:"var(--accent)"}),s[17]||(s[17]=e.createTextVNode(" 网页验证 — 打开 URL、截图、分析页面效果",-1))])])]),e.createCommentVNode(" 系统信息 "),e.createElementVNode("div",Wl,[s[23]||(s[23]=e.createElementVNode("div",{class:"section-title"},"系统信息",-1)),l.value?(e.openBlock(),e.createElementBlock("div",vl,"加载中...")):(e.openBlock(),e.createElementBlock("div",Jl,[e.createElementVNode("div",Zl,[s[19]||(s[19]=e.createElementVNode("span",{class:"info-label"},"主机名",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.hostname),1)]),e.createElementVNode("div",Ql,[s[20]||(s[20]=e.createElementVNode("span",{class:"info-label"},"操作系统",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.os),1)]),e.createElementVNode("div",Xl,[s[21]||(s[21]=e.createElementVNode("span",{class:"info-label"},"工作区",-1)),e.createElementVNode("span",Yl,e.toDisplayString(o.value.workspace),1)]),e.createElementVNode("div",_l,[s[22]||(s[22]=e.createElementVNode("span",{class:"info-label"},"平台信息",-1)),e.createElementVNode("span",null,e.toDisplayString(o.value.goos),1)])]))])])]),e.createCommentVNode(" 底部 "),e.createElementVNode("div",eo,[r.showHelpBtn?(e.openBlock(),e.createElementBlock("button",{key:0,class:"btn-primary",onClick:s[1]||(s[1]=d=>a.$emit("openHelp"))},[e.createVNode(P,{name:"book-open",size:14}),s[24]||(s[24]=e.createTextVNode(" 查看帮助文档 ",-1))])):e.createCommentVNode("v-if",!0),e.createElementVNode("button",{class:"btn-secondary",onClick:s[2]||(s[2]=d=>a.$emit("close"))},"关闭")])])]))}},[["__scopeId","data-v-cdb64a03"]]),to={class:"toast-container"},ro={class:"dlg-box",style:{"max-width":"400px"}},lo={class:"dlg-title"},oo={class:"dlg-body"},ao={class:"dlg-actions"},so={class:"dlg-box",style:{"max-width":"420px"}},io={class:"dlg-title"},co={class:"dlg-body",style:{display:"flex","flex-direction":"column",gap:"8px"}},po={style:{"font-size":"13px",color:"var(--text-secondary)"}},mo=["placeholder"],go={class:"dlg-actions"},ko={class:"dlg-box",style:{"max-width":"400px"}},ho={class:"dlg-title"},fo={class:"dlg-body",style:{"white-space":"pre-line"}},yo=O({__name:"GlobalDialogs",setup(r){const t=e.ref(null);e.watch(()=>y.dialogState.show,l=>{l&&y.dialogState.type==="prompt"&&e.nextTick(()=>{var a,s;(a=t.value)==null||a.focus(),(s=t.value)==null||s.select()})});function n(){if(y.dialogState.type==="prompt"){const l=y.dialogState.inputValue;y.dialogState.show=!1,y.dialogState.resolve&&y.dialogState.resolve(l)}else if(y.dialogState.type==="confirm"&&y.dialogState.checkboxLabel){const a=y.dialogState.checkboxValue;y.dialogState.show=!1,y.dialogState.resolve&&y.dialogState.resolve({confirmed:!0,checked:a})}else y.dialogState.show=!1,y.dialogState.resolve&&y.dialogState.resolve(!0);y.dialogState.resolve=null}function o(){y.dialogState.type==="confirm"&&y.dialogState.checkboxLabel?(y.dialogState.show=!1,y.dialogState.resolve&&y.dialogState.resolve({confirmed:!1,checked:y.dialogState.checkboxValue})):y.dialogState.type==="prompt"?(y.dialogState.show=!1,y.dialogState.resolve&&y.dialogState.resolve(null)):(y.dialogState.show=!1,y.dialogState.resolve&&y.dialogState.resolve(!1)),y.dialogState.resolve=null}return(l,a)=>(e.openBlock(),e.createElementBlock(e.Fragment,null,[e.createCommentVNode(" Toast 通知区域 "),e.createElementVNode("div",to,[(e.openBlock(!0),e.createElementBlock(e.Fragment,null,e.renderList(e.unref(y.dialogState).toasts,s=>(e.openBlock(),e.createElementBlock("div",{key:s.id,class:e.normalizeClass(["toast-item","toast-"+(s.type||"info")])},e.toDisplayString(s.message),3))),128))]),e.createCommentVNode(" Confirm 对话框 "),e.unref(y.dialogState).show&&e.unref(y.dialogState).type==="confirm"?(e.openBlock(),e.createElementBlock("div",{key:0,class:"dlg-overlay",onClick:e.withModifiers(o,["self"])},[e.createElementVNode("div",ro,[e.createElementVNode("div",lo,e.toDisplayString(e.unref(y.dialogState).title),1),e.createElementVNode("div",oo,e.toDisplayString(e.unref(y.dialogState).message),1),e.unref(y.dialogState).checkboxLabel?(e.openBlock(),e.createElementBlock("label",{key:0,class:"dlg-checkbox",onClick:a[1]||(a[1]=e.withModifiers(()=>{},["stop"]))},[e.withDirectives(e.createElementVNode("input",{type:"checkbox","onUpdate:modelValue":a[0]||(a[0]=s=>e.unref(y.dialogState).checkboxValue=s)},null,512),[[e.vModelCheckbox,e.unref(y.dialogState).checkboxValue]]),e.createElementVNode("span",null,e.toDisplayString(e.unref(y.dialogState).checkboxLabel),1)])):e.createCommentVNode("v-if",!0),e.createElementVNode("div",ao,[e.createElementVNode("button",{class:"dlg-btn",onClick:o},e.toDisplayString(e.unref(y.dialogState).cancelText),1),e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},e.toDisplayString(e.unref(y.dialogState).confirmText),1)])])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" Prompt 对话框 "),e.unref(y.dialogState).show&&e.unref(y.dialogState).type==="prompt"?(e.openBlock(),e.createElementBlock("div",{key:1,class:"dlg-overlay",onClick:e.withModifiers(o,["self"])},[e.createElementVNode("div",so,[e.createElementVNode("div",io,e.toDisplayString(e.unref(y.dialogState).title),1),e.createElementVNode("div",co,[e.createElementVNode("span",po,e.toDisplayString(e.unref(y.dialogState).message),1),e.withDirectives(e.createElementVNode("input",{ref_key:"promptInputRef",ref:t,"onUpdate:modelValue":a[2]||(a[2]=s=>e.unref(y.dialogState).inputValue=s),placeholder:e.unref(y.dialogState).inputPlaceholder,class:"dlg-input",onKeyup:[e.withKeys(n,["enter"]),e.withKeys(o,["escape"])]},null,40,mo),[[e.vModelText,e.unref(y.dialogState).inputValue]])]),e.createElementVNode("div",go,[e.createElementVNode("button",{class:"dlg-btn",onClick:o},e.toDisplayString(e.unref(y.dialogState).cancelText),1),e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},e.toDisplayString(e.unref(y.dialogState).confirmText),1)])])])):e.createCommentVNode("v-if",!0),e.createCommentVNode(" Alert 信息框 "),e.unref(y.dialogState).show&&e.unref(y.dialogState).type==="alert"?(e.openBlock(),e.createElementBlock("div",{key:2,class:"dlg-overlay",onClick:e.withModifiers(n,["self"])},[e.createElementVNode("div",ko,[e.createElementVNode("div",ho,e.toDisplayString(e.unref(y.dialogState).title),1),e.createElementVNode("div",fo,e.toDisplayString(e.unref(y.dialogState).message),1),e.createElementVNode("div",{class:"dlg-actions"},[e.createElementVNode("button",{class:"dlg-btn primary",onClick:n},"确定")])])])):e.createCommentVNode("v-if",!0)],64))}},[["__scopeId","data-v-0271e4ae"]]),uo=O({__name:"UiModals",setup(r){const t=e.ref(null);let n=null;function o(){y.showAbout.value=!1,y.showHelp.value=!0,y.helpDocTarget.value="getting-started"}function l(){y.showHelp.value=!1,y.showAbout.value=!0}return e.onMounted(()=>{n=xe.mountListSlot(t,"overlay",{isActive:a=>xe.isOverlayActive("overlay",a)})}),e.onUnmounted(()=>{n&&(n(),n=null)}),(a,s)=>(e.openBlock(),e.createElementBlock(e.Fragment,null,[e.unref(y.showSettings)?(e.openBlock(),e.createBlock(vt,{key:0,onClose:s[0]||(s[0]=d=>y.showSettings.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(y.showSystem)?(e.openBlock(),e.createBlock(mr,{key:1,onClose:s[1]||(s[1]=d=>y.showSystem.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(y.showSource)?(e.openBlock(),e.createBlock(fr,{key:2,onClose:s[2]||(s[2]=d=>y.showSource.value=!1)})):e.createCommentVNode("v-if",!0),e.unref(y.showHelp)?(e.openBlock(),e.createBlock($l,{key:3,onClose:s[3]||(s[3]=d=>y.showHelp.value=!1),onOpenAbout:l,initialDoc:e.unref(y.helpDocTarget)},null,8,["initialDoc"])):e.createCommentVNode("v-if",!0),e.unref(y.showAbout)?(e.openBlock(),e.createBlock(no,{key:4,onClose:s[4]||(s[4]=d=>y.showAbout.value=!1),onOpenHelp:o})):e.createCommentVNode("v-if",!0),e.createVNode(yo),e.createCommentVNode(" ★ overlay 槽位（list 型）：插件注册的浮动层条目叠加渲染（badge/toast/status pill 等） "),e.createElementVNode("div",{ref_key:"overlaySlotEl",ref:t,class:"plugin-overlay-host"},null,512)],64))}},[["__scopeId","data-v-519b8494"]]);function Eo(r){const t=e.createApp(uo);return t.mount(r),()=>{t.unmount()}}return j.mount=Eo,Object.defineProperty(j,Symbol.toStringTag,{value:"Module"}),j})({},window.__PAIRCODE_CORE.Vue,window.__PAIRCODE_CORE.uiState,window.__PAIRCODE_CORE.pluginRuntime,window.__PAIRCODE_CORE.api);
