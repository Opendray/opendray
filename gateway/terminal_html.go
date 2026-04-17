package gateway

// terminalHTML is a standalone xterm.js terminal page.
// Embedded in the Go binary and served at /terminal.html?session=<id>
// Flutter web uses an iframe to this page for native-quality terminal rendering.
const terminalHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<title>NTC Terminal</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/css/xterm.min.css">
<style>
  *{margin:0;padding:0;box-sizing:border-box;-webkit-tap-highlight-color:transparent}
  html,body{height:100%;overflow:hidden;background:#0b0d11;font-smooth:always;-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}
  #terminal{height:100%;width:100%;padding:6px}
  .xterm{height:100%!important;font-variant-ligatures:none}
  .xterm-viewport::-webkit-scrollbar{width:6px}
  .xterm-viewport::-webkit-scrollbar-track{background:transparent}
  .xterm-viewport::-webkit-scrollbar-thumb{background:#2a2e38;border-radius:3px}
</style>
</head>
<body>
<div id="terminal"></div>
<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@5.5.0/lib/xterm.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.10.0/lib/addon-fit.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-web-links@0.11.0/lib/addon-web-links.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-unicode11@0.8.0/lib/addon-unicode11.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-webgl@0.18.0/lib/addon-webgl.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-canvas@0.7.0/lib/addon-canvas.min.js"></script>
<script>
(function(){
  const params = new URLSearchParams(location.search);
  const sessionId = params.get('session');
  const wsBase = params.get('wsBase') || location.origin;
  if(!sessionId){document.body.innerText='Missing session param';return}

  // emit() — delivers events to the Flutter host.
  // Mobile WebView: NTC JavaScript channel is injected by Flutter; use it directly.
  // Web iframe: NTC is not available; fall back to postMessage to the parent frame.
  // Using NTC exclusively (when available) prevents the _injectMessageBridge doubling.
  function emit(type) {
    try {
      if(typeof NTC !== 'undefined') { NTC.postMessage(JSON.stringify({type:type})); return; }
    } catch(e){}
    try { window.parent.postMessage({type:type},'*'); } catch(e){}
  }

  // Detect mobile for optimal font size
  const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
  const dpr = window.devicePixelRatio || 1;

  // No CDN font wait — terminal starts immediately with system fonts.
  // Waiting for Google Fonts CDN blocks the entire terminal init on LANs without internet.
  // System font stack covers CJK characters natively (PingFang SC on iOS, Noto on Android).

  const term = new Terminal({
    fontFamily: "'SF Mono','JetBrains Mono','Cascadia Code','Fira Code',Menlo,Monaco,Consolas,'PingFang SC','Hiragino Sans GB','Noto Sans Mono CJK SC','Microsoft YaHei',monospace",
    fontSize: isMobile ? 13 : 14,
    fontWeight: 400,
    fontWeightBold: 600,
    letterSpacing: 0,
    lineHeight: 1.2,
    cursorBlink: true,
    cursorStyle: 'bar',
    cursorWidth: 2,
    allowProposedApi: true,
    scrollback: 10000,
    smoothScrollDuration: 0,
    minimumContrastRatio: 4.5,
    rescaleOverlappingGlyphs: true,
    drawBoldTextInBrightColors: false,
    convertEol: false,
    theme: {
      background:'#0b0d11',foreground:'#c0caf5',cursor:'#c0caf5',
      selectionBackground:'#33467c',selectionForeground:'#c0caf5',
      black:'#15161e',red:'#f7768e',green:'#9ece6a',yellow:'#e0af68',
      blue:'#7aa2f7',magenta:'#bb9af7',cyan:'#7dcfff',white:'#a9b1d6',
      brightBlack:'#414868',brightRed:'#f7768e',brightGreen:'#9ece6a',
      brightYellow:'#e0af68',brightBlue:'#7aa2f7',brightMagenta:'#bb9af7',
      brightCyan:'#7dcfff',brightWhite:'#c0caf5'
    }
  });

  const fitAddon = new FitAddon.FitAddon();
  const webLinksAddon = new WebLinksAddon.WebLinksAddon();
  const unicode11Addon = new Unicode11Addon.Unicode11Addon();
  term.loadAddon(fitAddon);
  term.loadAddon(webLinksAddon);
  term.loadAddon(unicode11Addon);
  term.unicode.activeVersion = '11';

  term.open(document.getElementById('terminal'));

  // Disable iOS/Android auto-modifications on xterm's hidden input —
  // autocapitalize ("Is" instead of "ls"), autocorrect, smart quotes, and
  // spellcheck all corrupt terminal input and break IME composition.
  if (term.textarea) {
    term.textarea.setAttribute('autocapitalize', 'none');
    term.textarea.setAttribute('autocorrect', 'off');
    term.textarea.setAttribute('autocomplete', 'off');
    term.textarea.setAttribute('spellcheck', 'false');
    term.textarea.setAttribute('inputmode', 'text');
  }

  // Minimal focus glue: tap on the terminal raises the keyboard by routing
  // focus to xterm's textarea from inside a user-gesture handler (Android
  // won't show the IME for programmatic focus outside a gesture). We only
  // steal focus if nothing useful is focused — never during IME composition.
  var composing = false;
  if (term.textarea) {
    term.textarea.addEventListener('compositionstart', function(){ composing = true; });
    term.textarea.addEventListener('compositionend',   function(){ composing = false; });
  }
  function focusIfUnfocused() {
    if (composing) return;
    var ae = document.activeElement;
    if (!ae || ae === document.body || ae === document.documentElement) {
      try { term.focus(); } catch(_){}
    }
  }
  document.getElementById('terminal')
    .addEventListener('touchstart', focusIfUnfocused, {passive: true});
  document.addEventListener('visibilitychange', function(){
    if (!document.hidden) focusIfUnfocused();
  });

  // Renderer selection:
  //   • Desktop  → WebGL (GPU, smoothest)
  //   • Mobile   → Canvas (2D canvas, much faster than the DOM renderer,
  //                         and unlike WebGL doesn't suffer iOS GPU context
  //                         loss when the app backgrounds)
  // The DOM renderer — the fallback if both addons fail — is very slow and
  // visibly stutters Claude's TUI repaints. Loading Canvas on mobile fixes
  // the "input lag / janky scrolling" symptom without risking blank screens.
  if (!isMobile) {
    try {
      const webglAddon = new WebglAddon.WebglAddon();
      webglAddon.onContextLoss(function(){
        try { webglAddon.dispose(); } catch(e2){}
        try { fitAddon.fit(); } catch(e2){}
      });
      term.loadAddon(webglAddon);
    } catch(e) {}
  } else {
    try {
      const canvasAddon = new CanvasAddon.CanvasAddon();
      term.loadAddon(canvasAddon);
    } catch(e) {}
  }

  fitAddon.fit();

  // WebSocket with auto-reconnect.
  // The previous implementation had no reconnect, so any socket close
  // (network blip, OS modal, Claude repaint stall, anything) left the
  // terminal permanently deaf — the user's first command worked, the
  // second disappeared into a closed socket. We now reconnect with
  // exponential backoff and route sends through sendRaw() so they always
  // use the latest socket.
  const scheme = wsBase.startsWith('https') ? 'wss' : 'ws';
  const host   = wsBase.replace(/^https?:\/\//, '');
  const wsUrl  = scheme + '://' + host + '/api/sessions/' + sessionId + '/ws';
  var ws = null;
  var reconnectAttempts = 0;
  var reconnectTimer    = null;
  var shouldReconnect   = true;

  function handleMessage(ev){
    if (ev.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(ev.data));
    } else {
      try {
        const msg = JSON.parse(ev.data);
        if      (msg.type === 'waiting_for_input') emit('ntc:idle');
        else if (msg.type === 'process_exit')      emit('ntc:exit');
        else if (msg.type === 'replay_start')      emit('ntc:replay_start');
        else if (msg.type === 'replay_end')        emit('ntc:replay_end');
      } catch(_){}
    }
  }

  function scheduleReconnect(){
    if (!shouldReconnect) return;
    if (reconnectTimer) return;
    // Exponential backoff capped at 5s. First retry waits 800 ms — long
    // enough to avoid racing the normal close → replay handshake, short
    // enough to feel responsive on a flaky mobile network.
    const delay = Math.min(800 * Math.pow(1.6, Math.max(0, reconnectAttempts - 1)), 5000);
    reconnectTimer = setTimeout(function(){
      reconnectTimer = null;
      connect();
    }, delay);
  }

  function connect(){
    reconnectAttempts += 1;
    try { ws = new WebSocket(wsUrl); } catch(_){ scheduleReconnect(); return; }
    ws.binaryType = 'arraybuffer';
    ws.onopen    = function(){
      reconnectAttempts = 0;
      emit('ntc:connected');
    };
    ws.onmessage = handleMessage;
    ws.onerror   = function(){ /* onclose handles the reconnect */ };
    ws.onclose   = function(){
      emit('ntc:disconnected');
      scheduleReconnect();
    };
  }
  connect();

  function sendRaw(bytes){
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(bytes);
      return true;
    }
    return false;
  }

  // User typing — one send per keystroke, no extra refocus (that was
  // interrupting IME composition mid-character).
  term.onData(function(data){
    sendRaw(new TextEncoder().encode(data));
  });

  // Exposed for Flutter WebView: send raw key sequences from quick-keys bar
  // or the image-attach flow. Pure WebSocket write — no focus side-effects.
  window.ntcSendKey = function(data){
    return sendRaw(new TextEncoder().encode(data));
  };

  // Expose a connection status getter for the Flutter side to poll if it
  // wants to show "reconnecting N/5" instead of a vague "...".
  window.ntcWsState = function(){
    if (!ws) return 'closed';
    switch (ws.readyState) {
      case WebSocket.CONNECTING: return 'connecting';
      case WebSocket.OPEN:       return 'open';
      case WebSocket.CLOSING:    return 'closing';
      case WebSocket.CLOSED:     return 'closed';
      default:                    return 'unknown';
    }
  };

  // Only forward resize to the server when cols/rows actually change.
  // Rapid keyboard-animation frames used to produce a storm of SIGWINCHs,
  // each of which made TUI apps (Claude CLI, vim) redraw their full UI.
  var lastCols = 0, lastRows = 0;
  term.onResize(function(size){
    if (size.cols === lastCols && size.rows === lastRows) return;
    lastCols = size.cols;
    lastRows = size.rows;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({type:'resize',cols:size.cols,rows:size.rows}));
    }
  });

  // Debounce ResizeObserver so intermediate keyboard-show/hide frames coalesce
  // into a single fit() once the size has actually settled. Without this, a
  // single keyboard toggle fires ~6 fit() calls and Claude renders its header
  // on each of them, producing visibly duplicated TUI chrome.
  var resizeTimer = null;
  var ro = new ResizeObserver(function(){
    if (resizeTimer) clearTimeout(resizeTimer);
    resizeTimer = setTimeout(function(){
      resizeTimer = null;
      try { fitAddon.fit(); } catch(_){}
    }, 120);
  });
  ro.observe(document.getElementById('terminal'));

  // Listen for messages from parent Flutter app
  window.addEventListener('message', function(ev){
    if(!ev.data || !ev.data.type) return;
    if(ev.data.type==='ntc:focus') focusIfUnfocused();
    if(ev.data.type==='ntc:close'){
      shouldReconnect = false;
      if (ws) ws.close();
    }
  });

  // Ping keepalive — server sees this as an empty frame and keeps the
  // connection warm through NATs / mobile-network idle timers.
  setInterval(function(){
    if (ws && ws.readyState === WebSocket.OPEN) ws.send(new Uint8Array(0));
  }, 30000);

  // Notify parent we're ready
  emit('ntc:ready');
})();
</script>
</body>
</html>`
