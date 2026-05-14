package main

// Built-in plugin registration lives at the cmd-level (not the plugin
// package) to avoid an import cycle: plugins import internal/plugin to
// call Register, so internal/plugin cannot in turn import the plugins.
//
// Adding a new built-in plugin:
//
//   1. Implement api.Plugin in internal/plugins/<name>.
//   2. In that package's init(), call plugin.Register(New()).
//   3. Add a blank import below.
//
// Out-of-tree plugins do NOT touch this file; they ride the gRPC
// sidecar protocol (v7.1+).

import (
	_ "github.com/HnyBadger/nightcrawler/internal/plugins/dns"
	// TODO(v7.0): wire remaining built-ins as they are ported:
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/tls"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/headers"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/paths"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/webshell"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/cms"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/methods"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/cors"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/gambling"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/redirect"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/disclosure"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/timing"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/xss"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/sqli"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/ports"
	// _ "github.com/HnyBadger/nightcrawler/internal/plugins/crtsh"
)
