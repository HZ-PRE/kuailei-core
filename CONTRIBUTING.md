Sdm uses [Go](https://go.dev), make sure that you have the correct version installed before starting development. You can use the following commands to check your installed version:


```shell
$ go version

# example response
go version go1.21.1 darwin/arm64
```

### Working with the Go Code

> if you're not interested in building/contributing to the Go code, you can skip this section

The Go code for Sdm can be found in the `sdm-core` folder, as a [git submodule](https://git-scm.com/book/en/v2/Git-Tools-Submodules) and in [core repository](https://github.com/sdm/sdm-next-core). The entrypoints for the desktop version are available in the [`sdm-core/custom`](https://github.com/sdm/sdm-next-core/tree/main/custom) folder and for the mobile version they can be found in the [`sdm-core/mobile`](https://github.com/sdm/sdm-next-core/tree/main/mobile) folder.

For the desktop version, we have to compile the Go code into a C shared library. We are providing a Makefile to generate the C shared libraries for all operating systems. The following Make commands will build sdm-core and copy the resulting output in [`sdm-core/bin`](https://github.com/sdm/sdm-next-core/tree/main/bin):

- `make windows-amd64`
- `make linux-amd64`
- `make macos-universal`

For the mobile version, we are using the [`gomobile`](https://github.com/golang/go/wiki/Mobile) tools. The following Make commands will build sdm-core for Android and iOS and copy the resulting output in [`sdm-core/bin`](https://github.com/sdm/sdm-next-core/tree/main/bin):

- `make android`
- `make ios`
