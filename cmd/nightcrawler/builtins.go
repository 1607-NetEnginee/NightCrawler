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
	_ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/dns"
	// TODO(v7.0): wire remaining built-ins as they are ported:
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/tls"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/headers"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/paths"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/webshell"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/cms"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/methods"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/cors"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/gambling"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/redirect"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/disclosure"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/timing"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/xss"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/sqli"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/ports"
	// _ "github.com/1607-NetEnginee/NightCrawler/internal/plugins/crtsh"
)
