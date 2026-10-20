package web

import "embed"

// Files contains the complete build-free frontend.
//
//go:embed index.html ui-state.js learning-visuals.js formula-renderer.js vendor/katex/katex.min.js vendor/katex/LICENSE app.js styles.css favicon.svg favicon.ico apple-touch-icon.png
var Files embed.FS
