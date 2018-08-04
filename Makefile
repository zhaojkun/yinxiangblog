all: route static
	go build
route:
	rg . > router.go
static:
	go-bindata -o static.go static/...

.PHONY: static
