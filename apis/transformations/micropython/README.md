# MicroPython

- [How to build/update `micropython.wasm`](#how-to-buildupdate-micropythonwasm)
- [Build the MicroPython executable](#build-the-micropython-executable)
- [Run the REPL](#run-the-repl)
- [Interpreting a Python script](#interpreting-a-python-script)

## How to build/update `micropython.wasm`

1. Install the Emscripten tools following the instructions at https://emscripten.org/docs/getting_started/downloads.html.
2. Pull the https://github.com/micropython/micropython repository
3. Follow the instructions in the README.md under `ports/webassembly`
4. Copy the file `ports/webassembly/build/firmware.wasm` over the WASM file `wasm/micropython.wasm` in this repository

## Build the MicroPython executable

Run:

```bash
go build -o micropython ./cli
```

## Run the REPL

After compiling `micropython`, simply run:

```
./micropython
```

## Interpreting a Python script

After compiling `micropython`, run:

```
./micropython path/to/filename.py
```

You can find some Python scripts in the [examples](examples) directory.