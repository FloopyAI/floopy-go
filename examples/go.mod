// Standalone module so the SDK itself stays dependency-light: the examples
// are not part of `go get github.com/FloopyAI/floopy-go`.
module github.com/FloopyAI/floopy-go/examples

go 1.23

require (
	github.com/FloopyAI/floopy-go v0.0.0
	github.com/openai/openai-go/v3 v3.37.0
)

require (
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

replace github.com/FloopyAI/floopy-go => ../
