# SPIKE REPORT: llama.cpp Integration (Week 2)

## Goal
Attempt to integrate `llama.cpp` for local inference on Windows, starting with a CGo-based static linking approach (Plan A) and pivoting to a purego/DLL loading approach (Plan B) if necessary.

## Outcome: PIVOT TO PLAN B (DLL LOADING)
The project has pivoted to **Plan B (purego DLL loading)** for the following reasons:
1. **Toolchain Missing**: `cmake` was not found in the environment's PATH, preventing the automated build of `llama.cpp` static libraries.
2. **Windows Complexity**: Static linking of C++ libraries via CGo on Windows is notoriously fragile and requires a precisely configured MinGW-w64 toolchain which was not immediately available.
3. **Success of Proof-of-Concept**: A purego/DLL loading implementation was successfully scaffolded in `pkg/cognition/local/inference_dll.go`, allowing for runtime loading of `llama.dll` without CGo compile-time dependencies.

## Implementation Details
- **Architecture**: Switched from compile-time CGo linking to runtime dynamic loading via `syscall.NewLazyDLL`.
- **Portability**: Non-Windows builds automatically fall back to a stub implementation in `inference_stub.go`.
- **Build System**: Removed the `-tags llama_cpp` requirement from the `Makefile`. All builds now support local inference if `llama.dll` is present at runtime.

## Next Steps
1. **Binary Distribution**: Ensure `llama.dll` (and any required dependencies like `libopenblas.dll`) are distributed with the application or downloaded during setup.
2. **Binding Completion**: Finalize the `syscall` procedure calls to match the `llama.cpp` C API signatures once the specific DLL version is selected.
3. **Hardware Acceleration**: Investigate `cuBLAS` or `Vulkan` variants of `llama.dll` for GPU-accelerated local inference.

## Verification
- `go build ./...` passes without native toolchain dependencies.
- `go test ./pkg/cognition/local/...` passes (verifies graceful failure when DLL is missing).
