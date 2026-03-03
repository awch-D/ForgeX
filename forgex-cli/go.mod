module github.com/awch-D/ForgeX/forgex-cli

go 1.25.0

require (
	github.com/awch-D/ForgeX/forgex-agent v0.0.0-20260303024956-e7a52011e492
	github.com/awch-D/ForgeX/forgex-cognition v0.0.0-00010101000000-000000000000
	github.com/awch-D/ForgeX/forgex-core v0.0.0
	github.com/awch-D/ForgeX/forgex-gear v0.0.0-20260303024956-e7a52011e492
	github.com/awch-D/ForgeX/forgex-intent v0.0.0-20260303024956-e7a52011e492
	github.com/awch-D/ForgeX/forgex-llm v0.0.0-20260303024956-e7a52011e492
	github.com/awch-D/ForgeX/forgex-mcp v0.0.0-20260303024956-e7a52011e492
	github.com/pterm/pterm v0.12.83
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/awch-D/ForgeX/forgex-governance v0.0.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.34 // indirect
)

require (
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	github.com/awch-D/ForgeX/forgex-evolution v0.0.0
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/containerd/console v1.0.5 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gookit/color v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/mattn/go-runewidth v0.0.20 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/term v0.40.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

replace github.com/awch-D/ForgeX/forgex-core => ../forgex-core

replace github.com/awch-D/ForgeX/forgex-evolution => ../forgex-evolution

replace github.com/awch-D/ForgeX/forgex-cognition => ../forgex-cognition

replace github.com/awch-D/ForgeX/forgex-intent => ../forgex-intent

replace github.com/awch-D/ForgeX/forgex-llm => ../forgex-llm

replace github.com/awch-D/ForgeX/forgex-agent => ../forgex-agent

replace github.com/awch-D/ForgeX/forgex-gear => ../forgex-gear

replace github.com/awch-D/ForgeX/forgex-mcp => ../forgex-mcp

replace github.com/awch-D/ForgeX/forgex-governance => ../forgex-governance

replace github.com/awch-D/ForgeX/forgex-server => ../forgex-server
