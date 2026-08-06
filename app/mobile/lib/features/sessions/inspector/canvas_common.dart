import 'package:flutter/material.dart';
import 'package:opendray/core/api/canvas_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Shared bits between the Canvas tab (list + thumbnail) and the full-screen
// Canvas viewer (zoom + annotate).

IconData canvasKindIcon(CanvasKind kind) => switch (kind) {
      CanvasKind.flow => Icons.account_tree_outlined,
      CanvasKind.mindmap => Icons.hub_outlined,
      CanvasKind.graph => Icons.share_outlined,
      CanvasKind.doc => Icons.description_outlined,
      CanvasKind.ui => Icons.web_asset_outlined,
    };

String canvasKindLabel(CanvasKind kind) => switch (kind) {
      CanvasKind.flow => t.sessions.inspector.canvas.kindFlow,
      CanvasKind.mindmap => t.sessions.inspector.canvas.kindMindmap,
      CanvasKind.graph => t.sessions.inspector.canvas.kindGraph,
      CanvasKind.doc => t.sessions.inspector.canvas.kindDoc,
      CanvasKind.ui => t.sessions.inspector.canvas.kindUi,
    };

/// The preview widths the operator can render a canvas at, mirroring the web
/// panel's viewport switcher. The page is laid out at this CSS width and the
/// webview zooms it to fit, so a phone can still judge a desktop layout.
enum CanvasViewport { phone, tablet, desktop }

int canvasViewportWidth(CanvasViewport v) => switch (v) {
      CanvasViewport.phone => 390,
      CanvasViewport.tablet => 820,
      CanvasViewport.desktop => 1280,
    };

IconData canvasViewportIcon(CanvasViewport v) => switch (v) {
      CanvasViewport.phone => Icons.smartphone,
      CanvasViewport.tablet => Icons.tablet_mac,
      CanvasViewport.desktop => Icons.desktop_windows_outlined,
    };

String canvasViewportLabel(CanvasViewport v) => switch (v) {
      CanvasViewport.phone => t.sessions.inspector.canvas.viewportPhone,
      CanvasViewport.tablet => t.sessions.inspector.canvas.viewportTablet,
      CanvasViewport.desktop => t.sessions.inspector.canvas.viewportDesktop,
    };

/// A mark the operator has placed but not yet sent. Coordinates are
/// percentages of the DOCUMENT (not the viewport), so they stay meaningful
/// however the operator zoomed or scrolled when placing them.
class CanvasMark {
  CanvasMark({required this.kind, required this.x, required this.y, this.w = 0, this.h = 0});

  final String kind;
  final double x;
  final double y;
  final double w;
  final double h;

  // Resolved from the page by the injected probe at the moment of the mark.
  String selector = '';
  String html = '';
  List<String> elements = const [];
  String note = '';
}

/// Rewrites the agent's HTML so it lays out at [width] CSS pixels and carries
/// the probe. Any viewport meta the agent wrote is replaced — the operator's
/// viewport choice wins.
String canvasDocument(String html, int width) {
  final stripped = html.replaceAll(
    RegExp(r'<meta[^>]*name=["\x27]viewport["\x27][^>]*>', caseSensitive: false),
    '',
  );
  final head = <String>[
    '<meta name="viewport" content="width=$width">',
    '<meta name="color-scheme" content="light dark">',
    '<style>$_markCss</style>',
  ].join();
  const tail = '<script>$_probeScript</script>';
  var out = stripped;
  if (out.contains('<head>')) {
    out = out.replaceFirst('<head>', '<head>$head');
  } else if (RegExp('<html[^>]*>', caseSensitive: false).hasMatch(out)) {
    out = out.replaceFirstMapped(
      RegExp('<html[^>]*>', caseSensitive: false),
      (m) => '${m[0]}<head>$head</head>',
    );
  } else {
    out = '$head$out';
  }
  return out.contains('</body>')
      ? out.replaceFirst('</body>', '$tail</body>')
      : '$out$tail';
}

const _markCss = '''
.__od-mark{position:absolute;z-index:2147483000;pointer-events:none;box-sizing:border-box}
.__od-mark.rect{border:2px solid #f43f5e;background:rgba(244,63,94,.12)}
.__od-badge{position:absolute;z-index:2147483001;pointer-events:none;
  width:22px;height:22px;margin:-11px 0 0 -11px;border-radius:50%;
  background:#f43f5e;color:#fff;font:bold 12px/22px sans-serif;text-align:center;
  box-shadow:0 1px 4px rgba(0,0,0,.5)}
''';

/// Injected into every preview. It does three jobs:
///   • convert a FRACTION of the visible area into page coordinates, so Dart
///     never has to reason about zoom or scroll;
///   • resolve what the operator marked (element selector + markup, or the
///     block a frame contains plus the components inside it);
///   • draw committed marks INTO the document, so they stay glued to the
///     content while the operator pinches and pans.
const _probeScript = r'''
(function(){
function cssPath(el){
  if(!(el instanceof Element)) return '';
  var path=[];
  while(el && el.nodeType===1 && path.length<5){
    var sel=el.nodeName.toLowerCase();
    if(el.id){ sel+='#'+el.id; path.unshift(sel); break; }
    var cls=(typeof el.className==='string'?el.className:'').trim();
    if(cls){ sel+='.'+cls.split(/\s+/).slice(0,2).join('.'); }
    var p=el.parentNode;
    if(p && p.children){
      var same=Array.prototype.filter.call(p.children,function(c){return c.nodeName===el.nodeName;});
      if(same.length>1){ sel+=':nth-child('+(Array.prototype.indexOf.call(p.children,el)+1)+')'; }
    }
    path.unshift(sel);
    el=el.parentNode;
  }
  return path.join(' > ');
}
// Fraction of the visible area -> client (CSS viewport) coordinates. Uses
// visualViewport so a pinch-zoomed page still maps correctly.
function toClient(fx, fy){
  var vv = window.visualViewport;
  if(vv) return {x: vv.offsetLeft + fx*vv.width, y: vv.offsetTop + fy*vv.height};
  return {x: fx*(window.innerWidth||0), y: fy*(window.innerHeight||0)};
}
function docSize(){
  var de=document.documentElement, b=document.body;
  return {
    w: Math.max(de?de.scrollWidth:0, b?b.scrollWidth:0, 1),
    h: Math.max(de?de.scrollHeight:0, b?b.scrollHeight:0, 1)
  };
}
// Client -> document percentage, so a mark means "40% down the page".
function toDocPct(cx, cy){
  var d=docSize();
  return {
    x: ((cx + (window.scrollX||0)) / d.w) * 100,
    y: ((cy + (window.scrollY||0)) / d.h) * 100
  };
}
window.__odProbeF = function(fx, fy){
  var p = toClient(fx, fy);
  var el = document.elementFromPoint(p.x, p.y);
  var pct = toDocPct(p.x, p.y);
  return JSON.stringify({
    selector: el?cssPath(el):'',
    html: el?el.outerHTML.slice(0,1500):'',
    text: el?(el.textContent||'').replace(/\s+/g,' ').trim().slice(0,48):'',
    x: pct.x, y: pct.y
  });
};
window.__odProbeRectF = function(fx, fy, fw, fh){
  var a = toClient(fx, fy), b = toClient(fx+fw, fy+fh);
  var x=a.x, y=a.y, w=b.x-a.x, h=b.y-a.y;
  function fits(el){var r=el.getBoundingClientRect(); return r.left<=x+2 && r.top<=y+2 && r.right>=x+w-2 && r.bottom>=y+h-2;}
  var box=document.elementFromPoint(x+w/2, y+h/2);
  while(box && box.parentElement && box!==document.body && !fits(box)) box=box.parentElement;
  var kids=[];
  if(box){
    var all=box.querySelectorAll('*');
    for(var i=0;i<all.length && kids.length<10;i++){
      var c=all[i], r=c.getBoundingClientRect();
      if(r.width<12 || r.height<12) continue;
      if(r.right<x || r.left>x+w || r.bottom<y || r.top>y+h) continue;
      var cls=(typeof c.className==='string'?c.className:'').trim();
      var label=c.nodeName.toLowerCase()+(cls?'.'+cls.split(/\s+/).slice(0,2).join('.'):'');
      var txt=(c.textContent||'').replace(/\s+/g,' ').trim().slice(0,32);
      kids.push(label+(txt?' "'+txt+'"':''));
    }
  }
  var pa=toDocPct(a.x,a.y), pb=toDocPct(b.x,b.y);
  return JSON.stringify({
    selector: box?cssPath(box):'',
    html: box?box.outerHTML.slice(0,1500):'',
    kids: kids,
    x: pa.x, y: pa.y, w: pb.x-pa.x, h: pb.y-pa.y
  });
};
// Draw a committed mark in DOCUMENT coordinates so it scrolls/zooms with the
// content instead of floating over it.
window.__odMark = function(n, kind, px, py, pw, ph){
  var d=docSize();
  var x=px/100*d.w, y=py/100*d.h;
  if(kind==='region'){
    var r=document.createElement('div');
    r.className='__od-mark rect';
    r.style.left=x+'px'; r.style.top=y+'px';
    r.style.width=(pw/100*d.w)+'px'; r.style.height=(ph/100*d.h)+'px';
    document.body.appendChild(r);
  }
  var b=document.createElement('div');
  b.className='__od-badge';
  b.style.left=x+'px'; b.style.top=y+'px';
  b.textContent=String(n);
  document.body.appendChild(b);
  return 'ok';
};
window.__odClearMarks = function(){
  var all=document.querySelectorAll('.__od-mark, .__od-badge');
  for(var i=0;i<all.length;i++) all[i].remove();
  return 'ok';
};
})();
''';
