# Typed vectors for C

C has no generics, and `void *` containers lose type safety. This example
generates header-only vectors for `int` and `double` from one template, so each
type gets its own struct and inline functions.

It generates `generated/vec_int.h` and `generated/vec_double.h`.

What it demonstrates:

- Deriving include guards and function names with `upper` and `snakeCase`.
- A fixed indentation that is not affected by the template tags.
- Reusing a header partial that reads `output`.

```bash
./bin/pre examples/c-generics/vectors.pre
gcc -fsyntax-only -x c examples/c-generics/generated/vec_int.h
```

Usage:

```c
#include "generated/vec_int.h"

VecInt scores = {0};
vec_int_push(&scores, 10);
vec_int_free(&scores);
```
