module github.com/awch-D/ForgeX/forgex-mcp

go 1.25.0

require github.com/awch-D/ForgeX/forgex-core v0.0.0-20260303024956-e7a52011e492

require github.com/mattn/go-sqlite3 v1.14.34 // indirect

require (
	github.com/awch-D/ForgeX/forgex-governance v0.0.0
	github.com/awch-D/ForgeX/forgex-sandbox v0.0.0
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
)

replace github.com/awch-D/ForgeX/forgex-governance => ../forgex-governance

replace github.com/awch-D/ForgeX/forgex-sandbox => ../forgex-sandbox
