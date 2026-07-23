module github.com/Algo2147483647/ray/studio

go 1.24.6

toolchain go1.24.10

require github.com/Algo2147483647/ray/engine v0.0.0

require (
	github.com/expr-lang/expr v1.17.8 // indirect
	gonum.org/v1/gonum v0.16.0
)

replace github.com/Algo2147483647/ray/engine => ../engine
