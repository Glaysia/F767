## Coding Conventions

- Favor plain C syntax across the firmware. Only minimal C++ features (e.g., struct methods) are acceptable when absolutely needed.
- Avoid namespaces, STL containers, and other high-level C++ abstractions in shared modules such as `Core/Src/user.cc`.
