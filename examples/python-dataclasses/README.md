# Python dataclasses

Data models are rendered into `dataclass` definitions with type hints, defaults
and a `to_dict` helper.

It generates `generated/user.py` and `generated/product.py`.

What it demonstrates:

- `snakeCase` for attribute names.
- Optional fields rendered as `T | None = None`.
- Quoted defaults coming straight from the frontmatter.
- The same template reused for every model.

```bash
./bin/pre examples/python-dataclasses/models.pre
python3 -m py_compile examples/python-dataclasses/generated/user.py
```

Usage:

```python
from generated.user import User

user = User(id=1, email="dev@example.com")
print(user.to_dict())
```
