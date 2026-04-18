# Waddle v2 — Native Dependency Audit

## onnxruntime-go (for Florence-2 Vision Engine)
- **Verdict:** [APPROVED]
- **Version:** v1.27.0
- **License:** MIT
- **DirectML:** Supported via `AppendExecutionProviderDirectML(0)`
- **Required DLLs:** `onnxruntime.dll` (DirectML build), `DirectML.dll`
- **Build tags:** None for DirectML — `onnx`/`!onnx` in our codebase only
- **Risk:** `DummyOrtDMLAPI` struct must be kept in sync with Microsoft headers
- **Notes:** This library provides an excellent C-wrapper strategy (`onnxruntime_wrapper.c`) to bypass the need for full DirectML C++ headers under MinGW. It passes the audit and has been added to `go.mod`.

## llama.cpp — Binding Strategy
- **Recommended approach:** [purego DLL]
- **Rationale:** The CGo static archive approach with MinGW-w64 creates massive cross-compilation headaches and MSVC/MinGW ABI incompatibility issues. The purego approach avoids CGo entirely by loading the official pre-built `llama.dll` dynamically at runtime, ensuring robust Windows compatibility.
- **CGo static verdict:** [NOT VIABLE on Windows]
- **purego DLL verdict:** [VIABLE]
- **Required DLLs:** `llama.dll` (plus `ggml-base.dll`, `ggml-cpu.dll`, `libomp.dll` depending on the build)
- **Build process:** Use `syscall.LoadLibraryW` on Windows (via `ebitengine/purego` on Unix) to dynamically load the official release DLLs. No CGo compilation step required.
- **Risk:** Requires `libffi` when calling functions that pass structs by value.
- **Notes:** Engineer A should proceed with the `purego` implementation. Pre-built binaries are available from the official `llama.cpp` releases.
