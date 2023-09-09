module twister

go 1.21.0

require (
	atomicgo.dev/robin v0.1.0
	github.com/elazarl/goproxy v0.0.0-20230808193330-2592e75ae04a
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2
)

replace github.com/elazarl/goproxy => ./goproxy/
